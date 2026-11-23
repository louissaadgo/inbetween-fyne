# Fyne for Inbetween

This is [Fyne](https://github.com/fyne-io/fyne) with the changes Inbetween's
iPad app needs. The branch `inbetween` is Fyne's release tag (`v2.8.2` now)
with one commit per change on top. Inbetween's iPad build uses this fork;
its desktop builds use Fyne as released.

## The changes

| Commit | What and why |
|---|---|
| Touch IDs | An ended touch frees its own ID only. Fyne freed every touch's ID whenever any touch ended, so the fingers still down came back under new IDs: a pinch lost its second finger as the first lifted, and a Pencil drawing under a lifting palm became a new touch. |
| UIScene | iOS 27 starts no app built with its SDK without the scene life cycle. A scene delegate (`GoAppSceneDelegate`, named in the app's Info.plist) makes the window and gives the life cycle and the files opened; the orientation comes from the window scene, as the status bar says nothing under scenes. |
| Rotation | A turn has no UIKit crossfade (snapshots, black for an OpenGL view), and the app is told its new size as the view takes it, its frame at that size drawn there and then. Told before, the new layout showed a moment in the old shape; told after, the old layout turned, stretched. |
| Painter | A window not yet painted frees no textures: its painter has no GL context yet, and the app crashed as a second window opened. |
| Shaders | `canvas.Shader` programs get their own vertex shader (`shaders/user.vert`, `user_es.vert`) that tells each fragment its pixel at full precision (`varying vec2 inbPix`, gl_FragCoord's x and y). GLSL ES has gl_FragCoord medium, which an iPad's GPU keeps to 16 bits: past 2048 pixels every other pixel was read twice. Inbetween's iOS shaders read `inbPix`; its tests link them with `user_es.vert` and check both vertex shaders write it. |
| Child window | A child window gets no Fyne head (a title and a close button): the app paints its own bar. |
| 120 Hz | Frames are asked for 120 times a second (ProMotion screens), not 60. |
| Shared programs | On a phone or a tablet, `canvas.Shader` objects of one fragment source share one compiled program (`internal/painter/gl/shader_share.go`). A program was built for each `Shader.Name`, the first time it was drawn, on the thread that draws. Inbetween draws its canvas as tiles, each a Shader of its own with the same source (the picture's, a live effect's, a moved layer's, a played frame's): dozens of compiles, tens of milliseconds each on an iPad, the first time anything was done, with the screen still for all of it. Each shader keeps its own textures and its own uniforms: before one is drawn, every uniform a shader of the program has set is set to this shader's value (0 when it never gave one), as in a program of its own. Computers keep a program a name, as Fyne has it (`-tags inbshare` tries the sharing there). |

The touch IDs, UIScene and painter fixes are bugs any Fyne app on iOS hits:
worth offering to Fyne as pull requests, after which they drop out of here.

## Moving to a new Fyne

    git fetch upstream --tags          # upstream: github.com/fyne-io/fyne
    git rebase --onto vX.Y.Z v2.8.2 inbetween
    git tag vX.Y.Z-inbetween.1
    git push origin inbetween vX.Y.Z-inbetween.1

then point Inbetween's iPad build at the new tag and build the iPad app.
