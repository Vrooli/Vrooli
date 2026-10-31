package toolfriction

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/runsignal"
	"agent-manager/internal/tokenaccounting"

	"github.com/google/uuid"
)

// session builds the calls of runs the way the read model hands them over:
// one fact per command with its call input and result text, observed through
// the production Extract.
type session struct {
	at      time.Time
	run     string
	sources []Source
	ports   PortOwners
}

func newSession() *session {
	return &session{at: time.Date(2026, 10, 29, 9, 0, 0, 0, time.UTC), run: "run-1"}
}

// shell runs one command line; facts are derived by the production deriver so
// command paths, ownership and outcomes are the real ones.
func (s *session) shell(command string, exit int, output string) *session {
	return s.tool("Bash", map[string]any{"command": command}, exit, output, 0)
}

func (s *session) tool(name string, input map[string]any, exit int, output string, tokens int64) *session {
	s.at = s.at.Add(time.Minute)
	id := fmt.Sprintf("call-%d", len(s.sources))
	call := &domain.ToolCallEventData{ToolName: name, ToolCallID: id, Input: input}
	ms := int64(1500)
	result := &domain.ToolResultEventData{ToolName: name, ToolCallID: id, Output: output, Success: exit == 0, ExitCode: &exit, DurationMS: &ms}
	events := []*domain.RunEvent{
		{ID: uuid.New(), EventType: domain.EventTypeToolCall, Timestamp: s.at, Data: call},
		{ID: uuid.New(), EventType: domain.EventTypeToolResult, Timestamp: s.at.Add(time.Second), Data: result},
	}
	for _, fact := range runsignal.DeriveInvocationFacts(events) {
		// Projection estimates every paired result's tokens from its text.
		fact.ResultTokens, fact.ResidencyRequests, fact.TokenBasis = tokens, 10, tokenaccounting.BasisEstimated
		s.sources = append(s.sources, Source{Fact: Fact{InvocationFact: fact, RunID: s.run, At: s.at, Runner: "codex"}, Call: call, Result: result})
	}
	return s
}

// codex runs a Codex exec call whose script runs each command through
// exec_command; every command reports its own nested result, as rollouts do.
func (s *session) codex(commands ...string) *session {
	s.at = s.at.Add(time.Minute)
	run, id := uuid.New(), fmt.Sprintf("exec-%d", len(s.sources))
	src := ""
	for _, command := range commands {
		src += fmt.Sprintf("await tools.exec_command({cmd:%q});\n", command)
	}
	events := []*domain.RunEvent{domain.NewToolCallEvent(run, "exec", id, map[string]any{"input": src})}
	for i, command := range commands {
		nested := domain.NewToolResultEvent(run, runsignal.NestedCommandTool, fmt.Sprintf("%s-%d", id, i), "ok", nil)
		exit, ms := 0, int64(800)
		data := nested.Data.(*domain.ToolResultEventData)
		data.ExitCode, data.DurationMS, data.Success = &exit, &ms, true
		record, _ := json.Marshal(map[string]any{"type": "CommandExecution", "command": []string{"/bin/bash", "-lc", command}, "exit_code": 0, "output_bytes": 400})
		data.RawOutputJSON = string(record)
		events = append(events, nested)
	}
	events = append(events, domain.NewToolResultEvent(run, "exec", id, "Script completed", nil))
	byID := map[string]*domain.RunEvent{}
	for _, event := range events {
		event.Timestamp, byID[event.ID.String()] = s.at, event
	}
	call := events[0].Data.(*domain.ToolCallEventData)
	for _, fact := range runsignal.DeriveInvocationFacts(events) {
		result, _ := byID[fact.ResultEventID].Data.(*domain.ToolResultEventData)
		s.sources = append(s.sources, Source{Fact: Fact{InvocationFact: fact, RunID: s.run, At: s.at, Runner: "codex"}, Call: call, Result: result})
	}
	return s
}

func (s *session) wait(d time.Duration) *session { s.at = s.at.Add(d); return s }

func (s *session) calls() []Call { return observeAll(s.sources, s.ports) }

// observeAll is what projection and the read do together: features extracted
// from the events, then attached to their facts.
func observeAll(sources []Source, ports PortOwners) []Call {
	features := Extract(sources, ports)
	out := make([]Call, len(sources))
	for i, source := range sources {
		out[i] = Observe(source.Fact, features[i])
	}
	return out
}

func (s *session) report() Report { return Analyze(Window{Calls: s.calls()}) }

func findingOf(t *testing.T, r Report, kind, owner string) Finding {
	t.Helper()
	for _, f := range r.Findings {
		if f.Kind == kind && f.Owner == owner {
			return f
		}
	}
	t.Fatalf("no %s finding owned by %s in %+v", kind, owner, r.Findings)
	return Finding{}
}

func profileOf(t *testing.T, r Report, path string) Profile {
	t.Helper()
	for _, p := range r.Profiles {
		if p.CommandPath == path {
			return p
		}
	}
	t.Fatalf("no profile for %q", path)
	return Profile{}
}

// One error code is one cluster whatever sentence each CLI wraps it in, and an
// error seen across several CLIs belongs to what they share: the control plane.
func TestOneErrorAcrossCLIsIsOneClusterOwnedByTheControlPlane(t *testing.T) {
	s := newSession()
	s.shell("search-hub query recall notes", 1, `Error: query "recall notes": unavailable: agent identity identity_refresh_unavailable; inspect run/session binding`)
	s.shell("prompt-manager skill read writing-standards", 1, "✅ installed CLI to /home/someone/.vrooli/bin/prompt-manager\nError: agent identity identity_refresh_unavailable; check run/session binding and owner-proof")
	s.shell("audio-tools settings provider --json", 1, "agent identity identity_refresh_unavailable; use a fresh authenticated owner session")
	f := findingOf(t, s.report(), KindFailure, "vrooli")
	if f.ErrorLine != "identity_refresh_unavailable" || f.Calls != 3 || f.CommandPath != "" || f.Unit != UnitCalls || f.Cost != 3 {
		t.Fatalf("finding = %+v", f)
	}
	for _, example := range f.Examples {
		if strings.Contains(example, "/home/") {
			t.Errorf("example %q is not redacted", example)
		}
	}
}

// A notice belongs to the CLI that printed it: a single command's own, and in a
// compound call the dominant scenario CLI rather than the control plane that
// chained it. The same notice spread over many CLIs with none above half stays
// with the control plane, unless it names its scenario (by CLI or display
// name): the Agent Manager broker warning is agent-manager's wherever printed.
func TestANoticeBelongsToTheCLIThatPrintedIt(t *testing.T) {
	const degraded = "Searched 10 corpora; 4 total hit(s) (some providers degraded — see health)"
	s := newSession()
	for i := 0; i < 3; i++ {
		s.shell("search-hub query recall "+fmt.Sprint(i), 0, degraded)
	}
	s.shell("vrooli scenario status agent-manager; search-hub query x", 0, degraded)
	for _, cli := range []string{"prompt-manager skill list", "device-control session list", "audio-tools settings provider", "test-genie runs list"} {
		s.shell(cli, 0, "Warning: the Agent Manager identity broker unavailable; continuing without it")
		s.shell(cli, 0, "Notice: shared credential cache unavailable; continuing without it")
	}
	r := s.report()
	owners := map[string]string{}
	for _, cl := range r.Clusters {
		owners[cl.Kind+" "+strings.SplitN(cl.Signature, " ", 2)[0]] = cl.Owner
	}
	if owners["degraded Searched"] != "search-hub" || owners["degraded Warning:"] != "agent-manager" || owners["degraded Notice:"] != "vrooli" {
		t.Errorf("notice owners = %v", owners)
	}
	for _, cl := range r.Clusters {
		if strings.HasPrefix(cl.Signature, "Searched") && cl.Calls != 4 {
			t.Errorf("a compound call reports its notice once: %+v", cl)
		}
	}
}

