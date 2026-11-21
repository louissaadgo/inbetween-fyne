// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build darwin && ios

#include "_cgo_export.h"
#include <pthread.h>
#include <stdio.h>
#include <sys/utsname.h>

#import <UIKit/UIKit.h>
#import <MobileCoreServices/MobileCoreServices.h>
#import <GLKit/GLKit.h>
#import <UserNotifications/UserNotifications.h>

struct utsname sysInfo;

static CGFloat keyboardHeight;

// goOrientation is the app's orientation (Inbetween's iPad build: its
// window scene's; the status bar's says nothing under the scene life
// cycle iOS 27 requires).
static UIInterfaceOrientation goOrientation(void) {
	for (UIScene *s in [UIApplication sharedApplication].connectedScenes) {
		if ([s isKindOfClass:[UIWindowScene class]]) {
			return ((UIWindowScene *)s).interfaceOrientation;
		}
	}
	return UIInterfaceOrientationPortrait;
}

@interface GoAppAppController : GLKViewController<UIContentContainer, GLKViewDelegate>
@end

@interface GoInputView : UITextField<UITextFieldDelegate>
@end

@interface GoAppAppDelegate : UIResponder<UIApplicationDelegate>
@property (strong, nonatomic) UIWindow *window;
@property (strong, nonatomic) GoAppAppController *controller;
@end

@implementation GoAppAppDelegate
- (BOOL)application:(UIApplication *)application didFinishLaunchingWithOptions:(NSDictionary *)launchOptions {
    int scale = 1;
    if ([[UIScreen mainScreen] respondsToSelector:@selector(displayLinkWithTarget:selector:)]) {
		scale = (int)[UIScreen mainScreen].scale; // either 1.0, 2.0, or 3.0.
	}
    CGSize size = [UIScreen mainScreen].nativeBounds.size;
    setDisplayMetrics((int)size.width, (int)size.height, scale);

	lifecycleAlive();
	// (Inbetween's iPad build: the window is made for the app's scene, by
	// GoAppSceneDelegate: iOS 27 starts no app without the scene life cycle)

	UNUserNotificationCenter *center = [UNUserNotificationCenter currentNotificationCenter];
	center.delegate = (id) self;

	return YES;
}

- (UISceneConfiguration *)application:(UIApplication *)application configurationForConnectingSceneSession:(UISceneSession *)session options:(UISceneConnectionOptions *)options {
	// (Inbetween's iPad build: the app's one scene, GoAppSceneDelegate's)
	UISceneConfiguration *c = [UISceneConfiguration configurationWithName:@"Default" sessionRole:session.role];
	c.delegateClass = NSClassFromString(@"GoAppSceneDelegate");
	return c;
}

- (void)applicationDidBecomeActive:(UIApplication * )application {
	lifecycleFocused();
}

- (void)applicationWillResignActive:(UIApplication *)application {
	lifecycleVisible();
}

- (void)applicationDidEnterBackground:(UIApplication *)application {
	lifecycleAlive();
}

- (void)applicationWillTerminate:(UIApplication *)application {
	lifecycleDead();
}

- (void)applicationDidReceiveMemoryWarning:(UIApplication *)application {
	lifecycleMemoryWarning();
}

- (void)documentPicker:(UIDocumentPickerViewController *)controller didPickDocumentsAtURLs:(NSArray <NSURL *>*)urls {
    if ([urls count] == 0) {
        return;
    }

    NSURL* url = urls[0];
    NSURL* toClose = NULL;
    BOOL secured = [url startAccessingSecurityScopedResource];
    if (secured) {
        toClose = url;
    }

    filePickerReturned((char*)[[url description] UTF8String], toClose);
}

- (void)documentPickerWasCancelled:(UIDocumentPickerViewController *)controller {
    filePickerReturned("", NULL);
}

- (void)userNotificationCenter:(UNUserNotificationCenter *)center
       willPresentNotification:(UNNotification *)notification
         withCompletionHandler:(void (^)(UNNotificationPresentationOptions options))completionHandler {
	completionHandler(UNNotificationPresentationOptionAlert);
}
@end

