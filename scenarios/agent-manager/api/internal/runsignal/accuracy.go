package runsignal

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"agent-manager/internal/domain"

	"github.com/google/uuid"
)

// ErrIncompleteLabelCoverage distinguishes a missing detector label from an unavailable corpus.
var ErrIncompleteLabelCoverage = errors.New("incomplete labelled detector coverage")

type DetectorAccuracy struct {
	ID        string  `json:"id"`
	Precision float64 `json:"precision"`
	Recall    float64 `json:"recall"`
	Threshold float64 `json:"threshold"`
}

// CorpusCase is the minimum provenance required to split labelled cases
// without leaking one source run across development and held-out sets.
// Callers must provide the source-run identity they actually observed; a
// generated case name is not an acceptable substitute for lineage.
type CorpusCase struct {
	Name      string `json:"name"`
	SourceRun string `json:"sourceRun"`
}

// CorpusSplit is deterministic and preserves input order within each split.
// The split decision is made once per source run, so all cases from one run
// remain together even when the run contributed several windows.
type CorpusSplit struct {
	Development []CorpusCase
	HeldOut     []CorpusCase
}

// SplitCorpusBySourceRun partitions labelled cases by source-run lineage.
// heldOutSourceRuns is an explicit review decision, rather than a random
// case-level sampler. This prevents duplicated windows from leaking tuning
// information into the held-out cohort.
func SplitCorpusBySourceRun(cases []CorpusCase, heldOutSourceRuns map[string]struct{}) (CorpusSplit, error) {
	if len(cases) == 0 {
		return CorpusSplit{}, errors.New("classification corpus is empty")
	}
	if len(heldOutSourceRuns) == 0 {
		return CorpusSplit{}, errors.New("held-out source-run set is empty")
	}
	heldOut := make(map[string]struct{}, len(heldOutSourceRuns))
	for lineage := range heldOutSourceRuns {
		lineage = strings.TrimSpace(lineage)
		if lineage == "" {
			return CorpusSplit{}, errors.New("held-out source-run set contains an empty lineage")
		}
		heldOut[lineage] = struct{}{}
	}
	seenNames := make(map[string]struct{}, len(cases))
	seenLineages := make(map[string]struct{}, len(cases))
	var split CorpusSplit
	for _, item := range cases {
		name := strings.TrimSpace(item.Name)
		lineage := strings.TrimSpace(item.SourceRun)
		if name == "" || lineage == "" {
			return CorpusSplit{}, fmt.Errorf("corpus case requires name and source-run lineage")
		}
		if _, exists := seenNames[name]; exists {
			return CorpusSplit{}, fmt.Errorf("duplicate corpus case %q", name)
		}
		seenNames[name] = struct{}{}
		seenLineages[lineage] = struct{}{}
		item.Name, item.SourceRun = name, lineage
		if _, heldOut := heldOut[lineage]; heldOut {
			split.HeldOut = append(split.HeldOut, item)
		} else {
			split.Development = append(split.Development, item)
		}
	}
	for lineage := range heldOut {
		if _, observed := seenLineages[lineage]; !observed {
			return CorpusSplit{}, fmt.Errorf("held-out source-run %q has no labelled cases", lineage)
		}
	}
	if len(split.Development) == 0 || len(split.HeldOut) == 0 {
		return CorpusSplit{}, errors.New("corpus split must contain development and held-out cases")
	}
	return split, nil
}

type accuracyLabels struct {
	Thresholds map[string]float64 `json:"thresholds"`
	Expected   []string           `json:"expected"`
	SourceRuns []string           `json:"sourceRuns"`
	Cases      []accuracyCase     `json:"cases"`
}

type accuracyCase struct {
	Name     string          `json:"name"`
	Reason   string          `json:"reason"`
	Expected []string        `json:"expected"`
	Facts    []accuracyFact  `json:"facts"`
	Events   []accuracyEvent `json:"events"`
}

type accuracyFact struct {
	CallEventID   string `json:"callEventId"`
	ResultEventID string `json:"resultEventId"`
	Outcome       string `json:"outcome"`
	Ownership     string `json:"ownership"`
	Executable    string `json:"executable"`
	CommandPath   string `json:"commandPath"`
	Fingerprint   string `json:"fingerprint"`
	Capability    string `json:"capability"`
	IntentClass   string `json:"intentClass"`
	HelpRecovery  bool   `json:"helpRecovery"`
}

type accuracyEvent struct {
	ID             string              `json:"id"`
	AtMilliseconds int64               `json:"atMilliseconds"`
	Kind           string              `json:"kind"`
	Role           string              `json:"role"`
	Content        string              `json:"content"`
	Terminal       bool                `json:"terminal"`
	ToolName       string              `json:"toolName"`
	Input          map[string]any      `json:"input"`
	Files          []map[string]string `json:"files"`
}

type detectorScore struct{ truePositive, falsePositive, falseNegative int }

func (s detectorScore) precision() float64 {
	return ratio(s.truePositive, s.falsePositive)
}

