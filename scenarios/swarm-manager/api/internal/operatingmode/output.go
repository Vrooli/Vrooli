package operatingmode

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

const resultEnvelopeKey = "operating_mode_result"

type PhaseResultParseStatus string

const (
	PhaseResultParseNoOutput           PhaseResultParseStatus = "no_output"
	PhaseResultParseNoStructuredResult PhaseResultParseStatus = "no_structured_result"
	PhaseResultParseMalformed          PhaseResultParseStatus = "malformed_structured_result"
	PhaseResultParseEmpty              PhaseResultParseStatus = "empty_structured_result"
	PhaseResultParseValid              PhaseResultParseStatus = "valid_structured_result"
)

type ParsedPhaseResult struct {
	Result PhaseResult
	Status PhaseResultParseStatus
}

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
	parsed, err := ParsePhaseResultDetailed(output)
	if err != nil {
		return PhaseResult{}, false, err
	}
	if parsed.Status != PhaseResultParseValid {
		return parsed.Result, false, nil
	}
	return parsed.Result, true, nil
}

func ParsePhaseResultDetailed(output string) (ParsedPhaseResult, error) {
	trimmed := strings.TrimSpace(output)
	if trimmed == "" {
		return ParsedPhaseResult{Status: PhaseResultParseNoOutput}, nil
	}
	candidates := []phaseResultCandidate{{body: trimmed, structuredHint: strings.Contains(trimmed, resultEnvelopeKey)}}
	for _, match := range fencedJSONBlockRE.FindAllStringSubmatch(trimmed, -1) {
		if len(match) > 1 {
			body := strings.TrimSpace(match[1])
			candidates = append(candidates, phaseResultCandidate{body: body, structuredHint: true})
		}
	}
	var lastErr error
	for _, candidate := range candidates {
		result, status, err := parsePhaseResultCandidate(candidate)
		if err != nil {
			lastErr = err
			continue
		}
		switch status {
		case PhaseResultParseValid, PhaseResultParseEmpty:
			return ParsedPhaseResult{Result: result, Status: status}, nil
		case PhaseResultParseMalformed:
			return ParsedPhaseResult{Status: status}, fmt.Errorf("parse %s: malformed structured result", resultEnvelopeKey)
		}
	}
	if lastErr != nil {
		return ParsedPhaseResult{Status: PhaseResultParseMalformed}, lastErr
	}
	return ParsedPhaseResult{Status: PhaseResultParseNoStructuredResult}, nil
}

type phaseResultCandidate struct {
	body           string
	structuredHint bool
}

func parsePhaseResultCandidate(candidate phaseResultCandidate) (PhaseResult, PhaseResultParseStatus, error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal([]byte(candidate.body), &raw); err != nil {
		if candidate.structuredHint {
			return PhaseResult{}, PhaseResultParseMalformed, err
		}
		return PhaseResult{}, PhaseResultParseNoStructuredResult, nil
	}
	payload, ok := raw[resultEnvelopeKey]
	if !ok {
		return PhaseResult{}, PhaseResultParseNoStructuredResult, nil
	}
	var result PhaseResult
	if err := json.Unmarshal(payload, &result); err != nil {
		return PhaseResult{}, PhaseResultParseMalformed, fmt.Errorf("parse %s: %w", resultEnvelopeKey, err)
	}
	if result.Progress != nil {
		if err := result.Progress.Validate(); err != nil {
			return PhaseResult{}, PhaseResultParseMalformed, err
		}
	}
	if !hasPhaseResultContent(result) {
		return result, PhaseResultParseEmpty, nil
	}
	return result, PhaseResultParseValid, nil
}

func hasPhaseResultContent(result PhaseResult) bool {
	return len(result.Artifacts) > 0 || result.Handoff != nil || len(result.Handoffs) > 0 ||
		result.Readiness != nil || result.Progress != nil || strings.TrimSpace(result.Verdict) != "" ||
		result.ReplanNeeded || result.BacklogSync != nil
}
