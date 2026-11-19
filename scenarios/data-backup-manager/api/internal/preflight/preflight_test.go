package preflight

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"data-backup-manager/internal/destinationreadiness"
	"data-backup-manager/internal/engine"
	"data-backup-manager/internal/failures"
	"data-backup-manager/internal/sources"
	"data-backup-manager/internal/sysmounts"
)

type targetLookup struct{ targets map[string]Target }

func (l targetLookup) TargetForRun(_ context.Context, id string) (Target, error) {
	t, ok := l.targets[id]
	if !ok {
		return Target{}, errors.New("target not found")
	}
	return t, nil
}

type destLookup struct {
	destinations    map[string]Destination
	withResolvedErr error
}

func (l destLookup) DestinationForRun(_ context.Context, id string) (Destination, error) {
	d, ok := l.destinations[id]
	if !ok {
		return Destination{}, errors.New("destination not found")
	}
	return d, nil
}

func (l destLookup) WithResolvedDestinationForRead(_ context.Context, id string, use func(Destination) error) error {
	if l.withResolvedErr != nil {
		return l.withResolvedErr
	}
	d, ok := l.destinations[id]
	if !ok {
		return errors.New("destination not found")
	}
	return use(d)
}

type readinessFake struct {
	report destinationreadiness.Report
	err    error
}

func (f readinessFake) Analyze(context.Context, destinationreadiness.AnalyzeInput) (destinationreadiness.Report, error) {
	return f.report, f.err
}

type readinessSpy struct {
	report destinationreadiness.Report
	input  destinationreadiness.AnalyzeInput
}

func (s *readinessSpy) Analyze(_ context.Context, input destinationreadiness.AnalyzeInput) (destinationreadiness.Report, error) {
	s.input = input
	return s.report, nil
}

type engineFake struct {
	statusCalls int
	statusErr   error
	listCalls   int
	listErr     error
	statsErr    error
	damageScan  engine.RepoDamageScan
	damageCalls int
}

func (e *engineFake) RepoCreate(context.Context, engine.RepoSpec) error  { return nil }
func (e *engineFake) RepoConnect(context.Context, engine.RepoSpec) error { return nil }
func (e *engineFake) RepoRebind(context.Context, string, string) error   { return nil }
func (e *engineFake) RepoStatus(context.Context, string) (engine.RepoStatus, error) {
	e.statusCalls++
	return engine.RepoStatus{}, e.statusErr
}
func (e *engineFake) PassphraseRef(string) string { return "secret-ref" }
func (e *engineFake) RepoStats(context.Context, string) (engine.RepoStats, error) {
	return engine.RepoStats{}, e.statsErr
}

func (e *engineFake) RepoDamageScan(context.Context, string) (engine.RepoDamageScan, error) {
	e.damageCalls++
	return e.damageScan, nil
}
func (e *engineFake) RepoDelete(context.Context, string) error { return nil }
func (e *engineFake) SnapshotCreate(context.Context, string, string, engine.SnapshotMetadata) (engine.Snapshot, error) {
	return engine.Snapshot{}, nil
}

func (e *engineFake) SnapshotList(context.Context, string, string) ([]engine.Snapshot, error) {
	e.listCalls++
	return nil, e.listErr
}

func (e *engineFake) SnapshotListByTags(context.Context, string, []string) ([]engine.Snapshot, error) {
	e.listCalls++
	return nil, e.listErr
}
func (e *engineFake) SnapshotRestore(context.Context, string, string, string) error { return nil }
func (e *engineFake) SnapshotVerify(context.Context, string, string, int) error     { return nil }
func (e *engineFake) BrowseSnapshot(context.Context, string, string, string) ([]engine.SnapshotEntry, error) {
	return nil, nil
}

func (e *engineFake) ApplyRetention(context.Context, string, string, engine.RetentionPolicy) (int, error) {
	return 0, nil
}

