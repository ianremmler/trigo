package main

import (
	"encoding/binary"
	"io/ioutil"
	"log"
	"math"
	"math/rand"
	"time"

	"github.com/ianremmler/trigo"
	"golang.org/x/mobile/app"
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
	transitionTime = 1 // seconds
	stateFile      = "/data/data/org.remmler.TriGo/state"
)

var (
	tri       *trigo.TriGo
	field     []trigo.Card
	state     gameState
	candidate = map[int]struct{}{}

	transitionStart time.Time
	transitionParam float32

	program  gl.Program
	position gl.Attrib
	color    gl.Uniform
	shading  gl.Uniform
	mvMat    gl.Uniform

	cardShape    = shape{verts: cardVerts}
	cardColor    = []float32{1, 1, 1, 1}
	selectColor  = []float32{0, 1, 1, 0.25}
	invalidColor = []float32{1, 0, 0, 0.25}

	siz size.Event
)

var colors = [][]float32{
	{1, 0, 0, 1},
	{0, 0.75, 0, 1},
	{0, 0, 1, 1},
}

var shapes = []shape{
	{verts: squareVerts},
	{verts: triVerts},
	{verts: hexVerts},
}

type shape struct {
	verts []float32
	buf   gl.Buffer
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
	newGame
)

func main() {
	app.Main(func(ap app.App) {
		for evt := range ap.Events() {
			switch evt := app.Filter(evt).(type) {
			case lifecycle.Event:
				switch evt.Crosses(lifecycle.StageVisible) {
				case lifecycle.CrossOn:
					start()
				case lifecycle.CrossOff:
					stop()
				}
			case size.Event:
				siz = evt
			case paint.Event:
				draw()
				ap.EndPaint(evt)
			case touch.Event:
				handleTouch(evt)
			}
		}
	})
}

func start() {
	rand.Seed(time.Now().UnixNano())
	if stateData, err := ioutil.ReadFile(stateFile); err == nil {
		tri = trigo.NewFromSavedState(stateData)
	} else {
		tri = trigo.NewStd()
		tri.Shuffle()
		tri.Deal()
	}
	field = tri.Field()

	var err error
	program, err = glutil.CreateProgram(vertShader, fragShader)
	if err != nil {
		log.Fatalln(err)
	}

	gl.Enable(gl.BLEND)
	gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)
	gl.LineWidth(4)

	cardShape.buf = gl.CreateBuffer()
	vertBytes := f32.Bytes(binary.LittleEndian, cardShape.verts...)
	gl.BindBuffer(gl.ARRAY_BUFFER, cardShape.buf)
	gl.BufferData(gl.ARRAY_BUFFER, vertBytes, gl.STATIC_DRAW)
	for i := range shapes {
		shapes[i].buf = gl.CreateBuffer()
		vertBytes = f32.Bytes(binary.LittleEndian, shapes[i].verts...)
		gl.BindBuffer(gl.ARRAY_BUFFER, shapes[i].buf)
		gl.BufferData(gl.ARRAY_BUFFER, vertBytes, gl.STATIC_DRAW)
	}

	position = gl.GetAttribLocation(program, "position")
	color = gl.GetUniformLocation(program, "color")
	shading = gl.GetUniformLocation(program, "shading")
	mvMat = gl.GetUniformLocation(program, "mvMat")

	startTransition(newGame)
}

func stop() {
	if stateData, err := tri.State(); err == nil {
		ioutil.WriteFile(stateFile, stateData, 0644)
	}
	gl.DeleteProgram(program)
	gl.DeleteBuffer(cardShape.buf)
	for i := range shapes {
		gl.DeleteBuffer(shapes[i].buf)
	}
}

func handleTouch(evt touch.Event) {
	if evt.Type != touch.TypeEnd || state != play {
		return
	}

	w, h, fw, fh := viewDims()
	rows, cols := 3, len(field)/3
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
	tri.Deal()
	if tri.NumMatches() == 0 {
		// we won!  play again...
		newState = win
		tri.Shuffle()
		tri.Deal()
	}
	startTransition(newState)
}

func startTransition(newState gameState) {
	state = newState
	transitionStart = time.Now()
	transitionParam = 0.0
}

