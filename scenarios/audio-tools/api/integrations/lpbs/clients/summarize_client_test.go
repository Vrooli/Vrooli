package clients

import (
	"context"
	"strings"
	"testing"

	"audio-tools/internal/ai/summarizechain"
)

func TestSummarizeClient_StubContract(t *testing.T) {
	c := NewSummarizeClient("http://lpbs.test")
	if c.BaseURL != "http://lpbs.test" {
		t.Fatalf("BaseURL not stored")
	}
	if c.IsAvailable(context.Background()) {
		t.Fatalf("expected IsAvailable=false until gateway lands")
	}
	if c.Model() == "" {
		t.Fatalf("Model() must not be empty")
	}
	_, err := c.Summarize(context.Background(), "tok", "user", summarizechain.Request{})
	if err == nil || !strings.Contains(err.Error(), "not implemented") {
		t.Fatalf("expected not-implemented error, got %v", err)
	}
}
