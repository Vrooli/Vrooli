package main

import (
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestBuildVrooliCmd_InjectsNoStaleCheck(t *testing.T) {
	cases := [][]string{
		{"scenario", "port", "vrooli-autoheal", "API_PORT"},
		{"scenario", "status", "vrooli-autoheal", "--json"},
		{"scenario", "start", "vrooli-autoheal", "--best-effort"},
		{"scenario", "restart", "vrooli-autoheal", "--best-effort"},
	}
	for _, sub := range cases {
		cmd := buildVrooliCmd("/tmp/vrooli", sub...)
		joined := strings.Join(cmd.Args, " ")
		if !strings.Contains(joined, "--no-stale-check") {
			t.Errorf("argv missing --no-stale-check for %v: %v", sub, cmd.Args)
			continue
		}
		idxFlag := indexOf(cmd.Args, "--no-stale-check")
		idxSub := indexOf(cmd.Args, sub[0])
		if idxFlag < 0 || idxSub < 0 || idxFlag > idxSub {
			t.Errorf("--no-stale-check must precede %q in %v", sub[0], cmd.Args)
		}
	}
	_ = runtime.GOOS
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

func indexOf(s []string, want string) int {
	for i, v := range s {
		if v == want {
			return i
		}
	}
	return -1
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
	result, err := runTick(cfg)
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
func TestAPIIsAliveTreatsAnyHTTPAnswerAsAlive(t *testing.T) {
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
			}))
			defer srv.Close()

			port := srv.Listener.Addr().(*net.TCPAddr).Port
			if !apiIsAlive(strconv.Itoa(port)) {
				t.Errorf("status %d reported as not alive; a process that answers HTTP is not the failure a restart fixes", tc.status)
			}
		})
	}
}

// The converse: nothing listening is exactly the condition a restart is for.
func TestAPIIsAliveReportsNothingListeningAsDead(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	port := srv.Listener.Addr().(*net.TCPAddr).Port
	srv.Close()

	if apiIsAlive(strconv.Itoa(port)) {
		t.Error("a closed port reported as alive; a genuinely dead API would never be restarted")
	}
	if apiIsAlive("") {
		t.Error("an undetected port reported as alive")
	}
}

// The 2026-08-01 deadlock, reproduced.
//
// exec.Cmd.CombinedOutput reads the child's pipe until EOF, and EOF requires
// every write end to be closed — not merely that the child exited. `vrooli
// scenario start` spawns the runtime supervisor, which inherits that pipe and
// holds it for as long as it runs. The loop's first act on boot blocked forever
// in Wait, never reached its tick loop, and never started anything.
//
// This test builds exactly that shape: a command that exits promptly after
// spawning a descendant which keeps the inherited output pipe open. The helper
// must return the command's output on the command's own timescale.
func TestVrooliCommandReturnsWhenADescendantHoldsThePipe(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell fixture is POSIX-specific")
	}
	script := filepath.Join(t.TempDir(), "fake-vrooli")
	// Echoes, then leaves a background child holding stdout/stderr for far
	// longer than this test is willing to wait.
	body := "#!/bin/sh\necho started\nsleep 300 &\nexit 0\n"
	if err := os.WriteFile(script, []byte(body), 0o755); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	config := &Config{VrooliCmdPath: script, VrooliRoot: t.TempDir()}

	done := make(chan struct{})
	var out []byte
	var err error
	go func() {
		out, err = runVrooliCommand(config, "scenario", "start", "x")
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(vrooliOutputGrace + 20*time.Second):
		t.Fatal("runVrooliCommand blocked on a pipe held by a descendant; this is the deadlock that stopped the supervisor from ever ticking")
	}

	if err != nil {
		t.Fatalf("runVrooliCommand: %v", err)
	}
	if !strings.Contains(string(out), "started") {
		t.Errorf("output = %q, want it to contain the command's own output", out)
	}
}

// The tolerance for inherited pipes must not swallow a genuine failure: a
// command that exits non-zero still has to be reported as an error.
func TestVrooliCommandStillReportsFailure(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell fixture is POSIX-specific")
	}
	script := filepath.Join(t.TempDir(), "fake-vrooli")
	if err := os.WriteFile(script, []byte("#!/bin/sh\necho boom >&2\nexit 3\n"), 0o755); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	config := &Config{VrooliCmdPath: script, VrooliRoot: t.TempDir()}

	out, err := runVrooliCommand(config, "scenario", "start", "x")
	if err == nil {
		t.Fatal("a command exiting 3 was reported as success")
	}
	if !strings.Contains(string(out), "boom") {
		t.Errorf("output = %q, want the failing command's diagnostics", out)
	}
}
