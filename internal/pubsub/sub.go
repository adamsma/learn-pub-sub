package pubsub

import (
	"encoding/json"
	"fmt"
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
)

type AckType int

const (
	Ack AckType = iota
	NackRequeue
	NackDiscard
)

func SubscribeJSON[T any](
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	simpleQueueType int, // an enum to represent "durable" or "transient"
	handler func(T) AckType,
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

			ack := handler(msg)

			switch ack {
			case Ack:
				err = deliv.Ack(false)
				log.Println("messsage acknowledge")
			case NackRequeue:
				err = deliv.Nack(false, true)
				log.Println("messsage nack and requeued")
			case NackDiscard:
				err = deliv.Nack(false, false)
				log.Println("messsage nack and discarded")
			default:
				err = fmt.Errorf("unknown ack type: %v", ack)
			}

			if err != nil {
				fmt.Printf("unable to acknowledge: %v", err)
			}

		}
	}()

	return nil

}
