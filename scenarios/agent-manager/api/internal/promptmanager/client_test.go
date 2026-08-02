package promptmanager

import (
	"context"
	"encoding/base64"
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

func TestHTTPClient_PublishFrictionUsesCanonicalIntakeAndAttribution(t *testing.T) {
	client := NewHTTPClientWithResolver(
		func(_ context.Context) (string, error) { return "http://localhost:12345", nil },
		httpx.DoerFunc(func(req *http.Request) (*http.Response, error) {
			if req.Method != http.MethodPost || req.URL.Path != "/api/v1/teams/meta-optimization/knowledge" {
				t.Fatalf("request = %s %s", req.Method, req.URL.Path)
			}
			var body struct {
				Topic   string `json:"topic"`
				Content string `json:"content"`
			}
			if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			if want := "friction-inbox/toolchain/agent-manager-finding-abcdef012345"; body.Topic != want {
				t.Fatalf("topic = %q, want %q", body.Topic, want)
			}
			if !strings.Contains(body.Content, "honesty_flags: [auto-generated]") {
				t.Fatalf("content lacks required honesty flag: %q", body.Content)
			}
			raw, err := base64.StdEncoding.DecodeString(req.Header.Get("X-Vrooli-Attribution"))
			if err != nil {
				t.Fatal(err)
			}
			var attribution map[string]string
			if err := json.Unmarshal(raw, &attribution); err != nil {
				t.Fatal(err)
			}
			if attribution["kind"] != "investigation" || attribution["run_id"] != "run-123" || attribution["spawn_origin"] != "investigation" {
				t.Fatalf("attribution = %#v", attribution)
			}
			return httpx.Response(http.StatusCreated, `{}`), nil
		}),
	)
	topic, err := client.PublishFriction(context.Background(), FrictionReport{InvestigationRunID: "run-123", Fingerprint: "abcdef0123456789", Category: "Tooling", Recommendation: "Fix the command", Evidence: "The command rejected valid input."})
	if err != nil {
		t.Fatal(err)
	}
	if topic != "friction-inbox/toolchain/agent-manager-finding-abcdef012345" {
		t.Fatalf("topic = %q", topic)
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

func TestHTTPClientAdminOperationsAndExperimentAssignment(t *testing.T) {
	client := NewHTTPClientWithResolver(
		func(_ context.Context) (string, error) { return "http://prompt-manager", nil },
		httpx.DoerFunc(func(req *http.Request) (*http.Response, error) {
			switch req.URL.Path {
			case "/api/v1/skills":
				if req.Method != http.MethodGet || req.URL.Query().Get("tag") != "ops tag" {
					t.Fatalf("list request=%s", req.URL.String())
				}
				return httpx.Response(http.StatusOK, `[{"id":"skill/a","name":"A"}]`), nil
			case "/api/v1/skills/skill/a":
				if req.Method == http.MethodGet {
					return httpx.Response(http.StatusOK, `{"id":"skill/a","content":"old"}`), nil
				}
				if req.Method == http.MethodPut {
					return httpx.Response(http.StatusOK, `{"id":"skill/a","content":"new"}`), nil
				}
			case "/api/v1/skills/skill/a/versions":
				return httpx.Response(http.StatusOK, `{"skillId":"skill/a","current":2,"versions":[{"version":2,"content":"new"}]}`), nil
			case "/api/v1/skills/skill/a/revert/1":
				return httpx.Response(http.StatusOK, ``), nil
			case "/api/v1/experiments/exp/1/assignments":
				return httpx.Response(http.StatusOK, `{"experimentId":"exp/1","skillId":"skill/a","variantId":"variant","content":"prompt","contentHash":"sha256:1"}`), nil
			}
			t.Fatalf("unexpected request %s %s", req.Method, req.URL.Path)
			return nil, nil
		}),
	)
	ctx := context.Background()
	if skills, err := client.ListSkills(ctx, "ops tag"); err != nil || len(skills) != 1 {
		t.Fatalf("skills=%+v err=%v", skills, err)
	}
	if skill, err := client.GetSkill(ctx, "skill/a"); err != nil || skill.Content != "old" {
		t.Fatalf("skill=%+v err=%v", skill, err)
	}
	content := "new"
	if skill, err := client.UpdateSkill(ctx, "skill/a", PromptSkillUpdate{Content: &content}); err != nil || skill.Content != "new" {
		t.Fatalf("updated=%+v err=%v", skill, err)
	}
	if versions, err := client.GetSkillVersions(ctx, "skill/a"); err != nil || versions.Current != 2 {
		t.Fatalf("versions=%+v err=%v", versions, err)
	}
	if err := client.RevertSkillVersion(ctx, "skill/a", 1); err != nil {
		t.Fatal(err)
	}
	assignment := AssignmentRequest{ExperimentID: "exp/1", SkillID: "skill/a", ExecutionID: "execution", NodeID: "node", AttemptKey: "attempt", IdempotencyKey: "key"}
	if snap, err := client.AssignExperimentPrompt(ctx, assignment); err != nil || snap.VariantID != "variant" || snap.Content != "prompt" {
		t.Fatalf("assignment=%+v err=%v", snap, err)
	}
	if _, err := client.AssignExperimentPrompt(ctx, AssignmentRequest{}); err == nil {
		t.Fatal("incomplete assignment accepted")
	}
}