// GoAppSceneDelegate is the app's one scene (Inbetween's iPad build: iOS 27
// starts no app without the scene life cycle): its window, made as the app
// delegate made it; the app's life cycle from the scene's (the app
// delegate's own isn't called under scenes); files opened in the app
// (Files' Open In), handed to the app delegate's application:openURL:
// options: where it has one.
@interface GoAppSceneDelegate : UIResponder<UIWindowSceneDelegate>
@property (strong, nonatomic) UIWindow *window;
@end

@implementation GoAppSceneDelegate
- (void)scene:(UIScene *)scene willConnectToSession:(UISceneSession *)session options:(UISceneConnectionOptions *)connectionOptions {
	if (![scene isKindOfClass:[UIWindowScene class]]) {
		return;
	}
	UIWindowScene *ws = (UIWindowScene *)scene;
	GoAppAppDelegate *app = (GoAppAppDelegate *)[UIApplication sharedApplication].delegate;
	UIWindow *w = [[UIWindow alloc] initWithWindowScene:ws];
	self.window = w;
	app.window = w;
	app.controller = [[GoAppAppController alloc] initWithNibName:nil bundle:nil];
	w.rootViewController = app.controller;
	[w makeKeyAndVisible];
	CGSize size = [UIScreen mainScreen].nativeBounds.size;
	updateConfig((int)size.width, (int)size.height, ws.interfaceOrientation);
	if (connectionOptions.URLContexts.count > 0) {
		[self scene:scene openURLContexts:connectionOptions.URLContexts];
	}
}

- (void)scene:(UIScene *)scene openURLContexts:(NSSet<UIOpenURLContext *> *)contexts {
	id<UIApplicationDelegate> app = [UIApplication sharedApplication].delegate;
	if (![app respondsToSelector:@selector(application:openURL:options:)]) {
		return;
	}
	for (UIOpenURLContext *c in contexts) {
		[app application:[UIApplication sharedApplication] openURL:c.URL options:@{}];
	}
}

- (void)sceneDidBecomeActive:(UIScene *)scene {
	lifecycleFocused();
}

- (void)sceneWillResignActive:(UIScene *)scene {
	lifecycleVisible();
}

- (void)sceneDidEnterBackground:(UIScene *)scene {
	lifecycleAlive();
}
@end

@interface GoAppAppController ()
@property (strong, nonatomic) EAGLContext *context;
@property (strong, nonatomic) GLKView *glview;
@property (strong, nonatomic) GoInputView *inputView;
@end

@implementation GoAppAppController
- (void)viewWillAppear:(BOOL)animated
{
	// TODO: replace by swapping out GLKViewController for a UIVIewController.
	[super viewWillAppear:animated];
	self.paused = YES;

    [[NSNotificationCenter defaultCenter] addObserver:self selector:@selector(keyboardWillShow:) name:UIKeyboardWillShowNotification object:nil];
    [[NSNotificationCenter defaultCenter] addObserver:self selector:@selector(keyboardWillHide:) name:UIKeyboardWillHideNotification object:nil];
}

- (void)viewWillDisappear:(BOOL)animated {
    [super viewWillDisappear:animated];

    [[NSNotificationCenter defaultCenter] removeObserver:self name:UIKeyboardWillShowNotification object:nil];
    [[NSNotificationCenter defaultCenter] removeObserver:self name:UIKeyboardWillHideNotification object:nil];
}

