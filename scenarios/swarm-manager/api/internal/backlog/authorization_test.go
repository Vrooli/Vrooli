package backlog

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/identity"
)

func bookkeepingOwnerContext() context.Context {
	return bookkeepingOwnerOn(context.Background())
}

func bookkeepingOwnerOn(ctx context.Context) context.Context {
	return identity.WithPrincipal(ctx, identity.Principal{Kind: identity.ActorHuman, Subject: "operator:test", Verified: true, Scopes: []string{"swarm-manager:write"}})
}

func TestArchiveRequiresVerifiedBoundedGrantAndPreservesHistory(t *testing.T) {
	h, root := setupTestHandler(t)
	createTestItem(t, root, KindExecute, BacklogItem{Name: "obsolete", Kind: KindExecute, Title: "Obsolete", Status: StatusBacklog})
	invoke := func(ctx context.Context) int {
		req := httptest.NewRequest(http.MethodPatch, "/api/v1/backlog/execute/obsolete/archive-item", nil).WithContext(ctx)
		req = mux.SetURLVars(req, map[string]string{"kind": "execute", "name": "obsolete"})
		rec := httptest.NewRecorder()
		h.Archive(rec, req)
		return rec.Code
	}
	if code := invoke(context.Background()); code != http.StatusForbidden {
		t.Fatalf("anonymous archive status=%d", code)
	}
	p := identity.Principal{Kind: identity.ActorAgent, Subject: "run", Owner: "operator:test", Verified: true, Scopes: []string{"swarm-manager:write"}}
	if code := invoke(identity.WithPrincipal(context.Background(), p)); code != http.StatusForbidden {
		t.Fatal("capability alone archived work")
	}
	p.Grant = &identity.OperationGrant{ID: "grant", Approver: "operator:test", Operations: []string{"swarm.bookkeeping"}, Targets: []string{"swarm-manager:backlog/execute/other"}, ExpiresAt: time.Now().Add(time.Hour)}
	if code := invoke(identity.WithPrincipal(context.Background(), p)); code != http.StatusForbidden {
		t.Fatal("wrong target archived work")
	}
	p.Grant.Targets = []string{"swarm-manager:backlog/execute/obsolete"}
	if code := invoke(identity.WithPrincipal(context.Background(), p)); code != http.StatusForbidden {
		t.Fatal("unknown work activity admitted bookkeeping")
	}
	h.executionActivity = inactiveBookkeepingActivity{}
	if code := invoke(identity.WithPrincipal(context.Background(), p)); code != http.StatusOK {
		t.Fatalf("approved archive status=%d", code)
	}
	item, err := h.store.LoadItem(KindExecute, "obsolete")
	if err != nil || item.ArchivedAt == nil || item.Status != StatusDropped || item.Title != "Obsolete" {
		t.Fatalf("history or resolved status lost: %+v %v", item, err)
	}
	if _, err := h.DecideReview(identity.WithPrincipal(context.Background(), p), KindExecute, "obsolete", ReviewDecideRequest{Decision: ReviewDecisionAccept, DecidedBy: "operator:test"}); err == nil {
		t.Fatal("bookkeeping agent accepted its own review")
	}
}

type inactiveBookkeepingActivity struct{}

func (inactiveBookkeepingActivity) HasActiveForBacklog(context.Context, string, string) bool {
	return false
}
