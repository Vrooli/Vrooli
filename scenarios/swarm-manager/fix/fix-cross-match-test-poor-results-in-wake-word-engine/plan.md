# Fix: Cross-Match Test Poor Results in Wake Word Engine

## Required Reading
- `implementation-plan-authoring` — canonical plan structure
- Source files in `scenarios/web-console/ui/src/hooks/voice/wakeword/` (engine.ts, dtw.ts, mfcc.ts, types.ts)
- Test files in `scenarios/web-console/ui/src/hooks/voice/__tests__/` (wakeWordEngine.test.ts, dtw.test.ts, mfcc.test.ts)

## Problem Statement

The MFCC+DTW wake word engine in web-console produces poor cross-match discrimination. When comparing genuinely different audio signals, the similarity scores are too compressed — different signals get scores that are unexpectedly close to matching signals, making it hard to reliably reject non-wake-word utterances while accepting real wake words.

### Root Cause Analysis

Three likely contributing factors have been identified through code review:

1. **`distanceToScore` sigmoid scaling is too flat (scale=1)**: The function `1 / (1 + distance * 1)` maps DTW distances to scores. For normalized DTW distances in the typical range of 0.3–3.0, this produces scores in the 0.25–0.77 range — a narrow band that makes threshold-based discrimination unreliable. The default threshold of 0.65 sits right in the middle of this compressed zone.

2. **CMS normalization removes too much discriminative information**: Cepstral Mean Subtraction is applied per-utterance at comparison time. While CMS helps with channel normalization, it removes the overall spectral tilt which carries speaker-identity and phonetic information. For short utterances (1–2 seconds), this can wash out important differences.

3. **DTW band may be too permissive**: `DTW_BAND_RATIO = 0.2` with a minimum of 10 frames allows significant warping that can artificially reduce distances between genuinely different signals.

### Potential Additional Factor

4. **Synthetic test signals don't expose the problem**: Current tests use pure tone harmonics (200Hz, 500Hz fundamentals). Real speech has far more complex spectral structure. The cross-match issue may only manifest with speech-like signals but existing tests can't detect it.

## Scope

### In Scope
- Diagnosing and fixing the score compression in `distanceToScore`
- Evaluating CMS normalization strategy
- Tuning DTW band ratio
- Adding cross-match discrimination tests with more realistic signals
- Adjusting `DEFAULT_WAKE_WORD_THRESHOLD` if the score distribution changes

### Out of Scope
- Replacing MFCC+DTW with neural embeddings (future work via the WakeWordEngine seam)
- Changes to `passiveListener.ts` (detection loop, VAD, ring buffer)
- UI/UX changes to enrollment or detection flow
- Backend API changes

## Approach

<!-- TBD — pending decisions on fix strategy -->

### Phase 1: Diagnostic
- Add a cross-match test that compares two distinct complex signals and logs the actual scores
- Verify the score compression hypothesis by examining raw DTW distances vs final scores

### Phase 2: Fix
- Adjust `distanceToScore` scale parameter to spread scores across a wider range
- Optionally tune CMS strategy (e.g., apply only to higher coefficients, skip c0)
- Re-calibrate `DEFAULT_WAKE_WORD_THRESHOLD` to match new score distribution

### Phase 3: Validation
- All existing tests continue to pass
- New cross-match tests demonstrate clear separation between match/non-match scores
- Score gap between identical-signal matches and different-signal comparisons is > 0.3

## Test Plan

### Existing Tests (must pass)
- `wakeWordEngine.test.ts` — 13 tests
- `dtw.test.ts` — 11 tests
- `mfcc.test.ts` — 13 tests

### New Tests
- Cross-match discrimination test: two distinct complex signals produce scores well below threshold
- Score separation test: identical signals score > 0.85, distinct signals score < 0.4 (with tuned parameters)
- Frequency-discrimination test: signals with different fundamentals produce adequately separated scores

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Changing score scale breaks existing enrollment templates | Users need to re-record wake words | Migration path: adjust threshold automatically or prompt re-enrollment |
| Tighter DTW band rejects valid tempo variations | False rejections increase | Test with time-warped signals at various stretch factors |
| CMS changes alter all scores unpredictably | Regression in existing match quality | Run full test suite after each change; compare score distributions |

## Acceptance Criteria

- [ ] Cross-match test passes with clear score separation (> 0.3 gap between match and non-match)
- [ ] All existing wake word engine tests pass
- [ ] `DEFAULT_WAKE_WORD_THRESHOLD` is calibrated to the new score distribution
- [ ] Score distribution is documented in test comments for future tuning reference
