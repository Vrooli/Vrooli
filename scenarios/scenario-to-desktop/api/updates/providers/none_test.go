package providers

import (
	"context"
	"testing"

	"scenario-to-desktop-api/generation"
	"scenario-to-desktop-api/updates"
)

func TestNoneProvider_Name(t *testing.T) {
	p := NewNoneProvider(&generation.UpdateConfig{})
	if p.Name() != "none" {
		t.Errorf("expected name 'none', got '%s'", p.Name())
	}
}

func TestNoneProvider_Validate(t *testing.T) {
	p := NewNoneProvider(&generation.UpdateConfig{})
	if err := p.Validate(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestNoneProvider_GetPublishConfig(t *testing.T) {
	p := NewNoneProvider(&generation.UpdateConfig{})
	cfg, err := p.GetPublishConfig("stable")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg != nil {
		t.Error("expected nil config from none provider")
	}
}

func TestNoneProvider_GenerateManifest(t *testing.T) {
	p := NewNoneProvider(&generation.UpdateConfig{})
	result, err := p.GenerateManifest(context.Background(), &updates.ManifestRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil {
		t.Error("expected nil result from none provider")
	}
}

func TestNoneProvider_RequiresManifestUpload(t *testing.T) {
	p := NewNoneProvider(&generation.UpdateConfig{})
	if p.RequiresManifestUpload() {
		t.Error("none provider should not require manifest upload")
	}
}
