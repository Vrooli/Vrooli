package agentharness

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// removalFixture is a host with no ambient state: a home holding the
// repository, a separate temporary root, and a variable table the test owns.
type removalFixture struct {
	home, repo, temp string
	vars             map[string]string
}

func newRemovalFixture(t *testing.T) removalFixture {
	t.Helper()
	base := t.TempDir()
	f := removalFixture{home: filepath.Join(base, "home"), temp: filepath.Join(base, "tmp"), vars: map[string]string{}}
	f.repo = filepath.Join(f.home, "Vrooli")
	for _, dir := range []string{filepath.Join(f.repo, ".git"), filepath.Join(f.repo, ".vrooli"), filepath.Join(f.repo, "scratch"), filepath.Join(f.repo, "scenarios", "app"), f.temp} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(f.repo, "notes.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	f.vars["HOME"] = f.home
	return f
}

func (f removalFixture) context(cwd string, rules ...PathRule) RemovalContext {
	return RemovalContext{
		WorkingDirectory: cwd,
		Home:             f.home,
		RepoRoot:         f.repo,
		ProjectConfigDir: ".vrooli",
		ScratchDir:       "scratch",
		TempRoots:        []string{f.temp},
		Lookup: func(name string) (string, bool) {
			value, ok := f.vars[name]
			return value, ok
		},
		Stat:  os.Lstat,
		Rules: rules,
	}
}

func (f removalFixture) decide(cwd, command string, rules ...PathRule) Decision {
	ctx := f.context(cwd, rules...)
	return ctx.EvaluateRemoval(parseRemovals(command, dialectPOSIX, removalEnv{WorkingDirectory: cwd, Home: ctx.Home, Lookup: ctx.Lookup}))
}

func TestRemovalFloorDecisions(t *testing.T) {
	f := newRemovalFixture(t)
	cases := []struct {
		name, cwd, command string
		want               DecisionAction
	}{
		{"named repository file", f.repo, "rm scenarios/web-console/ui/src/lib/viewportCorrection.ts", ActionAllow},
		{"absolute repository file", f.repo, "rm -f " + filepath.Join(f.repo, "a.txt"), ActionAllow},
		{"recursive flag on an existing file", f.repo, "rm -rf notes.txt", ActionAllow},
		{"temporary tree", f.repo, "rm -rf " + filepath.Join(f.temp, "work"), ActionAllow},
		{"temporary pattern", f.repo, "rm -rf " + filepath.Join(f.temp, "work") + "/*", ActionAllow},
		{"scratch tree", f.repo, "rm -rf scratch/probe", ActionAllow},
		{"scratch contents", f.repo, "rm -rf scratch/*", ActionAllow},
		{"find delete under temp", f.repo, "find " + filepath.Join(f.temp, "x") + " -name '*.log' -delete", ActionAllow},
		{"cd then relative delete", f.repo, "cd " + f.temp + " && rm -rf build", ActionAllow},
		{"assigned variable", f.repo, "X=" + filepath.Join(f.temp, "w") + "; rm -rf \"$X\"", ActionAllow},
		{"nested shell keeps the working directory", f.repo, "bash -c 'rm -f notes.txt'", ActionAllow},

		{"repository tree", f.repo, "rm -r scenarios/app", ActionAsk},
		{"repository pattern", f.repo, "rm *.log", ActionAsk},
		{"project configuration", f.repo, "rm .vrooli/service.json", ActionAsk},
		{"scratch root itself", f.repo, "rm -rf scratch", ActionAsk},
		{"temporary root itself", f.repo, "rm -rf " + f.temp, ActionAsk},
		{"unset variable", f.repo, "rm -rf $UNSET_VAR/x", ActionAsk},
		{"command substitution target", f.repo, "rm -rf \"$(mktemp -d)\"", ActionAsk},
		{"loop variable", f.repo, "for f in a b; do rm \"$f\"; done", ActionAsk},
		{"xargs hides its targets", f.repo, "find . -name '*.tmp' | xargs rm", ActionAsk},
		{"unparseable deletion", f.repo, "rm -rf 'unterminated", ActionAsk},
		{"relative path from unknown directory", "", "rm notes.txt", ActionAsk},

		{"filesystem root", f.repo, "rm -rf /", ActionDeny},
		{"sudo filesystem root", f.repo, "sudo rm -rf /", ActionDeny},
		{"system directory", f.repo, "rm -rf /etc", ActionDeny},
		{"outside every zone", f.repo, "rm -rf /usr/lib/example", ActionDeny},
		{"home root", f.repo, "rm -rf ~", ActionDeny},
		{"home contents", f.repo, "rm -rf ~/*", ActionDeny},
		{"home file", f.repo, "rm " + filepath.Join(f.home, "notes.txt"), ActionDeny},
		{"home tree", f.repo, "rm -rf ~/Documents", ActionDeny},
		{"home through a variable", f.repo, "rm -rf \"$HOME/Documents\"", ActionDeny},
		{"repository root", f.repo, "rm -rf " + f.repo, ActionDeny},
		{"repository contents recursively", f.repo, "rm -rf ./*", ActionDeny},
		{"parent of the working directory", filepath.Join(f.repo, "scenarios"), "rm -rf ..", ActionDeny},
		{"git history", f.repo, "rm .git/index.lock", ActionDeny},
		{"cd home then delete", f.repo, "cd ~ && rm -rf Documents", ActionDeny},
		{"worst target wins", f.repo, "rm -rf " + filepath.Join(f.temp, "ok") + " " + filepath.Join(f.home, "bad"), ActionDeny},
		{"quoted nested shell", f.repo, "bash -c \"rm -rf ~\"", ActionDeny},
		{"command substitution runs", f.repo, "echo $(rm -rf ~)", ActionDeny},
		{"find delete in home", f.repo, "find ~ -delete", ActionDeny},
		{"find exec rm at root", f.repo, "find / -exec rm {} \\;", ActionDeny},
		{"rsync delete into home", f.repo, "rsync -a --delete src/ ~/backup", ActionDeny},
		{"alias-bypassing escape", f.repo, "\\rm -rf ~/Documents", ActionDeny},
		{"absolute binary", f.repo, "/bin/rm -rf ~/Documents", ActionDeny},
		{"wrapped by env and timeout", f.repo, "env FOO=1 timeout 5 rm -rf ~/Documents", ActionDeny},
		{"eval", f.repo, "eval \"rm -rf ~/Documents\"", ActionDeny},
		{"here-document fed to a shell", f.repo, "bash <<'EOF'\nrm -rf ~/Documents\nEOF", ActionDeny},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := f.decide(tc.cwd, tc.command)
			if got.Action != tc.want {
				t.Fatalf("%q decided %s (%s), want %s", tc.command, got.Action, got.Reason, tc.want)
			}
			if got.Risk != RiskFilesystemRemoval || got.Reason == "" {
				t.Fatalf("decision = %+v, want a removal decision with a reason", got)
			}
		})
	}
}

