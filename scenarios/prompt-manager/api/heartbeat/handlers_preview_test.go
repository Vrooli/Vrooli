package heartbeat

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"prompt-manager/internal/paths"
	"prompt-manager/store"
	"strings"
	"testing"
)

func TestPreviewPromptHandler(t *testing.T) {
	ctx := context.Background()
	roots := paths.RootsForTest(t)
	fileStore := store.NewFileStore(roots)
	agentStore := fileStore.Agents().(*store.FileAgentStore)
	teamStore := fileStore.Teams().(*store.FileTeamStore)
	relationStore := fileStore.Relations()

	agent := &store.Agent{
		ID:          "agent-1",
		DisplayName: "Agent One",
		Status:      store.AgentStatusActive,
		FileOrder:   []string{"SOUL.md"},
	}

	if err := agentStore.Create(ctx, agent); err != nil {
		t.Fatalf("create agent: %v", err)
	}

	executor := NewExecutor(teamStore, agentStore, nil, "", nil, nil)
	handlers := NewHandlers(teamStore, agentStore, relationStore, nil, executor, nil, nil, nil)

	reqBody, _ := json.Marshal(PromptPreviewRequest{AgentID: agent.ID})
	req := httptest.NewRequest(http.MethodPost, "/prompt-preview", bytes.NewReader(reqBody))
	w := httptest.NewRecorder()

	handlers.PreviewPrompt(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var resp PromptPreviewResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if !strings.Contains(resp.Prompt, "# Agent Files (Markdown)") {
		t.Fatalf("expected prompt preview to include agent files section")
	}
}
