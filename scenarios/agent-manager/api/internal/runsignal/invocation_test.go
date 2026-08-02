package runsignal

import (
	"errors"
	"strings"
	"testing"

	"agent-manager/internal/domain"
	"github.com/google/uuid"
)

type assertErr struct{}

func (assertErr) Error() string { return "test failure" }

func TestDeriveInvocationFactsPairsAndRedacts(t *testing.T) {
	runID := uuid.New()
	call := domain.NewToolCallEvent(runID, "shell", "call-1", map[string]any{"command": "external-tool --token=top-secret"})
	result := domain.NewToolResultEvent(runID, "shell", "call-1", "", assertErr{})
	facts := DeriveInvocationFacts([]*domain.RunEvent{call, result})
	if len(facts) != 1 || facts[0].Outcome != "failure" || facts[0].ResultEventID != result.ID.String() {
		t.Fatalf("facts=%+v", facts)
	}
	if facts[0].PairingBasis != "tool_call_id" {
		t.Fatalf("pairing basis=%q", facts[0].PairingBasis)
	}
	if facts[0].Ownership != "external" || facts[0].Fingerprint == "" {
		t.Fatalf("fact=%+v", facts[0])
	}
}

func TestDeriveInvocationFactsPairsIdentifierlessTranscriptByOrdinal(t *testing.T) {
	runID := uuid.New()
	first := domain.NewToolCallEvent(runID, "shell", "", map[string]any{"command": "first"})
	second := domain.NewToolCallEvent(runID, "shell", "", map[string]any{"command": "second"})
	firstResult := domain.NewToolResultEvent(runID, "shell", "", "ok", nil)
	secondResult := domain.NewToolResultEvent(runID, "shell", "", "", assertErr{})
	facts := DeriveInvocationFacts([]*domain.RunEvent{first, second, firstResult, secondResult})
	if len(facts) != 2 {
		t.Fatalf("facts=%+v", facts)
	}
	if facts[0].ResultEventID != firstResult.ID.String() || facts[0].Outcome != "success" || facts[0].PairingBasis != "ordinal" {
		t.Fatalf("first fact=%+v", facts[0])
	}
	if facts[1].ResultEventID != secondResult.ID.String() || facts[1].Outcome != "failure" || facts[1].PairingBasis != "ordinal" {
		t.Fatalf("second fact=%+v", facts[1])
	}
}

func TestDeriveInvocationFactsPreservesUnpairedBasis(t *testing.T) {
	runID := uuid.New()
	facts := DeriveInvocationFacts([]*domain.RunEvent{domain.NewToolCallEvent(runID, "shell", "", map[string]any{"command": "missing"})})
	if len(facts) != 1 || facts[0].Outcome != "unknown" || facts[0].PairingBasis != "unpaired" {
		t.Fatalf("facts=%+v", facts)
	}
}

func TestDeriveInvocationFactsLabelsCompoundShell(t *testing.T) {
	runID := uuid.New()
	facts := DeriveInvocationFacts([]*domain.RunEvent{domain.NewToolCallEvent(runID, "shell", "call-1", map[string]any{"command": "agent-manager run report x && rm scratch"})})
	if len(facts) != 2 || facts[0].Ownership != OwnershipResolved || facts[1].Ownership != OwnershipExternal || facts[0].CallEventID != facts[1].CallEventID {
		t.Fatalf("fact=%+v", facts[0])
	}
}

func TestDeriveInvocationFactsDetectsHelpRecovery(t *testing.T) {
	runID := uuid.New()
	failed := domain.NewToolCallEvent(runID, "shell", "one", map[string]any{"command": "agent-manager"})
	failedResult := domain.NewToolResultEvent(runID, "shell", "one", "", assertErr{})
	help := domain.NewToolCallEvent(runID, "shell", "two", map[string]any{"command": "agent-manager --help"})
	retry := domain.NewToolCallEvent(runID, "shell", "three", map[string]any{"command": "agent-manager"})
	facts := DeriveInvocationFacts([]*domain.RunEvent{failed, failedResult, help, retry})
	if !facts[2].HelpRecovery || facts[2].RetryOfCallEventID != failed.ID.String() {
		t.Fatalf("facts=%+v", facts)
	}
}

