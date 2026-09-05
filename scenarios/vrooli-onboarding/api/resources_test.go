package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	vroolicli "github.com/vrooli/vrooli-cli-go"
)

// TestCategorize verifies category lookup for known and unknown resources.
// [REQ:REQ-P0-001] - Resource Discovery API
func TestCategorize(t *testing.T) {
	tests := []struct {
		name     string
		resource string
		want     string
	}{
		{"postgres uses its declared category", "postgres", "storage"},
		{"ollama is ai", "ollama", "ai"},
		{"redis uses its declared category", "redis", "storage"},
		{"minio is storage", "minio", "storage"},
		{"unknown falls back to general", "some-unknown-resource", "general"},
		{"empty string falls back to general", "", "general"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := categorize(tc.resource)
			if got != tc.want {
				t.Errorf("categorize(%q) = %q, want %q", tc.resource, got, tc.want)
			}
		})
	}
}

type resourceStatusFixture struct {
	Name      string
	Installed bool
	Running   bool
	Health    string
	Message   string
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
	fixture := resourceStatusFixture{
		Name:      raw["name"],
		Installed: strings.EqualFold(raw["installed"], "true"),
	}
	switch strings.ToLower(raw["status"]) {
	case "running":
		fixture.Running = true
		fixture.Health = "healthy"
		fixture.Message = "healthy"
	case "installed":
		fixture.Message = "available for manual start"
	case "stopped":
		fixture.Message = "stopped"
		fixture.Health = "stopped"
	default:
		fixture.Message = raw["status"]
	}
	return fixture
}

func writeResourcesFile(t *testing.T, _ string, resources []map[string]string) {
	t.Helper()
	fixtures := make([]resourceStatusFixture, 0, len(resources))
	for _, item := range resources {
		fixtures = append(fixtures, fixtureFromMap(item))
	}
	stubResourceStatusJSON(t, fixtures, nil)
}

// stubRunner is a vroolicli.Runner that returns canned output, letting tests
// drive loadResources without executing the real CLI.
type stubRunner struct {
	out []byte
	err error
}

func (s stubRunner) Run(context.Context, string, ...string) ([]byte, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.out, nil
}

func (s stubRunner) RunCombined(ctx context.Context, name string, args ...string) ([]byte, error) {
	return s.Run(ctx, name, args...)
}

// swapCLIClient points the package-level cliClient at a stub runner for the
// duration of the test.
func swapCLIClient(t *testing.T, out []byte, err error) {
	t.Helper()
	previous := cliClient
	t.Cleanup(func() {
		cliClient = previous
	})
	cliClient = vroolicli.New(vroolicli.WithRunner(stubRunner{out: out, err: err}))
}

func stubResourceStatusJSON(t *testing.T, fixtures []resourceStatusFixture, err error) {
	t.Helper()

	if err != nil {
		swapCLIClient(t, nil, err)
		return
	}
	payload := map[string]any{
		"resources": fixturesToCLI(fixtures),
		"success":   true,
	}
	data, marshalErr := json.Marshal(payload)
	if marshalErr != nil {
		t.Fatalf("marshal fixtures: %v", marshalErr)
	}
	swapCLIClient(t, data, nil)
}

func stubRawResourceStatusJSON(t *testing.T, raw string, err error) {
	t.Helper()
	if err != nil {
		swapCLIClient(t, nil, err)
		return
	}
	swapCLIClient(t, []byte(raw), nil)
}

func fixturesToCLI(fixtures []resourceStatusFixture) []map[string]any {
	items := make([]map[string]any, 0, len(fixtures))
	for _, fixture := range fixtures {
		items = append(items, map[string]any{
			"resource": map[string]any{
				"name": fixture.Name,
			},
			"installed": fixture.Installed,
			"running":   fixture.Running,
			"health":    fixture.Health,
			"message":   fixture.Message,
		})
	}
	return items
}

