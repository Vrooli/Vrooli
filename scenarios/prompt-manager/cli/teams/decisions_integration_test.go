//go:build integration

// Integration test for the decision-* CLI subcommands. Stands up an
// httptest.Server with a small in-memory implementation of the relevant
// routes and exercises the real HTTP wire path through a minimal
// appctx.Context implementation.
//
// This test is build-tagged because it touches more surface than the
// in-process unit tests and is intended to run in CI alongside the
// scenario-level test target. Run locally with:
//
//	go test -tags=integration ./teams/...
package teams

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
)

// httpContext is a minimal appctx.Context implementation that issues real
// HTTP requests against a base URL — used to drive integration tests
// without pulling in the full cliapp.ScenarioApp.
type httpContext struct {
	base   string
	client *http.Client
}

func newHTTPContext(base string) *httpContext {
	return &httpContext{base: strings.TrimRight(base, "/"), client: http.DefaultClient}
}

func (c *httpContext) do(method, path string, payload interface{}, result interface{}) error {
	var body io.Reader
	if payload != nil {
		raw, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		body = strings.NewReader(string(raw))
	}
	req, err := http.NewRequest(method, c.base+path, body)
	if err != nil {
		return err
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		raw, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("http %d: %s", resp.StatusCode, string(raw))
	}
	if result == nil {
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(result)
}

func (c *httpContext) Get(path string, result interface{}) error {
	return c.do(http.MethodGet, path, nil, result)
}

func (c *httpContext) GetWithQuery(path string, query url.Values, result interface{}) error {
	if query != nil {
		path = path + "?" + query.Encode()
	}
	return c.Get(path, result)
}

func (c *httpContext) Post(path string, payload, result interface{}) error {
	return c.do(http.MethodPost, path, payload, result)
}

func (c *httpContext) Put(path string, payload, result interface{}) error {
	return c.do(http.MethodPut, path, payload, result)
}

func (c *httpContext) Delete(path string) error {
	return c.do(http.MethodDelete, path, nil, nil)
}

// decisionStore is a tiny in-memory mock of the API's decision store.
type decisionStore struct {
	mu        sync.Mutex
	decisions map[string]map[string]*DecisionEntry // teamID -> decisionID -> entry
}

func newDecisionStore() *decisionStore {
	return &decisionStore{decisions: map[string]map[string]*DecisionEntry{}}
}

func (s *decisionStore) seed(teamID string, entry DecisionEntry) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.decisions[teamID]; !ok {
		s.decisions[teamID] = map[string]*DecisionEntry{}
	}
	e := entry
	s.decisions[teamID][entry.ID] = &e
}

func (s *decisionStore) get(teamID, decisionID string) *DecisionEntry {
	s.mu.Lock()
	defer s.mu.Unlock()
	if t, ok := s.decisions[teamID]; ok {
		if e, ok := t[decisionID]; ok {
			cp := *e
			return &cp
		}
	}
	return nil
}

// newDecisionAPIServer returns an httptest.Server that implements the subset
// of routes the decision-* CLI subcommands talk to, backed by the given store.
func newDecisionAPIServer(store *decisionStore) *httptest.Server {
	mux := http.NewServeMux()

	mux.HandleFunc("/teams/", func(w http.ResponseWriter, r *http.Request) {
		// Path shapes:
		//   /teams/{teamID}/decisions
		//   /teams/{teamID}/decisions/{decisionID}
		parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
		if len(parts) < 3 || parts[0] != "teams" || parts[2] != "decisions" {
			http.NotFound(w, r)
			return
		}
		teamID := parts[1]

		switch len(parts) {
		case 3: // collection: GET (list) only used here
			if r.Method != http.MethodGet {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			store.mu.Lock()
			defer store.mu.Unlock()
			entries := []DecisionEntry{}
			if t, ok := store.decisions[teamID]; ok {
				for _, e := range t {
					entries = append(entries, *e)
				}
			}
			_ = json.NewEncoder(w).Encode(DecisionListResponse{
				TeamID:  teamID,
				Entries: entries,
				Total:   len(entries),
				Last:    len(entries),
			})
		case 4: // item: PUT/PATCH/DELETE
			decisionID := parts[3]
			switch r.Method {
			case http.MethodPut, http.MethodPatch:
				var body map[string]interface{}
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					http.Error(w, "bad json", http.StatusBadRequest)
					return
				}
				store.mu.Lock()
				entry := store.decisions[teamID][decisionID]
				if entry == nil {
					store.mu.Unlock()
					http.Error(w, "not found", http.StatusNotFound)
					return
				}
				if v, ok := body["status"].(string); ok {
					entry.Status = v
				}
				if v, ok := body["selected"].(string); ok {
					entry.Selected = v
				}
				if v, ok := body["notes"].(string); ok {
					entry.Notes = v
				}
				if v, ok := body["freeform"].(string); ok {
					entry.Freeform = v
				}
				resp := *entry
				store.mu.Unlock()
				_ = json.NewEncoder(w).Encode(resp)
			case http.MethodDelete:
				store.mu.Lock()
				if t, ok := store.decisions[teamID]; ok {
					delete(t, decisionID)
				}
				store.mu.Unlock()
				w.WriteHeader(http.StatusNoContent)
			default:
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			}
		default:
			http.NotFound(w, r)
		}
	})

	return httptest.NewServer(mux)
}

