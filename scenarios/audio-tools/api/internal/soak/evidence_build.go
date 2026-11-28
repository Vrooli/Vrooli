//go:build audio_lab

package soak

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"os"
	"strings"
	"time"

	"audio-tools/internal/buildidentity"
	"audio-tools/internal/conformance"
	"audio-tools/internal/stt/session"
)

func buildRun(opt Options, runID string, started time.Time, observations []turnObservation, heapDelta int64) conformance.Run {
	run := conformance.Run{SchemaVersion: conformance.SchemaVersion, RunID: runID, Lane: opt.Lane, Profile: opt.Profile, Shape: evidenceShape(opt), SimulatedMinutes: opt.SimulatedMinutes, Turns: opt.Turns, Fault: opt.Fault, Cell: conformance.Cell{EngineID: opt.EngineID, ModelID: opt.ModelID, Strategy: opt.Strategy, Policy: opt.Policy}, Code: conformance.Code{CapturePackage: captureFingerprint(), Server: "sha256:" + buildidentity.SourceIdentity}, DurationMs: time.Since(started).Milliseconds()}
	run.Cell.ProviderTier = observedProviderTier(observations)
	if strings.TrimSpace(opt.Fault) != "" {
		return buildFaultRun(opt, runID, started, observations, heapDelta)
	}
	if run.Profile == "" {
		run.Profile = "realistic"
	}
	if run.Cell.Policy == "" {
		run.Cell.Policy = "default"
	}
	if run.Cell.Strategy == "" {
		run.Cell.Strategy = "product"
	}
	for _, name := range conformance.InvariantAssertions {
		run.Assertions = append(run.Assertions, invariant(name, observations, heapDelta, opt.FeedMS, opt.Lane, opt.EngineID, opt.ModelID))
	}
	if opt.Lane == conformance.LaneAccelerated {
		run.Assertions = append(run.Assertions, acceleratedDurationAssertion(observations, opt), conformance.Measured("accelerated_wall_budget", run.DurationMs <= 60_000, fmt.Sprintf("wall duration %dms; required <=60000ms", run.DurationMs)))
	}
	if opt.Lane == conformance.LaneRealtime {
		if len(observations) == 0 || observations[0].Diagnostic == nil {
			run.Assertions = append(run.Assertions,
				conformance.NotMeasured("capture_ready_latency", "the browser did not expose a completed stream diagnostic"),
				conformance.NotMeasured("first_partial_latency_stable", "the browser did not expose a completed stream diagnostic"),
				conformance.NotMeasured("committed_text_lag_stable", "the browser did not expose a completed stream diagnostic"),
			)
		} else {
			timing := realtimeTimingObservations(observations)
			run.Assertions = append(run.Assertions,
				realtimeAssertion("capture_ready_latency", observations, func(d *diagnostic) *float64 { return d.StartToCaptureReadyMS }),
				realtimeAssertion("first_partial_latency_stable", timing, func(d *diagnostic) *float64 { return d.FirstPartialLatencyMS }),
				realtimeAssertion("committed_text_lag_stable", timing, func(d *diagnostic) *float64 { return d.CommittedTextLagMS }),
			)
		}
		run.Assertions = append(run.Assertions, continuousInterimTextAssertion(observations))
		run.Assertions = append(run.Assertions, visibleUpdateGapAssertion(observations))
		run.Assertions = append(run.Assertions, stopToFinalTailAssertion(observations))
	}
	if strings.TrimSpace(opt.Reference) == "" {
		run.Assertions = append(run.Assertions, conformance.NotMeasured("word_error_rate_stable", "reference_text was not supplied; the driver will not invent a transcript reference"), conformance.NotMeasured("punctuation_rate_recorded", "reference_text was not supplied"), conformance.NotMeasured("capitalisation_rate_recorded", "reference_text was not supplied"))
	} else {
		// The transcript is intentionally held only in memory during this
		// function; quality details contain rates, never transcript text.
		run.Assertions = append(run.Assertions, qualityAcrossTurns(opt.Reference, observations, opt.Lane, opt.fixtureSamples, opt.ReferenceCycleMS, opt.FeedMS, realtimePlaybackPauseMS(opt))...)
	}
	return run
}

