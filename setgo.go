// Package setgo provides the base for implementing a card game
package setgo

import (
	"math"
	"math/rand"
)

// Card represents a playing card with attributes.
type Card struct {
	Attr  []int
	Blank bool
}

// SetGo represents an instance of a game and it's state.
type SetGo struct {
	numAttrs    int
	numAttrVals int
	fieldSize   int
	fieldExpand int
	cards       []Card
	deck        []int
	field       []int
}

// NewStd returns an instance of a standard game.
func NewStd() *SetGo {
	return New(4, 3, 12, 3)
}

// New returns an instance of a custom game.
func New(numAttrs, numAttrVals, fieldSize, fieldExpand int) *SetGo {
	numCards := 1
	for i := 0; i < numAttrs; i++ {
		numCards *= numAttrVals
	}
	s := &SetGo{
		numAttrs:    numAttrs,
		numAttrVals: numAttrVals,
		fieldSize:   fieldSize,
		fieldExpand: fieldExpand,
		cards:       make([]Card, numCards),
		deck:        make([]int, numCards),
		field:       make([]int, fieldSize),
	}
	for i := range s.cards {
		s.cards[i].Attr = make([]int, numAttrs)
	}
	s.genCards()
	s.Shuffle()
	return s
}

func (s *SetGo) genCards() {
	for i := range s.cards {
		div := 1
		for j := range s.cards[0].Attr {
			s.cards[i].Attr[j] = (i / div) % s.numAttrVals
			div *= s.numAttrVals
		}
	}
}

// DeckSize returns the number of cards currently in the deck.
func (s *SetGo) DeckSize() int {
	return len(s.deck)
}

// Card returns ith card from the set of all cards, or a blank card if i is out
// of range.
func (s *SetGo) Card(i int) Card {
	if i < 0 || i >= len(s.cards) {
		return Card{Blank: true}
	}
	return s.cards[i]
}

// FieldCard returns the ith field card, or a blank card if i is out of range.
func (s *SetGo) FieldCard(i int) Card {
	if i < 0 || i >= len(s.field) {
		return Card{Blank: true}
	}
	return s.Card(s.field[i])
}

// Shuffle refills and shuffles the deck, and clears the field.
func (s *SetGo) Shuffle() {
	s.deck = rand.Perm(len(s.cards))
	s.field = make([]int, s.fieldSize)
	for i := range s.field {
		s.field[i] = -1
	}
}

// Remove removes a set of cards from the field.
func (s *SetGo) Remove(set []int) {
	for _, i := range set {
		if i >= 0 && i < len(s.field) {
			s.field[i] = -1
		}
	}
}

// expandField adds new card slots to the field.
func (s *SetGo) expandField() {
	expand := make([]int, s.fieldExpand)
	for i := range expand {
		expand[i] = -1
	}
	s.field = append(s.field, expand...)
}

// tidyField moves cards to empty slots and shrinks field if possible.
func (s *SetGo) tidyField() {
	numExtra := len(s.field) - s.fieldSize
	for i, e := range s.field[s.fieldSize:] {
		if e < 0 {
			numExtra--
			continue
		}
		for j, c := range s.field[:s.fieldSize] {
			if c < 0 {
				s.field[j] = e
				s.field[s.fieldSize+i] = -1
				numExtra--
				break
			}
		}
	}
	expand := float64(s.fieldExpand)
	numExtra = int(math.Ceil(float64(numExtra)/expand) * expand)
	s.field = s.field[:s.fieldSize+numExtra]
}

// addCards fills empty field slots with new cards.
func (s *SetGo) addCards() {
	for i, c := range s.field {
		if c < 0 {
			if len(s.deck) == 0 {
				break
			}
			s.field[i] = s.deck[0]
			s.deck = s.deck[1:]
		}
	}
}

// Deal deals new cards to the field, expanding the field if necessary until at
// least one match is available.
func (s *SetGo) Deal() {
	s.tidyField()
	s.addCards()
	if s.NumSets() == 0 && len(s.deck) > 0 {
		s.expandField()
		s.addCards()
	}
}

// Field returns a slice of card indices representing the current field.
func (s *SetGo) Field() []Card {
	field := make([]Card, len(s.field))
	for i, c := range s.field {
		if c < 0 {
			field[i] = Card{Blank: true}
		} else {
			field[i] = s.cards[c]
		}
	}
	return field
}

// IsSet returns whether a given set of cards is a valid match.
func (s *SetGo) IsSet(candidate []int) bool {
	if len(candidate) != s.numAttrVals {
		return false
	}
	attrCheck := make([]int, s.numAttrs)
	for _, f := range candidate {
		if f < 0 || f >= len(s.field) {
			return false
		}
		c := s.field[f]
		if c < 0 || c >= len(s.cards) {
			return false
		}
		card := &s.cards[c]
		for i, val := range card.Attr {
			attrCheck[i] |= 1 << uint(val)
		}
	}
	for _, attr := range attrCheck {
		allSame := (attr != 0) && (attr&(attr-1) == 0)
		allDiff := (attr == 1<<uint(s.numAttrVals)-1)
		if !allSame && !allDiff {
			return false
		}
	}
	return true
}

// NumSets returns the number of matches in the field.
func (s *SetGo) NumSets() int {
	numSets := 0
	candidate := make([]int, s.numAttrVals)

	var recurse func(int, int)
	recurse = func(i, n int) {
		for j := n; j < len(s.field); j++ {
			candidate[i] = j
			if i == s.numAttrVals-1 {
				if s.IsSet(candidate) {
					numSets++
				}
			} else {
				recurse(i+1, j+1)
			}
		}
		return
	}
	recurse(0, 0)

	return numSets
}
