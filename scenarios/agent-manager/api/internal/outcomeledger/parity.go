package outcomeledger

import (
	"fmt"
	"sort"
	"strings"
)

// Parity measure names.
const (
	MeasureUsage      = "usage"
	MeasureToolFacts  = "tool_facts"
	MeasureOutcomes   = "tool_outcomes"
	MeasureMessages   = "message_signals"
	MeasureModel      = "model"
	MeasureMode       = "mode"
	MeasureEstimate   = "outcome"
	parityReasonFloor = 0.9
)

// ParityMeasure is one measure's coverage on one path and harness.
type ParityMeasure struct {
	Name                   string
	Numerator, Denominator int64
	NotApplicable          int64
	Reason                 string
}

// Share is numerator over denominator; zero for an empty denominator.
func (m ParityMeasure) Share() float64 {
	if m.Denominator == 0 {
		return 0
	}
	return float64(m.Numerator) / float64(m.Denominator)
}

// ParityRow is every measure for one path and harness.
type ParityRow struct {
	Path, Harness string
	Runs          int64
	Eligible      int64
	NoRequests    int64
	Excluded      map[string]int64
	Measures      []ParityMeasure
}

// Eligible reports a run that enters parity denominators. An attached row
// enters only when it may be a real session the ledger cannot see.
func Eligible(run RunFacts) bool {
	if run.Path == PathAttached {
		return run.AttachedClass == AttachedNoReader || run.AttachedClass == AttachedUnlinked
	}
	return run.UsageExpected()
}

// BuildParity reports coverage per start path and harness over runs. Every run
// counts once; a run with no commands is "no commands", never a friction gap.
func BuildParity(runs []RunFacts) []ParityRow {
	type key struct{ path, harness string }
	groups := map[key][]RunFacts{}
	for _, run := range runs {
		k := key{run.Path, run.Harness}
		groups[k] = append(groups[k], run)
	}
	out := make([]ParityRow, 0, len(groups))
	for k, list := range groups {
		out = append(out, parityRow(k.path, k.harness, list))
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Path != out[j].Path {
			return out[i].Path < out[j].Path
		}
		return out[i].Harness < out[j].Harness
	})
	return out
}

