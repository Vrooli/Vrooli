# Speech-to-Text — Voice Filter Retry Implementation Plan

## 1. Purpose

When the web-console's speech-to-text feature rejects a recording because
speaker verification did not recognize the user's voice, the current UX
loses the transcription entirely and flashes a 3-second banner. Users who
just finished a long monologue have no way to recover the audio. This
plan replaces the auto-dismissing notice with a persistent, explicitly
dismissable banner that offers **"Transcribe anyway without voice
filter"** — producing the transcript from the original audio, bypassing
the speaker-verification gate for that one submission.

The result: speaker verification remains the default filter; false
rejections no longer cost the user their message.

## 2. Greenfield Constraint (HARD RULE)

**No compatibility shims, fallback code paths, feature flags, legacy
exports, dead-code comments, or transitional "if new-path else
old-path" branches.**

Concretely forbidden in this plan:

- Keeping the 3-second `speakerNoticeTimerRef` auto-dismiss "just in
  case the new persistent banner breaks." Delete the ref, its timer,
  and its cleanup site in the same change.
- A `VITE_ENABLE_VOICE_RETRY` flag, query-string toggle, or admin
  setting to disable the new flow.
- A second `/voice/transcribe-bypass` endpoint next to the existing
  `/voice/transcribe`. The one endpoint gains one explicit query
  parameter; the old shape is not preserved in parallel.
- Renaming discarded state to `_unusedChunks`. Discarded state is
  removed.
- `// removed: old auto-dismiss path (see commit XYZ)` comments.
  Commit history is the history; the code is clean.
- Dual state fields (`rejectedAudioLegacy`, `rejectedAudioV2`). There
  is one retention field; its tests cover every case.

UI and API ship together. Scenario-internal consumers are updated in
the same diff. When a function's signature changes, every caller is
updated — no adapter.

## 3. Required Reading

A future agent resuming this plan must run, in order:

```bash
prompt-manager skill read implementation-plan-authoring scientific-debugging
prompt-manager skill read cli-steer api-steer utils-unification seam-discovery-and-enforcement
```

Also read before coding:

- `scenarios/web-console/docs/concepts/ARCHITECTURE.md`
  (section `voice-input` if present; otherwise start from
  `ui/src/hooks/useVoiceInput.ts`).
- `scenarios/web-console/docs/internal/SEAMS.md` (voice provider seam,
  `VoiceProvider` interface).

## 4. Problem Statement

### 4.1 Symptom

