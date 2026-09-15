package landing

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"slices"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/filerouting"
	"github.com/vrooli/api-core/schedule"
	"github.com/vrooli/api-core/storage"
	lpbsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1"
	lpbsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1/landing_page_business_suite_v1connect"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1/shared"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"landing-page-business-suite-api/internal/experimentation"
	"landing-page-business-suite-api/internal/presentation"
	"landing-page-business-suite-api/internal/presentationseed"
)

func presentationAdminFixture(t *testing.T) (*experimentation.ConfigStore, PresentationAdminHandler, presentation.Document) {
	t.Helper()
	store := experimentation.NewConfigStore(t.TempDir(), "", nil)
	if err := store.SaveVariant("control", &experimentation.VariantSnapshot{Variant: experimentation.VariantSnapshotMeta{
		Slug: "control", Name: "Control", Weight: 100, Status: "active",
		Axes: map[string]string{"persona": "silentFounder", "jtbd": "entrepreneurship", "conversionStyle": "emotional"},
	}}); err != nil {
		t.Fatalf("seed known preview variant: %v", err)
	}
	store.SetPresentationStorage(filerouting.New(storage.Paths{ConfigDir: t.TempDir()}), func(context.Context, presentation.Document) error { return nil })
	document := presentation.Document{
		SchemaVersion: 1,
		Bundle:        presentation.Bundle{Key: "business-suite", Name: "Business Suite", AppOrder: []string{}, MaxAppSlides: 0, PageID: "empty", EmptyPageID: "empty", DefaultLocale: "en", Locales: []string{"en"}},
		Apps:          []presentation.App{}, Assets: []presentation.Asset{},
		Pages: []presentation.Page{{ID: "empty", Locale: "en", Title: "A configured empty catalog", Description: "No products are published yet.", Theme: presentation.Theme{Variant: "signal", Primary: "#263e38", Background: "#eaeade", Accent: "#df7958"}, Navigation: presentation.Navigation{Label: "Navigation", Items: []presentation.NavigationItem{}}, Footer: presentation.Footer{Label: "Company", Links: []presentation.NavigationItem{}}, Blocks: []presentation.Block{}}},
	}
	document.Pages[0].Display = presentation.PageDisplay{Shell: presentation.ShellDisplay{BrandName: "Suite", BrandMark: "suite", BrandTarget: "/", SkipLabel: "Skip to content", MenuLabel: "Menu", FooterBrandName: "Suite", FooterBrandMark: "suite", FooterBrandTarget: "/", UnavailableReason: "Unavailable", PreviewLabel: "Private preview"}}
	return store, NewPresentationAdminHandler(store), document
}

