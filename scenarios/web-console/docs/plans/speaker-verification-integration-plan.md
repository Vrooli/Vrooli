# Speaker Verification Integration Plan

## Purpose

Integrate the standalone `speaker-verification` resource into `web-console` so the voice pipeline can enroll the user's voice, verify speech segments against that enrolled profile, and suppress background speakers before dictation or command handling commits text into the UI.

This plan assumes the `speaker-verification` resource remains the source of truth for enrollment, profile persistence, and similarity scoring. `web-console` should orchestrate it, not reimplement speaker recognition logic locally.

## Scope

### In Scope

- `web-console` capability detection for `speaker-verification`
- backend proxy/config endpoints needed by the UI
- enrollment UI in Settings
- persistent storage of `web-console` voice verification preferences
- segment-level verification before transcript commit
- UX for accepted, rejected, and uncertain segments
- tests covering the new integration seam
- documentation updates for seams and temporal flows

### Out of Scope

- changing the `speaker-verification` resource API shape
- reworking the core Whisper streaming architecture
- cloud fallback
- multi-user profile management beyond a single default enrolled speaker
- always-on wake-word detection

## Preconditions

Before `web-console` consumes the resource, fix these resource issues:

1. `/ready` must fail readiness semantically, not just return `{status:"not_ready"}` with HTTP 200.
2. lifecycle checks must parse readiness JSON instead of treating any 200 from `/ready` as ready.
3. integration tests must include a negative verification case, not only same-signal success.
4. committed `__pycache__` artifacts should be removed from the resource tree.

Without these fixes, `web-console` can present false confidence about speaker verification availability.

## Current Architecture

Relevant existing seams:

- `api/voice_stream_ws.go`
  - receives streaming audio, emits `partial`, `segment-final`, and `final`
- `ui/src/hooks/useVoiceInput.ts`
  - orchestrates VAD, provider lifecycle, segment accumulation, and command parsing
- `api/voice_config.go`
  - persists voice config for the scenario
- `ui/src/components/settings/VoiceInputSection.tsx`
  - already exposes voice settings and advanced streaming controls

Current behavior:

- VAD segments speech
- Whisper transcribes segments
- `useVoiceInput` routes final segment text either to dictation or command parsing

Missing behavior:

- segment audio is not checked against an enrolled speaker before text is accepted

## Core Design Decision

Verify **audio segments**, not transcript text.

Pipeline:

```text
mic -> VAD -> segment boundary -> segment audio snapshot -> speaker verification
    -> if accepted: keep Whisper segment-final
    -> if rejected: suppress segment-final
    -> if uncertain: surface lightweight UI feedback without auto-commit
```

This should happen at the same segment seam already used for high-quality retranscription.

## Seams to Add

### 1. Speaker Verification Resource Seam

Add a backend seam in `web-console` for calling the resource:

- capability check for `speaker-verification`
- profile list / profile existence
- enroll profile
- verify segment
- health/readiness summary

Recommended new backend files:

- `api/speaker_verification_client.go`
- `api/speaker_verification_config.go`
- `api/speaker_verification_handlers.go`

### 2. Voice Verification Decision Seam

Add a narrow decision layer between segment-final audio availability and transcript commit.

Recommended new backend abstraction:

```text
type SpeakerVerificationResult struct {
    Matched bool
    Score float64
    Threshold float64
    ProfileID string
}
```

This seam should make it easy to mock verification in tests without depending on a real resource.

## API Additions in Web Console

Add scenario-local API endpoints that proxy to the resource and persist scenario preferences.

### Profile and Status

- `GET /api/v1/voice/speaker/status`
  - capability available
  - resource healthy
  - profile configured
  - profile exists
  - threshold
  - verification enabled

- `GET /api/v1/voice/speaker/profiles`
  - list profiles from the resource

### Enrollment

- `POST /api/v1/voice/speaker/enroll`
  - accepts enrollment audio blob and metadata
  - forwards to resource `POST /v1/profiles`
  - optionally stores selected profile as the active profile in `web-console`

- `DELETE /api/v1/voice/speaker/profile`
  - removes the active profile binding in `web-console`
  - should not necessarily delete the resource profile unless explicitly requested

### Config

- `GET /api/v1/voice/speaker/config`
- `PUT /api/v1/voice/speaker/config`

Config fields:

- `speakerVerificationEnabled`
- `speakerProfileId`
- `speakerVerificationThreshold`
- `speakerVerificationMode`
  - `off | filter | advisory`
- `speakerVerificationRejectBehavior`
  - `drop | show-muted`

## Voice Config Persistence

Extend the existing voice config persistence in `api/voice_config.go` rather than creating a parallel store.

Recommended new fields:

- `SpeakerVerificationEnabled bool`
- `SpeakerProfileID string`
- `SpeakerVerificationThreshold float64`
- `SpeakerVerificationMode string`
- `SpeakerRejectBehavior string`

Validation rules:

- threshold must be in `[0,1]`
- `SpeakerProfileID` required when verification is enabled
- mode and reject behavior must be enum validated

## Backend Flow Changes

### Preferred Implementation

Add verification in the backend WebSocket flow where segment audio already exists.

In `api/voice_stream_ws.go`:

1. on `segment-boundary`, keep the current segment audio snapshot
2. call the speaker verification resource with that exact audio snapshot
3. if verification matches:
   - continue current behavior
   - emit `segment-final`
4. if verification rejects:
   - do not emit `segment-final`
   - optionally emit `segment-rejected`
5. if verification errors:
   - degrade gracefully according to config
   - either allow transcript through or suppress it, but do so explicitly

Why backend-side:

