package channelmanager

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	core "channel-manager/internal/channelmanager"
	"github.com/gorilla/mux"
	_ "modernc.org/sqlite"
)

func testFormats() []core.Format {
	return []core.Format{{Kind: "test", MIMETypes: []string{"application/test"}, MaxBytes: 1, MaxDurationSecs: 1, MinWidth: 1, MinHeight: 1, MaxWidth: 1, MaxHeight: 1}}
}

func TestManualWorkflowOverHTTP(t *testing.T) {
	service, err := core.New([]core.Platform{{ID: "x", DailyCeiling: 3, ActionKinds: []string{"engage", "publish"}, Formats: testFormats()}}, []core.Program{{ID: "warm", PlatformID: "x", Preconditions: []string{"region"}, Phases: []core.Phase{{ID: "p", Allowed: []string{"engage"}}}, Provenance: core.Provenance{SourceKind: "operator", Confidence: "speculative", CapturedAt: "today", RevisitTrigger: "five runs", Sources: []string{"manual"}}}})
	if err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite", "file:handler?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err = db.Exec(core.Schema()); err != nil {
		t.Fatal(err)
	}
	router := mux.NewRouter()
	Module(service, core.NewStore(db)).Mount(router)
	call := func(method, path string, body any) *httptest.ResponseRecorder {
		b, _ := json.Marshal(body)
		r := httptest.NewRequest(method, path, bytes.NewReader(b))
		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)
		return w
	}
	identity := map[string]any{"id": "x-1", "platform_id": "x", "purpose": "brand", "environment_ref": "device", "vault_ref": "vault://channel/x", "attestations": map[string]bool{"region": true}}
	if got := call(http.MethodPost, "/api/v1/channel-manager/identities", identity).Code; got != http.StatusCreated {
		t.Fatalf("create=%d", got)
	}
	if got := call(http.MethodPost, "/api/v1/channel-manager/identities/x-1/start", map[string]string{"program_id": "warm"}).Code; got != http.StatusOK {
		t.Fatalf("start=%d", got)
	}
	action := call(http.MethodPost, "/api/v1/channel-manager/actions", map[string]any{"identity_id": "x-1", "kind": "engage", "seed": 4})
	if action.Code != http.StatusCreated {
		t.Fatalf("enqueue=%d: %s", action.Code, action.Body.String())
	}
	var result core.Action
	_ = json.Unmarshal(action.Body.Bytes(), &result)
	if got := call(http.MethodPost, "/api/v1/channel-manager/actions/"+result.ID+"/complete", map[string]string{"evidence": "https://proof"}).Code; got != http.StatusOK {
		t.Fatalf("complete=%d", got)
	}
	if got := call(http.MethodGet, "/api/v1/channel-manager/overview", nil).Code; got != http.StatusOK {
		t.Fatalf("overview=%d", got)
	}
}

// [REQ:CHANMGR-P0-013] [REQ:CHANMGR-P0-014] Eligibility never fails open and
// release retries return the original record rather than creating another one.
func TestEligibilityAndIdempotentReleaseOverHTTP(t *testing.T) {
	service, err := core.New([]core.Platform{{ID: "x", DailyCeiling: 3, ActionKinds: []string{"publish"}, Formats: testFormats()}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err = service.CreateIdentity(core.Identity{ID: "active", PlatformID: "x", Purpose: "brand", EnvironmentRef: "env", LaneGrants: []string{"main"}, Status: "active"}); err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite", "file:release?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err = db.Exec(core.Schema()); err != nil {
		t.Fatal(err)
	}
	router := mux.NewRouter()
	Module(service, core.NewStore(db)).Mount(router)
	call := func(method, path string, body any) *httptest.ResponseRecorder {
		b, _ := json.Marshal(body)
		r := httptest.NewRequest(method, path, bytes.NewReader(b))
		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)
		return w
	}
	unknown := call(http.MethodGet, "/api/v1/channel-manager/identities/missing/eligibility?lane=main", nil)
	if unknown.Code != http.StatusOK || !strings.Contains(unknown.Body.String(), "unknown") {
		t.Fatalf("unknown eligibility=%s", unknown.Body.String())
	}
	first := call(http.MethodPost, "/api/v1/channel-manager/releases", map[string]string{"identity_id": "active", "lane": "main", "idempotency_key": "release-1"})
	if first.Code != http.StatusCreated {
		t.Fatalf("first=%s", first.Body.String())
	}
	second := call(http.MethodPost, "/api/v1/channel-manager/releases", map[string]string{"identity_id": "active", "lane": "main", "idempotency_key": "release-1"})
	if second.Code != http.StatusCreated || first.Body.String() != second.Body.String() {
		t.Fatalf("retry must return original: %s / %s", first.Body.String(), second.Body.String())
	}
}
