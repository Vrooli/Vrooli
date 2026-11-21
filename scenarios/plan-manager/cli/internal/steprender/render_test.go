package steprender

import (
	"testing"

	"github.com/stretchr/testify/require"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/shared"
)

func TestStepLinesAndRecommendedActions(t *testing.T) {
	step := &sharedv1.GuidedStep{
		StepKind:       "phase_context",
		Summary:        "Use context",
		RequiredInputs: []string{"validation"},
		Instructions:   []string{"Run setup"},
		Examples:       []string{"plan-manager exec status e1"},
		NextActions: []*sharedv1.NextAction{
			{
				Kind:   sharedv1.NextActionKind_NEXT_ACTION_KIND_RECOMMENDED,
				Argv:   []string{"exec", "status", "e1"},
				Reason: "Fetch status",
			},
			{
				Kind:   sharedv1.NextActionKind_NEXT_ACTION_KIND_RECOVERY,
				Argv:   []string{"log", "note-add", "e1", "--title", "needs review"},
				Reason: "Capture note",
			},
		},
	}

	require.Equal(t, []string{"`exec status e1` — Fetch status"}, RecommendedActions(step))
	require.Contains(t, StepLines(step), "Required input: validation")
	require.Contains(t, StepLines(step), "Example: plan-manager exec status e1")
	require.Contains(t, StepLines(step), "Recovery: `log note-add e1 --title 'needs review'` — Capture note")
}

func TestShellQuote(t *testing.T) {
	require.Equal(t, "plain/path", ShellQuote("plain/path"))
	require.Equal(t, "'needs review'", ShellQuote("needs review"))
	require.Equal(t, "'it'\\''s'", ShellQuote("it's"))
}

func TestChecklistLine(t *testing.T) {
	require.Empty(t, ChecklistLine(nil))
	line := ChecklistLine([]*sharedv1.ChecklistItem{
		{Key: "steps", State: "filled", Detail: "2 step(s)"},
		{Key: "validation", State: "violation", Detail: "duplicates acceptance"},
		{Key: "relevant_context", State: "missing"},
	})
	require.Equal(t, "Checklist: ✔ steps · ✖ validation (duplicates acceptance) · – relevant_context", line)
}
