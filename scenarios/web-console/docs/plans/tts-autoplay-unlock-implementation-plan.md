---
title: TTS Autoplay Unlock + Enable-Audio Affordance
status: draft
scope: scenarios/web-console
---

# TTS Autoplay Unlock + Enable-Audio Affordance — Implementation Plan

## 1. Purpose

Auto-TTS (automatic read-out of new assistant messages) is failing the
vast majority of the time because the browser's autoplay policy is
blocking `HTMLAudioElement.play()` on the Kokoro provider's audio
element. Summarization and synthesis both succeed. The failure is at
playback, surfaces as a rapidly-flickering amber `tts-error` banner,
and silently swallows the audio the user expected to hear.

This plan lands two complementary fixes:

- **A (primary)** — proactively unlock the Kokoro provider's reusable
  `HTMLAudioElement` on the first real user gesture, by running a
  zero-audible silent `play()` on that element from within the gesture
  stack, so all subsequent auto-TTS plays inherit the sticky
  activation.
- **B (fallback)** — when auto-TTS is rejected with a
  `NotAllowedError` and the unlock has not fired, show a single
  persistent "Enable voice" affordance. On click, it performs the
  unlock and replays the pending event. Never shows again once unlock
  succeeds.

Plus: kill the banner flicker at the source, so even if unlock fails,
the UI no longer oscillates 15+ times per message.

## 2. Required Reading

```bash
prompt-manager skill read cli-steer api-steer utils-unification seam-discovery-and-enforcement
```

No domain-specific skills were materially relevant (skill discovery
returned only `browser-automation-studio` and `video-studio`, neither
of which apply to web-console TTS plumbing).

## 3. Greenfield Constraint (hard rule)

**This is greenfield work.** Do not add compatibility shims, legacy
wrappers, `// removed` comments, renamed `_unused` variables, or dual
code paths. If an existing function's behavior changes, change it in
place. Delete code that is no longer reachable.

## 4. Problem Statement

### 4a. Evidence from live logs

From `vrooli.develop.web-console.start-api.log` (current session):

```
2026/04/24 14:04:21 tts-summarize: path=auto event=629a608d… in=2667 out=143
                    ratio=0.05 ms=325 done_reason=stop eval=32
2026/04/24 14:06:36 tts-playback: source=terminal_auto stage=error backend=kokoro
                    message=The request is not allowed by the user agent
                    or the platform in the current context…
 (× 15 entries over ~1 second)
```

- 1 successful auto-summarize
- 15 successful `POST /tts/synthesize`
- 15 playback rejections with the browser autoplay error string

### 4b. Why the banner "flickers"

`useTextToSpeech.speakParagraphs` is called once per paragraph in
auto-TTS. Each call:

1. Clears error state — `setState(s => ({ ...s, error: null }))` at
   `useTextToSpeech.ts:402`, `:445`, `:550`.
2. Calls `provider.speakFromBlob(blob)` → `audio.play()` rejects.
3. Sets `error: message` — `useTextToSpeech.ts:425`, `:495`, `:563`.
4. Next paragraph repeats from step 1.

The `tts-error` banner in `TerminalPane.tsx:790-794` is gated on a
`showTtsError` state derived from `ttsError` with a 5s auto-dismiss
(`TerminalPane.tsx:210-219`). The repeated set/clear of `error`
produces the observed flicker; the 5s timer explains why it stops
after ~5-10s even when playback failures continue.

### 4c. Why on-demand works

The user's click on `PlaybackModeControl` or any playback button is
a qualifying user gesture. The `audio.play()` call runs inside that
gesture's call stack, so the autoplay policy admits it.

### 4d. Why existing gesture listeners don't help

`useTextToSpeech.ts:99-113` attaches `pointerdown`/`keydown`/
`touchstart` listeners that set `audioUnlockedRef.current = true`.
This is a **flag only** — it never actually calls `play()` on the
Kokoro provider's `HTMLAudioElement`. The browser's autoplay policy
is not satisfied by a past gesture; it requires either a live gesture
in the call stack **or** a prior successful `play()` on the same
media element.

## 5. Scope

### 5a. In scope

- `scenarios/web-console/ui/src/hooks/useTextToSpeech.ts`
- `scenarios/web-console/ui/src/hooks/tts/KokoroProvider.ts`
- `scenarios/web-console/ui/src/hooks/tts/BrowserTTSProvider.ts`
  (read-only — we may need to expose a similar unlock method)
- `scenarios/web-console/ui/src/hooks/tts/types.ts`
  (extend `TTSProvider` interface with an `unlock()` method)