// A line that names a Vrooli CLI is that CLI's, whichever command printed it;
// one seen in only one CLI is that CLI's.
func TestALineNamingACLIBelongsToIt(t *testing.T) {
	s := newSession()
	for i := 0; i < 3; i++ {
		s.shell("vrooli scenario test agent-manager --phases programs", 1, "shared-package-lock event=waiting pid=1\n║ Timeout budget: 2m0s ║\nError: start run: resource_exhausted: test-genie admission is saturated (caller preview capacity); retry after an active run advances\nRuntime error: exit status 1")
		s.shell("agent-manager effort board", 1, fmt.Sprintf("Error: cohort watch %d not found", i))
	}
	r := s.report()
	if f := findingOf(t, r, KindFailure, "test-genie"); !strings.Contains(f.ErrorLine, "admission is saturated") || f.CommandPath != "vrooli scenario test" {
		t.Errorf("saturation = %+v", f)
	}
	if f := findingOf(t, r, KindFailure, "agent-manager"); f.ErrorLine != "Error: cohort watch <n> not found" {
		t.Errorf("one-CLI error = %+v", f)
	}
}

// A degradation a CLI reports while working (a park refused, a provider
// degraded) is a finding priced in the calls it affected: the time of an
// inline wait is the wait's by design, not the degradation's. A compound
// call's shared output counts once, and prose, lists and tables a CLI prints
// are not degradations.
func TestADegradationNoticeIsFoundWhateverTheOutcome(t *testing.T) {
	s := newSession()
	notice := "agent-manager park unavailable (park request: api error (409): cannot park Run) — waiting inline instead"
	s.shell("test-genie runs wait --json search-hub 20261030-011722-ea2cd695", 0, notice+"\nwait attached")
	s.shell("test-genie runs wait --json web-console 20261027-183802-904320f9", 1, notice+"\nphase unit failed")
	s.shell("test-genie runs wait --json agent-manager 20261028-021018-2ef95fa1", 0, notice)
	s.shell("test-genie runs wait --json search-hub 1 && vrooli scenario status search-hub", 0, notice)
	s.shell("prompt-manager skill read x", 0, "The parked turn is unavailable until the owner wakes it; this document explains why.\n- When a provider is unavailable the agent falls back instead to rg.\n| degraded | fall back instead |")
	r := s.report()
	f := findingOf(t, r, KindDegraded, "agent-manager")
	if f.Calls != 4 || f.Unit != UnitCalls || f.Cost != 4 || !strings.Contains(f.ErrorLine, "waiting inline") {
		t.Fatalf("finding = %+v", f)
	}
	for _, f := range r.Findings {
		if f.Owner == "prompt-manager" {
			t.Errorf("a document that mentions the word is not a notice: %+v", f)
		}
	}
}

// The failure rate counts only calls that succeeded or failed: no-match,
// refusals, a verify command's verdict and interrupts are not failures.
func TestFailureRateCountsOnlyDecidedCalls(t *testing.T) {
	s := newSession()
	s.shell("search-hub query a", 0, "ok")
	s.shell("search-hub query b", 1, "Error: index unavailable")
	s.shell("search-hub query c", 130, "^C")
	s.shell("rg nothing docs", 1, "")
	r := s.report()
	if p := profileOf(t, r, "search-hub query"); p.Decided != 2 || p.Failed != 1 || p.FailureRate != 0.5 {
		t.Errorf("search-hub = %+v", p)
	}
	if p := profileOf(t, r, "rg"); p.NoMatch != 1 || p.Failed != 0 {
		t.Errorf("rg = %+v", p)
	}
	verdict := observeAll([]Source{{Fact: Fact{InvocationFact: runsignal.InvocationFact{CommandPath: "agent-manager effort lint", Outcome: runsignal.OutcomeFailure, SemanticsKind: "verify", Ownership: runsignal.OwnershipResolved, Executable: "agent-manager"}}, Result: &domain.ToolResultEventData{Output: "FAIL resume-budget"}}}, nil)[0]
	if verdict.Failed() || verdict.Signature != "" {
		t.Errorf("a verify command's exit is its verdict, not a failure: %+v", verdict)
	}
	refused := observeAll([]Source{{Fact: Fact{InvocationFact: runsignal.InvocationFact{CommandPath: "vrooli locks", Outcome: runsignal.OutcomeRefused}}}}, nil)[0]
	if refused.Failed() || refused.decided() {
		t.Errorf("a refusal is not a failure: %+v", refused)
	}
}

// Shape guessing is a wrong-shape failure, or a help lookup that recovers a
// failure before a retry; reading help before the first try is not.
func TestShapeGuessingIsWrongShapeFailuresAndHelpThatRecoversOne(t *testing.T) {
	s := newSession()
	s.shell("agent-manager runs list", 1, `Error: Unknown command: runs (did you mean "run"?)`)
	s.shell("agent-manager --help", 0, "usage")
	s.shell("agent-manager run list", 0, "[]")
	s.shell("device-control session list --badflag", 1, "Error: unknown flag: --badflag")
	s.shell("program-runtime --help", 0, "usage")
	s.shell("program-runtime programs list", 0, "[]")
	r := s.report()
	if p := profileOf(t, r, "agent-manager"); p.ShapeGuesses != 2 {
		t.Errorf("agent-manager shape guesses = %d, want the failure and the help that recovered it", p.ShapeGuesses)
	}
	if p := profileOf(t, r, "device-control session list"); p.ShapeGuesses != 1 {
		t.Errorf("unknown flag = %+v", p)
	}
	if p := profileOf(t, r, "program-runtime"); p.ShapeGuesses != 0 {
		t.Errorf("help before a first try is reading the manual: %+v", p)
	}
	for _, c := range r.Clusters {
		if strings.Contains(c.Signature, "Unknown command") {
			t.Errorf("a wrong-shape failure is shape guessing, not a failure cluster: %+v", c)
		}
	}
}

