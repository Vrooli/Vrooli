package planview

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"swarm-manager/internal/backlog"
	"swarm-manager/internal/gates"

	"github.com/gorilla/mux"
	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
	"google.golang.org/protobuf/encoding/protojson"
)

func newTestHandler(t *testing.T, cfg Config) *mux.Router {
	t.Helper()
	if cfg.Now == nil {
		cfg.Now = func() time.Time { return testNow }
	}
	svc, err := NewService(cfg)
	if err != nil {
		t.Fatal(err)
	}
	router := mux.NewRouter()
	NewHandler(svc).RegisterRoutes(router)
	return router
}

func TestGetBoard_ProtoShape(t *testing.T) {
	items := []backlog.BacklogItem{
		bItem("fix", "runnable", backlog.StatusReady),
		bItem("fix", "blocked", backlog.StatusBacklog, "fix/runnable"),
	}
	router := newTestHandler(t, Config{
		Backlog: stubBacklog{items: items},
		Gates:   stubGates{gates: []gates.Gate{decideGate("fix", "runnable", 2, "fix/blocked")}},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/plan?window_seconds=3600", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}

	var resp apipb.PlanBoardResponse
	if err := protojson.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("response is not a valid PlanBoardResponse: %v\n%s", err, rec.Body.String())
	}
	if resp.Meta == nil || resp.Meta.WindowSeconds != 3600 {
		t.Errorf("unexpected meta: %+v", resp.Meta)
	}
	if resp.Next == nil || resp.Next.CardCount != 1 {
		t.Errorf("expected 1 next card (decide gate), got %+v", resp.Next)
	}
	if resp.Later == nil || resp.Later.CardCount != 1 {
		t.Errorf("expected 1 later card, got %+v", resp.Later)
	}
	group := resp.Later.Groups[0]
	if group.BlockerKind != BlockerGate || group.GateId == "" {
		t.Errorf("expected gate-blocked later group, got %+v", group)
	}
	if resp.Now == nil {
		t.Error("now summary missing")
	}
}

func TestGetBoard_InvalidWindowRejected(t *testing.T) {
	router := newTestHandler(t, Config{Backlog: stubBacklog{}, Gates: stubGates{}})

	for _, raw := range []string{"abc", "-5", "0"} {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/plan?window_seconds="+raw, nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("window_seconds=%q: expected 400, got %d", raw, rec.Code)
		}
	}
}

func TestGetBoard_BuildErrorMaps500(t *testing.T) {
	router := newTestHandler(t, Config{
		Backlog: stubBacklog{err: errors.New("boom")},
		Gates:   stubGates{},
	})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/plan", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "failed to build plan projection") {
		t.Errorf("expected user-safe error body, got %q", rec.Body.String())
	}
}
