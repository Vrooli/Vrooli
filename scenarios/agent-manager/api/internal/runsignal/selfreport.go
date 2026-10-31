package runsignal

import (
	_ "embed"
	"encoding/json"
	"regexp"
	"slices"
	"strings"
	"sync"

	"agent-manager/internal/domain"
)

// SelfReportClassifierVersion pins deterministic message-pattern extraction.
const (
	SelfReportClassifierVersion = "self-report.v4"
	selfReportSpanLimit         = 8
)

//go:embed testdata/classification/self-report-rule-pack.json
var embeddedSelfReportRulePack []byte

type SelfReportSpan struct {
	ClassifierVersion string `json:"classifierVersion"`
	EventID           string `json:"eventId"`
	RuleID            string `json:"ruleId"`
	CauseScope        string `json:"causeScope"`
	StartOffset       int    `json:"startOffset"`
	EndOffset         int    `json:"endOffset"`
	Text              string `json:"text"`
	Role              string `json:"role"`
	SpanCapped        bool   `json:"spanCapped,omitempty"`
}

type (
	selfReportRulePack struct {
		Version string           `json:"version"`
		Owner   string           `json:"owner"`
		Rules   []selfReportRule `json:"rules"`
	}
	selfReportRule struct {
		ID         string   `json:"id"`
		CauseScope string   `json:"causeScope"`
		Phrases    []string `json:"phrases"`
		// patterns match each phrase case-insensitively in the original text,
		// so offsets index it (lowercasing can change a string's byte length).
		patterns []*regexp.Regexp
	}
)

type (
	SelfReportDetector interface {
		Identifier() string
		ClassifierVersion() string
		CauseScope() string
		Detect(*domain.RunEvent) []SelfReportSpan
	}
	selfReportDetector struct{ rule selfReportRule }
)

func (d selfReportDetector) Identifier() string        { return d.rule.ID }
func (d selfReportDetector) ClassifierVersion() string { return SelfReportClassifierVersion }
func (d selfReportDetector) CauseScope() string        { return d.rule.CauseScope }
func (d selfReportDetector) Detect(event *domain.RunEvent) []SelfReportSpan {
	message, ok := event.Data.(*domain.MessageEventData)
	if !ok || !strings.EqualFold(message.Role, "assistant") {
		return nil
	}
	spans := make([]SelfReportSpan, 0)
	for p, pattern := range d.rule.patterns {
		matches := pattern.FindAllStringIndex(message.Content, selfReportSpanLimit+1)
		for i, match := range matches {
			at, end := match[0], match[1]
			spans = append(spans, SelfReportSpan{ClassifierVersion: d.ClassifierVersion(), EventID: event.ID.String(), RuleID: d.Identifier(), CauseScope: d.CauseScope(), StartOffset: at, EndOffset: end, Text: redact(message.Content[at:end]), Role: strings.ToLower(message.Role)})
			if len(spans) >= selfReportSpanLimit {
				// Capped when anything is left unreported: a further match of
				// this phrase or of any later one.
				spans[len(spans)-1].SpanCapped = i+1 < len(matches) || slices.ContainsFunc(d.rule.patterns[p+1:], func(later *regexp.Regexp) bool { return later.MatchString(message.Content) })
				return spans
			}
		}
	}
	return spans
}

var (
	selfReportPackOnce sync.Once
	selfReportPack     selfReportRulePack
)

func SelfReportDetectors() []SelfReportDetector {
	selfReportPackOnce.Do(func() {
		if err := json.Unmarshal(embeddedSelfReportRulePack, &selfReportPack); err != nil {
			panic("invalid embedded self-report rule pack: " + err.Error())
		}
		if selfReportPack.Version != SelfReportClassifierVersion || selfReportPack.Owner == "" {
			panic("invalid embedded self-report rule pack metadata")
		}
		for i, rule := range selfReportPack.Rules {
			if rule.ID == "" || rule.CauseScope == "" || len(rule.Phrases) == 0 {
				panic("invalid embedded self-report rule")
			}
			for _, phrase := range rule.Phrases {
				selfReportPack.Rules[i].patterns = append(selfReportPack.Rules[i].patterns, regexp.MustCompile("(?i)"+regexp.QuoteMeta(phrase)))
			}
		}
	})
	detectors := make([]SelfReportDetector, 0, len(selfReportPack.Rules))
	for _, rule := range selfReportPack.Rules {
		detectors = append(detectors, selfReportDetector{rule: rule})
	}
	return detectors
}

// DeriveSelfReportSpans examines assistant-authored messages only. The owned,
// versioned rule pack is embedded as data; classifiers remain network-free.
func DeriveSelfReportSpans(events []*domain.RunEvent) []SelfReportSpan {
	spans := []SelfReportSpan{}
	detectors := SelfReportDetectors()
	for _, event := range events {
		for _, detector := range detectors {
			spans = append(spans, detector.Detect(event)...)
		}
	}
	return spans
}
