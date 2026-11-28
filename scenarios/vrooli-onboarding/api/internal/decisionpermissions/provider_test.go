package decisionpermissions

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/identity"
	permission "github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/permissions"
)

type ownerProjectionFake struct {
	pages      map[string]Page
	refs       map[string]Reference
	readErr    error
	listCalls  int
	lastCursor string
}

func (f *ownerProjectionFake) ListPending(_ context.Context, limit int, cursor string) (Page, error) {
	f.listCalls++
	f.lastCursor = cursor
	if page, ok := f.pages[cursor]; ok {
		if len(page.Decisions) > limit {
			return Page{}, fmt.Errorf("fake page exceeds requested limit")
		}
		return page, nil
	}
	return Page{}, nil
}
func (f *ownerProjectionFake) ReadPending(_ context.Context, ref, _ string) (Reference, error) {
	if f.readErr != nil {
		return Reference{}, f.readErr
	}
	value, ok := f.refs[ref]
	if !ok {
		return Reference{}, connect.NewError(connect.CodeNotFound, fmt.Errorf("not found"))
	}
	return value, nil
}

func humanContext(subject string) context.Context {
	return identity.WithPrincipal(context.Background(), identity.Principal{Kind: identity.ActorHuman, Subject: subject, Verified: true})
}

func validReference(ref, owner, revision string) Reference {
	return Reference{
		DecisionRef: ref, OwnerID: owner, Source: "treasury", SourceEventType: "treasury.refund.decision_requested.v1",
		Revision: revision, State: "pending", CreatedAt: "2026-11-20T10:00:00Z", UpdatedAt: "2026-11-20T10:01:00Z",
	}
}

func TestRegistryListsAndReadsOnlySafeOwnerBoundPendingDecisionReferences(t *testing.T) {
	first, second := validReference("ask-1", "alice", "0"), validReference("ask-2", "alice", "3")
	owner := &ownerProjectionFake{pages: map[string]Page{"": {Decisions: []Reference{first}, NextCursor: "next"}, "next": {Decisions: []Reference{second}}}, refs: map[string]Reference{"ask-1": first, "ask-2": second}}
	provider := New(owner)
	registry, err := permission.NewRegistry(provider)
	if err != nil {
		t.Fatal(err)
	}
	ctx := humanContext("alice")
	inventory, err := registry.Inventory(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(inventory) != 2 || !strings.HasPrefix(inventory[0].ID, PermissionPrefix) || inventory[0].CanWrite {
		t.Fatalf("inventory = %+v", inventory)
	}
	id := permissionID(first.DecisionRef, first.Revision)
	read, err := registry.Read(ctx, id, permission.ReadRequest{ActorID: "alice", Owner: Owner})
	if err != nil {
		t.Fatal(err)
	}
	var got Reference
	if err := json.Unmarshal(read.Value, &got); err != nil {
		t.Fatal(err)
	}
	if got != first || read.Revision != first.Revision || read.Owner != Owner {
		t.Fatalf("read = %+v, value=%+v", read, got)
	}
	for _, hidden := range []string{"question", "answer", "answerer", "note", "correlation", "context", "extra", "opaque"} {
		if strings.Contains(string(read.Value), hidden) {
			t.Fatalf("read exposed forbidden field %q: %s", hidden, read.Value)
		}
	}
	if _, err := registry.Write(ctx, id, permission.WriteRequest{ActorID: "alice", Owner: Owner, ExpectedRevision: "0", IdempotencyKey: "resolve", Value: []byte(`{"answer":"approve"}`)}); connect.CodeOf(err) != connect.CodeUnknown {
		// Registry refuses on CanWrite before the owner is called; the exact
		// wrapped error is intentionally not part of the external contract.
		t.Fatalf("human decision was writable: %v", err)
	}
}

func TestProviderRefusesCrossOwnerStaleAndMalformedOwnerProjection(t *testing.T) {
	valid := validReference("ask-1", "alice", "0")
	for name, tc := range map[string]struct {
		ref      Reference
		revision string
		want     connect.Code
	}{
		"cross owner":      {ref: validReference("ask-1", "bob", "0"), revision: "0", want: connect.CodeDataLoss},
		"missing revision": {ref: validReference("ask-1", "alice", ""), revision: "", want: connect.CodeDataLoss},
		"resolved":         {ref: Reference{DecisionRef: "ask-1", OwnerID: "alice", Source: "treasury", SourceEventType: "treasury.refund.decision_requested.v1", Revision: "0", State: "answered", CreatedAt: "2026-11-20T10:00:00Z", UpdatedAt: "2026-11-20T10:01:00Z"}, revision: "0", want: connect.CodeDataLoss},
	} {
		t.Run(name, func(t *testing.T) {
			owner := &ownerProjectionFake{pages: map[string]Page{"": {Decisions: []Reference{tc.ref}}}}
			_, err := New(owner).List(humanContext("alice"))
			if connect.CodeOf(err) != tc.want {
				t.Fatalf("List error=%v code=%s", err, connect.CodeOf(err))
			}
		})
	}

	owner := &ownerProjectionFake{pages: map[string]Page{"": {Decisions: []Reference{valid}}}, refs: map[string]Reference{"ask-1": valid}}
	registry, err := permission.NewRegistry(New(owner))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := registry.Read(humanContext("bob"), permissionID("ask-1", "0"), permission.ReadRequest{ActorID: "bob", Owner: Owner}); connect.CodeOf(err) != connect.CodeDataLoss {
		t.Fatalf("cross-owner registry read was accepted: %v", err)
	}
	owner.readErr = connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("stale"))
	if _, err := registry.Read(humanContext("alice"), permissionID("ask-1", "0"), permission.ReadRequest{ActorID: "alice", Owner: Owner}); connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("stale owner revision code=%s err=%v", connect.CodeOf(err), err)
	}
}

func TestProviderRefusesNonHumanAndUnboundedOwnerInventory(t *testing.T) {
	if _, err := New(&ownerProjectionFake{}).List(identity.WithPrincipal(context.Background(), identity.Principal{Kind: identity.ActorService, Subject: "svc", Verified: true})); connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("service principal inventory error=%v", err)
	}
	owner := &ownerProjectionFake{pages: map[string]Page{}}
	cursor := ""
	for i := 0; i < maxInventory/pageSize; i++ {
		next := fmt.Sprintf("page-%d", i+1)
		decisions := make([]Reference, pageSize)
		for j := range decisions {
			decisions[j] = validReference(fmt.Sprintf("ask-%d-%d", i, j), "alice", "0")
		}
		owner.pages[cursor] = Page{Decisions: decisions, NextCursor: next}
		cursor = next
	}
	_, err := New(owner).List(humanContext("alice"))
	if connect.CodeOf(err) != connect.CodeResourceExhausted {
		t.Fatalf("oversize inventory code=%s err=%v", connect.CodeOf(err), err)
	}
}

func TestProviderRejectsPaginationCursorReplay(t *testing.T) {
	first := validReference("ask-1", "alice", "0")
	second := validReference("ask-2", "alice", "0")
	third := validReference("ask-3", "alice", "0")
	owner := &ownerProjectionFake{pages: map[string]Page{
		"":       {Decisions: []Reference{first}, NextCursor: "page-a"},
		"page-a": {Decisions: []Reference{second}, NextCursor: "page-b"},
		"page-b": {Decisions: []Reference{third}, NextCursor: "page-a"},
	}}
	_, err := New(owner).List(humanContext("alice"))
	if connect.CodeOf(err) != connect.CodeDataLoss {
		t.Fatalf("cursor replay code=%s err=%v", connect.CodeOf(err), err)
	}
}