func (s detectorScore) recall() float64 {
	return ratio(s.truePositive, s.falseNegative)
}

func ratio(truePositive, misses int) float64 {
	if total := truePositive + misses; total > 0 {
		return float64(truePositive) / float64(total)
	}
	return 0
}

// ClassificationAccuracy is the shared Go gate and Test Genie scorer.
func ClassificationAccuracy(repoRoot string) ([]DetectorAccuracy, error) {
	labels, err := readAccuracyLabels(filepath.Join(repoRoot, "scenarios", "agent-manager", "api", "internal", "runsignal", "testdata", "classification", "all-detectors.labels.json"))
	if err != nil {
		return nil, err
	}
	if !sameDetectorSet(labels.Expected, shippedDetectorIDs()) {
		return nil, fmt.Errorf("%w: labelled detectors %v do not cover shipped detectors %v", ErrIncompleteLabelCoverage, labels.Expected, shippedDetectorIDs())
	}
	scores, err := scoreDetectors(labels)
	if err != nil {
		return nil, err
	}
	result := make([]DetectorAccuracy, 0, len(labels.Thresholds))
	for id, threshold := range labels.Thresholds {
		score, ok := scores[id]
		if !ok {
			return nil, fmt.Errorf("%w: shipped detector %q has no labelled coverage", ErrIncompleteLabelCoverage, id)
		}
		result = append(result, DetectorAccuracy{ID: id, Precision: score.precision(), Recall: score.recall(), Threshold: threshold})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result, nil
}

func readAccuracyLabels(path string) (accuracyLabels, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return accuracyLabels{}, err
	}
	var labels accuracyLabels
	if err := json.Unmarshal(body, &labels); err != nil {
		return accuracyLabels{}, err
	}
	if len(labels.Expected) == 0 || len(labels.Expected) != len(labels.Thresholds) || len(labels.SourceRuns) == 0 || len(labels.Cases) == 0 {
		return accuracyLabels{}, fmt.Errorf("invalid labels: %+v", labels)
	}
	for _, fixture := range labels.Cases {
		if fixture.Name == "" || fixture.Reason == "" {
			return accuracyLabels{}, fmt.Errorf("classification fixture %q has no labeller reason", fixture.Name)
		}
	}
	return labels, nil
}

func shippedDetectorIDs() []string {
	ids := make([]string, 0, len(EpisodeDetectors())+len(SelfReportDetectors()))
	for _, d := range EpisodeDetectors() {
		ids = append(ids, "episode/"+d.Identifier())
	}
	for _, d := range SelfReportDetectors() {
		ids = append(ids, "span/"+d.Identifier())
	}
	sort.Strings(ids)
	return ids
}

func sameDetectorSet(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	seen := make(map[string]struct{}, len(left))
	for _, id := range left {
		seen[id] = struct{}{}
	}
	for _, id := range right {
		if _, ok := seen[id]; !ok {
			return false
		}
	}
	return true
}

func scoreDetectors(labels accuracyLabels) (map[string]detectorScore, error) {
	known := map[string]bool{}
	for _, id := range labels.Expected {
		known[id] = true
	}
	scores := map[string]detectorScore{}
	for _, fixture := range labels.Cases {
		if fixture.Name == "" {
			return nil, fmt.Errorf("classification fixture has no name")
		}
		expected := map[string]bool{}
		for _, id := range fixture.Expected {
			if !known[id] {
				return nil, fmt.Errorf("fixture %q labels unknown detector %q", fixture.Name, id)
			}
			expected[id] = true
		}
		actual := detectorOutput(fixture)
		for _, id := range labels.Expected {
			score := scores[id]
			switch {
			case expected[id] && actual[id]:
				score.truePositive++
			case expected[id]:
				score.falseNegative++
			case actual[id]:
				score.falsePositive++
			}
			scores[id] = score
		}
	}
	return scores, nil
}

func detectorOutput(fixture accuracyCase) map[string]bool {
	now := time.Date(2026, 7, 31, 0, 0, 0, 0, time.UTC)
	ids := map[string]uuid.UUID{}
	events := make([]*domain.RunEvent, 0, len(fixture.Events))
	for _, item := range fixture.Events {
		id := uuid.NewSHA1(uuid.NameSpaceURL, []byte(fixture.Name+":"+item.ID))
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
			event.Data = &domain.ToolResultEventData{Success: true}
		}
		events = append(events, event)
	}
	facts := make([]InvocationFact, 0, len(fixture.Facts))
	for _, item := range fixture.Facts {
		facts = append(facts, InvocationFact{CallEventID: ids[item.CallEventID].String(), ResultEventID: ids[item.ResultEventID].String(), Outcome: item.Outcome, Ownership: item.Ownership, Executable: item.Executable, CommandPath: item.CommandPath, Capability: item.Capability, IntentClass: item.IntentClass, Fingerprint: item.Fingerprint, HelpRecovery: item.HelpRecovery})
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
