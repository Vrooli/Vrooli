package integration_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	internalauthoring "plan-manager/internal/authoring"
	internalexecution "plan-manager/internal/execution"
	internalplanlog "plan-manager/internal/planlog"
	internalplans "plan-manager/internal/plans"

	"github.com/stretchr/testify/require"
)

// manifestIndex is group -> set of command names, parsed from cli/manifest.json.
type manifestIndex map[string]map[string]bool

func loadManifestIndex(t *testing.T) manifestIndex {
	t.Helper()
	path := filepath.Join("..", "..", "..", "cli", "manifest.json")
	raw, err := os.ReadFile(path)
	require.NoError(t, err, "read cli manifest")
	var m struct {
		Groups []struct {
			Name     string `json:"name"`
			Commands []struct {
				Name string `json:"name"`
			} `json:"commands"`
		} `json:"groups"`
	}
	require.NoError(t, json.Unmarshal(raw, &m))
	idx := manifestIndex{}
	for _, g := range m.Groups {
		idx[g.Name] = map[string]bool{}
		for _, c := range g.Commands {
			idx[g.Name][c.Name] = true
		}
	}
	require.NotEmpty(t, idx, "manifest must declare groups")
	return idx
}

// requireValidArgv asserts argv[0] is a real manifest group and argv[1] is a real
// command in that group — so no guided step ever recommends a command absent from
// the CLI manifest.
func (idx manifestIndex) requireValidArgv(t *testing.T, where string, argv []string) {
	t.Helper()
	if len(argv) == 0 {
		return
	}
	group := argv[0]
	cmds, ok := idx[group]
	require.Truef(t, ok, "%s: guided argv references unknown CLI group %q (argv=%v)", where, group, argv)
	require.GreaterOrEqualf(t, len(argv), 2, "%s: guided argv %v has a group but no command", where, argv)
	command := argv[1]
	require.Truef(t, cmds[command], "%s: guided argv references unknown command %q in group %q (argv=%v)", where, command, group, argv)
}

// validateExecStep validates every NextAction argv on an execution GuidedStep.
func (idx manifestIndex) validateExecStep(t *testing.T, where string, step internalexecution.GuidedStep) {
	for _, a := range step.NextActions {
		idx.requireValidArgv(t, where+" step="+step.StepKind, a.Argv)
	}
}

func (idx manifestIndex) validateAuthorStep(t *testing.T, where string, step internalauthoring.GuidedStep) {
	for _, a := range step.NextActions {
		idx.requireValidArgv(t, where+" step="+step.StepKind, a.Argv)
	}
}

func (idx manifestIndex) validateLogStep(t *testing.T, where string, step internalplanlog.GuidedStep) {
	for _, a := range step.NextActions {
		idx.requireValidArgv(t, where+" step="+step.StepKind, a.Argv)
	}
}

