package targets_test

import (
	"context"
	"errors"
	"testing"

	"data-backup-manager/internal/sources"
	"data-backup-manager/internal/targets"
	"data-backup-manager/internal/targets/mocks"
)

// TestRegisterTarget_Idempotent proves OT-P0-001 / DBM-REG-001: registration
// is an upsert keyed by (owner, name). An identical re-register is a no-op
// (UpdatedAt unchanged, no Update call); a changed re-register updates in
// place; the row count stays at one throughout.
func TestRegisterTarget_Idempotent(t *testing.T) {
	ctx := context.Background()
	repo := mocks.NewFakeRepository()
	svc := targets.NewService(repo)

	in := targets.RegisterInput{
		Owner:      "prompt-manager",
		Name:       "store",
		SourceKind: sources.KindFilesystem,
		Locator:    "store/teams",
	}

	created, err := svc.Register(ctx, in)
	if err != nil {
		t.Fatalf("first register: %v", err)
	}
	if created.ID == "" {
		t.Fatal("created target has empty id")
	}
	if got := repo.Creates.Load(); got != 1 {
		t.Fatalf("Creates = %d, want 1", got)
	}

	// Re-register identical → no-op: same id, UpdatedAt unchanged, no Update.
	again, err := svc.Register(ctx, in)
	if err != nil {
		t.Fatalf("identical re-register: %v", err)
	}
	if again.ID != created.ID {
		t.Fatalf("id changed on no-op re-register: %q != %q", again.ID, created.ID)
	}
	if !again.UpdatedAt.Equal(created.UpdatedAt) {
		t.Fatalf("UpdatedAt bumped on no-op re-register: %v != %v", again.UpdatedAt, created.UpdatedAt)
	}
	if got := repo.Updates.Load(); got != 0 {
		t.Fatalf("Updates = %d after no-op re-register, want 0", got)
	}

	// Re-register changed → update in place: same id, new spec, bumped time.
	changed := in
	changed.Locator = "store/teams/director-swarm"
	updated, err := svc.Register(ctx, changed)
	if err != nil {
		t.Fatalf("changed re-register: %v", err)
	}
	if updated.ID != created.ID {
		t.Fatalf("id changed on update: %q != %q", updated.ID, created.ID)
	}
	if updated.Locator != changed.Locator {
		t.Fatalf("locator = %q, want %q", updated.Locator, changed.Locator)
	}
	if !updated.UpdatedAt.After(created.UpdatedAt) {
		t.Fatalf("UpdatedAt not advanced on update: %v !> %v", updated.UpdatedAt, created.UpdatedAt)
	}
	if got := repo.Updates.Load(); got != 1 {
		t.Fatalf("Updates = %d, want 1", got)
	}

	// Exactly one row for this key throughout.
	all, err := svc.List(ctx, "", 0)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(all) != 1 {
		t.Fatalf("row count = %d, want 1 (no duplicate on re-register)", len(all))
	}
}

// TestRegisterTarget_Validation pins the typed validation errors.
func TestRegisterTarget_Validation(t *testing.T) {
	ctx := context.Background()
	svc := targets.NewService(mocks.NewFakeRepository())

	cases := []struct {
		name  string
		in    targets.RegisterInput
		field string
	}{
		{"missing owner", targets.RegisterInput{Name: "x", SourceKind: sources.KindFilesystem, Locator: "p"}, "owner"},
		{"missing name", targets.RegisterInput{Owner: "o", SourceKind: sources.KindFilesystem, Locator: "p"}, "name"},
		{"bad kind", targets.RegisterInput{Owner: "o", Name: "x", SourceKind: "nope", Locator: "p"}, "source_kind"},
		{"missing locator", targets.RegisterInput{Owner: "o", Name: "x", SourceKind: sources.KindFilesystem}, "locator"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := svc.Register(ctx, tc.in)
			var invalid targets.ErrInvalidTarget
			if !errors.As(err, &invalid) {
				t.Fatalf("want ErrInvalidTarget, got %v", err)
			}
			if invalid.Field != tc.field {
				t.Fatalf("field = %q, want %q", invalid.Field, tc.field)
			}
		})
	}
}

// TestDeregisterAndReconstruct proves DBM-REG-002: deregistration removes a
// target, and the catalog is reconstructable — a fresh re-register sequence
// rebuilds an equivalent catalog (the manager DB is a cache, not the source of
// truth).
func TestDeregisterAndReconstruct(t *testing.T) {
	ctx := context.Background()
	repo := mocks.NewFakeRepository()
	svc := targets.NewService(repo)

	seed := []targets.RegisterInput{
		{Owner: "prompt-manager", Name: "store", SourceKind: sources.KindFilesystem, Locator: "store/teams"},
		{Owner: "swarm-manager", Name: "db", SourceKind: sources.KindSQLite, Locator: "data/swarm.db"},
	}
	for _, in := range seed {
		if _, err := svc.Register(ctx, in); err != nil {
			t.Fatalf("seed register %s/%s: %v", in.Owner, in.Name, err)
		}
	}

	// Deregister one; assert removed and gone.
	removed, err := svc.Deregister(ctx, "prompt-manager", "store")
	if err != nil {
		t.Fatalf("deregister: %v", err)
	}
	if !removed {
		t.Fatal("deregister reported not removed")
	}
	after, _ := svc.List(ctx, "", 0)
	if len(after) != 1 {
		t.Fatalf("post-deregister count = %d, want 1", len(after))
	}

	// Deregister a non-existent key → removed=false, no error.
	removed, err = svc.Deregister(ctx, "prompt-manager", "store")
	if err != nil {
		t.Fatalf("second deregister: %v", err)
	}
	if removed {
		t.Fatal("deregister of absent target reported removed")
	}

	// Reconstruct from a fresh re-register sequence (simulating boot) into a
	// brand-new store; assert the resulting specs are equivalent.
	rebuilt := targets.NewService(mocks.NewFakeRepository())
	for _, in := range seed {
		if _, err := rebuilt.Register(ctx, in); err != nil {
			t.Fatalf("rebuild register %s/%s: %v", in.Owner, in.Name, err)
		}
	}
	got, _ := rebuilt.List(ctx, "", 0)
	if len(got) != len(seed) {
		t.Fatalf("rebuilt count = %d, want %d", len(got), len(seed))
	}
	bySpec := map[string]targets.Target{}
	for _, t2 := range got {
		bySpec[t2.Owner+"/"+t2.Name] = t2
	}
	for _, in := range seed {
		t2, ok := bySpec[in.Owner+"/"+in.Name]
		if !ok {
			t.Fatalf("rebuilt catalog missing %s/%s", in.Owner, in.Name)
		}
		if t2.SourceKind != in.SourceKind || t2.Locator != in.Locator {
			t.Fatalf("rebuilt spec mismatch for %s/%s: kind=%q locator=%q", in.Owner, in.Name, t2.SourceKind, t2.Locator)
		}
	}
}