In conversational voice mode, if speaker verification rejects either a
full recording (HTTP batch path) or an individual segment (streaming
path), the user sees a brief sky-blue banner ("Speaker verification
advisory…") that disappears after 3 seconds. The audio is discarded.
There is no way to recover the transcript. For a ~30-second message
this is a hard loss: the user has to speak it again.

### 4.2 Root cause (confirmed by code inspection)

1. `ui/src/hooks/useVoiceInput.ts:710-712`: on
   `provider.onSegmentRejected`, the hook sets a 3-second
   `setTimeout` to clear `speakerNotice`, then does nothing else.
2. `ui/src/hooks/voice/WhisperProvider.ts:61-62`: on
   `mediaRecorder.onstop`, the collected `chunks: Blob[]` are
   combined into one blob **and then `this.chunks = []`**.
   The blob is passed to the HTTP transcribe call; when the server
   rejects it, the blob is no longer referenced anywhere.
3. `ui/src/hooks/voice/VoiceStreamProvider.ts`: accumulates segment
   audio but does not retain a per-turn buffer across segment
   rejections; rejected segment bytes are dropped.
4. `api/voice_transcribe.go:61-81`:
   `evaluateSpeakerVerification` runs first; if
   `!decision.Allowed`, the function returns empty text without ever
   calling Whisper. There is no query parameter to skip the gate.
5. `api/voice_stream_ws.go:238-259` (segment) and
   `api/voice_stream_ws.go:549-570` (final): identical gate pattern
   in the streaming path; no bypass.

### 4.3 Consequence

The rejection is both silent (no recovery UI) and destructive (no
retained audio). The user has no path to get their transcript out
without re-recording. For a user who has voice enrolled and speaks
in a variable environment (background noise, angle of mic, cold, a
cold), rejections are periodic and painful.

## 5. Scope

### In scope

- `ui/src/hooks/useVoiceInput.ts`: remove the 3-second auto-dismiss
  entirely; add `rejectedAudio` state (one blob, most recent
  rejection only); add `retryWithoutFilter()` action; add
  `dismissRejection()` action.
- `ui/src/hooks/voice/WhisperProvider.ts`: retain the full-turn
  audio blob after `mediaRecorder.onstop`; expose it via a new
  `getLastTurnAudio(): Blob | null` seam; clear it only on
  explicit `disposeLastTurn()` (called after success, dismissal,
  or timeout).
- `ui/src/hooks/voice/VoiceStreamProvider.ts`: accumulate all
  segments (accepted and rejected) of a single turn into a raw
  PCM/WebM buffer held until turn end; expose the same
  `getLastTurnAudio()` / `disposeLastTurn()` seams.
- `ui/src/hooks/voice/types.ts`: the `VoiceProvider` interface
  gains `getLastTurnAudio` and `disposeLastTurn` as required
  methods (greenfield — both providers implement; no optional
  chaining).
- `ui/src/components/Workspace.tsx` (speaker-notice banner site):
  replace the fading banner with a persistent
  `VoiceRejectionBanner` component that has Dismiss and Transcribe-
  anyway buttons.
- `ui/src/components/VoiceRejectionBanner.tsx` (new): small,
  focused component; drives the two actions via props.
- `api/voice_transcribe.go`: accept `skip_speaker_verification=true`
  query parameter; when set, bypass the gate but log a counter.
- `api/voice_config.go`: expose `SpeakerVerificationBypassCount`
  metric.
- `api/voice_stream_ws.go`: **not extended** — streaming retry is
  out of scope (see §5 Out of scope rationale).
- Unit and integration tests for every state transition, provider
  method, and endpoint behavior (§10).

### Out of scope

- Streaming-mode mid-turn retry. Streaming provider accumulates
  the full-turn audio, but the retry itself is an HTTP POST to the
  batch endpoint using the retained buffer. Adding a streaming
  "retry without filter" control-message would complicate the
  wire protocol without user value (the retry is a one-shot
  operation, and HTTP is already the tail of the turn for the
  streaming path too).
- Multi-rejection retention (a queue of past rejected turns). Only
  the most recent rejection is retained. The previous rejection is
  disposed when a new one arrives.
- Settings UI for disabling speaker verification entirely. That
  already exists at
  `ui/src/components/settings/VoiceInputSection.tsx` — this plan
  does not touch it.
- Server-side speaker-verification tuning or model improvements.
  False rejection rate is a separate concern.
- Voice-mode wake-word or command handling. Untouched.
- Rate-limiting or anti-abuse around the bypass parameter beyond
  what's described in §9.4.

## 6. Current Technical Context

### 6.1 Frontend files (paths relative to `scenarios/web-console/ui/`)

| File | Role | Key lines |
|---|---|---|
| `src/hooks/useVoiceInput.ts` | Top-level voice-input hook; owns `VoiceInputState`, provider wiring, VAD. | 65 (state shape), 467 (notice timer ref), 568-570 (clear timer on start), 696-712 (rejection handlers + 3s timer), 915 / 1006 / 1013 / 1026 (cleanup paths). |
| `src/hooks/voice/types.ts` | `VoiceProvider` interface. | Interface definition. |
| `src/hooks/voice/WhisperProvider.ts` | HTTP batch provider. 98 LOC. | 13 (`chunks: Blob[]`), 44 (reset on start), 53 (push), 61-62 (blob build + reset). |
| `src/hooks/voice/VoiceStreamProvider.ts` | WebSocket streaming provider. 435 LOC. | Segment accumulation; `onSegmentRejected` callback. |
| `src/components/Workspace.tsx` | Renders the speaker notice banner. | 971-974 (banner site). |
| `src/components/VoiceMicButton.tsx` | Mic button + tooltip. 251 LOC. | 242 (error tooltip). |
| `src/components/settings/VoiceInputSection.tsx` | Speaker enrollment settings. | Existing; untouched. |

### 6.2 Backend files (paths relative to `scenarios/web-console/api/`)

| File | Role | Key lines |
|---|---|---|
| `voice_transcribe.go` | HTTP `/voice/transcribe` handler. | 61 (`evaluateSpeakerVerification`), 68 (decision branches), 77 (reject path). |
| `voice_stream_ws.go` | WebSocket streaming handler. | 42 (`VoiceMsgSegmentRejected`), 238 / 247 / 257 (segment gate), 549 / 556 / 566 (final gate). |
| `voice_config.go` | Voice runtime config and counters. | `PUT /voice/config`, `PUT /speaker-verification/config`. |
| `voice_transcribe_test.go` | HTTP transcribe tests. | Extend in §10. |
| `voice_stream_ws_test.go` | Streaming tests. | Untouched (streaming out of scope). |

### 6.3 Current UX (as of 2026-04-23)

- User holds mic, speaks, releases.
- `MediaRecorder.onstop` builds a single `audio/webm` blob.
- `WhisperProvider.transcribe(blob)` POSTs to
  `/voice/transcribe`.
- Server runs speaker verification; if allowed, runs Whisper and
  returns text. If not allowed, returns empty.
- Client's `onSegmentRejected` handler sets `speakerNotice`,
  starts a 3-second timer to clear it. Blob is no longer
  referenced.

### 6.4 What this plan adds

- Retention of the full-turn audio as a `Blob` held in the
  provider until explicitly disposed.
- A `rejectedAudio` slot in `VoiceInputState` containing a stable
  id, the blob, and the rejection metadata (score, threshold,
  timestamp). One slot, not a queue.
- A `retryWithoutFilter()` action that POSTs the same blob with
  `?skip_speaker_verification=true`.
- A `dismissRejection()` action.
- A persistent banner (`VoiceRejectionBanner`) rendered while
  `rejectedAudio !== null`.
- A server bypass query parameter that is scoped to the existing
  session auth; no new auth surface.

## 7. Target End State

After this plan:

1. The 3-second `speakerNoticeTimerRef` auto-dismiss is **deleted
   from the tree** (including its `useRef`, its `setTimeout`, its
   cleanup sites). `grep -n "speakerNoticeTimerRef" ui/src/` returns
   zero matches.
2. `VoiceInputState.speakerNotice` is replaced by
   `VoiceInputState.rejectedAudio` of type
   `VoiceRejection | null`:
   ```ts
   interface VoiceRejection {
     id: string;             // stable id for settlement matching
     blob: Blob;             // full-turn audio
     mimeType: string;       // "audio/webm;codecs=opus" or equivalent
     durationMs: number;
     score: number;          // speaker-verification score
     threshold: number;      // configured threshold at rejection
     createdAt: number;      // Date.now()
     status: "idle" | "retrying" | "failed";
     errorMessage?: string;
   }
   ```
3. `VoiceProvider` interface gains two required methods:
   `getLastTurnAudio(): LastTurnAudio | null` and
   `disposeLastTurn(): void`. Both `WhisperProvider` and
   `VoiceStreamProvider` implement them.
4. `useVoiceInput` exposes `retryWithoutFilter()` and
   `dismissRejection()` actions. `retryWithoutFilter` transitions
   `rejectedAudio.status` to `"retrying"`, POSTs
   `/voice/transcribe?skip_speaker_verification=true`, on success
   inserts the transcript via the same `onTranscript` path as
   normal, then calls `disposeRejection()`. On failure, sets
   `status: "failed"` with an error message; user can retry again
   or dismiss.
5. `VoiceRejectionBanner` renders in `Workspace.tsx` whenever
   `rejectedAudio !== null`. It shows:
   - The rejection reason (score vs threshold, duration).
   - A primary button "Transcribe anyway".
   - A secondary button "Dismiss".
   - While retrying: spinner + disabled buttons.
   - On retry failure: error text + the primary button relabeled
     "Retry" (still posts with bypass); dismiss remains available.
6. Server accepts `skip_speaker_verification=true` on
   `/voice/transcribe`; bypasses the gate for that request only;
   increments `voice_skip_verification_total` counter; logs a
   structured line `voice[%s] speaker verification bypassed`.
   Every other aspect of the endpoint is unchanged.
7. Only the most recent rejection is retained. Arrival of a new
   rejection disposes the previous one (blob garbage-collected;
   React state replaced atomically).
8. Rejection blob retention is capped at 5 minutes; a timer
   disposes stale rejections. Rationale: protects against a user
   forgetting the banner and accumulating memory over a long
   session.

## 8. Implementation Strategy (Phased)

Each phase ends green: typecheck, unit tests, lint. Later phases
depend on earlier phases' types and provider methods.

### Phase 0 — Preconditions and test scaffolding

- [ ] Add failing tests asserting the end state:
  - `ui/src/hooks/__tests__/useVoiceInput.retry.test.ts` (new):
    rejection populates `rejectedAudio`; `dismissRejection`
    clears it; `retryWithoutFilter` POSTs with the bypass flag
    and inserts transcript on success.
  - `ui/src/hooks/voice/__tests__/WhisperProvider.retention.test.ts`
    (new): `getLastTurnAudio` returns the last turn's blob until
    `disposeLastTurn` is called.
  - `ui/src/hooks/voice/__tests__/VoiceStreamProvider.retention.test.ts`
    (new): after a multi-segment turn with mixed accept/reject,
    `getLastTurnAudio` returns a single concatenated blob of the
    full turn.
  - `api/voice_transcribe_test.go` (extend): with
    `skip_speaker_verification=true`, server calls Whisper even
    when the decision would be `Allowed=false`; counter
    increments.

**Exit criteria:** new tests exist and fail with clear messages.

### Phase 1 — Provider retention seams

- [ ] `ui/src/hooks/voice/types.ts`:
  - Add `interface LastTurnAudio { blob: Blob; mimeType: string;
    durationMs: number; capturedAt: number; }`.
  - Add required methods to `VoiceProvider`:
    `getLastTurnAudio(): LastTurnAudio | null;
    disposeLastTurn(): void;`
  - `WebSpeechProvider` (which does not record audio) implements
    both trivially: `getLastTurnAudio` returns `null`;
    `disposeLastTurn` is a no-op. (WebSpeechProvider cannot
    retry; the UI will not offer the button when the provider
    reports `null`.)
- [ ] `ui/src/hooks/voice/WhisperProvider.ts`:
  - Replace the `chunks = []` reset at line 62 with a move to
    `lastTurn: LastTurnAudio | null`.
  - Implement `getLastTurnAudio` and `disposeLastTurn`.
  - `disposeLastTurn` must set `lastTurn = null` (dropping the
    last reference to the blob; blob is eligible for GC).
  - On new `start()`, dispose any previous `lastTurn` before
    beginning.
- [ ] `ui/src/hooks/voice/VoiceStreamProvider.ts`:
  - Accumulate every recorded chunk (all segments, accepted and
    rejected) into a single turn-scoped buffer.
  - Turn boundary = between `start()` and either
    `stop()`/`flush()` or external disposal.
  - On turn end, build the concatenated `Blob` once and store as
    `lastTurn`.
  - Implement the same two methods as the Whisper provider.
- [ ] Tests from Phase 0 turn green for the two retention test
  files.

**Exit:**
- `grep -n "chunks\s*=\s*\[\]" ui/src/hooks/voice/` shows exactly
  one site per provider (the `start()` pre-clear), not the
  post-stop reset.
- New provider tests green.

### Phase 2 — State shape + hook actions

- [ ] `ui/src/hooks/useVoiceInput.ts`:
  - Replace `speakerNotice: string | null` with
    `rejectedAudio: VoiceRejection | null` in
    `VoiceInputState`.
  - Delete `speakerNoticeTimerRef` and every usage
    (line 467, 568-570, 710-712, 1013). No timer, no auto-clear.
  - In `onSegmentRejected` handler:
    - Call `provider.getLastTurnAudio()`; if null, set
      `rejectedAudio` to a minimal record with `blob: null` flag
      (cannot retry) — see §9.3; if non-null, build the full
      `VoiceRejection`.
    - Dispose any previous `rejectedAudio.blob` by calling
      `provider.disposeLastTurn()` for the **previous**
      rejection's provider instance (providers are per-session,
      so this is straightforward: disposal runs on the current
      provider).
    - Replace state atomically.
  - Add action `dismissRejection()`: set `rejectedAudio: null`
    and call `provider.disposeLastTurn()`.
  - Add action `retryWithoutFilter()`:
    - Guard: `rejectedAudio && rejectedAudio.status !==
      "retrying"`.
    - Set `status: "retrying"`.
    - POST `/voice/transcribe?skip_speaker_verification=true`
      with `FormData` containing `audio = rejectedAudio.blob`.
    - On 200 and non-empty text: emit via the normal
      `onTranscript` path (same callback as a successful
      transcription); then `dismissRejection()`.
    - On 200 and empty text: set `status: "failed",
      errorMessage: "No speech detected"`; retain blob.
    - On non-2xx: set `status: "failed"` with the error body
      (truncated to 200 chars).
  - Add 5-minute disposal timer: a single `setTimeout` keyed to
    `rejectedAudio.id`; on fire, runs `dismissRejection()`.
    Replaced on new rejection.
