//go:build (!gles && !arm && !arm64 && !android && !ios && !mobile && !test_web_driver && !wasm) || (darwin && !mobile && !ios && !wasm && !test_web_driver)

package gl

import (
	"image"
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
)

// fakeGL stands in for OpenGL: it counts the programs built and keeps each
// program's uniforms as OpenGL would, to say what a draw saw.
type fakeGL struct {
	context // (the calls a shader's draw doesn't make: nil, never called)

	shaders, programs int
	sources           map[Shader]string
	locs              map[Program]map[string]Uniform
	names             map[Uniform]string
	values            map[Program]map[string]float32
	current           Program
	textures          int
	bound             map[uint32]Texture // texture unit: the texture bound to it
	unit              uint32
	draws             []fakeDraw
}

// fakeDraw is what a draw saw: the program, its uniforms then, and the
// textures bound to the units.
type fakeDraw struct {
	program  Program
	uniforms map[string]float32
	bound    map[uint32]Texture
}

func newFakeGL() *fakeGL {
	return &fakeGL{sources: map[Shader]string{}, locs: map[Program]map[string]Uniform{}, names: map[Uniform]string{},
		values: map[Program]map[string]float32{}, bound: map[uint32]Texture{}}
}

func (f *fakeGL) CreateShader(uint32) Shader           { f.shaders++; return Shader(f.shaders) }
func (f *fakeGL) ShaderSource(s Shader, src string)    { f.sources[s] = src }
func (f *fakeGL) CompileShader(Shader)                 {}
func (f *fakeGL) GetShaderInfoLog(Shader) string       { return "" }
func (f *fakeGL) GetProgramInfoLog(Program) string     { return "" }
func (f *fakeGL) GetProgrami(Program, uint32) int      { return 1 }
func (f *fakeGL) AttachShader(Program, Shader)         {}
func (f *fakeGL) LinkProgram(Program)                  {}
func (f *fakeGL) GetError() uint32                     { return 0 }
func (f *fakeGL) CreateBuffer() Buffer                 { return 1 }
func (f *fakeGL) BindBuffer(uint32, Buffer)            {}
func (f *fakeGL) BufferData(uint32, []float32, uint32) {}
func (f *fakeGL) BufferSubData(uint32, []float32)      {}
func (f *fakeGL) BlendFunc(uint32, uint32)             {}
func (f *fakeGL) EnableVertexAttribArray(Attribute)    {}
func (f *fakeGL) TexParameteri(uint32, uint32, int32)  {}
func (f *fakeGL) DeleteTexture(Texture)                {}
func (f *fakeGL) ActiveTexture(u uint32)               { f.unit = u }
func (f *fakeGL) BindTexture(_ uint32, t Texture)      { f.bound[f.unit] = t }
func (f *fakeGL) CreateTexture() Texture               { f.textures++; return Texture(f.textures) }
func (f *fakeGL) UseProgram(p Program)                 { f.current = p }
func (f *fakeGL) Uniform1f(u Uniform, v float32)       { f.values[f.current][f.names[u]] = v }
func (f *fakeGL) Uniform1i(u Uniform, v int32)         { f.values[f.current][f.names[u]] = float32(v) }
func (f *fakeGL) Uniform2f(Uniform, float32, float32)  {}
func (f *fakeGL) GetAttribLocation(Program, string) Attribute {
	return 0
}
func (f *fakeGL) Uniform4f(Uniform, float32, float32, float32, float32) {}
func (f *fakeGL) VertexAttribPointerWithOffset(Attribute, int, uint32, bool, int, int) {
}
func (f *fakeGL) TexImage2D(uint32, int, int, int, uint32, uint32, []uint8) {}

func (f *fakeGL) GetShaderi(s Shader, _ uint32) int {
	if strings.Contains(f.sources[s], "BAD") {
		return glFalse
	}
	return 1
}

func (f *fakeGL) CreateProgram() Program {
	f.programs++
	p := Program(f.programs)
	f.locs[p], f.values[p] = map[string]Uniform{}, map[string]float32{}
	return p
}

func (f *fakeGL) GetUniformLocation(p Program, name string) Uniform {
	if u, ok := f.locs[p][name]; ok {
		return u
	}
	u := Uniform(len(f.names) + 1)
	f.locs[p][name], f.names[u] = u, name
	return u
}

func (f *fakeGL) DrawArrays(uint32, int, int) {
	d := fakeDraw{program: f.current, uniforms: map[string]float32{}, bound: map[uint32]Texture{}}
	for k, v := range f.values[f.current] {
		d.uniforms[k] = v
	}
	for k, v := range f.bound {
		d.bound[k] = v
	}
	f.draws = append(f.draws, d)
}

// last is what the last draw saw.
func (f *fakeGL) last() fakeDraw { return f.draws[len(f.draws)-1] }

func sharePainter(t *testing.T, share bool) (*painter, *fakeGL) {
	t.Helper()
	was := shareShaderPrograms
	shareShaderPrograms = share
	t.Cleanup(func() { shareShaderPrograms = was })
	f := newFakeGL()
	return &painter{ctx: f, pixScale: 1, texScale: 1}, f
}

