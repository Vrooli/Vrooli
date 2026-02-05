package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"
)

// --- Mock implementations for handler unit tests ---

type mockUpdateAppLookup struct {
	GetAppFn func(bundleKey, appKey string) (*DownloadApp, error)
}

func (m *mockUpdateAppLookup) GetApp(bundleKey, appKey string) (*DownloadApp, error) {
	return m.GetAppFn(bundleKey, appKey)
}

type mockUpdateAssetLookup struct {
	GetAssetByVariantFn func(bundleKey, appKey, platform, variantKey string) (*DownloadAsset, error)
}

func (m *mockUpdateAssetLookup) GetAssetByVariant(bundleKey, appKey, platform, variantKey string) (*DownloadAsset, error) {
	return m.GetAssetByVariantFn(bundleKey, appKey, platform, variantKey)
}

type mockUpdateArtifactResolver struct {
	GetArtifactFn                  func(ctx context.Context, bundleKey string, id int64) (*DownloadArtifact, error)
	GetCurrentArtifactByFilenameFn func(ctx context.Context, bundleKey, appKey, variantKey, filename string) (*DownloadArtifact, error)
	PresignGetArtifactFn           func(ctx context.Context, bundleKey string, artifact DownloadArtifact) (string, error)
}

func (m *mockUpdateArtifactResolver) GetArtifact(ctx context.Context, bundleKey string, id int64) (*DownloadArtifact, error) {
	return m.GetArtifactFn(ctx, bundleKey, id)
}

func (m *mockUpdateArtifactResolver) GetCurrentArtifactByFilename(ctx context.Context, bundleKey, appKey, variantKey, filename string) (*DownloadArtifact, error) {
	return m.GetCurrentArtifactByFilenameFn(ctx, bundleKey, appKey, variantKey, filename)
}

func (m *mockUpdateArtifactResolver) PresignGetArtifact(ctx context.Context, bundleKey string, artifact DownloadArtifact) (string, error) {
	return m.PresignGetArtifactFn(ctx, bundleKey, artifact)
}

type mockBundleKeyProvider struct {
	key string
}

func (m *mockBundleKeyProvider) BundleKey() string { return m.key }

// newMockUpdateRequest creates a request with mux vars pre-set.
func newMockUpdateRequest(appKey, channel, file string) *http.Request {
	req := httptest.NewRequest(http.MethodGet,
		fmt.Sprintf("/api/v1/updates/%s/%s/%s", appKey, channel, file), nil)
	return mux.SetURLVars(req, map[string]string{
		"app_key": appKey,
		"channel": channel,
		"file":    file,
	})
}

// --- manifestFilenameToPlatform ---

func TestManifestFilenameToPlatform(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		want     string
	}{
		{"windows manifest", "latest.yml", "windows"},
		{"mac manifest", "latest-mac.yml", "mac"},
		{"linux manifest", "latest-linux.yml", "linux"},
		{"binary filename", "my-app-1.2.3.exe", ""},
		{"empty string", "", ""},
		{"unknown yml", "latest-freebsd.yml", ""},
		{"partial match", "latest", ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := manifestFilenameToPlatform(tc.filename)
			if got != tc.want {
				t.Errorf("manifestFilenameToPlatform(%q) = %q, want %q", tc.filename, got, tc.want)
			}
		})
	}
}

// --- channelToVariantKey ---

func TestChannelToVariantKey(t *testing.T) {
	tests := []struct {
		name    string
		channel string
		want    string
	}{
		{"stable maps to default", "stable", "default"},
		{"empty maps to default", "", "default"},
		{"beta passes through", "beta", "beta"},
		{"alpha passes through", "alpha", "alpha"},
		{"custom channel", "nightly", "nightly"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := channelToVariantKey(tc.channel)
			if got != tc.want {
				t.Errorf("channelToVariantKey(%q) = %q, want %q", tc.channel, got, tc.want)
			}
		})
	}
}

// --- buildElectronManifest ---

