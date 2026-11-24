package delivery

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	internal "landing-page-business-suite-api/internal/delivery"
)

type storefrontTestCatalog struct {
	upserts int
	lastApp internal.App
}

func (c *storefrontTestCatalog) ListApps(string) ([]internal.App, error) { return nil, nil }
func (c *storefrontTestCatalog) UpsertApp(app internal.App) (*internal.App, error) {
	c.upserts++
	c.lastApp = app
	return &app, nil
}
func (c *storefrontTestCatalog) DeleteApp(string, string) error { return nil }

func TestCreateAppRejectsUnsafeStorefrontBeforePersistence(t *testing.T) {
	catalog := &storefrontTestCatalog{}
	deps := AppDependencies{
		BundleKey: func() string { return "business_suite" },
		PathParam: func(*http.Request, string) (string, bool) { return "", false },
		DecodeJSON: func(w http.ResponseWriter, r *http.Request, target any) bool {
			if err := json.NewDecoder(r.Body).Decode(target); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return false
			}
			return true
		},
		WriteError: func(w http.ResponseWriter, status int, message, _ string) {
			http.Error(w, message, status)
		},
		WriteData:    func(w http.ResponseWriter, _ any) { w.WriteHeader(http.StatusOK) },
		WriteSuccess: func(w http.ResponseWriter) { w.WriteHeader(http.StatusOK) },
	}
	payload := AppRequest{
		AppKey: "aquila",
		Name:   "Aquila",
		Storefronts: []internal.Storefront{{
			Store: "app_store", Label: "App Store", URL: "https://apps.apple.com.evil.test/us/app/aquila/id123456",
		}},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/admin/downloads/apps", bytes.NewReader(body))
	CreateApp(deps, catalog)(recorder, request)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", recorder.Code, recorder.Body.String())
	}
	if catalog.upserts != 0 {
		t.Fatalf("invalid storefront reached UpsertApp %d times", catalog.upserts)
	}
}

func TestCreateAppPersistsOfficialStorefrontLink(t *testing.T) {
	catalog := &storefrontTestCatalog{}
	deps := AppDependencies{
		BundleKey: func() string { return "business_suite" },
		PathParam: func(*http.Request, string) (string, bool) { return "", false },
		DecodeJSON: func(w http.ResponseWriter, r *http.Request, target any) bool {
			if err := json.NewDecoder(r.Body).Decode(target); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return false
			}
			return true
		},
		WriteError: func(w http.ResponseWriter, status int, message, _ string) {
			http.Error(w, message, status)
		},
		WriteData:    func(w http.ResponseWriter, _ any) { w.WriteHeader(http.StatusOK) },
		WriteSuccess: func(w http.ResponseWriter) { w.WriteHeader(http.StatusOK) },
	}
	payload := AppRequest{
		AppKey: "aquila",
		Name:   "Aquila",
		Storefronts: []internal.Storefront{{
			Store: " play_store ", Label: " Google Play ", URL: " https://play.google.com/store/apps/details?id=org.vrooli.aquila ",
		}},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/admin/downloads/apps", bytes.NewReader(body))
	CreateApp(deps, catalog)(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", recorder.Code, recorder.Body.String())
	}
	if catalog.upserts != 1 {
		t.Fatalf("valid storefront reached UpsertApp %d times, want 1", catalog.upserts)
	}
	if len(catalog.lastApp.Storefronts) != 1 || catalog.lastApp.Storefronts[0].Store != "play_store" ||
		catalog.lastApp.Storefronts[0].Label != "Google Play" ||
		catalog.lastApp.Storefronts[0].URL != "https://play.google.com/store/apps/details?id=org.vrooli.aquila" {
		t.Fatalf("persisted storefront was not normalized: %+v", catalog.lastApp.Storefronts)
	}
}

func TestBuildAppFromPayloadRejectsNonPositiveManagedArtifactIDs(t *testing.T) {
	for _, artifactID := range []int64{0, -1} {
		t.Run(fmt.Sprintf("artifact_id_%d", artifactID), func(t *testing.T) {
			_, err := BuildAppFromPayload(AppRequest{
				AppKey: "aquila",
				Name:   "Aquila",
				Platforms: []AssetRequest{{
					Platform:       "linux",
					ArtifactSource: "managed",
					ArtifactID:     &artifactID,
					ReleaseVersion: "1.2.3",
				}},
			}, "business_suite", "")
			if err == nil {
				t.Fatalf("BuildAppFromPayload accepted managed artifact_id %d", artifactID)
			}
		})
	}
}

func TestBuildAppFromPayloadRequiresManagedChecksum(t *testing.T) {
	artifactID := int64(41)
	_, err := BuildAppFromPayload(AppRequest{
		AppKey: "aquila", Name: "Aquila",
		Platforms: []AssetRequest{{Platform: "linux", ArtifactSource: "managed", ArtifactID: &artifactID, ReleaseVersion: "1.2.3"}},
	}, "business_suite", "")
	if err == nil || !strings.Contains(err.Error(), "checksum is required for managed platform") {
		t.Fatalf("BuildAppFromPayload error = %v, want managed checksum validation", err)
	}
}

