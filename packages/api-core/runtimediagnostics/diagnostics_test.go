//go:build linux || darwin

package runtimediagnostics

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func testIdentity() Identity {
	return Identity{Scenario: "sample", Variant: "default", RuntimeInstanceID: "inst-test", Generation: "launch-test", PID: os.Getpid()}
}

func testServer(t *testing.T) (*Client, Identity, func()) {
	t.Helper()
	// A short path also respects Darwin's smaller Unix socket address limit.
	dir, err := os.MkdirTemp("", "vrd-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	identity := testIdentity()
	stop, err := Listen(context.Background(), dir, identity)
	if err != nil {
		t.Fatal(err)
	}
	client := NewClient(SocketPath(dir, identity))
	t.Cleanup(client.Close)
	t.Cleanup(stop)
	return client, identity, stop
}

func TestPrivateDiagnosticsIdentityAndCPUProfile(t *testing.T) {
	client, identity, _ := testServer(t)
	capabilities, err := client.Capabilities(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if capabilities.Identity != identity || capabilities.MaxSeconds != 30 {
		t.Fatalf("capabilities=%+v", capabilities)
	}
	profile, err := client.CPU(context.Background(), identity, 1)
	if err != nil {
		t.Fatal(err)
	}
	gzipReader, err := gzip.NewReader(bytes.NewReader(profile))
	if err != nil {
		t.Fatal(err)
	}
	defer gzipReader.Close()
	decoded, err := io.ReadAll(gzipReader)
	if err != nil || len(decoded) == 0 {
		t.Fatalf("empty or corrupt compressed profile: %v", err)
	}
	response, err := client.do(context.Background(), http.MethodGet, "/debug/pprof/heap", nil)
	if err == nil {
		response.Body.Close()
		t.Fatal("heap diagnostics unexpectedly exposed")
	}
}

func TestCPURequestRejectsWrongLaunchAndUnboundedDurations(t *testing.T) {
	client, identity, _ := testServer(t)
	wrong := identity
	wrong.Generation = "different"
	for _, request := range []profileRequest{{Identity: wrong, Seconds: 1}, {Identity: identity, Seconds: 0}, {Identity: identity, Seconds: 31}} {
		body, _ := json.Marshal(request)
		response, err := client.do(context.Background(), http.MethodPost, "/v1/cpu", bytes.NewReader(body))
		if err == nil {
			response.Body.Close()
			t.Fatalf("invalid request accepted: %+v", request)
		}
	}
	if profiling.Load() {
		t.Fatal("rejected request started profiler")
	}
}

func TestCPUDisconnectStopsCollectionAndAllowsNextProfile(t *testing.T) {
	client, identity, _ := testServer(t)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { _, err := client.CPU(ctx, identity, 30); done <- err }()
	deadline := time.Now().Add(2 * time.Second)
	for !profiling.Load() && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if !profiling.Load() {
		cancel()
		t.Fatal("CPU profile did not start")
	}
	// A second collector must not stop or replace the first one.
	if _, err := client.CPU(context.Background(), identity, 1); err == nil {
		cancel()
		t.Fatal("concurrent CPU profile accepted")
	}
	cancel()
	if err := <-done; err == nil {
		t.Fatal("cancelled profile returned success")
	}
	deadline = time.Now().Add(2 * time.Second)
	for profiling.Load() && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if profiling.Load() {
		t.Fatal("disconnected profile kept running")
	}
	if _, err := client.CPU(context.Background(), identity, 1); err != nil {
		t.Fatalf("profile after cancellation: %v", err)
	}
}

func TestDiagnosticsRequiresPrivateDirectoryAndDoesNotClobberPaths(t *testing.T) {
	dir, err := os.MkdirTemp("", "vrd-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	if err := os.Chmod(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := Listen(context.Background(), dir, testIdentity()); err == nil {
		t.Fatal("public directory accepted")
	}
	if err := os.Chmod(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	path := SocketPath(dir, testIdentity())
	if err := os.WriteFile(path, []byte("preserve"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Listen(context.Background(), dir, testIdentity()); err == nil {
		t.Fatal("existing file replaced by diagnostic socket")
	}
	data, err := os.ReadFile(path)
	if err != nil || string(data) != "preserve" {
		t.Fatalf("existing file changed: %v", err)
	}
	link := filepath.Join(dir, "link")
	if err := os.Symlink(dir, link); err != nil {
		t.Fatal(err)
	}
	if _, err := Listen(context.Background(), link, testIdentity()); err == nil {
		t.Fatal("symlink directory accepted")
	}
}

func TestDisabledDiagnosticsHasNoFilesystemEffect(t *testing.T) {
	stop, err := StartFromEnv(context.Background(), func(string) string { return "" })
	if err != nil {
		t.Fatal(err)
	}
	stop()
}

func TestDiagnosticServerCannotClaimAnotherProcess(t *testing.T) {
	identity := testIdentity()
	identity.PID++
	if _, err := Listen(context.Background(), "/unused", identity); err == nil {
		t.Fatal("diagnostic server claimed a different process")
	}
}
