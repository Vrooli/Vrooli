package resources

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"maps"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/binaryfetch"
	resourcedeployment "github.com/vrooli/vrooli/packages/resource-deployment"
)

type retentionFixture struct {
	root, artifacts, name string
	controller            *Controller
	supervisor            *ManagedServiceSupervisor
}

// newRetentionFixture installs a manifest pinned to "active" and one version
// directory per entry of versions; later entries are installed more recently.
func newRetentionFixture(t *testing.T, versions ...string) retentionFixture {
	t.Helper()
	root := t.TempDir()
	artifacts := filepath.Join(root, "artifacts")
	t.Setenv("VROOLI_RESOURCE_ARTIFACT_DIR", artifacts)
	t.Setenv("VROOLI_RESOURCE_STORAGE_ROOT", t.TempDir())
	manifest := managedServiceTestManifest(strings.Repeat("a", 64))
	manifest.Name = "retention-fixture"
	manifest.ManagedService.Artifact.Version = "active"
	manifest.ManagedService.ProviderPolicy.AllowedModes = []resourcedeployment.ProviderMode{resourcedeployment.ProviderManagedPrivate}
	manifest.ManagedService.ProviderPolicy.ExternalAccessCapabilities = nil
	manifest.ManagedService.Acquisition = &binaryfetch.Acquisition{
		Kind:    "url",
		Targets: []binaryfetch.AcquisitionTarget{{URL: "https://example.invalid/service", SHA256: strings.Repeat("b", 64)}},
	}
	manifestDir := filepath.Join(root, "resources", manifest.Name)
	if err := os.MkdirAll(manifestDir, 0o700); err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(manifestDir, "resource.json"), data, 0o600); err != nil {
		t.Fatal(err)
	}
	base := time.Now().Add(-time.Duration(len(versions)+1) * time.Hour)
	for i, version := range versions {
		dir := filepath.Join(artifacts, manifest.Name, version)
		if err := os.MkdirAll(filepath.Join(dir, "server"), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "server", "entry"), []byte(version), 0o700); err != nil {
			t.Fatal(err)
		}
		installed := base.Add(time.Duration(i) * time.Hour)
		if err := os.Chtimes(dir, installed, installed); err != nil {
			t.Fatal(err)
		}
	}
	supervisor, _, err := managedServiceSupervisorFor(manifest.Name)
	if err != nil {
		t.Fatal(err)
	}
	previous := artifactVersionsInUse
	artifactVersionsInUse = func(string) map[string]bool { return nil }
	t.Cleanup(func() { artifactVersionsInUse = previous })
	return retentionFixture{root: root, artifacts: artifacts, name: manifest.Name, controller: NewController(root, filepath.Join(root, "home")), supervisor: supervisor}
}

// supervise records a supervised instance of version; pid 0 means stopped.
func (f retentionFixture) supervise(t *testing.T, version, previous string, pid int) {
	t.Helper()
	state := ownershipState()
	state.InstanceID, state.ArtifactSHA256 = "fixture-instance", strings.Repeat("a", 64)
	state.ArtifactVersion, state.PreviousArtifactVersion, state.PID = version, previous, pid
	withOwnershipFacts(t, func(int) (map[string]string, error) {
		return map[string]string{managedServiceOwnershipTokenEnv: "owned"}, nil
	}, executableAt(state.ArtifactPath))
	if err := f.supervisor.writeState(state); err != nil {
		t.Fatal(err)
	}
}

func (f retentionFixture) present(version string) bool {
	_, err := os.Stat(filepath.Join(f.artifacts, f.name, version, "server", "entry"))
	return err == nil
}

func decisionsByVersion(plan ArtifactRetentionPlan) map[string]string {
	out := map[string]string{}
	for _, decision := range plan.Versions {
		out[decision.Version] = decision.Reason
	}
	return out
}

