package preflight

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestChecker_CheckBrowserless_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/pressure" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"pressure":{"running":0}}`))
	}))
	defer server.Close()

	c := NewChecker(server.URL)
	err := c.CheckBrowserless(context.Background())
	if err != nil {
		t.Errorf("CheckBrowserless() error = %v", err)
	}
}

func TestChecker_CheckBrowserless_Failure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	c := NewChecker(server.URL)
	err := c.CheckBrowserless(context.Background())
	if err == nil {
		t.Error("CheckBrowserless() expected error for unavailable service")
	}
}

func TestChecker_CheckUIDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	// No UI directory
	c := NewChecker("http://localhost:4110")
	if c.CheckUIDirectory(tmpDir) {
		t.Error("CheckUIDirectory() should return false when ui dir doesn't exist")
	}

	// Create UI directory
	uiDir := filepath.Join(tmpDir, "ui")
	if err := os.Mkdir(uiDir, 0o755); err != nil {
		t.Fatal(err)
	}

	if !c.CheckUIDirectory(tmpDir) {
		t.Error("CheckUIDirectory() should return true when ui dir exists")
	}
}

func TestChecker_CheckIframeBridge_Present(t *testing.T) {
	tmpDir := t.TempDir()
	uiDir := filepath.Join(tmpDir, "ui")
	if err := os.Mkdir(uiDir, 0o755); err != nil {
		t.Fatal(err)
	}

	packageJSON := `{
		"name": "test-ui",
		"dependencies": {
			"@vrooli/iframe-bridge": "^1.0.0"
		}
	}`

	if err := os.WriteFile(filepath.Join(uiDir, "package.json"), []byte(packageJSON), 0o644); err != nil {
		t.Fatal(err)
	}

	c := NewChecker("http://localhost:4110")
	status, err := c.CheckIframeBridge(context.Background(), tmpDir)
	if err != nil {
		t.Fatalf("CheckIframeBridge() error = %v", err)
	}

	if !status.DependencyPresent {
		t.Error("DependencyPresent should be true")
	}
	if status.Version != "^1.0.0" {
		t.Errorf("Version = %q, want %q", status.Version, "^1.0.0")
	}
}

func TestChecker_CheckIframeBridge_DevDependency(t *testing.T) {
	tmpDir := t.TempDir()
	uiDir := filepath.Join(tmpDir, "ui")
	if err := os.Mkdir(uiDir, 0o755); err != nil {
		t.Fatal(err)
	}

	packageJSON := `{
		"name": "test-ui",
		"devDependencies": {
			"@vrooli/iframe-bridge": "workspace:*"
		}
	}`

	if err := os.WriteFile(filepath.Join(uiDir, "package.json"), []byte(packageJSON), 0o644); err != nil {
		t.Fatal(err)
	}

	c := NewChecker("http://localhost:4110")
	status, err := c.CheckIframeBridge(context.Background(), tmpDir)
	if err != nil {
		t.Fatalf("CheckIframeBridge() error = %v", err)
	}

	if !status.DependencyPresent {
		t.Error("DependencyPresent should be true for devDependencies")
	}
	if status.Version != "workspace:*" {
		t.Errorf("Version = %q, want %q", status.Version, "workspace:*")
	}
}

func TestChecker_CheckIframeBridge_Missing(t *testing.T) {
	tmpDir := t.TempDir()
	uiDir := filepath.Join(tmpDir, "ui")
	if err := os.Mkdir(uiDir, 0o755); err != nil {
		t.Fatal(err)
	}

	packageJSON := `{
		"name": "test-ui",
		"dependencies": {
			"react": "^18.0.0"
		}
	}`

	if err := os.WriteFile(filepath.Join(uiDir, "package.json"), []byte(packageJSON), 0o644); err != nil {
		t.Fatal(err)
	}

	c := NewChecker("http://localhost:4110")
	status, err := c.CheckIframeBridge(context.Background(), tmpDir)
	if err != nil {
		t.Fatalf("CheckIframeBridge() error = %v", err)
	}

	if status.DependencyPresent {
		t.Error("DependencyPresent should be false when iframe-bridge is not listed")
	}
}

