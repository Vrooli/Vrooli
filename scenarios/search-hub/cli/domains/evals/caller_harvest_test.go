package evals

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	api "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	apiconnect "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api/apiconnect"
	domain "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
)

type callerHarvestFixture struct {
	apiconnect.UnimplementedAgentManagerServiceHandler
	run             *domain.Run
	events          []*domain.RunEvent
	eventRequestSeq []int64
}

func (f *callerHarvestFixture) GetRun(_ context.Context, req *connect.Request[api.GetRunRequest]) (*connect.Response[api.GetRunResponse], error) {
	if f.run == nil || req.Msg.GetRunId() != f.run.GetId() {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("run not found"))
	}
	return connect.NewResponse(&api.GetRunResponse{Run: f.run}), nil
}

func (f *callerHarvestFixture) GetRunEvents(_ context.Context, req *connect.Request[api.GetRunEventsRequest]) (*connect.Response[api.GetRunEventsResponse], error) {
	if f.run == nil || req.Msg.GetRunId() != f.run.GetId() {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("run not found"))
	}
	if req.Msg.GetLimit() != 1000 || len(req.Msg.GetEventTypes()) != 2 ||
		req.Msg.GetEventTypes()[0] != domain.RunEventType_RUN_EVENT_TYPE_MESSAGE ||
		req.Msg.GetEventTypes()[1] != domain.RunEventType_RUN_EVENT_TYPE_MESSAGE_DELETED || req.Msg.GetSummaryOnly() {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("unexpected event request"))
	}
	f.eventRequestSeq = append(f.eventRequestSeq, req.Msg.GetAfterSequence())
	return connect.NewResponse(&api.GetRunEventsResponse{Events: f.events}), nil
}

func newCallerHarvestTestClient(t *testing.T, fixture *callerHarvestFixture) callerHarvestClient {
	t.Helper()
	path, handler := apiconnect.NewAgentManagerServiceHandler(fixture)
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return apiconnect.NewAgentManagerServiceClient(server.Client(), server.URL)
}

func callerMessageEvent(id, runID string, sequence int64, role, source, content, conversationID string) *domain.RunEvent {
	return &domain.RunEvent{
		Id: id, RunId: runID, Sequence: sequence,
		EventType: domain.RunEventType_RUN_EVENT_TYPE_MESSAGE,
		Data: &domain.RunEvent_Message{Message: &domain.MessageEventData{
			Role: role, Source: source, Content: content, ConversationId: conversationID,
		}},
	}
}

func TestHarvestSelectedCallersUsesAgentManagerRunAndWritesPrivateCandidate(t *testing.T) {
	const runID = "123e4567-e89b-12d3-a456-426614174000"
	fixture := &callerHarvestFixture{
		run: &domain.Run{Id: runID, HarnessSessionId: "session-canonical"},
		events: []*domain.RunEvent{
			callerMessageEvent("event-1", runID, 0, "user", "operator", "private caller query", ""),
			callerMessageEvent("event-2", runID, 1, "assistant", "", "assistant response", "conversation-1"),
			callerMessageEvent("event-3", runID, 2, "user", "goal_continuation", "not a caller", ""),
		},
	}
	bundle, err := harvestSelectedCallers(context.Background(), newCallerHarvestTestClient(t, fixture), []string{runID})
	if err != nil {
		t.Fatal(err)
	}
	if len(bundle.Records) != 1 || bundle.Records[0].Sequence != 0 || bundle.Records[0].ConversationID != "conversation-1" || bundle.Records[0].Status != "candidate" || bundle.Records[0].AcceptanceEligible {
		t.Fatalf("candidate bundle = %+v", bundle)
	}
	if len(fixture.eventRequestSeq) != 1 || fixture.eventRequestSeq[0] != -1 {
		t.Fatalf("generated event request cursors = %v, want initial cursor [-1]", fixture.eventRequestSeq)
	}

	path := filepath.Join(t.TempDir(), "candidates.json")
	if err := writeCallerHarvestFile(path, bundle); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("candidate file mode = %o; want 600", info.Mode().Perm())
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "private caller query") || !strings.Contains(string(data), `"acceptance_eligible": false`) {
		t.Fatalf("candidate export omitted private query or acceptance gate: %s", data)
	}
	if err := writeCallerHarvestFile(path, bundle); err == nil {
		t.Fatal("existing candidate export should not be overwritten")
	}
}

