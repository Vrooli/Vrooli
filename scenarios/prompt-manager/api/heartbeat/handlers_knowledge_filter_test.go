package heartbeat

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"prompt-manager/internal/paths"
	"prompt-manager/store"

	"github.com/gorilla/mux"
)

func setupKnowledgeFilterTest(t *testing.T) (*Handlers, *store.FileTeamStore) {
	t.Helper()
	roots := paths.RootsForTest(t)
	fileStore := newFileStore(t, roots)
	teamStore := fileStore.Teams().(*store.FileTeamStore)
	agentStore := fileStore.Agents().(*store.FileAgentStore)
	relationStore := fileStore.Relations()
	executor := newTestExecutor(t, teamStore, agentStore, nil, "", nil, nil)
	handlers := NewHandlers(teamStore, agentStore, relationStore, nil, executor, nil, nil, nil)

	if err := teamStore.Create(context.Background(), newIndependentTestTeam("team-1", "Test Team")); err != nil {
		t.Fatalf("create team: %v", err)
	}

	ctx := context.Background()
	for _, e := range []store.KnowledgeEntry{
		{ID: "k1", At: "2026-05-01T00:00:00Z", Caller: "vw", Topic: "research-inbox/audience/foo", Content: "a"},
		{ID: "k2", At: "2026-05-01T00:00:01Z", Caller: "vw", Topic: "research-inbox/hook/bar", Content: "b"},
		{ID: "k3", At: "2026-05-01T00:00:02Z", Caller: "rs", Topic: "audience-scan/foo", Content: "c"},
	} {
		entry := e
		if err := teamStore.AppendKnowledge(ctx, "team-1", &entry); err != nil {
			t.Fatalf("AppendKnowledge: %v", err)
		}
	}
	return handlers, teamStore
}

func getKnowledgeViaHandler(t *testing.T, h *Handlers, query string) (int, KnowledgeListResponse, string) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/teams/team-1/knowledge"+query, nil)
	req = mux.SetURLVars(req, map[string]string{"id": "team-1"})
	w := httptest.NewRecorder()
	h.GetKnowledge(w, req)
	var resp KnowledgeListResponse
	body := w.Body.String()
	if w.Code == http.StatusOK {
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("decode: %v", err)
		}
	}
	return w.Code, resp, body
}

func TestGetKnowledge_TopicPrefixQueryParam(t *testing.T) {
	h, _ := setupKnowledgeFilterTest(t)

	code, resp, body := getKnowledgeViaHandler(t, h, "?topic_prefix=research-inbox/")
	if code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", code, body)
	}
	if len(resp.Entries) != 2 {
		t.Fatalf("want 2 entries, got %d: %+v", len(resp.Entries), resp.Entries)
	}
	for _, e := range resp.Entries {
		if e.Topic != "research-inbox/audience/foo" && e.Topic != "research-inbox/hook/bar" {
			t.Errorf("unexpected entry in prefix result: %+v", e)
		}
	}
}

func TestGetKnowledge_TopicPrefixNarrowsToSignalType(t *testing.T) {
	h, _ := setupKnowledgeFilterTest(t)

	code, resp, body := getKnowledgeViaHandler(t, h, "?topic_prefix=research-inbox/audience/")
	if code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", code, body)
	}
	if len(resp.Entries) != 1 || resp.Entries[0].ID != "k1" {
		t.Errorf("want only k1, got %+v", resp.Entries)
	}
}

func TestGetKnowledge_ExactTopicStillWorks(t *testing.T) {
	h, _ := setupKnowledgeFilterTest(t)

	code, resp, body := getKnowledgeViaHandler(t, h, "?topic=audience-scan/foo")
	if code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", code, body)
	}
	if len(resp.Entries) != 1 || resp.Entries[0].ID != "k3" {
		t.Errorf("want only k3, got %+v", resp.Entries)
	}
}

func TestGetKnowledge_TopicAndTopicPrefixMutuallyExclusive(t *testing.T) {
	h, _ := setupKnowledgeFilterTest(t)

	code, _, body := getKnowledgeViaHandler(t, h, "?topic=research-inbox&topic_prefix=research-inbox/")
	if code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d: %s", code, body)
	}
}
