package identity

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestAttenuateNarrowsScopesAndExpiry(t *testing.T) {
	now := time.Unix(1000, 0)
	parent := &Claims{RunID: uuid.New(), TaskID: uuid.New(), Subject: "owner@example", Scopes: []string{"vrooli-bridge:read", "vrooli-bridge:dispatch"}, ExpiresAt: 2000}
	child, err := Attenuate(parent, uuid.New(), uuid.New(), []string{"vrooli-bridge:dispatch"}, time.Unix(1500, 0), now)
	if err != nil {
		t.Fatal(err)
	}
	if len(child.Scopes) != 1 || child.Scopes[0] != "vrooli-bridge:dispatch" {
		t.Fatalf("child scopes = %#v", child.Scopes)
	}
	if child.ExpiresAt != 1500 || child.Subject != parent.Subject {
		t.Fatalf("child claims lost attenuation/provenance: %#v", child)
	}
}

func TestAttenuateRefusesWideningAndLateExpiry(t *testing.T) {
	now := time.Unix(1000, 0)
	parent := &Claims{Scopes: []string{"vrooli-bridge:read"}, ExpiresAt: 2000}
	if _, err := Attenuate(parent, uuid.New(), uuid.New(), []string{"vrooli-bridge:write"}, time.Unix(1500, 0), now); !errors.Is(err, ErrScopeWidening) {
		t.Fatalf("widening error = %v", err)
	}
	if _, err := Attenuate(parent, uuid.New(), uuid.New(), nil, time.Unix(2500, 0), now); !errors.Is(err, ErrExpiryWidening) {
		t.Fatalf("expiry error = %v", err)
	}
}

func TestAttenuateMaterializesWildcardAgainstConcreteParent(t *testing.T) {
	now := time.Unix(1000, 0)
	parent := &Claims{Scopes: []string{"vrooli-bridge:read", "vrooli-bridge:dispatch"}, ExpiresAt: 2000}
	child, err := Attenuate(parent, uuid.New(), uuid.New(), []string{"vrooli-bridge:*"}, time.Unix(1500, 0), now)
	if err != nil {
		t.Fatal(err)
	}
	if len(child.Scopes) != 2 || child.Scopes[0] != parent.Scopes[0] || child.Scopes[1] != parent.Scopes[1] {
		t.Fatalf("wildcard was not materialized: %#v", child.Scopes)
	}
}

func TestAttenuateRefusesPersonaActAsWidening(t *testing.T) {
	// [REQ:PSN-P0-002] Persona authorization uses the existing one-way
	// delegation chain; a child cannot invent a persona capability.
	now := time.Unix(1000, 0)
	parent := &Claims{Scopes: []string{"persona.act-as:persona-1"}, ExpiresAt: 2000}

	if _, err := Attenuate(parent, uuid.New(), uuid.New(), []string{"persona.act-as:persona-2"}, time.Unix(1500, 0), now); !errors.Is(err, ErrScopeWidening) {
		t.Fatalf("persona scope widening error = %v", err)
	}
	child, err := Attenuate(parent, uuid.New(), uuid.New(), []string{"persona.act-as:*"}, time.Unix(1500, 0), now)
	if err != nil {
		t.Fatalf("concrete persona scope should materialize through wildcard request: %v", err)
	}
	if len(child.Scopes) != 1 || child.Scopes[0] != "persona.act-as:persona-1" {
		t.Fatalf("wildcard request widened persona scope: %#v", child.Scopes)
	}
}
