package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"connectrpc.com/connect"
	lpbsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1"
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

func isolatedVariantStore(t *testing.T) *ConfigStore {
	t.Helper()
	source := filepath.Join("..", "config", "variants")
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
	store := NewConfigStore(target, filepath.Join(t.TempDir(), "branding.json"), defaultVariantSpace)
	if err := store.LoadAll(); err != nil {
		t.Fatal(err)
	}
	return store
}