func TestChecker_CheckIframeBridge_NoPackageJSON(t *testing.T) {
	tmpDir := t.TempDir()

	c := NewChecker("http://localhost:4110")
	status, err := c.CheckIframeBridge(context.Background(), tmpDir)
	if err != nil {
		t.Fatalf("CheckIframeBridge() error = %v", err)
	}

	if status.DependencyPresent {
		t.Error("DependencyPresent should be false when package.json doesn't exist")
	}
}

func TestChecker_CheckBundleFreshness_NoDist(t *testing.T) {
	tmpDir := t.TempDir()
	uiDir := filepath.Join(tmpDir, "ui")
	if err := os.Mkdir(uiDir, 0o755); err != nil {
		t.Fatal(err)
	}

	c := NewChecker("http://localhost:4110")
	status, err := c.CheckBundleFreshness(context.Background(), tmpDir)
	if err != nil {
		t.Fatalf("CheckBundleFreshness() error = %v", err)
	}

	if status.Fresh {
		t.Error("Fresh should be false when dist doesn't exist")
	}
}

func TestChecker_CheckBundleFreshness_HasDist(t *testing.T) {
	tmpDir := t.TempDir()
	distDir := filepath.Join(tmpDir, "ui", "dist")
	if err := os.MkdirAll(distDir, 0o755); err != nil {
		t.Fatal(err)
	}

	indexHTML := filepath.Join(distDir, "index.html")
	if err := os.WriteFile(indexHTML, []byte("<html></html>"), 0o644); err != nil {
		t.Fatal(err)
	}

	c := NewChecker("http://localhost:4110")
	status, err := c.CheckBundleFreshness(context.Background(), tmpDir)
	if err != nil {
		t.Fatalf("CheckBundleFreshness() error = %v", err)
	}

	if !status.Fresh {
		t.Errorf("Fresh should be true when dist exists and no service.json, got reason: %s", status.Reason)
	}
}

