package testdb

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// Handle owns the lifecycle of a Postgres test database for package tests.
type Handle struct {
	DSN        string
	container  testcontainers.Container
	externalDS bool
}

// Start boots a Postgres test database (or reuses TEST_DATABASE_URL when provided).
// Call Terminate when the package test run finishes.
func Start(ctx context.Context) (*Handle, error) {
	if url := strings.TrimSpace(os.Getenv("TEST_DATABASE_URL")); url != "" {
		return &Handle{DSN: url, externalDS: true}, nil
	}

	container, err := postgres.RunContainer(ctx,
		testcontainers.WithImage("postgres:15-alpine"),
		postgres.WithDatabase("bas_test"),
		postgres.WithUsername("test"),
		postgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("start postgres testcontainer: %w", err)
	}

	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		_ = container.Terminate(ctx)
		return nil, fmt.Errorf("get connection string: %w", err)
	}

	return &Handle{DSN: dsn, container: container}, nil
}

// Terminate stops the underlying container when owned by this handle.
func (h *Handle) Terminate(ctx context.Context) {
	if h == nil || h.externalDS || h.container == nil {
		return
	}
	_ = h.container.Terminate(ctx)
}
