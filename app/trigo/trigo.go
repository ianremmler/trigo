package main

import (
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	"image/png"
	"io/ioutil"
	"math"
	"math/rand"
	"time"

	"github.com/ianremmler/trigo"
	"golang.org/x/mobile/app"
	"golang.org/x/mobile/asset"
	"golang.org/x/mobile/event/lifecycle"
	"golang.org/x/mobile/event/paint"
	"golang.org/x/mobile/event/size"
	"golang.org/x/mobile/event/touch"
	"golang.org/x/mobile/exp/f32"
	"golang.org/x/mobile/exp/gl/glutil"
	"golang.org/x/mobile/gl"
)

const (
	cardAspRat     = 1.4
	transitionTime = 1 * time.Second
	transitionRate = 30 // fps
	charsPerRow    = 16
	stateFile      = "/data/data/org.remmler.TriGo/state"
)

var colors = [][]float32{
	{1, 0, 0, 1},
	{0, 0.75, 0, 1},
	{0, 0, 1, 1},
}

type shape struct {
	verts []float32
	buf   gl.Buffer
}

var shapes = []shape{
	{verts: squareVerts},
	{verts: triVerts},
	{verts: hexVerts},
}

type cardState int

const (
	normal cardState = iota
	selected
	invalid
	fadeOut
	fadeIn
)

type gameState int

const (
	play gameState = iota
	match
	deal
	win
	endGame
	newGame
)

var cardVerts = []float32{
	0, 0, 0,
	1, 0, 0,
	1, cardAspRat, 0,
	0, cardAspRat, 0,
}

var sec30 = float32(2 / math.Sqrt(3))

var squareVerts = []float32{
	-1, -1, 0,
	1, -1, 0,
	1, 1, 0,
	-1, 1, 0,
}

var triVerts = []float32{
	-sec30, -1, 0,
	0, 1, 0,
	sec30, -1, 0,
}

var hexVerts = []float32{
	-0.5 * sec30, -1, 0,
	-sec30, 0, 0,
	-0.5 * sec30, 1, 0,
	0.5 * sec30, 1, 0,
	sec30, 0, 0,
	0.5 * sec30, -1, 0,
}

var charVerts = []float32{
	0, 0, 0,
	1, 0, 0,
	1, 1, 0,
	0, 1, 0,
}

const cardVertShader = `
	#version 100
	uniform mat4 mat;

	attribute vec4 pos;
	void main() {
		gl_Position = mat * pos;
	}`

const cardFragShader = `
	#version 100
	precision mediump float;

	uniform vec4 color;
	uniform int shading;

	void main() {
		if (shading == 0) {
			discard;
		}
		if (shading == 1) {
			if (mod((gl_FragCoord.x + gl_FragCoord.y) / 8.0, 2.0) < 1.0) {
				discard;
			}
		}
		gl_FragColor = color;
	}`

const textVertShader = `
	#version 100
	uniform mat4 mat;
	attribute vec2 texCoords;
	varying vec2 fragTexCoord;
	attribute vec4 pos;

	void main() {
		fragTexCoord = texCoords;
		gl_Position = mat * pos;
	}`

const textFragShader = `
	#version 100
	precision mediump float;

	uniform sampler2D tex;
	uniform vec4 color;
	varying vec2 fragTexCoord;

	void main() {
		gl_FragColor = color * texture2D(tex, fragTexCoord);
	} `

var (
	ap       app.App
	siz      size.Event
	glctx    gl.Context
	fontTex  gl.Texture
	cardProg *prog
	textProg *prog

	tri       *trigo.TriGo
	field     []trigo.Card
	state     gameState
	matches   int
	deckSize  int
	candidate = map[int]struct{}{}

	transitionParam float32

	cardShape = shape{verts: cardVerts}
	charShape = shape{verts: charVerts}
	fontShape shape

	cardColor    = []float32{1, 1, 1, 1}
	selectColor  = []float32{0, 1, 1, 0.25}
	invalidColor = []float32{1, 0, 0, 0.25}
	textColor    = []float32{0, 1, 1, 1}
)

type TransitionEvent struct {
	T float32
}

type prog struct {
	p gl.Program
	u map[string]gl.Uniform
	a map[string]gl.Attrib
}

