package api

import (
	"context"
	"encoding/json"
	"github.com/vrooli/vrooli/internal/peerrecord"
	component "github.com/vrooli/vrooli/packages/desktop-component"
	"github.com/vrooli/vrooli/scenarios/scenario-to-desktop/runtime/components"
	"github.com/vrooli/vrooli/scenarios/scenario-to-desktop/runtime/manifest"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

type componentRuntimeFixture struct {
	*mockRuntime
	removed int
	kept    bool
}

func (f *componentRuntimeFixture) ComponentCatalog(context.Context) ([]component.SignedEntry, error) {
	return nil, nil
}
func (f *componentRuntimeFixture) ComponentInstallations(context.Context) ([]components.Installation, error) {
	return nil, nil
}
func (f *componentRuntimeFixture) InstallComponent(context.Context, string) (components.Installation, error) {
	return components.Installation{}, nil
}
func (f *componentRuntimeFixture) RemoveComponent(_ context.Context, _ string, keep bool) error {
	f.removed++
	f.kept = keep
	return nil
}

func TestComponentRemovalRequiresAuthenticatedExplicitDataChoice(t *testing.T) {
	rt := &componentRuntimeFixture{mockRuntime: &mockRuntime{manifest: &manifest.Manifest{}}}
	s := NewServer(rt, "secret")
	mux := http.NewServeMux()
	s.RegisterHandlers(mux)
	handler := s.AuthMiddleware(mux)
	for _, tc := range []struct {
		token, body string
		status      int
	}{
		{"", `{"id":"audio-tools","keep_data":false}`, 401},
		{"secret", `{"id":"audio-tools"}`, 400},
		{"secret", `{"id":"audio-tools","keep_data":"false"}`, 400},
		{"secret", `{"id":"../audio-tools","keep_data":false}`, 400},
		{"secret", `{"id":"audio-tools","keep_data":false}`, 200},
	} {
		req := httptest.NewRequest(http.MethodDelete, "/components", strings.NewReader(tc.body))
		req.Header.Set("Authorization", "Bearer "+tc.token)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != tc.status {
			t.Fatalf("body%s status%d want%d", tc.body, rec.Code, tc.status)
		}
	}
	if rt.removed != 1 || rt.kept {
		t.Fatalf("invalid removal effects: calls%d kept%t", rt.removed, rt.kept)
	}
}

func TestBareScenarioDiscoveryNeverSelectsUsageSlice(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	rt := &mockRuntime{manifest: &manifest.Manifest{App: manifest.App{Scenario: "web-console"}, Services: []manifest.Service{{ID: "agent-manager--usage", ComponentSlice: "usage", Type: "api-binary", Ports: &manifest.ServicePorts{Requested: []manifest.PortRequest{{Name: "api"}}}}}}, ports: map[string]map[string]int{"agent-manager--usage": {"api": 12345}}}
	s := NewServer(rt, "secret")
	rec := httptest.NewRecorder()
	s.handleScenarioPort(rec, httptest.NewRequest(http.MethodGet, "/scenario-port?scenario=agent-manager&port_name=API_PORT", nil))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("bare lookup selected feature slice: HTTP%d %s", rec.Code, rec.Body.String())
	}
}

func TestNativeScenarioPortHonorsPeerPrecedenceAndConstraints(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	bundle := &manifest.Manifest{App: manifest.App{Scenario: "web-console"}, Peers: []manifest.Peer{{Scenario: "audio-tools", VersionRange: ">=1.0.0 <2.0.0", EmbeddedVersion: "1.2.0"}}, Services: []manifest.Service{{ID: "audio-tools--api", Type: "api-binary", Ports: &manifest.ServicePorts{Requested: []manifest.PortRequest{{Name: "http"}}}}}}
	rt := &mockRuntime{manifest: bundle, ports: map[string]map[string]int{"audio-tools--api": {"http": 12345}}}
	server := NewServer(rt, "secret")
	mux := http.NewServeMux()
	server.RegisterHandlers(mux)
	handler := server.AuthMiddleware(mux)
	lookup := func(token string) (int, int) {
		req := httptest.NewRequest(http.MethodGet, "/scenario-port?scenario=audio-tools&port_name=API_PORT", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		var result struct{ Port int }
		_ = json.Unmarshal(rec.Body.Bytes(), &result)
		return rec.Code, result.Port
	}
	if code, _ := lookup(""); code != 401 {
		t.Fatalf("unauthenticated request: %d", code)
	}
	if code, port := lookup("secret"); code != 200 || port != 12345 {
		t.Fatalf("embedded: %d %d", code, port)
	}
	for _, tc := range []struct {
		version string
		want    int
	}{{"1.3.0", 23456}, {"2.0.0", 12345}} {
		if err := peerrecord.Write(home, peerrecord.Record{Scenario: "audio-tools", Version: tc.version, OwnerPID: os.Getpid(), Ports: map[string]int{"api": 23456}}); err != nil {
			t.Fatal(err)
		}
		if code, port := lookup("secret"); code != 200 || port != tc.want {
			t.Fatalf("version %s: %d %d", tc.version, code, port)
		}
	}
	bundle.Peers[0].EmbeddedVersion = "2.0.0"
	if code, _ := lookup("secret"); code != 404 {
		t.Fatalf("both candidates incompatible: %d", code)
	}
}
