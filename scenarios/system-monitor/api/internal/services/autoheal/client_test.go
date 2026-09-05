package autoheal

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestForensicsHappyPath(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/status" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"checks":[
			{"checkId":"system-pstore-evidence","status":"OK","category":"system"},
			{"checkId":"system-boot-history","status":"WARNING","category":"system"},
			{"checkId":"vrooli-api","status":"OK","category":"vrooli"},
			{"checkId":"system-cpu","status":"OK","category":"system"}
		]}`))
	}))
	defer srv.Close()

	c := NewClient(Config{BaseURL: srv.URL})
	env := c.Forensics(context.Background())
	if !env.Available {
		t.Fatalf("expected available, reason=%q", env.Reason)
	}
	// Should include both forensics checks + any other system-* check (system-cpu).
	if len(env.Checks) != 3 {
		t.Fatalf("got %d checks, want 3: %+v", len(env.Checks), env.Checks)
	}
}

func TestForensicsTimeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	c := NewClient(Config{BaseURL: srv.URL, Timeout: 30 * time.Millisecond})
	env := c.Forensics(context.Background())
	if env.Available {
		t.Fatal("expected unavailable on timeout")
	}
	if env.Reason == "" {
		t.Fatal("expected reason")
	}
}

func TestForensicsConnectionRefused(t *testing.T) {
	c := NewClient(Config{BaseURL: "http://127.0.0.1:1", Timeout: 200 * time.Millisecond})
	env := c.Forensics(context.Background())
	if env.Available {
		t.Fatal("expected unavailable on refused")
	}
}

func TestForensicsNon2xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":"boom"}`))
	}))
	defer srv.Close()
	c := NewClient(Config{BaseURL: srv.URL})
	env := c.Forensics(context.Background())
	if env.Available {
		t.Fatal("expected unavailable on 500")
	}
	if !strings.Contains(env.Reason, "500") {
		t.Errorf("reason: %q", env.Reason)
	}
}

func TestForensicsMalformedJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{not json`))
	}))
	defer srv.Close()
	c := NewClient(Config{BaseURL: srv.URL})
	env := c.Forensics(context.Background())
	if env.Available {
		t.Fatal("expected unavailable on malformed json")
	}
}

func TestForensicsNilClient(t *testing.T) {
	var c *Client
	env := c.Forensics(context.Background())
	if env.Available {
		t.Fatal("nil client should be unavailable")
	}
}
