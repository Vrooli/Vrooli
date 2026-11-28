package toolfacts

import (
	"encoding/json"
	"sort"
	"strconv"
	"strings"

	"github.com/google/uuid"

	"agent-manager/internal/domain"
	"agent-manager/internal/runsignal"
)

// Extractor reads the tool events of one run, oldest first, and accumulates
// its Facts. It holds a chunk of events at a time: events are released through
// runsignal.DeriveInvocationFacts at the first moment no call awaits its
// result, so a call is always judged with its own result and the nested command
// results Codex code-mode records around it. Counters are small integers keyed
// by 64-bit hashes, so memory does not grow with the length of the run.
//
// An Extractor is for one run and one goroutine.
type Extractor struct {
	rules Rules
	facts Facts

	buf         []*domain.RunEvent
	openIDs     map[string]bool
	chunkIDs    map[string]bool
	openOrdinal int

	pathCounts    map[uint64]uint32
	commandCounts map[uint64]uint32
	failedTools   map[string]bool

	all, nonWait repeatTracker

	parkKey    uint64
	parkStreak int

	goalStatuses map[string]int

	// recent holds the last MaxTail calls so the trailing windows can be
	// computed at the end without a second pass.
	recent  []callRecord
	recentN int
}

// callRecord is what a trailing window needs of one call.
type callRecord struct {
	identity uint64
	commands []uint64
}

// New returns an Extractor that applies the given rules.
func New(rules Rules) *Extractor {
	return &Extractor{
		rules:         rules,
		openIDs:       map[string]bool{},
		chunkIDs:      map[string]bool{},
		pathCounts:    map[uint64]uint32{},
		commandCounts: map[uint64]uint32{},
		failedTools:   map[string]bool{},
		goalStatuses:  map[string]int{},
		all:           newRepeatTracker(rules.UnchangedWindow),
		nonWait:       newRepeatTracker(rules.UnchangedWindow),
		recent:        make([]callRecord, rules.MaxTail()),
	}
}

// Add reads one event. Events that are not tool calls or tool results are
// ignored, so a caller can pass a run's whole event stream.
func (e *Extractor) Add(event *domain.RunEvent) {
	if event == nil {
		return
	}
	switch data := event.Data.(type) {
	case *domain.ToolCallEventData:
		if id := data.ToolCallID; id != "" {
			if e.openIDs[id] {
				// A second event for a call still in flight is its progress
				// update, not another call.
				return
			}
			if e.chunkIDs[id] {
				// Codex exec reuses ids across turns (item_10 again after
				// item_10 was answered). The invocation readers key a chunk's
				// calls by id, so a reused id starts a new chunk.
				e.release()
			}
			e.openIDs[id], e.chunkIDs[id] = true, true
		} else {
			e.openOrdinal++
		}
	case *domain.ToolResultEventData:
		switch {
		case data.ToolCallID != "" && e.openIDs[data.ToolCallID]:
			delete(e.openIDs, data.ToolCallID)
		case data.ToolCallID == "" && e.openOrdinal > 0:
			e.openOrdinal--
		}
	default:
		return
	}
	// The invocation readers key events by id; a parser that leaves it empty
	// would make every event the same event.
	if event.ID == uuid.Nil {
		event.ID = uuid.New()
	}
	e.facts.EventsRead++
	e.buf = append(e.buf, event)
	quiet := len(e.openIDs) == 0 && e.openOrdinal == 0
	if quiet && len(e.buf) >= e.rules.ChunkMinEvents {
		e.flush()
	} else if len(e.buf) >= e.rules.ChunkMaxEvents {
		e.release()
	}
}

// release ends the chunk before every call in it was answered. The calls
// still waiting are given up: their results, if they ever arrive, are judged
// on their own.
func (e *Extractor) release() {
	e.flush()
	clear(e.openIDs)
	e.openOrdinal = 0
}

// Pending is the number of events held, not yet released into the facts.
func (e *Extractor) Pending() int { return len(e.buf) }

// Facts releases the pending events and returns the facts of everything added.
// The Extractor may be used again afterwards.
func (e *Extractor) Facts() Facts {
	e.flush()
	out := e.facts
	out.UnchangedRepeats, out.LongestUnchangedRun = e.all.repeats, e.all.longest
	out.NonWaitUnchangedRepeats, out.NonWaitLongestUnchangedRun = e.nonWait.repeats, e.nonWait.longest
	out.Tails = e.tails()
	if len(e.goalStatuses) > 0 {
		out.GoalStatuses = make(map[string]int, len(e.goalStatuses))
		for status, count := range e.goalStatuses {
			out.GoalStatuses[status] = count
		}
	}
	out.FailedTools = make([]string, 0, len(e.failedTools))
	for name := range e.failedTools {
		out.FailedTools = append(out.FailedTools, name)
	}
	sort.Strings(out.FailedTools)
	if len(out.FailedTools) == 0 {
		out.FailedTools = nil
	}
	return out
}

