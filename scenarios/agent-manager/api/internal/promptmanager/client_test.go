package promptmanager

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"agent-manager/internal/testutil/httpx"
)

func TestNewHTTPClient_Defaults(t *testing.T) {
	client := NewHTTPClient()
	if client == nil {
		t.Fatal("expected client")
	}
	if client.baseURLResolver == nil {
		t.Fatal("expected baseURLResolver")
	}
	if client.httpClient == nil {
		t.Fatal("expected httpClient")
	}
}

func TestNewHTTPClientWithResolver_Defaults(t *testing.T) {
	client := NewHTTPClientWithResolver(nil, nil)
	if client.baseURLResolver == nil {
		t.Fatal("expected default resolver")
	}
	if client.httpClient == nil {
		t.Fatal("expected default http client")
	}
}

func TestHTTPClient_ReadSkill(t *testing.T) {
	client := NewHTTPClientWithResolver(
		func(_ context.Context) (string, error) { return "http://localhost:12345", nil },
		httpx.DoerFunc(func(req *http.Request) (*http.Response, error) {
			if req.Method != http.MethodPost {
				t.Errorf("expected POST, got %s", req.Method)
			}
			if req.URL.Path != "/api/v1/skills/read" {
				t.Errorf("expected /api/v1/skills/read, got %s", req.URL.Path)
			}

			var rr readRequest
			if err := json.NewDecoder(req.Body).Decode(&rr); err != nil {
				t.Fatalf("decode request: %v", err)
			}
			if len(rr.Identifiers) != 1 || rr.Identifiers[0] != "test-skill" {
				t.Errorf("expected identifier test-skill, got %v", rr.Identifiers)
			}
			if rr.Variables["ITEM_NAME"] != "my-item" {
				t.Errorf("expected variable ITEM_NAME=my-item, got %s", rr.Variables["ITEM_NAME"])
			}
			if rr.Output != "combined" {
				t.Errorf("expected output combined, got %s", rr.Output)
			}

			resp := readResponse{Combined: "rendered prompt content"}
			body, _ := json.Marshal(resp)
			return httpx.Response(http.StatusOK, string(body)), nil
		}),
	)

	result, err := client.ReadSkill(context.Background(), "test-skill", map[string]string{"ITEM_NAME": "my-item"}, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "rendered prompt content" {
		t.Errorf("expected 'rendered prompt content', got %q", result)
	}
}

func TestHTTPClient_ReadSkill_ServerError(t *testing.T) {
	client := NewHTTPClientWithResolver(
		func(_ context.Context) (string, error) { return "http://localhost:12345", nil },
		httpx.DoerFunc(func(_ *http.Request) (*http.Response, error) {
			return httpx.Response(http.StatusInternalServerError, "internal error"), nil
		}),
	)

	_, err := client.ReadSkill(context.Background(), "test-skill", nil, false)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "status 500") {
		t.Errorf("expected status 500 in error, got %v", err)
	}
}

func TestHTTPClient_ReadSkillSource_PinsWhenNoExperiment(t *testing.T) {
	client := NewHTTPClientWithResolver(
		func(_ context.Context) (string, error) { return "http://localhost:12345", nil },
		httpx.DoerFunc(func(req *http.Request) (*http.Response, error) {
			var rr readRequest
			if err := json.NewDecoder(req.Body).Decode(&rr); err != nil {
				t.Fatalf("decode request: %v", err)
			}
			if rr.VariantPolicy != "pinned" {
				t.Errorf("expected variantPolicy pinned, got %q", rr.VariantPolicy)
			}
			if rr.ExperimentID != "" {
				t.Errorf("expected empty experimentId, got %q", rr.ExperimentID)
			}
			resp := readResponse{
				Combined:     "content",
				CombinedHash: "sha256:abc",
				Skills: []struct {
					ID          string `json:"id"`
					Revision    int    `json:"revision,omitempty"`
					ContentHash string `json:"contentHash,omitempty"`
				}{{ID: "test-skill", Revision: 3}},
			}
			body, _ := json.Marshal(resp)
			return httpx.Response(http.StatusOK, string(body)), nil
		}),
	)

	snap, err := client.ReadSkillSource(context.Background(), "test-skill", "", nil, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if snap.VariantID != "control" {
		t.Errorf("expected control variant, got %q", snap.VariantID)
	}
	if snap.ExperimentID != "" {
		t.Errorf("expected empty experimentId, got %q", snap.ExperimentID)
	}
}

func TestHTTPClient_ReadSkillSource_ArmsExplicitExperiment(t *testing.T) {
	client := NewHTTPClientWithResolver(
		func(_ context.Context) (string, error) { return "http://localhost:12345", nil },
		httpx.DoerFunc(func(req *http.Request) (*http.Response, error) {
			var rr readRequest
			if err := json.NewDecoder(req.Body).Decode(&rr); err != nil {
				t.Fatalf("decode request: %v", err)
			}
			if rr.ExperimentID != "exp-1" {
				t.Errorf("expected experimentId exp-1, got %q", rr.ExperimentID)
			}
			if rr.VariantPolicy != "" {
				t.Errorf("expected no variantPolicy when armed, got %q", rr.VariantPolicy)
			}
			resp := readResponse{
				Combined:          "variant content",
				CombinedHash:      "sha256:def",
				SelectedVariantID: "variant-a",
				ExperimentID:      "exp-1",
				Skills: []struct {
					ID          string `json:"id"`
					Revision    int    `json:"revision,omitempty"`
					ContentHash string `json:"contentHash,omitempty"`
				}{{ID: "test-skill", Revision: 4}},
			}
			body, _ := json.Marshal(resp)
			return httpx.Response(http.StatusOK, string(body)), nil
		}),
	)

	snap, err := client.ReadSkillSource(context.Background(), "test-skill", "exp-1", nil, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if snap.VariantID != "variant-a" || snap.ExperimentID != "exp-1" {
		t.Errorf("expected armed variant-a/exp-1, got %q/%q", snap.VariantID, snap.ExperimentID)
	}
}

func TestHTTPClient_RecordExperimentOutcome(t *testing.T) {
	client := NewHTTPClientWithResolver(
		func(_ context.Context) (string, error) { return "http://localhost:12345", nil },
		httpx.DoerFunc(func(req *http.Request) (*http.Response, error) {
			if req.Method != http.MethodPost {
				t.Errorf("expected POST, got %s", req.Method)
			}
			if req.URL.Path != "/api/v1/experiments/exp-1/outcomes" {
				t.Errorf("unexpected path %s", req.URL.Path)
			}
			var outcome ExperimentOutcome
			if err := json.NewDecoder(req.Body).Decode(&outcome); err != nil {
				t.Fatalf("decode outcome: %v", err)
			}
			if outcome.VariantID != "variant-a" || outcome.Source != "agent-manager" {
				t.Errorf("unexpected outcome %+v", outcome)
			}
			return httpx.Response(http.StatusNoContent, ""), nil
		}),
	)

	err := client.RecordExperimentOutcome(context.Background(), "exp-1", ExperimentOutcome{
		VariantID: "variant-a",
		Source:    "agent-manager",
		Data:      json.RawMessage(`{"status":"complete"}`),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestHTTPClient_ReadSkill_ResolverError(t *testing.T) {
	client := NewHTTPClientWithResolver(
		func(_ context.Context) (string, error) {
			return "", context.DeadlineExceeded
		},
		nil,
	)

	_, err := client.ReadSkill(context.Background(), "test-skill", nil, false)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "resolve URL") {
		t.Errorf("expected resolve URL in error, got %v", err)
	}
}
