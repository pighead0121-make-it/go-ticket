package service

import (
	"context"

	"github.com/pighead0121-make-it/go-ticket/domain"
)

type TicketService struct {
	repo     domain.TicketRepository
	producer domain.QueueProducer
}

func NewTicketService(repo domain.TicketRepository, producer domain.QueueProducer) *TicketService {
	return &TicketService{
		repo:     repo,
		producer: producer,
	}
}

func (s *TicketService) RequestBooking(ctx context.Context, token string, productID, userID int) error {
	if err := s.repo.CreatePendingOrder(ctx, token, productID, userID); err != nil {
		return err
	}

	return s.producer.Publish(ctx, domain.BookingMessage{
		ProductID:  productID,
		UserID:     userID,
		OrderToken: token,
	})
}

func (s *TicketService) FulfillBooking(ctx context.Context, token string, productID int) error {
	return s.repo.PurchaseWithLock(ctx, token, productID)
}

func (s *TicketService) QueryStatus(ctx context.Context, token string) (domain.OrderStatus, error) {
	return s.repo.GetOrderStatus(ctx, token)
}

func (s *TicketService) QueryStock(ctx context.Context, productID int) (int, error) {
	return s.repo.GetStock(ctx, productID)
}
