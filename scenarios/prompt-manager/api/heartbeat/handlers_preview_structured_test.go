package heartbeat

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"prompt-manager/store"
)

func TestPreviewPromptStructuredHandler(t *testing.T) {
	ctx := context.Background()
	storeDir := t.TempDir()
	fileStore := store.NewFileStore(storeDir)
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
	if err := agentStore.CreateFile(ctx, agent.ID, "NOTES.md", "Notes content", false); err != nil {
		t.Fatalf("create notes file: %v", err)
	}

	executor := NewExecutor(teamStore, agentStore, nil, "", nil, nil)
	handlers := NewHandlers(teamStore, agentStore, relationStore, nil, executor, nil, nil, nil)

	reqBody, _ := json.Marshal(PromptPreviewRequest{AgentID: agent.ID})
	req := httptest.NewRequest(http.MethodPost, "/prompt-preview-structured", bytes.NewReader(reqBody))
	w := httptest.NewRecorder()

	handlers.PreviewPromptStructured(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp StructuredPromptPreviewResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if len(resp.Sections) == 0 {
		t.Fatalf("expected non-empty sections")
	}
	if resp.Sections[0].Kind != "agent-file" {
		t.Fatalf("expected first section kind to be agent-file, got %q", resp.Sections[0].Kind)
	}
	if resp.AgentID != agent.ID {
		t.Fatalf("expected agentId %q, got %q", agent.ID, resp.AgentID)
	}
}
