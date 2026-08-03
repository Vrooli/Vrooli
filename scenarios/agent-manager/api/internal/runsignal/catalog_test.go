package runsignal

import (
	"testing"

	"agent-manager/internal/availability"
)

func TestSafeTokensRejectsCompoundShell(t *testing.T) {
	if _, ok := safeTokens("agent-manager run report x; rm x"); ok {
		t.Fatal("compound shell was accepted")
	}
	if tokens, ok := safeTokens("agent-manager run report x"); !ok || len(tokens) != 4 {
		t.Fatalf("tokens=%v ok=%v", tokens, ok)
	}
	if tokens, ok := safeTokens(`grep '$4=="1w"' input`); !ok || len(tokens) != 3 {
		t.Fatalf("single-quoted literal tokens=%v ok=%v", tokens, ok)
	}
	if _, ok := safeTokens(`echo "$HOME"`); ok {
		t.Fatal("double-quoted expansion was accepted")
	}
}

func TestSegmentShellSplitsOnlyOutsideQuotes(t *testing.T) {
	segments, compound, reason := SegmentShell(`cd "a|b" && vrooli scenario test agent-manager | tee out`)
	if reason != "" || !compound || len(segments) != 3 || segments[0] != `cd "a|b"` {
		t.Fatalf("segments=%v compound=%v reason=%q", segments, compound, reason)
	}
}

func TestSegmentShellRefusesEvaluation(t *testing.T) {
	for _, command := range []string{"echo $(pwd)", "cat <(echo x)", "echo `pwd`"} {
		if segments, _, reason := SegmentShell(command); len(segments) != 0 || reason == "" {
			t.Fatalf("command %q segments=%v reason=%q", command, segments, reason)
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
	catalogCache.root, catalogCache.owners, catalogCache.snapshot = "", nil, ""
	catalogCache.Unlock()
	got := ResolveCatalog("agent-manager run cohort-report --run-ids x")
	if got.State != "resolved" || got.Command != "agent-manager run cohort-report" || got.Snapshot == "" {
		t.Fatalf("resolution=%+v root=%s", got, projectRoot())
	}
}

func TestResolveCatalogUnwrapsLiteralRunnerShellEnvelope(t *testing.T) {
	got := ResolveCatalog("/bin/bash -lc 'agent-manager run cohort-report --run-ids x'")
	if got.State != "resolved" || got.Command != "agent-manager run cohort-report" {
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
