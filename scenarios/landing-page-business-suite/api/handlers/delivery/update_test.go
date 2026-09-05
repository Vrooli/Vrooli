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
