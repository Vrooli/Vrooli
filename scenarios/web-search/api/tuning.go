package main

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// Tuning is web-search's boot-time control surface (documented in
// docs/reference/configuration.md). Each lever is read from a
// WEB_SEARCH_-prefixed env var; an unset, malformed, or out-of-range value
// yields the zero value here, which every consumer treats as "use the
// compiled default". Defaults therefore stay the SSOT in their owning
// packages (research / findings / livesearch).
type Tuning struct {
	// ConfidenceGate overrides research.HighConfidenceThreshold (0.75): the L3
	// reconcile gate above which a contradicting claim SUPERSEDES instead of
	// FLAGS. Valid range (0,1].
	ConfidenceGate float64
	// DecayHalfLife overrides findings.DecayHalfLife (180d): the read-path
	// confidence half-life; the GC min-age default derives from it (2×).
	DecayHalfLife time.Duration
	// GatherCap overrides research.MaxGatherFindings (20): the hard server-side
	// cap on the bounded GATHER sweep.
	GatherCap int
	// MaxResearchLoops overrides research.DefaultMaxResearchLoops (10): the
	// iteration budget written into the L3 task contract.
	MaxResearchLoops int
	// GovernorCapacity overrides livesearch.DefaultGovernorCapacity (60): live
	// SearXNG calls allowed per rolling minute before the service degrades.
	GovernorCapacity int
	// CacheTTL overrides livesearch.DefaultCacheTTL (5m): how long an identical
	// live query is served from cache without spending governor budget.
	CacheTTL time.Duration
	// FetchTimeout overrides fetch.DefaultHTTPTimeout (15s): the bound on one
	// HTTP-leg L2 page fetch.
	FetchTimeout time.Duration
	// FetchMaxBytes overrides fetch.DefaultMaxBodyBytes (2 MiB): the cap on
	// how much of a fetched page body is read.
	FetchMaxBytes int
	// BrowserEscalationOff disables the browser-automation-studio escalation
	// leg of the L2 fetch stack (WEB_SEARCH_BROWSER_ESCALATION=off). Default
	// is escalation ON, degrading to HTTP-only when BAS is unreachable.
	BrowserEscalationOff bool
	// MinReadableChars overrides fetch.DefaultMinReadableChars (200): the
	// JS-shell heuristic — an HTTP-leg result with fewer extracted characters
	// escalates to the browser leg.
	MinReadableChars int
	// SynthExcerptChars overrides research.DefaultExcerptChars (6000): the
	// per-document character budget the L2 excerpting step sends to the
	// synthesis model.
	SynthExcerptChars int
	// RelevantExcerptsOff disables the relevance-aware (chunk+embed) L2
	// excerpting (WEB_SEARCH_SYNTH_RELEVANT_EXCERPTS=off), reverting to
	// positional first-N-chars truncation. Default is relevance-aware ON,
	// degrading to positional automatically when the embedder is unreachable.
	RelevantExcerptsOff bool
}

// tuningFromEnv reads the control surface. Malformed values are ignored (the
// lever falls back to its compiled default) — a bad knob must never stop the
// scenario from booting.
func tuningFromEnv() Tuning {
	return Tuning{
		ConfidenceGate:   envFloat("WEB_SEARCH_HIGH_CONFIDENCE_THRESHOLD"),
		DecayHalfLife:    envDuration("WEB_SEARCH_DECAY_HALF_LIFE"),
		GatherCap:        envInt("WEB_SEARCH_MAX_GATHER_FINDINGS"),
		MaxResearchLoops: envInt("WEB_SEARCH_L3_MAX_LOOPS"),
		GovernorCapacity: envInt("WEB_SEARCH_GOVERNOR_CAPACITY"),
		CacheTTL:         envDuration("WEB_SEARCH_CACHE_TTL"),
		FetchTimeout:     envDuration("WEB_SEARCH_FETCH_TIMEOUT"),
		FetchMaxBytes:    envInt("WEB_SEARCH_FETCH_MAX_BYTES"),
		// Note the inverted sense: the lever is "escalation on/off" (default
		// on), the field is the off-switch so the zero value keeps the default.
		BrowserEscalationOff: envOff("WEB_SEARCH_BROWSER_ESCALATION"),
		MinReadableChars:     envInt("WEB_SEARCH_MIN_READABLE_CHARS"),
		SynthExcerptChars:    envInt("WEB_SEARCH_SYNTH_EXCERPT_CHARS"),
		// Same inverted sense as BrowserEscalationOff: relevance excerpting
		// defaults ON; the lever is its off-switch.
		RelevantExcerptsOff: envOff("WEB_SEARCH_SYNTH_RELEVANT_EXCERPTS"),
	}
}

// envOff reports whether an on/off lever is explicitly switched off. Unset or
// unrecognized values mean "leave the default (on)".
func envOff(key string) bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(key))) {
	case "off", "false", "0", "no", "disabled":
		return true
	default:
		return false
	}
}

// envFloat parses a positive float env var; 0 means unset/invalid.
func envFloat(key string) float64 {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return 0
	}
	v, err := strconv.ParseFloat(raw, 64)
	if err != nil || v <= 0 {
		return 0
	}
	return v
}

// envInt parses a positive integer env var; 0 means unset/invalid.
func envInt(key string) int {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return 0
	}
	v, err := strconv.Atoi(raw)
	if err != nil || v <= 0 {
		return 0
	}
	return v
}

// envDuration parses a positive Go duration env var (e.g. "5m", "4320h");
// 0 means unset/invalid.
func envDuration(key string) time.Duration {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return 0
	}
	d, err := time.ParseDuration(raw)
	if err != nil || d <= 0 {
		return 0
	}
	return d
}
