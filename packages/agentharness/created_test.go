package agentharness

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// stubResolver vouches for the paths it lists with a fixed certainty.
type stubResolver struct {
	name      string
	certainty Certainty
	paths     map[string]bool
}

func (s stubResolver) Name() string { return s.name }

// The birth-time stub corroborates only, like the real resolver.
func (s stubResolver) CorroboratesOnly() bool { return s.name == "birth-time" }
func (s stubResolver) Vouch(q CreationQuery) (Vouch, bool) {
	if s.paths[filepath.Clean(q.Path)] {
		return Vouch{Certainty: s.certainty, Source: s.name, Detail: "stub"}, true
	}
	return Vouch{}, false
}

func stub(name string, c Certainty, paths ...string) stubResolver {
	m := map[string]bool{}
	for _, p := range paths {
		m[filepath.Clean(p)] = true
	}
	return stubResolver{name: name, certainty: c, paths: m}
}

func evidence(tracked bool, resolvers ...CreationResolver) func() CreationEvidence {
	return func() CreationEvidence {
		return CreationEvidence{Resolvers: resolvers, Tracked: func(string, bool) (bool, bool) { return tracked, true }}
	}
}

func TestCreationEvidenceCombinesResolversAndRaisesAgreement(t *testing.T) {
	p := "/x/a"
	cases := []struct {
		name string
		res  []CreationResolver
		want Certainty
		ok   bool
	}{
		{"nobody vouches", []CreationResolver{stub("a", CertaintyMedium, "/other")}, 0, false},
		{"one low stays low", []CreationResolver{stub("transcript", CertaintyLow, p)}, CertaintyLow, true},
		{"strongest wins", []CreationResolver{stub("transcript", CertaintyLow, p), stub("sandbox", CertaintyRecorded, p)}, CertaintyRecorded, true},
		{"birth time alone stays low", []CreationResolver{stub("birth-time", CertaintyLow, p)}, CertaintyLow, true},
		{"birth time lifts an attributing low to medium", []CreationResolver{stub("transcript", CertaintyLow, p), stub("birth-time", CertaintyLow, p)}, CertaintyMedium, true},
		{"birth time does not raise an attributing medium", []CreationResolver{stub("transcript", CertaintyMedium, p), stub("birth-time", CertaintyLow, p)}, CertaintyMedium, true},
		{"birth time does not raise a recorded change", []CreationResolver{stub("workspace-sandbox", CertaintyRecorded, p), stub("birth-time", CertaintyLow, p)}, CertaintyRecorded, true},
		{"two attributing resolvers: low and medium reach recorded", []CreationResolver{stub("transcript", CertaintyLow, p), stub("workspace-sandbox", CertaintyMedium, p)}, CertaintyRecorded, true},
		{"two lows agree to medium", []CreationResolver{stub("a", CertaintyLow, p), stub("b", CertaintyLow, p)}, CertaintyMedium, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			v, ok := CreationEvidence{Resolvers: c.res}.Vouch(CreationQuery{Path: p})
			if ok != c.ok || v.Certainty != c.want {
				t.Fatalf("got %v %v, want %v %v", v.Certainty, ok, c.want, c.ok)
			}
		})
	}
	tracked := CreationEvidence{Resolvers: []CreationResolver{stub("sandbox", CertaintyRecorded, p)}, Tracked: func(string, bool) (bool, bool) { return true, true }}
	if _, ok := tracked.Vouch(CreationQuery{Path: p}); ok {
		t.Error("a path git tracks was vouched as scratch")
	}
	unknown := CreationEvidence{Resolvers: []CreationResolver{stub("sandbox", CertaintyRecorded, p)}, Tracked: func(string, bool) (bool, bool) { return false, false }}
	if _, ok := unknown.Vouch(CreationQuery{Path: p}); ok {
		t.Error("a path git could not be asked about was vouched")
	}
}