func TestArtifactRetentionDryRunDeletesNothing(t *testing.T) {
	f := newRetentionFixture(t, "v1", "v2", "v3", "active")
	plan, err := f.controller.PruneManagedServiceArtifacts(f.name, ArtifactRetentionOptions{KeepPrevious: 1})
	if err != nil {
		t.Fatal(err)
	}
	if plan.Applied || plan.RemovedBytes != 0 {
		t.Fatalf("dry run reported applied removal: %+v", plan)
	}
	want := map[string]string{"active": ArtifactRetainActive, "v3": ArtifactRetainRollback, "v2": ArtifactRetainRemove, "v1": ArtifactRetainRemove}
	if got := decisionsByVersion(plan); !maps.Equal(got, want) {
		t.Fatalf("decisions = %v, want %v", got, want)
	}
	if plan.ReclaimableBytes <= 0 || plan.ReclaimableBytes != removedVersionBytes(plan) {
		t.Fatalf("reclaimable = %d", plan.ReclaimableBytes)
	}
	for _, version := range []string{"v1", "v2", "v3", "active"} {
		if !f.present(version) {
			t.Fatalf("dry run removed %s", version)
		}
	}
}

func TestArtifactRetentionApplyKeepsActiveRunningInUseAndRollback(t *testing.T) {
	f := newRetentionFixture(t, "v1", "probed", "v3", "running", "v5", "active")
	// The supervised instance still runs an older version the manifest has
	// moved past, and an unsupervised process executes from "probed".
	f.supervise(t, "running", "v3", os.Getpid())
	artifactVersionsInUse = func(string) map[string]bool { return map[string]bool{"probed": true} }

	plan, err := f.controller.PruneManagedServiceArtifacts(f.name, ArtifactRetentionOptions{KeepPrevious: 1, Apply: true})
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{
		"active":  ArtifactRetainActive,
		"running": ArtifactRetainRunning,
		"probed":  ArtifactRetainInUse,
		// The supervisor's recorded previous version outranks a newer install.
		"v3": ArtifactRetainRollback,
		"v5": ArtifactRetainRemove,
		"v1": ArtifactRetainRemove,
	}
	if got := decisionsByVersion(plan); !maps.Equal(got, want) {
		t.Fatalf("decisions = %v, want %v", got, want)
	}
	for version, reason := range want {
		if kept := f.present(version); kept != (reason != ArtifactRetainRemove) {
			t.Fatalf("version %s present=%v, reason %s", version, kept, reason)
		}
	}
	if plan.RemovedBytes <= 0 || plan.RemovedBytes != removedVersionBytes(plan) {
		t.Fatalf("removed bytes = %d", plan.RemovedBytes)
	}
}

func TestArtifactRetentionKeepsNMostRecentPrevious(t *testing.T) {
	for _, tc := range []struct {
		keep int
		kept []string
	}{
		{keep: 0, kept: []string{"active"}},
		{keep: 2, kept: []string{"active", "v4", "v3"}},
		{keep: 10, kept: []string{"active", "v4", "v3", "v2", "v1"}},
	} {
		f := newRetentionFixture(t, "v1", "v2", "v3", "v4", "active")
		if _, err := f.controller.PruneManagedServiceArtifacts(f.name, ArtifactRetentionOptions{KeepPrevious: tc.keep, Apply: true}); err != nil {
			t.Fatal(err)
		}
		for _, version := range []string{"v1", "v2", "v3", "v4", "active"} {
			want := false
			for _, kept := range tc.kept {
				want = want || kept == version
			}
			if f.present(version) != want {
				t.Fatalf("keep=%d: version %s present=%v, want %v", tc.keep, version, !want, want)
			}
		}
	}
}

func retentionUpdateFixture(t *testing.T, f retentionFixture) (*managedUpdateStore, ManagedServiceUpdate) {
	t.Helper()
	u := updateTestInitial(t)
	u.Resource = f.name
	for _, c := range []*ManagedUpdateContract{&u.Target, &u.Rollback} {
		c.Manifest.Name, c.Artifact.Resource = f.name, f.name
	}
	for version, c := range map[string]*ManagedUpdateContract{"target": &u.Target, "retained": &u.Rollback} {
		c.Manifest.ManagedService.Artifact.Version = version
		c.Artifact.Version = version
		c.Artifact.Path = filepath.Join(f.artifacts, f.name, version, "server", "entry")
		c.Digest = c.digest()
	}
	u.Original.ArtifactPath = u.Rollback.Artifact.Path
	u.Original.ArtifactVersion = u.Rollback.Artifact.Version
	s, err := newManagedUpdateStore(f.name)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.Create(context.Background(), u, false); err != nil {
		t.Fatal(err)
	}
	return s, u
}

