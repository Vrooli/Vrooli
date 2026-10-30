// Package toolfriction profiles every command agents run, per command path and
// window, from invocation facts and the events they were derived from: how
// often it fails and why (clustered by normalised error signature), how often
// agents guess its shape, how much output it returns and how long that output
// stays in context, how slow it is and how often it is polled, and which
// scenario internals agents reach around a CLI to touch. It is a pure
// aggregation: no I/O, no clock, no model.
package toolfriction

import (
	"encoding/json"
	"regexp"
	"strings"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/runsignal"
	"agent-manager/internal/tokenaccounting"
)

// Fact is one invocation fact with the run dimensions the profiles read.
type Fact struct {
	runsignal.InvocationFact
	RunID  string
	At     time.Time
	Runner string
}

// Features are what a call's and result's text say about a command. They are
// read once, when the invocation fact is projected, and stored with it; none
// holds raw output (ErrorLine, NoticeLine and Command are redacted and
// bounded).
type Features struct {
	// Command is the redacted command line the fact was made from, "" when it
	// matched no pipeline of its call. Key is a read command (status, list,
	// get, show, wait…) with its target (ids, names, paths) kept and only
	// volatile values (cursors, limits, timeouts, timestamps) dropped, so
	// repeats of one query of one target compare equal and queries of different
	// targets do not; empty for any other command, which is never polled.
	Command, Key string
	Help         bool
	Filtered     bool
	JSONPaths    []string
	Wrapped      bool
	Truncated    bool
	// Signature is the normalised first meaningful line of a failure's error
	// text, ErrorLine the redacted line itself; ShapeGuess marks a failure that
	// says the command, subcommand, flag or argument is wrong.
	Signature, ErrorLine string
	ShapeGuess           bool
	// Notice is a degradation a Vrooli CLI call reports, whatever its
	// outcome ("agent-manager park unavailable … waiting inline"),
	// normalised; NoticeLine is the line.
	Notice, NoticeLine string
	// BypassOwner is the scenario whose internals the command reaches directly
	// and BypassMethod how (sqlite3 on its database, an HTTP call to its port,
	// a read of its state directory); empty when it does not.
	BypassOwner, BypassMethod string
}

// Call is one command a tool call ran: its identity, outcome and cost from the
// fact, and the features stored with it.
type Call struct {
	RunID, CallEventID string
	// ResultEventID is the result the call's outcome and duration come from:
	// segments of one command share it (see sharedDurations).
	ResultEventID string
	// OutputEventID is the result event holding the text the agent saw ("" when
	// the call was never paired with one).
	OutputEventID string
	At            time.Time
	Runner        string
	CommandPath   string
	// Owner is the Vrooli scenario (or "vrooli", the control plane) whose
	// catalogued CLI the command is; "" for a tool outside Vrooli.
	Owner   string
	Outcome string
	// Verdict is a verify command's non-zero exit (a lint, test or acceptance
	// verdict) and Interrupted an exit of 128 or more: neither is a failure. A
	// verify command that never reached its verdict (called with the wrong
	// shape, or stopped by an error of the tool: see reachedVerdict) failed,
	// and only then is its error line stored.
	Verdict, Interrupted bool
	DurationMS           int64
	HasDuration          bool
	// ResultTokens is what entered the agent's context (estimated from the
	// result text the harness handed the model) and TokenBasis how that is
	// known; unknown tokens are 0 with an unknown basis, never a measured 0
	// (see resultTokens). OutputBytes is what the command printed in total,
	// which a harness may cut before the model reads it (Codex's exec output
	// limit), 0 when the transcript does not say.
	ResultTokens, OutputBytes int64
	TokenBasis                tokenaccounting.Basis
	Residency                 int64
	// SkillID is the skill a `prompt-manager skill read` loaded and
	// ResidencySegment the compaction segment that load landed in: loads of one
	// skill in one segment are repeats of text still in context, where facts
	// record segments (SegmentsKnown).
	SkillID          string
	ResidencySegment int64
	// OwnWork marks a call reaching the internals of the scenario its run
	// works on (WorksOn): developing or reviewing a scenario includes
	// reading its database and state, which is not reaching around its CLI.
	// Set by the store that reads the run's tool calls.
	OwnWork bool
	// OlderParser marks a call from a run whose events an older transcript
	// parser derived: outcomes the current parser yields may be missing there,
	// which is not the same as absent. NoTranscript marks a call from a run
	// with no transcript record, whose parser is unknown. Set by the store
	// that knows the run's transcript record.
	OlderParser, NoTranscript bool
	Features
}