- [ ] `ui/src/components/Workspace.tsx`:
  - Delete the existing banner render at 971-974.
  - Render `<VoiceRejectionBanner>` when
    `voiceInput.rejectedAudio !== null`.

**Exit:**
- `grep -n "speakerNotice\|speakerNoticeTimerRef" ui/src/` returns
  zero matches.
- Hook-level tests from Phase 0 green for the retry action.

### Phase 3 — Banner component

- [ ] `ui/src/components/VoiceRejectionBanner.tsx` (new):
  - Props: `{ rejection: VoiceRejection; onRetry: () => void;
    onDismiss: () => void; }`.
  - Renders a top-of-conversation banner (same site as the old
    speaker notice).
  - Status layout:
    - `idle`: explanatory text + Transcribe-anyway + Dismiss.
    - `retrying`: spinner + disabled buttons.
    - `failed`: error text + Retry + Dismiss.
  - Shows `durationMs` (formatted) and score/threshold for user
    context.
  - When `rejection.blob === null` (WebSpeechProvider or
    mid-stream disposal edge case), hides the Transcribe-anyway
    button and shows only Dismiss; message explains "audio not
    retained for this provider."
- [ ] Tests:
  - `ui/src/components/__tests__/VoiceRejectionBanner.test.tsx`
    (new): renders all three states; button handlers fire;
    no-blob variant hides retry button.

