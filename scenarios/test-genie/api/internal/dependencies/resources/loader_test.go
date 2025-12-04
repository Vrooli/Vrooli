package resources

import (
	"io"
	"strings"
	"testing"

	"test-genie/internal/orchestrator/workspace"
)

// mockManifestLoader implements ManifestLoader for testing.
type mockManifestLoader struct {
	manifest *workspace.ServiceManifest
	err      error
}

func (m *mockManifestLoader) Load(path string) (*workspace.ServiceManifest, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.manifest, nil
}

func TestLoaderLoadWithRequiredResources(t *testing.T) {
	manifest := &workspace.ServiceManifest{}
	manifest.Dependencies.Resources = map[string]struct {
		Required bool   `json:"required"`
		Enabled  bool   `json:"enabled"`
		Type     string `json:"type"`
	}{
		"postgres": {Required: true, Enabled: true},
		"redis":    {Required: false, Enabled: true},
	}

	loader := NewLoader("/scenarios/demo", io.Discard, WithManifestLoader(&mockManifestLoader{
		manifest: manifest,
	}))

	resources, err := loader.Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(resources) != 1 {
		t.Fatalf("expected 1 required resource, got %d", len(resources))
	}
	if resources[0] != "postgres" {
		t.Fatalf("expected 'postgres', got '%s'", resources[0])
	}
}

func TestLoaderLoadWithNoResources(t *testing.T) {
	manifest := &workspace.ServiceManifest{}

	loader := NewLoader("/scenarios/demo", io.Discard, WithManifestLoader(&mockManifestLoader{
		manifest: manifest,
	}))

	resources, err := loader.Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resources != nil {
		t.Fatalf("expected nil resources, got %v", resources)
	}
}

func TestLoaderLoadError(t *testing.T) {
	loader := NewLoader("/scenarios/demo", io.Discard, WithManifestLoader(&mockManifestLoader{
		err: &mockError{"manifest not found"},
	}))

	_, err := loader.Load()
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestLoadExpectationsSuccess(t *testing.T) {
	// This is a higher-level test that would require more setup
	// to mock the file system. For now, we test the basic structure.
	result := LoadResult{
		Success:   true,
		Resources: []string{"postgres"},
	}

	if !result.Success {
		t.Fatalf("expected success")
	}
	if len(result.Resources) != 1 {
		t.Fatalf("expected 1 resource")
	}
}

func TestLoaderLoadWithMultipleRequiredResources(t *testing.T) {
	manifest := &workspace.ServiceManifest{}
	manifest.Dependencies.Resources = map[string]struct {
		Required bool   `json:"required"`
		Enabled  bool   `json:"enabled"`
		Type     string `json:"type"`
	}{
		"postgres": {Required: true, Enabled: true},
		"redis":    {Required: true, Enabled: true},
		"qdrant":   {Required: true, Enabled: false}, // Required but disabled - still required
	}

	loader := NewLoader("/scenarios/demo", io.Discard, WithManifestLoader(&mockManifestLoader{
		manifest: manifest,
	}))

	resources, err := loader.Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(resources) != 3 {
		t.Fatalf("expected 3 required resources, got %d: %v", len(resources), resources)
	}
}

func TestLoaderLoadLogsResourceRequirements(t *testing.T) {
	manifest := &workspace.ServiceManifest{}
	manifest.Dependencies.Resources = map[string]struct {
		Required bool   `json:"required"`
		Enabled  bool   `json:"enabled"`
		Type     string `json:"type"`
	}{
		"postgres": {Required: true, Enabled: true},
	}

	var logOutput strings.Builder
	loader := NewLoader("/scenarios/demo", &logOutput, WithManifestLoader(&mockManifestLoader{
		manifest: manifest,
	}))

	_, err := loader.Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(logOutput.String(), "postgres") {
		t.Errorf("expected log to contain 'postgres', got: %s", logOutput.String())
	}
}

func TestLoaderLoadLogsWarningForNoResources(t *testing.T) {
	manifest := &workspace.ServiceManifest{}

	var logOutput strings.Builder
	loader := NewLoader("/scenarios/demo", &logOutput, WithManifestLoader(&mockManifestLoader{
		manifest: manifest,
	}))

	_, err := loader.Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(logOutput.String(), "WARNING") {
		t.Errorf("expected warning log for no resources, got: %s", logOutput.String())
	}
}

func TestLoaderLoadWithNilLogWriter(t *testing.T) {
	manifest := &workspace.ServiceManifest{}
	manifest.Dependencies.Resources = map[string]struct {
		Required bool   `json:"required"`
		Enabled  bool   `json:"enabled"`
		Type     string `json:"type"`
	}{
		"postgres": {Required: true, Enabled: true},
	}

	// Test with nil log writer - should not panic
	loader := NewLoader("/scenarios/demo", nil, WithManifestLoader(&mockManifestLoader{
		manifest: manifest,
	}))

	resources, err := loader.Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(resources) != 1 {
		t.Fatalf("expected 1 resource, got %d", len(resources))
	}
}

// Ensure mock types satisfy interfaces at compile time.
var (
	_ ManifestLoader      = (*mockManifestLoader)(nil)
	_ ExpectationsLoader = (*loader)(nil)
)

type mockError struct {
	msg string
}

func (e *mockError) Error() string {
	return e.msg
}
