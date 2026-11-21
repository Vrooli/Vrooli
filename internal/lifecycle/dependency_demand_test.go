package lifecycle

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/hostsession"
	"github.com/vrooli/vrooli/internal/process"
	"github.com/vrooli/vrooli/internal/runtimesupervisor"
	"github.com/vrooli/vrooli/internal/scenario"
	"github.com/vrooli/vrooli/internal/scenarioruntime"
)

func TestStartScenarioStaleReplacementRetainsCurrentAndReleasesRemovedDemand(t *testing.T) {
	ctx := context.Background()
	root, home := t.TempDir(), t.TempDir()
	writeLifecycleFixtureManifest(t, root, scenario.ServiceManifest{
		Service: scenario.ServiceMetadata{Name: "parent"},
		Dependencies: scenario.Dependencies{Scenarios: map[string]scenario.Dependency{
			"current-dep": {Enabled: true, Required: true},
		}},
	})
	writeLifecycleFixtureManifest(t, root, scenario.ServiceManifest{Service: scenario.ServiceMetadata{Name: "current-dep"}})
	writeLifecycleFixtureManifest(t, root, scenario.ServiceManifest{Service: scenario.ServiceMetadata{Name: "removed-dep"}})

	store, err := scenarioruntime.NewSQLiteStore(ctx, scenarioruntime.Config{HomeDir: home})
	if err != nil {
		t.Fatalf("NewSQLiteStore: %v", err)
	}
	seedRunningLease(t, ctx, store, "parent", "parent-old", "old-build", scenarioruntime.SupervisionPolicyManaged)
	seedRunningLease(t, ctx, store, "current-dep", "current-dep-live", "", scenarioruntime.SupervisionPolicyDemand)
	seedRunningLease(t, ctx, store, "removed-dep", "removed-dep-live", "", scenarioruntime.SupervisionPolicyDemand)
	consumerID := scenarioruntime.DependencyConsumerID("parent", "live")
	currentID := seedDependencyDemand(t, ctx, store, "parent", "current-dep")
	removedID := seedDependencyDemand(t, ctx, store, "parent", "removed-dep")
	if err := store.Close(); err != nil {
		t.Fatalf("Close seed registry: %v", err)
	}

	t.Setenv(runtimesupervisor.ModeEnv, runtimesupervisor.ModeOff)
	runner := newA38LifecycleRunner(t, root, home)
	if _, err := runner.StartContext(ctx, "parent", StartOptions{}); err != nil {
		t.Fatalf("StartContext stale replacement: %v", err)
	}

	store, err = scenarioruntime.NewSQLiteStore(ctx, scenarioruntime.Config{HomeDir: home})
	if err != nil {
		t.Fatalf("reopen registry: %v", err)
	}
	defer store.Close()
	assertDemandLeaseStatus(t, ctx, home, currentID, scenarioruntime.DemandLeaseActive)
	assertDemandLeaseStatus(t, ctx, home, removedID, scenarioruntime.DemandLeaseReleased)
	claimed, err := store.ClaimDemandStopCandidates(ctx, time.Now(), "test expired demand")
	if err != nil {
		t.Fatalf("ClaimDemandStopCandidates: %v", err)
	}
	selected := make(map[string]bool, len(claimed))
	for _, instance := range claimed {
		selected[instance.Scenario] = true
	}
	if selected["current-dep"] || !selected["removed-dep"] {
		t.Fatalf("demand-stop selection after actual stale replacement = %v; current must be held, removed must be eligible (consumer %s)", selected, consumerID)
	}
}

