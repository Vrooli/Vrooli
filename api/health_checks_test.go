//go:build testing
// +build testing

package main

import (
	"context"
	"testing"
)

func TestPerformHealthCheck_PostgresSupported(t *testing.T) {
	origLookPath := lookPathFn
	origCommand := commandFn
	t.Cleanup(func() {
		lookPathFn = origLookPath
		commandFn = origCommand
	})

	lookPathFn = func(file string) (string, error) {
		return "/usr/bin/vrooli", nil
	}
	commandFn = func(ctx context.Context, name string, args ...string) ([]byte, error) {
		return []byte(`{"installed":true,"running":true,"healthy":true}`), nil
	}

	check := HealthCheckConfig{
		Name:     "postgres_connection",
		Type:     "postgres",
		Target:   "vrooli",
		Critical: true,
		Timeout:  3000,
	}

	if err := performHealthCheck(check, "prd-control-tower", map[string]int{}); err != nil {
		t.Fatalf("expected postgres health checks to be supported, got error: %v", err)
	}
}