// The guard loosens only what the zone's threshold allows, and everything else
// is decided exactly as before.
func TestRemovalLoosensOnlyForSufficientCreationEvidence(t *testing.T) {
	f := newRemovalFixture(t)
	made := filepath.Join(f.repo, "scenarios", "app", "generated")
	protected := filepath.Join(f.repo, "scenarios", "app", "data")
	for _, dir := range []string{made, filepath.Join(made, "deep"), protected} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	homeFile := filepath.Join(f.home, "stray.txt")
	if err := os.WriteFile(homeFile, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	askRule := PathRule{Root: protected, Action: ActionAsk, Reason: "tier safe_with_owner", Source: "repo-contract storage.roots/data"}
	denyRule := PathRule{Root: filepath.Join(made, "deep"), Action: ActionDeny, Reason: "durable", Source: "owner"}

	decide := func(command string, e func() CreationEvidence, rules ...PathRule) Decision {
		ctx := f.context(f.repo, rules...)
		ctx.Created = e
		return ctx.EvaluateRemoval(parseRemovals(command, dialectPOSIX, removalEnv{WorkingDirectory: f.repo, Home: ctx.Home, Lookup: ctx.Lookup}))
	}
	cases := []struct {
		name    string
		command string
		ev      func() CreationEvidence
		rules   []PathRule
		want    DecisionAction
		zone    string
	}{
		{"no evidence: tree still asks", "rm -r scenarios/app/generated", nil, nil, ActionAsk, "repository-tree"},
		{"low alone does not loosen", "rm -r scenarios/app/generated", evidence(false, stub("transcript", CertaintyLow, made)), nil, ActionAsk, "repository-tree"},
		{"birth time alone does not loosen", "rm -r scenarios/app/generated", evidence(false, stub("birth-time", CertaintyLow, made)), nil, ActionAsk, "repository-tree"},
		{"birth time plus a transcript command loosens", "rm -r scenarios/app/generated", evidence(false, stub("transcript", CertaintyLow, made), stub("birth-time", CertaintyLow, made)), nil, ActionAllow, "run-created"},
		{"medium loosens a repository tree", "rm -r scenarios/app/generated", evidence(false, stub("transcript", CertaintyMedium, made)), nil, ActionAllow, "run-created"},
		{"two lows agree and loosen", "rm -r scenarios/app/generated", evidence(false, stub("a", CertaintyLow, made), stub("b", CertaintyLow, made)), nil, ActionAllow, "run-created"},
		{"medium loosens a home path", "rm " + homeFile, evidence(false, stub("transcript", CertaintyMedium, homeFile)), nil, ActionAllow, "run-created"},
		{"git-tracked tree is not scratch", "rm -r scenarios/app/generated", evidence(true, stub("sandbox", CertaintyRecorded, made)), nil, ActionAsk, "repository-tree"},
		{"declared ask needs a recorded change", "rm -r scenarios/app/data", evidence(false, stub("transcript", CertaintyMedium, protected), stub("birth-time", CertaintyLow, protected)), []PathRule{askRule}, ActionAsk, "declared"},
		{"recorded opens a declared ask", "rm -r scenarios/app/data", evidence(false, stub("sandbox", CertaintyRecorded, protected)), []PathRule{askRule}, ActionAllow, "run-created"},
		{"declared deny is never loosened", "rm -r scenarios/app/generated/deep", evidence(false, stub("sandbox", CertaintyRecorded, filepath.Join(made, "deep"))), []PathRule{denyRule}, ActionDeny, "declared"},
		{"a created tree holding a protected root asks", "rm -r scenarios/app/generated", evidence(false, stub("sandbox", CertaintyRecorded, made)), []PathRule{denyRule}, ActionDeny, "declared-nested"},
		{"a pattern is never loosened", "rm -r scenarios/app/generated/*", evidence(false, stub("sandbox", CertaintyRecorded, made)), nil, ActionAsk, "repository-tree"},
		{"the repository root stays denied", "rm -rf " + f.repo, evidence(false, stub("sandbox", CertaintyRecorded, f.repo)), nil, ActionDeny, "repository-root"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			d := decide(c.command, c.ev, c.rules...)
			zone := ""
			for _, e := range d.Evidence {
				zone = e.Facts["zone"]
			}
			if d.Action != c.want || zone != c.zone {
				t.Fatalf("action %s zone %s, want %s %s: %s", d.Action, zone, c.want, c.zone, d.Reason)
			}
			if c.zone == "run-created" && !strings.Contains(d.Reason, "created by this session") {
				t.Errorf("the allowance does not say why: %s", d.Reason)
			}
		})
	}
}