**Exit:** banner tests green; manual `pnpm run dev` smoke is
not required (plan uses automated tests only).

### Phase 4 — Server bypass

- [ ] `api/voice_transcribe.go`:
  - Parse `skip_speaker_verification` from query string. Accept
    only `"true"`; any other value (including empty, `"1"`,
    `"yes"`) is treated as false. Rationale: explicit, typo-safe.
  - If true, skip the `evaluateSpeakerVerification` call;
    proceed directly to Whisper.
  - Increment `voice_skip_verification_total` counter each time
    the bypass is taken.
  - Emit structured log: `voice[session_id] speaker verification
    bypassed duration_ms=<N>`.
  - Existing `Allowed=false` handling is untouched for the
    non-bypass case.
- [ ] `api/voice_config.go`: register
  `voice_skip_verification_total` in the metrics registry (same
  pattern as existing voice metrics).
- [ ] Tests: Phase 0 server tests go green. Also verify:
  - `skip_speaker_verification=1` is **not** treated as true
    (only `"true"`).
  - Counter increments only when the bypass actually ran
    (otherwise stays flat).

**Exit:**
- `grep -n "skip_speaker_verification" api/` shows exactly one
  handler path.
- `GET /metrics | grep voice_skip_verification_total` shows the
  counter at 0 after cold start.