// SegmentsKnown reports whether the call's residency segment is real: its
// run's transcript was read by the current parser, which records compactions
// (the profiles read only current facts). A run read by an older parser, or
// with no transcript record, has no compaction events, so every call of it
// sits in segment 0 whatever happened.
func (c Call) SegmentsKnown() bool {
	return !c.OlderParser && !c.NoTranscript
}

// IsCommand reports whether the call ran a command; the others are file-read
// tool calls, read for bypasses only.
func (c Call) IsCommand() bool { return c.CommandPath != "" }

// LoadsSkill reports whether the call put a skill in context: a skill read
// whose output went through a filter (`skill read x | rg y`) did not.
func (c Call) LoadsSkill() bool { return c.SkillID != "" && !c.Filtered }

// OwnWorkShare is the share of a run's tool calls that name any scenario's
// tree (`scenarios/<name>/`) which must name one scenario's tree outside its
// data directory for the run to be working on that scenario. Read from
// the transcripts of 2026-10-23..30: of the 44 runs that ran sqlite3 on
// agent-manager's database, 38 name its tree in 59–97% of such calls (they
// develop or review agent-manager); runs reaching test-genie's port or
// workflow-health's state name those trees in 0–24%.
const OwnWorkShare = 0.5

// WorksOn reports whether a run works on a scenario: of its tool calls that
// name any scenario's source tree (named), at least OwnWorkShare name that
// scenario's (own).
func WorksOn(own, named int) bool {
	return named > 0 && float64(own) >= OwnWorkShare*float64(named)
}

// Bypasses reports whether the call reached a scenario's internals around its
// CLI: a direct read of its database, port or state by a run that is not
// working on that scenario itself (OwnWork).
func (c Call) Bypasses() bool { return c.BypassOwner != "" && !c.OwnWork }

// sharedDurations marks the calls whose duration is a whole chain's: one
// result reports on several commands (`a && b && c`), and the command's time
// is held by one of them (runsignal outputHolders) so that time sums once.
// That time is not the holder's own latency. Pipe readers carry no command
// words, so `status x | jq` keeps its duration as status's own. A store that
// does not read ResultEventID groups by call, which also sets aside separate
// commands of one Codex exec: latency then misses them, never misattributes.
func sharedDurations(calls []Call) []bool {
	type key struct{ run, call, result string }
	heads := map[key]int{}
	for _, c := range calls {
		if c.Command != "" {
			heads[key{c.RunID, c.CallEventID, c.ResultEventID}]++
		}
	}
	out := make([]bool, len(calls))
	for i, c := range calls {
		out[i] = heads[key{c.RunID, c.CallEventID, c.ResultEventID}] > 1
	}
	return out
}

// resultKey names the result a call's outcome, duration and output come from.
func resultKey(c Call) string { return c.RunID + "\x00" + c.CallEventID + "\x00" + c.ResultEventID }

// sizedResults are the results some segment holds known tokens for. A segment
// whose result is sized elsewhere (its command's output sits on the holder
// segment) is not a command of unknown size.
func sizedResults(calls []Call) map[string]bool {
	out := map[string]bool{}
	for _, c := range calls {
		if c.TokensKnown() {
			out[resultKey(c)] = true
		}
	}
	return out
}

