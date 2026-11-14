// Responsibility: structured coding limit evidence and bounded backup consent decisions.
package cliutil

import (
	"encoding/json"
	"time"
)

// CodingLimit is evidence from the provider's structured error channel or the
// usage owner's fresh reading. Callers must never fill it from terminal prose
// or client-supplied assertions. ObservedAt is assigned by the reader.
type CodingLimit struct {
	Harness    string    `json:"harness"`
	Kind       string    `json:"kind"`
	ObservedAt time.Time `json:"observed_at"`
	ResetsAt   time.Time `json:"resets_at,omitempty"`
}

func (l CodingLimit) Exhausted(now time.Time) bool {
	return l.Harness != "" && (l.Kind == "provider_limit" || l.Kind == "quota_exhausted") &&
		!l.ObservedAt.IsZero() && !l.ObservedAt.After(now) && now.Sub(l.ObservedAt) <= 5*time.Minute &&
		(l.ResetsAt.IsZero() || now.Before(l.ResetsAt))
}

// CodingBackupConsent is deliberate, bounded permission for a particular
// harness/account/model. MaxCredits is enforced by one shared parent key in
// LPBS, not reset by minting a new session key. No credential belongs here.
type CodingBackupConsent struct {
	Harness    string    `json:"harness"`
	Account    string    `json:"business_account_id"`
	Model      string    `json:"model"`
	MaxCredits int64     `json:"max_credits"`
	ExpiresAt  time.Time `json:"expires_at"`
	Automatic  bool      `json:"automatic"`
}

func (c CodingBackupConsent) Valid(now time.Time) bool {
	return SupportsModelSource(c.Harness) && c.Account != "" && c.Model != "" && c.MaxCredits > 0 &&
		now.Before(c.ExpiresAt) && c.ExpiresAt.Sub(now) <= 24*time.Hour
}

// CodingBackupDecision never grants eligibility. The caller still asks the
// door for the exact harness and model before issuing a child credential.
// A live terminal always requires a fresh user restart action, even after
// automatic backup was approved for new sessions and resumable runs.
func CodingBackupDecision(now time.Time, limit CodingLimit, consent *CodingBackupConsent, mode string, restartConfirmed bool) string {
	if !limit.Exhausted(now) {
		return "subscription"
	}
	if !SupportsModelSource(limit.Harness) {
		return "different_harness_required"
	}
	if consent == nil || consent.Harness != limit.Harness || !consent.Valid(now) {
		return "consent_required"
	}
	switch mode {
	case "live":
		if !restartConfirmed {
			return "restart_required"
		}
	case "new", "unattended":
		if !consent.Automatic {
			return "consent_required"
		}
	default:
		return "consent_required"
	}
	return "credits"
}

// DetectCodingLimit reads only explicit structured provider error envelopes.
// Generic 429/rate_limit_error and assistant/tool text are intentionally absent.
// The journey fixtures for these shapes are labelled synthetic until real
// provider captures can be supplied by the operator.
func DetectCodingLimit(harness string, data []byte, observed time.Time) (CodingLimit, bool) {
	var e struct {
		Type  string `json:"type"`
		Error struct {
			Code string `json:"code"`
			Type string `json:"type"`
			Data struct {
				ResponseBody string `json:"responseBody"`
			} `json:"data"`
		} `json:"error"`
		Payload struct {
			Type string `json:"type"`
			Code string `json:"code"`
		} `json:"payload"`
		Rate struct {
			Status      string  `json:"status"`
			Utilization float64 `json:"utilization"`
			ResetsAt    int64   `json:"resetsAt"`
		} `json:"rate_limit_info"`
	}
	if json.Unmarshal(data, &e) != nil {
		return CodingLimit{}, false
	}
	code := ""
	switch harness {
	case "codex":
		if e.Type == "error" || e.Type == "turn.failed" {
			code = e.Error.Code
		}
		if e.Type == "event_msg" && e.Payload.Type == "error" {
			code = e.Payload.Code
		}
	case "opencode":
		if e.Type == "error" {
			code = e.Error.Code
			if e.Error.Data.ResponseBody != "" {
				var body struct {
					Error struct {
						Code string `json:"code"`
					} `json:"error"`
				}
				if json.Unmarshal([]byte(e.Error.Data.ResponseBody), &body) == nil {
					code = body.Error.Code
				}
			}
		}
	case "claude":
		if e.Type == "rate_limit_event" && e.Rate.Status == "rejected" && e.Rate.Utilization >= 1 {
			limit := CodingLimit{Harness: harness, Kind: "provider_limit", ObservedAt: observed}
			if e.Rate.ResetsAt > 0 {
				limit.ResetsAt = time.Unix(e.Rate.ResetsAt, 0)
			}
			return limit, true
		}
	default:
		return CodingLimit{}, false
	}
	if code != "usage_limit_reached" && code != "insufficient_quota" && code != "quota_exhausted" {
		return CodingLimit{}, false
	}
	return CodingLimit{Harness: harness, Kind: "provider_limit", ObservedAt: observed}, true
}
