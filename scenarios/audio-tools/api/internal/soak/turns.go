//go:build audio_lab

package soak

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"runtime"
	"strings"

	"audio-tools/internal/conformance"
	"audio-tools/internal/stt/session"
)

type turnSamples struct {
	diagnostics    []diagnostic
	timing         []*diagnostic
	interimCount   int
	interimVisible int
}

// runTurns owns the ordered browser turn sequence and its heap observations.
func runTurns(ctx context.Context, d *driver, start sessionStart, ledgers *session.Registry, opt Options, runErr error) ([]turnObservation, int64, error) {
	observations := make([]turnObservation, 0, opt.Turns)
	heapBefore := stableHeapAlloc()
	maxHeapDelta := int64(0)
	for turn := 0; runErr == nil && turn < opt.Turns; turn++ {
		observation, observed, err := runTurn(ctx, d, start, ledgers, opt, turn)
		if err != nil {
			runErr = err
			break
		}
		if observed {
			observations = append(observations, observation)
			heapDelta := int64(stableHeapAlloc()) - int64(heapBefore)
			if heapDelta > maxHeapDelta {
				maxHeapDelta = heapDelta
			}
		}
	}
	// Include the final boundary even when the last turn exits through a
	// recoverable capture failure after its browser observation was recorded.
	finalHeapDelta := int64(stableHeapAlloc()) - int64(heapBefore)
	if finalHeapDelta > maxHeapDelta {
		maxHeapDelta = finalHeapDelta
	}
	return observations, maxHeapDelta, runErr
}

func runTurn(ctx context.Context, d *driver, start sessionStart, ledgers *session.Registry, opt Options, turn int) (turnObservation, bool, error) {
	base := turn * turnStepStride
	feedChunks := feedChunkCount(opt.FeedMS)
	samples, err := captureTurn(ctx, d, start.SessionID, opt, base, feedChunks)
	if err != nil {
		return turnObservation{}, false, err
	}
	tailBase := base + 5 + feedChunks
	observation, err := finishTurn(ctx, d, start, ledgers, opt, tailBase, samples)
	if err != nil {
		return turnObservation{}, false, err
	}
	return observation, true, nil
}

func captureTurn(ctx context.Context, d *driver, sessionID string, opt Options, base, feedChunks int) (turnSamples, error) {
	samples := turnSamples{
		diagnostics: make([]diagnostic, 0, feedChunks),
		timing:      make([]*diagnostic, 0, feedChunks),
	}
	if opt.Lane == conformance.LaneRealtime && strings.TrimSpace(opt.Fault) == "" {
		// Arm playback immediately before the gesture to avoid leading silence.
		if err := d.postBroker(ctx, "/browser/sessions/"+url.PathEscape(sessionID)+"/audio/restart", map[string]any{}, nil); err != nil {
			return turnSamples{}, err
		}
	}
	steps := []struct {
		index  int
		name   string
		action map[string]any
	}{
		{base + 2, "start", map[string]any{"click": map[string]any{"selector": d.productSelector("ready")}}},
		{base + 3, "recording", map[string]any{"wait": map[string]any{"selector": d.productSelector("recording"), "state": "WAIT_STATE_ATTACHED", "timeout_ms": 15000}}},
	}
	for _, step := range steps {
		if err := d.step(ctx, sessionID, step.index, step.name, step.action); err != nil {
			return turnSamples{}, err
		}
	}
	if processed := d.productSelector("processed"); processed != "" {
		if err := d.step(ctx, sessionID, base+4, "clear-processed", map[string]any{"wait": map[string]any{"selector": processed, "state": "WAIT_STATE_DETACHED", "timeout_ms": 2000}}); err != nil {
			return turnSamples{}, err
		}
	}
	if err := d.feed(ctx, sessionID, base+5, feedChunks, opt.FeedMS, func(observed *diagnostic, interimVisible bool) {
		samples.timing = append(samples.timing, observed)
		if observed != nil {
			samples.diagnostics = append(samples.diagnostics, *observed)
		}
		samples.interimCount++
		if interimVisible {
			samples.interimVisible++
		}
	}); err != nil {
		return turnSamples{}, err
	}
	if opt.Lane == conformance.LaneRealtime && strings.TrimSpace(opt.Fault) == "" {
		if err := d.postBroker(ctx, "/browser/sessions/"+url.PathEscape(sessionID)+"/audio/stop", map[string]any{}, nil); err != nil {
			return turnSamples{}, err
		}
	}
	return samples, nil
}