func updateState() {
	if state == play {
		return
	}

	delta := float32(time.Now().Sub(transitionStart).Seconds())
	transitionParam = delta / transitionTime
	if transitionParam < 1 {
		return
	}

	// transition time's up
	oldFieldSize := len(field)
	field = tri.Field()
	switch state {
	case match:
		startTransition(deal)
	case win:
		startTransition(newGame)
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
	s := []float32{}
	// 	for i := range mat {
	// 		s = append(s, mat[i][:]...)
	// 	}
	for i := range mat {
		for j := range mat[i] {
			s = append(s, mat[j][i])
		}
	}
	return s
}

func drawCard(mat *f32.Mat4, card *trigo.Card, st cardState) {
	num, clr, shp, fil := card.Attr[0], card.Attr[1], card.Attr[2], card.Attr[3]

	// card base
	gl.UniformMatrix4fv(mvMat, mat4ToSlice(mat))

	gl.BindBuffer(gl.ARRAY_BUFFER, cardShape.buf)
	gl.EnableVertexAttribArray(position)
	gl.VertexAttribPointer(position, 3, gl.FLOAT, false, 0, 0)
	gl.Uniform1i(shading, 2)
	gl.Uniform4fv(color, cardColor)
	gl.DrawArrays(gl.TRIANGLE_FAN, 0, len(cardShape.verts)/3)
	gl.DisableVertexAttribArray(position)

	// symbols
	gl.BindBuffer(gl.ARRAY_BUFFER, shapes[shp].buf)
	gl.EnableVertexAttribArray(position)
	gl.VertexAttribPointer(position, 3, gl.FLOAT, false, 0, 0)
	gl.Uniform4fv(color, colors[clr])
	for i := 0; i <= num; i++ {
		shapeMat := *mat
		offset := float32(i+1) / (float32(num) + 2)
		shapeMat.Translate(&shapeMat, 0.5, offset*cardAspRat, 0)
		shapeMat.Scale(&shapeMat, 0.1, 0.1, 0)
		gl.UniformMatrix4fv(mvMat, mat4ToSlice(&shapeMat))
		gl.Uniform1i(shading, fil)
		gl.DrawArrays(gl.TRIANGLE_FAN, 0, len(shapes[shp].verts)/3)
		gl.Uniform1i(shading, 2)
		gl.DrawArrays(gl.LINE_LOOP, 0, len(shapes[shp].verts)/3)
	}
	gl.DisableVertexAttribArray(position)

	if st == normal {
		return
	}

	// card special effects
	gl.UniformMatrix4fv(mvMat, mat4ToSlice(mat))

	gl.BindBuffer(gl.ARRAY_BUFFER, cardShape.buf)
	gl.EnableVertexAttribArray(position)
	gl.VertexAttribPointer(position, 3, gl.FLOAT, false, 0, 0)
	switch st {
	case fadeOut:
		gl.Uniform4f(color, 0, 0, 0, transitionParam)
	case fadeIn:
		gl.Uniform4f(color, 0, 0, 0, 1-transitionParam)
	case selected:
		gl.Uniform4fv(color, selectColor)
	case invalid:
		gl.Uniform4fv(color, invalidColor)
	}
	gl.DrawArrays(gl.TRIANGLE_FAN, 0, len(cardShape.verts)/3)
	gl.DisableVertexAttribArray(position)
}

// viewDims returns the display width/height and field width/height in units
// based on the width of a card.
func viewDims() (float32, float32, float32, float32) {
	cols := len(field) / 3
	fieldWidth, fieldHeight := float32(cols), float32(3*cardAspRat)
	fieldAspRat := fieldWidth / fieldHeight
	dispAspRat := float32(siz.WidthPx) / float32(siz.HeightPx)

	width, height := fieldWidth, fieldHeight
	// add letterboxing to preserve aspect ratio
	if dispAspRat > fieldAspRat {
		width = fieldHeight * dispAspRat
	} else {
		height = fieldWidth / dispAspRat
	}
	return width, height, fieldWidth, fieldHeight
}

func draw() {
	updateState()

	gl.ClearColor(0, 0, 0, 1)
	gl.Clear(gl.COLOR_BUFFER_BIT)
	gl.UseProgram(program)

	w, h, fw, fh := viewDims()
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
}

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

const vertShader = `
	#version 100
	uniform mat4 mvMat;

	attribute vec4 position;
	void main() {
		gl_Position = mvMat * position;
	}`

const fragShader = `
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