func newProg(ctx gl.Context, vertShader, fragShader string, uni []string, attrib []string) *prog {
	p := &prog{u: map[string]gl.Uniform{}, a: map[string]gl.Attrib{}}
	var err error
	p.p, err = glutil.CreateProgram(glctx, vertShader, fragShader)
	if err != nil {
		return nil
	}
	for _, name := range uni {
		p.u[name] = glctx.GetUniformLocation(p.p, name)
	}
	for _, name := range attrib {
		p.a[name] = glctx.GetAttribLocation(p.p, name)
	}
	return p
}

func main() {
	app.Main(func(a app.App) {
		ap = a
		for evt := range ap.Events() {
			switch evt := ap.Filter(evt).(type) {
			case lifecycle.Event:
				switch evt.Crosses(lifecycle.StageVisible) {
				case lifecycle.CrossOn:
					glctx, _ = evt.DrawContext.(gl.Context)
					start()
				case lifecycle.CrossOff:
					stop()
					glctx = nil
				}
			case size.Event:
				siz = evt
			case paint.Event:
				draw()
			case touch.Event:
				handleTouch(evt)
				draw()
			case TransitionEvent:
				transitionParam = evt.T
				if evt.T >= 1 { // transitionion complete
					updateState()
				}
				draw()
			}
		}
	})
}

func start() {
	rand.Seed(time.Now().UnixNano())

	setupCardProg()
	setupTextProg()

	if stateData, err := ioutil.ReadFile(stateFile); err == nil {
		tri = trigo.NewFromSavedState(stateData)
	} else {
		tri = trigo.NewStd()
		tri.Shuffle()
		tri.Deal()
	}
	field = tri.Field()
	deckSize = tri.DeckSize()
	matches = tri.MatchesFound()

	glctx.Enable(gl.BLEND)
	glctx.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)
	glctx.LineWidth(4)

	startTransition(newGame)
}

func setupCardProg() error {
	cardProg = newProg(glctx, cardVertShader, cardFragShader,
		[]string{"mat", "color", "shading"}, []string{"pos"})
	if cardProg == nil {
		return errors.New("error creating card program")
	}
	cardShape.buf = glctx.CreateBuffer()
	vertBytes := f32.Bytes(binary.LittleEndian, cardShape.verts...)
	glctx.BindBuffer(gl.ARRAY_BUFFER, cardShape.buf)
	glctx.BufferData(gl.ARRAY_BUFFER, vertBytes, gl.STATIC_DRAW)
	for i := range shapes {
		shapes[i].buf = glctx.CreateBuffer()
		vertBytes = f32.Bytes(binary.LittleEndian, shapes[i].verts...)
		glctx.BindBuffer(gl.ARRAY_BUFFER, shapes[i].buf)
		glctx.BufferData(gl.ARRAY_BUFFER, vertBytes, gl.STATIC_DRAW)
	}
	return nil
}

func setupTextProg() error {
	textProg = newProg(glctx, textVertShader, textFragShader,
		[]string{"mat", "color"}, []string{"pos", "texCoords"})
	if textProg == nil {
		return errors.New("error creating text program")
	}
	fontData, err := asset.Open("font.png")
	if err != nil {
		return err
	}
	img, err := png.Decode(fontData)
	if err != nil {
		return err
	}
	fontImg, ok := img.(*image.NRGBA)
	if !ok {
		return errors.New("invalid font format")
	}
	if bounds := fontImg.Bounds(); bounds.Max.X != 256 || bounds.Max.Y != 256 {
		return errors.New("invalid font dimensions")
	}
	fontTex = glctx.CreateTexture()
	glctx.BindTexture(gl.TEXTURE_2D, fontTex)
	glctx.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST)
	glctx.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST)
	glctx.TexImage2D(gl.TEXTURE_2D, 0, 256, 256, gl.RGBA, gl.UNSIGNED_BYTE, fontImg.Pix)

	fontVerts := []float32{}
	for i := 0; i < 16; i++ {
		t1, t0 := float32(i)/16, float32(i+1)/16
		for j := 0; j < 16; j++ {
			s0, s1 := float32(j)/16, float32(j+1)/16
			fontVerts = append(fontVerts, s0, t0, s1, t0, s1, t1, s0, t1)
		}
	}

	fontShape.verts = fontVerts
	fontShape.buf = glctx.CreateBuffer()
	texCoordBytes := f32.Bytes(binary.LittleEndian, fontShape.verts...)
	glctx.BindBuffer(gl.ARRAY_BUFFER, fontShape.buf)
	glctx.BufferData(gl.ARRAY_BUFFER, texCoordBytes, gl.STATIC_DRAW)

	charShape.buf = glctx.CreateBuffer()
	charBytes := f32.Bytes(binary.LittleEndian, charShape.verts...)
	glctx.BindBuffer(gl.ARRAY_BUFFER, charShape.buf)
	glctx.BufferData(gl.ARRAY_BUFFER, charBytes, gl.STATIC_DRAW)

	return nil
}

