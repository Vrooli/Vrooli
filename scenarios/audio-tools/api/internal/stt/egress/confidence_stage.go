package egress

import "context"

// ConfidenceStage is the signal-domain stage. It drops segments whose
// acoustic confidence indicates the audio was silence/noise rather than
// speech, mirroring faster-whisper's own no-speech rule: a segment is
// suppressed when its mean no_speech_prob exceeds NoSpeechThreshold AND its
// mean avg_logprob falls below LogProbThreshold (both conditions, so a
// confidently-decoded segment that happens to contain a pause is kept).
//
// The stage is a graceful no-op when the segment carries no confidence
// signals (Confidence == nil) — e.g. a native-streaming engine whose
// manifest declares an empty confidenceSignals set. The Segmenter only adds
// this stage for engines that provide the signals, but the nil guard keeps
// the stage safe if it is ever wired unconditionally.
type ConfidenceStage struct {
	NoSpeechThreshold float64
	LogProbThreshold  float64
}

// Name identifies the stage in Gate.Stages().
func (s ConfidenceStage) Name() string { return "confidence" }

// Apply drops the segment when both no-speech and log-prob signals cross
// their thresholds.
func (s ConfidenceStage) Apply(_ context.Context, in SegmentDecision) SegmentDecision {
	c := in.Confidence
	if c == nil {
		return in
	}
	if c.NoSpeechProb > s.NoSpeechThreshold && c.AvgLogProb < s.LogProbThreshold {
		in.Outcome = Drop
		in.Reason = "low_confidence"
	}
	return in
}

var _ Stage = ConfidenceStage{}
