package resources

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrooli/binaryfetch"
	testresource "github.com/vrooli/vrooli/internal/resources/resourcestest"
)

// runningArtifactFixture installs version 1 of a URL-acquired managed service,
// records it as running from its artifact path, and then re-declares the same
// version with new bytes, which is what happened to postgres on 2026-10-08.
func runningArtifactFixture(t *testing.T) (*Controller, ResourceManifest, string, *ManagedServiceSupervisor, *httptest.Server) {
	t.Helper()
	release := []byte("#!/bin/sh\necho replacement\n" + strings.Repeat("#", 2048) + "\n")
	sum := sha256.Sum256(release)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write(release) }))
	t.Cleanup(server.Close)

	root, home := t.TempDir(), t.TempDir()
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("VROOLI_RESOURCE_ARTIFACT_DIR", t.TempDir())
	manifest := testresource.ResourceManifest("fixture", testresource.WithResourceDriver("managed-service"))
	manifest.ManagedService.Artifact.SHA256, manifest.ManagedService.Artifact.SHA256ByPlatform = hex.EncodeToString(sum[:]), nil
	manifest.ManagedService.Acquisition = &binaryfetch.Acquisition{Kind: "url", Targets: []binaryfetch.AcquisitionTarget{{URL: server.URL + "/server", SHA256: hex.EncodeToString(sum[:])}}}
	testresource.WriteResourceManifest(t, root, "fixture", manifest)
	controller := NewController(root, home)
	loaded, err := controller.ResourceManifest("fixture")
	if err != nil {
		t.Fatal(err)
	}
	path, err := managedServiceArtifactPath(controller, loaded)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("launched image"), 0o755); err != nil {
		t.Fatal(err)
	}
	supervisor, _, err := managedServiceSupervisorFor("fixture")
	if err != nil {
		t.Fatal(err)
	}
	withOwnershipFacts(t, func(int) (map[string]string, error) {
		return map[string]string{managedServiceOwnershipTokenEnv: "owned"}, nil
	}, executableAt(path))
	launched := sha256.Sum256([]byte("launched image"))
	if err := supervisor.writeState(ManagedServiceState{
		InstanceID: "owned-fixture", PID: os.Getpid(), Incarnation: "boot:original",
		ArtifactPath: path, ArtifactVersion: loaded.ManagedService.Artifact.Version,
		ArtifactSHA256: hex.EncodeToString(launched[:]), OwnershipTokenHash: managedServiceTokenHash("owned"),
	}); err != nil {
		t.Fatal(err)
	}
	return controller, loaded, path, supervisor, server
}

// Acquisition must never rewrite the path a running service launched from:
// the replacement waits beside it, the running image stays verifiable, and
// the next start applies it.
func TestEnsureStagesReplacementBesideARunningArtifact(t *testing.T) {
	controller, manifest, path, supervisor, server := runningArtifactFixture(t)
	ctx := context.Background()

	if err := ensureManagedServiceArtifact(ctx, controller, manifest); err != nil {
		t.Fatalf("ensure while running: %v", err)
	}
	if body, err := os.ReadFile(path); err != nil || string(body) != "launched image" {
		t.Fatalf("running artifact was rewritten: %q, %v", body, err)
	}
	state, running, err := supervisor.Status()
	if err != nil || !running {
		t.Fatalf("fixture is not running: %v", err)
	}
	if err := verifyManagedServiceRunningArtifact(ctx, state); err != nil {
		t.Fatalf("running image no longer verifies against its launch digest: %v", err)
	}
	pending := managedServiceArtifactPendingPath(path)
	if body, err := os.ReadFile(pending); err != nil || !strings.Contains(string(body), "replacement") {
		t.Fatalf("replacement was not staged beside the running artifact: %q, %v", body, err)
	}

	// The service stops (restart's stop step); the next start's ensure applies
	// the staged bytes without downloading them again.
	server.Close() // the staged copy must be applied without another download
	stopped := state
	stopped.PID = 0
	if err := supervisor.writeState(stopped); err != nil {
		t.Fatal(err)
	}
	if err := ensureManagedServiceArtifact(ctx, controller, manifest); err != nil {
		t.Fatalf("ensure after stop: %v", err)
	}
	if body, err := os.ReadFile(path); err != nil || !strings.Contains(string(body), "replacement") {
		t.Fatalf("staged replacement was not applied: %q, %v", body, err)
	}
	if _, err := os.Stat(pending); !os.IsNotExist(err) {
		t.Fatalf("applied replacement left its staged copy: %v", err)
	}
}