func TestBirthTimeResolverVouchesOnlyAfterTheSessionStart(t *testing.T) {
	if _, ok := birthTime(t.TempDir()); !ok {
		t.Skip("this file system does not record birth time")
	}
	dir := t.TempDir()
	file := filepath.Join(dir, "scratch.txt")
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	if v, ok := NewBirthTimeResolver(now.Add(-time.Hour), now).Vouch(CreationQuery{Path: file}); !ok || v.Certainty != CertaintyLow {
		t.Fatalf("a file made after the session started was not vouched: %+v %v", v, ok)
	}
	if _, ok := NewBirthTimeResolver(now.Add(time.Hour), now).Vouch(CreationQuery{Path: file}); ok {
		t.Fatal("a file born before the session start was vouched")
	}
	if _, ok := NewBirthTimeResolver(time.Time{}, now).Vouch(CreationQuery{Path: file}); ok {
		t.Fatal("an unknown start vouched")
	}
	// A session that began days ago only counts the last 24 hours.
	if r := NewBirthTimeResolver(now.Add(-72*time.Hour), now); r.start.Before(now.Add(-25 * time.Hour)) {
		t.Fatalf("an old session start was not capped: %s", r.start)
	}
	if _, ok := NewBirthTimeResolver(now.Add(-time.Hour), now).Vouch(CreationQuery{Path: filepath.Join(dir, "missing")}); ok {
		t.Fatal("a missing path was vouched")
	}
}

// Real shapes: Claude Code writes a toolUseResult of type "create" and Bash
// tool_use blocks; Codex writes apply_patch and exec_command tool calls.
const claudeTranscript = `{"type":"user","timestamp":"2026-11-15T17:00:00.000Z","cwd":"/work","message":{"role":"user","content":"hi"}}
{"type":"assistant","timestamp":"2026-11-15T17:00:05.000Z","cwd":"/work","message":{"role":"assistant","content":[{"type":"tool_use","name":"Bash","input":{"command":"mkdir -p /work/out/nested && touch notes.txt"}}]}}
{"type":"user","timestamp":"2026-11-15T17:00:09.000Z","toolUseResult":{"type":"create","filePath":"/work/made.html"},"message":{"role":"user","content":[]}}
{"type":"user","timestamp":"2026-11-15T17:00:10.000Z","toolUseResult":{"type":"update","filePath":"/work/edited.go"},"message":{"role":"user","content":[]}}
not json at all
`

const codexTranscript = `{"timestamp":"2026-11-16T02:42:13.917Z","type":"session_meta","payload":{"type":"","timestamp":"2026-11-16T02:42:13.840Z","cwd":"/work"}}
{"timestamp":"2026-11-16T02:42:17.968Z","type":"response_item","payload":{"type":"custom_tool_call","name":"apply_patch","cwd":"/work","input":"*** Begin Patch\n*** Add File: pkg/new.go\n+package pkg\n*** Update File: pkg/old.go\n*** End Patch"}}
{"timestamp":"2026-11-16T02:42:18.968Z","type":"response_item","payload":{"type":"function_call","name":"exec_command","cwd":"/work","arguments":"{\"cmd\":\"mkdir -p tmp/probe\"}"}}
`

