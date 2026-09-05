package looks

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	apidb "github.com/vrooli/api-core/database"

	db "github.com/vrooli/api-core/databasetest"

	looksv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/looks"
)

func newLooksDB(t *testing.T) *sql.DB {
	t.Helper()
	d := db.NewSQLite(t)
	if err := apidb.EnsureSchemas(context.Background(), d, apidb.SchemaProviderFunc(Schema)); err != nil {
		t.Fatalf("ensure schemas: %v", err)
	}
	return d
}

func sampleCustom() *looksv1.Look {
	return &looksv1.Look{
		Name: "Warm Sepia",
		Kind: looksv1.LookKind_LOOK_KIND_FILM,
		Steps: []*looksv1.LookStep{
			{Operation: "filter", Kind: looksv1.StepKind_STEP_KIND_DETERMINISTIC, Params: map[string]string{"filter": "sepia"}},
		},
	}
}

func TestListReturnsBuiltinsOnFreshDB(t *testing.T) {
	st := NewStore(newLooksDB(t))
	list, err := st.List(context.Background(), looksv1.LookKind_LOOK_KIND_UNSPECIFIED)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != len(BuiltinLooks()) {
		t.Fatalf("fresh list should be the built-ins (%d), got %d", len(BuiltinLooks()), len(list))
	}
	if !list[0].GetBuiltin() {
		t.Error("built-ins should sort first")
	}
}

func TestCreateAssignsSlugAndPersists(t *testing.T) {
	ctx := context.Background()
	st := NewStore(newLooksDB(t))

	created, err := st.Create(ctx, sampleCustom())
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if created.GetId() != "warm-sepia" {
		t.Errorf("slug id = %q, want warm-sepia", created.GetId())
	}
	if created.GetBuiltin() {
		t.Error("custom Look must not be builtin")
	}
	if created.GetCreatedAt() == "" || created.GetUpdatedAt() == "" {
		t.Error("timestamps should be set")
	}

	got, err := st.Get(ctx, "warm-sepia")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.GetName() != "Warm Sepia" {
		t.Errorf("round-trip name = %q", got.GetName())
	}

	// And it shows up in the list after the built-ins.
	list, _ := st.List(ctx, looksv1.LookKind_LOOK_KIND_UNSPECIFIED)
	if len(list) != len(BuiltinLooks())+1 {
		t.Errorf("list should include the custom Look, got %d", len(list))
	}
}

func TestCreateRejectsBuiltinCollision(t *testing.T) {
	ctx := context.Background()
	st := NewStore(newLooksDB(t))
	l := sampleCustom()
	l.Id = "noir" // collides with a built-in
	l.Name = "noir"
	_, err := st.Create(ctx, l)
	if !errors.Is(err, ErrIDCollision) {
		t.Fatalf("want ErrIDCollision, got %v", err)
	}
}

func TestCreateRejectsDuplicateCustom(t *testing.T) {
	ctx := context.Background()
	st := NewStore(newLooksDB(t))
	if _, err := st.Create(ctx, sampleCustom()); err != nil {
		t.Fatalf("first create: %v", err)
	}
	if _, err := st.Create(ctx, sampleCustom()); !errors.Is(err, ErrIDCollision) {
		t.Fatalf("want ErrIDCollision on duplicate, got %v", err)
	}
}

func TestCreateRejectsInvalid(t *testing.T) {
	ctx := context.Background()
	st := NewStore(newLooksDB(t))
	bad := &looksv1.Look{Name: "No Steps"}
	if _, err := st.Create(ctx, bad); !errors.Is(err, ErrInvalid) {
		t.Fatalf("want ErrInvalid for a Look with no steps, got %v", err)
	}
	badStep := sampleCustom()
	badStep.Name = "Bad Step"
	badStep.Steps[0].Operation = "not_a_real_op"
	if _, err := st.Create(ctx, badStep); !errors.Is(err, ErrInvalid) {
		t.Fatalf("want ErrInvalid for an unknown step op, got %v", err)
	}
}

func TestUpdateAndDeleteBuiltinRefused(t *testing.T) {
	ctx := context.Background()
	st := NewStore(newLooksDB(t))
	bi := builtinByID(t, "noir")
	bi.Name = "Hacked"
	if _, err := st.Update(ctx, bi); !errors.Is(err, ErrBuiltinReadOnly) {
		t.Fatalf("update built-in: want ErrBuiltinReadOnly, got %v", err)
	}
	if err := st.Delete(ctx, "noir"); !errors.Is(err, ErrBuiltinReadOnly) {
		t.Fatalf("delete built-in: want ErrBuiltinReadOnly, got %v", err)
	}
}

func TestUpdateCustomMutatesAndPreservesCreatedAt(t *testing.T) {
	ctx := context.Background()
	st := NewStore(newLooksDB(t))
	created, err := st.Create(ctx, sampleCustom())
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	upd := sampleCustom()
	upd.Id = created.GetId()
	upd.Name = "Cooler Sepia"
	updated, err := st.Update(ctx, upd)
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if updated.GetName() != "Cooler Sepia" {
		t.Errorf("name not updated: %q", updated.GetName())
	}
	if updated.GetCreatedAt() != created.GetCreatedAt() {
		t.Errorf("created_at must be preserved across update")
	}
}

func TestDeleteCustom(t *testing.T) {
	ctx := context.Background()
	st := NewStore(newLooksDB(t))
	if _, err := st.Create(ctx, sampleCustom()); err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := st.Delete(ctx, "warm-sepia"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := st.Get(ctx, "warm-sepia"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("want ErrNotFound after delete, got %v", err)
	}
	if err := st.Delete(ctx, "warm-sepia"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("delete missing: want ErrNotFound, got %v", err)
	}
}

func TestSetThumbnailPersists(t *testing.T) {
	ctx := context.Background()
	st := NewStore(newLooksDB(t))
	if _, err := st.Create(ctx, sampleCustom()); err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := st.SetThumbnail(ctx, "warm-sepia", "thumb/abc.png"); err != nil {
		t.Fatalf("set thumbnail: %v", err)
	}
	got, _ := st.Get(ctx, "warm-sepia")
	if got.GetThumbnailRef() != "thumb/abc.png" {
		t.Errorf("thumbnail_ref = %q", got.GetThumbnailRef())
	}
	if err := st.SetThumbnail(ctx, "noir", "thumb/x.png"); !errors.Is(err, ErrBuiltinReadOnly) {
		t.Fatalf("set thumbnail on built-in: want ErrBuiltinReadOnly, got %v", err)
	}
}

func TestListKindFilter(t *testing.T) {
	ctx := context.Background()
	st := NewStore(newLooksDB(t))
	film, err := st.List(ctx, looksv1.LookKind_LOOK_KIND_FILM)
	if err != nil {
		t.Fatalf("list film: %v", err)
	}
	if len(film) == 0 {
		t.Fatal("expected at least one FILM built-in")
	}
	for _, l := range film {
		if l.GetKind() != looksv1.LookKind_LOOK_KIND_FILM {
			t.Errorf("kind filter leaked %v", l.GetKind())
		}
	}
}
