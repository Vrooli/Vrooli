package runsignal

import "testing"

func TestSafeTokensRejectsCompoundShell(t *testing.T) {
	if _, ok := safeTokens("agent-manager run report x; rm x"); ok {
		t.Fatal("compound shell was accepted")
	}
	if tokens, ok := safeTokens("agent-manager run report x"); !ok || len(tokens) != 4 {
		t.Fatalf("tokens=%v ok=%v", tokens, ok)
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
