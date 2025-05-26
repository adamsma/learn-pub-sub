package main

import (
	"fmt"

	"github.com/bootdotdev/learn-pub-sub-starter/internal/gamelogic"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/routing"
)

func handlerPause(gs *gamelogic.GameState) func(routing.PlayingState) {

	fx := func(pst routing.PlayingState) {
		defer fmt.Print("> ")
		gs.HandlePause(pst)
	}

	return fx

}
