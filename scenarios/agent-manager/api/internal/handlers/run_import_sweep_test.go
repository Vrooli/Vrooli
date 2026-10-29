package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"agent-manager/internal/orchestration"
	"agent-manager/internal/transcriptindex"
)

type transcriptImporterFake struct {
	summary transcriptindex.Summary
	err     error
	calls   int
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
