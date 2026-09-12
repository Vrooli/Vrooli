package convert

import (
	"testing"
	"time"

	domain "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-events/v1/domain"
	"github.com/vrooli/vrooli/scenarios/vrooli-events/internal/store"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func receiptEnvelope(t *testing.T) *domain.EventEnvelope {
	t.Helper()
	data, err := anypb.New(&domain.ReceiptData{Outcome: "success", StatusCode: 201, Projection: mustStruct(t, map[string]any{"plan.id": "plan-1"})})
	if err != nil {
		t.Fatal(err)
	}
	return &domain.EventEnvelope{
		EventId: "evt-123", EventType: "vrooli.events.receipt.v1", OccurredAt: timestamppb.New(time.Now().UTC()),
		Source:      &domain.EventSource{Scenario: "agent-manager", ActorKind: "agent"},
		Target:      &domain.EventTarget{Scenario: "plan-manager", Operation: "POST /plans/CreatePlan", Protocol: "connect"},
		Correlation: &domain.EventCorrelation{AgentRunId: "run-1", WorkflowExecutionId: "workflow-1", WorkflowNodeId: "create", Attempt: 2},
		Attribution: &domain.EventAttribution{SubjectKind: "agent", SubjectId: "agent-1", Verified: true}, Data: data,
	}
}

func mustStruct(t *testing.T, fields map[string]any) *structpb.Struct {
	t.Helper()
	v, err := structpb.NewStruct(fields)
	if err != nil {
		t.Fatal(err)
	}
	return v
}

func TestEnvelopeToEventRoundTrip(t *testing.T) {
	env := receiptEnvelope(t)
	event, err := EnvelopeToEvent(env)
	if err != nil {
		t.Fatalf("EnvelopeToEvent: %v", err)
	}
	if event.EventID != env.EventId || event.SourceScenario != env.Source.Scenario || event.TargetScenario != env.Target.Scenario || event.CorrelationID != env.Correlation.AgentRunId {
		t.Fatalf("derived event indexes = %#v", event)
	}
	if len(event.Payload) == 0 {
		t.Fatal("canonical envelope was not persisted")
	}
	got, err := EventToEnvelope(event)
	if err != nil {
		t.Fatalf("EventToEnvelope: %v", err)
	}
	if got.GetTarget().GetOperation() != env.GetTarget().GetOperation() || got.GetCorrelation().GetWorkflowNodeId() != "create" || !got.GetAttribution().GetVerified() {
		t.Fatalf("round trip lost universal fields: %#v", got)
	}
	data := &domain.ReceiptData{}
	if err := got.GetData().UnmarshalTo(data); err != nil || data.GetProjection().GetFields()["plan.id"].GetStringValue() != "plan-1" {
		t.Fatalf("round trip lost typed data: %v %#v", err, data)
	}
}

func TestEventToEnvelopeRejectsNonCanonicalPayload(t *testing.T) {
	_, err := EventToEnvelope(store.Event{EventID: "legacy", Payload: []byte("not-a-canonical-envelope")})
	if err == nil {
		t.Fatal("expected hard-cut payload rejection")
	}
}
