package queue_test

import (
	"context"
	"errors"
	"testing"

	"github.com/pighead0121-make-it/go-ticket/domain"
	"github.com/pighead0121-make-it/go-ticket/queue"
	"github.com/pighead0121-make-it/go-ticket/service"
)

type mockConsumer struct {
	msgToDeliver domain.BookingMessage
}

func (m *mockConsumer) Consume(handler func(msg domain.BookingMessage) error) error {
	return handler(m.msgToDeliver)
}

func (m *mockConsumer) Close() {}

type mockWorkerRepo struct {
	purchaseErr error
	calledToken string
	calledProd  int
}

func (r *mockWorkerRepo) CreatePendingOrder(ctx context.Context, token string, productID, userID int) error {
	return nil
}
func (r *mockWorkerRepo) PurchaseWithLock(ctx context.Context, token string, productID int) error {
	r.calledToken = token
	r.calledProd = productID
	return r.purchaseErr
}
func (r *mockWorkerRepo) GetOrderStatus(ctx context.Context, token string) (domain.OrderStatus, error) {
	return "", nil
}
func (r *mockWorkerRepo) GetStock(ctx context.Context, productID int) (int, error) {
	return 0, nil
}

func TestTicketWorker_Start(t *testing.T) {
	t.Run("Successfully fulfills booking message", func(t *testing.T) {
		repo := &mockWorkerRepo{purchaseErr: nil}
		svc := service.NewTicketService(repo, nil)
		consumer := &mockConsumer{
			msgToDeliver: domain.BookingMessage{
				OrderToken: "tok-xyz",
				ProductID:  1,
				UserID:     99,
			},
		}

		worker := queue.NewTicketWorker(consumer, svc)
		if err := worker.Start(); err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
		if repo.calledToken != "tok-xyz" || repo.calledProd != 1 {
			t.Errorf("worker did not forward expected args: token=%s, prod=%d", repo.calledToken, repo.calledProd)
		}
	})

	t.Run("Propagates ErrOutOfStock when sold out", func(t *testing.T) {
		repo := &mockWorkerRepo{purchaseErr: domain.ErrOutOfStock}
		svc := service.NewTicketService(repo, nil)
		consumer := &mockConsumer{
			msgToDeliver: domain.BookingMessage{OrderToken: "tok-soldout", ProductID: 1},
		}

		worker := queue.NewTicketWorker(consumer, svc)
		err := worker.Start()
		if !errors.Is(err, domain.ErrOutOfStock) {
			t.Fatalf("expected ErrOutOfStock, got %v", err)
		}
	})
}