// Failed is a call that went wrong: a failure that is neither a verify
// command's verdict nor an interrupt.
func (c Call) Failed() bool {
	return c.Outcome == runsignal.OutcomeFailure && !c.Verdict && !c.Interrupted
}

// decided is a call whose outcome says whether the command worked: success or
// a real failure. No-match, refusals, verdicts, interrupts and unknowns are not.
func (c Call) decided() bool {
	return c.Outcome == runsignal.OutcomeSuccess || c.Failed()
}

// PortOwners names the scenario whose API serves a local port, "" when none
// does (a scenario's UI port names no one: it is not an internal to reach).
type PortOwners func(port int) string

// Profiled is a fact the profiles read: a command (code excluded) or a file
// read. Only these carry stored Features.
func Profiled(f runsignal.InvocationFact) bool {
	return f.Capability != "code" && (f.CommandPath != "" || f.Capability == runsignal.KindFileRead)
}

// Observe is a fact with the features stored for it.
func Observe(f Fact, features Features) Call {
	c := base(f)
	c.Features = features
	if c.Verdict {
		// The stored error line of a verify command is read again by the
		// current rule: a report line (a passing test, a lint item) is the
		// verdict's text, not a tool error, so it neither fails nor clusters.
		if c.Verdict = features.Signature == "" || reachedVerdict(features.ErrorLine, features.ShapeGuess); c.Verdict {
			c.Signature, c.ErrorLine = "", ""
		}
	}
	return c
}

func base(f Fact) Call {
	c := Call{RunID: f.RunID, CallEventID: f.CallEventID, ResultEventID: f.ResultEventID, OutputEventID: f.OutputEventID, At: f.At, Runner: f.Runner, CommandPath: f.CommandPath, Outcome: f.Outcome, OutputBytes: f.OutputBytes, Residency: f.ResidencyRequests, SkillID: f.SkillID, ResidencySegment: f.ResidencySegment}
	c.ResultTokens, c.TokenBasis = resultTokens(f)
	if f.Ownership == runsignal.OwnershipResolved {
		c.Owner = f.Executable
	}
	if f.DurationMS != nil && *f.DurationMS >= 0 {
		c.DurationMS, c.HasDuration = *f.DurationMS, true
	}
	if f.Outcome == runsignal.OutcomeFailure {
		c.Verdict = f.SemanticsKind == "verify"
		c.Interrupted = f.ExitCode != nil && *f.ExitCode >= 128
	}
	return c
}

// resultTokens is what a call put in context and how that is known. Counted
// result text keeps its basis. A result whose stored text is empty although
// the command printed bytes (a nested Codex command whose output the
// transcript kept only as a size), or that the projection never paired, is
// estimated from the printed bytes: an upper bound where the harness cut the
// output. With neither, a result never paired, or a skill load that stored no
// text (a skill read prints the skill, so empty text was not captured), is
// unknown: zero tokens with an unknown basis, which readers count apart
// instead of reading as zero.
func resultTokens(f Fact) (int64, tokenaccounting.Basis) {
	switch {
	case f.ResultTokens > 0 && f.TokenBasis == tokenaccounting.BasisMeasured:
		return f.ResultTokens, tokenaccounting.BasisMeasured
	case f.ResultTokens > 0:
		return f.ResultTokens, tokenaccounting.BasisEstimated
	case f.OutputBytes > 0:
		return tokenaccounting.EstimateBytes(f.OutputBytes), tokenaccounting.BasisEstimated
	case f.TokenBasis == tokenaccounting.BasisUnknown || f.TokenBasis == "" || f.SkillID != "":
		return 0, tokenaccounting.BasisUnknown
	}
	return 0, f.TokenBasis
}

// TokensKnown reports whether the call's result tokens are a count or an
// estimate rather than unknown.
func (c Call) TokensKnown() bool { return c.TokenBasis != tokenaccounting.BasisUnknown }

