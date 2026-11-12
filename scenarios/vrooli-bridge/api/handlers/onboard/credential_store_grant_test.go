package onboard

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/api-core/databasetest"
	internalgrant "vrooli-bridge/internal/credentialgrant"
	internalonboard "vrooli-bridge/internal/onboard"
)

type fakeGrants struct {
	repo    *databasetest.SliceRepo[internalgrant.Grant]
	creates []internalgrant.CreateInput
}

func newFakeGrants(initial ...internalgrant.Grant) *fakeGrants {
	repo := databasetest.NewSliceRepo(
		func(grant internalgrant.Grant) string { return grant.ID },
		func(grant *internalgrant.Grant, id string) { grant.ID = id },
		func(id string) error { return fmt.Errorf("grant %s not found", id) },
	)
	repo.Items = append(repo.Items, initial...)
	return &fakeGrants{repo: repo}
}

func (f *fakeGrants) List(ctx context.Context, _ string) ([]internalgrant.Grant, error) {
	return f.repo.List(ctx, int(^uint(0)>>1))
}

func (f *fakeGrants) Create(ctx context.Context, in internalgrant.CreateInput) (internalgrant.Grant, error) {
	f.creates = append(f.creates, in)
	grant := internalgrant.Grant{ID: "grant-1", NodeID: in.NodeID, LogicalID: in.LogicalID, Field: in.Field, Class: in.Class, Retention: in.Retention, Generation: 1, GrantedAt: time.Now()}
	return f.repo.Create(ctx, grant)
}

// The node's unlock grant is created once, as an ephemeral infrastructure
// grant for exactly the escrow address, and delivered on every ensure. The
// grant record is metadata; the passphrase never appears in it.
func TestNodeStoreGrantIsCreatedOnceAsEphemeralInfrastructure(t *testing.T) {
	grants := newFakeGrants()
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
	stored, err := grants.repo.List(context.Background(), int(^uint(0)>>1))
	if err != nil {
		t.Fatal(err)
	}
	record, _ := json.Marshal(stored)
	if strings.Contains(string(record), "escrowed-secret") {
		t.Fatal("the grant record must not carry the passphrase")
	}
}

// A revoked grant does not count as held; a fresh one is issued.
func TestNodeStoreGrantReplacesARevokedGrant(t *testing.T) {
	logicalID := internalonboard.CredentialStoreLogicalID("m1")
	grants := newFakeGrants(internalgrant.Grant{
		NodeID: "node-1", LogicalID: logicalID, Field: "passphrase",
		Class: internalgrant.ClassInfrastructure, Retention: internalgrant.RetentionEphemeral, RevokedAt: time.Now(),
	})
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
