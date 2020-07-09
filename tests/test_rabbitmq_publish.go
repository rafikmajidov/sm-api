package main

import (
	"encoding/json"
	"github.com/streadway/amqp"
	"log"
)

func failOnError(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s", msg, err)
	}
}

func main() {
	type QueueMessage struct {
		UserId    string `json:"user_id"`
		SaAppCode string `json:sa_app_code`
	}
	queueMessage := QueueMessage{UserId: "rafik2", SaAppCode: "test_sa_code2"}

	js, err := json.Marshal(queueMessage)
	if err == nil {
		// Save JSON blob to Redis

		conn, err := amqp.Dial("amqp://cityrealty:br@vo99!Fm@10.4.1.72:5672/feed")
		failOnError(err, "Failed to connect to RabbitMQ")
		defer conn.Close()

		ch, err := conn.Channel()
		failOnError(err, "Failed to open a channel")
		defer ch.Close()

		q, err := ch.QueueDeclare(
			"goapiserverlocal", // name
			true,               // durable
			false,              // delete when unused
			false,              // exclusive
			false,              // no-wait
			nil,                // arguments
		)
		failOnError(err, "Failed to declare a queue")

		err = ch.Publish(
			"",     // exchange
			q.Name, // routing key
			false,  // mandatory
			false,  // immediate
			amqp.Publishing{
				DeliveryMode: amqp.Persistent,
				ContentType:  "text/plain",
				Body:         js,
			})
		log.Printf(" [x] Sent %s", js)
		failOnError(err, "Failed to publish a message")
	}

}
