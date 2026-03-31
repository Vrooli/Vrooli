# Implementation Plan: Live Wake Word Testing Feature

## Required Reading

```bash
prompt-manager skill read react-coherence implementation-plan-authoring scenario-generation
```

## 1. Purpose

Add a live wake word testing mode to the web-console Voice Input Settings where the user can speak their enrolled wake word and see real-time match/reject feedback with score visualization — enabling iterative threshold tuning and confidence that detection works before relying on it.

## 2. Problem Statement

The web-console has a complete wake word enrollment pipeline (record 3-5 samples, extract MFCC features, save template) and a passive listener that runs in the background. However, there is **no way to verify detection works** before enabling passive mode. The existing "Test cross-match" button only validates that enrolled samples are similar to each other — it does not test whether a fresh spoken utterance would actually trigger detection.

Users need a "try it now" mode to:
- Speak the wake word and see if it matches
- See the match score to understand threshold sensitivity
- Iterate on threshold settings with immediate feedback
- Build confidence before enabling always-on passive listening

## 3. Scope

### In Scope
- Live test UI in VoiceInputSection (settings page)
- Real-time microphone capture → MFCC extraction → DTW comparison against enrolled template
- Visual feedback: match/reject indicator, score bar with threshold line, score history
- Threshold visualization relative to match scores
- Custom hook with extracted state machine for unit testability

### Out of Scope
- Modifying the PassiveListener or core detection engine
- Adding new wake word algorithms or models
- Backend changes (all testing is client-side using existing engine)
- Changes to the VoiceMicButton or main voice input flow
- ROC curves or statistical analysis tools
- Batch testing / automated test suites

## 4. Current Technical Context

### Key Files
| File | Role |
|------|------|
| `scenarios/web-console/ui/src/components/settings/VoiceInputSection.tsx` | Settings UI with enrollment, cross-match test |
| `scenarios/web-console/ui/src/hooks/voice/wakeword/engine.ts` | MFCC+DTW engine (`extractFeatures`, `compareBest`) |
| `scenarios/web-console/ui/src/hooks/voice/wakeword/passiveListener.ts` | Passive listener (VAD + ring buffer + detection loop) |
| `scenarios/web-console/ui/src/hooks/voice/wakeword/types.ts` | `AudioFeatures`, `WakeWordTemplate`, `MatchResult`, constants |
| `scenarios/web-console/ui/src/hooks/voice/vad.ts` | VAD utilities (`createVadRefs`, `vadTick`) |
| `scenarios/web-console/ui/src/hooks/useVoiceInput.ts` | Voice input orchestrator |

### New File
| File | Role |
|------|------|
| `scenarios/web-console/ui/src/hooks/voice/wakeword/useWakeWordTest.ts` | Custom hook encapsulating live test state machine and audio capture logic |

### Existing Capabilities
- **WakeWordEngine.extractFeatures()**: Takes `Float32Array` PCM + sampleRate → `AudioFeatures` (MFCC)
- **WakeWordEngine.compareBest()**: Takes candidate features + template samples + threshold → `MatchResult` `{isMatch, score, index}`
- **MediaRecorder + AudioContext.decodeAudioData()**: Already used in enrollment recording (WebM → PCM → MFCC)
- **Cross-match test**: Leave-one-out validation of enrolled samples against each other
- Engine functions are **stateless and pure** — can be called directly without PassiveListener

### Enrollment Recording Pattern (to reuse)
```
getUserMedia({ audio: true })
→ new MediaRecorder(stream, { mimeType: "audio/webm;codecs=opus" })
→ recorder.start(250)  // 250ms chunks
→ onstop: Blob → arrayBuffer → AudioContext({ sampleRate: 16000 }).decodeAudioData()
→ decoded.getChannelData(0)  // mono Float32Array
→ engine.extractFeatures(pcm, 16000) → AudioFeatures
```

## 5. Contract Decisions