func TestRemovalParserIgnoresNonDeletions(t *testing.T) {
	for _, command := range []string{
		"find . -name '*.go' | grep x",
		"grep -rn 'rm -rf' docs",
		"echo rm -rf /",
		"ls # rm -rf /",
		"cat <<EOF\nrm -rf /\nEOF",
		"git rm tracked.txt",
		"command -v rm",
		"find . -name x -exec grep y {} \\;",
		"rm",
	} {
		if targets := parseRemovals(command, dialectPOSIX, removalEnv{WorkingDirectory: "/work"}); len(targets) != 0 {
			t.Errorf("%q produced targets %+v, want none", command, targets)
		}
	}
}

func TestRemovalParserTargets(t *testing.T) {
	env := removalEnv{WorkingDirectory: "/work", Home: "/home/u", Lookup: func(string) (string, bool) { return "", false }}
	cases := []struct {
		command   string
		want      []string
		recursive bool
	}{
		{"rm x 2>/dev/null", []string{"/work/x"}, false},
		{"rm -- -weird", []string{"/work/-weird"}, false},
		{"x=$(rm -rf /y)", []string{"/y"}, true},
		{"find . -type f -exec rm {} +", []string{"/work"}, true},
		{"find -L /a /b -delete", []string{"/a", "/b"}, true},
		{"eval eval eval rm -rf /x", []string{"/x"}, true},
		{"truncate -s 0 /work/log", []string{"/work/log"}, false},
		{"unlink a", []string{"/work/a"}, false},
		{"(cd /elsewhere && rm -r y); rm -r z", []string{"/elsewhere/y", "/work/z"}, true},
		{"$'rm' -r q", []string{"/work/q"}, true},
	}
	for _, tc := range cases {
		targets := parseRemovals(tc.command, dialectPOSIX, env)
		var paths []string
		for _, target := range targets {
			paths = append(paths, target.Path)
			if target.Recursive != tc.recursive || target.Unresolved != "" {
				t.Errorf("%q target %+v, want recursive=%v and resolved", tc.command, target, tc.recursive)
			}
		}
		if strings.Join(paths, "|") != strings.Join(tc.want, "|") {
			t.Errorf("%q targets = %v, want %v", tc.command, paths, tc.want)
		}
	}
}

