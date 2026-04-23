package feedback

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/mux"
)

// newHandlerEnv builds the full service+handler stack against temp-dir
// storage so HTTP tests exercise the real wiring. Uses the same fixtures
// as service_test.go — initiative "ui-rewrite" with one item "execute/foo".
type handlerEnv struct {
	*serviceEnv
	handler *Handler
	router  *mux.Router
}

func newHandlerEnv(t *testing.T) *handlerEnv {
	t.Helper()
	env := newServiceEnv(t)
	handler := NewHandler(env.svc)
	router := mux.NewRouter()
	handler.RegisterRoutes(router)
	return &handlerEnv{serviceEnv: env, handler: handler, router: router}
}

func (h *handlerEnv) do(method, path string, body io.Reader, contentType string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, body)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	rec := httptest.NewRecorder()
	h.router.ServeHTTP(rec, req)
	return rec
}

func (h *handlerEnv) decodeRound(t *testing.T, rec *httptest.ResponseRecorder) Round {
	t.Helper()
	var round Round
	if err := json.NewDecoder(rec.Body).Decode(&round); err != nil {
		t.Fatalf("decode round: %v (body=%s)", err, rec.Body.String())
	}
	return round
}

func TestHandler_Start_JSON(t *testing.T) {
	env := newHandlerEnv(t)
	body := `{"type":"feedback","text":"fix the ui"}`
	rec := env.do("POST", "/api/v1/initiatives/ui-rewrite/feedback", strings.NewReader(body), "application/json")
	if rec.Code != http.StatusCreated {
		t.Fatalf("status: got %d, body=%s", rec.Code, rec.Body.String())
	}
	round := env.decodeRound(t, rec)
	if round.Number != 1 {
		t.Fatalf("expected round 1, got %d", round.Number)
	}
	if round.Status != RoundStatusAgentThinking {
		t.Fatalf("expected agent_thinking, got %s", round.Status)
	}
}

func TestHandler_Start_Note(t *testing.T) {
	env := newHandlerEnv(t)
	body := `{"type":"note","text":"first note"}`
	rec := env.do("POST", "/api/v1/initiatives/ui-rewrite/feedback", strings.NewReader(body), "application/json")
	if rec.Code != http.StatusCreated {
		t.Fatalf("status: got %d, body=%s", rec.Code, rec.Body.String())
	}
	round := env.decodeRound(t, rec)
	if round.Status != RoundStatusDismissed {
		t.Fatalf("expected dismissed, got %s", round.Status)
	}
}

func TestHandler_Start_LockConflictReturns409(t *testing.T) {
	env := newHandlerEnv(t)
	if err := env.lock.Acquire("ui-rewrite", Holder{RunID: "prior", Purpose: "feedback"}); err != nil {
		t.Fatal(err)
	}
	body := `{"type":"feedback","text":"preempt"}`
	rec := env.do("POST", "/api/v1/initiatives/ui-rewrite/feedback", strings.NewReader(body), "application/json")
	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d body=%s", rec.Code, rec.Body.String())
	}
	var payload map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	holder, ok := payload["holder"].(map[string]any)
	if !ok || holder["run_id"] != "prior" {
		t.Fatalf("expected holder in 409 body, got %+v", payload)
	}
}

