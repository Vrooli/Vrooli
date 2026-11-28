package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"agent-manager/internal/orchestration"
	"agent-manager/internal/transcriptindex"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/identity"
)

type transcriptImporterFake struct {
	summary transcriptindex.Summary
	err     error
	calls   int
}

type replacementImporterFake struct {
	transcriptImporterFake
	runID         string
	fileKey       string
	expectedRunID string
}

func (f *replacementImporterFake) ReplacementIssues(_ context.Context, runID string) (transcriptindex.ReplacementIssues, error) {
	f.runID = runID
	return transcriptindex.ReplacementIssues{Items: []transcriptindex.ReplacementIssue{{FileKey: "key", Reason: "reconciliation required"}}}, nil
}

func (f *replacementImporterFake) ReconcileReplacement(_ context.Context, runID, fileKey string) error {
	if f.expectedRunID != "" && f.expectedRunID != runID {
		return errors.New("run mismatch")
	}
	f.runID, f.fileKey = runID, fileKey
	return nil
}

func TestReplacementReconciliationRouterEnforcesOperatorAndTarget(t *testing.T) {
	id := uuid.NewString()
	fake := &replacementImporterFake{expectedRunID: id}
	h := New(orchestration.EmptyHandlerServices(), WithTranscriptImporter(fake))
	router := mux.NewRouter()
	h.RegisterRoutes(router)
	wrapped := AgentOperationMiddleware(router)
	call := func(target string, principal identity.Principal) *httptest.ResponseRecorder {
		r := httptest.NewRequest(http.MethodPost, "/api/v1/runs/"+target+"/transcript-reconcile", strings.NewReader(`{"fileKey":"key"}`))
		r.RemoteAddr = "127.0.0.1:4321"
		r = r.WithContext(identity.WithPrincipal(r.Context(), principal))
		w := httptest.NewRecorder()
		wrapped.ServeHTTP(w, r)
		return w
	}
	agent := identity.Principal{Kind: identity.ActorAgent, Subject: uuid.NewString(), Verified: true, Scopes: []string{"agent-manager:write"}}
	if w := call(id, agent); w.Code != http.StatusForbidden || fake.fileKey != "" {
		t.Fatalf("ungranted agent admitted: %d %s", w.Code, w.Body.String())
	}
	operator := identity.Principal{Kind: identity.ActorHuman, Subject: "operator", Verified: true, Source: identity.SourceCloudflareAccess, Scopes: []string{"agent-manager:write"}}
	if w := call(uuid.NewString(), operator); w.Code == http.StatusOK || fake.fileKey != "" {
		t.Fatalf("cross-run target admitted: %d %s", w.Code, w.Body.String())
	}
	if w := call(id, operator); w.Code != http.StatusOK || fake.runID != id || fake.fileKey != "key" {
		t.Fatalf("operator refused: %d %s", w.Code, w.Body.String())
	}
}

func TestReplacementReconciliationHandlerFencesInputs(t *testing.T) {
	fake := &replacementImporterFake{}
	h := New(orchestration.EmptyHandlerServices(), WithTranscriptImporter(fake))
	id := uuid.NewString()
	request := func(method, body string) *http.Request {
		return mux.SetURLVars(httptest.NewRequest(method, "/", strings.NewReader(body)), map[string]string{"id": id})
	}
	w := httptest.NewRecorder()
	h.ListTranscriptReplacementIssues(w, request(http.MethodGet, ""))
	if w.Code != http.StatusOK || fake.runID != id || !strings.Contains(w.Body.String(), `"fileKey":"key"`) {
		t.Fatalf("inspection: %d %s", w.Code, w.Body.String())
	}
	w = httptest.NewRecorder()
	h.ReconcileTranscriptReplacement(w, request(http.MethodPost, `{"fileKey":"key"}`))
	if w.Code != http.StatusOK || fake.runID != id || fake.fileKey != "key" {
		t.Fatalf("reconciliation: %d %s", w.Code, w.Body.String())
	}
	for _, body := range []string{`{"fileKey":"key","extra":1}`, `{"fileKey":"key","fileKey":"other"}`, `{"fileKey":"key"}{}`, `{"fileKey":"` + strings.Repeat("x", 5000) + `"}`} {
		fake.fileKey = ""
		w = httptest.NewRecorder()
		h.ReconcileTranscriptReplacement(w, request(http.MethodPost, body))
		if w.Code == http.StatusOK || fake.fileKey != "" {
			t.Fatalf("invalid request admitted: %d", w.Code)
		}
	}
}

func (f *transcriptImporterFake) RunOnce(context.Context) (transcriptindex.Summary, error) {
	f.calls++
	return f.summary, f.err
}

func TestImportTranscriptSweepReturnsTheSweepSummary(t *testing.T) {
	fake := &transcriptImporterFake{summary: transcriptindex.Summary{Discovered: 7, Adopted: 2, Read: 5}}
	h := New(orchestration.EmptyHandlerServices(), WithTranscriptImporter(fake))

	rr := httptest.NewRecorder()
	h.ImportTranscriptSweep(rr, httptest.NewRequest(http.MethodPost, "/api/v1/runs/import-sweep", nil))

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var got transcriptindex.Summary
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Discovered != 7 || got.Adopted != 2 || got.Read != 5 || len(got.Failures) != 0 {
		t.Fatalf("summary = %#v", got)
	}
	if fake.calls != 1 {
		t.Fatalf("calls = %d, want exactly one sweep per request", fake.calls)
	}
}

// Without an importer the endpoint must say so. Returning an empty summary
// would read as "scanned everything, found nothing" — the same shape a healthy
// no-op sweep produces — and hide the misconfiguration it is meant to surface.
func TestImportTranscriptSweepReportsUnavailableRatherThanEmptySuccess(t *testing.T) {
	h := New(orchestration.EmptyHandlerServices())

	rr := httptest.NewRecorder()
	h.ImportTranscriptSweep(rr, httptest.NewRequest(http.MethodPost, "/api/v1/runs/import-sweep", nil))

	if rr.Code == http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}

// A typed-nil scheduler stored in the interface is not nil, so the option has
// to reject it or the endpoint answers with an empty sweep instead.
func TestWithTranscriptImporterRejectsTypedNilScheduler(t *testing.T) {
	var scheduler *orchestration.TranscriptIndex
	h := New(orchestration.EmptyHandlerServices(), WithTranscriptImporter(scheduler))

	rr := httptest.NewRecorder()
	h.ImportTranscriptSweep(rr, httptest.NewRequest(http.MethodPost, "/api/v1/runs/import-sweep", nil))

	if rr.Code == http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestImportTranscriptSweepSurfacesSweepFailure(t *testing.T) {
	fake := &transcriptImporterFake{err: errors.New("harness root unreadable")}
	h := New(orchestration.EmptyHandlerServices(), WithTranscriptImporter(fake))

	rr := httptest.NewRecorder()
	h.ImportTranscriptSweep(rr, httptest.NewRequest(http.MethodPost, "/api/v1/runs/import-sweep", nil))

	if rr.Code == http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}
