//go:build linux || darwin

package server

import (
	"context"
	"net/http"
	"os"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/vrooli/api-core/runtimediagnostics"
)

func TestStandardServerOwnsDiagnosticSocketLifecycle(t *testing.T) {
	directory, err := os.MkdirTemp("", "vsd-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(directory)
	identity := runtimediagnostics.Identity{Scenario: "sample", Variant: "default", RuntimeInstanceID: "inst-server", Generation: "generation-server", PID: os.Getpid()}
	env := map[string]string{runtimediagnostics.DirectoryEnv: directory, "VROOLI_SCENARIO": identity.Scenario, "VROOLI_VARIANT": identity.Variant, "VROOLI_RUNTIME_INSTANCE_ID": identity.RuntimeInstanceID, "VROOLI_COMPONENT_GENERATION": identity.Generation}
	started := make(chan struct{}, 1)
	signals := make(chan os.Signal, 1)
	done := make(chan error, 1)
	go func() {
		done <- Run(Config{Handler: http.NotFoundHandler(), Port: "0", EnvGetter: func(key string) string { return env[key] }, signalChan: signals, Logger: func(format string, _ ...interface{}) {
			if strings.HasPrefix(format, "Starting server") {
				started <- struct{}{}
			}
		}})
	}()
	select {
	case <-started:
	case err := <-done:
		t.Fatalf("server exited: %v", err)
	case <-time.After(2 * time.Second):
		t.Fatal("server did not start")
	}
	client := runtimediagnostics.NewClient(runtimediagnostics.SocketPath(directory, identity))
	defer client.Close()
	capabilities, capErr := client.Capabilities(context.Background())
	signals <- syscall.SIGTERM
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(4 * time.Second):
		t.Fatal("server did not stop")
	}
	if capErr != nil || capabilities.Identity != identity {
		t.Fatalf("diagnostic identity=%+v err=%v", capabilities, capErr)
	}
	if _, err := os.Lstat(runtimediagnostics.SocketPath(directory, identity)); !os.IsNotExist(err) {
		t.Fatalf("diagnostic socket survived shutdown: %v", err)
	}
}