func TestCheckGroupsOneSharedCredentialFailure(t *testing.T) {
	eng := &engineFake{statusErr: errors.New("read repository passphrase: credential is not configured")}
	registry := sources.NewRegistry(&captureFake{kind: sources.KindFilesystem})
	r := Check(context.Background(), Input{
		Plan: Plan{TargetIDs: []string{"t1", "t2"}, DestinationIDs: []string{"d1"}},
		Targets: targetLookup{targets: map[string]Target{
			"t1": {ID: "t1", Kind: sources.KindFilesystem}, "t2": {ID: "t2", Kind: sources.KindFilesystem},
		}},
		Destinations: destLookup{destinations: map[string]Destination{"d1": {ID: "d1", Name: "primary"}}},
		Engine:       eng, Sources: registry,
	})
	if r.Ready || !r.Blocked {
		t.Fatalf("preflight posture = ready=%v blocked=%v", r.Ready, r.Blocked)
	}
	if len(r.Incidents) != 1 {
		t.Fatalf("incidents = %d, want one grouped incident: %+v", len(r.Incidents), r.Incidents)
	}
	if got := r.Incidents[0].Code; string(got) != "credential_missing" {
		t.Fatalf("code = %q", got)
	}
	if len(r.Incidents[0].TargetIDs) != 2 {
		t.Fatalf("affected targets = %v", r.Incidents[0].TargetIDs)
	}
	if len(r.BlockedDestinations) != 1 || len(r.BlockedTargets) != 0 {
		t.Fatalf("blocked maps = destinations=%v targets=%v", r.BlockedDestinations, r.BlockedTargets)
	}
	if eng.statusCalls != 1 {
		t.Fatalf("repo status calls = %d, want one per shared destination", eng.statusCalls)
	}
	if eng.listCalls != 0 {
		t.Fatalf("snapshot list calls = %d, want none after failed repo status", eng.listCalls)
	}
	if got := r.Summary(); got == "" || len(got) > 300 {
		t.Fatalf("summary should be bounded and useful: %q", got)
	}
}

func TestCheckBlocksRepositoryWithUnreadableSnapshotMetadata(t *testing.T) {
	eng := &engineFake{listErr: errors.New("snapshot manifest blob is missing")}
	registry := sources.NewRegistry(&captureFake{kind: sources.KindFilesystem})
	r := Check(context.Background(), Input{
		Plan: Plan{TargetIDs: []string{"t1", "t2"}, DestinationIDs: []string{"d1"}},
		Targets: targetLookup{targets: map[string]Target{
			"t1": {ID: "t1", Kind: sources.KindFilesystem}, "t2": {ID: "t2", Kind: sources.KindFilesystem},
		}},
		Destinations: destLookup{destinations: map[string]Destination{"d1": {ID: "d1", Name: "elements-local"}}},
		Engine:       eng, Sources: registry,
	})
	if r.Ready || !r.Blocked {
		t.Fatalf("preflight posture = ready=%v blocked=%v", r.Ready, r.Blocked)
	}
	if len(r.Incidents) != 1 || r.Incidents[0].Code != failures.RepositoryInvalid {
		t.Fatalf("incidents = %+v, want one repository integrity incident", r.Incidents)
	}
	if r.Incidents[0].Message != "repository snapshot metadata could not be enumerated" || len(r.Incidents[0].TargetIDs) != 2 {
		t.Fatalf("incident should be redacted and grouped: %+v", r.Incidents[0])
	}
	if eng.statusCalls != 1 || eng.listCalls != 1 {
		t.Fatalf("repo probes = status:%d list:%d, want once each", eng.statusCalls, eng.listCalls)
	}
}

func TestCheckIncludesKopiaOwnerDamageClassificationWhenUsageFails(t *testing.T) {
	eng := &engineFake{statsErr: errors.New("usage failed"), damageScan: engine.RepoDamageScan{
		Unreadable: []engine.RepoDamageEntry{{Class: "repository_blob", Problem: "io_error"}},
	}}
	registry := sources.NewRegistry(&captureFake{kind: sources.KindFilesystem})
	r := Check(context.Background(), Input{
		Plan:         Plan{TargetIDs: []string{"t1"}, DestinationIDs: []string{"d1"}},
		Targets:      targetLookup{targets: map[string]Target{"t1": {ID: "t1", Kind: sources.KindFilesystem}}},
		Destinations: destLookup{destinations: map[string]Destination{"d1": {ID: "d1", Name: "elements-local"}}},
		Engine:       eng, Sources: registry,
	})
	if r.Ready || !r.Blocked || len(r.Incidents) != 1 || r.Incidents[0].Code != failures.RepositoryInvalid {
		t.Fatalf("preflight posture = %+v", r)
	}
	if r.Incidents[0].Message != "Kopia classified unreadable repository entries: 1 repository_blob io_error" || eng.damageCalls != 1 {
		t.Fatalf("preflight did not surface one owner-classified damage result: incident=%+v scanCalls=%d", r.Incidents[0], eng.damageCalls)
	}
}

