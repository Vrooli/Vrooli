package preflight

import (
	"context"
	"errors"
	"testing"

	"data-backup-manager/internal/destinationreadiness"
	"data-backup-manager/internal/engine"
	"data-backup-manager/internal/failures"
	"data-backup-manager/internal/sources"
)

type targetLookup struct{ targets map[string]Target }

func (l targetLookup) TargetForRun(_ context.Context, id string) (Target, error) {
	t, ok := l.targets[id]
	if !ok {
		return Target{}, errors.New("target not found")
	}
	return t, nil
}

type destLookup struct{ destinations map[string]Destination }

func (l destLookup) DestinationForRun(_ context.Context, id string) (Destination, error) {
	d, ok := l.destinations[id]
	if !ok {
		return Destination{}, errors.New("destination not found")
	}
	return d, nil
}

type readinessFake struct {
	report destinationreadiness.Report
	err    error
}

func (f readinessFake) Analyze(context.Context, destinationreadiness.AnalyzeInput) (destinationreadiness.Report, error) {
	return f.report, f.err
}

type engineFake struct {
	statusCalls int
	statusErr   error
}

func (e *engineFake) RepoCreate(context.Context, engine.RepoSpec) error { return nil }
func (e *engineFake) RepoStatus(context.Context, string) (engine.RepoStatus, error) {
	e.statusCalls++
	return engine.RepoStatus{}, e.statusErr
}
func (e *engineFake) PassphraseRef(string) string { return "secret-ref" }
func (e *engineFake) RepoStats(context.Context, string) (engine.RepoStats, error) {
	return engine.RepoStats{}, nil
}
func (e *engineFake) RepoDelete(context.Context, string) error { return nil }
func (e *engineFake) SnapshotCreate(context.Context, string, string, engine.SnapshotMetadata) (engine.Snapshot, error) {
	return engine.Snapshot{}, nil
}

func (e *engineFake) SnapshotList(context.Context, string, string) ([]engine.Snapshot, error) {
	return nil, nil
}
func (e *engineFake) SnapshotRestore(context.Context, string, string, string) error { return nil }
func (e *engineFake) SnapshotVerify(context.Context, string, string, int) error     { return nil }
func (e *engineFake) BrowseSnapshot(context.Context, string, string, string) ([]engine.SnapshotEntry, error) {
	return nil, nil
}
func (e *engineFake) PolicySet(context.Context, string, string, int) error { return nil }

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
	if got := r.Summary(); got == "" || len(got) > 300 {
		t.Fatalf("summary should be bounded and useful: %q", got)
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

type captureFake struct{ kind sources.SourceKind }

func (f *captureFake) Kind() sources.SourceKind { return f.kind }
func (f *captureFake) Capture(context.Context, sources.CaptureSpec) (sources.Artifact, error) {
	return sources.Artifact{}, nil
}
func (f *captureFake) Restore(context.Context, sources.RestoreSpec) error { return nil }
