package validation

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestSyntheticPCMIsContinuousDeterministicPCM(t *testing.T) {
	first := syntheticPCM(syntheticSampleRate, syntheticFrameDuration, 0)
	second := syntheticPCM(syntheticSampleRate, syntheticFrameDuration, int64(len(first)/syntheticBytesPerSample))

	require.Len(t, first, 3200)
	require.Len(t, second, 3200)
	combined := syntheticPCM(syntheticSampleRate, 2*syntheticFrameDuration, 0)
	require.Equal(t, combined[:len(first)], first)
	require.Equal(t, combined[len(first):], second, "frame boundaries must retain the same phase")
	require.Equal(t, first, syntheticPCM(syntheticSampleRate, syntheticFrameDuration, 0))
	require.NotEqual(t, first, syntheticPCM(syntheticSampleRate, syntheticFrameDuration, 37), "a non-boundary offset must change the phase")
}

func TestRunAndRecordRejectsShortQualificationBeforeExternalEffects(t *testing.T) {
	h := newHandlersWithClock(nil, func() time.Time { return time.Unix(0, 0) })
	err := h.runAndRecord(t.Context(), t.TempDir(), filepath.Join(t.TempDir(), "evidence.json"), 59*time.Minute, os.Stdout, false)
	require.ErrorContains(t, err, "at least 60 minutes")
}

func TestMarkOutOfBandValidationsUpdatesEveryValidation(t *testing.T) {
	dir := t.TempDir()
	modulePath := filepath.Join(dir, "module.json")
	module := map[string]any{
		"requirements": []any{
			map[string]any{
				"id": "ATD-P0-TEST",
				"validation": []any{
					map[string]any{"type": "automation", "phase": "performance", "status": "planned", "ref": "one", "out_of_band": true, "valid_for_days": 7},
					map[string]any{"type": "test", "phase": "unit", "status": "implemented", "ref": "two"},
				},
			},
		},
	}
	data, err := json.Marshal(module)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(modulePath, data, 0o644))

	when := time.Date(2026, time.August, 4, 18, 0, 0, 0, time.UTC)
	updated, err := markOutOfBandValidations(dir, when)
	require.NoError(t, err)
	require.Equal(t, 1, updated)

	stale, err := findStaleValidations(dir, when.Add(6*24*time.Hour))
	require.NoError(t, err)
	require.Empty(t, stale)
	stale, err = findStaleValidations(dir, when.Add(8*24*time.Hour))
	require.NoError(t, err)
	require.Len(t, stale, 1)
}

func TestFindStaleValidationsFailsClosedForMissingEvidence(t *testing.T) {
	dir := t.TempDir()
	data := []byte(`{"requirements":[{"id":"ATD-P0-TEST","validation":[{"type":"automation","phase":"performance","status":"planned","ref":"one","out_of_band":true,"valid_for_days":7}]}]}`)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "module.json"), data, 0o644))

	stale, err := findStaleValidations(dir, time.Now())
	require.NoError(t, err)
	require.Equal(t, []string{"ATD-P0-TEST (one)"}, stale)
}