// Output weight: tokens per call, context tax (tokens × requests that re-read
// them), the share cut by filters, jq paths read after --json and harness
// truncation.
func TestOutputWeightMeasuresTokensContextTaxFiltersAndPaths(t *testing.T) {
	s := newSession()
	s.tool("Bash", map[string]any{"command": "vrooli locks --json"}, 0, "Total output lines: 4000\n{...}", 70_000)
	s.tool("Bash", map[string]any{"command": "vrooli locks --json | jq '.registry_claims[].scenario'"}, 0, "[]", 9_000)
	s.tool("Bash", map[string]any{"command": "vrooli locks --json | head -20"}, 0, "{", 9_000)
	r := s.report()
	p := profileOf(t, r, "vrooli locks")
	if p.MaxTokens != 70_000 || p.MedianTokens != 9_000 || p.ContextTax != 880_000 || p.FilteredShare < 0.66 || p.TruncatedShare < 0.33 || len(p.JSONPaths) != 1 || p.JSONPaths[0] != ".registry_claims[].scenario" {
		t.Fatalf("profile = %+v", p)
	}
	f := findingOf(t, r, KindOutputWeight, "vrooli")
	if f.Unit != UnitTokens || f.Cost != 880_000 || f.CommandPath != "vrooli locks" || len(f.Examples) != 1 || f.Examples[0] != "vrooli locks --json" {
		t.Fatalf("finding = %+v", f)
	}
}

// A harness that cuts a command's output before the model reads it (Codex
// shows about 10k tokens of a 1 MiB listing) hides the weight from the result
// tokens: what the command printed still makes it heavy, and the finding says
// both sizes.
func TestOutputWeightSeesWhatWasPrintedBeyondWhatTheModelRead(t *testing.T) {
	s := newSession()
	for i := 0; i < 3; i++ {
		s.tool("Bash", map[string]any{"command": "vrooli capacity list"}, 0, "{...}", 2_000)
	}
	calls := s.mutate("vrooli capacity list", func(c *Call) { c.OutputBytes = 1 << 20 })
	var f Finding
	for _, found := range Analyze(Window{Calls: calls}).Findings {
		if found.Kind == KindOutputWeight {
			f = found
		}
	}
	if f.CommandPath != "vrooli capacity list" || f.Cost == 0 || !strings.Contains(f.Description, "printed about 262.1k tokens") {
		t.Errorf("finding = %+v", f)
	}
}

// Polling is one read command run three or more times with gaps under ten
// minutes; a longer gap starts a new series and a write command is never polled.
func TestPollingIsARepeatedReadCommandWithShortGaps(t *testing.T) {
	s := newSession()
	for i := 0; i < 9; i++ {
		s.shell("device-control session list --json", 0, `{"sessions": []}`).wait(2 * time.Minute)
	}
	s.wait(30 * time.Minute)
	s.shell("device-control session list --json", 0, `{"sessions": []}`)
	for i := 0; i < 4; i++ {
		s.shell("vrooli scenario restart agent-manager", 0, "restarted")
	}
	s.shell("sleep 30; test-genie runs status agent-manager 1", 0, "running")
	s.shell("sleep 30; test-genie runs status agent-manager 1", 0, "running")
	s.shell("sleep 30; test-genie runs status agent-manager 1", 0, "passed")
	// Different targets are not one series: each run is asked about once, and a
	// blocking wait that returned already is the wait.
	for _, id := range []string{"run-a", "run-b", "run-c", "run-d"} {
		s.shell("workflow-health validate get "+id+" --json", 0, "{}")
		s.shell("workflow-health validate wait "+id+" --timeout 10m --json", 0, "{}")
	}
	// A feed read again from a later cursor is the same target.
	for _, after := range []string{"57", "59", "60"} {
		s.shell("agent-manager effort signals eff-1 --after "+after, 0, "[]")
	}
	// A wait that timed out and is asked again on the same run is polling.
	for i := 0; i < 3; i++ {
		s.shell("workflow-health validate wait run-z --timeout 1m", 1, "Error: timed out")
	}
	r := s.report()
	if p := profileOf(t, r, "device-control session list"); p.Polled != 9 {
		t.Errorf("polled = %d, want the nine calls of the series, not the late one", p.Polled)
	}
	if p := profileOf(t, r, "vrooli scenario restart"); p.Polled != 0 {
		t.Errorf("a write command is never polled: %+v", p)
	}
	if p := profileOf(t, r, "test-genie runs status"); p.Polled != 3 || p.Wrapped != 3 {
		t.Errorf("one target re-queried inside sleep wrappers: %+v", p)
	}
	if p := profileOf(t, r, "workflow-health validate get"); p.Polled != 0 {
		t.Errorf("one get per run is not polling: %+v", p)
	}
	if p := profileOf(t, r, "workflow-health validate wait"); p.Polled != 3 {
		t.Errorf("completed waits are never polling; the repeated timed-out wait of one run is: %+v", p)
	}
	if p := profileOf(t, r, "agent-manager effort signals"); p.Polled != 3 {
		t.Errorf("a cursor is volatile, the feed is the target: %+v", p)
	}
	findingOf(t, r, KindPolling, "device-control")
}

// A bypass reaches a scenario's internals around its CLI: its database, its
// port, its state directory, whether by command or by a read tool.
func TestBypassesNameTheScenarioWhoseInternalsWereReached(t *testing.T) {
	s := newSession()
	s.ports = func(port int) string { // the production registry names API ports only
		if port == 18800 {
			return "agent-manager"
		}
		return ""
	}
	s.shell(`sqlite3 "file:data/agent-manager.db?mode=ro" "select count(*) from runs"`, 0, "3")
	s.shell(`cd ~/Vrooli && sqlite3 scenarios/storage-manager/data/storage.db .tables`, 0, "asks")
	s.shell("curl -s http://localhost:18800/api/v1/runs", 0, "[]")
	// UI ports, and health probes of an API port, are smoke checks.
	s.shell("curl -s http://localhost:35000/assets/index-ab12.css", 0, "")
	s.shell("curl -s http://localhost:18800/health", 0, "ok")
	s.shell("cat ~/.vrooli/state/vrooli/workflow-health/runs/latest.json", 0, "{}")
	s.tool("Read", map[string]any{"file_path": "/home/someone/.vrooli/state/vrooli/web-console_presentation/sessions/a.json"}, 0, "{}", 0)
	s.shell("sqlite3 /tmp/scratch.db .tables", 0, "")
	s.shell(`sqlite3 data/notes.db .tables`, 0, "")
	want := map[string]string{"agent-manager": BypassDatabase + "|" + BypassHTTP, "storage-manager": BypassDatabase, "workflow-health": BypassState, "web-console": BypassState}
	got := map[string][]string{}
	for _, c := range s.calls() {
		if c.BypassOwner != "" {
			got[c.BypassOwner] = append(got[c.BypassOwner], c.BypassMethod)
		}
	}
	if len(got) != len(want) {
		t.Fatalf("bypasses = %v, want %v", got, want)
	}
	for owner, methods := range want {
		if strings.Join(got[owner], "|") != methods {
			t.Errorf("%s bypass = %v, want %s", owner, got[owner], methods)
		}
	}
}

// An empty list is read beside its coverage: no commands means cannot tell.
func TestCoverageSaysWhatTheProfilesCouldSee(t *testing.T) {
	if got := (Report{}).Coverage.Sentence(); !strings.Contains(got, "cannot tell") {
		t.Errorf("empty coverage = %q", got)
	}
	s := newSession()
	s.shell("search-hub query a", 0, "ok")
	s.tool("exec", map[string]any{"cmd": "ls docs"}, 0, "", 0)
	s.sources[1].Fact.Outcome = runsignal.OutcomeUnknown
	if got := s.report().Coverage.Sentence(); got != "2 commands, outcome known for 50% (1 Vrooli CLI calls, 100% known)" {
		t.Errorf("coverage = %q", got)
	}
}

