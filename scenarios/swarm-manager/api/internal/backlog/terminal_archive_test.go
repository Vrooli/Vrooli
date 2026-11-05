package backlog

import (
	"context"
	"fmt"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/identity"
	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
)

func terminalArchiveContext() context.Context {
	return identity.WithPrincipal(context.Background(), identity.Principal{Kind: identity.ActorAgent, Subject: "bookkeeper", Owner: "owner", Verified: true, Scopes: []string{"swarm-manager:write"}, Grant: &identity.OperationGrant{ID: "consent", Approver: "owner", Operations: []string{"swarm.bookkeeping"}, Targets: []string{"swarm-manager:*"}, ExpiresAt: time.Now().Add(time.Hour)}})
}

type terminalArchiveActiveWork struct{}

func (terminalArchiveActiveWork) HasActiveForBacklog(context.Context, string, string) bool {
	return true
}

func TestTerminalArchiveBoundsAndActiveReviewGuard(t *testing.T) {
	h, root := setupTestHandler(t)
	createTestItem(t, root, KindExecute, BacklogItem{Name: "reviewing", Kind: KindExecute, Title: "Reviewing", Status: StatusCompleted, Updated: "reviewed"})
	h.executionActivity = terminalArchiveActiveWork{}
	svc := NewConnectService(h)
	req := &apipb.ArchiveTerminalItemsRequest{Execute: true, Items: []*apipb.TerminalArchiveTarget{terminalTarget("reviewing", StatusCompleted)}}
	result, err := svc.ArchiveTerminalItems(terminalArchiveContext(), connect.NewRequest(req))
	if err != nil || result.Msg.ArchivedCount != 0 || result.Msg.Results[0].Code != "refused" {
		t.Fatalf("active review result = %+v, %v", result, err)
	}
	item, _ := h.store.LoadItem(KindExecute, "reviewing")
	if item.ArchivedAt != nil {
		t.Fatal("active review archived")
	}
	for i := 0; i < 50; i++ {
		req.Items = append(req.Items, terminalTarget(fmt.Sprintf("item-%d", i), StatusDropped))
	}
	if _, err := svc.ArchiveTerminalItems(terminalArchiveContext(), connect.NewRequest(req)); connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("unbounded batch accepted: %v", err)
	}
}

func terminalTarget(name string, status BacklogStatus) *apipb.TerminalArchiveTarget {
	return &apipb.TerminalArchiveTarget{Kind: "execute", Name: name, ExpectedStatus: string(status), ExpectedUpdated: "reviewed"}
}

func TestTerminalArchivePreviewExecutionAndReplayPreserveVerdicts(t *testing.T) {
	h, root := setupTestHandler(t)
	h.executionActivity = inactiveBookkeepingActivity{}
	for _, status := range []BacklogStatus{StatusCompleted, StatusDropped} {
		createTestItem(t, root, KindExecute, BacklogItem{Name: string(status), Kind: KindExecute, Title: "Retained", Status: status, Updated: "reviewed", Note: "Keep evidence", DependsOn: []string{"execute/ancestor"}})
	}
	svc := NewConnectService(h)
	req := &apipb.ArchiveTerminalItemsRequest{Items: []*apipb.TerminalArchiveTarget{terminalTarget("completed", StatusCompleted), terminalTarget("dropped", StatusDropped)}}
	preview, err := svc.ArchiveTerminalItems(terminalArchiveContext(), connect.NewRequest(req))
	if err != nil || preview.Msg.EligibleCount != 2 || preview.Msg.ArchivedCount != 0 || preview.Msg.Executed {
		t.Fatalf("preview = %+v, %v", preview, err)
	}
	item, _ := h.store.LoadItem(KindExecute, "completed")
	if item.ArchivedAt != nil {
		t.Fatal("preview archived work")
	}
	req.Execute = true
	result, err := svc.ArchiveTerminalItems(terminalArchiveContext(), connect.NewRequest(req))
	if err != nil || result.Msg.ArchivedCount != 2 {
		t.Fatalf("execution = %+v, %v", result, err)
	}
	for _, status := range []BacklogStatus{StatusCompleted, StatusDropped} {
		item, _ := h.store.LoadItem(KindExecute, string(status))
		if item.Status != status || item.ArchivedAt == nil || item.Note != "Keep evidence" || len(item.DependsOn) != 1 {
			t.Fatalf("retained verdict/evidence lost: %+v", item)
		}
	}
	replay, err := svc.ArchiveTerminalItems(terminalArchiveContext(), connect.NewRequest(req))
	if err != nil || replay.Msg.ArchivedCount != 0 || replay.Msg.Results[0].Code != "already_archived" {
		t.Fatalf("replay = %+v, %v", replay, err)
	}
}