func TestStartScenarioEmptyDependencyGraphReleasesPriorDemand(t *testing.T) {
	ctx := context.Background()
	root, home := t.TempDir(), t.TempDir()
	writeLifecycleFixtureManifest(t, root, scenario.ServiceManifest{Service: scenario.ServiceMetadata{Name: "parent"}})
	writeLifecycleFixtureManifest(t, root, scenario.ServiceManifest{Service: scenario.ServiceMetadata{Name: "removed-dep"}})
	store, err := scenarioruntime.NewSQLiteStore(ctx, scenarioruntime.Config{HomeDir: home})
	if err != nil {
		t.Fatalf("NewSQLiteStore: %v", err)
	}
	seedRunningLease(t, ctx, store, "parent", "parent-old", "old-build", scenarioruntime.SupervisionPolicyManaged)
	seedRunningLease(t, ctx, store, "removed-dep", "removed-dep-live", "", scenarioruntime.SupervisionPolicyDemand)
	removedID := seedDependencyDemand(t, ctx, store, "parent", "removed-dep")
	if err := store.Close(); err != nil {
		t.Fatalf("Close seed registry: %v", err)
	}

	t.Setenv(runtimesupervisor.ModeEnv, runtimesupervisor.ModeOff)
	runner := newA38LifecycleRunner(t, root, home)
	if _, err := runner.StartContext(ctx, "parent", StartOptions{}); err != nil {
		t.Fatalf("StartContext stale replacement with empty dependency graph: %v", err)
	}
	assertDemandLeaseStatus(t, ctx, home, removedID, scenarioruntime.DemandLeaseReleased)
}

func TestStartScenarioDependencyFailureBeforeParentStopPreservesPriorHold(t *testing.T) {
	ctx := context.Background()
	root, home := t.TempDir(), t.TempDir()
	writeLifecycleFixtureManifest(t, root, scenario.ServiceManifest{
		Service: scenario.ServiceMetadata{Name: "parent"},
		Dependencies: scenario.Dependencies{Scenarios: map[string]scenario.Dependency{
			"new-dep": {Enabled: true, Required: true},
		}},
	})
	writeLifecycleFixtureManifest(t, root, scenario.ServiceManifest{
		Service: scenario.ServiceMetadata{Name: "new-dep"},
		Lifecycle: scenario.Lifecycle{Setup: scenario.Phase{Steps: []scenario.PhaseStep{{
			Name: "injected-setup-failure", Exec: []string{"bash", "-c", "exit 23"},
		}}}},
	})
	store, err := scenarioruntime.NewSQLiteStore(ctx, scenarioruntime.Config{HomeDir: home})
	if err != nil {
		t.Fatalf("NewSQLiteStore: %v", err)
	}
	_, err = store.CreateLease(ctx, scenarioruntime.Instance{
		InstanceID: "parent-old", Scenario: "parent", Variant: "live", Status: scenarioruntime.StatusRunning,
		BuildIdentity: "old-build", HostBootID: "boot-test", SupervisionPolicy: scenarioruntime.SupervisionPolicyManaged,
	}, scenarioruntime.DefaultHeartbeatTTL)
	if err != nil {
		t.Fatalf("CreateLease(parent): %v", err)
	}
	priorID := seedDependencyDemand(t, ctx, store, "parent", "new-dep")
	if err := store.Close(); err != nil {
		t.Fatalf("Close seed registry: %v", err)
	}

	t.Setenv(runtimesupervisor.ModeEnv, runtimesupervisor.ModeOff)
	runner := newA38LifecycleRunner(t, root, home)
	_, err = runner.StartContext(ctx, "parent", StartOptions{ForceSetup: true})
	if err == nil {
		t.Fatal("replacement should fail while preparing its required dependency")
	}
	assertDemandLeaseStatus(t, ctx, home, priorID, scenarioruntime.DemandLeaseActive)
	store, err = scenarioruntime.NewSQLiteStore(ctx, scenarioruntime.Config{HomeDir: home})
	if err != nil {
		t.Fatalf("reopen registry: %v", err)
	}
	defer store.Close()
	instances, err := store.ListInstances(ctx, scenarioruntime.InstanceFilter{Scenario: "parent", Variant: "live", Statuses: scenarioruntime.ActiveInstanceStatuses()})
	if err != nil || len(instances) != 1 || instances[0].InstanceID != "parent-old" {
		t.Fatalf("old parent must remain active after candidate dependency failure: instances=%+v err=%v", instances, err)
	}
}