func TestRemovalParserStopsAtNestingLimit(t *testing.T) {
	targets := parseRemovals("eval eval eval eval rm -rf /x", dialectPOSIX, removalEnv{WorkingDirectory: "/work"})
	if len(targets) != 1 || targets[0].Unresolved == "" {
		t.Fatalf("targets = %+v, want one unresolved target past the nesting limit", targets)
	}
}

func TestRemovalWindowsShells(t *testing.T) {
	f := newRemovalFixture(t)
	f.vars["WORK"] = filepath.Join(f.temp, "work")
	decide := func(dialect, command string) Decision {
		ctx := f.context(f.repo)
		event := ToolEvent{Runner: "codex", Tool: "shell", Shell: command, Context: map[string]string{"shell_dialect": dialect}}
		return ctx.EvaluateRemoval(removalTargetsFor(event, removalEnv{WorkingDirectory: f.repo, Home: f.home, Lookup: ctx.Lookup}))
	}
	cases := []struct {
		dialect, command string
		want             DecisionAction
	}{
		{"powershell", "Remove-Item -Path scratch/x -Recurse -Force", ActionAllow},
		{"powershell", "Remove-Item -LiteralPath notes.txt -ErrorAction SilentlyContinue", ActionAllow},
		{"powershell", "ri $env:WORK -r", ActionAllow},
		{"powershell", "Remove-Item -Recurse -Force ~/Documents", ActionDeny},
		{"powershell", "rm -r $HOME", ActionDeny},
		{"powershell", "Remove-Item scratch/a,~/b", ActionDeny},
		{"powershell", "Get-ChildItem x | Remove-Item", ActionAsk},
		{"powershell", "Remove-Item $target -Recurse", ActionAsk},
		{"powershell", "Set-Location ~; Remove-Item Documents -Recurse", ActionDeny},
		// Paths use the host separator: backslash is only a separator on Windows.
		{"cmd", "rd /s /q " + filepath.Join("scenarios", "app"), ActionAsk},
		{"cmd", "del /s /q %WORK%", ActionAllow},
		{"cmd", "rmdir /s /q %USERPROFILE_UNSET%", ActionAsk},
		{"", "pwsh -NoProfile -Command \"Remove-Item -Recurse ~/Documents\"", ActionDeny},
		{"", "powershell -EncodedCommand AAAA", ActionAsk},
		{"", "cmd /c \"rd /s /q " + filepath.Join("scratch", "old") + "\"", ActionAllow},
	}
	for _, tc := range cases {
		if got := decide(tc.dialect, tc.command); got.Action != tc.want {
			t.Errorf("[%s] %q decided %s (%s), want %s", tc.dialect, tc.command, got.Action, got.Reason, tc.want)
		}
	}
	if targets := parseRemovals("Remove-Item x -WhatIf", dialectPowerShell, removalEnv{WorkingDirectory: "/work"}); len(targets) != 0 {
		t.Fatalf("-WhatIf produced targets %+v, want none", targets)
	}
	targets := parseRemovals(`Remove-Item -LiteralPath 'C:\Users\me\x' -Recurse`, dialectPowerShell, removalEnv{WorkingDirectory: `C:\work`})
	if len(targets) != 1 || targets[0].Raw != `'C:\Users\me\x'` || !targets[0].Recursive {
		t.Fatalf("literal path targets = %+v, want the quoted Windows path, recursive", targets)
	}
}