func finishTurn(ctx context.Context, d *driver, start sessionStart, ledgers *session.Registry, opt Options, tailBase int, samples turnSamples) (turnObservation, error) {
	serverSnapshot, serverSeen := initialServerSnapshot(ctx, d, start.SessionID, ledgers, tailBase)
	if err := stopAndWaitTurn(ctx, d, start.SessionID, opt, tailBase); err != nil {
		return turnObservation{}, err
	}
	if err := d.step(ctx, start.SessionID, tailBase+4, "settle", map[string]any{"wait": map[string]any{"duration_ms": 500}}); err != nil {
		return turnObservation{}, err
	}
	var page pageObservation
	if err := d.evaluate(ctx, start.SessionID, tailBase+5, &page); err != nil {
		return turnObservation{}, err
	}
	observed := decodeDiagnostic(page.Diagnostic)
	serverSnapshot, serverSeen = settledServerSnapshot(ledgers, observed, serverSnapshot, serverSeen)
	if opt.MobileRecovery {
		if err := verifyMobileRecovery(ctx, d, start.SessionID, tailBase); err != nil {
			return turnObservation{}, err
		}
		d.mobileRecoveryPassed = true
	}
	observation := turnObservation{
		Transcript: page.Transcript, Server: serverSnapshot, ServerSeen: serverSeen,
		Samples: samples.diagnostics, TimingSamples: samples.timing,
		InterimSampleCount: samples.interimCount, InterimVisibleSamples: samples.interimVisible,
		Diagnostic: observed,
	}
	if !observation.ServerSeen && observation.Diagnostic != nil && ledgers != nil {
		observation.Server, observation.ServerSeen = ledgers.ConformanceSnapshot(observation.Diagnostic.SessionID)
	}
	return observation, nil
}

func initialServerSnapshot(ctx context.Context, d *driver, sessionID string, ledgers *session.Registry, tailBase int) (session.Snapshot, bool) {
	var snapshot session.Snapshot
	var seen bool
	var page pageObservation
	if err := d.evaluate(ctx, sessionID, tailBase+1000, &page); err == nil {
		if observed := decodeDiagnostic(page.Diagnostic); observed != nil && ledgers != nil {
			snapshot, seen = ledgers.ConformanceSnapshot(observed.SessionID)
		}
	}
	return snapshot, seen
}

func stopAndWaitTurn(ctx context.Context, d *driver, sessionID string, opt Options, tailBase int) error {
	if err := stopTurnIfNeeded(ctx, d, sessionID, opt, tailBase); err != nil {
		return err
	}
	if strings.TrimSpace(opt.Fault) != "" {
		err := d.step(ctx, sessionID, tailBase+2, "fault-terminal", map[string]any{"wait": map[string]any{"selector": d.productSelector("terminal"), "state": "WAIT_STATE_ATTACHED", "timeout_ms": 60000}})
		return attachBrowserDiagnosticOnError(ctx, d, sessionID, tailBase+6, err)
	}
	if processed := d.productSelector("processed"); processed != "" {
		processedTimeoutMS := 15000
		if opt.Lane == conformance.LaneAccelerated {
			processedTimeoutMS = 60000
		}
		err := d.step(ctx, sessionID, tailBase+2, "processed", map[string]any{"wait": map[string]any{"selector": processed, "state": "WAIT_STATE_ATTACHED", "timeout_ms": processedTimeoutMS}})
		if err = attachBrowserDiagnosticOnError(ctx, d, sessionID, tailBase+6, err); err != nil {
			return err
		}
	}
	if strings.TrimSpace(opt.Fault) == "" {
		if err := d.step(ctx, sessionID, tailBase+3, "captured", map[string]any{"wait": map[string]any{"selector": d.productSelector("terminal"), "state": "WAIT_STATE_ATTACHED", "timeout_ms": 60000}}); err != nil {
			return err
		}
	}
	return nil
}

func stopTurnIfNeeded(ctx context.Context, d *driver, sessionID string, opt Options, tailBase int) error {
	shouldStop := true
	if strings.TrimSpace(opt.Fault) != "" {
		var page pageObservation
		if err := d.evaluate(ctx, sessionID, tailBase+1, &page); err == nil {
			if raw, ok := page.Diagnostic.(map[string]any); ok {
				if state, _ := raw["state"].(string); state == "completed" || state == "failed" || state == "cancelled" {
					shouldStop = false
				}
			}
		}
	}
	if !shouldStop {
		return nil
	}
	return d.step(ctx, sessionID, tailBase, "stop", map[string]any{"click": map[string]any{"selector": d.productSelector("ready")}})
}