// extract reads one command fact with its call and result. pipelines are the
// commands the fact was made from (its nested command's, else its call's) and
// occurrence how many earlier facts of them share its command path, which
// picks the fact's own pipeline.
func extract(f Fact, occurrence int, pipelines []runsignal.Pipeline, call *domain.ToolCallEventData, result *domain.ToolResultEventData, ports PortOwners) Features {
	c := base(f)
	pipeline, found := runsignal.PipelineFor(pipelines, f.CommandPath, occurrence)
	if found {
		c.readPipeline(pipeline, pipelines)
	}
	c.BypassOwner, c.BypassMethod = bypassOf(f, call, pipeline, found, ports)
	text := resultText(result)
	c.Truncated = truncated(text)
	if f.Outcome == runsignal.OutcomeFailure && !c.Interrupted {
		c.ShapeGuess = shapeError.MatchString(strings.ToLower(text))
		if line := redact(firstErrorLine(text, f.ExitCode)); !c.Verdict || !reachedVerdict(line, c.ShapeGuess) {
			c.ErrorLine, c.Signature = line, signature(line)
		}
	}
	if c.Owner != "" {
		if line := noticeLine(text); line != "" {
			c.NoticeLine = redact(line)
			c.Notice = normalise(c.NoticeLine)
		}
	}
	return c.Features
}

// readPipeline takes the command's shape from its pipeline: the example and
// key, a help probe, output filters and jq paths, and a polling wrapper.
func (c *Call) readPipeline(p runsignal.Pipeline, all []runsignal.Pipeline) {
	c.Command = redact(strings.Join(p.Words, " "))
	if readVerbs[lastWord(c.CommandPath)] {
		c.Key = pollKey(p.Words)
	}
	for _, word := range p.Words[1:] {
		if word == "--help" || word == "-h" || word == "help" {
			c.Help = true
		}
	}
	jsonOut := false
	for _, word := range p.Words {
		jsonOut = jsonOut || word == "--json" || word == "-o=json" || word == "--output=json"
	}
	for _, filter := range p.Filters {
		if outputFilters[filter[0]] {
			c.Filtered = true
		}
		if jsonOut && filter[0] == "jq" {
			c.JSONPaths = append(c.JSONPaths, jqPaths(filter[1:])...)
		}
	}
	c.Wrapped = p.Loop
	for _, other := range all {
		if len(other.Words) > 0 && (other.Words[0] == "sleep" || other.Words[0] == "watch") {
			c.Wrapped = true
		}
		if strings.Contains(strings.Join(other.Words, " "), "time.sleep") {
			c.Wrapped = true
		}
	}
}

// readVerbs end the command path of a command that reads state without
// changing it: running one again and again is polling.
var readVerbs = map[string]bool{"status": true, "list": true, "get": true, "show": true, "wait": true, "state": true, "board": true, "signals": true, "health": true, "logs": true, "describe": true, "inspect": true, "watch": true, "timeline": true, "freshness": true, "ps": true, "ls": true}

// volatileFlags take a value that changes between re-queries of one target
// without changing the target: a feed cursor or window, a page size, a
// deadline. `effort signals <effort> --after 57` then `--after 59` is the
// same feed read again, so the cursor is not part of the target.
var volatileFlags = map[string]bool{"--after": true, "--since": true, "--until": true, "--before": true, "--cursor": true, "--offset": true, "--page": true, "--limit": true, "--tail": true, "--lines": true, "-n": true, "--timeout": true, "--interval": true, "--max-time": true}

// pollKey is the comparable form of a read command: its words with the values
// of volatile flags and any timestamps dropped. Identifiers, names and paths
// stay, so two reads of different runs are different targets.
func pollKey(words []string) string {
	kept := make([]string, 0, len(words))
	for i := 0; i < len(words); i++ {
		word := words[i]
		if name, _, joined := strings.Cut(word, "="); joined && volatileFlags[name] {
			continue
		}
		if volatileFlags[word] {
			i++
			continue
		}
		kept = append(kept, timeWord.ReplaceAllString(word, "<time>"))
	}
	return redact(strings.Join(kept, " "))
}

