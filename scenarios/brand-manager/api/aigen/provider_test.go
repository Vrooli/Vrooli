package aigen

import (
	"context"
	"fmt"
	"testing"
)

// mockProvider implements Provider for testing. [REQ:BM-REQ-AI-CHAIN]
type mockProvider struct {
	name      string
	available bool
	textResp  *TextResponse
	textErr   error
	imageResp *ImageResponse
	imageErr  error
}

func (m *mockProvider) Name() string                     { return m.name }
func (m *mockProvider) Available(_ context.Context) bool { return m.available }
func (m *mockProvider) GenerateText(_ context.Context, _ TextRequest) (*TextResponse, error) {
	return m.textResp, m.textErr
}
func (m *mockProvider) GenerateImage(_ context.Context, _ ImageRequest) (*ImageResponse, error) {
	return m.imageResp, m.imageErr
}

// TestChainFallback verifies the chain tries providers in order. [REQ:BM-REQ-AI-CHAIN]
func TestChainFallback(t *testing.T) {
	failing := &mockProvider{name: "failing", available: true, textErr: fmt.Errorf("fail")}
	working := &mockProvider{
		name:      "working",
		available: true,
		textResp:  &TextResponse{Text: "ok", Provider: "working", Model: "test"},
	}

	chain := NewChain(failing, working)
	resp, err := chain.GenerateText(context.Background(), TextRequest{Prompt: "test"})
	if err != nil {
		t.Fatalf("expected success, got: %v", err)
	}
	if resp.Provider != "working" {
		t.Errorf("expected provider 'working', got %q", resp.Provider)
	}
}

// TestChainAllFail verifies error when all providers fail. [REQ:BM-REQ-AI-CHAIN]
func TestChainAllFail(t *testing.T) {
	p1 := &mockProvider{name: "p1", available: true, textErr: fmt.Errorf("err1")}
	p2 := &mockProvider{name: "p2", available: true, textErr: fmt.Errorf("err2")}

	chain := NewChain(p1, p2)
	_, err := chain.GenerateText(context.Background(), TextRequest{Prompt: "test"})
	if err == nil {
		t.Fatal("expected error when all providers fail")
	}
}

// TestChainSkipsUnavailable verifies unavailable providers are skipped. [REQ:BM-REQ-AI-CHAIN]
func TestChainSkipsUnavailable(t *testing.T) {
	unavail := &mockProvider{name: "unavail", available: false}
	working := &mockProvider{
		name:      "avail",
		available: true,
		textResp:  &TextResponse{Text: "ok", Provider: "avail", Model: "m"},
	}

	chain := NewChain(unavail, working)
	resp, err := chain.GenerateText(context.Background(), TextRequest{Prompt: "test"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Provider != "avail" {
		t.Errorf("expected provider 'avail', got %q", resp.Provider)
	}
}

// TestChainAvailable checks the Available method. [REQ:BM-REQ-AI-CHAIN]
func TestChainAvailable(t *testing.T) {
	t.Run("no providers", func(t *testing.T) {
		chain := NewChain()
		if chain.Available(context.Background()) {
			t.Error("empty chain should not be available")
		}
	})
	t.Run("one available", func(t *testing.T) {
		chain := NewChain(
			&mockProvider{name: "no", available: false},
			&mockProvider{name: "yes", available: true},
		)
		if !chain.Available(context.Background()) {
			t.Error("chain with one available provider should be available")
		}
	})
}

// TestChainImageFallback verifies image generation fallback. [REQ:BM-REQ-AI-IMAGE]
func TestChainImageFallback(t *testing.T) {
	noImage := &mockProvider{name: "text-only", available: true, imageErr: fmt.Errorf("no images")}
	hasImage := &mockProvider{
		name:      "image-provider",
		available: true,
		imageResp: &ImageResponse{Data: []byte("png"), MimeType: "image/png", Provider: "image-provider", Model: "dalle"},
	}

	chain := NewChain(noImage, hasImage)
	resp, err := chain.GenerateImage(context.Background(), ImageRequest{Prompt: "logo"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Provider != "image-provider" {
		t.Errorf("expected provider 'image-provider', got %q", resp.Provider)
	}
}

// TestChainProviders verifies Providers() returns the list. [REQ:BM-REQ-AI-CHAIN]
func TestChainProviders(t *testing.T) {
	p1 := &mockProvider{name: "a"}
	p2 := &mockProvider{name: "b"}
	chain := NewChain(p1, p2)
	if len(chain.Providers()) != 2 {
		t.Errorf("expected 2 providers, got %d", len(chain.Providers()))
	}
}