- (void)viewDidLoad {
	[super viewDidLoad];
	self.context = [[EAGLContext alloc] initWithAPI:kEAGLRenderingAPIOpenGLES2];
	self.inputView = [[GoInputView alloc] initWithFrame:CGRectMake(0, 0, 0, 0)];
	self.inputView.delegate = self.inputView;
	self.inputView.autocapitalizationType = UITextAutocapitalizationTypeNone;
	self.inputView.autocorrectionType = UITextAutocorrectionTypeNo;
	[self.view addSubview:self.inputView];
	self.glview = (GLKView*)self.view;
	self.glview.drawableDepthFormat = GLKViewDrawableDepthFormat24;
	self.glview.multipleTouchEnabled = true; // TODO expose setting to user.
	self.glview.context = self.context;
	self.glview.userInteractionEnabled = YES;
	//self.glview.enableSetNeedsDisplay = YES; // only invoked once

	// Do not use the GLKViewController draw loop.
	//self.paused = YES;
	//self.resumeOnDidBecomeActive = NO;
	//self.preferredFramesPerSecond = 0;

	int scale = 1;
	if ([[UIScreen mainScreen] respondsToSelector:@selector(displayLinkWithTarget:selector:)]) {
		scale = (int)[UIScreen mainScreen].scale; // either 1.0, 2.0, or 3.0.
	}
	setScreen(scale);

	CGSize size = [UIScreen mainScreen].nativeBounds.size;
	UIInterfaceOrientation orientation = goOrientation();
	updateConfig((int)size.width, (int)size.height, orientation);

    self.glview.enableSetNeedsDisplay = NO;
    CADisplayLink* displayLink = [CADisplayLink displayLinkWithTarget:self selector:@selector(render:)];
    [displayLink addToRunLoop:[NSRunLoop currentRunLoop] forMode:NSDefaultRunLoopMode];
}

- (void)viewWillTransitionToSize:(CGSize)ptSize withTransitionCoordinator:(id<UIViewControllerTransitionCoordinator>)coordinator {
	[coordinator animateAlongsideTransition:^(id<UIViewControllerTransitionCoordinatorContext> context) {
		// TODO(crawshaw): come up with a plan to handle animations.
	} completion:^(id<UIViewControllerTransitionCoordinatorContext> context) {
		UIInterfaceOrientation orientation = goOrientation();
		CGSize size = [UIScreen mainScreen].nativeBounds.size;
		updateConfig((int)size.width, (int)size.height, orientation);
	}];
}

- (void)render:(CADisplayLink*)displayLink {
    if (needsDraw()) {
        [self.glview display];
    }
}

- (void)glkView:(GLKView *)view drawInRect:(CGRect)rect {
    drawloop();
}

#define TOUCH_TYPE_BEGIN 0 // touch.TypeBegin
#define TOUCH_TYPE_MOVE  1 // touch.TypeMove
#define TOUCH_TYPE_END   2 // touch.TypeEnd

static void sendTouches(int change, NSSet* touches) {
	CGFloat scale = [UIScreen mainScreen].nativeScale;
	for (UITouch* touch in touches) {
		CGPoint p = [touch locationInView:touch.view];
		sendTouch((GoUintptr)touch, (GoUintptr)change, p.x*scale, p.y*scale);
	}
}

- (void)touchesBegan:(NSSet*)touches withEvent:(UIEvent*)event {
	sendTouches(TOUCH_TYPE_BEGIN, touches);
}

- (void)touchesMoved:(NSSet*)touches withEvent:(UIEvent*)event {
	sendTouches(TOUCH_TYPE_MOVE, touches);
}

- (void)touchesEnded:(NSSet*)touches withEvent:(UIEvent*)event {
	sendTouches(TOUCH_TYPE_END, touches);
}

- (void)touchesCanceled:(NSSet*)touches withEvent:(UIEvent*)event {
    sendTouches(TOUCH_TYPE_END, touches);
}

- (void) traitCollectionDidChange: (UITraitCollection *) previousTraitCollection {
    [super traitCollectionDidChange: previousTraitCollection];

	UIInterfaceOrientation orientation = goOrientation();
	CGSize size = [UIScreen mainScreen].nativeBounds.size;
	updateConfig((int)size.width, (int)size.height, orientation);
}