### Phase 5 — Integration and observability

- [ ] Frontend integration test
  `ui/src/__tests__/voice-rejection-retry.test.tsx` (new):
  mounts Workspace with a fake provider; simulates a turn
  ending in rejection; banner appears; clicking "Transcribe
  anyway" posts to a mock server with the bypass flag; on
  success, transcript appears in the input; banner closes.
- [ ] Debug window: extend the existing `__wc_voice_debug`
  surface (or create it if missing at
  `ui/src/hooks/voice/debug.ts`) with `lastRejection` and
  `lastRetryOutcome` fields.
- [ ] Docs updated:
  - `docs/internal/SEAMS.md`: VoiceProvider `getLastTurnAudio`
    / `disposeLastTurn` seams.
  - `docs/concepts/ARCHITECTURE.md` (voice-input section if
    present): new `rejectedAudio` state, banner flow, server
    bypass contract.
  - `docs/internal/ERROR-SEMANTICS.md`: `skip_speaker_verification`
    bypass semantics; metric name; structured log key.

**Exit:**
- `pnpm test` and `go test ./...` green.
- Docs reflect the new contract.

## 9. Contract Decisions

### 9.1 `VoiceProvider` interface (TS)

Every provider implements:

```ts
interface VoiceProvider {
  // ... existing methods
  getLastTurnAudio(): LastTurnAudio | null;
  disposeLastTurn(): void;
}

interface LastTurnAudio {
  blob: Blob;
  mimeType: string;
  durationMs: number;
  capturedAt: number; // epoch ms
}
```

- Provider retains the most recent turn's audio until
  `disposeLastTurn()` is called, or a new `start()` begins (which
  first disposes).
- Providers that cannot produce a blob (e.g.,
  `WebSpeechProvider`) return `null` from `getLastTurnAudio`
  always; `disposeLastTurn` is a no-op.

### 9.2 `VoiceInputState.rejectedAudio`

Single slot, not a queue. When a new rejection arrives:

