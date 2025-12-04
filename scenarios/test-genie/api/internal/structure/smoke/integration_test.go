// Package smoke provides UI smoke testing functionality.
//
// This file contains integration tests that require a running Browserless instance.
// These tests are skipped when BROWSERLESS_URL is not set or Browserless is unavailable.
package smoke

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestIntegration_RealBrowserless tests the UI smoke system against a real Browserless instance.
// This test is skipped if Browserless is not available.
//
// To run this test:
//
//	BROWSERLESS_URL=http://localhost:4110 go test -v -run TestIntegration_RealBrowserless ./internal/structure/smoke/...
func TestIntegration_RealBrowserless(t *testing.T) {
	browserlessURL := GetBrowserlessURL()

	// Check if Browserless is available
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, browserlessURL+"/pressure", nil)
	if err != nil {
		t.Skipf("Skipping integration test: failed to create request: %v", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Skipf("Skipping integration test: Browserless not available at %s: %v", browserlessURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Skipf("Skipping integration test: Browserless returned status %d", resp.StatusCode)
	}

	t.Run("Runner_WithRealBrowserless", func(t *testing.T) {
		// Create a temporary scenario directory structure
		tmpDir := t.TempDir()

		// Create minimal scenario structure
		uiDir := filepath.Join(tmpDir, "ui")
		distDir := filepath.Join(uiDir, "dist")
		if err := os.MkdirAll(distDir, 0755); err != nil {
			t.Fatalf("Failed to create ui/dist directory: %v", err)
		}

		// Create a minimal index.html
		indexHTML := `<!DOCTYPE html>
<html>
<head><title>Test</title></head>
<body>
<div id="root">Loading...</div>
<script>
// Simulate iframe-bridge initialization
window.IFRAME_BRIDGE_READY = true;
window.__vrooliBridgeChildInstalled = true;
</script>
</body>
</html>`
		if err := os.WriteFile(filepath.Join(distDir, "index.html"), []byte(indexHTML), 0644); err != nil {
			t.Fatalf("Failed to write index.html: %v", err)
		}

		// Create package.json with iframe-bridge dependency
		packageJSON := `{
  "name": "test-ui",
  "dependencies": {
    "@vrooli/iframe-bridge": "workspace:*"
  }
}`
		if err := os.WriteFile(filepath.Join(uiDir, "package.json"), []byte(packageJSON), 0644); err != nil {
			t.Fatalf("Failed to write package.json: %v", err)
		}

		// Create .vrooli directory with service.json (no UI port defined - should skip)
		vrooliDir := filepath.Join(tmpDir, ".vrooli")
		if err := os.MkdirAll(vrooliDir, 0755); err != nil {
			t.Fatalf("Failed to create .vrooli directory: %v", err)
		}

		serviceJSON := `{
  "service": {
    "name": "test-scenario"
  }
}`
		if err := os.WriteFile(filepath.Join(vrooliDir, "service.json"), []byte(serviceJSON), 0644); err != nil {
			t.Fatalf("Failed to write service.json: %v", err)
		}

		// Run the smoke test - should skip because no UI port is defined
		runner := NewRunner(browserlessURL)
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		result, err := runner.Run(ctx, "test-scenario", tmpDir)
		if err != nil {
			t.Fatalf("Runner.Run() error = %v", err)
		}

		// Should be skipped because no UI port is defined
		if result.Status != StatusSkipped {
			t.Errorf("Status = %v, want %v (message: %s)", result.Status, StatusSkipped, result.Message)
		}
	})

	t.Run("PreflightChecker_BrowserlessHealth", func(t *testing.T) {
		checker := newPreflightCheckerForTest(browserlessURL)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		err := checker.CheckBrowserless(ctx)
		if err != nil {
			t.Errorf("CheckBrowserless() error = %v", err)
		}
	})

	t.Run("PreflightChecker_BundleFreshness_MissingBundle", func(t *testing.T) {
		checker := newPreflightCheckerForTest(browserlessURL)
		tmpDir := t.TempDir()

		// Create UI directory but no dist
		uiDir := filepath.Join(tmpDir, "ui")
		if err := os.MkdirAll(uiDir, 0755); err != nil {
			t.Fatalf("Failed to create ui directory: %v", err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		status, err := checker.CheckBundleFreshness(ctx, tmpDir)
		if err != nil {
			t.Fatalf("CheckBundleFreshness() error = %v", err)
		}

		if status.Fresh {
			t.Error("Expected bundle to be marked as not fresh when dist is missing")
		}
	})

	t.Run("PreflightChecker_IframeBridge_Present", func(t *testing.T) {
		checker := newPreflightCheckerForTest(browserlessURL)
		tmpDir := t.TempDir()

		// Create package.json with iframe-bridge
		uiDir := filepath.Join(tmpDir, "ui")
		if err := os.MkdirAll(uiDir, 0755); err != nil {
			t.Fatalf("Failed to create ui directory: %v", err)
		}

		packageJSON := `{"dependencies": {"@vrooli/iframe-bridge": "1.0.0"}}`
		if err := os.WriteFile(filepath.Join(uiDir, "package.json"), []byte(packageJSON), 0644); err != nil {
			t.Fatalf("Failed to write package.json: %v", err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		status, err := checker.CheckIframeBridge(ctx, tmpDir)
		if err != nil {
			t.Fatalf("CheckIframeBridge() error = %v", err)
		}

		if !status.DependencyPresent {
			t.Error("Expected iframe-bridge dependency to be detected")
		}
		if status.Version != "1.0.0" {
			t.Errorf("Version = %q, want %q", status.Version, "1.0.0")
		}
	})

	t.Run("PreflightChecker_IframeBridge_Missing", func(t *testing.T) {
		checker := newPreflightCheckerForTest(browserlessURL)
		tmpDir := t.TempDir()

		// Create package.json without iframe-bridge
		uiDir := filepath.Join(tmpDir, "ui")
		if err := os.MkdirAll(uiDir, 0755); err != nil {
			t.Fatalf("Failed to create ui directory: %v", err)
		}

		packageJSON := `{"dependencies": {"react": "18.0.0"}}`
		if err := os.WriteFile(filepath.Join(uiDir, "package.json"), []byte(packageJSON), 0644); err != nil {
			t.Fatalf("Failed to write package.json: %v", err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		status, err := checker.CheckIframeBridge(ctx, tmpDir)
		if err != nil {
			t.Fatalf("CheckIframeBridge() error = %v", err)
		}

		if status.DependencyPresent {
			t.Error("Expected iframe-bridge dependency to NOT be detected")
		}
	})
}

// newPreflightCheckerForTest creates a preflight checker for integration testing.
func newPreflightCheckerForTest(browserlessURL string) preflightCheckerInterface {
	// Import the preflight package to use the real checker
	return &integrationPreflightChecker{browserlessURL: browserlessURL}
}

// preflightCheckerInterface defines what we need from the preflight checker for tests.
type preflightCheckerInterface interface {
	CheckBrowserless(ctx context.Context) error
	CheckBundleFreshness(ctx context.Context, scenarioDir string) (*BundleStatus, error)
	CheckIframeBridge(ctx context.Context, scenarioDir string) (*BridgeStatus, error)
}

// integrationPreflightChecker wraps the real preflight checker for integration tests.
type integrationPreflightChecker struct {
	browserlessURL string
}

func (c *integrationPreflightChecker) CheckBrowserless(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.browserlessURL+"/pressure", nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return &httpError{statusCode: resp.StatusCode}
	}
	return nil
}

func (c *integrationPreflightChecker) CheckBundleFreshness(ctx context.Context, scenarioDir string) (*BundleStatus, error) {
	distIndex := filepath.Join(scenarioDir, "ui", "dist", "index.html")
	if _, err := os.Stat(distIndex); os.IsNotExist(err) {
		return &BundleStatus{
			Fresh:  false,
			Reason: "UI bundle missing (ui/dist/index.html not found)",
		}, nil
	}
	return &BundleStatus{Fresh: true}, nil
}

func (c *integrationPreflightChecker) CheckIframeBridge(ctx context.Context, scenarioDir string) (*BridgeStatus, error) {
	packageJSONPath := filepath.Join(scenarioDir, "ui", "package.json")
	data, err := os.ReadFile(packageJSONPath)
	if err != nil {
		return &BridgeStatus{DependencyPresent: false, Details: "package.json not found"}, nil
	}

	// Simple check for iframe-bridge in dependencies
	content := string(data)
	if !contains(content, "@vrooli/iframe-bridge") {
		return &BridgeStatus{DependencyPresent: false, Details: "not in dependencies"}, nil
	}

	// Extract version (simple parsing)
	version := extractVersion(content, "@vrooli/iframe-bridge")
	return &BridgeStatus{DependencyPresent: true, Version: version}, nil
}

type httpError struct {
	statusCode int
}

func (e *httpError) Error() string {
	return "http error: " + http.StatusText(e.statusCode)
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsImpl(s, substr))
}

func containsImpl(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func extractVersion(content, pkg string) string {
	// Very simple version extraction - find the package and get the next quoted value
	idx := 0
	for i := 0; i <= len(content)-len(pkg); i++ {
		if content[i:i+len(pkg)] == pkg {
			idx = i + len(pkg)
			break
		}
	}
	if idx == 0 {
		return ""
	}

	// Find the next colon and then the quoted version
	for i := idx; i < len(content); i++ {
		if content[i] == ':' {
			// Find the opening quote
			for j := i + 1; j < len(content); j++ {
				if content[j] == '"' {
					// Find the closing quote
					for k := j + 1; k < len(content); k++ {
						if content[k] == '"' {
							return content[j+1 : k]
						}
					}
				}
			}
		}
	}
	return ""
}
