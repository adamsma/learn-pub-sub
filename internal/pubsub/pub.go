package pubsub

import (
	"context"
	"encoding/json"
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	QueueTypeDurable = iota
	QueueTypeTransient
)

func PublishJSON[T any](ch *amqp.Channel, exchange, key string, val T) error {

	jsonData, err := json.Marshal(val)
	if err != nil {
		log.Fatal(err)
	}

	msg := amqp.Publishing{
		ContentType: "application/json",
		Body:        jsonData,
	}

	err = ch.PublishWithContext(context.Background(), exchange, key, false, false, msg)

	return err

}

func DeclareAndBind(
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	simpleQueueType int, // an enum to represent "durable" or "transient"
) (*amqp.Channel, amqp.Queue, error) {

	mqChan, err := conn.Channel()
	if err != nil {
		log.Fatal("unable to create RabbitMQ channel")
	}

	q, err := mqChan.QueueDeclare(
		queueName,
		simpleQueueType == QueueTypeDurable,
		simpleQueueType == QueueTypeTransient,
		simpleQueueType == QueueTypeTransient,
		false,
		nil,
	)
	if err != nil {
		log.Fatalf("unable to declare queue: %v", err)
	}

	mqChan.QueueBind(queueName, key, exchange, false, nil)

	return mqChan, q, nil
}
