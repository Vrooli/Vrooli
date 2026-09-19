package providers

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"storage-manager/internal/cleanup"
	cleanupfakes "storage-manager/internal/testutil/cleanup"
)

type fakeInstanceRegistry struct {
	running map[string]struct{}
	err     error
	calls   int
}

func (r *fakeInstanceRegistry) RunningInstances(context.Context) (map[string]struct{}, error) {
	r.calls++
	if r.err != nil {
		return nil, r.err
	}
	return r.running, nil
}

var instanceNow = time.Date(2026, 9, 17, 12, 0, 0, 0, time.UTC)

// instanceFixture builds a host shaped like the real one: two class roots that
// hold live instance directories, non-live instance directories, and the host
// safeguard directories that share the "<word>_<word>" shape without belonging
// to any scenario.
func instanceFixture(t *testing.T, running map[string]struct{}) (*OrphanedInstanceStorageProvider, *cleanupfakes.FileSystem, *fakeInstanceRegistry, string) {
	t.Helper()
	root := t.TempDir()
	dataRoot := filepath.Join(root, "data", "vrooli")
	stateRoot := filepath.Join(root, "state", "vrooli")
	quarantine := filepath.Join(root, "state", "storage-manager", "quarantine", "instances")

	old := instanceNow.Add(-72 * time.Hour)
	files := map[string]cleanup.FileInfo{
		dataRoot:  {Path: dataRoot, IsDir: true},
		stateRoot: {Path: stateRoot, IsDir: true},
	}
	addDir := func(parent, name string, size int64) {
		dir := filepath.Join(parent, name)
		files[dir] = cleanup.FileInfo{Path: dir, IsDir: true, ModTime: old}
		blob := filepath.Join(dir, "db.sqlite")
		files[blob] = cleanup.FileInfo{Path: blob, Size: size, ModTime: old}
	}
	// Stopped non-live instance: the case this provider exists for.
	addDir(dataRoot, "web-console_presentation", 4096)
	addDir(stateRoot, "web-console_presentation", 512)
	// Running non-live instance.
	addDir(dataRoot, "audio-tools_presentation", 8192)
	// Live instances. Their directories carry no variant suffix.
	addDir(dataRoot, "web-console", 1<<20)
	addDir(dataRoot, "audio-tools", 1<<20)
	// A host safeguard directory: the "<prefix>_<suffix>" shape with no
	// scenario behind the prefix.
	addDir(stateRoot, "agent_session_containment", 64)
	// A namespace whose scenario no longer exists in the repository.
	addDir(dataRoot, "deleted-scenario_shadow", 128)
	// "<scenario>_live" is not a shape the naming SSOT produces.
	addDir(dataRoot, "web-console_live", 256)

	fsys := &cleanupfakes.FileSystem{Root: root, Files: files, AllowRemove: true}
	registry := &fakeInstanceRegistry{running: running}
	provider := NewOrphanedInstanceStorageProvider(fsys, registry, cleanupfakes.Clock{Time: instanceNow}, OrphanedInstanceStorageProviderConfig{
		NamespaceRoots: []InstanceNamespaceRoot{
			{Class: "data", Path: dataRoot},
			{Class: "state", Path: stateRoot},
		},
		KnownScenarios: []string{"web-console", "audio-tools", "web", "git-control-tower"},
		Quarantine:     quarantine,
	})
	return provider, fsys, registry, quarantine
}

func instancePolicy() cleanup.ProviderPolicy {
	return cleanup.ProviderPolicy{Enabled: true, ApprovalMode: cleanup.ApprovalModeOwner}
}

func instancePreview(t *testing.T, provider *OrphanedInstanceStorageProvider) cleanup.Preview {
	t.Helper()
	preview, err := provider.Preview(context.Background(), cleanup.PreviewRequest{
		Scope:  cleanup.ObservationScope{Now: instanceNow},
		Policy: instancePolicy(),
	})
	if err != nil {
		t.Fatalf("Preview() error = %v", err)
	}
	return preview
}

func previewPaths(preview cleanup.Preview) []string {
	out := make([]string, 0, len(preview.Items))
	for _, item := range preview.Items {
		out = append(out, item.Path)
	}
	return out
}

func TestOrphanedInstanceStorageDetectsStoppedVariantAcrossClasses(t *testing.T) {
	provider, _, _, _ := instanceFixture(t, map[string]struct{}{"audio-tools@presentation": {}})
	preview := instancePreview(t, provider)
	paths := previewPaths(preview)

	found := 0
	for _, path := range paths {
		if filepath.Base(path) == "web-console_presentation" {
			found++
		}
	}
	if found != 2 {
		t.Fatalf("expected the stopped instance's data and state directories, got %v", paths)
	}
	if got := sumPreviewBytes(preview.Items); got != 4096+512 {
		t.Fatalf("expected the measured size of both directories, got %d", got)
	}
}

