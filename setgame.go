package setgame

import (
	"math"
	"math/rand"
)

type Card struct {
	Attribs []int
	Blank   bool
}

type SetGame struct {
	numAttribs    int
	numAttribVals int
	fieldSize     int
	fieldExpand   int
	cards         []Card
	deck          []int
	field         []int
}

func NewStd() *SetGame {
	return New(4, 3, 12, 3)
}

func New(numAttribs, numAttribVals, fieldSize, fieldExpand int) *SetGame {
	numCards := 1
	for i := 0; i < numAttribs; i++ {
		numCards *= numAttribVals
	}
	s := &SetGame{
		numAttribs:    numAttribs,
		numAttribVals: numAttribVals,
		fieldSize:     fieldSize,
		fieldExpand:   fieldExpand,
		cards:         make([]Card, numCards),
		deck:          make([]int, numCards),
		field:         make([]int, fieldSize),
	}
	for i := range s.cards {
		s.cards[i].Attribs = make([]int, numAttribs)
	}
	s.genCards()
	s.Shuffle()
	return s
}

func (s *SetGame) genCards() {
	for i := range s.cards {
		div := 1
		for j := range s.cards[0].Attribs {
			s.cards[i].Attribs[j] = (i / div) % s.numAttribVals
			div *= s.numAttribVals
		}
	}
}

func (s *SetGame) Card(idx int) *Card {
	if idx < 0 || idx >= len(s.cards) {
		return nil
	}
	card := s.cards[idx]
	return &card
}

func (s *SetGame) Shuffle() {
	s.deck = rand.Perm(len(s.cards))
	s.field = make([]int, s.fieldSize)
	for i := range s.field {
		s.field[i] = -1
	}
}

func (s *SetGame) Remove(list ...int) {
	for _, idx := range list {
		if idx >= 0 && idx < len(s.field) {
			s.field[idx] = -1
		}
	}
}

func (s *SetGame) expandField() {
	exp := make([]int, s.fieldExpand)
	for i := range exp {
		exp[i] = -1
	}
	s.field = append(s.field, exp...)
}

func (s *SetGame) tidyField() {
	numExtra := len(s.field) - s.fieldSize
	for i, extraIdx := range s.field[s.fieldSize:] {
		if extraIdx >= 0 {
			for j, idx := range s.field[:s.fieldSize] {
				if idx < 0 {
					s.field[j] = extraIdx
					s.field[i] = -1
					numExtra--
					break
				}
			}
		} else {
			numExtra--
		}
	}
	expand := float64(s.fieldExpand)
	numExtra = int(math.Ceil(float64(numExtra) / expand) * expand)
	s.field = s.field[:s.fieldSize + numExtra]
}

func (s *SetGame) addCards() {
	for i, idx := range s.field {
		if idx < 0 {
			if len(s.deck) == 0 {
				break
			}
			s.field[i] = s.deck[0]
			s.deck = s.deck[1:]
		}
	}
}

func (s *SetGame) Deal() {
	s.tidyField()
	s.addCards()
	if s.NumSets() == 0 && len(s.deck) > 0 {
		s.expandField()
		s.addCards()
	}
}

func (s *SetGame) Field() []Card {
	field := make([]Card, len(s.field))
	for i, idx := range s.field {
		if idx < 0 {
			field[i] = Card{Blank: true}
		} else {
			field[i] = s.cards[idx]
		}
	}
	return field
}

func (s *SetGame) IsSet(candidate []int) bool {
	if len(candidate) != s.numAttribVals {
		return false
	}
	attribCk := make([]map[int]struct{}, s.numAttribs)
	for i := range attribCk {
		attribCk[i] = map[int]struct{}{}
	}
	for _, idx := range candidate {
		if idx < 0 || idx >= len(s.field) {
			return false
		}
		fieldIdx := s.field[idx]
		if fieldIdx < 0 || fieldIdx >= len(s.cards) {
			return false
		}
		card := &s.cards[fieldIdx]
		for j, val := range card.Attribs {
			attribCk[j][val] = struct{}{}
		}
	}
	for _, attrib := range attribCk {
		if len(attrib) != 1 && len(attrib) != s.numAttribVals {
			return false
		}
	}
	return true
}

func (s *SetGame) NumSets() int {
	numSets := 0
	combinations(len(s.field), s.numAttribVals, func(combo []int) {
		candidate := make([]int, s.numAttribVals)
		for i, idx := range combo {
			candidate[i] = idx
		}
		if s.IsSet(candidate) {
			numSets++
		}
	})
	return numSets
}

// stolen from rosetta code
func combinations(n, m int, emit func([]int)) {
	s := make([]int, m)
	last := m - 1
	var rc func(int, int)
	rc = func(i, next int) {
		for j := next; j < n; j++ {
			s[i] = j
			if i == last {
				emit(s)
			} else {
				rc(i+1, j+1)
			}
		}
		return
	}
	rc(0, 0)
}
