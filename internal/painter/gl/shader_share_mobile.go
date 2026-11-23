//go:build android || ios || mobile || inbshare

package gl

// shareShaderPrograms: on a phone or a tablet (and with -tags inbshare, to
// try it on a computer), shaders of one source share one program
// (shader_share.go).
var shareShaderPrograms = true