func testShader(name, src string, u map[string]float32) *canvas.Shader {
	s := canvas.NewShader(name, []byte(src), []byte(src))
	s.Uniforms = u
	s.Resize(fyne.NewSize(10, 10))
	return s
}

func (p *painter) drawTest(s *canvas.Shader) {
	p.drawShader(s, fyne.NewPos(0, 0), fyne.NewSize(100, 100))
}

// Shaders of one source share one program, each drawn with its own
// uniforms: the values it gave, and 0 for those only another shader of the
// program gave (as in a program of its own); a uniform a shader no longer
// names keeps the value it last gave it.
func TestShadersOfOneSourceShareAProgram(t *testing.T) {
	p, f := sharePainter(t, true)
	a := testShader("tile.a", "void main(){}", map[string]float32{"u": 1, "v": 2})
	b := testShader("tile.b", "void main(){}", map[string]float32{"u": 5})
	p.drawTest(a)
	if f.programs != 1 {
		t.Fatalf("one shader drawn: %d programs built", f.programs)
	}
	p.drawTest(b)
	if f.programs != 1 {
		t.Fatalf("a second shader of the same source built a program of its own (%d)", f.programs)
	}
	if d := f.last(); d.uniforms["u"] != 5 || d.uniforms["v"] != 0 {
		t.Errorf("the second shader was drawn with u=%v v=%v, want its own 5 and (never given) 0", d.uniforms["u"], d.uniforms["v"])
	}
	p.drawTest(a)
	if d := f.last(); d.uniforms["u"] != 1 || d.uniforms["v"] != 2 {
		t.Errorf("the first shader, drawn again, saw u=%v v=%v, want its own 1 and 2", d.uniforms["u"], d.uniforms["v"])
	}
	// a uniform it no longer names: as it last gave it
	delete(a.Uniforms, "v")
	p.drawTest(b)
	p.drawTest(a)
	if d := f.last(); d.uniforms["v"] != 2 {
		t.Errorf("a uniform a shader no longer names is %v, want what it last gave it (2)", d.uniforms["v"])
	}
	// another source: another program
	c := testShader("ring", "void main(){ /* ring */ }", map[string]float32{"u": 9})
	p.drawTest(c)
	if f.programs != 2 {
		t.Errorf("a shader of another source: %d programs, want 2", f.programs)
	}
	if d := f.last(); d.program == f.draws[0].program {
		t.Error("a shader of another source was drawn with the first's program")
	}
	p.drawTest(a)
	if d := f.last(); d.uniforms["u"] != 1 || d.program != f.draws[0].program {
		t.Errorf("after another source's shader, the first saw u=%v in program %v", d.uniforms["u"], d.program)
	}
}

// Each shader keeps its own textures on a shared program.
func TestSharedProgramKeepsEachShadersTextures(t *testing.T) {
	p, f := sharePainter(t, true)
	a := testShader("tile.a", "void main(){}", nil)
	b := testShader("tile.b", "void main(){}", nil)
	a.Textures = map[string]image.Image{"tex": image.NewRGBA(image.Rect(0, 0, 2, 2))}
	b.Textures = map[string]image.Image{"tex": image.NewRGBA(image.Rect(0, 0, 2, 2))}
	p.drawTest(a)
	ta := f.last().bound[texture0]
	p.drawTest(b)
	tb := f.last().bound[texture0]
	if ta == tb || ta == 0 || tb == 0 {
		t.Fatalf("the two shaders drew with textures %v and %v: each has its own", ta, tb)
	}
	p.drawTest(a)
	if got := f.last().bound[texture0]; got != ta {
		t.Errorf("the first shader, drawn again, had texture %v bound, want its own (%v)", got, ta)
	}
	if f.textures != 2 {
		t.Errorf("%d textures made for two shaders drawn three times, want 2 (none made again)", f.textures)
	}
}

// A source that doesn't compile is tried once, whatever the number of
// shaders that have it.
func TestSharedProgramFailsOnce(t *testing.T) {
	p, f := sharePainter(t, true)
	for _, name := range []string{"x.1", "x.2", "x.3"} {
		p.drawTest(testShader(name, "BAD", nil))
	}
	if len(f.draws) != 0 {
		t.Errorf("a shader that didn't compile was drawn")
	}
	if f.shaders > 2 {
		t.Errorf("a source that doesn't compile was compiled %d times (shaders made), want once", f.shaders)
	}
}

// Where programs aren't shared (a computer), each shader has its own, as
// in Fyne.
func TestShadersUnsharedHaveTheirOwnPrograms(t *testing.T) {
	p, f := sharePainter(t, false)
	a := testShader("tile.a", "void main(){}", map[string]float32{"u": 1, "v": 2})
	b := testShader("tile.b", "void main(){}", map[string]float32{"u": 5})
	p.drawTest(a)
	p.drawTest(b)
	p.drawTest(a)
	if f.programs != 2 {
		t.Fatalf("two shaders, unshared: %d programs, want 2", f.programs)
	}
	if d := f.last(); d.uniforms["u"] != 1 || d.uniforms["v"] != 2 {
		t.Errorf("unshared, the first shader saw u=%v v=%v", d.uniforms["u"], d.uniforms["v"])
	}
}