func TestStartScenarioDiamondRecordsDemandForEachReadyConsumer(t *testing.T) {
	ctx := context.Background()
	root, home := t.TempDir(), t.TempDir()
	writeLifecycleFixtureManifest(t, root, scenario.ServiceManifest{
		Service: scenario.ServiceMetadata{Name: "parent"},
		Dependencies: scenario.Dependencies{Scenarios: map[string]scenario.Dependency{
			"branch-b": {Enabled: true, Required: true},
			"branch-c": {Enabled: true, Required: true},
		}},
	})
	writeLifecycleFixtureManifest(t, root, scenario.ServiceManifest{
		Service: scenario.ServiceMetadata{Name: "branch-b"},
		Dependencies: scenario.Dependencies{Scenarios: map[string]scenario.Dependency{
			"shared-d": {Enabled: true, Required: true},
		}},
	})
	writeLifecycleFixtureManifest(t, root, scenario.ServiceManifest{
		Service: scenario.ServiceMetadata{Name: "branch-c"},
		Dependencies: scenario.Dependencies{Scenarios: map[string]scenario.Dependency{
			"shared-d": {Enabled: true, Required: true},
		}},
	})
	writeLifecycleFixtureManifest(t, root, scenario.ServiceManifest{Service: scenario.ServiceMetadata{Name: "shared-d"}})

	t.Setenv(runtimesupervisor.ModeEnv, runtimesupervisor.ModeOff)
	runner := newA38LifecycleRunner(t, root, home)
	if _, err := runner.StartContext(ctx, "parent", StartOptions{}); err != nil {
		t.Fatalf("StartContext diamond: %v", err)
	}
	bHold := scenarioruntime.DependencyLeaseID(scenarioruntime.DependencyConsumerID("branch-b", "live"), "shared-d", "live")
	cHold := scenarioruntime.DependencyLeaseID(scenarioruntime.DependencyConsumerID("branch-c", "live"), "shared-d", "live")
	assertDemandLeaseStatus(t, ctx, home, bHold, scenarioruntime.DemandLeaseActive)
	assertDemandLeaseStatus(t, ctx, home, cHold, scenarioruntime.DemandLeaseActive)

	if err := runner.StopContext(ctx, "branch-b", StopOptions{Variant: "live"}); err != nil {
		t.Fatalf("StopContext(branch-b): %v", err)
	}
	assertDemandLeaseStatus(t, ctx, home, bHold, scenarioruntime.DemandLeaseReleased)
	assertDemandLeaseStatus(t, ctx, home, cHold, scenarioruntime.DemandLeaseActive)
	store, err := scenarioruntime.NewSQLiteStore(ctx, scenarioruntime.Config{HomeDir: home})
	if err != nil {
		t.Fatalf("reopen registry: %v", err)
	}
	defer store.Close()
	claimed, err := store.ClaimDemandStopCandidates(ctx, time.Now(), "test surviving diamond demand")
	if err != nil {
		t.Fatalf("ClaimDemandStopCandidates: %v", err)
	}
	for _, instance := range claimed {
		if instance.Scenario == "shared-d" {
			t.Fatal("shared-d became eligible for demand stop after branch-b stopped while branch-c remained live")
		}
	}
}

func newA38LifecycleRunner(t *testing.T, root, home string) *Runner {
	t.Helper()
	return newLifecycleRunnerForTest(t, root, home, func(deps *lifecycleDeps) {
		deps.hostSession = func(context.Context, string) (hostsession.Snapshot, error) {
			return hostsession.Snapshot{BootID: "boot-test", SessionID: "session-test"}, nil
		}
		deps.readScenarioRecords = func(string, string) ([]process.Record, error) { return nil, nil }
		deps.isPIDRunning = func(int) bool { return true }
		deps.ensureRuntimeSupervisor = func(context.Context, string, io.Writer, io.Writer) error { return nil }
	})
}

func seedRunningLease(t *testing.T, ctx context.Context, store *scenarioruntime.SQLiteStore, name, instanceID, buildIdentity, policy string) {
	t.Helper()
	if _, err := store.CreateLease(ctx, scenarioruntime.Instance{
		InstanceID: instanceID, Scenario: name, Variant: "live", Status: scenarioruntime.StatusRunning,
		BuildIdentity: buildIdentity, HostBootID: "boot-test", SupervisionPolicy: policy,
	}, scenarioruntime.DefaultHeartbeatTTL); err != nil {
		t.Fatalf("CreateLease(%s): %v", name, err)
	}
}

