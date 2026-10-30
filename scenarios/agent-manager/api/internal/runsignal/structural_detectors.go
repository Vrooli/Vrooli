package runsignal

import (
	"strconv"
	"strings"
	"time"

	"agent-manager/internal/domain"
)

// These detectors intentionally use only typed invocation facts and event
// timing. Phrase spans may corroborate a signal but never create one of these
// structural episodes on their own.

// repeatLookahead is how many later calls are searched for the repeat of a
// call: the calls between are other work, not a state change.
const repeatLookahead = 3

// changesState says an event between two identical calls can make the second
// one new work: an edit, a status or lifecycle change, a new artifact, an error
// or the operator speaking.
func changesState(event *domain.RunEvent) bool {
	if event == nil {
		return false
	}
	if len(changedPaths(event)) > 0 {
		return true
	}
	switch event.EventType {
	case domain.EventTypeStatus, domain.EventTypeLifecycle, domain.EventTypeArtifact, domain.EventTypeError, domain.EventTypeGoalStatusChanged:
		return true
	}
	switch data := event.Data.(type) {
	case *domain.MessageEventData:
		return strings.EqualFold(data.Role, "user")
	case *domain.ToolCallEventData:
		name := strings.ToLower(data.ToolName)
		if strings.Contains(name, "edit") || strings.Contains(name, "write") || strings.Contains(name, "patch") {
			return true
		}
		for _, value := range data.Input {
			if text, ok := value.(string); ok && strings.Contains(text, "*** Begin Patch") {
				return true
			}
		}
	}
	return false
}

// isWaiting is a call that only passes time: a wait or sleep, as a tool or a command.
func isWaiting(fact InvocationFact) bool {
	tool := strings.ToLower(fact.ToolName)
	return fact.Capability == "wait" || strings.Contains(tool, "sleep") || fact.CommandPath == "sleep"
}

// detectRepeatedWork finds a command run again within a few calls of itself
// when nothing changed in between: no edit, status change or operator message,
// no wait between (a polling loop), the first run did not fail while the second
// passes (that is a fix), and the second began after the first returned (one
// request emitted twice is not a repeat). Status and listing commands repeat by
// design.
func detectRepeatedWork(ctx EpisodeDetectorContext) []FrictionEpisode {
	var out []FrictionEpisode
	pos := eventPositions(ctx.Events)
	for i, a := range ctx.Facts {
		if !comparableFingerprintFact(a) || isPollingCommand(a.CommandPath) || isWaiting(a) {
			continue
		}
		for j := i + 1; j < len(ctx.Facts) && j <= i+repeatLookahead; j++ {
			b := ctx.Facts[j]
			if b.CallEventID == a.CallEventID {
				continue
			}
			if isWaiting(b) {
				break
			}
			if !comparableFingerprintFact(b) || b.Fingerprint != a.Fingerprint {
				continue
			}
			if a.Outcome == OutcomeFailure && b.Outcome == OutcomeSuccess {
				break
			}
			stateChanged := false
			for _, event := range eventsBetween(ctx.Events, pos, a.CallEventID, b.CallEventID) {
				stateChanged = stateChanged || changesState(event)
			}
			if !stateChanged {
				out = append(out, newEpisode("repeated-work", a, b, ctx.EventsByID, ctx.Events))
			}
			break
		}
	}
	return out
}

// retryCalls is how many calls after a failure may hold its retry.
const retryCalls = 2

