package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration"
	"github.com/google/uuid"
)

type importIdentityService struct {
	orchestration.RunService
	ids   []uuid.UUID
	err   error
	calls int
}

func (s *importIdentityService) FindImportProvenance(_ context.Context, harness, session string) ([]uuid.UUID, error) {
	s.calls++
	if harness != "codex" || session != "session-1" {
		return nil, errors.New("unexpected identity")
	}
	return s.ids, s.err
}

func TestFindImportIdentityDistinguishesAbsenceAmbiguityAndFailure(t *testing.T) {
	id1, id2 := uuid.New(), uuid.New()
	for _, tc := range []struct {
		name, query, state string
		ids                []uuid.UUID
		err                error
		status             int
		calls              int
	}{
		{"not observed", "harness=codex&sessionId=session-1", "not_observed", nil, nil, 200, 1},
		{"matched alias", "harness=resource%3Acodex%2Fsessions&sessionId=session-1", "matched", []uuid.UUID{id1}, nil, 200, 1},
		{"ambiguous", "harness=codex&sessionId=session-1", "ambiguous", []uuid.UUID{id1, id2}, nil, 200, 1},
		{"unsupported", "harness=unknown&sessionId=session-1", "", nil, nil, 400, 0},
		{"missing", "harness=codex", "", nil, nil, 400, 0},
		{"unavailable", "harness=codex&sessionId=session-1", "", nil, errors.New("store down"), 500, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			svc := &importIdentityService{ids: tc.ids, err: tc.err}
			h := &Handler{svc: orchestration.HandlerServices{RunService: svc}}
			w := httptest.NewRecorder()
			h.FindImportIdentity(w, httptest.NewRequest(http.MethodGet, "/api/v1/import/identity?"+tc.query, nil))
			if w.Code != tc.status || svc.calls != tc.calls {
				t.Fatalf("status=%d calls=%d body=%s", w.Code, svc.calls, w.Body.String())
			}
			if tc.state != "" {
				var result struct {
					State string      `json:"state"`
					Scope string      `json:"scope"`
					IDs   []uuid.UUID `json:"runIds"`
				}
				if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil || result.State != tc.state || result.Scope != "imported_run_provenance" || len(result.IDs) != len(tc.ids) {
					t.Fatalf("response=%s err=%v", w.Body.String(), err)
				}
			}
		})
	}
}

func TestFindImportIdentityRoute(t *testing.T) {
	h, router := setupTestHandler(t)
	svc := &importIdentityService{RunService: h.svc.RunService}
	h.svc.RunService = svc
	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/import/identity?harness=codex&sessionId=session-1", nil))
	if w.Code != http.StatusOK || svc.calls != 1 {
		t.Fatalf("route status=%d calls=%d body=%s", w.Code, svc.calls, w.Body.String())
	}
}

func TestScanRunnerSessionsUsesOpaqueKeysAndReadsCodexConversationTitle(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "2026", "07", "rollout.jsonl")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	contents := "{\"type\":\"session_meta\",\"payload\":{\"id\":\"session-1\"}}\n{\"type\":\"response_item\",\"payload\":{\"role\":\"user\",\"content\":[{\"text\":\"Repair the import workflow\"}]}}\n"
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
	sessions, err := scanRunnerSessions(runnerSessionSource{RunnerType: domain.RunnerTypeCodex, Root: root}, map[string]string{"session-1": "run-1"})
	if err != nil {
		t.Fatal(err)
	}
	if len(sessions) != 1 {
		t.Fatalf("sessions = %d, want 1", len(sessions))
	}
	if got := sessions[0]; got.Key != "2026/07/rollout.jsonl" || got.Title != "Repair the import workflow" || got.ImportedRunID != "run-1" {
		t.Fatalf("unexpected session: %#v", got)
	}
}

func TestSafeSessionPathRejectsTraversal(t *testing.T) {
	root := t.TempDir()
	source := runnerSessionSource{Root: root}
	if _, ok := safeSessionPath(source, "../secrets.jsonl"); ok {
		t.Fatal("path traversal accepted")
	}
	if _, ok := safeSessionPath(source, "/tmp/session.jsonl"); ok {
		t.Fatal("absolute path accepted")
	}
	path := filepath.Join(root, "session.jsonl")
	if err := os.WriteFile(path, []byte("{}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got, ok := safeSessionPath(source, "session.jsonl"); !ok || got != path {
		t.Fatalf("safe path = %q, %v", got, ok)
	}
}