// Commands from runs an older parser read are counted apart: their unknown
// outcomes are a parser gap, so the coverage line says so instead of letting
// the low known share read as "nothing failed".
func TestCoverageSeparatesCommandsFromRunsAnOlderParserRead(t *testing.T) {
	s := newSession()
	s.shell("search-hub query a", 0, "ok")
	s.shell("search-hub query b", 0, "ok")
	calls := s.calls()
	if got := Analyze(Window{Calls: calls}).Coverage.Sentence(); strings.Contains(got, "older parser") {
		t.Errorf("current-parser coverage names an older parser: %q", got)
	}
	calls[1].OlderParser = true
	report := Analyze(Window{Calls: calls})
	if report.Coverage.OlderParser != 1 || report.Coverage.Commands != 2 {
		t.Fatalf("coverage = %+v", report.Coverage)
	}
	if got := report.Coverage.Sentence(); !strings.Contains(got, "; 1 commands from runs read by an older parser; outcomes there may be missing") {
		t.Errorf("coverage = %q", got)
	}
}

// Findings in different units rank by their share of that unit's total, and a
// fingerprint is stable for the same owner and problem.
func TestFindingsRankByTheirShareOfTheirUnitsTotal(t *testing.T) {
	findings := func() []Finding {
		s := newSession()
		for i := 0; i < 4; i++ {
			s.shell("swarm-manager captures create --text x", 1, "Error: capacity broker unavailable")
		}
		for i := 0; i < 3; i++ {
			s.tool("Bash", map[string]any{"command": "prompt-manager skill read big"}, 0, "# Skill", 50_000)
		}
		// Failures of tools nobody here owns do not dilute a Vrooli failure.
		for i := 0; i < 20; i++ {
			s.shell("grep nothing x", 1, "")
		}
		return s.report().Findings
	}
	first, second := findings(), findings()
	if len(first) != 2 {
		t.Fatalf("findings = %+v", first)
	}
	byKind := map[string]Finding{}
	for _, f := range first {
		byKind[f.Kind] = f
	}
	if f := byKind[KindOutputWeight]; f.CostShare != 1 {
		t.Errorf("output weight = %+v", f)
	}
	if f := byKind[KindFailure]; f.CostShare != 1 {
		t.Errorf("4 of the window's 4 Vrooli failures (not of 24 failed calls) = %+v", f)
	}
	for i := range first {
		if first[i].Fingerprint != second[i].Fingerprint {
			t.Errorf("fingerprint of %s is not stable", first[i].Kind)
		}
	}
}

// calls with a mutation applied to the ones matching a command path, for
// outcomes and durations the session builder does not set.
func (s *session) mutate(path string, change func(*Call)) []Call {
	calls := s.calls()
	for i := range calls {
		if calls[i].CommandPath == path {
			change(&calls[i])
		}
	}
	return calls
}

// A command that blocks until its target is done is not polled when it ran its
// course or its outcome is unknown; only a wait known to have failed or timed
// out and asked again is.
func TestBlockingWaitsArePolledOnlyWhenKnownToHaveFailed(t *testing.T) {
	s := newSession()
	for i := 0; i < 5; i++ {
		s.shell("workflow-health validate wait run-a --timeout 10m", 0, "{}")
	}
	unknown := s.mutate("workflow-health validate wait", func(c *Call) { c.Outcome = runsignal.OutcomeUnknown })
	if p := profileOf(t, Analyze(Window{Calls: unknown}), "workflow-health validate wait"); p.Polled != 0 || p.Unknown != 5 {
		t.Errorf("an unknown outcome is not a timeout: %+v", p)
	}
	for i := 0; i < 3; i++ {
		s.shell("workflow-health validate wait run-z --timeout 1m", 1, "Error: timed out")
	}
	if p := profileOf(t, s.report(), "workflow-health validate wait"); p.Polled != 3 {
		t.Errorf("the three waits known to have timed out on one run: %+v", p)
	}
	// Whether a command blocks is its manifest's to say, not its verb's.
	s = newSession()
	for i := 0; i < 5; i++ {
		s.shell("device-control device watch dev-1", 0, "{}")
	}
	if p := profileOf(t, s.report(), "device-control device watch"); p.Polled != 0 {
		t.Errorf("a declared blocking watch that ran its course is not polling: %+v", p)
	}
}

// A fingerprint names the problem, not its evidence: a second command path
// reporting the same failure does not reopen a dismissed or reported finding.
func TestFingerprintSurvivesASecondCommandPath(t *testing.T) {
	s := newSession()
	for i := 0; i < 3; i++ {
		s.shell("swarm-manager captures create --text x", 1, "Error: disk quota exceeded for vault")
	}
	before := findingOf(t, s.report(), KindFailure, "swarm-manager")
	for i := 0; i < 3; i++ {
		s.shell("swarm-manager captures list", 1, "Error: disk quota exceeded for vault")
	}
	after := findingOf(t, s.report(), KindFailure, "swarm-manager")
	if before.CommandPath == "" || after.CommandPath != "" || before.Fingerprint != after.Fingerprint {
		t.Errorf("fingerprint %s (%q) became %s (%q)", before.Fingerprint, before.CommandPath, after.Fingerprint, after.CommandPath)
	}
}

// Failures and notices that share a root cause are one finding that lists its
// lines, and a finding made by one run is capped and says so.
func TestRootCauseRollsUpFailuresAndNotices(t *testing.T) {
	s := newSession()
	for i := 0; i < 4; i++ {
		s.shell("device-control session list", 1, "Error: identity_refresh_unavailable")
	}
	for i := 0; i < 3; i++ {
		s.shell("agent-manager run list", 0, "Warning: the Agent Manager identity broker is unavailable; continuing without it")
	}
	for i := 0; i < 3; i++ {
		s.shell("test-genie runs wait x", 0, "agent-manager park unavailable (park: agent identity identity_refresh_unavailable) — waiting inline instead")
	}
	r := s.report()
	var rolled []Finding
	for _, f := range r.Findings {
		if len(f.Subs) > 0 && strings.Contains(f.Description, "identity") {
			rolled = append(rolled, f)
		}
	}
	if len(rolled) != 1 || len(rolled[0].Subs) < 3 || rolled[0].Calls != 10 || rolled[0].Kind != KindFailure {
		t.Fatalf("want one identity finding listing every line, got %+v", r.Findings)
	}
	// identity_refresh_unavailable is the Agent Manager broker's refusal: its
	// lines name it, so the roll-up is agent-manager's, not the control plane's.
	if rolled[0].Owner != "agent-manager" {
		t.Errorf("identity roll-up owner = %q, want agent-manager", rolled[0].Owner)
	}
}

