package operations

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"swarm-manager/internal/agentactivity"

	"github.com/gorilla/mux"
)

func newTestHandler(t *testing.T, records []agentactivity.Record) *mux.Router {
	t.Helper()
	agg, err := NewAggregator(AggregatorConfig{
		Activities: &fakeActivityLister{records: records},
		Governance: &fakeGovernance{resp: defaultGovernance()},
		Now:        fixedNow(),
	})
	if err != nil {
		t.Fatalf("NewAggregator: %v", err)
	}
	r := mux.NewRouter()
	NewHandler(agg).RegisterRoutes(r)
	return r
}

func decodeView(t *testing.T, body []byte) OperationsView {
	t.Helper()
	var view OperationsView
	if err := json.Unmarshal(body, &view); err != nil {
		t.Fatalf("decode: %v\nbody=%s", err, string(body))
	}
	return view
}

func TestHandler_DefaultWindow(t *testing.T) {
	r := newTestHandler(t, []agentactivity.Record{})
	req := httptest.NewRequest("GET", "/api/v1/operations", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", rec.Code, rec.Body.String())
	}
	view := decodeView(t, rec.Body.Bytes())
	if view.WindowSeconds != int(DefaultWindow.Seconds()) {
		t.Fatalf("WindowSeconds = %d, want %d", view.WindowSeconds, int(DefaultWindow.Seconds()))
	}
	if len(view.Lanes) != 4 {
		t.Fatalf("Lanes len = %d, want 4", len(view.Lanes))
	}
}

