package heartbeat

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"prompt-manager/internal/paths"
	"prompt-manager/internal/store"

	"github.com/gorilla/mux"
)

func TestCloseTeamCorpusHandlerHidesEntryFromDefaultInboxListing(t *testing.T) {
	roots := paths.RootsForTest(t)
	fileStore := newFileStore(t, roots)
	teamStore := fileStore.Teams().(*store.FileTeamStore)
	agentStore := fileStore.Agents().(*store.FileAgentStore)
	h := NewHandlers(HandlersDeps{
		TeamStore:     teamStore,
		AgentStore:    agentStore,
		RelationStore: fileStore.Relations(),
		Executor:      newTestExecutor(t, teamStore, agentStore, nil, "", nil, nil),
	})
	if err := teamStore.Create(context.Background(), newIndependentTestTeam(scenarioQATeamID, "Scenario QA")); err != nil {
		t.Fatal(err)
	}
	for _, entry := range []store.KnowledgeEntry{
		{ID: "knw-a", Topic: "bug-inbox/code-defect/a", Content: "a"},
		{ID: "knw-b", Topic: "bug-inbox/code-defect/b", Content: "b"},
	} {
		entry := entry
		if err := teamStore.AppendTeamCorpus(context.Background(), scenarioQATeamID, &entry); err != nil {
			t.Fatal(err)
		}
	}
	header := encodeAttribution(t, store.AttributionInfo{Kind: store.KnowledgeKindOperatorDirect, SpawnOrigin: store.SpawnOriginOperatorCLI})

	closeEntry := func(id string, body CloseTeamCorpusRequest, withHeader bool) *httptest.ResponseRecorder {
		t.Helper()
		raw, _ := json.Marshal(body)
		req := httptest.NewRequest(http.MethodPost, "/teams/"+scenarioQATeamID+"/knowledge/"+id+"/close", bytes.NewReader(raw))
		if withHeader {
			req.Header.Set(attributionHeaderName, header)
		}
		req = mux.SetURLVars(req, map[string]string{"id": scenarioQATeamID, "knowledgeId": id})
		w := httptest.NewRecorder()
		h.CloseTeamCorpusHandler(w, req)
		return w
	}
	list := func(query string) KnowledgeListResponse {
		t.Helper()
		req := httptest.NewRequest(http.MethodGet, "/teams/"+scenarioQATeamID+"/knowledge?topic_prefix=bug-inbox/&"+query, nil)
		req = mux.SetURLVars(req, map[string]string{"id": scenarioQATeamID})
		w := httptest.NewRecorder()
		h.ListTeamCorpus(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("list = %d: %s", w.Code, w.Body.String())
		}
		var resp KnowledgeListResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatal(err)
		}
		return resp
	}

	if w := closeEntry("knw-a", CloseTeamCorpusRequest{Reason: "fixed"}, false); w.Code != http.StatusBadRequest {
		t.Fatalf("close without attribution = %d, want 400", w.Code)
	}
	if w := closeEntry("knw-a", CloseTeamCorpusRequest{Reason: "routed-to"}, true); w.Code != http.StatusBadRequest {
		t.Fatalf("routed-to without ref = %d, want 400", w.Code)
	}
	if w := closeEntry("knw-missing", CloseTeamCorpusRequest{Reason: "stale"}, true); w.Code != http.StatusNotFound {
		t.Fatalf("close missing = %d, want 404", w.Code)
	}
	w := closeEntry("knw-a", CloseTeamCorpusRequest{Reason: "duplicate-of", Ref: "knw-b", Note: "same root cause"}, true)
	if w.Code != http.StatusOK {
		t.Fatalf("close = %d: %s", w.Code, w.Body.String())
	}
	var closed store.KnowledgeEntry
	if err := json.Unmarshal(w.Body.Bytes(), &closed); err != nil {
		t.Fatal(err)
	}
	if closed.Closure == nil || closed.Closure.Reason != "duplicate-of" || closed.Closure.Ref != "knw-b" || closed.Closure.Actor == "" {
		t.Fatalf("closure = %+v", closed.Closure)
	}

	open := list("last=1")
	if open.Total != 1 || len(open.Entries) != 1 || open.Entries[0].ID != "knw-b" {
		t.Fatalf("default inbox listing = %+v, want only knw-b", open)
	}
	all := list("include_closed=true")
	if all.Total != 2 || len(all.Entries) != 2 {
		t.Fatalf("include_closed listing = %+v, want both entries", all)
	}
}