func TestBuildElectronManifest(t *testing.T) {
	artifact := &DownloadArtifact{
		ReleaseVersion:   "1.2.3",
		OriginalFilename: "my-app-1.2.3.exe",
		SHA512:           "abc123sha512base64==",
		SizeBytes:        85234567,
		UpdatedAt:        time.Date(2026, 2, 5, 12, 0, 0, 0, time.UTC),
	}

	manifest := string(buildElectronManifest(artifact))

	// Verify required fields are present
	checks := []struct {
		label string
		want  string
	}{
		{"version", "version: 1.2.3"},
		{"path", "path: my-app-1.2.3.exe"},
		{"sha512 top-level", "sha512: abc123sha512base64=="},
		{"releaseDate", "releaseDate: 2026-02-05T12:00:00Z"},
		{"files url", "url: my-app-1.2.3.exe"},
		{"files sha512", "sha512: abc123sha512base64=="},
		{"files size", "size: 85234567"},
	}

	for _, c := range checks {
		if !strings.Contains(manifest, c.want) {
			t.Errorf("manifest missing %s: expected to contain %q\ngot:\n%s", c.label, c.want, manifest)
		}
	}
}

// --- handleUpdateFile integration tests ---

func TestHandleUpdateFile_MissingAppKey(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	downloads := NewDownloadService(db)
	hosting := NewDownloadHostingService(db)
	plans := newTestPlanService(t, "test_bundle")

	handler := handleUpdateFile(downloads, downloads, hosting, plans)

	// mux vars without app_key
	req := httptest.NewRequest(http.MethodGet, "/api/v1/updates///latest.yml", nil)
	req = mux.SetURLVars(req, map[string]string{"app_key": "", "channel": "stable", "file": "latest.yml"})
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestHandleUpdateFile_AppNotFound(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	downloads := NewDownloadService(db)
	hosting := NewDownloadHostingService(db)
	plans := newTestPlanService(t, "test_bundle")

	handler := handleUpdateFile(downloads, downloads, hosting, plans)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/updates/nonexistent/stable/latest.yml", nil)
	req = mux.SetURLVars(req, map[string]string{"app_key": "nonexistent", "channel": "stable", "file": "latest.yml"})
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp["error_type"] != ApiErrorTypeNotFound {
		t.Errorf("expected error_type %q, got %q", ApiErrorTypeNotFound, resp["error_type"])
	}
}

func TestHandleUpdateFile_APIKeyGating_Forbidden(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	cleanupDownloadApps(t, db)

	downloads := NewDownloadService(db)
	hosting := NewDownloadHostingService(db)
	plans := newTestPlanService(t, "test_bundle")

	// Create an app with an update_api_key
	_, err := downloads.UpsertDownloadApp(DownloadApp{
		BundleKey:    "test_bundle",
		AppKey:       "gated-app",
		Name:         "Gated App",
		UpdateAPIKey: "secret-key-123",
	})
	if err != nil {
		t.Fatalf("failed to create app: %v", err)
	}

	handler := handleUpdateFile(downloads, downloads, hosting, plans)

	t.Run("no key returns 403", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/updates/gated-app/stable/latest.yml", nil)
		req = mux.SetURLVars(req, map[string]string{"app_key": "gated-app", "channel": "stable", "file": "latest.yml"})
		w := httptest.NewRecorder()

		handler(w, req)

		if w.Code != http.StatusForbidden {
			t.Errorf("expected 403, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("wrong key returns 403", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/updates/gated-app/stable/latest.yml", nil)
		req = mux.SetURLVars(req, map[string]string{"app_key": "gated-app", "channel": "stable", "file": "latest.yml"})
		req.Header.Set("X-Update-Key", "wrong-key")
		w := httptest.NewRecorder()

		handler(w, req)

		if w.Code != http.StatusForbidden {
			t.Errorf("expected 403, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("correct key passes gating", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/updates/gated-app/stable/latest.yml", nil)
		req = mux.SetURLVars(req, map[string]string{"app_key": "gated-app", "channel": "stable", "file": "latest.yml"})
		req.Header.Set("X-Update-Key", "secret-key-123")
		w := httptest.NewRecorder()

		handler(w, req)

		// Should get past API key check — 404 because no asset exists for this platform
		if w.Code != http.StatusNotFound {
			t.Errorf("expected 404 (no asset), got %d: %s", w.Code, w.Body.String())
		}
	})
}

func TestHandleUpdateFile_PublicApp_NoKeyRequired(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	cleanupDownloadApps(t, db)

	downloads := NewDownloadService(db)
	hosting := NewDownloadHostingService(db)
	plans := newTestPlanService(t, "test_bundle")

	// Create an app without update_api_key (public)
	_, err := downloads.UpsertDownloadApp(DownloadApp{
		BundleKey: "test_bundle",
		AppKey:    "public-app",
		Name:      "Public App",
	})
	if err != nil {
		t.Fatalf("failed to create app: %v", err)
	}

	handler := handleUpdateFile(downloads, downloads, hosting, plans)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/updates/public-app/stable/latest.yml", nil)
	req = mux.SetURLVars(req, map[string]string{"app_key": "public-app", "channel": "stable", "file": "latest.yml"})
	w := httptest.NewRecorder()

	handler(w, req)

	// Should get past API key check — 404 because no asset exists
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 (no asset), got %d: %s", w.Code, w.Body.String())
	}
}

// --- Mock-based handler unit tests (no database required) ---

func TestHandleUpdateFileMock_ManifestServed(t *testing.T) {
	artifactID := int64(42)
	artifact := &DownloadArtifact{
		ID:               artifactID,
		ReleaseVersion:   "2.0.0",
		OriginalFilename: "my-app-2.0.0.exe",
		SHA512:           "fakeSHA512base64==",
		SizeBytes:        50000000,
		UpdatedAt:        time.Date(2026, 2, 5, 14, 0, 0, 0, time.UTC),
	}

	apps := &mockUpdateAppLookup{
		GetAppFn: func(_, _ string) (*DownloadApp, error) {
			return &DownloadApp{AppKey: "test-app"}, nil
		},
	}
	assets := &mockUpdateAssetLookup{
		GetAssetByVariantFn: func(_, _, _, _ string) (*DownloadAsset, error) {
			return &DownloadAsset{ArtifactID: &artifactID}, nil
		},
	}
	resolver := &mockUpdateArtifactResolver{
		GetArtifactFn: func(_ context.Context, _ string, _ int64) (*DownloadArtifact, error) {
			return artifact, nil
		},
	}
	bundles := &mockBundleKeyProvider{key: "test_bundle"}

	handler := handleUpdateFile(apps, assets, resolver, bundles)
	req := newMockUpdateRequest("test-app", "stable", "latest.yml")
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/x-yaml" {
		t.Errorf("expected Content-Type application/x-yaml, got %q", ct)
	}
	body := w.Body.String()
	for _, want := range []string{"version: 2.0.0", "sha512: fakeSHA512base64==", "size: 50000000"} {
		if !strings.Contains(body, want) {
			t.Errorf("manifest missing %q\ngot:\n%s", want, body)
		}
	}
}

