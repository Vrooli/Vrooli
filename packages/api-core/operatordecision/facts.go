package operatordecision

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// Facts is the domain-neutral description of an operator decision: what the
// operator is asked, the menu of answers, which is recommended, whether acting is
// reversible, and the correlation the resolution carries back verbatim. It is the
// producer side of the notification-hub ask contract, ported from
// goalhome.Decision (its Validate + RequestPayload). The hub reads these fields
// to render the Ask; nothing scenario-specific leaks into the hub.
//
// Facts.Payload() produces the `decision_requested` event payload map; a producer
// wraps it with BuildEvent and publishes it via a Publisher.
type Facts struct {
	// Subject is the short notification title (e.g. "Run abc123 needs review").
	Subject string
	// Question is the full prompt shown to the operator.
	Question string
	// Options is the answer menu; >=2 required. Key is what the handler receives,
	// Label what the operator sees.
	Options []Option
	// Recommended is the option key the producer suggests; required and must be one
	// of Options.
	Recommended string
	// RecommendationReason optionally explains the recommendation.
	RecommendationReason string
	// DefaultAnswer, when set, is applied by the hub at the deadline with no answer.
	// It is allowed ONLY when Reversible is true and must be one of Options.
	DefaultAnswer string
	// Reversible declares that acting on the decision can be undone; it is the gate
	// for allowing a silent DefaultAnswer.
	Reversible bool
	// Urgency is "", "low", "normal", "high" or "critical".
	Urgency string
	// Note is optional free text carried to the operator.
	Note string
	// ContextURL optionally deep-links the operator to a review surface. A producer
	// resolves this (e.g. from its web-console base) BEFORE building Facts.
	ContextURL string
	// Deadline, when set, bounds how long the hub waits before applying the default.
	Deadline time.Time
	// Correlation is returned verbatim on notification_hub.ask.resolved.v1, so the
	// resolver can find the durable record the ask belongs to.
	Correlation map[string]string
}

// Option is one answer in a decision's menu. Key is the stable machine key the
// resolution carries; Label is the human-facing text.
type Option struct {
	Key   string
	Label string
}

// Facts limits mirror the notification-hub ask contract, so a decision this
// package accepts is never refused by the hub. They match goalhome's Decision
// limits.
const (
	MaxQuestion    = 600
	MaxReason      = 300
	MaxOptions     = 8
	MaxOptionLabel = 200
)

