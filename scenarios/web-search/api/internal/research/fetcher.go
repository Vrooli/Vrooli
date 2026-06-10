// Package research is the L2/L3 deep-research domain that sits above the L0/L1
// live-search path.
//
// L2 (l2_pipeline.go) is a synchronous single-pass pipeline: take a query, ask
// the live-search service for the top-N candidate URLs, fetch each page via the
// Fetcher seam, extract readable text, and run a single always-cited synthesis
// over the fetched content. Auto-capture is opt-in for L2 — when requested the
// distilled claims are written to the findings store (FINDING_SOURCE_L2).
//
// L3 (l3_agent.go) hands the harder iterative research-and-reconcile loop to an
// agent-manager run and returns a run handle the caller polls.
//
// Every external dependency sits behind an interface seam so the pipeline is
// fully testable with no live network: Fetcher (the internal/research/fetch
// package's HTTP-first + browser-automation-studio escalation stack in
// production), Searcher (live-search), Synthesizer (LLM), and the
// agent-manager Client. main.go wires the production impls from env; tests
// inject fakes.
package research

import "context"

// Fetcher is the L2 page-fetch seam: it returns the readable body text of a
// URL. Production wires fetch.EscalatingFetcher (plain HTTP first, per-URL
// escalation to a browser-automation-studio capture); tests inject a fake to
// pin behavior with no live network.
//
// The seam deliberately stays one method: the 2026-06 live assessment showed
// a substrate-level route bug (the old browserless coupling) could hide
// behind seam-injected tests, so the substrate now lives in its own package
// (fetch/) with a non-hermetic live smoke test, and this seam carries no
// transport detail at all.
type Fetcher interface {
	// Fetch returns the readable text of url. A non-nil error means the fetch or
	// extraction failed (the pipeline skips that page and continues).
	Fetch(ctx context.Context, url string) (text string, err error)
}
