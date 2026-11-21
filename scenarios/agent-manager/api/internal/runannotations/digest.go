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

// digestCaps is how much of each digest section one rendering keeps.
type digestCaps struct {
	operator      int
	operatorChars int
	agent         int
	agentChars    int
	failedTools   int
	errorChars    int
	flags         int
	signals       bool
}

// reviewCapLadder is the gold reviewer's view: the same fields with room for
// more text. A reviewer needs the evidence that decides the label; the model's
// small-context caps should not decide whether a human can judge a run. Blindness
// is about the stored answer, not the evidence, so a wider packet is safe.
var reviewCapLadder = []digestCaps{
	{operator: 8, operatorChars: 700, agent: 6, agentChars: 500, failedTools: 12, errorChars: 500, flags: 6, signals: true},
	{operator: 6, operatorChars: 500, agent: 4, agentChars: 400, failedTools: 8, errorChars: 400, flags: 6, signals: true},
	{operator: 4, operatorChars: 400, agent: 3, agentChars: 300, failedTools: 6, errorChars: 300, flags: 6, signals: true},
}

// DefaultReviewDigestLimit bounds the gold reviewer's packet. It is larger than
// the model digest because a person can read more than a small local model.
const DefaultReviewDigestLimit = 6000

// digestFormat versions how a run's state is rendered for a model: the sections,
// their steps and the default priority below. It is part of the definition hash
// of every label whose answer reads that state, so a change to any of them makes
// stored answers and measurements from the old rendering stale instead of being
// mistaken for answers to the new input. Bump it with the change.
const digestFormat = 2

// Digest sections are the evidence sources a run state is made of. Each shrinks
// on its own, through the steps below, and drops out at its last step.
const (
	sectionOperator = "operator_messages"
	sectionAgent    = "agent_text"
	sectionTool     = "tool_summary"
	sectionFlags    = "deterministic_flags"
	sectionSignals  = "existing_signals"
)

// defaultDigestPriority orders the sections most important first when a label
// declares nothing: how a run ended (the last tool error and failed tools, the
// agent's closing words) outranks early-run detail, so under a small context
// window the early part of a run is lost before the end of it.
var defaultDigestPriority = []string{sectionTool, sectionAgent, sectionFlags, sectionOperator, sectionSignals}

// defaultDigestFocus is how many leading sections of the default order a label
// that declares none is taken to judge.
const defaultDigestFocus = 2

// sectionStep is one size of a section: how many items it keeps and how long
// each may be. For the tool section items counts failed tools and chars bounds
// the last error; for signals items is 1 (kept) or 0 (dropped).
type sectionStep struct{ items, chars int }

// sectionSteps lists each section's sizes from full to dropped.
var sectionSteps = map[string][]sectionStep{
	sectionOperator: {{3, 280}, {3, 200}, {2, 150}, {1, 100}, {0, 0}},
	sectionAgent:    {{2, 220}, {1, 160}, {1, 110}, {1, 60}, {0, 0}},
	sectionTool:     {{4, 200}, {3, 140}, {3, 100}, {2, 80}, {1, 40}, {0, 0}},
	sectionFlags:    {{6, 80}, {3, 80}, {1, 80}, {0, 0}},
	sectionSignals:  {{1, 0}, {0, 0}},
}

// DigestPriority orders the digest sections a label's evidence names, most
// important first, by their first appearance. A label lists its evidence in the
// order it needs it; sections it does not name are shrunk before any it does.
func DigestPriority(evidence []EvidenceSelector) []string {
	var order []string
	seen := map[string]bool{}
	for _, selector := range evidence {
		if _, shrinks := sectionSteps[selector.Source]; shrinks && !seen[selector.Source] {
			seen[selector.Source] = true
			order = append(order, selector.Source)
		}
	}
	return order
}