// A probe-backed degraded state is a warning, not a block. Preflight carries no
// separate redundancy policy: the write verdict decides, and copy count is
// reported by coverage.
func TestCheckAllowsAProbeBackedDegradedReadiness(t *testing.T) {
	registry := sources.NewRegistry(&captureFake{kind: sources.KindFilesystem})
	r := Check(context.Background(), Input{
		Plan: Plan{TargetIDs: []string{"t1"}, DestinationIDs: []string{"d1"}},
		Targets: targetLookup{targets: map[string]Target{
			"t1": {ID: "t1", Kind: sources.KindFilesystem},
		}},
		Destinations: destLookup{destinations: map[string]Destination{
			"d1": {ID: "d1", Name: "external", BackendKind: "filesystem", Location: "/mnt/external"},
		}},
		Engine: &engineFake{}, Sources: registry,
		Readiness: readinessFake{report: destinationreadiness.Report{
			OverallSeverity: destinationreadiness.SeverityWarning,
			Checks: []destinationreadiness.CheckResult{{
				Code: "filesystem_state", Severity: destinationreadiness.SeverityWarning,
				Message: "filesystem state is unobservable; a bounded write probe passed and another verified copy exists",
			}},
		}},
	})
	if !r.Ready || r.Blocked {
		t.Fatalf("preflight posture = ready=%v blocked=%v, want degraded readiness admitted", r.Ready, r.Blocked)
	}
	if len(r.Incidents) != 0 {
		t.Fatalf("incidents = %+v, want none for a degraded-but-allowed destination", r.Incidents)
	}
}

func TestCheckAttributesFailedWriteProbeToTheDestination(t *testing.T) {
	registry := sources.NewRegistry(&captureFake{kind: sources.KindFilesystem})
	r := Check(context.Background(), Input{
		Plan: Plan{TargetIDs: []string{"t1"}, DestinationIDs: []string{"d1"}},
		Targets: targetLookup{targets: map[string]Target{
			"t1": {ID: "t1", Kind: sources.KindFilesystem},
		}},
		Destinations: destLookup{destinations: map[string]Destination{
			"d1": {ID: "d1", Name: "external", BackendKind: "filesystem", Location: "/mnt/external"},
		}},
		Engine: &engineFake{}, Sources: registry,
		Readiness: readinessFake{report: destinationreadiness.Report{
			OverallSeverity: destinationreadiness.SeverityFail,
			Checks: []destinationreadiness.CheckResult{{
				Code: "write_probe", Severity: destinationreadiness.SeverityFail,
				Message: "bounded write probe failed: EIO", NextAction: "restore writable, error-free access",
			}},
		}},
	})
	if r.Ready || !r.Blocked {
		t.Fatalf("preflight posture = ready=%v blocked=%v", r.Ready, r.Blocked)
	}
	if len(r.Incidents) != 1 || r.Incidents[0].Code != failures.DestinationWriteProbe {
		t.Fatalf("incidents = %+v, want destination_write_probe", r.Incidents)
	}
}

func TestCheckBlocksReadOnlyFilesystemBeforeCapture(t *testing.T) {
	registry := sources.NewRegistry(&captureFake{kind: sources.KindFilesystem})
	r := Check(context.Background(), Input{
		Plan: Plan{TargetIDs: []string{"t1"}, DestinationIDs: []string{"d1"}},
		Targets: targetLookup{targets: map[string]Target{
			"t1": {ID: "t1", Kind: sources.KindFilesystem},
		}},
		Destinations: destLookup{destinations: map[string]Destination{
			"d1": {ID: "d1", Name: "primary", BackendKind: "filesystem", Location: "/mnt/elements"},
		}},
		Engine: &engineFake{}, Sources: registry,
		Readiness: readinessFake{report: destinationreadiness.Report{
			OverallSeverity: destinationreadiness.SeverityFail,
			Checks: []destinationreadiness.CheckResult{{
				Code: "mounted_read_write", Severity: destinationreadiness.SeverityFail,
			}},
		}},
	})
	if r.Ready || !r.Blocked {
		t.Fatalf("preflight posture = ready=%v blocked=%v", r.Ready, r.Blocked)
	}
	if len(r.Incidents) != 1 || r.Incidents[0].Code != failures.DestinationReadOnly {
		t.Fatalf("incidents = %+v, want destination_read_only", r.Incidents)
	}
}

