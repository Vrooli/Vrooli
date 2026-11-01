package handlers

import (
	"context"
	"math"
	"net/http"
	"sort"
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
	now     func() time.Time
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
		if c.clock().Before(c.expires) {
			result := c.value
			// A window can reset while cached. With nothing current left the
			// cached answer says nothing, so a new read replaces it.
			result.Windows = currentCodexWindows(result.Windows, c.clock())
			if result.Match != "likely" || len(result.Windows) > 0 {
				c.mu.Unlock()
				return result
			}
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
		now := c.clock().UTC()
		windows = currentCodexWindows(windows, now)
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

func (c *liveQuotaCache) clock() time.Time {
	if c.now != nil {
		return c.now()
	}
	return time.Now()
}

// currentCodexWindows returns a new slice of the windows whose reset has not
// passed, so the cached value is never edited in place.
func currentCodexWindows(windows []codecs.CodexLiveQuotaWindow, now time.Time) []codecs.CodexLiveQuotaWindow {
	current := make([]codecs.CodexLiveQuotaWindow, 0, len(windows))
	for _, window := range windows {
		if window.ResetsAt == nil || window.ResetsAt.After(now) {
			current = append(current, window)
		}
	}
	return current
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

// Readings from one account's sessions can differ by a tick, because each session
// renders at its own moment. A larger gap or a different reset identifies a
// different account.
const (
	claudeLiveQuotaSource             = "Claude Code statusline"
	claudeSameAccountPercentTolerance = 5.0
	claudeSameAccountResetTolerance   = time.Minute
)

// claudeLiveQuota returns each window's newest usable statusline reading. The
// store skips a window's repeated reading whichever session rendered it, so one
// account's windows can have different observation times and different sessions.
// Different sessions may use different accounts, and two tests leave the estimate
// unavailable rather than blending them:
//
//   - Two sessions report a window with different resets, at any age within
//     freshness. A window has one live reset per account, so this is proof. Each
//     session's latest reading of the window is what counts.
//   - Two sessions' readings of a window differ beyond the tolerance within two
//     minutes. A percent says nothing across a longer gap: usage moves, and an
//     idle session keeps rendering its cached value. Replayed against this host's
//     one-account history, comparing percents across sessions at any age raised
//     13 false conflicts in 675 readings; comparing resets raised none.
//
// Sessions of two accounts whose windows share a reset, or that never report the
// same window, cannot be told apart and are returned together. That is why the
// result is "likely" and never "verified".
func (h *PricingHandler) claudeLiveQuota(ctx context.Context, now time.Time) liveQuota {
	result := unavailable("No recent coherent Claude statusline sample is available.")
	result.Source = claudeLiveQuotaSource
	if h.quotaStore == nil {
		return result
	}
	rows, err := h.quotaStore.ListQuotaObservations(ctx, "anthropic", "", "", 100)
	if err != nil {
		return result
	}
	type sessionWindow struct {
		evidence string
		minutes  int64
	}
	usable := make([]pricing.QuotaObservation, 0, len(rows))
	newest := map[int64]pricing.QuotaObservation{}
	latest := map[sessionWindow]pricing.QuotaObservation{}
	for _, o := range rows {
		if o.Provider != "anthropic" || o.AccountScope != "" || o.Provenance != pricing.ClaudeStatuslineProvenance || o.EvidenceRef == "" || o.UsedPercent == nil || *o.UsedPercent < 0 || *o.UsedPercent > 100 || o.WindowMinutes == nil || *o.WindowMinutes <= 0 || o.ObservedAt.After(now.Add(time.Minute)) || now.Sub(o.ObservedAt) > pricing.QuotaFreshnessMaxAge || (o.ResetAt != nil && !o.ResetAt.After(now)) {
			continue
		}
		usable = append(usable, o)
		if current, ok := newest[*o.WindowMinutes]; !ok || o.ObservedAt.After(current.ObservedAt) {
			newest[*o.WindowMinutes] = o
		}
		key := sessionWindow{o.EvidenceRef, *o.WindowMinutes}
		if current, ok := latest[key]; !ok || o.ObservedAt.After(current.ObservedAt) {
			latest[key] = o
		}
	}
	if len(newest) == 0 {
		return result
	}
	for _, o := range usable {
		chosen := newest[*o.WindowMinutes]
		if o.EvidenceRef != chosen.EvidenceRef && o.ObservedAt.After(chosen.ObservedAt.Add(-2*time.Minute)) && differentClaudeAccount(chosen, o) {
			return conflictingClaudeAccounts()
		}
	}
	for _, o := range latest {
		if differentClaudeReset(newest[*o.WindowMinutes], o) {
			return conflictingClaudeAccounts()
		}
	}
	for minutes, o := range newest {
		observedAt := o.ObservedAt
		if observedAt.After(result.ObservedAt) {
			result.ObservedAt = observedAt
		}
		// The statusline pool identifies the window, not a separate subscription.
		result.Windows = append(result.Windows, codecs.CodexLiveQuotaWindow{Pool: "subscription", WindowMinutes: minutes, UsedPercent: *o.UsedPercent, ResetsAt: o.ResetAt, ObservedAt: &observedAt})
	}
	sort.Slice(result.Windows, func(i, j int) bool { return result.Windows[i].WindowMinutes < result.Windows[j].WindowMinutes })
	result.Match = "likely"
	result.Reason = "Most recent Claude statusline sample; the selected launch account is not confirmed."
	return result
}

func differentClaudeAccount(a, b pricing.QuotaObservation) bool {
	if math.Abs(*a.UsedPercent-*b.UsedPercent) > claudeSameAccountPercentTolerance {
		return true
	}
	return differentClaudeReset(a, b)
}

func differentClaudeReset(a, b pricing.QuotaObservation) bool {
	if a.ResetAt == nil || b.ResetAt == nil {
		return false
	}
	gap := a.ResetAt.Sub(*b.ResetAt)
	return gap > claudeSameAccountResetTolerance || gap < -claudeSameAccountResetTolerance
}

func conflictingClaudeAccounts() liveQuota {
	result := unavailable("Recent Claude sessions report conflicting usage; the account is not identifiable.")
	result.Source = claudeLiveQuotaSource
	return result
}
