package main

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/vrooli/vrooli/packages/proto/descriptorimage"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
	domain "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-events/v1/domain"
	"github.com/vrooli/vrooli/scenarios/vrooli-events/internal/policy"
	"github.com/vrooli/vrooli/scenarios/vrooli-events/internal/store"
	"google.golang.org/protobuf/proto"
)

func TestReceiptCapturePolicySnapshotUsesTypedContract(t *testing.T) {
	_, ts := newTestServer(t)
	body := `{"policy_id":"plan-manager-create-plan","enabled":true,"selector":{"target_scenario":"plan-manager","operation":"POST /vrooli.plan_manager.v1.plans.PlansService/CreatePlan","protocol":"connect","event_type":"vrooli.events.receipt.v1"},"response_type":"vrooli.plan_manager.v1.plans.CreatePlanResponse","response_projection_paths":["plan.id"],"retention_days":30,"access":{"read_principals":["agent-manager"]}}`
	resp, err := http.Post(ts.URL+"/api/v1/receipt-capture-policies", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create status=%d", resp.StatusCode)
	}
	resp, err = http.Post(ts.URL+"/api/v1/receipt-capture-policies", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("reconcile status=%d", resp.StatusCode)
	}
	resp, err = http.Get(ts.URL + "/api/v1/policies/snapshot")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var snapshot struct {
		Version  string `json:"version"`
		Policies []struct {
			PolicyID string `json:"policy_id"`
			Selector struct {
				Target    string `json:"target_scenario"`
				Operation string `json:"operation"`
				Protocol  string `json:"protocol"`
				EventType string `json:"event_type"`
			} `json:"selector"`
			ResponseType string   `json:"response_type"`
			Paths        []string `json:"response_projection_paths"`
		} `json:"receipt_capture_policies"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&snapshot); err != nil {
		t.Fatal(err)
	}
	if snapshot.Version == "" || len(snapshot.Policies) != 1 {
		t.Fatalf("snapshot=%+v", snapshot)
	}
	p := snapshot.Policies[0]
	if p.PolicyID != "plan-manager-create-plan" || p.Selector.Target != "plan-manager" || p.Selector.Operation != "POST /vrooli.plan_manager.v1.plans.PlansService/CreatePlan" || p.Selector.Protocol != "connect" || p.Selector.EventType != receiptEventType || p.ResponseType != "vrooli.plan_manager.v1.plans.CreatePlanResponse" || len(p.Paths) != 1 || p.Paths[0] != "plan.id" {
		t.Fatalf("policy=%+v", p)
	}
}

func TestReceiptCaptureDeclarationReconcileIsIdempotent(t *testing.T) {
	_, ts := newTestServer(t)
	repoRoot, err := filepath.Abs("../../..")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(repoRoot, "scenarios", "agent-manager", ".vrooli", "service.json")); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PROJECT_ROOT", repoRoot)
	body := `{"scenario":"agent-manager","dryRun":true}`
	response, err := http.Post(ts.URL+"/api/v1/receipt-capture-policies/reconcile", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("dry-run status=%d", response.StatusCode)
	}
	response, err = http.Post(ts.URL+"/api/v1/receipt-capture-policies/reconcile", "application/json", strings.NewReader(`{"scenario":"agent-manager"}`))
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("first reconcile status=%d", response.StatusCode)
	}
	response, err = http.Post(ts.URL+"/api/v1/receipt-capture-policies/reconcile", "application/json", strings.NewReader(`{"scenario":"agent-manager"}`))
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("repeat reconcile status=%d", response.StatusCode)
	}
}

func TestValidateDeclaredResponseRejectsUndeclaredProjection(t *testing.T) {
	repoRoot, err := filepath.Abs("../../..")
	if err != nil {
		t.Fatal(err)
	}
	if err := validateDeclaredResponse(repoRoot, "POST /agent_manager.v1.AgentManagerService/CreateRun", "agent_manager.v1.CreateRunResponse", []string{"run.id"}); err != nil {
		t.Fatalf("valid declaration rejected: %v", err)
	}
	if err := validateDeclaredResponse(repoRoot, "POST /agent_manager.v1.AgentManagerService/CreateRun", "agent_manager.v1.CreateRunResponse", []string{"run.not_a_field"}); err == nil {
		t.Fatal("undeclared projection was accepted")
	}
	if err := validateDeclaredResponse(repoRoot, "POST /agent_manager.v1.AgentManagerService/CreateRun", "agent_manager.v1.StopRunResponse", []string{"status"}); err == nil {
		t.Fatal("mismatched response type was accepted")
	}
}

func BenchmarkValidateDeclaredResponseWithSnapshot(b *testing.B) {
	repoRoot, err := filepath.Abs("../../..")
	if err != nil {
		b.Fatal(err)
	}
	source, err := descriptorimage.NewForRepo(repoRoot)
	if err != nil {
		b.Fatal(err)
	}
	snapshot, err := source.Snapshot()
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := validateDeclaredResponseWithSnapshot(repoRoot, "POST /agent_manager.v1.AgentManagerService/CreateRun", "agent_manager.v1.CreateRunResponse", []string{"run.id"}, snapshot); err != nil {
			b.Fatal(err)
		}
	}
}

func TestCaptureValidationRequiresReconciledPolicy(t *testing.T) {
	srv, _ := newTestServer(t)
	repoRoot, err := filepath.Abs("../../..")
	if err != nil {
		t.Fatal(err)
	}
	handler := newCaptureValidationHandler(repoRoot, srv.policyStore, srv.store)
	response, err := handler.ValidateScenario(context.Background(), connect.NewRequest(&scenariovalidationv1.ValidateScenarioRequest{Scenario: "agent-manager"}))
	if err != nil {
		t.Fatal(err)
	}
	if response.Msg.GetStatus() != scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_FAILED {
		t.Fatalf("unreconciled validation status = %s", response.Msg.GetStatus())
	}
	if response.Msg.GetMetrics() == nil || response.Msg.GetAssessment().GetFindings()[0].GetMaturity() == nil {
		t.Fatalf("validation response lacks provider contract metadata: %#v", response.Msg)
	}
	rules, err := loadCaptureDeclarationRulesAtRoot(repoRoot, "agent-manager")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := srv.policyStore.ReconcileReceiptProjections(context.Background(), rules); err != nil {
		t.Fatal(err)
	}
	response, err = handler.ValidateScenario(context.Background(), connect.NewRequest(&scenariovalidationv1.ValidateScenarioRequest{Scenario: "agent-manager"}))
	if err != nil {
		t.Fatal(err)
	}
	if response.Msg.GetStatus() != scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_FAILED {
		t.Fatalf("unexercised validation status = %s, want failed", response.Msg.GetStatus())
	}
	findings := response.Msg.GetAssessment().GetFindings()
	if len(findings) != 1 || findings[0].GetCode() != "event_capture.policy_unexercised" || !strings.Contains(findings[0].GetMessage(), "agent-manager-search-hub-query") {
		t.Fatalf("unexercised validation findings = %#v", findings)
	}
	for _, rule := range rules {
		body, marshalErr := proto.Marshal(&domain.EventEnvelope{EventId: "receipt-" + rule.PolicyID, EventType: receiptEventType, Target: &domain.EventTarget{Scenario: rule.TargetScenario, Operation: rule.OperationPattern, Protocol: rule.Protocol}})
		if marshalErr != nil {
			t.Fatal(marshalErr)
		}
		if _, insertErr := srv.store.Insert(context.Background(), store.Event{EventID: "receipt-" + rule.PolicyID, SourceScenario: "agent-manager", TargetScenario: rule.TargetScenario, EventType: receiptEventType, Payload: body}); insertErr != nil {
			t.Fatal(insertErr)
		}
	}
	response, err = handler.ValidateScenario(context.Background(), connect.NewRequest(&scenariovalidationv1.ValidateScenarioRequest{Scenario: "agent-manager"}))
	if err != nil || response.Msg.GetStatus() != scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_PASSED || !response.Msg.GetAssessment().GetLocal().GetClean() {
		t.Fatalf("emitted validation = %#v err=%v", response.Msg, err)
	}
}

func TestCaptureValidationSupportsExplicitUnexercisedAndRejectsLiteralInstances(t *testing.T) {
	rule := policy.ReceiptProjectionRule{PolicyID: "new-policy", TargetScenario: "test-genie", OperationPattern: "POST /service/Start", Protocol: "connect", NeverExercised: true}
	if !rule.NeverExercised || literalInstanceIdentifier.MatchString(rule.OperationPattern) {
		t.Fatalf("marker or generic operation invalid: %#v", rule)
	}
	if !literalInstanceIdentifier.MatchString("GET /api/v1/runs/99be776d-95ef-460c-a4aa-c06372d1f715/report") {
		t.Fatal("literal UUID operation was accepted")
	}
	if receiptEmitted(context.Background(), nil, policy.ReceiptProjectionRule{TargetScenario: "test-genie", OperationPattern: "POST /service/Start"}) {
		t.Fatal("missing emission evidence was accepted")
	}
}
