package glfw

import "time"

// The loop's pace and the frames' times are kept by package pace
// (Inbetween's: the loop ticks at the screen's rate, and an app is told
// when each frame should show).

// SetFrameHook has fn told, on the main goroutine before each frame the
// loop may draw, when that frame should show (nil: no more). What it
// changes is drawn in that frame. (Inbetween: a film played picks its
// frame for that moment.)
func (d *gLDriver) SetFrameHook(fn func(present time.Time)) { d.pace.SetHook(fn) }

// FramePeriod is a refresh of the screen as measured from the frames drawn
// (as its mode says until some are).
func (d *gLDriver) FramePeriod() time.Duration { return d.pace.Period() }
