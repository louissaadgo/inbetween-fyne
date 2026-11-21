#version 110
attribute vec2 vert;
attribute vec2 normal;
uniform vec2 frame;
varying vec2 inbPix;
void main() {
	vec2 p = vert + normal;
	gl_Position = vec4(p, 0.0, 1.0);
	inbPix = (p * 0.5 + 0.5) * frame;
}
