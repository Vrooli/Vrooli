package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestIdentityTransport_InjectsHeader(t *testing.T) {
	var gotHeader string
	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		gotHeader = r.Header.Get(headerAgentIdentityToken)
	}))
	defer server.Close()

	client := &http.Client{
		Transport: &identityTransport{
			base:  http.DefaultTransport,
			token: "test-token-abc",
		},
	}

	req, err := http.NewRequest("GET", server.URL+"/test", nil)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	resp.Body.Close()

	if gotHeader != "test-token-abc" {
		t.Errorf("expected header %q, got %q", "test-token-abc", gotHeader)
	}
}

func TestIdentityTransport_NoTokenNoHeader(t *testing.T) {
	var gotHeader string
	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		gotHeader = r.Header.Get(headerAgentIdentityToken)
	}))
	defer server.Close()

	client := &http.Client{
		Transport: &identityTransport{
			base:  http.DefaultTransport,
			token: "",
		},
	}

	req, err := http.NewRequest("GET", server.URL+"/test", nil)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	resp.Body.Close()

	if gotHeader != "" {
		t.Errorf("expected no header, got %q", gotHeader)
	}
}