func TestPresentationAdminEditorPreviewAndPublication(t *testing.T) { // [REQ:LP-PRES-004] [REQ:LP-PRES-012]
	ctx := context.Background()
	store, handler, document := presentationAdminFixture(t)
	wire, err := PresentationDocumentProto(document)
	if err != nil {
		t.Fatal(err)
	}
	draft, err := handler.SaveDraft(ctx, connect.NewRequest(&lpbsv1.SavePresentationDraftRequest{VariantSlug: "control", Document: wire}))
	if err != nil {
		t.Fatal(err)
	}
	if draft.Msg.GetState().GetGeneration() != 1 || draft.Msg.GetState().GetActiveRevision() != "" || len(draft.Msg.GetRevision()) != 64 {
		t.Fatalf("invalid draft response: %+v", draft.Msg)
	}
	if _, err := store.GetPublishedPresentation(ctx, "control"); !errors.Is(err, experimentation.ErrPresentationNotFound) {
		t.Fatalf("saved draft became public: %v", err)
	}
	preview, err := handler.Preview(ctx, connect.NewRequest(&lpbsv1.PreviewPresentationRequest{VariantSlug: "control", Revision: draft.Msg.Revision, Route: "/"}))
	if err != nil {
		t.Fatal(err)
	}
	if preview.Header().Get("Cache-Control") != "private, no-store" || preview.Header().Get("X-Robots-Tag") != "noindex, nofollow" || !preview.Msg.Presentation.Diagnostics.Preview || preview.Msg.Presentation.Diagnostics.ResolvedRevision != draft.Msg.Revision {
		t.Fatalf("preview cache or revision isolation missing: %+v", preview)
	}
	if _, err := handler.Publish(ctx, connect.NewRequest(&lpbsv1.ActivatePresentationRequest{VariantSlug: "control", Revision: draft.Msg.Revision, ExpectedGeneration: 0})); connect.CodeOf(err) != connect.CodeAborted {
		t.Fatalf("stale editor did not conflict: %v", err)
	}
	published, err := handler.Publish(ctx, connect.NewRequest(&lpbsv1.ActivatePresentationRequest{VariantSlug: "control", Revision: draft.Msg.Revision, ExpectedGeneration: 1}))
	if err != nil || published.Msg.State.ActiveRevision != draft.Msg.Revision || published.Msg.State.Generation != 2 {
		t.Fatalf("publish response: %+v %v", published, err)
	}
	loaded, err := handler.GetPresentation(ctx, connect.NewRequest(&lpbsv1.GetPresentationRequest{VariantSlug: "control"}))
	if err != nil || loaded.Msg.Document.Pages[0].Title != document.Pages[0].Title || loaded.Header().Get("Cache-Control") != "private, no-store" {
		t.Fatalf("editor readback: %+v %v", loaded, err)
	}
	if _, err := handler.Rollback(ctx, connect.NewRequest(&lpbsv1.ActivatePresentationRequest{VariantSlug: "control", Revision: strings.Repeat("c", 64), ExpectedGeneration: 2})); connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("unknown rollback target accepted: %v", err)
	}
}

