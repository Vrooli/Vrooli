package resources

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	resourcedeployment "github.com/vrooli/vrooli/packages/resource-deployment"
)

func TestManagedServiceSupervisorStartsVerifiedArtifactAndStops(t *testing.T) {
	if testing.Short() {
		t.Skip("uses a helper process")
	}
	dir := t.TempDir()
	artifactPath := os.Args[0]
	body, err := os.ReadFile(artifactPath)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(body)
	artifact := resourcedeployment.ServiceArtifact{Path: "fixture", Version: "1.0.0", SHA256: fmt.Sprintf("%x", sum)}
	supervisor := newManagedServiceSupervisor(filepath.Join(dir, "state.json"), filepath.Join(dir, "service.log"))
	state, err := supervisor.Start(artifactPath, artifact, []string{"-test.run=TestManagedServiceFixtureProcess", "--"}, append(os.Environ(), "VROOLI_MANAGED_SERVICE_FIXTURE=1"), dir)
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if state.PID <= 0 {
		t.Fatalf("Start() pid = %d", state.PID)
	}
	if state.InstanceID == "" {
		t.Fatal("Start() did not persist a managed-service identity")
	}
	if got, running, err := supervisor.Status(); err != nil || !running || got.PID != state.PID {
		t.Fatalf("Status() = (%+v, %t, %v), want live recorded process", got, running, err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := supervisor.Stop(ctx); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
	stopped, running, err := supervisor.Status()
	if err != nil || running || stopped.PID != 0 || stopped.InstanceID != state.InstanceID {
		t.Fatalf("Status() after stop = (%+v, running=%t, err=%v)", stopped, running, err)
	}
	restarted, err := supervisor.Start(artifactPath, artifact, []string{"-test.run=TestManagedServiceFixtureProcess", "--"}, append(os.Environ(), "VROOLI_MANAGED_SERVICE_FIXTURE=1"), dir)
	if err != nil {
		t.Fatalf("restart Start() error = %v", err)
	}
	if restarted.InstanceID != state.InstanceID || restarted.PreviousArtifactVersion != artifact.Version {
		t.Fatalf("restart lineage = %+v, want preserved instance and previous artifact", restarted)
	}
	if err := supervisor.Stop(ctx); err != nil {
		t.Fatalf("stop restarted service: %v", err)
	}
}

func TestManagedServiceSupervisorRejectsTamperedArtifact(t *testing.T) {
	dir := t.TempDir()
	artifactPath := filepath.Join(dir, "fixture")
	if err := os.WriteFile(artifactPath, []byte("not executable"), 0o700); err != nil {
		t.Fatal(err)
	}
	supervisor := newManagedServiceSupervisor(filepath.Join(dir, "state.json"), filepath.Join(dir, "service.log"))
	artifact := resourcedeployment.ServiceArtifact{Path: "fixture", Version: "1.0.0", SHA256: strings.Repeat("0", 64)}
	if _, err := supervisor.Start(artifactPath, artifact, nil, nil, dir); err == nil || !strings.Contains(err.Error(), "checksum mismatch") {
		t.Fatalf("Start() error = %v, want checksum mismatch", err)
	}
}

func TestManagedServiceSupervisorRejectsLivePIDWithoutOwnershipToken(t *testing.T) {
	dir := t.TempDir()
	supervisor := newManagedServiceSupervisor(filepath.Join(dir, "state.json"), filepath.Join(dir, "service.log"))
	if err := supervisor.writeState(ManagedServiceState{
		InstanceID:      "ms-test",
		OwnershipToken:  "wrong-token",
		PID:             os.Getpid(),
		ArtifactSHA256:  strings.Repeat("a", 64),
		ArtifactVersion: "1.0.0",
	}); err != nil {
		t.Fatal(err)
	}
	if _, running, err := supervisor.Status(); err == nil || running || !strings.Contains(err.Error(), "ownership token") {
		t.Fatalf("Status() = (running=%t, err=%v), want ownership denial", running, err)
	}
}

func TestManagedServiceFixtureProcess(t *testing.T) {
	if os.Getenv("VROOLI_MANAGED_SERVICE_FIXTURE") != "1" {
		return
	}
	for {
		time.Sleep(time.Second)
	}
}
