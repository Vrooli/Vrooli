package bundles

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"deployment-manager/codesigning"
	"deployment-manager/profiles"
)

type bundleSigningRepo struct {
	config *codesigning.SigningConfig
	err    error
}

func (r bundleSigningRepo) Get(context.Context, string) (*codesigning.SigningConfig, error) {
	return r.config, r.err
}
func (r bundleSigningRepo) Save(context.Context, string, *codesigning.SigningConfig) error {
	return nil
}
func (r bundleSigningRepo) Delete(context.Context, string) error { return nil }
func (r bundleSigningRepo) GetForPlatform(context.Context, string, string) (interface{}, error) {
	return nil, nil
}
func (r bundleSigningRepo) SaveForPlatform(context.Context, string, string, interface{}) error {
	return nil
}
func (r bundleSigningRepo) DeleteForPlatform(context.Context, string, string) error { return nil }

func TestGenerateSigningConfigValidatesAndHandlesDisabledProfiles(t *testing.T) {
	log := func(string, map[string]interface{}) {}
	noRepo := NewHandlerWithSigning(nil, nil, nil, log)
	rec := httptest.NewRecorder()
	noRepo.GenerateSigningConfig(rec, httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"profile_id":"p1"}`)))
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("missing repo status = %d", rec.Code)
	}
	for body, want := range map[string]string{"{": "400", `{}`: "400"} {
		rec = httptest.NewRecorder()
		noRepo.GenerateSigningConfig(rec, httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body)))
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("body %q status = %d, want %s", body, rec.Code, want)
		}
	}
	disabled := NewHandlerWithSigning(nil, nil, bundleSigningRepo{config: &codesigning.SigningConfig{Enabled: false}}, log)
	rec = httptest.NewRecorder()
	disabled.GenerateSigningConfig(rec, httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"profile_id":"p1"}`)))
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "disabled") {
		t.Fatalf("disabled config = %d %s", rec.Code, rec.Body.String())
	}
	failing := NewHandlerWithSigning(nil, nil, bundleSigningRepo{err: errors.New("database")}, log)
	rec = httptest.NewRecorder()
	failing.GenerateSigningConfig(rec, httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"profile_id":"p1"}`)))
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("repository error status = %d", rec.Code)
	}
}

func TestBundleSigningHelpers(t *testing.T) {
	manifest := &Manifest{}
	h := NewHandlerWithSigning(nil, nil, bundleSigningRepo{config: &codesigning.SigningConfig{Enabled: true}}, func(string, map[string]interface{}) {})
	if err := h.loadSigningConfig(context.Background(), "p1", manifest); err != nil || manifest.CodeSigning == nil {
		t.Fatalf("load signing config = %#v, %v", manifest.CodeSigning, err)
	}
	if err := h.loadSigningConfig(context.Background(), "", manifest); err != nil {
		t.Fatal(err)
	}
	if err := NewHandlerWithSigning(nil, nil, bundleSigningRepo{err: errors.New("broken")}, nil).loadSigningConfig(context.Background(), "p1", manifest); err == nil {
		t.Fatal("signing repository error returned nil")
	}
	h.applySwapsToManifest(manifest, []profiles.Swap{{From: "postgres", To: "sqlite"}})
	if len(manifest.Swaps) != 1 || manifest.Swaps[0].Replacement != "sqlite" {
		t.Fatalf("manifest swaps = %#v", manifest.Swaps)
	}
}