func TestHarvestSelectedCallersRequiresConversationOrHarnessIdentity(t *testing.T) {
	const runID = "123e4567-e89b-12d3-a456-426614174000"
	t.Run("uses canonical harness session when conversation IDs are absent", func(t *testing.T) {
		fixture := &callerHarvestFixture{run: &domain.Run{Id: runID, HarnessSessionId: "session-canonical"}, events: []*domain.RunEvent{
			callerMessageEvent("event-1", runID, 0, "user", "operator", "identified by harness", ""),
		}}
		bundle, err := harvestSelectedCallers(context.Background(), newCallerHarvestTestClient(t, fixture), []string{runID})
		if err != nil {
			t.Fatal(err)
		}
		if len(bundle.Records) != 1 || bundle.Records[0].ConversationID != "session-canonical" {
			t.Fatalf("canonical harness source identity = %+v", bundle.Records)
		}
	})
	t.Run("omits messages without any durable source identity", func(t *testing.T) {
		fixture := &callerHarvestFixture{run: &domain.Run{Id: runID}, events: []*domain.RunEvent{
			callerMessageEvent("event-1", runID, 0, "user", "operator", "query without source identity", ""),
		}}
		bundle, err := harvestSelectedCallers(context.Background(), newCallerHarvestTestClient(t, fixture), []string{runID})
		if err != nil {
			t.Fatal(err)
		}
		if len(bundle.Records) != 0 {
			t.Fatalf("candidate without durable conversation identity = %+v", bundle.Records)
		}
	})
}

func TestHarvestSelectedCallersDoesNotGuessBetweenConversationIdentities(t *testing.T) {
	const runID = "123e4567-e89b-12d3-a456-426614174000"
	fixture := &callerHarvestFixture{
		run: &domain.Run{Id: runID, HarnessSessionId: "session-canonical"},
		events: []*domain.RunEvent{
			callerMessageEvent("event-1", runID, 0, "user", "operator", "ambiguous caller", ""),
			callerMessageEvent("event-2", runID, 1, "assistant", "", "", "conversation-1"),
			callerMessageEvent("event-3", runID, 2, "assistant", "", "", "conversation-2"),
		},
	}
	bundle, err := harvestSelectedCallers(context.Background(), newCallerHarvestTestClient(t, fixture), []string{runID})
	if err != nil {
		t.Fatal(err)
	}
	if len(bundle.Records) != 0 {
		t.Fatalf("candidate with ambiguous durable conversation identity = %+v", bundle.Records)
	}
}

func TestHarvestSelectedCallersFiltersDeletedOperatorMessages(t *testing.T) {
	const runID = "123e4567-e89b-12d3-a456-426614174000"
	fixture := &callerHarvestFixture{
		run: &domain.Run{Id: runID},
		events: []*domain.RunEvent{
			callerMessageEvent("event-deleted", runID, 0, "user", "operator", "removed query", "conversation-1"),
			{
				Id: "delete-marker", RunId: runID, Sequence: 1,
				EventType: domain.RunEventType_RUN_EVENT_TYPE_MESSAGE_DELETED,
				Data:      &domain.RunEvent_MessageDeleted{MessageDeleted: &domain.MessageDeletedEventData{TargetEventId: "event-deleted"}},
			},
		},
	}
	bundle, err := harvestSelectedCallers(context.Background(), newCallerHarvestTestClient(t, fixture), []string{runID})
	if err != nil {
		t.Fatal(err)
	}
	if len(bundle.Records) != 0 {
		t.Fatalf("deleted operator message was harvested: %+v", bundle.Records)
	}
}

func TestParseCallerRunIDsRequiresUniqueExplicitIDs(t *testing.T) {
	ids, err := parseCallerRunIDs("run-2, run-1")
	if err != nil || len(ids) != 2 || ids[0] != "run-1" {
		t.Fatalf("parseCallerRunIDs = %v, %v", ids, err)
	}
	for _, raw := range []string{"", "run-1,run-1"} {
		if _, err := parseCallerRunIDs(raw); err == nil {
			t.Errorf("parseCallerRunIDs(%q) unexpectedly succeeded", raw)
		}
	}
}

func TestAgentManagerBaseRestrictsCallerTextReadsToLoopback(t *testing.T) {
	if _, err := configuredAgentManagerBase("https://agent-manager.example.test"); err == nil {
		t.Fatal("remote Agent Manager API should be rejected")
	}
	base, err := configuredAgentManagerBase("http://localhost:8090")
	if err != nil || base != "http://localhost:8090" {
		t.Fatalf("configured local Agent Manager base = %q, %v", base, err)
	}
}

func TestEvalsRegisterWiresHarvestCallersAsPrivateLocalAction(t *testing.T) {
	manifest, err := os.ReadFile(filepath.Join("..", "..", "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	group, err := Register(nil, manifest)
	if err != nil {
		t.Fatal(err)
	}
	for _, command := range group.Subcommands {
		if command.Name == "harvest-callers" {
			if command.NeedsAPI || command.PrimitiveEvidence() != cliapp.PrimitiveAction {
				t.Fatalf("harvest-callers command = needsAPI:%t primitive:%q", command.NeedsAPI, command.PrimitiveEvidence())
			}
			return
		}
	}
	t.Fatal("harvest-callers was not registered")
}
