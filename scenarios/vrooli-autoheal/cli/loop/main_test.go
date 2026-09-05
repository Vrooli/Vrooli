package main

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

func TestInstallExecutableReplacesTargetAtomically(t *testing.T) {
	target := filepath.Join(t.TempDir(), "installed-loop")
	if err := os.WriteFile(target, []byte("old"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := installExecutable(target); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(target)
	if err != nil {
		t.Fatal(err)
	}
	if info.Size() == 0 {
		t.Fatal("installed executable is empty")
	}
	if info.Mode().Perm() != 0o755 {
		t.Fatalf("installed executable mode = %o, want 755", info.Mode().Perm())
	}
}

func TestValidateLocalEndpoint(t *testing.T) {
	tests := []struct {
		name string
		url  string
		want bool
	}{
		{name: "localhost", url: "http://localhost:8080/api", want: true},
		{name: "ipv4 loopback", url: "http://127.0.0.1:8080/api", want: true},
		{name: "ipv6 loopback", url: "http://[::1]:8080/api", want: true},
		{name: "remote host", url: "https://example.com/api", want: false},
		{name: "userinfo", url: "http://user@127.0.0.1:8080/api", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := validateLocalEndpoint(tt.url) == nil; got != tt.want {
				t.Fatalf("validateLocalEndpoint(%q) accepted = %v, want %v", tt.url, got, tt.want)
			}
		})
	}
}

func TestLocalHealthEndpointValidatesPort(t *testing.T) {
	for _, test := range []struct {
		port string
		want string
		ok   bool
	}{
		{port: "43123", want: "http://localhost:43123/health", ok: true},
		{port: "0", ok: false},
		{port: "65536", ok: false},
		{port: "not-a-port", ok: false},
	} {
		got, err := localHealthEndpoint(test.port)
		if test.ok {
			if err != nil || got != test.want {
				t.Fatalf("localHealthEndpoint(%q) = %q, %v; want %q", test.port, got, err, test.want)
			}
			continue
		}
		if err == nil {
			t.Fatalf("localHealthEndpoint(%q) unexpectedly accepted %q", test.port, got)
		}
	}
}

func TestRunTick_RequestsCompactResponse(t *testing.T) {
	t.Helper()

	var gotCompact string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		gotCompact = r.URL.Query().Get("compact")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"status":"ok","summary":{"total":1,"ok":1,"warning":0,"critical":0}}`))
	}))
	defer ts.Close()

	cfg := &Config{TickEndpoint: ts.URL + "/api/v1/tick"}
	result, err := runTick(context.Background(), cfg)
	if err != nil {
		t.Fatalf("runTick() error = %v", err)
	}
	if gotCompact != "true" {
		t.Fatalf("compact query param = %q, want true", gotCompact)
	}
	if result.Status != "ok" {
		t.Fatalf("status = %q, want ok", result.Status)
	}
}

// A supervisor must restart a process that has stopped answering, and must not
// restart one that is answering slowly. Conflating the two is what made a
// working retention cycle unrecoverable: every restart aborted it and the next
// attempt met the same load.
//
// The liveness question is now identity-scoped: it asks whether AUTOHEAL is
// answering, not whether anything is.
func TestAutohealIsAliveTreatsAnyAutohealAnswerAsAlive(t *testing.T) {
	for _, tc := range []struct {
		name   string
		status int
	}{
		{"healthy", http.StatusOK},
		{"service unavailable", http.StatusServiceUnavailable},
		{"internal error", http.StatusInternalServerError},
	} {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(`{"service":"Vrooli Autoheal API","status":"degraded"}`))
			}))
			defer srv.Close()

			port := srv.Listener.Addr().(*net.TCPAddr).Port
			if !autohealIsAlive(context.Background(), strconv.Itoa(port)) {
				t.Errorf("status %d reported as not alive; a process that answers is not the failure a restart fixes", tc.status)
			}
		})
	}
}

// The converse: nothing listening is exactly the condition a restart is for.
func TestAutohealIsAliveReportsNothingListeningAsDead(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	port := srv.Listener.Addr().(*net.TCPAddr).Port
	srv.Close()

	if autohealIsAlive(context.Background(), strconv.Itoa(port)) {
		t.Error("a closed port reported as alive; a genuinely dead API would never be restarted")
	}
	if autohealIsAlive(context.Background(), "") {
		t.Error("an undetected port reported as alive")
	}
}

// The 2026-09-01 sabotage, reproduced: an orphaned mock server answering 200 on
// a probed port must NOT be mistaken for autoheal. This is the check that keeps
// a stray fixture from suppressing recovery indefinitely.
func TestAutohealIsAliveRejectsForeignProcess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"service":"mock-api","status":"ok"}`))
	}))
	defer srv.Close()

	port := srv.Listener.Addr().(*net.TCPAddr).Port
	if autohealIsAlive(context.Background(), strconv.Itoa(port)) {
		t.Fatal("a foreign service answering 200 must not be adopted as autoheal")
	}
	if isAutohealAPI(context.Background(), strconv.Itoa(port)) {
		t.Error("identity probe must reject a foreign service")
	}
}

func TestBodyIdentifiesAutoheal(t *testing.T) {
	cases := []struct {
		name string
		body string
		want bool
	}{
		{"autoheal api", `{"service":"Vrooli Autoheal API"}`, true},
		{"case insensitive", `{"service":"vrooli AUTOHEAL api"}`, true},
		{"foreign service", `{"service":"mock-api"}`, false},
		{"no service field", `{"status":"healthy"}`, false},
		{"not json", `<html>hello</html>`, false},
		{"empty", ``, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := bodyIdentifiesAutoheal([]byte(tc.body)); got != tc.want {
				t.Errorf("body %q -> %v, want %v", tc.body, got, tc.want)
			}
		})
	}
}
