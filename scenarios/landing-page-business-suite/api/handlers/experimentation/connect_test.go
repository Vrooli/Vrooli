package variant

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"connectrpc.com/connect"
	lpbsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1/shared"
	"google.golang.org/protobuf/types/known/structpb"
	"landing-page-business-suite-api/internal/experimentation"
)

func TestVariantConnectHandler_CreateArchiveAndSnapshotLifecycle(t *testing.T) {
	store := isolatedVariantStore(t)
	handler := newVariantConnectHandler(store)
	ctx := context.Background()

	created, err := handler.CreateVariant(ctx, connect.NewRequest(&lpbsv1.CreateVariantRequest{
		Slug: "connect-lifecycle", Name: "Connect Lifecycle", Weight: 25,
		Axes: map[string]string{"persona": "silentFounder", "jtbd": "entrepreneurship", "conversionStyle": "emotional"},
	}))
	if err != nil {
		t.Fatalf("CreateVariant() error = %v", err)
	}
	if got := created.Msg.GetVariant(); got.GetStatus() != "active" || got.GetSlug() != "connect-lifecycle" {
		t.Fatalf("created variant = %#v", got)
	}
	if _, err := handler.UpdateVariant(ctx, connect.NewRequest(&lpbsv1.UpdateVariantRequest{
		Slug: "connect-lifecycle", SeoConfig: &sharedv1.VariantSEOConfig{Title: "Connect title", CanonicalPath: "/connect"},
	})); err != nil {
		t.Fatalf("UpdateVariant() SEO configuration error = %v", err)
	}
	updated, err := handler.GetVariant(ctx, connect.NewRequest(&lpbsv1.GetVariantRequest{Slug: "connect-lifecycle"}))
	if err != nil {
		t.Fatalf("GetVariant() after SEO update error = %v", err)
	}
	if got := updated.Msg.GetVariant().GetSeoConfig(); got.GetTitle() != "Connect title" || got.GetCanonicalPath() != "/connect" {
		t.Fatalf("updated SEO configuration = %#v", got)
	}

	exported, err := handler.ExportVariantSnapshot(ctx, connect.NewRequest(&lpbsv1.ExportVariantSnapshotRequest{Slug: "connect-lifecycle"}))
	if err != nil {
		t.Fatalf("ExportVariantSnapshot() error = %v", err)
	}
	if len(exported.Msg.GetSnapshot().GetSections()) == 0 {
		t.Fatal("new variant must inherit control sections")
	}

	archived, err := handler.ArchiveVariant(ctx, connect.NewRequest(&lpbsv1.ArchiveVariantRequest{Slug: "connect-lifecycle"}))
	if err != nil {
		t.Fatalf("ArchiveVariant() error = %v", err)
	}
	if got := archived.Msg.GetVariant().GetStatus(); got != "archived" {
		t.Fatalf("archive status = %q, want archived", got)
	}
	if _, err := handler.GetPublicVariant(ctx, connect.NewRequest(&lpbsv1.GetPublicVariantRequest{Slug: "connect-lifecycle"})); connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("GetPublicVariant archived error = %v, code = %v", err, connect.CodeOf(err))
	}

	imported, err := handler.ImportVariantSnapshot(ctx, connect.NewRequest(&lpbsv1.ImportVariantSnapshotRequest{Slug: "connect-lifecycle", Snapshot: exported.Msg.GetSnapshot()}))
	if err != nil {
		t.Fatalf("ImportVariantSnapshot() error = %v", err)
	}
	if imported.Msg.GetSnapshot().GetSlug() != "connect-lifecycle" {
		t.Fatalf("imported snapshot = %#v", imported.Msg.GetSnapshot())
	}
	public, err := handler.GetPublicVariant(ctx, connect.NewRequest(&lpbsv1.GetPublicVariantRequest{Slug: "connect-lifecycle"}))
	if err != nil {
		t.Fatalf("GetPublicVariant restored error = %v", err)
	}
	if public.Msg.GetVariant().GetStatus() != "active" {
		t.Fatalf("restored status = %q", public.Msg.GetVariant().GetStatus())
	}
	synced, err := handler.SyncVariantSnapshots(ctx, connect.NewRequest(&lpbsv1.SyncVariantSnapshotsRequest{}))
	if err != nil {
		t.Fatalf("SyncVariantSnapshots() error = %v", err)
	}
	if synced.Msg.GetCount() <= 0 {
		t.Fatalf("synced count = %d", synced.Msg.GetCount())
	}
}

