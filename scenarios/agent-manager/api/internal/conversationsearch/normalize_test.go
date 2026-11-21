package conversationsearch

import (
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/stretchr/testify/require"
)

func TestNormalizerClassifiesContentPolicy(t *testing.T) {
	t.Parallel()
	normalizer := fixtureNormalizer(t, "recipe-v1", 512, 32)
	tests := []struct {
		name   string
		mutate func(*SourceMessage)
		class  ContentClass
	}{
		{name: "operator prose", class: ContentClassProse},
		{name: "quoted prose", mutate: func(source *SourceMessage) { source.Content = "> copied analysis\n> with a correction" }, class: ContentClassQuotedProse},
		{name: "system context", mutate: func(source *SourceMessage) { source.Role = "system" }, class: ContentClassInjectedContext},
		{name: "agents instructions", mutate: func(source *SourceMessage) { source.Content = "# AGENTS.md instructions\nDo work" }, class: ContentClassInjectedContext},
		{name: "tool call", mutate: func(source *SourceMessage) { source.Role = "tool_call" }, class: ContentClassToolCall},
		{name: "tool result", mutate: func(source *SourceMessage) { source.Role = "tool" }, class: ContentClassToolResult},
		{name: "evidence duplicate", mutate: func(source *SourceMessage) { source.EvidenceOnly = true }, class: ContentClassEvidenceOnlyDuplicate},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			source := testSourceMessage()
			if test.mutate != nil {
				test.mutate(&source)
			}
			documents, err := normalizer.Normalize(source)
			require.NoError(t, err)
			require.Len(t, documents, 1)
			require.Equal(t, test.class, documents[0].ContentClass)
			require.Equal(t, test.class == ContentClassProse || test.class == ContentClassQuotedProse, DefaultSearchContentClass(test.class))
		})
	}
}

func TestNormalizerExcludesDeletedDisallowedAndEmptyMessages(t *testing.T) {
	t.Parallel()
	normalizer := fixtureNormalizer(t, "recipe-v1", 512, 32)
	for _, mutate := range []func(*SourceMessage){
		func(source *SourceMessage) { source.Deleted = true },
		func(source *SourceMessage) { source.Disallowed = true },
		func(source *SourceMessage) { source.Content = " \r\n " },
	} {
		source := testSourceMessage()
		mutate(&source)
		documents, err := normalizer.Normalize(source)
		require.NoError(t, err)
		require.Empty(t, documents)
	}
}

func TestNormalizerChunksUnicodeDeterministicallyWithinByteBounds(t *testing.T) {
	t.Parallel()
	content := strings.Repeat("Measured phase capacity 🚀 remains deterministic. ", 20)
	normalizer := fixtureNormalizer(t, "recipe-v1", 128, 24)
	source := testSourceMessage()
	source.Content = content

	first, err := normalizer.Normalize(source)
	require.NoError(t, err)
	second, err := normalizer.Normalize(source)
	require.NoError(t, err)
	require.Equal(t, first, second)
	require.Greater(t, len(first), 2)
	for index, document := range first {
		require.LessOrEqual(t, len(document.Content), 128)
		require.True(t, utf8.ValidString(document.Content))
		require.Equal(t, index, document.ChunkIndex)
		require.Equal(t, len(first), document.ChunkTotal)
		require.Equal(t, content[document.StartByte:document.EndByte], document.Content)
	}
}

func TestNormalizerRecipeDriftChangesIdentityWithoutVolatileMetadata(t *testing.T) {
	t.Parallel()
	source := testSourceMessage()
	one, err := fixtureNormalizer(t, "recipe-v1", 512, 32).Normalize(source)
	require.NoError(t, err)
	two, err := fixtureNormalizer(t, "recipe-v2", 512, 32).Normalize(source)
	require.NoError(t, err)
	require.NotEqual(t, one[0].DocumentID, two[0].DocumentID)
	require.NotEqual(t, one[0].SourceHash, two[0].SourceHash)

	source.RunLabel = "renamed after indexing"
	source.RunStatus = "archived"
	volatile, err := fixtureNormalizer(t, "recipe-v1", 512, 32).Normalize(source)
	require.NoError(t, err)
	require.Equal(t, one[0].DocumentID, volatile[0].DocumentID)
	require.Equal(t, one[0].SourceHash, volatile[0].SourceHash)
	require.Equal(t, "renamed after indexing", volatile[0].RunLabel)
}

func TestGoldenCorrectionIsDefaultProseAndInjectedBoilerplateIsNot(t *testing.T) {
	t.Parallel()
	normalizer := fixtureNormalizer(t, "recipe-v1", 512, 32)
	correction := testSourceMessage()
	correction.Content = "The correction is that suite limits remain fixed, but phase admission is adaptive from measured resource history."
	injected := testSourceMessage()
	injected.EventID = "event-injected"
	injected.Content = "# AGENTS.md instructions\nCritical rules and setup tooling"

	correctionDocs, err := normalizer.Normalize(correction)
	require.NoError(t, err)
	injectedDocs, err := normalizer.Normalize(injected)
	require.NoError(t, err)
	require.Equal(t, ContentClassProse, correctionDocs[0].ContentClass)
	require.Equal(t, ContentClassInjectedContext, injectedDocs[0].ContentClass)
	require.NotEqual(t, correctionDocs[0].ContentClass, injectedDocs[0].ContentClass)
}

func TestNormalizerUsesStableEvidenceReferenceAndOmitsAttachmentBodies(t *testing.T) {
	t.Parallel()
	documents, err := fixtureNormalizer(t, "recipe-v1", 512, 32).Normalize(testSourceMessage())
	require.NoError(t, err)
	require.Equal(t, "agent-manager://runs/run-1/events/event-1", documents[0].EvidenceRef)
	require.NotContains(t, documents[0].EvidenceRef, "/home/")
}

func fixtureNormalizer(t *testing.T, recipe string, maximum, overlap int) *Normalizer {
	t.Helper()
	normalizer, err := NewNormalizer(NormalizerConfig{
		RecipeVersion: recipe,
		MaxChunkBytes: maximum,
		OverlapBytes:  overlap,
		Now: func() time.Time {
			return time.Date(2026, 9, 4, 12, 30, 0, 0, time.UTC)
		},
	})
	require.NoError(t, err)
	return normalizer
}

func testSourceMessage() SourceMessage {
	return SourceMessage{
		RunID: "run-1", EventID: "event-1", MessageID: "message-1", Sequence: 1,
		Role: "operator", OccurredAt: time.Date(2026, 9, 4, 12, 0, 0, 0, time.UTC),
		Content: "Review the corrected reasoning about phase admission.", Harness: "claude-code",
		SourceSessionID: "session-public", ProviderOrigin: "claude", Importer: "agent-manager.transcript-import",
		ProjectScope: "/workspace/project", CWDScope: "/workspace/project", Runner: "claude-code",
		Model: "model", Profile: "profile", RunStatus: "complete", RunLabel: "fixture",
		Tags: []string{" recall ", "recall"}, Workloads: []string{"implementation"},
	}
}