// detectRetryAfterFailure finds a failed command called again, by the same
// program, in one of the next two calls. A retry may be one segment of a longer
// command, so the segments of the call that follows are all candidates.
func detectRetryAfterFailure(ctx EpisodeDetectorContext) []FrictionEpisode {
	var out []FrictionEpisode
	for i, before := range ctx.Facts {
		if !failed(before) || before.Executable == "" {
			continue
		}
		calls, last := 0, before.CallEventID
		for _, after := range ctx.Facts[i+1:] {
			if after.CallEventID != last {
				calls++
				last = after.CallEventID
			}
			if calls > retryCalls {
				break
			}
			if after.CallEventID == before.CallEventID || !sameProgram(before, after) || (after.Outcome != OutcomeSuccess && after.Outcome != OutcomeFailure) {
				continue
			}
			e := newEpisode("retry-after-failure", before, after, ctx.EventsByID, ctx.Events)
			e.EvidenceEventIDs = []string{before.CallEventID, before.ResultEventID, after.CallEventID, after.ResultEventID}
			e.FailedJoinedCalls = 1
			out = append(out, e)
			break
		}
	}
	return out
}

// argShape is how a call spells its arguments, values aside: flags by name,
// every other word as a value. Retrying a failed command with another shape
// is changing how it is called.
func argShape(fact InvocationFact, byID map[string]*domain.RunEvent) string {
	tokens := factTokens(fact, byID)
	skip := len(strings.Fields(fact.CommandPath))
	if skip > len(tokens) {
		return ""
	}
	shape := []string{fact.CommandPath}
	valueOfFlag := false
	for _, token := range tokens[skip:] {
		switch {
		case isFlag(token):
			shape = append(shape, strings.SplitN(token, "=", 2)[0])
			valueOfFlag = !strings.Contains(token, "=")
		case valueOfFlag:
			valueOfFlag = false
		default:
			shape = append(shape, "_")
		}
	}
	return strings.Join(shape, " ")
}

// sameGroup is the same program and first subcommand word (`test-genie runs`):
// the commands of one group take the same kind of arguments.
func sameGroup(a, b InvocationFact) bool {
	group := func(f InvocationFact) string {
		return strings.Join(strings.Fields(f.CommandPath)[:min(2, len(strings.Fields(f.CommandPath)))], " ")
	}
	return sameProgram(a, b) && group(a) == group(b)
}

// detectFlagHunting finds a project command that failed and was then called
// again in its group with another argument shape: the agent is guessing how to
// call it.
func detectFlagHunting(ctx EpisodeDetectorContext) []FrictionEpisode {
	var out []FrictionEpisode
	skipTo := 0
	for i, first := range ctx.Facts {
		if i < skipTo || first.CommandPath == "" || first.Ownership != OwnershipResolved || !failed(first) || isHelpProbe(first, ctx.EventsByID) {
			continue
		}
		shapes := map[string]bool{argShape(first, ctx.EventsByID): true}
		tried := []InvocationFact{first}
		for j := i + 1; j < len(ctx.Facts) && j <= i+detectorWindow; j++ {
			current := ctx.Facts[j]
			if !sameGroup(first, current) || isHelpProbe(current, ctx.EventsByID) {
				continue
			}
			shapes[argShape(current, ctx.EventsByID)] = true
			tried = append(tried, current)
			if len(shapes) >= 2 {
				e := newEpisode("flag-hunting", first, current, ctx.EventsByID, ctx.Events)
				e.price(tried, ctx.EventsByID)
				e.CycleCount = len(shapes)
				e.EvidenceEventIDs = []string{first.CallEventID, current.CallEventID}
				out = append(out, e)
				skipTo = j + 1 // the retries are this episode's, not new ones
				break
			}
		}
	}
	return out
}

// detectAbandonedCommandFamily finds a failure of a Vrooli command that no
// later call of its scenario ran to completion. A search that found nothing still ran.
func detectAbandonedCommandFamily(ctx EpisodeDetectorContext) []FrictionEpisode {
	var out []FrictionEpisode
	for i, fail := range ctx.Facts {
		if fail.Ownership != OwnershipResolved || !failed(fail) {
			continue
		}
		last := fail
		resolved := false
		for j := i + 1; j < len(ctx.Facts) && j <= i+detectorWindow; j++ {
			candidate := ctx.Facts[j]
			if !sameProgram(fail, candidate) {
				continue
			}
			last = candidate
			if candidate.Outcome == OutcomeSuccess || candidate.Outcome == OutcomeNoMatch {
				resolved = true
				break
			}
		}
		if !resolved {
			e := newEpisode("abandoned-command-family", fail, last, ctx.EventsByID, ctx.Events)
			e.FailedJoinedCalls = 1
			out = append(out, e)
		}
	}
	return out
}