- avoids sending raw segment audio to a second browser-side pipeline
- keeps trust decisions server-side
- reuses the exact segment audio already snapped in `voice_stream_ws.go`
- is easier to test with mocked resource responses

### WebSocket Message Additions

Add new server-to-client messages:

- `{ type: "segment-accepted", segmentIndex, score }`
- `{ type: "segment-rejected", segmentIndex, score, threshold }`
- `{ type: "speaker-status", enabled, profileConfigured }`

Do not send full rejected text to the UI by default.

## UI and UX Changes

### Settings

Extend `ui/src/components/settings/VoiceInputSection.tsx` with a new section:

- `Use speaker verification`
- `Active voice profile`
- `Enroll / Re-enroll voice`
- threshold slider
- verification mode selector
- status line:
  - resource available
  - profile configured
  - last enroll timestamp if available

Recommended UX copy:

- "Only accept dictation that matches your enrolled voice."

### Enrollment Flow

Enrollment should be explicit and guided:

1. user opens Settings -> Voice Input
2. taps `Enroll voice`
3. sees instructions:
   - quiet environment preferred
   - read prompted phrase for 5-10 seconds
4. records enrollment audio
5. upload and enroll via `web-console` backend
6. show success/failure and active profile state

Optional v1 prompt text:

- ask the user to read a fixed phrase plus a short free-form sentence

### Runtime UX

When verification is enabled:

- accepted segments behave exactly like current segments
- rejected segments should not pollute the text input
- optionally show a subtle non-blocking notice like:
  - `Ignored speech that did not match your voice`

Avoid noisy toasts for every reject while walking.

## Client Hook Changes

### `useVoiceInput`

Keep `useVoiceInput` as the orchestration seam, but do not make it implement speaker verification logic itself.

Changes:

- handle new WebSocket message types
- track accepted/rejected segment state
- only commit transcripts that the backend has accepted
- keep command parsing after acceptance, not before

New state suggestions:

- `speakerVerificationEnabled`
- `speakerProfileConfigured`
- `lastRejectedSegmentAt`
- `lastVerificationScore`

### `VoiceStreamProvider`

Extend message handling only.

Do not move speaker verification into the provider. It should stay transport-focused.

## Capability Detection

Update `web-console` capability checks so the UI can distinguish:

- Whisper available
- speaker verification available
- both available

Expected capability wiring:

- add a new dependency slug and feature set for `speaker-verification`
- expose it in the integrations/settings UI

Suggested features:

- `speaker-verification`
- `voice-enrollment`

## Modes and Fallbacks

### Verification Modes

- `off`
  - current behavior
- `filter`
  - reject non-matching segments before commit
- `advisory`
  - still commit, but annotate mismatches for debugging/tuning

Ship `filter` and `off` first. `advisory` is useful for tuning but can wait if scope needs tightening.

### Failure Policy

If the resource is unavailable mid-session:

- show one fallback notice
- continue with existing Whisper-only behavior only if the user has explicitly allowed fallback

Recommended config:

- `fallbackToWhisperWithoutVerification: false` by default for users who intentionally enabled voice filtering

## Test Plan

### Backend Unit Tests

Add tests for:

- speaker verification config validation
- resource client request/response mapping
- segment acceptance and rejection routing
- fallback policy handling

Recommended files:

- `api/speaker_verification_client_test.go`
- `api/speaker_verification_config_test.go`
- `api/voice_stream_ws_test.go`

### Backend Integration Tests

Use a fake speaker verification server:

- accepted segment returns match
- rejected segment returns mismatch
- resource timeout/error path

Assertions:

- accepted segment produces `segment-final`
- rejected segment suppresses `segment-final`
- command parsing never runs on rejected segments

### UI Tests

Add tests for:

- settings enrollment controls
- profile status rendering
- threshold persistence
- rejected segment does not update input
- accepted segment still updates input
- fallback notice rendering

Recommended files:

- `ui/src/__tests__/settings-voice-speaker-verification.test.tsx`
- `ui/src/__tests__/useVoiceInput-speaker-verification.test.ts`

### Manual Validation

Manual scenarios:

1. enroll in a quiet room
2. speak normally with TV/background speech present
3. verify own speech passes
4. verify TV speech is suppressed
5. verify command phrase works only for enrolled speaker
6. disable verification and confirm current behavior still works

## Documentation Updates

Update:

- `docs/internal/SEAMS.md`
  - add resource seam and acceptance decision seam
- `docs/internal/TEMPORAL-FLOWS.md`
  - add segment verification flow before transcript commit
- `docs/internal/INVARIANTS.md`
  - accepted text must never come from an unverified segment when filter mode is enabled

## Execution Phases

### Phase 1: Preconditions and Seams

- fix resource readiness semantics
- add scenario resource client and config types
- add capability wiring

### Phase 2: Backend Integration

- add proxy endpoints
- extend `voice_config.go`
- verify segment audio in `voice_stream_ws.go`
- add new WebSocket message types

### Phase 3: Settings and Enrollment UX

- add settings UI
- add enrollment flow
- persist active profile and threshold

### Phase 4: Runtime UX

- surface accepted/rejected segment handling in `useVoiceInput`
- wire notices and fallback behavior

### Phase 5: Validation and Docs

- backend tests
- UI tests
- manual validation pass
- seam and flow documentation updates

## Acceptance Criteria

The integration is done when:

- a user can enroll their voice from `web-console`
- verification settings persist
- non-matching speakers do not populate dictation text in filter mode
- command parsing only runs on accepted segments
- turning verification off restores current behavior without regressions
- backend and UI tests cover acceptance, rejection, and fallback paths
