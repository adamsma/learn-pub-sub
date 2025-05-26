package pubsub

import (
	"encoding/json"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

func SubscribeJSON[T any](
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	simpleQueueType int, // an enum to represent "durable" or "transient"
	handler func(T),
) error {

	ch, _, err := DeclareAndBind(
		conn, exchange, queueName, key, simpleQueueType,
	)
	if err != nil {
		return err
	}

	delivChan, err := ch.Consume(
		queueName, "", false, false, false, false, nil,
	)
	if err != nil {
		return err
	}

	unmarshaller := func(data []byte) (T, error) {
		var msg T
		err := json.Unmarshal(data, &msg)
		return msg, err

	}

	go func() {

		defer ch.Close()

		for deliv := range delivChan {
			msg, err := unmarshaller(deliv.Body)
			if err != nil {
				fmt.Printf("error in processing message: %v", err)
				continue
			}
			handler(msg)

			err = deliv.Ack(false)
			if err != nil {
				fmt.Printf("unable to acknowledge: %v", err)
			}

		}
	}()

	return nil

}