func TestOrphanedInstanceStorageNeverProposesRunningInstance(t *testing.T) {
	provider, _, _, _ := instanceFixture(t, map[string]struct{}{"audio-tools@presentation": {}})
	preview := instancePreview(t, provider)

	for _, path := range previewPaths(preview) {
		if filepath.Base(path) == "audio-tools_presentation" {
			t.Fatalf("running instance storage was proposed: %s", path)
		}
	}
	if !warningsContain(preview.Warnings, "audio-tools@presentation: instance is running") {
		t.Fatalf("expected a skip warning naming the running instance, got %v", preview.Warnings)
	}
}

func TestOrphanedInstanceStorageNeverProposesLiveInstance(t *testing.T) {
	// Nothing is running at all: the strongest condition under which a bug
	// would offer a live instance's directory.
	provider, _, _, _ := instanceFixture(t, map[string]struct{}{})
	preview := instancePreview(t, provider)

	for _, path := range previewPaths(preview) {
		switch filepath.Base(path) {
		case "web-console", "audio-tools":
			t.Fatalf("live instance storage was proposed: %s", path)
		case "web-console_live":
			t.Fatalf("a non-canonical \"_live\" directory was proposed: %s", path)
		}
	}
}

func TestOrphanedInstanceStorageRefusesUnattributableDirectories(t *testing.T) {
	provider, _, _, _ := instanceFixture(t, map[string]struct{}{})
	preview := instancePreview(t, provider)

	for _, path := range previewPaths(preview) {
		switch filepath.Base(path) {
		case "agent_session_containment":
			t.Fatalf("a host safeguard directory was proposed: %s", path)
		case "deleted-scenario_shadow":
			t.Fatalf("a directory with no known scenario was proposed: %s", path)
		}
	}
	// Exactly the three unattributable names, and not the two live instance
	// directories: a live namespace that fell into this bucket would mean the
	// live check ran too late to protect it.
	if !warningsContain(preview.Warnings, "3 directories could not be attributed to a known scenario") {
		t.Fatalf("expected the three unattributable directories counted and live ones excluded, got %v", preview.Warnings)
	}
}

func TestOrphanedInstanceStorageBlocksWhenRegistryUnreadable(t *testing.T) {
	provider, _, registry, _ := instanceFixture(t, nil)
	registry.err = errors.New("process registry unavailable")

	preview := instancePreview(t, provider)
	if len(preview.Items) != 0 {
		t.Fatalf("expected no items when liveness is unknown, got %v", previewPaths(preview))
	}
	if !strings.Contains(preview.BlockedReason, "instance registry could not be read") {
		t.Fatalf("expected a blocked reason naming the registry, got %q", preview.BlockedReason)
	}
}

func TestOrphanedInstanceStorageDeclaresProvenOrphanRoots(t *testing.T) {
	provider, _, _, _ := instanceFixture(t, map[string]struct{}{})
	meta := provider.Metadata()
	if err := meta.Validate(); err != nil {
		t.Fatalf("metadata is invalid: %v", err)
	}
	if len(meta.ProvenOrphanRoots) != 2 {
		t.Fatalf("expected both class roots declared so the protection boundary lets findings through, got %v", meta.ProvenOrphanRoots)
	}
	if meta.SafetyTier != cleanup.SafetyTierSafeWithOwner || meta.DefaultMode != cleanup.ProviderModeDisabled || meta.DefaultApproval != cleanup.ApprovalModeOwner {
		t.Fatalf("provider must stay disabled and owner-gated: %+v", meta)
	}
}

func TestOrphanedInstanceStorageApplyQuarantinesOnlyStoppedInstances(t *testing.T) {
	provider, fsys, _, quarantine := instanceFixture(t, map[string]struct{}{"audio-tools@presentation": {}})
	preview := instancePreview(t, provider)

	result, err := provider.Apply(context.Background(), cleanup.ApplyRequest{
		ProviderVersion: "v1",
		ApprovalMode:    cleanup.ApprovalModeOwner,
		IdempotencyKey:  "instance-apply-1",
		Preview:         preview,
	})
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	if len(fsys.Renamed) != len(preview.Items) {
		t.Fatalf("expected every previewed directory quarantined, got %v", fsys.Renamed)
	}
	for _, move := range fsys.Renamed {
		if !strings.HasPrefix(move[1], quarantine+string(filepath.Separator)) {
			t.Fatalf("directory left the quarantine root: %v", move)
		}
	}
	if result.ReclaimedBytes != 0 {
		t.Fatalf("quarantine must not claim reclaimed bytes, got %d", result.ReclaimedBytes)
	}
	if len(fsys.Removed) != 0 {
		t.Fatalf("apply deleted instead of quarantining: %v", fsys.Removed)
	}
}

