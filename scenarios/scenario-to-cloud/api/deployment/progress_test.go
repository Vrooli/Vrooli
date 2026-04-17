package deployment

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func TestChannelEmitter_Creation(t *testing.T) {
	t.Parallel()

	emitter := NewChannelEmitter(10)
	if emitter == nil {
		t.Fatal("NewChannelEmitter() returned nil")
	}
	if emitter.closed {
		t.Error("NewChannelEmitter() should not be closed")
	}
	if emitter.ch == nil {
		t.Error("NewChannelEmitter() channel should not be nil")
	}
}

func TestChannelEmitter_Emit(t *testing.T) {
	t.Parallel()

	emitter := NewChannelEmitter(10)

	event := Event{
		Type:      "step_started",
		Step:      "upload",
		StepTitle: "Uploading bundle",
		Progress:  25.0,
		Message:   "Test message",
	}

	emitter.Emit(event)

	select {
	case received := <-emitter.Channel():
		if received.Type != "step_started" {
			t.Errorf("Type = %q, want %q", received.Type, "step_started")
		}
		if received.Step != "upload" {
			t.Errorf("Step = %q, want %q", received.Step, "upload")
		}
		if received.StepTitle != "Uploading bundle" {
			t.Errorf("StepTitle = %q, want %q", received.StepTitle, "Uploading bundle")
		}
		if received.Progress != 25.0 {
			t.Errorf("Progress = %f, want 25.0", received.Progress)
		}
		if received.Timestamp == "" {
			t.Error("Timestamp should be auto-filled")
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Emit() did not send event to channel")
	}
}

func TestChannelEmitter_PreservesExistingTimestamp(t *testing.T) {
	t.Parallel()

	emitter := NewChannelEmitter(10)
	customTimestamp := "2025-01-15T12:00:00Z"

	event := Event{
		Type:      "step_completed",
		Step:      "upload",
		Timestamp: customTimestamp,
	}

	emitter.Emit(event)

	select {
	case received := <-emitter.Channel():
		if received.Timestamp != customTimestamp {
			t.Errorf("Timestamp = %q, want %q (should preserve existing)", received.Timestamp, customTimestamp)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Emit() did not send event to channel")
	}
}

func TestChannelEmitter_Close(t *testing.T) {
	t.Parallel()

	emitter := NewChannelEmitter(10)

	emitter.Close()

	if !emitter.closed {
		t.Error("Close() should set closed to true")
	}

	// Verify channel is closed
	select {
	case _, ok := <-emitter.Channel():
		if ok {
			t.Error("Channel should be closed after Close()")
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Channel should return immediately when closed")
	}
}

func TestChannelEmitter_CloseIdempotent(t *testing.T) {
	t.Parallel()

	emitter := NewChannelEmitter(10)

	// Multiple close calls should not panic
	emitter.Close()
	emitter.Close()
	emitter.Close()
}

func TestChannelEmitter_EmitAfterClose(t *testing.T) {
	t.Parallel()

	emitter := NewChannelEmitter(10)
	emitter.Close()

	// Should not panic when emitting after close
	event := Event{
		Type: "step_started",
		Step: "upload",
	}
	emitter.Emit(event) // Should silently ignore
}

func TestNoOpEmitter(t *testing.T) {
	t.Parallel()

	var emitter NoOpEmitter

	// These should not panic
	emitter.Emit(Event{Type: "test"})
	emitter.Close()
	emitter.Emit(Event{Type: "test"})
}

func TestAllSteps(t *testing.T) {
	t.Parallel()

	steps := AllSteps()

	if len(steps) == 0 {
		t.Fatal("AllSteps() returned empty slice")
	}

	// Check first two steps are bundle_build and preflight
	if steps[0].ID != "bundle_build" {
		t.Errorf("First step ID = %q, want %q", steps[0].ID, "bundle_build")
	}
	if steps[1].ID != "preflight" {
		t.Errorf("Second step ID = %q, want %q", steps[1].ID, "preflight")
	}

	// Verify each step has non-empty ID and Title
	for i, step := range steps {
		if step.ID == "" {
			t.Errorf("Step %d has empty ID", i)
		}
		if step.Title == "" {
			t.Errorf("Step %d (%s) has empty Title", i, step.ID)
		}
	}
}

func TestSetupSteps(t *testing.T) {
	t.Parallel()

	if len(SetupSteps) == 0 {
		t.Fatal("SetupSteps is empty")
	}

	// Check expected setup step IDs
	expectedIDs := []string{"mkdir", "bootstrap", "upload", "extract", "setup", "autoheal", "verify_setup"}
	for i, expected := range expectedIDs {
		if i >= len(SetupSteps) {
			t.Errorf("Missing expected step: %s", expected)
			continue
		}
		if SetupSteps[i].ID != expected {
			t.Errorf("SetupSteps[%d].ID = %q, want %q", i, SetupSteps[i].ID, expected)
		}
	}
}

func TestDeploySteps(t *testing.T) {
	t.Parallel()

	if len(DeploySteps) == 0 {
		t.Fatal("DeploySteps is empty")
	}

	// Verify all steps have positive weights
	for _, step := range DeploySteps {
		if step.Weight <= 0 {
			t.Errorf("DeployStep %s has non-positive weight: %f", step.ID, step.Weight)
		}
	}
}

func TestDefaultPortsFetcherUsesContractResolvedServicePath(t *testing.T) {
	root := t.TempDir()
	writeRepoContractFixture(t, root)
	t.Setenv("SCENARIO_TO_CLOUD_REPO_ROOT", root)

	writeJSONFile(t, filepath.Join(root, "scenarios", "demo", ".vrooli", "service.json"), map[string]interface{}{
		"service": map[string]interface{}{"name": "demo"},
		"ports": map[string]interface{}{
			"api": map[string]interface{}{"port": 8080},
			"ui":  map[string]interface{}{"port": 3000},
		},
	})

	got, err := (&DefaultPortsFetcher{}).FetchPorts(context.Background(), "demo")
	if err != nil {
		t.Fatalf("FetchPorts: %v", err)
	}
	want := map[string]int{"api": 8080, "ui": 3000}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("FetchPorts = %#v, want %#v", got, want)
	}
}

func TestServiceJSONDependenciesFetcherUsesContractResolvedServicePath(t *testing.T) {
	root := t.TempDir()
	writeRepoContractFixture(t, root)
	t.Setenv("SCENARIO_TO_CLOUD_REPO_ROOT", root)

	writeJSONFile(t, filepath.Join(root, "scenarios", "demo", ".vrooli", "service.json"), map[string]interface{}{
		"service": map[string]interface{}{"name": "demo"},
		"dependencies": map[string]interface{}{
			"resources": map[string]interface{}{
				"postgres": map[string]interface{}{"enabled": true},
				"redis":    map[string]interface{}{"enabled": true},
				"vault":    map[string]interface{}{"enabled": false},
			},
			"scenarios": map[string]interface{}{
				"dep-a": map[string]interface{}{"enabled": true},
				"dep-b": map[string]interface{}{"enabled": false},
			},
		},
	})

	resources, scenarios, err := ServiceJSONDependenciesFetcher("demo")
	if err != nil {
		t.Fatalf("ServiceJSONDependenciesFetcher: %v", err)
	}
	if !reflect.DeepEqual(resources, []string{"postgres", "redis"}) {
		t.Fatalf("resources = %#v, want %#v", resources, []string{"postgres", "redis"})
	}
	if !reflect.DeepEqual(scenarios, []string{"dep-a"}) {
		t.Fatalf("scenarios = %#v, want %#v", scenarios, []string{"dep-a"})
	}
}

func writeRepoContractFixture(t *testing.T, root string) {
	t.Helper()

	for _, dir := range []string{".vrooli", "scenarios", "resources", "packages", "cmd", "internal"} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module fixture\n\ngo 1.24.0\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	contract := `{
  "$schema": "schemas/repo-contract.schema.json",
  "version": "1.0.0",
  "platform": {"mode": "cross_platform_go_native", "legacy_project_bash_supported": false},
  "root": {"markers": {"required_dirs": [".vrooli", "scenarios", "resources", "packages", "cmd", "internal"], "required_files": ["go.mod"]}},
  "layout": {"project_config_dir": ".vrooli", "scenario_dir": "scenarios", "resource_dir": "resources", "template_dir": "templates", "package_dir": "packages", "command_dir": "cmd", "internal_dir": "internal", "docs_dir": "docs"},
  "scenario": {"required_files": [".vrooli/service.json"], "well_known_paths": {"service": ".vrooli/service.json", "docs": "docs", "requirements": "requirements", "api": "api", "ui": "ui", "cli": "cli", "initialization": "initialization"}},
  "resource": {"manifest": "resource.json", "well_known_paths": {"docs": "docs", "initialization": "initialization"}},
  "globs": {"syntax": "doublestar", "root_relative": true, "case_sensitive": true, "allow_absolute": false, "path_format": "slash_normalized"},
  "environment": {"variables": {"repo_root": "VROOLI_ROOT", "source_root": "VROOLI_SOURCE_ROOT", "sandbox_id": "VROOLI_SANDBOX_ID", "sandbox_merged": "VROOLI_SANDBOX_MERGED", "sandbox_scope": "VROOLI_SANDBOX_SCOPE"}},
  "sandbox": {"full_repo_scopes": ["", ".", "/"], "scenario_scope_prefix": "scenarios/"},
  "profiles": {
    "mini_vrooli_bundle": {
      "description": "fixture profile",
      "parameters": ["scenario", "resources[*]"],
      "include": [".vrooli", "cmd", "internal", "packages", "scenarios/{scenario}", "resources/{resources[*]}"],
      "optional_include": ["docs", "go.mod"],
      "exclude": [".git/**"]
    }
  }
}`
	if err := os.WriteFile(filepath.Join(root, ".vrooli", "repo-contract.json"), []byte(contract), 0o644); err != nil {
		t.Fatalf("write repo-contract.json: %v", err)
	}
}

func writeJSONFile(t *testing.T, path string, payload map[string]interface{}) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir for %s: %v", path, err)
	}
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal %s: %v", path, err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
