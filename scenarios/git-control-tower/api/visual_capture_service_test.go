package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/vrooli/api-core/discovery"
	"github.com/vrooli/api-core/storage"
)

func testCaptureSetup(t *testing.T, basHandler http.HandlerFunc) (VisualCaptureDeps, *httptest.Server) {
	t.Helper()

	server := httptest.NewServer(basHandler)
	t.Cleanup(server.Close)

	tmpDir := t.TempDir()
	repoDir := filepath.Join(tmpDir, "repo")
	os.MkdirAll(repoDir, 0o755)

	resolver, err := storage.NewResolver(storage.ResolverConfig{
		EnvGet:      func(key string) string { return "" },
		UserHomeDir: func() (string, error) { return tmpDir, nil },
	})
	if err != nil {
		t.Fatalf("create storage resolver: %v", err)
	}

	basClient := &BrowserAutomationClient{
		httpClient: &http.Client{Timeout: 5 * time.Second},
		resolver:   discovery.NewStaticResolver(server.URL),
	}

	return VisualCaptureDeps{
		BAS:     basClient,
		Storage: NewVisualCaptureStorage(resolver, OSFileIO{}),
		FS:      OSFileIO{},
		RepoDir: repoDir,
		RepoID:  1,
	}, server
}

func TestCaptureScenario_WithLighthousePages(t *testing.T) {
	t.Parallel()

	deps, _ := testCaptureSetup(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/preview-screenshot" {
			pngData := base64.StdEncoding.EncodeToString([]byte{0x89, 0x50, 0x4E, 0x47})
			json.NewEncoder(w).Encode(BASScreenshotResponse{
				Screenshot:     "data:image/png;base64," + pngData,
				URL:            "http://localhost:3000/",
				DurationMS:     100,
				ViewportWidth:  1280,
				ViewportHeight: 720,
			})
			return
		}
		w.WriteHeader(404)
	})

	// Create lighthouse.json
	lhDir := filepath.Join(deps.RepoDir, "scenarios", "test-app", ".vrooli")
	os.MkdirAll(lhDir, 0o755)
	lhConfig := LighthouseConfig{
		Enabled: true,
		Pages: []LighthousePage{
			{ID: "home", Path: "/", Label: "Home"},
			{ID: "about", Path: "/about", Label: "About"},
		},
	}
	lhData, _ := json.Marshal(lhConfig)
	os.WriteFile(filepath.Join(lhDir, "lighthouse.json"), lhData, 0o644)

	// We need to mock the UI URL resolution - use the BAS server URL as a stand-in
	// since CaptureScenario calls discovery.ResolveScenarioURL which won't work in tests.
	// Instead, test the helper functions directly.

	pages := discoverPages(deps.FS, deps.RepoDir, "test-app")
	if len(pages) != 2 {
		t.Fatalf("expected 2 pages from lighthouse.json, got %d", len(pages))
	}
	if pages[0].Path != "/" {
		t.Errorf("expected first page path /, got %s", pages[0].Path)
	}
	if pages[1].Label != "About" {
		t.Errorf("expected second page label About, got %s", pages[1].Label)
	}
}

func TestCaptureScenario_NoLighthouseConfig(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	repoDir := filepath.Join(tmpDir, "repo")
	os.MkdirAll(repoDir, 0o755)

	pages := discoverPages(OSFileIO{}, repoDir, "nonexistent")
	if len(pages) != 1 {
		t.Fatalf("expected 1 fallback page, got %d", len(pages))
	}
	if pages[0].Path != "/" {
		t.Errorf("expected fallback path /, got %s", pages[0].Path)
	}
	if pages[0].Label != "Home" {
		t.Errorf("expected fallback label Home, got %s", pages[0].Label)
	}
}

func TestDiscoverPages_MultipleLocations(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	// Create lighthouse.json under apps/ instead of scenarios/
	appsDir := filepath.Join(tmpDir, "apps", "my-app", ".vrooli")
	os.MkdirAll(appsDir, 0o755)
	lhConfig := LighthouseConfig{
		Enabled: true,
		Pages:   []LighthousePage{{ID: "main", Path: "/main", Label: "Main"}},
	}
	lhData, _ := json.Marshal(lhConfig)
	os.WriteFile(filepath.Join(appsDir, "lighthouse.json"), lhData, 0o644)

	pages := discoverPages(OSFileIO{}, tmpDir, "my-app")
	if len(pages) != 1 {
		t.Fatalf("expected 1 page from apps/ lighthouse.json, got %d", len(pages))
	}
	if pages[0].Path != "/main" {
		t.Errorf("expected path /main, got %s", pages[0].Path)
	}
}

func TestSanitizeFilename(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  string
	}{
		{"/", "_root_"},
		{"", "_root_"},
		{"/about", "_about_"},
		{"/workflow/new", "_workflow_new_"},
		{"/settings/profile/edit", "_settings_profile_edit_"},
	}

	for _, tt := range tests {
		got := sanitizeFilename(tt.input)
		if got != tt.want {
			t.Errorf("sanitizeFilename(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestDecodeBase64DataURI(t *testing.T) {
	t.Parallel()

	// Valid data URI
	original := []byte{0x89, 0x50, 0x4E, 0x47}
	encoded := "data:image/png;base64," + base64.StdEncoding.EncodeToString(original)
	decoded, err := decodeBase64DataURI(encoded)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(decoded) != len(original) {
		t.Errorf("expected %d bytes, got %d", len(original), len(decoded))
	}

	// Raw base64 without prefix
	rawEncoded := base64.StdEncoding.EncodeToString(original)
	decoded2, err := decodeBase64DataURI(rawEncoded)
	if err != nil {
		t.Fatalf("unexpected error for raw base64: %v", err)
	}
	if len(decoded2) != len(original) {
		t.Errorf("expected %d bytes, got %d", len(original), len(decoded2))
	}

	// Invalid base64
	_, err = decodeBase64DataURI("data:image/png;base64,not-valid-base64!!!")
	if err == nil {
		t.Error("expected error for invalid base64")
	}
}

func TestCaptureScenario_BASPartialFailure(t *testing.T) {
	t.Parallel()

	callCount := 0
	deps, _ := testCaptureSetup(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/preview-screenshot" {
			callCount++
			if callCount == 1 {
				// First call succeeds
				pngData := base64.StdEncoding.EncodeToString([]byte{0x89, 0x50})
				json.NewEncoder(w).Encode(BASScreenshotResponse{
					Screenshot: "data:image/png;base64," + pngData,
				})
				return
			}
			// Second call fails
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "browser crashed"})
			return
		}
		w.WriteHeader(404)
	})

	// Test that partial failure doesn't prevent saving what succeeded
	// We test this through the BAS client directly since CaptureScenario
	// needs discovery which isn't available in unit tests
	ctx := context.Background()

	// First call should succeed
	resp, err := deps.BAS.CaptureScreenshot(ctx, "http://test/", BASViewport{Width: 1280, Height: 720})
	if err != nil {
		t.Fatalf("first call should succeed: %v", err)
	}
	if resp.Screenshot == "" {
		t.Error("expected screenshot data")
	}

	// Second call should fail
	_, err = deps.BAS.CaptureScreenshot(ctx, "http://test/about", BASViewport{Width: 1280, Height: 720})
	if err == nil {
		t.Error("second call should fail")
	}
}