func TestOneRunDoesNotRankAsAPattern(t *testing.T) {
	s := newSession()
	data := "sqlite3 scenarios/agent-manager/data/agent-manager.db 'select 1'"
	s.run = "run-loop"
	for i := 0; i < 60; i++ {
		s.shell(data, 0, "1")
	}
	for _, run := range []string{"run-a", "run-b", "run-c"} {
		s.run = run
		s.shell(data, 0, "1")
		s.shell(data, 0, "1")
	}
	f := findingOf(t, s.report(), KindBypass, "agent-manager")
	if f.Calls != 66 || f.Runs != 4 || f.TopRunShare < 0.9 || f.Cost != 26 {
		t.Errorf("66 calls, 60 in one run: cost counts that run at most 10x the median (2): 20+2+2+2: %+v", f)
	}
	if !strings.Contains(f.Description, "in one run") {
		t.Errorf("a concentrated finding says so: %s", f.Description)
	}
}

// A help lookup is shape friction only around a call that went wrong: after a
// failure of the program, or between two calls of the same command (a retry).
// Navigating a subcommand index by help, however often, is reading the
// manual. The unknown command agents guessed is named with the CLI's
// suggestion.
func TestHelpLookupsAreShapeFrictionOnlyAroundAFailureOrRetry(t *testing.T) {
	s := newSession()
	for i := 0; i < 6; i++ {
		s.shell("vrooli-memory journal note --text x", 0, "ok")
		s.shell("vrooli-memory journal note --help", 0, "usage: journal note")
	}
	s.shell("vrooli-memory journal note --text x", 0, "ok")
	for _, verb := range []string{"list", "show", "wait", "status", "artifacts", "findings"} {
		s.shell("test-genie runs --help", 0, "Commands: list show wait")
		s.shell("test-genie runs "+verb+" --help", 0, "usage")
		s.shell("test-genie runs "+verb+" r1", 0, "ok")
	}
	s.shell("program-runtime library run x", 1, "Error: input name is required")
	s.shell("program-runtime library run --help", 0, "usage")
	s.shell("program-runtime library run x --input name=y", 0, "ok")
	for i := 0; i < 3; i++ {
		s.shell("agent-manager runs list", 1, `Error: Unknown command: runs (did you mean "run"?)`)
	}
	r := s.report()
	if p := profileOf(t, r, "vrooli-memory journal note"); p.ShapeGuesses != 6 || p.HelpLookups != 6 {
		t.Errorf("each lookup sits between two calls of the command, a retry: %+v", p)
	}
	for _, path := range []string{"test-genie runs", "test-genie runs list"} {
		if p := profileOf(t, r, path); p.HelpLookups == 0 || p.ShapeGuesses != 0 {
			t.Errorf("navigating the help index with no failed or retried call is reading the manual: %+v", p)
		}
	}
	if p := profileOf(t, r, "program-runtime library run"); p.ShapeGuesses != 1 {
		t.Errorf("a lookup after a failure, then a retry, is a guess: %+v", p)
	}
	f := findingOf(t, r, KindShapeGuess, "agent-manager")
	if len(f.Guesses) != 1 || f.Guesses[0].Token != "runs" || f.Guesses[0].Suggestion != "run" || f.Guesses[0].Calls != 3 {
		t.Errorf("guesses = %+v", f.Guesses)
	}
}

// Latency: a query-shaped command is slow when its p90 is over the interactive
// budget and the excess is a real share of the window's command time. A wait,
// or a command that does work, is never slow for taking time.
func TestSlowQueriesAreFoundButWaitsAndWorkAreNot(t *testing.T) {
	s := newSession()
	for i := 0; i < 6; i++ {
		s.shell("device-control session list", 0, "[]")
		s.shell("test-genie runs wait x", 0, "done")
		s.shell("test-genie scenarios start x", 0, "ok")
		s.shell("vrooli scenario status x", 0, "ok")
	}
	calls := s.calls()
	for i := range calls {
		calls[i].DurationMS = 60_000
		if calls[i].CommandPath == "vrooli scenario status" {
			calls[i].DurationMS = 400
		}
	}
	r := Analyze(Window{Calls: calls})
	f := findingOf(t, r, KindSlow, "device-control")
	if f.Unit != UnitMS || f.Cost != 6*55_000 {
		t.Errorf("excess beyond the budget, not whole duration: %+v", f)
	}
	for _, f := range r.Findings {
		if f.Kind == KindSlow && f.Owner != "device-control" {
			t.Errorf("only an interactive query is slow: %+v", f)
		}
	}
}

// Secrets and home directories never leave as evidence.
func TestEvidenceIsRedacted(t *testing.T) {
	for in, want := range map[string]string{
		"curl -H 'Authorization: Bearer abc.def' http://x": "curl -H 'Authorization: <redacted> abc.def' http://x",
		"tool --token=s3cr3t /home/alice/Vrooli":           "tool --token=<redacted> ~/Vrooli",
		"echo " + strings.Repeat("Q", 50):                  "echo <redacted>",
	} {
		if got := redact(in); got != want {
			t.Errorf("redact(%q) = %q, want %q", in, got, want)
		}
	}
	if got := signature("Error: run 9d932d63-e894-473c-86d2-d0fa278a8db1 failed after 45s at /tmp/x.json"); got != "Error: run <id> failed after <n> at <path>" {
		t.Errorf("signature = %q", got)
	}
}

// A run's score is the share of its commands that failed, guessed a shape or
// polled, each call counted once; reads that succeeded and no-match greps are
// not friction.
func TestScoreRunCountsEachFrictionalCallOnce(t *testing.T) {
	at := time.Date(2026, 10, 30, 9, 0, 0, 0, time.UTC)
	call := func(i int, outcome string, f Features) Call {
		return Call{RunID: "r", CallEventID: fmt.Sprint(i), At: at.Add(time.Duration(i) * time.Minute), CommandPath: "test-genie runs show", Owner: "test-genie", Outcome: outcome, Features: f}
	}
	calls := []Call{
		call(0, runsignal.OutcomeFailure, Features{ShapeGuess: true}), // failed and guessed: one call
		call(1, runsignal.OutcomeSuccess, Features{Key: "test-genie runs show {id}"}),
		call(2, runsignal.OutcomeSuccess, Features{Key: "test-genie runs show {id}"}),
		call(3, runsignal.OutcomeSuccess, Features{Key: "test-genie runs show {id}"}), // three polls
		call(4, runsignal.OutcomeNoMatch, Features{}),
		call(5, runsignal.OutcomeUnknown, Features{}),
		{RunID: "r", At: at}, // a file read has no command path
	}
	got := ScoreRun(calls)
	if got.Commands != 6 || got.OutcomeKnown != 5 || got.Failed != 1 || got.ShapeGuesses != 1 || got.Polled != 3 || got.Frictional != 4 || got.Score != 4.0/6 {
		t.Fatalf("score = %+v, want 4 frictional of 6 commands", got)
	}
	if empty := ScoreRun(nil); empty.Commands != 0 || empty.Score != 0 {
		t.Fatalf("empty = %+v", empty)
	}
}

