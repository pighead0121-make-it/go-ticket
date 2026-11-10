package repository

import (
	"context"
	"errors"

	"github.com/pighead0121-make-it/go-ticket/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Connection Setup

func NewPostgresPool(ctx context.Context, connStr string) (*pgxpool.Pool, error) {
	config, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		return nil, err
	}
	config.MaxConns = 25
	config.MinConns = 5
	return pgxpool.NewWithConfig(ctx, config)
}

func InitSchema(ctx context.Context, pool *pgxpool.Pool) error {
	queries := []string{
		`DROP TABLE IF EXISTS orders;`,
		`DROP TABLE IF EXISTS products;`,
		`CREATE TABLE products (
			id SERIAL PRIMARY KEY,
			name VARCHAR(50) NOT NULL,
			stock INT NOT NULL CHECK (stock >= 0)
		);`,
		`CREATE TABLE orders (
			id SERIAL PRIMARY KEY,
			order_token VARCHAR(64) UNIQUE NOT NULL,
			product_id INT NOT NULL,
			user_id INT NOT NULL,
			status VARCHAR(20) NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);`,
		`INSERT INTO products (id, name, stock) VALUES (1, '2026 meme cat world tour', 10);`,
	}

	for _, q := range queries {
		if _, err := pool.Exec(ctx, q); err != nil {
			return err
		}
	}
	return nil
}

// SQL Operations

type PostgresTicketRepo struct {
	pool *pgxpool.Pool
}

func NewPostgresTicketRepo(pool *pgxpool.Pool) *PostgresTicketRepo {
	return &PostgresTicketRepo{pool: pool}
}

func (r *PostgresTicketRepo) CreatePendingOrder(ctx context.Context, token string, productID, userID int) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO orders (order_token, product_id, user_id, status)
		VALUES ($1, $2, $3, $4)`,
		token, productID, userID, domain.StatusPending)
	return err
}

func (r *PostgresTicketRepo) PurchaseWithLock(ctx context.Context, token string, productID int) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var stock int
	err = tx.QueryRow(ctx,
		`SELECT stock FROM products WHERE id = $1 FOR UPDATE`,
		productID).Scan(&stock)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrTicketNotFound
		}
		return err
	}

	if stock <= 0 {
		_, _ = tx.Exec(ctx,
			`UPDATE orders SET status = $1 WHERE order_token = $2`,
			domain.StatusSoldOut, token)
		_ = tx.Commit(ctx)
		return domain.ErrOutOfStock
	}

	_, err = tx.Exec(ctx,
		`UPDATE products SET stock = stock - 1 WHERE id = $1`,
		productID)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx,
		`UPDATE orders SET status = $1 WHERE order_token = $2`,
		domain.StatusSuccess, token)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (r *PostgresTicketRepo) GetOrderStatus(ctx context.Context, token string) (domain.OrderStatus, error) {
	var status string
	err := r.pool.QueryRow(ctx,
		`SELECT status FROM orders WHERE order_token = $1`,
		token).Scan(&status)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", nil
		}
		return "", nil
	}
	return domain.OrderStatus(status), nil
}

func (r *PostgresTicketRepo) GetStock(ctx context.Context, productID int) (int, error) {
	var stock int
	err := r.pool.QueryRow(ctx,
		`SELECT stock FROM products WHERE id = $1`,
		productID).Scan(&stock)
	return stock, err
}
