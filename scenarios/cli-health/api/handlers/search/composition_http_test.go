package search

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCompositionSearchHTTPHandlerSourcePositiveAndNegative(t *testing.T) {
	handler := CompositionSearchHTTPHandler()
	call := func(body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/cli-composition/search", bytes.NewBufferString(body))
		rec := httptest.NewRecorder()
		handler(rec, req)
		return rec
	}
	positive := call(`{"query":"ScenarioApp command groups subcommand composition","scenario":"cli-health"}`)
	if positive.Code != http.StatusOK {
		t.Fatalf("positive status=%d body=%s", positive.Code, positive.Body.String())
	}
	var response struct {
		Scenario string `json:"scenario"`
		Scope    string `json:"scope"`
		Results  []struct {
			ID          string `json:"id"`
			Attestation string `json:"attestation"`
			Evidence    []struct {
				Path string `json:"path"`
			} `json:"evidence"`
		} `json:"results"`
		Degradations []struct {
			Code string `json:"code"`
		} `json:"degradations"`
	}
	if err := json.Unmarshal(positive.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if len(response.Results) != 1 || response.Results[0].ID != "cli-health:cli-composition:scenario-app" || response.Results[0].Attestation != "DERIVED" {
		t.Fatalf("source composition not surfaced: %+v", response)
	}
	if response.Scenario != "cli-health" || !strings.Contains(response.Scope, "scenario:cli-health") {
		t.Fatalf("resolved scope identity missing: %+v", response)
	}
	tooLarge := call(`{"query":"` + strings.Repeat("x", maxCompositionRequestBytes) + `","scenario":"cli-health"}`)
	if tooLarge.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("oversized status=%d want 413", tooLarge.Code)
	}
	negative := call(`{"query":"lunar chlorophyll gearbox","scenario":"cli-health"}`)
	if negative.Code != http.StatusOK {
		t.Fatalf("negative status=%d body=%s", negative.Code, negative.Body.String())
	}
	if err := json.Unmarshal(negative.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if len(response.Results) != 0 {
		t.Fatalf("negative query returned a strong hit: %+v", response.Results)
	}
	if response.Scenario != "cli-health" || !strings.Contains(response.Scope, "scenario:cli-health") {
		t.Fatalf("negative response lost its selected scenario scope: %+v", response)
	}
	invalidScope := call(`{"query":"CLI composition","scenario":"../outside"}`)
	if invalidScope.Code != http.StatusOK {
		t.Fatalf("invalid-scope status=%d body=%s", invalidScope.Code, invalidScope.Body.String())
	}
	if err := json.Unmarshal(invalidScope.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Scenario != "" || !strings.Contains(response.Scope, "unresolved") || len(response.Results) != 0 || len(response.Degradations) == 0 || response.Degradations[0].Code != "invalid_scope_or_query" {
		t.Fatalf("invalid scope must not claim a selected scenario: %+v", response)
	}
}
