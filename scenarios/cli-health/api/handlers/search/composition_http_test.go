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
		Results []struct {
			ID          string `json:"id"`
			Attestation string `json:"attestation"`
			Evidence    []struct {
				Path string `json:"path"`
			} `json:"evidence"`
		} `json:"results"`
	}
	if err := json.Unmarshal(positive.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if len(response.Results) != 1 || response.Results[0].ID != "cli-health:cli-composition:scenario-app" || response.Results[0].Attestation != "DERIVED" {
		t.Fatalf("source composition not surfaced: %+v", response)
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
}
