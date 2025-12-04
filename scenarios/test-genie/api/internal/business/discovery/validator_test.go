package discovery

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	reqdiscovery "test-genie/internal/requirements/discovery"
)

// mockDiscoverer implements reqdiscovery.Discoverer for testing.
type mockDiscoverer struct {
	files []reqdiscovery.DiscoveredFile
	err   error
}

func (m *mockDiscoverer) Discover(ctx context.Context, scenarioRoot string) ([]reqdiscovery.DiscoveredFile, error) {
	return m.files, m.err
}

func TestDiscover_Success(t *testing.T) {
	mock := &mockDiscoverer{
		files: []reqdiscovery.DiscoveredFile{
			{AbsolutePath: "/test/requirements/index.json", RelativePath: "index.json", IsIndex: true},
			{AbsolutePath: "/test/requirements/01-core/module.json", RelativePath: "01-core/module.json", IsIndex: false, ModuleDir: "01-core"},
			{AbsolutePath: "/test/requirements/02-extra/module.json", RelativePath: "02-extra/module.json", IsIndex: false, ModuleDir: "02-extra"},
		},
	}

	v := NewWithDiscoverer("/test", mock, io.Discard)
	result := v.Discover(context.Background(), true)

	if !result.Success {
		t.Fatalf("expected success, got error: %v", result.Error)
	}
	if result.ModuleCount != 2 {
		t.Errorf("expected 2 modules, got %d", result.ModuleCount)
	}
	if len(result.Files) != 3 {
		t.Errorf("expected 3 files, got %d", len(result.Files))
	}
	if len(result.Observations) == 0 {
		t.Error("expected observations to be recorded")
	}
}

func TestDiscover_NoRequirementsDir(t *testing.T) {
	mock := &mockDiscoverer{
		err: reqdiscovery.ErrNoRequirementsDir,
	}

	v := NewWithDiscoverer("/test", mock, io.Discard)
	result := v.Discover(context.Background(), true)

	if result.Success {
		t.Fatal("expected failure when requirements dir missing")
	}
	if result.FailureClass != FailureClassMisconfiguration {
		t.Errorf("expected misconfiguration failure class, got %s", result.FailureClass)
	}
	if result.Remediation == "" {
		t.Error("expected remediation guidance")
	}
}

func TestDiscover_SystemError(t *testing.T) {
	mock := &mockDiscoverer{
		err: errors.New("permission denied"),
	}

	v := NewWithDiscoverer("/test", mock, io.Discard)
	result := v.Discover(context.Background(), true)

	if result.Success {
		t.Fatal("expected failure for system error")
	}
	if result.FailureClass != FailureClassSystem {
		t.Errorf("expected system failure class, got %s", result.FailureClass)
	}
}

func TestDiscover_NoModulesWithRequireModules(t *testing.T) {
	mock := &mockDiscoverer{
		files: []reqdiscovery.DiscoveredFile{
			{AbsolutePath: "/test/requirements/index.json", RelativePath: "index.json", IsIndex: true},
		},
	}

	v := NewWithDiscoverer("/test", mock, io.Discard)
	result := v.Discover(context.Background(), true)

	if result.Success {
		t.Fatal("expected failure when no modules found and requireModules=true")
	}
	if result.FailureClass != FailureClassMisconfiguration {
		t.Errorf("expected misconfiguration failure class, got %s", result.FailureClass)
	}
}

func TestDiscover_NoModulesWithoutRequireModules(t *testing.T) {
	mock := &mockDiscoverer{
		files: []reqdiscovery.DiscoveredFile{
			{AbsolutePath: "/test/requirements/index.json", RelativePath: "index.json", IsIndex: true},
		},
	}

	v := NewWithDiscoverer("/test", mock, io.Discard)
	result := v.Discover(context.Background(), false)

	if !result.Success {
		t.Fatalf("expected success when requireModules=false, got error: %v", result.Error)
	}
	if result.ModuleCount != 0 {
		t.Errorf("expected 0 modules, got %d", result.ModuleCount)
	}
}

func TestDiscover_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	mock := &mockDiscoverer{
		err: ctx.Err(),
	}

	v := NewWithDiscoverer("/test", mock, io.Discard)
	result := v.Discover(ctx, true)

	if result.Success {
		t.Fatal("expected failure for cancelled context")
	}
}

// Integration test using real filesystem
func TestDiscover_Integration(t *testing.T) {
	tmpDir := t.TempDir()
	reqDir := filepath.Join(tmpDir, "requirements")
	if err := os.MkdirAll(reqDir, 0o755); err != nil {
		t.Fatalf("failed to create requirements dir: %v", err)
	}

	// Create index.json
	indexContent := `{"imports":["01-core/module.json"]}`
	if err := os.WriteFile(filepath.Join(reqDir, "index.json"), []byte(indexContent), 0o644); err != nil {
		t.Fatalf("failed to write index: %v", err)
	}

	// Create module
	moduleDir := filepath.Join(reqDir, "01-core")
	if err := os.MkdirAll(moduleDir, 0o755); err != nil {
		t.Fatalf("failed to create module dir: %v", err)
	}
	moduleContent := `{"requirements":[{"id":"REQ-001","title":"Test"}]}`
	if err := os.WriteFile(filepath.Join(moduleDir, "module.json"), []byte(moduleContent), 0o644); err != nil {
		t.Fatalf("failed to write module: %v", err)
	}

	v := New(tmpDir, io.Discard)
	result := v.Discover(context.Background(), true)

	if !result.Success {
		t.Fatalf("expected success, got error: %v", result.Error)
	}
	if result.ModuleCount != 1 {
		t.Errorf("expected 1 module, got %d", result.ModuleCount)
	}
}

// Ensure validator satisfies the interface at compile time
var _ Validator = (*validator)(nil)

// Ensure mock satisfies the requirements/discovery interface
var _ reqdiscovery.Discoverer = (*mockDiscoverer)(nil)

// Benchmarks

func BenchmarkDiscover(b *testing.B) {
	mock := &mockDiscoverer{
		files: []reqdiscovery.DiscoveredFile{
			{AbsolutePath: "/test/requirements/index.json", IsIndex: true},
			{AbsolutePath: "/test/requirements/01-core/module.json", IsIndex: false, ModuleDir: "01-core"},
			{AbsolutePath: "/test/requirements/02-extra/module.json", IsIndex: false, ModuleDir: "02-extra"},
		},
	}

	v := NewWithDiscoverer("/test", mock, io.Discard)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		v.Discover(ctx, true)
	}
}
