// Package pace keeps a window loop's pace and its frames' times
// (Inbetween's: see INBETWEEN.md).
package pace

import (
	"sync"
	"time"
)

// Fyne's loop looked at events and drew 60 times a second, whatever the
// screen: a 120 or 144 Hz screen was given 60 frames, and a 59.88 Hz one
// beat against the 60 Hz tick (a frame now and then drawn a refresh late).
// The loop ticks at the screen's own rate instead (the fastest screen's,
// 60 at least), twice it where the compositor says when a frame may be
// drawn (Wayland's frame callbacks: one is caught within half a refresh).
//
// It also keeps when the frames went to the screen (the swaps: with the
// swap interval 1 a swap returns at the screen's refresh) and how far
// apart they come (the refresh as it really is: 59.88 Hz, not the 60 the
// mode says). An app that plays something (a film) is told, before each
// frame is drawn, when the frame should show (SetFrameHook): it picks what
// to draw for that moment, not for when it was last told the time, so the
// frames keep their cadence (24 a second on 60 Hz: three refreshes, then
// two, every time).

// refresh is a refresh as the screen's mode says: its own rate down to 20
// a second (a film's 24 or 25 on a TV or a projector, Windows' 23 for
// 23.976), 60 where the mode says none (0, or Windows' 1: the hardware's
// default).
func refresh(hz int) time.Duration {
	if hz < 20 {
		hz = 60
	}
	return time.Second / time.Duration(hz)
}

// Pacer keeps the frames' times.
type Pacer struct {
	mu     sync.Mutex
	hz     int           // the fastest screen's refresh (its mode)
	period time.Duration // a refresh as measured (0: not yet)
	last   time.Time     // the last swap
	hook   func(present time.Time)
	// Refreshed is when the screens' rate was read last.
	Refreshed time.Time
}

// Tick is the loop's tick: the fastest screen's refresh, twice it where
// frames are gated by the compositor.
func (p *Pacer) Tick(gated bool) time.Duration {
	p.mu.Lock()
	hz := p.hz
	p.mu.Unlock()
	hz = max(60, min(hz, 360))
	if gated {
		hz *= 2
	}
	return time.Second / time.Duration(hz)
}

// SetHz notes the fastest screen's refresh.
func (p *Pacer) SetHz(hz int) {
	p.mu.Lock()
	p.hz = hz
	p.mu.Unlock()
}

// Swapped notes a frame gone to the screen at t: a refresh measured from
// the swap before, when they come a whole number of refreshes apart (one
// to four: a film's frame drawn every second or third refresh), within a
// tenth of one and of the mode's (a late frame isn't one to measure by).
func (p *Pacer) Swapped(t time.Time) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.last.IsZero() {
		d := t.Sub(p.last)
		nominal := refresh(p.hz)
		ref := p.period
		if ref == 0 {
			ref = nominal
		}
		// (within a tenth of the screen's mode too: where swaps aren't held
		// to its refresh, no vsync, there's none to measure, and the measure
		// stays the mode's)
		near := func(v, of time.Duration) bool { return v-of < of/10 && v-of > -of/10 }
		if k := (d + ref/2) / ref; k >= 1 && k <= 4 {
			one := d / k
			if near(one, ref) && near(one, nominal) {
				if p.period == 0 {
					p.period = one
				} else {
					p.period += (one - p.period) / 16
				}
			}
		}
	}
	p.last = t
}

// Present is when a frame drawn now should show: the next refresh after
// now, from the last swap by the refresh as measured (or as the mode says).
func (p *Pacer) Present(now time.Time) time.Time {
	p.mu.Lock()
	defer p.mu.Unlock()
	period := p.period
	if period == 0 {
		period = refresh(p.hz)
	}
	if p.last.IsZero() || now.Sub(p.last) > time.Second {
		return now.Add(period)
	}
	n := now.Sub(p.last)/period + 1
	return p.last.Add(n * period)
}

// Period is a refresh of the screen as measured (or as its mode says).
func (p *Pacer) Period() time.Duration {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.period > 0 {
		return p.period
	}
	return refresh(p.hz)
}

// SetHook sets the hook told when each frame should show (nil: none).
func (p *Pacer) SetHook(fn func(present time.Time)) {
	p.mu.Lock()
	p.hook = fn
	p.mu.Unlock()
}

// Hook tells the hook, if any, when a frame drawn at now should show.
func (p *Pacer) Hook(now time.Time) {
	p.mu.Lock()
	fn := p.hook
	p.mu.Unlock()
	if fn != nil {
		fn(p.Present(now))
	}
}
