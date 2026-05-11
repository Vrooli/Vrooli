package main

import (
	"bytes"
	"context"
	"os"
	"testing"

	"react-vite-temporal-model/internal/cli"
)

func TestIntegrationRealQuintCheck(t *testing.T) {
	if os.Getenv("VROOLI_TEMPORAL_MODEL_INTEGRATION") != "1" {
		t.Skip("set VROOLI_TEMPORAL_MODEL_INTEGRATION=1 to run real Quint validation")
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if err := cli.Run(context.Background(), []string{"check", "--root", "../.."}, &stdout, &stderr); err != nil {
		t.Fatalf("real Quint check failed: %v\nstdout:\n%s\nstderr:\n%s", err, stdout.String(), stderr.String())
	}
}
