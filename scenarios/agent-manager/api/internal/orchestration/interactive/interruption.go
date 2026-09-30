package interactive

import "strings"

// ResumableInterruption is a provider/runtime condition that stopped a turn
// but can be retried without creating a new Web Console session.
type ResumableInterruption struct {
	Kind    string
	Message string
}

// ClassifyScreenInterruption recognizes stable, user-visible provider failure
// shapes. Keep this table additive and provider-neutral: harness-specific
// screen readers can call it after obtaining their plain-text screen.
func ClassifyScreenInterruption(screen string) (ResumableInterruption, bool) {
	lower := strings.ToLower(screen)
	markers := []struct {
		kind   string
		marker string
	}{
		{"model_capacity", "selected model is at capacity"},
		{"model_capacity", "model is at capacity"},
		{"model_capacity", "temporarily at capacity"},
		{"rate_limit", "rate limit exceeded"},
		{"rate_limit", "too many requests"},
		{"provider_unavailable", "service unavailable"},
		{"provider_unavailable", "temporarily unavailable"},
	}
	for _, candidate := range markers {
		if strings.Contains(lower, candidate.marker) {
			return ResumableInterruption{Kind: candidate.kind, Message: candidate.marker}, true
		}
	}
	return ResumableInterruption{}, false
}
