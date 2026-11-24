package standinggrantpermissions

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/vrooli/api-core/identity"
	"github.com/vrooli/api-core/scopecatalog"
	"github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/permissions"
)

type fakeOwner struct {
	pages map[string]Page
	err   error
}

func (f fakeOwner) List(_ context.Context, _ int, cursor string) (Page, error) {
	if f.err != nil {
		return Page{}, f.err
	}
	return f.pages[cursor], nil
}
func (f fakeOwner) Read(_ context.Context, kind, reference, revision string) (OwnerValue, error) {
	page := f.pages[""]
	value := OwnerValue{Owner: page.Owner, SourceRevision: page.SourceRevision}
	if kind == "grant" {
		for _, g := range page.Grants {
			if g.ID == reference {
				copy := g
				value.Grant = &copy
			}
		}
	}
	if kind == "scope" {
		for _, s := range page.Scopes {
			if s.Value == reference {
				copy := s
				value.Scope = &copy
			}
		}
	}
	return value, nil
}
func testGrant(id string) Grant {
	return Grant{ID: id, Subject: "worker", Approver: "human-owner", Operations: []string{"run.create"}, Targets: []string{"agent-manager:*"}, State: "approved", ApprovedAt: time.Now().Add(-time.Hour), ExpiresAt: time.Now().Add(time.Hour)}
}
func testContext() context.Context {
	return identity.WithPrincipal(context.Background(), identity.Principal{Kind: identity.ActorHuman, Subject: "human-owner", Verified: true})
}

func TestProviderListsAndReadsOwnerBoundMetadata(t *testing.T) {
	owner := fakeOwner{pages: map[string]Page{"": {Owner: Owner, SourceRevision: "r1", Scopes: []scopecatalog.Scope{{Scenario: Owner, Value: "agent-manager:read", Command: "permissions/list"}}, Grants: []Grant{testGrant("g1")}}}}
	p := New(owner)
	descriptors, err := p.List(testContext())
	if err != nil {
		t.Fatal(err)
	}
	if len(descriptors) != 2 || descriptors[0].CanWrite || descriptors[0].Owner != Owner {
		t.Fatalf("descriptor: %+v", descriptors)
	}
	var d, scopeDescriptor permissions.Descriptor
	for _, descriptor := range descriptors {
		if descriptor.SubjectKind == "standing-grant" {
			d = descriptor
		}
		if descriptor.SubjectKind == "scope-declaration" {
			scopeDescriptor = descriptor
		}
	}
	if d.ID == "" || scopeDescriptor.ID == "" {
		t.Fatalf("grant and scope descriptors absent: %+v", descriptors)
	}
	req := permissions.ReadRequest{ActorID: "human-owner", Owner: Owner}
	decision, err := p.AuthorizeRead(testContext(), d, req)
	if err != nil || !decision.Allowed {
		t.Fatalf("read authorization: %+v %v", decision, err)
	}
	registry, err := permissions.NewRegistry(p)
	if err != nil {
		t.Fatal(err)
	}
	result, err := registry.Read(testContext(), d.ID, req)
	if err != nil {
		t.Fatal(err)
	}
	var grant Grant
	if err := json.Unmarshal(result.Value, &grant); err != nil {
		t.Fatal(err)
	}
	if grant.ID != "g1" || result.Revision != "r1" {
		t.Fatalf("read result: %+v / %+v", grant, result)
	}
	scopeResult, err := registry.Read(testContext(), scopeDescriptor.ID, req)
	if err != nil {
		t.Fatal(err)
	}
	var scope scopecatalog.Scope
	if err := json.Unmarshal(scopeResult.Value, &scope); err != nil || scope.Value != "agent-manager:read" {
		t.Fatalf("scope read: %+v %v", scope, err)
	}
	if _, err := p.Authorize(testContext(), d, permissions.WriteRequest{}); err == nil {
		t.Fatal("owner-only grant mutation was exposed")
	}
}

func TestProviderRejectsCrossOwnerAndInvalidOwnerProjection(t *testing.T) {
	for _, tc := range []struct {
		name string
		page Page
	}{
		{name: "cross owner", page: Page{Owner: "other", SourceRevision: "r1", Grants: []Grant{testGrant("g1")}}},
		{name: "unknown status", page: Page{Owner: Owner, SourceRevision: "r1", Grants: []Grant{func() Grant { g := testGrant("g1"); g.State = "pending"; return g }()}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := New(fakeOwner{pages: map[string]Page{"": tc.page}}).List(testContext()); err == nil {
				t.Fatal("invalid projection accepted")
			}
		})
	}
}