- (void)keyboardWillShow:(NSNotification *)note {
    CGSize keyboardSize = [[[note userInfo] objectForKey:UIKeyboardFrameEndUserInfoKey] CGRectValue].size;
    keyboardHeight = keyboardSize.height;

    CGSize size = [UIScreen mainScreen].nativeBounds.size;
	UIInterfaceOrientation orientation = goOrientation();
	updateConfig((int)size.width, (int)size.height, orientation);
}

- (void)keyboardWillHide:(NSNotification *)note {
    keyboardHeight = 0;

    CGSize size = [UIScreen mainScreen].nativeBounds.size;
	UIInterfaceOrientation orientation = goOrientation();
	updateConfig((int)size.width, (int)size.height, orientation);
}

@end

@implementation GoInputView

- (BOOL)canBecomeFirstResponder {
    return YES;
}

- (void)deleteBackward {
    keyboardDelete();
}

-(BOOL)textField:(UITextField *)textField shouldChangeCharactersInRange:(NSRange)range replacementString:(NSString *)string {
    keyboardTyped((char *)[string UTF8String]);
    return NO;
}

- (BOOL)textFieldShouldReturn:(UITextField *)textField {
    if ([self returnKeyType] != UIReturnKeyDone) {
        keyboardTyped("\n");
        return YES;
    }

    dispatch_async(dispatch_get_main_queue(), ^{
        [self resignFirstResponder];
    });

    return NO;
}

@end

void runApp(void) {
	char * argv[] = {};
	@autoreleasepool {
		UIApplicationMain(0, argv, nil, NSStringFromClass([GoAppAppDelegate class]));
	}
}

void makeCurrentContext(GLintptr context) {
	EAGLContext* ctx = (EAGLContext*)context;
	if (![EAGLContext setCurrentContext:ctx]) {
		// TODO(crawshaw): determine how terrible this is. Exit?
		NSLog(@"failed to set current context");
	}
}

void swapBuffers(GLintptr context) {
	__block EAGLContext* ctx = (EAGLContext*)context;
	dispatch_sync(dispatch_get_main_queue(), ^{
		[EAGLContext setCurrentContext:ctx];
		[ctx presentRenderbuffer:GL_RENDERBUFFER];
	});
}

uint64_t threadID() {
	uint64_t id;
	if (pthread_threadid_np(pthread_self(), &id)) {
		abort();
	}
	return id;
}

UIEdgeInsets getDevicePadding() {
    if (@available(iOS 11.0, *)) {
        UIWindow *window = UIApplication.sharedApplication.keyWindow;

        UIEdgeInsets inset = window.safeAreaInsets;
        if (keyboardHeight != 0) {
            inset.bottom = keyboardHeight;
        }
        return inset;
    }

    return UIEdgeInsetsZero;
}

bool isDark() {
    UIViewController *rootVC = [[[[UIApplication sharedApplication] delegate] window] rootViewController];
    return rootVC.traitCollection.userInterfaceStyle == UIUserInterfaceStyleDark;
}

#define DEFAULT_KEYBOARD_CODE 0
#define SINGLELINE_KEYBOARD_CODE 1
#define NUMBER_KEYBOARD_CODE 2

void showKeyboard(int keyboardType) {
    GoAppAppDelegate *appDelegate = (GoAppAppDelegate *)[[UIApplication sharedApplication] delegate];
    GoInputView *view = appDelegate.controller.inputView;

    dispatch_async(dispatch_get_main_queue(), ^{
        switch (keyboardType)
        {
            case DEFAULT_KEYBOARD_CODE:
                [view setKeyboardType:UIKeyboardTypeDefault];
                [view setReturnKeyType:UIReturnKeyDefault];
                break;
            case SINGLELINE_KEYBOARD_CODE:
                [view setKeyboardType:UIKeyboardTypeDefault];
                [view setReturnKeyType:UIReturnKeyDone];
                break;
            case NUMBER_KEYBOARD_CODE:
                [view setKeyboardType:UIKeyboardTypeNumberPad];
                [view setReturnKeyType:UIReturnKeyDone];
                break;
            default:
                NSLog(@"unknown keyboard type, use default");
                [view setKeyboardType:UIKeyboardTypeDefault];
                [view setReturnKeyType:UIReturnKeyDefault];
                break;
        }
        // refresh settings if keyboard is already open
        [view reloadInputViews];

        BOOL ret = [view becomeFirstResponder];
    });
}

