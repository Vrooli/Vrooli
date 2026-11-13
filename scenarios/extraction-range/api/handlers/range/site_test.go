package rangehandler

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	rangeDomain "extraction-range/internal/range"
	"extraction-range/internal/rangeengine"
	"github.com/vrooli/api-core/database"
	_ "modernc.org/sqlite"
)

func TestScoreboardPersistsAcrossHandlerRestart(t *testing.T) {
	primary, err := sql.Open("sqlite", "file:scoreboard-test?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer primary.Close()
	db := database.NewFromPrimary(primary)
	if _, err := primary.ExecContext(context.Background(), rangeDomain.Schema()); err != nil {
		t.Fatal(err)
	}
	h, err := New(db)
	if err != nil {
		t.Fatal(err)
	}
	levelsByFamily := map[string]rangeengine.Level{}
	for _, level := range h.engine.List() {
		if _, found := levelsByFamily[level.Family]; !found {
			levelsByFamily[level.Family] = level
		}
	}
	if len(levelsByFamily) != 10 {
		t.Fatalf("range has %d families, want 10", len(levelsByFamily))
	}
	for _, level := range levelsByFamily {
		body, _ := json.Marshal(map[string]any{"tool": "persistence-test", "level_id": level.ID, "records": level.Answer})
		post := httptest.NewRecorder()
		h.PracticeAPI().ServeHTTP(post, httptest.NewRequest(http.MethodPost, "/score", bytes.NewReader(body)))
		if post.Code != http.StatusOK {
			t.Fatalf("POST /score for %s: %d %s", level.ID, post.Code, post.Body.String())
		}
	}
	restarted, err := New(db)
	if err != nil {
		t.Fatal(err)
	}
	get := httptest.NewRecorder()
	restarted.PracticeAPI().ServeHTTP(get, httptest.NewRequest(http.MethodGet, "/scoreboard?tool=persistence-test", nil))
	if get.Code != http.StatusOK {
		t.Fatalf("GET /scoreboard: %d %s", get.Code, get.Body.String())
	}
	var payload struct {
		Scores []map[string]any `json:"scores"`
	}
	if err := json.Unmarshal(get.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if len(payload.Scores) != 10 {
		t.Fatalf("wanted persisted scores spanning 10 families, got %d: %s", len(payload.Scores), get.Body.String())
	}
	families := map[string]bool{}
	for _, score := range payload.Scores {
		families[score["family"].(string)] = true
		if score["passed"] != true {
			t.Fatalf("score did not persist as pass: %v", score)
		}
	}
	if len(families) != 10 {
		t.Fatalf("persisted scoreboard covers %d families: %s", len(families), get.Body.String())
	}
}

func TestScriptLoadedRowsExposeStableFieldsAfterBrowserInteraction(t *testing.T) {
	h, err := New()
	if err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	h.PracticeSite().ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/practice/script-loaded-endless-scroll-04", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("practice page status=%d", response.Code)
	}
	for _, want := range []string{"id=\"load\"", "data-source=\"/practice/script-loaded-endless-scroll-04/data\"", "data-field='id'", "data-field='name'", "data-field='category'", "data-field='value'"} {
		if !strings.Contains(response.Body.String(), want) {
			t.Errorf("practice page lacks stable browser target %q", want)
		}
	}
}

// [REQ:EXTRACTI-P0-003] The operator/tool contract exposes metadata and owned
// target links while keeping answers out of level listings.
func TestRESTPracticeContractScoresAndPublishesBestPerToolLevel(t *testing.T) {
	h, err := New()
	if err != nil {
		t.Fatal(err)
	}
	api := h.PracticeAPI()

	list := httptest.NewRecorder()
	api.ServeHTTP(list, httptest.NewRequest(http.MethodGet, "/levels", nil))
	if list.Code != http.StatusOK {
		t.Fatalf("GET /levels: status=%d body=%s", list.Code, list.Body.String())
	}
	var index struct {
		Levels []map[string]any `json:"levels"`
	}
	if err := json.Unmarshal(list.Body.Bytes(), &index); err != nil {
		t.Fatal(err)
	}
	if len(index.Levels) < 40 {
		t.Fatalf("level route returned %d levels", len(index.Levels))
	}
	if _, leaks := index.Levels[0]["answer"]; leaks {
		t.Fatal("level index exposes held answers")
	}

	level, ok := h.engine.Get("plain-tables-01")
	if !ok {
		t.Fatal("fixture missing")
	}
	payload, err := json.Marshal(map[string]any{"tool": "integration-tool", "level_id": level.ID, "records": level.Answer})
	if err != nil {
		t.Fatal(err)
	}
	score := httptest.NewRecorder()
	api.ServeHTTP(score, httptest.NewRequest(http.MethodPost, "/score", bytes.NewReader(payload)))
	if score.Code != http.StatusOK {
		t.Fatalf("POST /score: status=%d body=%s", score.Code, score.Body.String())
	}
	var result struct {
		F1       float64 `json:"f1"`
		Passed   bool    `json:"passed"`
		Expected int     `json:"expected"`
	}
	if err := json.Unmarshal(score.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if !result.Passed || result.F1 != 1 || result.Expected != len(level.Answer) {
		t.Fatalf("unexpected score: %+v", result)
	}

	board := httptest.NewRecorder()
	api.ServeHTTP(board, httptest.NewRequest(http.MethodGet, "/scoreboard?tool=integration-tool", nil))
	if board.Code != http.StatusOK || !bytes.Contains(board.Body.Bytes(), []byte(level.ID)) {
		t.Fatalf("scoreboard missed score: status=%d body=%s", board.Code, board.Body.String())
	}
}

func TestPracticePageRendersTargetAndSessionGateIsSimulated(t *testing.T) {
	h, err := New()
	if err != nil {
		t.Fatal(err)
	}
	target := httptest.NewRecorder()
	h.PracticeSite().ServeHTTP(target, httptest.NewRequest(http.MethodGet, "/practice/plain-tables-01", nil))
	if target.Code != http.StatusOK || !bytes.Contains(target.Body.Bytes(), []byte("<table>")) {
		t.Fatalf("plain-table target did not render: %d", target.Code)
	}
	protected := httptest.NewRecorder()
	h.PracticeSite().ServeHTTP(protected, httptest.NewRequest(http.MethodGet, "/practice/login-sessions-01", nil))
	if protected.Code != http.StatusOK || !bytes.Contains(protected.Body.Bytes(), []byte("Practice sign in")) {
		t.Fatalf("session challenge did not render sign-in page: %d", protected.Code)
	}
}

func TestOwnedAccessInstructionsAndTransientRetryAreUsable(t *testing.T) {
	h, err := New()
	if err != nil {
		t.Fatal(err)
	}
	level, ok := h.engine.Get("login-sessions-01")
	if !ok {
		t.Fatal("session level missing")
	}
	if !strings.Contains(level.Task, level.Page.SessionCode) || !strings.Contains(level.Task, "active") {
		t.Fatalf("task omits safe access instructions or scored field: %q", level.Task)
	}
	form := url.Values{"code": {level.Page.SessionCode}}
	req := httptest.NewRequest(http.MethodPost, "/practice/"+level.ID+"/session", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	login := httptest.NewRecorder()
	h.PracticeSite().ServeHTTP(login, req)
	if login.Code != http.StatusSeeOther || login.Result().Cookies()[0].Value != level.Page.SessionCode {
		t.Fatalf("synthetic login failed: status=%d cookies=%v", login.Code, login.Result().Cookies())
	}
	pageReq := httptest.NewRequest(http.MethodGet, "/practice/"+level.ID, nil)
	pageReq.AddCookie(login.Result().Cookies()[0])
	page := httptest.NewRecorder()
	h.PracticeSite().ServeHTTP(page, pageReq)
	if page.Code != http.StatusOK || !bytes.Contains(page.Body.Bytes(), []byte("<table>")) {
		t.Fatalf("authorized session did not reveal target rows: %d", page.Code)
	}

	resilience, ok := h.engine.Get("rate-errors-timeouts-02")
	if !ok {
		t.Fatal("retry level missing")
	}
	first := httptest.NewRecorder()
	h.PracticeSite().ServeHTTP(first, httptest.NewRequest(http.MethodGet, "/practice/"+resilience.ID, nil))
	if first.Code != http.StatusTooManyRequests || first.Header().Get("Retry-After") != "2" {
		t.Fatalf("first attempt should be retryable 429: status=%d headers=%v", first.Code, first.Header())
	}
	second := httptest.NewRecorder()
	h.PracticeSite().ServeHTTP(second, httptest.NewRequest(http.MethodGet, "/practice/"+resilience.ID, nil))
	if second.Code != http.StatusOK || !bytes.Contains(second.Body.Bytes(), []byte("<table>")) {
		t.Fatalf("retry did not recover with records: %d", second.Code)
	}
}

func TestEveryLayoutAndTargetMechanismContainsScoredActiveField(t *testing.T) {
	h, err := New()
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"changing-layouts-01?layout=cards", "messy-markup-01", "script-loaded-endless-scroll-01", "downloads-01?format=csv"} {
		r := httptest.NewRecorder()
		h.PracticeSite().ServeHTTP(r, httptest.NewRequest(http.MethodGet, "/practice/"+id, nil))
		if r.Code != http.StatusOK {
			t.Errorf("%s returned %d", id, r.Code)
			continue
		}
		if !bytes.Contains(r.Body.Bytes(), []byte("active")) {
			t.Errorf("%s hides the scored active field", id)
		}
	}
}

func TestSyntheticTimeoutDelayUsesInjectableClockSeam(t *testing.T) {
	h, err := New()
	if err != nil {
		t.Fatal(err)
	}
	var waited time.Duration
	h.sleep = func(delay time.Duration) { waited = delay }
	response := httptest.NewRecorder()
	h.PracticeSite().ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/practice/rate-errors-timeouts-04", nil))
	if waited != 350*time.Millisecond {
		t.Fatalf("recorded delay %s, want 350ms", waited)
	}
	if response.Code != http.StatusOK || !bytes.Contains(response.Body.Bytes(), []byte("<table>")) {
		t.Fatalf("delayed page did not serve records: status=%d", response.Code)
	}
}
