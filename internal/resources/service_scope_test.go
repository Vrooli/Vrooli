package resources

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	platform "github.com/vrooli/platform-go"
	resourcedeployment "github.com/vrooli/vrooli/packages/resource-deployment"
)

func TestMissingCompanionScopeAllowsFreshLaunch(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("cgroup ownership is Linux-specific")
	}
	ref := platform.ScopeRef{Kind: platform.ScopeKindCgroup, Name: "absent", Path: filepath.Join("/vrooli-missing-companion", filepath.Base(t.TempDir()))}
	pids, err := companionScopeProcesses(ref)
	if err != nil || len(pids) != 0 {
		t.Fatalf("absent scope must permit a fresh companion launch, got pids=%v err=%v", pids, err)
	}
}

func TestInvalidCompanionScopeDoesNotAuthorizeFreshLaunch(t *testing.T) {
	if _, err := companionScopeProcesses(platform.ScopeRef{}); err == nil {
		t.Fatal("invalid ownership scope must fail closed")
	}
}

func TestResourceServiceScopeNameIsStableAndSanitized(t *testing.T) {
	first := resourceServiceScopeName("kokoro", "service")
	second := resourceServiceScopeName("kokoro", "service")
	if first != second {
		t.Fatalf("scope name is not stable: %q vs %q", first, second)
	}
	if !strings.HasPrefix(first, "vrooli-service-resource-kokoro-service-") {
		t.Fatalf("scope name %q lost its operator-legible stem", first)
	}
	odd := resourceServiceScopeName("My_Res.2", "companion-Edge")
	if strings.ContainsAny(odd, "_.ABCDEFGHIJKLMNOPQRSTUVWXYZ") {
		t.Fatalf("scope name %q is not sanitized", odd)
	}
	if resourceServiceScopeName("a", "bc") == resourceServiceScopeName("ab", "c") {
		t.Fatal("tuple boundary must distinguish resource and component")
	}
	long := resourceServiceScopeName(strings.Repeat("k", 300), "service")
	if len(long) > 200 {
		t.Fatalf("scope name length %d exceeds the unit bound", len(long))
	}
}

func TestCompanionRecoveryRequiresExactCommandAndCurrentParent(t *testing.T) {
	bin := "/operator/bin/resource-fixture"
	for _, tc := range []struct {
		name, executable, command string
		args                      []string
		want                      bool
	}{
		{"exact", bin, bin + " activity-edge --parent-pid 42", []string{"activity-edge"}, true},
		{"atomic replacement", bin + " (deleted)", bin + " activity-edge --parent-pid 42", []string{"activity-edge"}, true},
		{"same name elsewhere", "/other/resource-fixture", bin + " activity-edge --parent-pid 42", []string{"activity-edge"}, false},
		{"old parent", bin, bin + " activity-edge --parent-pid 41", []string{"activity-edge"}, false},
		{"other command", bin, bin + " capacity-sync --parent-pid 42", []string{"activity-edge"}, false},
		{"extra arguments", bin, bin + " activity-edge --parent-pid 42 --other", []string{"activity-edge"}, false},
		{"ambiguous argv projection", bin, bin + " two words --parent-pid 42", []string{"two words"}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := companionCommandMatches(bin, tc.args, 42, tc.executable, tc.command); got != tc.want {
				t.Fatalf("identity match=%t, want %t", got, tc.want)
			}
		})
	}
}

func TestManagedServiceSupervisorPlacesStartedProcess(t *testing.T) {
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
	var placed int
	supervisor.place = func(pid int) error {
		placed = pid
		return nil
	}
	state, err := supervisor.Start(artifactPath, artifact, []string{"-test.run=TestManagedServiceFixtureProcess", "--"}, append(os.Environ(), "VROOLI_MANAGED_SERVICE_FIXTURE=1"), dir, nil)
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer func() { _ = supervisor.forceStop(state.PID) }()
	if placed != state.PID {
		t.Fatalf("place saw pid %d, want started pid %d", placed, state.PID)
	}
}

func TestManagedServiceSupervisorPlacementFailureIsNotFatal(t *testing.T) {
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
	logPath := filepath.Join(dir, "service.log")
	supervisor := newManagedServiceSupervisor(filepath.Join(dir, "state.json"), logPath)
	supervisor.place = func(int) error { return fmt.Errorf("no manager bus") }
	state, err := supervisor.Start(artifactPath, artifact, []string{"-test.run=TestManagedServiceFixtureProcess", "--"}, append(os.Environ(), "VROOLI_MANAGED_SERVICE_FIXTURE=1"), dir, nil)
	if err != nil {
		t.Fatalf("Start() error = %v, want placement failure to stay non-fatal", err)
	}
	defer func() { _ = supervisor.forceStop(state.PID) }()
	log, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(log), "not placed in its own scope") {
		t.Fatalf("service log %q does not record the placement warning", string(log))
	}
}
