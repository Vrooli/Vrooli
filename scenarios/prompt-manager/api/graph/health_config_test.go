package graph

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestValidateHealthConfig_DefaultValid(t *testing.T) {
	if err := ValidateHealthConfig(DefaultHealthConfig()); err != nil {
		t.Fatalf("expected default config to validate, got %v", err)
	}
}

func TestValidateHealthConfig_RejectsZeroWeightSum(t *testing.T) {
	cfg := DefaultHealthConfig()
	cfg.Team = HealthWeights{}
	if err := ValidateHealthConfig(cfg); err == nil {
		t.Fatalf("expected validation error for zero team weights")
	}
}

func TestHealthConfigStore_GetDefaultsWhenMissing(t *testing.T) {
	dir := t.TempDir()
	s := NewHealthConfigStore(dir)

	cfg, err := s.Get(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Team.OutgoingEdges != 1.0 {
		t.Fatalf("expected default outgoing weight 1.0, got %f", cfg.Team.OutgoingEdges)
	}
}

func TestHealthConfigStore_PutAndGet(t *testing.T) {
	dir := t.TempDir()
	s := NewHealthConfigStore(dir)

	cfg := DefaultHealthConfig()
	cfg.Agent.CodeUsage = 0.9
	if err := s.Put(context.Background(), cfg); err != nil {
		t.Fatalf("put error: %v", err)
	}

	loaded, err := s.Get(context.Background())
	if err != nil {
		t.Fatalf("get error: %v", err)
	}
	if loaded.Agent.CodeUsage != 0.9 {
		t.Fatalf("expected codeUsage 0.9, got %f", loaded.Agent.CodeUsage)
	}

	expectedPath := filepath.Join(dir, "config", "graph-health.json")
	if !storeFileExists(expectedPath) {
		t.Fatalf("expected persisted config at %s", expectedPath)
	}
}

func storeFileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
