package runsignal

import (
	"strconv"
	"strings"

	"agent-manager/internal/domain"
)

// Time kinds of an episode's WallClockMS.
const (
	// TimeRedundant is the own duration of the episode's redundant calls
	// ("redundant call time").
	TimeRedundant = "redundant"
	// TimeIdle is a stall's idle gap ("idle gap").
	TimeIdle = "idle"
)

// CostRule says which calls of an episode are redundant, and so what it costs.
type CostRule string

const (
	CostEnd   CostRule = "end"   // the last call repeats the first
	CostStart CostRule = "start" // the first call is the failed or corrected attempt
	CostBoth  CostRule = "both"  // both bounding calls
	CostRange CostRule = "range" // every call the detector names
	CostGap   CostRule = "gap"   // no call: the idle gap
	CostNone  CostRule = "none"  // not priced
)

// PatternSpec is the one definition of a friction pattern: its identity,
// classifier version, cause scope, what its time measures, which calls it
// prices, the sentence that describes it and the rule that detects it. Nothing
// else lists patterns.
type PatternSpec struct {
	ID, Version, Cause string
	TimeKind           string
	Cost               CostRule
	// Informational patterns are reported on a run but never priced or shown as
	// a usage friction problem.
	Informational bool
	// Quarantined, when set, says why the pattern fails the precision gate
	// (testdata/precision): it stays in the read model but is excluded from
	// usage friction, R-friction and the UI.
	Quarantined string
	// Description is the plain sentence: {cmd} is the quoted command, {Cmd} the
	// same capitalised, {element} the repeated element. DescriptionRepeated, when
	// set, replaces it for episodes that repeat ({n} times).
	Description, DescriptionRepeated string
	detect                           func(EpisodeDetectorContext) []FrictionEpisode
}

func (s PatternSpec) Identifier() string                                  { return s.ID }
func (s PatternSpec) ClassifierVersion() string                           { return s.Version }
func (s PatternSpec) Detect(ctx EpisodeDetectorContext) []FrictionEpisode { return s.detect(ctx) }
func (s PatternSpec) CauseScope() string                                  { return s.Cause }

// EpisodeDetector is a pure, registered classification rule.
type EpisodeDetector interface {
	Identifier() string
	ClassifierVersion() string
	CauseScope() string
	Detect(EpisodeDetectorContext) []FrictionEpisode
}

// PatternSpecs is the complete deterministic registry, in detection order.
func PatternSpecs() []PatternSpec {
	v := EpisodeClassifierVersion
	const (
		redundant, workaround, toolchain, execution, agent = TimeRedundant, "recurring-workaround", "toolchain", "run-execution", agentCauseScope
	)
	return []PatternSpec{
		{"repeated-work", v, workaround, redundant, CostEnd, false, "below the gate (see testdata/precision/results.json): a repeat after a tool reinstalled its CLI is indistinguishable without reading output", "{Cmd} runs again, back to back, with the same input.", "", detectRepeatedWork},
		{"command-failure", v, toolchain, redundant, CostStart, false, "", "{Cmd} failed: {element}", "", detectCommandFailures},
		{"help-recovery", v, execution, redundant, CostStart, false, "no labelled episodes", "{Cmd} fails and the agent opens help to recover.", "", adjacentEpisode("help-recovery", func(a, b InvocationFact) bool { return failed(a) && b.HelpRecovery })},
		{"stall", v, execution, TimeIdle, CostGap, false, "no true labels in the corpus; a false detection (a long gap the reduced window cannot attribute)", "A run sits idle between tool calls for a long stretch.", "", detectStalls},
		{"poll-loop", v, workaround, redundant, CostRange, false, "no labelled episodes", "An agent polls {cmd} over and over instead of waiting once.", "", detectPollLoops},
		{"wrong-primitive", v, workaround, redundant, CostRange, false, "no true labels in the corpus (sampled searches were in-repo, not recall); a false detection", "Agents run {cmd} repeatedly instead of the preferred `{element}`.", "", func(ctx EpisodeDetectorContext) []FrictionEpisode {
			return DetectWrongPrimitives(ctx.Facts, ctx.EventsByID, ctx.Events)
		}},
		{"oscillation", v, workaround, redundant, CostRange, false, "no true labels in the corpus (sampled episodes were supervisor status cycles)", "An agent alternates between two calls of {cmd} without making progress.", "", detectOscillations},
		{"edit-revert", v, workaround, redundant, CostBoth, false, "too few matched true detections or too wide a confidence interval to show (see testdata/precision/results.json)", "A file edit is undone soon after it is made.", "", detectEditReverts},
		{"wait-misuse", v, toolchain, redundant, CostRange, false, "too few matched true detections or too wide a confidence interval to show (see testdata/precision/results.json)", "An agent repeats a short wait instead of one blocking wait.", "An agent repeats a short wait {n} times instead of one blocking wait.", detectWaitMisuse},
		{"blocked-then-abandoned", v, execution, redundant, CostNone, false, "no labelled episodes", "A run hits a blocker and stops without resolving it.", "", detectBlockedThenAbandoned},
		{"guidance-repair", v, agent, redundant, CostNone, false, "", "The operator corrects the agent, and it switches approach.", "", detectGuidanceRepair},
		{"handoff-continuation", v, execution, redundant, CostNone, true, "informational: reported on a run, never priced or shown", "The agent declares it is done, then keeps working with no new instruction.", "", detectHandoffContinuation},
		{"fallback-after-capability", v, toolchain, redundant, CostEnd, false, "below the gate (see testdata/precision/results.json): python/curl one-liners inside Codex exec scripts are not parsed as commands, and one labelled-false window reaches a service directly", "{Cmd} works, then the agent falls back to a tool outside Vrooli for the same job.", "", detectFallbackAfterCapability},
		{"capability-abandoned", v, toolchain, redundant, CostStart, false, "below the gate (see testdata/precision/results.json): its true labels are not caught; abandoned-command-family now covers failures never followed by a success", "{Cmd} fails and the agent gives up on it instead of retrying.", "", detectCapabilityAbandoned},
		{"retry-after-failure", v, execution, redundant, CostStart, false, "below the gate (see testdata/precision/results.json): it fires in nearly every window that contains a failed command followed by a retry, including windows labelled for other patterns", "{Cmd} fails and the agent retries it straight away.", "", detectRetryAfterFailure},
		{"flag-hunting", v, toolchain, redundant, CostRange, false, "recall below the gate (see testdata/precision/results.json): true retries also change argument values, not only flags; value-aware shapes added a false positive", "After {cmd} fails, the agent tries several different flag sets for it.", "", detectFlagHunting},
		{"abandoned-command-family", v, toolchain, redundant, CostStart, false, "below the gate (see testdata/precision/results.json): it fires on most failed commands that are never retried, which is ordinary work", "{Cmd} fails and never succeeds before the agent moves on.", "", detectAbandonedCommandFamily},
		{"read-then-reread", v, workaround, redundant, CostEnd, false, "below the gate (see testdata/precision/results.json): a first read whose result is missing from the window is indistinguishable from a successful one", "An agent reads a file, then reads the same file again with no edit in between.", "", detectReadThenReread},
	}
}