// TestGuidedArgvAreValidManifestCommands is the anti-drift guard the plan
// requires: every guided NextAction.argv emitted by the authoring, execution, and
// log wizards must reference a real CLI manifest command. It exercises the real
// services so future command renames or new steps fail this test immediately.
func TestGuidedArgvAreValidManifestCommands(t *testing.T) {
	ctx := context.Background()
	idx := loadManifestIndex(t)
	_, plansSvc, _, authoringSvc, executionSvc, logSvc := newStack(t)

	// --- authoring: drive the continue loop, validating every step's argv. ---
	session, startStep, err := authoringSvc.StartSession(ctx, "Argv guard", "argv-guard", "")
	require.NoError(t, err)
	idx.validateAuthorStep(t, "author StartSession", startStep)

	for guard := 0; guard < 30; guard++ {
		sess, section, phase, ready, _, step, cerr := authoringSvc.ContinueAuthoring(ctx, session.ID)
		require.NoError(t, cerr)
		session = sess
		idx.validateAuthorStep(t, "author continue", step)
		if ready {
			break
		}
		switch step.StepKind {
		case "purpose", "scope", "regression_anchor", "definition_of_done":
			_, _, _, err = authoringSvc.SubmitSection(ctx, session.ID, section.Key, "Guard fixture content for "+string(section.Key)+".")
			require.NoError(t, err)
		case "global_relevant_context":
			_, _, _, err = authoringSvc.SubmitSection(ctx, session.ID, internalauthoring.SectionRelevantContext, "NO_CONTEXT: guard fixture needs no plan-wide setup.")
			require.NoError(t, err)
		case "phase_outline":
			_, p, _, _, aerr := authoringSvc.AddPhase(ctx, session.ID, "Implement", "Do the work.")
			require.NoError(t, aerr)
			phase = p
			_ = phase
		case "phase_references":
			_, _, _, err = authoringSvc.SubmitPhaseField(ctx, session.ID, phase.ID, internalauthoring.PhaseFieldReferences, "[CODE: scenarios/plan-manager/api/main.go]")
			require.NoError(t, err)
		case "phase_acceptance":
			_, _, _, err = authoringSvc.SubmitPhaseField(ctx, session.ID, phase.ID, internalauthoring.PhaseFieldAcceptance, "Tests pass.")
			require.NoError(t, err)
		case "phase_relevant_context":
			_, _, _, err = authoringSvc.SubmitPhaseField(ctx, session.ID, phase.ID, internalauthoring.PhaseFieldRelevantContext, "NO_CONTEXT: phase needs no extra setup.")
			require.NoError(t, err)
		case "validation_recovery":
			_, _, _, err = authoringSvc.SubmitSection(ctx, session.ID, internalauthoring.SectionReferences, "NO_CODE_REFS: guard fixture.")
			require.NoError(t, err)
		case "final_review":
			_, fstep, ferr := authoringSvc.Finalize(ctx, session.ID)
			require.NoError(t, ferr)
			idx.validateAuthorStep(t, "author finalize", fstep)
		default:
			t.Fatalf("argv guard: unhandled authoring step %q", step.StepKind)
		}
	}

	// --- execution: drive start/status/transition/complete, validating argv. ---
	plan, err := plansSvc.Create(ctx, internalplans.Plan{
		Title: "Argv exec",
		RelevantContext: []internalplans.RelevantContextItem{
			{Kind: internalplans.RelevantContextCommand, Label: "setup", RepeatPolicy: internalplans.RelevantContextOncePerExecution},
		},
		Phases: []internalplans.Phase{
			{Title: "One", Acceptance: "done", Status: internalplans.PhaseStatusTodo},
			{Title: "Two", Acceptance: "done", Status: internalplans.PhaseStatusTodo},
		},
	})
	require.NoError(t, err)

	exec, _, contStep, err := executionSvc.ContinueExecution(ctx, plan.ID, "", "run-argv")
	require.NoError(t, err)
	idx.validateExecStep(t, "exec continue", contStep)

	_, _, statusStep, err := executionSvc.GetStatus(ctx, exec.ID)
	require.NoError(t, err)
	idx.validateExecStep(t, "exec status", statusStep)

	for _, ph := range plan.Phases {
		_, _, tstep, terr := executionSvc.TransitionPhase(ctx, exec.ID, ph.ID, internalexecution.PhaseTransitionInputs{
			ToStatus: internalplans.PhaseStatusDone, ValidationOverrideReason: "argv guard fixture",
		})
		require.NoError(t, terr)
		idx.validateExecStep(t, "exec transition", tstep)
	}

	_, _, completeStep, err := executionSvc.Complete(ctx, exec.ID, internalexecution.CompletionInputs{})
	require.NoError(t, err)
	idx.validateExecStep(t, "exec complete", completeStep)

	_, handoffStep, err := executionSvc.GetHandoff(ctx, exec.ID)
	require.NoError(t, err)
	idx.validateExecStep(t, "exec handoff", handoffStep)

	// --- log: drive add/list/promote/sync, validating argv. ---
	_, _, decStep, err := logSvc.AddDecision(ctx, internalplanlog.AddInputs{PlanOrExecution: exec.ID, Title: "argv decision"})
	require.NoError(t, err)
	idx.validateLogStep(t, "log decision-add", decStep)

	finding, _, fStep, err := logSvc.AddFinding(ctx, internalplanlog.AddInputs{PlanOrExecution: exec.ID, Title: "argv finding"})
	require.NoError(t, err)
	idx.validateLogStep(t, "log finding-add", fStep)

	_, _, bugStep, err := logSvc.AddBug(ctx, internalplanlog.AddInputs{PlanOrExecution: exec.ID, Title: "argv bug"})
	require.NoError(t, err)
	idx.validateLogStep(t, "log bug-add", bugStep)

	_, _, listStep, err := logSvc.ListEntries(ctx, internalplanlog.Filter{ExecutionID: exec.ID})
	require.NoError(t, err)
	idx.validateLogStep(t, "log list", listStep)

	_, _, promStep, err := logSvc.PromoteEntry(ctx, finding.ID, internalplanlog.EntryType("record"), "", "", "")
	require.NoError(t, err)
	idx.validateLogStep(t, "log promote", promStep)
}
