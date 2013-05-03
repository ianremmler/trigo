package setgame_test

import (
	"github.com/ianremmler/setgame"

	"math/rand"
	"testing"
	"time"
)

func TestSetGame(t *testing.T) {
	rand.Seed(time.Now().UnixNano())
	set := setgame.NewStd()
	set.Shuffle()
	set.Deal()
 	t.Log(set.Field())
	t.Log(set.NumSets())
}
