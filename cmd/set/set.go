package main

import (
	"github.com/ianremmler/setgame"
	"github.com/wsxiaoys/terminal/color"

	"fmt"
	"math/rand"
	"os"
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
}

func newGame() {
	set.Shuffle()
	set.Deal()
	str := ""
	for {
		printField()
		if set.NumSets() == 0 {
			fmt.Println("\nNo more sets.")
			os.Exit(0)
		}
		fmt.Print("\n> ")
		fmt.Scan(&str)
		if len(str) < 3 {
			fmt.Println("You must name 3 cards.")
			continue
		}
		candidate := make([]int, 3)
		for i := range candidate {
			candidate[i] = int(str[i]) - 'a'
		}

		fmt.Println()
		for _, c := range candidate {
			fmt.Printf("%s ", printCard(c))
		}
		if set.IsSet(candidate) {
			fmt.Println("is a set!\n")
			set.Remove(candidate)
			set.Deal()
		} else {
			fmt.Println("is not a set.\n")
		}
	}
}

func printCard(i int) string {
	card := set.FieldCard(i)
	str := ""
	if card.Blank {
		str = "[       ]"
	} else {
		num, clr, shp, fil := card.Attr[0], card.Attr[1], card.Attr[2], card.Attr[3]
		shapeStr := strings.Repeat(" "+shapes[shp][fil], num+1)
		colorStr := colors[clr]
		padStr := strings.Repeat(" ", 2-num)
		str = "[" + padStr + colorStr + shapeStr + padStr + "@| ]"
	}
	return color.Sprint(str)
}

func printField() {
	field := set.Field()
	for i := range field {
		numCards := len(field)
		numCols := numCards / 3
		f := (i*3)%numCards + (i / numCols)
		tag := string(int('a') + f)
		fmt.Printf("%s.%s", tag, printCard(f))
		if (i+1)%numCols == 0 {
			fmt.Println()
		} else {
			fmt.Print("  ")
		}
	}
}