func parityRow(path, harness string, runs []RunFacts) ParityRow {
	row := ParityRow{Path: path, Harness: harness, Runs: int64(len(runs)), Excluded: map[string]int64{}}
	usage := ParityMeasure{Name: MeasureUsage}
	tools := ParityMeasure{Name: MeasureToolFacts}
	outcomes := ParityMeasure{Name: MeasureOutcomes}
	messages := ParityMeasure{Name: MeasureMessages}
	model := ParityMeasure{Name: MeasureModel}
	mode := ParityMeasure{Name: MeasureMode}
	estimate := ParityMeasure{Name: MeasureEstimate}
	var evicted, older, olderCommands, unreadable, noSignal, inProgress, completed, noMessageRuns, noRequests int64
	for _, run := range runs {
		if path == PathAttached && !Eligible(run) {
			row.Excluded[run.AttachedClass]++
			continue
		}
		if path != PathAttached && !run.UsageExpected() {
			row.NoRequests++
			noRequests++
			continue
		}
		row.Eligible++
		usage.Denominator++
		if run.HasRequests() {
			usage.Numerator++
		} else if run.FileStatus == "unreadable" {
			unreadable++
		}
		switch {
		case run.TranscriptToolCalls > 0 || run.FactRows > 0:
			tools.Denominator++
			if run.FactRows > 0 {
				tools.Numerator++
			} else if run.RecallEvicted {
				evicted++
			}
		case run.HasRequests():
			tools.NotApplicable++
		}
		if run.OlderParser {
			older++
			olderCommands += run.Commands
			outcomes.NotApplicable += run.Commands
		} else {
			outcomes.Denominator += run.Commands
			outcomes.Numerator += run.KnownCommands
		}
		switch {
		case run.Child(), path == PathHeadless, path == PathAttached:
			messages.NotApplicable++
		case run.OperatorMessages == 0 && (run.LinkKind == LinkGoal || run.LinkKind == LinkWebConsole):
			// A worker thread of a linked session: its prompts come from the
			// orchestrating thread, not an operator.
			messages.NotApplicable++
		default:
			messages.Denominator++
			if run.OperatorMessages > 0 {
				messages.Numerator++
			} else {
				noMessageRuns++
			}
		}
		if !run.HasRequests() {
			model.NotApplicable++
			mode.NotApplicable++
			estimate.NotApplicable++
			continue
		}
		model.Denominator++
		if run.ActualModel != "" && run.ActualModel != "unknown" {
			model.Numerator++
		}
		mode.Denominator++
		if run.ModeRelation != "" {
			mode.Numerator++
		}
		estimate.Denominator++
		if run.UnitOutcome == OutcomeCompleted {
			completed++ // lifecycle only: not an outcome
		} else if run.UnitOutcome != "" && run.UnitOutcome != OutcomeUnknown {
			estimate.Numerator++
		} else if run.UnitSource == SourceInProgress {
			inProgress++
		} else {
			noSignal++
		}
	}
	usage.NotApplicable = noRequests
	reason := func(m *ParityMeasure, text string) {
		if m.Denominator > 0 && m.Share() < parityReasonFloor {
			m.Reason = text
		} else if m.Denominator == 0 && text != "" && m.Name == MeasureOutcomes && olderCommands > 0 {
			m.Reason = text
		}
	}
	switch {
	case path == PathAttached && HasTranscriptReader(harness):
		reason(&usage, "attached row with no matching transcript: identity only, ran 3 minutes or longer, and no imported session started beside it")
	case path == PathAttached:
		reason(&usage, "no transcript reader for "+harness+"; see the gaps")
	case unreadable > 0:
		reason(&usage, fmt.Sprintf("%d transcripts are not read to the end: see usage coverage", unreadable))
	default:
		reason(&usage, "transcripts not yet read to the end")
	}
	if path == PathAttached {
		reason(&tools, "identity-only row: nothing to read")
		reason(&outcomes, "identity-only row: nothing to read")
	} else {
		reason(&tools, fmt.Sprintf("%d runs have tool calls in their usage rows but no facts: %d left recall before projection, the rest were never projected", tools.Denominator-tools.Numerator, evicted))
		reason(&outcomes, fmt.Sprintf("compound commands where no segment outcome can be proved, and calls the transcript never paired; another %d commands from %d runs were read by an older parser and are not counted (AM-12a)", olderCommands, older))
		if outcomes.Denominator == 0 && olderCommands > 0 {
			outcomes.Reason = fmt.Sprintf("all %d commands are from %d runs read by an older parser (AM-12a); current parser has none yet", olderCommands, older)
		}
	}
	reason(&messages, fmt.Sprintf("%d sessions have no classified operator message: a harness-launched session has none, or its events left recall before classification", noMessageRuns))
	reason(&model, "usage rows name no model")
	reason(&estimate, fmt.Sprintf("%d runs finished with no evidence of acceptance (completed); %d have no outcome signal and stay unknown; %d still in progress", completed, noSignal, inProgress))
	row.Measures = []ParityMeasure{usage, tools, outcomes, messages, model, mode, estimate}
	return row
}

// ExcludedNames lists a row's excluded classes in a stable order.
func (r ParityRow) ExcludedNames() []string {
	names := make([]string, 0, len(r.Excluded))
	for name := range r.Excluded {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// Gap is one known difference between paths and why it remains.
type Gap struct {
	Subject, Status string
	Count           int64
	Reason          string
}

// Gap statuses.
const (
	GapClosed    = "closed"
	GapOpen      = "open"
	GapExplained = "explained"
)

// StaticGapReasons explain the harnesses with no transcript reader.
var StaticGapReasons = map[string]string{
	"opencode":    "sessions live in a 15 GB SQLite database, not JSONL; a reader is a new parser project, and no real session ran in the last 7 days",
	"grok":        "events.jsonl has per-turn events and no token usage, so the existing parser would give no quota evidence; real sessions are rare",
	"antigravity": "brain directory holds artifacts, not requests or tokens; no reader is possible from it",
}

// NoReaderGaps lists each harness that has rows but no reader.
func NoReaderGaps(runs []RunFacts) []Gap {
	counts := map[string]int64{}
	for _, run := range runs {
		if run.Path == PathAttached && !HasTranscriptReader(run.Harness) {
			counts[run.Harness] += 0
			if run.AttachedClass == AttachedNoReader {
				counts[run.Harness]++
			}
		}
	}
	names := make([]string, 0, len(counts))
	for name := range counts {
		names = append(names, name)
	}
	sort.Strings(names)
	var out []Gap
	for _, name := range names {
		reason := StaticGapReasons[name]
		if reason == "" {
			reason = "no transcript reader exists for this harness"
		}
		out = append(out, Gap{Subject: strings.TrimSpace(name) + " transcripts", Status: GapOpen, Count: counts[name], Reason: reason})
	}
	return out
}
