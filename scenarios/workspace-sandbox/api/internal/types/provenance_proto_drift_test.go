package types

import (
	"strings"
	"testing"

	wsdomain "github.com/vrooli/vrooli/packages/proto/gen/go/workspace-sandbox/v1/domain"
)

// TestProvenanceFileStateMatchesProto pins the Go ProvenanceFileState
// constants to the proto enum at
// packages/proto/schemas/workspace-sandbox/v1/domain/applied_change.proto.
//
// The proto file is the cross-scenario contract; this test fails loud if a
// future edit to the .proto enum drops or renames a value without updating
// the Go strings (or vice versa). The Go strings are the kebab-cased
// projection of the proto enum names: FILE_STATE_PENDING_REVIEW becomes
// "pending-review", etc.
func TestProvenanceFileStateMatchesProto(t *testing.T) {
	cases := []struct {
		proto wsdomain.FileState
		want  ProvenanceFileState
	}{
		{wsdomain.FileState_FILE_STATE_APPLIED, ProvenanceFileStateApplied},
		{wsdomain.FileState_FILE_STATE_PENDING_REVIEW, ProvenanceFileStatePendingReview},
		{wsdomain.FileState_FILE_STATE_DENIED, ProvenanceFileStateDenied},
	}
	for _, tc := range cases {
		got := protoEnumToWire(tc.proto.String(), "FILE_STATE_")
		if ProvenanceFileState(got) != tc.want {
			t.Errorf("proto %s → %q, want %q", tc.proto, got, tc.want)
		}
	}

	// Also verify the count: if someone adds a new FileState to proto and
	// forgets to add a Go constant, this catches it.
	const expectedNonZero = 3 // APPLIED, PENDING_REVIEW, DENIED
	count := 0
	for v := range wsdomain.FileState_name {
		if v != 0 {
			count++
		}
	}
	if count != expectedNonZero {
		t.Errorf("proto FileState has %d non-zero values, expected %d — update ProvenanceFileState constants in types.go", count, expectedNonZero)
	}
}

// TestRunOutcomeMatchesProto pins the workspace-sandbox wire values for
// run_outcome to the proto enum. workspace-sandbox stores RunOutcome as a
// plain string field (the agent-manager wire shape predates the proto
// schema), so this test verifies that the kebab-cased projection of every
// proto value is one a writer might legitimately emit.
func TestRunOutcomeMatchesProto(t *testing.T) {
	wantSet := map[string]bool{
		"success":   true,
		"failure":   true,
		"cancelled": true,
		"timeout":   true,
	}
	gotSet := map[string]bool{}
	for v, name := range wsdomain.RunOutcome_name {
		if v == 0 {
			continue
		}
		gotSet[protoEnumToWire(name, "RUN_OUTCOME_")] = true
	}
	for k := range wantSet {
		if !gotSet[k] {
			t.Errorf("proto RunOutcome missing wire value %q (drift between proto and writer)", k)
		}
	}
	for k := range gotSet {
		if !wantSet[k] {
			t.Errorf("proto RunOutcome has new wire value %q — agent-manager apply-at-run-end caller must emit it", k)
		}
	}
}

// protoEnumToWire converts a SCREAMING_SNAKE proto enum value name like
// "FILE_STATE_PENDING_REVIEW" into the kebab-cased wire string
// "pending-review", stripping the prefix.
func protoEnumToWire(name, prefix string) string {
	trimmed := strings.TrimPrefix(name, prefix)
	return strings.ToLower(strings.ReplaceAll(trimmed, "_", "-"))
}