1. Call `provider.disposeLastTurn()` for the **current**
   provider — this is the provider that produced the new
   rejection, and calling dispose **before** the provider's
   capture of the new turn's blob is invalid — so the correct
   order is:
   - Capture new rejection's `LastTurnAudio`.
   - Build new `VoiceRejection`.
   - Replace state (old blob drops last reference; GC eligible).
   - Do **not** call `disposeLastTurn` here — the provider owns
     the reference until the user dismisses or the disposal
     timer fires; only then call `disposeLastTurn`.

   This ordering avoids a double-reference bug (both state and
   provider holding the blob indefinitely). The provider
   releases on dispose; state releases on `null` assignment.

### 9.3 Provider without retained audio

If `provider.getLastTurnAudio()` returns `null` at the time of
rejection (Web Speech provider; or edge-case disposal), the
rejection is still surfaced via the banner — but with
`blob: null` and the Transcribe-anyway button hidden. Rationale:
the user still gets the explanation; they just can't retry
without re-recording. Type uses a discriminated union to
guarantee this at compile time:

```ts
type VoiceRejection =
  | { id; kind: "retryable"; blob: Blob; /* ... */ }
  | { id; kind: "explanatory"; reason: string; /* ... */ };
```

### 9.4 Server bypass parameter

- Endpoint: `POST /voice/transcribe`.
- Query: `skip_speaker_verification=true` (exact string match).
- Auth: the endpoint's existing session-level auth applies
  unchanged. If an unauthenticated or wrongly-scoped call hits
  `/voice/transcribe`, it's rejected as today — the bypass does
  not weaken auth.
- Rate-limiting: the endpoint's existing rate limit applies.
  Bypass does not uncap.
- Observability: `voice_skip_verification_total` counter +
  structured log line per bypass.
- Server does **not** persist any knowledge that the bypass was
  used; downstream consumers of the transcript cannot tell a
  bypassed result from a non-bypassed one. This is intentional —
  the transcript is the transcript.

### 9.5 Retention TTL

- 5 minutes from `createdAt`.
- After TTL, `dismissRejection()` fires.
- Rationale: long enough for a distracted user to come back;
  short enough that a forgotten banner doesn't pin a few MB of
  audio for the whole session.

### 9.6 Retry error UX

- Server returns 500 / 4xx → status `failed`, errorMessage from
  response body (truncated).
- Server returns 200 with empty text → status `failed`,
  errorMessage `"No speech detected in audio"`.
- Network error → status `failed`, errorMessage `"Network
  error"`.
- User can retry as many times as they want while the blob is
  retained. Each retry is an independent POST.

## 10. Testing Plan (Automated)

All verification is automated. No manual test steps.

### 10.1 Frontend unit tests (Vitest)

`ui/src/hooks/voice/__tests__/WhisperProvider.retention.test.ts`
(new):

- Record a turn → `getLastTurnAudio()` returns a non-null blob
  of the expected total byte size.
- `disposeLastTurn()` → subsequent `getLastTurnAudio()` returns
  null.
- Second `start()` without explicit dispose → previous turn's
  audio replaced; `getLastTurnAudio()` returns the new turn's
  audio only.

`ui/src/hooks/voice/__tests__/VoiceStreamProvider.retention.test.ts`
(new):

- Multi-segment turn: 3 segments, 1 rejected, 2 accepted →
  `getLastTurnAudio()` blob contains the concatenation of all 3
  segments' raw audio (verified by byte-length summation).
- Turn ends mid-segment → last partial segment is still
  included.
- `disposeLastTurn()` clears.

`ui/src/hooks/__tests__/useVoiceInput.retry.test.ts` (new):

- Rejection arrives → `rejectedAudio !== null` with correct
  blob, score, threshold.
- `dismissRejection()` → `rejectedAudio === null`; provider's
  `disposeLastTurn` called exactly once.
- `retryWithoutFilter()` → POSTs with
  `skip_speaker_verification=true`; on 200 success, transcript
  delivered via onTranscript, rejection cleared, provider
  disposed.
- `retryWithoutFilter()` on server 500 → status transitions
  `idle → retrying → failed`; blob retained; user can retry.
- Second rejection arrives while first is in state → first's
  blob is replaced (only one slot); no memory growth.
- 5-minute disposal timer fires → `rejectedAudio === null`.

### 10.2 Component test (Vitest + React Testing Library)

`ui/src/components/__tests__/VoiceRejectionBanner.test.tsx`
(new):

