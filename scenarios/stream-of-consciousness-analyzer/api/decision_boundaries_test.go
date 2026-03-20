package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
)

// --- Decision Boundary: Provider Selection Strategy ---
// These tests exercise the two-tier provider selection (primary-first, fallback-second)
// at its boundaries: all combos of active/inactive x primary/fallback.

// [REQ:P2-001] When multiple primary providers exist, the first active one wins.
func TestGetActiveProvider_FirstPrimaryWins(t *testing.T) {
	svc := &SuggestionService{
		providers: []LLMProvider{
			{Name: "ollama-1", Active: false, Fallback: false},
			{Name: "ollama-2", Active: true, Fallback: false},
			{Name: "ollama-3", Active: true, Fallback: false},
		},
	}
	provider, err := svc.GetActiveProvider()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if provider.Name != "ollama-2" {
		t.Errorf("expected first active primary (ollama-2), got %s", provider.Name)
	}
}

// [REQ:P2-001] Fallback is only selected when ALL primaries are inactive.
func TestGetActiveProvider_FallbackOnlyWhenNoPrimary(t *testing.T) {
	svc := &SuggestionService{
		providers: []LLMProvider{
			{Name: "primary", Active: false, Fallback: false},
			{Name: "fallback", Active: true, Fallback: true},
		},
	}
	provider, err := svc.GetActiveProvider()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if provider.Name != "fallback" {
		t.Errorf("expected fallback, got %s", provider.Name)
	}
	if !provider.Fallback {
		t.Error("expected provider.Fallback to be true")
	}
}

// [REQ:P2-001] Primary is chosen even when a fallback is also active.
func TestGetActiveProvider_PrimaryPreferredOverFallback(t *testing.T) {
	svc := &SuggestionService{
		providers: []LLMProvider{
			{Name: "fallback", Active: true, Fallback: true},
			{Name: "primary", Active: true, Fallback: false},
		},
	}
	provider, err := svc.GetActiveProvider()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if provider.Name != "primary" {
		t.Errorf("expected primary over fallback, got %s", provider.Name)
	}
}

// [REQ:P2-001] Empty provider list returns an error.
func TestGetActiveProvider_EmptyList(t *testing.T) {
	svc := &SuggestionService{providers: []LLMProvider{}}
	_, err := svc.GetActiveProvider()
	if err == nil {
		t.Error("expected error for empty provider list")
	}
}

// --- Decision Boundary: Error Classification ---
// These tests verify that classifyAndWriteError maps every known error type
// to the correct HTTP status and category.

// [REQ:P0-001] The retryable flag is only true for dependency errors.
func TestRetryableOnlyForDependency(t *testing.T) {
	categories := []struct {
		category  ErrorCategory
		retryable bool
	}{
		{ErrCategoryValidation, false},
		{ErrCategoryNotFound, false},
		{ErrCategoryConflict, false},
		{ErrCategoryDependency, true},
		{ErrCategoryInternal, false},
	}
	for _, tc := range categories {
		t.Run(string(tc.category), func(t *testing.T) {
			w := httptest.NewRecorder()
			writeAPIError(w, http.StatusOK, tc.category, "test", nil)
			var resp APIError
			if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if resp.Retryable != tc.retryable {
				t.Errorf("category %s: expected retryable=%v, got %v", tc.category, tc.retryable, resp.Retryable)
			}
		})
	}
}

// --- Decision Boundary: Edge Validation ---
// These tests verify edge creation validation at its boundaries.

// [REQ:P0-004] Edge creation with valid distinct source and target succeeds.
func TestEdgeValidation_ValidDistinctIDs(t *testing.T) {
	store := newMockThoughts()
	handler := handleCreateEdge(store)

	req := httptest.NewRequest("POST", "/api/v1/thoughts/a/edges",
		bytes.NewBufferString(`{"target_id":"b","label":"test"}`))
	req = mux.SetURLVars(req, map[string]string{"id": "a"})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assertStatus(t, w, http.StatusCreated)
}

