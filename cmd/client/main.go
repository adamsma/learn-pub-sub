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

	mqChan, err := conn.Channel()
	if err != nil {
		log.Fatal("unable to create RabbitMQ channel")
	}

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

	handlerMove := func(move gamelogic.ArmyMove) pubsub.AckType {
		outcome := gamestate.HandleMove(move)
		fmt.Print("> ")

		switch outcome {
		case gamelogic.MoveOutComeSafe:
			return pubsub.Ack
		case gamelogic.MoveOutcomeMakeWar:
			pubsub.PublishJSON(
				mqChan,
				routing.ExchangePerilTopic,
				routing.WarRecognitionsPrefix+"."+username,
				"War has begun for "+username,
			)
			return pubsub.NackRequeue
		case gamelogic.MoveOutcomeSamePlayer:
			fallthrough
		default:
			return pubsub.NackDiscard
		}
	}

	// subscribe to move queue
	err = pubsub.SubscribeJSON(
		conn,
		routing.ExchangePerilTopic,
		"army_moves."+username,
		"army_moves.*",
		pubsub.QueueTypeTransient,
		handlerMove,
	)
	if err != nil {
		log.Fatalf("unable to subscribe to move queue: %v", err)
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

			mv, err := gamestate.CommandMove(input)
			if err != nil {
				fmt.Printf("invalid move command: %v\n", err)
				continue
			}

			err = pubsub.PublishJSON(
				mqChan,
				string(routing.ExchangePerilTopic),
				"army_moves."+username,
				mv,
			)
			if err != nil {
				fmt.Printf("err broadcasting move command: %v\n", err)
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
