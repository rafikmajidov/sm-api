package ellimango

import (
	"encoding/json"
	"errors"
	"github.com/streadway/amqp"
	"log"
	"os"
	"time"
)

type Rabbitmq struct {
	Env           string
	RabbitmqUser  string
	RabbitmqHost  string
	RabbitmqVhost string
}

func (rabbitmq *Rabbitmq) Connect() (*amqp.Connection, *amqp.Channel, error) {
	conn := &amqp.Connection{}
	ch := &amqp.Channel{}
	err := errors.New("")

	conn, err = amqp.Dial("amqp://" + rabbitmq.RabbitmqUser + ":br@vo99!Fm@" + rabbitmq.RabbitmqHost + ":5672/" + rabbitmq.RabbitmqVhost)
	if err != nil {
		log.Println("Rabbitmq connect error", err)
		// go sendEmail("Rafik Majidov", "rmajidov@reol.com", fmt.Sprintf("Rabbitmq connect error %v", err), "Rabbitmq connect error")
	} else {
		ch, err = conn.Channel()
		if err != nil {
			log.Println("Rabbitmq failed to open a channel", err)
			// go sendEmail("Rafik Majidov", "rmajidov@reol.com", fmt.Sprintf("Rabbitmq failed to open a channel %v", err), "Rabbitmq failed to open a channel")
		}
	}

	return conn, ch, err
}

func (rabbitmq *Rabbitmq) Close(conn *amqp.Connection, ch *amqp.Channel) error {
	// first close channel, then connection
	err := ch.Close()
	if err != nil {
		log.Println("Rabbitmq close channel error", err)
		// go sendEmail("Rafik Majidov", "rmajidov@reol.com", fmt.Sprintf("Rabbitmq close channel error %v", err), "Rabbitmq close channel error")
	}
	err = conn.Close()
	if err != nil {
		log.Println("Rabbitmq close error", err)
		// go sendEmail("Rafik Majidov", "rmajidov@reol.com", fmt.Sprintf("Rabbitmq close error %v", err), "Rabbitmq close error")
	}
	log.Println("Close rabbitmq error", err)
	return err
}

func (rabbitmq *Rabbitmq) Publish(ch *amqp.Channel, queueName string, queueMessage interface{}) {
	helper := Helper{Env: rabbitmq.Env}
	q, err := ch.QueueDeclare(
		queueName, // name
		true,      // durable
		false,     // delete when unused
		false,     // exclusive
		false,     // no-wait
		nil,       // arguments
	)

	if err != nil {
		log.Println("Rabbitmq failed to declare a queue", err)
		rabbitmq.SaveInFile(queueName, queueMessage)
		// go sendEmail("Rafik Majidov", "rmajidov@reol.com", fmt.Sprintf("Rabbitmq failed to declare a queue %v", err), "Rabbitmq failed to declare a queue")
	} else {
		js, err := json.Marshal(queueMessage)
		if err == nil {
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
			helper.Debug("[x] Sent", string(js))
			if err != nil {
				log.Println("Rabbitmq publish error", err)
				// go sendEmail("Rafik Majidov", "rmajidov@reol.com", fmt.Sprintf("Rabbitmq publish error %v", err), "Rabbitmq publish error")
			}
		} else {
			log.Println("Rabbitmq publish json marshall error", err, ", message ", queueMessage)
		}
	}
}

func (rabbitmq *Rabbitmq) SaveInFile(queueName string, queueMessage interface{}) {
	fileName := "/tmp/" + queueName + ".log"
	js, err := json.Marshal(queueMessage)
	if err == nil {
		// if file does not exist, create it
		if _, err := os.Stat(fileName); os.IsNotExist(err) {
			file, err := os.Create(fileName)
			if err != nil {
				log.Println("Rabbitmq saveinfile could not create the file "+fileName, err)
			} else {
				file.Chmod(0777)
				defer file.Close()
			}
		}

		// open files append/write
		file, err := os.OpenFile(fileName, os.O_APPEND|os.O_WRONLY, os.ModeAppend)
		if err != nil {
			log.Println("Rabbitmq saveinfile could not open the file "+fileName, err)
		} else {
			_, err = file.WriteString(string(js) + "\n")
			if err != nil {
				log.Println("Rabbitmq saveinfile could not write to the file "+fileName, err)
			}
			defer file.Close()
		}
	} else {
		log.Println("Rabbitmq saveinfile json marshall error", err, ", message ", queueMessage)
	}
}

// go routine to save task in rabbitmq
func (rabbitmq *Rabbitmq) WorkerSaveTask(queueMessage *RabbitmqTask) {
	helper := Helper{Env: rabbitmq.Env}
	queueName := RABBITMQ_TASKS_QUEUE + rabbitmq.Env

	helper.Debug("rabbitmqWorkerSaveTask() before sending to rabbitmq time.Now " + time.Now().String())

	conn, ch, err := rabbitmq.Connect()
	// no connection error, no channel opening error
	if err == nil {
		rabbitmq.Publish(ch, queueName, queueMessage)
		rabbitmq.Close(conn, ch)
		// connection error, or channel opening error
	} else {
		rabbitmq.SaveInFile(queueName, queueMessage)
	}
	helper.Debug("rabbitmqWorkerSaveTask() after sending to rabbitmq time.Now " + time.Now().String())
}
