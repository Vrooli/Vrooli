// Package handlers — forensics surfaces.
//
// PRAGMATIC DEVIATION FROM PLAN §7.4 ("REST + protobuf-JSON"):
// these endpoints emit plain JSON via httputil.JSON with typed Go structs,
// not generated proto messages. The plan called for adding ~6 new proto
// messages (ForensicsSummary, BootHistory, PstoreReport, MCEReport,
// LogEntry, LogQueryResponse) plus a convert/ layer to mirror the metrics
// pattern. That cost was disproportionate for a forensics surface that:
//   - never re-uses the messages outside this scenario
//   - is consumed by one UI feature with no cross-scenario clients
//   - already mixes both styles in the existing codebase (/health is plain
//     JSON, /metrics/* is proto)
//
// If we later need typed clients (CLI generation, cross-scenario consumers)
// promoting these to proto is a mechanical migration. — 2026-05-07.
package handlers

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/httputil"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/services/autoheal"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/services/forensics"
)

// ForensicsService is the subset of *forensics.Service the handler needs.
type ForensicsService interface {
	Pstore() forensics.Envelope
	BootHistory(ctx context.Context) forensics.Envelope
	MCE(ctx context.Context) forensics.Envelope
}

// AutohealClient is the subset of *autoheal.Client the handler needs.
type AutohealClient interface {
	Forensics(ctx context.Context) autoheal.Envelope
}

// ForensicsHandler serves /api/v1/forensics/*.
type ForensicsHandler struct {
	svc      ForensicsService
	autoheal AutohealClient
	log      *slog.Logger
	now      func() time.Time
}

// NewForensicsHandler builds the handler.
func NewForensicsHandler(svc ForensicsService, ah AutohealClient, log *slog.Logger) *ForensicsHandler {
	return &ForensicsHandler{svc: svc, autoheal: ah, log: log, now: time.Now}
}

// Pstore handles GET /api/v1/forensics/pstore.
func (h *ForensicsHandler) Pstore(w http.ResponseWriter, r *http.Request) {
	httputil.JSON(w, h.svc.Pstore()) //nolint:errcheck
}

// BootHistory handles GET /api/v1/forensics/boot-history.
func (h *ForensicsHandler) BootHistory(w http.ResponseWriter, r *http.Request) {
	httputil.JSON(w, h.svc.BootHistory(r.Context())) //nolint:errcheck
}

// MCE handles GET /api/v1/forensics/mce.
func (h *ForensicsHandler) MCE(w http.ResponseWriter, r *http.Request) {
	httputil.JSON(w, h.svc.MCE(r.Context())) //nolint:errcheck
}

// SummaryResponse aggregates the four forensics signals.
type SummaryResponse struct {
	GeneratedAt time.Time          `json:"generatedAt"`
	Pstore      forensics.Envelope `json:"pstore"`
	BootHistory forensics.Envelope `json:"bootHistory"`
	MCE         forensics.Envelope `json:"mce"`
	Autoheal    autoheal.Envelope  `json:"autoheal"`
}

// Summary handles GET /api/v1/forensics/summary.
func (h *ForensicsHandler) Summary(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	resp := SummaryResponse{
		GeneratedAt: h.now().UTC(),
		Pstore:      h.svc.Pstore(),
		BootHistory: h.svc.BootHistory(ctx),
		MCE:         h.svc.MCE(ctx),
	}
	if h.autoheal != nil {
		resp.Autoheal = h.autoheal.Forensics(ctx)
	} else {
		resp.Autoheal = autoheal.Envelope{Reason: "autoheal client not configured"}
	}
	httputil.JSON(w, resp) //nolint:errcheck
}
