package fetch_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"web-search/internal/research"
	"web-search/internal/research/fetch"

	"github.com/stretchr/testify/require"
)

// fakeLeg is a scriptable fetch leg recording its calls.
type fakeLeg struct {
	text  string
	err   error
	calls int
}

func (f *fakeLeg) Fetch(_ context.Context, _ string) (string, error) {
	f.calls++
	return f.text, f.err
}

// The escalating fetcher must satisfy the research-package seam — this is the
// compile-time pin for the production wiring in main.go.
var _ research.Fetcher = (*fetch.EscalatingFetcher)(nil)

func TestEscalatingFetcherGoodContentNoEscalation(t *testing.T) {
	httpLeg := &fakeLeg{text: strings.Repeat("good readable content. ", 20)}
	browser := &fakeLeg{text: "browser content"}
	f := &fetch.EscalatingFetcher{HTTP: httpLeg, Browser: browser}

	text, err := f.Fetch(context.Background(), "https://example.com")
	require.NoError(t, err)
	require.Contains(t, text, "good readable content")
	require.Equal(t, 1, httpLeg.calls)
	require.Zero(t, browser.calls, "browser leg must not fire for good HTTP content")
}

func TestEscalatingFetcherHTTPFailureEscalates(t *testing.T) {
	httpLeg := &fakeLeg{err: errors.New("connection refused")}
	browser := &fakeLeg{text: strings.Repeat("rendered content. ", 20)}
	f := &fetch.EscalatingFetcher{HTTP: httpLeg, Browser: browser}

	text, err := f.Fetch(context.Background(), "https://example.com")
	require.NoError(t, err)
	require.Contains(t, text, "rendered content")
	require.Equal(t, 1, browser.calls)
}

func TestEscalatingFetcherThinContentEscalates(t *testing.T) {
	httpLeg := &fakeLeg{text: "Please enable JavaScript"}
	browser := &fakeLeg{text: strings.Repeat("the actual article body. ", 20)}
	f := &fetch.EscalatingFetcher{HTTP: httpLeg, Browser: browser}

	text, err := f.Fetch(context.Background(), "https://example.com")
	require.NoError(t, err)
	require.Contains(t, text, "actual article body")
	require.Equal(t, 1, browser.calls)
}

func TestEscalatingFetcherNoBrowserLegHTTPOnly(t *testing.T) {
	// HTTP error with no browser leg: the error propagates.
	httpErr := &fakeLeg{err: errors.New("boom")}
	f := &fetch.EscalatingFetcher{HTTP: httpErr}
	_, err := f.Fetch(context.Background(), "https://example.com")
	require.Error(t, err)

	// Thin content with no browser leg: keep the thin text over nothing.
	thin := &fakeLeg{text: "tiny"}
	f = &fetch.EscalatingFetcher{HTTP: thin}
	text, err := f.Fetch(context.Background(), "https://example.com")
	require.NoError(t, err)
	require.Equal(t, "tiny", text)
}

func TestEscalatingFetcherBrowserFailureKeepsThinHTTPResult(t *testing.T) {
	httpLeg := &fakeLeg{text: "thin shell"}
	browser := &fakeLeg{err: errors.New("BAS down")}
	f := &fetch.EscalatingFetcher{HTTP: httpLeg, Browser: browser}

	text, err := f.Fetch(context.Background(), "https://example.com")
	require.NoError(t, err)
	require.Equal(t, "thin shell", text)
}

func TestEscalatingFetcherBothLegsFailReturnsHTTPError(t *testing.T) {
	httpLeg := &fakeLeg{err: errors.New("http boom")}
	browser := &fakeLeg{err: errors.New("browser boom")}
	f := &fetch.EscalatingFetcher{HTTP: httpLeg, Browser: browser}

	_, err := f.Fetch(context.Background(), "https://example.com")
	require.Error(t, err)
	require.Contains(t, err.Error(), "http boom")
}

func TestEscalatingFetcherCustomThreshold(t *testing.T) {
	// 30 chars of content with a 10-char threshold: no escalation.
	httpLeg := &fakeLeg{text: strings.Repeat("x", 30)}
	browser := &fakeLeg{text: "browser"}
	f := &fetch.EscalatingFetcher{HTTP: httpLeg, Browser: browser, MinReadableChars: 10}

	_, err := f.Fetch(context.Background(), "https://example.com")
	require.NoError(t, err)
	require.Zero(t, browser.calls)
}