// visibleUpdateGapAssertion requires a complete rendered-revision metric for
// every voiced turn. Missing/null arrays, inconsistent revision counts, and
// non-finite/negative values fail closed instead of silently dropping samples.
func visibleUpdateGapAssertion(observations []turnObservation) conformance.Assertion {
	const name = "visible_update_gap_within_budget"
	if len(observations) == 0 {
		return conformance.NotMeasured(name, "no realtime turn observations were available")
	}
	values := make([]float64, 0)
	for index, observation := range observations {
		d := observation.Diagnostic
		if d == nil || d.VisibleUpdateGapsMS == nil {
			return conformance.NotMeasured(name, fmt.Sprintf("turn %d omitted visibleUpdateGapsMs", index+1))
		}
		if d.VisibleRevisionCount < 2 || len(d.VisibleUpdateGapsMS) != d.VisibleRevisionCount-1 {
			return conformance.NotMeasured(name, fmt.Sprintf("turn %d has incomplete visible-update evidence: revisions=%d gaps=%d", index+1, d.VisibleRevisionCount, len(d.VisibleUpdateGapsMS)))
		}
		for _, gap := range d.VisibleUpdateGapsMS {
			if math.IsNaN(gap) || math.IsInf(gap, 0) || gap < 0 {
				return conformance.NotMeasured(name, fmt.Sprintf("turn %d has malformed visible-update gap", index+1))
			}
			values = append(values, gap)
		}
	}
	p95 := percentile(values, 0.95)
	maximum := values[0]
	for _, gap := range values[1:] {
		if gap > maximum {
			maximum = gap
		}
	}
	return conformance.Measured(name, p95 <= 750 && maximum <= 2_000, fmt.Sprintf("rendered transcript revision gaps across %d turns: p95=%.2fms (<=750ms), max=%.2fms (<=2000ms), samples=%d", len(observations), p95, maximum, len(values)))
}

func buildFaultRun(opt Options, runID string, started time.Time, observations []turnObservation, heapDelta int64) conformance.Run {
	run := conformance.Run{SchemaVersion: conformance.SchemaVersion, RunID: runID, Lane: opt.Lane, Profile: opt.Profile, Shape: evidenceShape(opt), SimulatedMinutes: opt.SimulatedMinutes, Turns: opt.Turns, Fault: opt.Fault, Cell: conformance.Cell{EngineID: opt.EngineID, ModelID: opt.ModelID, Strategy: opt.Strategy, Policy: opt.Policy}, Code: conformance.Code{CapturePackage: captureFingerprint(), Server: "sha256:" + buildidentity.SourceIdentity}, DurationMs: time.Since(started).Milliseconds()}
	run.Cell.ProviderTier = observedProviderTier(observations)
	run.Assertions = append(run.Assertions, conformance.Measured("fault_profile_observed", strings.TrimSpace(opt.Fault) != "" && len(observations) > 0, fmt.Sprintf("fault=%q; observed browser turns=%d", opt.Fault, len(observations))))
	intervals, duplicates := faultIntervalEvidence(observations)
	run.Assertions = append(run.Assertions, conformance.Measured("fault_interval_accounting", intervals && len(observations) > 0, "captured and sent interval sequences remained ordered; terminal faults may stop before processing acknowledgement"))
	run.Assertions = append(run.Assertions, conformance.Measured("fault_no_duplicate_committed_segments", duplicates, "committed segment identifiers remained unique across the injected fault and any recovery"))
	terminal, details := faultTerminalEvidence(observations)
	run.Assertions = append(run.Assertions, conformance.Measured("fault_recovered_or_terminal", terminal && len(observations) > 0, strings.Join(details, "; ")))
	_ = heapDelta
	return run
}

// observedProviderTier returns a tier only when every browser turn reports the
// same recognized owner-emitted tier. Missing or conflicting observations stay
// blank and cannot be converted into route qualification.
func observedProviderTier(observations []turnObservation) string {
	var observed string
	if len(observations) == 0 {
		return ""
	}
	for _, observation := range observations {
		if observation.Diagnostic == nil {
			return ""
		}
		tier := observation.Diagnostic.ProviderTier
		if tier != "local" && tier != "byok" && tier != "vrooli" {
			return ""
		}
		if observed != "" && observed != tier {
			return ""
		}
		observed = tier
	}
	return observed
}

