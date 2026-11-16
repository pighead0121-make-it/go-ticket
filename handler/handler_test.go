package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pighead0121-make-it/go-ticket/domain"
	"github.com/pighead0121-make-it/go-ticket/handler"
	"github.com/pighead0121-make-it/go-ticket/service"
)

type stubRepo struct {
	createErr error
	status    domain.OrderStatus
	statusErr error
}

func (s *stubRepo) CreatePendingOrder(ctx context.Context, token string, productID, userID int) error {
	return s.createErr
}
func (s *stubRepo) PurchaseWithLock(ctx context.Context, token string, productID int) error {
	return nil
}
func (s *stubRepo) GetOrderStatus(ctx context.Context, token string) (domain.OrderStatus, error) {
	return s.status, s.statusErr
}
func (s *stubRepo) GetStock(ctx context.Context, productID int) (int, error) {
	return 0, nil
}

type stubProducer struct {
	pubErr error
}

func (p *stubProducer) Publish(ctx context.Context, msg domain.BookingMessage) error {
	return p.pubErr
}

func TestCreateOrder(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		body       string
		repoErr    error
		pubErr     error
		wantStatus int
	}{
		{
			name:       "202 Accepted - valid order request",
			method:     http.MethodPost,
			body:       `{"product_id": 1, "user_id": 100}`,
			wantStatus: http.StatusAccepted,
		},
		{
			name:       "405 Method Not Allowed - GET instead of POST",
			method:     http.MethodGet,
			body:       ``,
			wantStatus: http.StatusMethodNotAllowed,
		},
		{
			name:       "400 Bad Request - malformed JSON",
			method:     http.MethodPost,
			body:       `{"product_id": "invalid"}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "400 Bad Request - zero or negative IDs",
			method:     http.MethodPost,
			body:       `{"product_id": 0, "user_id": -1}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "503 Service Unavailable - service failure",
			method:     http.MethodPost,
			body:       `{"product_id": 1, "user_id": 100}`,
			repoErr:    errors.New("db down"),
			wantStatus: http.StatusServiceUnavailable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := service.NewTicketService(&stubRepo{createErr: tt.repoErr}, &stubProducer{pubErr: tt.pubErr})
			h := handler.NewHTTPHandler(svc)

			req := httptest.NewRequest(tt.method, "/api/order", bytes.NewBufferString(tt.body))
			rec := httptest.NewRecorder()

			h.CreateOrder(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("expected status %d, got %d (body: %s)", tt.wantStatus, rec.Code, rec.Body.String())
			}

			if tt.wantStatus == http.StatusAccepted {
				var resp map[string]string
				if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
					t.Fatalf("failed to decode response JSON: %v", err)
				}
				if resp["status"] != "queued" || resp["order_token"] == "" {
					t.Errorf("unexpected response body: %+v", resp)
				}
			}
		})
	}
}

func TestGetOrderStatus(t *testing.T) {
	tests := []struct {
		name       string
		query      string
		mockStatus domain.OrderStatus
		mockErr    error
		wantStatus int
	}{
		{
			name:       "200 OK - valid token returns status",
			query:      "?token=abc-123",
			mockStatus: domain.StatusSuccess,
			wantStatus: http.StatusOK,
		},
		{
			name:       "400 Bad Request - missing token query param",
			query:      "",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "500 Internal Server Error - repo query fails",
			query:      "?token=missing-tok",
			mockErr:    domain.ErrOrderNotFound,
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := service.NewTicketService(&stubRepo{status: tt.mockStatus, statusErr: tt.mockErr}, &stubProducer{})
			h := handler.NewHTTPHandler(svc)

			req := httptest.NewRequest(http.MethodGet, "/api/order/status"+tt.query, nil)
			rec := httptest.NewRecorder()

			h.GetOrderStatus(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("expected status %d, got %d", tt.wantStatus, rec.Code)
			}
		})
	}
}
