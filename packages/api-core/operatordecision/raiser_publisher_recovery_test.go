package operatordecision

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/vrooli/api-core/eventbus"
)

// jsonPendingStore exercises the same PendingStore contract across a store
// reconstruction, rather than letting the test's in-memory map stand in for
// saved operator work.
type jsonPendingStore struct {
	mu   sync.Mutex
	path string
}

func (s *jsonPendingStore) rows() (map[string]*PendingAsk, error) {
	data, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return map[string]*PendingAsk{}, nil
	}
	if err != nil {
		return nil, err
	}
	rows := map[string]*PendingAsk{}
	if err := json.Unmarshal(data, &rows); err != nil {
		return nil, err
	}
	return rows, nil
}

func (s *jsonPendingStore) save(rows map[string]*PendingAsk) error {
	data, err := json.Marshal(rows)
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0600)
}

func (s *jsonPendingStore) Get(_ context.Context, id string) (*PendingAsk, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rows, err := s.rows()
	if err != nil || rows[id] == nil {
		return rows[id], err
	}
	copy := *rows[id]
	return &copy, nil
}

func (s *jsonPendingStore) Put(_ context.Context, ask *PendingAsk) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	rows, err := s.rows()
	if err != nil {
		return err
	}
	copy := *ask
	rows[ask.ID] = &copy
	return s.save(rows)
}

func (s *jsonPendingStore) CountOutstandingForSubject(_ context.Context, subject string, now time.Time) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rows, err := s.rows()
	if err != nil {
		return 0, err
	}
	count := 0
	for _, row := range rows {
		if row.Subject == subject && row.Outstanding(now) {
			count++
		}
	}
	return count, nil
}

func TestRaiserDisabledPublisherPreservesAskAndRetriesAfterRecovery(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "pending.json")
	store := &jsonPendingStore{path: path}
	clock := fakeClock{t: time.Date(2026, 11, 21, 12, 0, 0, 0, time.UTC)}
	ask := PendingAsk{ID: "decision-1", Subject: "deployment-7", Extra: map[string]string{"revision": "rev-42", "operator_note": "retain this"}}
	facts := Facts{Question: "Apply revision?", Recommended: "reject", Options: []Option{{Key: "approve"}, {Key: "reject"}}}
	raiser := Raiser{Source: "deployment-manager", Kind: "rollout", Store: store, Publisher: eventbus.Client{}, Clock: clock}

	raised, err := raiser.Raise(ctx, ask, facts)
	if !raised || !errors.Is(err, ErrPublisherUnavailable) {
		t.Fatalf("disabled publisher: raised=%v err=%v", raised, err)
	}
	saved, err := store.Get(ctx, ask.ID)
	if err != nil || saved == nil || saved.Published || saved.EventID == "" || saved.Extra["revision"] != "rev-42" || saved.Extra["operator_note"] != "retain this" {
		t.Fatalf("disabled provider must retain the exact unnotified ask: ask=%#v err=%v", saved, err)
	}
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	var receivedBodies [][]byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		receivedBodies = append(receivedBodies, body)
		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	// Reconstruct the store as after a process restart, then retry through the
	// recovered local Events endpoint. The test does not use the real service.
	store = &jsonPendingStore{path: path}
	raiser.Store = store
	raiser.Publisher = eventbus.Client{BaseURL: server.URL}
	if raised, err := raiser.Raise(ctx, ask, facts); !raised || err != nil {
		t.Fatalf("recovered publisher retry: raised=%v err=%v", raised, err)
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	saved, err = store.Get(ctx, ask.ID)
	if err != nil || saved == nil || !saved.Published || saved.EventID == "" || saved.Extra["operator_note"] != "retain this" {
		t.Fatalf("recovered publication must retain and mark the same ask: ask=%#v err=%v", saved, err)
	}
	var envelope map[string]any
	if len(receivedBodies) == 1 {
		if err := json.Unmarshal(receivedBodies[0], &envelope); err != nil {
			t.Fatal(err)
		}
	}
	eventID, _ := envelope["eventId"].(string)
	if eventID == "" {
		eventID, _ = envelope["event_id"].(string)
	}
	if len(receivedBodies) != 1 || eventID != saved.EventID || !strings.Contains(string(receivedBodies[0]), "deployment-manager.rollout.decision_requested.v1") {
		t.Fatalf("expected one recovered decision event with saved id %q, got %s", saved.EventID, receivedBodies)
	}
	if string(before) == string(after) {
		t.Fatal("successful publication receipt must be durably recorded after recovery")
	}
	if raised, err := raiser.Raise(ctx, ask, facts); !raised || err != nil || len(receivedBodies) != 1 {
		t.Fatalf("published ask must not be replayed: raised=%v err=%v events=%d", raised, err, len(receivedBodies))
	}
}
