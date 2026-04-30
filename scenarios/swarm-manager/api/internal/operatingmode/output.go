package operatingmode

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

const resultEnvelopeKey = "operating_mode_result"

type PhaseResult struct {
	Artifacts    []ArtifactResult `json:"artifacts,omitempty"`
	Handoff      *Handoff         `json:"handoff,omitempty"`
	Handoffs     []Handoff        `json:"handoffs,omitempty"`
	Readiness    *ReadinessReport `json:"readiness,omitempty"`
	Progress     *ProgressState   `json:"progress,omitempty"`
	Verdict      string           `json:"verdict,omitempty"`
	ReplanNeeded bool             `json:"replan_needed,omitempty"`
	BacklogSync  *BacklogSyncPlan `json:"backlog_sync,omitempty"`
}

type ArtifactResult struct {
	Path        string `json:"path"`
	Content     string `json:"content"`
	ContentType string `json:"content_type,omitempty"`
}

type BacklogSyncPlan struct {
	CompletedItems []string        `json:"completed_items,omitempty"`
	CreatedItems   []string        `json:"created_items,omitempty"`
	UpdatedItems   []string        `json:"updated_items,omitempty"`
	Proposal       json.RawMessage `json:"proposal,omitempty"`
	Rationale      string          `json:"rationale,omitempty"`
}

var fencedJSONBlockRE = regexp.MustCompile("(?s)```(?:json)?\\s*(\\{.*?\\})\\s*```")

func ParsePhaseResult(output string) (PhaseResult, bool, error) {
	trimmed := strings.TrimSpace(output)
	if trimmed == "" {
		return PhaseResult{}, false, nil
	}
	candidates := []string{trimmed}
	for _, match := range fencedJSONBlockRE.FindAllStringSubmatch(trimmed, -1) {
		if len(match) > 1 {
			candidates = append(candidates, strings.TrimSpace(match[1]))
		}
	}
	var lastErr error
	for _, candidate := range candidates {
		result, ok, err := parsePhaseResultCandidate(candidate)
		if err != nil {
			lastErr = err
			continue
		}
		if ok {
			return result, true, nil
		}
	}
	if lastErr != nil {
		return PhaseResult{}, false, lastErr
	}
	return PhaseResult{}, false, nil
}

func parsePhaseResultCandidate(candidate string) (PhaseResult, bool, error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal([]byte(candidate), &raw); err != nil {
		return PhaseResult{}, false, nil
	}
	payload, ok := raw[resultEnvelopeKey]
	if !ok {
		payload = []byte(candidate)
	}
	var result PhaseResult
	if err := json.Unmarshal(payload, &result); err != nil {
		return PhaseResult{}, true, fmt.Errorf("parse %s: %w", resultEnvelopeKey, err)
	}
	if result.Progress != nil {
		if err := result.Progress.Validate(); err != nil {
			return PhaseResult{}, true, err
		}
	}
	return result, hasPhaseResultContent(result), nil
}

func hasPhaseResultContent(result PhaseResult) bool {
	return len(result.Artifacts) > 0 || result.Handoff != nil || len(result.Handoffs) > 0 ||
		result.Readiness != nil || result.Progress != nil || strings.TrimSpace(result.Verdict) != "" ||
		result.ReplanNeeded || result.BacklogSync != nil
}
