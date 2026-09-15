package onboard

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	internalgrant "vrooli-bridge/internal/credentialgrant"
	internalonboard "vrooli-bridge/internal/onboard"
)

type fakeGrants struct {
	grants  []internalgrant.Grant
	creates []internalgrant.CreateInput
}

func (f *fakeGrants) List(context.Context, string) ([]internalgrant.Grant, error) {
	return append([]internalgrant.Grant(nil), f.grants...), nil
}

func (f *fakeGrants) Create(_ context.Context, in internalgrant.CreateInput) (internalgrant.Grant, error) {
	f.creates = append(f.creates, in)
	grant := internalgrant.Grant{ID: "grant-1", NodeID: in.NodeID, LogicalID: in.LogicalID, Field: in.Field, Class: in.Class, Retention: in.Retention, Generation: 1, GrantedAt: time.Now()}
	f.grants = append(f.grants, grant)
	return grant, nil
}

// The node's unlock grant is created once, as an ephemeral infrastructure
// grant for exactly the escrow address, and delivered on every ensure. The
// grant record is metadata; the passphrase never appears in it.
func TestNodeStoreGrantIsCreatedOnceAsEphemeralInfrastructure(t *testing.T) {
	grants := &fakeGrants{}
	deliveries := 0
	ensurer := NewNodeStoreGrantEnsurer(grants, func(context.Context, string) error { deliveries++; return nil })
	logicalID := internalonboard.CredentialStoreLogicalID("451ea636-a80f-4080-82b7-fa65d0e3289a")

	for i := 0; i < 2; i++ {
		if err := ensurer.EnsureNodeStoreGrant(context.Background(), "node-1", logicalID, internalonboard.CredentialStoreEscrowField); err != nil {
			t.Fatalf("ensure %d: %v", i, err)
		}
	}
	if len(grants.creates) != 1 {
		t.Fatalf("creates = %d, want exactly one grant across repeated ensures", len(grants.creates))
	}
	created := grants.creates[0]
	if created.Class != internalgrant.ClassInfrastructure || created.Retention != internalgrant.RetentionEphemeral {
		t.Fatalf("grant class/retention = %s/%s, want infrastructure/ephemeral", created.Class, created.Retention)
	}
	if created.LogicalID != "vrooli-bridge/node-credential-store/451ea636-a80f-4080-82b7-fa65d0e3289a" || created.Field != "passphrase" {
		t.Fatalf("grant address = %s:%s", created.LogicalID, created.Field)
	}
	if deliveries != 2 {
		t.Fatalf("deliveries = %d, want a delivery on every ensure", deliveries)
	}
	record, _ := json.Marshal(grants.grants)
	if strings.Contains(string(record), "escrowed-secret") {
		t.Fatal("the grant record must not carry the passphrase")
	}
}

// A revoked grant does not count as held; a fresh one is issued.
func TestNodeStoreGrantReplacesARevokedGrant(t *testing.T) {
	logicalID := internalonboard.CredentialStoreLogicalID("m1")
	grants := &fakeGrants{grants: []internalgrant.Grant{{
		NodeID: "node-1", LogicalID: logicalID, Field: "passphrase",
		Class: internalgrant.ClassInfrastructure, Retention: internalgrant.RetentionEphemeral, RevokedAt: time.Now(),
	}}}
	if err := NewNodeStoreGrantEnsurer(grants, nil).EnsureNodeStoreGrant(context.Background(), "node-1", logicalID, "passphrase"); err != nil {
		t.Fatal(err)
	}
	if len(grants.creates) != 1 {
		t.Fatalf("creates = %d, want a new grant after revocation", len(grants.creates))
	}
}

// The escrow address parses under the credential authority's identity grammar.
func TestCredentialStoreEscrowIdentityIsAccepted(t *testing.T) {
	if _, err := (authorityEscrow{}).identity("451ea636-a80f-4080-82b7-fa65d0e3289a"); err != nil {
		t.Fatalf("escrow identity rejected: %v", err)
	}
	if _, err := (authorityEscrow{}).identity("node-25c7e426-c76c-421a-8351-aaf964589802"); err != nil {
		t.Fatalf("node-keyed escrow identity rejected: %v", err)
	}
}
