package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	clitest "swarm-manager/cli/internal/testutil"
)

// `proposals get` is the fallback surface when the UI is unavailable, and it
// used to print `m1  update_item` — no target, no rationale, no payload. These
// tests pin the payload to the output so it cannot silently thin out again.

func renderOne(t *testing.T, mutation cliMutation) string {
	t.Helper()
	return clitest.CaptureStdout(t, func() error {
		renderMutation(mutation)
		return nil
	})
}

func requireContains(t *testing.T, out string, wants ...string) {
	t.Helper()
	for _, want := range wants {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q\n--- got ---\n%s", want, out)
		}
	}
}

func TestRenderMutationShowsCreatedItemInsteadOfEmptyTarget(t *testing.T) {
	out := renderOne(t, cliMutation{
		ID: "m1",
		Op: "add_item",
		Item: &cliItemSpec{
			Kind:            "execute",
			Name:            "capture-health",
			Title:           "Operators can inspect capture classification health",
			Description:     "Add a read-only health command.",
			Priority:        4,
			Effort:          "M",
			Tags:            []string{"cli", "captures"},
			AcceptanceAllow: []string{"shows the capture id"},
		},
	})
	// add_item carries no Target by design; the ref must come from the spec.
	requireContains(t, out,
		"execute/capture-health",
		"Operators can inspect capture classification health",
		"priority 4",
		"effort M",
		"Add a read-only health command.",
		"acceptance 1: shows the capture id",
	)
	if strings.Contains(out, "(no target)") {
		t.Errorf("add_item rendered a blank target:\n%s", out)
	}
}

func TestRenderMutationShowsPatchedFieldsAndValues(t *testing.T) {
	out := renderOne(t, cliMutation{
		ID:        "m1",
		Op:        "update_item",
		Target:    "execute/remove-redundant-dbm-self-registration",
		Rationale: "Read-only inspection found register.go still present.",
		Patch:     map[string]any{"description": "Remove the remaining package and tests.", "priority": float64(6)},
	})
	requireContains(t, out,
		"execute/remove-redundant-dbm-self-registration",
		"why: Read-only inspection found register.go still present.",
		"patch: description, priority",
		"description = Remove the remaining package and tests.",
		"priority = 6",
	)
}

func TestRenderMutationMarksAnExplicitClear(t *testing.T) {
	// An empty value in a patch is a clear, not an absent field.
	out := renderOne(t, cliMutation{ID: "m1", Op: "update_item", Target: "execute/a", Patch: map[string]any{"note": ""}})
	requireContains(t, out, "note = (cleared)")
}

func TestRenderMutationNamesPatchFieldsThisBuildPredates(t *testing.T) {
	out := renderOne(t, cliMutation{ID: "m1", Op: "update_item", Target: "execute/a", Patch: map[string]any{"future_field": "x"}})
	requireContains(t, out, "future_field")
}

func TestRenderMutationDescribesDetachAsAnAction(t *testing.T) {
	out := renderOne(t, cliMutation{ID: "m1", Op: "move_milestone", Target: "execute/a", Milestone: ""})
	requireContains(t, out, "milestone -> (detach from milestone)")
}

func TestRenderMutationShowsBothEdgeEndpoints(t *testing.T) {
	out := renderOne(t, cliMutation{ID: "m1", Op: "add_edge", From: "execute/a", To: "execute/b"})
	requireContains(t, out, "edge: execute/a depends on execute/b")
}

func TestRenderMutationFlagsAHalfSpecifiedEdge(t *testing.T) {
	out := renderOne(t, cliMutation{ID: "m1", Op: "add_edge", From: "execute/a"})
	requireContains(t, out, "(unset)")
}

func TestRenderMutationListsArchivedMergeSources(t *testing.T) {
	out := renderOne(t, cliMutation{
		ID:      "m1",
		Op:      "merge_items",
		Sources: []string{"execute/a", "execute/b"},
		Item:    &cliItemSpec{Kind: "execute", Name: "merged", Title: "Merged"},
	})
	requireContains(t, out, "source (archived): execute/a", "source (archived): execute/b", "execute/merged")
}

func TestRenderMutationListsResetScopes(t *testing.T) {
	out := renderOne(t, cliMutation{ID: "m1", Op: "reset_artifacts", Target: "execute/a", ResetScope: []string{"review", "plan_unbind"}})
	requireContains(t, out, "removes: review, plan_unbind")
}

