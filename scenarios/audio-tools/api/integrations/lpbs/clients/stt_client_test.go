package clients

import (
	"context"
	"strings"
	"testing"

	"audio-tools/internal/ai/sttchain"
)

func TestSTTClient_StubContract(t *testing.T) {
	c := NewSTTClient("http://lpbs.test")
	if c.BaseURL != "http://lpbs.test" {
		t.Fatalf("BaseURL not stored")
	}
	if c.HTTPClient == nil {
		t.Fatalf("HTTPClient missing default")
	}
	if c.IsAvailable(context.Background()) {
		t.Fatalf("expected IsAvailable=false until gateway lands")
	}
	if c.Model() == "" {
		t.Fatalf("Model() must not be empty")
	}
	_, err := c.Transcribe(context.Background(), "tok", "user", sttchain.Request{})
	if err == nil || !strings.Contains(err.Error(), "not implemented") {
		t.Fatalf("expected not-implemented error, got %v", err)
	}
}
