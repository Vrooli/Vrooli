// Package fetch is web-search's L2 page-fetch layer: given a result URL it
// returns readable body text for synthesis.
//
// Greenfield design (2026-06, replacing the deleted browserless coupling):
//
//   - HTTPFetcher is the default leg — a plain, browserless-free HTTP GET with
//     readable-text extraction. Most result pages are server-rendered; paying
//     a real browser's 2–10s per page for them is waste.
//   - BASFetcher (bas.go) is the browser leg — a one-shot
//     browser-automation-studio CaptureService.Capture call with inline DOM
//     return. Browser rendering goes through the BAS scenario only; web-search
//     never talks to a raw browser resource.
//   - EscalatingFetcher composes them: try HTTP, escalate per-URL to the
//     browser leg when the HTTP leg fails outright or yields too little
//     readable text (the JS-shell heuristic). With no browser leg configured
//     (or BAS down) it degrades to HTTP-only.
//
// Every leg satisfies the research.Fetcher seam, so the L2 pipeline and its
// contract tests are untouched by the substrate swap.
package fetch

import (
	"context"
	"log"
	"strings"
)

// Fetcher mirrors research.Fetcher so this package stays import-cycle-free;
// satisfaction of the research seam is asserted where the legs are wired
// (main.go) and in tests.
type Fetcher interface {
	Fetch(ctx context.Context, url string) (text string, err error)
}

// DefaultMinReadableChars is the JS-shell heuristic threshold: an HTTP-leg
// result below this many extracted characters triggers browser escalation.
// Real article pages extract thousands of characters; JS shells typically
// extract a title and a "please enable JavaScript" notice.
const DefaultMinReadableChars = 200

// EscalatingFetcher tries the cheap HTTP leg first and escalates per-URL to
// the browser leg only when needed.
type EscalatingFetcher struct {
	// HTTP is the default leg. Required.
	HTTP Fetcher
	// Browser is the escalation leg (nil disables escalation entirely).
	Browser Fetcher
	// MinReadableChars overrides DefaultMinReadableChars when > 0.
	MinReadableChars int
	// Logger receives escalation decisions (nil = silent).
	Logger *log.Logger
}

// Fetch implements the research.Fetcher contract.
func (f *EscalatingFetcher) Fetch(ctx context.Context, url string) (string, error) {
	text, httpErr := f.HTTP.Fetch(ctx, url)
	threshold := f.MinReadableChars
	if threshold <= 0 {
		threshold = DefaultMinReadableChars
	}

	thin := httpErr == nil && len(strings.TrimSpace(text)) < threshold
	if httpErr == nil && !thin {
		return text, nil
	}
	if f.Browser == nil {
		if httpErr != nil {
			return "", httpErr
		}
		// Thin content with no browser leg: return what we have — the L2
		// pipeline tolerates weak pages better than missing pages.
		return text, nil
	}

	if f.Logger != nil {
		switch {
		case httpErr != nil:
			f.Logger.Printf("research: fetch %q escalating to browser (http leg failed: %v)", url, httpErr)
		default:
			f.Logger.Printf("research: fetch %q escalating to browser (thin content: %d chars)", url, len(strings.TrimSpace(text)))
		}
	}

	browserText, browserErr := f.Browser.Fetch(ctx, url)
	if browserErr != nil {
		if f.Logger != nil {
			f.Logger.Printf("research: fetch %q browser leg failed: %v", url, browserErr)
		}
		if httpErr == nil {
			return text, nil // keep the thin HTTP result over nothing
		}
		return "", httpErr
	}
	return browserText, nil
}