// doRequest performs an HTTP request against the test server and returns the recorder.
func doRequest(t *testing.T, srv *Server, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	var req *http.Request
	if body != "" {
		req = httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	// Test requests represent the local browser/CLI caller. Remote-boundary
	// tests construct an explicit non-loopback RemoteAddr instead.
	req.RemoteAddr = "127.0.0.1:12345"
	w := httptest.NewRecorder()
	srv.router.ServeHTTP(w, req)
	return w
}

func doGet(t *testing.T, srv *Server, path string) *httptest.ResponseRecorder {
	t.Helper()
	return doRequest(t, srv, http.MethodGet, path, "")
}

func doPost(t *testing.T, srv *Server, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	return doRequest(t, srv, http.MethodPost, path, body)
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
		t.Fatalf("failed to decode response: %v", err)
	}
}

func newTestServer(t *testing.T, fixtures any) *Server {
	t.Helper()
	var normalized []resourceStatusFixture
	switch typed := fixtures.(type) {
	case []resourceStatusFixture:
		normalized = typed
	case []map[string]string:
		normalized = make([]resourceStatusFixture, 0, len(typed))
		for _, item := range typed {
			normalized = append(normalized, fixtureFromMap(item))
		}
	default:
		t.Fatalf("unsupported fixture type %T", fixtures)
	}

	stubResourceStatusJSON(t, normalized, nil)
	return NewServer()
}

// TestLoadResources verifies loading from the Vrooli CLI JSON output.
// [REQ:REQ-P0-001] - Resource Discovery API
func TestLoadResources(t *testing.T) {
	stubResourceStatusJSON(t, []resourceStatusFixture{
		{Name: "postgres", Installed: true, Running: true, Health: "healthy", Message: "healthy"},
		{Name: "ollama", Installed: true, Running: false, Message: "available for manual start"},
		{Name: "mystery", Installed: false, Running: false, Message: "not installed"},
	}, nil)

	resources, err := loadResources()
	if err != nil {
		t.Fatalf("loadResources() error: %v", err)
	}

	if len(resources) != 3 {
		t.Fatalf("expected 3 resources, got %d", len(resources))
	}

	if resources[0].Name != "postgres" {
		t.Errorf("resources[0].Name = %q, want %q", resources[0].Name, "postgres")
	}
	if resources[0].Category != "storage" {
		t.Errorf("resources[0].Category = %q, want %q", resources[0].Category, "storage")
	}
	if resources[0].Status != "running" {
		t.Errorf("resources[0].Status = %q, want %q", resources[0].Status, "running")
	}
	if !resources[0].Installed {
		t.Error("resources[0].Installed = false, want true")
	}

	if resources[1].Category != "ai" {
		t.Errorf("resources[1].Category = %q, want %q", resources[1].Category, "ai")
	}
	if resources[1].Status != "installed" {
		t.Errorf("resources[1].Status = %q, want %q", resources[1].Status, "installed")
	}

	if resources[2].Status != "stopped" {
		t.Errorf("resources[2].Status = %q, want %q", resources[2].Status, "stopped")
	}
	if resources[2].Category != "general" {
		t.Errorf("resources[2].Category = %q, want %q", resources[2].Category, "general")
	}
}

// TestLoadResourcesCommandFailure verifies error propagation when CLI execution fails.
// [REQ:REQ-P0-001] - Resource Discovery API
func TestLoadResourcesCommandFailure(t *testing.T) {
	stubResourceStatusJSON(t, nil, errors.New("command failed"))

	_, err := loadResources()
	if err == nil {
		t.Fatal("expected error when CLI command fails, got nil")
	}
}

// TestHandleListResources verifies the GET /api/v1/resources endpoint.
// [REQ:REQ-P0-001] - Resource Discovery API
func TestHandleListResources(t *testing.T) {
	srv := newTestServer(t, []resourceStatusFixture{
		{Name: "postgres", Installed: true, Running: true, Health: "healthy"},
		{Name: "redis", Installed: true, Running: false, Health: "stopped", Message: "stopped"},
	})

	w := doGet(t, srv, "/api/v1/resources")
	requireStatus(t, w, http.StatusOK)

	var body map[string]any
	decodeJSON(t, w, &body)

	resources, ok := body["resources"].([]any)
	if !ok {
		t.Fatal("response missing 'resources' array")
	}
	if len(resources) != 2 {
		t.Errorf("expected 2 resources, got %d", len(resources))
	}

	count, ok := body["count"].(float64)
	if !ok {
		t.Fatal("response missing 'count'")
	}
	if int(count) != 2 {
		t.Errorf("count = %v, want 2", count)
	}

	if _, ok := body["loaded_at"]; !ok {
		t.Error("response missing 'loaded_at'")
	}
}