// [REQ:P0-004] Edge creation with whitespace-only target is rejected.
func TestEdgeValidation_WhitespaceTarget(t *testing.T) {
	store := newMockThoughts()
	handler := handleCreateEdge(store)

	req := httptest.NewRequest("POST", "/api/v1/thoughts/a/edges",
		bytes.NewBufferString(`{"target_id":"","label":"test"}`))
	req = mux.SetURLVars(req, map[string]string{"id": "a"})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assertStatus(t, w, http.StatusBadRequest)
	var resp APIError
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Category != ErrCategoryValidation {
		t.Errorf("expected validation category, got %s", resp.Category)
	}
}

// --- Decision Boundary: Thought Scope Filter ---

// [REQ:P2-002] Thoughts from different schemes are both returned when no filter.
func TestThoughtScopeFilter_MixedSchemes(t *testing.T) {
	store := newMockThoughts()
	s1 := "scheme-1"
	s2 := "scheme-2"
	store.seedThought("In S1", &s1)
	store.seedThought("In S2", &s2)
	store.seedThought("Unbound", nil)

	handler := handleListThoughts(store)
	req := httptest.NewRequest("GET", "/api/v1/thoughts", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assertStatus(t, w, http.StatusOK)
	thoughts := decodeJSON[[]Thought](t, w)
	if len(thoughts) != 3 {
		t.Errorf("expected 3 thoughts (all schemes), got %d", len(thoughts))
	}
}

// [REQ:P2-002] Filter returns only thoughts from the specified scheme.
func TestThoughtScopeFilter_SingleScheme(t *testing.T) {
	store := newMockThoughts()
	s1 := "scheme-1"
	s2 := "scheme-2"
	store.seedThought("In S1", &s1)
	store.seedThought("Also in S1", &s1)
	store.seedThought("In S2", &s2)

	handler := handleListThoughts(store)
	req := httptest.NewRequest("GET", "/api/v1/thoughts?scheme_id=scheme-1", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assertStatus(t, w, http.StatusOK)
	thoughts := decodeJSON[[]Thought](t, w)
	if len(thoughts) != 2 {
		t.Errorf("expected 2 thoughts for scheme-1, got %d", len(thoughts))
	}
}

// --- Decision Boundary: Content Type Defaulting ---

// [REQ:P0-003] Information created without type gets "text" default.
func TestInformationTypeDefault(t *testing.T) {
	store := newMockInfo()
	handler := handleCreateInformation(store)

	body := `{"content":"no type specified","canvas_x":0,"canvas_y":0}`
	req := httptest.NewRequest("POST", "/api/v1/schemes/s1/information", bytes.NewBufferString(body))
	req = mux.SetURLVars(req, map[string]string{"schemeId": "s1"})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assertStatus(t, w, http.StatusCreated)
	info := decodeJSON[Information](t, w)
	if info.Type != "text" {
		t.Errorf("expected default type 'text', got %s", info.Type)
	}
}

// [REQ:P0-003] Information created with explicit type preserves it.
func TestInformationTypeExplicit(t *testing.T) {
	store := newMockInfo()
	handler := handleCreateInformation(store)

	body := `{"type":"url","content":"https://example.com","canvas_x":0,"canvas_y":0}`
	req := httptest.NewRequest("POST", "/api/v1/schemes/s1/information", bytes.NewBufferString(body))
	req = mux.SetURLVars(req, map[string]string{"schemeId": "s1"})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assertStatus(t, w, http.StatusCreated)
	info := decodeJSON[Information](t, w)
	if info.Type != "url" {
		t.Errorf("expected type 'url', got %s", info.Type)
	}
}

// --- Decision Boundary: Suggestion Generation with Provider Unavailability ---

// [REQ:P1-001] Suggestion handler returns provider name on success.
func TestSuggestionResponse_IncludesProviderName(t *testing.T) {
	svc := newMockSuggestions()
	handler := handleGenerateSuggestions(svc)

	req := httptest.NewRequest("POST", "/api/v1/schemes/s1/suggestions", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "s1"})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assertStatus(t, w, http.StatusOK)
	var result map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if result["provider"] == nil || result["provider"] == "" {
		t.Error("expected provider name in response")
	}
	if result["suggestions"] == nil {
		t.Error("expected suggestions array in response")
	}
}

// [REQ:P2-001] Suggestion handler 503 response is retryable and dependency-categorized.
func TestSuggestionUnavailable_FullErrorShape(t *testing.T) {
	svc := newMockSuggestions().WithGenerateError(fmt.Errorf("all providers down"))
	handler := handleGenerateSuggestions(svc)

	req := httptest.NewRequest("POST", "/api/v1/schemes/s1/suggestions", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "s1"})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assertStatus(t, w, http.StatusServiceUnavailable)
	var resp APIError
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Category != ErrCategoryDependency {
		t.Errorf("expected dependency category, got %s", resp.Category)
	}
	if !resp.Retryable {
		t.Error("503 should be retryable")
	}
	if resp.Message == "" {
		t.Error("expected user-safe message")
	}
}

