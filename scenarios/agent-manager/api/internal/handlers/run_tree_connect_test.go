package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration"

	"github.com/google/uuid"
	"github.com/vrooli/api-core/identity"
	apiconnect "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api/apiconnect"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
)

// Waiting and reading a tree only observe runs. Any verified run identity may
// use them, whether or not it is parked, a parent of the target, or launched
// by Agent Manager.
func TestNestedRunReadsAreOpenToAnyVerifiedRun(t *testing.T) {
	principal := identity.Principal{Kind: identity.ActorAgent, Subject: "unrelated-run", Verified: true, Source: identity.SourceAgentProvenance}
	for _, procedure := range []string{apiconnect.AgentManagerServiceWaitRunProcedure, apiconnect.AgentManagerServiceWaitRunChildrenProcedure, apiconnect.AgentManagerServiceGetRunTreeProcedure} {
		request := httptest.NewRequest(http.MethodPost, procedure, strings.NewReader(`{}`))
		request = request.WithContext(identity.WithPrincipal(request.Context(), principal))
		passed := false
		recorder := httptest.NewRecorder()
		AgentOperationMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { passed = true; w.WriteHeader(http.StatusNoContent) })).ServeHTTP(recorder, request)
		if !passed {
			t.Fatalf("%s refused a verified run identity: %d", procedure, recorder.Code)
		}
	}
}

func TestRunTreeProtoCarriesLineageUsageAndOrphans(t *testing.T) {
	rootID, childID := uuid.New(), uuid.New()
	started := time.Date(2026, 11, 6, 12, 0, 0, 0, time.UTC)
	authority := started.Add(30 * time.Hour)
	root := &orchestration.RunTreeNode{Run: &domain.Run{ID: rootID, Status: domain.RunStatusCancelled, ResolvedConfig: &domain.RunConfig{RunnerType: domain.RunnerTypeCodex}, RequestedModel: "gpt-5"}}
	root.Children = []*orchestration.RunTreeNode{{
		Run:            &domain.Run{ID: childID, ParentRunID: &rootID, Depth: 1, Status: domain.RunStatusRunning, ActualModel: "gpt-5-mini", GoalDelivery: "native_verified", StartedAt: &started, CharterPath: "/work/CHARTER.md", OwnerExpiresAt: &authority},
		WeightedTokens: 1200, WeightedTokensKnown: true, ElapsedSeconds: 90, Orphaned: true,
	}}
	got := runTreeNodeToProto(root)
	if got.GetRunner() != "codex" || got.GetModel() != "gpt-5" || got.ParentRunId != nil || got.GetDepth() != 0 {
		t.Fatalf("root node = %+v", got)
	}
	child := got.GetChildren()[0]
	if child.GetParentRunId() != rootID.String() || child.GetDepth() != 1 || child.GetStatus() != domainpb.RunStatus_RUN_STATUS_RUNNING ||
		child.GetModel() != "gpt-5-mini" || child.GetGoalDelivery() != "native_verified" || !child.GetOrphaned() ||
		child.GetWeightedTokens() != 1200 || !child.GetWeightedTokensKnown() || child.GetElapsedSeconds() != 90 || child.GetCharterPath() != "/work/CHARTER.md" ||
		!child.GetStartedAt().AsTime().Equal(started) || !child.GetOwnerExpiresAt().AsTime().Equal(authority) || got.OwnerExpiresAt != nil {
		t.Fatalf("child node = %+v", child)
	}
}

func TestChildrenWaitProtoMatchesTheParkWakeShape(t *testing.T) {
	ended := time.Date(2026, 11, 6, 12, 5, 0, 0, time.UTC)
	result := &orchestration.ChildrenWaitResult{
		WakeReason: orchestration.ChildrenWakeEnded,
		Ended:      []orchestration.ChildRunSummary{{RunID: uuid.New(), Status: domain.RunStatusNeedsReview, EndedAt: &ended}},
		Active:     []orchestration.ChildRunSummary{{RunID: uuid.New(), Status: domain.RunStatusParked}},
		ObservedAt: ended.Add(time.Second),
	}
	got := childrenWaitResultToProto(result)
	if got.GetWakeReason() != "child_run_ended" || got.GetEnded()[0].GetStatus() != "needs_review" || !got.GetEnded()[0].GetEndedAt().AsTime().Equal(ended) ||
		got.GetActive()[0].GetStatus() != "parked" || got.GetActive()[0].EndedAt != nil || !got.GetObservedAt().AsTime().Equal(result.ObservedAt) {
		t.Fatalf("children wait response = %+v", got)
	}
}
