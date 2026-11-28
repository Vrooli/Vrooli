package authorization

import (
	"context"
	"testing"

	"scenario-authenticator/internal/audit"
)

type memoryStore struct {
	assignments map[string]map[string]bool
}

func (m *memoryStore) GrantScope(_ context.Context, principal, scope string) ([]string, error) {
	if m.assignments[principal] == nil {
		m.assignments[principal] = map[string]bool{}
	}
	m.assignments[principal][scope] = true
	return m.ListScopes(context.Background(), principal)
}

func (m *memoryStore) RevokeScope(_ context.Context, principal, scope string) ([]string, error) {
	delete(m.assignments[principal], scope)
	return m.ListScopes(context.Background(), principal)
}

func (m *memoryStore) ListScopes(_ context.Context, principal string) ([]string, error) {
	var out []string
	for scope := range m.assignments[principal] {
		out = append(out, scope)
	}
	return out, nil
}

type memoryAudit struct{ actions []string }

func (m *memoryAudit) Log(_ context.Context, event audit.Event) error {
	m.actions = append(m.actions, event.Action+":"+event.Metadata["scope"].(string))
	return nil
}

func (m *memoryAudit) List(context.Context, audit.Filter) ([]audit.Record, error) { return nil, nil }

func TestServiceStoresOpaqueScopesAndAuditsMutations(t *testing.T) {
	store := &memoryStore{assignments: map[string]map[string]bool{}}
	logger := &memoryAudit{}
	service := NewService(store, logger)
	ctx := context.Background()
	const scope = "some:opaque.scope/v1"

	if _, err := service.Grant(ctx, "principal-1", scope, Meta{RealmID: "default"}); err != nil {
		t.Fatalf("grant: %v", err)
	}
	scopes, err := service.List(ctx, "principal-1")
	if err != nil || len(scopes) != 1 || scopes[0] != scope {
		t.Fatalf("list = %#v, err=%v", scopes, err)
	}
	if _, err := service.Revoke(ctx, "principal-1", scope, Meta{RealmID: "default"}); err != nil {
		t.Fatalf("revoke: %v", err)
	}
	if got := logger.actions; len(got) != 2 || got[0] != "scope.granted:"+scope || got[1] != "scope.revoked:"+scope {
		t.Fatalf("audit actions = %#v", got)
	}
}

func TestServiceRejectsBlankScopeWithoutInterpretingVocabulary(t *testing.T) {
	service := NewService(&memoryStore{assignments: map[string]map[string]bool{}}, nil)
	if _, err := service.Grant(context.Background(), "principal-1", " \t", Meta{}); err != ErrInvalidScope {
		t.Fatalf("grant blank error = %v", err)
	}
}
