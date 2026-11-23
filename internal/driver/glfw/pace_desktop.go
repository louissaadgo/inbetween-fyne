//go:build !wasm && !test_web_driver

package glfw

import "github.com/go-gl/glfw/v3.4/glfw"

// fastestRefresh is the refresh rate of the fastest screen connected (0:
// none known).
func (d *gLDriver) fastestRefresh() int {
	hz := 0
	for _, m := range glfw.GetMonitors() {
		if mode := m.GetVideoMode(); mode != nil {
			hz = max(hz, mode.RefreshRate)
		}
	}
	return hz
}
