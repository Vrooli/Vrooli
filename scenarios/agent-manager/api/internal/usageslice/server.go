// Package usageslice composes Agent Manager's real quota API without its
// orchestration, run database, UI, or desktop shell. The resource-owned Claude
// statusline file is the durable source; this read model is rebuilt on launch.
package usageslice

import (
	"context"
	"net/http"
	"sort"
	"sync"
	"time"

	"agent-manager/internal/handlers"
	"agent-manager/internal/pricing"
)

// New returns the same live-quota handler used by the full Agent Manager.
// Ingestion is synchronous before each Claude read so an optional install does
// not need a background service or a separate ingestion interval to catch up.
func New(samplesPath string) http.Handler {
	store := &observations{rows: make(map[string]pricing.QuotaObservation)}
	reader := &pricing.ClaudeStatuslineReader{Path: samplesPath, Store: store}
	quota := handlers.NewPricingHandlerWithQuota(nil, nil, nil, store)
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"healthy","service":"agent-manager","slice":"usage"}`))
	})
	mux.HandleFunc("GET /api/v1/pricing/live-quota", func(w http.ResponseWriter, r *http.Request) {
		// The owner's reader treats absent samples as unavailable. Avoid serving
		// a retained reading after a read failure by propagating an honest status.
		if agent := r.URL.Query().Get("agent"); agent == "claude" || agent == "claude-code" {
			if _, err := reader.Ingest(r.Context()); err != nil {
				http.Error(w, "usage samples unavailable", http.StatusServiceUnavailable)
				return
			}
		}
		quota.LiveQuota(w, r)
	})
	return mux
}

// observations retains the newest fact per session/window. The real quota
// handler needs those distinct sessions to detect conflicting accounts. Facts
// outside its freshness horizon are discarded; restarts replay the owner feed.
type observations struct {
	mu   sync.Mutex
	rows map[string]pricing.QuotaObservation
}

func (s *observations) RecordQuotaObservation(ctx context.Context, o *pricing.QuotaObservation) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := o.Validate(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	for key, row := range s.rows {
		if now.Sub(row.ObservedAt) > pricing.QuotaFreshnessMaxAge {
			delete(s.rows, key)
		}
	}
	if now.Sub(o.ObservedAt) > pricing.QuotaFreshnessMaxAge {
		return nil
	}
	key := o.Provider + "\x00" + o.EvidenceRef + "\x00" + o.Pool + "\x00" + o.Window
	if old, ok := s.rows[key]; !ok || o.ObservedAt.After(old.ObservedAt) {
		// The reader creates a new immutable observation for every feed entry.
		s.rows[key] = *o
	}
	return nil
}

func (s *observations) ListQuotaObservations(ctx context.Context, provider, pool, window string, limit int) ([]pricing.QuotaObservation, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	rows := make([]pricing.QuotaObservation, 0, len(s.rows))
	for _, row := range s.rows {
		if provider != "" && row.Provider != provider || pool != "" && row.Pool != pool || window != "" && row.Window != window {
			continue
		}
		rows = append(rows, row)
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].ObservedAt.After(rows[j].ObservedAt) })
	if limit > 0 && len(rows) > limit {
		rows = rows[:limit]
	}
	return rows, nil
}