func TestIntegration_DecisionAcceptTransition(t *testing.T) {
	store := newDecisionStore()
	store.seed("team-1", DecisionEntry{
		ID:     "dec-1",
		By:     "agent",
		Topic:  "Pick a database",
		Status: "pending",
		Options: []DecisionOption{
			{Key: "A", Label: "postgres"},
			{Key: "B", Label: "mysql"},
		},
	})
	srv := newDecisionAPIServer(store)
	defer srv.Close()

	ctx := newHTTPContext(srv.URL)

	// Pre-condition: pending, no selection.
	pre := store.get("team-1", "dec-1")
	if pre.Status != "pending" || pre.Selected != "" {
		t.Fatalf("setup wrong: %+v", pre)
	}

	if err := cmdDecisionAccept(ctx, []string{
		"team-1", "dec-1",
		"--selected=A",
		"--notes=postgres has the operator team's experience",
	}); err != nil {
		t.Fatalf("cmdDecisionAccept error = %v", err)
	}

	post := store.get("team-1", "dec-1")
	if post == nil {
		t.Fatal("decision disappeared after accept")
	}
	if post.Status != "accepted" {
		t.Errorf("status = %q, want accepted", post.Status)
	}
	if post.Selected != "A" {
		t.Errorf("selected = %q, want A", post.Selected)
	}
	if !strings.Contains(post.Notes, "postgres") {
		t.Errorf("notes = %q, want to contain 'postgres'", post.Notes)
	}
}

func TestIntegration_DecisionRejectTransition(t *testing.T) {
	store := newDecisionStore()
	store.seed("team-1", DecisionEntry{ID: "dec-2", Status: "pending"})
	srv := newDecisionAPIServer(store)
	defer srv.Close()

	ctx := newHTTPContext(srv.URL)
	if err := cmdDecisionReject(ctx, []string{"team-1", "dec-2", "--notes=duplicate of dec-1"}); err != nil {
		t.Fatalf("cmdDecisionReject error = %v", err)
	}
	got := store.get("team-1", "dec-2")
	if got.Status != "rejected" {
		t.Errorf("status = %q, want rejected", got.Status)
	}
	if !strings.Contains(got.Notes, "duplicate") {
		t.Errorf("notes = %q, want to contain 'duplicate'", got.Notes)
	}
}

func TestIntegration_DecisionUpdateMultiField(t *testing.T) {
	store := newDecisionStore()
	store.seed("team-1", DecisionEntry{ID: "dec-3", Status: "pending"})
	srv := newDecisionAPIServer(store)
	defer srv.Close()

	ctx := newHTTPContext(srv.URL)
	if err := cmdDecisionUpdate(ctx, []string{
		"team-1", "dec-3",
		"--status=accepted",
		"--selected=__other__",
		"--freeform=write-in answer",
		"--notes=context for future-self",
	}); err != nil {
		t.Fatalf("cmdDecisionUpdate error = %v", err)
	}
	got := store.get("team-1", "dec-3")
	if got.Status != "accepted" || got.Selected != "__other__" || got.Freeform != "write-in answer" {
		t.Errorf("decision = %+v, want accepted/__other__/write-in answer", got)
	}
}

func TestIntegration_DecisionDeleteRemovesEntry(t *testing.T) {
	store := newDecisionStore()
	store.seed("team-1", DecisionEntry{ID: "dec-4", Status: "pending"})
	srv := newDecisionAPIServer(store)
	defer srv.Close()

	ctx := newHTTPContext(srv.URL)
	if err := cmdDecisionDelete(ctx, []string{"team-1", "dec-4", "--yes"}); err != nil {
		t.Fatalf("cmdDecisionDelete error = %v", err)
	}
	if got := store.get("team-1", "dec-4"); got != nil {
		t.Errorf("decision still present after delete: %+v", got)
	}
}

func TestIntegration_DecisionShowFindsEntry(t *testing.T) {
	store := newDecisionStore()
	store.seed("team-1", DecisionEntry{ID: "dec-5", By: "agent", Decision: "ship", Rationale: "ready"})
	srv := newDecisionAPIServer(store)
	defer srv.Close()

	ctx := newHTTPContext(srv.URL)
	if err := cmdDecisionShow(ctx, []string{"team-1", "dec-5", "--json"}); err != nil {
		t.Fatalf("cmdDecisionShow error = %v", err)
	}
}