- `scenarios/web-console/ui/src/components/TerminalPane.tsx`
  (remove the flickering `tts-error` banner or replace with
  sticky-error behavior)
- `scenarios/web-console/ui/src/components/Workspace.tsx`
  (mount new `EnableAudioBanner` when appropriate)
- **NEW** `scenarios/web-console/ui/src/components/EnableAudioBanner.tsx`
- Unit tests under `components/__tests__/` and `hooks/__tests__/`

### 5b. Out of scope

- Backend summarization changes (already works).
- Browser TTS (SpeechSynthesis) autoplay handling — SpeechSynthesis
  has its own gating that is generally less strict; if we find it
  fails similarly, file a follow-up.
- The unrelated `voiceInput.fallbackNotice` banner
  (`Workspace.tsx:1046-1050`) — that's STT input, not TTS output.
- Auto-summarize backend logic.
- Pre-synthesis caching (`tts-audio-precache` — separate plan).

## 6. Current Technical Context

### 6a. Failure surface

- `KokoroProvider.ts:63` — `speak()` → `this.audio.play().catch(reject)`
- `KokoroProvider.ts:120` — `speakSequence()` → same pattern
- `KokoroProvider.ts:155` — `speakFromBlob()` → same pattern
- `useTextToSpeech.ts:451-471` — cache-first path calls
  `provider.speakFromBlob(blob)` directly and **does not fall back**
  to the browser backend on failure (unlike `executeSpeak` at
  `:192-207`).

### 6b. Reusable audio element

`KokoroProvider` creates a single `this.audio = new Audio()` in its
constructor (line 31). This is the unlock target — one successful
`play()` on this element within a gesture makes subsequent plays on
the **same element** (with different `src`) allowed even after the
gesture expires.

### 6c. Auto-TTS trigger

Two paths, both in `TerminalPane.tsx`:

- On-arrival (WebSocket event): `handleConversationEvent`
  (`:230-285`) calls `speakParagraphs` when
  `autoTtsEnabled && activePane && event.role === "assistant"`.
- Catch-up on pane focus: effect at `:308-344` replays pending
  assistant events when the user refocuses the pane.

Both paths hit the same failure mode.

## 7. Target End State

1. User opens web-console and sends a prompt. Pressing a key or
   clicking anywhere unlocks the Kokoro audio element silently —
   they hear nothing, but the provider is now activated.
2. Assistant response arrives. Auto-TTS runs. Playback succeeds on
   the first paragraph and every subsequent one.
3. In the edge case where no gesture has happened before the first
   playback attempt (or unlock fails), the user sees **one**
   persistent "Enable voice for this session" button in the
   messages pane area. Clicking it unlocks and replays the
   pending event. The button does not reappear.
4. The amber `tts-error` banner at the top of the terminal pane
   no longer flickers: transient playback errors either don't
   surface there at all, or render once and stay until dismissed
   or superseded.
5. The `tts-playback stage=error backend=kokoro` log entries with
   the autoplay-policy message disappear from the server log for
   the common case.

## 8. Implementation Strategy

### Phase 1 — Provider `unlock()` method (primary change)

**File:** `scenarios/web-console/ui/src/hooks/tts/types.ts`

Extend `TTSProvider` interface:

```ts
export interface TTSProvider {
  // …existing
  /**
   * Attempt to unlock audio playback by running a silent play() from
   * within a user-gesture call stack. Resolves `true` if the
   * underlying media element is now activated, `false` if the
   * browser rejected the play (caller should then show the
   * enable-audio affordance).
   */
  unlock(): Promise<boolean>;
}
```

**File:** `scenarios/web-console/ui/src/hooks/tts/KokoroProvider.ts`

Add:

```ts
private unlocked = false;

async unlock(): Promise<boolean> {
  if (this.unlocked) return true;
  // ~80-byte silent WAV (1 sample, RIFF header). Using a data URL
  // keeps this inline and cheap — no asset loading, no network.
  const SILENT_WAV =
    "data:audio/wav;base64,UklGRiQAAABXQVZFZm10IBAAAAABAAEARKwAAIhYAQACABAAZGF0YQAAAAA=";
  try {
    this.audio.src = SILENT_WAV;
    await this.audio.play();
    this.audio.pause();
    this.audio.removeAttribute("src");
    this.audio.load();
    this.unlocked = true;
    return true;
  } catch {
    return false;
  }
}

isUnlocked(): boolean { return this.unlocked; }
```

**File:** `scenarios/web-console/ui/src/hooks/tts/BrowserTTSProvider.ts`

