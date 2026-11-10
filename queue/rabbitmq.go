package queue

import (
	"context"
	"encoding/json"

	"github.com/pighead0121-make-it/go-ticket/domain"

	amqp "github.com/rabbitmq/amqp091-go"
)

const TicketQueueName = "ticket_orders"

type RabbitQueue struct {
	conn *amqp.Connection
	ch   *amqp.Channel
}

func NewRabbitQueue(connStr string) (*RabbitQueue, error) {
	conn, err := amqp.Dial(connStr)
	if err != nil {
		return nil, err
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, err
	}

	_, err = ch.QueueDeclare(
		TicketQueueName,
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		ch.Close()
		conn.Close()
		return nil, err
	}

	return &RabbitQueue{conn: conn, ch: ch}, nil
}

func (r *RabbitQueue) Close() {
	if r.ch != nil {
		r.ch.Close()
	}
	if r.conn != nil {
		r.conn.Close()
	}
}

func (r *RabbitQueue) Publish(ctx context.Context, msg domain.BookingMessage) error {
	body, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	return r.ch.PublishWithContext(ctx,
		"",
		TicketQueueName,
		false,
		false,
		amqp.Publishing{
			DeliveryMode: amqp.Persistent,
			ContentType:  "application/json",
			Body:         body,
		})
}

func (r *RabbitQueue) Consume(handler func(msg domain.BookingMessage) error) error {
	if err := r.ch.Qos(1, 0, false); err != nil {
		return err
	}

	msgs, err := r.ch.Consume(
		TicketQueueName,
		"",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	go func() {
		for d := range msgs {
			var msg domain.BookingMessage
			if err := json.Unmarshal(d.Body, &msg); err != nil {
				d.Nack(false, false)
				continue
			}

			_ = handler(msg)
			d.Ack(false)
		}
	}()

	return nil
}