func (e *Extractor) flush() {
	if len(e.buf) == 0 {
		return
	}
	e.consume(e.buf)
	clear(e.buf)
	e.buf = e.buf[:0]
	clear(e.chunkIDs)
}

// consume derives the facts of one chunk of events.
func (e *Extractor) consume(events []*domain.RunEvent) {
	outputOf := map[string]string{}
	for _, fact := range runsignal.DeriveInvocationFacts(events) {
		// A command segment with a process exit code that failed. A search or
		// comparison that found nothing (exit 1 of rg, grep, diff, test) is its
		// own outcome, not a failure.
		if fact.Outcome == runsignal.OutcomeFailure && fact.ExitCode != nil {
			e.facts.FailedCommands++
		}
		if _, seen := outputOf[fact.CallEventID]; !seen {
			outputOf[fact.CallEventID] = fact.OutputEventID
		}
	}
	byID := make(map[string]*domain.RunEvent, len(events))
	for _, event := range events {
		byID[event.ID.String()] = event
	}
	for _, event := range events {
		switch data := event.Data.(type) {
		case *domain.ToolCallEventData:
			e.call(data, byID[outputOf[event.ID.String()]])
		case *domain.ToolResultEventData:
			e.result(data)
		}
	}
}

func (e *Extractor) result(data *domain.ToolResultEventData) {
	if data.OutcomeKnown != nil && !*data.OutcomeKnown {
		return
	}
	if data.Success && strings.TrimSpace(data.Error) == "" {
		return
	}
	e.facts.ToolFailures++
	if name := strings.TrimSpace(data.ToolName); name != "" {
		e.failedTools[name] = true
	}
	if text := strings.TrimSpace(data.Error); text != "" {
		e.facts.LastError = boundText(text, 400)
	}
}

func (e *Extractor) call(data *domain.ToolCallEventData, output *domain.RunEvent) {
	e.facts.ToolCalls++
	pipelines := runsignal.CallPipelines(data)
	e.facts.Commands += len(pipelines)

	path := e.inputPath(data.Input)
	texts := make([]string, 0, len(pipelines))
	hashes := make([]uint64, 0, len(pipelines))
	waiting := false
	for _, pipeline := range pipelines {
		text := pipelineText(pipeline)
		if text == "" {
			continue
		}
		texts = append(texts, text)
		hashed := hashString(text)
		hashes = append(hashes, hashed)
		if e.bump(e.commandCounts, hashed) {
			e.facts.RepeatedCommands++
		}
		if e.rules.wait(text) {
			waiting = true
			e.facts.WaitCalls++
		}
		if key, parks := e.rules.park(text); parks {
			e.park(hashString(key))
		}
		if path == "" {
			// A shell command carries its target in its words; without
			// extracting it, repeated reads are invisible.
			if target := commandTarget(pipeline.Words); target != "" && e.bump(e.pathCounts, hashString(target)) {
				e.facts.RepeatedPaths++
			}
		}
	}
	if path != "" && e.bump(e.pathCounts, hashString(path)) {
		e.facts.RepeatedPaths++
	}

	// A call's identity is what it asked and how it was answered. An
	// unanswered call has none: it can neither repeat nor be repeated.
	var identity uint64
	if resultHash, ok := e.resultIdentity(output); ok {
		identity = hashString(e.callIdentity(data, texts)) ^ (resultHash * 0x9e3779b97f4a7c15)
		if identity == 0 {
			identity = 1
		}
	}
	if status, updates := e.rules.GoalUpdate.status(data); updates {
		e.facts.GoalUpdates++
		e.facts.LastGoalStatus, e.facts.CallsAfterLastGoalUpdate = status, 0
		if status != "" {
			e.goalStatuses[status]++
		}
	} else if e.facts.GoalUpdates > 0 {
		e.facts.CallsAfterLastGoalUpdate++
	}
	e.all.observe(identity)
	if !waiting {
		e.nonWait.observe(identity)
	}
	if len(e.recent) > 0 {
		e.recent[e.recentN%len(e.recent)] = callRecord{identity: identity, commands: hashes}
	}
	e.recentN++
}

// park records one park call. The streak is of consecutive parks (other work
// between them does not break it) on the same key: a run that wakes and parks
// on the same thing again, with nothing resolved.
func (e *Extractor) park(key uint64) {
	e.facts.ParkCalls++
	if e.parkStreak > 0 && key == e.parkKey {
		e.parkStreak++
	} else {
		e.parkKey, e.parkStreak = key, 1
	}
	if e.parkStreak > e.facts.LongestSameParkStreak {
		e.facts.LongestSameParkStreak = e.parkStreak
	}
}

// bump counts one occurrence and reports whether it was a repeat.
func (e *Extractor) bump(counts map[uint64]uint32, key uint64) bool {
	counts[key]++
	return counts[key] > 1
}

