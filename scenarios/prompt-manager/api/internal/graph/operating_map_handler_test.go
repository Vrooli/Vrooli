package graph

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetOperatingMap_ServesSharedAssembler(t *testing.T) {
	h := NewHandlers(&mockGraphIndexProvider{})
	h.SetOperatingMapStore(staticOperatingMapProvider{mapResult: OperatingMap{Teams: []OperatingMapTeam{{ID: "team-a"}}}})
	r := httptest.NewRecorder()
	h.GetOperatingMap(r, httptest.NewRequest(http.MethodGet, "/operating-models/map", nil))
	if r.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", r.Code, r.Body.String())
	}
	if got := r.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("content type = %q", got)
	}
	if got := r.Body.String(); got == "" || got[0] != '{' {
		t.Fatalf("unexpected response %q", got)
	}
}

type staticOperatingMapProvider struct{ mapResult OperatingMap }

func (p staticOperatingMapProvider) Get(context.Context) (OperatingMap, error) {
	return p.mapResult, nil
}
