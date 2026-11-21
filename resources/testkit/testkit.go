// Package testkit contains small, resource-module-local fixtures shared by
// resource CLI tests. It deliberately owns only transport and output setup;
// resource-specific fakes remain in the package that understands them.
package testkit

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

// Harness is the common I/O surface used by handler, adapter, and service
// tests. Options let a caller supply only the seams it needs.
type Harness struct {
	In     io.Reader
	Stdout *bytes.Buffer
	Stderr *bytes.Buffer
}

type Option func(*Harness)

func WithInput(value string) Option { return func(h *Harness) { h.In = bytes.NewBufferString(value) } }

// Handlers returns isolated stdin/stdout/stderr for HTTP or command handler
// tests. The testing handle is accepted so cleanup remains tied to the test.
func Handlers(t testing.TB, options ...Option) *Harness {
	t.Helper()
	return newHarness(options...)
}

// Adapter returns isolated I/O for adapter tests.
func Adapter(t testing.TB, options ...Option) *Harness {
	t.Helper()
	return newHarness(options...)
}

// Service returns isolated I/O for service tests.
func Service(t testing.TB, options ...Option) *Harness {
	t.Helper()
	return newHarness(options...)
}

func newHarness(options ...Option) *Harness {
	h := &Harness{In: bytes.NewBuffer(nil), Stdout: new(bytes.Buffer), Stderr: new(bytes.Buffer)}
	for _, option := range options {
		if option != nil {
			option(h)
		}
	}
	return h
}

// WriteJSON writes a fixture to a path with stable indentation.
func WriteJSON(t testing.TB, path string, value any) {
	t.Helper()
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatalf("marshal JSON fixture: %v", err)
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o600); err != nil {
		t.Fatalf("write JSON fixture %s: %v", path, err)
	}
}

// WriteJSONResponse writes a JSON HTTP fixture with the standard content type.
func WriteJSONResponse(t testing.TB, w http.ResponseWriter, value any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(value); err != nil {
		t.Fatalf("encode response: %v", err)
	}
}

// Server is a convenience for tests that need a local HTTP upstream.
func Server(t testing.TB, handler http.Handler) *httptest.Server {
	t.Helper()
	return httptest.NewServer(handler)
}
