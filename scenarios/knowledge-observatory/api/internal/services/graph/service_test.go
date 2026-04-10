package graph

import (
	"context"
	"testing"
)

func TestNormalizeAndValidateDefaults(t *testing.T) {
	req := Request{Center: "  root ", Depth: 0, Limit: 0, Threshold: 0}
	if err := req.NormalizeAndValidate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if req.Center != "root" {
		t.Fatalf("expected trimmed center, got %q", req.Center)
	}
	if req.Depth != defaultDepth {
		t.Fatalf("expected default depth %d, got %d", defaultDepth, req.Depth)
	}
	if req.Limit != defaultLimit {
		t.Fatalf("expected default limit %d, got %d", defaultLimit, req.Limit)
	}
	if req.Threshold != defaultThreshold {
		t.Fatalf("expected default threshold %v, got %v", defaultThreshold, req.Threshold)
	}
}

func TestNormalizeAndValidateRequiresCenter(t *testing.T) {
	req := Request{Center: " "}
	if err := req.NormalizeAndValidate(); err == nil {
		t.Fatalf("expected error for empty center")
	}
}

func TestExtractContentFromPayload(t *testing.T) {
	if got := extractContentFromPayload(nil); got != "" {
		t.Fatalf("expected empty content for nil payload")
	}
	if got := extractContentFromPayload(map[string]interface{}{"content": "alpha"}); got != "alpha" {
		t.Fatalf("expected content value, got %q", got)
	}
	if got := extractContentFromPayload(map[string]interface{}{"text": "beta"}); got != "beta" {
		t.Fatalf("expected text value, got %q", got)
	}
}

func TestGraphRequiresDependencies(t *testing.T) {
	service := &Service{}
	if _, err := service.Graph(context.Background(), Request{Center: "root"}); err == nil {
		t.Fatalf("expected dependency error")
	}
}
