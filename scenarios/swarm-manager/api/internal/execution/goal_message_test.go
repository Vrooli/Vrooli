package execution

import (
	"errors"
	"strings"
	"testing"

	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/shared"
)

func TestComposeGoalMessageShapeARendersSlotsAndNoteVerbatim(t *testing.T) {
	note := "Focus on the goal path first.\nDo not touch the sibling plans."
	message, err := ComposeGoalMessage(GoalMessageInput{
		PlanShape:       "phased",
		PlanSlug:        "unattended-self-improvement-platform",
		PlanExecutionID: "exec-123",
		AcceptanceAllow: []string{"scenarios/swarm-manager/api/**"},
		ScopePolicy:     "extend-with-record",
		Budget:          "8000000 tokens / 2 hours",
		OperatorNote:    note,
	})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(message, "{{") || strings.Contains(message, "<slug>") || strings.Contains(message, "<acceptance_allow") {
		t.Fatalf("goal message still contains template syntax: %q", message)
	}
	if !strings.Contains(message, "Plan unattended-self-improvement-platform is complete to its intent") {
		t.Fatalf("destination slot missing: %q", message)
	}
	if !strings.HasSuffix(message, "Operator note: "+note) {
		t.Fatalf("operator note not byte-for-byte at the end: %q", message)
	}
	// Slot order: destination, read-first, proof, boundary, ... operator note.
	order := []string{"is complete to its intent", "Read first:", "Proof:", "Boundary:", "Adjacent defects:", "Quality:", "Blocked means", "Budget:", "Handoff:", "Non-goals:", "Operator note:"}
	last := -1
	for _, marker := range order {
		idx := strings.Index(message, marker)
		if idx < 0 || idx < last {
			t.Fatalf("slot %q out of order in %q", marker, message)
		}
		last = idx
	}
	if len([]rune(message)) > GoalPromptMaxChars {
		t.Fatalf("message exceeds the prompt bound: %d", len([]rune(message)))
	}
}

func TestComposeGoalMessageShapeBRendersMandate(t *testing.T) {
	message, err := ComposeGoalMessage(GoalMessageInput{
		PlanShape:    "mandate",
		ScenarioName: "audio-tools",
		ProgramName:  "audio-tools-improve",
		OperatorNote: "Keep it unattended.",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(message, "Bring every required row in audio-tools-improve in band") {
		t.Fatalf("mandate destination missing: %q", message)
	}
	if strings.Contains(message, "{{") || strings.Contains(message, "<...>") {
		t.Fatalf("mandate message contains template syntax: %q", message)
	}
	if !strings.HasSuffix(message, "Operator note: Keep it unattended.") {
		t.Fatalf("operator note missing: %q", message)
	}
}

func TestGoalPlanShapeClassifiesMandateFromSetpointProgram(t *testing.T) {
	plan := &sharedv1.Plan{
		TargetOutcome:    "Every required row of `program-runtime library run audio-tools.setpoint-read` reads in band.",
		DefinitionOfDone: "Bands are in TESTING.md.",
	}
	shape, scenario := goalPlanShape(plan)
	if shape != "mandate" {
		t.Fatalf("expected mandate, got %q", shape)
	}
	if scenario != "audio-tools" {
		t.Fatalf("expected scenario audio-tools, got %q", scenario)
	}
}

func TestGoalPlanShapeLeavesPhasedPlanPhased(t *testing.T) {
	plan := &sharedv1.Plan{
		TargetOutcome:    "The platform runs unattended.",
		DefinitionOfDone: "Every phase is done in Plan Manager with evidence.",
	}
	if shape, scenario := goalPlanShape(plan); shape != "phased" || scenario != "" {
		t.Fatalf("expected phased with no scenario, got %q/%q", shape, scenario)
	}
	if shape, _ := goalPlanShape(nil); shape != "phased" {
		t.Fatalf("nil plan must default to phased, got %q", shape)
	}
}

func TestComposeGoalMessageMandateCarriesBoardProofAndImproveSkill(t *testing.T) {
	message, err := ComposeGoalMessage(GoalMessageInput{
		PlanShape:       "mandate",
		PlanSlug:        "audio-tools-portable-voice-mandate",
		PlanExecutionID: "exec-123",
		ScenarioName:    "audio-tools",
		ProgramName:     "audio-tools-portable-voice-mandate",
		AcceptanceAllow: []string{"scenarios/audio-tools/**"},
		ScopePolicy:     "extend-with-record",
		OperatorNote:    "Run goal mode; one run.",
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"Bring every required row in audio-tools-improve in band on audio-tools.setpoint-read",
		"scenario-improvement-campaign, audio-tools-improve",
		"Evidence: use authorized local/simulated paths",
		"Never edit bands, thresholds, sensors, receipts, or board logic",
		"Authority: audio-tools-portable-voice-mandate grants development inside scenarios/audio-tools/**",
		"Mandate lane: use local/simulated paths",
	} {
		if !strings.Contains(message, want) {
			t.Fatalf("mandate message missing %q: %q", want, message)
		}
	}
	if strings.Contains(message, "implementation-plan-execution") {
		t.Fatalf("mandate message must not point at the phased receiving skill: %q", message)
	}
	if !strings.HasSuffix(message, "Operator note: Run goal mode; one run.") {
		t.Fatalf("operator note missing: %q", message)
	}
	if len([]rune(message)) > GoalPromptMaxChars {
		t.Fatalf("message exceeds the prompt bound: %d", len([]rune(message)))
	}
}

func TestComposeGoalMessageAllowsPromptLongerThanUntil(t *testing.T) {
	message, err := ComposeGoalMessage(GoalMessageInput{
		PlanShape:    "mandate",
		ScenarioName: "audio-tools",
		ProgramName:  "audio-tools-improve",
		OperatorNote: strings.Repeat("x", GoalUntilMaxChars),
	})
	if err != nil {
		t.Fatalf("long task prompt should be accepted independently of until: %v", err)
	}
	if len([]rune(message)) <= GoalUntilMaxChars {
		t.Fatalf("test prompt did not exceed until limit: %d", len([]rune(message)))
	}
}

func TestComposeGoalMessageFailsLoudlyWhenPromptTooLong(t *testing.T) {
	_, err := ComposeGoalMessage(GoalMessageInput{
		PlanShape:    "phased",
		PlanSlug:     "big",
		OperatorNote: strings.Repeat("x", GoalPromptMaxChars+1),
	})
	var tooLong *GoalMessageTooLongError
	if err == nil || !strings.Contains(err.Error(), "over the 16384-character limit") {
		t.Fatalf("expected a typed overflow error, got %v", err)
	}
	if !errors.As(err, &tooLong) {
		t.Fatalf("error is not *GoalMessageTooLongError: %T", err)
	}
}

func TestRenderFinishLineIsShortAndExplicit(t *testing.T) {
	line := RenderFinishLine(GoalMessageInput{PlanShape: "mandate", ScenarioName: "audio-tools"})
	if len([]rune(line)) > GoalUntilMaxChars {
		t.Fatalf("finish line exceeds Agent Manager until limit: %d", len([]rune(line)))
	}
	for _, want := range []string{"fresh owner-backed evidence", "exclusions are recorded", "evidence audit passes"} {
		if !strings.Contains(line, want) {
			t.Fatalf("finish line missing %q: %q", want, line)
		}
	}
}