func TestDeriveInvocationFactsFingerprintsArgumentShapeWithoutValues(t *testing.T) {
	runID := uuid.New()
	first := domain.NewToolCallEvent(runID, "shell", "one", map[string]any{"command": "sed -i s/old/new/ docs/guide.md"})
	second := domain.NewToolCallEvent(runID, "shell", "two", map[string]any{"command": "sed -i s/before/after/ api/main.go"})
	repeat := domain.NewToolCallEvent(runID, "shell", "three", map[string]any{"command": "sed -i s/old/new/ docs/guide.md"})
	facts := DeriveInvocationFacts([]*domain.RunEvent{first, second, repeat})
	if facts[0].Fingerprint == facts[1].Fingerprint {
		t.Fatalf("different edit shapes collapsed: %+v", facts)
	}
	if facts[0].Fingerprint != facts[2].Fingerprint {
		t.Fatalf("identical commands diverged: %+v", facts)
	}
	if facts[0].Executable != "sed" || facts[0].Ownership != "external" {
		t.Fatalf("external executable was not recorded: %+v", facts[0])
	}
}

func TestDeriveInvocationFactsKeepsUnparseableReason(t *testing.T) {
	runID := uuid.New()
	call := domain.NewToolCallEvent(runID, "shell", "one", map[string]any{"command": "git status $(pwd)"})
	fact := DeriveInvocationFacts([]*domain.RunEvent{call})[0]
	if fact.Ownership != OwnershipUnparseable || fact.OwnershipReason == "" || fact.Executable != "" {
		t.Fatalf("compound shell was promoted: %+v", fact)
	}
}

func TestNormalizedArgumentShapeOmitsArgumentValues(t *testing.T) {
	shape := normalizedArgumentShape(map[string]any{"command": "git commit --token=super-secret /home/alice/private.go"})
	for _, value := range []string{"super-secret", "/home/alice", "private.go"} {
		if strings.Contains(shape, value) {
			t.Fatalf("shape leaked %q: %q", value, shape)
		}
	}
}

func TestDeriveInvocationFactsClassifiesBoundedRedactedFailureSignatures(t *testing.T) {
	runID := uuid.New()
	missing := domain.NewToolCallEvent(runID, "shell", "missing", map[string]any{"command": "cat /tmp/x"})
	missingResult := domain.NewToolResultEvent(runID, "shell", "missing", "", errors.New("open /home/alice/token=secret: no such file or directory"))
	permission := domain.NewToolCallEvent(runID, "shell", "permission", map[string]any{"command": "cat /tmp/y"})
	permissionResult := domain.NewToolResultEvent(runID, "shell", "permission", "", errors.New("permission denied: /private/key"))
	exit := domain.NewToolCallEvent(runID, "shell", "exit", map[string]any{"command": "false"})
	exitResult := domain.NewToolResultEvent(runID, "shell", "exit", "", errors.New("command exited with code 2"))
	facts := DeriveInvocationFacts([]*domain.RunEvent{missing, missingResult, permission, permissionResult, exit, exitResult})
	want := []string{"missing_file", "permission_denied", "exit_code_2"}
	for i, signature := range want {
		if facts[i].FailureSignature != signature || facts[i].SignatureTruncated || len(signature) > FailureSignatureMaxLength {
			t.Fatalf("fact %d = %+v", i, facts[i])
		}
		for _, forbidden := range []string{"/home", "token", "secret", "/private"} {
			if strings.Contains(facts[i].FailureSignature, forbidden) {
				t.Fatalf("signature leaked %q: %q", forbidden, facts[i].FailureSignature)
			}
		}
	}
}