func TestHandleUpdateFileMock_BinaryRedirect(t *testing.T) {
	artifact := &DownloadArtifact{
		ID:               99,
		OriginalFilename: "my-app-2.0.0.exe",
		Bucket:           "test-bucket",
		ObjectKey:        "artifacts/my-app-2.0.0.exe",
	}

	apps := &mockUpdateAppLookup{
		GetAppFn: func(_, _ string) (*DownloadApp, error) {
			return &DownloadApp{AppKey: "test-app"}, nil
		},
	}
	assets := &mockUpdateAssetLookup{}
	resolver := &mockUpdateArtifactResolver{
		GetCurrentArtifactByFilenameFn: func(_ context.Context, _, _, _, _ string) (*DownloadArtifact, error) {
			return artifact, nil
		},
		PresignGetArtifactFn: func(_ context.Context, _ string, _ DownloadArtifact) (string, error) {
			return "https://s3.example.com/presigned-url", nil
		},
	}
	bundles := &mockBundleKeyProvider{key: "test_bundle"}

	handler := handleUpdateFile(apps, assets, resolver, bundles)
	req := newMockUpdateRequest("test-app", "stable", "my-app-2.0.0.exe")
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("expected 302, got %d: %s", w.Code, w.Body.String())
	}
	loc := w.Header().Get("Location")
	if loc != "https://s3.example.com/presigned-url" {
		t.Errorf("expected redirect to presigned URL, got %q", loc)
	}
}