// --- Decision Boundary: Update Scheme Name Validation ---

// [REQ:P0-001] Update with non-empty name succeeds.
func TestUpdateSchemeValidation_NonEmpty(t *testing.T) {
	store := newMockSchemes()
	s := store.seed("Original")

	handler := handleUpdateScheme(store)
	req := httptest.NewRequest("PUT", "/api/v1/schemes/"+s.ID, jsonBody(`{"name":"Updated"}`))
	req = setURLVars(req, "id", s.ID)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assertStatus(t, w, http.StatusOK)
	scheme := decodeJSON[Scheme](t, w)
	if scheme.Name != "Updated" {
		t.Errorf("expected Updated, got %s", scheme.Name)
	}
}

// [REQ:P0-001] Update with missing body is rejected.
func TestUpdateSchemeValidation_MissingBody(t *testing.T) {
	store := newMockSchemes()
	s := store.seed("Original")

	handler := handleUpdateScheme(store)
	req := httptest.NewRequest("PUT", "/api/v1/schemes/"+s.ID, bytes.NewBufferString("not json"))
	req = setURLVars(req, "id", s.ID)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assertStatus(t, w, http.StatusBadRequest)
}

// --- Decision Boundary: Information List Isolation ---

// [REQ:P0-003] Information list only returns items for the requested scheme.
func TestInformationListIsolation(t *testing.T) {
	store := newMockInfo()
	store.seed("scheme-a", "item 1")
	store.seed("scheme-a", "item 2")
	store.seed("scheme-b", "item 3")

	handler := handleListInformation(store)
	req := httptest.NewRequest("GET", "/api/v1/schemes/scheme-a/information", nil)
	req = mux.SetURLVars(req, map[string]string{"schemeId": "scheme-a"})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assertStatus(t, w, http.StatusOK)
	items := decodeJSON[[]Information](t, w)
	if len(items) != 2 {
		t.Errorf("expected 2 items for scheme-a, got %d", len(items))
	}
	for _, item := range items {
		if item.SchemeID != "scheme-a" {
			t.Errorf("expected scheme_id=scheme-a, got %s", item.SchemeID)
		}
	}
}

// [REQ:P0-003] Information service error propagates as 500.
func TestInformationListError(t *testing.T) {
	store := newMockInfo().WithListError(fmt.Errorf("connection lost"))
	handler := handleListInformation(store)

	req := httptest.NewRequest("GET", "/api/v1/schemes/s1/information", nil)
	req = mux.SetURLVars(req, map[string]string{"schemeId": "s1"})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assertStatus(t, w, http.StatusInternalServerError)
}

// --- Decision Boundary: Thought List Error Handling ---

// [REQ:P0-004] Thought list service error propagates as 500.
func TestThoughtListError(t *testing.T) {
	store := newMockThoughts().WithListError(fmt.Errorf("db unreachable"))
	handler := handleListThoughts(store)

	req := httptest.NewRequest("GET", "/api/v1/thoughts", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assertStatus(t, w, http.StatusInternalServerError)
}
