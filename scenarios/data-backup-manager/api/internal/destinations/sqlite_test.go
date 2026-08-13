package destinations_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	apidb "github.com/vrooli/api-core/database"

	localdb "data-backup-manager/internal/database"
	"data-backup-manager/internal/destinations"
	db "github.com/vrooli/api-core/databasetest"

	"github.com/vrooli/api-core/scheduletest"
)

// newDestSchemaDB returns a fresh sqlite handle with system + destinations schema
// applied, so repository tests get a real table without touching the central registry.
func newDestSchemaDB(t *testing.T) *sql.DB {
	t.Helper()
	d := db.NewSQLite(t)
	if err := apidb.EnsureSchemas(context.Background(), d,
		apidb.SchemaProviderFunc(localdb.SystemSchema),
		apidb.SchemaProviderFunc(destinations.Schema),
	); err != nil {
		t.Fatalf("ensure schemas: %v", err)
	}
	return d
}

// TestSQLiteRepository_RoundTrip pins the SQL-level semantics: create, get by
// id, get by name, update (cap_bytes, cap_policy), list ordering, delete, and
// the unique name constraint.
func TestSQLiteRepository_RoundTrip(t *testing.T) {
	ctx := context.Background()
	clk := scheduletest.New(time.Time{})
	repo := destinations.NewSQLiteRepository(newDestSchemaDB(t), clk)

	created, err := repo.Create(ctx, destinations.Destination{
		Name:                "primary",
		BackendKind:         destinations.BackendFilesystem,
		Location:            "/mnt/backup",
		CapBytes:            0,
		CapPolicy:           destinations.CapPolicyAlertBlock,
		EncryptionAlgorithm: "AES256-GCM-HMAC-SHA256",
		SecretRef:           "vrooli/kopia/primary:repository-passphrase",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if created.ID == "" || created.CreatedAt.IsZero() || !created.CreatedAt.Equal(created.UpdatedAt) {
		t.Fatalf("create did not populate id/timestamps: %+v", created)
	}

	gotID, err := repo.GetByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("get by id: %v", err)
	}
	if gotID.Name != "primary" || gotID.BackendKind != destinations.BackendFilesystem {
		t.Fatalf("get by id mismatch: %+v", gotID)
	}
	if gotID.EncryptionAlgorithm != "AES256-GCM-HMAC-SHA256" {
		t.Fatalf("EncryptionAlgorithm not round-tripped: %q", gotID.EncryptionAlgorithm)
	}

	gotName, err := repo.GetByName(ctx, "primary")
	if err != nil {
		t.Fatalf("get by name: %v", err)
	}
	if gotName.ID != created.ID {
		t.Fatalf("get by name returned wrong id: %q", gotName.ID)
	}

	// Update cap_bytes and cap_policy, advance the clock.
	clk.Advance(time.Second)
	created.CapBytes = 5000
	created.CapPolicy = destinations.CapPolicyAlertOnly
	updated, err := repo.Update(ctx, created)
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if !updated.UpdatedAt.After(updated.CreatedAt) {
		t.Fatalf("update did not advance UpdatedAt: %+v", updated)
	}
	reread, _ := repo.GetByID(ctx, created.ID)
	if reread.CapBytes != 5000 || reread.CapPolicy != destinations.CapPolicyAlertOnly {
		t.Fatalf("update not persisted: %+v", reread)
	}

	// List ordering by name.
	if _, err := repo.Create(ctx, destinations.Destination{
		Name:        "secondary",
		BackendKind: destinations.BackendS3,
		Location:    "my-bucket",
		CapPolicy:   destinations.CapPolicyAlertBlock,
	}); err != nil {
		t.Fatalf("create second: %v", err)
	}
	all, err := repo.List(ctx, 100)
	if err != nil || len(all) != 2 {
		t.Fatalf("list all: n=%d err=%v", len(all), err)
	}
	if all[0].Name != "primary" || all[1].Name != "secondary" {
		t.Fatalf("list not ordered by name: %+v", all)
	}

	// Delete.
	removed, err := repo.Delete(ctx, created.ID)
	if err != nil || !removed {
		t.Fatalf("delete: removed=%v err=%v", removed, err)
	}
	var notFound destinations.ErrDestinationNotFound
	if _, err := repo.GetByID(ctx, created.ID); !errors.As(err, &notFound) {
		t.Fatalf("expected ErrDestinationNotFound after delete, got %v", err)
	}
}

// TestSQLiteRepository_UniqueNameConstraint proves the unique name constraint
// rejects a duplicate insert (the service validates before inserting, but the
// constraint is the backstop).
func TestSQLiteRepository_UniqueNameConstraint(t *testing.T) {
	ctx := context.Background()
	repo := destinations.NewSQLiteRepository(newDestSchemaDB(t), scheduletest.New(time.Time{}))
	base := destinations.Destination{
		Name:        "dup",
		BackendKind: destinations.BackendFilesystem,
		Location:    "/mnt/dup",
		CapPolicy:   destinations.CapPolicyAlertBlock,
	}
	if _, err := repo.Create(ctx, base); err != nil {
		t.Fatalf("first create: %v", err)
	}
	if _, err := repo.Create(ctx, base); err == nil {
		t.Fatal("expected unique-constraint failure on duplicate name")
	}
}
