package delivery

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	internal "landing-page-business-suite-api/internal/delivery"
)

func TestRequireUpdateAPIKeyRejectsMissingAppKey(t *testing.T) {
	status := 0
	deps := updateTestDependencies(map[string]string{}, func(_ http.ResponseWriter, got int, _, _ string) { status = got })
	RequireUpdateAPIKey(deps, updateAppStub{})(func(http.ResponseWriter, *http.Request) {}).ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))
	if status != http.StatusBadRequest {
		t.Fatalf("status=%d", status)
	}
}

func TestUpdateFileRejectsMissingChannelBeforeLookup(t *testing.T) {
	status := 0
	deps := updateTestDependencies(map[string]string{"app_key": "desktop"}, func(_ http.ResponseWriter, got int, _, _ string) { status = got })
	UpdateFile(deps, updateAssetStub{}, updateArtifactStub{}).ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))
	if status != http.StatusBadRequest {
		t.Fatalf("status=%d", status)
	}
}

func TestUpdateFileRejectsHaltedChannelBeforeAssetLookup(t *testing.T) {
	status := 0
	lookups := 0
	deps := updateTestDependencies(map[string]string{"app_key": "desktop", "channel": "stable", "file": "latest.yml"}, func(_ http.ResponseWriter, got int, _, _ string) { status = got })
	assets := haltedUpdateAssetStub{halted: true, lookup: &lookups}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	UpdateFile(deps, assets, updateArtifactStub{}).ServeHTTP(httptest.NewRecorder(), req)
	if status != http.StatusGone || lookups != 0 {
		t.Fatalf("halted channel status=%d asset_lookups=%d", status, lookups)
	}
}

func TestPutUpdatePolicyRejectsInvalidIntervalBeforeStore(t *testing.T) {
	status, called := 0, false
	deps := updateTestDependencies(map[string]string{"app_key": "desktop"}, func(_ http.ResponseWriter, got int, _, _ string) { status = got })
	apps := updatePolicyStub{get: &internal.App{}, update: func(string, string, internal.UpdatePolicy) error { called = true; return nil }}
	req := httptest.NewRequest(http.MethodPut, "/", bytes.NewBufferString(`{"check_interval_hours":0,"update_mode":"optional"}`))
	PutUpdatePolicy(deps, apps).ServeHTTP(httptest.NewRecorder(), req)
	if status != http.StatusBadRequest || called {
		t.Fatalf("status=%d called=%t", status, called)
	}
}

func TestValidateDownloadRedirectRejectsUnsafeDestinations(t *testing.T) {
	for _, raw := range []string{"", "/relative", "ftp://downloads.example/app.zip", "https://user:pass@downloads.example/app.zip", "https://downloads.example/app.zip#fragment"} {
		if err := validateDownloadRedirect(raw); err == nil {
			t.Errorf("validateDownloadRedirect(%q) accepted unsafe destination", raw)
		}
	}
	if err := validateDownloadRedirect("https://downloads.example/app.zip?signature=short-lived"); err != nil {
		t.Fatalf("valid signed destination rejected: %v", err)
	}
}

func updateTestDependencies(params map[string]string, writeError func(http.ResponseWriter, int, string, string)) UpdateDependencies {
	return UpdateDependencies{
		BundleKey:  func() string { return "bundle" },
		PathParam:  func(_ *http.Request, key string) (string, bool) { value := params[key]; return value, value != "" },
		WriteError: writeError,
		WriteData:  func(http.ResponseWriter, any) {},
		DecodeJSON: func(_ http.ResponseWriter, r *http.Request, target any) bool { return jsonDecode(r, target) },
	}
}

func jsonDecode(r *http.Request, target any) bool {
	if r.Body == nil {
		return false
	}
	return json.NewDecoder(r.Body).Decode(target) == nil
}

type updateAppStub struct {
	app *internal.App
	err error
}

func (s updateAppStub) GetApp(string, string) (*internal.App, error) { return s.app, s.err }

type updateAssetStub struct{}

func (updateAssetStub) GetAssetByVariant(string, string, string, string) (*internal.Asset, error) {
	return nil, errors.New("unexpected lookup")
}

type haltedUpdateAssetStub struct {
	halted bool
	lookup *int
}

func (s haltedUpdateAssetStub) GetAssetByVariant(string, string, string, string) (*internal.Asset, error) {
	*s.lookup = *s.lookup + 1
	return nil, errors.New("asset lookup should not run while channel is halted")
}

func (s haltedUpdateAssetStub) IsChannelHalted(string, string, string) (bool, error) {
	return s.halted, nil
}

type updateArtifactStub struct{}

func (updateArtifactStub) GetArtifact(context.Context, string, int64) (*internal.Artifact, error) {
	return nil, nil
}

func (updateArtifactStub) GetCurrentArtifactByFilename(context.Context, string, string, string, string) (*internal.Artifact, error) {
	return nil, nil
}

func (updateArtifactStub) PresignGetArtifact(context.Context, string, internal.Artifact) (string, error) {
	return "", nil
}

type updatePolicyStub struct {
	get    *internal.App
	update func(string, string, internal.UpdatePolicy) error
}

func (s updatePolicyStub) GetApp(string, string) (*internal.App, error) { return s.get, nil }
func (s updatePolicyStub) UpdateAppPolicy(b, a string, p internal.UpdatePolicy) error {
	return s.update(b, a, p)
}
