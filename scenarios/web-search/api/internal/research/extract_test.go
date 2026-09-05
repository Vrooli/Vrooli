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
<form class="cookie-consent">We value your privacy. Accept all cookies</form>
<article><h1>The Title</h1><p>The body sentence one.</p><p>Body sentence two.</p></article>
<aside>Related ads</aside>
<footer>Copyright 2026</footer>
</body></html>`

	got := research.ExtractReadableText(html)

	require.Contains(t, got, "The Title")
	require.Contains(t, got, "The body sentence one.")
	require.Contains(t, got, "Body sentence two.")

	for _, boiler := range []string{"color:red", "var a=1", "Home", "About", "Site banner", "Related ads", "Copyright 2026", "Accept all cookies"} {
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

// TestExtractReadableTextNonEmptyAcrossArticleFixtures pins the REQ-P1-001
// robustness bar: across a varied set of real-world article HTML shapes, text
// extraction must return non-empty readable text for at least 90% of them.
func TestExtractReadableTextNonEmptyAcrossArticleFixtures(t *testing.T) {
	fixtures := []string{
		// Semantic <article> markup.
		`<html><body><nav>menu</nav><article><h1>Go 1.25 released</h1><p>The release adds new tooling.</p></article><footer>(c)</footer></body></html>`,
		// Classic div-soup content wrapper.
		`<html><body><div id="page"><div class="content"><h2>Local elections</h2><p>Turnout rose this year.</p><p>Officials confirmed the count.</p></div></div></body></html>`,
		// Old-school table layout.
		`<html><body><table><tr><td>sidebar</td><td><h1>Recipe</h1><p>Mix flour and water.</p></td></tr></table></body></html>`,
		// Entity-heavy text.
		`<html><body><p>AT&amp;T &amp; Verizon&nbsp;announced &quot;new&quot; plans.</p></body></html>`,
		// Attribute-heavy modern markup with data attributes.
		`<html><body><main data-testid="article-body" class="css-1x2y"><p data-block="true">Markets closed higher on Friday.</p></main></body></html>`,
		// List-driven article body.
		`<html><body><article><h1>Checklist</h1><ul><li>Step one is preparation.</li><li>Step two is execution.</li></ul></article></body></html>`,
		// Heavily commented page.
		`<html><body><!-- header --><div><!-- ad slot --><p>Researchers published the dataset.</p><!-- footer --></div></body></html>`,
		// Plain-text (non-HTML) fetch result.
		`The plain body of a text/plain article response.`,
		// Blockquote-led long-form piece.
		`<html><body><article><blockquote>An opening quotation.</blockquote><p>The essay continues with analysis.</p></article></body></html>`,
		// Article wrapped in chrome (nav, aside, footer) plus inline script.
		`<html><body><nav>top nav</nav><script>track();</script><section><h1>Storm warning</h1><p>Heavy rain is expected tonight.</p></section><aside>related</aside><footer>legal</footer></body></html>`,
	}

	nonEmpty := 0
	for i, html := range fixtures {
		if research.ExtractReadableText(html) != "" {
			nonEmpty++
		} else {
			t.Logf("fixture %d extracted to empty text", i)
		}
	}
	ratio := float64(nonEmpty) / float64(len(fixtures))
	require.GreaterOrEqual(t, ratio, 0.9, "extraction must be non-empty for at least 90%% of article fixtures (got %d/%d)", nonEmpty, len(fixtures))
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