func attachBrowserDiagnosticOnError(ctx context.Context, d *driver, sessionID string, index int, err error) error {
	if err == nil {
		return nil
	}
	var page pageObservation
	if evalErr := d.evaluate(ctx, sessionID, index, &page); evalErr == nil && page.Diagnostic != nil {
		if diagnosticJSON, marshalErr := json.Marshal(page.Diagnostic); marshalErr == nil {
			return fmt.Errorf("%w; browser_diagnostic=%s", err, string(diagnosticJSON))
		}
	}
	return err
}

func settledServerSnapshot(ledgers *session.Registry, observed *diagnostic, snapshot session.Snapshot, seen bool) (session.Snapshot, bool) {
	if observed == nil || ledgers == nil {
		return snapshot, seen
	}
	if settled, ok := ledgers.ConformanceSnapshot(observed.SessionID); ok && (!seen || len(settled.Committed) >= len(snapshot.Committed)) {
		return settled, true
	}
	return snapshot, seen
}

func verifyMobileRecovery(ctx context.Context, d *driver, sessionID string, tailBase int) error {
	steps := []struct {
		index    int
		name     string
		selector string
	}{
		{tailBase + 6, "open-mobile-coverage", "[data-testid=\"dictation-turn-details\"] summary"},
		{tailBase + 7, "export-mobile-diagnostic", "[data-testid=\"dictation-export-diagnostic\"]"},
	}
	for _, step := range steps {
		if err := d.step(ctx, sessionID, step.index, step.name, map[string]any{"click": map[string]any{"selector": step.selector}}); err != nil {
			return err
		}
	}
	var page pageObservation
	if err := d.evaluate(ctx, sessionID, tailBase+8, &page); err != nil {
		return err
	}
	if !page.CoverageVisible || !page.ExportAvailable || !page.DiagnosticMetadataOnly {
		return fmt.Errorf("mobile recovery evidence incomplete: coverage_visible=%t export_available=%t metadata_only=%t", page.CoverageVisible, page.ExportAvailable, page.DiagnosticMetadataOnly)
	}
	return nil
}

// stableHeapAlloc measures live Go heap rather than allocator high-water mark.
// Qualification runs can process an hour of audio and allocate many short-lived
// JSON/PCM wrappers; forcing collection at an observation boundary prevents
// those transient allocations from being reported as retained memory. This is
// called outside an active stream turn, so it does not perturb capture timing.
func stableHeapAlloc() uint64 {
	runtime.GC()
	var stats runtime.MemStats
	runtime.ReadMemStats(&stats)
	return stats.HeapAlloc
}

func feedChunkCount(feedMS int) int {
	if feedMS <= 0 {
		return 1
	}
	return (feedMS + feedChunkMS - 1) / feedChunkMS
}

// feed keeps each browser wait request below the driver/proxy timeout horizon.
// The recording button is not touched between chunks, so the browser product
// still observes one continuous turn and the WebSocket remains open throughout.
func (d *driver) feed(ctx context.Context, sessionID string, index, chunks, totalMS int, sample func(*diagnostic, bool)) error {
	remaining := totalMS
	for chunk := 0; chunk < chunks; chunk++ {
		duration := feedChunkMS
		if remaining < duration {
			duration = remaining
		}
		if duration <= 0 {
			break
		}
		if err := d.step(ctx, sessionID, index+chunk, fmt.Sprintf("feed-%d", chunk), map[string]any{"wait": map[string]any{"duration_ms": duration}}); err != nil {
			return err
		}
		if d.opt.Lane == conformance.LaneRealtime && strings.TrimSpace(d.opt.Fault) == "" && sample != nil {
			var page pageObservation
			if err := d.evaluate(ctx, sessionID, index+1000+chunk, &page); err != nil {
				return err
			}
			sample(decodeDiagnostic(page.Diagnostic), strings.TrimSpace(page.Interim) != "")
		}
		remaining -= duration
	}
	return nil
}

func decodeDiagnostic(raw any) *diagnostic {
	if raw == nil {
		return nil
	}
	data, err := json.Marshal(raw)
	if err != nil {
		return nil
	}
	var out diagnostic
	if err := json.Unmarshal(data, &out); err != nil {
		return nil
	}
	return &out
}

// requestTimeout covers the declared wall-clock capture budget plus time for
// browser startup, final processing, and teardown. The previous fixed
// ten-minute timeout made a valid fifteen-minute lane impossible to observe
// and would have silently invalidated the sixty-minute qualification lane.