func TestTerminalArchiveRefusesChangedUnapprovedAndUnfinishedRecords(t *testing.T) {
	h, root := setupTestHandler(t)
	h.executionActivity = inactiveBookkeepingActivity{}
	createTestItem(t, root, KindExecute, BacklogItem{Name: "changed", Kind: KindExecute, Title: "Changed", Status: StatusCompleted, Updated: "newer"})
	createTestItem(t, root, KindExecute, BacklogItem{Name: "unfinished", Kind: KindExecute, Title: "Unfinished", Status: StatusBacklog, Updated: "reviewed"})
	svc := NewConnectService(h)
	req := &apipb.ArchiveTerminalItemsRequest{Execute: true, Items: []*apipb.TerminalArchiveTarget{terminalTarget("changed", StatusCompleted), terminalTarget("unfinished", StatusDropped)}}
	result, err := svc.ArchiveTerminalItems(terminalArchiveContext(), connect.NewRequest(req))
	if err != nil || result.Msg.ArchivedCount != 0 || result.Msg.SkippedCount != 2 || result.Msg.Results[0].Code != "changed" {
		t.Fatalf("stale result = %+v, %v", result, err)
	}
	result, err = svc.ArchiveTerminalItems(context.Background(), connect.NewRequest(req))
	if err != nil || result.Msg.Results[0].Code != "refused" {
		t.Fatalf("unapproved result = %+v, %v", result, err)
	}
	for _, name := range []string{"changed", "unfinished"} {
		item, _ := h.store.LoadItem(KindExecute, name)
		if item.ArchivedAt != nil {
			t.Fatalf("refused record archived: %s", name)
		}
	}
	req.Items = append(req.Items, req.Items[0])
	if _, err := svc.ArchiveTerminalItems(terminalArchiveContext(), connect.NewRequest(req)); connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("duplicate input accepted: %v", err)
	}
}

type changingTerminalActivity struct {
	store Store
	calls int
}

func (a *changingTerminalActivity) HasActiveForBacklog(context.Context, string, string) bool {
	a.calls++
	if a.calls == 2 {
		item, _ := a.store.LoadItem(KindExecute, "racing")
		item.Updated, item.Note = "newer", "Intervening operator edit"
		_ = a.store.SaveItem(item)
	}
	return false
}

func TestTerminalArchiveCannotOverwriteAnInterveningEdit(t *testing.T) {
	h, root := setupTestHandler(t)
	createTestItem(t, root, KindExecute, BacklogItem{Name: "racing", Kind: KindExecute, Title: "Racing", Status: StatusCompleted, Updated: "reviewed"})
	h.executionActivity = &changingTerminalActivity{store: h.store}
	result, err := NewConnectService(h).ArchiveTerminalItems(terminalArchiveContext(), connect.NewRequest(&apipb.ArchiveTerminalItemsRequest{Execute: true, Items: []*apipb.TerminalArchiveTarget{terminalTarget("racing", StatusCompleted)}}))
	if err != nil || result.Msg.ArchivedCount != 0 || result.Msg.Results[0].Code != "changed" {
		t.Fatalf("racing result = %+v, %v", result, err)
	}
	item, _ := h.store.LoadItem(KindExecute, "racing")
	if item.ArchivedAt != nil || item.Note != "Intervening operator edit" {
		t.Fatalf("intervening edit overwritten: %+v", item)
	}
}
