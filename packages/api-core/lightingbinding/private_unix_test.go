//go:build linux

package lightingbinding

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"testing"
)

func TestPrivateLightingSocketPermissionsAndIdentity(t *testing.T) {
	directory, err := os.MkdirTemp("", "lighting-private-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(directory)
	identity := Identity{Scenario: "lighting-control", Variant: "live", RuntimeInstanceID: "one", Generation: "one", PID: os.Getpid()}
	listener, err := Listen(directory, identity)
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(SocketPath(directory, identity))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0600 {
		t.Fatalf("socket mode %v", info.Mode())
	}
	if _, err := Listen(directory, identity); err == nil {
		t.Fatal("second owner replaced existing live socket")
	}
	_ = listener.Close()
	if err := os.Chmod(directory, 0755); err != nil {
		t.Fatal(err)
	}
	if listener, err := Listen(directory, identity); err == nil {
		listener.Close()
		t.Fatal("public directory accepted")
	}
	if err := os.Chmod(directory, 0700); err != nil {
		t.Fatal(err)
	}
	foreign := identity
	foreign.Generation = "two"
	foreign.PID++
	if listener, err := Listen(directory, foreign); err == nil {
		listener.Close()
		t.Fatal("foreign process identity accepted")
	}
	link := filepath.Join(directory, "link")
	if err := os.Symlink(directory, link); err != nil {
		t.Fatal(err)
	}
	if listener, err := Listen(link, identity); err == nil {
		listener.Close()
		t.Fatal("symlink directory accepted")
	}
}

func TestClientRejectsListenerWhoseKernelPIDDiffersFromClaim(t *testing.T) {
	directory, err := os.MkdirTemp("", "lighting-peer-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(directory)
	identity := Identity{Scenario: "lighting-control", Variant: "live", RuntimeInstanceID: "one", Generation: "one", PID: os.Getpid()}
	listener, err := Listen(directory, identity)
	if err != nil {
		t.Fatal(err)
	}
	server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(State{Version: Version, Identity: identity})
	})}
	done := make(chan struct{})
	go func() { defer close(done); _ = server.Serve(listener) }()
	defer func() { _ = server.Close(); <-done }()
	spoofed := NewClient(SocketPath(directory, identity), os.Getpid()+1)
	defer spoofed.Close()
	if _, err := spoofed.State(context.Background()); err == nil {
		t.Fatal("self-reported identity overrode kernel peer PID")
	}
}
