package main

import (
	"context"
	"crypto/sha512"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
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

func TestHandleUpdateFileMock(t *testing.T) {
	const artifactID = int64(42)
	artifact := &DownloadArtifact{
		ID: artifactID, ReleaseVersion: "2.0.0", OriginalFilename: "my-app-2.0.0.exe",
		SHA512: "fakeSHA512base64==", SizeBytes: 50000000,
		UpdatedAt: time.Date(2026, 2, 5, 14, 0, 0, 0, time.UTC),
		Bucket:    "test-bucket", ObjectKey: "artifacts/my-app-2.0.0.exe",
	}

	cases := []struct {
		name                string
		file                string
		artifactByID        bool
		currentArtifact     bool
		assetArtifactID     *int64
		releaseNotes        string
		missingSHA512       bool
		presignError        bool
		wantStatus          int
		wantContentType     string
		wantCacheControl    string
		wantLocation        string
		wantBodyFragments   []string
		wantAssetCalls      int
		wantArtifactIDCalls int
		wantCurrentCalls    int
		wantPresignCalls    int
	}{
		{
			name: "manifest served", file: "latest.yml", artifactByID: true, assetArtifactID: int64Pointer(artifactID),
			wantStatus: http.StatusOK, wantContentType: "application/x-yaml", wantCacheControl: "no-cache",
			wantBodyFragments: []string{"version: 2.0.0", "sha512: fakeSHA512base64==", "size: 50000000"},
			wantAssetCalls:    1, wantArtifactIDCalls: 1,
		},
		{
			name: "manifest includes release notes", file: "latest.yml", artifactByID: true,
			assetArtifactID: int64Pointer(artifactID), releaseNotes: "Bug fixes and improvements",
			wantStatus: http.StatusOK, wantContentType: "application/x-yaml", wantCacheControl: "no-cache",
			wantBodyFragments: []string{"releaseNotes: Bug fixes and improvements"},
			wantAssetCalls:    1, wantArtifactIDCalls: 1,
		},
		{
			name: "binary redirects to signed URL", file: "my-app-2.0.0.exe", currentArtifact: true,
			wantStatus: http.StatusFound, wantCacheControl: "no-store", wantLocation: "https://s3.example.com/presigned-url",
			wantCurrentCalls: 1, wantPresignCalls: 1,
		},
		{
			name: "manifest rejects artifact without digest", file: "latest.yml", artifactByID: true,
			assetArtifactID: int64Pointer(artifactID), missingSHA512: true,
			wantStatus: http.StatusNotFound, wantBodyFragments: []string{"sha512"},
			wantAssetCalls: 1, wantArtifactIDCalls: 1,
		},
		{
			name: "manifest rejects missing artifact link", file: "latest-mac.yml", assetArtifactID: nil,
			wantStatus: http.StatusNotFound, wantBodyFragments: []string{"no managed artifact linked"}, wantAssetCalls: 1,
		},
		{
			name: "binary reports presign failure", file: "my-app-2.0.0.exe", currentArtifact: true, presignError: true,
			wantStatus: http.StatusInternalServerError, wantCurrentCalls: 1, wantPresignCalls: 1,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			assetCalls, artifactIDCalls, currentCalls, presignCalls := 0, 0, 0, 0
			assets := &mockUpdateAssetLookup{
				GetAssetByVariantFn: func(_, _, _, _ string) (*DownloadAsset, error) {
					assetCalls++
					return &DownloadAsset{ArtifactID: tc.assetArtifactID, ReleaseNotes: tc.releaseNotes}, nil
				},
			}
			resolvedArtifact := artifact
			if tc.missingSHA512 {
				copy := *artifact
				copy.SHA512 = ""
				resolvedArtifact = &copy
			}
			resolver := &mockUpdateArtifactResolver{
				GetArtifactFn: func(_ context.Context, _ string, _ int64) (*DownloadArtifact, error) {
					artifactIDCalls++
					return resolvedArtifact, nil
				},
				GetCurrentArtifactByFilenameFn: func(_ context.Context, _, _, _, _ string) (*DownloadArtifact, error) {
					currentCalls++
					return artifact, nil
				},
				PresignGetArtifactFn: func(_ context.Context, _ string, _ DownloadArtifact) (string, error) {
					presignCalls++
					if tc.presignError {
						return "", fmt.Errorf("storage unavailable")
					}
					return "https://s3.example.com/presigned-url", nil
				},
			}
			if !tc.artifactByID {
				resolver.GetArtifactFn = func(context.Context, string, int64) (*DownloadArtifact, error) {
					artifactIDCalls++
					return nil, fmt.Errorf("unexpected artifact-by-id lookup")
				}
			}
			if !tc.currentArtifact {
				resolver.GetCurrentArtifactByFilenameFn = func(context.Context, string, string, string, string) (*DownloadArtifact, error) {
					currentCalls++
					return nil, fmt.Errorf("unexpected current-artifact lookup")
				}
			}

			handler := handleUpdateFile(assets, resolver, &mockBundleKeyProvider{key: "test_bundle"})
			req := newMockUpdateRequest("test-app", "stable", tc.file)
			w := httptest.NewRecorder()
			handler(w, req)

			if w.Code != tc.wantStatus {
				t.Fatalf("status = %d, body = %s; want %d", w.Code, w.Body.String(), tc.wantStatus)
			}
			if tc.wantContentType != "" {
				if got := w.Header().Get("Content-Type"); got != tc.wantContentType {
					t.Errorf("Content-Type = %q, want %q", got, tc.wantContentType)
				}
			}
			if got := w.Header().Get("Cache-Control"); got != tc.wantCacheControl {
				t.Errorf("Cache-Control = %q, want %q", got, tc.wantCacheControl)
			}
			if got := w.Header().Get("Location"); got != tc.wantLocation {
				t.Errorf("Location = %q, want %q", got, tc.wantLocation)
			}
			for _, want := range tc.wantBodyFragments {
				if !strings.Contains(w.Body.String(), want) {
					t.Errorf("response body omits %q: %s", want, w.Body.String())
				}
			}
			if assetCalls != tc.wantAssetCalls || artifactIDCalls != tc.wantArtifactIDCalls ||
				currentCalls != tc.wantCurrentCalls || presignCalls != tc.wantPresignCalls {
				t.Errorf("asset/id/current/presign calls = %d/%d/%d/%d, want %d/%d/%d/%d",
					assetCalls, artifactIDCalls, currentCalls, presignCalls,
					tc.wantAssetCalls, tc.wantArtifactIDCalls, tc.wantCurrentCalls, tc.wantPresignCalls)
			}
		})
	}
}

