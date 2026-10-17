package pipeline

import "testing"

func applied(score, voiced float64) SpeakerDecision {
	return SpeakerDecision{Applied: true, Sufficient: true, Score: score, VoicedSeconds: voiced}
}

func TestSessionSpeakerState_WarmupNeverRejects(t *testing.T) {
	s := NewSessionSpeakerState(SpeakerConfig{Mode: "filter", Threshold: 0.5, MinDecisionSeconds: 5.0, ScoreSmoothing: 1.0})
	// 4s of low-scoring audio is still under the 5s warm-up window.
	allowed, _, _ := s.Observe(applied(0.1, 4.0))
	if !allowed {
		t.Fatalf("warm-up must allow")
	}
}

func TestSessionSpeakerState_RejectsAfterWarmup(t *testing.T) {
	// alpha=1 makes the EMA equal the latest score, isolating the warm-up logic.
	s := NewSessionSpeakerState(SpeakerConfig{Mode: "filter", Threshold: 0.5, MinDecisionSeconds: 3.0, ScoreSmoothing: 1.0})
	if allowed, _, _ := s.Observe(applied(0.1, 2.0)); !allowed {
		t.Fatalf("first 2s segment is warm-up, must allow")
	}
	allowed, score, reason := s.Observe(applied(0.1, 2.0)) // accrued 4s >= 3s
	if allowed {
		t.Fatalf("post-warmup low score must reject")
	}
	if score >= 0.5 {
		t.Fatalf("smoothed score should be below threshold, got %v", score)
	}
	if reason == "" {
		t.Fatalf("rejection must carry a reason")
	}
}

func TestSessionSpeakerState_AllowsHighScoreAfterWarmup(t *testing.T) {
	s := NewSessionSpeakerState(SpeakerConfig{Mode: "filter", Threshold: 0.5, MinDecisionSeconds: 1.0, ScoreSmoothing: 1.0})
	allowed, score, _ := s.Observe(applied(0.9, 2.0))
	if !allowed {
		t.Fatalf("high score past warm-up must allow")
	}
	if score < 0.5 {
		t.Fatalf("expected high smoothed score, got %v", score)
	}
}

func TestSessionSpeakerState_InsufficientIgnored(t *testing.T) {
	s := NewSessionSpeakerState(SpeakerConfig{Mode: "filter", Threshold: 0.5, MinDecisionSeconds: 1.0, ScoreSmoothing: 1.0})
	// An insufficient segment must not accrue voiced time or move the EMA.
	allowed, _, _ := s.Observe(SpeakerDecision{Applied: false, Sufficient: false, Score: 0.0, VoicedSeconds: 0.3})
	if !allowed {
		t.Fatalf("insufficient segment must allow")
	}
	if s.HasEvidence() {
		t.Fatalf("insufficient segment must not become evidence")
	}
	if s.accruedVoiced != 0 {
		t.Fatalf("insufficient segment must not accrue voiced seconds, got %v", s.accruedVoiced)
	}
}

func TestSessionSpeakerState_AdvisoryNeverRejects(t *testing.T) {
	s := NewSessionSpeakerState(SpeakerConfig{Mode: "advisory", Threshold: 0.9, MinDecisionSeconds: 0.1, ScoreSmoothing: 1.0})
	allowed, _, _ := s.Observe(applied(0.01, 5.0))
	if !allowed {
		t.Fatalf("advisory must always allow")
	}
}

func TestSessionSpeakerState_EMASmoothsAcrossSegments(t *testing.T) {
	s := NewSessionSpeakerState(SpeakerConfig{Mode: "advisory", Threshold: 0.5, MinDecisionSeconds: 0.1, ScoreSmoothing: 0.5})
	_, _, _ = s.Observe(applied(1.0, 1.0))      // seed EMA = 1.0
	_, score, _ := s.Observe(applied(0.0, 1.0)) // EMA = 0.5*0 + 0.5*1.0 = 0.5
	if score < 0.49 || score > 0.51 {
		t.Fatalf("expected smoothed EMA ~0.5, got %v", score)
	}
}

func TestSessionSpeakerState_DefaultsApplied(t *testing.T) {
	s := NewSessionSpeakerState(SpeakerConfig{Mode: "filter", Threshold: 0.5})
	if s.alpha != DefaultScoreSmoothing {
		t.Fatalf("expected default alpha %v, got %v", DefaultScoreSmoothing, s.alpha)
	}
	if s.minDecisionSeconds != DefaultMinDecisionSeconds {
		t.Fatalf("expected default min-decision %v, got %v", DefaultMinDecisionSeconds, s.minDecisionSeconds)
	}
}
