# TTS Start-Muted + Explicit Mute Toggle — Implementation Plan

## 1. Purpose
Add a TikTok-style "load muted, tap to unmute" behavior to the web-console TTS audio bar, plus a real persisted mute concept (separate from `volume === 0`) so unmuting restores the user's prior volume cleanly. The speaker icon in the bottom audio bar becomes context-sensitive: when muted it is a single-tap unmute; when unmuted it opens the existing volume/speed popover (which gains a mute control).

## 2. Required Reading
Run before implementing:

```bash
prompt-manager skill read documentation-health test seam-discovery-and-enforcement react-coherence ux api-steer
```

Plus the plan-authoring conventions:

```bash
prompt-manager skill read implementation-plan-authoring plan-skill-discovery
```

## 3. Greenfield Constraint (HARD RULE)
**This is greenfield work.** The web-console scenario currently has no users other than the project owner. Do not add compatibility shims, dual-read fallbacks, deprecated-state mirroring, `// removed` markers, or `_unused` renames. Old behavior (`isMuted = volume === 0`) is removed outright and replaced with the explicit `isMuted` state described below. Tests that asserted the old derivation must be updated, not preserved alongside new ones.

## 4. Problem Statement
Today (`AudioPlayerBar.tsx:92`):
- `isMuted` is a derived computation: `volume === 0`. There is no real mute state.
- The speaker icon has only one role: open the audio popover. There is no one-tap mute/unmute.
- TTS auto-speak (`autoTtsEnabled` in `useWorkspaceStore`) plays at full volume the moment an assistant message arrives after app load — there is no "soft gate" that lets audio queue silently until the user opts in.
- Volume is session-only (not persisted), so toggling mute by setting `volume = 0` loses the user's previous volume choice on unmute.

Desired behavior:
1. App loads with TTS audio muted by default (configurable via a persisted setting that defaults to `true`).
2. The audio bar's speaker icon shows muted state; **single tap unmutes** without opening the popover.
3. Once unmuted, tapping the same icon opens the existing popover (current behavior).
4. Inside the popover, the user can mute again via an explicit mute control next to the volume slider. Re-muting from the popover does not require setting volume to 0.
5. On unmute, the user's prior volume is restored — never silently reset.

## 5. Scope

**In scope:**
- New persisted preference `startMutedOnLoad: boolean` (default `true`) in `useWorkspaceStore`.
- New session-scoped `isMuted: boolean` state (not persisted) wired through the TTS playback path.
- Context-sensitive behavior on the `tts-audio-button` in `AudioPlayerBar.tsx`.
- New mute toggle UI inside `AudioSettingsContent.tsx`.
- New "Start muted on app load" toggle in `TtsSettingsSection.tsx`.
- Updated unit tests for `AudioPlayerBar`, `AudioSettingsContent`, the workspace store, and the settings section.
- Updated integration test `workspace-tts-replay-bar.test.tsx` if it exercises mute paths.

**Out of scope:**
- Changing `autoTtsEnabled` semantics. Auto-speak still gates whether TTS *fires* at all; mute only gates audible output.
- Server-side persistence of `startMutedOnLoad` (it is a client UX preference, not a TTS config field). Do **not** add it to the `updateTTSConfig` API or `TTSConfig` type.
- Mobile-specific haptics or animation polish beyond what already exists for the popover.
- Changing volume persistence semantics (volume remains session-only).

## 6. Current Technical Context

**Files (with anchor line numbers):**
- `scenarios/web-console/ui/src/components/AudioPlayerBar.tsx`
  - `isMuted` derivation: line 92
  - Audio button (icon + popover trigger): lines 179–189
  - Popover renders `AudioSettingsContent`: lines 209–217 (mobile sheet) and 232–240 (desktop popover)
- `scenarios/web-console/ui/src/components/tts/AudioSettingsContent.tsx`
  - Volume slider block: lines 39–63
  - Speed presets block: lines 66–93
- `scenarios/web-console/ui/src/components/settings/TtsSettingsSection.tsx`
  - "Auto-speak AI responses" toggle: lines 195–209 (insert new toggle right after this row)
  - Server-persistence helper `persistTtsConfig`: line 123 (do NOT extend — client-only pref)