// tails computes the repeat facts of the final N calls, each window on its own
// as if the run began there.
func (e *Extractor) tails() []TailFacts {
	if len(e.rules.TailCalls) == 0 {
		return nil
	}
	held := min(e.recentN, len(e.recent))
	out := make([]TailFacts, 0, len(e.rules.TailCalls))
	for _, size := range e.rules.TailCalls {
		size = min(size, held)
		tracker := newRepeatTracker(e.rules.UnchangedWindow)
		commands := map[uint64]uint32{}
		tail := TailFacts{Window: size}
		for i := e.recentN - size; i < e.recentN; i++ {
			record := e.recent[i%len(e.recent)]
			tracker.observe(record.identity)
			for _, hashed := range record.commands {
				if commands[hashed]++; commands[hashed] > 1 {
					tail.RepeatedCommands++
				}
			}
		}
		tail.Calls = size
		tail.UnchangedRepeats, tail.LongestUnchangedRun = tracker.repeats, tracker.longest
		out = append(out, tail)
	}
	return out
}

// repeatTracker counts calls whose identity occurred within the last window
// calls, and the longest chain of such repeats of one identity.
type repeatTracker struct {
	window  int
	calls   int
	last    map[uint64]int
	chain   map[uint64]int
	repeats int
	longest int
}

func newRepeatTracker(window int) repeatTracker {
	return repeatTracker{window: window, last: map[uint64]int{}, chain: map[uint64]int{}}
}

// observe takes the next call's identity; 0 is an unanswered call, which takes
// a place in the window but repeats nothing.
func (t *repeatTracker) observe(identity uint64) {
	t.calls++
	if identity == 0 {
		return
	}
	if last, seen := t.last[identity]; seen && t.calls-last <= t.window {
		t.repeats++
		t.chain[identity]++
		t.longest = max(t.longest, t.chain[identity])
	} else {
		t.chain[identity] = 0
	}
	t.last[identity] = t.calls
	if t.calls%(t.window*8) == 0 {
		for key, last := range t.last {
			if t.calls-last > t.window {
				delete(t.last, key)
				delete(t.chain, key)
			}
		}
	}
}

// callIdentity is what a call asked, normalized: the commands it ran (the tool
// is beside the point: bash, exec and a shell tool run the same command), or,
// for a call that runs none, the tool and its canonical input.
func (e *Extractor) callIdentity(data *domain.ToolCallEventData, commands []string) string {
	if len(commands) > 0 {
		return e.rules.rewrite(strings.Join(commands, " ; "), appliesArgs)
	}
	input, _ := json.Marshal(data.Input)
	return e.rules.rewrite(strings.ToLower(data.ToolName)+"\x00"+string(input), appliesArgs)
}

// resultIdentity says whether two results of the same call are the same
// result: its outcome and a hash of its normalized, whitespace-collapsed text.
func (e *Extractor) resultIdentity(event *domain.RunEvent) (uint64, bool) {
	if event == nil {
		return 0, false
	}
	data, ok := event.Data.(*domain.ToolResultEventData)
	if !ok {
		return 0, false
	}
	parts := make([]string, 0, 2+len(data.OutputBlocks))
	parts = append(parts, data.Error, data.Output)
	for _, block := range data.OutputBlocks {
		parts = append(parts, block.Text)
	}
	text := e.rules.rewrite(sampleParts(parts, e.rules.ResultHeadBytes, e.rules.ResultTailBytes), appliesResult)
	exit := "-"
	if data.ExitCode != nil {
		exit = strconv.Itoa(*data.ExitCode)
	}
	outcome := "f"
	if data.Success {
		outcome = "s"
	}
	return hashFields(outcome+exit, text), true
}

// inputPath is the file a call's own input names.
func (e *Extractor) inputPath(input map[string]any) string {
	for _, key := range e.rules.PathKeys {
		if value, ok := input[key].(string); ok && strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

// pipelineText is a command as the agent ran it: its words and the filters its
// output went through. Two commands that differ in any word are different work.
func pipelineText(pipeline runsignal.Pipeline) string {
	parts := []string{strings.Join(pipeline.Words, " ")}
	for _, filter := range pipeline.Filters {
		parts = append(parts, strings.Join(filter, " "))
	}
	return strings.TrimSpace(strings.Join(parts, " | "))
}

// commandTarget picks one path-like word out of a command. The first absolute
// or explicitly relative path is the best available stand-in for the file an
// action touched; commands with no path return "". The program itself (the
// first word) is never a file the action read.
func commandTarget(words []string) string {
	if len(words) > 1 {
		words = words[1:]
	}
	for _, word := range words {
		for _, token := range strings.Fields(word) {
			token = strings.Trim(token, `"'\;,()[]{}`)
			if strings.HasPrefix(token, "/") && len(token) > 1 {
				return token
			}
			if strings.HasPrefix(token, "./") || strings.HasPrefix(token, "../") {
				return token
			}
		}
	}
	return ""
}
