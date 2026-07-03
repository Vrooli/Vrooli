package migration

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLedgerAccountsForEveryLegacyAPIRuleFile(t *testing.T) {
	repoRoot := findRepoRoot(t)
	legacyDir := filepath.Join(repoRoot, "scenarios", "scenario-auditor", "api", "rules", "api")

	entries, err := os.ReadDir(legacyDir)
	require.NoError(t, err)

	var legacyFiles []string
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") || name == "types.go" {
			continue
		}
		legacyFiles = append(legacyFiles, name)
	}
	sort.Strings(legacyFiles)

	ledger := Ledger()
	ledgerFiles := make([]string, 0, len(ledger))
	seen := make(map[string]Rule, len(ledger))
	for _, rule := range ledger {
		require.NotEmpty(t, rule.File)
		require.NotEmpty(t, rule.RuleID)
		require.NotEmpty(t, rule.Owner)
		require.NotEmpty(t, rule.Rationale)
		require.NotEmpty(t, rule.APIHealthMapping)
		require.Contains(t, []Decision{
			DecisionRedesigned,
			DecisionKept,
			DecisionDelegated,
			DecisionDeferred,
			DecisionRejected,
		}, rule.Decision)
		if existing, ok := seen[rule.File]; ok {
			t.Fatalf("duplicate migration ledger entry for %s: %#v and %#v", rule.File, existing, rule)
		}
		seen[rule.File] = rule
		ledgerFiles = append(ledgerFiles, rule.File)
	}
	sort.Strings(ledgerFiles)

	require.Equal(t, legacyFiles, ledgerFiles)
}

func TestLedgerDocumentsBoundaryDecisions(t *testing.T) {
	ledger := Ledger()
	byFile := make(map[string]Rule, len(ledger))
	for _, rule := range ledger {
		byFile[rule.File] = rule
	}

	require.Equal(t, DecisionDelegated, byFile["security_headers.go"].Decision)
	require.Equal(t, "security-health", byFile["security_headers.go"].Owner)
	require.Equal(t, DecisionDelegated, byFile["file_close.go"].Decision)
	require.NotContains(t, byFile["file_close.go"].Owner, "api-health")

	require.Equal(t, DecisionRedesigned, byFile["http_response_close.go"].Decision)
	require.Equal(t, "api.response_body_unclosed", byFile["http_response_close.go"].APIHealthMapping)
	require.Equal(t, DecisionRedesigned, byFile["health_check.go"].Decision)
	require.Contains(t, byFile["health_check.go"].APIHealthMapping, "api.health_probe")
}

func findRepoRoot(t *testing.T) string {
	t.Helper()

	wd, err := os.Getwd()
	require.NoError(t, err)

	for {
		candidate := filepath.Join(wd, "scenarios", "scenario-auditor", "api", "rules", "api")
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return wd
		}
		parent := filepath.Dir(wd)
		if parent == wd {
			t.Fatal("repo root not found")
		}
		wd = parent
	}
}