func TestTranscriptReaderFindsCreationsAndRatesThemByHowItKnows(t *testing.T) {
	claude, first := ReadTranscriptCreations(strings.NewReader(claudeTranscript))
	if first.Format(time.RFC3339) != "2026-11-15T17:00:00Z" {
		t.Errorf("first entry time = %s", first)
	}
	want := map[string]Certainty{"/work/made.html": CertaintyMedium, "/work/out/nested": CertaintyLow, "/work/notes.txt": CertaintyLow}
	for path, certainty := range want {
		if got, ok := claude[path]; !ok || got.Certainty != certainty {
			t.Errorf("claude: %s = %+v (listed %v), want %v", path, got, ok, certainty)
		}
	}
	if !claude["/work/out/nested"].Dir || claude["/work/notes.txt"].Dir {
		t.Errorf("mkdir must mark a folder and touch a file: %+v %+v", claude["/work/out/nested"], claude["/work/notes.txt"])
	}
	if _, ok := claude["/work/edited.go"]; ok {
		t.Error("an updated file was reported as created")
	}

	codex, first := ReadTranscriptCreations(strings.NewReader(codexTranscript))
	if first.IsZero() {
		t.Error("codex start time not read")
	}
	if got := codex["/work/pkg/new.go"]; got.Certainty != CertaintyMedium {
		t.Errorf("an apply_patch Add File should be medium: %+v", got)
	}
	if _, ok := codex["/work/pkg/old.go"]; ok {
		t.Error("an updated file was reported as created")
	}
	if got := codex["/work/tmp/probe"]; got.Certainty != CertaintyLow || !got.Dir {
		t.Errorf("an exec mkdir should be a low folder: %+v", got)
	}
}

