package runsignal

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"agent-manager/internal/availability"
)

// ResolveCatalog reads a line with the one shell lexer: wrappers,
// redirections and cd prefixes are not commands, and a line that runs several
// commands, or a command only the shell could name, is unknown.
func TestResolveCatalogResolvesOneSimpleCommand(t *testing.T) {
	for command, want := range map[string]CatalogResolution{
		"vrooli scenario status business-health --json 2>&1":       {Owner: "vrooli", State: availability.Resolved},
		"bash -lc 'vrooli scenario status --json'":                 {Owner: "vrooli", State: availability.Resolved},
		`cd "a|b" && agent-manager run report x`:                   {Owner: "agent-manager", State: availability.Resolved},
		"vrooli scenario status business-health --json | head -40": {State: availability.Unknown},
		`cd "a|b" && vrooli scenario test agent-manager | tee out`: {State: availability.Unknown},
		"$(which vrooli) scenario status":                          {State: availability.Unknown},
		"echo 'unterminated":                                       {State: availability.Unknown},
	} {
		got := ResolveCatalog(command)
		if got.Owner != want.Owner || got.State != want.State {
			t.Fatalf("ResolveCatalog(%q) = %+v, want owner %q state %q", command, got, want.Owner, want.State)
		}
	}
}

func TestResolveCatalogIncludesDeclaredCrossScenarioCommands(t *testing.T) {
	for _, command := range []string{
		"prompt-manager skill read scenario-work-ladder",
		"prompt-manager discover imported transcript classification",
		"plan-manager plans render investigation-plan",
	} {
		resolution := ResolveCatalog(command)
		if resolution.State != availability.Resolved {
			t.Fatalf("ResolveCatalog(%q) = %+v, want resolved", command, resolution)
		}
	}
}

func TestResolveCatalogAcceptsDeclaredGroupsAndLeadingFlags(t *testing.T) {
	for _, command := range []string{
		"plan-manager plans --help",
		"plan-manager --auto-start author start --title example",
		"test-genie runs --help",
	} {
		resolution := ResolveCatalog(command)
		if resolution.State != availability.Resolved {
			t.Fatalf("ResolveCatalog(%q) = %+v, want resolved", command, resolution)
		}
	}
}

func TestResolveCatalogUsesManifestIndex(t *testing.T) {
	catalogCache.Lock()
	catalogCache.current = nil
	catalogCache.Unlock()
	got := ResolveCatalog("agent-manager run report x")
	if got.State != "resolved" || got.Command != "agent-manager run report" || got.Snapshot == "" {
		t.Fatalf("resolution=%+v root=%s", got, projectRoot())
	}
}

func TestResolveCatalogUnwrapsLiteralRunnerShellEnvelope(t *testing.T) {
	got := ResolveCatalog("/bin/bash -lc 'agent-manager run report x'")
	if got.State != "resolved" || got.Command != "agent-manager run report" {
		t.Fatalf("resolution=%+v", got)
	}
	if got := ResolveCatalog("/bin/bash -lc 'agent-manager run report x; rm x'"); got.State != "unknown" {
		t.Fatalf("compound wrapper resolution=%+v", got)
	}
}

func TestResolveCatalogNamesSafelyParsedExternalExecutable(t *testing.T) {
	got := ResolveCatalog("/usr/bin/git status --short")
	if got.State != "external" || got.Owner != "git" || got.Command != "git" {
		t.Fatalf("resolution=%+v", got)
	}
}

func TestResolveCatalogUnwrapsLiteralDoubleQuotedRunnerEnvelope(t *testing.T) {
	got := ResolveCatalog(`/bin/bash -lc "sed -n 1,10p docs/readme.md"`)
	if got.State != "external" || got.Owner != "sed" {
		t.Fatalf("resolution=%+v", got)
	}
}

// A manifest group does not list every operation under it: the next word still
// selects the operation, so `runs wait` and `runs start` profile apart, while a
// declared leaf keeps its own path.
func TestCommandPathKeepsTheOperationUnderADeclaredGroup(t *testing.T) {
	for command, want := range map[string]string{
		"test-genie runs wait --json a 1":   "test-genie runs wait",
		"test-genie runs start x":           "test-genie runs start",
		"test-genie runs list --scenario x": "test-genie runs list",
		"test-genie runs":                   "test-genie runs",
		"vrooli scenario logs x":            "vrooli scenario logs",
	} {
		resolution := ResolveCatalog(command)
		if got := commandPathFor(resolution, strings.Fields(command)); got != want {
			t.Errorf("commandPathFor(%q) = %q, want %q", command, got, want)
		}
	}
}