// digestRungs lists the renderings to try, largest first, as the step each
// section is at. A section no asked label names is dropped before any section
// one names shrinks, least important first. The named sections then shrink
// together one step at a time, each starting after the one ranked below it, so
// the most important keeps the most room while none is starved: a state that
// keeps a last error but no agent words judges a different thing than the
// label asked about.
func digestRungs(declared []string) []map[string]int {
	focus := make([]string, 0, len(sectionSteps))
	named := map[string]bool{}
	for _, section := range declared {
		if _, shrinks := sectionSteps[section]; shrinks && !named[section] {
			named[section] = true
			focus = append(focus, section)
		}
	}
	if len(focus) == 0 {
		focus = append(focus, defaultDigestPriority[:defaultDigestFocus]...)
		for _, section := range focus {
			named[section] = true
		}
	}
	levels := map[string]int{}
	snapshot := func() map[string]int {
		copied := make(map[string]int, len(levels))
		for section, level := range levels {
			copied[section] = level
		}
		return copied
	}
	rungs := []map[string]int{snapshot()}
	for index := len(defaultDigestPriority) - 1; index >= 0; index-- {
		section := defaultDigestPriority[index]
		if named[section] {
			continue
		}
		for level := 1; level < len(sectionSteps[section]); level++ {
			levels[section] = level
			rungs = append(rungs, snapshot())
		}
	}
	for tick := 1; ; tick++ {
		moved := false
		for rank, section := range focus {
			level := min(max(tick-(len(focus)-1-rank), 0), len(sectionSteps[section])-1)
			if level != levels[section] {
				levels[section], moved = level, true
			}
		}
		if !moved && tick > len(focus) {
			break
		}
		if moved {
			rungs = append(rungs, snapshot())
		}
	}
	return rungs
}

// capsAt is the rendering with each section at the step levels names.
func capsAt(levels map[string]int) digestCaps {
	step := func(section string) sectionStep { return sectionSteps[section][levels[section]] }
	operator, agent, tool, flags, signals := step(sectionOperator), step(sectionAgent), step(sectionTool), step(sectionFlags), step(sectionSignals)
	return digestCaps{
		operator: operator.items, operatorChars: operator.chars,
		agent: agent.items, agentChars: agent.chars,
		failedTools: tool.items, errorChars: tool.chars,
		flags: flags.items, signals: signals.items > 0,
	}
}

// BuildDigest renders the bounded digest with the default limit.
func BuildDigest(facts RunFacts) string {
	body, err := BuildDigestBounded(facts, DefaultDigestLimit, nil)
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
//
// priority is the sections the labels asked about declare, most important first
// (see DigestPriority); nil asks for the default, the end of the run. What the
// labels judge is the last thing to lose room (see digestRungs).
func BuildDigestBounded(facts RunFacts, max int, priority []string) (string, error) {
	if max <= 0 {
		max = DefaultDigestLimit
	}
	for _, levels := range digestRungs(priority) {
		body, err := json.Marshal(digestFromFacts(facts, capsAt(levels)))
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
	return BuildDigestBounded(facts, max, nil)
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
			ToolCalls:        facts.ToolSummary.ToolCalls,
		},
		Tool: DigestTool{
			Calls:            facts.ToolSummary.ToolCalls,
			Failures:         facts.ToolSummary.ToolFailures,
			FailedCommands:   facts.ToolSummary.FailedCommands,
			RepeatedPaths:    facts.ToolSummary.RepeatedPaths,
			RepeatedCommands: facts.ToolSummary.RepeatedCommands,
			FailedTools:      BoundList(facts.ToolSummary.FailedTools, caps.failedTools, 80),
			LastError:        BoundText(facts.ToolSummary.LastError, caps.errorChars),
		},
		Flags: DigestFlags{
			StepbackTriggers: BoundList(facts.DeterministicFlags.StepbackTriggers, caps.flags, 80),
			ToolFindings:     BoundList(facts.DeterministicFlags.ToolFindings, caps.flags, 80),
			CompactionCount:  facts.DeterministicFlags.CompactionCount,
		},
	}
	for _, message := range lastMessages(facts.OperatorMessages, caps.operator) {
		digest.OperatorText = append(digest.OperatorText, BoundText(message.Text, caps.operatorChars))
	}
	for _, message := range lastMessages(facts.AgentText, caps.agent) {
		digest.AgentText = append(digest.AgentText, BoundText(message.Text, caps.agentChars))
	}
	if caps.signals && len(facts.ExistingSignals) > 0 {
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
