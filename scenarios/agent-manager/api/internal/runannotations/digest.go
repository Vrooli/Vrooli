package runannotations

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// Digest is the deterministic, bounded JSON state a decision model reads. It
// contains only fields a declarative evidence selector names; no transcript is
// ever copied wholesale.
type Digest struct {
	RunID           string             `json:"run_id"`
	Workload        DigestWorkload     `json:"workload"`
	HarnessKind     string             `json:"harness_kind,omitempty"`
	RunMode         string             `json:"run_mode,omitempty"`
	DurationMin     float64            `json:"duration_min"`
	Status          string             `json:"status,omitempty"`
	TerminalClass   string             `json:"terminal_class,omitempty"`
	StopReason      string             `json:"stop_reason,omitempty"`
	ExitCode        *int               `json:"exit_code,omitempty"`
	Counts          DigestCounts       `json:"counts"`
	OperatorText    []string           `json:"operator_messages,omitempty"`
	AgentText       []string           `json:"agent_text,omitempty"`
	Tool            DigestTool         `json:"tool"`
	Flags           DigestFlags        `json:"deterministic_flags"`
	ExistingSignals map[string]float64 `json:"existing_signals,omitempty"`
}

// DigestWorkload identifies the run's workload.
type DigestWorkload struct {
	Kind string `json:"kind"`
	Key  string `json:"key,omitempty"`
}

// DigestCounts keeps counts even when text is dropped for size.
type DigestCounts struct {
	OperatorMessages int `json:"operator_messages"`
	AgentText        int `json:"agent_text"`
	ToolCalls        int `json:"tool_calls"`
}

// DigestTool is the bounded tool summary.
type DigestTool struct {
	Calls            int      `json:"calls"`
	Failures         int      `json:"failures"`
	FailedCommands   int      `json:"failed_commands"`
	RepeatedPaths    int      `json:"repeated_paths"`
	RepeatedCommands int      `json:"repeated_commands"`
	FailedTools      []string `json:"failed_tools,omitempty"`
	LastError        string   `json:"last_error,omitempty"`
}

// DigestFlags are deterministic observations, never model guesses.
type DigestFlags struct {
	StepbackTriggers []string `json:"stepback_triggers,omitempty"`
	ToolFindings     []string `json:"tool_findings,omitempty"`
	CompactionCount  int      `json:"compaction_count"`
}

type digestCaps struct {
	operator        int
	operatorChars   int
	agent           int
	agentChars      int
	failedTools     int
	errorChars      int
	includeAgent    bool
	includeOperator bool
}

// reviewCapLadder is the gold reviewer's view: the same fields with room for
// more text. A reviewer needs the evidence that decides the label; the model's
// small-context caps should not decide whether a human can judge a run. Blindness
// is about the stored answer, not the evidence, so a wider packet is safe.
var reviewCapLadder = []digestCaps{
	{operator: 8, operatorChars: 700, agent: 6, agentChars: 500, failedTools: 12, errorChars: 500, includeAgent: true, includeOperator: true},
	{operator: 6, operatorChars: 500, agent: 4, agentChars: 400, failedTools: 8, errorChars: 400, includeAgent: true, includeOperator: true},
	{operator: 4, operatorChars: 400, agent: 3, agentChars: 300, failedTools: 6, errorChars: 300, includeAgent: true, includeOperator: true},
}

// DefaultReviewDigestLimit bounds the gold reviewer's packet. It is larger than
// the model digest because a person can read more than a small local model.
const DefaultReviewDigestLimit = 6000

var digestCapLadder = []digestCaps{
	{operator: 3, operatorChars: 280, agent: 2, agentChars: 220, failedTools: 4, errorChars: 200, includeAgent: true, includeOperator: true},
	{operator: 3, operatorChars: 200, agent: 1, agentChars: 160, failedTools: 3, errorChars: 140, includeAgent: true, includeOperator: true},
	{operator: 2, operatorChars: 150, agent: 1, agentChars: 110, failedTools: 3, errorChars: 100, includeAgent: true, includeOperator: true},
	{operator: 1, operatorChars: 100, agent: 0, agentChars: 0, failedTools: 2, errorChars: 80, includeAgent: false, includeOperator: true},
	{operator: 0, operatorChars: 0, agent: 0, agentChars: 0, failedTools: 1, errorChars: 0, includeAgent: false, includeOperator: false},
}

// BuildDigest renders the bounded digest with the default limit.
func BuildDigest(facts RunFacts) string {
	body, err := BuildDigestBounded(facts, DefaultDigestLimit)
	if err != nil {
		// The minimal rendering is provably below the default limit; this branch
		// exists so a future limit cannot silently emit invalid JSON.
		minimal, _ := json.Marshal(Digest{RunID: facts.RunID, Workload: DigestWorkload{Kind: facts.Workload.Kind}})
		return string(minimal)
	}
	return body
}