func TestTranscriptResolverNeedsEveryFileInATreeListed(t *testing.T) {
	dir := t.TempDir()
	tree := filepath.Join(dir, "out")
	if err := os.MkdirAll(filepath.Join(tree, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	a, b := filepath.Join(tree, "a.txt"), filepath.Join(tree, "sub", "b.txt")
	for _, f := range []string{a, b} {
		if err := os.WriteFile(f, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	created := map[string]CreatedPath{
		tree: {Path: tree, Certainty: CertaintyLow, Dir: true},
		a:    {Path: a, Certainty: CertaintyMedium},
	}
	r := NewTranscriptResolver(created)
	if _, ok := r.Vouch(CreationQuery{Path: tree, Recursive: true}); ok {
		t.Fatal("a tree with an unlisted file was vouched")
	}
	created[b] = CreatedPath{Path: b, Certainty: CertaintyMedium}
	v, ok := r.Vouch(CreationQuery{Path: tree, Recursive: true})
	if !ok || v.Certainty != CertaintyLow {
		t.Fatalf("a fully listed tree should be vouched at the weakest listing: %+v %v", v, ok)
	}
	if v, ok := r.Vouch(CreationQuery{Path: a}); !ok || v.Certainty != CertaintyMedium {
		t.Fatalf("a listed file: %+v %v", v, ok)
	}
	other := filepath.Join(dir, "preexisting")
	if err := os.MkdirAll(other, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, ok := r.Vouch(CreationQuery{Path: other, Recursive: true}); ok {
		t.Fatal("a folder the session did not make was vouched")
	}
}

func TestLocateTranscriptsSearchesTheSessionsOwnHomesFirst(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	session := "abc123"
	write := func(path string) string {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("{}\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		return path
	}
	claude := write(filepath.Join(home, ".claude", "projects", "-work", session+".jsonl"))
	userCodex := write(filepath.Join(home, ".codex", "sessions", "2026", "11", "15", "rollout-2026-11-15T10-00-00-"+session+".jsonl"))
	sessionHome := t.TempDir()
	ownCodex := write(filepath.Join(sessionHome, "sessions", "2026", "11", "15", "rollout-2026-11-15T10-00-00-"+session+".jsonl"))
	named := write(filepath.Join(t.TempDir(), "named.jsonl"))
	write(filepath.Join(home, ".codex", "sessions", "2026", "11", "15", "rollout-other-zzz.jsonl"))
	write(filepath.Join(home, ".codex", "sessions", "2026", "11", "15", "rollout-2026-11-15T10-00-00-xabc123.jsonl")) // a longer id that merely ends the same

	got := LocateTranscripts(RunIdentity{HarnessSession: session, TranscriptPath: named, CodexHome: sessionHome})
	want := []string{named, claude, ownCodex, userCodex}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("transcripts = %v, want %v", got, want)
	}
	if got := LocateTranscripts(RunIdentity{HarnessSession: "../" + session}); len(got) != 0 {
		t.Fatalf("a session id with path characters found %v", got)
	}
}

func TestSandboxResolverVouchesForRecordedAdditionsOnly(t *testing.T) {
	merged := t.TempDir()
	tree := filepath.Join(merged, "new-feature")
	if err := os.MkdirAll(tree, 0o755); err != nil {
		t.Fatal(err)
	}
	added, edited := filepath.Join(tree, "a.go"), filepath.Join(merged, "existing.go")
	for _, f := range []string{added, edited} {
		if err := os.WriteFile(f, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/sandboxes/sb-1/diff" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"files": []map[string]string{
			{"filePath": "new-feature/a.go", "changeType": "added"},
			{"filePath": "existing.go", "changeType": "modified"},
		}})
	}))
	defer server.Close()
	changes, err := FetchSandboxChanges(context.Background(), server.URL, "sb-1")
	if err != nil || len(changes) != 2 {
		t.Fatalf("changes = %v, %v", changes, err)
	}
	r := NewSandboxResolver(merged, changes)
	if v, ok := r.Vouch(CreationQuery{Path: added}); !ok || v.Certainty != CertaintyRecorded {
		t.Fatalf("an added file: %+v %v", v, ok)
	}
	if _, ok := r.Vouch(CreationQuery{Path: edited}); ok {
		t.Fatal("a modified file was vouched as created")
	}
	if v, ok := r.Vouch(CreationQuery{Path: tree, Recursive: true}); !ok || v.Certainty != CertaintyRecorded {
		t.Fatalf("a tree of only additions: %+v %v", v, ok)
	}
	if err := os.WriteFile(filepath.Join(tree, "preexisting.go"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, ok := r.Vouch(CreationQuery{Path: tree, Recursive: true}); ok {
		t.Fatal("a tree holding an unrecorded file was vouched")
	}
	if _, ok := r.Vouch(CreationQuery{Path: t.TempDir()}); ok {
		t.Fatal("a path outside the sandbox was vouched")
	}
	if _, err := FetchSandboxChanges(context.Background(), "", "sb-1"); err == nil {
		t.Fatal("an unknown service address must be an error, not an empty record")
	}
	down := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Error(w, "no", 500) }))
	defer down.Close()
	if _, err := FetchSandboxChanges(context.Background(), down.URL, "sb-1"); err == nil {
		t.Fatal("a failing service must be an error")
	}
}

// End to end through the runtime: the session's own scratch is allowed, the
// switch turns it off, each allowance is audited, and the same command for a
// path the session did not create is decided as before.
func TestRuntimeLoosensForTheSessionsOwnFilesAndAuditsIt(t *testing.T) {
	f := newRemovalFixture(t)
	made := filepath.Join(f.repo, "scenarios", "app", "scratch-out")
	other := filepath.Join(f.repo, "scenarios", "app", "someone-elses")
	for _, dir := range []string{made, other} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	state := t.TempDir()
	rt := Runtime{
		Store:   NewBundleStore(state),
		Removal: func(string) RemovalContext { return f.context(f.repo) },
		Creation: func(ToolEvent, time.Time) CreationEvidence {
			return evidence(false, stub("sandbox", CertaintyRecorded, made))()
		},
	}
	event := func(path string) ToolEvent {
		return ToolEvent{Runner: "codex", Tool: "shell", Shell: "rm -r " + path, WorkingDirectory: f.repo, Context: map[string]string{ContextSessionID: "s-1"}}
	}
	d, err := rt.Evaluate(event(made))
	if err != nil || d.Action != ActionAllow {
		t.Fatalf("own scratch: %v %v: %s", d.Action, err, d.Reason)
	}
	if audit := ReadCreationAudit(state, time.Time{}); len(audit) != 1 || audit[0].Path != made || audit[0].Source != "sandbox" || audit[0].Certainty != "recorded" || audit[0].Session != "s-1" {
		t.Fatalf("audit = %+v", audit)
	}
	if d, _ := rt.Evaluate(event(other)); d.Action != ActionAsk {
		t.Fatalf("a path the session did not create must be decided as before, got %s", d.Action)
	}
	if err := SaveCreationSettings(state, CreationSettings{Enabled: false, By: "operator"}); err != nil {
		t.Fatal(err)
	}
	if d, _ := rt.Evaluate(event(made)); d.Action != ActionAsk {
		t.Fatalf("with the switch off the guard must ask, got %s", d.Action)
	}
	if err := SaveCreationSettings(state, CreationSettings{Enabled: true}); err != nil {
		t.Fatal(err)
	}
	if d, _ := rt.Evaluate(event(made)); d.Action != ActionAllow {
		t.Fatalf("switch back on: %s", d.Action)
	}
}

