package domain

import (
	"context"
	"errors"
)

// Business Errors

var (
	ErrOutOfStock     = errors.New("out of stock")
	ErrTicketNotFound = errors.New("ticket not found")
)

// Entities

type OrderStatus string

const (
	StatusPending OrderStatus = "pending"
	StatusSuccess OrderStatus = "success"
	StatusSoldOut OrderStatus = "sold_out"
)

type Product struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Stock int    `json:"stock"`
}

type Order struct {
	ID         int         `json:"id"`
	OrderToken string      `json:"order_token"`
	ProductID  int         `json:"product_id"`
	UserID     int         `json:"user_id"`
	Status     OrderStatus `json:"status"`
}

// Queue Mwssage
type BookingMessage struct {
	ProductID  int    `json:"product_id"`
	UserID     int    `json:"user_id"`
	OrderToken string `json:"order_token"`
}

// HTTP Requst
type OrderRequest struct {
	ProductID int `json:"product_id"`
	UserID    int `json:"user_id"`
}

// Interface

// Repository
type TicketRepository interface {
	CreatePendingOrder(ctx context.Context, token string, productID, userID int) error
	PurchaseWithLock(ctx context.Context, token string, productID int) error
	GetOrderStatus(ctx context.Context, token string) (OrderStatus, error)
	GetStock(ctx context.Context, productID int) (int, error)
}

// Queue
type QueueProducer interface {
	Publish(ctx context.Context, msq BookingMessage) error
}

type QueueConsumer interface {
	Consume(handler func(msg BookingMessage) error) error
	Close()
}
