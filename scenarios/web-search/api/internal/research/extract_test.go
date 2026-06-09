package research_test

import (
	"strings"
	"testing"

	"web-search/internal/research"

	"github.com/stretchr/testify/require"
)

func TestExtractReadableTextStripsBoilerplate(t *testing.T) {
	html := `<html><head><style>.x{color:red}</style><script>var a=1;</script></head>
<body>
<nav><a href="/">Home</a><a href="/about">About</a></nav>
<header>Site banner</header>
<article><h1>The Title</h1><p>The body sentence one.</p><p>Body sentence two.</p></article>
<aside>Related ads</aside>
<footer>Copyright 2026</footer>
</body></html>`

	got := research.ExtractReadableText(html)

	require.Contains(t, got, "The Title")
	require.Contains(t, got, "The body sentence one.")
	require.Contains(t, got, "Body sentence two.")

	for _, boiler := range []string{"color:red", "var a=1", "Home", "About", "Site banner", "Related ads", "Copyright 2026"} {
		require.NotContains(t, got, boiler, "boilerplate %q must be stripped", boiler)
	}
}

func TestExtractReadableTextDecodesEntitiesAndCollapsesWhitespace(t *testing.T) {
	html := `<p>Tom &amp; Jerry   say&nbsp;hello</p>`
	got := research.ExtractReadableText(html)
	require.Equal(t, "Tom & Jerry say hello", got)
}

func TestExtractReadableTextPlainPassthrough(t *testing.T) {
	require.Equal(t, "just plain text", research.ExtractReadableText("  just plain text  "))
	require.Empty(t, research.ExtractReadableText(""))
}

func TestExtractReadableTextNonEmptyForArticle(t *testing.T) {
	// A representative article body must yield non-empty readable text.
	html := `<html><body><div class="content"><h1>Headline</h1>` +
		strings.Repeat("<p>A paragraph of real article content.</p>", 5) +
		`</div></body></html>`
	got := research.ExtractReadableText(html)
	require.NotEmpty(t, got)
	require.Contains(t, got, "Headline")
	require.Contains(t, got, "real article content")
}
