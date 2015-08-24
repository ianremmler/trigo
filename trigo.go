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

// state represents a complete game state.
type state struct {
	NumAttrs    int
	NumAttrVals int
	FieldSize   int
	FieldExpand int
	Cards       []Card
	Deck        []int
	Field       []int
}

// TriGo represents an instance of a game and it's state.
type TriGo struct {
	st *state
}

// NewStd returns an instance of a standard game.
func NewStd() *TriGo {
	return New(4, 3, 12, 3)
}

// NewFromSavedState returns a game instance initialized to the given state.
func NewFromSavedState(state []byte) *TriGo {
	s := &TriGo{}
	buf := bytes.NewReader(state)
	dec := gob.NewDecoder(buf)
	if dec.Decode(&s.st) != nil {
		return nil
	}
	return s
}

// New returns an instance of a custom game.
func New(numAttrs, numAttrVals, fieldSize, fieldExpand int) *TriGo {
	numCards := 1
	for i := 0; i < numAttrs; i++ {
		numCards *= numAttrVals
	}
	s := &TriGo{}
	s.st = &state{
		NumAttrs:    numAttrs,
		NumAttrVals: numAttrVals,
		FieldSize:   fieldSize,
		FieldExpand: fieldExpand,
		Cards:       make([]Card, numCards),
		Deck:        make([]int, numCards),
		Field:       make([]int, fieldSize),
	}
	for i := range s.st.Cards {
		s.st.Cards[i].Attr = make([]int, numAttrs)
	}
	s.genCards()
	s.Shuffle()
	return s
}

func (s *TriGo) State() ([]byte, error) {
	buf := &bytes.Buffer{}
	enc := gob.NewEncoder(buf)
	if err := enc.Encode(s.st); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (s *TriGo) genCards() {
	for i := range s.st.Cards {
		div := 1
		for j := range s.st.Cards[0].Attr {
			s.st.Cards[i].Attr[j] = (i / div) % s.st.NumAttrVals
			div *= s.st.NumAttrVals
		}
	}
}

// DeckSize returns the number of cards currently in the deck.
func (s *TriGo) DeckSize() int {
	return len(s.st.Deck)
}

// Card returns ith card from the set of all cards, or a blank card if i is out
// of range.
func (s *TriGo) Card(i int) Card {
	if i < 0 || i >= len(s.st.Cards) {
		return Card{Blank: true}
	}
	return s.st.Cards[i]
}

// FieldCard returns the ith field card, or a blank card if i is out of range.
func (s *TriGo) FieldCard(i int) Card {
	if i < 0 || i >= len(s.st.Field) {
		return Card{Blank: true}
	}
	return s.Card(s.st.Field[i])
}

// Shuffle refills and shuffles the deck, and clears the field.
func (s *TriGo) Shuffle() {
	s.st.Deck = rand.Perm(len(s.st.Cards))
	s.st.Field = make([]int, s.st.FieldSize)
	for i := range s.st.Field {
		s.st.Field[i] = -1
	}
}

// Remove removes a match from the field.
func (s *TriGo) Remove(match []int) {
	for _, i := range match {
		if i >= 0 && i < len(s.st.Field) {
			s.st.Field[i] = -1
		}
	}
}

// expandField adds new card slots to the field.
func (s *TriGo) expandField() {
	expand := make([]int, s.st.FieldExpand)
	for i := range expand {
		expand[i] = -1
	}
	s.st.Field = append(s.st.Field, expand...)
}

// tidyField moves cards to empty slots and shrinks field if possible.
func (s *TriGo) tidyField() {
	numExtra := len(s.st.Field) - s.st.FieldSize
	for i, e := range s.st.Field[s.st.FieldSize:] {
		if e < 0 {
			numExtra--
			continue
		}
		for j, c := range s.st.Field[:s.st.FieldSize] {
			if c < 0 {
				s.st.Field[j] = e
				s.st.Field[s.st.FieldSize+i] = -1
				numExtra--
				break
			}
		}
	}
	expand := float64(s.st.FieldExpand)
	numExtra = int(math.Ceil(float64(numExtra)/expand) * expand)
	s.st.Field = s.st.Field[:s.st.FieldSize+numExtra]
}

// addCards fills empty field slots with new cards.
func (s *TriGo) addCards() {
	for i, c := range s.st.Field {
		if c < 0 {
			if len(s.st.Deck) == 0 {
				break
			}
			s.st.Field[i] = s.st.Deck[0]
			s.st.Deck = s.st.Deck[1:]
		}
	}
}

// Deal deals new cards to the field, expanding the field if necessary until at
// least one match is available.
func (s *TriGo) Deal() {
	s.tidyField()
	s.addCards()
	if s.NumMatches() == 0 && len(s.st.Deck) > 0 {
		s.expandField()
		s.addCards()
	}
}

// Field returns a slice of card indices representing the current field.
func (s *TriGo) Field() []Card {
	field := make([]Card, len(s.st.Field))
	for i, c := range s.st.Field {
		if c < 0 {
			field[i] = Card{Blank: true}
		} else {
			field[i] = s.st.Cards[c]
		}
	}
	return field
}

// IsMatch returns whether a given match candidate is valid
func (s *TriGo) IsMatch(candidate []int) bool {
	if len(candidate) != s.st.NumAttrVals {
		return false
	}
	attrCheck := make([]int, s.st.NumAttrs)
	for _, f := range candidate {
		if f < 0 || f >= len(s.st.Field) {
			return false
		}
		c := s.st.Field[f]
		if c < 0 || c >= len(s.st.Cards) {
			return false
		}
		card := &s.st.Cards[c]
		for i, val := range card.Attr {
			attrCheck[i] |= 1 << uint(val)
		}
	}
	for _, attr := range attrCheck {
		allSame := (attr != 0) && (attr&(attr-1) == 0)
		allDiff := (attr == 1<<uint(s.st.NumAttrVals)-1)
		if !allSame && !allDiff {
			return false
		}
	}
	return true
}

// NumMatches returns the number of matches in the field.
func (s *TriGo) NumMatches() int {
	numMatches := 0
	candidate := make([]int, s.st.NumAttrVals)

	var recurse func(int, int)
	recurse = func(i, n int) {
		for j := n; j < len(s.st.Field); j++ {
			candidate[i] = j
			if i == s.st.NumAttrVals-1 {
				if s.IsMatch(candidate) {
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