### Settled (Round 1)

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Audio capture approach | **Lightweight MediaRecorder loop** | Reuses enrollment recording pattern. Simple, no coupling to PassiveListener. Each test is a discrete recording. |
| Test interaction model | **Push-to-test button** (hold to record, release to compare) | Explicit, mirrors enrollment UX. No VAD needed. Each attempt clearly delineated. |
| Score visualization | **Score bar with threshold line** | Horizontal bar (0-1) with vertical threshold marker. Green above, red below. Intuitive and compact. |
| Test result history | **Scrollable list of last 10 attempts** | Each entry: score bar, pass/fail, timestamp. Enables threshold tuning across attempts. |
| Testing strategy | **Unit tests for the test-loop hook** | Extract logic into custom hook, unit test state machine (idle → recording → comparing → result). Mock engine calls. |

### Settled (Round 2)

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Mutual exclusion mechanism | **Shared `isRecording` flag in VoiceInputSection** | Simple boolean state that both enrollment and test check before starting. Enrollment sets it when `wwRecordingIdx !== null`, test hook accepts it as a `disabled` parameter. Lightweight, explicit, coordination stays in parent. |
| Hook file location | **`hooks/voice/wakeword/useWakeWordTest.ts`**, inline UI in VoiceInputSection | Hook lives alongside other wake word code. UI stays inline since it's a settings subsection, not a reusable component. |
| Recording duration limits | **Min 0.5s, max 3s, auto-stop at max** | Wake words are typically 0.5–2s. 3s max prevents overly long recordings that dilute DTW matching. If released before 0.5s, show hint to hold longer. |
| Score bar implementation | **Plain div-based bar with CSS/Tailwind** | Container div with inner div sized to score %. Threshold as absolute-positioned vertical line. Green/red via Tailwind classes based on `isMatch`. No extra dependencies, matches existing settings page style. |

## 6. Target End State

A "Test Live Detection" section in VoiceInputSection that:
1. Has a prominent push-to-test button (hold to record, release to compare)
2. Shows recording state feedback while button is held
3. Auto-stops recording at 3s; rejects recordings shorter than 0.5s with a hint
4. On release, extracts MFCC and compares against enrolled template
5. Displays match result as a div-based score bar with threshold line (green above, red below)
6. Maintains a scrollable history of last 10 test attempts
7. Each history entry shows: score bar mini-version, pass/fail badge, timestamp
8. Disabled when no wake word template is enrolled or when enrollment is recording (mutual exclusion via shared `isRecording` flag)

## 7. Implementation Strategy

### Phase 1: Custom Hook (`useWakeWordTest`)

**File:** `scenarios/web-console/ui/src/hooks/voice/wakeword/useWakeWordTest.ts`

Extract all test logic into a testable custom hook:

```typescript
interface WakeWordTestState {
  status: "idle" | "recording" | "comparing" | "result";
  currentResult: { score: number; isMatch: boolean; timestamp: number } | null;
  history: Array<{ score: number; isMatch: boolean; timestamp: number }>;
  error: string | null;
}

interface UseWakeWordTestReturn {
  state: WakeWordTestState;
  startRecording: () => void;
  stopRecording: () => void;   // triggers compare
  clearHistory: () => void;
}
```

- State machine: `idle` → (hold button) → `recording` → (release button) → `comparing` → `result` → `idle`
- On `stopRecording`: collect MediaRecorder chunks → decode → extractFeatures → compareBest → push to history
- Max 10 history entries (FIFO)
- Accept engine + template + threshold as parameters (dependency injection for testability)
- **Recording bounds**: auto-stop timer at 3s; if recording < 0.5s, set error with hint instead of comparing
- **Mutual exclusion**: accept a `disabled` parameter; when true, `startRecording` is a no-op

### Phase 2: UI Component

Add "Test Live Detection" section in VoiceInputSection below the cross-match test:

- **Shared `isRecording` flag**: boolean state in VoiceInputSection, set `true` when `wwRecordingIdx !== null` OR test is recording. Both enrollment and test check this before starting.
- **Push-to-test button**: Large, prominent. Shows "Hold to Test" / "Recording..." / "Comparing..." states. Disabled when `isRecording` (enrollment active) or no template enrolled.
- **Score bar**: Plain div container with inner div sized to `score * 100%`. Absolute-positioned vertical line at `threshold * 100%` from left. Green fill when `isMatch`, red otherwise. Tailwind utility classes.
- **History list**: Scrollable div, newest first. Each row: mini score bar, pass/fail badge (`text-green-500` / `text-red-500`), relative timestamp. Max 10 entries.
- **Disabled state**: When no template enrolled, show explanatory message. When enrollment is recording, disable test button with tooltip.