Implement `unlock()` as a no-op returning `true` (SpeechSynthesis
doesn't gate this way; if we discover otherwise during validation,
file a follow-up — don't build speculative plumbing now).

### Phase 2 — Wire unlock into the gesture listeners

**File:** `scenarios/web-console/ui/src/hooks/useTextToSpeech.ts`

Replace the flag-only gesture handler at `:99-113` with a handler
that calls `provider.unlock()` synchronously in the gesture stack:

```ts
useEffect(() => {
  if (typeof window === "undefined") return;
  const unlock = () => {
    const p = providerRef.current;
    if (!p) {
      audioUnlockedRef.current = true;
      setState((s) => ({ ...s, browserAudioReady: true }));
      return;
    }
    // Must run synchronously inside the gesture call stack.
    // Don't await — fire-and-forget, update state on settle.
    p.unlock().then((ok) => {
      if (ok) {
        audioUnlockedRef.current = true;
        setState((s) => ({ ...s, browserAudioReady: true }));
      }
    });
  };
  window.addEventListener("pointerdown", unlock, { passive: true });
  window.addEventListener("keydown", unlock, { passive: true });
  window.addEventListener("touchstart", unlock, { passive: true });
  return () => {
    window.removeEventListener("pointerdown", unlock);
    window.removeEventListener("keydown", unlock);
    window.removeEventListener("touchstart", unlock);
  };
}, []);
```

Edge case — the provider is set asynchronously after backend
detection. If a gesture happens before the provider exists, we fall
back to flag-only behavior; the next gesture after provider mount
performs the real unlock. Do not bind the effect to `state.backend`
— that would re-attach listeners on every backend change and miss
early gestures.

### Phase 3 — Detect `NotAllowedError` and expose `needsUnlock`

**File:** `scenarios/web-console/ui/src/hooks/useTextToSpeech.ts`

Add a new state field `needsUnlock: boolean` (default `false`).

In both error handlers (`speak` at `:411-432` and `speakParagraphs`
at `:485-496`), classify the error:

```ts
function isAutoplayBlocked(err: unknown): boolean {
  if (!(err instanceof Error)) return false;
  return (
    err.name === "NotAllowedError" ||
    /not allowed by the user agent/i.test(err.message)
  );
}
```

On `NotAllowedError`:
- Do NOT set `error: message` (prevents the banner flicker).
- Set `needsUnlock: true`.
- Emit a distinct diagnostics event (`stage: "autoplay_blocked"`)
  so the server log can distinguish this from real synthesis
  errors.

Any successful unlock or successful play clears `needsUnlock` back
to `false`.

### Phase 4 — Stop the error reset per paragraph

**File:** `scenarios/web-console/ui/src/hooks/useTextToSpeech.ts`

Remove `error: null` from the "attempt" setStates at `:402`, `:445`,
`:550`. Instead, clear `error` only when a play actually succeeds
(`updateSuccess` already does this at `:133`). This makes errors
sticky across the paragraph chain and kills the flicker even for
non-autoplay errors.

### Phase 5 — Cache-first path falls back on autoplay block

**File:** `scenarios/web-console/ui/src/hooks/useTextToSpeech.ts`

At `:451-471` the cache-first path calls `provider.speakFromBlob`
directly and the outer `catch` at `:485-496` just sets error state.
Add the same fallback-to-browser logic present in `executeSpeak` at
`:198-205`: if the error is autoplay-blocked AND preference is
`auto` AND the browser backend is supported, retry via
`runBrowserSpeak(paragraphs.join("\n\n"))`. If the browser backend
also fails, then set `needsUnlock`.

### Phase 6 — `EnableAudioBanner` component

**New file:** `scenarios/web-console/ui/src/components/EnableAudioBanner.tsx`

Minimal, persistent affordance. Amber, top-of-pane, one button
("Enable voice") + dismiss X. Testids:
`enable-audio-banner`, `enable-audio-banner-enable`,
`enable-audio-banner-dismiss`.

Props:

```ts
interface EnableAudioBannerProps {
  onEnable: () => Promise<void>;   // calls provider.unlock() + replay
  onDismiss: () => void;
}
```

Visual style: match `SummarizeErrorBanner` but distinct copy; not
gated by a timer.

### Phase 7 — Mount the banner

**File:** `scenarios/web-console/ui/src/components/Workspace.tsx`

Add local state `showEnableAudio: boolean` keyed off the new
`needsUnlock` signal from `useTextToSpeech`. Wire via an
`onNeedsUnlock` callback prop from `TerminalPane` → `Workspace`
(same pattern used for `onSummarizeError`).

On "Enable voice" click:
1. Call `ttsProvider.unlock()` (exposed via a new hook return
   value, or via a TerminalPane-exposed imperative handle).
2. If unlock succeeds, find the most recent pending assistant
   event (same logic as the pane-refocus effect at
   `TerminalPane.tsx:308-344`) and replay it.
3. Hide the banner.

On "Dismiss": hide the banner without unlocking; it won't reappear
for the remainder of the session. Next session it can appear again.

### Phase 8 — Remove the flickering banner

**File:** `scenarios/web-console/ui/src/components/TerminalPane.tsx`

Delete lines `208-219` (the `showTtsError` state + 5s auto-dismiss
effect) and `790-794` (the rendering block). Transient `ttsError`
is no longer user-facing; real errors flow through
`needsUnlock` → `EnableAudioBanner` (autoplay case) or through
existing diagnostics (synthesis failures, which are rare and
already surface via the fallback mechanism).

If we later want a persistent TTS-error banner for non-autoplay
failures, add it as a new component — don't resurrect the flickering
one.

### Phase 9 — Cleanup & verification

Per the mandatory convention:

1. `cd scenarios/web-console && npm run lint` — fix all warnings
   in modified files, including pre-existing.
2. `cd scenarios/web-console/ui && npx tsc --noEmit` — fix all
   errors, including pre-existing.
3. `cd scenarios/web-console/ui && npx vitest run` — all tests
   pass.
4. `cd scenarios/web-console/api && go build ./... && go vet ./...`
   (no backend changes expected, but a sanity check).
5. `vrooli scenario restart web-console`
6. Verify health:
   `curl -s http://localhost:<API_PORT>/health` and
   `curl -s -o /dev/null -w '%{http_code}\n' http://localhost:<UI_PORT>/`.
7. Tail `vrooli.develop.web-console.start-api.log` and confirm
   that after a round-trip prompt/response the
   `tts-playback … stage=error … not allowed` lines are absent
   (Phase 1-2 verification). The
   `stage=autoplay_blocked` diagnostic may appear once if the
   gesture hadn't happened before the first message — that's
   expected and proves the new path is engaging.

## 9. Contract Decisions

- `TTSProvider.unlock()` returns `Promise<boolean>`. `true` means
  the media element is activated. `false` means the browser
  rejected the silent play. Callers interpret `false` as "show the
  enable-audio affordance."
- `unlock()` MUST be safe to call multiple times. Second and later
  calls short-circuit on the `unlocked` flag.
- `unlock()` MUST NOT throw. Any error is converted to `false`.
- `useTextToSpeech` exposes `needsUnlock: boolean` and a method
  `unlockAudio(): Promise<boolean>` (wrapper around the current
  provider's `unlock()` with state updates).
- `needsUnlock` transitions:
  - Any successful play → `false`.
  - `NotAllowedError` from any play path → `true`.
  - Successful `unlockAudio()` call → `false`.
  - Explicit dismiss by user → `false` (+ session-scoped suppress
    flag so it doesn't reappear until reload).
- Banner is rendered at Workspace level (not per-pane) because
  unlock is document-wide, not pane-scoped.

## 10. Testing Plan

All automated. No manual checklist.

### 10a. `KokoroProvider.test.ts` (new or extend existing)

- `unlock()` resolves `true` when mocked `audio.play()` resolves.
- `unlock()` resolves `false` when mocked `audio.play()` rejects
  with `NotAllowedError`.
- `unlock()` short-circuits on second call (mock called once).
- `unlock()` swallows unexpected error classes and returns `false`.
- After successful `unlock()`, a subsequent `speakFromBlob()` on
  the same mocked element calls `play()` once and resolves.

### 10b. `useTextToSpeech.test.ts`

- First user gesture triggers `provider.unlock()`.
- `needsUnlock` starts `false`.
- `speakParagraphs` rejected with `NotAllowedError` sets
  `needsUnlock: true` and does NOT set `error`.
- `speakParagraphs` rejected with a generic error sets `error`
  and does NOT set `needsUnlock`.
- Successful play clears `needsUnlock`.
- Cache-first path, on `NotAllowedError` with `backendPreference:
  "auto"` and browser supported, falls back to browser provider
  and does not set `needsUnlock`.
- Error state is NOT reset at the start of each paragraph
  (feed two paragraphs, fail both, assert error set once and
  stays).

### 10c. `EnableAudioBanner.test.tsx` (new)

- Renders nothing when `open` is false.
- Renders enable + dismiss buttons when `open` is true.
- Clicking enable calls `onEnable` exactly once.
- Clicking dismiss calls `onDismiss` exactly once.
- Component is accessible: `role="status"` or similar, aria-label
  on buttons.

### 10d. `Workspace.test.tsx` / `TerminalPane.test.tsx`

- When `onNeedsUnlock(true)` fires, `EnableAudioBanner` is
  mounted with `open=true`.
- Clicking "Enable voice" invokes `unlockAudio` and (on success)
  replays the most recent pending assistant event.
- Dismissing hides the banner and suppresses it for the session
  (re-firing `onNeedsUnlock(true)` after dismiss does NOT
  re-show).
- The old `tts-error` banner testid is absent.

### 10e. Diagnostic event

- `reportTTSEvent` is called with `stage: "autoplay_blocked"`
  exactly once when the autoplay-block is first detected (not
  per paragraph).

## 11. Rollout / Validation Checklist

- [ ] Phase 1-8 implemented
- [ ] `npx vitest run` green in `scenarios/web-console/ui`
- [ ] `npx tsc --noEmit` clean
- [ ] `npm run lint` clean in modified files
- [ ] `go build ./... && go vet ./...` clean in
      `scenarios/web-console/api`
- [ ] `vrooli scenario restart web-console` completes, health
      endpoints return 200
- [ ] After restart, in a fresh browser tab:
      user sends a prompt → clicks anywhere → response arrives →
      audio plays. Server log shows no `stage=error … not allowed`
      entries.
- [ ] In an incognito tab with audio-autoplay disabled at the
      system level: first message triggers the Enable-Audio
      banner; clicking it unlocks and replays.

## 12. Risks & Mitigations

| Risk | Mitigation |
|---|---|
| Silent WAV data URL is rejected by some browsers | Keep the data URL; if it proves problematic, switch to a pre-baked `silent.mp3` asset. Detect in the `unlock()` test matrix. |
| The silent play is audible on some devices (volume-dependent pop) | The WAV is 1 sample. If reports come in, replace with `audio.muted = true` before play() + unmute after — but only if reproducible. |
| `BrowserTTSProvider` actually does need a gesture-time unlock | Plan says no-op; if logs show SpeechSynthesis autoplay failures, file follow-up. Don't pre-build. |
| Gesture fires before provider is mounted | Listener no-ops gracefully; the next gesture after mount unlocks. Worst case, user clicks twice. |
| Replay on "Enable voice" plays the wrong event | Use the exact same "pending assistant events" query as `TerminalPane.tsx:308-313`. Reuse, don't reinvent. |
| Removing the `tts-error` banner hides real synthesis failures | Synthesis failures throw from `synthesizeTTS` and surface via the `reportTTSEvent` diagnostics + provider fallback. No user-facing banner is acceptable because those failures are rare and the fallback path handles them. If user feedback says otherwise, add a dedicated component. |

## 13. Non-Goals / Prohibited Patterns

- Do NOT add backwards-compatibility shims for the old
  `tts-error` banner or the old `audioUnlockedRef` flag-only
  behavior.
- Do NOT keep a dual code path where old gesture listeners live
  alongside the new `unlock()` call.
- Do NOT add a comment-out block for removed code. Delete it.
- Do NOT add a new "TTS failures" catch-all banner to replace
  `tts-error`. If we need one later, design it separately.
- Do NOT couple the unlock to STT input (`voiceInput.fallbackNotice`
  is a different feature — do not touch it here).
- Do NOT introduce a setInterval or 5s timer to re-attempt
  unlock. Unlock is gesture-driven; if no gesture, the banner
  tells the user what to do.

## 14. Definition of Done

1. Provider `unlock()` method exists on `KokoroProvider` (real)
   and `BrowserTTSProvider` (no-op), typed in `TTSProvider`.
2. First real user gesture triggers `unlock()` via
   `useTextToSpeech`. Verified by a unit test.
3. `NotAllowedError` on playback no longer produces a flickering
   `ttsError` — instead it flips `needsUnlock` once. Verified by
   a unit test.
4. `EnableAudioBanner` renders at Workspace level when
   `needsUnlock === true` and un-dismissed. Click enables +
   replays. Click-dismiss hides for session.
5. Old `tts-error` banner and its 5s timer are deleted from
   `TerminalPane.tsx`. No flicker in integration test.
6. All lint, type, and test issues in modified files are fixed,
   including pre-existing ones. 
7. Scenario restarted via `vrooli scenario restart web-console`
   and both API and UI health endpoints return 200.
8. Fresh session sanity: one round-trip message produces
   audible TTS without user clicking any banner (assuming the
   user has clicked *anywhere* between loading the tab and the
   response arriving — which they will have, because they typed
   a prompt).
