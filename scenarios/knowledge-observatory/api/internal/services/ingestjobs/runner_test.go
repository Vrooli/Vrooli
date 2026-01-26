package ingestjobs

import (
	"context"
	"testing"
)

func TestNormalizeTags(t *testing.T) {
	out := normalizeTags([]string{" alpha ", "", "beta", "alpha", "beta"})
	if len(out) != 2 {
		t.Fatalf("expected 2 unique tags, got %v", out)
	}
	if out[0] != "alpha" || out[1] != "beta" {
		t.Fatalf("unexpected tag normalization: %v", out)
	}
}

func TestProcessOneRequiresConfig(t *testing.T) {
	runner := &Runner{}
	if _, err := runner.processOne(context.Background()); err == nil {
		t.Fatalf("expected configuration error")
	}
}
