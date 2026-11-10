package queue

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/pighead0121-make-it/go-ticket/domain"
	"github.com/pighead0121-make-it/go-ticket/service"
)

type TicketWorker struct {
	consumer domain.QueueConsumer
	service  *service.TicketService
}

func NewTicketWorker(consumer domain.QueueConsumer, service *service.TicketService) *TicketWorker {
	return &TicketWorker{
		consumer: consumer,
		service:  service,
	}
}

func (w *TicketWorker) Start() error {
	log.Println("TicketWorker is running...")
	return w.consumer.Consume(func(msg domain.BookingMessage) error {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		err := w.service.FulfillBooking(ctx, msg.OrderToken, msg.ProductID)
		if err != nil {
			if errors.Is(err, domain.ErrOutOfStock) {
				log.Printf("[Workder] out of stock (Token: %s)", msg.OrderToken)
			} else {
				log.Printf("[Worker] failed to process order: %v", err)
			}
			return err
		}

		log.Printf("[Worker] booking successful! (Token: %s)", msg.OrderToken)
		return nil
	})
}