- Renders with `kind: "retryable"` → shows Transcribe-anyway
  and Dismiss buttons.
- Renders with `kind: "explanatory"` → shows only Dismiss.
- Click Transcribe-anyway → `onRetry` called.
- Click Dismiss → `onDismiss` called.
- Renders with `status: "retrying"` → buttons disabled;
  spinner visible.
- Renders with `status: "failed"` → error text visible, Retry
  and Dismiss both enabled.

### 10.3 Integration test (Vitest + RTL)

`ui/src/__tests__/voice-rejection-retry.test.tsx` (new):

- Mount Workspace with a fake `WhisperProvider` and mock fetch.
- Simulate end-of-turn with server rejection.
- Banner appears.
- Click Transcribe-anyway. Mock server returns 200 with text.
- Banner closes. Text appears in the input box (via the
  existing onTranscript path).
- Assert fetch was called with URL containing
  `skip_speaker_verification=true`.

### 10.4 Backend tests (Go)

`api/voice_transcribe_test.go` (extend):

- Request with `skip_speaker_verification=true`: verification
  gate is bypassed; Whisper is invoked; response contains the
  transcript; counter `voice_skip_verification_total`
  incremented.
- Request with `skip_speaker_verification=false` (or omitted):
  existing behavior; counter not incremented.
- Request with `skip_speaker_verification=1`, `yes`, `TRUE`,
  `true ` (trailing space): treated as false (strict equality).
- Request with the bypass flag + a session that has no enrolled
  speaker: behavior matches non-bypass of the same session
  (Whisper runs; no-op filter).
- Counter is an int64 exposed at `/metrics` (existing
  registration pattern).

### 10.5 Static-assertion tests

`api/greenfield_assertions_test.go` (extend) or
`api/voice_assertions_test.go` (new):

- `TestAssertSkipVerificationIsStrictTrue` — parses the handler
  source; fails if `skip_speaker_verification` is matched via
  anything other than exact `"true"` comparison.
- `TestAssertNoSpeakerNoticeTimer` — greps `ui/src/` for
  `speakerNoticeTimerRef`; fails on any match (legacy is gone).
- `TestAssertVoiceProviderHasRetentionMethods` — parses
  `ui/src/hooks/voice/types.ts`; fails if the interface lacks
  `getLastTurnAudio` or `disposeLastTurn`.

## 11. Rollout / Validation Checklist

- [ ] All Phase 0–5 tests green.
- [ ] `cd scenarios/web-console/ui && pnpm test` green.
- [ ] `cd scenarios/web-console/ui && pnpm run typecheck` green.
- [ ] `cd scenarios/web-console/ui && pnpm run lint` green
  (all issues in modified files fixed, including pre-existing —
  per `feedback_planning_guidelines.md`).
- [ ] `cd scenarios/web-console/api && go build ./... && go test ./... -timeout 300s` green.
- [ ] `cd scenarios/web-console/api && gofumpt -l .` shows no
  diff; `golangci-lint run ./...` clean.
- [ ] Greenfield static-assertion tests (§10.5) green.
- [ ] Docs updated (`SEAMS.md`, `ARCHITECTURE.md`,
  `ERROR-SEMANTICS.md`).
- [ ] `grep -rn "speakerNotice" scenarios/web-console/ui/src/`
  returns zero matches in source; only `rejectedAudio` exists.
- [ ] Scenario restart is user-driven. Plan writes code only.
  User runs `vrooli scenario restart web-console` after the
  last phase (per `feedback_no_restart_active_scenario.md` and
  `feedback_use_vrooli_scenario_restart.md`).
- [ ] After user restart:
  `curl -s http://localhost:<port>/metrics | rg voice_skip_verification_total`
  shows the counter at 0.

## 12. Risks + Mitigations

