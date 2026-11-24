package main

import (
	"context"
	"errors"
	"io"
	"log"
	"testing"
	"time"

	"tunnel-manager/internal/config"
)

type recoveringBootstrap struct{ calls int }

func (b *recoveringBootstrap) BootstrapConfiguredCloudflare(context.Context) (config.BootstrapResult, error) {
	b.calls++
	if b.calls == 1 {
		return config.BootstrapResult{}, errors.New("temporarily unavailable")
	}
	return config.BootstrapResult{Written: true}, nil
}

func TestDerivedCredentialBootstrapRetriesTransientFailure(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	b := &recoveringBootstrap{}
	runDerivedCredentialBootstrap(ctx, b, log.New(io.Discard, "", 0), time.Millisecond)
	if b.calls != 2 {
		t.Fatalf("bootstrap attempts = %d, want 2", b.calls)
	}
}