func stop() {
	if stateData, err := tri.State(); err == nil {
		ioutil.WriteFile(stateFile, stateData, 0644)
	}

	glctx.DeleteProgram(cardProg.p)
	glctx.DeleteBuffer(cardShape.buf)
	for i := range shapes {
		glctx.DeleteBuffer(shapes[i].buf)
	}

	glctx.DeleteProgram(textProg.p)
	glctx.DeleteBuffer(charShape.buf)
	glctx.DeleteBuffer(fontShape.buf)
	glctx.DeleteTexture(fontTex)
}

func handleTouch(evt touch.Event) {
	if evt.Type != touch.TypeEnd {
		return
	}

	switch state {
	case play:
	case endGame:
		if transitionParam >= 1 {
			startTransition(newGame)
		}
		return
	default:
		return
	}

	rows, cols := 3, len(field)/3
	fw, fh := float32(rows)*cardAspRat, float32(cols)
	w, h := fitAreaDims(fw, fh)
	s := float32(evt.X) / float32(siz.WidthPx)       // x fraction across display
	t := float32(evt.Y) / float32(siz.HeightPx)      // y fraction across display
	marginX, marginY := 0.5*(w-fw)/fw, 0.5*(h-fh)/fh // "letterbox", if any
	c := int(math.Floor(float64(s*w/fw-marginX) * float64(cols)))
	r := int(math.Floor(float64(t*h/fh-marginY) * float64(rows)))

	idx := -1
	if r >= 0 && r < rows && c >= 0 && c < cols {
		idx = 3*c + (2 - r)
	}

	if idx >= 0 && idx < len(field) && !field[idx].Blank {
		updateCandidate(idx)
	}
}

func updateCandidate(idx int) {
	if _, ok := candidate[idx]; ok {
		delete(candidate, idx)
	} else if len(candidate) < 3 {
		candidate[idx] = struct{}{}
	}
	if len(candidate) < 3 {
		return
	}
	check := []int{}
	for idx := range candidate {
		check = append(check, idx)
	}
	if !tri.IsMatch(check) {
		return
	}
	// still here... we got a match!
	newState := match
	tri.Remove(check)
	matches = tri.MatchesFound()
	tri.Deal()
	if tri.FieldMatches() == 0 {
		// we won!
		newState = win
		tri.Shuffle()
		tri.Deal()
	}
	startTransition(newState)
}

func startTransition(newState gameState) {
	if newState == play {
		return
	}

	state = newState
	transitionParam = 0.0
	go func() {
		startTime := time.Now()
		tick := time.NewTicker(time.Second / transitionRate)
		for elapsed := 0 * time.Second; elapsed < transitionTime; {
			now := <-tick.C
			elapsed = now.Sub(startTime)
			ap.Send(TransitionEvent{float32(elapsed) / float32(transitionTime)})
		}
		tick.Stop()
	}()
}

func updateState() {
	if state == play {
		return
	}

	oldFieldSize := len(field)
	field = tri.Field()
	switch state {
	case match:
		deckSize = tri.DeckSize()
		startTransition(deal)
	case win:
		matches = 0
		deckSize = tri.DeckSize()
		startTransition(endGame)
	case endGame:
	default:
		state = play
		candidate = map[int]struct{}{}
	}
	if state != deal {
		return
	}

	if len(field) > oldFieldSize {
		// add new cards to candidate just so they'll fade in
		for i := oldFieldSize; i < len(field); i++ {
			candidate[i] = struct{}{}
		}
	}
}

func mat4ToSlice(mat *f32.Mat4) []float32 {
	s := make([]float32, 4*4)
	for i := range mat {
		for j := range mat[i] {
			s[4*i+j] = mat[j][i]
		}
	}
	return s
}

