package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/pighead0121-make-it/go-ticket/handler"
	"github.com/pighead0121-make-it/go-ticket/queue"
	"github.com/pighead0121-make-it/go-ticket/repository"
	"github.com/pighead0121-make-it/go-ticket/service"
)

func main() {
	ctx := context.Background()

	// Dependency Initialization
	pgPool, err := repository.NewPostgresPool(ctx, "postgres://user:password@localhost:5432/ticket_db?sslmode=disable")
	if err != nil {
		log.Fatalf("postgreSQL connection failed: %v", err)
	}
	defer pgPool.Close()

	if err := repository.InitSchema(ctx, pgPool); err != nil {
		log.Fatalf("schema initialization failed: %v", err)
	}

	rabbitMQ, err := queue.NewRabbitQueue("amqp://guest:guest@localhost:5672/")
	if err != nil {
		log.Fatalf("rabbitMQ connection failed: %v", err)
	}
	defer rabbitMQ.Close()

	// Dependency Injection
	repo := repository.NewPostgresTicketRepo(pgPool)
	ticketService := service.NewTicketService(repo, rabbitMQ)

	// Start Background Worker
	ticketWorker := queue.NewTicketWorker(rabbitMQ, ticketService)
	if err := ticketWorker.Start(); err != nil {
		log.Fatalf("failed to sstart worker: %v", err)
	}

	// Start HTTP Server
	httpHandler := handler.NewHTTPHandler(ticketService)
	mux := http.NewServeMux()
	mux.HandleFunc("/api/order", httpHandler.CreateOrder)
	mux.HandleFunc("/api/order/status", httpHandler.GetOrderStatus)

	server := &http.Server{
		Addr:    ":3000",
		Handler: mux,
	}

	go func() {
		log.Printf("server is running on: http://localhost%s\n", server.Addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("HTTP server failed: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("shutdown signal received. closing cleanly...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("force quit server: %v", err)
	}
	log.Println("safe shutdown")
}
