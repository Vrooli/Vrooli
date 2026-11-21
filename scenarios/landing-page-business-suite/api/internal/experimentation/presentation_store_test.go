package experimentation

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/filerouting"
	"github.com/vrooli/api-core/storage"
	"landing-page-business-suite-api/internal/presentation"
)

func presentationStoreFixture(t *testing.T) (*ConfigStore, *filerouting.RoutedRoots) {
	t.Helper()
	dir := t.TempDir()
	roots := filerouting.New(storage.Paths{ConfigDir: dir})
	store := NewConfigStore(filepath.Join(dir, "variants"), "", nil)
	store.SetPresentationStorage(roots, func(context.Context, presentation.Document, *presentation.Document) error { return nil })
	return store, roots
}

func minimalPresentation() presentation.Document {
	page := presentation.Page{
		Display: presentation.PageDisplay{Shell: presentation.ShellDisplay{BrandName: "Suite", BrandMark: "suite", BrandTarget: "/", SkipLabel: "Skip to content", MenuLabel: "Menu", FooterBrandName: "Suite", FooterBrandMark: "suite", FooterBrandTarget: "/", UnavailableReason: "Unavailable", PreviewLabel: "Private preview"}},
		ID:      "empty", Locale: "en", Title: "A configured empty page", Description: "No products are published yet.",
		Theme:      presentation.Theme{Variant: "signal", Primary: "#263e38", Background: "#eaeade", Accent: "#df7958"},
		Navigation: presentation.Navigation{Label: "Navigation", Items: []presentation.NavigationItem{}},
		Footer:     presentation.Footer{Label: "Company", Links: []presentation.NavigationItem{}},
		Blocks:     []presentation.Block{},
	}
	return presentation.Document{
		SchemaVersion: 1,
		Bundle:        presentation.Bundle{Key: "business-suite", Name: "Business Suite", AppOrder: []string{}, MaxAppSlides: 0, PageID: "empty", EmptyPageID: "empty", DefaultLocale: "en", Locales: []string{"en"}},
		Apps:          []presentation.App{}, Pages: []presentation.Page{page}, Assets: []presentation.Asset{},
	}
}

