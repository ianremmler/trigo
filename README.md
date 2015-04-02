SetGo is an implementation of the rules of the card game Set in Go.

The setgo package provides the basic game engine, with customizable game
parameters.

Two front ends are included.  cmd/setgo is a terminal application, and requires
unicode and ANSI color support.  app/setgo is a mobile app, which currently
runs on Android, using golang.org/x/mobile (which is currently very
preliminary and rough around the edges).

I am not affiliated with the makers of Set.  See http://setgame.com/ for the
rules and more about the game.  And buy the actual card game.  It's awesome.