func drawCard(mat *f32.Mat4, card *trigo.Card, st cardState) {
	num, clr, shp, fil := card.Attr[0], card.Attr[1], card.Attr[2], card.Attr[3]

	// card base

	glctx.UniformMatrix4fv(cardProg.u["mat"], mat4ToSlice(mat))
	glctx.BindBuffer(gl.ARRAY_BUFFER, cardShape.buf)
	glctx.EnableVertexAttribArray(cardProg.a["pos"])
	glctx.VertexAttribPointer(cardProg.a["pos"], 3, gl.FLOAT, false, 0, 0)
	glctx.Uniform1i(cardProg.u["shading"], 2)
	glctx.Uniform4fv(cardProg.u["color"], cardColor)
	glctx.DrawArrays(gl.TRIANGLE_FAN, 0, len(cardShape.verts)/3)
	glctx.DisableVertexAttribArray(cardProg.a["pos"])

	// symbols

	glctx.BindBuffer(gl.ARRAY_BUFFER, shapes[shp].buf)
	glctx.EnableVertexAttribArray(cardProg.a["pos"])
	glctx.VertexAttribPointer(cardProg.a["pos"], 3, gl.FLOAT, false, 0, 0)
	glctx.Uniform4fv(cardProg.u["color"], colors[clr])
	for i := 0; i <= num; i++ {
		shapeMat := *mat
		offset := float32(i+1) / (float32(num) + 2)
		shapeMat.Translate(&shapeMat, 0.5, offset*cardAspRat, 0)
		shapeMat.Scale(&shapeMat, 0.1, 0.1, 0)
		glctx.UniformMatrix4fv(cardProg.u["mat"], mat4ToSlice(&shapeMat))
		glctx.Uniform1i(cardProg.u["shading"], fil)
		glctx.DrawArrays(gl.TRIANGLE_FAN, 0, len(shapes[shp].verts)/3)
		glctx.Uniform1i(cardProg.u["shading"], 2)
		glctx.DrawArrays(gl.LINE_LOOP, 0, len(shapes[shp].verts)/3)
	}
	glctx.DisableVertexAttribArray(cardProg.a["pos"])

	if st == normal {
		return
	}

	// card special effects

	glctx.UniformMatrix4fv(cardProg.u["mat"], mat4ToSlice(mat))
	glctx.BindBuffer(gl.ARRAY_BUFFER, cardShape.buf)
	glctx.EnableVertexAttribArray(cardProg.a["pos"])
	glctx.VertexAttribPointer(cardProg.a["pos"], 3, gl.FLOAT, false, 0, 0)
	switch st {
	case fadeOut:
		glctx.Uniform4f(cardProg.u["color"], 0, 0, 0, transitionParam)
	case fadeIn:
		glctx.Uniform4f(cardProg.u["color"], 0, 0, 0, 1-transitionParam)
	case selected:
		glctx.Uniform4fv(cardProg.u["color"], selectColor)
	case invalid:
		glctx.Uniform4fv(cardProg.u["color"], invalidColor)
	}
	glctx.DrawArrays(gl.TRIANGLE_FAN, 0, len(cardShape.verts)/3)
	glctx.DisableVertexAttribArray(cardProg.a["pos"])
}

// fitAreaDims returns the view dimensions that will maximally contain the
// given area.
func fitAreaDims(areaWidth, areaHeight float32) (float32, float32) {
	areaAspRat := areaWidth / areaHeight
	dispAspRat := float32(siz.WidthPx) / float32(siz.HeightPx)

	width, height := areaWidth, areaHeight
	// add letterboxing to preserve aspect ratio
	if dispAspRat > areaAspRat {
		width = areaHeight * dispAspRat
	} else {
		height = areaWidth / dispAspRat
	}
	return width, height
}

func draw() {
	glctx.ClearColor(0, 0, 0, 0)
	glctx.Clear(gl.COLOR_BUFFER_BIT)

	switch state {
	case endGame:
		drawEnd()
	default:
		drawField()
	}
	ap.Publish()
}

