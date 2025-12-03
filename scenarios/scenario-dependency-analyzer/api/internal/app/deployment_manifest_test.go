package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"scenario-dependency-analyzer/internal/deployment"
	types "scenario-dependency-analyzer/internal/types"
)

func TestBuildBundleManifestSkeletonValidatesAgainstSchema(t *testing.T) {
	scenarioDir := t.TempDir()

	if err := os.MkdirAll(filepath.Join(scenarioDir, "ui", "dist"), 0o755); err != nil {
		t.Fatalf("failed to create ui dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(scenarioDir, "ui", "dist", "index.html"), []byte("<html></html>"), 0o644); err != nil {
		t.Fatalf("failed to write ui entry: %v", err)
	}

	if err := os.MkdirAll(filepath.Join(scenarioDir, "api"), 0o755); err != nil {
		t.Fatalf("failed to create api dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(scenarioDir, "api", "test-scenario-api"), []byte("#!/bin/bash\necho ok\n"), 0o755); err != nil {
		t.Fatalf("failed to write api binary placeholder: %v", err)
	}

	cfg := &types.ServiceConfig{}
	cfg.Service.Name = "test-scenario"
	cfg.Service.DisplayName = "Test Scenario"
	cfg.Service.Description = "Example scenario for bundle skeleton"
	cfg.Service.Version = "1.2.3"

	nodes := []types.DeploymentDependencyNode{
		{
			Name:         "postgres",
			Type:         "resource",
			Alternatives: []string{"sqlite"},
			Children:     nil,
		},
	}

	manifest := deployment.BuildBundleManifest("test-scenario", scenarioDir, time.Now(), nodes, cfg)
	if manifest.Skeleton == nil {
		t.Fatalf("expected skeleton to be populated")
	}

	payload, err := json.Marshal(manifest.Skeleton)
	if err != nil {
		t.Fatalf("failed to marshal skeleton: %v", err)
	}
	if err := validateDesktopBundleManifestBytes(payload); err != nil {
		t.Fatalf("skeleton failed schema validation: %v", err)
	}

	if len(manifest.Skeleton.Swaps) == 0 {
		t.Fatalf("expected swaps to include at least one alternative")
	}

	var foundUI bool
	for _, svc := range manifest.Skeleton.Services {
		if svc.Type == "ui-bundle" {
			foundUI = true
		}
		if svc.ID == "test-scenario-api" && svc.Health.PortName != "api" {
			t.Fatalf("api skeleton should expose api port health check")
		}
	}
	if !foundUI {
		t.Fatalf("expected ui-bundle service to be included when ui/dist exists")
	}
}
