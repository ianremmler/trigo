package main

import (
	"github.com/ianremmler/setgo"
	"github.com/wsxiaoys/terminal"
	"github.com/wsxiaoys/terminal/color"

	"fmt"
	"math/rand"
	"strings"
	"time"
)

const (
	keys = "qazwsxedcrfvtgbyhn"
)

var (
	colors = []string{"@r", "@g", "@m"}
	shapes = [][]string{
		{"□", "◨", "■"},
		{"○", "◑", "●"},
		{"△", "◮", "▲"},
	}
	setsFound = 0
	set       *setgo.SetGo
)

func main() {
	rand.Seed(time.Now().UnixNano())
	set = setgo.NewStd()
	play()
}

func play() {
	set.Shuffle()
	set.Deal()

	terminal.Stdout.Clear()
	terminal.Stdout.Move(0, 0)
	fmt.Println("Set... Go!\n")

	for {
		printField()
		fmt.Printf("\n[sets: %02d, deck: %02d] > ", setsFound, set.DeckSize())
		str := ""
		fmt.Scan(&str)

		terminal.Stdout.Clear()
		terminal.Stdout.Move(0, 0)

		if len(str) < 3 {
			fmt.Println("You must enter 3 cards.\n")
			continue
		}
		candidate := make([]int, 3)
		candidateStr := ""
		for i := range candidate {
			candidate[i] = strings.Index(keys, string(str[i]))
			candidateStr += fmt.Sprintf("%s ", printCard(candidate[i]))
		}
		if set.IsSet(candidate) {
			set.Remove(candidate)
			if set.NumSets() == 0 {
				fmt.Println("You found all the sets!  Let's play again.\n")
				setsFound = 0
				set.Shuffle()
			} else {
				fmt.Printf("Woohoo!  %s is a set!\n\n", candidateStr)
				setsFound++
			}
			set.Deal()
		} else {
			fmt.Printf("D'oh!  %s is not a set.\n\n", candidateStr)
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
		tag := "?"
		if f >= 0 && f < len(keys) {
			tag = string(keys[f])
		}
		fmt.Printf("%s.%s", tag, printCard(f))
		if (i+1)%numCols == 0 {
			fmt.Println()
		} else {
			fmt.Print("  ")
		}
	}
}