func TestHandler_List(t *testing.T) {
	env := newHandlerEnv(t)
	_, err := env.svc.StartRound(context.Background(), StartRoundRequest{
		InitiativeName: "ui-rewrite",
		Type:           RoundTypeNote,
		Text:           "one",
	})
	if err != nil {
		t.Fatal(err)
	}
	rec := env.do("GET", "/api/v1/initiatives/ui-rewrite/feedback", nil, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status: %d body=%s", rec.Code, rec.Body.String())
	}
	var payload struct {
		Rounds []Round `json:"rounds"`
		Count  int     `json:"count"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if payload.Count != 1 || len(payload.Rounds) != 1 {
		t.Fatalf("expected 1 round, got %+v", payload)
	}
}

func TestHandler_Get_NotFound(t *testing.T) {
	env := newHandlerEnv(t)
	rec := env.do("GET", "/api/v1/initiatives/ui-rewrite/feedback/99", nil, "")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestHandler_Continue_AfterAgentTurn(t *testing.T) {
	env := newHandlerEnv(t)
	r1 := env.do("POST", "/api/v1/initiatives/ui-rewrite/feedback",
		strings.NewReader(`{"type":"feedback","text":"start"}`), "application/json")
	if r1.Code != http.StatusCreated {
		t.Fatalf("start: %d", r1.Code)
	}
	round := env.decodeRound(t, r1)

	// Simulate agent turn via HTTP.
	turnBody := `{"body":"Reply without a proposal."}`
	rTurn := env.do("POST", "/api/v1/initiatives/ui-rewrite/feedback/1/agent-turn",
		strings.NewReader(turnBody), "application/json")
	if rTurn.Code != http.StatusOK {
		t.Fatalf("agent-turn: %d body=%s", rTurn.Code, rTurn.Body.String())
	}

	// Continue.
	rCont := env.do("POST", "/api/v1/initiatives/ui-rewrite/feedback/1/continue",
		strings.NewReader(`{"text":"try again"}`), "application/json")
	if rCont.Code != http.StatusOK {
		t.Fatalf("continue: %d body=%s", rCont.Code, rCont.Body.String())
	}
	cont := env.decodeRound(t, rCont)
	if cont.Status != RoundStatusAgentThinking {
		t.Fatalf("expected agent_thinking, got %s", cont.Status)
	}
	_ = round
}

func TestHandler_Decide_AcceptAppliesMutations(t *testing.T) {
	env := newHandlerEnv(t)
	// Start.
	_ = env.do("POST", "/api/v1/initiatives/ui-rewrite/feedback",
		strings.NewReader(`{"type":"feedback","text":"start"}`), "application/json")

	// Agent turn with a valid proposal.
	proposalBody := `{"body":"Plan:\n` + "```" + `json\n` +
		`{\"form\":\"mutation_list\",\"mutations\":[{\"id\":\"m1\",\"op\":\"change_priority\",\"target\":\"execute/foo\",\"priority\":9}]}` +
		`\n` + "```" + `"}`
	rTurn := env.do("POST", "/api/v1/initiatives/ui-rewrite/feedback/1/agent-turn",
		strings.NewReader(proposalBody), "application/json")
	if rTurn.Code != http.StatusOK {
		t.Fatalf("agent-turn: %d body=%s", rTurn.Code, rTurn.Body.String())
	}

	// Decide accept.
	rDecide := env.do("POST", "/api/v1/initiatives/ui-rewrite/feedback/1/decide",
		strings.NewReader(`{"kind":"accept","accepted_mutation_ids":["m1"]}`), "application/json")
	if rDecide.Code != http.StatusOK {
		t.Fatalf("decide: %d body=%s", rDecide.Code, rDecide.Body.String())
	}
	var payload map[string]any
	if err := json.NewDecoder(rDecide.Body).Decode(&payload); err != nil {
		t.Fatalf("decode decide: %v", err)
	}
	if payload["apply_result"] == nil {
		t.Fatalf("expected apply_result in decide response: %+v", payload)
	}

	// Item priority should now be 9.
	item, _ := env.bStore.LoadItem("execute", "foo")
	if item.Priority != 9 {
		t.Fatalf("expected priority 9, got %d", item.Priority)
	}
}

func TestHandler_Dismiss_SetsStatus(t *testing.T) {
	env := newHandlerEnv(t)
	_ = env.do("POST", "/api/v1/initiatives/ui-rewrite/feedback",
		strings.NewReader(`{"type":"feedback","text":"start"}`), "application/json")
	// Simulate agent turn with no proposal so round moves to awaiting_user.
	_ = env.do("POST", "/api/v1/initiatives/ui-rewrite/feedback/1/agent-turn",
		strings.NewReader(`{"body":"no proposal"}`), "application/json")
	rec := env.do("POST", "/api/v1/initiatives/ui-rewrite/feedback/1/dismiss",
		strings.NewReader(`{"rationale":"no"}`), "application/json")
	if rec.Code != http.StatusOK {
		t.Fatalf("dismiss: %d body=%s", rec.Code, rec.Body.String())
	}
	round := env.decodeRound(t, rec)
	if round.Status != RoundStatusDismissed {
		t.Fatalf("expected dismissed, got %s", round.Status)
	}
}

func TestHandler_LockStatus(t *testing.T) {
	env := newHandlerEnv(t)

	rec := env.do("GET", "/api/v1/initiatives/ui-rewrite/feedback/lock", nil, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("lock status: %d", rec.Code)
	}
	var payload map[string]any
	_ = json.NewDecoder(rec.Body).Decode(&payload)
	if payload["locked"] != false {
		t.Fatalf("expected locked=false, got %+v", payload)
	}

	if err := env.lock.Acquire("ui-rewrite", Holder{RunID: "run-x", Purpose: "feedback"}); err != nil {
		t.Fatal(err)
	}

	rec2 := env.do("GET", "/api/v1/initiatives/ui-rewrite/feedback/lock", nil, "")
	var payload2 map[string]any
	_ = json.NewDecoder(rec2.Body).Decode(&payload2)
	if payload2["locked"] != true {
		t.Fatalf("expected locked=true, got %+v", payload2)
	}
}

func TestHandler_Attachments_UploadAndServe(t *testing.T) {
	env := newHandlerEnv(t)

	// Build a multipart request with a PNG attachment.
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	_ = w.WriteField("type", "feedback")
	_ = w.WriteField("text", "with screenshot")

	// Minimal PNG magic bytes so the content-type sniff works. The
	// handler only checks the declared Content-Type header, so we can
	// ship arbitrary bytes as long as the header is set correctly.
	filePart, err := w.CreatePart(textprotoPNGHeader("hello.png"))
	if err != nil {
		t.Fatal(err)
	}
	_, _ = filePart.Write([]byte("\x89PNG\r\n\x1a\n"))
	_ = w.Close()

	req := httptest.NewRequest("POST", "/api/v1/initiatives/ui-rewrite/feedback", &body)
	req.Header.Set("Content-Type", w.FormDataContentType())
	rec := httptest.NewRecorder()
	env.router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("start w/ attachment: %d body=%s", rec.Code, rec.Body.String())
	}
	round := env.decodeRound(t, rec)
	if len(round.Submission.AttachmentIDs) != 1 {
		t.Fatalf("expected 1 attachment id, got %+v", round.Submission.AttachmentIDs)
	}

	// Fetch the attachment back.
	id := round.Submission.AttachmentIDs[0]
	// Drop the "attachments/" prefix; the URL expects the leaf id.
	leaf := strings.TrimPrefix(id, "attachments/")
	rec2 := env.do("GET", "/api/v1/initiatives/ui-rewrite/feedback/1/attachments/"+leaf, nil, "")
	if rec2.Code != http.StatusOK {
		t.Fatalf("fetch attachment: %d", rec2.Code)
	}
	if !strings.HasPrefix(rec2.Body.String(), "\x89PNG") {
		t.Fatalf("expected PNG bytes, got %q", rec2.Body.String()[:16])
	}
}

// textprotoPNGHeader constructs a multipart file-part header with a PNG
// Content-Type. Used by TestHandler_Attachments_UploadAndServe to ensure
// the MIME type attaches correctly to the uploaded file. Inlined so the
// test file is self-contained.
func textprotoPNGHeader(filename string) map[string][]string {
	return map[string][]string{
		"Content-Disposition": {"form-data; name=\"files\"; filename=\"" + filename + "\""},
		"Content-Type":        {"image/png"},
	}
}
