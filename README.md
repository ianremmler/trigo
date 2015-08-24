# TriGo

TriGo is a pattern matching card game.

The trigo package provides the basic game engine, with customizable game parameters.

Two front ends are included.  `app/trigo` is a mobile app, which currently runs on Android, using [golang.org/x/mobile](https://golang.org/x/mobile).  `cmd/trigo` is a terminal app, and requires unicode and ANSI color support.

## The game
- Each card has four attributes: number, shape, color, and fill.
- Each attribute has three possible values (terminal version in parentheses):
  - number: 1, 2, 3
  - shape: triangle, square, hexagon (circle)
  - color: red, green, blue (magenta)
  - fill: outline, striped (filled with outline), solid
- The deck consists of the 81 possible combinations of the attributes (3^4).
- The goal is to find matches which consist of three cards that satisfy the following rule.  For each of the four attribute, the three cards must be all the same or all different.  To put it another way, if two cards have the same attribute and the third is different, the cards are not a match.  Matches can have a mix of all-same and all-different for the four attributes.

## Play
- A 4x3 grid of cards is dealt.
- Select three cards to make a match.
  - In the mobile app, tap or click the three cards.
  - In the terminal app, type the letters corresponding to the cards and press '<Enter>'.
- If the cards form a valid match, they are removed and new cards are dealt in their place.
- At any time during play, if there are no possible matches, extra rows of cards are dealt until there is at least one possible match.
- Play continues until all cards have been dealt and valid matches remain.
- When all matches have been found, the deck is reshuffled and a new game
  begins.
