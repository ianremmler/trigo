package main

import (
	"encoding/binary"
	"fmt"
	"log"
	"math"
	"math/rand"
	"time"

	"github.com/ianremmler/setgo"
	"golang.org/x/mobile/app"
	"golang.org/x/mobile/app/debug"
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
	set       *setgo.SetGo
	program   gl.Program
	position  gl.Attrib
	color     gl.Uniform
	shading   gl.Uniform
	doOutline gl.Uniform
	mvMat     gl.Uniform
	touchLoc  geom.Point

	cardShape = shape{verts: cardVerts}
	cardColor = []float32{1, 1, 1, 1}
)

var colors = [][]float32{
	{1.0, 0.0, 0.0, 1.0},
	{0.0, 1.0, 0.0, 1.0},
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
	doOutline = gl.GetUniformLocation(program, "doOutline")
	mvMat = gl.GetUniformLocation(program, "mvMat")

	touchLoc = geom.Point{geom.Width / 2, geom.Height / 2}
}

func stop() {
	gl.DeleteProgram(program)
	gl.DeleteBuffer(cardShape.buf)
	for i := range shapes {
		gl.DeleteBuffer(shapes[i].buf)
	}
}

func touch(t event.Touch) {
	touchLoc = t.Loc
}

func drawCard(card setgo.Card) {
	// 	num, clr, shp, fil := card.Attr[0], card.Attr[1], card.Attr[2], card.Attr[3]
	gl.BindBuffer(gl.ARRAY_BUFFER, cardShape.buf)
	gl.EnableVertexAttribArray(position)
	gl.VertexAttribPointer(position, coordsPerVertex, gl.FLOAT, false, 0, 0)

	gl.Uniform1i(shading, 2)
	gl.Uniform4fv(color, cardColor)
	gl.DrawArrays(gl.TRIANGLE_FAN, 0, len(cardShape.verts))
	gl.Uniform4f(color, 0, 0, 0, 0)
	gl.DrawArrays(gl.LINE_LOOP, 0, len(cardShape.verts))

	gl.DisableVertexAttribArray(position)
	// 	gl.Uniform4fv(color, colors[clr])
}

func draw() {
	field := set.Field()
	cols := len(field) / 3
	fieldAspRat := float32(cols) / (3 * cardAspRat)
	dispAspRat := float32(geom.Width / geom.Height)
	// 	aspRat := dispAspRat / fieldAspRat

	ow, oh := float32(cols), float32(3*cardAspRat)
	w, h := ow, oh
	if dispAspRat > fieldAspRat {
		w = oh * dispAspRat
	} else {
		h = ow / dispAspRat
	}
	fmt.Println(" ->", w, h)
	// 	if aspRat > 1 {
	// 		w = h * aspRat
	// 	} else {
	// 		h = w / aspRat
	// 	}

	gl.ClearColor(0, 0, 0, 1)
	gl.Clear(gl.COLOR_BUFFER_BIT)

	gl.UseProgram(program)

	// 	gl.Uniform4f(color, 0, green, 0, 1)
	// 	gl.Uniform2f(offset, float32(touchLoc.X/geom.Width), float32(touchLoc.Y/geom.Height))

	mat := f32.Mat4{}
	mat.Identity()
	// 	mat.Scale(&mat, 0.5*h, 0.5*w, 1)
	mat.Scale(&mat, 1.0/(0.5*w), 1.0/(0.5*h), 1)
	for i := range field {
		x, y := float32(i/3), cardAspRat*float32(i%3)
		mv := mat
		fmt.Println("i:", i, "x:", x, "y:", y, "w:", w, "h:", h, "px:", x-0.5*w, "py:", y-0.5*h)
		mv.Translate(&mv, x-0.5*ow, y-0.5*oh, 0)
		mvMat.WriteMat4(&mv)
		drawCard(field[i])
	}

	debug.DrawFPS()
}

var cardVerts = []float32{
	0, 0, 0,
	1, 0, 0,
	1, cardAspRat, 0,
	0, cardAspRat, 0,
}

var squareVerts = []float32{
	0, 0, 0,
	1, 0, 0,
	1, 1, 0,
	0, 1, 0,
}

var sin60 = float32(math.Sqrt(3) / 2)

var triVerts = []float32{
	-sin60, 0, 0,
	0.5, 1, 0,
	sin60, 0, 0,
}

var hexVerts = []float32{
	-sin60, 0, 0,
	-1, 0.5, 0,
	-sin60, 1, 0,
	sin60, 1, 0,
	1, 0.5, 0,
	sin60, 0, 0,
}

const (
	coordsPerVertex = 3
)

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
