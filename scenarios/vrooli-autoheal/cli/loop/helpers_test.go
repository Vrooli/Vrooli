package main

import (
	"context"
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

// fakeAPI serves /health with the given service name and answers /tick, so
// a test can stand in for the autoheal API (service "Vrooli Autoheal API")
// or for a stranger squatting a port (service "mock-api").
func fakeAPI(t *testing.T, service string) string {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/health":
			_, _ = w.Write([]byte(`{"service":"` + service + `","status":"healthy"}`))
		case "/api/v1/tick":
			_, _ = w.Write([]byte(`{"success":true,"status":"ok","summary":{"total":1,"ok":1,"warning":0,"critical":0}}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)
	return strconv.Itoa(srv.Listener.Addr().(*net.TCPAddr).Port)
}

// closedPort returns a loopback port nothing listens on.
func closedPort(t *testing.T) string {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	_ = listener.Close()
	return strconv.Itoa(port)
}

// fakeVrooli writes a POSIX shell stand-in for the CLI. Every invocation is
// appended to <dir>/calls so a test can assert what the loop asked for.
// The body receives the argv as $1 $2 ...; $CALLS names the call log.
func fakeVrooli(t *testing.T, body string) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("shell fixture is POSIX-specific")
	}
	dir := t.TempDir()
	script := filepath.Join(dir, "vrooli")
	content := "#!/bin/sh\nCALLS=" + filepath.Join(dir, "calls") + "\necho \"$*\" >> \"$CALLS\"\n" + body
	if err := os.WriteFile(script, []byte(content), 0o755); err != nil {
		t.Fatal(err)
	}
	return script
}

// vrooliCalls returns the argv lines the fake recorded.
func vrooliCalls(t *testing.T, script string) []string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(filepath.Dir(script), "calls"))
	if err != nil {
		return nil
	}
	return strings.Split(strings.TrimSpace(string(data)), "\n")
}

func countCalls(calls []string, prefix string) int {
	n := 0
	for _, call := range calls {
		if strings.HasPrefix(call, prefix) {
			n++
		}
	}
	return n
}

// usageBody makes the fake reject everything the way the root parser does.
const usageBody = "echo \"Unknown command: $1\" >&2\necho \"Run 'vrooli --help' for usage information\" >&2\nexit 1\n"

// contractBody answers the preflight's two probes; everything else is a
// usage error unless the caller appends more cases before it.
func contractBody(port string) string {
	return `case "$1 $2" in
  "version --json") echo '{"cli_version":"1.0.0","platform_version":"2.0.0"}'; exit 0;;
  "scenario status") echo '{"success":true,"scenario":{"name":"vrooli-autoheal","ports":{"API_PORT":` + orZero(port) + `}},"runtime":{"ports":{}}}'; exit 0;;
esac
` + usageBody
}

func orZero(port string) string {
	if port == "" {
		return "0"
	}
	return port
}

// isolatedHome points the runtime home at a fresh directory so state,
// process and log lookups never touch the operator's ~/.vrooli.
func isolatedHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("API_PORT", "")
	return home
}

// testConfig is a Config with fast timings and no probe list, pointed at an
// empty repo root so the process registry is whatever the test writes.
func testConfig(t *testing.T) *Config {
	t.Helper()
	return &Config{
		TickInterval:        time.Second,
		MaxFailures:         3,
		StartupTimeout:      400 * time.Millisecond,
		HealthCheckInterval: 50 * time.Millisecond,
		VrooliRoot:          t.TempDir(),
		ScenarioName:        "vrooli-autoheal",
		ManageAPILifecycle:  true,
	}
}

// writeRegistryFile writes a process-registry entry under the repo root.
func writeRegistryFile(t *testing.T, config *Config, name, content string) {
	t.Helper()
	dir := filepath.Join(config.VrooliRoot, ".vrooli", "processes", "scenarios", config.ScenarioName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// startBody makes `scenario start|restart` succeed by publishing port into
// the registry, the observable effect of a real start.
func startBody(config *Config, port string) string {
	portFile := filepath.Join(config.VrooliRoot, ".vrooli", "processes", "scenarios", config.ScenarioName, "port")
	return `case "$1 $2" in
  "scenario start"|"scenario restart") mkdir -p "` + filepath.Dir(portFile) + `"; echo ` + port + ` > "` + portFile + `"; echo started; exit 0;;
esac
`
}

// testLoop builds a loop whose sleeps return immediately and are recorded.
func testLoop(t *testing.T, config *Config) (*loop, *[]time.Duration) {
	t.Helper()
	l := newLoop(config, &statusWriter{path: filepath.Join(t.TempDir(), "loop-status.json")})
	var slept []time.Duration
	l.sleep = func(ctx context.Context, d time.Duration) bool {
		slept = append(slept, d)
		return ctx.Err() == nil
	}
	return l, &slept
}
