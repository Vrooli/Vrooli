package capacitysync

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"
)

func TestConfigUsesMeasuredRAMClaim(t *testing.T) {
	h := &Handlers{Stderr: io.Discard, Exec: func(context.Context, string, ...string) ([]byte, error) { return nil, nil }, Interval: time.Second}
	cfg := h.config()
	if cfg.Resource != resourceName || cfg.ResourceKind != "ram" {
		t.Fatalf("resource identity = %q/%q, want %q/ram", cfg.Resource, cfg.ResourceKind, resourceName)
	}
	if cfg.PreferredBytes != defaultPreferred || cfg.FloorBytes != defaultPreferred {
		t.Fatalf("RAM reservation = %d/%d, want %d/%d", cfg.PreferredBytes, cfg.FloorBytes, defaultPreferred, defaultPreferred)
	}
}

func TestObserveUsesOnlyTheUnstructuredRAMClaim(t *testing.T) {
	var calls []string
	h := &Handlers{Exec: func(_ context.Context, name string, args ...string) ([]byte, error) {
		calls = append(calls, name+" "+strings.Join(args, " "))
		return []byte(`{"claims":[{"owner_id":"other","owner_kind":"resource","resource_kind":"ram","amount_bytes":10},{"owner_id":"unstructured-io","owner_kind":"resource","resource_kind":"vram","amount_bytes":20},{"owner_id":"unstructured-io","owner_kind":"resource","resource_kind":"ram","amount_bytes":30,"observed_bytes":40}]}`), nil
	}}
	got, err := h.Observe(context.Background())
	if err != nil {
		t.Fatalf("Observe: %v", err)
	}
	if got.Bytes != 40 {
		t.Fatalf("footprint = %d, want observed RAM 40", got.Bytes)
	}
	if len(calls) != 1 || calls[0] != "vrooli capacity list --owner unstructured-io --active --json" {
		t.Fatalf("calls = %v, want one read-only capacity-list request", calls)
	}
}

func TestObserveDoesNotCreateClaimAndPreservesReadFailure(t *testing.T) {
	h := &Handlers{Exec: func(context.Context, string, ...string) ([]byte, error) {
		return []byte(`{"claims":[]}`), nil
	}}
	got, err := h.Observe(context.Background())
	if err != nil || got.Bytes != 0 {
		t.Fatalf("Observe without claim = %+v, %v; want zero without error", got, err)
	}

	wantErr := errors.New("capacity unavailable")
	h.Exec = func(context.Context, string, ...string) ([]byte, error) { return nil, wantErr }
	if _, err := h.Observe(context.Background()); !errors.Is(err, wantErr) {
		t.Fatalf("Observe error = %v, want %v", err, wantErr)
	}
}