func TestChecker_CheckBundleFreshness_StaleBundle(t *testing.T) {
	tmpDir := t.TempDir()

	// Create dist directory with old timestamp
	distDir := filepath.Join(tmpDir, "ui", "dist")
	if err := os.MkdirAll(distDir, 0o755); err != nil {
		t.Fatal(err)
	}
	indexHTML := filepath.Join(distDir, "index.html")
	if err := os.WriteFile(indexHTML, []byte("<html></html>"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create service.json with ui-bundle check config
	// Note: filepath.Glob doesn't support ** - use single level globs
	vrooliDir := filepath.Join(tmpDir, ".vrooli")
	if err := os.MkdirAll(vrooliDir, 0o755); err != nil {
		t.Fatal(err)
	}
	serviceJSON := `{
		"lifecycle": {
			"setup": {
				"condition": {
					"checks": [
						{
							"type": "ui-bundle",
							"source_globs": ["ui/src/*", "ui/package.json"],
							"dist_path": "ui/dist"
						}
					]
				}
			}
		}
	}`
	if err := os.WriteFile(filepath.Join(vrooliDir, "service.json"), []byte(serviceJSON), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create src directory with newer file
	srcDir := filepath.Join(tmpDir, "ui", "src")
	if err := os.MkdirAll(srcDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Set dist directory to be older than src file
	oldTime := time.Now().Add(-1 * time.Hour)
	if err := os.Chtimes(distDir, oldTime, oldTime); err != nil {
		t.Fatal(err)
	}

	// Create a newer source file
	srcFile := filepath.Join(srcDir, "App.tsx")
	if err := os.WriteFile(srcFile, []byte("export default function App() {}"), 0o644); err != nil {
		t.Fatal(err)
	}

	c := NewChecker("http://localhost:4110")
	status, err := c.CheckBundleFreshness(context.Background(), tmpDir)
	if err != nil {
		t.Fatalf("CheckBundleFreshness() error = %v", err)
	}

	if status.Fresh {
		t.Error("Fresh should be false when source files are newer than dist")
	}
	if status.Reason == "" {
		t.Error("Reason should be set when bundle is stale")
	}
}

func TestChecker_CheckBundleFreshness_WithServiceJSONFreshBundle(t *testing.T) {
	tmpDir := t.TempDir()

	// Create service.json with ui-bundle check config first
	// Note: filepath.Glob doesn't support ** - use single level globs
	vrooliDir := filepath.Join(tmpDir, ".vrooli")
	if err := os.MkdirAll(vrooliDir, 0o755); err != nil {
		t.Fatal(err)
	}
	serviceJSON := `{
		"lifecycle": {
			"setup": {
				"condition": {
					"checks": [
						{
							"type": "ui-bundle",
							"source_globs": ["ui/src/*", "ui/package.json"],
							"dist_path": "ui/dist"
						}
					]
				}
			}
		}
	}`
	if err := os.WriteFile(filepath.Join(vrooliDir, "service.json"), []byte(serviceJSON), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create src directory with older file
	srcDir := filepath.Join(tmpDir, "ui", "src")
	if err := os.MkdirAll(srcDir, 0o755); err != nil {
		t.Fatal(err)
	}
	srcFile := filepath.Join(srcDir, "App.tsx")
	if err := os.WriteFile(srcFile, []byte("export default function App() {}"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Set src file to be older
	oldTime := time.Now().Add(-1 * time.Hour)
	if err := os.Chtimes(srcFile, oldTime, oldTime); err != nil {
		t.Fatal(err)
	}

	// Create dist directory with newer timestamp
	distDir := filepath.Join(tmpDir, "ui", "dist")
	if err := os.MkdirAll(distDir, 0o755); err != nil {
		t.Fatal(err)
	}
	indexHTML := filepath.Join(distDir, "index.html")
	if err := os.WriteFile(indexHTML, []byte("<html></html>"), 0o644); err != nil {
		t.Fatal(err)
	}

	c := NewChecker("http://localhost:4110")
	status, err := c.CheckBundleFreshness(context.Background(), tmpDir)
	if err != nil {
		t.Fatalf("CheckBundleFreshness() error = %v", err)
	}

	if !status.Fresh {
		t.Errorf("Fresh should be true when dist is newer than source, got reason: %s", status.Reason)
	}
}

func TestChecker_CheckBundleFreshness_InvalidServiceJSON(t *testing.T) {
	tmpDir := t.TempDir()

	// Create dist directory
	distDir := filepath.Join(tmpDir, "ui", "dist")
	if err := os.MkdirAll(distDir, 0o755); err != nil {
		t.Fatal(err)
	}
	indexHTML := filepath.Join(distDir, "index.html")
	if err := os.WriteFile(indexHTML, []byte("<html></html>"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create invalid service.json
	vrooliDir := filepath.Join(tmpDir, ".vrooli")
	if err := os.MkdirAll(vrooliDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(vrooliDir, "service.json"), []byte("not valid json"), 0o644); err != nil {
		t.Fatal(err)
	}

	c := NewChecker("http://localhost:4110")
	status, err := c.CheckBundleFreshness(context.Background(), tmpDir)
	if err != nil {
		t.Fatalf("CheckBundleFreshness() error = %v", err)
	}

	// Should still return fresh when config can't be parsed
	if !status.Fresh {
		t.Error("Fresh should be true when service.json is invalid (fallback behavior)")
	}
}

func TestChecker_CheckBundleFreshness_NoUIBundleCheck(t *testing.T) {
	tmpDir := t.TempDir()

	// Create dist directory
	distDir := filepath.Join(tmpDir, "ui", "dist")
	if err := os.MkdirAll(distDir, 0o755); err != nil {
		t.Fatal(err)
	}
	indexHTML := filepath.Join(distDir, "index.html")
	if err := os.WriteFile(indexHTML, []byte("<html></html>"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create service.json without ui-bundle check
	vrooliDir := filepath.Join(tmpDir, ".vrooli")
	if err := os.MkdirAll(vrooliDir, 0o755); err != nil {
		t.Fatal(err)
	}
	serviceJSON := `{
		"lifecycle": {
			"setup": {
				"condition": {
					"checks": [
						{"type": "other-check"}
					]
				}
			}
		}
	}`
	if err := os.WriteFile(filepath.Join(vrooliDir, "service.json"), []byte(serviceJSON), 0o644); err != nil {
		t.Fatal(err)
	}

	c := NewChecker("http://localhost:4110")
	status, err := c.CheckBundleFreshness(context.Background(), tmpDir)
	if err != nil {
		t.Fatalf("CheckBundleFreshness() error = %v", err)
	}

	if !status.Fresh {
		t.Error("Fresh should be true when no ui-bundle check in service.json")
	}
}

func TestChecker_CheckIframeBridge_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	uiDir := filepath.Join(tmpDir, "ui")
	if err := os.Mkdir(uiDir, 0o755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(uiDir, "package.json"), []byte("not valid json"), 0o644); err != nil {
		t.Fatal(err)
	}

	c := NewChecker("http://localhost:4110")
	status, err := c.CheckIframeBridge(context.Background(), tmpDir)
	if err != nil {
		t.Fatalf("CheckIframeBridge() error = %v", err)
	}

	if status.DependencyPresent {
		t.Error("DependencyPresent should be false for invalid package.json")
	}
	if status.Details == "" {
		t.Error("Details should describe the parse error")
	}
}

func TestChecker_WithHTTPClient(t *testing.T) {
	customClient := &http.Client{Timeout: 5 * time.Second}
	c := NewChecker("http://localhost:4110", WithHTTPClient(customClient))

	if c.httpClient != customClient {
		t.Error("httpClient should be set to custom client")
	}
}

func TestChecker_WithAppRoot(t *testing.T) {
	c := NewChecker("http://localhost:4110", WithAppRoot("/custom/root"))

	if c.appRoot != "/custom/root" {
		t.Errorf("appRoot = %q, want %q", c.appRoot, "/custom/root")
	}
}

func TestChecker_CheckBrowserless_Unreachable(t *testing.T) {
	c := NewChecker("http://localhost:99999") // Invalid port
	err := c.CheckBrowserless(context.Background())
	if err == nil {
		t.Error("CheckBrowserless() should return error for unreachable server")
	}
}

func TestChecker_CheckBundleFreshness_DistDirMissing(t *testing.T) {
	tmpDir := t.TempDir()

	// Create service.json with custom dist path that doesn't exist
	vrooliDir := filepath.Join(tmpDir, ".vrooli")
	if err := os.MkdirAll(vrooliDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// First create ui/dist/index.html so the initial check passes
	distDir := filepath.Join(tmpDir, "ui", "dist")
	if err := os.MkdirAll(distDir, 0o755); err != nil {
		t.Fatal(err)
	}
	indexHTML := filepath.Join(distDir, "index.html")
	if err := os.WriteFile(indexHTML, []byte("<html></html>"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create service.json with a different dist path that doesn't exist
	serviceJSON := `{
		"lifecycle": {
			"setup": {
				"condition": {
					"checks": [
						{
							"type": "ui-bundle",
							"dist_path": "build/output"
						}
					]
				}
			}
		}
	}`
	if err := os.WriteFile(filepath.Join(vrooliDir, "service.json"), []byte(serviceJSON), 0o644); err != nil {
		t.Fatal(err)
	}

	c := NewChecker("http://localhost:4110")
	status, err := c.CheckBundleFreshness(context.Background(), tmpDir)
	if err != nil {
		t.Fatalf("CheckBundleFreshness() error = %v", err)
	}

	if status.Fresh {
		t.Error("Fresh should be false when dist_path directory doesn't exist")
	}
}