func TestHandleUpdateFileMock_MissingSHA512(t *testing.T) {
	artifactID := int64(42)

	apps := &mockUpdateAppLookup{
		GetAppFn: func(_, _ string) (*DownloadApp, error) {
			return &DownloadApp{AppKey: "test-app"}, nil
		},
	}
	assets := &mockUpdateAssetLookup{
		GetAssetByVariantFn: func(_, _, _, _ string) (*DownloadAsset, error) {
			return &DownloadAsset{ArtifactID: &artifactID}, nil
		},
	}
	resolver := &mockUpdateArtifactResolver{
		GetArtifactFn: func(_ context.Context, _ string, _ int64) (*DownloadArtifact, error) {
			return &DownloadArtifact{ID: artifactID, SHA512: ""}, nil
		},
	}
	bundles := &mockBundleKeyProvider{key: "test_bundle"}

	handler := handleUpdateFile(apps, assets, resolver, bundles)
	req := newMockUpdateRequest("test-app", "stable", "latest.yml")
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for missing SHA512, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "sha512") {
		t.Errorf("expected error message to mention sha512, got: %s", w.Body.String())
	}
}

func TestHandleUpdateFileMock_NilArtifactID(t *testing.T) {
	apps := &mockUpdateAppLookup{
		GetAppFn: func(_, _ string) (*DownloadApp, error) {
			return &DownloadApp{AppKey: "test-app"}, nil
		},
	}
	assets := &mockUpdateAssetLookup{
		GetAssetByVariantFn: func(_, _, _, _ string) (*DownloadAsset, error) {
			return &DownloadAsset{ArtifactID: nil}, nil
		},
	}
	resolver := &mockUpdateArtifactResolver{}
	bundles := &mockBundleKeyProvider{key: "test_bundle"}

	handler := handleUpdateFile(apps, assets, resolver, bundles)
	req := newMockUpdateRequest("test-app", "stable", "latest-mac.yml")
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for nil artifact_id, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "no managed artifact linked") {
		t.Errorf("expected 'no managed artifact linked' message, got: %s", w.Body.String())
	}
}

func TestHandleUpdateFileMock_PresignFailure(t *testing.T) {
	artifact := &DownloadArtifact{
		ID:               99,
		OriginalFilename: "my-app-2.0.0.exe",
	}

	apps := &mockUpdateAppLookup{
		GetAppFn: func(_, _ string) (*DownloadApp, error) {
			return &DownloadApp{AppKey: "test-app"}, nil
		},
	}
	assets := &mockUpdateAssetLookup{}
	resolver := &mockUpdateArtifactResolver{
		GetCurrentArtifactByFilenameFn: func(_ context.Context, _, _, _, _ string) (*DownloadArtifact, error) {
			return artifact, nil
		},
		PresignGetArtifactFn: func(_ context.Context, _ string, _ DownloadArtifact) (string, error) {
			return "", fmt.Errorf("storage unavailable")
		},
	}
	bundles := &mockBundleKeyProvider{key: "test_bundle"}

	handler := handleUpdateFile(apps, assets, resolver, bundles)
	req := newMockUpdateRequest("test-app", "stable", "my-app-2.0.0.exe")
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d: %s", w.Code, w.Body.String())
	}
}

func TestHandleUpdateFileMock_APIKeyGating(t *testing.T) {
	apps := &mockUpdateAppLookup{
		GetAppFn: func(_, _ string) (*DownloadApp, error) {
			return &DownloadApp{AppKey: "gated", UpdateAPIKey: "secret"}, nil
		},
	}
	assets := &mockUpdateAssetLookup{
		GetAssetByVariantFn: func(_, _, _, _ string) (*DownloadAsset, error) {
			return nil, ErrDownloadNotFound
		},
	}
	resolver := &mockUpdateArtifactResolver{}
	bundles := &mockBundleKeyProvider{key: "b"}

	handler := handleUpdateFile(apps, assets, resolver, bundles)

	t.Run("missing key", func(t *testing.T) {
		req := newMockUpdateRequest("gated", "stable", "latest.yml")
		w := httptest.NewRecorder()
		handler(w, req)
		if w.Code != http.StatusForbidden {
			t.Errorf("expected 403, got %d", w.Code)
		}
	})

	t.Run("correct key", func(t *testing.T) {
		req := newMockUpdateRequest("gated", "stable", "latest.yml")
		req.Header.Set("X-Update-Key", "secret")
		w := httptest.NewRecorder()
		handler(w, req)
		// Gets past key check, hits 404 from mock asset lookup
		if w.Code != http.StatusNotFound {
			t.Errorf("expected 404 (past key check), got %d", w.Code)
		}
	})
}