// A roll-up's failures and notices are shares of different totals: failures
// of the window's Vrooli CLI failed calls, notices of its Vrooli CLI calls.
// Each share names its base; the finding ranks by the larger, never by both
// kinds' calls over the failures.
func TestRootCauseRollUpStatesEachKindOverItsOwnBase(t *testing.T) {
	s := newSession()
	for i := 0; i < 4; i++ {
		s.shell("device-control session list", 1, "Error: identity_refresh_unavailable")
	}
	for i := 0; i < 6; i++ {
		s.shell("agent-manager run list", 0, "Warning: the Agent Manager identity broker is unavailable; continuing without it")
	}
	for i := 0; i < 4; i++ {
		s.shell("search-hub query x", 1, "Error: search index locked")
	}
	for i := 0; i < 10; i++ {
		s.shell("search-hub query y", 0, "ok")
	}
	r := s.report()
	var rolled Finding
	for _, f := range r.Findings {
		if f.RootCause == "identity" {
			rolled = f
		}
	}
	if len(rolled.Shares) != 3 {
		t.Fatalf("identity roll-up shares = %+v, want one per kind of harm", rolled.Shares)
	}
	failure, degraded, wait := rolled.Shares[0], rolled.Shares[1], rolled.Shares[2]
	if wait.Kind != HarmWallClock || wait.Unit != UnitMS || wait.Cost != 9_000 || wait.Base != "the window's Vrooli CLI time (36.0 s)" {
		t.Errorf("wall-clock share = %+v, want the 6 degraded calls' 9 s over the Vrooli CLI time", wait)
	}
	if failure.Kind != KindFailure || failure.Cost != 4 || failure.Share != 0.5 || failure.Base != "the window's 8 Vrooli CLI failed calls" {
		t.Errorf("failure share = %+v, want 4 of the 8 Vrooli failures", failure)
	}
	if degraded.Kind != KindDegraded || degraded.Cost != 6 || degraded.Share != 6.0/24 || degraded.Base != "the window's 24 Vrooli CLI calls" {
		t.Errorf("degraded share = %+v, want 6 of the 24 Vrooli CLI calls", degraded)
	}
	if rolled.CostShare != 0.5 {
		t.Errorf("cost share = %v, want the larger share (0.5), not 10 calls over 8 failures", rolled.CostShare)
	}
	if got := failure.Sentence(); got != "failure 4 calls = 50.0% of the window's 8 Vrooli CLI failed calls" {
		t.Errorf("sentence = %q", got)
	}
}

// A roll-up states every count in calls: a line printed by two segments of
// one call is one call of that line, the lines' counts add up to the kinds'
// uncapped counts, and a kind's share that the per-run cap cut says how many
// of how many calls it counts.
func TestRootCauseRollUpCountsCallsInEveryUnit(t *testing.T) {
	s := newSession()
	for i := 0; i < 2; i++ {
		s.shell("device-control session list", 1, "Error: identity_refresh_unavailable")
	}
	for i := 0; i < 2; i++ {
		s.shell("agent-manager run list", 0, "Warning: the Agent Manager identity broker is unavailable; continuing without it")
	}
	for _, run := range []string{"run-2", "run-3", "run-4"} {
		s.run = run
		s.shell("device-control session list", 1, "Error: identity_refresh_unavailable")
	}
	// One run's loop: 40 failures against one in each of the other runs.
	s.run = "run-5"
	for i := 0; i < 40; i++ {
		s.shell("device-control session list", 1, "Error: identity_refresh_unavailable")
	}
	// A Codex exec call that ran the command twice: two nested results of one
	// call and command path, one line each.
	calls := s.calls()
	for _, i := range []int{0, 1} {
		twice := calls[i]
		twice.ResultEventID += "-2"
		calls = append(calls, twice)
	}
	var rolled Finding
	for _, f := range Analyze(Window{Calls: calls}).Findings {
		if f.RootCause == "identity" {
			rolled = f
		}
	}
	lines := map[string]int{}
	for _, sub := range rolled.Subs {
		lines[sub.Kind] += sub.Calls
	}
	if lines[KindFailure] != 45 || lines[KindDegraded] != 2 || rolled.Calls != 47 {
		t.Fatalf("lines = %v over %d calls, want 45 failed calls (the exec call that ran it twice once each) and 2 warned", lines, rolled.Calls)
	}
	if !strings.Contains(rolled.Description, `failure "identity_refresh_unavailable" in 45 calls`) || strings.Contains(rolled.Description, " x") {
		t.Errorf("description = %q, want each line in calls", rolled.Description)
	}
	failure := rolled.Shares[0]
	if failure.Kind != KindFailure || failure.Uncapped != 45 || failure.Cost != 2+1+1+1+10 || !strings.HasPrefix(failure.Sentence(), "failure 15 of 45 calls (each run at most 10x the median run's) = ") {
		t.Errorf("failure share = %+v: %q", failure, failure.Sentence())
	}
}

// A degradation notice is a share of the Vrooli CLI calls, not of every
// command: thousands of grep and sed calls do not push it under the floor.
func TestDegradedIsAShareOfVrooliCalls(t *testing.T) {
	s := newSession()
	for i := 0; i < 5; i++ {
		s.shell("test-genie runs wait r", 0, "agent-manager park unavailable (park request: runner type is unavailable) — waiting inline instead")
		s.shell("test-genie runs list", 0, "[]")
	}
	for i := 0; i < 990; i++ {
		s.shell("rg pattern docs", 0, "match")
	}
	f := findingOf(t, s.report(), KindDegraded, "agent-manager")
	if f.CostShare != 0.5 || f.Shares[0].Base != "the window's 10 Vrooli CLI calls" {
		t.Errorf("degraded = %+v, want 5 of the 10 Vrooli CLI calls (not of 1000 commands)", f.Shares)
	}
}

// Slow time is a share of Vrooli CLI time: a long build of another tool does
// not hide it.
func TestSlowIsAShareOfVrooliTime(t *testing.T) {
	s := newSession()
	for i := 0; i < 6; i++ {
		s.shell("device-control session list", 0, "[]")
		s.shell("go build ./...", 0, "")
	}
	calls := s.calls()
	for i := range calls {
		calls[i].DurationMS = 10_000
		if calls[i].Owner == "" {
			calls[i].DurationMS = 10_000_000
		}
	}
	f := findingOf(t, Analyze(Window{Calls: calls}), KindSlow, "device-control")
	if f.Shares[0].Share != 0.5 || !strings.Contains(f.Shares[0].Base, "Vrooli CLI time") {
		t.Errorf("slow = %+v, want 30 s of the 60 s of Vrooli CLI time", f.Shares)
	}
}

// A run's cap is 10x the median of the runs that contributed: runs that hold
// the command but none of its harm (skill reads with no tokens) do not pull
// the median to zero and cap the real runs at nothing.
func TestCappedMedianIsOverContributingRuns(t *testing.T) {
	perRun := map[string]int64{"a": 0, "b": 0, "c": 0, "d": 0, "e": 0, "f": 1_000, "g": 2_000, "h": 20_000_000}
	cost, top := Capped(perRun)
	if cost != 1_000+2_000+20_000 {
		t.Errorf("cost = %d, want 1000 + 2000 + 10x the median 2000", cost)
	}
	if top < 0.99 {
		t.Errorf("top share = %v", top)
	}
	if cost, _ := Capped(map[string]int64{"a": 0, "b": 0}); cost != 0 {
		t.Errorf("no contribution costs %d", cost)
	}
}