void hideKeyboard() {
    GoAppAppDelegate *appDelegate = (GoAppAppDelegate *)[[UIApplication sharedApplication] delegate];
    GoInputView *view = appDelegate.controller.inputView;

    dispatch_async(dispatch_get_main_queue(), ^{
        [view resignFirstResponder];
    });
}

NSMutableArray *docTypesForMimeExts(char *mimes, char *exts) {
    NSMutableArray *docTypes = [NSMutableArray array];
    if (mimes != NULL && strlen(mimes) > 0) {
        NSString *mimeList = [NSString stringWithUTF8String:mimes];

        if ([mimeList isEqualToString:@"application/x-directory"]) {
            [docTypes addObject:(NSString*)kUTTypeFolder];
        } else {
            NSArray *mimeItems = [mimeList componentsSeparatedByString:@"|"];

            for (NSString *mime in mimeItems)  {
                NSString *UTI = (NSString *) UTTypeCreatePreferredIdentifierForTag(kUTTagClassMIMEType, (CFStringRef)mime, NULL);

                [docTypes addObject:UTI];
            }
        }
    } else if (exts != NULL && strlen(exts) > 0) {
        NSString *extList = [NSString stringWithUTF8String:exts];
        NSArray *extItems = [extList componentsSeparatedByString:@"|"];

        for (NSString *ext in extItems)  {
            NSString *UTI = (NSString *) UTTypeCreatePreferredIdentifierForTag(kUTTagClassFilenameExtension, (CFStringRef)ext, NULL);

            [docTypes addObject:UTI];
        }
    } else {
        [docTypes addObject:@"public.data"];
    }

    return docTypes;
}

void showFileOpenPicker(char* mimes, char *exts) {
    GoAppAppDelegate *appDelegate = (GoAppAppDelegate *)[[UIApplication sharedApplication] delegate];

    NSMutableArray *docTypes = docTypesForMimeExts(mimes, exts);

    UIDocumentPickerViewController *documentPicker = [[UIDocumentPickerViewController alloc]
        initWithDocumentTypes:docTypes inMode:UIDocumentPickerModeOpen];
    documentPicker.delegate = (id) appDelegate;

    dispatch_async(dispatch_get_main_queue(), ^{
        [appDelegate.controller presentViewController:documentPicker animated:YES completion:nil];
    });
}

void showFileSavePicker(char* mimes, char *exts) {
    GoAppAppDelegate *appDelegate = (GoAppAppDelegate *)[[UIApplication sharedApplication] delegate];

    NSMutableArray *docTypes = docTypesForMimeExts(mimes, exts);

    NSURL *temporaryDirectoryURL = [NSURL fileURLWithPath: NSTemporaryDirectory() isDirectory: YES];
    NSURL *temporaryFileURL = [temporaryDirectoryURL URLByAppendingPathComponent:@"filename"];

    char* bytes = "\n";
    NSData *data = [NSData dataWithBytes:bytes length:1];
    BOOL ok = [data writeToURL:temporaryFileURL atomically:YES];

    UIDocumentPickerViewController *documentPicker = [[UIDocumentPickerViewController alloc]
        initWithURL:temporaryFileURL inMode:UIDocumentPickerModeMoveToService];
    documentPicker.delegate = (id) appDelegate;

    dispatch_async(dispatch_get_main_queue(), ^{
        [appDelegate.controller presentViewController:documentPicker animated:YES completion:nil];
    });
}

void closeFileResource(void* urlPtr) {
    if (urlPtr == NULL) {
        return;
    }

    NSURL* url = (NSURL*) urlPtr;
    [url stopAccessingSecurityScopedResource];
}