func drawField() {
	fw, fh := float32(len(field)/3), float32(3*cardAspRat)
	w, h := fitAreaDims(fw, fh)
	mat := f32.Mat4{}
	mat.Identity()
	mat.Scale(&mat, 1.0/(0.5*w), 1.0/(0.5*h), 1)

	st := normal
	switch state {
	case win:
		st = fadeOut
	case newGame:
		st = fadeIn
	}

	glctx.UseProgram(cardProg.p)
	for i := range field {
		if field[i].Blank {
			continue
		}
		x, y := float32(i/3), cardAspRat*float32(i%3)
		cardMat := mat
		cardMat.Translate(&cardMat, x-0.5*fw, y-0.5*fh, 0)
		// shrink just a bit to separate cards
		cardMat.Translate(&cardMat, 0.5, 0.5*cardAspRat, 0)
		cardMat.Scale(&cardMat, 1.0-0.02*cardAspRat, 1.0-0.02, 1)
		cardMat.Translate(&cardMat, -0.5, -0.5*cardAspRat, 0)

		cardSt := st
		if st == normal {
			if _, ok := candidate[i]; ok {
				switch {
				case state == match:
					cardSt = fadeOut
				case state == deal:
					cardSt = fadeIn
				case len(candidate) < 3:
					cardSt = selected
				default:
					cardSt = invalid
				}
			}
		}
		drawCard(&cardMat, &field[i], cardSt)
	}

	textMat := mat
	color := append([]float32(nil), textColor...)
	switch state {
	case newGame:
		color[3] = transitionParam
	case win:
		color[3] = 1 - transitionParam
	}

	textMat.Translate(&textMat, -0.5*fw, 0.5*fh, 0)
	textMat.Scale(&textMat, w/charsPerRow, w/charsPerRow, 1)
	textMat.Translate(&textMat, 0.0, 0.5, 1)
	drawText(fmt.Sprintf("DECK: %d", deckSize), textMat, color)

	textMat = mat
	textMat.Translate(&textMat, -0.5*fw, -0.5*fh, 0)
	textMat.Scale(&textMat, w/charsPerRow, w/charsPerRow, 1)
	textMat.Translate(&textMat, 0.0, -1.5, 1)
	drawText(fmt.Sprintf("MATCHES: %d", matches), textMat, color)
}

func drawEnd() {
	msg := []string{"YOU", "DID", "IT!"}
	w, h := fitAreaDims(3.0, 3.0) // 3 chars per line, 3 lines
	mat := f32.Mat4{}
	mat.Identity()
	mat.Scale(&mat, 1.0/(0.5*w), 1.0/(0.5*h), 1)
	mat.Translate(&mat, -1.5, 0.5, 0)
	// mat.Scale(&mat, 0.5/3, 0.5/3, 1)
	for i := range msg {
		textMat := mat
		textMat.Translate(&textMat, 0, float32(-i), 0)
		color := append([]float32(nil), textColor...)
		color[3] = transitionParam
		drawText(msg[i], textMat, color)
	}
}

// drawText draws text in the position and orientation defined by mat
// each character is unit height and width (1.0 x 1.0)
func drawText(text string, mat f32.Mat4, color []float32) {
	glctx.UseProgram(textProg.p)
	glctx.BindBuffer(gl.ARRAY_BUFFER, charShape.buf)
	glctx.EnableVertexAttribArray(textProg.a["pos"])
	glctx.VertexAttribPointer(textProg.a["pos"], 3, gl.FLOAT, false, 0, 0)
	glctx.BindBuffer(gl.ARRAY_BUFFER, fontShape.buf)
	glctx.BindTexture(gl.TEXTURE_2D, fontTex)
	glctx.EnableVertexAttribArray(textProg.a["texCoords"])
	glctx.Uniform4fv(textProg.u["color"], color)
	for _, c := range text {
		if c > 255 {
			continue
		}
		glctx.VertexAttribPointer(textProg.a["texCoords"], 2, gl.FLOAT, false, 0, int(c)*32)
		glctx.UniformMatrix4fv(textProg.u["mat"], mat4ToSlice(&mat))
		glctx.DrawArrays(gl.TRIANGLE_FAN, 0, len(charShape.verts)/3)
		mat.Translate(&mat, 1.0, 0.0, 0.0)
	}
	glctx.DisableVertexAttribArray(textProg.a["pos"])
	glctx.DisableVertexAttribArray(textProg.a["texCoords"])
}