func faultIntervalEvidence(observations []turnObservation) (bool, bool) {
	intervals, duplicates := true, true
	seenSegments := map[string]struct{}{}
	for _, observation := range observations {
		diagnostic := observation.Diagnostic
		if diagnostic == nil || diagnostic.CapturedSequence < 0 || diagnostic.SentSequence < -1 || diagnostic.ProcessedSequence < -1 || (diagnostic.SentSequence >= 0 && diagnostic.SentSequence > diagnostic.CapturedSequence) {
			intervals = false
		}
		for _, segment := range observation.Server.Committed {
			if _, seen := seenSegments[segment.ID]; seen {
				duplicates = false
			}
			seenSegments[segment.ID] = struct{}{}
		}
	}
	return intervals, duplicates
}

func faultTerminalEvidence(observations []turnObservation) (bool, []string) {
	terminal := true
	details := make([]string, 0, len(observations))
	for _, observation := range observations {
		diagnostic := observation.Diagnostic
		if diagnostic == nil {
			terminal = false
			continue
		}
		if diagnostic.State != "completed" && diagnostic.State != "failed" && diagnostic.State != "cancelled" && diagnostic.TerminalReason == "" {
			terminal = false
		}
		details = append(details, fmt.Sprintf("state=%s terminal=%s captured=%d processed=%d", diagnostic.State, diagnostic.TerminalReason, diagnostic.CapturedSequence, diagnostic.ProcessedSequence))
	}
	return terminal, details
}

func continuousInterimTextAssertion(observations []turnObservation) conformance.Assertion {
	if len(observations) == 0 {
		return conformance.NotMeasured("continuous_interim_text_visible", "no completed browser turn was observed")
	}
	for index, observation := range observations {
		if observation.InterimSampleCount == 0 {
			return conformance.NotMeasured("continuous_interim_text_visible", fmt.Sprintf("turn %d exposed no interim-text visibility samples", index+1))
		}
		if observation.InterimVisibleSamples != observation.InterimSampleCount {
			return conformance.Measured("continuous_interim_text_visible", false, fmt.Sprintf("turn %d kept interim text visible at %d/%d feed checkpoints", index+1, observation.InterimVisibleSamples, observation.InterimSampleCount))
		}
	}
	return conformance.Measured("continuous_interim_text_visible", true, fmt.Sprintf("interim text remained visible at %d/%d realtime feed checkpoints across %d turn(s)", totalInterimVisibleSamples(observations), totalInterimSampleCount(observations), len(observations)))
}

func totalInterimSampleCount(observations []turnObservation) int {
	total := 0
	for _, observation := range observations {
		total += observation.InterimSampleCount
	}
	return total
}

func totalInterimVisibleSamples(observations []turnObservation) int {
	total := 0
	for _, observation := range observations {
		total += observation.InterimVisibleSamples
	}
	return total
}

func qualityAcrossTurns(reference string, observations []turnObservation, lane conformance.Lane, fixtureSamples int64, referenceCycleMS int, feedMS int, playbackPauseMS int) []conformance.Assertion {
	const names = 3
	passed := [names]bool{true, true, true}
	details := [names][]string{}
	for index, observation := range observations {
		measured := measureTurnQuality(reference, observation, lane, fixtureSamples, referenceCycleMS, feedMS, playbackPauseMS)
		for metric, assertion := range measured {
			if metric == 0 && observation.ServerSeen {
				assertion.Detail = fmt.Sprintf("%s server_received_sequence=%d server_processed_sequence=%d committed_segments=%d", assertion.Detail, observation.Server.ReceivedSequence, observation.Server.ProcessedSequence, len(observation.Server.Committed))
			}
			passed[metric] = passed[metric] && assertion.Outcome == conformance.OutcomePassed
			details[metric] = append(details[metric], fmt.Sprintf("turn %d: %s", index+1, assertion.Detail))
		}
	}
	if len(observations) == 0 {
		return []conformance.Assertion{
			conformance.NotMeasured("word_error_rate_stable", "no completed browser turn was observed"),
			conformance.NotMeasured("punctuation_rate_recorded", "no completed browser turn was observed"),
			conformance.NotMeasured("capitalisation_rate_recorded", "no completed browser turn was observed"),
		}
	}
	return []conformance.Assertion{
		conformance.Measured("word_error_rate_stable", passed[0], strings.Join(details[0], "; ")),
		conformance.Measured("punctuation_rate_recorded", passed[1], strings.Join(details[1], "; ")),
		conformance.Measured("capitalisation_rate_recorded", passed[2], strings.Join(details[2], "; ")),
	}
}