func TestCheckBlocksUnknownFilesystemReadinessBeforeCapture(t *testing.T) {
	registry := sources.NewRegistry(&captureFake{kind: sources.KindFilesystem})
	eng := &engineFake{}
	r := Check(context.Background(), Input{
		Plan: Plan{TargetIDs: []string{"t1"}, DestinationIDs: []string{"d1"}},
		Targets: targetLookup{targets: map[string]Target{
			"t1": {ID: "t1", Kind: sources.KindFilesystem},
		}},
		Destinations: destLookup{destinations: map[string]Destination{
			"d1": {ID: "d1", Name: "external", BackendKind: "filesystem", Location: "/mnt/external"},
		}},
		Engine: eng, Sources: registry,
		Readiness: readinessFake{report: destinationreadiness.Report{
			OverallSeverity: destinationreadiness.SeverityUnknown,
			Checks: []destinationreadiness.CheckResult{{
				Code: "filesystem_health", Severity: destinationreadiness.SeverityUnknown,
			}},
		}},
	})
	if r.Ready || !r.Blocked {
		t.Fatalf("preflight posture = ready=%v blocked=%v, want unknown readiness blocked", r.Ready, r.Blocked)
	}
	if len(r.Incidents) != 1 || r.Incidents[0].Code != failures.DestinationInaccessible {
		t.Fatalf("incidents = %+v, want one destination readiness blocker", r.Incidents)
	}
	if r.Incidents[0].Message != "destination filesystem readiness is unknown" {
		t.Fatalf("incident message = %q, want unknown readiness explanation", r.Incidents[0].Message)
	}
	if eng.statusCalls != 1 || eng.listCalls != 1 {
		t.Fatalf("read-only repository probes = status:%d list:%d, want one each", eng.statusCalls, eng.listCalls)
	}
}

func TestCheckAllowsFilesystemReadinessWarning(t *testing.T) {
	registry := sources.NewRegistry(&captureFake{kind: sources.KindFilesystem})
	identity := destinationreadiness.DeviceIdentity{DevicePath: "/dev/usb1", Filesystem: "ntfs", UUID: "uuid-a", Serial: "serial-a"}
	readiness := &readinessSpy{report: destinationreadiness.Report{
		OverallSeverity:      destinationreadiness.SeverityWarning,
		FilesystemState:      sysmounts.FilesystemStateUnknown,
		OfflineCheckVerified: true,
		Checks: []destinationreadiness.CheckResult{{
			Code: "filesystem_state", Severity: destinationreadiness.SeverityPass,
		}},
	}}
	r := Check(context.Background(), Input{
		Plan: Plan{TargetIDs: []string{"t1"}, DestinationIDs: []string{"d1"}},
		Targets: targetLookup{targets: map[string]Target{
			"t1": {ID: "t1", Kind: sources.KindFilesystem},
		}},
		Destinations: destLookup{destinations: map[string]Destination{
			"d1": {ID: "d1", Name: "external", BackendKind: "filesystem", Location: "/mnt/external", DeviceIdentity: &identity},
		}},
		Engine: &engineFake{}, Sources: registry,
		Readiness: readiness,
	})
	if !r.Ready || r.Blocked || len(r.Incidents) != 0 {
		t.Fatalf("warning-only preflight = %+v, want ready without blockers", r)
	}
	if readiness.input.DestinationID != "d1" || readiness.input.ExpectedDevice.UUID != identity.UUID {
		t.Fatalf("readiness input=%+v, want destination-bound stable volume identity", readiness.input)
	}
}

type captureFake struct{ kind sources.SourceKind }

func (f *captureFake) Kind() sources.SourceKind { return f.kind }
func (f *captureFake) Capture(context.Context, sources.CaptureSpec) (sources.Artifact, error) {
	return sources.Artifact{}, nil
}
func (f *captureFake) Restore(context.Context, sources.RestoreSpec) error { return nil }

// [REQ:DBM-DEST-DURABLE] Temporary storage cannot pass durable readiness,
// even if the repository itself answers successfully.
func TestTemporaryDestinationIsBlockedBeforeEngine(t *testing.T) {
	e := &engineFake{}
	r := Check(context.Background(), Input{
		Plan:         Plan{TargetIDs: []string{"t"}, DestinationIDs: []string{"d"}},
		Targets:      targetLookup{targets: map[string]Target{"t": {ID: "t", Kind: sources.KindFilesystem, Locator: "/source"}}},
		Destinations: destLookup{destinations: map[string]Destination{"d": {ID: "d", Name: "temp", BackendKind: "filesystem", Location: "/tmp/backup"}}}, Engine: e,
		Sources: sources.NewProductionRegistry(sources.ExecRunner{}),
	})
	if r.Ready || e.statusCalls != 0 || len(r.Incidents) == 0 || r.Incidents[0].Code != "destination_ephemeral" {
		t.Fatalf("temporary destination accepted: %+v", r)
	}
}

