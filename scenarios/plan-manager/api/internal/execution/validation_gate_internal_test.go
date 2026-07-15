package execution

import (
	"strings"
	"testing"

	planmodel "plan-manager/internal/planmodel"
)

// TestValidationGateFailureNamesActualCondition pins the fix for
// knw-1784053356805823492: the gate rejected fresh, passing, execution-bound
// validations with the self-contradictory message "last validation staleness is
// fresh" because validationBlockerReason only ever reported verdict/staleness,
// never the provenance condition (wrong execution id, stale scope generation,
// not full inventory, missing scenario coverage) that actually failed. Every
// rejection message must name the condition that truly failed — never one that
// passed — and a satisfying validation must yield no blocker at all.
func TestValidationGateFailureNamesActualCondition(t *testing.T) {
	const execID = "exec-1"
	const gen = 3
	freshPass := ValidationResult{
		Verdict:         "pass",
		Staleness:       planmodel.StalenessFresh,
		ExecutionID:     execID,
		ScopeGeneration: gen,
		FullInventory:   true,
		SelectedMembers: []string{"bar", "foo"},
	}

	cases := []struct {
		name            string
		res             ValidationResult
		hasVal          bool
		requireFull     bool
		requiredMembers []string
		wantEmpty       bool
		wantContains    string
		wantNotContains string
	}{
		{
			name:            "fresh full-inventory pass satisfies completion gate",
			res:             freshPass,
			hasVal:          true,
			requireFull:     true,
			requiredMembers: nil,
			wantEmpty:       true,
		},
		{
			name: "fresh phase-scoped pass satisfies a phase transition (no full-inventory demand)",
			res: func() ValidationResult {
				r := freshPass
				r.FullInventory = false // phase-scoped ticket
				return r
			}(),
			hasVal:          true,
			requireFull:     false,
			requiredMembers: []string{"foo"},
			wantEmpty:       true,
		},
		{
			name:         "no stored result",
			res:          ValidationResult{},
			hasVal:       false,
			requireFull:  false,
			wantContains: "no stored validation result",
		},
		{
			name: "verdict not pass",
			res: func() ValidationResult {
				r := freshPass
				r.Verdict = "fail"
				return r
			}(),
			hasVal:       true,
			wantContains: "verdict is fail",
		},
		{
			name: "stale pass reports staleness (not a contradiction)",
			res: func() ValidationResult {
				r := freshPass
				r.Staleness = planmodel.StalenessDefinitelyStale
				return r
			}(),
			hasVal:       true,
			wantContains: "staleness is definitely_stale",
		},
		{
			name: "wrong execution id is named, not misreported as fresh staleness",
			res: func() ValidationResult {
				r := freshPass
				r.ExecutionID = "exec-OLD"
				return r
			}(),
			hasVal:          true,
			requireFull:     false,
			wantContains:    "bound to a different execution",
			wantNotContains: "staleness is fresh",
		},
		{
			name: "stale scope generation is named, not misreported as fresh staleness",
			res: func() ValidationResult {
				r := freshPass
				r.ScopeGeneration = gen - 1
				return r
			}(),
			hasVal:          true,
			requireFull:     false,
			wantContains:    "scope generation",
			wantNotContains: "staleness is fresh",
		},
		{
			name: "phase-scoped validation cannot satisfy full-inventory completion",
			res: func() ValidationResult {
				r := freshPass
				r.FullInventory = false
				return r
			}(),
			hasVal:          true,
			requireFull:     true,
			wantContains:    "full-inventory",
			wantNotContains: "staleness is fresh",
		},
		{
			name: "missing required scenario coverage is named",
			res: func() ValidationResult {
				r := freshPass
				r.SelectedMembers = []string{"foo"}
				return r
			}(),
			hasVal:          true,
			requireFull:     false,
			requiredMembers: []string{"foo", "bar"},
			wantContains:    "did not cover required scenario(s): bar",
			wantNotContains: "staleness is fresh",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := validationGateFailure(tc.res, tc.hasVal, tc.res.Staleness, execID, gen, tc.requireFull, tc.requiredMembers)
			if tc.wantEmpty {
				if got != "" {
					t.Fatalf("expected no blocker, got %q", got)
				}
				return
			}
			if !strings.Contains(got, tc.wantContains) {
				t.Fatalf("blocker %q does not contain %q", got, tc.wantContains)
			}
			if tc.wantNotContains != "" && strings.Contains(got, tc.wantNotContains) {
				t.Fatalf("blocker %q must NOT contain the passed-condition message %q", got, tc.wantNotContains)
			}
		})
	}
}

// TestValidationBlockerReasonEmptyOnFreshPass locks the invariant that a fresh
// passing validation yields no blocker string, so the message can never
// contradict itself by reporting a condition that actually held.
func TestValidationBlockerReasonEmptyOnFreshPass(t *testing.T) {
	res := ValidationResult{Verdict: "pass", Staleness: planmodel.StalenessFresh}
	if got := validationBlockerReason(res, true, planmodel.StalenessFresh); got != "" {
		t.Fatalf("fresh pass must produce no blocker, got %q", got)
	}
	if !validationIsRecentPass(res, true, planmodel.StalenessFresh) {
		t.Fatal("fresh pass must be recognized as a recent pass")
	}
}
