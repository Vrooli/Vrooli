package resources

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/vrooli/binaryfetch"
	"github.com/vrooli/vrooli/internal/artifactlock"
	resourcedeployment "github.com/vrooli/vrooli/packages/resource-deployment"
)

func TestManagedAcquisitionLeavesQueuedOwnerLockOnCancellation(t *testing.T) {
	for _, kind := range []string{"resource", "resource-data"} {
		t.Run(kind, func(t *testing.T) {
			name := "canceled-acquisition-" + kind
			release, err := artifactlock.AcquireWithPath(kind+":"+name, func(string) (string, error) { return filepath.Join(t.TempDir(), "owned.lock"), nil })
			if err != nil {
				t.Fatal(err)
			}
			defer release()
			root := t.TempDir()
			controller := NewController(root, filepath.Join(root, "home"))
			manifest := ResourceManifest{Name: name, ManagedService: &resourcedeployment.ManagedService{Acquisition: &binaryfetch.Acquisition{Kind: "url"}}}
			if kind == "resource-data" {
				manifest.ManagedService.DataArtifacts = []resourcedeployment.ManagedServiceDataArtifact{{Name: "fixture", Path: "fixture.bin", Acquisition: &binaryfetch.Acquisition{Kind: "url"}}}
			}
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
			defer cancel()
			done := make(chan error, 1)
			go func() {
				if kind == "resource" {
					done <- ensureManagedServiceArtifact(ctx, controller, manifest)
				} else {
					done <- ensureManagedServiceDataArtifacts(ctx, controller, manifest)
				}
			}()
			select {
			case err := <-done:
				if !errors.Is(err, context.DeadlineExceeded) {
					t.Fatalf("queued cancellation changed into acquisition: %v", err)
				}
			case <-time.After(time.Second):
				release()
				select {
				case <-done:
				case <-time.After(time.Second):
				}
				t.Fatal("canceled acquisition remained queued behind owner")
			}
		})
	}
}

func TestCanceledManagedStartNeverSpawnsOrPublishesState(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "service")
	body := []byte("verified fixture")
	if err := os.WriteFile(path, body, 0o700); err != nil {
		t.Fatal(err)
	}
	artifact := resourcedeployment.ServiceArtifact{Path: "service", Version: "1.0.0", SHA256: fmt.Sprintf("%x", sha256.Sum256(body))}
	supervisor := newManagedServiceSupervisor(filepath.Join(dir, "state.json"), filepath.Join(dir, "service.log"))
	calls := 0
	supervisor.start = func(*exec.Cmd) error { calls++; return errors.New("must not spawn") }
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := supervisor.StartContext(ctx, path, artifact, nil, nil, dir, nil); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled start continued verification/spawn: %v", err)
	}
	if calls != 0 {
		t.Fatal("canceled start spawned service")
	}
	if _, err := os.Stat(supervisor.statePath); !os.IsNotExist(err) {
		t.Fatalf("canceled start published state: %v", err)
	}
	if _, err := os.Stat(supervisor.logPath); !os.IsNotExist(err) {
		t.Fatalf("canceled start wrote service log: %v", err)
	}
}
