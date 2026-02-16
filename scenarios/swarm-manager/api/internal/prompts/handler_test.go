package prompts

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"

	"swarm-manager/internal/promptmanager"
)

type mockClient struct {
	promptmanager.MockClient
}

func setupPromptHandler() *Handler {
	return NewHandler("", &mockClient{
		MockClient: promptmanager.MockClient{
			Skills: []promptmanager.PromptSkill{
				{
					ID:          "swarm-manager-clarify-idea",
					Name:        "Clarify",
					Description: "Clarify prompts",
					Draft:       false,
					UpdatedAt:   "2026-02-16T00:00:00Z",
				},
			},
			Skill: promptmanager.PromptSkill{
				ID:          "swarm-manager-clarify-idea",
				Name:        "Clarify",
				Description: "Clarify prompts",
				Content:     "Use {{ITEM_TITLE}} in {{ITEM_FOLDER}}",
				Draft:       false,
				UpdatedAt:   "2026-02-16T00:00:00Z",
			},
			Versions: promptmanager.PromptSkillVersions{
				SkillID: "swarm-manager-clarify-idea",
				Current: 2,
				Versions: []promptmanager.PromptSkillVersion{
					{Version: 1, Content: "old", Name: "Clarify", UpdatedAt: "2026-02-15T00:00:00Z"},
				},
			},
			Result: "rendered prompt",
		},
	})
}

func TestMap_ReturnsBindings(t *testing.T) {
	h := setupPromptHandler()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/prompts/map", nil)
	w := httptest.NewRecorder()

	h.Map(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
	if !bytes.Contains(w.Body.Bytes(), []byte(`"skill_id":"swarm-manager-clarify-idea"`)) {
		t.Fatalf("expected clarify binding, got %s", w.Body.String())
	}
}

func TestListSkills_OnlySwarmManagerIDs(t *testing.T) {
	h := setupPromptHandler()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/prompts/skills", nil)
	w := httptest.NewRecorder()

	h.ListSkills(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
	if !bytes.Contains(w.Body.Bytes(), []byte(`"id":"swarm-manager-clarify-idea"`)) {
		t.Fatalf("expected swarm-manager skill in response, got %s", w.Body.String())
	}
}

func TestPreview_RendersPrompt(t *testing.T) {
	h := setupPromptHandler()
	body := map[string]any{
		"skill_id": "swarm-manager-clarify-idea",
		"variables": map[string]string{
			"ITEM_TITLE": "My Idea",
		},
	}
	payload, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/prompts/preview", bytes.NewReader(payload))
	w := httptest.NewRecorder()

	h.Preview(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
	if !bytes.Contains(w.Body.Bytes(), []byte(`"prompt":"rendered prompt"`)) {
		t.Fatalf("expected rendered prompt response, got %s", w.Body.String())
	}
}

func TestUpdateSkill_RejectsMissingRequiredVariables(t *testing.T) {
	h := setupPromptHandler()
	body := map[string]any{
		"content": "No variables here",
	}
	payload, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/prompts/skills/swarm-manager-clarify-idea", bytes.NewReader(payload))
	req = mux.SetURLVars(req, map[string]string{"id": "swarm-manager-clarify-idea"})
	w := httptest.NewRecorder()

	h.UpdateSkill(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
}

func (m *mockClient) ReadSkill(_ context.Context, _ string, _ map[string]string, _ bool) (string, error) {
	return m.Result, m.Err
}
