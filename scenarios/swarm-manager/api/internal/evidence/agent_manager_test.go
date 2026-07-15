package evidence

import (
	"context"
	"testing"
	"time"

	"github.com/vrooli/api-core/database"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"google.golang.org/protobuf/types/known/timestamppb"
	"swarm-manager/internal/agentmanager"
)

type fakeAgentManagerReader struct {
	events []*domainpb.RunEvent
	diff   agentmanager.RunDiff
}

func (f fakeAgentManagerReader) GetRunEvents(_ context.Context, _ string, options agentmanager.RunEventsOptions) ([]*domainpb.RunEvent, bool, error) {
	var page []*domainpb.RunEvent
	for _, event := range f.events {
		if event.GetSequence() > options.AfterSequence {
			page = append(page, event)
		}
	}
	return page, false, nil
}

func (f fakeAgentManagerReader) GetRunDiff(context.Context, string) (agentmanager.RunDiff, error) {
	return f.diff, nil
}

func TestReconcileAgentManagerNormalizesBoundedToolAndDiffEvidence(t *testing.T) {
	service, db := newEvidenceService(t, &stubOwnerIndex{owners: []Owner{{Kind: OwnerAgentSession, ID: "session-1"}}}, &stubOwnerIndex{})
	reader := fakeAgentManagerReader{
		events: []*domainpb.RunEvent{
			{Id: "tool-call", RunId: "run-42", Sequence: 1, Timestamp: timestamppb.New(time.Date(2026, 7, 12, 20, 0, 0, 0, time.UTC)), Data: &domainpb.RunEvent_ToolCall{ToolCall: &domainpb.ToolCallEventData{ToolName: "Write", ToolCallId: "call-1"}}},
			{Id: "tool-result", RunId: "run-42", Sequence: 2, Timestamp: timestamppb.New(time.Date(2026, 7, 12, 20, 0, 1, 0, time.UTC)), Data: &domainpb.RunEvent_ToolResult{ToolResult: &domainpb.ToolResultEventData{ToolName: "Write", ToolCallId: "call-1", Success: true, Output: "secret source text is deliberately ignored"}}},
		},
		diff: agentmanager.RunDiff{RunID: "run-42", SandboxID: "sandbox-1", GeneratedAt: "2026-07-12T20:00:02Z", Files: []agentmanager.RunDiffFile{{Path: "api/main.go", ChangeType: "modified"}}},
	}
	results, err := service.ReconcileAgentManager(context.Background(), reader, "run-42")
	if err != nil {
		t.Fatalf("ReconcileAgentManager: %v", err)
	}
	if len(results) != 3 {
		t.Fatalf("results = %d, want tool call, tool result, and diff file", len(results))
	}
	var rawOutput int
	if err := db.QueryRow(`SELECT COUNT(*) FROM evidence_observations WHERE metadata_json LIKE '%secret source text%'`).Scan(&rawOutput); err != nil {
		t.Fatal(err)
	}
	if rawOutput != 0 {
		t.Fatal("collector persisted raw tool output")
	}
	if complete, err := NewStore(database.NewFromPrimary(db)).HasTerminalWatermark(context.Background(), "agent-manager-events", "run-42", "agent_tool"); err != nil || !complete {
		t.Fatalf("agent tool terminal watermark = %v, %v; want true", complete, err)
	}
	if complete, err := NewStore(database.NewFromPrimary(db)).HasTerminalWatermark(context.Background(), "agent-manager-diff", "run-42", "repository_change"); err != nil || !complete {
		t.Fatalf("diff terminal watermark = %v, %v; want true", complete, err)
	}
}