func TestDestinationReadinessChangeBlocksRepositoryProbes(t *testing.T) {
	e := &engineFake{}
	r := Check(context.Background(), Input{
		Plan:         Plan{TargetIDs: []string{"t"}, DestinationIDs: []string{"d"}},
		Targets:      targetLookup{targets: map[string]Target{"t": {ID: "t", Kind: sources.KindFilesystem, Locator: "/source"}}},
		Destinations: destLookup{destinations: map[string]Destination{"d": {ID: "d", Name: "external", BackendKind: "filesystem", Location: "/mnt/external"}}, withResolvedErr: errors.New("readiness changed")},
		Engine:       e,
		Sources:      sources.NewRegistry(&captureFake{kind: sources.KindFilesystem}),
	})
	if r.Ready || !r.Blocked || len(r.Incidents) == 0 {
		t.Fatalf("preflight = %+v, want a destination blocker", r)
	}
	if e.statusCalls != 0 || e.listCalls != 0 {
		t.Fatalf("repository probed after destination gate failed: status=%d list=%d", e.statusCalls, e.listCalls)
	}
}

// One source that cannot be enumerated (a root-owned subdirectory) is that
// target's problem. It used to leave the destination's capacity "unknown", which
// preflight reported as "destination filesystem readiness is unknown" and so
// stopped every target of the plan for as long as the directory stayed unreadable.
type capacityAwareReadiness struct {
	got destinationreadiness.AnalyzeInput
}

func (c *capacityAwareReadiness) Analyze(_ context.Context, in destinationreadiness.AnalyzeInput) (destinationreadiness.Report, error) {
	c.got = in
	if in.Capacity.LogicalSourceBytes == 0 {
		return destinationreadiness.Report{OverallSeverity: destinationreadiness.SeverityUnknown, Checks: []destinationreadiness.CheckResult{{
			Code: "capacity", Severity: destinationreadiness.SeverityUnknown, Message: "target size not selected; capacity fit is unknown",
		}}}, nil
	}
	return destinationreadiness.Report{OverallSeverity: destinationreadiness.SeverityPass}, nil
}

func TestCheckBlocksOnlyTheUnreadableTargetNotTheDestination(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("permission denial cannot be produced as root")
	}
	root := t.TempDir()
	good := filepath.Join(root, "good")
	bad := filepath.Join(root, "bad")
	for _, dir := range []string{good, filepath.Join(bad, "sealed")} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(good, "data"), make([]byte, 4096), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bad, "sealed", "x"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(filepath.Join(bad, "sealed"), 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(filepath.Join(bad, "sealed"), 0o755) })

	readiness := &capacityAwareReadiness{}
	r := Check(context.Background(), Input{
		Plan: Plan{TargetIDs: []string{"good", "bad"}, DestinationIDs: []string{"d1"}},
		Targets: targetLookup{targets: map[string]Target{
			"good": {ID: "good", Kind: sources.KindFilesystem, Locator: good},
			"bad":  {ID: "bad", Kind: sources.KindFilesystem, Locator: bad},
		}},
		Destinations: destLookup{destinations: map[string]Destination{
			"d1": {ID: "d1", Name: "local", BackendKind: "filesystem", Location: "/mnt/local"},
		}},
		Engine: &engineFake{}, Sources: sources.NewRegistry(&captureFake{kind: sources.KindFilesystem}),
		Readiness: readiness, CheckSourcePaths: true,
	})
	if _, blocked := r.BlockedDestinations["d1"]; blocked {
		t.Fatalf("an unreadable source stopped the destination for every target: %+v", r.Incidents)
	}
	if _, blocked := r.BlockedTargets["bad"]; !blocked {
		t.Fatalf("the unreadable target must be blocked on its own: %+v", r.Incidents)
	}
	if _, blocked := r.BlockedTargets["good"]; blocked {
		t.Fatalf("a readable target was blocked: %+v", r.Incidents)
	}
	if readiness.got.Capacity.LogicalSourceBytes == 0 {
		t.Fatal("capacity must be judged over the targets that will actually be written")
	}
	for _, in := range r.Incidents {
		if in.Code == failures.DestinationInaccessible {
			t.Fatalf("unexpected destination incident: %+v", in)
		}
	}
}