### Phase 3: Unit Tests

**File:** `scenarios/web-console/ui/src/hooks/voice/__tests__/useWakeWordTest.test.ts`

Test the `useWakeWordTest` hook state machine:
- Transitions: idle → recording → comparing → result → idle
- History accumulation and FIFO cap at 10
- Error states: no template → error, recording < 0.5s → error with hint, recording failure
- Auto-stop at 3s max duration
- `disabled` parameter prevents recording start
- Mock `WakeWordEngine` for deterministic score results
- Cleanup: verify mic stream release on unmount

### Phase 4: Polish

- Visual transitions for recording state (pulse animation on button while recording)
- Error handling: mic permission denied, AudioContext issues
- Cleanup: release mic stream on unmount or navigation
- Verify mutual exclusion works in both directions (test blocks enrollment, enrollment blocks test)

## 8. Testing Plan

### Unit Tests (Primary)
- **Hook state machine**: Test all transitions via `renderHook` with mocked engine
- **History management**: Verify FIFO behavior, max 10 entries, clearHistory
- **Recording bounds**: Verify min 0.5s rejection, max 3s auto-stop
- **Disabled state**: Verify `startRecording` is no-op when disabled
- **Error paths**: No template enrolled, recording start failure, decode failure
- **Cleanup**: Verify mic stream release on unmount

### Manual Verification
- Speak enrolled wake word → green match with high score
- Speak random words → red reject with low score
- Threshold adjustment → reflected in pass/fail with same scores
- Mic released after stopping test (check browser indicator)
- Cannot record enrollment and test simultaneously
- Recording auto-stops at 3s
- Short press (< 0.5s) shows "hold longer" hint

## 9. Rollout / Validation Checklist

- [ ] Live test mode starts and stops cleanly
- [ ] Mic is released when testing stops
- [ ] Recording auto-stops at 3s max duration
- [ ] Recordings < 0.5s show hint instead of comparing
- [ ] Match scores align with cross-match validation scores
- [ ] Threshold changes are reflected in real-time results
- [ ] Mutual exclusion works: enrollment blocks test, test blocks enrollment
- [ ] Works across Chrome, Firefox, Safari (getUserMedia compatibility)
- [ ] Hook unit tests pass

## 10. Risks + Mitigations

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Browser mic permission UX varies | Medium | Low | Reuse existing permission flow from enrollment |
| AudioContext limit (6-8 per page) | Medium | Medium | Reuse shared AudioContext — create once, pass to hook |
| MFCC extraction blocks main thread | Low | Medium | Already benchmarked at <20ms — acceptable for test UI |
| Concurrent mic access (test vs enrollment) | Medium | High | Shared `isRecording` flag in VoiceInputSection; both check before starting |
| WebM decoding fails on some browsers | Low | Medium | Use same MIME type fallback as enrollment |
| Very short recordings produce garbage MFCC | Medium | Low | Enforce 0.5s minimum; show hint to hold longer |

## 11. Non-goals / Prohibited Patterns

- Do NOT modify PassiveListener or engine internals
- Do NOT add new npm dependencies for audio processing
- Do NOT stream audio to the backend — all processing is client-side
- Do NOT add feature flags — this is a settings-page addition
- Do NOT add backwards-compatibility shims
- Do NOT extract a separate component for the test UI — keep inline in VoiceInputSection

## 12. Definition of Done

- User can hold push-to-test button to record, release to compare
- Recording auto-stops at 3s; recordings < 0.5s show hint to hold longer
- Speaking the enrolled wake word shows a match with score bar (green, above threshold)
- Speaking other words shows a reject with score bar (red, below threshold)
- Score bar uses plain div with Tailwind, threshold shown as vertical line
- Score history (last 10) persists during the test session
- Mic is released when testing stops or user navigates away
- Mutual exclusion with enrollment recording via shared `isRecording` flag
- Hook unit tests pass with mocked engine
- No regressions to enrollment or passive listener flows
