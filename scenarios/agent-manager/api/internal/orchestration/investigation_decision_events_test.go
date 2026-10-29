package orchestration

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"agent-manager/internal/domain"

	"github.com/google/uuid"
	"github.com/vrooli/api-core/operatordecision"
)

func investigationOutputFixture(t *testing.T, raw string) investigationDecisionOutput {
	t.Helper()
	var out investigationDecisionOutput
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		t.Fatal(err)
	}
	return out
}

func TestInvestigationDecisionAsksToApplyEveryRecommendation(t *testing.T) {
	ended := time.Date(2026, 10, 29, 12, 0, 0, 0, time.UTC)
	run := &domain.Run{ID: uuid.New(), CreatedAt: ended.Add(-time.Hour), EndedAt: &ended}
	out := investigationOutputFixture(t, `{"summary":"Runs stalled on a missing port.","categories":[{"name":"Env","recommendations":[{"text":"Declare the port"},{"text":"Add a health check"}]},{"name":"Docs","recommendations":[{"text":"  "}]}]}`)
	facts := investigationDecisionFacts(run, out, "https://console.example.test")
	if err := facts.Validate(); err != nil {
		t.Fatalf("facts invalid: %v", err)
	}
	if !strings.Contains(facts.Question, "Apply all 2 recommendation") || facts.Recommended != investigationApplyAllAnswer {
		t.Errorf("question %q / recommended %q", facts.Question, facts.Recommended)
	}
	for _, want := range []string{"1. Declare the port", "2. Add a health check"} {
		if !strings.Contains(facts.TechnicalDetails, want) {
			t.Errorf("details missing %q: %q", want, facts.TechnicalDetails)
		}
	}
	if !strings.Contains(facts.BigPicture, "Runs stalled on a missing port.") || !strings.Contains(facts.BigPicture, "choose them in Agent Manager") {
		t.Errorf("big picture %q", facts.BigPicture)
	}
	if facts.Correlation[investigationDecisionRunKey] != run.ID.String() || !strings.Contains(facts.ContextURL, run.ID.String()) {
		t.Errorf("correlation %v / link %q", facts.Correlation, facts.ContextURL)
	}
	first := operatordecision.EventID(investigationDecisionEvent(run, out, "https://console.example.test"))
	if again := operatordecision.EventID(investigationDecisionEvent(run, out, "https://console.example.test")); first != again {
		t.Fatal("repeated nudges must publish the same event id so the hub raises one ask")
	}
}

func TestInvestigationDecisionWithNoRecommendationsRecommendsClosing(t *testing.T) {
	run := &domain.Run{ID: uuid.New(), CreatedAt: time.Now()}
	facts := investigationDecisionFacts(run, investigationOutputFixture(t, `{"summary":"Nothing wrong.","categories":[]}`), "")
	if err := facts.Validate(); err != nil {
		t.Fatalf("facts invalid: %v", err)
	}
	if facts.Recommended != investigationRejectAnswer || facts.Options[0].Label != "Apply nothing" {
		t.Errorf("recommended %q options %+v", facts.Recommended, facts.Options)
	}
}

func TestInvestigationDecisionEffectActsOnlyOnAnAttributedAnswer(t *testing.T) {
	effect := investigationDecisionEffect{o: &Orchestrator{}}
	if !effect.Handles(InvestigationDecisionEventType) || effect.Handles(RunApprovalDecisionEventType) {
		t.Fatal("effect must claim only investigation decisions")
	}
	runID := uuid.NewString()
	for name, res := range map[string]operatordecision.DecisionResolution{
		"expired":        {State: operatordecision.StateExpired, Answer: investigationApplyAllAnswer, AnsweredBy: "owner", Correlation: map[string]string{investigationDecisionRunKey: runID}},
		"no approver":    {State: operatordecision.StateAnswered, Answer: investigationApplyAllAnswer, Correlation: map[string]string{investigationDecisionRunKey: runID}},
		"unknown answer": {State: operatordecision.StateAnswered, Answer: "maybe", AnsweredBy: "owner", Correlation: map[string]string{investigationDecisionRunKey: runID}},
		"no run":         {State: operatordecision.StateAnswered, Answer: investigationRejectAnswer, AnsweredBy: "owner"},
	} {
		// The orchestrator has no stores: reaching them would panic, so a pass
		// proves the answer was ignored before any effect.
		handled, err := effect.Apply(context.Background(), res)
		if !handled || err != nil {
			t.Errorf("%s: handled=%v err=%v, want a benign no-op", name, handled, err)
		}
	}
}

