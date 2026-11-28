//go:build audio_lab

package soak

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"audio-tools/internal/conformance"
)

func acceleratedDurationAssertion(observations []turnObservation, opt Options) conformance.Assertion {
	if opt.SimulatedMinutes <= 0 {
		return conformance.NotMeasured("accelerated_duration_target", "simulated_minutes was not supplied")
	}
	var captured int64
	for _, observation := range observations {
		if observation.Diagnostic == nil {
			return conformance.NotMeasured("accelerated_duration_target", "browser diagnostic was absent")
		}
		captured += observation.Diagnostic.CapturedSamples
	}
	target := int64(opt.SimulatedMinutes) * 60 * canonicalSampleRate
	return conformance.Measured("accelerated_duration_target", captured >= target, fmt.Sprintf("virtual corpus captured %d/%d samples (%.2f/%.2f simulated minutes)", captured, target, float64(captured)/canonicalSampleRate/60, float64(target)/canonicalSampleRate/60))
}

func signalTurnCount(observations []turnObservation) int {
	count := 0
	for _, observation := range observations {
		if observation.Diagnostic != nil && observation.Diagnostic.SignalObserved {
			count++
		}
	}
	return count
}

func allSignalObserved(observations []turnObservation) bool {
	return len(observations) > 0 && signalTurnCount(observations) == len(observations)
}

// cursorsUnobserved reports whether a turn's wire cursors never left the
// StreamDiagnosticRecorder's -1 sentinel, together with the reason to publish.
//
// The three cursors mark three stages: captured (audio reached the batcher),
// sent (a frame reached the socket), processed (the server acknowledged it).
// Sequence 0 is a real interval, so -1 — not 0 — is the "never happened"
// sentinel for each. The returned reason names the stage that was never
// reached and carries the turn's terminal state and error codes, because those
// are what actually explain it.
func cursorsUnobserved(d *diagnostic) (string, bool) {
	switch {
	case d.CapturedSequence < 0:
		return fmt.Sprintf("no wire interval was captured in the browser (%s)", describeTurnOutcome(d)), true
	case d.SentSequence < 0:
		return fmt.Sprintf("%d wire interval(s) were captured but none reached the socket, so sent/processed coverage was never observable (%s)", d.CapturedSequence+1, describeTurnOutcome(d)), true
	case d.ProcessedSequence < 0:
		return fmt.Sprintf("%d wire interval(s) were sent but the server acknowledged none, so processed coverage was never observable (%s)", d.SentSequence+1, describeTurnOutcome(d)), true
	}
	return "", false
}

// describeTurnOutcome renders the triage context an evidence reader needs to
// act on a red or unmeasured turn without re-running the soak.
func describeTurnOutcome(d *diagnostic) string {
	parts := []string{fmt.Sprintf("state=%s", nonEmpty(d.State, "unknown"))}
	if strings.TrimSpace(d.TerminalReason) != "" {
		parts = append(parts, fmt.Sprintf("terminal=%s", d.TerminalReason))
	}
	if len(d.ErrorCodes) > 0 {
		parts = append(parts, fmt.Sprintf("errors=[%s]", strings.Join(d.ErrorCodes, " ")))
	}
	if len(d.StatusCodes) > 0 {
		parts = append(parts, fmt.Sprintf("status=[%s]", strings.Join(d.StatusCodes, " ")))
	}
	return strings.Join(parts, " ")
}

