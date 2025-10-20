package services

import (
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
)

type RabbitService struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	queue   string
}

func NewRabbitService(amqpURL, queueName string) (*RabbitService, error) {
	conn, err := amqp.Dial(amqpURL)
	if err != nil {
		return nil, err
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, err
	}

	_, err = ch.QueueDeclare(
		queueName,
		true,  // durable
		false, // autoDelete
		false, // exclusive
		false, // noWait
		nil,   // args
	)
	if err != nil {
		ch.Close()
		conn.Close()
		return nil, err
	}

	return &RabbitService{
		conn:    conn,
		channel: ch,
		queue:   queueName,
	}, nil
}

// Close fecha a conexão e o canal
func (r *RabbitService) Close() {
	if r.channel != nil {
		r.channel.Close()
	}
	if r.conn != nil {
		r.conn.Close()
	}
}

// Consume inicia o consumo de mensagens
func (r *RabbitService) Consume(handler func(body []byte) error) error {
	msgs, err := r.channel.Consume(
		r.queue,
		"",    // consumer
		false, // autoAck
		false, // exclusive
		false, // noLocal
		false, // noWait
		nil,   // args
	)
	if err != nil {
		return err
	}

	go func() {
		for msg := range msgs {
			if err := handler(msg.Body); err != nil {
				log.Printf("Erro ao processar mensagem: %v", err)
			}
		}
	}()

	return nil
}