func TestHandler_AcceptsCustomWindow(t *testing.T) {
	r := newTestHandler(t, []agentactivity.Record{})
	req := httptest.NewRequest("GET", "/api/v1/operations?window=PT1H30M", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	view := decodeView(t, rec.Body.Bytes())
	want := int((90 * time.Minute).Seconds())
	if view.WindowSeconds != want {
		t.Fatalf("WindowSeconds = %d, want %d", view.WindowSeconds, want)
	}
}

func TestHandler_RejectsWindowOver24h(t *testing.T) {
	r := newTestHandler(t, []agentactivity.Record{})
	req := httptest.NewRequest("GET", "/api/v1/operations?window=PT25H", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestHandler_RejectsMalformedWindow(t *testing.T) {
	r := newTestHandler(t, []agentactivity.Record{})
	req := httptest.NewRequest("GET", "/api/v1/operations?window=3h", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestHandler_FilterByLane(t *testing.T) {
	now := fixedNow()()
	records := []agentactivity.Record{
		{
			ActivityID: "i1", OwnerType: agentactivity.OwnerBacklog, OwnerKind: "feature",
			OwnerName: "i1", Purpose: agentactivity.PurposeWorkshop, Status: agentactivity.StatusRunning,
			RequestedAt: now.Format(time.RFC3339),
		},
		{
			ActivityID: "e1", OwnerType: agentactivity.OwnerBacklog, OwnerKind: "feature",
			OwnerName: "e1", Purpose: agentactivity.PurposeProcess, Status: agentactivity.StatusRunning,
			RequestedAt: now.Format(time.RFC3339),
		},
	}
	r := newTestHandler(t, records)
	req := httptest.NewRequest("GET", "/api/v1/operations?lane=execute", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	view := decodeView(t, rec.Body.Bytes())
	if len(view.Activities) != 1 || view.Activities[0].ActivityID != "e1" {
		t.Fatalf("Activities = %+v, want [e1]", view.Activities)
	}
}

func TestHandler_FilterByLane_Bracketed(t *testing.T) {
	now := fixedNow()()
	records := []agentactivity.Record{
		{
			ActivityID: "e1", OwnerType: agentactivity.OwnerBacklog, OwnerKind: "feature",
			OwnerName: "e1", Purpose: agentactivity.PurposeProcess, Status: agentactivity.StatusRunning,
			RequestedAt: now.Format(time.RFC3339),
		},
		{
			ActivityID: "i1", OwnerType: agentactivity.OwnerBacklog, OwnerKind: "feature",
			OwnerName: "i1", Purpose: agentactivity.PurposeWorkshop, Status: agentactivity.StatusRunning,
			RequestedAt: now.Format(time.RFC3339),
		},
	}
	r := newTestHandler(t, records)
	req := httptest.NewRequest("GET", "/api/v1/operations?lane%5B%5D=execute&lane%5B%5D=investigate", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	view := decodeView(t, rec.Body.Bytes())
	if len(view.Activities) != 2 {
		t.Fatalf("Activities len = %d, want 2", len(view.Activities))
	}
}

func TestHandler_RejectsInvalidLane(t *testing.T) {
	r := newTestHandler(t, []agentactivity.Record{})
	req := httptest.NewRequest("GET", "/api/v1/operations?lane=deploy", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestHandler_FilterByStatus(t *testing.T) {
	now := fixedNow()()
	records := []agentactivity.Record{
		{
			ActivityID: "running", OwnerType: agentactivity.OwnerBacklog, OwnerKind: "feature",
			OwnerName: "x", Purpose: agentactivity.PurposeProcess, Status: agentactivity.StatusRunning,
			RequestedAt: now.Format(time.RFC3339),
		},
		{
			ActivityID: "review", OwnerType: agentactivity.OwnerBacklog, OwnerKind: "feature",
			OwnerName: "y", Purpose: agentactivity.PurposeProcess, Status: agentactivity.StatusNeedsReview,
			RequestedAt: now.Format(time.RFC3339),
		},
	}
	r := newTestHandler(t, records)
	req := httptest.NewRequest("GET", "/api/v1/operations?status=needs_review", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	view := decodeView(t, rec.Body.Bytes())
	if len(view.Activities) != 1 || view.Activities[0].ActivityID != "review" {
		t.Fatalf("Activities = %+v, want [review]", view.Activities)
	}
}

func TestHandler_SearchByOwner(t *testing.T) {
	now := fixedNow()()
	records := []agentactivity.Record{
		{
			ActivityID: "alpha", OwnerType: agentactivity.OwnerBacklog, OwnerKind: "feature",
			OwnerName: "alpha-thing", OwnerTitle: "Alpha thing", Purpose: agentactivity.PurposeProcess,
			Status: agentactivity.StatusRunning, RequestedAt: now.Format(time.RFC3339),
		},
		{
			ActivityID: "beta", OwnerType: agentactivity.OwnerBacklog, OwnerKind: "feature",
			OwnerName: "beta-thing", OwnerTitle: "Beta thing", Purpose: agentactivity.PurposeProcess,
			Status: agentactivity.StatusRunning, RequestedAt: now.Format(time.RFC3339),
		},
	}
	r := newTestHandler(t, records)
	req := httptest.NewRequest("GET", "/api/v1/operations?q=ALPHA", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	view := decodeView(t, rec.Body.Bytes())
	if len(view.Activities) != 1 || view.Activities[0].ActivityID != "alpha" {
		t.Fatalf("Activities = %+v, want [alpha]", view.Activities)
	}
}

func TestParseISO8601Duration(t *testing.T) {
	cases := []struct {
		in   string
		want time.Duration
		err  bool
	}{
		{"PT3H", 3 * time.Hour, false},
		{"PT1H30M", 90 * time.Minute, false},
		{"PT45M", 45 * time.Minute, false},
		{"PT90S", 90 * time.Second, false},
		{"pt2h", 2 * time.Hour, false},
		{"", 0, true},
		{"3H", 0, true},
		{"PT", 0, true},
		{"PT3", 0, true},
		{"PTH", 0, true},
		{"PT0S", 0, true},
		{"P3D", 0, true},
		{"PT1.5H", 0, true},
		{"PT3M2H", 0, true},
	}
	for _, c := range cases {
		got, err := parseISO8601Duration(c.in)
		if c.err {
			if err == nil {
				t.Errorf("parseISO8601Duration(%q) = %v, want error", c.in, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("parseISO8601Duration(%q) error: %v", c.in, err)
			continue
		}
		if got != c.want {
			t.Errorf("parseISO8601Duration(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}
