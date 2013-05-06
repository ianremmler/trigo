package setgo_test

import (
	"github.com/ianremmler/setgo"
	"github.com/wsxiaoys/terminal/color"

	"fmt"
	"math/rand"
	"strings"
	"testing"
	"time"
)

var (
	colors = []string{"@r", "@g", "@m"}
	shapes = [][]string{
		{"□", "▣", "■"},
		{"○", "◉", "●"},
		{"◇", "◈", "◆"},
	}
)

func TestSetGo(t *testing.T) {
	rand.Seed(time.Now().UnixNano())
	set := setgo.NewStd()
	for i := 0; i < 5; i++ {
		set.Shuffle()
		set.Deal()
		printField(set.Field())
		fmt.Println()
	}
}

func BenchmarkSetGo(b *testing.B) {
	rand.Seed(time.Now().UnixNano())
	set := setgo.NewStd()
	for i := 0; i < b.N; i++ {
		set.Shuffle()
		set.Deal()
		set.NumSets()
	}
}

func printField(field []setgo.Card) {
	for i := range field {
		idx := (i*3)%12 + (i / 4)
		card := field[idx]
		tag := string(int('a') + idx)
		num, clr, shp, fil := card.Attr[0], card.Attr[1], card.Attr[2], card.Attr[3]
		shapeStr := strings.Repeat(" "+shapes[shp][fil], num+1)
		colorStr := colors[clr]
		padStr := strings.Repeat(" ", 2-num)
		color.Print(tag + " [" + padStr + colorStr + shapeStr + padStr + "@| ]")
		if (i+1)%4 == 0 {
			fmt.Println()
		} else {
			fmt.Print("  ")
		}
	}
}