// Polling a command whose owner declares a blocking sibling names it: the wait
// exists, agents are not using it. Siblings come from the CLI manifests, not
// from verbs: `run get` is offered `run events --follow`, never the
// `run await-result` re-read.
func TestPollingNamesTheBlockingSiblingTheManifestDeclares(t *testing.T) {
	s := newSession()
	for i := 0; i < 9; i++ {
		s.shell("agent-manager run get run-1", 0, "running")
		s.shell("device-control session list --json", 0, "[]")
	}
	r := s.report()
	if f := findingOf(t, r, KindPolling, "agent-manager"); strings.Join(f.Blocking, ",") != "agent-manager run events --follow" || f.Steered != 9 || !strings.Contains(f.Description, "although `agent-manager run events --follow` blocks") {
		t.Errorf("run get polling = %+v", f)
	}
	if f := findingOf(t, r, KindPolling, "device-control"); len(f.Blocking) != 0 || f.Steered != 0 || strings.Contains(f.Description, "although") || !strings.Contains(f.Description, "9 polled calls have no blocking sibling") {
		t.Errorf("a command with no blocking sibling = %+v", f)
	}
	// A wait on one target does not stand in for a listing beside it.
	s = newSession()
	for i := 0; i < 9; i++ {
		s.shell("agent-manager run list --json", 0, "[]")
	}
	if f := findingOf(t, s.report(), KindPolling, "agent-manager"); len(f.Blocking) != 0 || f.Steered != 0 {
		t.Errorf("polling a listing = %+v, want no blocking sibling", f)
	}
}

// A wait declared to block only after another command acted on its target
// (`vrooli scenario wait` after a start or restart) is offered only for the
// polls that followed one on the same target: inside the poll stretch or
// within ten minutes before it. Polls with no start, after a start of another
// target or long before them have no blocking sibling, and a start that names
// no target (`make restart` in the scenario's directory) is not claimed.
func TestAConditionalWaitClaimsOnlyThePollsItsPreconditionCovers(t *testing.T) {
	s := newSession()
	poll := func(target string, n int) {
		for i := 0; i < n; i++ {
			s.shell("vrooli scenario status "+target+" --json", 0, "starting")
		}
	}
	s.shell("vrooli scenario restart agent-manager", 0, "restarting")
	poll("agent-manager", 4) // steered
	s.wait(time.Hour)
	poll("test-genie", 5) // no start
	s.wait(time.Hour)
	s.shell("make -C scenarios/web-console restart", 0, "restarting")
	poll("web-console", 3) // steered: make names its target with -C
	s.wait(time.Hour)
	s.shell("make restart", 0, "restarting")
	poll("notification-hub", 3) // unnamed
	s.wait(time.Hour)
	s.shell("vrooli scenario start search-hub", 0, "starting")
	poll("prompt-manager", 3) // another target
	s.wait(time.Hour)
	s.shell("vrooli scenario start device-control", 0, "starting")
	s.wait(20 * time.Minute)
	poll("device-control", 3) // the start is long before the stretch
	r := s.report()
	p := profileOf(t, r, "vrooli scenario status")
	if p.Polled != 21 || p.Steered != 7 || p.Unnamed != 3 || strings.Join(p.Applied, ",") != "vrooli scenario wait" {
		t.Fatalf("profile polled %d steered %d unnamed %d applied %v, want 21, 7, 3, [vrooli scenario wait]", p.Polled, p.Steered, p.Unnamed, p.Applied)
	}
	f := findingOf(t, r, KindPolling, "vrooli")
	if f.Steered != 7 || strings.Contains(f.Description, "although") || !strings.Contains(f.Description, "`vrooli scenario wait` blocks only after `vrooli scenario start` or `vrooli scenario restart` or `make start` or `make restart` of the same target, which 7 of them followed (3 more followed one that names no target, which is not claimed); 11 polled calls have no blocking sibling") {
		t.Errorf("conditional wait = %+v", f)
	}
	// No poll followed a start: the wait is named nowhere.
	s = newSession()
	poll("test-genie", 9)
	if f := findingOf(t, s.report(), KindPolling, "vrooli"); len(f.Blocking) != 0 || f.Steered != 0 || !strings.Contains(f.Description, "which none of them followed; 9 polled calls have no blocking sibling") {
		t.Errorf("no start = %+v", f)
	}
}

// Shape guessing is priced by its costly part, the calls that failed for
// their shape and must be made again; help lookups around them are stated,
// not priced, and a command looked up often but never called wrongly is no
// finding. Its breadth is the runs that guessed, not every run that called
// the command, and a guess is named only with minGuessCalls of evidence.
func TestShapeGuessingIsPricedByWrongShapeCalls(t *testing.T) {
	s := newSession()
	for i := 0; i < 5; i++ {
		s.shell("vrooli-memory journal note --help", 0, "usage: journal note")
		s.shell("vrooli-memory journal note --text x", 0, "ok")
	}
	for _, run := range []string{"run-2", "run-3", "run-4", "run-5"} {
		s.run = run
		s.shell("vrooli-memory journal note --text x", 0, "ok")
	}
	if r := s.report(); slices.ContainsFunc(r.Findings, func(f Finding) bool { return f.Kind == KindShapeGuess }) {
		t.Fatalf("help lookups alone are not priced as guessing: %+v", r.Findings)
	}
	s.run = "run-1"
	s.shell("vrooli-memory journal note --title x", 1, "Error: unknown flag: --title")
	s.shell("vrooli-memory journal note --kind x", 1, "Error: unknown flag: --kind")
	for i := 0; i < 3; i++ {
		s.shell("vrooli-memory journal note --tags x", 1, "Error: unknown flag: --tags")
	}
	r := s.report()
	f := findingOf(t, r, KindShapeGuess, "vrooli-memory")
	if f.WrongShape != 5 || f.Cost != 5 || f.HarmRuns != 1 || f.Runs != 5 || f.Score != Score(1, 1, 5) || r.Totals.VrooliWrongShape != 5 {
		t.Errorf("priced %d of %d wrong-shape calls in %d of %d runs, score %v; want the 5 wrong-shape calls in the 1 run that guessed", f.Cost, f.WrongShape, f.HarmRuns, f.Runs, f.Score)
	}
	if f.Shares[0].Base != "the window's 5 Vrooli CLI wrong-shape calls" {
		t.Errorf("base = %q", f.Shares[0].Base)
	}
	if len(f.Guesses) != 1 || f.Guesses[0].Token != "--tags" || strings.Contains(f.Description, "--title") {
		t.Errorf("guesses = %+v (%s), want only the guess with three calls named", f.Guesses, f.Description)
	}
}

