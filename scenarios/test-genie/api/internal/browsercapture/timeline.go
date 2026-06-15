package browsercapture

import (
	"strings"

	"test-genie/internal/evidence"
	"test-genie/internal/playbooks/execution"
)

// timelineToEvidence maps a BAS execution timeline into engine-agnostic
// evidence. The mapping honors the historical smoke contract:
//
//   - handshake: derived from the handshake assert step (passed → signaled;
//     failed/absent → timed out);
//   - network failures + console: extracted by the shared
//     execution.NetworkFailures / execution.ConsoleEntries helpers so smoke and
//     playbooks read the timeline identically (one home, no duplication);
//   - screenshot: the frame screenshot reference from the screenshot step.
//
// Note (fidelity): BAS forwards uncaught page exceptions to the console as
// error-level entries rather than a separate page-error stream, so PageErrors is
// not separately populated here — such exceptions are counted as console errors
// (non-fatal per the contract). The handshake assert remains the hard-fail gate
// that catches a UI broken badly enough not to boot.
func timelineToEvidence(req Request, tl parsedTimeline) evidence.Evidence {
	ev := evidence.Evidence{
		URL:    req.ScenarioURL,
		Label:  req.Label,
		Loaded: true,
	}

	ev.Handshake = handshakeFromTimeline(tl)
	ev.Console = execution.ConsoleEntries(tl)
	ev.Network = execution.NetworkFailures(tl)
	ev.ScreenshotRef = screenshotRef(tl)

	return ev
}

// handshakeFromTimeline reads the handshake outcome from the handshake assert
// frame. A passing assert means the readiness marker appeared (signaled). A
// failing/absent assert means it never did (timed out).
func handshakeFromTimeline(tl parsedTimeline) evidence.Handshake {
	if tl == nil {
		return evidence.Handshake{Signaled: false, TimedOut: true, Error: "no timeline"}
	}
	for _, fr := range tl.Frames {
		if fr.NodeID != nodeHandshake {
			continue
		}
		if fr.Assertion != nil && fr.Assertion.Passed {
			return evidence.Handshake{Signaled: true, DurationMs: int64(fr.DurationMs)}
		}
		msg := ""
		if fr.Assertion != nil {
			msg = firstNonEmpty(fr.Assertion.Error, fr.Assertion.Message)
		}
		if msg == "" {
			msg = fr.Error
		}
		return evidence.Handshake{Signaled: false, TimedOut: true, DurationMs: int64(fr.DurationMs), Error: msg}
	}
	// The handshake step never ran (an earlier step failed). Treat as not
	// signaled so the verdict fails closed.
	return evidence.Handshake{Signaled: false, TimedOut: true, Error: "handshake step did not execute"}
}

// screenshotRef returns the frame screenshot reference from the screenshot step.
func screenshotRef(tl parsedTimeline) string {
	if tl == nil {
		return ""
	}
	for _, fr := range tl.Frames {
		if fr.NodeID != nodeScreens || fr.Screenshot == nil {
			continue
		}
		return firstNonEmpty(fr.Screenshot.URL, fr.Screenshot.ArtifactID)
	}
	return ""
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
