package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"scenario-to-cloud/apierrors"
	"scenario-to-cloud/closure"
	"scenario-to-cloud/domain"
)

func fixtureClosureService(t *testing.T, repo string) *closure.Service {
	t.Helper()
	root := filepath.Join("closure", "testdata", repo)
	return &closure.Service{
		Catalog:          closure.NewLayoutCatalog(filepath.Join(root, "scenarios"), filepath.Join(root, "resources")),
		HostRequirements: closure.DeclaredHostRequirements{},
		DefaultPlatform:  closure.Platform{OS: "linux", Arch: "amd64"},
	}
}

func newClosureTestServer(t *testing.T, repo string) *Server {
	t.Helper()
	previous := closureServiceOverride
	closureServiceOverride = fixtureClosureService(t, repo)
	t.Cleanup(func() { closureServiceOverride = previous })
	srv := newTestServer()
	srv.registerClosureRoutes(srv.router.PathPrefix("/api/v1").Subrouter())
	return srv
}

func decodeClosure(t *testing.T, rec *httptest.ResponseRecorder) domain.Closure {
	t.Helper()
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	var resolved domain.Closure
	if err := json.Unmarshal(rec.Body.Bytes(), &resolved); err != nil {
		t.Fatalf("decode closure: %v", err)
	}
	return resolved
}

func decodeAPIError(t *testing.T, rec *httptest.ResponseRecorder) apierrors.Envelope {
	t.Helper()
	var envelope apierrors.Envelope
	if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode error envelope: %v (%s)", err, rec.Body.String())
	}
	return envelope
}

// TestClosureEndpointsReturnTheSameExplanation proves GET and POST forms of
// the same request yield byte-identical closures, so the closure document is
// the single explanation surface. [REQ:STC-P0-021]
func TestClosureEndpointsReturnTheSameExplanation(t *testing.T) {
	srv := newClosureTestServer(t, "basic")

	getRec := httptest.NewRecorder()
	srv.router.ServeHTTP(getRec, httptest.NewRequest(http.MethodGet, "/api/v1/closure?scenario=app&environment=production&os=linux&arch=amd64", nil))
	fromGet := decodeClosure(t, getRec)

	body, _ := json.Marshal(closure.Request{ScenarioID: "app", Environment: "production", OS: "linux", Arch: "amd64"})
	postRec := httptest.NewRecorder()
	srv.router.ServeHTTP(postRec, httptest.NewRequest(http.MethodPost, "/api/v1/closure/explain", bytes.NewReader(body)))
	fromPost := decodeClosure(t, postRec)

	if !bytes.Equal(bytes.TrimSpace(getRec.Body.Bytes()), bytes.TrimSpace(postRec.Body.Bytes())) {
		t.Fatalf("GET and POST closures differ")
	}
	if fromGet.Digest == "" || fromGet.Digest != fromPost.Digest {
		t.Fatalf("digests: %q vs %q", fromGet.Digest, fromPost.Digest)
	}
	if ok, err := closure.Verify(fromGet); err != nil || !ok {
		t.Fatalf("wire closure must verify: ok=%v err=%v", ok, err)
	}
	if len(fromGet.Components) == 0 || len(fromGet.Unsupported) != 0 {
		t.Fatalf("closure: %+v", fromGet)
	}
}

// TestClosureExplainCarriesUnsupportedAndOverrides proves the POST form
// applies overrides and reports unsupported dependencies by name.
// [REQ:STC-P0-022]
func TestClosureExplainCarriesUnsupportedAndOverrides(t *testing.T) {
	srv := newClosureTestServer(t, "basic")
	body, _ := json.Marshal(closure.Request{
		ScenarioID: "app", OS: "linux", Arch: "arm64",
		Overrides: closure.RequestOverride{AutoRestart: map[string]bool{"app": true}, DeselectOptional: []string{"optional-helper"}},
	})
	rec := httptest.NewRecorder()
	srv.router.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/v1/closure/explain", bytes.NewReader(body)))
	resolved := decodeClosure(t, rec)
	var named bool
	for _, entry := range resolved.Unsupported {
		if entry.Component == "store" && entry.ReasonCode == closure.ReasonMissingPlatformArtifact {
			named = true
		}
	}
	if !named {
		t.Fatalf("unsupported must name store: %+v", resolved.Unsupported)
	}
	for _, component := range resolved.Components {
		if component.Kind == domain.ClosureKindScenario && component.ID == "optional-helper" {
			t.Fatalf("deselected optional must be absent")
		}
		if component.Kind == domain.ClosureKindScenario && component.ID == "app" && (!component.Supervision.AutoRestart || component.Supervision.AutoRestartSource != "override") {
			t.Fatalf("auto-restart override must apply: %+v", component.Supervision)
		}
	}
}

// TestClosureEndpointErrors proves the typed error model: invalid_request
// for missing platform, closure_cycle and closure_unavailable pass through
// with their details and statuses. [REQ:STC-P0-022]
func TestClosureEndpointErrors(t *testing.T) {
	srv := newClosureTestServer(t, "basic")
	rec := httptest.NewRecorder()
	srv.router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/closure?scenario=app&os=linux", nil))
	if rec.Code != http.StatusBadRequest || decodeAPIError(t, rec).Error.Code != apierrors.CodeInvalidRequest {
		t.Fatalf("missing arch: %d %s", rec.Code, rec.Body.String())
	}

	srv = newClosureTestServer(t, "cycle")
	rec = httptest.NewRecorder()
	srv.router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/closure?scenario=alpha&os=linux&arch=amd64", nil))
	envelope := decodeAPIError(t, rec)
	if rec.Code != http.StatusUnprocessableEntity || envelope.Error.Code != closure.CodeCycle || envelope.Error.Details["path"] == nil {
		t.Fatalf("cycle: %d %s", rec.Code, rec.Body.String())
	}

	srv = newClosureTestServer(t, "missing")
	rec = httptest.NewRecorder()
	srv.router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/closure?scenario=app&os=linux&arch=amd64", nil))
	envelope = decodeAPIError(t, rec)
	if rec.Code != apierrors.StatusFor(apierrors.CodeClosureUnavailable) || envelope.Error.Code != apierrors.CodeClosureUnavailable || envelope.Error.Details["component"] != "absent-store" {
		t.Fatalf("unavailable: %d %s", rec.Code, rec.Body.String())
	}
}
