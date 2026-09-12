package pipeline

// SessionSpeakerState folds the per-segment speaker decisions of ONE streaming
// session into a stable verdict. A single VAD segment is short and noisy, so
// deciding identity segment-by-segment makes the verdict swing mid-utterance.
// Instead this accumulates evidence:
//
//   - Undetermined segments (verification did not apply, or the resource judged
//     the segment to carry too little voiced audio) are ignored — they never
//     move the estimate and never cause a rejection.
//   - Applied segments update an exponential moving average of the score and
//     accrue voiced seconds.
//   - No rejection happens until at least MinDecisionSeconds of voiced audio has
//     accrued (the warm-up window), so a short first utterance is never falsely
//     rejected. After warm-up, filter mode rejects when the smoothed score is
//     below threshold; advisory mode never rejects.
//
// It is constructed once per session (see currentSpeakerIsolation) and mutated
// from the single egress goroutine, so it needs no internal locking.
type SessionSpeakerState struct {
	mode               string
	threshold          float64
	alpha              float64
	minDecisionSeconds float64

	accruedVoiced float64
	emaScore      float64
	haveEMA       bool
}

// NewSessionSpeakerState builds the per-session accumulator, applying defaults
// for any zero tuning value.
func NewSessionSpeakerState(cfg SpeakerConfig) *SessionSpeakerState {
	alpha := cfg.ScoreSmoothing
	if alpha <= 0 || alpha > 1 {
		alpha = DefaultScoreSmoothing
	}
	minDecision := cfg.MinDecisionSeconds
	if minDecision <= 0 {
		minDecision = DefaultMinDecisionSeconds
	}
	return &SessionSpeakerState{
		mode:               cfg.Mode,
		threshold:          cfg.Threshold,
		alpha:              alpha,
		minDecisionSeconds: minDecision,
	}
}

// Observe folds one segment decision into the session estimate and returns the
// session-level (allowed, smoothedScore, reason). reason is non-empty only on a
// rejection.
func (s *SessionSpeakerState) Observe(d SpeakerDecision) (bool, float64, string) {
	// Undetermined evidence: do not update the estimate, never reject.
	if !d.Applied || !d.Sufficient {
		return true, s.emaScore, ""
	}

	if s.haveEMA {
		s.emaScore = s.alpha*d.Score + (1-s.alpha)*s.emaScore
	} else {
		s.emaScore = d.Score
		s.haveEMA = true
	}
	s.accruedVoiced += d.VoicedSeconds

	if s.mode == "advisory" {
		return true, s.emaScore, ""
	}
	// Warm-up: not enough voiced audio yet to commit to a rejection.
	if s.accruedVoiced < s.minDecisionSeconds {
		return true, s.emaScore, ""
	}
	if s.emaScore >= s.threshold {
		return true, s.emaScore, ""
	}
	return false, s.emaScore, "speaker score below threshold over session"
}

// SmoothedScore is the current EMA score (0 before any applied segment).
func (s *SessionSpeakerState) SmoothedScore() float64 { return s.emaScore }

// HasEvidence reports whether at least one applied segment has updated the EMA.
func (s *SessionSpeakerState) HasEvidence() bool { return s.haveEMA }
