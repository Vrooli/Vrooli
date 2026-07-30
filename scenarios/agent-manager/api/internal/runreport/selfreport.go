package runreport

import (
	"strings"

	"agent-manager/internal/domain"
)

// SelfReportClassifierVersion pins deterministic message-pattern extraction.
const SelfReportClassifierVersion = "self-report.v1"

type SelfReportSpan struct {
	ClassifierVersion string `json:"classifierVersion"`
	EventID           string `json:"eventId"`
	RuleID            string `json:"ruleId"`
	CauseScope        string `json:"causeScope"`
	StartOffset       int    `json:"startOffset"`
	EndOffset         int    `json:"endOffset"`
	Text              string `json:"text"`
}

var selfReportRules = []struct{ id, pattern, scope string }{
	{"cannot-find", "cannot find", "toolchain"},
	{"repeated-failure", "keeps failing", "run-execution"},
	{"strategy-change", "instead i will", "recurring-workaround"},
	{"blocked", "i am blocked", "run-execution"},
	{"permission-request", "need permission", "prompt-team-agent-storage"},
}

// DeriveSelfReportSpans examines assistant-authored messages only. The rule
// table is deliberately literal and versioned so no model or HTTP client is
// needed to make a transcript investigable.
func DeriveSelfReportSpans(events []*domain.RunEvent) []SelfReportSpan {
	spans := []SelfReportSpan{}
	for _, event := range events {
		message, ok := event.Data.(*domain.MessageEventData)
		if !ok || !strings.EqualFold(message.Role, "assistant") {
			continue
		}
		lower := strings.ToLower(message.Content)
		for _, rule := range selfReportRules {
			at := strings.Index(lower, rule.pattern)
			if at < 0 {
				continue
			}
			end := at + len(rule.pattern)
			spans = append(spans, SelfReportSpan{ClassifierVersion: SelfReportClassifierVersion, EventID: event.ID.String(), RuleID: rule.id, CauseScope: rule.scope, StartOffset: at, EndOffset: end, Text: redact(message.Content[at:end])})
		}
	}
	return spans
}