// maxRereadEvents bounds how far apart two reads may be and still be a re-read.
const maxRereadEvents = 40

// readPathOf is the file a read fact names: the read tool's path, else the last
// operand of the command.
func readPathOf(fact InvocationFact, byID map[string]*domain.RunEvent) string {
	if path := ReadPath(toolCallData(byID[fact.CallEventID])); path != "" {
		return path
	}
	tokens := factTokens(fact, byID)
	for i := len(tokens) - 1; i > 0; i-- {
		if !isFlag(tokens[i]) {
			return tokens[i]
		}
	}
	return ""
}

// volatileRead is a source that changes by itself, so reading it again is
// following it: process and kernel files, logs,
// JSON lines, and a followed tail or head.
func volatileRead(path string, tokens []string) bool {
	switch {
	case strings.HasPrefix(path, "/proc/"), strings.HasPrefix(path, "/sys/"), strings.HasSuffix(path, ".log"), strings.HasSuffix(path, ".jsonl"):
		return true
	}
	for _, token := range tokens {
		if token == "-f" || token == "-F" || token == "--follow" {
			return true
		}
	}
	return false
}

// readRange is the lines a read covers when its command says so: `sed -n
// A,Bp` and `head -n N`. A read with no stated range covers the whole file.
func readRange(tokens []string) (lo, hi int) {
	lo, hi = 1, 1<<30
	for i, token := range tokens {
		switch {
		case token == "-n" && i+1 < len(tokens) && tokens[0] == "sed":
			if a, b, ok := strings.Cut(strings.TrimSuffix(strings.Trim(tokens[i+1], `'"`), "p"), ","); ok {
				x, errA := strconv.Atoi(a)
				y, errB := strconv.Atoi(b)
				if errA == nil && errB == nil {
					return x, y
				}
			}
		case tokens[0] == "head" && (token == "-n" && i+1 < len(tokens) || len(token) > 1 && token[0] == '-' && token[1] >= '0' && token[1] <= '9'):
			value := strings.TrimPrefix(token, "-")
			if token == "-n" {
				value = tokens[i+1]
			}
			if n, err := strconv.Atoi(value); err == nil {
				return 1, n
			}
		}
	}
	return lo, hi
}

// factRange is the lines a read fact covers: a read tool's offset and limit,
// else what its command states.
func factRange(fact InvocationFact, byID map[string]*domain.RunEvent) (lo, hi int) {
	if call := toolCallData(byID[fact.CallEventID]); call != nil && ReadPath(call) != "" {
		lo, hi = 1, 2000 // a read tool's default window
		if offset, ok := call.Input["offset"].(float64); ok && offset >= 1 {
			lo = int(offset)
		}
		if limit, ok := call.Input["limit"].(float64); ok && limit >= 1 {
			hi = lo + int(limit) - 1
		} else {
			hi = lo + 1999
		}
		return lo, hi
	}
	return readRange(factTokens(fact, byID))
}

// mostlyOverlap says one range covers at least half of the shorter of the two.
func mostlyOverlap(aLo, aHi, bLo, bHi int) bool {
	overlap := min(aHi, bHi) - max(aLo, bLo) + 1
	shorter := min(aHi-aLo, bHi-bLo) + 1
	return overlap > 0 && 2*overlap >= shorter
}

