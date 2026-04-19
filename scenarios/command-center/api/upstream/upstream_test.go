package upstream

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSwarm_OKResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/stats" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"throughput":42}`))
	}))
	defer srv.Close()

	c := NewSwarm(srv.URL)
	raw, err := c.Fetch(context.Background(), "/api/v1/stats")
	if err != nil {
		t.Fatalf("fetch: %v", err)
	}
	if string(raw) != `{"throughput":42}` {
		t.Errorf("unexpected body: %s", raw)
	}
	if c.Name() != "swarm" {
		t.Errorf("name=%q", c.Name())
	}
}

func TestVrooli_EmptyBaseURLIsNotAvailable(t *testing.T) {
	c := NewVrooli("")
	_, err := c.Fetch(context.Background(), "/scenarios")
	if !errors.Is(err, ErrNotAvailable) {
		t.Errorf("expected ErrNotAvailable, got %v", err)
	}
}

func TestLPBS_404FallsThroughToGapMode(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	c := NewLPBS(srv.URL, "sekret")
	_, err := c.Fetch(context.Background(), "/api/v1/admin/dashboard/summary")
	if !errors.Is(err, ErrNotAvailable) {
		t.Errorf("expected ErrNotAvailable on 404, got %v", err)
	}
}

func TestLPBS_SendsBearerToken(t *testing.T) {
	received := ""
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received = r.Header.Get("Authorization")
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	c := NewLPBS(srv.URL, "sekret-token")
	if _, err := c.Fetch(context.Background(), "/any"); err != nil {
		t.Fatalf("fetch: %v", err)
	}
	if received != "Bearer sekret-token" {
		t.Errorf("missing bearer, got %q", received)
	}
}

func TestLPBS_NoTokenSkipsHeader(t *testing.T) {
	received := ""
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received = r.Header.Get("Authorization")
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	c := NewLPBS(srv.URL, "")
	if _, err := c.Fetch(context.Background(), "/any"); err != nil {
		t.Fatalf("fetch: %v", err)
	}
	if received != "" {
		t.Errorf("expected no Authorization header, got %q", received)
	}
}

func TestBaseClient_Returns5xxAsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`boom`))
	}))
	defer srv.Close()

	c := NewSwarm(srv.URL)
	_, err := c.Fetch(context.Background(), "/x")
	if err == nil {
		t.Fatal("expected error on 500")
	}
	if errors.Is(err, ErrNotAvailable) {
		t.Error("500 should be a normal error, not ErrNotAvailable")
	}
}
