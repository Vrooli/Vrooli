//go:build linux

package cliutil

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
)

func TestDynamicCredentialAdoptionAndRefusalDoNotChangeEnvironment(t *testing.T) {
	directory, err := os.MkdirTemp("", "agc-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(directory) })
	path := filepath.Join(directory, "broker.sock")
	listener, err := net.Listen("unix", path)
	if err != nil {
		t.Fatal(err)
	}
	var state atomic.Int32
	state.Store(http.StatusNoContent)
	server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if int(state.Load()) == http.StatusOK {
			json.NewEncoder(w).Encode(map[string]string{"token": "approved-current"})
			return
		}
		w.WriteHeader(int(state.Load()))
	})}
	go server.Serve(listener)
	t.Cleanup(func() { server.Close() })
	if token, err := resolveAgentCredential(context.Background(), path, "inherited"); err != nil || token != "inherited" {
		t.Fatal("unbound process lost inherited identity")
	}
	state.Store(http.StatusOK)
	if token, err := resolveAgentCredential(context.Background(), path, "inherited"); err != nil || token != "approved-current" {
		t.Fatal("existing process failed to adopt current consent")
	}
	state.Store(http.StatusForbidden)
	token, err := resolveAgentCredential(context.Background(), path, "inherited")
	if err == nil || token != "" {
		t.Fatal("revocation fell back to older authority")
	}
	var refused *AgentAuthorizationError
	if !errors.As(err, &refused) {
		t.Fatalf("refusal is not reported as an authorization problem: %v", err)
	}
	server.Close()
	if _, err := resolveAgentCredential(context.Background(), path, "inherited"); err == nil {
		t.Fatal("a broken authority silently downgraded to inherited authority")
	}
}

func TestProcessBindingRefusesPIDReuseAndAnotherSession(t *testing.T) {
	root, err := ObserveProcessBinding(os.Getpid())
	if err != nil {
		t.Fatal(err)
	}
	if !ProcessDescendsFrom(os.Getpid(), root) {
		t.Fatal("process did not match its kernel identity")
	}
	root.Start += "-reused"
	if ProcessDescendsFrom(os.Getpid(), root) {
		t.Fatal("reused process identity accepted")
	}
	root, err = ObserveProcessBinding(os.Getpid())
	if err != nil {
		t.Fatal(err)
	}
	root.Cgroup += "/other"
	if ProcessDescendsFrom(os.Getpid(), root) {
		t.Fatal("different session boundary accepted")
	}
}

func TestAmbientIdentityIsNotSentToRemoteDestinations(t *testing.T) {
	t.Setenv(EnvIdentityToken, "private-agent-proof")
	client := NewHTTPClient(HTTPClientOptions{})
	client.SetInvocationHeaderSource(InvocationHeaders("swarm-manager", "backlog update"))
	req := httptest.NewRequest(http.MethodPost, "https://unrelated.example.test/collect", nil)
	client.ApplyRequestHeaders(req)
	if req.Header.Get(HeaderAgentIdentityToken) != "" {
		t.Fatal("ambient identity left the local scenario transport")
	}
}

type credentialRedirectTransport func(*http.Request) (*http.Response, error)

func (f credentialRedirectTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestRedirectCannotCarryAgentIdentityToAnExternalHost(t *testing.T) {
	seen := false
	next := credentialRedirectTransport(func(req *http.Request) (*http.Response, error) {
		if req.URL.Hostname() == "localhost" {
			return &http.Response{StatusCode: http.StatusFound, Header: http.Header{"Location": []string{"https://untrusted.example.test/collect"}}, Body: io.NopCloser(strings.NewReader("")), Request: req}, nil
		}
		seen = true
		if req.Header.Get(HeaderAgentIdentityToken) != "" {
			t.Error("redirect disclosed the credential of the trusted first hop")
		}
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader("")), Request: req}, nil
	})
	client := &http.Client{Transport: identityForwardingTransport{next: next}}
	req, _ := http.NewRequest(http.MethodGet, "http://localhost/start", nil)
	// An explicit first-hop identity avoids consulting any live authority.
	req.Header.Set(HeaderAgentIdentityToken, "request-local-proof")
	response, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if !seen {
		t.Fatal("redirect was not exercised")
	}
}
