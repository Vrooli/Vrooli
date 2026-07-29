package runreport

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
	got := resolveCatalog("agent-manager run cohort-report --run-ids x")
	if got.State != "resolved" || got.Command != "agent-manager run cohort-report" || got.Snapshot == "" {
		t.Fatalf("resolution=%+v root=%s", got, projectRoot())
	}
}

func TestResolveCatalogUnwrapsLiteralRunnerShellEnvelope(t *testing.T) {
	got := resolveCatalog("/bin/bash -lc 'agent-manager run cohort-report --run-ids x'")
	if got.State != "resolved" || got.Command != "agent-manager run cohort-report" {
		t.Fatalf("resolution=%+v", got)
	}
	if got := resolveCatalog("/bin/bash -lc 'agent-manager run report x; rm x'"); got.State != "unknown" {
		t.Fatalf("compound wrapper resolution=%+v", got)
	}
}
