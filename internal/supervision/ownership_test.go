package supervision

import (
	"errors"
	"testing"
	"time"
)

type staticOwners []Owner

func (s staticOwners) Owners() ([]Owner, error) { return s, nil }

type staticProcesses map[int]ProcessInfo

func (s staticProcesses) Processes() (map[int]ProcessInfo, error) { return s, nil }

type failingProcesses struct{ err error }

func (s failingProcesses) Processes() (map[int]ProcessInfo, error) { return nil, s.err }

func TestBuildIndexGivenLiveOwnedDaemonThenOwnerResolves(t *testing.T) {
	started := time.Date(2026, 8, 29, 0, 0, 0, 0, time.UTC)
	index, err := BuildIndex(
		staticProcesses{4242: {PID: 4242, StartedAt: started}},
		staticOwners{{Kind: OwnerKindResource, Name: "ollama", PID: 4242, StartedAt: started.Add(time.Second)}},
	)
	if err != nil {
		t.Fatalf("BuildIndex: %v", err)
	}
	owner, ok := index.Owner(4242)
	if !ok || owner.Name != "ollama" {
		t.Fatalf("Owner(4242) = %#v, %v", owner, ok)
	}
}

func TestBuildIndexGivenReusedPIDThenStaleRecordDoesNotProtectIt(t *testing.T) {
	processStarted := time.Date(2026, 8, 29, 0, 0, 0, 0, time.UTC)
	index, err := BuildIndex(
		staticProcesses{4244: {PID: 4244, StartedAt: processStarted}},
		staticOwners{{Kind: OwnerKindResource, Name: "ollama", PID: 4244, StartedAt: processStarted.Add(time.Minute)}},
	)
	if err != nil {
		t.Fatalf("BuildIndex: %v", err)
	}
	if owner, ok := index.Owner(4244); ok {
		t.Fatalf("stale record protected reused pid: %#v", owner)
	}
}

func TestBuildIndexGivenDeadPIDThenRecordIsIgnored(t *testing.T) {
	index, err := BuildIndex(staticProcesses{}, staticOwners{{Kind: OwnerKindResource, Name: "redis", PID: 4245, StartedAt: time.Now()}})
	if err != nil {
		t.Fatalf("BuildIndex: %v", err)
	}
	if _, ok := index.Owner(4245); ok {
		t.Fatal("dead pid unexpectedly resolved")
	}
}

func TestBuildIndexGivenUnsupportedProcessEvidenceThenTypedErrorIsPreserved(t *testing.T) {
	sourceErr := &UnsupportedProcessEvidenceError{Platform: "windows"}
	_, err := BuildIndex(failingProcesses{err: sourceErr}, staticOwners{})
	if !errors.Is(err, sourceErr) || !IsUnsupportedProcessEvidence(err) {
		t.Fatalf("error = %v, want typed unsupported evidence", err)
	}
}