| Risk | Mitigation |
|---|---|
| Retained blob leaks memory if user ignores banner in a long session. | 5-minute TTL auto-dispose (§9.5). Single-slot retention caps memory at one turn's audio. |
| `VoiceStreamProvider` full-turn concatenation changes memory profile on streaming path. | Turn length p99 is well under the buffer caps already in place for the streaming provider. Add a test asserting an upper bound on retained buffer size (e.g., 10 MB). On overflow, drop the oldest segment's audio from retention (still surface banner with `kind: "explanatory"`). |
| Server bypass parameter gets cached by intermediaries (CDN/proxy). | The endpoint is a `POST` to a private origin with auth headers. No public cacheability. Verified by response `Cache-Control: no-store` already emitted by the handler; if not, add it in the same change. |
| User repeatedly clicks Transcribe-anyway during a retry. | `retryWithoutFilter` guards on `status === "retrying"` and no-ops. Button is also visually disabled during retrying. |
| Bypass parameter abused to systematically skip verification. | Bypass requires the same session auth; rate limits apply. Counter + structured log make mass-bypass usage immediately visible. The design choice is: speaker verification is an assistive filter, not a security boundary. |
| Rejection of a Web Speech provider turn (no blob) surprises the user with a Dismiss-only banner. | Banner copy explicitly states the limitation for that provider. Settings UI is where users configure providers; this plan does not hide the provider choice. |
| A new rejection arrives exactly as a retry is in-flight. | The in-flight retry completes (transcript delivered or failure recorded) before any new rejection's blob replaces state. Because each retry uses its own blob reference captured at action time, the in-flight promise is safe even after state replacement. |
| Tests flake on `Blob` / `MediaRecorder` in jsdom. | Vitest helper for mocking `MediaRecorder` already exists in this codebase (see `useVoiceInput.test.ts` imports); extend it rather than reinvent. |
| 5-minute TTL removes a rejection right before the user clicks. | Acceptable; banner shows a fading indicator at 4:30 ("Audio expires in 30 s"). Configurable via a single constant; not an env var (greenfield). |

## 13. Non-goals / Prohibited Patterns

- **No compatibility layers.** The old `speakerNotice: string |
  null` shape is removed wholesale; every reader is updated in
  the same change.
- **No feature flag** for the persistent banner, the retention
  behavior, or the server bypass. All three are all-on
  immediately.
- **No wrappers** or shims around `VoiceProvider` to preserve
  pre-retention callers.
- **No manual test checklists** replacing automated verification.
  (Honors `feedback_testing_over_manual.md`.)
- **No in-UI banner auto-dismissal** for the rejection state.
  Dismissal is explicit (click Dismiss, successful retry, or TTL
  expiry). No "animate out after N seconds" behavior.
- **No silent retry** without user action. The user must press
  Transcribe-anyway. Rationale: the first pass decided the audio
  shouldn't be transcribed; the retry is a deliberate override
  by the user.
- **No second endpoint** like `/voice/transcribe-bypass`. One
  endpoint, one strict query parameter.
- **No server-side heuristic** that flips
  `skip_speaker_verification` based on score proximity to
  threshold. The bypass is user-driven only.
- **No commented-out code.** Deleted paths are deleted.
- **No `TODO` / `FIXME` / `XXX` / `HACK` comments for
  follow-ups.** Follow-ups become tasks.
- **No emojis** in source, comments, or tests.
- **No git commits, reverts, or resets** by the implementing
  agent. (Honors `feedback_no_git_mutations.md`.)
- **No scenario restart** by the implementing agent. User
  restarts. (Honors `feedback_no_restart_active_scenario.md`.)
- **No queue of past rejections.** One slot only. Older
  rejections are dropped.

## 14. Definition of Done

All must be true:

1. §11 rollout checklist fully green.
2. §10.5 static-assertion tests enforce in CI:
   (a) `speakerNoticeTimerRef` is gone,
   (b) `VoiceProvider` interface has retention methods,
   (c) the server bypass accepts only `"true"` exactly.
3. `grep -rn "speakerNotice" scenarios/web-console/ui/src/`
   returns zero matches.
4. `grep -rn "setTimeout.*speakerNotice\|speakerNoticeTimer"
   scenarios/web-console/ui/src/` returns zero matches.
5. Retry integration test (§10.3) passes: rejection → banner →
   Transcribe-anyway → transcript in input → banner gone.
6. Memory retention test (§10.1) passes: after dispose, the blob
   is no longer reachable from provider state or hook state.
7. Server bypass counter (§9.4) wired to `/metrics` and
   incrementing in the corresponding test.
8. Banner component (§10.2) covers all three states
   (`idle`, `retrying`, `failed`) and both kinds
   (`retryable`, `explanatory`).
9. Docs (`SEAMS.md`, `ARCHITECTURE.md`, `ERROR-SEMANTICS.md`)
   updated with the new seams, flow, and semantics.
10. No deprecated, compat, legacy, migration-bridge, or fallback
    code exists in any file touched by this plan.

---

**Plan file:** `scenarios/web-console/docs/plans/stt-voice-filter-retry-implementation-plan.md`
**Owner:** unassigned (claim via TaskUpdate).
**Created:** 2026-04-23.
