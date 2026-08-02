package runsignal

import (
	_ "embed"
	"encoding/json"
	"strings"
	"sync"

	"agent-manager/internal/domain"
)

// SelfReportClassifierVersion pins deterministic message-pattern extraction.
const SelfReportClassifierVersion = "self-report.v3"
const selfReportSpanLimit = 8

//go:embed testdata/classification/self-report-rule-pack.json
var embeddedSelfReportRulePack []byte

type SelfReportSpan struct {
	ClassifierVersion  string `json:"classifierVersion"`
	EventID            string `json:"eventId"`
	RuleID             string `json:"ruleId"`
	CauseScope         string `json:"causeScope"`
	StartOffset        int    `json:"startOffset"`
	EndOffset          int    `json:"endOffset"`
	Text               string `json:"text"`
	Role               string `json:"role"`
	OperatorCorrection bool   `json:"operatorCorrection,omitempty"`
	SpanCapped         bool   `json:"spanCapped,omitempty"`
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
	if !ok || (d.rule.ID == "operator-correction" && !strings.EqualFold(message.Role, "user")) || (d.rule.ID != "operator-correction" && !strings.EqualFold(message.Role, "assistant")) {
		return nil
	}
	lower := strings.ToLower(message.Content)
	spans := make([]SelfReportSpan, 0)
	for _, phrase := range d.rule.Phrases {
		needle := strings.ToLower(phrase)
		for start := 0; start < len(lower); {
			at := strings.Index(lower[start:], needle)
			if at < 0 {
				break
			}
			at += start
			end := at + len(phrase)
			spans = append(spans, SelfReportSpan{ClassifierVersion: d.ClassifierVersion(), EventID: event.ID.String(), RuleID: d.Identifier(), CauseScope: d.CauseScope(), StartOffset: at, EndOffset: end, Text: redact(message.Content[at:end]), Role: strings.ToLower(message.Role), OperatorCorrection: d.rule.ID == "operator-correction"})
			if len(spans) >= selfReportSpanLimit {
				spans[len(spans)-1].SpanCapped = strings.Index(lower[end:], needle) >= 0
				return spans
			}
			start = end
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
		for _, rule := range selfReportPack.Rules {
			if rule.ID == "" || rule.CauseScope == "" || len(rule.Phrases) == 0 {
				panic("invalid embedded self-report rule")
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
	for _, event := range events {
		for _, detector := range SelfReportDetectors() {
			spans = append(spans, detector.Detect(event)...)
		}
	}
	return spans
}
