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
	lastSkillID   string
	lastVariables map[string]string
	lastWithScope bool
}

func setupPromptHandler() *Handler {
	h, _ := setupPromptHandlerWithMock()
	return h
}

func setupPromptHandlerWithMock() (*Handler, *mockClient) {
	client := &mockClient{
		MockClient: promptmanager.MockClient{
			Skills: []promptmanager.PromptSkill{
				{
					ID:          "swarm-manager-workshop",
					Name:        "Clarify",
					Description: "Clarify prompts",
					Draft:       false,
					UpdatedAt:   "2026-02-16T00:00:00Z",
				},
			},
			Skill: promptmanager.PromptSkill{
				ID:          "swarm-manager-workshop",
				Name:        "Clarify",
				Description: "Clarify prompts",
				Content:     "Use {{ITEM_TITLE}} in {{ITEM_FOLDER}}",
				Draft:       false,
				UpdatedAt:   "2026-02-16T00:00:00Z",
			},
			Versions: promptmanager.PromptSkillVersions{
				SkillID: "swarm-manager-workshop",
				Current: 2,
				Versions: []promptmanager.PromptSkillVersion{
					{Version: 1, Content: "old", Name: "Clarify", UpdatedAt: "2026-02-15T00:00:00Z"},
				},
			},
			Result: "rendered prompt",
		},
	}
	return NewHandler("", client), client
}

func TestMap_ReturnsBindings(t *testing.T) {
	h := setupPromptHandler()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/prompts/map", nil)
	w := httptest.NewRecorder()

	h.Map(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
	if !bytes.Contains(w.Body.Bytes(), []byte(`"skill_id":"swarm-manager-workshop"`)) {
		t.Fatalf("expected workshop binding, got %s", w.Body.String())
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
	if !bytes.Contains(w.Body.Bytes(), []byte(`"id":"swarm-manager-workshop"`)) {
		t.Fatalf("expected swarm-manager skill in response, got %s", w.Body.String())
	}
}

func TestPreview_RendersPrompt(t *testing.T) {
	h, client := setupPromptHandlerWithMock()
	body := map[string]any{
		"skill_id": "swarm-manager-workshop",
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
	if !bytes.Contains(w.Body.Bytes(), []byte(`"with_scope":false`)) {
		t.Fatalf("expected with_scope false by default, got %s", w.Body.String())
	}
	if client.lastWithScope {
		t.Fatalf("expected preview ReadSkill with scope disabled")
	}
}

func TestSimulate_DisablesScopeByDefault(t *testing.T) {
	h, client := setupPromptHandlerWithMock()
	body := map[string]any{
		"kind": "idea",
		"mode": "workshop",
	}
	payload, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/prompts/simulate", bytes.NewReader(payload))
	w := httptest.NewRecorder()

	h.Simulate(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
	if client.lastSkillID != "swarm-manager-workshop" {
		t.Fatalf("expected workshop skill, got %q", client.lastSkillID)
	}
	if client.lastWithScope {
		t.Fatalf("expected simulate ReadSkill with scope disabled")
	}
}

func TestUpdateSkill_RejectsMissingRequiredVariables(t *testing.T) {
	h := setupPromptHandler()
	body := map[string]any{
		"content": "No variables here",
	}
	payload, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/prompts/skills/swarm-manager-workshop", bytes.NewReader(payload))
	req = mux.SetURLVars(req, map[string]string{"id": "swarm-manager-workshop"})
	w := httptest.NewRecorder()

	h.UpdateSkill(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
}

func (m *mockClient) ReadSkill(_ context.Context, skillID string, variables map[string]string, withScope bool) (string, error) {
	m.lastSkillID = skillID
	m.lastVariables = variables
	m.lastWithScope = withScope
	return m.Result, m.Err
}
