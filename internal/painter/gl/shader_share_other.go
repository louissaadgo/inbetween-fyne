//go:build !(android || ios || mobile || inbshare)

package gl

// shareShaderPrograms: a computer's shaders each have their program, as in
// Fyne (shader_share.go: a phone's or a tablet's share one a source).
var shareShaderPrograms = false
