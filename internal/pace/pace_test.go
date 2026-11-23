package pace

import (
	"testing"
	"time"
)

// The loop ticks at the fastest screen's rate (60 at least, at most 360),
// twice it where the compositor gates the frames.
func TestTick(t *testing.T) {
	var p Pacer
	for _, c := range []struct {
		hz    int
		gated bool
		want  time.Duration
	}{
		{0, false, time.Second / 60}, {60, false, time.Second / 60}, {144, false, time.Second / 144},
		{60, true, time.Second / 120}, {120, true, time.Second / 240}, {1000, false, time.Second / 360},
	} {
		p.SetHz(c.hz)
		if got := p.Tick(c.gated); got != c.want {
			t.Errorf("%d Hz gated %v: a tick of %v, want %v", c.hz, c.gated, got, c.want)
		}
	}
}

// A refresh is measured from the swaps (59.88 Hz, not the mode's 60), a
// swap far from a refresh's apart left out; a frame drawn shows at the
// next refresh after now, by it.
func TestPresent(t *testing.T) {
	var p Pacer
	p.SetHz(60)
	t0 := time.Unix(1000, 0)
	real := time.Second * 1000 / 59880 // (59.88 Hz)
	for i := 0; i < 200; i++ {
		p.Swapped(t0.Add(time.Duration(i) * real))
	}
	p.Swapped(t0.Add(200*real + 3*time.Millisecond)) // (a late one: left out of the measure)
	last := t0.Add(200*real + 3*time.Millisecond)
	if d := p.Period() - real; d < -50*time.Microsecond || d > 50*time.Microsecond {
		t.Fatalf("a refresh measured %v, it is %v", p.Period(), real)
	}
	now := last.Add(real / 3)
	got := p.Present(now)
	if got.Sub(last) != p.Period() {
		t.Fatalf("a frame drawn a third of a refresh after the swap shows %v after it, want %v", got.Sub(last), p.Period())
	}
	if got = p.Present(last.Add(p.Period() * 5 / 2)); got.Sub(last) != 3*p.Period() {
		t.Fatalf("two and a half refreshes on: shows %v after the swap", got.Sub(last))
	}
	var told time.Time
	p.SetHook(func(at time.Time) { told = at })
	p.Hook(now)
	if !told.Equal(p.Present(now)) {
		t.Fatal("the hook isn't told when the frame shows")
	}
	p.SetHook(nil)
	told = time.Time{}
	p.Hook(now)
	if !told.IsZero() {
		t.Fatal("a hook let go of is told")
	}
	// no swap for long: a refresh from now
	if got := p.Present(last.Add(2 * time.Second)); got.Sub(last.Add(2*time.Second)) != p.Period() {
		t.Fatalf("long after the last swap: %v", got.Sub(last.Add(2*time.Second)))
	}
}

// Swaps every second or third refresh (a film's 24 frames on 60 Hz) still
// measure a refresh.
func TestPeriodFromCadence(t *testing.T) {
	var p Pacer
	p.SetHz(60)
	real := time.Second * 1000 / 59880
	at := time.Unix(50, 0)
	for i := 0; i < 300; i++ {
		p.Swapped(at)
		at = at.Add(real * time.Duration(2+i%2))
	}
	if d := p.Period() - real; d < -50*time.Microsecond || d > 50*time.Microsecond {
		t.Fatalf("a refresh measured %v, it is %v", p.Period(), real)
	}
}

// A TV's or a projector's film rate is a refresh measured by its own (24,
// 25, Windows' 23 for 23.976), not held to 30 a second at least; a mode of
// no rate (0, Windows' 1) is taken as 60.
func TestPeriodOfFilmScreens(t *testing.T) {
	for _, c := range []struct {
		hz   int
		real time.Duration
	}{
		{24, time.Second * 1001 / 24000}, // (23.976 Hz)
		{23, time.Second * 1001 / 24000}, // (Windows' 23.976)
		{24, time.Second / 24},
		{25, time.Second / 25},
		{0, time.Second * 1000 / 59880},
		{1, time.Second * 1000 / 59880},
	} {
		var p Pacer
		p.SetHz(c.hz)
		if c.hz < 20 && p.Period() != time.Second/60 {
			t.Errorf("a mode of %d Hz: a refresh of %v before any swap, want 60 Hz's", c.hz, p.Period())
		}
		at := time.Unix(70, 0)
		for i := 0; i < 200; i++ {
			p.Swapped(at)
			at = at.Add(c.real)
		}
		if d := p.Period() - c.real; d < -50*time.Microsecond || d > 50*time.Microsecond {
			t.Errorf("a mode of %d Hz: a refresh measured %v, it is %v", c.hz, p.Period(), c.real)
		}
	}
}

// Swaps at any time (no vsync: nothing holds them to a refresh) leave the
// measure within a tenth of the screen's mode.
func TestPeriodWithoutVsync(t *testing.T) {
	var p Pacer
	p.SetHz(60)
	at := time.Unix(10, 0)
	for i := 0; i < 2000; i++ {
		p.Swapped(at)
		at = at.Add(time.Duration(20+(i*37)%23) * time.Millisecond) // (24 to 42 ms apart, any)
	}
	nominal := time.Second / 60
	if d := p.Period() - nominal; d > nominal/10 || d < -nominal/10 {
		t.Fatalf("a refresh measured %v without vsync, the mode's %v", p.Period(), nominal)
	}
}