func TestRunIdentityReadsTheEventAndEnvironment(t *testing.T) {
	env := map[string]string{"VROOLI_AGENT_SESSION": "vrooli-agent-claude-code-1", "CLAUDE_CODE_SESSION_ID": "from-env", "CODEX_HOME": "/s/home",
		"VROOLI_SANDBOX_ID": "sb", "VROOLI_SANDBOX_MERGED": "/m", "VROOLI_AGENT_STARTED_AT": "2026-11-15T10:00:00Z"}
	lookup := func(k string) (string, bool) { v, ok := env[k]; return v, ok }
	id := RunIdentityFromEvent(ToolEvent{Runner: "claude-code", WorkingDirectory: "/w", Context: map[string]string{ContextSessionID: "from-hook", ContextTranscriptPath: "/t.jsonl"}}, lookup)
	if id.Session != "vrooli-agent-claude-code-1" || id.HarnessSession != "from-hook" || id.TranscriptPath != "/t.jsonl" || id.CodexHome != "/s/home" ||
		id.SandboxID != "sb" || id.SandboxMerged != "/m" || id.Start.Format(time.RFC3339) != "2026-11-15T10:00:00Z" {
		t.Fatalf("identity = %+v", id)
	}
	if id := RunIdentityFromEvent(ToolEvent{}, func(string) (string, bool) { return "", false }); id.HarnessSession != "" || !id.Start.IsZero() {
		t.Fatalf("an empty environment must give an empty identity: %+v", id)
	}
}

