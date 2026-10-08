package board

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"content-desk/integrations/programruntime"

	"github.com/gorilla/mux"
)

type fakeRunner struct {
	result programruntime.Result
	err    error
	name   string
	inputs map[string]any
}

func (f *fakeRunner) RunDeclared(_ context.Context, name string, inputs map[string]any) (programruntime.Result, error) {
	f.name = name
	f.inputs = inputs
	return f.result, f.err
}

func mount(t *testing.T, runner programruntime.Runner) *mux.Router {
	t.Helper()
	if runner == nil {
		// Keep the route mounted so the path itself is exercised.
		runner = &fakeRunner{err: errors.New("missing")}
	}
	mux := mux.NewRouter()
	Module(runner).Mount(mux)
	return mux
}

func TestReadHandlerReturnsEnvelopeVerbatim(t *testing.T) {
	envelope := `{"program":"content-desk.board-read","status":"ok","signals":{"offer_read_status":"read"},"errors":[],"evidence":["content-desk/campaigns/list"]}`
	runner := &fakeRunner{result: programruntime.Result{
		Envelope:  []byte(envelope),
		ProgramID: "prog_test",
		Terminal:  true,
		Status:    "PROGRAM_STATUS_SUCCEEDED",
	}}
	rec := httptest.NewRecorder()
	mount(t, runner).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/board", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("content-type = %q, want application/json", got)
	}
	if rec.Body.String() != envelope {
		t.Fatalf("body = %s, want the envelope unchanged", rec.Body.String())
	}
	if runner.name != ProgramName {
		t.Fatalf("ran %q, want %q", runner.name, ProgramName)
	}
	if runner.inputs != nil {
		t.Fatalf("inputs = %v, want nil (no baseline on the browser edge)", runner.inputs)
	}
}

func TestReadHandlerUnavailable(t *testing.T) {
	rec := httptest.NewRecorder()
	mount(t, &fakeRunner{err: errors.New("program-runtime unreachable")}).
		ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/board", nil))

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", rec.Code)
	}
	var body struct {
		Code string `json:"code"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode error envelope: %v", err)
	}
	if body.Code != "unavailable" {
		t.Fatalf("code = %q, want unavailable", body.Code)
	}
}

func TestReadHandlerRejectsNonJSONEnvelope(t *testing.T) {
	runner := &fakeRunner{result: programruntime.Result{Envelope: []byte("not json"), Terminal: true}}
	rec := httptest.NewRecorder()
	mount(t, runner).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/board", nil))

	if rec.Code != http.StatusBadGateway {
		t.Fatalf("status = %d, want 502", rec.Code)
	}
}

func TestReadHandlerRejectsMissingRunner(t *testing.T) {
	rec := httptest.NewRecorder()
	mux := mux.NewRouter()
	Module(nil).Mount(mux)
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/board", nil))

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", rec.Code)
	}
}