func TestOrphanedInstanceStorageApplyRefusesInstanceThatRestarted(t *testing.T) {
	provider, fsys, registry, _ := instanceFixture(t, map[string]struct{}{})
	preview := instancePreview(t, provider)

	// The instance came back between plan and apply.
	registry.running = map[string]struct{}{"web-console@presentation": {}}
	result, err := provider.Apply(context.Background(), cleanup.ApplyRequest{
		ProviderVersion: "v1",
		ApprovalMode:    cleanup.ApprovalModeOwner,
		IdempotencyKey:  "instance-apply-2",
		Preview:         preview,
	})
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	for _, move := range fsys.Renamed {
		if strings.Contains(move[0], "web-console_presentation") {
			t.Fatalf("storage of a restarted instance was quarantined: %v", move)
		}
	}
	if !warningsContain(result.Warnings, "instance started again") {
		t.Fatalf("expected a restart warning, got %v", result.Warnings)
	}
}

func TestOrphanedInstanceStorageApplyRequiresApprovalAndQuarantine(t *testing.T) {
	provider, fsys, _, _ := instanceFixture(t, map[string]struct{}{})
	preview := instancePreview(t, provider)

	result, err := provider.Apply(context.Background(), cleanup.ApplyRequest{
		ProviderVersion: "v1",
		ApprovalMode:    cleanup.ApprovalModeNone,
		IdempotencyKey:  "instance-apply-3",
		Preview:         preview,
	})
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	if len(result.SkippedItems) != len(preview.Items) || len(fsys.Renamed) != 0 {
		t.Fatalf("unapproved apply acted: %+v", result)
	}

	unquarantined := NewOrphanedInstanceStorageProvider(fsys, &fakeInstanceRegistry{running: map[string]struct{}{}}, cleanupfakes.Clock{Time: instanceNow}, OrphanedInstanceStorageProviderConfig{
		NamespaceRoots: []InstanceNamespaceRoot{{Class: "data", Path: filepath.Join(fsys.Root, "data", "vrooli")}},
		KnownScenarios: []string{"web-console"},
	})
	result, err = unquarantined.Apply(context.Background(), cleanup.ApplyRequest{
		ProviderVersion: "v1",
		ApprovalMode:    cleanup.ApprovalModeOwner,
		IdempotencyKey:  "instance-apply-4",
		Preview:         preview,
	})
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	if len(fsys.Renamed) != 0 || !warningsContain(result.Warnings, "no quarantine root is configured") {
		t.Fatalf("apply without a quarantine root must refuse: %+v", result)
	}
}

func TestOrphanedInstanceStorageMinAgeHoldsFreshStorage(t *testing.T) {
	provider, _, _, _ := instanceFixture(t, map[string]struct{}{})
	preview, err := provider.Preview(context.Background(), cleanup.PreviewRequest{
		Scope:  cleanup.ObservationScope{Now: instanceNow},
		Policy: cleanup.ProviderPolicy{Enabled: true, ApprovalMode: cleanup.ApprovalModeOwner, MinAge: 30 * 24 * time.Hour},
	})
	if err != nil {
		t.Fatalf("Preview() error = %v", err)
	}
	if len(preview.Items) != 0 {
		t.Fatalf("expected storage younger than the minimum age to be held, got %v", previewPaths(preview))
	}
	// A held candidate must be visible. Silence here is indistinguishable from
	// a detector that missed the directory entirely.
	if !warningsContain(preview.Warnings, "3 stopped instance directories are younger than the 720h0m0s minimum age; held") {
		t.Fatalf("expected the age hold to be reported, got %v", preview.Warnings)
	}
}

// One instance owns a directory per class root. Its running-skip warning must
// be reported once, not once per root.
func TestOrphanedInstanceStorageDeduplicatesRunningWarning(t *testing.T) {
	provider, _, _, _ := instanceFixture(t, map[string]struct{}{"web-console@presentation": {}})
	preview := instancePreview(t, provider)

	seen := 0
	for _, warning := range preview.Warnings {
		if strings.Contains(warning, "web-console@presentation: instance is running") {
			seen++
		}
	}
	if seen != 1 {
		t.Fatalf("expected one running warning for an instance present in two class roots, got %d: %v", seen, preview.Warnings)
	}
}

func warningsContain(warnings []string, needle string) bool {
	for _, warning := range warnings {
		if strings.Contains(warning, needle) {
			return true
		}
	}
	return false
}