func TestRemovalProviderRules(t *testing.T) {
	f := newRemovalFixture(t)
	cache := filepath.Join(f.home, ".cache", "tool")
	logs := filepath.Join(f.home, ".vrooli", "logs")
	keep := filepath.Join(f.temp, "cache", "keep")
	rules := []PathRule{
		{Root: cache, Action: ActionAllow, Reason: "regenerable cache", Source: "storage.roots/tool-cache", Provider: "storage-manager"},
		{Root: logs, Action: ActionAsk, Reason: "owner-managed logs", Source: "storage.roots/logs", Provider: "storage-manager"},
		{Root: filepath.Join(f.temp, "cache"), Action: ActionAllow, Reason: "regenerable", Source: "storage.roots/temp-cache", Provider: "storage-manager"},
		{Root: keep, Action: ActionDeny, Reason: "pinned artifacts", Source: "storage.roots/keep", Provider: "storage-manager"},
		{Root: f.home, Action: ActionAllow, Reason: "an over-broad declaration", Source: "storage.roots/home", Provider: "rogue"},
	}
	cases := []struct {
		command string
		want    DecisionAction
		source  string
	}{
		{"rm -rf " + filepath.Join(cache, "x"), ActionAllow, "storage-manager"},
		{"rm " + filepath.Join(logs, "a.log"), ActionAsk, "storage-manager"},
		{"rm -rf " + filepath.Join(f.temp, "cache", "junk"), ActionAllow, "storage-manager"},
		{"rm -rf " + filepath.Join(f.temp, "cache"), ActionDeny, "storage-manager"},
		{"rm " + filepath.Join(f.home, "notes.txt"), ActionDeny, ""},
		{"rm -rf " + filepath.Join(f.home, ".cache"), ActionDeny, ""},
	}
	for _, tc := range cases {
		got := f.decide(f.repo, tc.command, rules...)
		if got.Action != tc.want || got.ProviderID != tc.source {
			t.Errorf("%q decided %s by %q (%s), want %s by %q", tc.command, got.Action, got.ProviderID, got.Reason, tc.want, tc.source)
		}
	}
}

func TestRemovalResolvesSymlinksInTheParentOnly(t *testing.T) {
	f := newRemovalFixture(t)
	link := filepath.Join(f.temp, "link")
	if err := os.Symlink(f.home, link); err != nil {
		t.Skip("symlinks unavailable:", err)
	}
	if got := f.decide(f.repo, "rm "+link); got.Action != ActionAllow {
		t.Fatalf("deleting the link decided %s (%s), want allow", got.Action, got.Reason)
	}
	if got := f.decide(f.repo, "rm -rf "+link+"/"); got.Action != ActionDeny {
		t.Fatalf("deleting through the link decided %s (%s), want deny", got.Action, got.Reason)
	}
}

func TestRemovalComparesCaseInsensitivelyWhenTheHostDoes(t *testing.T) {
	f := newRemovalFixture(t)
	ctx := f.context(f.repo)
	ctx.CaseInsensitive = true
	upper := strings.ToUpper(f.home)
	got := ctx.EvaluateRemoval(parseRemovals("rm -rf "+upper, dialectPOSIX, removalEnv{WorkingDirectory: f.repo}))
	if got.Action != ActionDeny {
		t.Fatalf("case-folded home decided %s (%s), want deny", got.Action, got.Reason)
	}
}

func pathSnapshot(now time.Time, rules ...PathRule) ProviderSnapshot {
	snapshot := testSnapshot(now, "storage-manager", MaturityEnforcing, HealthHealthy, EvidenceClean)
	snapshot.Capabilities[0].ID = "filesystem-dispositions"
	snapshot.Rules = []PolicyRule{{Risk: RiskDependencyAdd, Action: ActionAllow, Reason: "must not leak into dependency decisions"}}
	snapshot.Scope.Risks = []RiskClass{RiskFilesystemRemoval}
	snapshot.PathRules = rules
	return snapshot
}

func TestRuntimeDecidesRemovalFromFloorAndFreshRules(t *testing.T) {
	f := newRemovalFixture(t)
	now := time.Now().UTC()
	cache := filepath.Join(f.home, ".cache", "tool")
	store := NewBundleStore(t.TempDir())
	rt := Runtime{Profile: ProfileAdvisory, Store: store, Now: func() time.Time { return now }, Removal: func(cwd string) RemovalContext { return f.context(cwd) }}
	event := ToolEvent{Runner: "claude-code", Tool: "Bash", Shell: "rm -rf " + filepath.Join(cache, "x"), WorkingDirectory: f.repo}

	missing, err := rt.Evaluate(event)
	if err != nil {
		t.Fatal(err)
	}
	if missing.Action != ActionDeny || !missing.Degraded || !hasEvidence(missing, "PATH_RULES_UNAVAILABLE") {
		t.Fatalf("without rules = %+v, want a degraded floor deny", missing)
	}

	if err := store.PublishProvider(pathSnapshot(now, PathRule{Root: cache, Action: ActionAllow, Reason: "regenerable cache"})); err != nil {
		t.Fatal(err)
	}
	fresh, err := rt.Evaluate(event)
	if err != nil {
		t.Fatal(err)
	}
	if fresh.Action != ActionAllow || fresh.Degraded || fresh.ProviderID != "storage-manager" || fresh.EventID == "" {
		t.Fatalf("with fresh rules = %+v, want allow by storage-manager", fresh)
	}

	rt.Now = func() time.Time { return now.Add(2 * time.Hour) }
	stale, err := rt.Evaluate(event)
	if err != nil {
		t.Fatal(err)
	}
	if stale.Action != ActionDeny || !hasEvidence(stale, "PATH_RULES_UNUSABLE") {
		t.Fatalf("with expired rules = %+v, want the floor deny with unusable evidence", stale)
	}
}

