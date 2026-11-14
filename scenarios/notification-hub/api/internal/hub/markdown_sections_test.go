package hub

import (
	"fmt"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/require"
)

func TestMarkdownSectionsReadsLevelTwoAndThreeHeadings(t *testing.T) {
	doc := "---\ntitle: Plan\n## not a heading in front matter\n---\n# Title ignored\n\nintro\n\n## Storage\n\n### Retention `window`\n\n## Billing ##\n\n#### too deep\n\n    ## indented code\n"
	sections, _, err := markdownSections(strings.NewReader(doc), "plan.md")
	require.NoError(t, err)
	var got [][3]string
	for _, s := range sections {
		require.Equal(t, "plan.md", s.Attachment)
		got = append(got, [3]string{s.Key, s.Title, s.Anchor})
	}
	require.Equal(t, [][3]string{
		{"storage", "Storage", "storage"},
		{"retention-window", "Retention window", "retention-window"},
		{"billing", "Billing", "billing"},
	}, got)
}

func TestMarkdownSectionsIgnoresFencedCode(t *testing.T) {
	doc := "## Real\n\n```sh\n## comment, not a heading\n```\n\n~~~\n## also not\n~~~\n\n## After\n"
	sections, _, err := markdownSections(strings.NewReader(doc), "p")
	require.NoError(t, err)
	require.Len(t, sections, 2)
	require.Equal(t, "after", sections[1].Key)
}

func TestMarkdownSectionsKeepsKeysAndAnchorsUnique(t *testing.T) {
	doc := "## Plan\n## Plan\n## Plan\n## 日本語\n## ???\n## Plan 1\n"
	sections, _, err := markdownSections(strings.NewReader(doc), "p")
	require.NoError(t, err)
	var keys, anchors []string
	for _, s := range sections {
		keys = append(keys, s.Key)
		anchors = append(anchors, s.Anchor)
	}
	require.Equal(t, []string{"plan", "plan-1", "plan-2", "section-4", "section-5", "plan-1-1"}, keys)
	require.Equal(t, []string{"plan", "plan-1", "plan-2", "日本語", "", "plan-1-1"}, anchors,
		"anchors follow GitHub's slugger; a heading that slugs to the already-numbered plan-1 is renumbered")
	seen := map[string]bool{}
	for _, key := range keys {
		require.False(t, seen[key])
		seen[key] = true
	}
}

func TestMarkdownSectionsNeedsHeadings(t *testing.T) {
	_, _, err := markdownSections(strings.NewReader("# Only a title\n\ntext\n"), "p")
	require.ErrorContains(t, err, "no ## or ### headings")
}

func TestMarkdownSectionsKeysStayWithinTheHubKeyLimitAndUnique(t *testing.T) {
	long := strings.Repeat("a", 100)
	doc := "## " + long + "\n## " + long + "\n## " + long + "\n"
	sections, _, err := markdownSections(strings.NewReader(doc), "p")
	require.NoError(t, err)
	seen := map[string]bool{}
	for _, s := range sections {
		require.LessOrEqual(t, len(s.Key), maxSectionKeyLength, s.Key)
		require.False(t, seen[s.Key], "keys must stay unique after cutting: %s", s.Key)
		seen[s.Key] = true
	}
	require.True(t, strings.HasSuffix(sections[1].Key, "-2") || strings.HasSuffix(sections[1].Key, "-1"))
}

func TestMarkdownSectionsKeyCutNeverEndsInAHyphen(t *testing.T) {
	doc := "## " + strings.Repeat("a", 63) + " b\n"
	sections, _, err := markdownSections(strings.NewReader(doc), "p")
	require.NoError(t, err)
	require.Equal(t, strings.Repeat("a", 63), sections[0].Key)
}

func TestMarkdownSectionsCutTitlesAndAnchorsToTheHubLimits(t *testing.T) {
	doc := "## " + strings.Repeat("é", 300) + "\n## " + strings.Repeat("word ", 100) + "\n"
	sections, _, err := markdownSections(strings.NewReader(doc), "p")
	require.NoError(t, err)
	for _, s := range sections {
		require.LessOrEqual(t, utf8.RuneCountInString(s.Title), maxSectionTitle)
		require.LessOrEqual(t, utf8.RuneCountInString(s.Anchor), maxSectionAnchor)
		require.NotEmpty(t, s.Key)
	}
}

func TestMarkdownSectionsReturnsAtMostFortyAndCountsTheRest(t *testing.T) {
	var doc strings.Builder
	for i := 0; i < 55; i++ {
		fmt.Fprintf(&doc, "## S%d\n", i)
	}
	sections, total, err := markdownSections(strings.NewReader(doc.String()), "p")
	require.NoError(t, err)
	require.Len(t, sections, maxReviewSections)
	require.Equal(t, 55, total)
	require.Equal(t, "s39", sections[39].Key)
}
