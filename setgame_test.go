package setgame_test

import (
	"github.com/ianremmler/setgame"
	"github.com/wsxiaoys/terminal/color"

	"fmt"
	"math/rand"
	"strings"
	"testing"
	"time"
)

var (
	colors = []string{"@r", "@b", "@m"}
	shapes = [][]string{
		{"□", "▣", "■"},
		{"○", "◉", "●"},
		{"◇", "◈", "◆"},
	}
)

func TestSetGame(t *testing.T) {
	rand.Seed(time.Now().UnixNano())
	set := setgame.NewStd()
	for i := 0; i < 20; i++ {
		set.Shuffle()
		set.Deal()
		fmt.Println(set.NumSets())
	}
}

func printField(field []setgame.Card) {
	for i := range field {
		idx := (i * 3) % 12 + (i / 4)
		card := field[idx]
		tag := string(int('a') + idx)
		num, clr, shp, fil := card.Attribs[0], card.Attribs[1], card.Attribs[2], card.Attribs[3]
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
