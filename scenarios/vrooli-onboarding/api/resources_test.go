package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	vroolicli "github.com/vrooli/vrooli-cli-go"
)

type resourceStatusFixture struct {
	Name               string
	Installed, Running bool
	Health, Message    string
}
type resourceReadModel struct {
	Name, DisplayName, Description, Category string
	Enabled, Installed                       bool
}

var (
	testResPostgres  = map[string]string{"name": "postgres", "status": "running", "installed": "true"}
	testResRedis     = map[string]string{"name": "redis", "status": "running", "installed": "true"}
	testResOllama    = map[string]string{"name": "ollama", "status": "installed", "installed": "true"}
	testResNextcloud = map[string]string{"name": "nextcloud", "status": "running", "installed": "true"}
	testResStopped   = map[string]string{"name": "redis", "status": "stopped", "installed": "true"}
	testResMystery   = map[string]string{"name": "mystery", "status": "stopped", "installed": "false"}
)

func fixtureFromMap(raw map[string]string) resourceStatusFixture {
	fixture := resourceStatusFixture{Name: raw["name"], Installed: strings.EqualFold(raw["installed"], "true")}
	switch strings.ToLower(raw["status"]) {
	case "running":
		fixture.Running, fixture.Health, fixture.Message = true, "healthy", "healthy"
	case "installed":
		fixture.Message = "available for manual start"
	case "stopped":
		fixture.Health, fixture.Message = "stopped", "stopped"
	default:
		fixture.Message = raw["status"]
	}
	return fixture
}

type stubRunner struct {
	out []byte
	err error
}

func (s stubRunner) Run(context.Context, string, ...string) ([]byte, error) { return s.out, s.err }
func (s stubRunner) RunCombined(ctx context.Context, name string, args ...string) ([]byte, error) {
	return s.Run(ctx, name, args...)
}

type blockingRunner struct{}

func (blockingRunner) Run(ctx context.Context, _ string, _ ...string) ([]byte, error) {
	<-ctx.Done()
	return nil, ctx.Err()
}

func (r blockingRunner) RunCombined(ctx context.Context, name string, args ...string) ([]byte, error) {
	return r.Run(ctx, name, args...)
}

func swapCLIClient(t *testing.T, out []byte, err error) {
	t.Helper()
	previous := cliClient
	t.Cleanup(func() { cliClient = previous })
	cliClient = vroolicli.New(vroolicli.WithRunner(stubRunner{out: out, err: err}))
}

func stubResourceStatusJSON(t *testing.T, fixtures []resourceStatusFixture, err error) {
	t.Helper()
	if err != nil {
		swapCLIClient(t, nil, err)
		return
	}
	items := make([]map[string]any, 0, len(fixtures))
	for _, item := range fixtures {
		items = append(items, map[string]any{"resource": map[string]any{"name": item.Name}, "installed": item.Installed, "running": item.Running, "health": item.Health, "message": item.Message})
	}
	data, marshalErr := json.Marshal(map[string]any{"resources": items, "success": true})
	if marshalErr != nil {
		t.Fatal(marshalErr)
	}
	swapCLIClient(t, data, nil)
}

func writeResourcesFile(t *testing.T, _ string, raw []map[string]string) {
	t.Helper()
	fixtures := make([]resourceStatusFixture, 0, len(raw))
	for _, item := range raw {
		fixtures = append(fixtures, fixtureFromMap(item))
	}
	stubResourceStatusJSON(t, fixtures, nil)
}

func doRequest(t *testing.T, srv *Server, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	var req *http.Request
	if body != "" {
		req = httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	req.RemoteAddr = "127.0.0.1:12345"
	// NewServer's test harness uses an explicit deterministic local-session
	// proof. Production requests obtain the equivalent token through the
	// bundled desktop runtime bridge; no test relies on the host OS user.
	req.Header.Set("Authorization", "LocalSession test-personal-local-session")
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)
	return w
}

func newJSONRequest(method, path, body string) *http.Request {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	return req
}

func recordRequest(handler http.Handler, req *http.Request) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	return w
}

func doPost(t *testing.T, srv *Server, path, body string) *httptest.ResponseRecorder {
	return doRequest(t, srv, http.MethodPost, path, body)
}

func doGet(t *testing.T, srv *Server, path string) *httptest.ResponseRecorder {
	return doRequest(t, srv, http.MethodGet, path, "")
}

func requireStatus(t *testing.T, w *httptest.ResponseRecorder, want int) {
	t.Helper()
	if w.Code != want {
		t.Fatalf("status = %d, want %d; body: %s", w.Code, want, w.Body.String())
	}
}

func decodeJSON(t *testing.T, w *httptest.ResponseRecorder, dst any) {
	t.Helper()
	if err := json.Unmarshal(w.Body.Bytes(), dst); err != nil {
		t.Fatal(err)
	}
}

func newTestServer(t *testing.T, fixtures any) *Server {
	t.Helper()
	raw, ok := fixtures.([]map[string]string)
	if !ok {
		t.Fatalf("unsupported fixtures: %T", fixtures)
	}
	normalized := make([]resourceStatusFixture, 0, len(raw))
	for _, item := range raw {
		normalized = append(normalized, fixtureFromMap(item))
	}
	stubResourceStatusJSON(t, normalized, nil)
	return NewServer()
}

func TestResourceConnectHealthAndList(t *testing.T) {
	srv := newTestServer(t, []map[string]string{testResPostgres, testResStopped})
	list := doPost(t, srv, "/vrooli.vrooli_onboarding.v1.resources.ResourcesService/ListResources", `{"target":"local"}`)
	requireStatus(t, list, http.StatusOK)
	if !strings.Contains(list.Body.String(), `"resources"`) {
		t.Fatal("list response omitted resources")
	}
	health := doPost(t, srv, "/vrooli.vrooli_onboarding.v1.resources.ResourcesService/GetResourceHealth", `{"target":"local"}`)
	requireStatus(t, health, http.StatusOK)
	if !strings.Contains(health.Body.String(), `"healthyCount":1`) {
		t.Fatalf("unexpected health response: %s", health.Body.String())
	}
}

func TestResourceServiceErrorsRemainVisible(t *testing.T) {
	swapCLIClient(t, nil, errors.New("command failed"))
	response := doPost(t, NewServer(), "/vrooli.vrooli_onboarding.v1.resources.ResourcesService/ListResources", `{"target":"local"}`)
	if response.Code != http.StatusInternalServerError || !strings.Contains(response.Body.String(), "command failed") {
		t.Fatalf("unexpected error response: %d %s", response.Code, response.Body.String())
	}
}

func TestResourceStatusProbeHonorsItsBoundedTimeout(t *testing.T) {
	previous := cliClient
	t.Cleanup(func() { cliClient = previous })
	cliClient = vroolicli.New(vroolicli.WithTimeout(10*time.Millisecond), vroolicli.WithRunner(blockingRunner{}))

	started := time.Now()
	response := doPost(t, NewServer(), "/vrooli.vrooli_onboarding.v1.resources.ResourcesService/GetResourceHealth", `{"target":"local"}`)
	if elapsed := time.Since(started); elapsed > 500*time.Millisecond {
		t.Fatalf("resource probe took %s, want less than 500ms", elapsed)
	}
	if response.Code != http.StatusInternalServerError || !strings.Contains(response.Body.String(), "context deadline exceeded") {
		t.Fatalf("unexpected bounded probe response: %d %s", response.Code, response.Body.String())
	}
}
