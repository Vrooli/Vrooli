package cache

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	factsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/code-facts/v1/facts"
)

func TestParseTargetSupportsScenarioPrefix(t *testing.T) {
	got := parseTarget("scenario:code-facts")

	require.Equal(t, factsv1.TargetKind_TARGET_KIND_SCENARIO, got.GetKind())
	require.Equal(t, "code-facts", got.GetScenario())
}

func TestParseTargetDefaultsToPath(t *testing.T) {
	got := parseTarget("/tmp/repo")

	require.Equal(t, factsv1.TargetKind_TARGET_KIND_PATH, got.GetKind())
	require.Equal(t, "/tmp/repo", got.GetPath())
}

func TestParseTargetBlankMeansClearAllCompatibleNilTarget(t *testing.T) {
	require.Nil(t, parseTarget("   "))
}

func TestCacheEntryLinesIncludeBudgetRelevantMetadata(t *testing.T) {
	lines := cacheEntryLines([]*factsv1.CacheMetadata{{
		Scope:      "graph",
		CacheKey:   "key-1",
		State:      "hit",
		SourceHash: "source",
		ConfigHash: "config",
		HitCount:   7,
	}})

	require.Len(t, lines, 1)
	line := lines[0]
	for _, want := range []string{"graph", "key-1", "hit", "source=source", "config=config", "hits=7"} {
		require.True(t, strings.Contains(line, want), "line %q should include %q", line, want)
	}
}
