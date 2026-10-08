package main

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrooli/api-core/storage"
	"github.com/vrooli/vrooli/internal/operatorstate"
)

func TestMain(m *testing.M) {
	home, err := os.MkdirTemp("", "vrooli-onboarding-test-home-")
	if err != nil {
		panic(err)
	}
	previous, hadPrevious := os.LookupEnv("HOME")
	if err := os.Setenv("HOME", home); err != nil {
		panic(err)
	}
	stateRoot, err := os.MkdirTemp("", "vrooli-onboarding-test-state-")
	if err != nil {
		panic(err)
	}
	previousStateRoot, hadPreviousStateRoot := os.LookupEnv("VROOLI_STATE_ROOT")
	if err := os.Setenv("VROOLI_STATE_ROOT", stateRoot); err != nil {
		panic(err)
	}
	code := m.Run()
	if hadPrevious {
		_ = os.Setenv("HOME", previous)
	} else {
		_ = os.Unsetenv("HOME")
	}
	if hadPreviousStateRoot {
		_ = os.Setenv("VROOLI_STATE_ROOT", previousStateRoot)
	} else {
		_ = os.Unsetenv("VROOLI_STATE_ROOT")
	}
	_ = os.RemoveAll(home)
	_ = os.RemoveAll(stateRoot)
	os.Exit(code)
}

func operatorStateFixturePath(t *testing.T, root string) string {
	t.Helper()
	return operatorStateFixturePathAt(t, filepath.Join(root, "test-storage"))
}

func operatorStateFixturePathAt(t *testing.T, storageRoot string) string {
	t.Helper()
	t.Setenv("VROOLI_STORAGE_ROOT", storageRoot)
	resolver, err := storage.NewResolver(storage.ResolverConfig{AppID: "vrooli", Profile: storage.ProfileAuto})
	if err != nil {
		t.Fatalf("create test storage resolver: %v", err)
	}
	paths, err := resolver.Resolve(storage.Options{ScenarioID: "vrooli-onboarding"})
	if err != nil {
		t.Fatalf("resolve test operator state: %v", err)
	}
	path := filepath.Join(paths.StateDir, operatorstate.StateFile)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatalf("create test operator state directory: %v", err)
	}
	return path
}

func newV2Root(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	t.Setenv("VROOLI_ROOT", root)
	statePath := operatorStateFixturePath(t, root)
	previous := operatorStatePath
	operatorStatePath = func() (string, error) { return statePath, nil }
	t.Cleanup(func() { operatorStatePath = previous })
	return root
}

// [REQ:REQ-P0-001] Helper function tests

func TestWriteJSON(t *testing.T) {
	w := httptest.NewRecorder()
	writeJSON(w, http.StatusCreated, map[string]string{"key": "value"})

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusCreated)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("Content-Type = %q, want application/json", ct)
	}
	if !strings.Contains(w.Body.String(), `"key":"value"`) {
		t.Fatalf("body = %q, expected to contain key:value", w.Body.String())
	}
}

func TestWriteResourceLoadError(t *testing.T) {
	w := httptest.NewRecorder()
	writeResourceLoadError(w, errors.New("disk full"))

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
	body := w.Body.String()
	if !strings.Contains(body, "failed to load resources") {
		t.Fatalf("body missing prefix: %s", body)
	}
	if !strings.Contains(body, "disk full") {
		t.Fatalf("body missing error: %s", body)
	}
}

func TestDecodeJSONBodySuccess(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"test"}`))

	var dst struct{ Name string }
	ok := decodeJSONBody(w, r, &dst)
	if !ok {
		t.Fatal("decodeJSONBody returned false, want true")
	}
	if dst.Name != "test" {
		t.Fatalf("Name = %q, want %q", dst.Name, "test")
	}
}

func TestDecodeJSONBodyMalformed(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{not valid json`))

	var dst struct{ Name string }
	ok := decodeJSONBody(w, r, &dst)
	if ok {
		t.Fatal("decodeJSONBody returned true for invalid JSON")
	}
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
	if !strings.Contains(w.Body.String(), "invalid JSON body") {
		t.Fatalf("body = %q, expected invalid JSON error", w.Body.String())
	}
}

func TestDecodeJSONBodyEmpty(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(""))

	var dst struct{ Name string }
	ok := decodeJSONBody(w, r, &dst)
	if ok {
		t.Fatal("decodeJSONBody returned true for empty body")
	}
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

// TestRecoveryMiddleware verifies the handler wraps panics.
func TestRecoveryMiddleware(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("VROOLI_ROOT", dir)
	writeResourcesFile(t, dir, []map[string]string{testResPostgres})

	srv := NewServer()
	handler := srv.Handler()

	// Inject a panicking route for testing
	srv.router.HandleFunc("/panic", func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	}).Methods("GET")

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("panic recovery: status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestSecurityHeadersMiddleware(t *testing.T) {
	srv := NewServer()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	response := httptest.NewRecorder()
	srv.Handler().ServeHTTP(response, req)

	for name, want := range map[string]string{
		"Strict-Transport-Security": "max-age=31536000; includeSubDomains",
		"X-Content-Type-Options":    "nosniff",
		"X-Frame-Options":           "DENY",
		"X-XSS-Protection":          "0",
	} {
		if got := response.Header().Get(name); got != want {
			t.Errorf("%s = %q, want %q", name, got, want)
		}
	}
}
