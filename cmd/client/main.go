package main

import (
	"fmt"
	"log"

	"github.com/bootdotdev/learn-pub-sub-starter/internal/gamelogic"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/pubsub"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/routing"
	amqp "github.com/rabbitmq/amqp091-go"
)

func main() {
	fmt.Println("Starting Peril client...")

	const url = "amqp://guest:guest@localhost:5672/"
	conn, err := amqp.Dial(url)
	if err != nil {
		log.Fatalf("unable to connect to RabbitMQ: %v", err)
	}
	defer conn.Close()

	username, err := gamelogic.ClientWelcome()
	if err != nil {
		log.Fatalf("invalid username: %v", err)
	}

	// create new game state
	gamestate := gamelogic.NewGameState(username)

	// subscribe to pause queue
	err = pubsub.SubscribeJSON(
		conn,
		routing.ExchangePerilDirect,
		fmt.Sprintf("%s.%s", routing.PauseKey, username),
		routing.PauseKey,
		pubsub.QueueTypeTransient,
		handlerPause(gamestate),
	)
	if err != nil {
		log.Fatalf("unable to subscribe to pause queue: %v", err)
	}

gameloop:
	for {

		input := gamelogic.GetInput()

		switch input[0] {
		case "spawn":

			err = gamestate.CommandSpawn(input)
			if err != nil {
				fmt.Printf("invalid spawn command: %v\n", err)
				continue
			}

		case "move":

			_, err := gamestate.CommandMove(input)
			if err != nil {
				fmt.Printf("invalid move command: %v\n", err)
				continue
			}
			fmt.Println("Move successful!")

		case "status":

			gamestate.CommandStatus()

		case "help":

			gamelogic.PrintClientHelp()

		case "spam":

			fmt.Println("Spamming not allowed yet!")

		case "quit":

			gamelogic.PrintQuit()
			break gameloop

		default:
			fmt.Printf("unknown command: %s", input[0])
		}

	}

}
