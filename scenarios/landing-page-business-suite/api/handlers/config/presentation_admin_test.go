package landing

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/filerouting"
	"github.com/vrooli/api-core/storage"
	lpbsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1"
	lpbsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1/landing_page_business_suite_v1connect"
	"landing-page-business-suite-api/internal/experimentation"
	"landing-page-business-suite-api/internal/presentation"
)

func presentationAdminFixture(t *testing.T) (*experimentation.ConfigStore, PresentationAdminHandler, presentation.Document) {
	t.Helper()
	store := experimentation.NewConfigStore("", "", nil)
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
