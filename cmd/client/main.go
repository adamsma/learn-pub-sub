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

		defer fmt.Print("> ")
		outcome := gamestate.HandleMove(move)

		switch outcome {

		case gamelogic.MoveOutComeSafe:
			return pubsub.Ack

		case gamelogic.MoveOutcomeMakeWar:
			err := pubsub.PublishJSON(
				mqChan,
				routing.ExchangePerilTopic,
				routing.WarRecognitionsPrefix+"."+username,
				gamelogic.RecognitionOfWar{
					Attacker: move.Player, Defender: gamestate.Player,
				},
			)

			if err != nil {
				return pubsub.NackRequeue
			}

			return pubsub.Ack

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

	handlerWar := func(rw gamelogic.RecognitionOfWar) pubsub.AckType {

		defer fmt.Print("> ")
		outcome, _, _ := gamestate.HandleWar(rw)

		switch outcome {
		case gamelogic.WarOutcomeNotInvolved:
			return pubsub.NackRequeue
		case gamelogic.WarOutcomeNoUnits:
			return pubsub.NackDiscard
		case gamelogic.WarOutcomeOpponentWon:
			return pubsub.Ack
		case gamelogic.WarOutcomeYouWon:
			return pubsub.Ack
		case gamelogic.WarOutcomeDraw:
			return pubsub.Ack
		default:
			fmt.Printf("unknown war outcome: %v", outcome)
			return pubsub.NackDiscard
		}

	}

	// subscribe to war queue
	err = pubsub.SubscribeJSON(
		conn,
		routing.ExchangePerilTopic,
		"war",
		routing.WarRecognitionsPrefix+".*",
		pubsub.QueueTypeDurable,
		handlerWar,
	)
	if err != nil {
		log.Fatalf("unable to subscribe to war queue: %v", err)
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
