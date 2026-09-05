package resources

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"os/signal"
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
	state, err := supervisor.Start(artifactPath, artifact, []string{"-test.run=TestManagedServiceFixtureProcess", "--"}, append(os.Environ(), "VROOLI_MANAGED_SERVICE_FIXTURE=1"), dir, nil)
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
	restarted, err := supervisor.Start(artifactPath, artifact, []string{"-test.run=TestManagedServiceFixtureProcess", "--"}, append(os.Environ(), "VROOLI_MANAGED_SERVICE_FIXTURE=1"), dir, nil)
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

func TestManagedServiceSupervisorDerivesIdentityForHostToolArtifact(t *testing.T) {
	if testing.Short() {
		t.Skip("uses a helper process")
	}
	dir := t.TempDir()
	artifactPath := os.Args[0]
	artifact := resourcedeployment.ServiceArtifact{
		Path:         "fixture",
		Version:      "1.0.0",
		Verification: "host-tool",
	}
	supervisor := newManagedServiceSupervisor(filepath.Join(dir, "state.json"), filepath.Join(dir, "service.log"))
	state, err := supervisor.Start(artifactPath, artifact, []string{"-test.run=TestManagedServiceFixtureProcess", "--"}, append(os.Environ(), "VROOLI_MANAGED_SERVICE_FIXTURE=1"), dir, nil)
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer func() { _ = supervisor.Stop(context.Background()) }()
	if len(state.ArtifactSHA256) != sha256.Size*2 {
		t.Fatalf("derived artifact identity = %q, want SHA-256 digest", state.ArtifactSHA256)
	}
	if _, running, err := supervisor.Status(); err != nil || !running {
		t.Fatalf("Status() = running=%t err=%v, want live host-tool process", running, err)
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
	if _, err := supervisor.Start(artifactPath, artifact, nil, nil, dir, nil); err == nil || !strings.Contains(err.Error(), "checksum mismatch") {
		t.Fatalf("Start() error = %v, want checksum mismatch", err)
	}
}

func TestManagedServiceSupervisorAppliesDeclaredProcessLimits(t *testing.T) {
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
	limits := &resourcedeployment.ProcessLimits{MemoryHighPercent: 60, MemoryMaxPercent: 70, OOMScoreAdjust: 500}
	state, err := supervisor.Start(artifactPath, artifact, []string{"-test.run=TestManagedServiceFixtureProcess", "--"}, append(os.Environ(), "VROOLI_MANAGED_SERVICE_FIXTURE=1"), dir, limits)
	if err != nil {
		t.Fatalf("Start() with process limits error = %v", err)
	}
	defer func() { _ = supervisor.Stop(context.Background()) }()
	// A declared memory percentage must NEVER become an RLIMIT_AS cap. RLIMIT_AS
	// bounds *virtual* address space, not resident memory, so it delivers none of
	// the declared host protection while breaking every workload that reserves
	// large virtual ranges. CUDA is the case that bit us: llama.cpp reserves a
	// 32 GiB VMM pool on first inference, which a 60%-of-RAM address-space cap
	// rejects with "CUDA error: out of memory" even on an otherwise idle GPU.
	got, err := readAddressSpaceLimit(state.PID)
	if err != nil {
		t.Fatalf("read RLIMIT_AS: %v", err)
	}
	self, err := readAddressSpaceLimit(os.Getpid())
	if err != nil {
		t.Fatalf("read our own RLIMIT_AS: %v", err)
	}
	if got != self {
		t.Fatalf("RLIMIT_AS = %+v, want it left inherited at %+v; a memory percentage must not cap address space", got, self)
	}
	value, err := os.ReadFile(filepath.Join("/proc", fmt.Sprint(state.PID), "oom_score_adj"))
	if err != nil {
		t.Fatalf("read oom_score_adj: %v", err)
	}
	if strings.TrimSpace(string(value)) != "500" {
		t.Fatalf("oom_score_adj = %q, want 500", strings.TrimSpace(string(value)))
	}
}

func TestManagedServiceSupervisorRejectsLivePIDWithoutOwnershipToken(t *testing.T) {
	dir := t.TempDir()
	supervisor := newManagedServiceSupervisor(filepath.Join(dir, "state.json"), filepath.Join(dir, "service.log"))
	if err := supervisor.writeState(ManagedServiceState{
		InstanceID:         "ms-test",
		OwnershipTokenHash: managedServiceTokenHash("wrong-token"),
		PID:                os.Getpid(),
		ArtifactSHA256:     strings.Repeat("a", 64),
		ArtifactVersion:    "1.0.0",
	}); err != nil {
		t.Fatal(err)
	}
	if _, running, err := supervisor.Status(); err == nil || running || !strings.Contains(err.Error(), "ownership token") {
		t.Fatalf("Status() = (running=%t, err=%v), want ownership denial", running, err)
	}
}

func TestManagedServiceSupervisorUsesExplicitInterruptShutdown(t *testing.T) {
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
	_, err = supervisor.Start(artifactPath, artifact, []string{"-test.run=TestManagedServiceFixtureProcess", "--"}, append(os.Environ(), "VROOLI_MANAGED_SERVICE_FIXTURE=1", "VROOLI_MANAGED_SERVICE_FIXTURE_INTERRUPT=1"), dir, nil)
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := supervisor.StopWithSignal(ctx, os.Interrupt); err != nil {
		t.Fatalf("StopWithSignal() error = %v", err)
	}
	if _, running, err := supervisor.Status(); err != nil || running {
		t.Fatalf("Status() after interrupt = (running=%t, err=%v), want stopped", running, err)
	}
}

func TestManagedServiceSupervisorEscalatesAfterShutdownDeadline(t *testing.T) {
	dir := t.TempDir()
	token := "supervisor-escalation-token"
	t.Setenv(managedServiceOwnershipTokenEnv, token)
	alive := true
	forced := false
	supervisor := newManagedServiceSupervisor(filepath.Join(dir, "state.json"), filepath.Join(dir, "service.log"))
	supervisor.isRunning = func(int) bool { return alive }
	supervisor.signal = func(int, os.Signal) error { return nil }
	supervisor.terminate = func(int) error { return nil }
	supervisor.forceStop = func(int) error {
		forced = true
		alive = false
		return nil
	}
	if err := supervisor.writeState(ManagedServiceState{
		InstanceID:         "escalation",
		OwnershipTokenHash: managedServiceTokenHash(token),
		PID:                os.Getpid(),
		ArtifactPath:       os.Args[0],
		ArtifactSHA256:     strings.Repeat("a", 64),
		ArtifactVersion:    "1.0.0",
	}); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
	defer cancel()
	if err := supervisor.StopWithSignal(ctx, os.Interrupt); err != nil {
		t.Fatalf("StopWithSignal() error = %v", err)
	}
	if !forced {
		t.Fatal("shutdown deadline did not trigger forceful escalation")
	}
	state, running, err := supervisor.Status()
	if err != nil || running || state.PID != 0 {
		t.Fatalf("Status() after escalation = (%+v, running=%t, err=%v), want stopped record", state, running, err)
	}
}

func TestManagedServiceFixtureProcess(t *testing.T) {
	if os.Getenv("VROOLI_MANAGED_SERVICE_FIXTURE") != "1" {
		return
	}
	if os.Getenv("VROOLI_MANAGED_SERVICE_FIXTURE_INTERRUPT") == "1" {
		interrupts := make(chan os.Signal, 1)
		signal.Notify(interrupts, os.Interrupt)
		defer signal.Stop(interrupts)
		for {
			select {
			case <-interrupts:
				return
			default:
				time.Sleep(10 * time.Millisecond)
			}
		}
	}
	for {
		time.Sleep(time.Second)
	}
}
