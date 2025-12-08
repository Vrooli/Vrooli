package testdb

import (
	"context"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"github.com/testcontainers/testcontainers-go"
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

	req := testcontainers.ContainerRequest{
		Image:        "postgres:15-alpine",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_DB":       "bas_test",
			"POSTGRES_USER":     "test",
			"POSTGRES_PASSWORD": "test",
		},
		WaitingFor: wait.ForLog("database system is ready to accept connections").
			WithOccurrence(2).
			WithStartupTimeout(60 * time.Second),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, fmt.Errorf("start postgres testcontainer: %w", err)
	}

	host, err := container.Host(ctx)
	if err != nil {
		_ = container.Terminate(ctx)
		return nil, fmt.Errorf("get host: %w", err)
	}

	mapped, err := container.MappedPort(ctx, "5432/tcp")
	if err != nil {
		_ = container.Terminate(ctx)
		return nil, fmt.Errorf("get mapped port: %w", err)
	}

	dsn := fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=disable", "test", "test", net.JoinHostPort(host, mapped.Port()), "bas_test")

	return &Handle{DSN: dsn, container: container}, nil
}

// Terminate stops the underlying container when owned by this handle.
func (h *Handle) Terminate(ctx context.Context) {
	if h == nil || h.externalDS || h.container == nil {
		return
	}
	_ = h.container.Terminate(ctx)
}
