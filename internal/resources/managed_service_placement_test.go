package resources

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/vrooli/vrooli/internal/process"
	resourcedeployment "github.com/vrooli/vrooli/packages/resource-deployment"
)

func TestManagedServiceEnrolledRecoveryRefusesCallerBoundPlacement(t *testing.T) {
	for _, failure := range []string{"adoption", "observation"} {
		t.Run(failure, func(t *testing.T) {
			dir := t.TempDir()
			body, err := os.ReadFile(os.Args[0])
			if err != nil {
				t.Fatal(err)
			}
			artifact := resourcedeployment.ServiceArtifact{Path: "fixture", Version: "1", SHA256: fmt.Sprintf("%x", sha256.Sum256(body))}
			s := newManagedServiceSupervisor(filepath.Join(dir, "state.json"), filepath.Join(dir, "service.log"))
			pid := 0
			s.place = func(value int) error {
				pid = value
				if failure == "adoption" {
					return errors.New("native placement unavailable")
				}
				return nil
			}
			s.verifyPlacement = func(int) error {
				if failure == "observation" {
					return errors.New("still in caller scope")
				}
				return nil
			}
			if _, err := s.StartContext(context.Background(), os.Args[0], artifact, []string{"-test.run=^TestManagedServiceFixtureProcess$"}, []string{"VROOLI_MANAGED_SERVICE_FIXTURE=1"}, dir, nil); err == nil {
				t.Fatal("unqualified placement acknowledged as a recovered service")
			}
			if pid <= 0 || process.IsPIDRunning(pid) {
				t.Fatalf("rejected recovery left the owned root alive: %d", pid)
			}
			if _, err := os.Stat(s.statePath); !os.IsNotExist(err) {
				t.Fatalf("rejected recovery published serving state: %v", err)
			}
		})
	}
}
