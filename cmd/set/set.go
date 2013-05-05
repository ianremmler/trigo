package main

import (
	"github.com/ianremmler/setgame"
	"github.com/wsxiaoys/terminal/color"

	"fmt"
	"math/rand"
	"strings"
	"time"
)

var (
	colors = []string{"@r", "@g", "@m"}
	shapes = [][]string{
		{"□", "▣", "■"},
		{"○", "◉", "●"},
		{"◇", "◈", "◆"},
	}
	set *setgame.SetGame
)

func main() {
	rand.Seed(time.Now().UnixNano())
	set = setgame.NewStd()
	newGame()
	set.Remove(11)
	printField()
}

func newGame() {
	set.Shuffle()
	set.Deal()
	str := ""
	for {
		printField()
		fmt.Print("\n> ")
		fmt.Scan(&str)
		if len(str) < 3 {
			fmt.Println("You must name 3 cards.")
			continue
		}
		candidate := make([]int, 3)
		for i := 0; i < 3; i++ {
			candidate[i] = int(str[i]) - 'a'
		}
		if set.IsSet(candidate) {
			fmt.Println("\nset!\n")
			set.Remove(candidate...)
			set.Deal()
		} else {
			fmt.Println("\nsorry...\n")
		}
	}
}

func printField() {
	field := set.Field()
	for i := range field {
		numCards := len(field)
		numCols := numCards / 3
		idx := (i * 3) % numCards + (i / numCols)
		tag := string(int('a') + idx)
		card := field[idx]
		str := ""
		if card.Blank {
			str = tag + " [       ]"
		} else {
			num, clr, shp, fil := card.Attribs[0], card.Attribs[1], card.Attribs[2], card.Attribs[3]
			shapeStr := strings.Repeat(" "+shapes[shp][fil], num+1)
			colorStr := colors[clr]
			padStr := strings.Repeat(" ", 2-num)
			str = tag + " [" + padStr + colorStr + shapeStr + padStr + "@| ]"
		}
		color.Print(str)
		if (i+1)%numCols == 0 {
			fmt.Println()
		} else {
			fmt.Print("  ")
		}
	}
}
