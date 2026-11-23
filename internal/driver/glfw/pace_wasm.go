//go:build wasm || test_web_driver

package glfw

func (d *gLDriver) fastestRefresh() int { return 60 }
