package handlers

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/services/autoheal"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/services/forensics"
)

type fakeForensics struct {
	pstore   forensics.Envelope
	bootHist forensics.Envelope
	mce      forensics.Envelope
}

func (f *fakeForensics) Pstore() forensics.Envelope                       { return f.pstore }
func (f *fakeForensics) BootHistory(_ context.Context) forensics.Envelope { return f.bootHist }
func (f *fakeForensics) MCE(_ context.Context) forensics.Envelope         { return f.mce }

type fakeAutoheal struct {
	env autoheal.Envelope
}

func (f *fakeAutoheal) Forensics(_ context.Context) autoheal.Envelope { return f.env }

func newForensicsHandler(svc *fakeForensics, ah *fakeAutoheal) *ForensicsHandler {
	if ah == nil {
		return NewForensicsHandler(svc, nil, slog.Default())
	}
	return NewForensicsHandler(svc, ah, slog.Default())
}

func TestForensicsHandlerPstore(t *testing.T) {
	svc := &fakeForensics{pstore: forensics.Envelope{Available: true, Data: forensics.PstoreReport{Path: "/sys/fs/pstore"}}}
	h := newForensicsHandler(svc, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/forensics/pstore", nil)
	w := httptest.NewRecorder()
	h.Pstore(w, req)
	if w.Code != 200 {
		t.Errorf("status = %d", w.Code)
	}
	var got map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got["available"] != true {
		t.Errorf("available: %v", got["available"])
	}
}

func TestForensicsHandlerNotProvisioned(t *testing.T) {
	svc := &fakeForensics{pstore: forensics.Envelope{Reason: "not present"}}
	h := newForensicsHandler(svc, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/forensics/pstore", nil)
	w := httptest.NewRecorder()
	h.Pstore(w, req)
	// Never 5xx for not-provisioned.
	if w.Code != 200 {
		t.Errorf("status = %d, want 200", w.Code)
	}
	var got map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &got)
	if got["available"] != false {
		t.Errorf("available: %v", got["available"])
	}
	if got["reason"] != "not present" {
		t.Errorf("reason: %v", got["reason"])
	}
}

func TestForensicsHandlerSummaryAggregates(t *testing.T) {
	svc := &fakeForensics{
		pstore:   forensics.Envelope{Available: true},
		bootHist: forensics.Envelope{Reason: "no journalctl"},
		mce:      forensics.Envelope{Reason: "ras-mc-ctl not installed"},
	}
	ah := &fakeAutoheal{env: autoheal.Envelope{Available: true, Checks: []autoheal.ForensicsRelevantCheck{{CheckID: "system-pstore-evidence", Status: "OK"}}}}
	h := newForensicsHandler(svc, ah)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/forensics/summary", nil)
	w := httptest.NewRecorder()
	h.Summary(w, req)
	if w.Code != 200 {
		t.Errorf("status: %d", w.Code)
	}
	var resp SummaryResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if !resp.Pstore.Available {
		t.Error("pstore should be available")
	}
	if resp.BootHistory.Available {
		t.Error("boot history should be unavailable")
	}
	if !resp.Autoheal.Available {
		t.Error("autoheal should be available")
	}
	if len(resp.Autoheal.Checks) != 1 {
		t.Errorf("checks: %+v", resp.Autoheal.Checks)
	}
}

func TestForensicsHandlerSummaryAutohealOffline(t *testing.T) {
	svc := &fakeForensics{}
	ah := &fakeAutoheal{env: autoheal.Envelope{Reason: "autoheal unreachable"}}
	h := newForensicsHandler(svc, ah)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/forensics/summary", nil)
	w := httptest.NewRecorder()
	h.Summary(w, req)
	if w.Code != 200 {
		t.Errorf("status: %d", w.Code)
	}
	var resp SummaryResponse
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Autoheal.Available {
		t.Error("autoheal should be unavailable")
	}
}

func TestForensicsHandlerSummaryNilAutoheal(t *testing.T) {
	svc := &fakeForensics{}
	h := newForensicsHandler(svc, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/forensics/summary", nil)
	w := httptest.NewRecorder()
	h.Summary(w, req)
	var resp SummaryResponse
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Autoheal.Available {
		t.Error("autoheal should be unavailable when nil")
	}
}