// BuildDigestBounded renders valid JSON no larger than max characters. It
// shrinks optional text deterministically and never truncates JSON syntax.
func BuildDigestBounded(facts RunFacts, max int) (string, error) {
	if max <= 0 {
		max = DefaultDigestLimit
	}
	for _, caps := range digestCapLadder {
		body, err := json.Marshal(digestFromFacts(facts, caps))
		if err != nil {
			return "", err
		}
		if len(body) <= max {
			return string(body), nil
		}
	}
	minimal, err := json.Marshal(Digest{RunID: facts.RunID, Workload: DigestWorkload{Kind: facts.Workload.Kind}})
	if err != nil {
		return "", err
	}
	if len(minimal) > max {
		return "", fmt.Errorf("digest limit %d is smaller than the minimal digest %d", max, len(minimal))
	}
	return string(minimal), nil
}

// BuildReviewDigest renders the bounded state a human reviewer judges. It uses
// the same fields and the same deterministic truncation as the model digest,
// with wider caps, and never includes the stored answer.
func BuildReviewDigest(facts RunFacts, max int) (string, error) {
	if max <= 0 {
		max = DefaultReviewDigestLimit
	}
	for _, caps := range reviewCapLadder {
		body, err := json.Marshal(digestFromFacts(facts, caps))
		if err != nil {
			return "", err
		}
		if len(body) <= max {
			return string(body), nil
		}
	}
	return BuildDigestBounded(facts, max)
}

func digestFromFacts(facts RunFacts, caps digestCaps) Digest {
	digest := Digest{
		RunID:         facts.RunID,
		Workload:      DigestWorkload{Kind: facts.Workload.Kind, Key: BoundText(facts.Workload.Key, 80)},
		HarnessKind:   facts.HarnessKind,
		RunMode:       facts.RunMode,
		DurationMin:   facts.Timings.DurationMinutes,
		Status:        facts.Terminal.Status,
		TerminalClass: facts.Terminal.Class,
		StopReason:    facts.Terminal.StopReason,
		ExitCode:      facts.Terminal.ExitCode,
		Counts: DigestCounts{
			OperatorMessages: len(facts.OperatorMessages),
			AgentText:        len(facts.AgentText),
			ToolCalls:        facts.ToolSummary.Calls,
		},
		Tool: DigestTool{
			Calls:            facts.ToolSummary.Calls,
			Failures:         facts.ToolSummary.Failures,
			FailedCommands:   facts.ToolSummary.FailedCommands,
			RepeatedPaths:    facts.ToolSummary.RepeatedPaths,
			RepeatedCommands: facts.ToolSummary.RepeatedCommands,
			FailedTools:      BoundList(facts.ToolSummary.FailedTools, caps.failedTools, 80),
			LastError:        BoundText(facts.ToolSummary.LastError, caps.errorChars),
		},
		Flags: DigestFlags{
			StepbackTriggers: BoundList(facts.DeterministicFlags.StepbackTriggers, 6, 80),
			ToolFindings:     BoundList(facts.DeterministicFlags.ToolFindings, 6, 80),
			CompactionCount:  facts.DeterministicFlags.CompactionCount,
		},
	}
	if caps.includeOperator {
		for _, message := range lastMessages(facts.OperatorMessages, caps.operator) {
			digest.OperatorText = append(digest.OperatorText, BoundText(message.Text, caps.operatorChars))
		}
	}
	if caps.includeAgent {
		for _, message := range lastMessages(facts.AgentText, caps.agent) {
			digest.AgentText = append(digest.AgentText, BoundText(message.Text, caps.agentChars))
		}
	}
	if len(facts.ExistingSignals) > 0 {
		digest.ExistingSignals = map[string]float64{}
		labels := make([]string, 0, len(facts.ExistingSignals))
		for label := range facts.ExistingSignals {
			labels = append(labels, label)
		}
		sort.Strings(labels)
		for _, label := range labels {
			if probability := facts.ExistingSignals[label].Probability; probability != nil {
				digest.ExistingSignals[label] = *probability
			}
		}
		if len(digest.ExistingSignals) == 0 {
			digest.ExistingSignals = nil
		}
	}
	return digest
}

func lastMessages(messages []MessageFact, count int) []MessageFact {
	if count <= 0 || len(messages) == 0 {
		return nil
	}
	if len(messages) <= count {
		return messages
	}
	return messages[len(messages)-count:]
}

// BoundText trims to at most limit runes and marks the cut with an ellipsis.
func BoundText(text string, limit int) string {
	text = strings.TrimSpace(text)
	if limit <= 0 || text == "" {
		return ""
	}
	runes := []rune(text)
	if len(runes) <= limit {
		return string(runes)
	}
	if limit == 1 {
		return "…"
	}
	return string(runes[:limit-1]) + "…"
}

// BoundList trims a list and each entry.
func BoundList(values []string, count, chars int) []string {
	if count <= 0 || len(values) == 0 {
		return nil
	}
	if len(values) > count {
		values = values[:count]
	}
	out := make([]string, 0, len(values))
	for _, value := range values {
		if bounded := BoundText(value, chars); bounded != "" {
			out = append(out, bounded)
		}
	}
	return out
}