func TestPresentationDraftPublishRollback(t *testing.T) { // [REQ:LP-PRES-004] [REQ:LP-PRES-012]
	ctx := context.Background()
	store, _ := presentationStoreFixture(t)
	doc := minimalPresentation()
	first, err := store.SavePresentationDraft(ctx, "control", doc, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.GetPublishedPresentation(ctx, "control"); !errors.Is(err, ErrPresentationNotFound) {
		t.Fatalf("draft became public: %v", err)
	}
	state, err := store.PublishPresentation(ctx, "control", first.DraftRevision, first.Generation)
	if err != nil {
		t.Fatal(err)
	}
	public, err := store.GetPublishedPresentation(ctx, "control")
	if err != nil || public.Revision != first.DraftRevision {
		t.Fatalf("published revision: %+v %v", public, err)
	}
	public.Document.Pages[0].Title = "caller mutation"
	again, err := store.GetPublishedPresentation(ctx, "control")
	if err != nil || again.Document.Pages[0].Title != doc.Pages[0].Title {
		t.Fatalf("mutable public cache: %+v %v", again, err)
	}
	doc.Pages[0].Title = "Updated configured page"
	second, err := store.SavePresentationDraft(ctx, "control", doc, state.Generation)
	if err != nil {
		t.Fatal(err)
	}
	state, err = store.PublishPresentation(ctx, "control", second.DraftRevision, second.Generation)
	if err != nil {
		t.Fatal(err)
	}
	rolled, err := store.RollbackPresentation(ctx, "control", first.DraftRevision, state.Generation)
	if err != nil || rolled.ActiveRevision != first.DraftRevision {
		t.Fatalf("rollback: %+v %v", rolled, err)
	}
	if len(rolled.PublishedRevisions) != 2 {
		t.Fatalf("lost revision history: %+v", rolled)
	}
	restarted := NewConfigStore("", "", nil)
	restarted.SetPresentationStorage(store.presentationRoots, store.presentationVerifier)
	reloaded, err := restarted.GetPublishedPresentation(ctx, "control")
	if err != nil || reloaded.Document.Pages[0].Title != "A configured empty page" {
		t.Fatalf("restart lost content: %+v %v", reloaded, err)
	}
}

func TestPresentationMutationConflictAndVerification(t *testing.T) { // [REQ:LP-PRES-004] [REQ:LP-PRES-009]
	ctx := context.Background()
	store, roots := presentationStoreFixture(t)
	doc := minimalPresentation()
	first, err := store.SavePresentationDraft(ctx, "control", doc, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.SavePresentationDraft(ctx, "control", doc, 0); !errors.Is(err, ErrPresentationConflict) {
		t.Fatalf("stale write: %v", err)
	}
	store.SetPresentationStorage(roots, nil)
	if _, err := store.PublishPresentation(ctx, "control", first.DraftRevision, first.Generation); !errors.Is(err, ErrPresentationUnqualified) {
		t.Fatalf("missing publication verifier accepted: %v", err)
	}
	store.SetPresentationStorage(roots, func(context.Context, presentation.Document, *presentation.Document) error { return errors.New("released asset bytes missing") })
	if _, err := store.PublishPresentation(ctx, "control", first.DraftRevision, first.Generation); !errors.Is(err, ErrPresentationUnqualified) {
		t.Fatalf("failed asset verification accepted: %v", err)
	}
	state, err := store.GetPresentationState(ctx, "control")
	if err != nil || state.Generation != first.Generation || state.ActiveRevision != "" {
		t.Fatalf("failed publication changed head: %+v %v", state, err)
	}
	if _, err := store.RollbackPresentation(ctx, "control", first.DraftRevision, state.Generation); !errors.Is(err, ErrPresentationNotFound) {
		t.Fatalf("never-published draft became rollback target: %v", err)
	}
}

func TestPresentationConcurrentWritersAndIsolation(t *testing.T) { // [REQ:LP-PRES-004]
	ctx := context.Background()
	store, roots := presentationStoreFixture(t)
	doc := minimalPresentation()
	var wg sync.WaitGroup
	results := make(chan error, 8)
	for range 8 {
		wg.Add(1)
		go func() { defer wg.Done(); _, err := store.SavePresentationDraft(ctx, "control", doc, 0); results <- err }()
	}
	wg.Wait()
	close(results)
	winners := 0
	for err := range results {
		if err == nil {
			winners++
		} else if !errors.Is(err, ErrPresentationConflict) {
			t.Fatal(err)
		}
	}
	if winners != 1 {
		t.Fatalf("expected one CAS winner, got %d", winners)
	}
	testCtx := database.WithTestMode(ctx)
	if _, err := store.SavePresentationDraft(testCtx, "control", doc, 0); err == nil {
		t.Fatal("test mode without lease wrote live")
	}
	if err := roots.InstallTestRoots(storage.Paths{ConfigDir: t.TempDir()}, "presentation-test", time.Minute); err != nil {
		t.Fatal(err)
	}
	testState, err := store.SavePresentationDraft(testCtx, "control", doc, 0)
	if err != nil || testState.Generation != 1 {
		t.Fatalf("isolated draft: %+v %v", testState, err)
	}
	live, err := store.GetPresentationState(ctx, "control")
	if err != nil || live.Generation != 1 {
		t.Fatalf("test changed live: %+v %v", live, err)
	}
	if roots.LeaseStats().PrimaryWritesDuringTestMode != 0 || roots.LeaseStats().TestRootWrites == 0 {
		t.Fatalf("routing stats: %+v", roots.LeaseStats())
	}
}

func TestPresentationIndependentStoresShareAtomicHead(t *testing.T) { // [REQ:LP-PRES-004] [REQ:LP-PRES-012]
	ctx := context.Background()
	_, roots := presentationStoreFixture(t)
	const writers = 12
	start := make(chan struct{})
	results := make(chan error, writers)
	for range writers {
		store := NewConfigStore("", "", nil)
		store.SetPresentationStorage(roots, nil)
		go func() {
			<-start
			_, err := store.SavePresentationDraft(ctx, "control", minimalPresentation(), 0)
			results <- err
		}()
	}
	close(start)
	winners := 0
	for range writers {
		err := <-results
		if err == nil {
			winners++
		} else if !errors.Is(err, ErrPresentationConflict) {
			t.Fatal(err)
		}
	}
	if winners != 1 {
		t.Fatalf("independent stores lost compare-and-swap isolation: %d writers won", winners)
	}
}

func TestPresentationPublicationRacesDraftEdit(t *testing.T) { // [REQ:LP-PRES-004]
	ctx := context.Background()
	store, roots := presentationStoreFixture(t)
	first, err := store.SavePresentationDraft(ctx, "control", minimalPresentation(), 0)
	if err != nil {
		t.Fatal(err)
	}
	verifying, release := make(chan struct{}), make(chan struct{})
	store.SetPresentationStorage(roots, func(context.Context, presentation.Document, *presentation.Document) error {
		close(verifying)
		<-release
		return nil
	})
	result := make(chan error, 1)
	go func() {
		_, err := store.PublishPresentation(ctx, "control", first.DraftRevision, first.Generation)
		result <- err
	}()
	<-verifying
	doc := minimalPresentation()
	doc.Pages[0].Title = "A newer editor draft"
	newer, saveErr := store.SavePresentationDraft(ctx, "control", doc, first.Generation)
	close(release)
	if err := <-result; !errors.Is(err, ErrPresentationConflict) {
		t.Fatalf("publication replaced a newer draft: %v", err)
	}
	if saveErr != nil {
		t.Fatal(saveErr)
	}
	state, err := store.GetPresentationState(ctx, "control")
	if err != nil || state.ActiveRevision != "" || state.DraftRevision != newer.DraftRevision {
		t.Fatalf("racing publication changed state: %+v %v", state, err)
	}
}

func TestPresentationCorruptionAndUnsafeReferencesFailClosed(t *testing.T) { // [REQ:LP-PRES-004] [REQ:LP-PRES-012]
	ctx := context.Background()
	store, roots := presentationStoreFixture(t)
	doc := minimalPresentation()
	if _, err := store.SavePresentationDraft(ctx, "../escape", doc, 0); err == nil {
		t.Fatal("unsafe variant accepted")
	}
	state, err := store.SavePresentationDraft(ctx, "control", doc, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.GetPresentationRevision(ctx, "control", "../head"); err == nil {
		t.Fatal("unsafe revision accepted")
	}
	dir, _ := roots.PickRequired(ctx, storage.ClassConfig)
	path := filepath.Join(dir, "presentations", "control", "revisions", state.DraftRevision+".json")
	if err := os.WriteFile(path, []byte(`{"schema_version":1,"corrupted":true}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := store.GetPresentationRevision(ctx, "control", state.DraftRevision); !errors.Is(err, ErrPresentationCorrupt) {
		t.Fatalf("corrupt revision returned: %v", err)
	}
	if _, err := store.SavePresentationDraft(ctx, "control", doc, state.Generation); !errors.Is(err, ErrPresentationCorrupt) {
		t.Fatalf("corrupt immutable bytes overwritten: %v", err)
	}
}
