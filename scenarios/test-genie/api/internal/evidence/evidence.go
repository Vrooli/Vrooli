package evidence

import (
	"fmt"
	"strings"
)

// Status is the analyzed outcome of a piece of [Evidence].
type Status string

const (
	// StatusPassed means the surface loaded and every assertion held.
	StatusPassed Status = "passed"
	// StatusFailed means at least one assertion failed; [Verdict.Message]
	// carries the single most significant reason.
	StatusFailed Status = "failed"
)

// ConsoleEntry is one browser console message captured during the session.
type ConsoleEntry struct {
	// Level is the console level as reported by the browser
	// ("log", "info", "warn"/"warning", "error", ...).
	Level string
	// Message is the rendered console text.
	Message string
}

// NetworkEntry is one failed or erroring network request captured during the
// session (a 4xx/5xx response, or a request that failed outright).
type NetworkEntry struct {
	URL          string
	Method       string
	ResourceType string
	// Status is the HTTP status code when the request produced a response;
	// nil when the request failed before a response (e.g. a transport error).
	Status *int
	// ErrorText is the transport error text when Status is nil.
	ErrorText string
}

// PageError is one uncaught page-level JavaScript exception.
type PageError struct {
	Message string
	Stack   string
}

// Handshake captures the iframe-bridge readiness outcome. A scenario UI that
// embeds @vrooli/iframe-bridge must signal ready once it boots inside the host
// frame; Signaled == false is a hard failure.
type Handshake struct {
	Signaled   bool
	TimedOut   bool
	DurationMs int64
	Error      string
}

// StorageShimEntry records a storage-API patch result observed in the page
// (the iframe-bridge shims localStorage/sessionStorage in sandboxed frames).
type StorageShimEntry struct {
	Prop    string
	Patched bool
	Reason  string
}

// Evidence is the engine-agnostic observation set from loading a single UI
// surface in a browser. It carries only the browser execution signals Test
// Genie owns: load completion, iframe-bridge readiness, console messages,
// network failures, and page exceptions. Generic visual judgments over
// screenshots, DOM, and layout artifacts are delegated to ui-health.
type Evidence struct {
	// URL is the surface that was loaded.
	URL string
	// Label optionally names the surface (e.g. a page path) for multi-surface
	// runs; empty for the single-page case.
	Label string
	// Loaded reports whether the browser session itself executed to completion.
	// False means the automation failed before producing observations (an
	// engine/navigation error), and LoadError explains why.
	Loaded bool
	// LoadError carries the engine error when Loaded is false.
	LoadError string

	Handshake   Handshake
	Console     []ConsoleEntry
	Network     []NetworkEntry
	PageErrors  []PageError
	StorageShim []StorageShimEntry

	// ScreenshotRef points at the captured screenshot artifact (path or id).
	// Carried for diagnostics; it does not affect the verdict.
	ScreenshotRef string
}

// Verdict is the analyzed outcome of one [Evidence].
type Verdict struct {
	Status Status
	// Message is the single most significant outcome line: a passing summary,
	// or the highest-precedence failure reason.
	Message string

	NetworkFailureCount int
	PageErrorCount      int
	ConsoleErrorCount   int
	ConsoleWarningCount int
}

// Passed reports whether the verdict is a pass.
func (v Verdict) Passed() bool { return v.Status == StatusPassed }

// Analyze applies the UI-load verdict rules to a piece of evidence.
//
// Failure precedence, highest first:
//  1. the browser session did not execute (engine/navigation failure);
//  2. the iframe-bridge handshake never signaled ready;
//  3. one or more network requests failed;
//  4. one or more uncaught page exceptions occurred.
//
// Console errors are counted but are not, on their own, a failure (a passing UI
// may log handled errors); console warnings are surfaced as a count for the
// caller to report as a warning. Pixel/DOM/layout visual findings are owned by
// ui-health and should be surfaced by the caller as separate observations.
func Analyze(ev Evidence) Verdict {
	v := Verdict{
		Status:              StatusPassed,
		Message:             "UI loaded successfully",
		NetworkFailureCount: len(ev.Network),
		PageErrorCount:      len(ev.PageErrors),
	}

	for _, entry := range ev.Console {
		switch entry.Level {
		case "error":
			v.ConsoleErrorCount++
		case "warn", "warning":
			v.ConsoleWarningCount++
		}
	}

	switch {
	case !ev.Loaded:
		v.Status = StatusFailed
		v.Message = ev.LoadError
		if v.Message == "" {
			v.Message = "browser execution failed"
		}
	case !ev.Handshake.Signaled:
		v.Status = StatusFailed
		v.Message = "Iframe bridge never signaled ready. See: docs/phases/structure/ui-smoke.md#handshake-timeout"
	case len(ev.Network) > 0:
		v.Status = StatusFailed
		v.Message = formatNetworkFailures(ev.Network)
	case len(ev.PageErrors) > 0:
		v.Status = StatusFailed
		v.Message = fmt.Sprintf("UI exception: %s", ev.PageErrors[0].Message)
	}

	return v
}

// formatNetworkFailures renders a human-readable summary of network failures,
// capping the enumeration so the message stays readable.
func formatNetworkFailures(failures []NetworkEntry) string {
	if len(failures) == 0 {
		return ""
	}
	if len(failures) == 1 {
		return formatSingleNetworkFailure(failures[0])
	}

	var messages []string
	for i, entry := range failures {
		if i >= 5 {
			messages = append(messages, fmt.Sprintf("... and %d more", len(failures)-5))
			break
		}
		messages = append(messages, formatSingleNetworkFailure(entry))
	}
	return fmt.Sprintf("Network failures (%d total): %s", len(failures), strings.Join(messages, "; "))
}

// formatSingleNetworkFailure formats one network failure entry.
func formatSingleNetworkFailure(entry NetworkEntry) string {
	switch {
	case entry.Status != nil:
		return fmt.Sprintf("HTTP %d → %s", *entry.Status, entry.URL)
	case entry.ErrorText != "":
		return fmt.Sprintf("%s → %s", entry.ErrorText, entry.URL)
	default:
		return fmt.Sprintf("Request error → %s", entry.URL)
	}
}
