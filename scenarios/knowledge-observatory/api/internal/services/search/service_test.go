package search

import (
	"context"
	"testing"
)

func TestNormalizeAndValidateDefaults(t *testing.T) {
	req := Request{Query: "  hello ", Limit: 0, Threshold: 0}
	if err := req.NormalizeAndValidate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if req.Query != "hello" {
		t.Fatalf("expected trimmed query, got %q", req.Query)
	}
	if req.Limit != defaultSearchLimit {
		t.Fatalf("expected default limit %d, got %d", defaultSearchLimit, req.Limit)
	}
	if req.Threshold != defaultSearchThreshold {
		t.Fatalf("expected default threshold %v, got %v", defaultSearchThreshold, req.Threshold)
	}
}

func TestNormalizeAndValidateRequiresQuery(t *testing.T) {
	req := Request{Query: " "}
	if err := req.NormalizeAndValidate(); err == nil {
		t.Fatalf("expected error for empty query")
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

func TestSearchRequiresDependencies(t *testing.T) {
	service := &Service{}
	if _, err := service.Search(context.Background(), Request{Query: "hello"}); err == nil {
		t.Fatalf("expected dependency error")
	}
}
