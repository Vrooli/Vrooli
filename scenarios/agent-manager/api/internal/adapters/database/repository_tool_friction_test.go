package database

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"agent-manager/internal/invocationreadmodel"
	"agent-manager/internal/runsignal"
	"agent-manager/internal/tokenaccounting"
	"agent-manager/internal/toolfriction"
)

// ToolCalls reads a window's projected command facts and file reads (never
// code, never a fact the profiles do not read) with the features stored at
// projection, without touching run_events.
func TestToolCallsReadStoredFeaturesOfTheWindow(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()
	now := time.Date(2026, 10, 30, 6, 0, 0, 0, time.UTC)
	stored := toolfriction.Features{
		Command: "vrooli scenario test x | head", Key: "", Filtered: true, JSONPaths: []string{".items[].id", ".state"}, Truncated: true,
		Signature: "Error: test-genie admission is saturated", ErrorLine: "Error: test-genie admission is saturated", ShapeGuess: true,
		Notice: "park unavailable", NoticeLine: "park unavailable, waiting inline", BypassOwner: "agent-manager", BypassMethod: toolfriction.BypassHTTP,
	}
	fact := func(call, capability, path, ownership, outcome string, segment int, at time.Time, tool *toolfriction.Features) invocationreadmodel.Fact {
		return invocationreadmodel.Fact{InvocationFact: runsignal.InvocationFact{Version: runsignal.InvocationFactVersion, CallEventID: call, ToolName: "Bash", Capability: capability, CommandPath: path, Executable: "vrooli", Ownership: ownership, Outcome: outcome, SegmentIndex: segment, Fingerprint: call, Availability: "available", ResultTokens: 40, ResidencyRequests: 3}, RunID: "run", OccurredAt: at, TimeBasis: "call_event", ProfileID: "p", RunnerType: "claude-code", Model: "m", Tag: "t", RunStatus: "complete", Tool: tool}
	}
	facts := []invocationreadmodel.Fact{
		fact("call-1", "shell", "vrooli scenario test", runsignal.OwnershipResolved, runsignal.OutcomeFailure, 0, now, &stored),
		fact("call-1", "shell", "head", runsignal.OwnershipExternal, runsignal.OutcomeUnknown, 1, now, &toolfriction.Features{}),
		fact("call-2", runsignal.KindFileRead, "", runsignal.OwnershipNotACommand, runsignal.OutcomeUnknown, 0, now, &toolfriction.Features{BypassOwner: "web-console", BypassMethod: toolfriction.BypassState}),
		fact("call-3", "code", "", runsignal.OwnershipNotACommand, runsignal.OutcomeUnknown, 0, now, nil),
		fact("call-4", "shell", "ls", runsignal.OwnershipExternal, runsignal.OutcomeSuccess, 0, now.Add(-48*time.Hour), &toolfriction.Features{}),
	}
	facts[0].SkillID, facts[0].ResidencySegment = "idea-workshop", 2
	facts[0].ResultEventID = "result-1" // the nested result the segment's outcome and duration come from
	repo := &invocationReadModelRepository{db: db}
	if err := repo.Replace(ctx, facts, invocationreadmodel.Watermark{RunID: "run", LastEventID: "result-1", LastEventAt: now, ClassifierVersion: runsignal.InvocationFactVersion, ProjectedAt: now}); err != nil {
		t.Fatal(err)
	}
	window, err := NewUsageAttributionRepository(db).ToolCalls(ctx, now.Add(-time.Hour), now.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	calls := window.Calls
	if len(calls) != 3 {
		t.Fatalf("calls = %d, want the two command facts and the file read", len(calls))
	}
	if calls[0].ResultEventID != "result-1" || calls[1].ResultEventID != "" {
		t.Fatalf("result event ids = %q, %q; want the stored one and none", calls[0].ResultEventID, calls[1].ResultEventID)
	}
	first := calls[0]
	if first.CommandPath != "vrooli scenario test" || first.ResultTokens != 40 || first.TokenBasis != tokenaccounting.BasisEstimated || first.Residency != 3 || first.SkillID != "idea-workshop" || first.ResidencySegment != 2 || first.Owner != "vrooli" || !first.Failed() {
		t.Fatalf("first = %+v", first)
	}
	if !reflect.DeepEqual(first.Features, stored) {
		t.Errorf("features = %+v, want %+v", first.Features, stored)
	}
	if calls[2].Features.BypassOwner != "web-console" {
		t.Errorf("file read features = %+v", calls[2].Features)
	}
}

// A call is marked as read by an older parser when its run's transcript record
// names a version below the harness's current one (AM-12a); a run with no
// transcript record is marked as having none, not as older: its parser is
// unknown. RunToolFriction joins one run's record the same way.
func TestToolCallsMarkRunsAnOlderParserRead(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()
	now := time.Date(2026, 10, 30, 6, 0, 0, 0, time.UTC)
	runs := []struct {
		id, harness string
		version     int // -1: no transcript record
		older, none bool
	}{
		{"run-current", "codex", 1, false, false},
		{"run-older", "codex", 0, true, false},
		{"run-unrecorded", "codex", -1, false, true},
		{"run-claude", "claude-code", 0, false, false}, // the claude codec declares no version
	}
	repo := &invocationReadModelRepository{db: db}
	for _, run := range runs {
		fact := invocationreadmodel.Fact{InvocationFact: runsignal.InvocationFact{Version: runsignal.InvocationFactVersion, CallEventID: "call-" + run.id, ToolName: "Bash", Capability: "shell", CommandPath: "ls", Ownership: runsignal.OwnershipExternal, Outcome: runsignal.OutcomeSuccess, Fingerprint: run.id, Availability: "available"}, RunID: run.id, OccurredAt: now, TimeBasis: "call_event", ProfileID: "p", RunnerType: run.harness, Model: "m", Tag: "t", RunStatus: "complete", Tool: &toolfriction.Features{}}
		if err := repo.Replace(ctx, []invocationreadmodel.Fact{fact}, invocationreadmodel.Watermark{RunID: run.id, LastEventID: "e", LastEventAt: now, ClassifierVersion: runsignal.InvocationFactVersion, ProjectedAt: now}); err != nil {
			t.Fatal(err)
		}
		if run.version >= 0 {
			if _, err := db.ExecContext(ctx, `INSERT INTO transcript_files (file_key, path, root_kind, root_path, harness, run_id, parser_version, first_seen_at, indexed_at) VALUES (?, ?, 'test', '/', ?, ?, ?, ?, ?)`, run.id, "/"+run.id, run.harness, run.id, run.version, now.Format(time.RFC3339Nano), now.Format(time.RFC3339Nano)); err != nil {
				t.Fatal(err)
			}
		}
	}
	current := func(harness string) int {
		if harness == "codex" {
			return 1
		}
		return 0
	}
	window, err := NewUsageAttributionRepository(db).WithTranscriptParserVersions(current).ToolCalls(ctx, now.Add(-time.Hour), now.Add(time.Hour))
	if err != nil || len(window.Calls) != len(runs) {
		t.Fatalf("calls = %d, %v", len(window.Calls), err)
	}
	byRun := map[string][2]bool{}
	for _, call := range window.Calls {
		byRun[call.RunID] = [2]bool{call.OlderParser, call.NoTranscript}
	}
	for _, run := range runs {
		if got := byRun[run.id]; got != [2]bool{run.older, run.none} {
			t.Errorf("%s older parser, no transcript = %v, want %v, %v", run.id, got, run.older, run.none)
		}
	}
	if got := toolfriction.Analyze(window).Coverage; got.OlderParser != 1 || got.NoTranscript != 1 || !strings.Contains(got.Sentence(), "1 commands from runs with no transcript record, whose parser is unknown") {
		t.Errorf("coverage = %+v: %s", got, got.Sentence())
	}
	if score, err := NewUsageAttributionRepository(db).WithTranscriptParserVersions(current).RunToolFriction(ctx, "run-older"); err != nil || score.Commands != 1 {
		t.Errorf("one run's score = %+v, %v", score, err)
	}
	unjudged, err := NewUsageAttributionRepository(db).ToolCalls(ctx, now.Add(-time.Hour), now.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	for _, call := range unjudged.Calls {
		if call.OlderParser {
			t.Errorf("%s marked older without a parser source", call.RunID)
		}
	}
}

// Facts of an older invocation fact version were derived by older rules: the
// window does not read them (their features would be scored by rules they
// were not made for) but counts their runs and command calls, so the coverage
// line says what awaits re-derivation instead of reading it as absent.
func TestToolCallsReadOnlyTheCurrentFactVersionAndCountTheRest(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()
	now := time.Date(2026, 10, 30, 6, 0, 0, 0, time.UTC)
	repo := &invocationReadModelRepository{db: db}
	project := func(run, version string, paths ...string) {
		var facts []invocationreadmodel.Fact
		for i, path := range paths {
			call := fmt.Sprintf("%s-call-%d", run, i)
			capability := "shell"
			if path == "" {
				capability = "code"
			}
			facts = append(facts, invocationreadmodel.Fact{InvocationFact: runsignal.InvocationFact{Version: version, CallEventID: call, ToolName: "Bash", Capability: capability, CommandPath: path, Ownership: runsignal.OwnershipExternal, Outcome: runsignal.OutcomeSuccess, Fingerprint: call, Availability: "available"}, RunID: run, OccurredAt: now, TimeBasis: "call_event", ProfileID: "p", RunnerType: "codex", Model: "m", Tag: "t", RunStatus: "complete", Tool: &toolfriction.Features{}})
		}
		if err := repo.Replace(ctx, facts, invocationreadmodel.Watermark{RunID: run, LastEventID: "e", LastEventAt: now, ClassifierVersion: version, ProjectedAt: now}); err != nil {
			t.Fatal(err)
		}
	}
	project("run-current", runsignal.InvocationFactVersion, "ls", "rg x")
	project("run-old-a", "invocation-fact.v1", "ls", "rg x", "")
	project("run-old-b", "invocation-fact.v2", "git status")
	window, err := NewUsageAttributionRepository(db).ToolCalls(ctx, now.Add(-time.Hour), now.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	for _, call := range window.Calls {
		if call.RunID != "run-current" {
			t.Errorf("read a stale fact of %s", call.RunID)
		}
	}
	want := toolfriction.Stale{Runs: 2, Calls: 3, Paths: map[string]int{"ls": 1, "rg x": 1, "git status": 1}}
	if len(window.Calls) != 2 || !reflect.DeepEqual(window.Stale, want) {
		t.Fatalf("calls %d, stale %+v; want 2 current calls and 2 stale runs with 3 command calls (code excluded), one per path", len(window.Calls), window.Stale)
	}
	if got := toolfriction.Analyze(window).Coverage.Sentence(); !strings.Contains(got, "2 runs (3 calls) awaiting re-derivation; findings exclude them") {
		t.Errorf("coverage = %q", got)
	}
}

// The window's reads stay bounded by the window: the transcript records are
// searched for the runs in scope (never a scan aggregating every run's), and
// the stale count reads the window's facts once.
func TestToolCallReadsAreBoundedByTheWindow(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	plan := func(query string, args ...any) []string {
		t.Helper()
		rows, err := db.QueryContext(context.Background(), "EXPLAIN QUERY PLAN "+query, args...)
		if err != nil {
			t.Fatal(err)
		}
		defer rows.Close()
		var details []string
		for rows.Next() {
			var id, parent, unused int
			var detail string
			if err := rows.Scan(&id, &parent, &unused, &detail); err != nil {
				t.Fatal(err)
			}
			details = append(details, detail)
		}
		return details
	}
	for name, steps := range map[string][]string{
		"window": plan(toolCallsQuery(toolWindow), "v", "a", "b", "v", "a", "b"),
		"run":    plan(toolCallsQuery("f.run_id = ?"), "v", "r", "v", "r"),
	} {
		var searched bool
		for _, step := range steps {
			if strings.Contains(step, "SCAN transcript_files") {
				t.Errorf("%s: %q reads every run's transcript records: %v", name, step, steps)
			}
			searched = searched || strings.HasPrefix(step, "SEARCH transcript_files")
		}
		if !searched {
			t.Errorf("%s: transcript records are not searched by run: %v", name, steps)
		}
	}
	var factReads int
	for _, step := range plan(staleCallsQuery, "v", "file-read", "a", "b") {
		if strings.Contains(step, " f ") || strings.HasSuffix(step, " f") {
			factReads++
			if !strings.HasPrefix(step, "SEARCH") {
				t.Errorf("stale count %q is not a window search", step)
			}
		}
	}
	if factReads != 1 {
		t.Errorf("stale count reads the facts %d times, want once", factReads)
	}
}

// A run working on the scenario it reads does not bypass it: of the run's tool
// calls that name any scenario's source tree, half or more name that
// scenario's (outside its data directory). The reads of a run that works on
// another scenario stay bypasses.
func TestToolCallsMarkReadsOfTheScenarioARunWorksOn(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()
	now := time.Date(2026, 10, 30, 6, 0, 0, 0, time.UTC)
	repo := &invocationReadModelRepository{db: db}
	calls := map[string][]string{
		// develops agent-manager: 3 of its 4 calls naming a tree name agent-manager's source
		"run-dev": {`{"input":{"command":"cd /home/u/Vrooli/scenarios/agent-manager/api && go test ./..."}}`, `{"input":{"file_path":"/home/u/Vrooli/scenarios/agent-manager/api/internal/x.go"}}`, `{"input":{"command":"gofumpt -w scenarios/agent-manager/api/x.go"}}`, `{"input":{"command":"sqlite3 scenarios/agent-manager/data/agent-manager.db .tables"}}`},
		// works on test-genie and reads agent-manager's database on the way
		"run-other": {`{"input":{"command":"cd scenarios/test-genie && make test"}}`, `{"input":{"file_path":"scenarios/test-genie/api/main.go"}}`, `{"input":{"file_path":"scenarios/agent-manager/README.md"}}`, `{"input":{"command":"sqlite3 scenarios/agent-manager/data/agent-manager.db .tables"}}`},
		// reads agent-manager's database a lot while working on test-genie: its
		// data directory is not agent-manager's source
		"run-heavy": {`{"input":{"command":"sqlite3 scenarios/agent-manager/data/agent-manager.db 'select 1'"}}`, `{"input":{"command":"sqlite3 scenarios/agent-manager/data/agent-manager.db 'select 2'"}}`, `{"input":{"command":"sqlite3 scenarios/agent-manager/data/agent-manager.db 'select 3'"}}`, `{"input":{"command":"cd scenarios/test-genie && make test"}}`, `{"input":{"file_path":"scenarios/test-genie/api/main.go"}}`},
	}
	for run, payloads := range calls {
		fact := invocationreadmodel.Fact{InvocationFact: runsignal.InvocationFact{Version: runsignal.InvocationFactVersion, CallEventID: "call-" + run, ToolName: "Bash", Capability: "shell", CommandPath: "sqlite3", Ownership: runsignal.OwnershipExternal, Outcome: runsignal.OutcomeSuccess, Fingerprint: run, Availability: "available"}, RunID: run, OccurredAt: now, TimeBasis: "call_event", ProfileID: "p", RunnerType: "claude-code", Model: "m", Tag: "t", RunStatus: "complete", Tool: &toolfriction.Features{Command: "sqlite3 scenarios/agent-manager/data/agent-manager.db .tables", BypassOwner: "agent-manager", BypassMethod: toolfriction.BypassDatabase}}
		if err := repo.Replace(ctx, []invocationreadmodel.Fact{fact}, invocationreadmodel.Watermark{RunID: run, LastEventID: "e", LastEventAt: now, ClassifierVersion: runsignal.InvocationFactVersion, ProjectedAt: now}); err != nil {
			t.Fatal(err)
		}
		for i, payload := range payloads {
			if _, err := db.ExecContext(ctx, `INSERT INTO run_events (id, run_id, sequence, event_type, data) VALUES (?, ?, ?, 'tool_call', ?)`, fmt.Sprintf("%s-%d", run, i), run, i, payload); err != nil {
				t.Fatal(err)
			}
		}
	}
	window, err := NewUsageAttributionRepository(db).ToolCalls(ctx, now.Add(-time.Hour), now.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	own := map[string]bool{}
	for _, c := range window.Calls {
		own[c.RunID] = c.OwnWork
	}
	if !own["run-dev"] || own["run-other"] || own["run-heavy"] {
		t.Fatalf("own work = %v, want the agent-manager developer's read only", own)
	}
}

// One load serves every read of a slot: concurrent reads wait for the load in
// flight, back-to-back reads within the slot share it, a later slot or an
// expired entry loads again, a failed load is not kept, and a window over the
// memory bound is served but not kept.
func TestToolCallMemoSharesOneLoadPerSlot(t *testing.T) {
	ctx := context.Background()
	clock := time.Date(2026, 10, 30, 6, 0, 10, 0, time.UTC)
	memo := newToolCallMemo(time.Minute, 3, func() time.Time { return clock })
	var loads []string
	release := make(chan struct{})
	size, fail := 2, false
	read := func(_ context.Context, from, to time.Time) (toolfriction.Window, error) {
		loads = append(loads, from.Format("15:04:05")+"-"+to.Format("15:04:05"))
		<-release
		if fail {
			return toolfriction.Window{}, fmt.Errorf("locked")
		}
		return toolfriction.Window{Calls: make([]toolfriction.Call, size)}, nil
	}
	from := clock.Add(-7 * 24 * time.Hour)
	results := make(chan int, 4)
	for i := 0; i < 4; i++ {
		go func(i int) {
			w, _ := memo.load(ctx, from.Add(time.Duration(i)*time.Second), clock.Add(time.Duration(i)*time.Second), read)
			results <- len(w.Calls)
		}(i)
	}
	time.Sleep(20 * time.Millisecond)
	close(release)
	for i := 0; i < 4; i++ {
		if n := <-results; n != 2 {
			t.Fatalf("a concurrent read got %d calls", n)
		}
	}
	if len(loads) != 1 || loads[0] != "06:00:00-06:00:00" {
		t.Fatalf("loads = %v, want one load of the slot's bounds", loads)
	}
	clock = clock.Add(30 * time.Second)
	memo.load(ctx, from.Add(40*time.Second), clock, read) // the same slot, 30 s later
	if len(loads) != 1 {
		t.Fatalf("a back-to-back read loaded again: %v", loads)
	}
	clock = clock.Add(time.Minute)
	memo.load(ctx, from.Add(90*time.Second), clock, read)
	if len(loads) != 2 {
		t.Fatalf("a later slot did not load: %v", loads)
	}
	if len(memo.entries) != 1 {
		t.Errorf("the expired slot is still held: %d entries", len(memo.entries))
	}
	fail = true
	clock = clock.Add(time.Minute)
	if _, err := memo.load(ctx, from.Add(150*time.Second), clock, read); err == nil {
		t.Fatal("the failed load was not returned")
	}
	fail = false
	memo.load(ctx, from.Add(150*time.Second), clock, read)
	if len(loads) != 4 {
		t.Fatalf("a failed load was kept: %v", loads)
	}
	size = 4 // over the bound of 3 calls
	clock = clock.Add(time.Minute)
	if w, _ := memo.load(ctx, from.Add(210*time.Second), clock, read); len(w.Calls) != 4 {
		t.Fatalf("the large window was not served: %d calls", len(w.Calls))
	}
	held := 0
	for _, e := range memo.entries {
		held += len(e.window.Calls)
	}
	if held > 3 {
		t.Errorf("the memo holds %d calls, over its bound of 3", held)
	}
}

// A shared load belongs to no caller: the first caller giving up (its
// context ended) neither cancels the load nor fails the reads waiting on it.
func TestToolCallMemoLoadOutlivesTheFirstCaller(t *testing.T) {
	clock := time.Date(2026, 10, 30, 6, 0, 10, 0, time.UTC)
	memo := newToolCallMemo(time.Minute, 10, func() time.Time { return clock })
	started, release := make(chan struct{}), make(chan struct{})
	var loadErr error
	read := func(ctx context.Context, _, _ time.Time) (toolfriction.Window, error) {
		close(started)
		<-release
		loadErr = ctx.Err()
		return toolfriction.Window{Calls: make([]toolfriction.Call, 2)}, nil
	}
	from := clock.Add(-7 * 24 * time.Hour)
	first, cancel := context.WithCancel(context.Background())
	firstErr := make(chan error, 1)
	go func() {
		_, err := memo.load(first, from, clock, read)
		firstErr <- err
	}()
	<-started
	waiter := make(chan int, 1)
	go func() {
		w, err := memo.load(context.Background(), from, clock, read)
		if err != nil {
			t.Errorf("the waiter failed with its first caller: %v", err)
		}
		waiter <- len(w.Calls)
	}()
	cancel()
	if err := <-firstErr; !errors.Is(err, context.Canceled) {
		t.Fatalf("the first caller did not stop at its own cancel: %v", err)
	}
	close(release)
	if n := <-waiter; n != 2 || loadErr != nil {
		t.Fatalf("waiter got %d calls; the load saw context error %v", n, loadErr)
	}
}

// A panicking load releases its waiters with an error and is not kept: the
// key loads again instead of blocking forever.
func TestToolCallMemoPanicReleasesTheKey(t *testing.T) {
	clock := time.Date(2026, 10, 30, 6, 0, 10, 0, time.UTC)
	memo := newToolCallMemo(time.Minute, 10, func() time.Time { return clock })
	panics := true
	read := func(context.Context, time.Time, time.Time) (toolfriction.Window, error) {
		if panics {
			panic("boom")
		}
		return toolfriction.Window{Calls: make([]toolfriction.Call, 1)}, nil
	}
	from := clock.Add(-7 * 24 * time.Hour)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := memo.load(ctx, from, clock, read); err == nil || !strings.Contains(err.Error(), "panic: boom") {
		t.Fatalf("panicking load returned %v, want its panic as the error", err)
	}
	panics = false
	if w, err := memo.load(ctx, from, clock, read); err != nil || len(w.Calls) != 1 {
		t.Fatalf("the key after a panic: %d calls, %v", len(w.Calls), err)
	}
}

// Windows longer than memoMaxSpan load per read and are never held; a new
// slot of a span drops the earlier slots of that span, so a reload holds one
// copy of the week beside the one loading.
func TestToolCallMemoSkipsLongWindowsAndSupersededSlots(t *testing.T) {
	ctx := context.Background()
	clock := time.Date(2026, 10, 30, 6, 0, 10, 0, time.UTC)
	memo := newToolCallMemo(time.Hour, 1_000, func() time.Time { return clock })
	loads := 0
	read := func(context.Context, time.Time, time.Time) (toolfriction.Window, error) {
		loads++
		return toolfriction.Window{Calls: make([]toolfriction.Call, 1)}, nil
	}
	month := clock.Add(-30 * 24 * time.Hour)
	memo.load(ctx, month, clock, read)
	memo.load(ctx, month, clock, read)
	if loads != 2 || len(memo.entries) != 0 {
		t.Fatalf("a 30-day window: %d loads, %d entries held; want each read loaded and none held", loads, len(memo.entries))
	}
	week, day := clock.Add(-7*24*time.Hour), clock.Add(-24*time.Hour)
	memo.load(ctx, week, clock, read)
	memo.load(ctx, day, clock, read)
	clock = clock.Add(5 * time.Minute)
	later := clock.Add(time.Hour)
	memo.load(ctx, week.Add(time.Hour), later, read) // the week's next slot
	spans := map[time.Duration]int{}
	for key := range memo.entries {
		spans[key.to.Sub(key.from)]++
	}
	if spans[7*24*time.Hour] != 1 || spans[24*time.Hour] != 1 {
		t.Fatalf("held spans %v, want the newest week slot and the day", spans)
	}
}

// ToolOutputs reads the text fields a result holds (error, output, block
// text), never its raw JSON copies, cut at MaxOutputChars.
func TestToolOutputsReadTextFieldsOnly(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()
	long := strings.Repeat("x", MaxOutputChars+500)
	for id, payload := range map[string]string{
		"short": `{"output":"Summary: run-1","outputBlocks":[{"type":"input_text","text":"skill idea-workshop","rawJson":"{\"text\":\"RAW-COPY\"}"}],"rawOutputJson":"RAW-OUTPUT"}`,
		"long":  `{"output":"` + long + `"}`,
		"bad":   `not json`,
	} {
		if _, err := db.ExecContext(ctx, `INSERT INTO run_events (id, run_id, sequence, event_type, data) VALUES (?, 'run', ?, 'tool_result', ?)`, id, len(id), payload); err != nil {
			t.Fatal(err)
		}
	}
	got, err := NewUsageAttributionRepository(db).ToolOutputs(ctx, []string{"short", "long", "bad", "missing"})
	if err != nil {
		t.Fatal(err)
	}
	if got["short"] != "\nSummary: run-1\nskill idea-workshop" {
		t.Errorf("short = %q", got["short"])
	}
	if len(got["long"]) != MaxOutputChars {
		t.Errorf("long = %d characters, want the first %d", len(got["long"]), MaxOutputChars)
	}
	if _, ok := got["bad"]; ok || len(got) != 2 {
		t.Errorf("outputs = %v keys, want short and long", len(got))
	}
}

// A run's score carries how far its events are projected at the current fact
// version; a watermark of another version is no projection at it.
func TestRunToolFrictionSaysHowFarTheRunIsProjected(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()
	now := time.Date(2026, 10, 30, 6, 0, 0, 0, time.UTC)
	repo := &invocationReadModelRepository{db: db}
	for run, version := range map[string]string{"run-current": runsignal.InvocationFactVersion, "run-old": "invocation-fact.v1"} {
		fact := invocationreadmodel.Fact{InvocationFact: runsignal.InvocationFact{Version: version, CallEventID: "call-" + run, ToolName: "Bash", Capability: "shell", CommandPath: "ls", Ownership: runsignal.OwnershipExternal, Outcome: runsignal.OutcomeSuccess, Fingerprint: run, Availability: "available"}, RunID: run, OccurredAt: now, TimeBasis: "call_event", ProfileID: "p", RunnerType: "codex", Model: "m", Tag: "t", RunStatus: "complete", Tool: &toolfriction.Features{}}
		if err := repo.Replace(ctx, []invocationreadmodel.Fact{fact}, invocationreadmodel.Watermark{RunID: run, LastEventID: "e", LastEventAt: now, ClassifierVersion: version, ProjectedAt: now}); err != nil {
			t.Fatal(err)
		}
	}
	store := NewUsageAttributionRepository(db)
	if score, err := store.RunToolFriction(ctx, "run-current"); err != nil || !score.Through.Equal(now) || score.Commands != 1 {
		t.Errorf("current = %+v, %v", score, err)
	}
	if score, err := store.RunToolFriction(ctx, "run-old"); err != nil || !score.Through.IsZero() || score.Commands != 0 {
		t.Errorf("old = %+v, %v", score, err)
	}
}
