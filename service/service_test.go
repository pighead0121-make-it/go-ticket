package service_test

import (
	"context"
	"errors"
	"testing"

	"github.com/pighead0121-make-it/go-ticket/domain"
	"github.com/pighead0121-make-it/go-ticket/service"
)

// Mock TicketRepository
type mockTicketRepo struct {
	createPendingOrderFunc func(ctx context.Context, token string, productID, userID int) error
	purchaseWithLockFunc   func(ctx context.Context, token string, productID int) error
	getOrderStatusFunc     func(ctx context.Context, token string) (domain.OrderStatus, error)
	getStockFunc           func(ctx context.Context, productID int) (int, error)
}

func (m *mockTicketRepo) CreatePendingOrder(ctx context.Context, token string, productID, userID int) error {
	return m.createPendingOrderFunc(ctx, token, productID, userID)
}

func (m *mockTicketRepo) PurchaseWithLock(ctx context.Context, token string, productID int) error {
	return m.purchaseWithLockFunc(ctx, token, productID)
}

func (m *mockTicketRepo) GetOrderStatus(ctx context.Context, token string) (domain.OrderStatus, error) {
	return m.getOrderStatusFunc(ctx, token)
}

func (m *mockTicketRepo) GetStock(ctx context.Context, productID int) (int, error) {
	return m.getStockFunc(ctx, productID)
}

// Mock QueueProducer
type mockQueueProducer struct {
	publishFunc func(ctx context.Context, msg domain.BookingMessage) error
}

func (m *mockQueueProducer) Publish(ctx context.Context, msg domain.BookingMessage) error {
	return m.publishFunc(ctx, msg)
}

func TestRequestBooking(t *testing.T) {
	tests := []struct {
		name      string
		repoErr   error
		pubErr    error
		wantErr   bool
		expectPub bool
	}{
		{
			name:      "Success - creates pending order and publishes message",
			repoErr:   nil,
			pubErr:    nil,
			wantErr:   false,
			expectPub: true,
		},
		{
			name:      "Failure - repo fails to create pending order (does not publish)",
			repoErr:   errors.New("db insert error"),
			pubErr:    nil,
			wantErr:   true,
			expectPub: false,
		},
		{
			name:      "Failure - queue publish fails",
			repoErr:   nil,
			pubErr:    errors.New("rabbitmq channel closed"),
			wantErr:   true,
			expectPub: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			published := false
			repo := &mockTicketRepo{
				createPendingOrderFunc: func(ctx context.Context, token string, productID, userID int) error {
					return tt.repoErr
				},
			}
			producer := &mockQueueProducer{
				publishFunc: func(ctx context.Context, msg domain.BookingMessage) error {
					published = true
					if msg.OrderToken != "tok-123" || msg.ProductID != 1 || msg.UserID != 42 {
						t.Errorf("unexpected message payload: %+v", msg)
					}
					return tt.pubErr
				},
			}

			svc := service.NewTicketService(repo, producer)
			err := svc.RequestBooking(context.Background(), "tok-123", 1, 42)

			if (err != nil) != tt.wantErr {
				t.Fatalf("expected error = %v, got %v", tt.wantErr, err)
			}
			if published != tt.expectPub {
				t.Errorf("expected published = %v, got %v", tt.expectPub, published)
			}
		})
	}
}

func TestFulfillBooking(t *testing.T) {
	repo := &mockTicketRepo{
		purchaseWithLockFunc: func(ctx context.Context, token string, productID int) error {
			if productID == 99 {
				return domain.ErrOutOfStock
			}
			return nil
		},
	}
	svc := service.NewTicketService(repo, nil)

	if err := svc.FulfillBooking(context.Background(), "tok-1", 1); err != nil {
		t.Errorf("expected nil error, got %v", err)
	}

	if err := svc.FulfillBooking(context.Background(), "tok-2", 99); !errors.Is(err, domain.ErrOutOfStock) {
		t.Errorf("expected ErrOutOfStock, got %v", err)
	}
}

func TestQueryStatusAndStock(t *testing.T) {
	repo := &mockTicketRepo{
		getOrderStatusFunc: func(ctx context.Context, token string) (domain.OrderStatus, error) {
			if token == "valid-token" {
				return domain.StatusSuccess, nil
			}
			return "", domain.ErrOrderNotFound
		},
		getStockFunc: func(ctx context.Context, productID int) (int, error) {
			return 10, nil
		},
	}
	svc := service.NewTicketService(repo, nil)

	status, err := svc.QueryStatus(context.Background(), "valid-token")
	if err != nil || status != domain.StatusSuccess {
		t.Errorf("expected status %v, got %v (err: %v)", domain.StatusSuccess, status, err)
	}

	stock, err := svc.QueryStock(context.Background(), 1)
	if err != nil || stock != 10 {
		t.Errorf("expected stock 10, got %d (err: %v)", stock, err)
	}
}