func TestRemovalIgnoresRolloutProfile(t *testing.T) {
	f := newRemovalFixture(t)
	for _, profile := range []RolloutProfile{ProfileAdvisory, ProfileEnforcing} {
		rt := Runtime{Profile: profile, Removal: func(cwd string) RemovalContext { return f.context(cwd) }}
		allowed, err := rt.Evaluate(ToolEvent{Runner: "codex", Tool: "shell", Shell: "rm scratch/a", WorkingDirectory: f.repo})
		if err != nil {
			t.Fatal(err)
		}
		denied, err := rt.Evaluate(ToolEvent{Runner: "codex", Tool: "shell", Shell: "rm -rf ~/Documents", WorkingDirectory: f.repo})
		if err != nil {
			t.Fatal(err)
		}
		if allowed.Action != ActionAllow || denied.Action != ActionDeny {
			t.Fatalf("profile %s decided %s and %s, want allow and deny", profile, allowed.Action, denied.Action)
		}
	}
}

func TestRiskScopedSnapshotDoesNotVouchForOtherRisks(t *testing.T) {
	now := time.Now().UTC()
	store := NewBundleStore(t.TempDir())
	if err := store.PublishProvider(pathSnapshot(now, PathRule{Root: "/srv/cache", Action: ActionAllow, Reason: "regenerable"})); err != nil {
		t.Fatal(err)
	}
	decision, err := Runtime{Profile: ProfileGuarded, Store: store, Now: func() time.Time { return now }}.Evaluate(testEvent())
	if err != nil {
		t.Fatal(err)
	}
	if decision.Action != ActionDeny || decision.UsedSnapshot {
		t.Fatalf("dependency decision = %+v, want the guarded fallback deny with no snapshot used", decision)
	}
}

func TestClassifierRecognisesRemovalBeforeOpaque(t *testing.T) {
	if got := ClassifyToolEvent(ToolEvent{Runner: "codex", Tool: "shell", Shell: "cd /tmp && rm -rf build"}); got != RiskFilesystemRemoval {
		t.Fatalf("compound deletion classified %s, want %s", got, RiskFilesystemRemoval)
	}
	if got := ClassifyToolEvent(ToolEvent{Runner: "codex", Tool: "shell", Shell: "ls -la | grep rm"}); got != RiskOpaque {
		t.Fatalf("pipeline mentioning rm classified %s, want %s", got, RiskOpaque)
	}
	if got := ClassifyToolEvent(ToolEvent{Runner: "codex", Tool: "rm", Arguments: []string{"-rf", "/x"}}); got != RiskFilesystemRemoval {
		t.Fatalf("argv deletion classified %s, want %s", got, RiskFilesystemRemoval)
	}
}

func TestValidateSnapshotRejectsUnsafePathRules(t *testing.T) {
	now := time.Now().UTC()
	for name, rule := range map[string]PathRule{
		"relative root":   {Root: "cache", Action: ActionAllow, Reason: "x"},
		"filesystem root": {Root: "/", Action: ActionAllow, Reason: "x"},
		"unknown action":  {Root: "/srv/cache", Action: ActionRoute, Reason: "x"},
		"missing reason":  {Root: "/srv/cache", Action: ActionAllow},
	} {
		if err := ValidateSnapshot(pathSnapshot(now, rule)); err == nil {
			t.Errorf("%s: snapshot validated, want a rejection", name)
		}
	}
}

func hasEvidence(decision Decision, code string) bool {
	for _, evidence := range decision.Evidence {
		if evidence.Code == code {
			return true
		}
	}
	return false
}
