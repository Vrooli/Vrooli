package handlers

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

// =============================================================================
// Connection Handling Tests
// =============================================================================

// mockResponseWriter implements http.ResponseWriter and can simulate errors.
type mockResponseWriter struct {
	headers    http.Header
	body       strings.Builder
	statusCode int
	writeErr   error
	mu         sync.Mutex
}

func newMockResponseWriter() *mockResponseWriter {
	return &mockResponseWriter{
		headers:    make(http.Header),
		statusCode: http.StatusOK,
	}
}

func (m *mockResponseWriter) Header() http.Header {
	return m.headers
}

func (m *mockResponseWriter) Write(data []byte) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.writeErr != nil {
		return 0, m.writeErr
	}
	return m.body.Write(data)
}

func (m *mockResponseWriter) WriteHeader(code int) {
	m.statusCode = code
}

func (m *mockResponseWriter) simulateDisconnect() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.writeErr = errors.New("client disconnected")
}

// TestSSEWriter_ClientDisconnect verifies handling of client disconnect mid-stream.
func TestSSEWriter_ClientDisconnect(t *testing.T) {
	mock := newMockResponseWriter()
	writer := NewSSEWriter(mock)

	// First write succeeds
	err := writer.SendContent("Hello")
	if err != nil {
		t.Fatalf("first write should succeed: %v", err)
	}

	// Simulate disconnect
	mock.simulateDisconnect()

	// Second write should fail
	err = writer.SendContent("World")
	if err == nil {
		t.Error("expected error after disconnect")
	}
}

// =============================================================================
// Concurrent Write Tests
// =============================================================================

// TestSSEWriter_ConcurrentWrites tests concurrent writes are serialized correctly.
func TestSSEWriter_ConcurrentWrites(t *testing.T) {
	rec := httptest.NewRecorder()
	writer := NewSSEWriter(rec)

	var wg sync.WaitGroup
	numWriters := 10
	numEvents := 100

	// Spawn multiple goroutines writing events
	for i := 0; i < numWriters; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numEvents; j++ {
				writer.SendContent("test")
			}
		}(i)
	}

	wg.Wait()

	// All events should be written
	events := parseSSEDataEvents(rec.Body.String())
	expected := numWriters * numEvents
	if len(events) != expected {
		t.Errorf("expected %d events, got %d", expected, len(events))
	}
}
