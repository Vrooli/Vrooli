package cliinstall

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	testkitgo "github.com/vrooli/repo-contract-go/repocontracttest"
)

type recordingRemover struct {
	entries []InstallEntry
}

func (r *recordingRemover) Remove(entry InstallEntry) error {
	r.entries = append(r.entries, entry)
	return nil
}

type rootDeletingRemover struct {
	delegate *recordingRemover
	root     string
	deleted  bool
}

func (r *rootDeletingRemover) Remove(entry InstallEntry) error {
	if err := r.delegate.Remove(entry); err != nil {
		return err
	}
	if !r.deleted {
		r.deleted = true
		return os.RemoveAll(r.root)
	}
	return nil
}

func newUninstallTestService(t *testing.T) (*uninstallService, string) {
	t.Helper()
	fixture := testkitgo.NewRepoFixture(t)
	fixture.WriteRepoContract(t)
	t.Setenv("VROOLI_ROOT", fixture.Root)
	remover := &recordingRemover{}
	service, err := NewUninstallService(fixture.Root, fixture.Home, remover, func(string, time.Time) error { return nil })
	if err != nil {
		t.Fatalf("NewUninstallService: %v", err)
	}
	concrete := service.(*uninstallService)
	concrete.hostname = func() (string, error) { return "swarminator", nil }
	concrete.now = func() time.Time { return time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC) }
	return concrete, fixture.Home
}

