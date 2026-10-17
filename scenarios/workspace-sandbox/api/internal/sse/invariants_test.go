package sse_test

import (
	"errors"
	"io"
	"net/http"
	"testing"

	"workspace-sandbox/internal/sse"
)

// TestInvariants is the canonical entry point for SSE-package
// invariants from docs/internal/INVARIANTS.md.
func TestInvariants(t *testing.T) {
	t.Run("I-SSE-2", invariantHTTPWriterRequiresFlusher)
}

// nonFlushingResponseWriter is an http.ResponseWriter that does NOT
// implement http.Flusher. Used to drive the I-SSE-2 contract.
type nonFlushingResponseWriter struct{}

func (nonFlushingResponseWriter) Header() http.Header         { return http.Header{} }
func (nonFlushingResponseWriter) Write(b []byte) (int, error) { return io.Discard.Write(b) }
func (nonFlushingResponseWriter) WriteHeader(statusCode int)  {}

// I-SSE-2 — SSE writers refuse construction without an http.Flusher.
// Pre-2026-04-28 a custom middleware silently broke the Flusher
// contract; constructing an HTTPWriter against a writer without
// Flusher used to succeed and the resulting stream buffered. Now
// construction must fail with ErrFlusherUnsupported.
func invariantHTTPWriterRequiresFlusher(t *testing.T) {
	t.Helper()
	_, err := sse.NewHTTPWriter(nonFlushingResponseWriter{})
	if err == nil {
		t.Fatal("NewHTTPWriter accepted a writer without http.Flusher; want ErrFlusherUnsupported")
	}
	if !errors.Is(err, sse.ErrFlusherUnsupported) {
		t.Errorf("NewHTTPWriter err = %v, want ErrFlusherUnsupported", err)
	}
}
