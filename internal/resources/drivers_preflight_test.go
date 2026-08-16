package resources

import (
	"net"
	"strconv"
	"strings"
	"testing"
)

// inferenceLikeManifest mirrors a generic docker-service manifest closely enough to
// exercise the host-port preflight (a single http port mapped 1:1).
func inferenceLikeManifest(hostPort int) ResourceManifest {
	return ResourceManifest{
		Name:   "inference-service",
		Driver: "docker-service",
		Ports: []ResourcePort{
			{Name: "http", Container: 11434, Host: hostPort},
		},
		Runtime: ResourceRuntime{Image: "example/inference-service:0.30.10"},
	}
}

func TestPreflightPortConflictFailsWhenHostOwnsPort(t *testing.T) {
	// Occupy a port the way a stray host-systemd Ollama would.
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer listener.Close()

	hostPort := listener.Addr().(*net.TCPAddr).Port
	manifest := inferenceLikeManifest(hostPort)

	err = preflightPortConflict(manifest)
	if err == nil {
		t.Fatalf("expected conflict error for occupied port %d, got nil", hostPort)
	}
	msg := err.Error()
	// The message must be actionable: name the resource, the port, and remediation.
	for _, want := range []string{"inference-service", strconv.Itoa(hostPort), "Docker container", "systemctl disable"} {
		if !strings.Contains(msg, want) {
			t.Errorf("error message missing %q; got: %s", want, msg)
		}
	}
}

func TestPreflightPortConflictPassesWhenPortFree(t *testing.T) {
	// Grab a free port, then release it so the preflight sees it as available.
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	hostPort := listener.Addr().(*net.TCPAddr).Port
	if err := listener.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	manifest := inferenceLikeManifest(hostPort)
	if err := preflightPortConflict(manifest); err != nil {
		t.Fatalf("expected no conflict for free port %d, got: %v", hostPort, err)
	}
}

func TestPreflightPortConflictIgnoresPortlessManifest(t *testing.T) {
	manifest := ResourceManifest{Name: "inference-service", Driver: "docker-service"}
	if err := preflightPortConflict(manifest); err != nil {
		t.Fatalf("expected no conflict for manifest without ports, got: %v", err)
	}
}