// blockingFlags make a command wait for its target (or a deadline) before it
// returns.
var blockingFlags = map[string]bool{"--wait": true, "--timeout": true, "--wait-timeout": true}

// blocksUntilDone is a command that waits for its target to finish before it
// returns (its manifest declares it blocking, or it has a --wait or --timeout
// flag): one such call already is the wait, so repeating it is not polling.
func blocksUntilDone(commandPath string, words []string) bool {
	if runsignal.Blocks(commandPath, words) {
		return true
	}
	for _, word := range words {
		if name, _, _ := strings.Cut(word, "="); blockingFlags[name] {
			return true
		}
	}
	return false
}

// waited is a blocking call that did not fail: a wait that ran its course, or
// one whose outcome is unknown, which is not a timeout. Only a known failure
// (the wait itself failed or timed out) leaves a wait to count as polling.
func (c Call) waited() bool {
	return !c.Failed() && blocksUntilDone(c.CommandPath, strings.Fields(c.Command))
}

// target is what a call acts on: its first word after the executable that is
// neither a command-path word nor a flag, a `scenarios/<name>` path read as
// the name (`make -C scenarios/x restart`). "" when the command names none
// (`make restart` run in the scenario's directory).
func (c Call) target() string {
	path := strings.Fields(c.CommandPath)
	words := strings.Fields(c.Command)
	next := 1
	for _, word := range words[min(1, len(words)):] {
		if next < len(path) && word == path[next] {
			next++
			continue
		}
		if strings.HasPrefix(word, "-") {
			continue
		}
		if _, rest, ok := strings.Cut(word, "scenarios/"); ok {
			word, _, _ = strings.Cut(rest, "/")
		}
		return strings.TrimSuffix(word, "/")
	}
	return ""
}

func lastWord(path string) string {
	if i := strings.LastIndexByte(path, ' '); i >= 0 {
		return path[i+1:]
	}
	return path
}

// interactiveVerbs end the command path of a command expected to answer at
// once: queries of state. Commands that run work (tests, builds, starts) or
// wait by design (wait, watch, logs follow) are never slow for taking time.
var interactiveVerbs = map[string]bool{"status": true, "list": true, "get": true, "show": true, "read": true, "state": true, "board": true, "signals": true, "health": true, "describe": true, "inspect": true, "timeline": true, "freshness": true, "ps": true, "ls": true, "query": true, "search": true, "info": true, "version": true, "locks": true}

// outputFilters are programs that cut down another command's output.
var outputFilters = map[string]bool{"head": true, "tail": true, "grep": true, "rg": true, "jq": true, "python": true, "python3": true, "sed": true, "awk": true, "cut": true, "wc": true}

var jqPath = regexp.MustCompile(`\.[A-Za-z_][A-Za-z0-9_]*(?:\[\])?(?:\.[A-Za-z_][A-Za-z0-9_]*(?:\[\])?)*`)

// jqPaths are the field paths a jq filter reads, e.g. `.items[].id`.
func jqPaths(args []string) []string {
	for _, arg := range args {
		if strings.HasPrefix(arg, "-") {
			continue
		}
		return jqPath.FindAllString(arg, 4)
	}
	return nil
}

// nestedArgv is the argv a Codex rollout CommandExecution result reports: the
// command as it ran, more precise than the exec script that built it.
func nestedArgv(result *domain.ToolResultEventData) []string {
	if result == nil || result.ToolName != runsignal.NestedCommandTool || result.RawOutputJSON == "" {
		return nil
	}
	var record struct {
		Command []string `json:"command"`
	}
	if json.Unmarshal([]byte(result.RawOutputJSON), &record) != nil {
		return nil
	}
	return record.Command
}