// The local versioned legacy adapter uses the existing administrative string
// table for lossless recovery. This checks the authoritative storage/transport
// boundary, independently of the UI conversion implementation.
func TestPresentationLegacyRecoveryPersistsPrivatelyWithoutPublication(t *testing.T) { // [REQ:LP-PRES-002] [REQ:LP-PRES-004] [REQ:LP-PRES-012]
	ctx := context.Background()
	store, handler, _ := presentationAdminFixture(t)
	document, err := presentationseed.Recommended()
	if err != nil {
		t.Fatal(err)
	}
	source := "{\n  \"private_snapshot\": \"Plan — café\",\n  \"markup\": \"<script>PRIVATE-RECOVERY-MARKER</script>\"\n}\n"
	encoded := base64.StdEncoding.EncodeToString([]byte(source))
	const prefix = "legacy-import.fixture."
	document.Strings["en"][prefix+"source.0000"] = encoded[:64]
	document.Strings["en"][prefix+"source.0001"] = encoded[64:]
	document.Strings["en"][prefix+"receipt"] = "PRIVATE-MIGRATION-RECEIPT"
	for i := range document.Pages {
		if document.Pages[i].ID == document.Apps[1].PageID {
			document.Pages[i].Blocks = append(document.Pages[i].Blocks, presentation.Block{
				ID: "legacy-recovery", Kind: presentation.BlockProductStory, Version: 1, Variant: "three-column",
				Content: presentation.ProductStoryContent{Heading: "Private recovered narrative", Body: "Imported text awaiting review", Items: []presentation.StoryItem{}},
			})
		}
	}
	wire, err := PresentationDocumentProto(document)
	if err != nil {
		t.Fatal(err)
	}
	saved, err := handler.SaveDraft(ctx, connect.NewRequest(&lpbsv1.SavePresentationDraftRequest{VariantSlug: "control", Document: wire}))
	if err != nil {
		t.Fatal(err)
	}
	if saved.Msg.State.Generation != 1 || saved.Msg.State.ActiveRevision != "" {
		t.Fatalf("recovery save changed publication: %+v", saved.Msg.State)
	}
	if _, err := store.GetPublishedPresentation(ctx, "control"); !errors.Is(err, experimentation.ErrPresentationNotFound) {
		t.Fatalf("recovery draft became public: %v", err)
	}
	loaded, err := handler.GetPresentation(ctx, connect.NewRequest(&lpbsv1.GetPresentationRequest{VariantSlug: "control", Revision: saved.Msg.Revision}))
	if err != nil {
		t.Fatal(err)
	}
	values := loaded.Msg.Document.Strings["en"].Values
	recovered, err := base64.StdEncoding.DecodeString(values[prefix+"source.0000"] + values[prefix+"source.0001"])
	if err != nil || string(recovered) != source || values[prefix+"receipt"] != "PRIVATE-MIGRATION-RECEIPT" {
		t.Fatalf("lossless administrative recovery changed: %q %v", recovered, err)
	}
	if loaded.Header().Get("Cache-Control") != "private, no-store" {
		t.Fatal("administrative recovery was cacheable")
	}
	if _, err := handler.SaveDraft(ctx, connect.NewRequest(&lpbsv1.SavePresentationDraftRequest{VariantSlug: "control", Document: wire, ExpectedGeneration: 0})); connect.CodeOf(err) != connect.CodeAborted {
		t.Fatalf("stale recovery save bypassed generation guard: %v", err)
	}
	preview, err := handler.Preview(ctx, connect.NewRequest(&lpbsv1.PreviewPresentationRequest{VariantSlug: "control", Revision: saved.Msg.Revision, Route: "/apps/browser-automation-studio"}))
	if err != nil {
		t.Fatal(err)
	}
	if !preview.Msg.Presentation.Diagnostics.Preview || !preview.Msg.Presentation.Diagnostics.Noindex {
		t.Fatal("private recovery preview lost its labels")
	}
	previewJSON, err := protojson.Marshal(preview.Msg)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(previewJSON, []byte("Private recovered narrative")) {
		t.Fatal("explicit private preview omitted mapped recovery narrative")
	}
	for _, private := range []string{prefix, encoded[:64], "PRIVATE-MIGRATION-RECEIPT", "PRIVATE-RECOVERY-MARKER"} {
		if bytes.Contains(previewJSON, []byte(private)) {
			t.Fatalf("renderer projection leaked administrative recovery %q", private)
		}
	}
	if _, err := presentation.Resolve(document, presentation.ResolveRequest{Route: "/apps/browser-automation-studio"}); !errors.Is(err, presentation.ErrNotFound) {
		t.Fatalf("disabled recovery profile acquired a public route: %v", err)
	}
}

