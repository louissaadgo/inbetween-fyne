//go:build !wasm && !test_web_driver && !no_glfw && !mobile

package glfw

import (
	"testing"

	"github.com/go-gl/glfw/v3.4/glfw"

	"fyne.io/fyne/v2"
)

// A modifier key held down repeats (Windows repeats Shift, Ctrl and Alt):
// each repeat keeps it held. Taken for a release, the driver's modifiers
// said Shift was up half a second after it went down, while it still was.
func TestDesktopModifierCorrectedRepeatKeepsTheKeyHeld(t *testing.T) {
	for _, tt := range []struct {
		name   string
		mods   glfw.ModifierKey
		key    glfw.Key
		action glfw.Action
		want   fyne.KeyModifier
	}{
		{"Shift down", 0, glfw.KeyLeftShift, glfw.Press, fyne.KeyModifierShift},
		{"Shift repeating", glfw.ModShift, glfw.KeyLeftShift, glfw.Repeat, fyne.KeyModifierShift},
		{"Shift repeating, its mods without it (X11)", 0, glfw.KeyLeftShift, glfw.Repeat, fyne.KeyModifierShift},
		{"Ctrl repeating with Shift held", glfw.ModShift | glfw.ModControl, glfw.KeyLeftControl, glfw.Repeat, fyne.KeyModifierShift | fyne.KeyModifierControl},
		{"Shift up", glfw.ModShift, glfw.KeyLeftShift, glfw.Release, 0},
		{"A repeating, Shift held", glfw.ModShift, glfw.KeyA, glfw.Repeat, fyne.KeyModifierShift},
	} {
		if got := desktopModifierCorrected(tt.mods, tt.key, tt.action); got != tt.want {
			t.Errorf("%s: %v, want %v", tt.name, got, tt.want)
		}
	}
}
