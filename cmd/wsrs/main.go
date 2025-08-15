package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"

	"github.com/rocketseat-education/semana-tech-go-react-server/internal/api"
	"github.com/rocketseat-education/semana-tech-go-react-server/internal/logger"
	"github.com/rocketseat-education/semana-tech-go-react-server/internal/store/pgstore"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

func main() {
	ctx := context.Background()

	if err := godotenv.Load(); err != nil {
		logger.Default.Fatal(ctx, "failed to load environment variables", "error", err)
	}

	logger.Default.Info(ctx, "starting application")

	pool, err := pgxpool.New(ctx, fmt.Sprintf(
		"user=%s password=%s host=%s port=%s dbname=%s",
		os.Getenv("WSRS_DATABASE_USER"),
		os.Getenv("WSRS_DATABASE_PASSWORD"),
		os.Getenv("WSRS_DATABASE_HOST"),
		os.Getenv("WSRS_DATABASE_PORT"),
		os.Getenv("WSRS_DATABASE_NAME"),
	))
	if err != nil {
		logger.Default.Fatal(ctx, "failed to create database connection pool", "error", err)
	}

	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		logger.Default.Fatal(ctx, "failed to ping database", "error", err)
	}

	logger.Default.Info(ctx, "database connection established")

	handler := api.NewHandler(pgstore.New(pool))

	logger.Default.Info(ctx, "starting HTTP server", "port", 8080)

	go func() {
		if err := http.ListenAndServe(":8080", handler); err != nil {
			if !errors.Is(err, http.ErrServerClosed) {
				logger.Default.Fatal(ctx, "HTTP server error", "error", err)
			}
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit

	logger.Default.Info(ctx, "shutting down application")
}
