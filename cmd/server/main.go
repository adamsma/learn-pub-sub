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
	fmt.Println("Starting Peril server...")

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

	state := routing.PlayingState{}

	fmt.Println("Connection established")

	// bind peril_topic
	_, _, err = pubsub.DeclareAndBind(
		conn,
		routing.ExchangePerilTopic,
		routing.GameLogSlug,
		fmt.Sprintf("%s.*", routing.GameLogSlug),
		pubsub.QueueTypeDurable,
		nil,
	)
	if err != nil {
		log.Fatalf("unable to create new queue: %v", err)
	}

	gamelogic.PrintServerHelp()

gameloop:
	for {

		input := gamelogic.GetInput()

		if len(input) == 0 {
			continue
		}

		switch input[0] {
		case "pause":

			log.Println("Pausing game")
			state.IsPaused = true

			err = pubsub.PublishJSON(
				mqChan,
				routing.ExchangePerilDirect,
				routing.PauseKey,
				state,
			)
			if err != nil {
				log.Fatalf("error publishing message: %v", err)
			}

		case "resume":

			log.Println("Resuming game")
			state.IsPaused = false

			err = pubsub.PublishJSON(
				mqChan,
				routing.ExchangePerilDirect,
				routing.PauseKey,
				state,
			)
			if err != nil {
				log.Fatalf("error publishing message: %v", err)
			}

		case "quit":
			log.Println("Quitting")
			break gameloop
		default:
			log.Printf("unknown command: %s", input[0])
		}

	}

	fmt.Println("\nExiting...")
}
