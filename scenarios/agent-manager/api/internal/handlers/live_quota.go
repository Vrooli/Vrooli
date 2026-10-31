package handlers

import (
	"context"
	"net/http"
	"strings"
	"sync"
	"time"

	"agent-manager/internal/adapters/runner/codecs"
	"agent-manager/internal/pricing"
)

// liveQuota reports actual provider values, paired with an explicit assessment
// of their relationship to a future launch. No account IDs leave this boundary.
type liveQuota struct {
	Match      string                        `json:"match"`
	Reason     string                        `json:"reason"`
	Source     string                        `json:"source"`
	ObservedAt time.Time                     `json:"observedAt,omitempty"`
	Windows    []codecs.CodexLiveQuotaWindow `json:"windows"`
}

type liveQuotaCache struct {
	mu      sync.Mutex
	value   liveQuota
	expires time.Time
	pending chan struct{}
	read    func(context.Context) ([]codecs.CodexLiveQuotaWindow, error)
}

func defaultCodexQuotaRead(ctx context.Context) ([]codecs.CodexLiveQuotaWindow, error) {
	c, err := codecs.NewCodex()
	if err != nil {
		return nil, err
	}
	return c.ReadLiveQuota(ctx)
}

// cachedCodexLiveQuota coalesces requests and limits subprocess duration.
// Cancellation of the leader cancels its read; subsequent callers can retry.
func (c *liveQuotaCache) cachedCodexLiveQuota(ctx context.Context) liveQuota {
	for {
		c.mu.Lock()
		if time.Now().Before(c.expires) {
			result := c.value
			c.mu.Unlock()
			return result
		}
		if c.pending != nil {
			pending := c.pending
			c.mu.Unlock()
			select {
			case <-pending:
				continue
			case <-ctx.Done():
				return unavailable("Usage check cancelled.")
			}
		}
		c.pending = make(chan struct{})
		pending := c.pending
		reader := c.read
		if reader == nil {
			reader = defaultCodexQuotaRead
		}
		c.mu.Unlock()
		readCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		windows, err := reader(readCtx)
		cancel()
		now := time.Now().UTC()
		current := windows[:0]
		for _, window := range windows {
			if window.ResetsAt == nil || window.ResetsAt.After(now) {
				current = append(current, window)
			}
		}
		windows = current
		result := unavailable("Codex usage is unavailable in the Agent Manager environment.")
		result.Source = "Codex app-server"
		if err == nil && len(windows) > 0 {
			result = liveQuota{Match: "likely", Reason: "Reading from the local Agent Manager login; the selected launch account is not confirmed.", Source: "Codex app-server", ObservedAt: now, Windows: windows}
		}
		c.mu.Lock()
		c.value = result
		if ctx.Err() == nil {
			if result.Match == "likely" {
				c.expires = now.Add(time.Minute)
			} else {
				c.expires = now.Add(10 * time.Second)
			}
		}
		c.pending = nil
		close(pending)
		c.mu.Unlock()
		return result
	}
}

func unavailable(reason string) liveQuota {
	return liveQuota{Match: "unavailable", Reason: reason, Windows: []codecs.CodexLiveQuotaWindow{}}
}

func (h *PricingHandler) LiveQuota(w http.ResponseWriter, r *http.Request) {
	agent := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("agent")))
	var result liveQuota
	switch agent {
	case "codex":
		h.liveQuotaMu.Lock()
		if h.liveQuota == nil {
			h.liveQuota = &liveQuotaCache{}
		}
		cache := h.liveQuota
		h.liveQuotaMu.Unlock()
		result = cache.cachedCodexLiveQuota(r.Context())
	case "claude", "claude-code":
		result = h.claudeLiveQuota(r.Context(), time.Now().UTC())
	default:
		result = unavailable("Live usage is not supported for this agent.")
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, result)
}

// claudeLiveQuota returns only windows from the same newest statusline sample.
// Different sessions may use different accounts; conflicting recent samples
// leave the estimate unavailable rather than blending those sessions.
func (h *PricingHandler) claudeLiveQuota(ctx context.Context, now time.Time) liveQuota {
	result := unavailable("No recent coherent Claude statusline sample is available.")
	result.Source = "Claude Code statusline"
	if h.quotaStore == nil {
		return result
	}
	rows, err := h.quotaStore.ListQuotaObservations(ctx, "anthropic", "", "", 100)
	if err != nil {
		return result
	}
	var chosen *pricing.QuotaObservation
	for i := range rows {
		o := &rows[i]
		if o.Provider != "anthropic" || o.AccountScope != "" || o.Provenance != pricing.ClaudeStatuslineProvenance || o.EvidenceRef == "" || o.UsedPercent == nil || o.WindowMinutes == nil || *o.WindowMinutes <= 0 || o.ObservedAt.After(now.Add(time.Minute)) || now.Sub(o.ObservedAt) > pricing.QuotaFreshnessMaxAge || (o.ResetAt != nil && !o.ResetAt.After(now)) {
			continue
		}
		if chosen == nil || o.ObservedAt.After(chosen.ObservedAt) {
			chosen = o
		}
	}
	if chosen == nil {
		return result
	}
	for _, o := range rows {
		if o.Provider != "anthropic" || o.AccountScope != "" || o.Provenance != pricing.ClaudeStatuslineProvenance || o.UsedPercent == nil || o.WindowMinutes == nil {
			continue
		}
		if o.EvidenceRef != chosen.EvidenceRef && o.ObservedAt.After(chosen.ObservedAt.Add(-2*time.Minute)) && o.ObservedAt.Before(chosen.ObservedAt.Add(2*time.Minute)) {
			for _, same := range rows {
				if same.EvidenceRef != chosen.EvidenceRef || !same.ObservedAt.Equal(chosen.ObservedAt) || same.WindowMinutes == nil || *same.WindowMinutes != *o.WindowMinutes || same.UsedPercent == nil {
					continue
				}
				if *same.UsedPercent != *o.UsedPercent || (same.ResetAt != nil && o.ResetAt != nil && !same.ResetAt.Equal(*o.ResetAt)) {
					return unavailable("Recent Claude sessions report conflicting usage; the account is not identifiable.")
				}
			}
		}
		if o.EvidenceRef != chosen.EvidenceRef || !o.ObservedAt.Equal(chosen.ObservedAt) || (o.ResetAt != nil && !o.ResetAt.After(now)) || *o.WindowMinutes <= 0 || *o.UsedPercent < 0 || *o.UsedPercent > 100 {
			continue
		}
		// The statusline pool identifies the window, not a separate subscription.
		result.Windows = append(result.Windows, codecs.CodexLiveQuotaWindow{Pool: "subscription", WindowMinutes: *o.WindowMinutes, UsedPercent: *o.UsedPercent, ResetsAt: o.ResetAt})
	}
	if len(result.Windows) == 0 {
		return result
	}
	result.Match = "likely"
	result.Reason = "Most recent Claude statusline sample; the selected launch account is not confirmed."
	result.ObservedAt = chosen.ObservedAt
	return result
}