func writeRawInstallRecord(t *testing.T, home string, record InstallRecord) {
	t.Helper()
	path, err := InstallRecordPath(home)
	if err != nil {
		t.Fatalf("InstallRecordPath: %v", err)
	}
	data, err := jsonMarshal(record)
	if err != nil {
		t.Fatalf("marshal record: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir record: %v", err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write record: %v", err)
	}
}

func jsonMarshal(value any) ([]byte, error) {
	return json.MarshalIndent(value, "", "  ")
}

func safetyCode(t *testing.T, err error) string {
	t.Helper()
	var safety *SafetyError
	if !errors.As(err, &safety) {
		t.Fatalf("error %v is not SafetyError", err)
	}
	return safety.Code
}

func TestLoadInstallRecordMigratesPreviousVersionBeforeValidation(t *testing.T) {
	_, home := newUninstallTestService(t)
	path := filepath.Join(home, "created.txt")
	writeRawInstallRecord(t, home, InstallRecord{
		Version: 1,
		Entries: []InstallEntry{{Scope: ScopeRuntime, Kind: EntryFile, Path: path, Prefix: home}},
	})

	record, err := LoadInstallRecord(home)
	if err != nil {
		t.Fatalf("LoadInstallRecord should migrate version 1: %v", err)
	}
	if record.Version != installRecordVersion {
		t.Fatalf("migrated version = %d, want %d", record.Version, installRecordVersion)
	}
	if len(record.Entries) != 1 || record.Entries[0].Path != path {
		t.Fatalf("migrated entries = %#v, want the recorded file preserved", record.Entries)
	}
}

func TestUninstallPlanAndApplyUseOnlyFrozenRecordedEntries(t *testing.T) {
	service, home := newUninstallTestService(t)
	path := filepath.Join(home, "Vrooli", "created.txt")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("before"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := RecordInstallEntries(home, InstallEntry{Scope: ScopeRuntime, Kind: EntryFile, Path: path, Prefix: filepath.Dir(path)}); err != nil {
		t.Fatalf("RecordInstallEntries: %v", err)
	}

	plan, err := service.Plan(UninstallRequest{Mode: UninstallPlanMode, Scope: ScopeRuntime, ConfirmTarget: "swarminator"})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if len(plan.Entries) != 1 || plan.Entries[0].Path != path {
		t.Fatalf("plan entries = %#v", plan.Entries)
	}
	receipt, err := service.Apply(UninstallRequest{Mode: UninstallApplyMode, PlanID: plan.ID, Scope: ScopeRuntime, ConfirmTarget: "swarminator", BreakGlass: "token"})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if len(receipt.Removed) != 1 || receipt.Removed[0].Path != path {
		t.Fatalf("receipt = %#v", receipt)
	}
	remover := service.remover.(*recordingRemover)
	if len(remover.entries) != 1 || remover.entries[0].Path != path {
		t.Fatalf("removal calls = %#v", remover.entries)
	}
}

func TestUninstallApplyPersistsProgressAfterSourceRootDisappears(t *testing.T) {
	service, home := newUninstallTestService(t)
	first := filepath.Join(home, "owned", "first.txt")
	second := filepath.Join(home, "owned", "second.txt")
	if err := os.MkdirAll(filepath.Dir(first), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{first, second} {
		if err := os.WriteFile(path, []byte("owned"), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := RecordInstallEntries(home, InstallEntry{Scope: ScopeRuntime, Kind: EntryFile, Path: path, Prefix: filepath.Dir(path)}); err != nil {
			t.Fatalf("RecordInstallEntries(%s): %v", path, err)
		}
	}

	plan, err := service.Plan(UninstallRequest{Mode: UninstallPlanMode, Scope: ScopeRuntime, ConfirmTarget: "swarminator"})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	remover := &recordingRemover{}
	service.remover = &rootDeletingRemover{delegate: remover, root: service.root}
	receipt, err := service.Apply(UninstallRequest{Mode: UninstallApplyMode, PlanID: plan.ID, Scope: ScopeRuntime, ConfirmTarget: "swarminator", BreakGlass: "token"})
	if err != nil {
		t.Fatalf("Apply after source-root removal: %v", err)
	}
	if len(receipt.Removed) != 2 {
		t.Fatalf("receipt removed %d entries, want 2: %#v", len(receipt.Removed), receipt)
	}
	path, err := service.planPath(plan.ID)
	if err != nil {
		t.Fatalf("planPath: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("persisted plan after source-root removal: %v", err)
	}
}

func TestUnrecordedInstallProducesEmptyPlan(t *testing.T) {
	service, _ := newUninstallTestService(t)
	plan, err := service.Plan(UninstallRequest{Mode: UninstallPlanMode, Scope: ScopeAll, ConfirmTarget: "swarminator"})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if len(plan.Entries) != 0 || len(plan.Disk) != 0 {
		t.Fatalf("unrecorded plan = %#v", plan)
	}
}

func TestLaunchdDomainForPathDistinguishesSystemDaemons(t *testing.T) {
	if got := launchdDomainForPath("/Library/LaunchDaemons/com.vrooli.bridge.vrooli-bridge-provisioner.plist"); got != "system" {
		t.Fatalf("LaunchDaemon domain = %q, want system", got)
	}
	if got := launchdDomainForPath("/Users/runner/Library/LaunchAgents/com.example.agent.plist"); got != "gui/"+currentUserID() {
		t.Fatalf("LaunchAgent domain = %q, want gui/%s", got, currentUserID())
	}
}

func TestFileRemoverDefersOnlyExplicitServiceNames(t *testing.T) {
	r := fileRemover{deferredServiceNames: map[string]struct{}{"vrooli-bridge-provisioner": {}}}
	deferred := InstallEntry{Kind: EntryService, ServiceName: "vrooli-bridge-provisioner"}
	launchdDeferred := InstallEntry{Kind: EntryService, ServiceName: "com.vrooli.bridge.vrooli-bridge-provisioner"}
	other := InstallEntry{Kind: EntryService, ServiceName: "other-service"}
	if !r.defersServiceStop(deferred) {
		t.Fatal("explicitly deferred service was not deferred")
	}
	if !r.defersServiceStop(launchdDeferred) {
		t.Fatal("launchd label for explicitly deferred service was not deferred")
	}
	if r.defersServiceStop(other) {
		t.Fatal("unrelated service was deferred")
	}
}

func TestRecordProjectSetupRecordsOwnedCheckoutWithoutScanning(t *testing.T) {
	home := t.TempDir()
	root := filepath.Join(t.TempDir(), "Vrooli")
	if err := os.Mkdir(root, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "preexisting.txt"), []byte("existing"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := RecordProjectSetup(root, home); err != nil {
		t.Fatalf("RecordProjectSetup: %v", err)
	}
	record, err := LoadInstallRecord(home)
	if err != nil {
		t.Fatalf("LoadInstallRecord: %v", err)
	}
	if len(record.Entries) != 1 {
		t.Fatalf("expected one explicit checkout entry, got %d: %#v", len(record.Entries), record.Entries)
	}
	if record.Entries[0].Path != root || record.Entries[0].Kind != EntryDirectory || record.Entries[0].Scope != ScopeRuntime {
		t.Fatalf("unexpected checkout entry: %#v", record.Entries[0])
	}
}

func TestUninstallRefusesTargetBeforeReadingInventory(t *testing.T) {
	service, home := newUninstallTestService(t)
	writeRawInstallRecord(t, home, InstallRecord{Version: 99, Entries: []InstallEntry{{Scope: ScopeRuntime, Kind: EntryFile, Path: "/outside", Prefix: "/outside"}}})
	_, err := service.Plan(UninstallRequest{Mode: UninstallPlanMode, Scope: ScopeAll, ConfirmTarget: "minimouse"})
	if got := safetyCode(t, err); got != "target_mismatch" {
		t.Fatalf("safety code = %q, want target_mismatch", got)
	}
}

func TestUninstallApplyRequiresBreakGlassBeforePlanRead(t *testing.T) {
	service, _ := newUninstallTestService(t)
	service.verify = nil
	_, err := service.Apply(UninstallRequest{Mode: UninstallApplyMode, PlanID: "0123456789abcdef0123456789abcdef", ConfirmTarget: "swarminator"})
	if got := safetyCode(t, err); got != "break_glass_required" {
		t.Fatalf("safety code = %q, want break_glass_required", got)
	}

	service.verify = func(string, time.Time) error { return errors.New("expired") }
	_, err = service.Apply(UninstallRequest{Mode: UninstallApplyMode, PlanID: "0123456789abcdef0123456789abcdef", ConfirmTarget: "swarminator", BreakGlass: "expired"})
	if got := safetyCode(t, err); got != "break_glass_required" {
		t.Fatalf("expired safety code = %q, want break_glass_required", got)
	}
}

func TestUninstallRejectsUnsafeRecordedPaths(t *testing.T) {
	cases := []struct {
		name  string
		entry InstallEntry
		setup func(t *testing.T, entry InstallEntry)
		want  string
	}{
		{
			name:  "outside prefix",
			entry: InstallEntry{Scope: ScopeRuntime, Kind: EntryFile, Path: "/tmp/outside", Prefix: "/tmp/owned"},
			want:  "path_outside_prefix",
		},
		{
			name: "home itself",
			want: "path_forbidden",
			setup: func(t *testing.T, entry InstallEntry) {
				service, home := newUninstallTestService(t)
				writeRawInstallRecord(t, home, InstallRecord{Version: installRecordVersion, Entries: []InstallEntry{{Scope: ScopeRuntime, Kind: EntryDirectory, Path: home, Prefix: home}}})
				_, err := service.Plan(UninstallRequest{Mode: UninstallPlanMode, Scope: ScopeRuntime, ConfirmTarget: "swarminator"})
				if got := safetyCode(t, err); got != "path_forbidden" {
					t.Fatalf("safety code = %q, want path_forbidden", got)
				}
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.setup != nil {
				tc.setup(t, tc.entry)
				return
			}
			service, home := newUninstallTestService(t)
			writeRawInstallRecord(t, home, InstallRecord{Version: installRecordVersion, Entries: []InstallEntry{tc.entry}})
			_, err := service.Plan(UninstallRequest{Mode: UninstallPlanMode, Scope: ScopeRuntime, ConfirmTarget: "swarminator"})
			if got := safetyCode(t, err); got != tc.want {
				t.Fatalf("safety code = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestUninstallRejectsSymlinkLeavingPrefix(t *testing.T) {
	service, home := newUninstallTestService(t)
	prefix := filepath.Join(home, "owned")
	if err := os.MkdirAll(prefix, 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(prefix, "link")
	if err := os.Symlink(t.TempDir(), link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	writeRawInstallRecord(t, home, InstallRecord{Version: installRecordVersion, Entries: []InstallEntry{{Scope: ScopeRuntime, Kind: EntryDirectory, Path: link, Prefix: prefix}}})
	_, err := service.Plan(UninstallRequest{Mode: UninstallPlanMode, Scope: ScopeRuntime, ConfirmTarget: "swarminator"})
	if got := safetyCode(t, err); got != "symlink_outside_prefix" {
		t.Fatalf("safety code = %q, want symlink_outside_prefix", got)
	}
}

func TestUninstallRejectsFrozenDiskDrift(t *testing.T) {
	service, home := newUninstallTestService(t)
	path := filepath.Join(home, "owned", "file")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("before"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := RecordInstallEntries(home, InstallEntry{Scope: ScopeRuntime, Kind: EntryFile, Path: path, Prefix: filepath.Dir(path)}); err != nil {
		t.Fatal(err)
	}
	plan, err := service.Plan(UninstallRequest{Mode: UninstallPlanMode, Scope: ScopeRuntime, ConfirmTarget: "swarminator"})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("after"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err = service.Apply(UninstallRequest{Mode: UninstallApplyMode, PlanID: plan.ID, Scope: ScopeRuntime, ConfirmTarget: "swarminator", BreakGlass: "token"})
	if got := safetyCode(t, err); got != "plan_stale" {
		t.Fatalf("safety code = %q, want plan_stale", got)
	}
	if got := len(service.remover.(*recordingRemover).entries); got != 0 {
		t.Fatalf("remover called %d times after drift", got)
	}
}

func TestUninstallVolatileEntryRemainsRemovableWithoutContentHash(t *testing.T) {
	service, home := newUninstallTestService(t)
	path := filepath.Join(home, "agent-state")
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(path, "heartbeat"), []byte("before"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := RecordInstallEntries(home, InstallEntry{
		Scope: ScopeAgent, Kind: EntryDirectory, Path: path, Prefix: path, Volatile: true,
	}); err != nil {
		t.Fatalf("RecordInstallEntries: %v", err)
	}

	plan, err := service.Plan(UninstallRequest{Mode: UninstallPlanMode, Scope: ScopeAgent, ConfirmTarget: "swarminator"})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if len(plan.Remove) != 1 || !plan.Remove[0].Volatile {
		t.Fatalf("volatile plan removal = %#v", plan.Remove)
	}
	if len(plan.Disk) != 0 {
		t.Fatalf("volatile plan unexpectedly froze content: %#v", plan.Disk)
	}

	// A live agent may update this directory after inventory. The exact ledger
	// entry remains authorized, while unrelated recorded content is still the
	// only removal target.
	if err := os.WriteFile(filepath.Join(path, "heartbeat"), []byte("after"), 0o600); err != nil {
		t.Fatal(err)
	}
	receipt, err := service.Apply(UninstallRequest{Mode: UninstallApplyMode, PlanID: plan.ID, Scope: ScopeAgent, ConfirmTarget: "swarminator", BreakGlass: "token"})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if len(receipt.Removed) != 1 || receipt.Removed[0].Path != path {
		t.Fatalf("volatile receipt = %#v", receipt)
	}
}

func TestClassifyEntriesAppliesPackageOwnershipRule(t *testing.T) {
	packageEntry := func(name string, before ObservedBefore, action InstallAction, shared, attributable bool) InstallEntry {
		return InstallEntry{
			Scope: ScopeRuntime,
			Kind:  EntryPackage,
			Path:  name,
			Provenance: InstallProvenance{
				PackageName:    name,
				ObservedBefore: before,
				Action:         action,
				Shared:         shared,
				Attributable:   attributable,
			},
		}
	}
	remove, keep, cannot := classifyEntries([]InstallEntry{
		packageEntry("installed", ObservedAbsent, ActionInstalled, false, true),
		packageEntry("present", ObservedPresent, ActionInstalled, false, true),
		packageEntry("adopted", ObservedAbsent, ActionAdopted, false, true),
		packageEntry("shared", ObservedAbsent, ActionInstalled, true, true),
		packageEntry("unknown", ObservedUnknown, ActionInstalled, false, false),
		{Scope: ScopeRuntime, Kind: EntryFile, Path: "/owned/file", Prefix: "/owned"},
	})
	if got := len(remove); got != 2 {
		t.Fatalf("remove count = %d, want 2", got)
	}
	if remove[0].Kind != EntryPackage || remove[1].Kind != EntryFile {
		t.Fatalf("remove entries = %#v", remove)
	}
	if len(keep) != 3 || len(cannot) != 1 {
		t.Fatalf("keep=%d cannot=%d, want 3 and 1", len(keep), len(cannot))
	}
	for _, decision := range append(keep, cannot...) {
		if strings.TrimSpace(decision.Reason) == "" {
			t.Fatalf("decision %q has no reason", decision.Path)
		}
	}
}

func TestInstallRecordMigratesLegacyPackageToUnknownProvenance(t *testing.T) {
	home := t.TempDir()
	writeRawInstallRecord(t, home, InstallRecord{Version: 1, Entries: []InstallEntry{{
		Scope: ScopeRuntime, Kind: EntryPackage, Path: "legacy-tool",
	}}})
	record, err := LoadInstallRecord(home)
	if err != nil {
		t.Fatalf("LoadInstallRecord: %v", err)
	}
	if record.Version != installRecordVersion {
		t.Fatalf("record version = %d, want %d", record.Version, installRecordVersion)
	}
	entry := record.Entries[0]
	if entry.Provenance.ObservedBefore != ObservedUnknown || entry.Provenance.Attributable {
		t.Fatalf("legacy provenance = %#v, want unknown and unattributable", entry.Provenance)
	}
	_, _, cannot := classifyEntries(record.Entries)
	if len(cannot) != 1 {
		t.Fatalf("legacy cannot-attribute entries = %#v", cannot)
	}
}

func TestRecordContainerRuntimePersistsProviderAndEndpoint(t *testing.T) {
	home := t.TempDir()
	if err := RecordContainerRuntime(home, "colima", "docker://colima", "minimouse", ObservedAbsent, ActionInstalled); err != nil {
		t.Fatalf("RecordContainerRuntime: %v", err)
	}
	record, err := LoadInstallRecord(home)
	if err != nil {
		t.Fatalf("LoadInstallRecord: %v", err)
	}
	if len(record.RuntimeProviders) != 1 {
		t.Fatalf("runtime providers = %#v, want one entry", record.RuntimeProviders)
	}
	provider := record.RuntimeProviders[0]
	if provider.Provider != "colima" || provider.Endpoint != "docker://colima" || provider.Action != ActionInstalled || !provider.Attributable {
		t.Fatalf("runtime provider = %#v", provider)
	}
}

func TestUninstallPlanRendersContainerRemovalOrderWithoutDiscovery(t *testing.T) {
	service, home := newUninstallTestService(t)
	if err := RecordInstallEntries(home,
		InstallEntry{Scope: ScopeRuntime, Kind: EntryImage, Path: "fixture:1", Resource: "fixture", Provenance: InstallProvenance{ObservedBefore: ObservedAbsent, Action: ActionInstalled, Attributable: true}},
		InstallEntry{Scope: ScopeRuntime, Kind: EntryContainer, Path: "fixture-container", Resource: "fixture", Provenance: InstallProvenance{ObservedBefore: ObservedAbsent, Action: ActionInstalled, Attributable: true}},
		InstallEntry{Scope: ScopeRuntime, Kind: EntryVolume, Path: "fixture-volume", Resource: "fixture", Provenance: InstallProvenance{ObservedBefore: ObservedAbsent, Action: ActionInstalled, Attributable: true}},
		InstallEntry{Scope: ScopeRuntime, Kind: EntryNetwork, Path: "fixture-network", Resource: "fixture", Provenance: InstallProvenance{ObservedBefore: ObservedAbsent, Action: ActionInstalled, Attributable: true}},
	); err != nil {
		t.Fatalf("RecordInstallEntries: %v", err)
	}
	plan, err := service.Plan(UninstallRequest{Mode: UninstallPlanMode, Scope: ScopeRuntime, ConfirmTarget: "swarminator"})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	want := []InstallEntryKind{EntryNetwork, EntryContainer, EntryVolume, EntryImage}
	for i, kind := range want {
		if plan.Remove[i].Kind != kind {
			t.Fatalf("remove[%d] kind = %s, want %s", i, plan.Remove[i].Kind, kind)
		}
	}
	if len(plan.Disk) != 0 {
		t.Fatalf("container plan unexpectedly has filesystem disk snapshots: %#v", plan.Disk)
	}
}

var _ Remover = fileRemover{}
