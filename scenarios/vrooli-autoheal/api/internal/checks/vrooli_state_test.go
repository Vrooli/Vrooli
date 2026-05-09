package checks

import (
	"context"
	"errors"
	"strings"
	"testing"
)

type fakeCommandRunner struct {
	output []byte
	err    error
	calls  []string
}

func (f *fakeCommandRunner) CombinedOutput(_ context.Context, name string, args ...string) ([]byte, error) {
	f.calls = append(f.calls, name+" "+strings.Join(args, " "))
	return f.output, f.err
}

func TestVrooliCLIStateReaderListsRegistryClaimsAndLegacyLocks(t *testing.T) {
	runner := &fakeCommandRunner{output: []byte(`{
		"success": true,
		"registry_claims": [
			{"claim_id":"claim-alpha-api","instance_id":"inst-alpha","scenario":"alpha","port":15080,"claim_status":"bound","instance_status":"running"}
		],
		"locks": [
			{"port":15081,"scenario":"beta","pid":4100,"path":"/tmp/.port_15081.lock","timestamp":"2026-05-08T12:00:00Z"}
		]
	}`)}
	reader := NewVrooliCLIStateReader(runner)

	locks, err := reader.ListPortLocks()
	if err != nil {
		t.Fatalf("ListPortLocks: %v", err)
	}
	if len(locks) != 2 {
		t.Fatalf("locks = %#v", locks)
	}
	if locks[0].Source != "registry_claim" || locks[0].ClaimID != "claim-alpha-api" || locks[0].InstanceStatus != "running" {
		t.Fatalf("registry lock = %#v", locks[0])
	}
	if locks[1].Source != "legacy_lock" || locks[1].PID != 4100 || locks[1].Timestamp == 0 {
		t.Fatalf("legacy lock = %#v", locks[1])
	}
	if len(runner.calls) != 1 || runner.calls[0] != "vrooli locks --json" {
		t.Fatalf("calls = %#v", runner.calls)
	}
}

func TestFallbackVrooliStateReaderFallsBackWhenCLIUnavailable(t *testing.T) {
	tmp := t.TempDir()
	primary := NewVrooliCLIStateReader(&fakeCommandRunner{err: errors.New("vrooli missing")})
	fallback := NewRealVrooliStateReader(tmp)
	reader := NewFallbackVrooliStateReader(primary, fallback)

	locks, err := reader.ListPortLocks()
	if err != nil {
		t.Fatalf("ListPortLocks: %v", err)
	}
	if len(locks) != 0 {
		t.Fatalf("locks = %#v", locks)
	}
}

func TestVrooliCLIStateReaderRegistryCleanupDelegatesToCoreCleanup(t *testing.T) {
	runner := &fakeCommandRunner{}
	reader := NewVrooliCLIStateReader(runner)

	if err := reader.RemovePortLock(PortLock{Source: "registry_claim", ClaimID: "claim-alpha-api"}); err != nil {
		t.Fatalf("RemovePortLock: %v", err)
	}
	if len(runner.calls) != 1 || runner.calls[0] != "vrooli cleanup locks" {
		t.Fatalf("calls = %#v", runner.calls)
	}
}