func TestGitTrackedFailsClosedInsideAWorkTree(t *testing.T) {
	outside := t.TempDir()
	file := filepath.Join(outside, "f.txt")
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if tracked, known := GitTracked(file, false); tracked || !known {
		t.Fatalf("outside any work tree: tracked=%v known=%v", tracked, known)
	}
	repo := t.TempDir()
	run := func(args ...string) {
		out, err := gitOutput(repo, args...)
		if err != nil {
			t.Skipf("git unavailable: %v %s", err, out)
		}
	}
	run("init", "-q")
	run("-c", "user.name=t", "-c", "user.email=t@example.com", "commit", "-q", "--allow-empty", "-m", "x")
	tracked := filepath.Join(repo, "kept.txt")
	untracked := filepath.Join(repo, "scratch.txt")
	for _, p := range []string{tracked, untracked} {
		if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	run("add", "kept.txt")
	if got, known := GitTracked(tracked, false); !got || !known {
		t.Errorf("tracked file: %v %v", got, known)
	}
	if got, known := GitTracked(untracked, false); got || !known {
		t.Errorf("untracked file: %v %v", got, known)
	}
	if got, known := GitTracked(repo, true); !got || !known {
		t.Errorf("a tree holding a tracked file: %v %v", got, known)
	}
}

// The real chain, with nothing stubbed: the hook names a transcript, the
// transcript says the session made a folder and the files in it, the file
// system agrees the folder was born after the session began, and git has
// nothing to say about it. Two sources agree, so the guard allows the removal;
// birth time on its own does not.
func TestHostCreationEvidenceEndToEnd(t *testing.T) {
	if _, ok := birthTime(t.TempDir()); !ok {
		t.Skip("this file system does not record birth time")
	}
	f := newRemovalFixture(t)
	tree := filepath.Join(f.repo, "scenarios", "app", "outdir")
	if err := os.MkdirAll(tree, 0o755); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(tree, "result.json")
	if err := os.WriteFile(file, []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	started := time.Now().UTC().Add(-30 * time.Minute).Format(time.RFC3339Nano)
	transcript := filepath.Join(t.TempDir(), "session.jsonl")
	lines := []string{
		`{"type":"user","timestamp":"` + started + `","cwd":"` + f.repo + `","message":{"role":"user","content":"go"}}`,
		`{"type":"assistant","timestamp":"` + started + `","cwd":"` + f.repo + `","message":{"role":"assistant","content":[{"type":"tool_use","name":"Bash","input":{"command":"mkdir -p ` + tree + `"}}]}}`,
		`{"type":"user","timestamp":"` + started + `","toolUseResult":{"type":"create","filePath":"` + file + `"},"message":{"role":"user","content":[]}}`,
	}
	if err := os.WriteFile(transcript, []byte(strings.Join(lines, "\n")+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	rt := Runtime{Store: NewBundleStore(t.TempDir()), Removal: func(string) RemovalContext { return f.context(f.repo) }}
	event := ToolEvent{Runner: "claude-code", Tool: "Bash", Shell: "rm -rf " + tree, WorkingDirectory: f.repo, Context: map[string]string{ContextTranscriptPath: transcript, ContextSessionID: "s"}}
	d, err := rt.Evaluate(event)
	if err != nil || d.Action != ActionAllow {
		t.Fatalf("the session's own folder was not allowed: %v %v: %s", d.Action, err, d.Reason)
	}
	if !strings.Contains(d.Reason, "birth-time+transcript") {
		t.Errorf("the reason should name both sources: %s", d.Reason)
	}
	// Birth time alone is low certainty: it cannot tell this session's folders
	// from another process's, so a folder the transcript never mentions is
	// decided as before.
	other := filepath.Join(f.repo, "scenarios", "app", "unlisted")
	if err := os.MkdirAll(other, 0o755); err != nil {
		t.Fatal(err)
	}
	event.Shell = "rm -rf " + other
	if d, _ := rt.Evaluate(event); d.Action != ActionAsk {
		t.Fatalf("birth time alone must not loosen: %s: %s", d.Action, d.Reason)
	}
	// With the transcript's start in the future of the folder, nothing vouches.
	late := filepath.Join(t.TempDir(), "late.jsonl")
	future := time.Now().UTC().Add(time.Hour).Format(time.RFC3339Nano)
	if err := os.WriteFile(late, []byte(`{"type":"user","timestamp":"`+future+`","message":{"role":"user","content":"go"}}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	event.Context[ContextTranscriptPath] = late
	if d, _ := rt.Evaluate(event); d.Action != ActionAsk {
		t.Fatalf("a folder born before the session began must be decided as before, got %s: %s", d.Action, d.Reason)
	}
}

// An owner declares generated names, not folders: each existing child whose
// name matches is disposable, even inside a protected parent, and only while it
// holds nothing but what the tool makes.
func TestMatchRulesCoverGeneratedNamesInsideProtectedData(t *testing.T) {
	f := newRemovalFixture(t)
	data := filepath.Join(f.home, ".vrooli", "data")
	leak := filepath.Join(data, "treasury_cf2-123")
	odd := filepath.Join(data, "treasury_cf2-456")
	live := filepath.Join(data, "treasury")
	for _, dir := range []string{leak, odd, live} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	for _, name := range []string{"treasury.db", "treasury.db-wal", "treasury.db-shm"} {
		if err := os.WriteFile(filepath.Join(leak, name), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	for _, name := range []string{"treasury.db", "notes.md"} {
		if err := os.WriteFile(filepath.Join(odd, name), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	rules := []PathRule{
		{Root: data, Action: ActionDeny, Reason: "durable data", Source: "runtime-home data"},
		{Root: data, Match: "*_cf[25]-*", Only: []string{"*.db", "*.db-wal", "*.db-shm"}, Action: ActionAllow, Reason: "test-run folders of the rehearsal tests", Source: "disposable/treasury-rehearsal"},
	}
	cases := map[string]struct {
		path string
		want DecisionAction
	}{
		"matching folder holding only databases": {leak, ActionAllow},
		"matching folder holding something else": {odd, ActionAsk},
		"the live data folder stays protected":   {live, ActionDeny},
		"the parent is not covered":              {data, ActionDeny},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			d := f.decide(f.repo, "rm -rf "+c.path, rules...)
			if d.Action != c.want {
				t.Fatalf("%s: %s, want %s: %s", c.path, d.Action, c.want, d.Reason)
			}
		})
	}
}

func TestPathRulePatternsAreValidated(t *testing.T) {
	bad := []PathRule{
		{Root: "/a/b", Action: ActionAllow, Reason: "r", Match: "x/y"},
		{Root: "/a/b", Action: ActionAllow, Reason: "r", Match: "[unclosed"},
		{Root: "/a/b", Action: ActionDeny, Reason: "r", Only: []string{"*.db"}},
	}
	for _, rule := range bad {
		if validatePathRulePatterns(rule) == nil {
			t.Errorf("accepted %+v", rule)
		}
	}
}

// The runtime home's tmp folder is a temporary root: an agent may clear what
// is inside it, but not the folder itself.
func TestHostRemovalContextTreatsTheRuntimeHomeTmpAsATemporaryRoot(t *testing.T) {
	repo := newRemovalFixture(t).repo
	contract, err := os.ReadFile(filepath.Join(findRepoRoot(t), ".vrooli", "repo-contract.json"))
	if err != nil {
		t.Skip("no repository contract to copy")
	}
	if err := os.WriteFile(filepath.Join(repo, ".vrooli", "repo-contract.json"), contract, 0o644); err != nil {
		t.Fatal(err)
	}
	home := filepath.Dir(repo)
	t.Setenv("HOME", home)
	ctx := HostRemovalContext(repo)
	tmp := ""
	for _, root := range ctx.TempRoots {
		if strings.HasSuffix(root, filepath.Join(".vrooli", "tmp")) {
			tmp = root
		}
	}
	if tmp == "" || !strings.HasPrefix(tmp, home) {
		t.Fatalf("the runtime home tmp is not a temporary root: %v", ctx.TempRoots)
	}
	scratch := filepath.Join(tmp, "my-scratch")
	if err := os.MkdirAll(scratch, 0o755); err != nil {
		t.Fatal(err)
	}
	decide := func(path string) Decision {
		return ctx.EvaluateRemoval(parseRemovals("rm -rf "+path, dialectPOSIX, removalEnv{WorkingDirectory: repo, Home: ctx.Home, Lookup: ctx.Lookup}))
	}
	if d := decide(scratch); d.Action != ActionAllow {
		t.Fatalf("scratch inside the runtime tmp: %s: %s", d.Action, d.Reason)
	}
	if d := decide(tmp); d.Action != ActionAsk {
		t.Fatalf("the runtime tmp itself must still ask, got %s", d.Action)
	}
}

func findRepoRoot(t *testing.T) string {
	t.Helper()
	dir, _ := os.Getwd()
	for ; dir != "/"; dir = filepath.Dir(dir) {
		if _, err := os.Stat(filepath.Join(dir, ".vrooli", "repo-contract.json")); err == nil {
			return dir
		}
	}
	t.Skip("not inside a Vrooli checkout")
	return ""
}