func TestCreateAppBindsManagedAssetToArtifactRecordBeforePersistence(t *testing.T) {
	artifactID := int64(91)
	checksum := strings.Repeat("a", 64)
	catalog := &storefrontTestCatalog{}
	deps := AppDependencies{
		BundleKey: func() string { return "business_suite" },
		GetArtifact: func(_ context.Context, bundle string, id int64) (*internal.Artifact, error) {
			if bundle != "business_suite" || id != artifactID {
				t.Fatalf("artifact lookup bundle=%q id=%d", bundle, id)
			}
			return &internal.Artifact{BundleKey: bundle, AppKey: "aquila", Platform: "linux", ReleaseVersion: "1.2.3", SHA256: checksum}, nil
		},
		PathParam: func(*http.Request, string) (string, bool) { return "", false },
		DecodeJSON: func(w http.ResponseWriter, r *http.Request, target any) bool {
			return json.NewDecoder(r.Body).Decode(target) == nil
		},
		WriteError:   func(w http.ResponseWriter, status int, message, _ string) { http.Error(w, message, status) },
		WriteData:    func(w http.ResponseWriter, _ any) { w.WriteHeader(http.StatusOK) },
		WriteSuccess: func(w http.ResponseWriter) { w.WriteHeader(http.StatusOK) },
	}
	payload := AppRequest{AppKey: "aquila", Name: "Aquila", Platforms: []AssetRequest{{Platform: "linux", ArtifactSource: "managed", ArtifactID: &artifactID, ReleaseVersion: "1.2.3", Checksum: checksum}}}
	body, _ := json.Marshal(payload)
	recorder := httptest.NewRecorder()
	CreateApp(deps, catalog)(recorder, httptest.NewRequest(http.MethodPost, "/admin/downloads/apps", bytes.NewReader(body)))
	if recorder.Code != http.StatusOK || catalog.upserts != 1 {
		t.Fatalf("status=%d upserts=%d body=%s", recorder.Code, catalog.upserts, recorder.Body.String())
	}
}

func TestCreateAppRejectsManagedArtifactChecksumMismatchBeforePersistence(t *testing.T) {
	artifactID := int64(91)
	catalog := &storefrontTestCatalog{}
	deps := AppDependencies{
		BundleKey: func() string { return "business_suite" },
		GetArtifact: func(context.Context, string, int64) (*internal.Artifact, error) {
			return &internal.Artifact{BundleKey: "business_suite", AppKey: "aquila", Platform: "linux", ReleaseVersion: "1.2.3", SHA256: strings.Repeat("a", 64)}, nil
		},
		PathParam: func(*http.Request, string) (string, bool) { return "", false },
		DecodeJSON: func(w http.ResponseWriter, r *http.Request, target any) bool {
			return json.NewDecoder(r.Body).Decode(target) == nil
		},
		WriteError:   func(w http.ResponseWriter, status int, message, _ string) { http.Error(w, message, status) },
		WriteData:    func(w http.ResponseWriter, _ any) { w.WriteHeader(http.StatusOK) },
		WriteSuccess: func(w http.ResponseWriter) { w.WriteHeader(http.StatusOK) },
	}
	payload := AppRequest{AppKey: "aquila", Name: "Aquila", Platforms: []AssetRequest{{Platform: "linux", ArtifactSource: "managed", ArtifactID: &artifactID, ReleaseVersion: "1.2.3", Checksum: strings.Repeat("b", 64)}}}
	body, _ := json.Marshal(payload)
	recorder := httptest.NewRecorder()
	CreateApp(deps, catalog)(recorder, httptest.NewRequest(http.MethodPost, "/admin/downloads/apps", bytes.NewReader(body)))
	if recorder.Code != http.StatusConflict || catalog.upserts != 0 {
		t.Fatalf("status=%d upserts=%d body=%s", recorder.Code, catalog.upserts, recorder.Body.String())
	}
}

func TestCreateAppFailsClosedWhenManagedArtifactReaderIsNotConfigured(t *testing.T) {
	artifactID := int64(91)
	catalog := &storefrontTestCatalog{}
	deps := AppDependencies{
		BundleKey: func() string { return "business_suite" },
		PathParam: func(*http.Request, string) (string, bool) { return "", false },
		DecodeJSON: func(w http.ResponseWriter, r *http.Request, target any) bool {
			return json.NewDecoder(r.Body).Decode(target) == nil
		},
		WriteError:   func(w http.ResponseWriter, status int, message, _ string) { http.Error(w, message, status) },
		WriteData:    func(w http.ResponseWriter, _ any) { w.WriteHeader(http.StatusOK) },
		WriteSuccess: func(w http.ResponseWriter) { w.WriteHeader(http.StatusOK) },
	}
	payload := AppRequest{AppKey: "aquila", Name: "Aquila", Platforms: []AssetRequest{{Platform: "linux", ArtifactSource: "managed", ArtifactID: &artifactID, ReleaseVersion: "1.2.3", Checksum: strings.Repeat("a", 64)}}}
	body, _ := json.Marshal(payload)
	recorder := httptest.NewRecorder()
	CreateApp(deps, catalog)(recorder, httptest.NewRequest(http.MethodPost, "/admin/downloads/apps", bytes.NewReader(body)))
	if recorder.Code != http.StatusServiceUnavailable || catalog.upserts != 0 {
		t.Fatalf("status=%d upserts=%d body=%s", recorder.Code, catalog.upserts, recorder.Body.String())
	}
}
