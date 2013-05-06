package setgo

import (
	"math"
	"math/rand"
)

type Card struct {
	Attr  []int
	Blank bool
}

type SetGo struct {
	numAttrs    int
	numAttrVals int
	fieldSize   int
	fieldExpand int
	cards       []Card
	deck        []int
	field       []int
}

func NewStd() *SetGo {
	return New(4, 3, 12, 3)
}

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

func (s *SetGo) DeckSize() int {
	return len(s.deck)
}

func (s *SetGo) Card(i int) Card {
	if i < 0 || i >= len(s.cards) {
		return Card{Blank: true}
	}
	return s.cards[i]
}

func (s *SetGo) FieldCard(i int) Card {
	if i < 0 || i >= len(s.field) {
		return Card{Blank: true}
	}
	return s.Card(s.field[i])
}

func (s *SetGo) Shuffle() {
	s.deck = rand.Perm(len(s.cards))
	s.field = make([]int, s.fieldSize)
	for i := range s.field {
		s.field[i] = -1
	}
}

func (s *SetGo) Remove(set []int) {
	if len(set) != s.numAttrVals {
		return
	}
	for _, i := range set {
		if i >= 0 && i < len(s.field) {
			s.field[i] = -1
		}
	}
}

func (s *SetGo) expandField() {
	expand := make([]int, s.fieldExpand)
	for i := range expand {
		expand[i] = -1
	}
	s.field = append(s.field, expand...)
}

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

func (s *SetGo) Deal() {
	s.tidyField()
	s.addCards()
	if s.NumSets() == 0 && len(s.deck) > 0 {
		s.expandField()
		s.addCards()
	}
}

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

func (s *SetGo) IsSet(candidate []int) bool {
	if len(candidate) != s.numAttrVals {
		return false
	}
	attrCheck := make([]map[int]struct{}, s.numAttrs)
	for i := range attrCheck {
		attrCheck[i] = map[int]struct{}{}
	}
	for _, f := range candidate {
		if f < 0 || f >= len(s.field) {
			return false
		}
		c := s.field[f]
		if c < 0 || c >= len(s.cards) {
			return false
		}
		card := &s.cards[c]
		for j, val := range card.Attr {
			attrCheck[j][val] = struct{}{}
		}
	}
	for _, attr := range attrCheck {
		if len(attr) != 1 && len(attr) != s.numAttrVals {
			return false
		}
	}
	return true
}

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
