// Package aigen implements the AIProviderChain pattern for brand element generation.
// It provides a unified interface over multiple AI backends (Ollama, OpenRouter)
// with automatic fallback: Ollama is tried first (local, free), then OpenRouter (cloud).
//
// [REQ:BM-REQ-AI-CHAIN]
// DOC: docs/concepts/ARCHITECTURE.md#ai-generation
package aigen

import (
	"context"
	"fmt"
	"strings"
)

// Provider is the interface that all AI backends must implement.
// [REQ:BM-REQ-AI-CHAIN]
type Provider interface {
	// Name returns the provider identifier (e.g. "ollama", "openrouter").
	Name() string
	// Available reports whether the provider is currently reachable.
	Available(ctx context.Context) bool
	// GenerateText sends a prompt and returns the text response.
	// [REQ:BM-REQ-AI-TEXT]
	GenerateText(ctx context.Context, req TextRequest) (*TextResponse, error)
	// GenerateImage sends an image generation prompt and returns image bytes.
	// [REQ:BM-REQ-AI-IMAGE]
	GenerateImage(ctx context.Context, req ImageRequest) (*ImageResponse, error)
}

// TextRequest describes a text generation request.
type TextRequest struct {
	Prompt      string `json:"prompt"`
	Model       string `json:"model,omitempty"`
	Temperature float64 `json:"temperature,omitempty"`
	MaxTokens   int    `json:"max_tokens,omitempty"`
}

// TextResponse holds the result of a text generation call.
type TextResponse struct {
	Text     string `json:"text"`
	Provider string `json:"provider"`
	Model    string `json:"model"`
}

// ImageRequest describes an image generation request.
type ImageRequest struct {
	Prompt string `json:"prompt"`
	Model  string `json:"model,omitempty"`
	Width  int    `json:"width,omitempty"`
	Height int    `json:"height,omitempty"`
}

// ImageResponse holds the result of an image generation call.
type ImageResponse struct {
	Data     []byte `json:"-"`
	MimeType string `json:"mime_type"`
	Provider string `json:"provider"`
	Model    string `json:"model"`
}

// Chain tries providers in order until one succeeds. [REQ:BM-REQ-AI-CHAIN]
type Chain struct {
	providers []Provider
}

// NewChain creates a provider chain with the given providers (tried in order).
func NewChain(providers ...Provider) *Chain {
	return &Chain{providers: providers}
}

// GenerateText tries each provider in order for text generation.
// [REQ:BM-REQ-AI-TEXT]
func (c *Chain) GenerateText(ctx context.Context, req TextRequest) (*TextResponse, error) {
	var errs []string
	for _, p := range c.providers {
		if !p.Available(ctx) {
			errs = append(errs, fmt.Sprintf("%s: unavailable", p.Name()))
			continue
		}
		resp, err := p.GenerateText(ctx, req)
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", p.Name(), err))
			continue
		}
		return resp, nil
	}
	return nil, fmt.Errorf("all providers failed: %s", strings.Join(errs, "; "))
}

// GenerateImage tries each provider in order for image generation.
// [REQ:BM-REQ-AI-IMAGE]
func (c *Chain) GenerateImage(ctx context.Context, req ImageRequest) (*ImageResponse, error) {
	var errs []string
	for _, p := range c.providers {
		if !p.Available(ctx) {
			errs = append(errs, fmt.Sprintf("%s: unavailable", p.Name()))
			continue
		}
		resp, err := p.GenerateImage(ctx, req)
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", p.Name(), err))
			continue
		}
		return resp, nil
	}
	return nil, fmt.Errorf("all providers failed: %s", strings.Join(errs, "; "))
}

// Available returns true if at least one provider is available.
func (c *Chain) Available(ctx context.Context) bool {
	for _, p := range c.providers {
		if p.Available(ctx) {
			return true
		}
	}
	return false
}

// Providers returns the chain's providers (for introspection/testing).
func (c *Chain) Providers() []Provider {
	return c.providers
}