func measureTurnQuality(reference string, observation turnObservation, lane conformance.Lane, fixtureSamples int64, referenceCycleMS int, feedMS int, playbackPauseMS int) []conformance.Assertion {
	turnReference := reference
	cycleSamples := fixtureSamples
	if referenceCycleMS > 0 {
		cycleSamples = int64(referenceCycleMS) * canonicalSampleRate / 1000
	}
	if lane == conformance.LaneRealtime && playbackPauseMS > 0 {
		// Playback silence consumes the feed window but contributes no words.
		cycleSamples += int64(playbackPauseMS) * canonicalSampleRate / 1000
	}
	if cycleSamples > 0 && observation.Diagnostic != nil && observation.Diagnostic.CapturedSamples > 0 {
		// Recorder setup/drain are not speech; use source playback duration.
		referenceSamples := observation.Diagnostic.CapturedSamples
		if lane == conformance.LaneRealtime && feedMS > 0 {
			referenceSamples = int64(feedMS) * canonicalSampleRate / 1000
		}
		repetitions := int(math.Round(float64(referenceSamples) / float64(cycleSamples)))
		if repetitions < 1 {
			repetitions = 1
		}
		turnReference = repeatReference(reference, repetitions)
	}
	hypothesis := observation.Transcript
	if committed := committedTranscript(observation.Server.Committed); committed != "" {
		// The server ledger is the owner-backed complete-turn settlement source.
		hypothesis = committed
	}
	return conformance.MeasureQuality(conformance.QualityObservation{Reference: turnReference, Hypothesis: hypothesis, MaxWER: 0.10, MinPunctuationRate: 0, MinCapitalisationRate: 0})
}

func committedTranscript(segments []session.Segment) string {
	parts := make([]string, 0, len(segments))
	for _, segment := range segments {
		if text := strings.TrimSpace(segment.Text); text != "" {
			parts = append(parts, text)
		}
	}
	return strings.TrimSpace(strings.Join(parts, " "))
}

func repeatReference(reference string, repetitions int) string {
	reference = strings.TrimSpace(reference)
	if repetitions <= 1 || reference == "" {
		return reference
	}
	return strings.TrimSpace(strings.Repeat(reference+" ", repetitions))
}

// wavSampleCount reads only the RIFF chunks needed to determine how many PCM
// samples one BAS fixture cycle contains. Invalid/non-WAV test fixtures return
// an error and leave the quality oracle's one-cycle behavior unchanged.
func wavSampleCount(path string) (int64, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	if len(data) < 12 || string(data[:4]) != "RIFF" || string(data[8:12]) != "WAVE" {
		return 0, errors.New("fixture is not a RIFF/WAVE file")
	}
	var blockAlign uint16
	for offset := 12; offset+8 <= len(data); {
		chunkID := string(data[offset : offset+4])
		chunkSize := int(binary.LittleEndian.Uint32(data[offset+4 : offset+8]))
		offset += 8
		if chunkSize < 0 || offset+chunkSize > len(data) {
			return 0, errors.New("fixture contains an invalid RIFF chunk")
		}
		switch chunkID {
		case "fmt ":
			if chunkSize < 14 {
				return 0, errors.New("fixture fmt chunk is too short")
			}
			blockAlign = binary.LittleEndian.Uint16(data[offset+12 : offset+14])
		case "data":
			if blockAlign == 0 {
				return 0, errors.New("fixture has no valid PCM block alignment")
			}
			return int64(chunkSize) / int64(blockAlign), nil
		}
		offset += chunkSize
		if chunkSize%2 == 1 {
			offset++
		}
	}
	return 0, errors.New("fixture has no data chunk")
}