var (
	factsKeyRule = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{0,63}$`)
	factsUrgency = map[string]bool{"": true, "low": true, "normal": true, "high": true, "critical": true}
)

// Validate checks Facts against the notification-hub ask contract: a non-empty
// bounded question, 2-8 options with unique valid keys, a recommended option that
// exists, and a default that is allowed only for a reversible decision and is one
// of the options. It ports goalhome.Decision.Validate, minus the decision-id rule
// (correlation here is generic, not a goal-home-scoped id).
func (f Facts) Validate() error {
	switch {
	case strings.TrimSpace(f.Question) == "" || len(f.Question) > MaxQuestion:
		return fmt.Errorf("the question is required and at most %d bytes", MaxQuestion)
	case len(f.Options) < 2 || len(f.Options) > MaxOptions:
		return fmt.Errorf("a decision needs 2-%d options", MaxOptions)
	case len(f.RecommendationReason) > MaxReason:
		return fmt.Errorf("the recommendation reason is at most %d bytes", MaxReason)
	case !factsUrgency[f.Urgency]:
		return fmt.Errorf("urgency must be low, normal, high or critical")
	}
	seen := map[string]bool{}
	for _, option := range f.Options {
		if !factsKeyRule.MatchString(option.Key) {
			return fmt.Errorf("option key %q must be 1-64 letters, digits, '.', '_', ':' or '-'", option.Key)
		}
		if seen[option.Key] {
			return fmt.Errorf("option key %q is repeated", option.Key)
		}
		if len(option.Label) > MaxOptionLabel {
			return fmt.Errorf("option %q label exceeds %d bytes", option.Key, MaxOptionLabel)
		}
		seen[option.Key] = true
	}
	if !seen[f.Recommended] {
		return fmt.Errorf("the recommended option %q is not one of the options", f.Recommended)
	}
	if f.DefaultAnswer != "" {
		if !seen[f.DefaultAnswer] {
			return fmt.Errorf("the default %q is not one of the options", f.DefaultAnswer)
		}
		if !f.Reversible {
			return fmt.Errorf("a default is allowed only for a reversible decision")
		}
	}
	if err := validateContextURL(f.ContextURL); err != nil {
		return err
	}
	return nil
}

// validateContextURL refuses a context_url whose scheme could execute or embed
// code when rendered as an operator-clickable href. Only an empty value, an
// http(s) absolute URL, or a root-relative path ("/...") is allowed; a
// javascript:, data:, vbscript:, file:, etc. URL is rejected — otherwise a
// producer-supplied link flows unmodified to the inbox and becomes an XSS /
// phishing vector.
func validateContextURL(raw string) error {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	// A root-relative path is safe (same-origin, no scheme). Reject the
	// scheme-relative "//host" form — it inherits the page scheme and points
	// off-origin, which is the phishing case a bare "/path" avoids.
	if strings.HasPrefix(raw, "/") {
		if strings.HasPrefix(raw, "//") {
			return fmt.Errorf("context_url %q must be http(s) or a root-relative path, not a scheme-relative URL", raw)
		}
		return nil
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("context_url %q is not a valid URL: %w", raw, err)
	}
	switch strings.ToLower(parsed.Scheme) {
	case "http", "https":
		return nil
	default:
		return fmt.Errorf("context_url %q must be empty, http(s), or a root-relative path (scheme %q is not allowed)", raw, parsed.Scheme)
	}
}

// Label returns an option's label, or the key when the option has none. It
// mirrors goalhome.Decision.Label.
func (f Facts) Label(key string) string {
	for _, option := range f.Options {
		if option.Key == key && option.Label != "" {
			return option.Label
		}
	}
	return key
}

// Payload is the decision_requested event payload: facts only, the shape
// notification-hub's decision intake reads. Correlation comes back verbatim on
// notification_hub.ask.resolved.v1. It ports goalhome.Decision.RequestPayload;
// the Subject is a field here rather than a separate argument.
func (f Facts) Payload() map[string]any {
	options := make([]any, 0, len(f.Options))
	for _, option := range f.Options {
		options = append(options, map[string]any{"key": option.Key, "label": f.Label(option.Key)})
	}
	payload := map[string]any{
		"subject":     f.Subject,
		"question":    f.Question,
		"options":     options,
		"recommended": f.Recommended,
		"reversible":  f.Reversible,
		"urgency":     f.Urgency,
	}
	// recommendation_reason, default_answer, context_url and note are omitted when
	// empty so a decision that carries none of them (a run approval, an
	// authorization ask) emits the same bytes it did before this substrate existed
	// — no silent default_answer and no empty context_url key. A producer that sets
	// them gets them verbatim. notification-hub tolerates either absence or an empty
	// string, so a producer that WANTS the always-present shape (the effort decision
	// renderer in goalhome) keeps building its own payload.
	if f.RecommendationReason != "" {
		payload["recommendation_reason"] = f.RecommendationReason
	}
	if f.DefaultAnswer != "" {
		payload["default_answer"] = f.DefaultAnswer
	}
	if f.ContextURL != "" {
		payload["context_url"] = f.ContextURL
	}
	if f.Note != "" {
		payload["note"] = f.Note
	}
	if !f.Deadline.IsZero() {
		payload["deadline"] = f.Deadline.UTC().Format(time.RFC3339)
	}
	if len(f.Correlation) > 0 {
		values := make(map[string]any, len(f.Correlation))
		for key, value := range f.Correlation {
			values[key] = value
		}
		payload["correlation"] = values
	}
	return payload
}
