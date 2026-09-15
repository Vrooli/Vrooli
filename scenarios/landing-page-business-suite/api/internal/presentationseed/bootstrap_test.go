package presentationseed

import (
	"context"
	"errors"
	"testing"

	"github.com/vrooli/api-core/filerouting"
	"github.com/vrooli/api-core/storage"
	"landing-page-business-suite-api/internal/experimentation"
	"landing-page-business-suite-api/internal/presentation"
)

func TestSeedOnlyInitializesMissingDraftAndPreservesEditorWork(t *testing.T) { // [REQ:LP-PRES-004] [REQ:LP-PRES-008]
	ctx := context.Background()
	roots := filerouting.New(storage.Paths{ConfigDir: t.TempDir()})
	store := experimentation.NewConfigStore("", "", nil)
	store.SetPresentationStorage(roots, nil)
	first, created, err := EnsureDraft(ctx, store, "control", "business_suite")
	if err != nil || !created || first.Generation != 1 || first.ActiveRevision != "" {
		t.Fatalf("initial seed: %+v %t %v", first, created, err)
	}
	if _, err := store.GetPublishedPresentation(ctx, "control"); !errors.Is(err, experimentation.ErrPresentationNotFound) {
		t.Fatalf("seed leaked publication: %v", err)
	}
	loaded, err := store.GetPresentationRevision(ctx, "control", first.DraftRevision)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Document.Bundle.Key != "business_suite" {
		t.Fatalf("draft bundle key %q does not match configured commerce/delivery owner business_suite", loaded.Document.Bundle.Key)
	}
	loaded.Document.Pages[0].Title = "An editor's custom Aquila title"
	custom, err := store.SavePresentationDraft(ctx, "control", loaded.Document, first.Generation)
	if err != nil {
		t.Fatal(err)
	}
	store.SetPresentationStorage(roots, func(context.Context, presentation.Document) error { return nil }) // isolated receipt fixture only
	published, err := store.PublishPresentation(ctx, "control", custom.DraftRevision, custom.Generation)
	if err != nil {
		t.Fatal(err)
	}
	restarted := experimentation.NewConfigStore("", "", nil)
	restarted.SetPresentationStorage(roots, nil)
	again, created, err := EnsureDraft(ctx, restarted, "control", "a-different-owner")
	if err != nil || created || again.Generation != published.Generation || again.ActiveRevision != published.ActiveRevision {
		t.Fatalf("restart overwrote user work: %+v %t %v", again, created, err)
	}
}

func TestSeedRequiresRoutedStorageAndFreshDocuments(t *testing.T) {
	store := experimentation.NewConfigStore("", "", nil)
	if _, _, err := EnsureDraft(context.Background(), store, "control", "business_suite"); err == nil {
		t.Fatal("missing storage silently accepted")
	}
	a, err := Recommended()
	if err != nil {
		t.Fatal(err)
	}
	a.Pages[0].Title = "mutated"
	b, err := Recommended()
	if err != nil {
		t.Fatal(err)
	}
	if b.Pages[0].Title == "mutated" {
		t.Fatal("shipped document shared mutable memory")
	}
}

func TestSeedUsesConfiguredBundleIdentityWithoutChangingPreservedFacts(t *testing.T) { // [REQ:LP-PRES-003]
	ctx := context.Background()
	store := experimentation.NewConfigStore("", "", nil)
	store.SetPresentationStorage(filerouting.New(storage.Paths{ConfigDir: t.TempDir()}), nil)
	state, created, err := EnsureDraft(ctx, store, "control", "configured-owner-bundle")
	if err != nil || !created {
		t.Fatalf("initialize configured bundle: %t %v", created, err)
	}
	revision, err := store.GetPresentationRevision(ctx, "control", state.DraftRevision)
	if err != nil {
		t.Fatal(err)
	}
	if revision.Document.Bundle.Key != "configured-owner-bundle" {
		t.Fatal("bootstrap guessed the template bundle key")
	}
	original, err := Recommended()
	if err != nil {
		t.Fatal(err)
	}
	if revision.Document.Strings["en"]["bas.preservation.bundle_key"] != original.Strings["en"]["bas.preservation.bundle_key"] {
		t.Fatal("bootstrap rewrote preserved historical delivery facts")
	}
	for _, key := range []string{"", " ", " invalid-owner "} {
		if _, _, err := EnsureDraft(ctx, store, "new-variant", key); err == nil {
			t.Fatalf("invalid owner identity %q was accepted", key)
		}
	}
}