// A polled read's blocking siblings are the commands the owning CLI's manifest
// declares replace it (semantics.blocks.replaces), in declaration order:
// groups, declared durable or streaming exceptions, the control plane's
// cli/manifest.json, and a flag that makes a command block. A verb is never
// evidence: `run await-result` re-reads a parked result, so `run get` is
// offered `run events --follow`. A listing has none: a wait on one target does
// not stand in for it.
func TestBlockingSiblingsComeFromTheManifests(t *testing.T) {
	for path, want := range map[string]string{
		"vrooli scenario status":         "vrooli scenario wait",
		"workflow-health validate get":   "workflow-health validate wait",
		"test-genie runs show":           "test-genie runs wait",
		"agent-manager run get":          "agent-manager run events --follow",
		"agent-manager run events":       "agent-manager run events --follow",
		"program-runtime programs get":   "program-runtime programs wait",
		"device-control device state":    "device-control device watch",
		"test-genie runs wait":           "",
		"device-control session list":    "",
		"agent-manager run await-result": "",
		"agent-manager run list":         "",
		"device-control device list":     "",
		"vrooli scenario logs":           "",
	} {
		got := BlockingSiblings(path)
		if want == "" && len(got) != 0 || want != "" && (len(got) == 0 || got[0].Command() != want) {
			t.Errorf("%s: siblings %v, want first %q", path, got, want)
		}
	}
	var runs []string
	for _, b := range BlockingSiblings("test-genie runs show") {
		runs = append(runs, b.Command())
	}
	if strings.Join(runs, ",") != "test-genie runs wait,test-genie runs wait-all,test-genie runs follow" {
		t.Errorf("test-genie runs siblings = %v, want wait, wait-all, follow", runs)
	}
	if got := BlockingSiblings("vrooli scenario status"); len(got) != 1 || strings.Join(got[0].After, ",") != "vrooli scenario start,vrooli scenario restart,make start,make restart" {
		t.Errorf("scenario wait precondition = %+v, want after start/restart (vrooli and make)", got)
	}
	if !Blocks("agent-manager run events", []string{"agent-manager", "run", "events", "x", "--follow"}) || Blocks("agent-manager run events", []string{"agent-manager", "run", "events", "x"}) || Blocks("agent-manager run await-result", nil) || !Blocks("test-genie runs wait", nil) {
		t.Error("Blocks must follow the manifest: run events only with --follow, runs wait always, await-result never")
	}
}

// The manifest index is parsed once for resolutions, semantics and blocking
// siblings and descriptions, and rebuilt when a manifest changes: the snapshot a resolution
// carries names the manifests it was resolved against. An unchanged manifest
// keeps the same index.
func TestCatalogRebuildsWhenAManifestChanges(t *testing.T) {
	root := t.TempDir()
	t.Setenv("VROOLI_PROJECT_ROOT", root)
	recheck := catalogRecheck
	catalogRecheck = 0
	t.Cleanup(func() {
		catalogRecheck = recheck
		catalogCache.Lock()
		catalogCache.current = nil
		catalogCache.Unlock()
	})
	path := filepath.Join(root, "scenarios", "widget", "cli", "manifest.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	write := func(body string) {
		t.Helper()
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	// Verbs that sound blocking are not siblings until the manifest says so.
	write(`{"name":"widget","groups":[{"name":"job","commands":[{"name":"status"},{"name":"wait"},{"name":"await-result"}]}]}`)
	before := ResolveCatalog("widget job status x")
	if before.State != availability.Resolved || len(BlockingSiblings("widget job status")) != 0 {
		t.Fatalf("first index: %+v siblings %v", before, BlockingSiblings("widget job status"))
	}
	if first, again := currentCatalog(), currentCatalog(); first != again {
		t.Fatal("an unchanged manifest rebuilt the index")
	}
	write(`{"name":"widget","groups":[{"name":"job","commands":[{"name":"status"},{"name":"wait","semantics":{"kind":"verify","blocks":{"replaces":["widget job status"]}}}]}]}`)
	after := ResolveCatalog("widget job wait x")
	if after.State != availability.Resolved || after.SemanticsKind != "verify" || after.Snapshot == before.Snapshot {
		t.Fatalf("after the edit: %+v (snapshot before %s)", after, before.Snapshot)
	}
	if got := BlockingSiblings("widget job status"); len(got) != 1 || got[0].Command() != "widget job wait" {
		t.Fatalf("siblings after the edit = %v", got)
	}
	if CurrentCatalogSnapshot() != after.Snapshot {
		t.Fatalf("snapshot %s, resolution used %s", CurrentCatalogSnapshot(), after.Snapshot)
	}
}
