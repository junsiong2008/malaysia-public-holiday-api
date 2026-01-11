package db

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

var Pool *pgxpool.Pool

func Connect() error {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return fmt.Errorf("DATABASE_URL environment variable is not set")
	}

	config, err := pgxpool.ParseConfig(dbURL)
	if err != nil {
		return fmt.Errorf("unable to parse database config: %w", err)
	}

	config.MaxConns = 10
	config.MinConns = 2
	config.MaxConnLifetime = time.Hour
	config.MaxConnIdleTime = 30 * time.Minute

	Pool, err = pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		return fmt.Errorf("unable to create connection pool: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := Pool.Ping(ctx); err != nil {
		return fmt.Errorf("unable to connect to database: %w", err)
	}

	fmt.Println("Connected to database successfully")
	return nil
}

func Close() {
	if Pool != nil {
		Pool.Close()
	}
}