// Result tokens carry how they are known. A nested Codex command whose result
// text was not kept but whose printed size was is estimated from the bytes; a
// skill load that stored no text and no size is unknown, never 0 tokens; and
// the coverage line counts unknowns.
func TestResultTokensCarryTheirBasis(t *testing.T) {
	fact := func(tokens, bytes int64, basis tokenaccounting.Basis, skill string) Fact {
		f := Fact{RunID: "r"}
		f.CommandPath, f.Capability, f.ResultTokens, f.OutputBytes, f.TokenBasis, f.SkillID = "prompt-manager skill read", "shell", tokens, bytes, basis, skill
		return f
	}
	for name, tc := range map[string]struct {
		fact   Fact
		tokens int64
		basis  tokenaccounting.Basis
	}{
		"counted":                      {fact(900, 0, tokenaccounting.BasisEstimated, "s"), 900, tokenaccounting.BasisEstimated},
		"printed but not stored":       {fact(0, 12_683, tokenaccounting.BasisEstimated, "s"), 3_171, tokenaccounting.BasisEstimated},
		"never paired":                 {fact(0, 0, tokenaccounting.BasisUnknown, ""), 0, tokenaccounting.BasisUnknown},
		"skill load without text":      {fact(0, 0, tokenaccounting.BasisEstimated, "s"), 0, tokenaccounting.BasisUnknown},
		"command that printed nothing": {fact(0, 0, tokenaccounting.BasisEstimated, ""), 0, tokenaccounting.BasisEstimated},
	} {
		c := Observe(tc.fact, Features{})
		if c.ResultTokens != tc.tokens || c.TokenBasis != tc.basis {
			t.Errorf("%s: %d tokens (%s), want %d (%s)", name, c.ResultTokens, c.TokenBasis, tc.tokens, tc.basis)
		}
	}
	unsized, sized := fact(0, 0, tokenaccounting.BasisEstimated, "s"), fact(10, 0, tokenaccounting.BasisEstimated, "s")
	unsized.CallEventID, sized.CallEventID = "c1", "c2"
	report := Analyze(Window{Calls: []Call{Observe(unsized, Features{}), Observe(sized, Features{})}})
	if report.Coverage.UnknownTokens != 1 || !strings.Contains(report.Coverage.Sentence(), "result tokens unknown for 1 commands") {
		t.Errorf("coverage = %+v %q", report.Coverage, report.Coverage.Sentence())
	}
}

// A compound command's output sits on its holder segment; its other segments
// are not commands of unknown size. Only a result no segment sizes counts.
func TestHeldOutputIsNotUnknownTokens(t *testing.T) {
	segment := func(call string, index int, tokens int64, basis tokenaccounting.Basis) Call {
		f := Fact{RunID: "r"}
		f.CallEventID, f.ResultEventID, f.SegmentIndex = call, call+"-out", index
		f.CommandPath, f.Capability, f.ResultTokens, f.TokenBasis = "vrooli scenario status", "shell", tokens, basis
		return Observe(f, Features{})
	}
	report := Analyze(Window{Calls: []Call{
		segment("held", 0, 0, tokenaccounting.BasisUnknown),
		segment("held", 1, 0, tokenaccounting.BasisUnknown),
		segment("held", 2, 400, tokenaccounting.BasisEstimated),
		segment("lost", 0, 0, tokenaccounting.BasisUnknown),
		segment("lost", 1, 0, tokenaccounting.BasisUnknown),
	}})
	if report.Coverage.UnknownTokens != 2 {
		t.Errorf("unknown tokens = %d, want 2 (only the result no segment sizes)", report.Coverage.UnknownTokens)
	}
}

// Silence over stale facts says so: a window with nothing at the current fact
// version but runs awaiting re-derivation is "cannot tell", with how many.
func TestCoverageStatesWhatAwaitsReDerivation(t *testing.T) {
	got := Analyze(Window{Stale: Stale{Runs: 281, Calls: 165_120}}).Coverage.Sentence()
	if !strings.Contains(got, "cannot tell") || !strings.Contains(got, "281 runs (165120 calls) awaiting re-derivation; findings exclude them") {
		t.Errorf("coverage = %q", got)
	}
}

// A run is scorable only when enough of its outcomes are known.
func TestRunScoreIsScorableOnlyWithKnownOutcomes(t *testing.T) {
	for _, tc := range []struct {
		score RunScore
		want  bool
	}{
		{RunScore{}, false},
		{RunScore{Commands: 10, OutcomeKnown: 3}, false},
		{RunScore{Commands: 10, OutcomeKnown: 8}, true},
	} {
		if got := tc.score.Scorable(); got != tc.want {
			t.Errorf("%+v scorable = %v, want %v", tc.score, got, tc.want)
		}
	}
}

// Every segment of a compound command carries its own words, even when one
// nested result reports on the whole command: `status treasury && status
// money-ledger` is two targets, and two skill reads are two skills.
func TestEachSegmentOfACompoundCommandKeepsItsOwnWords(t *testing.T) {
	s := newSession().codex(
		"vrooli scenario status agent-manager",
		"search-hub query \"treasury receipt\"",
		"test-genie runs status --json treasury r1 && vrooli scenario status treasury --json | jq '{status: .scenario.status}' && vrooli scenario status money-ledger --json | jq '{status: .scenario.status}'",
		"prompt-manager skill read music-generation; prompt-manager skill read local-experimentation",
	)
	var got []string
	keys := map[string]bool{}
	for _, c := range s.calls() {
		if c.CommandPath == "vrooli scenario status" {
			keys[c.Key] = true
		}
		if c.Command != "" {
			got = append(got, c.Command+" ["+c.SkillID+"]")
		}
	}
	want := []string{
		"vrooli scenario status agent-manager []",
		"search-hub query treasury receipt []",
		"test-genie runs status --json treasury r1 []",
		"vrooli scenario status treasury --json []",
		"vrooli scenario status money-ledger --json []",
		"prompt-manager skill read music-generation [music-generation]",
		"prompt-manager skill read local-experimentation [local-experimentation]",
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("commands:\n%s\nwant:\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
	if len(keys) != 3 {
		t.Fatalf("three status targets are three poll keys: %v", keys)
	}
}

// A chain's time is held by one of its segments so time sums once, but it is
// not that segment's latency: `make build && device-control session list`
// never makes the list slow. A pipe reader shares nothing: `list | jq` is the
// list's own time.
func TestAChainsHeldTimeIsNotItsHoldersLatency(t *testing.T) {
	s := newSession()
	for i := 0; i < 6; i++ {
		s.shell("make build && device-control session list --json", 0, "[]")
		s.shell("agent-manager run list --json | jq .runs", 0, "{}")
	}
	calls := s.calls()
	for i := range calls {
		if calls[i].HasDuration {
			calls[i].DurationMS = 60_000
		}
	}
	r := Analyze(Window{Calls: calls})
	if p := profileOf(t, r, "device-control session list"); p.Timed != 0 || p.TotalMS != 6*60_000 {
		t.Errorf("the chain's time is summed once but is no latency: %+v", p)
	}
	for _, f := range r.Findings {
		if f.Kind == KindSlow && f.Owner != "agent-manager" {
			t.Errorf("a held chain time is blamed on its holder: %+v", f)
		}
	}
	if f := findingOf(t, r, KindSlow, "agent-manager"); f.Cost != 6*55_000 {
		t.Errorf("a piped query keeps its own time: %+v", f)
	}
}
