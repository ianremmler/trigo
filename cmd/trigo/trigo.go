package main

import (
	"github.com/ianremmler/trigo"
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
	matchesFound = 0
	tri          *trigo.TriGo
)

func main() {
	rand.Seed(time.Now().UnixNano())
	tri = trigo.NewStd()
	play()
}

func play() {
	tri.Shuffle()
	tri.Deal()

	terminal.Stdout.Clear()
	terminal.Stdout.Move(0, 0)
	fmt.Println("TriGo!\n")

	for {
		printField()
		fmt.Printf("\n[matches: %02d, deck: %02d] > ", matchesFound, tri.DeckSize())
		str := ""
		fmt.Scan(&str)

		terminal.Stdout.Clear()
		terminal.Stdout.Move(0, 0)

		str = strings.TrimSpace(str)
		if len(str) != 3 {
			fmt.Printf("You must enter 3 cards.\n\n")
			continue
		}
		candidate := make([]int, 3)
		candidateStr := ""
		seen := map[int]struct{}{}
		isValid := true
		for i := 0; i < len(str); i++ {
			idx := strings.Index(keys, string(str[i]))
			if idx < 0 {
				isValid = false
				break
			}
			if _, ok := seen[idx]; ok {
				isValid = false
				break
			}
			seen[idx] = struct{}{}
			candidate[i] = idx
			candidateStr += printCard(idx)
			if i < len(candidate)-1 {
				candidateStr += " "
			}
		}
		if !isValid {
			fmt.Printf("Invalid cards.  Try again.\n\n")
			continue
		}
		if tri.IsMatch(candidate) {
			tri.Remove(candidate)
			tri.Deal()
			if tri.FieldMatches() == 0 {
				fmt.Println("You found all the matches!  Let's play again.\n")
				matchesFound = 0
				tri.Shuffle()
				tri.Deal()
			} else {
				color.Printf("@g✔@| %s @g✔\n\n", candidateStr)
				matchesFound++
			}
		} else {
			color.Printf("@r✘@| %s @r✘\n\n", candidateStr)
		}
	}
}

func printCard(i int) string {
	card := tri.FieldCard(i)
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
	field := tri.Field()
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