func TestRenderMutationSaysNoTargetRatherThanPrintingNothing(t *testing.T) {
	out := renderOne(t, cliMutation{ID: "m1", Op: "add_item"})
	requireContains(t, out, "(no target)")
}

func TestPatchedFieldNamesIsStableAcrossCalls(t *testing.T) {
	// Ranging a map directly reorders output every run, which makes the
	// command useless for diffing two proposals.
	patch := map[string]any{"tags": []any{"a"}, "title": "t", "priority": float64(1), "note": "n"}
	first := strings.Join(patchedFieldNames(patch), ",")
	for range 20 {
		if got := strings.Join(patchedFieldNames(patch), ","); got != first {
			t.Fatalf("field order unstable: %q then %q", first, got)
		}
	}
	if first != "title,note,priority,tags" {
		t.Errorf("field order = %q, want declaration order title,note,priority,tags", first)
	}
}

func TestFormatPatchValueRendersEachJSONShape(t *testing.T) {
	cases := map[string]struct {
		value any
		want  string
	}{
		"string":       {"hello", "hello"},
		"empty string": {"", "(cleared)"},
		"null":         {nil, "(cleared)"},
		"number":       {float64(6), "6"},
		"list":         {[]any{"a", "b"}, "a, b"},
		"empty list":   {[]any{}, "(cleared)"},
	}
	for name, testCase := range cases {
		t.Run(name, func(t *testing.T) {
			if got := formatPatchValue(testCase.value); got != testCase.want {
				t.Errorf("formatPatchValue(%v) = %q, want %q", testCase.value, got, testCase.want)
			}
		})
	}
}

// realProposalPayload is prop_135298b80000c4bd as stored on 2026-08-09: the
// proposal whose new description the decision surface never displayed. It is
// embedded verbatim so the regression is pinned to real data rather than to a
// fixture that could drift into agreeing with the bug.
const realProposalPayload = `{
  "form": "mutation_list",
  "rationale": "The backlog item still targets real redundant swarm-manager DBM self-registration code, but current code drift means the production boot call no longer exists; refreshing the item avoids sending execution agents to remove a nonexistent startup hook and adds the stale storage-audit doc cleanup.",
  "mutations": [
    {
      "id": "m1",
      "op": "update_item",
      "target": "execute/remove-redundant-dbm-self-registration",
      "patch": {
        "description": "Remove swarm-manager's remaining data-backup-manager self-registration package and tests: ` + "`scenarios/swarm-manager/api/internal/backup/register.go`" + ` and ` + "`scenarios/swarm-manager/api/internal/backup/register_test.go`" + `. Also update stale swarm-manager storage documentation that still claims ` + "`internal/backup.EnsureBackupTargets`" + ` self-registers at boot."
      },
      "rationale": "Read-only inspection found register.go and its tests still present, no non-test references to EnsureBackupTargets outside that package."
    }
  ]
}`

func TestRealProposalPayloadRendersItsNewDescription(t *testing.T) {
	var payload mutationSummary
	if err := json.Unmarshal([]byte(realProposalPayload), &payload); err != nil {
		t.Fatalf("decode real payload: %v", err)
	}
	if len(payload.Mutations) != 1 {
		t.Fatalf("mutations = %d, want 1", len(payload.Mutations))
	}
	out := clitest.CaptureStdout(t, func() error {
		if payload.Rationale != "" {
			fmt.Printf("  Rationale: %s\n", payload.Rationale)
		}
		for _, mutation := range payload.Mutations {
			renderMutation(mutation)
		}
		return nil
	})

	requireContains(t, out,
		"execute/remove-redundant-dbm-self-registration",
		"patch: description",
		// The substantive change: the boot-hook clause is gone and a doc
		// cleanup is added. Neither was visible on any surface before.
		"Remove swarm-manager's remaining data-backup-manager self-registration package",
		"still claims",
		"Read-only inspection found register.go",
	)
	// The pre-fix output was exactly "    m1  update_item" and nothing else.
	if strings.Count(out, "\n") < 4 {
		t.Errorf("output collapsed back to a one-line summary:\n%s", out)
	}
}

func TestTruncateDoesNotSplitMultiByteCharacters(t *testing.T) {
	// Byte slicing here emitted replacement glyphs for any em dash or
	// accented character that straddled the cut.
	got := truncate("aaa—bbb", 4)
	if !strings.HasPrefix(got, "aaa—") {
		t.Errorf("truncate mangled a multi-byte rune: %q", got)
	}
	if strings.ContainsRune(got, '�') {
		t.Errorf("truncate produced a replacement character: %q", got)
	}
}