- `scenarios/web-console/ui/src/stores/useWorkspaceStore.ts`
  - State shape: lines 61–64 (TTS-related fields)
  - Default values: lines 167–170
  - Setters: lines 280–283
  - Persist `version: 12` and migrate fn: lines 333–387
  - `partialize` whitelist: lines 389–418
- `scenarios/web-console/ui/src/components/Workspace.tsx`
  - `handleTtsSetVolume`: lines 572–574
  - Bar render with `onSetVolume={handleTtsSetVolume}`: line 1316
- `scenarios/web-console/ui/src/hooks/useSessionManager.ts`
  - `setTtsVolumeOnPane`: lines 371–376 (delegates to terminal ref's `setTtsVolume`)
- `scenarios/web-console/ui/src/hooks/useTextToSpeech.ts`
  - Default state init incl. `volume: 1`: line 89
  - `setVolume` provider call site (used by `setTtsVolume` exported through terminal ref)

**Tests that currently encode the old contract (must be updated, not shimmed):**
- `scenarios/web-console/ui/src/components/__tests__/AudioPlayerBar.test.tsx` — esp. line 137 "volume slider changes call onSetVolume" and any test that depends on `volume === 0` to render the muted icon.
- `scenarios/web-console/ui/src/__tests__/workspace-tts-replay-bar.test.tsx` — line 105 mocks `setTtsVolumeOnPane`; will need `setTtsMuted` equivalent.
- `scenarios/web-console/ui/src/__tests__/useWorkspaceStore.test.ts` — version-migration tests must extend to v13.
- `scenarios/web-console/ui/src/__tests__/settings-auto-tts.test.tsx` — used as the structural reference for the new "start muted on load" toggle test.

## 7. Target End State

1. `useWorkspaceStore` holds `startMutedOnLoad: boolean` (persisted, default `true`) with setter `setStartMutedOnLoad`. Persist version bumped to **13** with a v12→v13 migration that defaults the new field to `true`.
2. The TTS playback hook (`useTextToSpeech.ts`) exposes an `isMuted` boolean and `setMuted(next: boolean)` method on its public state/ref interface. Internally, mute is applied by sending `volume = 0` to the active provider when `isMuted` is true, while preserving the user's last non-zero volume in a separate ref/state field for restore. `setMuted(false)` reapplies the preserved volume to the provider.
3. On app load (first time `useTextToSpeech` initializes for a session), if `startMutedOnLoad` is true, `isMuted` initializes to `true`. Volume initial value remains `1` (so unmuting yields full volume immediately, matching user expectation).
4. `AudioPlayerBar.tsx`:
   - Icon: `isMuted ? VolumeX : Volume2` (driven by the new explicit `isMuted` prop, not derived from `volume`).
   - Click handler:
     ```
     if (isMuted) onSetMuted(false);   // single-tap unmute, no popover
     else setShowPopover(p => !p);     // existing behavior
     ```
   - New required prop `isMuted: boolean` and `onSetMuted: (next: boolean) => void`.
5. `AudioSettingsContent.tsx`:
   - New mute control rendered above or beside the volume slider label (e.g., a small icon button in the "Volume" header row). Test id: `${testIdPrefix}-mute-toggle`.
   - When `isMuted` is true the volume slider visually dims (use existing `cn` patterns; reuse `opacity-50 cursor-not-allowed` style already used for disabled scrub in AudioPlayerBar) but remains interactive — moving it auto-unmutes (calls `onSetMuted(false)` then `onVolumeChange(value)`).
   - New props `isMuted: boolean` and `onSetMuted: (next: boolean) => void`. Both required.
6. `TtsSettingsSection.tsx`: new `SettingsRow` for "Start muted on app load" using `SettingsToggle`, placed immediately after the "Auto-speak AI responses" row (after line 209). Wired to `startMutedOnLoad` / `setStartMutedOnLoad` from the store. Hint text: "When enabled, audio is muted on app load. Tap the speaker icon to unmute." No call to `persistTtsConfig` (client-only).
7. `Workspace.tsx`: a new `handleTtsSetMuted` callback parallels `handleTtsSetVolume`, calling a new `setTtsMutedOnPane` from `useSessionManager`. Bar usage updated:
   ```tsx
   <AudioPlayerBar
     ...
     isMuted={ttsState.isMuted}
     onSetMuted={handleTtsSetMuted}
     ...
   />
   ```
8. `useSessionManager.ts`: new `setTtsMutedOnPane(sessionId, next)` that delegates to `terminalRef.setTtsMuted(next)`. Exported from the hook.
9. The audio bar's popover open/close state is unchanged. Only the trigger condition changes.

## 8. Contract Decisions

- **Mute and volume are independent.** Volume is the user's chosen output level (slider value). `isMuted` is an orthogonal boolean. The provider receives `effectiveVolume = isMuted ? 0 : volume`.
- **`startMutedOnLoad` default is `true`.** Justification: the project is greenfield with one user (the project owner), who has explicitly chosen this default. It is the safer / less-surprising behavior when audio capability spans multiple devices including mobile. Document this in a code comment at the default-value site only — do not also document elsewhere.
- **`startMutedOnLoad` is client-only.** Stored in `useWorkspaceStore` localStorage. Not added to the `TTSConfig` server schema. Cross-device sync is explicitly not a goal.
- **Auto-speak is unaffected.** When `autoTtsEnabled = true` and `isMuted = true`, TTS still queues and "plays" silently (provider receives audio at volume 0). Pause/seek/scrub remain functional. Rationale: this matches the TikTok analogy — content is playing, you just can't hear it until you tap.
- **Unmute restores the prior volume slider value, not a hardcoded default.** If the user lowered the slider to 0.3 last session and muted, unmuting yields 0.3, not 1.0.
- **Re-muting from the popover does not zero the slider.** The slider continues to show the user's preferred volume; only the icon and the underlying provider output reflect mute state.
- **Dragging the volume slider while muted auto-unmutes.** Specifically: if `isMuted` and the user changes the slider to any value > 0, call `onSetMuted(false)` *before* `onVolumeChange(value)` (order matters so the provider doesn't briefly receive `volume=newValue` while still being session-muted).
- **Persist version bump from 12 → 13.** Migration sets `state.startMutedOnLoad ??= true`. The `partialize` block must include `startMutedOnLoad`.

## 9. Implementation Strategy (Phased)

### Phase 1 — Store: add `startMutedOnLoad`
- Add field to `WorkspaceState` (around line 64) and `setStartMutedOnLoad` to `WorkspaceActions` (around line 118).
- Default value `true` (around line 170); add inline comment justifying the default.
- Setter `setStartMutedOnLoad: (enabled) => set({ startMutedOnLoad: enabled })` (around line 283).
- Bump `version: 12` → `version: 13` (line 333).
- Add migration block:
  ```ts
  if (version < 13) {
    state.startMutedOnLoad ??= true;
  }
  ```
- Add `startMutedOnLoad: state.startMutedOnLoad` to the `partialize` whitelist (around line 417).
- Update / extend `useWorkspaceStore.test.ts` to cover the v12→v13 migration and default value.

### Phase 2 — TTS hook: explicit mute state
- In `useTextToSpeech.ts`:
  - Add `isMuted: boolean` to the state shape (default = `useWorkspaceStore.getState().startMutedOnLoad`). Read once at hook init; do **not** subscribe to the store from inside the hook to keep test surface narrow.
  - Add `setMuted(next: boolean)` and ensure every place that calls `provider.setVolume(level)` is routed through a single helper that applies `effectiveVolume = isMuted ? 0 : volume`.
  - Expose `isMuted` and `setMuted` via the imperative ref handle that backs `setTtsVolume` etc.
- Add unit tests for the hook covering: init under `startMutedOnLoad=true`, init under `startMutedOnLoad=false`, mute → setVolume → unmute restores correct value.

### Phase 3 — Plumbing through SessionManager + Workspace
- `useSessionManager.ts`: add `setTtsMutedOnPane(sessionId, next)` (mirror of `setTtsVolumeOnPane`, line 371) and export it.
- `Workspace.tsx`: add `handleTtsSetMuted = useCallback((next) => { if (store.activePane) setTtsMutedOnPane(store.activePane, next); }, ...)` next to `handleTtsSetVolume` (line 572). Read `isMuted` from the same TTS state object the bar already consumes for `volume` (extend the state read site, no separate subscription).

### Phase 4 — `AudioPlayerBar` UI
- Add required props `isMuted: boolean`, `onSetMuted: (next: boolean) => void` to `AudioPlayerBarProps`.
- Replace `const isMuted = volume === 0;` (line 92) with use of the prop.
- Replace the audio button onClick (line 183) with the conditional behavior described in §7.4.
- Pass `isMuted` and `onSetMuted` down to both `AudioSettingsContent` instances (mobile sheet and desktop popover, lines 209 and 232).
- Update tests (`AudioPlayerBar.test.tsx`):
  - Default mock props now include `isMuted: false, onSetMuted: vi.fn()`.
  - New test: when `isMuted=true`, click on `tts-audio-button` calls `onSetMuted(false)` and does NOT open the popover.
  - New test: when `isMuted=false`, click on `tts-audio-button` opens the popover (existing behavior, but assert it explicitly).
  - Drop or update the prior "volume === 0 → muted icon" derivation test — replace with "isMuted prop drives the icon."

### Phase 5 — `AudioSettingsContent` UI
- Add required props `isMuted: boolean`, `onSetMuted: (next: boolean) => void`.
- Add a mute icon button in the volume label row (`testid={\`${testIdPrefix}-mute-toggle\`}`).
- Update `handleVolumeChange` to call `onSetMuted(false)` first when the slider moves while `isMuted` is true.
- Apply visual dim styling to the slider when muted (do not disable it — leave it interactive).
- New unit tests in `AudioSettingsContent.test.tsx` (create if missing) covering:
  - Mute toggle calls `onSetMuted` with the negation of current `isMuted`.
  - Slider drag while muted calls `onSetMuted(false)` THEN `onVolumeChange` with the new value (assert call order).
  - When `isMuted=true`, slider has the dimmed class (visual regression check via class assertion only — no snapshots).

### Phase 6 — `TtsSettingsSection`: "Start muted on app load" toggle
- Read `startMutedOnLoad` and `setStartMutedOnLoad` from `useWorkspaceStore`.
- Render a new `SettingsRow` after line 209:
  ```tsx
  <SettingsRow
    label="Start muted on app load"
    hint="When enabled, audio is muted on app load. Tap the speaker icon to unmute."
    control={(
      <SettingsToggle
        testId="start-muted-toggle"
        checked={startMutedOnLoad}
        onClick={() => setStartMutedOnLoad(!startMutedOnLoad)}
      />
    )}
  />
  ```
- Add a new test file (or extend `settings-auto-tts.test.tsx`'s structural pattern) verifying:
  - Toggle reflects store value.
  - Clicking flips the store.
  - No call to `updateTTSConfig` is made (client-only — the absence of an API call is the contract).

### Phase 7 — Cleanup & verification (mandatory)
- Run type checking: `cd scenarios/web-console/ui && npx tsc --noEmit`. **Fix every error, including pre-existing.**
- Run lint: `cd scenarios/web-console/ui && npx eslint .`. Fix every warning in modified files (and any other warning encountered as part of running it).
- Run all UI unit tests: `cd scenarios/web-console/ui && npm test`. All must pass.
- `vrooli scenario restart web-console`.
- Verify health: `curl -s http://localhost:<web-console-port>/api/health` returns 200, and the UI loads in a browser.
- Manually smoke-test by opening the web-console UI and confirming:
  - First load shows muted icon.
  - Single tap on speaker unmutes; no popover opens.
  - Second tap opens popover.
  - Popover mute toggle re-mutes; the slider value persists.
  - Slider drag while muted unmutes.
  - Toggling "Start muted on app load" off, then reloading, leaves audio unmuted on next load.

## 10. Testing Plan (Automated)

Per the project's "prefer automated tests" feedback, every behavior above is validated by an automated test, not a manual checklist. Manual smoke is the very last sanity step in §9 Phase 7 only.

| Layer | File | New/Updated Tests |
|---|---|---|
| Store | `__tests__/useWorkspaceStore.test.ts` | (new) v12→v13 migration sets `startMutedOnLoad=true`; (new) setter updates value; (new) `partialize` includes the field. |
| Hook | `hooks/__tests__/useTextToSpeech.test.ts` (create if missing, else extend) | Init with `startMutedOnLoad=true` yields `isMuted=true`; `setMuted(false)` then `setVolume(0.4)` then `setMuted(true)` then `setMuted(false)` returns provider volume to 0.4; mute applies `effectiveVolume=0` to provider. |
| Bar | `components/__tests__/AudioPlayerBar.test.tsx` | (update) Icon driven by `isMuted` prop; (new) muted-tap unmutes without opening popover; (new) unmuted-tap opens popover; (update) volume-slider test still calls `onSetVolume`. |
| Popover | `components/__tests__/AudioSettingsContent.test.tsx` (create) | Mute button toggles via `onSetMuted`; slider drag while muted calls `onSetMuted(false)` before `onVolumeChange(value)`; muted-state class is applied. |
| Settings | `__tests__/settings-start-muted.test.tsx` (create, mirror `settings-auto-tts.test.tsx`) | Toggle reflects + flips store; no API call to `updateTTSConfig`. |
| Workspace integration | `__tests__/workspace-tts-replay-bar.test.tsx` | (update) Mock now exports `setTtsMutedOnPane`; assert it is called when bar's `onSetMuted` fires. |

## 11. Rollout / Validation Checklist
- [ ] All automated tests in §10 pass (`npm test` clean).
- [ ] `tsc --noEmit` clean for `scenarios/web-console/ui`.
- [ ] `eslint .` clean for modified files (and any pre-existing warnings touched).
- [ ] `vrooli scenario restart web-console` succeeds.
- [ ] Health endpoint returns 200 after restart.
- [ ] Manual smoke (§9 Phase 7) passes.
- [ ] No new entries added to the server `TTSConfig` schema (greenfield boundary respected).

## 12. Risks + Mitigations

| Risk | Mitigation |
|---|---|
| Provider's `setVolume(0)` may pause audio output on some backends instead of just silencing it. | The `useTextToSpeech` mute helper must always pass `0` for muted state and the actual `volume` for unmuted state. Add a hook test that asserts `provider.setVolume` is called with `0` on mute and the prior value on unmute. |
| First-load race: if `useTextToSpeech` initializes before the Zustand `persist` rehydrate completes, `startMutedOnLoad` may read its in-memory default rather than the stored value. | Read the value via `useWorkspaceStore.getState()` at hook initialization time, after Zustand's `persist` middleware has rehydrated (which is synchronous for localStorage). If the workspace has any deferred-rehydrate logic, gate the hook init behind `useWorkspaceStore.persist.hasHydrated()`. |
| Existing tests asserting `volume === 0 ⇒ muted icon` will fail after the change. | Per the greenfield constraint, **update those tests** to drive the icon via the `isMuted` prop. Do not retain the old assertion as a fallback. |
| Users muted via the popover may forget mute is on (no audible feedback). | The bar icon already shows `VolumeX` in muted state. No additional cue needed; this matches industry norms (YouTube, TikTok). |
| `autoTtsEnabled=true` + `startMutedOnLoad=true` may feel confusing on first run ("why is nothing happening?"). | Hint text on the new toggle explicitly says "Tap the speaker icon to unmute." That is sufficient given the project owner is the only user. |

## 13. Non-goals / Prohibited Patterns
- No compatibility shim that keeps `volume === 0` synonymous with mute. The new `isMuted` is the sole source of truth.
- No server-side persistence of `startMutedOnLoad`. Do not add it to `TTSConfig`, do not call `updateTTSConfig` for it, do not extend `persistTtsConfig`.
- No additional storage of "preserved volume for unmute" in localStorage — keep that in hook-local state/ref only. Volume itself remains session-scoped.
- No new abstraction layer (e.g., a "MuteController" class) for what is fundamentally one boolean and a setter. Inline the logic where it lives.
- No changes to `autoTtsEnabled` semantics or wiring.
- No third permutation toggle ("mute on background", "mute on tab switch", etc.). Scope is exactly the two toggles described.

## 14. Definition of Done
1. All bullets in §11 are checked.
2. The audio bar speaker icon is the single user-facing entry point for unmute (one tap).
3. The audio popover contains a working mute toggle that does not zero the volume slider.
4. `useWorkspaceStore` persists `startMutedOnLoad` (default `true`) and survives a v13 migration test.
5. App reload with `startMutedOnLoad=true` consistently boots into muted state across at least three reloads.
6. App reload with `startMutedOnLoad=false` boots unmuted and respects the previously chosen volume.
7. No code mentions `// removed`, `_unused`, or "compat" in the diff.
8. `vrooli scenario restart web-console` followed by a health check succeeds.