// TestHandleGetResource verifies GET /api/v1/resources/{name} for an existing resource.
// [REQ:REQ-P0-002] - Resource Detail View
func TestHandleGetResource(t *testing.T) {
	srv := newTestServer(t, []resourceStatusFixture{
		{Name: "postgres", Installed: true, Running: true, Health: "healthy"},
		{Name: "ollama", Installed: true, Running: false, Message: "available for manual start"},
	})

	w := doGet(t, srv, "/api/v1/resources/postgres")
	requireStatus(t, w, http.StatusOK)

	var res Resource
	decodeJSON(t, w, &res)

	if res.Name != "postgres" {
		t.Errorf("Name = %q, want %q", res.Name, "postgres")
	}
	if res.Category != "storage" {
		t.Errorf("Category = %q, want %q", res.Category, "storage")
	}
}

// TestHandleGetResourceNotFound verifies 404 for unknown resource names.
// [REQ:REQ-P0-002] - Resource Detail View
func TestHandleGetResourceNotFound(t *testing.T) {
	srv := newTestServer(t, []resourceStatusFixture{
		{Name: "postgres", Installed: true, Running: true, Health: "healthy"},
	})

	w := doGet(t, srv, "/api/v1/resources/nonexistent")
	requireStatus(t, w, http.StatusNotFound)

	var body map[string]string
	decodeJSON(t, w, &body)
	if body["error"] == "" {
		t.Error("expected error message in response")
	}
}

// TestHandleGetResourceCaseInsensitive verifies case-insensitive matching.
// [REQ:REQ-P0-002] - Resource Detail View
func TestHandleGetResourceCaseInsensitive(t *testing.T) {
	srv := newTestServer(t, []resourceStatusFixture{
		{Name: "postgres", Installed: true, Running: true, Health: "healthy"},
	})

	w := doGet(t, srv, "/api/v1/resources/Postgres")
	requireStatus(t, w, http.StatusOK)
}

// TestHandleListResourcesLoadError verifies 500 when CLI status fails.
// [REQ:REQ-P0-001] - Resource Discovery API
func TestHandleListResourcesLoadError(t *testing.T) {
	stubResourceStatusJSON(t, nil, errors.New("command failed"))
	srv := NewServer()

	w := doGet(t, srv, "/api/v1/resources")
	requireStatus(t, w, http.StatusInternalServerError)

	var body map[string]string
	decodeJSON(t, w, &body)
	if !strings.Contains(body["error"], "failed to load resources") {
		t.Errorf("expected resource load error, got %q", body["error"])
	}
}

// TestHandleGetResourceLoadError verifies 500 when CLI status fails.
// [REQ:REQ-P0-002] - Resource Detail View
func TestHandleGetResourceLoadError(t *testing.T) {
	stubResourceStatusJSON(t, nil, errors.New("command failed"))
	srv := NewServer()

	w := doGet(t, srv, "/api/v1/resources/postgres")
	requireStatus(t, w, http.StatusInternalServerError)
}

// TestLoadResourcesInvalidJSON verifies error for malformed CLI output.
// [REQ:REQ-P0-001] - Resource Discovery API
func TestLoadResourcesInvalidJSON(t *testing.T) {
	stubRawResourceStatusJSON(t, "{bad json", nil)

	_, err := loadResources()
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

// TestLoadResourcesEmptyList verifies loading an empty resources list.
// [REQ:REQ-P0-001] - Resource Discovery API
func TestLoadResourcesEmptyList(t *testing.T) {
	stubResourceStatusJSON(t, []resourceStatusFixture{}, nil)

	resources, err := loadResources()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resources) != 0 {
		t.Errorf("expected 0 resources, got %d", len(resources))
	}
}

