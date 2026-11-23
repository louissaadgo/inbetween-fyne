package gl

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
)

// Inbetween's: canvas.Shader objects of one fragment source share one
// program (where shareShaderPrograms is on: shader_share_mobile.go).
//
// A program was built for each Shader.Name. Inbetween draws its canvas as
// tiles, each a Shader object of its own (its own textures and uniforms)
// with the same source: the tiles of the picture, of a live effect, of a
// layer being moved, of a frame played. Each was compiled and linked the
// first time it was drawn, on the thread that draws: on an iPad tens of
// milliseconds each, dozens of them the first time anything was done (the
// app opened, a slider dragged, a layer moved, a film played), with the
// screen still for all of it. Compiled once, a source's program is used by
// every shader that has it: the first costs what it did, the rest nothing.
//
// What is a shader's own stays its own: its textures (shaderState.textures)
// and its uniforms. A program keeps the uniforms last set in it, so before a
// shader is drawn every uniform any shader of the program has set is set to
// this shader's value for it: the last it gave, or 0 when it never gave
// one, as in a program of its own (setSharedUniforms). The values a program
// has are remembered with it (programState.uniforms), so only those that
// differ from the shader drawn before are sent.

// sharedProgram is the program the shaders of one fragment source share.
type sharedProgram struct {
	program programState
	valid   bool                // false: the source didn't compile (not tried again)
	known   map[string]struct{} // the uniforms any of its shaders has set
}

// sharedShaderProgram gives shader its state (new: its name has none yet)
// on the program of its source, built the first time a shader has it.
func (p *painter) sharedShaderProgram(shader *canvas.Shader) (*shaderState, bool) {
	if p.shaderShared == nil {
		p.shaderShared = make(map[string]*sharedProgram)
	}
	frag := userShaderFragment(shader)
	sp := p.shaderShared[string(frag)]
	if sp == nil {
		sp = &sharedProgram{known: make(map[string]struct{})}
		ref, err := p.createProgramFromSource(userShaderVertexSource(), frag)
		if err != nil {
			fyne.LogError("Failed to compile shader "+shader.Name, err)
		} else {
			sp.valid = true
			sp.program = programState{
				ref:        ref,
				buff:       p.createBuffer(16),
				uniforms:   make(map[string]*uniformState),
				attributes: make(map[string]Attribute),
			}
		}
		p.shaderShared[string(frag)] = sp // (a failure too: not compiled again for each shader of the source)
	}
	state := &shaderState{valid: sp.valid}
	if sp.valid {
		state.program, state.shared, state.sets = sp.program, sp, make(map[string]float32)
	}
	p.shaderPrograms[shader.Name] = state
	return state, state.valid
}

// setSharedUniforms sets the program's uniforms as shader has them: each
// uniform a shader of the program has ever set, to the value this shader
// last gave it (0: it never gave one).
func (p *painter) setSharedUniforms(state *shaderState, shader *canvas.Shader) {
	sp := state.shared
	for name, v := range shader.Uniforms {
		state.sets[name] = v
		sp.known[name] = struct{}{}
	}
	for name := range sp.known {
		p.SetUniform1f(state.program, name, state.sets[name])
	}
}