func TestStaleReplacementKeepsCurrentDependencyDemandAndReleasesRemoved(t *testing.T) {
	ctx := context.Background()
	root, home := t.TempDir(), t.TempDir()
	writeLifecycleFixtureManifest(t, root, scenario.ServiceManifest{Service: scenario.ServiceMetadata{Name: "parent"}})
	store, err := scenarioruntime.NewSQLiteStore(ctx, scenarioruntime.Config{HomeDir: home})
	if err != nil {
		t.Fatalf("NewSQLiteStore: %v", err)
	}
	seedDemandConsumer(t, ctx, store, "parent", "parent-live", scenarioruntime.StatusRunning)
	seedDemandConsumer(t, ctx, store, "current-dep", "current-dep-live", scenarioruntime.StatusRunning)
	seedDemandConsumer(t, ctx, store, "removed-dep", "removed-dep-live", scenarioruntime.StatusRunning)
	currentID := seedDependencyDemand(t, ctx, store, "parent", "current-dep")
	removedID := seedDependencyDemand(t, ctx, store, "parent", "removed-dep")
	if err := store.Close(); err != nil {
		t.Fatalf("Close seed registry: %v", err)
	}

	runner := newLifecycleRunnerForTest(t, root, home, nil)
	consumer := scenario.Scenario{Slug: "parent", Variant: "live"}
	session := newStartSession(ctx)
	if err := runner.snapshotDependencyDemandLeases(ctx, consumer, session); err != nil {
		t.Fatalf("snapshotDependencyDemandLeases: %v", err)
	}
	consumerID := scenarioruntime.DependencyConsumerID(consumer.Slug, consumer.Variant)
	session.recordDependencyLease(consumerID, currentID)
	session.markDependencyConsumerStopped(consumerID)
	if err := runner.releaseDependencyLeasesForConsumerExcept(ctx, consumer.Slug, consumer.Variant, "scenario restart", session.dependencyLeaseSetForConsumer(consumerID)); err != nil {
		t.Fatalf("release removed dependency holds: %v", err)
	}

	store, err = scenarioruntime.NewSQLiteStore(ctx, scenarioruntime.Config{HomeDir: home})
	if err != nil {
		t.Fatalf("reopen registry: %v", err)
	}
	defer store.Close()
	leases, err := store.ListDemandLeases(ctx, scenarioruntime.DemandLeaseFilter{ConsumerID: consumerID})
	if err != nil {
		t.Fatalf("ListDemandLeases: %v", err)
	}
	statuses := make(map[string]string, len(leases))
	for _, lease := range leases {
		statuses[lease.LeaseID] = lease.Status
	}
	if statuses[currentID] != scenarioruntime.DemandLeaseActive {
		t.Fatalf("current graph dependency lease status = %q, want active", statuses[currentID])
	}
	if statuses[removedID] != scenarioruntime.DemandLeaseReleased {
		t.Fatalf("removed dependency lease status = %q, want released", statuses[removedID])
	}
	claimed, err := store.ClaimDemandStopCandidates(ctx, time.Now(), "test expired demand")
	if err != nil {
		t.Fatalf("ClaimDemandStopCandidates: %v", err)
	}
	selected := make(map[string]bool, len(claimed))
	for _, instance := range claimed {
		selected[instance.Scenario] = true
	}
	if selected["current-dep"] {
		t.Fatal("current graph dependency with a retained lease was selected for demand stop")
	}
	if !selected["removed-dep"] {
		t.Fatal("removed dependency with no remaining lease was not selected for demand stop")
	}
}