// TestHandleListResourcesEmpty verifies GET /api/v1/resources with no resources.
// [REQ:REQ-P0-001] - Resource Discovery API
func TestHandleListResourcesEmpty(t *testing.T) {
	srv := newTestServer(t, []resourceStatusFixture{})

	w := doGet(t, srv, "/api/v1/resources")
	requireStatus(t, w, http.StatusOK)

	var body map[string]any
	decodeJSON(t, w, &body)

	count := body["count"].(float64)
	if int(count) != 0 {
		t.Errorf("expected count=0, got %v", count)
	}
}

// TestHandleListResourcesResponseHasLoadedAt verifies loaded_at timestamp format.
// [REQ:REQ-P0-001] - Resource Discovery API
func TestHandleListResourcesResponseHasLoadedAt(t *testing.T) {
	srv := newTestServer(t, []resourceStatusFixture{
		{Name: "postgres", Installed: true, Running: true, Health: "healthy"},
	})

	w := doGet(t, srv, "/api/v1/resources")
	requireStatus(t, w, http.StatusOK)

	var body map[string]any
	decodeJSON(t, w, &body)

	loadedAt, ok := body["loaded_at"].(string)
	if !ok || loadedAt == "" {
		t.Error("expected non-empty loaded_at timestamp string")
	}
}

// TestCategorizeSpecificResources verifies categorization for more resource types.
// [REQ:REQ-P0-001] - Resource Discovery API
func TestCategorizeSpecificResources(t *testing.T) {
	tests := []struct {
		resource string
		want     string
	}{
		{"vault", "storage"},
		{"qdrant", "storage"},
		{"nextcloud", "general"},
		{"n8n", "general"},
	}

	for _, tc := range tests {
		got := categorize(tc.resource)
		if got != tc.want {
			t.Errorf("categorize(%q) = %q, want %q", tc.resource, got, tc.want)
		}
	}
}

// TestLoadResourcesStatusNormalization verifies status derivation from CLI state.
// [REQ:REQ-P0-001] - Resource Discovery API
func TestLoadResourcesStatusNormalization(t *testing.T) {
	stubResourceStatusJSON(t, []resourceStatusFixture{
		{Name: "a", Installed: true, Running: true, Health: "healthy"},
		{Name: "b", Installed: true, Running: false, Message: "available for manual start"},
		{Name: "c", Installed: true, Running: false, Health: "stopped"},
		{Name: "d", Installed: true, Running: true, Message: "available"},
		{Name: "e", Installed: false, Running: false, Message: "not installed"},
		{Name: "f", Installed: false, Running: false, Message: ""},
	}, nil)

	resources, err := loadResources()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []string{"running", "installed", "stopped", "running", "stopped", "stopped"}
	for i, want := range expected {
		if resources[i].Status != want {
			t.Errorf("resources[%d].Status = %q, want %q (input was %q)",
				i, resources[i].Status, want, resources[i].Name)
		}
	}
}

// TestHandleGetResourceAllFields verifies the full response structure for a single resource.
// [REQ:REQ-P0-002] - Resource Detail View
func TestHandleGetResourceAllFields(t *testing.T) {
	srv := newTestServer(t, []resourceStatusFixture{
		{Name: "postgres", Installed: true, Running: true, Health: "healthy"},
	})

	w := doGet(t, srv, "/api/v1/resources/postgres")
	requireStatus(t, w, http.StatusOK)

	var res Resource
	decodeJSON(t, w, &res)

	if res.Name != "postgres" {
		t.Errorf("Name = %q, want %q", res.Name, "postgres")
	}
	if res.Category != "storage" {
		t.Errorf("Category = %q, want %q", res.Category, "storage")
	}
	if res.Status != "running" {
		t.Errorf("Status = %q, want %q", res.Status, "running")
	}
	if !res.Installed {
		t.Error("Installed = false, want true")
	}
}

// TestHandleUnknownRoute verifies 405 for wrong method on known endpoint.
// [REQ:REQ-P0-001] - Resource Discovery API
func TestHandleDeleteResources(t *testing.T) {
	srv := newTestServer(t, []resourceStatusFixture{
		{Name: "postgres", Installed: true, Running: true, Health: "healthy"},
	})

	w := doRequest(t, srv, http.MethodDelete, "/api/v1/resources", "")
	if w.Code == http.StatusOK {
		t.Error("DELETE /api/v1/resources should not return 200")
	}
}