// detectReadThenReread finds a file read again with no change to it between:
// the same path, however the two reads were spelled, within one working stretch
// and 40 events, with no compaction between, and only when the first read
// succeeded. Two reads whose stated line ranges barely overlap read different text.
func detectReadThenReread(ctx EpisodeDetectorContext) []FrictionEpisode {
	var out []FrictionEpisode
	stretch := workingStretches(ctx.Events)
	position := eventPositions(ctx.Events)
	for i, first := range ctx.Facts {
		if first.Capability != "file-read" || resultReportedFailure(first, ctx.EventsByID) {
			continue // a read that failed (a script error) is retried, not repeated
		}
		firstTokens := factTokens(first, ctx.EventsByID)
		firstPath := readPathOf(first, ctx.EventsByID)
		if firstPath == "" || volatileRead(firstPath, firstTokens) {
			continue
		}
		firstLo, firstHi := factRange(first, ctx.EventsByID)
		for j := i + 1; j < len(ctx.Facts); j++ {
			second := ctx.Facts[j]
			if second.Capability != "file-read" || readPathOf(second, ctx.EventsByID) != firstPath {
				continue
			}
			// A read after a long idle gap starts a new piece of work, not a re-read.
			if a, ok := stretch[first.CallEventID]; ok && stretch[second.CallEventID] != a {
				break
			}
			if position[second.CallEventID]-position[first.CallEventID] > maxRereadEvents {
				break
			}
			between := eventsBetween(ctx.Events, position, first.CallEventID, second.CallEventID)
			if compactedBetween(between) {
				break
			}
			// A changed file turns the second read into deliberate verification.
			if fileChangedAt(between, firstPath) {
				continue
			}
			if secondLo, secondHi := factRange(second, ctx.EventsByID); !mostlyOverlap(firstLo, firstHi, secondLo, secondHi) {
				continue
			}
			e := newEpisode("read-then-reread", first, second, ctx.EventsByID, ctx.Events)
			e.CycleCount = 2
			out = append(out, e)
			break
		}
	}
	return out
}

func fileChangedAt(events []*domain.RunEvent, path string) bool {
	for _, event := range events {
		if _, changed := changedPaths(event)[path]; changed {
			return true
		}
	}
	return false
}

// workSpans answers whether two events lie in one working stretch: no gap over
// workingStretchGap between consecutive events and no boundary event (park,
// wake, stop) between them. Events it does not know count as one stretch.
type workSpans struct {
	stretch map[string]int
}

func newWorkSpans(events []*domain.RunEvent) workSpans {
	out := make(map[string]int, len(events))
	stretch := 0
	var prev *domain.RunEvent
	for _, event := range events {
		if event == nil {
			continue
		}
		if prev != nil && (event.Timestamp.Sub(prev.Timestamp) > workingStretchGap || isBoundaryEvent(event) || isBoundaryEvent(prev)) {
			stretch++
		}
		out[event.ID.String()] = stretch
		prev = event
	}
	return workSpans{stretch: out}
}

func (w workSpans) sameStretch(a, b string) bool {
	sa, okA := w.stretch[a]
	sb, okB := w.stretch[b]
	return !okA || !okB || sa == sb
}

// workingStretchGap is the idle time that ends a working stretch.
const workingStretchGap = 30 * time.Minute

// workingStretches numbers the working stretches of ordered events by event
// ID: a gap over workingStretchGap between consecutive events starts the next.
func workingStretches(events []*domain.RunEvent) map[string]int {
	out := make(map[string]int, len(events))
	stretch := 0
	for i, event := range events {
		if event == nil {
			continue
		}
		if i > 0 && events[i-1] != nil && event.Timestamp.Sub(events[i-1].Timestamp) > workingStretchGap {
			stretch++
		}
		out[event.ID.String()] = stretch
	}
	return out
}

func toolCallData(event *domain.RunEvent) *domain.ToolCallEventData {
	if event == nil {
		return nil
	}
	call, _ := event.Data.(*domain.ToolCallEventData)
	return call
}