func TestFailedReplacementPreservesOldDemandUntilOldOwnerStops(t *testing.T) {
	ctx := context.Background()
	root, home := t.TempDir(), t.TempDir()
	writeLifecycleFixtureManifest(t, root, scenario.ServiceManifest{Service: scenario.ServiceMetadata{Name: "parent"}})
	store, err := scenarioruntime.NewSQLiteStore(ctx, scenarioruntime.Config{HomeDir: home})
	if err != nil {
		t.Fatalf("NewSQLiteStore: %v", err)
	}
	seedDemandConsumer(t, ctx, store, "parent", "parent-live", scenarioruntime.StatusRunning)
	seedDemandConsumer(t, ctx, store, "old-dep", "old-dep-live", scenarioruntime.StatusRunning)
	seedDemandConsumer(t, ctx, store, "new-dep", "new-dep-live", scenarioruntime.StatusRunning)
	oldID := seedDependencyDemand(t, ctx, store, "parent", "old-dep")
	if err := store.Close(); err != nil {
		t.Fatalf("Close seed registry: %v", err)
	}

	runner := newLifecycleRunnerForTest(t, root, home, nil)
	consumer := scenario.Scenario{Slug: "parent", Variant: "live"}
	consumerID := scenarioruntime.DependencyConsumerID(consumer.Slug, consumer.Variant)
	session := newStartSession(ctx)
	if err := runner.snapshotDependencyDemandLeases(ctx, consumer, session); err != nil {
		t.Fatalf("snapshotDependencyDemandLeases: %v", err)
	}
	store, err = scenarioruntime.NewSQLiteStore(ctx, scenarioruntime.Config{HomeDir: home})
	if err != nil {
		t.Fatalf("open registry for candidate lease: %v", err)
	}
	newID := seedDependencyDemand(t, ctx, store, "parent", "new-dep")
	if err := store.Close(); err != nil {
		t.Fatalf("Close candidate registry: %v", err)
	}
	session.recordDependencyLease(consumerID, oldID)
	session.recordDependencyLease(consumerID, newID)

	// A candidate dependency failure before stopping the old parent must leave
	// its reused stable hold active, but discard a newly introduced candidate.
	if err := runner.releaseDependencyDemandIfCandidate(ctx, session, consumer, oldID, "dependency start failed"); err != nil {
		t.Fatalf("releaseDependencyDemandIfCandidate(existing): %v", err)
	}
	if err := runner.releaseStartSessionDemand(ctx, session); err != nil {
		t.Fatalf("releaseStartSessionDemand before old stop: %v", err)
	}
	assertDemandLeaseStatus(t, ctx, home, oldID, scenarioruntime.DemandLeaseActive)
	assertDemandLeaseStatus(t, ctx, home, newID, scenarioruntime.DemandLeaseReleased)

	// Once the old owner is positively stopped, a failed replacement must
	// release the retained current hold too.
	session.markDependencyConsumerStopped(consumerID)
	if err := runner.releaseStartSessionDemand(ctx, session); err != nil {
		t.Fatalf("releaseStartSessionDemand after old stop: %v", err)
	}
	assertDemandLeaseStatus(t, ctx, home, oldID, scenarioruntime.DemandLeaseReleased)
}

func seedDemandConsumer(t *testing.T, ctx context.Context, store *scenarioruntime.SQLiteStore, name, instanceID, status string) {
	t.Helper()
	if _, err := store.CreateInstance(ctx, scenarioruntime.Instance{
		InstanceID:        instanceID,
		Scenario:          name,
		Variant:           "live",
		Status:            status,
		SupervisionPolicy: scenarioruntime.SupervisionPolicyDemand,
	}); err != nil {
		t.Fatalf("CreateInstance(%s): %v", name, err)
	}
}

func seedDependencyDemand(t *testing.T, ctx context.Context, store *scenarioruntime.SQLiteStore, consumer, dependency string) string {
	t.Helper()
	consumerID := scenarioruntime.DependencyConsumerID(consumer, "live")
	leaseID := scenarioruntime.DependencyLeaseID(consumerID, dependency, "live")
	if _, err := store.AcquireDemandLease(ctx, scenarioruntime.DemandLease{
		LeaseID:    leaseID,
		Scenario:   dependency,
		Variant:    "live",
		ConsumerID: consumerID,
		Kind:       scenarioruntime.DemandLeaseDependency,
	}, scenarioruntime.DefaultDemandLeaseTTL); err != nil {
		t.Fatalf("AcquireDemandLease(%s): %v", leaseID, err)
	}
	return leaseID
}

func assertDemandLeaseStatus(t *testing.T, ctx context.Context, home, leaseID, want string) {
	t.Helper()
	store, err := scenarioruntime.NewSQLiteStore(ctx, scenarioruntime.Config{HomeDir: home})
	if err != nil {
		t.Fatalf("open registry for lease %s: %v", leaseID, err)
	}
	defer store.Close()
	leases, err := store.ListDemandLeases(ctx, scenarioruntime.DemandLeaseFilter{})
	if err != nil {
		t.Fatalf("ListDemandLeases(%s): %v", leaseID, err)
	}
	for _, lease := range leases {
		if lease.LeaseID == leaseID {
			if lease.Status != want {
				t.Fatalf("lease %s status = %q, want %q", leaseID, lease.Status, want)
			}
			return
		}
	}
	t.Fatalf("lease %s not found", leaseID)
}