// Opt-in bridge for the actual browser adapter: export the canonical generated
// seed, then validate its imported result through the real administrative owner.
// No production server, source snapshot, or public revision is mutated.
func TestPresentationLegacyImportAdapterDocumentIntegration(t *testing.T) { // [REQ:LP-PRES-002] [REQ:LP-PRES-004]
	exportPath := os.Getenv("LPBS_LEGACY_SEED_WIRE_OUTPUT")
	importPath := os.Getenv("LPBS_LEGACY_IMPORT_DOCUMENT")
	if exportPath == "" && importPath == "" {
		t.Skip("opt-in: export a seed or validate an adapter-generated document")
	}
	if exportPath != "" {
		document, err := presentationseed.Recommended()
		if err != nil {
			t.Fatal(err)
		}
		wire, err := PresentationDocumentProto(document)
		if err != nil {
			t.Fatal(err)
		}
		data, err := (protojson.MarshalOptions{UseProtoNames: true, Indent: "  "}).Marshal(wire)
		if err != nil {
			t.Fatal(err)
		}
		file, err := os.OpenFile(exportPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		if err != nil {
			t.Fatal(err)
		}
		_, writeErr := file.Write(data)
		closeErr := file.Close()
		if writeErr != nil || closeErr != nil {
			t.Fatalf("write isolated seed fixture: %v %v", writeErr, closeErr)
		}
	}
	if importPath == "" {
		return
	}
	data, err := os.ReadFile(importPath)
	if err != nil || len(data) > 8<<20 {
		t.Fatalf("read bounded adapter document: %v", err)
	}
	var wire sharedv1.ProductPresentationDocument
	if err := protojson.Unmarshal(data, &wire); err != nil {
		t.Fatal(err)
	}
	source, err := os.ReadFile(os.Getenv("LPBS_LEGACY_IMPORT_SOURCE"))
	if err != nil || len(source) > 1<<20 {
		t.Fatalf("read bounded original source fixture: %v", err)
	}
	store, handler, _ := presentationAdminFixture(t)
	ctx := context.Background()
	saved, err := handler.SaveDraft(ctx, connect.NewRequest(&lpbsv1.SavePresentationDraftRequest{VariantSlug: "control", Document: &wire}))
	if err != nil {
		t.Fatalf("actual adapter document rejected by authoritative validation: %v", err)
	}
	if saved.Msg.State.ActiveRevision != "" || saved.Msg.State.Generation != 1 {
		t.Fatal("adapter document changed public publication")
	}
	loaded, err := handler.GetPresentation(ctx, connect.NewRequest(&lpbsv1.GetPresentationRequest{VariantSlug: "control", Revision: saved.Msg.Revision}))
	if err != nil || !proto.Equal(loaded.Msg.GetDocument(), &wire) {
		t.Fatalf("adapter document changed during typed storage roundtrip: %v", err)
	}
	values := loaded.Msg.Document.GetStrings()["en"].GetValues()
	var keys []string
	for key := range values {
		if strings.HasPrefix(key, "legacy-v1-") && strings.Contains(key, "-source-") {
			keys = append(keys, key)
		}
	}
	slices.Sort(keys)
	var recovered []byte
	for _, key := range keys {
		chunk, err := base64.StdEncoding.DecodeString(values[key])
		if err != nil {
			t.Fatalf("invalid source chunk: %v", err)
		}
		recovered = append(recovered, chunk...)
	}
	if len(keys) == 0 || !bytes.Equal(recovered, source) {
		t.Fatal("actual adapter source chunks are not byte-exact")
	}
	if _, err := store.GetPublishedPresentation(ctx, "control"); !errors.Is(err, experimentation.ErrPresentationNotFound) {
		t.Fatalf("actual adapter draft became public: %v", err)
	}
}

func TestPresentationAdminEphemeralPreviewIsReadOnlyAndContentIdentified(t *testing.T) { // [REQ:LP-PRES-012]
	store, _, document := presentationAdminFixture(t)
	handler := NewPresentationAdminHandler(store)
	wire, err := PresentationDocumentProto(document)
	if err != nil {
		t.Fatal(err)
	}
	decodedDocument, err := PresentationDocumentFromProto(wire)
	if err != nil {
		t.Fatal(err)
	}
	identity, err := presentation.PreviewIdentity(decodedDocument)
	if err != nil {
		t.Fatal(err)
	}
	before, err := store.GetPresentationState(context.Background(), "control")
	if err != nil {
		t.Fatal(err)
	}

	preview, err := handler.Preview(context.Background(), connect.NewRequest(&lpbsv1.PreviewPresentationRequest{
		VariantSlug: "control", Route: "/", Locale: "en", Document: wire,
	}))
	if err != nil {
		t.Fatal(err)
	}
	if preview.Header().Get("Cache-Control") != "private, no-store" || preview.Header().Get("X-Robots-Tag") != "noindex, nofollow" {
		t.Fatalf("ephemeral preview cache policy: %q %q", preview.Header().Get("Cache-Control"), preview.Header().Get("X-Robots-Tag"))
	}
	diagnostics := preview.Msg.GetPresentation().GetDiagnostics()
	if !diagnostics.GetPreview() || !diagnostics.GetNoindex() || !diagnostics.GetNoStore() || diagnostics.GetResolvedRevision() != identity || diagnostics.GetResolvedVariant() != "control" {
		t.Fatalf("ephemeral preview identity/diagnostics: %+v", diagnostics)
	}
	after, err := store.GetPresentationState(context.Background(), "control")
	if err != nil || after.Generation != before.Generation || after.ActiveRevision != before.ActiveRevision || after.DraftRevision != before.DraftRevision {
		t.Fatalf("ephemeral preview changed presentation state: before=%+v after=%+v err=%v", before, after, err)
	}
}

func TestPresentationAdminPreviewRequiresExactlyOneInput(t *testing.T) { // [REQ:LP-PRES-012]
	_, handler, document := presentationAdminFixture(t)
	wire, err := PresentationDocumentProto(document)
	if err != nil {
		t.Fatal(err)
	}
	for _, request := range []*lpbsv1.PreviewPresentationRequest{
		{VariantSlug: "control", Route: "/"},
		{VariantSlug: "control", Revision: "revision", Document: wire, Route: "/"},
		{VariantSlug: " control", Document: wire, Route: "/"},
	} {
		if _, err := handler.Preview(context.Background(), connect.NewRequest(request)); connect.CodeOf(err) != connect.CodeInvalidArgument {
			t.Fatalf("invalid preview request returned %v: %v", connect.CodeOf(err), err)
		}
	}
}

func TestPresentationAdminEphemeralPreviewRequiresActiveTestLease(t *testing.T) { // [REQ:LP-PRES-012]
	primaryDir := t.TempDir()
	store := experimentation.NewConfigStore(t.TempDir(), "", nil)
	if err := store.SaveVariant("control", &experimentation.VariantSnapshot{Variant: experimentation.VariantSnapshotMeta{
		Slug: "control", Name: "Control", Weight: 100, Status: "active",
		Axes: map[string]string{"persona": "silentFounder", "jtbd": "entrepreneurship", "conversionStyle": "emotional"},
	}}); err != nil {
		t.Fatalf("seed known preview variant: %v", err)
	}
	roots := filerouting.New(storage.Paths{ConfigDir: primaryDir})
	store.SetPresentationStorage(roots, func(context.Context, presentation.Document) error { return nil })
	handler := NewPresentationAdminHandler(store)
	document := presentationAdminDocumentForLeaseTest()
	wire, err := PresentationDocumentProto(document)
	if err != nil {
		t.Fatal(err)
	}
	testContext := database.WithTestMode(context.Background())
	request := connect.NewRequest(&lpbsv1.PreviewPresentationRequest{VariantSlug: "control", Route: "/", Document: wire})
	if _, err := handler.Preview(testContext, request); err == nil {
		t.Fatal("ephemeral preview accepted a missing test lease")
	}
	if stats := roots.LeaseStats(); stats.PrimaryWritesDuringTestMode != 0 || stats.TestRootWrites != 0 {
		t.Fatalf("missing lease caused a storage write: %+v", stats)
	}

	leaseDir := t.TempDir()
	if err := roots.InstallTestRoots(storage.Paths{ConfigDir: leaseDir}, "preview-lease", time.Minute); err != nil {
		t.Fatal(err)
	}
	if _, err := handler.Preview(testContext, request); err != nil {
		t.Fatalf("active lease rejected ephemeral preview: %v", err)
	}
	if stats := roots.LeaseStats(); stats.PrimaryWritesDuringTestMode != 0 || stats.TestRootWrites != 0 {
		t.Fatalf("read-only preview wrote storage: %+v", stats)
	}

	unknownRequest := connect.NewRequest(&lpbsv1.PreviewPresentationRequest{VariantSlug: "unknown", Route: "/", Document: wire})
	if _, err := handler.Preview(testContext, unknownRequest); connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("unknown preview variant returned %v: %v", connect.CodeOf(err), err)
	}
	if stats := roots.LeaseStats(); stats.PrimaryWritesDuringTestMode != 0 || stats.TestRootWrites != 0 {
		t.Fatalf("unknown variant check wrote storage: %+v", stats)
	}

	clock := schedule.NewFake(time.Now())
	roots.SetClock(clock)
	clock.Advance(2 * time.Minute)
	if _, err := handler.Preview(testContext, request); err == nil {
		t.Fatal("ephemeral preview accepted an expired test lease")
	}
	if stats := roots.LeaseStats(); stats.PrimaryWritesDuringTestMode != 0 || stats.TestRootWrites != 0 {
		t.Fatalf("expired lease caused a storage write: %+v", stats)
	}
}

