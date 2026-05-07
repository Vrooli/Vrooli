package handlers

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"system-monitor-api/internal/services/journal"
)

type fakeReader struct {
	entries  []journal.LogEntry
	boots    []journal.BootRecord
	err      error
	lastOpts journal.QueryOpts
	avail    bool
}

func (f *fakeReader) QueryLogs(_ context.Context, opts journal.QueryOpts) ([]journal.LogEntry, error) {
	f.lastOpts = opts
	return f.entries, f.err
}

func (f *fakeReader) ListBoots(_ context.Context) ([]journal.BootRecord, error) {
	return f.boots, f.err
}
func (f *fakeReader) Available(_ context.Context) bool { return f.avail }

func TestLogsHandlerHappyPath(t *testing.T) {
	r := &fakeReader{entries: []journal.LogEntry{
		{Message: "hello", Cursor: "s=abc;i=1"},
		{Message: "world", Cursor: "s=abc;i=2"},
	}}
	h := NewLogsHandler(r, slog.Default())
	req := httptest.NewRequest(http.MethodGet, "/api/v1/logs?unit=docker&priority=err&grep=panic", nil)
	w := httptest.NewRecorder()
	h.Logs(w, req)
	if w.Code != 200 {
		t.Fatalf("status: %d", w.Code)
	}
	var resp LogsResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if !resp.Available {
		t.Fatalf("expected available")
	}
	if len(resp.Entries) != 2 {
		t.Errorf("entries: %d", len(resp.Entries))
	}
	if resp.NextCursor == "" {
		t.Error("expected next cursor")
	}
	// Verify filters were passed through.
	if r.lastOpts.Priority != "err" || r.lastOpts.Grep != "panic" {
		t.Errorf("opts: %+v", r.lastOpts)
	}
	if len(r.lastOpts.Unit) != 1 || r.lastOpts.Unit[0] != "docker" {
		t.Errorf("unit: %v", r.lastOpts.Unit)
	}
}

func TestLogsHandlerCursorRoundTrip(t *testing.T) {
	cursor := "s=abc;i=42"
	encoded := base64.RawURLEncoding.EncodeToString([]byte(cursor))
	r := &fakeReader{entries: []journal.LogEntry{{Message: "x", Cursor: "s=abc;i=43"}}}
	h := NewLogsHandler(r, slog.Default())
	req := httptest.NewRequest(http.MethodGet, "/api/v1/logs?cursor="+encoded, nil)
	w := httptest.NewRecorder()
	h.Logs(w, req)
	if w.Code != 200 {
		t.Fatalf("status: %d", w.Code)
	}
	if r.lastOpts.AfterCursor != cursor {
		t.Errorf("AfterCursor=%q, want %q", r.lastOpts.AfterCursor, cursor)
	}
	var resp LogsResponse
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Direction != "forward" {
		t.Errorf("direction with cursor should default to forward, got %q", resp.Direction)
	}
}

func TestLogsHandlerInvalidCursor(t *testing.T) {
	r := &fakeReader{}
	h := NewLogsHandler(r, slog.Default())
	req := httptest.NewRequest(http.MethodGet, "/api/v1/logs?cursor=!!!!", nil)
	w := httptest.NewRecorder()
	h.Logs(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status: %d", w.Code)
	}
}

func TestLogsHandlerLimitClamped(t *testing.T) {
	r := &fakeReader{}
	h := NewLogsHandler(r, slog.Default())
	req := httptest.NewRequest(http.MethodGet, "/api/v1/logs?limit=99999", nil)
	w := httptest.NewRecorder()
	h.Logs(w, req)
	if r.lastOpts.Tail != logsMaxLimit {
		t.Errorf("Tail=%d, want %d", r.lastOpts.Tail, logsMaxLimit)
	}
}

func TestLogsHandlerLimitDefault(t *testing.T) {
	r := &fakeReader{}
	h := NewLogsHandler(r, slog.Default())
	req := httptest.NewRequest(http.MethodGet, "/api/v1/logs", nil)
	w := httptest.NewRecorder()
	h.Logs(w, req)
	if r.lastOpts.Tail != logsDefaultLimit {
		t.Errorf("Tail=%d, want %d", r.lastOpts.Tail, logsDefaultLimit)
	}
	if !r.lastOpts.Reverse {
		t.Error("default should be reverse direction")
	}
}

func TestLogsHandlerJournalError(t *testing.T) {
	r := &fakeReader{err: errors.New("journalctl not found")}
	h := NewLogsHandler(r, slog.Default())
	req := httptest.NewRequest(http.MethodGet, "/api/v1/logs", nil)
	w := httptest.NewRecorder()
	h.Logs(w, req)
	// Not 5xx — graceful envelope.
	if w.Code != 200 {
		t.Errorf("status: %d", w.Code)
	}
	var resp LogsResponse
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Available {
		t.Error("should be unavailable on error")
	}
}

func TestLogsHandlerNilReader(t *testing.T) {
	h := NewLogsHandler(nil, slog.Default())
	req := httptest.NewRequest(http.MethodGet, "/api/v1/logs", nil)
	w := httptest.NewRecorder()
	h.Logs(w, req)
	if w.Code != 200 {
		t.Errorf("status: %d", w.Code)
	}
	var resp LogsResponse
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Available {
		t.Error("nil reader should be unavailable")
	}
}

func TestUnitsHandler(t *testing.T) {
	r := &fakeReader{entries: []journal.LogEntry{
		{Unit: "docker.service"},
		{Unit: "docker.service"},
		{UserUnit: "user@1000.service"},
	}}
	h := NewLogsHandler(r, slog.Default())
	req := httptest.NewRequest(http.MethodGet, "/api/v1/logs/units", nil)
	w := httptest.NewRecorder()
	h.Units(w, req)
	var resp UnitsResponse
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if !resp.Available {
		t.Fatal("expected available")
	}
	if len(resp.Units) != 2 {
		t.Errorf("units: %v", resp.Units)
	}
}

func TestBootsHandler(t *testing.T) {
	r := &fakeReader{boots: []journal.BootRecord{{Index: -1, BootID: "abc"}, {Index: 0, BootID: "def"}}}
	h := NewLogsHandler(r, slog.Default())
	req := httptest.NewRequest(http.MethodGet, "/api/v1/logs/boots", nil)
	w := httptest.NewRecorder()
	h.Boots(w, req)
	var resp BootsResponse
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if !resp.Available || len(resp.Boots) != 2 {
		t.Errorf("resp: %+v", resp)
	}
}

func TestBootsHandlerError(t *testing.T) {
	r := &fakeReader{err: errors.New("denied")}
	h := NewLogsHandler(r, slog.Default())
	req := httptest.NewRequest(http.MethodGet, "/api/v1/logs/boots", nil)
	w := httptest.NewRecorder()
	h.Boots(w, req)
	if w.Code != 200 {
		t.Errorf("status: %d", w.Code)
	}
	var resp BootsResponse
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Available {
		t.Error("expected unavailable")
	}
}
