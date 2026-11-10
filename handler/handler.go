package handler

import (
	"encoding/json"
	"net/http"

	"github.com/pighead0121-make-it/go-ticket/domain"
	"github.com/pighead0121-make-it/go-ticket/service"

	"github.com/google/uuid"
)

type HTTPHandler struct {
	service *service.TicketService
}

func NewHTTPHandler(service *service.TicketService) *HTTPHandler {
	return &HTTPHandler{service: service}
}

func (h *HTTPHandler) CreateOrder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "only POST is supported", http.StatusMethodNotAllowed)
		return
	}

	var req domain.OrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	if req.ProductID <= 0 || req.UserID <= 0 {
		http.Error(w, "product_id & user_id connot be empty", http.StatusBadRequest)
		return
	}

	token := uuid.New().String()
	if err := h.service.RequestBooking(r.Context(), token, req.ProductID, req.UserID); err != nil {
		http.Error(w, "the service is busy. please try again later", http.StatusServiceUnavailable)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{
		"status":      "queued",
		"order_token": token,
		"message":     "queued. please check results via token.",
	})
}

func (h *HTTPHandler) GetOrderStatus(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		http.Error(w, "token connot be empty", http.StatusBadRequest)
		return
	}

	status, err := h.service.QueryStatus(r.Context(), token)
	if err != nil {
		http.Error(w, "query failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Coontent-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"order_token": token,
		"status":      status,
	})
}
