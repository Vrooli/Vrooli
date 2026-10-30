package runsignal

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"agent-manager/internal/domain"

	"github.com/google/uuid"
)

// ErrIncompleteLabelCoverage distinguishes a missing detector label from an unavailable corpus.
var ErrIncompleteLabelCoverage = errors.New("incomplete labelled detector coverage")

// DetectorAccuracy is one detector's score on the committed structural corpus
// (testdata/classification), published by the agent-conformance response. The
// per-pattern precision gate over real runs is testdata/precision.
type DetectorAccuracy struct {
	ID        string  `json:"id"`
	Precision float64 `json:"precision"`
	Recall    float64 `json:"recall"`
	Threshold float64 `json:"threshold"`
}

type (
	accuracyLabels struct {
		Thresholds map[string]float64 `json:"thresholds"`
		Expected   []string           `json:"expected"`
		Cases      []accuracyCase     `json:"cases"`
	}
	accuracyCase struct {
		Name     string         `json:"name"`
		Expected []string       `json:"expected"`
		Facts    []accuracyFact `json:"facts"`
		Events   []accuracyItem `json:"events"`
	}
	accuracyFact struct {
		CallEventID      string `json:"callEventId"`
		ResultEventID    string `json:"resultEventId"`
		Outcome          string `json:"outcome"`
		Executable       string `json:"executable"`
		Ownership        string `json:"ownership"`
		CommandPath      string `json:"commandPath"`
		Capability       string `json:"capability"`
		IntentClass      string `json:"intentClass"`
		Fingerprint      string `json:"fingerprint"`
		HelpRecovery     bool   `json:"helpRecovery"`
		FailureSignature string `json:"failureSignature"`
	}
	accuracyItem struct {
		ID             string              `json:"id"`
		AtMilliseconds int64               `json:"atMilliseconds"`
		Kind           string              `json:"kind"`
		ToolName       string              `json:"toolName"`
		Input          map[string]any      `json:"input"`
		Files          []map[string]string `json:"files"`
		Content        string              `json:"content"`
		Terminal       bool                `json:"terminal"`
	}
	detectorScore struct{ truePositive, falsePositive, falseNegative int }
)

func (s detectorScore) precision() float64 {
	if s.truePositive+s.falsePositive == 0 {
		return 1
	}
	return float64(s.truePositive) / float64(s.truePositive+s.falsePositive)
}

func (s detectorScore) recall() float64 {
	if s.truePositive+s.falseNegative == 0 {
		return 1
	}
	return float64(s.truePositive) / float64(s.truePositive+s.falseNegative)
}

// ClassificationAccuracy scores every shipped detector against the committed
// labelled corpus under repoRoot. Every shipped detector must be labelled.
func ClassificationAccuracy(repoRoot string) ([]DetectorAccuracy, error) {
	body, err := os.ReadFile(filepath.Join(repoRoot, "scenarios", "agent-manager", "api", "internal", "runsignal", "testdata", "classification", "all-detectors.labels.json"))
	if err != nil {
		return nil, err
	}
	var labels accuracyLabels
	if err := json.Unmarshal(body, &labels); err != nil {
		return nil, err
	}
	labelled := map[string]bool{}
	for _, id := range labels.Expected {
		labelled[id] = true
	}
	shipped := make([]string, 0)
	for _, d := range EpisodeDetectors() {
		shipped = append(shipped, "episode/"+d.Identifier())
	}
	for _, d := range SelfReportDetectors() {
		shipped = append(shipped, "span/"+d.Identifier())
	}
	sort.Strings(shipped)
	for _, id := range shipped {
		if !labelled[id] {
			return nil, fmt.Errorf("%w: shipped detector %q has no labelled coverage", ErrIncompleteLabelCoverage, id)
		}
	}
	scores := map[string]detectorScore{}
	for _, c := range labels.Cases {
		want := map[string]bool{}
		for _, id := range c.Expected {
			want[id] = true
		}
		got := c.detect()
		for _, id := range shipped {
			s := scores[id]
			switch {
			case want[id] && got[id]:
				s.truePositive++
			case got[id]:
				s.falsePositive++
			case want[id]:
				s.falseNegative++
			}
			scores[id] = s
		}
	}
	out := make([]DetectorAccuracy, 0, len(shipped))
	for _, id := range shipped {
		threshold, ok := labels.Thresholds[id]
		if !ok {
			return nil, fmt.Errorf("%w: shipped detector %q has no threshold", ErrIncompleteLabelCoverage, id)
		}
		out = append(out, DetectorAccuracy{ID: id, Precision: scores[id].precision(), Recall: scores[id].recall(), Threshold: threshold})
	}
	return out, nil
}

// detect runs the production detectors over a labelled case.
func (c accuracyCase) detect() map[string]bool {
	now := time.Date(2026, 7, 31, 0, 0, 0, 0, time.UTC)
	ids := map[string]uuid.UUID{}
	events := make([]*domain.RunEvent, 0, len(c.Events))
	for _, item := range c.Events {
		id := uuid.NewSHA1(uuid.NameSpaceURL, []byte(c.Name+":"+item.ID))
		ids[item.ID] = id
		event := &domain.RunEvent{ID: id, Timestamp: now.Add(time.Duration(item.AtMilliseconds) * time.Millisecond)}
		switch item.Kind {
		case "assistant_message":
			event.EventType, event.Data = domain.EventTypeMessage, &domain.MessageEventData{Role: "assistant", Content: item.Content, Terminal: item.Terminal}
		case "user_message":
			event.EventType, event.Data = domain.EventTypeMessage, &domain.MessageEventData{Role: "user", Content: item.Content}
		case "tool_call":
			input := item.Input
			if input == nil {
				input = map[string]any{}
			}
			if len(item.Files) > 0 {
				input["files"] = item.Files
			}
			event.EventType, event.Data = domain.EventTypeToolCall, &domain.ToolCallEventData{ToolName: item.ToolName, Input: input}
		case "tool_result":
			event.EventType, event.Data = domain.EventTypeToolResult, &domain.ToolResultEventData{Success: true}
		}
		events = append(events, event)
	}
	facts := make([]InvocationFact, 0, len(c.Facts))
	for _, f := range c.Facts {
		facts = append(facts, InvocationFact{CallEventID: ids[f.CallEventID].String(), ResultEventID: ids[f.ResultEventID].String(), Outcome: f.Outcome, Ownership: f.Ownership, Executable: f.Executable, CommandPath: f.CommandPath, Capability: f.Capability, IntentClass: f.IntentClass, Fingerprint: f.Fingerprint, HelpRecovery: f.HelpRecovery, FailureSignature: f.FailureSignature})
	}
	actual := map[string]bool{}
	for _, episode := range DeriveEpisodes(facts, events) {
		actual["episode/"+episode.Pattern] = true
	}
	for _, span := range DeriveSelfReportSpans(events) {
		actual["span/"+span.RuleID] = true
	}
	return actual
}
