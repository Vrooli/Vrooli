# Fix VAD False Silence Detection in Noisy/Outdoor Environments

## Required Reading

- `prompt-manager skill read scientific-debugging` — Hypothesis-driven debugging methodology
- `prompt-manager skill read test` — Test authoring patterns

## Problem Statement

In persistent voice mode, the VAD frequently triggers false silence detection and stops recording during active speech. This is most pronounced in outdoor/noisy environments (walking, wind, traffic) where background noise profiles differ significantly from the initial calibration snapshot.

### Root Cause Analysis

The VAD state machine (`vad.ts`) has several architectural weaknesses for variable-noise environments:

1. **Calibration is a single 500ms snapshot**: The initial calibration computes a simple average of all RMS samples collected in the first 500ms. A single outlier (wind gust, car horn) during this window biases thresholds for the entire session. There is no outlier rejection or robust estimator.

2. **Fixed threshold multipliers (1.5x / 3x)**: The relationship between noise floor and speech/silence thresholds is hardcoded. Outdoor environments with higher baseline noise may need narrower ratios (speech is not proportionally louder), while quiet indoor environments could use wider ratios.

3. **Noise floor adaptation pauses during speech**: The sliding window correctly excludes speech-state samples (regression fix), but this means thresholds cannot adapt when the user is speaking through changing conditions (e.g., walking from a quiet area into a noisy street mid-sentence).

4. **No re-entry hysteresis in watchingSilence**: When transitioning from `watchingSilence` back to `speechDetected`, the VAD requires RMS > `speechThreshold` (the full 3x threshold). A softer re-entry threshold would prevent cutting off speech that briefly dips below the high threshold but is clearly still active.

## Scope

### Files In Scope
- `scenarios/web-console/ui/src/hooks/voice/vad.ts` — VAD state machine (primary)
- `scenarios/web-console/ui/src/hooks/voice/__tests__/vad.test.ts` — VAD tests
- `scenarios/web-console/ui/src/hooks/voice/audioUtils.ts` — Audio filter chain (if bandpass changes needed)
- `scenarios/web-console/ui/src/hooks/useVoiceInput.ts` — Orchestrator (if new config params needed)

### Acceptance Criteria
- `acceptance_allow`: `scenarios/web-console/ui/src/hooks/voice/**`, `scenarios/web-console/ui/src/hooks/useVoiceInput.ts`
- VAD maintains speech detection during simulated outdoor noise profiles (wind, traffic, crowd)
- No regression in quiet-environment detection accuracy
- Pure-function architecture preserved (no shared mutable state)
- All existing tests continue to pass

### Out of Scope
- ML-based VAD replacement (e.g., Silero VAD)
- Server-side audio processing
- UI changes for manual threshold configuration
- Wake word engine changes

## Approach

<!-- TBD — pending decisions on calibration strategy, threshold model, and re-entry hysteresis -->

### Phase 1: Robust Calibration
<!-- TBD — depends on D1 (calibration strategy) -->

### Phase 2: Adaptive Threshold Model
<!-- TBD — depends on D2 (threshold model) -->

### Phase 3: Watchside Re-entry Hysteresis
<!-- TBD — depends on D3 (re-entry threshold) -->

### Phase 4: Test Coverage for Outdoor Profiles
- Add test scenarios simulating: wind noise, traffic noise, crowd noise, transitioning environments
- Each scenario should verify VAD maintains speech detection through realistic noise patterns
- Test that calibration outlier rejection works correctly

## Risks

| Risk | Impact | Mitigation |
|------|--------|------------|
| Threshold changes cause missed real silence (over-sensitive to noise) | High — recording never stops | Test with genuine silence after noisy speech; keep silence timeout as hard backstop |
| Calibration changes break indoor/quiet accuracy | Medium | Regression test suite with quiet-environment profiles |
| Re-entry hysteresis causes infinite speech detection | Medium | Bound the re-entry multiplier; ensure silence timeout still triggers |
| Increased complexity of pure-function interface | Low | Keep all new params in VadRefs; no external state |

## Test Plan

- Unit tests for `computeSlidingNoiseFloor` with noisy input distributions
- Unit tests for new calibration logic (outlier rejection, robust estimator)
- Lifecycle tests simulating outdoor noise profiles (sustained high-variance RMS)
- Regression tests ensuring all existing behaviors are preserved
- Edge case: calibration during wind gust → verify thresholds are not inflated

## Dependencies

None — this is a self-contained fix within the voice input module.