func TestInvestigationAwaitingApprovalIsAskedInTheInboxAndAnsweredThere(t *testing.T) {
	ctx := context.Background()
	o, repos, launcher := newDurableInvestigateOrchestrator(t)
	publisher := &capturingAuthPublisher{}
	o.approvalPublisher = publisher

	execution, err := o.StartWorkflowExecution(ctx, StartWorkflowExecutionRequest{Owner: "agent-manager", WorkflowKey: "agent-manager/investigate", Input: json.RawMessage(`{"context":"failed run"}`), IdempotencyKey: "inv-exec-" + t.Name()})
	if err != nil {
		t.Fatal(err)
	}
	investigateRun := waitForNodeRun(t, o.workflowExecutions, execution.ID, "investigate")
	findings := map[string]any{"summary": "The linter binary was missing.", "categories": []map[string]any{{"name": "Environment", "recommendations": []map[string]any{{"text": "Install golangci-lint."}, {"text": "Pin its version."}}}}}
	// Production records the validated result on the run row, which the
	// findings and the decision read.
	value, _ := json.Marshal(findings)
	stored, err := repos.Runs.Get(ctx, investigateRun)
	if err != nil || stored == nil {
		t.Fatalf("investigate run not stored: %v", err)
	}
	ended := time.Now().UTC()
	stored.EndedAt = &ended
	stored.Status, stored.Tag = domain.RunStatusComplete, "agent-manager-investigation"
	stored.Result = &domain.RunResult{Success: true, Structured: &domain.StructuredResult{Status: domain.StructuredResultSuccess, Value: value}}
	if err := repos.Runs.Update(ctx, stored); err != nil {
		t.Fatal(err)
	}
	launcher.fake.complete(investigateRun, findings)
	o.nudgeWorkflowForRun(investigateRun)
	waitForWaitNode(t, o, execution.ID, "await-approval")

	askRef := ""
	for _, event := range publisher.events {
		if event.EventType == InvestigationDecisionEventType {
			askRef = operatordecision.EventID(event)
		}
	}
	if askRef == "" {
		t.Fatal("an investigation waiting for approval raised no inbox decision")
	}

	answer := operatordecision.DecisionResolution{State: operatordecision.StateAnswered, Answer: investigationApplyAllAnswer, AnsweredBy: "owner", SourceEventType: InvestigationDecisionEventType, Correlation: map[string]string{investigationDecisionRunKey: investigateRun.String()}}
	effect := o.InvestigationDecisionEffect()
	if handled, err := effect.Apply(ctx, answer); !handled || err != nil {
		t.Fatalf("apply answer: handled=%v err=%v", handled, err)
	}
	applyRun := waitForNodeRun(t, o.workflowExecutions, execution.ID, "apply")
	withdrawn := ""
	for _, event := range publisher.events {
		if event.EventType == InvestigationDecisionWithdrawalEventType {
			withdrawn, _ = event.Payload["requester_ref"].(string)
		}
	}
	if withdrawn != askRef {
		t.Fatalf("decision withdrew %q, want the raised ask %q", withdrawn, askRef)
	}
	if prompt := launcher.fake.promptFor(applyRun); !strings.Contains(prompt, `"selected":["Install golangci-lint.","Pin its version."]`) {
		t.Fatalf("apply prompt does not carry every recommendation: %q", prompt)
	}
	// A redelivered answer finds the investigation decided and changes nothing.
	if handled, err := effect.Apply(ctx, answer); !handled || err != nil {
		t.Fatalf("redelivery: handled=%v err=%v", handled, err)
	}
}
