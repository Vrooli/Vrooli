package httpx

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/apihttptest"
)

func TestJSONRequestEncodesBodyAndRouteVars(t *testing.T) {
	req := JSONRequest(t, http.MethodPost, "/teams/team-1", map[string]string{"name": "Team"}, map[string]string{"id": "team-1"})

	if got := req.Header.Get("Content-Type"); got != "application/json" {
		t.Fatalf("expected content type application/json, got %q", got)
	}
	if got := mux.Vars(req)["id"]; got != "team-1" {
		t.Fatalf("expected route var id=team-1, got %q", got)
	}
	var body map[string]string
	if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
		t.Fatalf("decode request body: %v", err)
	}
	if body["name"] != "Team" {
		t.Fatalf("expected encoded body name Team, got %q", body["name"])
	}
}

func TestDecodeJSONAndAssertStatus(t *testing.T) {
	recorder := Recorder()
	recorder.Code = http.StatusAccepted
	recorder.Body.WriteString(`{"ok":true}`)

	apihttptest.AssertStatus(t, recorder.Result(), http.StatusAccepted)
	resp := DecodeJSON[struct {
		OK bool `json:"ok"`
	}](t, recorder)
	if !resp.OK {
		t.Fatal("expected decoded response ok=true")
	}
}