// resultText is what the agent read, bounded: the error, then the output and
// the text of its first blocks.
func resultText(result *domain.ToolResultEventData) string {
	if result == nil {
		return ""
	}
	parts := []string{clip(result.Error, ExcerptHead), excerpt(result.Output)}
	for _, block := range result.OutputBlocks[:min(len(result.OutputBlocks), 2)] {
		parts = append(parts, clip(block.Text, ExcerptHead))
	}
	return strings.Join(parts, "\n")
}

// truncationMarkers are what Claude Code and Codex print when they cut a tool
// output down before the model reads it.
var truncationMarkers = []string{"<persisted-output>", "Output too large", "tokens truncated", "Total output lines:", "lines truncated", "[... omitted", "characters truncated"}

func truncated(text string) bool {
	for _, marker := range truncationMarkers {
		if strings.Contains(text, marker) {
			return true
		}
	}
	return false
}

// Source is one command fact with the call and result events it was derived
// from.
type Source struct {
	Fact   Fact
	Call   *domain.ToolCallEventData
	Result *domain.ToolResultEventData
}

// Extract reads the features of sources in fact order (by call, then segment),
// counting how many earlier facts share a command path so each fact finds its
// own pipeline. A fact joined to a nested Codex command result is read from
// that command's argv and counted among the facts of that command, so
// `status a && status b` run as one command gives its second fact `status b`;
// other facts are read from the call, lexed once, and counted among all the
// call's facts. A compound call's commands share one output, so a degradation
// it reports counts once, on the segment that printed it (see noticeHolder).
func Extract(sources []Source, ports PortOwners) []Features {
	out := make([]Features, 0, len(sources))
	seen := map[string]int{}
	pipelines := map[string][]runsignal.Pipeline{}
	noticed := map[string][]int{}
	for i, source := range sources {
		callKey := source.Fact.CallEventID + "\x00" + source.Fact.CommandPath
		key := callKey
		parsed, ok := pipelines[source.Fact.CallEventID]
		if !ok {
			parsed = runsignal.CallPipelines(source.Call)
			pipelines[source.Fact.CallEventID] = parsed
		}
		argv := nestedArgv(source.Result)
		if len(argv) > 0 {
			key += "\x00" + source.Fact.ResultEventID
			parsed = runsignal.ArgvPipelines(argv)
		}
		f := extract(source.Fact, seen[key], parsed, source.Call, source.Result, ports)
		seen[key]++
		if key != callKey {
			seen[callKey]++
		}
		if f.Notice != "" && argv == nil {
			noticed[source.Fact.CallEventID] = append(noticed[source.Fact.CallEventID], i)
		}
		out = append(out, f)
	}
	for _, indexes := range noticed {
		holder := noticeHolder(sources, out, indexes)
		for _, i := range indexes {
			if i != holder {
				out[i].Notice, out[i].NoticeLine = "", ""
			}
		}
	}
	return out
}

// noticeHolder is the segment of a call that printed the notice its shared
// output carries. A single command prints its own. In a compound call the
// output does not say which segment wrote the line, so it is the segment whose
// scenario the line names, else the call's dominant Vrooli CLI (most
// segments, first on a tie): the control plane's `vrooli` only when no
// scenario CLI ran, since it chains other CLIs and rarely prints their notices.
func noticeHolder(sources []Source, features []Features, indexes []int) int {
	counts := map[string]int{}
	first := map[string]int{}
	for _, i := range indexes {
		owner := sources[i].Fact.Executable
		if _, ok := first[owner]; !ok {
			first[owner] = i
		}
		counts[owner]++
	}
	if named := namedOwner(features[indexes[0]].NoticeLine); named != "" {
		if i, ok := first[named]; ok {
			return i
		}
	}
	best, bestCount := indexes[0], 0
	for _, i := range indexes {
		owner := sources[i].Fact.Executable
		weight := counts[owner]
		if owner == controlPlane {
			weight = 0
		}
		if weight > bestCount {
			best, bestCount = first[owner], weight
		}
	}
	return best
}
