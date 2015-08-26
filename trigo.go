// Package trigo provides the base for implementing a card game
package trigo

import (
	"bytes"
	"encoding/gob"
	"math"
	"math/rand"
)

// Card represents a playing card with attributes.
type Card struct {
	Attr  []int
	Blank bool
}

// gameState represents a complete game state.
type gameState struct {
	NumAttrs    int
	NumAttrVals int
	FieldSize   int
	FieldExpand int
	Cards       []Card
	Deck        []int
	Field       []int
}

// TriGo represents an instance of a game and its state.
type TriGo struct {
	state *gameState
}

// NewStd returns an instance of a standard game.
func NewStd() *TriGo {
	return New(4, 3, 12, 3)
}

// NewFromSavedState returns a game instance initialized to the given state.
func NewFromSavedState(state []byte) *TriGo {
	t := &TriGo{}
	buf := bytes.NewReader(state)
	dec := gob.NewDecoder(buf)
	if dec.Decode(&t.state) != nil {
		return nil
	}
	return t
}

// New returns an instance of a custom game.
func New(numAttrs, numAttrVals, fieldSize, fieldExpand int) *TriGo {
	numCards := 1
	for i := 0; i < numAttrs; i++ {
		numCards *= numAttrVals
	}
	t := &TriGo{}
	t.state = &gameState{
		NumAttrs:    numAttrs,
		NumAttrVals: numAttrVals,
		FieldSize:   fieldSize,
		FieldExpand: fieldExpand,
		Cards:       make([]Card, numCards),
		Deck:        make([]int, numCards),
		Field:       make([]int, fieldSize),
	}
	for i := range t.state.Cards {
		t.state.Cards[i].Attr = make([]int, numAttrs)
	}
	t.genCards()
	t.Shuffle()
	return t
}

func (t *TriGo) State() ([]byte, error) {
	buf := &bytes.Buffer{}
	enc := gob.NewEncoder(buf)
	if err := enc.Encode(t.state); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (t *TriGo) genCards() {
	for i := range t.state.Cards {
		div := 1
		for j := range t.state.Cards[0].Attr {
			t.state.Cards[i].Attr[j] = (i / div) % t.state.NumAttrVals
			div *= t.state.NumAttrVals
		}
	}
}

// DeckSize returns the number of cards currently in the deck.
func (t *TriGo) DeckSize() int {
	return len(t.state.Deck)
}

// Card returns ith card from the set of all cards, or a blank card if i is out
// of range.
func (t *TriGo) Card(i int) Card {
	if i < 0 || i >= len(t.state.Cards) {
		return Card{Blank: true}
	}
	return t.state.Cards[i]
}

// FieldCard returns the ith field card, or a blank card if i is out of range.
func (t *TriGo) FieldCard(i int) Card {
	if i < 0 || i >= len(t.state.Field) {
		return Card{Blank: true}
	}
	return t.Card(t.state.Field[i])
}

// Shuffle refills and shuffles the deck, and clears the field.
func (t *TriGo) Shuffle() {
	t.state.Deck = rand.Perm(len(t.state.Cards))
	t.state.Field = make([]int, t.state.FieldSize)
	for i := range t.state.Field {
		t.state.Field[i] = -1
	}
}

// Remove removes a match from the field.
func (t *TriGo) Remove(match []int) {
	for _, i := range match {
		if i >= 0 && i < len(t.state.Field) {
			t.state.Field[i] = -1
		}
	}
}

// expandField adds new card slots to the field.
func (t *TriGo) expandField() {
	expand := make([]int, t.state.FieldExpand)
	for i := range expand {
		expand[i] = -1
	}
	t.state.Field = append(t.state.Field, expand...)
}

// tidyField moves cards to empty slots and shrinks field if possible.
func (t *TriGo) tidyField() {
	numExtra := len(t.state.Field) - t.state.FieldSize
	for i, e := range t.state.Field[t.state.FieldSize:] {
		if e < 0 {
			numExtra--
			continue
		}
		for j, c := range t.state.Field[:t.state.FieldSize] {
			if c < 0 {
				t.state.Field[j] = e
				t.state.Field[t.state.FieldSize+i] = -1
				numExtra--
				break
			}
		}
	}
	expand := float64(t.state.FieldExpand)
	numExtra = int(math.Ceil(float64(numExtra)/expand) * expand)
	t.state.Field = t.state.Field[:t.state.FieldSize+numExtra]
}

// addCards fills empty field slots with new cards.
func (t *TriGo) addCards() {
	for i, c := range t.state.Field {
		if c < 0 {
			if len(t.state.Deck) == 0 {
				break
			}
			t.state.Field[i] = t.state.Deck[0]
			t.state.Deck = t.state.Deck[1:]
		}
	}
}

// Deal deals new cards to the field, expanding the field if necessary until at
// least one match is available.
func (t *TriGo) Deal() {
	t.tidyField()
	t.addCards()
	if t.NumMatches() == 0 && len(t.state.Deck) > 0 {
		t.expandField()
		t.addCards()
	}
}

// Field returns a slice of card indices representing the current field.
func (t *TriGo) Field() []Card {
	field := make([]Card, len(t.state.Field))
	for i, c := range t.state.Field {
		if c < 0 {
			field[i] = Card{Blank: true}
		} else {
			field[i] = t.state.Cards[c]
		}
	}
	return field
}

// IsMatch returns whether a given match candidate is valid
func (t *TriGo) IsMatch(candidate []int) bool {
	if len(candidate) != t.state.NumAttrVals {
		return false
	}
	attrCheck := make([]int, t.state.NumAttrs)
	for _, f := range candidate {
		if f < 0 || f >= len(t.state.Field) {
			return false
		}
		c := t.state.Field[f]
		if c < 0 || c >= len(t.state.Cards) {
			return false
		}
		card := &t.state.Cards[c]
		for i, val := range card.Attr {
			attrCheck[i] |= 1 << uint(val)
		}
	}
	for _, attr := range attrCheck {
		allSame := (attr != 0) && (attr&(attr-1) == 0)
		allDiff := (attr == 1<<uint(t.state.NumAttrVals)-1)
		if !allSame && !allDiff {
			return false
		}
	}
	return true
}

// NumMatches returns the number of matches in the field.
func (t *TriGo) NumMatches() int {
	numMatches := 0
	candidate := make([]int, t.state.NumAttrVals)

	var recurse func(int, int)
	recurse = func(i, n int) {
		for j := n; j < len(t.state.Field); j++ {
			candidate[i] = j
			if i == t.state.NumAttrVals-1 {
				if t.IsMatch(candidate) {
					numMatches++
				}
			} else {
				recurse(i+1, j+1)
			}
		}
		return
	}
	recurse(0, 0)

	return numMatches
}
