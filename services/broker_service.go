package services

// import (
// 	"context"
// 	"log"
// 	"time"

// 	amqp "github.com/rabbitmq/amqp091-go"
// )

// type RabbitService struct {
// 	conn    *amqp.Connection
// 	channel *amqp.Channel
// 	queue   string
// }

// func NewRabbitService(amqpURL, queueName string) (*RabbitService, error) {
// 	conn, err := amqp.Dial(amqpURL)
// 	if err != nil {
// 		return nil, err
// 	}

// 	ch, err := conn.Channel()
// 	if err != nil {
// 		conn.Close()
// 		return nil, err
// 	}

// 	_, err = ch.QueueDeclare(
// 		queueName,
// 		true,
// 		false,
// 		false,
// 		false,
// 		nil,
// 	)
// 	if err != nil {
// 		ch.Close()
// 		conn.Close()
// 		return nil, err
// 	}

// 	return &RabbitService{
// 		conn:    conn,
// 		channel: ch,
// 		queue:   queueName,
// 	}, nil
// }

// func (r *RabbitService) Publish(body []byte) error {
// 	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
// 	defer cancel()

// 	err := r.channel.PublishWithContext(ctx,
// 		"",
// 		r.queue,
// 		false,
// 		false,
// 		amqp.Publishing{
// 			ContentType:  "application/json",
// 			Body:         body,
// 			DeliveryMode: amqp.Persistent,
// 		},
// 	)
// 	if err != nil {
// 		return err
// 	}

// 	log.Printf("[RabbitMQ] Mensagem publicada na fila '%s' (%d bytes)", r.queue, len(body))
// 	return nil
// }

//	func (r *RabbitService) Close() {
//		if r.channel != nil {
//			r.channel.Close()
//		}
//		if r.conn != nil {
//			r.conn.Close()
//		}
//	}
// package rabbit

import (
	"log"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type RabbitService struct {
	conn *amqp.Connection
}

func NewRabbitService(amqpURL string) (*RabbitService, error) {
	conn, err := amqp.DialConfig(amqpURL, amqp.Config{
		Heartbeat: 10 * time.Second,
		Locale:    "en_US",
	})

	if err != nil {
		log.Printf("[RabbitMQ] Erro ao conectar: %v", err)
		return nil, err
	}

	log.Println("[RabbitMQ] Conexão estabelecida com sucesso")
	return &RabbitService{conn: conn}, nil
}

func (r *RabbitService) ConsumeMultiple(queues []string, handler func(queue string, body []byte) error) error {
	for _, q := range queues {

		ch, err := r.conn.Channel()
		if err != nil {
			return err
		}

		// garante a existência da fila
		_, err = ch.QueueDeclare(q, true, false, false, false, nil)
		if err != nil {
			ch.Close()
			return err
		}

		msgs, err := ch.Consume(
			q,
			"",
			false,
			false,
			false,
			false,
			nil,
		)
		if err != nil {
			ch.Close()
			return err
		}

		// cria uma goroutine por fila
		go func(queueName string, deliveries <-chan amqp.Delivery, c *amqp.Channel) {
			for msg := range deliveries {
				if err := handler(queueName, msg.Body); err != nil {
					msg.Nack(false, true)
					continue
				}
				msg.Ack(false)
			}

			c.Close()
		}(q, msgs, ch)
	}

	return nil
}

// func (r *RabbitService) Publish(queue string, body []byte) error {
// 	ch, err := r.conn.Channel()
// 	if err != nil {
// 		return err
// 	}
// 	defer ch.Close()

//		return ch.Publish(
//			"",    // exchange padrão
//			queue, // routing key = nome da fila
//			false, // mandatory
//			false, // immediate
//			amqp.Publishing{
//				ContentType: "application/json",
//				Body:        body,
//			},
//		)
//	}
func (r *RabbitService) Publish(exchange, routingKey string, body []byte) error {

	ch, err := r.conn.Channel()
	if err != nil {
		return err
	}
	defer ch.Close()

	return ch.Publish(
		exchange,
		routingKey,
		false,
		false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		},
	)
}

// Consume inicia o consumo de mensagens de uma fila específica
func (r *RabbitService) Consume(queue string, handler func(body []byte) error) error {
	ch, err := r.conn.Channel()
	if err != nil {
		return err
	}

	// garante que a fila existe
	_, err = ch.QueueDeclare(
		queue,
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		ch.Close()
		return err
	}

	msgs, err := ch.Consume(
		queue,
		"",    // consumer tag
		false, // autoAck
		false, // exclusive
		false, // noLocal
		false, // noWait
		nil,
	)
	if err != nil {
		ch.Close()
		return err
	}

	// o canal só fecha quando o loop terminar
	go func() {
		for msg := range msgs {
			if err := handler(msg.Body); err != nil {
				log.Printf("Erro ao processar mensagem na fila '%s': %v", queue, err)
				msg.Nack(false, true) // requeue
				continue
			}
			msg.Ack(false)
		}
		ch.Close()
	}()

	return nil
}

func (r *RabbitService) Close() {
	if r.conn != nil {
		r.conn.Close()
	}
}