// PatternByID finds a pattern's definition; false means no such pattern is
// detected any more.
func PatternByID(id string) (PatternSpec, bool) {
	for _, spec := range PatternSpecs() {
		if spec.ID == id {
			return spec, true
		}
	}
	return PatternSpec{}, false
}

// Shown says whether episodes of a pattern may be shown to a reader: the
// pattern is still detected, passes the precision gate (not Quarantined) and is
// not informational. Every episode read boundary filters with it; only the
// precision test and the durable read model read unfiltered.
func Shown(pattern string) bool {
	spec, known := PatternByID(pattern)
	return known && !spec.Informational && spec.Quarantined == ""
}

// ShownPatternIDs lists every pattern for which Shown is true, for readers that
// filter in SQL.
func ShownPatternIDs() []string {
	var ids []string
	for _, spec := range PatternSpecs() {
		if Shown(spec.ID) {
			ids = append(ids, spec.ID)
		}
	}
	return ids
}

// ShownEpisodes keeps the episodes of shown patterns and counts the rest, so a
// list can say how many were hidden instead of silently dropping them.
func ShownEpisodes(episodes []FrictionEpisode) (shown []FrictionEpisode, hidden int) {
	shown = make([]FrictionEpisode, 0, len(episodes))
	for _, episode := range episodes {
		if Shown(episode.Pattern) {
			shown = append(shown, episode)
		} else {
			hidden++
		}
	}
	return shown, hidden
}

// EpisodeDetectors is the registry as detectors.
func EpisodeDetectors() []EpisodeDetector {
	specs := PatternSpecs()
	out := make([]EpisodeDetector, len(specs))
	for i, spec := range specs {
		out[i] = spec
	}
	return out
}

// Render fills the description template for a command, a repeated element and
// a repeat count.
func (s PatternSpec) Render(command, element string, repeats int) string {
	text := s.Description
	if repeats > 1 && s.DescriptionRepeated != "" {
		text = strings.ReplaceAll(s.DescriptionRepeated, "{n}", strconv.Itoa(repeats))
	}
	quoted := "a command"
	if command != "" {
		quoted = "`" + command + "`"
	}
	text = strings.ReplaceAll(text, "{Cmd}", strings.ToUpper(quoted[:1])+quoted[1:])
	text = strings.ReplaceAll(text, "{cmd}", quoted)
	if element == "" {
		// No captured element: drop the separator that would introduce it.
		text = strings.TrimSuffix(strings.ReplaceAll(text, ": {element}", "."), "{element}")
	}
	return strings.ReplaceAll(text, "{element}", element)
}

// price sets the episode's cost from the calls it names as redundant: their
// own argument and result tokens and their own duration, one turn per call.
func (e *FrictionEpisode) price(calls []InvocationFact, byID map[string]*domain.RunEvent) {
	e.Turns, e.Tokens, e.WallClockMS = 0, 0, 0
	seen := map[string]bool{}
	for _, call := range calls {
		e.Turns++
		e.Tokens += int(call.ArgTokens + call.ResultTokens)
		if !seen[call.CallEventID] {
			seen[call.CallEventID] = true
			e.WallClockMS += callDurationMS(call, byID)
		}
	}
	if e.Turns == 0 {
		e.Turns = 1
	}
}

// pricedCalls is the redundant calls a pattern's cost rule names among an
// episode's bounding calls.
func pricedCalls(rule CostRule, start, end InvocationFact) []InvocationFact {
	switch rule {
	case CostEnd:
		return []InvocationFact{end}
	case CostStart:
		return []InvocationFact{start}
	case CostBoth, CostRange:
		if start.CallEventID == end.CallEventID {
			return []InvocationFact{start}
		}
		return []InvocationFact{start, end}
	}
	return nil
}
