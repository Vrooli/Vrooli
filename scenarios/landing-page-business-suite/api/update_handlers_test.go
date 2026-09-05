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

	"landing-page-business-suite-api/internal/delivery"

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

	t.Run("without release notes", func(t *testing.T) {
		manifest := string(buildElectronManifest(artifact, ""))

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

		if strings.Contains(manifest, "releaseNotes") {
			t.Errorf("manifest should not include releaseNotes when empty\ngot:\n%s", manifest)
		}
	})

	t.Run("with release notes", func(t *testing.T) {
		manifest := string(buildElectronManifest(artifact, "Bug fixes and improvements"))

		if !strings.Contains(manifest, "releaseNotes: Bug fixes and improvements") {
			t.Errorf("manifest should include releaseNotes\ngot:\n%s", manifest)
		}
	})
}

// --- handleUpdateFile integration tests ---

func TestHandleUpdateFile_MissingAppKey(t *testing.T) {
	db := setupTestDB(t)

	downloads := NewDownloadService(db)
	hosting := NewDownloadHostingService(db)
	plans := newTestPlanService(t, "test_bundle")

	middleware := requireUpdateAPIKey(downloads, plans)
	handler := middleware(handleUpdateFile(downloads, hosting, plans))

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

	downloads := NewDownloadService(db)
	hosting := NewDownloadHostingService(db)
	plans := newTestPlanService(t, "test_bundle")

	middleware := requireUpdateAPIKey(downloads, plans)
	handler := middleware(handleUpdateFile(downloads, hosting, plans))

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
	cleanupDownloadApps(t, db)

	downloads := NewDownloadService(db)
	hosting := NewDownloadHostingService(db)
	plans := newTestPlanService(t, "test_bundle")

	// Create an app with an update_api_key
	_, err := downloads.UpsertApp(DownloadApp{
		BundleKey:    "test_bundle",
		AppKey:       "gated-app",
		Name:         "Gated App",
		UpdateAPIKey: "secret-key-123",
	})
	if err != nil {
		t.Fatalf("failed to create app: %v", err)
	}

	middleware := requireUpdateAPIKey(downloads, plans)
	handler := middleware(handleUpdateFile(downloads, hosting, plans))

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
	cleanupDownloadApps(t, db)

	downloads := NewDownloadService(db)
	hosting := NewDownloadHostingService(db)
	plans := newTestPlanService(t, "test_bundle")

	// Create an app without update_api_key (public)
	_, err := downloads.UpsertApp(DownloadApp{
		BundleKey: "test_bundle",
		AppKey:    "public-app",
		Name:      "Public App",
	})
	if err != nil {
		t.Fatalf("failed to create app: %v", err)
	}

	middleware := requireUpdateAPIKey(downloads, plans)
	handler := middleware(handleUpdateFile(downloads, hosting, plans))

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

	handler := handleUpdateFile(assets, resolver, bundles)
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

func TestHandleUpdateFileMock_ManifestWithReleaseNotes(t *testing.T) {
	artifactID := int64(42)
	artifact := &DownloadArtifact{
		ID:               artifactID,
		ReleaseVersion:   "2.0.0",
		OriginalFilename: "my-app-2.0.0.exe",
		SHA512:           "fakeSHA512base64==",
		SizeBytes:        50000000,
		UpdatedAt:        time.Date(2026, 2, 5, 14, 0, 0, 0, time.UTC),
	}

	assets := &mockUpdateAssetLookup{
		GetAssetByVariantFn: func(_, _, _, _ string) (*DownloadAsset, error) {
			return &DownloadAsset{ArtifactID: &artifactID, ReleaseNotes: "Bug fixes and improvements"}, nil
		},
	}
	resolver := &mockUpdateArtifactResolver{
		GetArtifactFn: func(_ context.Context, _ string, _ int64) (*DownloadArtifact, error) {
			return artifact, nil
		},
	}
	bundles := &mockBundleKeyProvider{key: "test_bundle"}

	handler := handleUpdateFile(assets, resolver, bundles)
	req := newMockUpdateRequest("test-app", "stable", "latest.yml")
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	if !strings.Contains(body, "releaseNotes: Bug fixes and improvements") {
		t.Errorf("manifest should include releaseNotes\ngot:\n%s", body)
	}
}

func TestHandleUpdateFileMock_BinaryRedirect(t *testing.T) {
	artifact := &DownloadArtifact{
		ID:               99,
		OriginalFilename: "my-app-2.0.0.exe",
		Bucket:           "test-bucket",
		ObjectKey:        "artifacts/my-app-2.0.0.exe",
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

	handler := handleUpdateFile(assets, resolver, bundles)
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

	handler := handleUpdateFile(assets, resolver, bundles)
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
	assets := &mockUpdateAssetLookup{
		GetAssetByVariantFn: func(_, _, _, _ string) (*DownloadAsset, error) {
			return &DownloadAsset{ArtifactID: nil}, nil
		},
	}
	resolver := &mockUpdateArtifactResolver{}
	bundles := &mockBundleKeyProvider{key: "test_bundle"}

	handler := handleUpdateFile(assets, resolver, bundles)
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

	handler := handleUpdateFile(assets, resolver, bundles)
	req := newMockUpdateRequest("test-app", "stable", "my-app-2.0.0.exe")
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d: %s", w.Code, w.Body.String())
	}
}

func TestRequireUpdateAPIKeyMiddleware_Mock(t *testing.T) {
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
	bundles := &mockBundleKeyProvider{key: "b"}

	middleware := requireUpdateAPIKey(apps, bundles)
	handler := middleware(handleUpdateFile(assets, &mockUpdateArtifactResolver{}, bundles))

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

// --- Channel discovery tests ---

type mockChannelDiscoveryLookup struct {
	ListChannelsFn func(bundleKey, appKey string) ([]delivery.ChannelInfo, error)
}

func (m *mockChannelDiscoveryLookup) ListChannels(bundleKey, appKey string) ([]delivery.ChannelInfo, error) {
	return m.ListChannelsFn(bundleKey, appKey)
}

func TestHandleChannelDiscovery(t *testing.T) {
	channelsMock := &mockChannelDiscoveryLookup{
		ListChannelsFn: func(_, _ string) ([]delivery.ChannelInfo, error) {
			return []delivery.ChannelInfo{
				{Channel: "stable", Platform: "windows", Version: "1.0.0", UpdatedAt: "2026-01-01T00:00:00Z"},
				{Channel: "beta", Platform: "windows", Version: "1.1.0-beta", UpdatedAt: "2026-01-02T00:00:00Z"},
			}, nil
		},
	}
	bundles := &mockBundleKeyProvider{key: "test_bundle"}

	handler := handleChannelDiscovery(channelsMock, bundles)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/updates/test-app/channels", nil)
	req = mux.SetURLVars(req, map[string]string{"app_key": "test-app"})
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var result []delivery.ChannelInfo
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 channels, got %d", len(result))
	}
}

// --- Update policy tests ---

func TestHandleUpdatePolicy_CRUD(t *testing.T) {
	db := setupTestDB(t)
	cleanupDownloadApps(t, db)

	downloads := NewDownloadService(db)
	plans := newTestPlanService(t, "test_bundle")

	_, err := downloads.UpsertApp(DownloadApp{
		BundleKey: "test_bundle",
		AppKey:    "policy-app",
		Name:      "Policy App",
	})
	if err != nil {
		t.Fatalf("failed to create app: %v", err)
	}

	t.Run("GET returns defaults", func(t *testing.T) {
		handler := handleGetUpdatePolicy(downloads, plans)
		req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/download-apps/policy-app/update-policy", nil)
		req = mux.SetURLVars(req, map[string]string{"app_key": "policy-app"})
		w := httptest.NewRecorder()

		handler(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
		var policy map[string]interface{}
		if err := json.NewDecoder(w.Body).Decode(&policy); err != nil {
			t.Fatalf("failed to decode: %v", err)
		}
		if policy["update_mode"] != "optional" {
			t.Errorf("expected default update_mode 'optional', got %v", policy["update_mode"])
		}
	})

	t.Run("PUT updates policy", func(t *testing.T) {
		handler := handlePutUpdatePolicy(downloads, plans)
		body := strings.NewReader(`{"check_interval_hours": 12, "update_mode": "mandatory", "allow_downgrade": true}`)
		req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/download-apps/policy-app/update-policy", body)
		req.Header.Set("Content-Type", "application/json")
		req = mux.SetURLVars(req, map[string]string{"app_key": "policy-app"})
		w := httptest.NewRecorder()

		handler(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("PUT rejects invalid mode", func(t *testing.T) {
		handler := handlePutUpdatePolicy(downloads, plans)
		body := strings.NewReader(`{"check_interval_hours": 4, "update_mode": "invalid", "allow_downgrade": false}`)
		req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/download-apps/policy-app/update-policy", body)
		req.Header.Set("Content-Type", "application/json")
		req = mux.SetURLVars(req, map[string]string{"app_key": "policy-app"})
		w := httptest.NewRecorder()

		handler(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
		}
	})
}

// --- UpsertAsset variant_key test ---

func TestUpsertAsset_NonDefaultVariantKey(t *testing.T) {
	db := setupTestDB(t)
	cleanupDownloadApps(t, db)

	downloads := NewDownloadService(db)
	plans := newTestPlanService(t, "test_bundle")

	_, err := downloads.UpsertApp(DownloadApp{
		BundleKey: plans.BundleKey(),
		AppKey:    "variant-app",
		Name:      "Variant App",
	})
	if err != nil {
		t.Fatalf("failed to create app: %v", err)
	}

	// Upsert with beta variant
	_, err = downloads.UpsertAsset(context.Background(), DownloadAsset{
		BundleKey:      plans.BundleKey(),
		AppKey:         "variant-app",
		Platform:       "windows",
		VariantKey:     "beta",
		ArtifactURL:    "https://example.com/beta.exe",
		ArtifactSource: "direct",
		ReleaseVersion: "1.0.0-beta",
	})
	if err != nil {
		t.Fatalf("upsert with beta variant_key failed: %v", err)
	}

	// Also upsert with default variant for same platform
	_, err = downloads.UpsertAsset(context.Background(), DownloadAsset{
		BundleKey:      plans.BundleKey(),
		AppKey:         "variant-app",
		Platform:       "windows",
		ArtifactURL:    "https://example.com/stable.exe",
		ArtifactSource: "direct",
		ReleaseVersion: "1.0.0",
	})
	if err != nil {
		t.Fatalf("upsert with default variant_key failed: %v", err)
	}

	// Fetch beta variant
	beta, err := downloads.GetAssetByVariant(plans.BundleKey(), "variant-app", "windows", "beta")
	if err != nil {
		t.Fatalf("failed to get beta variant: %v", err)
	}
	if beta.ReleaseVersion != "1.0.0-beta" {
		t.Errorf("expected beta version '1.0.0-beta', got %q", beta.ReleaseVersion)
	}

	// Fetch default variant
	stable, err := downloads.GetAssetByVariant(plans.BundleKey(), "variant-app", "windows", "default")
	if err != nil {
		t.Fatalf("failed to get default variant: %v", err)
	}
	if stable.ReleaseVersion != "1.0.0" {
		t.Errorf("expected stable version '1.0.0', got %q", stable.ReleaseVersion)
	}
}

// --- ListChannels integration test ---

func TestListChannels_Integration(t *testing.T) {
	db := setupTestDB(t)
	cleanupDownloadApps(t, db)

	downloads := NewDownloadService(db)
	plans := newTestPlanService(t, "test_bundle")

	_, err := downloads.UpsertApp(DownloadApp{
		BundleKey: plans.BundleKey(),
		AppKey:    "channels-app",
		Name:      "Channels App",
	})
	if err != nil {
		t.Fatalf("failed to create app: %v", err)
	}

	// Add stable windows
	_, err = downloads.UpsertAsset(context.Background(), DownloadAsset{
		BundleKey:      plans.BundleKey(),
		AppKey:         "channels-app",
		Platform:       "windows",
		ArtifactURL:    "https://example.com/stable.exe",
		ArtifactSource: "direct",
		ReleaseVersion: "1.0.0",
	})
	if err != nil {
		t.Fatalf("upsert stable failed: %v", err)
	}

	// Add beta windows
	_, err = downloads.UpsertAsset(context.Background(), DownloadAsset{
		BundleKey:      plans.BundleKey(),
		AppKey:         "channels-app",
		Platform:       "windows",
		VariantKey:     "beta",
		ArtifactURL:    "https://example.com/beta.exe",
		ArtifactSource: "direct",
		ReleaseVersion: "1.1.0-beta",
	})
	if err != nil {
		t.Fatalf("upsert beta failed: %v", err)
	}

	channels, err := downloads.ListChannels(plans.BundleKey(), "channels-app")
	if err != nil {
		t.Fatalf("ListChannels failed: %v", err)
	}

	if len(channels) != 2 {
		t.Fatalf("expected 2 channels, got %d", len(channels))
	}

	// Find the beta channel
	var foundBeta bool
	for _, ch := range channels {
		if ch.Channel == "beta" && ch.Platform == "windows" && ch.Version == "1.1.0-beta" {
			foundBeta = true
		}
	}
	if !foundBeta {
		t.Errorf("expected to find beta channel, channels: %+v", channels)
	}
}

// --- Verification endpoint tests ---

type mockVerifyArtifactResolver struct {
	GetArtifactFn        func(ctx context.Context, bundleKey string, id int64) (*DownloadArtifact, error)
	PresignGetArtifactFn func(ctx context.Context, bundleKey string, artifact DownloadArtifact) (string, error)
	HeadArtifactFn       func(ctx context.Context, bundleKey string, artifact DownloadArtifact) error
}

func (m *mockVerifyArtifactResolver) GetArtifact(ctx context.Context, bundleKey string, id int64) (*DownloadArtifact, error) {
	return m.GetArtifactFn(ctx, bundleKey, id)
}

func (m *mockVerifyArtifactResolver) PresignGetArtifact(ctx context.Context, bundleKey string, artifact DownloadArtifact) (string, error) {
	return m.PresignGetArtifactFn(ctx, bundleKey, artifact)
}

func (m *mockVerifyArtifactResolver) HeadArtifact(ctx context.Context, bundleKey string, artifact DownloadArtifact) error {
	return m.HeadArtifactFn(ctx, bundleKey, artifact)
}

func TestHandleUpdateVerify_LightweightMatch(t *testing.T) {
	artifactID := int64(42)

	assets := &mockUpdateAssetLookup{
		GetAssetByVariantFn: func(_, _, _, _ string) (*DownloadAsset, error) {
			return &DownloadAsset{ArtifactID: &artifactID}, nil
		},
	}
	resolver := &mockVerifyArtifactResolver{
		GetArtifactFn: func(_ context.Context, _ string, _ int64) (*DownloadArtifact, error) {
			return &DownloadArtifact{
				ID:             artifactID,
				ReleaseVersion: "2.0.0",
				SHA512:         "abc123",
			}, nil
		},
	}
	bundles := &mockBundleKeyProvider{key: "test_bundle"}

	handler := handleUpdateVerify(assets, resolver, bundles)
	req := httptest.NewRequest(http.MethodGet,
		"/api/v1/updates/test-app/verify?channel=stable&platform=windows&expected_version=2.0.0", nil)
	req = mux.SetURLVars(req, map[string]string{"app_key": "test-app"})
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if resp["match"] != true {
		t.Errorf("expected match=true, got %v", resp["match"])
	}
	if resp["actual_version"] != "2.0.0" {
		t.Errorf("expected actual_version=2.0.0, got %v", resp["actual_version"])
	}
	// Deep fields should NOT be present in lightweight mode
	if _, ok := resp["artifact_accessible"]; ok {
		t.Errorf("artifact_accessible should not be present in lightweight mode")
	}
	if _, ok := resp["presign_valid"]; ok {
		t.Errorf("presign_valid should not be present in lightweight mode")
	}
}

func TestHandleUpdateVerify_LightweightMismatch(t *testing.T) {
	artifactID := int64(42)

	assets := &mockUpdateAssetLookup{
		GetAssetByVariantFn: func(_, _, _, _ string) (*DownloadAsset, error) {
			return &DownloadAsset{ArtifactID: &artifactID}, nil
		},
	}
	resolver := &mockVerifyArtifactResolver{
		GetArtifactFn: func(_ context.Context, _ string, _ int64) (*DownloadArtifact, error) {
			return &DownloadArtifact{
				ID:             artifactID,
				ReleaseVersion: "1.9.0",
				SHA512:         "abc123",
			}, nil
		},
	}
	bundles := &mockBundleKeyProvider{key: "test_bundle"}

	handler := handleUpdateVerify(assets, resolver, bundles)
	req := httptest.NewRequest(http.MethodGet,
		"/api/v1/updates/test-app/verify?channel=stable&platform=windows&expected_version=2.0.0", nil)
	req = mux.SetURLVars(req, map[string]string{"app_key": "test-app"})
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if resp["match"] != false {
		t.Errorf("expected match=false for version mismatch, got %v", resp["match"])
	}
	if resp["actual_version"] != "1.9.0" {
		t.Errorf("expected actual_version=1.9.0, got %v", resp["actual_version"])
	}
}

func TestHandleUpdateVerify_DeepMode(t *testing.T) {
	artifactID := int64(42)

	assets := &mockUpdateAssetLookup{
		GetAssetByVariantFn: func(_, _, _, _ string) (*DownloadAsset, error) {
			return &DownloadAsset{ArtifactID: &artifactID}, nil
		},
	}
	resolver := &mockVerifyArtifactResolver{
		GetArtifactFn: func(_ context.Context, _ string, _ int64) (*DownloadArtifact, error) {
			return &DownloadArtifact{
				ID:             artifactID,
				ReleaseVersion: "2.0.0",
				SHA512:         "abc123",
				Bucket:         "test-bucket",
				ObjectKey:      "artifacts/test.exe",
			}, nil
		},
		HeadArtifactFn: func(_ context.Context, _ string, _ DownloadArtifact) error {
			return nil // S3 HEAD succeeds
		},
		PresignGetArtifactFn: func(_ context.Context, _ string, _ DownloadArtifact) (string, error) {
			return "https://s3.example.com/presigned", nil
		},
	}
	bundles := &mockBundleKeyProvider{key: "test_bundle"}

	handler := handleUpdateVerify(assets, resolver, bundles)
	req := httptest.NewRequest(http.MethodGet,
		"/api/v1/updates/test-app/verify?channel=stable&platform=windows&expected_version=2.0.0&deep=true", nil)
	req = mux.SetURLVars(req, map[string]string{"app_key": "test-app"})
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if resp["match"] != true {
		t.Errorf("expected match=true, got %v", resp["match"])
	}
	if resp["artifact_accessible"] != true {
		t.Errorf("expected artifact_accessible=true, got %v", resp["artifact_accessible"])
	}
	if resp["presign_valid"] != true {
		t.Errorf("expected presign_valid=true, got %v", resp["presign_valid"])
	}
}

func TestHandleUpdateVerify_DeepModeFailed(t *testing.T) {
	artifactID := int64(42)

	assets := &mockUpdateAssetLookup{
		GetAssetByVariantFn: func(_, _, _, _ string) (*DownloadAsset, error) {
			return &DownloadAsset{ArtifactID: &artifactID}, nil
		},
	}
	resolver := &mockVerifyArtifactResolver{
		GetArtifactFn: func(_ context.Context, _ string, _ int64) (*DownloadArtifact, error) {
			return &DownloadArtifact{
				ID:             artifactID,
				ReleaseVersion: "2.0.0",
				SHA512:         "abc123",
			}, nil
		},
		HeadArtifactFn: func(_ context.Context, _ string, _ DownloadArtifact) error {
			return fmt.Errorf("S3 not accessible")
		},
		PresignGetArtifactFn: func(_ context.Context, _ string, _ DownloadArtifact) (string, error) {
			return "", fmt.Errorf("presign failed")
		},
	}
	bundles := &mockBundleKeyProvider{key: "test_bundle"}

	handler := handleUpdateVerify(assets, resolver, bundles)
	req := httptest.NewRequest(http.MethodGet,
		"/api/v1/updates/test-app/verify?channel=stable&platform=windows&expected_version=2.0.0&deep=true", nil)
	req = mux.SetURLVars(req, map[string]string{"app_key": "test-app"})
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if resp["artifact_accessible"] != false {
		t.Errorf("expected artifact_accessible=false, got %v", resp["artifact_accessible"])
	}
	if resp["presign_valid"] != false {
		t.Errorf("expected presign_valid=false, got %v", resp["presign_valid"])
	}
}

func TestHandleUpdateVerify_MissingParams(t *testing.T) {
	assets := &mockUpdateAssetLookup{}
	resolver := &mockVerifyArtifactResolver{}
	bundles := &mockBundleKeyProvider{key: "test_bundle"}

	handler := handleUpdateVerify(assets, resolver, bundles)
	req := httptest.NewRequest(http.MethodGet,
		"/api/v1/updates/test-app/verify?channel=stable", nil)
	req = mux.SetURLVars(req, map[string]string{"app_key": "test-app"})
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing params, got %d: %s", w.Code, w.Body.String())
	}
}
