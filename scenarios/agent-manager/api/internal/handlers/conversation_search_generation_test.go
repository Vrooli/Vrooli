package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/mux"
	aisearch "github.com/vrooli/ai-go/search"
)

type generationOwnerFixture struct {
	previewed bool
	applied   bool
}

func (f *generationOwnerFixture) InspectGenerationLifecycle(context.Context, aisearch.GenerationRetentionPolicy) (aisearch.GenerationInspection, error) {
	return aisearch.GenerationInspection{Alias: "fixture"}, nil
}

func (f *generationOwnerFixture) PreviewGenerationCleanup(_ context.Context, _ aisearch.GenerationRetentionPolicy, identity string) (aisearch.GenerationCleanupPlan, error) {
	f.previewed = true
	return aisearch.GenerationCleanupPlan{PlanIdentity: identity, IdempotencyKey: identity, PolicyRevision: "test", SnapshotIdentity: "snapshot"}, nil
}

func (f *generationOwnerFixture) ApplyGenerationCleanup(_ context.Context, plan aisearch.GenerationCleanupPlan) (aisearch.GenerationCleanupReceipt, error) {
	f.applied = plan.Approved
	return aisearch.GenerationCleanupReceipt{IdempotencyKey: plan.IdempotencyKey}, nil
}

func TestConversationSearchGenerationHandlerRequiresOwnerTokenAndSupportsPreviewApply(t *testing.T) {
	owner := &generationOwnerFixture{}
	router := mux.NewRouter()
	NewConversationSearchGenerationHandler(owner, func() string { return "secret" }).RegisterRoutes(router)

	unauthorized := httptest.NewRequest(http.MethodGet, "/api/v1/conversation-search/generations", nil)
	unauthorizedResponse := httptest.NewRecorder()
	router.ServeHTTP(unauthorizedResponse, unauthorized)
	if unauthorizedResponse.Code != http.StatusUnauthorized {
		t.Fatalf("unauthorized status=%d, want %d", unauthorizedResponse.Code, http.StatusUnauthorized)
	}

	previewRequest := httptest.NewRequest(http.MethodPost, "/api/v1/conversation-search/generations/cleanup/preview", strings.NewReader(`{"planIdentity":"plan-1"}`))
	previewRequest.Header.Set("X-Search-Control-Token", "secret")
	previewResponse := httptest.NewRecorder()
	router.ServeHTTP(previewResponse, previewRequest)
	if previewResponse.Code != http.StatusOK || !owner.previewed {
		t.Fatalf("preview status=%d called=%v", previewResponse.Code, owner.previewed)
	}

	applyRequest := httptest.NewRequest(http.MethodPost, "/api/v1/conversation-search/generations/cleanup/apply", strings.NewReader(`{"idempotencyKey":"plan-1","policyRevision":"test","snapshotIdentity":"snapshot","approved":true}`))
	applyRequest.Header.Set("X-Search-Control-Token", "secret")
	applyResponse := httptest.NewRecorder()
	router.ServeHTTP(applyResponse, applyRequest)
	if applyResponse.Code != http.StatusOK || !owner.applied {
		t.Fatalf("apply status=%d approved=%v", applyResponse.Code, owner.applied)
	}
}