func TestArtifactRetentionKeepsActiveUpdateArtifactsEvenWithoutLiveProcesses(t *testing.T) {
	f := newRetentionFixture(t, "retained", "target", "unrelated", "active")
	retentionUpdateFixture(t, f)
	plan, err := f.controller.PruneManagedServiceArtifacts(f.name, ArtifactRetentionOptions{KeepPrevious: 0, Apply: true})
	if err != nil {
		t.Fatal(err)
	}
	for _, version := range []string{"retained", "target"} {
		if !f.present(version) || decisionsByVersion(plan)[version] == ArtifactRetainRemove {
			t.Fatalf("active update lost its %s artifact during caller-death recovery", version)
		}
	}
	if f.present("unrelated") {
		t.Fatal("update pin disabled unrelated artifact retention")
	}
}

func TestArtifactRetentionRefusesCorruptUpdateInventoryWithoutDeletingAnything(t *testing.T) {
	f := newRetentionFixture(t, "retained", "target", "unrelated", "active")
	s, _ := retentionUpdateFixture(t, f)
	if err := os.WriteFile(s.path, []byte(`{"schema_version":1,"records":`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := f.controller.PruneManagedServiceArtifacts(f.name, ArtifactRetentionOptions{KeepPrevious: 0, Apply: true}); err == nil {
		t.Fatal("unknown update inventory was treated as no active updates")
	}
	for _, version := range []string{"retained", "target", "unrelated", "active"} {
		if !f.present(version) {
			t.Fatalf("corrupt update inventory permitted deleting %s", version)
		}
	}
}

func TestManagedUpdateInterlocksOrdinaryStopAndUninstall(t *testing.T) {
	for _, action := range []string{"stop", "uninstall"} {
		t.Run(action, func(t *testing.T) {
			f := newRetentionFixture(t, "retained", "target", "active")
			_, u := retentionUpdateFixture(t, f)
			m, err := f.controller.LoadManifest(filepath.Join(f.root, "resources", f.name, "resource.json"))
			if err != nil {
				t.Fatal(err)
			}
			err = (managedServiceDriver{}).Run(context.Background(), f.controller, Resource{Name: f.name}, m, action, nil, io.Discard, io.Discard)
			if !errors.Is(err, ErrManagedUpdateConflict) || !strings.Contains(err.Error(), u.ID) {
				t.Fatalf("ordinary %s bypassed active update ownership: %v", action, err)
			}
		})
	}
}

func TestManagedUpdateInterlocksSharedBrokerOwnerLifecycle(t *testing.T) {
	f := newRetentionFixture(t, "retained", "target", "active")
	_, u := retentionUpdateFixture(t, f)
	m, err := f.controller.LoadManifest(filepath.Join(f.root, "resources", f.name, "resource.json"))
	if err != nil {
		t.Fatal(err)
	}
	owner, err := NewManagedServiceOwnerLifecycle(f.controller, m)
	if err != nil {
		t.Fatal(err)
	}
	for _, action := range []string{"start", "restart", "stop"} {
		_, err := owner.Manage(context.Background(), ManagedInstance{Resource: f.name, Provider: resourcedeployment.ProviderManagedShared}, action)
		if !errors.Is(err, ErrManagedUpdateConflict) || !strings.Contains(err.Error(), u.ID) {
			t.Fatalf("shared broker %s bypassed active update: %v", action, err)
		}
	}
}

func TestManagedUpdateLifecycleFenceSurvivesUnknownInventoryAndReleasesAfterTerminalReceipt(t *testing.T) {
	f := newRetentionFixture(t, "retained", "target", "active")
	s, u := retentionUpdateFixture(t, f)
	last := updateStoreTrace(t, s, u, updateSuccessTrace())
	if !last.Phase.Terminal() {
		t.Fatal("fixture did not complete promotion")
	}
	if err := ensureManagedUpdateLifecycleAvailable(f.name, "stop"); err != nil {
		t.Fatalf("terminal serving receipt retained an active operation fence: %v", err)
	}
	if err := os.WriteFile(s.path, []byte(`{"schema_version":1,"records":`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := ensureManagedUpdateLifecycleAvailable(f.name, "stop"); err == nil || !strings.Contains(err.Error(), "read managed update ownership") {
		t.Fatalf("unknown inventory released the lifecycle fence: %v", err)
	}
}

func TestArtifactRetentionAfterActivationOnlyOnVersionChange(t *testing.T) {
	f := newRetentionFixture(t, "v1", "v2", "v3", "active")
	manifest, err := f.controller.LoadManifest(filepath.Join(f.root, "resources", f.name, "resource.json"))
	if err != nil {
		t.Fatal(err)
	}
	// A restart of the same version must never prune.
	f.supervise(t, "active", "active", os.Getpid())
	retainManagedServiceArtifactsAfterActivation(context.Background(), f.controller, manifest, f.supervisor)
	for _, version := range []string{"v1", "v2", "v3", "active"} {
		if !f.present(version) {
			t.Fatalf("same-version restart removed %s", version)
		}
	}
	// Activating a new version keeps it plus the replaced one.
	f.supervise(t, "active", "v2", os.Getpid())
	retainManagedServiceArtifactsAfterActivation(context.Background(), f.controller, manifest, f.supervisor)
	for version, want := range map[string]bool{"active": true, "v2": true, "v3": false, "v1": false} {
		if f.present(version) != want {
			t.Fatalf("after activation version %s present=%v, want %v", version, !want, want)
		}
	}
}

func TestArtifactRetentionCountsHardlinkedBytesOnlyWhenEveryLinkIsRemoved(t *testing.T) {
	f := newRetentionFixture(t, "v1", "v2", "active")
	shared := filepath.Join(f.artifacts, f.name, "v1", "server", "shared")
	if err := os.WriteFile(shared, make([]byte, 1<<20), 0o600); err != nil {
		t.Fatal(err)
	}
	// Composed versions share cached files through hardlinks: the same bytes
	// live in a removed and a kept version, so removing v1 frees none of them.
	if err := os.Link(shared, filepath.Join(f.artifacts, f.name, "active", "server", "shared")); err != nil {
		t.Skipf("hardlinks unsupported: %v", err)
	}
	plan, err := f.controller.PruneManagedServiceArtifacts(f.name, ArtifactRetentionOptions{KeepPrevious: 0})
	if err != nil {
		t.Fatal(err)
	}
	if plan.ReclaimableBytes >= 1<<20 {
		t.Fatalf("reclaimable = %d counts bytes still linked from the active version", plan.ReclaimableBytes)
	}
	if v1 := decisionsByVersion(plan)["v1"]; v1 != ArtifactRetainRemove {
		t.Fatalf("v1 decision = %s", v1)
	}
	for _, decision := range plan.Versions {
		if decision.Version == "v1" && decision.Bytes < 1<<20 {
			t.Fatalf("v1 size = %d omits its shared file", decision.Bytes)
		}
	}
}

// removedVersionBytes is the reclaim a fixture without shared files expects.
func removedVersionBytes(plan ArtifactRetentionPlan) int64 {
	var total int64
	for _, decision := range plan.Versions {
		if !decision.Keep {
			total += decision.Bytes
		}
	}
	return total
}

func TestScanArtifactVersionsInUseFindsOwnWorkingDirectory(t *testing.T) {
	if _, err := os.Stat("/proc/self/cwd"); err != nil {
		t.Skip("no /proc")
	}
	root := t.TempDir()
	dir := filepath.Join(root, "v9", "server")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)
	if !scanArtifactVersionsInUse(root)["v9"] {
		t.Fatal("process working in a version directory was not reported in use")
	}
}
