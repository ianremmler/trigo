package main

import (
	"encoding/binary"
	"log"
	"math"
	"math/rand"
	"time"

	"github.com/ianremmler/setgo"
	"golang.org/x/mobile/app"
	"golang.org/x/mobile/event"
	"golang.org/x/mobile/f32"
	"golang.org/x/mobile/geom"
	"golang.org/x/mobile/gl"
	"golang.org/x/mobile/gl/glutil"
)

const (
	cardAspRat = 1.4
)

var (
	set   *setgo.SetGo
	field []setgo.Card

	candidate = map[int]struct{}{}

	program  gl.Program
	position gl.Attrib
	color    gl.Uniform
	shading  gl.Uniform
	mvMat    gl.Uniform

	cardShape = shape{verts: cardVerts}
	cardColor = []float32{1, 1, 1, 1}
)

var colors = [][]float32{
	{1.0, 0.0, 0.0, 1.0},
	{0.0, 0.75, 0.0, 1.0},
	{0.0, 0.0, 1.0, 1.0},
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

type state int

const (
	normal state = iota
	selected
	invalid
)

func main() {
	app.Run(app.Callbacks{
		Start: start,
		Stop:  stop,
		Draw:  draw,
		Touch: touch,
	})
}

func start() {
	rand.Seed(time.Now().UnixNano())
	set = setgo.NewStd()
	set.Deal()

	var err error
	program, err = glutil.CreateProgram(vertShader, fragShader)
	if err != nil {
		log.Printf("error creating GL program: %v", err)
		return
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

	field = set.Field()
}

func stop() {
	gl.DeleteProgram(program)
	gl.DeleteBuffer(cardShape.buf)
	for i := range shapes {
		gl.DeleteBuffer(shapes[i].buf)
	}
}

func touch(evt event.Touch) {
	if evt.Type != event.TouchEnd {
		return
	}

	w, h, fw, fh := viewDims()
	dw, dh := float32(geom.Width), float32(geom.Height)
	tx, ty := float32(evt.Loc.X), float32(evt.Loc.Y)

	rows, cols := 3, len(field)/3
	c := int(float32(cols) * (tx/dw*w/fw - 0.5*(w-fw)/fw))
	r := int(float32(rows) * (ty/dh*h/fh - 0.5*(h-fh)/fh))

	idx := -1
	if r >= 0 && r < rows && c >= 0 && c < cols {
		idx = 3*c + (2 - r)
	}

	if idx >= 0 {
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
	if !set.IsSet(check) {
		return
	}
	// still here... we got a set!
	candidate = map[int]struct{}{}
	set.Remove(check)
	set.Deal()
	if set.NumSets() == 0 {
		set.Shuffle()
		set.Deal()
	}
	field = set.Field()
}

func drawCard(mat *f32.Mat4, card *setgo.Card, st state) {
	num, clr, shp, fil := card.Attr[0], card.Attr[1], card.Attr[2], card.Attr[3]

	mvMat.WriteMat4(mat)

	gl.BindBuffer(gl.ARRAY_BUFFER, cardShape.buf)
	gl.EnableVertexAttribArray(position)
	gl.VertexAttribPointer(position, 3, gl.FLOAT, false, 0, 0)
	gl.Uniform1i(shading, 2)
	gl.Uniform4fv(color, cardColor)
	gl.DrawArrays(gl.TRIANGLE_FAN, 0, len(cardShape.verts)/3)
	gl.DisableVertexAttribArray(position)

	gl.BindBuffer(gl.ARRAY_BUFFER, shapes[shp].buf)
	gl.EnableVertexAttribArray(position)
	gl.VertexAttribPointer(position, 3, gl.FLOAT, false, 0, 0)
	gl.Uniform4fv(color, colors[clr])

	for i := 0; i <= num; i++ {
		shapeMat := *mat
		offset := float32(i+1) / (float32(num) + 2)
		shapeMat.Translate(&shapeMat, 0.5, offset*cardAspRat, 0)
		shapeMat.Scale(&shapeMat, 0.1, 0.1, 0)
		mvMat.WriteMat4(&shapeMat)

		gl.Uniform1i(shading, fil)
		gl.DrawArrays(gl.TRIANGLE_FAN, 0, len(shapes[shp].verts)/3)
		gl.Uniform1i(shading, 2)
		gl.DrawArrays(gl.LINE_LOOP, 0, len(shapes[shp].verts)/3)
	}
	gl.DisableVertexAttribArray(position)

	if st == normal {
		return
	}

	mvMat.WriteMat4(mat)
	gl.BindBuffer(gl.ARRAY_BUFFER, cardShape.buf)
	gl.EnableVertexAttribArray(position)
	gl.VertexAttribPointer(position, 3, gl.FLOAT, false, 0, 0)
	switch st {
	case selected:
		gl.Uniform4f(color, 0, 1, 1, 0.25)
	case invalid:
		gl.Uniform4f(color, 1, 0, 0, 0.25)
	}
	gl.DrawArrays(gl.TRIANGLE_FAN, 0, len(cardShape.verts)/3)
	gl.DisableVertexAttribArray(position)
}

func viewDims() (float32, float32, float32, float32) {
	cols := len(field) / 3
	fieldAspRat := float32(cols) / (3 * cardAspRat)
	dispAspRat := float32(geom.Width / geom.Height)

	fieldWidth, fieldHeight := float32(cols), float32(3*cardAspRat)
	width, height := fieldWidth, fieldHeight
	if dispAspRat > fieldAspRat {
		width = fieldHeight * dispAspRat
	} else {
		height = fieldWidth / dispAspRat
	}
	return width, height, fieldWidth, fieldHeight
}

func viewMat(w, h, fw, fh float32) f32.Mat4 {
	mat := f32.Mat4{}
	mat.Identity()
	mat.Scale(&mat, 1.0/(0.5*w), 1.0/(0.5*h), 1)
	return mat
}

func draw() {
	gl.ClearColor(0, 0, 0, 1)
	gl.Clear(gl.COLOR_BUFFER_BIT)
	gl.UseProgram(program)

	w, h, fw, fh := viewDims()
	mat := viewMat(w, h, fw, fh)
	for i := range field {
		x, y := float32(i/3), cardAspRat*float32(i%3)
		cardMat := mat
		cardMat.Translate(&cardMat, x-0.5*fw, y-0.5*fh, 0)
		// shrink just a bit to separate cards
		cardMat.Translate(&cardMat, 0.5, 0.5*cardAspRat, 0)
		cardMat.Scale(&cardMat, 1.0-0.02*cardAspRat, 1.0-0.02, 1)
		cardMat.Translate(&cardMat, -0.5, -0.5*cardAspRat, 0)

		st := normal
		if _, ok := candidate[i]; ok {
			if len(candidate) < 3 {
				st = selected
			} else {
				st = invalid
			}
		}
		drawCard(&cardMat, &field[i], st)
	}
}

var cardVerts = []float32{
	0, 0, 0,
	1, 0, 0,
	1, cardAspRat, 0,
	0, cardAspRat, 0,
}

var sin60 = float32(math.Sqrt(3) / 2)

var squareVerts = []float32{
	-sin60, -sin60, 0,
	sin60, -sin60, 0,
	sin60, sin60, 0,
	-sin60, sin60, 0,
}

var triVerts = []float32{
	-sin60, -sin60, 0,
	0, sin60, 0,
	sin60, -sin60, 0,
}

var hexVerts = []float32{
	-0.5, -sin60, 0,
	-1, 0, 0,
	-0.5, sin60, 0,
	0.5, sin60, 0,
	1, 0, 0,
	0.5, -sin60, 0,
}

const vertShader = `#version 100
uniform mat4 mvMat;

attribute vec4 position;
void main() {
	gl_Position = mvMat * position;
}`

const fragShader = `#version 100
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
