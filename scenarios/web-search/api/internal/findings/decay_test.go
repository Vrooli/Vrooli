package findings_test

import (
	"testing"
	"time"

	"web-search/internal/findings"

	"github.com/stretchr/testify/require"
)

func TestEffectiveConfidenceDecaysWithAge(t *testing.T) {
	now := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)

	// Fresh finding: effective == stored.
	fresh := findings.Finding{Confidence: 0.8, RetrievalDate: now}
	require.InDelta(t, 0.8, findings.EffectiveConfidence(fresh, now), 1e-9)

	// One half-life old: effective == half the stored confidence.
	oneHalfLife := findings.Finding{Confidence: 0.8, RetrievalDate: now.Add(-findings.DecayHalfLife)}
	require.InDelta(t, 0.4, findings.EffectiveConfidence(oneHalfLife, now), 1e-6)

	// Two half-lives old: a quarter.
	twoHalfLives := findings.Finding{Confidence: 0.8, RetrievalDate: now.Add(-2 * findings.DecayHalfLife)}
	require.InDelta(t, 0.2, findings.EffectiveConfidence(twoHalfLives, now), 1e-6)

	// Strictly monotonic: older is never more confident than newer.
	older := findings.EffectiveConfidence(twoHalfLives, now)
	newer := findings.EffectiveConfidence(oneHalfLife, now)
	require.Less(t, older, newer)
}

func TestEffectiveConfidenceFallsBackToCreatedAt(t *testing.T) {
	now := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	// No retrieval date — age anchors on created_at.
	f := findings.Finding{Confidence: 1.0, CreatedAt: now.Add(-findings.DecayHalfLife)}
	require.InDelta(t, 0.5, findings.EffectiveConfidence(f, now), 1e-6)
}

func TestEffectiveConfidenceZeroAndFutureStamps(t *testing.T) {
	now := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)

	// Zero stamps: no decay, returns clamped stored confidence.
	zero := findings.Finding{Confidence: 0.7}
	require.InDelta(t, 0.7, findings.EffectiveConfidence(zero, now), 1e-9)
	require.Equal(t, time.Duration(0), findings.Age(zero, now))

	// Future retrieval date clamps age to 0 (never amplifies confidence).
	future := findings.Finding{Confidence: 0.7, RetrievalDate: now.Add(time.Hour)}
	require.InDelta(t, 0.7, findings.EffectiveConfidence(future, now), 1e-9)
	require.Equal(t, time.Duration(0), findings.Age(future, now))
}
