package hub

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// fixtures returns the pages in testdata/<dir>, keyed by file name; the part
// of the name before "--" is what the page is expected to be.
func fixtures(t *testing.T, dir string) map[string][]byte {
	t.Helper()
	files, err := filepath.Glob(filepath.Join("testdata", dir, "*.html"))
	require.NoError(t, err)
	require.NotEmpty(t, files)
	pages := map[string][]byte{}
	for _, file := range files {
		page, err := os.ReadFile(file)
		require.NoError(t, err)
		pages[filepath.Base(file)] = page
	}
	return pages
}

func expectation(name string) string {
	expected, _, _ := strings.Cut(name, "--")
	return expected
}

// The frame shows a page in the scheme its colours were written for; the UI
// makes the same call from the same fixtures (theme.test.ts).
func TestPageColorSchemeFollowsThePagesOwnColours(t *testing.T) {
	for name, page := range fixtures(t, "page-schemes") {
		require.Equal(t, expectation(name), PageColorScheme(string(page)), name)
	}
}

func TestExplainerKitMirrorsTheUIKit(t *testing.T) {
	ui, err := os.ReadFile(filepath.Join("..", "..", "..", "ui", "src", "features", "packet", "explainer", "kit", "kit.css"))
	require.NoError(t, err)
	require.Equal(t, string(ui), kitCSS,
		"api/internal/hub/explainer_kit.css must equal the UI kit; copy ui/src/features/packet/explainer/kit/kit.css over it")
	// Text in the kit's own text colour on a white card: the kit's dark text
	// token makes it unreadable in dark mode only.
	_, found := pageKit.Unreadable([]byte(`<html><head><meta name="color-scheme" content="light dark"></head><body><div style="background:#ffffff">Card</div></body></html>`))
	require.Len(t, found, 1)
	require.Equal(t, SchemeDark, found[0].Scheme)
	require.Equal(t, "#f8fafc", found[0].Foreground, "the dark tokens come from the kit's dark block")
}

func TestPageReadabilityFindsTextTheOperatorCannotRead(t *testing.T) {
	for name, page := range fixtures(t, "page-readable") {
		_, found := pageKit.Unreadable(page)
		if expectation(name) == "readable" {
			require.Empty(t, found, name)
		} else {
			require.NotEmpty(t, found, name)
		}
	}
	// Every scheme fixture is readable as the frame shows it, including the
	// page that was unreadable in dark mode before it was shown in light.
	for name, page := range fixtures(t, "page-schemes") {
		_, found := pageKit.Unreadable(page)
		require.Empty(t, found, name)
	}
}

func TestPageUnreadableNamesTheTextTheColoursAndTheFix(t *testing.T) {
	page := fixtures(t, "page-readable")["unreadable--white-card-in-a-themed-page.html"]
	warnings := checkReadable(PacketExplainer{Field: "attachments[0]", Name: "page.html", HTML: page})
	require.Len(t, warnings, 1)
	message := warnings[0].Message
	require.Equal(t, WarnPageUnreadable, warnings[0].Code)
	require.Contains(t, message, `in dark mode "Option A" (h2 in div.card) is #f8fafc on #ffffff, contrast 1.0:1`)
	require.Contains(t, message, "var(--hub-surface)")
	require.Contains(t, message, "same rule")
}