func nonEmpty(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func invariant(name string, observations []turnObservation, heapDelta int64, feedMS int, lane conformance.Lane, expectedEngineID string, expectedModelID string) conformance.Assertion {
	if readiness, ready := invariantReadiness(name, observations); !ready {
		return readiness
	}
	switch name {
	case "interval_accounting_exactly_once":
		return intervalAccountingAssertion(name, observations)
	case "capture_signal_observed":
		return conformance.Measured(name, allSignalObserved(observations), fmt.Sprintf("non-silent PCM observed on %d/%d browser turns", signalTurnCount(observations), len(observations)))
	case "browser_retention_bounded":
		return browserRetentionAssertion(name, observations)
	case "server_retention_bounded":
		return serverRetentionAssertion(name, observations)
	case "capture_callback_processing_budget":
		return captureCallbackProcessingBudgetAssertion(name, observations)
	case "wire_rate_within_budget":
		return wireRateAssertion(name, observations, feedMS, lane)
	case "zero_duplicate_committed_segments":
		return duplicateSegmentsAssertion(name, observations)
	case "zero_silent_terminal_outcomes":
		return terminalOutcomeAssertion(name, observations)
	case "provider_cell_identity":
		return providerIdentityAssertion(name, observations, expectedEngineID, expectedModelID)
	case "peak_memory_flat":
		return conformance.Measured(name, heapDelta < maxLedgerBytes, fmt.Sprintf("audio-tools server heap delta=%d bytes across turns", heapDelta))
	default:
		return conformance.NotMeasured(name, "driver has no observation for this assertion")
	}
}

func invariantReadiness(name string, observations []turnObservation) (conformance.Assertion, bool) {
	if len(observations) == 0 {
		return conformance.NotMeasured(name, "no completed browser turn was observed"), false
	}
	for _, observation := range observations {
		if observation.Diagnostic == nil {
			return conformance.NotMeasured(name, "browser diagnostic was absent"), false
		}
		if !observation.ServerSeen && name == "server_retention_bounded" {
			return conformance.NotMeasured(name, "server ledger snapshot was unavailable for the browser session"), false
		}
	}
	return conformance.Assertion{}, true
}

func intervalAccountingAssertion(name string, observations []turnObservation) conformance.Assertion {
	// A wire cursor still holding its -1 sentinel means the browser never
	// reached that stage — not that intervals went missing. Report what was
	// not observed rather than attributing it to accounting.
	for _, observation := range observations {
		if reason, unobserved := cursorsUnobserved(observation.Diagnostic); unobserved {
			return conformance.NotMeasured(name, reason)
		}
	}
	for _, observation := range observations {
		diagnostic := observation.Diagnostic
		if diagnostic.SentSequence < diagnostic.CapturedSequence-1 || diagnostic.ProcessedSequence < diagnostic.SentSequence {
			return conformance.Measured(name, false, fmt.Sprintf("captured=%d sent=%d processed=%d; expected zero-based sent/processed coverage for %d captured intervals (%s)", diagnostic.CapturedSequence, diagnostic.SentSequence, diagnostic.ProcessedSequence, diagnostic.CapturedSequence+1, describeTurnOutcome(diagnostic)))
		}
	}
	return conformance.Measured(name, true, fmt.Sprintf("%d browser turns observed with monotonic capture coverage; signal_observed=%t", len(observations), allSignalObserved(observations)))
}

func browserRetentionAssertion(name string, observations []turnObservation) conformance.Assertion {
	for _, observation := range observations {
		if observation.Diagnostic.RetainedBytes < 0 || observation.Diagnostic.RetainedBytes > maxLedgerBytes {
			return conformance.Measured(name, false, "browser retained-byte bound exceeded")
		}
	}
	return conformance.Measured(name, true, "metadata-only browser retained-byte snapshots stayed within 64MiB")
}

func serverRetentionAssertion(name string, observations []turnObservation) conformance.Assertion {
	for _, observation := range observations {
		if observation.Server.RetainedBytes < 0 || observation.Server.RetainedBytes > maxLedgerBytes {
			return conformance.Measured(name, false, "server retained-byte bound exceeded")
		}
	}
	return conformance.Measured(name, true, "server ledger retained-byte snapshots stayed within 64MiB")
}

func captureCallbackProcessingBudgetAssertion(name string, observations []turnObservation) conformance.Assertion {
	maxRatio := float64(0)
	for _, observation := range observations {
		d := observation.Diagnostic
		if d.CaptureCallbackCount <= 0 || d.CaptureCallbackMaxCostRatio == nil {
			return conformance.Measured(name, false, "capture callback processing-cost measurement missing")
		}
		if !isFinite(*d.CaptureCallbackMaxCostRatio) || *d.CaptureCallbackMaxCostRatio < 0 {
			return conformance.Measured(name, false, "capture callback processing-cost measurement invalid")
		}
		maxRatio = math.Max(maxRatio, *d.CaptureCallbackMaxCostRatio)
	}
	return conformance.Measured(name, maxRatio <= maxCaptureCallbackCostRatio, fmt.Sprintf("worst capture callback processing/audio duration ratio=%.3f across %d callbacks; limit=%.2f", maxRatio, totalCaptureCallbacks(observations), maxCaptureCallbackCostRatio))
}

func isFinite(value float64) bool { return !math.IsNaN(value) && !math.IsInf(value, 0) }

func totalCaptureCallbacks(observations []turnObservation) int {
	total := 0
	for _, observation := range observations {
		total += observation.Diagnostic.CaptureCallbackCount
	}
	return total
}

func wireRateAssertion(name string, observations []turnObservation, feedMS int, lane conformance.Lane) conformance.Assertion {
	if feedMS <= 0 {
		return conformance.NotMeasured(name, "feed duration was not available to calculate canonical wire rate")
	}
	minRate, maxRate := float64(0), float64(0)
	for _, observation := range observations {
		diagnostic := observation.Diagnostic
		seconds := float64(feedMS) / 1000
		if diagnostic.FirstCaptureAtMS != nil && diagnostic.LastCaptureAtMS != nil && *diagnostic.LastCaptureAtMS > *diagnostic.FirstCaptureAtMS {
			seconds = (*diagnostic.LastCaptureAtMS - *diagnostic.FirstCaptureAtMS) / 1000
		}
		if lane == conformance.LaneAccelerated {
			seconds = float64(diagnostic.CapturedSamples) / canonicalSampleRate
		}
		rate := float64(diagnostic.CapturedSamples) / seconds
		if rate <= 0 {
			return conformance.Measured(name, false, fmt.Sprintf("capturedSamples=%d produced a non-positive wire rate", diagnostic.CapturedSamples))
		}
		if minRate == 0 || rate < minRate {
			minRate = rate
		}
		if rate > maxRate {
			maxRate = rate
		}
	}
	lower, upper := canonicalSampleRate*minWireRateFraction, canonicalSampleRate*maxWireRateFraction
	return conformance.Measured(name, minRate >= lower && maxRate <= upper, fmt.Sprintf("captured wire rate %.0f..%.0f samples/s; expected %.0f..%.0f", minRate, maxRate, lower, upper))
}

func duplicateSegmentsAssertion(name string, observations []turnObservation) conformance.Assertion {
	for _, observation := range observations {
		ids := map[string]bool{}
		for _, segment := range observation.Server.Committed {
			if ids[segment.ID] {
				return conformance.Measured(name, false, "duplicate committed segment id observed")
			}
			ids[segment.ID] = true
		}
	}
	return conformance.Measured(name, true, "server ledger committed segment ids were unique")
}

func terminalOutcomeAssertion(name string, observations []turnObservation) conformance.Assertion {
	for _, observation := range observations {
		diagnostic := observation.Diagnostic
		if diagnostic.State != "completed" && diagnostic.State != "failed" && diagnostic.TerminalReason == "" {
			return conformance.Measured(name, false, fmt.Sprintf("state=%q terminalReason=%q", diagnostic.State, diagnostic.TerminalReason))
		}
	}
	return conformance.Measured(name, true, "every observed turn exposed a terminal state or reason")
}

func providerIdentityAssertion(name string, observations []turnObservation, expectedEngineID string, expectedModelID string) conformance.Assertion {
	for _, observation := range observations {
		diagnostic := observation.Diagnostic
		if diagnostic.ProviderID == "" || diagnostic.ModelID == "" {
			return conformance.NotMeasured(name, "the product path did not expose provider identity")
		}
		if diagnostic.ProviderID != expectedEngineID || diagnostic.ModelID != expectedModelID {
			return conformance.Measured(name, false, fmt.Sprintf("observed provider=%q model=%q; expected provider=%q model=%q", diagnostic.ProviderID, diagnostic.ModelID, expectedEngineID, expectedModelID))
		}
	}
	return conformance.Measured(name, true, fmt.Sprintf("provider=%q model=%q observed for %d turns", expectedEngineID, expectedModelID, len(observations)))
}

func realtimeTimingObservations(observations []turnObservation) []turnObservation {
	out := make([]turnObservation, 0, len(observations))
	for _, observation := range observations {
		if len(observation.TimingSamples) > 0 {
			// Stability is a property of the live stream. Do not compare the
			// first periodic sample with the terminal flush snapshot: manual
			// stop intentionally retains a short settle window so the final
			// captured audio is delivered. Terminal delivery is covered by
			// zero_silent_terminal_outcomes and exact interval accounting.
			for _, sample := range observation.TimingSamples {
				out = append(out, turnObservation{Diagnostic: sample})
			}
			// A non-segmenting provider may expose committed-tail lag only in
			// its terminal diagnostic. Use it when no periodic sample carries
			// that metric; retain the existing rule that genuine periodic
			// samples exclude the terminal flush from drift comparisons.
			if observation.Diagnostic != nil && observation.Diagnostic.CommittedTextLagMS != nil {
				hasPeriodicCommittedLag := false
				for _, sample := range observation.TimingSamples {
					if sample != nil && sample.CommittedTextLagMS != nil {
						hasPeriodicCommittedLag = true
						break
					}
				}
				if !hasPeriodicCommittedLag {
					out = append(out, turnObservation{Diagnostic: observation.Diagnostic})
				}
			}
			continue
		}
		if observation.Diagnostic != nil && observation.Diagnostic.CommittedTextLagMS != nil {
			out = append(out, turnObservation{Diagnostic: observation.Diagnostic})
			continue
		}
		for index := range observation.Samples {
			sample := observation.Samples[index]
			out = append(out, turnObservation{Diagnostic: &sample})
		}
	}
	return out
}

func realtimeAssertion(name string, observations []turnObservation, value func(*diagnostic) *float64) conformance.Assertion {
	values := make([]float64, 0, len(observations))
	measured := false
	for _, o := range observations {
		if o.Diagnostic == nil {
			if name == "committed_text_lag_stable" {
				// Commit lag is undefined before the first committed result in
				// each turn. Periodic snapshots therefore legitimately carry a
				// null value while capture is still being processed.
				continue
			}
			if measured {
				return conformance.NotMeasured(name, "the product path stopped exposing this timing after it was measured")
			}
			continue
		}
		v := value(o.Diagnostic)
		if v == nil {
			if name == "committed_text_lag_stable" {
				continue
			}
			if measured {
				return conformance.NotMeasured(name, "the product path stopped exposing this timing after it was measured")
			}
			continue
		}
		measured = true
		values = append(values, *v)
	}
	if len(values) == 0 {
		return conformance.NotMeasured(name, "no realtime observations were available")
	}
	first, last := values[0], values[len(values)-1]
	// Every switch arm below either assigns passed or returns.
	var passed bool
	bound := ""
	switch name {
	case "capture_ready_latency":
		p95 := percentile(values, 0.95)
		const targetMS = 500.0
		maximum := first
		for _, value := range values[1:] {
			if value > maximum {
				maximum = value
			}
		}
		if p95 > targetMS || maximum > 1_500 {
			return conformance.Measured(name, false, fmt.Sprintf("capture readiness exceeded approved bounds: p95=%.2fms target<=%.0fms max=%.2fms target<=1500ms across %d realtime observations", p95, targetMS, maximum, len(values)))
		}
		passed = true
		bound = fmt.Sprintf("p95 <= 500ms (%.2fms); max <= 1500ms (%.2fms)", p95, maximum)
	case "first_partial_latency_stable":
		p95 := percentile(values, 0.95)
		const targetMS = 1_000.0
		maximum := first
		for _, value := range values[1:] {
			if value > maximum {
				maximum = value
			}
		}
		if p95 > targetMS || maximum > 2_000 {
			decomposition := ""
			if len(observations) > 0 && observations[0].Diagnostic != nil {
				d := observations[0].Diagnostic
				decomposition = fmt.Sprintf("; first milestones capture_start=%s first_capture=%s first_signal=%s first_sent=%s first_partial=%s", formatMillis(d.CaptureStartedAtMS), formatMillis(d.FirstCaptureAtMS), formatMillis(d.FirstSignalAtMS), formatMillis(d.FirstSentAtMS), formatMillis(d.FirstPartialAtMS))
			}
			return conformance.Measured(name, false, fmt.Sprintf("%s exceeded approved speech-to-visible-partial bounds: p95=%.2fms target<=%.0fms max=%.2fms target<=2000ms across %d realtime observations; first=%.2fms last=%.2fms%s", name, p95, targetMS, maximum, len(values), first, last, decomposition))
		}
		limit := first * 1.2
		passed = last <= limit
		bound = fmt.Sprintf("p95 <= 1000ms (%.2fms); max <= 2000ms (%.2fms); last <= first*1.2 (%.2fms)", p95, maximum, limit)
	case "committed_text_lag_stable":
		// Committed lag is reported in millisecond-resolution snapshots. A
		// small increase can therefore be scheduler/measurement jitter rather
		// than retained queue growth. Keep the bound strict for real drift,
		// while allowing at most 5ms or 20% (whichever is larger) of bounded
		// periodic variation. Terminal flush latency is excluded before this
		// function, so this tolerance cannot hide an end-of-turn delivery gap.
		limit := first + math.Max(5, first*0.2)
		passed = last <= limit
		bound = fmt.Sprintf("last <= first+max(5ms,first*0.2) (%.2fms)", limit)
	default:
		return conformance.NotMeasured(name, "driver has no stability bound for this timing")
	}
	return conformance.Measured(name, passed, fmt.Sprintf("%s observed across %d realtime turns; first=%.2fms last=%.2fms; %s", name, len(values), first, last, bound))
}

// stopToFinalTailAssertion qualifies the browser's monotonic interval from
// the user's stop action through delivery of the final committed tail. Every
// realtime turn must provide a complete final event; a missing/null field is
// deliberately not converted into a pass (including batch fallback).
func stopToFinalTailAssertion(observations []turnObservation) conformance.Assertion {
	const name = "stop_to_final_tail_latency"
	if len(observations) == 0 {
		return conformance.NotMeasured(name, "no realtime turn observations were available")
	}
	values := make([]float64, 0, len(observations))
	for i, observation := range observations {
		d := observation.Diagnostic
		if d == nil {
			return conformance.NotMeasured(name, fmt.Sprintf("turn %d has no browser diagnostic", i+1))
		}
		if d.State != "completed" || d.TerminalReason != "final" || !d.DoneSent || d.ProcessedSequence < d.CapturedSequence {
			return conformance.NotMeasured(name, fmt.Sprintf("turn %d did not expose a complete final tail (state=%q terminal=%q doneSent=%t processed=%d captured=%d)", i+1, d.State, d.TerminalReason, d.DoneSent, d.ProcessedSequence, d.CapturedSequence))
		}
		if d.StopToFinalTailMS == nil {
			return conformance.NotMeasured(name, fmt.Sprintf("turn %d has no stop-to-final-tail event; fallback or missing browser telemetry cannot qualify", i+1))
		}
		if math.IsNaN(*d.StopToFinalTailMS) || math.IsInf(*d.StopToFinalTailMS, 0) || *d.StopToFinalTailMS < 0 {
			return conformance.Measured(name, false, fmt.Sprintf("turn %d reported invalid stop-to-final-tail duration %.2fms", i+1, *d.StopToFinalTailMS))
		}
		values = append(values, *d.StopToFinalTailMS)
	}
	p95 := percentile(values, 0.95)
	maximum := values[0]
	for _, value := range values[1:] {
		if value > maximum {
			maximum = value
		}
	}
	passed := p95 <= 1_000 && maximum <= 2_000
	return conformance.Measured(name, passed, fmt.Sprintf("stop-to-final-tail across %d complete realtime turns; p95=%.2fms target<=1000ms; max=%.2fms target<=2000ms", len(values), p95, maximum))
}

func formatMillis(value *float64) string {
	if value == nil {
		return "unobserved"
	}
	return fmt.Sprintf("%.0f", *value)
}

// percentile returns the nearest-rank percentile from a copy so the input
// order remains the turn order used by the stability check. The acceptance
// protocol uses p95 over all paced attempts, including failures handled by
// the caller as missing telemetry.
func percentile(values []float64, fraction float64) float64 {
	if len(values) == 0 {
		return 0
	}
	ordered := append([]float64(nil), values...)
	sort.Float64s(ordered)
	rank := int(math.Ceil(fraction*float64(len(ordered)))) - 1
	if rank < 0 {
		rank = 0
	}
	if rank >= len(ordered) {
		rank = len(ordered) - 1
	}
	return ordered[rank]
}

func captureFingerprint() string {
	dirs := []string{}
	if root := strings.TrimSpace(os.Getenv("VROOLI_ROOT")); root != "" {
		dirs = append(dirs, root)
	}
	if cwd, err := os.Getwd(); err == nil {
		for dir := cwd; dir != filepath.Dir(dir); dir = filepath.Dir(dir) {
			dirs = append(dirs, dir)
		}
	}
	for _, root := range dirs {
		packageDir := filepath.Join(root, "packages", "audio-capture-browser")
		paths := []string{filepath.Join(packageDir, "package.json"), filepath.Join(packageDir, "dist", "index.js")}
		hash := sha256.New()
		valid := true
		for _, path := range paths {
			data, err := os.ReadFile(path)
			if err != nil {
				valid = false
				break
			}
			_, _ = hash.Write([]byte(filepath.ToSlash(path)))
			_, _ = hash.Write(data)
		}
		if valid {
			return "sha256:" + hex.EncodeToString(hash.Sum(nil))
		}
	}
	return "unknown"
}
