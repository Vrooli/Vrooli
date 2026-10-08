package policy_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// The engine layer appends, embeds, clusters, summarizes, retrieves, collapses
// and federates. None of that is specific to coding agents. A second policy
// scope — a team ledger, a conjecture search, an investigation corpus — is a
// data and configuration change only if the engine never learns this scope's
// vocabulary. This test is what keeps that true under future edits.
//
// If this fails, the fix is to move the value into internal/policy and resolve
// it through the scope, not to add the new literal to the allow list.
func TestEnginePackagesNameNoDomainVocabulary(t *testing.T) {
	enginePackages := []string{"../recall", "../forest", "../journal"}

	// Facet ids and retention policy names are this scope's vocabulary. Both are
	// declared in facet_definitions/facet_policies and resolved per scope.
	forbidden := []string{
		"standing-rule", "environment-fact", "gotcha", "episode", "thread", "entity-record",
		"pinned-or-review", "expire-on-resolution",
	}

	for _, pkg := range enginePackages {
		entries, err := os.ReadDir(pkg)
		require.NoError(t, err)
		for _, entry := range entries {
			name := entry.Name()
			if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
				continue
			}
			path := filepath.Join(pkg, name)
			data, err := os.ReadFile(path)
			require.NoError(t, err)
			for _, line := range strings.Split(string(data), "\n") {
				for _, word := range forbidden {
					// Report the offending line, never the whole file: a dumped
					// engine source drowns the one fact the reader needs.
					require.NotContainsf(t, line, word,
						"%s names the domain value %q in:\n\t%s\nEngine packages must resolve domain vocabulary through internal/policy so a second scope needs no engine change.",
						path, word, strings.TrimSpace(line))
				}
			}
		}
	}
}