func presentationAdminDocumentForLeaseTest() presentation.Document {
	return presentation.Document{
		SchemaVersion: 1,
		Bundle:        presentation.Bundle{Key: "business-suite", Name: "Business Suite", PageID: "empty", EmptyPageID: "empty", DefaultLocale: "en", Locales: []string{"en"}},
		Pages:         []presentation.Page{{ID: "empty", Locale: "en", Title: "Empty", Description: "Empty", Theme: presentation.Theme{Variant: "signal", Primary: "#263e38", Background: "#eaeade", Accent: "#df7958"}, Navigation: presentation.Navigation{Label: "Navigation"}, Footer: presentation.Footer{Label: "Footer"}, Display: presentation.PageDisplay{Shell: presentation.ShellDisplay{BrandName: "Suite", BrandMark: "suite", BrandTarget: "/", SkipLabel: "Skip", MenuLabel: "Menu", FooterBrandName: "Suite", FooterBrandMark: "suite", FooterBrandTarget: "/", UnavailableReason: "Unavailable", PreviewLabel: "Preview"}}}},
	}
}

func TestPresentationAdminMountProtectsEveryProcedure(t *testing.T) { // [REQ:LP-PRES-004] [REQ:LP-PRES-012]
	store, _, _ := presentationAdminFixture(t)
	router := mux.NewRouter()
	guardCalls := 0
	RegisterPresentationAdminRoutes(router, store, func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			guardCalls++
			http.Error(w, "administrator session required", http.StatusUnauthorized)
		}
	})
	procedures := []string{
		lpbsconnect.ProductPresentationAdminServiceGetPresentationProcedure,
		lpbsconnect.ProductPresentationAdminServiceSaveDraftProcedure,
		lpbsconnect.ProductPresentationAdminServicePreviewProcedure,
		lpbsconnect.ProductPresentationAdminServicePublishProcedure,
		lpbsconnect.ProductPresentationAdminServiceRollbackProcedure,
	}
	for _, procedure := range procedures {
		request := httptest.NewRequest(http.MethodPost, procedure, bytes.NewBufferString(`{"variant_slug":"control"}`))
		request.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		if response.Code != http.StatusUnauthorized {
			t.Fatalf("unguarded procedure %s: %d", procedure, response.Code)
		}
	}
	if guardCalls != len(procedures) {
		t.Fatalf("only %d/%d procedures traversed authentication", guardCalls, len(procedures))
	}
	state, err := store.GetPresentationState(context.Background(), "control")
	if err != nil || state.Generation != 0 {
		t.Fatalf("unauthenticated request changed state: %+v %v", state, err)
	}
}

func TestPresentationErrorsDoNotLeakPrivateStorage(t *testing.T) { // [REQ:LP-PRES-009] [REQ:LP-PRES-012]
	for _, input := range []error{
		errors.New("open /private/credentials/secret: missing"),
		fmt.Errorf("%w: owner evidence /private/proofs", experimentation.ErrPresentationUnqualified),
		fmt.Errorf("%w: /private/revisions", experimentation.ErrPresentationCorrupt),
	} {
		err := presentationConnectError(input)
		if strings.Contains(err.Error(), "/private") || strings.Contains(err.Error(), "secret") {
			t.Fatalf("private diagnostic leaked: %v", err)
		}
	}
}
