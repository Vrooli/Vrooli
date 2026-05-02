package memberflow

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gorilla/mux"
)

func newRouter(h *Handlers) *mux.Router {
	r := mux.NewRouter()
	r.HandleFunc("/teams/{id}/members/{agentId}/topics", h.GetMember).Methods("GET")
	r.HandleFunc("/teams/{id}/members/{agentId}/topics", h.PutMember).Methods("PUT")
	r.HandleFunc("/teams/{id}/topics", h.GetTeam).Methods("GET")
	r.HandleFunc("/topics/graph", h.GetGraph).Methods("GET")
	r.HandleFunc("/topics/drain-status", h.GetDrainStatus).Methods("GET")
	return r
}

func newStore(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "teams"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	return root
}

func TestGetMember_NotPresent(t *testing.T) {
	store := newStore(t)
	r := newRouter(NewHandlers(store))
	req := httptest.NewRequest("GET", "/teams/marketing-crew/members/researcher/topics", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	var resp MemberTopicsResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Exists {
		t.Errorf("expected Exists=false")
	}
	if !resp.Topics.IsEmpty() {
		t.Errorf("expected empty Topics for non-existent file")
	}
}

func TestPutMember_RoundTrip(t *testing.T) {
	store := newStore(t)
	h := NewHandlers(store)
	r := newRouter(h)

	body := bytes.NewBufferString(`{
		"intake": [{"prefix": "research-inbox/*", "drained_by_skill": "marketing-research-router"}],
		"output": [{"prefix": "audience-scan/*", "destination_kind": "knowledge"}],
		"raises_capability_gaps": true
	}`)
	req := httptest.NewRequest("PUT", "/teams/marketing-crew/members/researcher/topics", body)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("PUT status = %d, body=%s", w.Code, w.Body.String())
	}

	// Read back
	req2 := httptest.NewRequest("GET", "/teams/marketing-crew/members/researcher/topics", nil)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK {
		t.Fatalf("GET status = %d", w2.Code)
	}
	var resp MemberTopicsResponse
	if err := json.NewDecoder(w2.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !resp.Exists || !resp.Topics.RaisesCapabilityGaps {
		t.Errorf("round-trip lost data: %+v", resp)
	}
	if len(resp.Topics.Intake) != 1 || resp.Topics.Intake[0].DrainedBySkill != "marketing-research-router" {
		t.Errorf("intake mismatch: %+v", resp.Topics.Intake)
	}
}

func TestPutMember_RejectsMalformed(t *testing.T) {
	store := newStore(t)
	r := newRouter(NewHandlers(store))
	body := bytes.NewBufferString(`{ not json `)
	req := httptest.NewRequest("PUT", "/teams/t/members/m/topics", body)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for malformed JSON, got %d", w.Code)
	}
}

func TestPutMember_RejectsSchemaViolation(t *testing.T) {
	store := newStore(t)
	r := newRouter(NewHandlers(store))
	body := bytes.NewBufferString(`{"intake":[{"prefix":"*","drained_by_skill":"x"}]}`)
	req := httptest.NewRequest("PUT", "/teams/t/members/m/topics", body)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for bare-* prefix, got %d", w.Code)
	}
}

func TestGetTeam_AggregatesMembers(t *testing.T) {
	store := newStore(t)
	h := NewHandlers(store)

	if err := WriteMember(store, "marketing-crew", "researcher", Topics{
		Intake: []IntakeEntry{{Prefix: "research-inbox/*", DrainedBySkill: "marketing-research-router"}},
	}); err != nil {
		t.Fatalf("WriteMember: %v", err)
	}
	if err := WriteMember(store, "marketing-crew", "publisher", Topics{}); err != nil {
		t.Fatalf("WriteMember: %v", err)
	}

	r := newRouter(h)
	req := httptest.NewRequest("GET", "/teams/marketing-crew/topics", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d", w.Code)
	}
	var resp TeamTopicsResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Team != "marketing-crew" {
		t.Errorf("team mismatch")
	}
	if len(resp.Members) != 2 {
		t.Errorf("expected 2 members, got %d", len(resp.Members))
	}
}

func TestGetGraph_BuildsExpectedNodes(t *testing.T) {
	store := newStore(t)
	if err := WriteMember(store, "marketing-crew", "researcher", Topics{
		Intake: []IntakeEntry{
			{Prefix: "research-inbox/*", DrainedBySkill: "marketing-research-router"},
		},
		Output: []OutputEntry{
			{Prefix: "audience-scan/*", DestinationKind: DestinationKnowledge},
		},
		ExternalProducers:    []string{"vision-walk", "operator"},
		DecisionsOwned:       []string{"audience-update"},
		RaisesCapabilityGaps: true,
	}); err != nil {
		t.Fatalf("WriteMember: %v", err)
	}

	r := newRouter(NewHandlers(store))
	req := httptest.NewRequest("GET", "/topics/graph", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}

	var resp GraphResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}

	// Member node + 2 external + 1 input prefix + 1 output prefix + 1 decision + 1 cap-gap = 7 nodes
	wantNodes := map[string]string{
		"member:marketing-crew/researcher": "member",
		"external:vision-walk":             "external",
		"external:operator":                "external",
		"prefix:research-inbox/*":          "knowledge_sink",
		"prefix:audience-scan/*":           "knowledge_sink",
		"decision:audience-update":         "decision",
		"capability-gap":                   "capability_gap",
	}
	if got := len(resp.Nodes); got != len(wantNodes) {
		t.Errorf("node count = %d, want %d (nodes=%+v)", got, len(wantNodes), resp.Nodes)
	}
	for _, n := range resp.Nodes {
		want, ok := wantNodes[n.ID]
		if !ok {
			t.Errorf("unexpected node %q (kind=%s)", n.ID, n.Kind)
			continue
		}
		if n.Kind != want {
			t.Errorf("node %q has kind=%q, want %q", n.ID, n.Kind, want)
		}
	}
}

func TestGetGraph_EmptyStore(t *testing.T) {
	store := newStore(t)
	r := newRouter(NewHandlers(store))
	req := httptest.NewRequest("GET", "/topics/graph", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d", w.Code)
	}
	var resp GraphResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Nodes) != 0 || len(resp.Edges) != 0 {
		t.Errorf("expected empty graph, got %+v", resp)
	}
}

func TestGetDrainStatus(t *testing.T) {
	store := newStore(t)
	r := newRouter(NewHandlers(store))
	req := httptest.NewRequest("GET", "/topics/drain-status", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("status=%d", w.Code)
	}
}