func int64Pointer(value int64) *int64 { return &value }

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
	ReadArtifactFn       func(ctx context.Context, bundleKey string, artifact DownloadArtifact) (io.ReadCloser, int64, string, error)
}

type observableReadCloser struct {
	io.Reader
	closeCalls *int
}

func (r observableReadCloser) Close() error {
	*r.closeCalls++
	return nil
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

func (m *mockVerifyArtifactResolver) ReadArtifact(ctx context.Context, bundleKey string, artifact DownloadArtifact) (io.ReadCloser, int64, string, error) {
	return m.ReadArtifactFn(ctx, bundleKey, artifact)
}

func TestHandleUpdateVerify(t *testing.T) {
	digest := func(value []byte) string {
		sum := sha512.Sum512(value)
		return base64.StdEncoding.EncodeToString(sum[:])
	}
	expectedBytes := []byte("verified installer bytes")
	changedBytes := []byte("modified installer bytes")

	cases := []struct {
		name               string
		deep               bool
		expectedVersion    string
		artifactVersion    string
		expectedSHA512     string
		artifactSHA512     string
		actualBytes        []byte
		declaredSize       int64
		headError          bool
		presignError       bool
		readError          bool
		wantMatch          bool
		wantMetadataSHA512 bool
		wantAccessible     bool
		wantPresignValid   bool
		wantBytesVerified  bool
		wantActualSize     int
		wantActualSHA512   string
		wantHeadCalls      int
		wantPresignCalls   int
		wantReadCalls      int
		wantBodyCloseCalls int
	}{
		{
			name: "lightweight match", expectedVersion: "2.0.0", artifactVersion: "2.0.0",
			expectedSHA512: "abc123", artifactSHA512: "abc123", wantMatch: true, wantMetadataSHA512: true,
		},
		{
			name: "lightweight version mismatch", expectedVersion: "2.0.0", artifactVersion: "1.9.0",
			expectedSHA512: "abc123", artifactSHA512: "abc123", wantMetadataSHA512: true,
		},
		{
			name: "deep artifact match", deep: true, expectedVersion: "2.0.0", artifactVersion: "2.0.0",
			expectedSHA512: digest(expectedBytes), artifactSHA512: digest(expectedBytes), actualBytes: expectedBytes, declaredSize: int64(len(expectedBytes)),
			wantMatch: true, wantMetadataSHA512: true, wantAccessible: true, wantPresignValid: true,
			wantBytesVerified: true, wantActualSize: len(expectedBytes), wantActualSHA512: digest(expectedBytes),
			wantHeadCalls: 1, wantPresignCalls: 1, wantReadCalls: 1,
		},
		{
			name: "deep storage checks fail", deep: true, expectedVersion: "2.0.0", artifactVersion: "2.0.0",
			expectedSHA512: "abc123", artifactSHA512: "abc123", headError: true, presignError: true,
			wantAccessible: false, wantPresignValid: false, wantHeadCalls: 1, wantPresignCalls: 1,
		},
		{
			name: "deep downloaded bytes differ", deep: true, expectedVersion: "2.0.0", artifactVersion: "2.0.0",
			expectedSHA512: digest(expectedBytes), artifactSHA512: digest(expectedBytes), actualBytes: changedBytes, declaredSize: int64(len(expectedBytes)),
			wantAccessible: true, wantPresignValid: true, wantActualSize: len(changedBytes),
			wantActualSHA512: digest(changedBytes), wantHeadCalls: 1, wantPresignCalls: 1, wantReadCalls: 1,
		},
		{
			name: "read_error_closes_returned_body", deep: true, expectedVersion: "2.0.0", artifactVersion: "2.0.0",
			expectedSHA512: digest(expectedBytes), artifactSHA512: digest(expectedBytes), actualBytes: expectedBytes, declaredSize: int64(len(expectedBytes)),
			readError: true, wantAccessible: true, wantPresignValid: true,
			wantHeadCalls: 1, wantPresignCalls: 1, wantReadCalls: 1, wantBodyCloseCalls: 1,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			artifactID := int64(42)
			headCalls, presignCalls, readCalls := 0, 0, 0
			bodyCloseCalls := 0
			assets := &mockUpdateAssetLookup{
				GetAssetByVariantFn: func(_, _, _, _ string) (*DownloadAsset, error) {
					return &DownloadAsset{ArtifactID: &artifactID}, nil
				},
			}
			artifact := &DownloadArtifact{
				ID: artifactID, ReleaseVersion: tc.artifactVersion, SHA512: tc.artifactSHA512,
				Bucket: "test-bucket", ObjectKey: "artifacts/test.exe",
				SizeBytes: tc.declaredSize,
			}
			resolver := &mockVerifyArtifactResolver{
				GetArtifactFn: func(_ context.Context, _ string, _ int64) (*DownloadArtifact, error) { return artifact, nil },
				HeadArtifactFn: func(_ context.Context, _ string, _ DownloadArtifact) error {
					headCalls++
					if tc.headError {
						return fmt.Errorf("storage unavailable")
					}
					return nil
				},
				PresignGetArtifactFn: func(_ context.Context, _ string, _ DownloadArtifact) (string, error) {
					presignCalls++
					if tc.presignError {
						return "", fmt.Errorf("presign unavailable")
					}
					return "https://s3.example.com/presigned", nil
				},
				ReadArtifactFn: func(_ context.Context, _ string, _ DownloadArtifact) (io.ReadCloser, int64, string, error) {
					readCalls++
					if tc.readError {
						return observableReadCloser{Reader: strings.NewReader(string(tc.actualBytes)), closeCalls: &bodyCloseCalls}, int64(len(tc.actualBytes)), "application/octet-stream", fmt.Errorf("read unavailable")
					}
					return io.NopCloser(strings.NewReader(string(tc.actualBytes))), int64(len(tc.actualBytes)), "application/octet-stream", nil
				},
			}

			query := url.Values{
				"channel": {"stable"}, "platform": {"windows"},
				"expected_version": {tc.expectedVersion}, "expected_sha512": {tc.expectedSHA512},
			}
			if tc.deep {
				query.Set("deep", "true")
			}
			req := httptest.NewRequest(http.MethodGet, "/api/v1/updates/test-app/verify?"+query.Encode(), nil)
			req = mux.SetURLVars(req, map[string]string{"app_key": "test-app"})
			w := httptest.NewRecorder()
			handleUpdateVerify(assets, resolver, &mockBundleKeyProvider{key: "test_bundle"})(w, req)

			if w.Code != http.StatusOK {
				t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
			}
			var got map[string]any
			if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			want := map[string]any{
				"app_key": "test-app", "channel": "stable", "platform": "windows",
				"expected_version": tc.expectedVersion, "actual_version": tc.artifactVersion,
				"actual_sha512": tc.artifactSHA512, "sha512_match": tc.wantMetadataSHA512,
				"match": tc.wantMatch,
			}
			if tc.deep {
				want["artifact_accessible"] = tc.wantAccessible
				want["presign_valid"] = tc.wantPresignValid
				want["bytes_verified"] = tc.wantBytesVerified
				want["actual_size_bytes"] = float64(tc.wantActualSize)
				want["actual_sha512"] = tc.wantActualSHA512
				want["sha512_match"] = tc.wantBytesVerified
			}
			for key, expected := range want {
				if actual, ok := got[key]; !ok || actual != expected {
					t.Errorf("%s = %#v (present %v), want %#v", key, actual, ok, expected)
				}
			}
			if !tc.deep {
				for _, key := range []string{"artifact_accessible", "presign_valid", "bytes_verified", "actual_size_bytes"} {
					if _, exists := got[key]; exists {
						t.Errorf("lightweight response unexpectedly contains %q: %#v", key, got[key])
					}
				}
			}
			if headCalls != tc.wantHeadCalls || presignCalls != tc.wantPresignCalls || readCalls != tc.wantReadCalls {
				t.Errorf("HEAD/presign/read calls = %d/%d/%d, want %d/%d/%d", headCalls, presignCalls, readCalls, tc.wantHeadCalls, tc.wantPresignCalls, tc.wantReadCalls)
			}
			if bodyCloseCalls != tc.wantBodyCloseCalls {
				t.Errorf("returned read body close calls = %d, want %d", bodyCloseCalls, tc.wantBodyCloseCalls)
			}
		})
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