func TestVariantConnectHandler_RejectsInvalidLifecycleRequests(t *testing.T) {
	handler := newVariantConnectHandler(isolatedVariantStore(t))
	ctx := context.Background()
	if _, err := handler.CreateVariant(ctx, connect.NewRequest(&lpbsv1.CreateVariantRequest{Slug: "bad", Name: "Bad"})); connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("missing axes error = %v, code = %v", err, connect.CodeOf(err))
	}
	if _, err := handler.ListVariants(ctx, connect.NewRequest(&lpbsv1.ListVariantsRequest{StatusFilter: "deleted"})); connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("invalid filter error = %v, code = %v", err, connect.CodeOf(err))
	}
}

func TestVariantConnectHandler_ManagesSectionsByVariantScopedKey(t *testing.T) {
	handler := newVariantConnectHandler(isolatedVariantStore(t))
	ctx := context.Background()
	const slug = "section-contract"
	if _, err := handler.CreateVariant(ctx, connect.NewRequest(&lpbsv1.CreateVariantRequest{Slug: slug, Name: "Section Contract", Axes: map[string]string{"persona": "silentFounder", "jtbd": "entrepreneurship", "conversionStyle": "emotional"}})); err != nil {
		t.Fatalf("CreateVariant() error = %v", err)
	}
	listed, err := handler.GetVariantSections(ctx, connect.NewRequest(&lpbsv1.GetVariantSectionsRequest{Slug: slug}))
	if err != nil {
		t.Fatalf("GetVariantSections() error = %v", err)
	}
	if len(listed.Msg.GetSections()) == 0 || listed.Msg.GetSections()[0].GetKey() == "" {
		t.Fatalf("sections must expose persisted keys: %#v", listed.Msg.GetSections())
	}
	key := listed.Msg.GetSections()[0].GetKey()
	updated, err := handler.UpdateVariantSection(ctx, connect.NewRequest(&lpbsv1.UpdateVariantSectionRequest{
		Slug: slug, SectionKey: key, Content: &structpb.Struct{Fields: map[string]*structpb.Value{"title": structpb.NewStringValue("Updated title")}},
	}))
	if err != nil {
		t.Fatalf("UpdateVariantSection() error = %v", err)
	}
	if got := updated.Msg.GetSection().GetContent().GetFields()["title"].GetStringValue(); got != "Updated title" {
		t.Fatalf("updated content title = %q", got)
	}
	created, err := handler.CreateVariantSection(ctx, connect.NewRequest(&lpbsv1.CreateVariantSectionRequest{
		Slug: slug, Section: &sharedv1.ContentSection{SectionType: "cta", Content: &structpb.Struct{Fields: map[string]*structpb.Value{}}, Order: 99, Enabled: true},
	}))
	if err != nil {
		t.Fatalf("CreateVariantSection() error = %v", err)
	}
	createdKey := created.Msg.GetSection().GetKey()
	if createdKey == "" {
		t.Fatal("created section key is empty")
	}
	if _, err := handler.GetVariantSection(ctx, connect.NewRequest(&lpbsv1.GetVariantSectionRequest{Slug: slug, SectionKey: createdKey})); err != nil {
		t.Fatalf("GetVariantSection() error = %v", err)
	}
	deleted, err := handler.DeleteVariantSection(ctx, connect.NewRequest(&lpbsv1.DeleteVariantSectionRequest{Slug: slug, SectionKey: createdKey}))
	if err != nil || !deleted.Msg.GetDeleted() {
		t.Fatalf("DeleteVariantSection() response = %#v, error = %v", deleted, err)
	}
}

func isolatedVariantStore(t *testing.T) *experimentation.ConfigStore {
	t.Helper()
	source := filepath.Join("..", "..", "..", "config", "variants")
	target := filepath.Join(t.TempDir(), "variants")
	if err := os.MkdirAll(target, 0o700); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(source)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		data, err := os.ReadFile(filepath.Join(source, entry.Name()))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(target, entry.Name()), data, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	store := experimentation.NewConfigStore(target, filepath.Join(t.TempDir(), "branding.json"), experimentation.DefaultVariantSpace())
	if err := store.LoadAll(); err != nil {
		t.Fatal(err)
	}
	return store
}
