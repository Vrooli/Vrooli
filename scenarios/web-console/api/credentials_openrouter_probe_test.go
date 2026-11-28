package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestProbeOpenRouterKeyUsesSharedModelsTransport(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/models" {
			t.Errorf("request = %s %s, want GET /api/v1/models", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer fixture-key" {
			t.Errorf("authorization = %q, want fixture bearer key", got)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	if err := probeOpenRouterKey(context.Background(), "fixture-key", server.Client(), server.URL+"/api/v1"); err != nil {
		t.Fatalf("2xx response should validate the credential: %v", err)
	}
}

func TestProbeOpenRouterKeyPreservesUnauthorizedFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":{"message":"invalid key"}}`))
	}))
	defer server.Close()

	err := probeOpenRouterKey(context.Background(), "bad-key", server.Client(), server.URL+"/api/v1")
	if err == nil || !strings.Contains(err.Error(), "401") {
		t.Fatalf("401 probe error = %v, want status failure", err)
	}
}

func TestProbeOpenRouterKeyHonorsRequestTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	if err := probeOpenRouterKey(ctx, "fixture-key", server.Client(), server.URL+"/api/v1"); err == nil {
		t.Fatal("expected timeout error")
	}
}
