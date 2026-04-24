# TTS Playback Bar Unification & Mode-Toggle Surfacing

## 1. Purpose

Consolidate the two duplicated TTS audio-settings popovers, remove the heavy styling swap that makes summarized mode read as a "different bar," fix the viewport overflow that clips the volume button in summarized mode, and surface summarized/original mode switching (including summarization level) as a first-class control on the bar itself.

## 2. Required Reading

```bash
prompt-manager skill read ux react-coherence refactor
```

## 3. Problem Statement

The web console has one `AudioPlayerBar` component and one near-duplicate popover body inside `MessagesPane`. From a user perspective this presents three distinct problems:

1. **"Two bars" illusion.** In summarized mode, `AudioPlayerBar` swaps background (`bg-amber-950/30`), border color, volume icon color, scrub accent, adds a left-side pill badge, and uses amber popover thumb styling — six simultaneous visual changes that make it read as a separate component even though it is one.
2. **Mode toggle is effectively hidden.** The summarized ↔ original toggle and the on-demand summarize button only exist inside the audio-settings popover (behind the volume icon). There is no at-a-glance mode indicator that is also clickable.
3. **Volume icon clips out of viewport in summarized mode.** The bar is a flex row with no `min-w-0` on the scrub, `min-w-[5rem]` on the time display, and no overflow strategy. Adding the summarized pill pushes the rightmost button (volume/settings) outside the container on narrow panes, making mode switching unreachable — the root of the user-reported "can't even click it" bug.
4. **Duplicated popover body.** `AudioSettingsContent` in `AudioPlayerBar.tsx:50-161` and `AudioPopoverContent` in `MessagesPane.tsx:59-164` are near-identical copies. They will drift.
5. **Summarization level is buried in global settings.** `TtsSettingsSection.tsx:397-418` lets users pick Light / Moderate / Heavy, but reaching it mid-playback requires leaving the conversation.

## 4. Scope

### In scope

- `scenarios/web-console/ui/src/components/AudioPlayerBar.tsx` — layout fix, styling de-escalation, new inline mode control.
- `scenarios/web-console/ui/src/components/MessagesPane.tsx` — replace `AudioPopoverContent` with the shared component.
- New shared component file: `scenarios/web-console/ui/src/components/tts/AudioSettingsContent.tsx` (or similar co-located path).
- `scenarios/web-console/ui/src/components/Workspace.tsx` — pass summarization-level props/handlers to the bar.
- Wiring to `updateTTSSummarizeConfig` + re-summarize on level change (API already exists in `lib/api.ts:1036`).
- Tests in `scenarios/web-console/ui/src/components/__tests__/AudioPlayerBar.test.tsx` and a new test for the shared settings component.

### Out of scope

- Backend summarization pipeline changes.
- Mobile bottom sheet redesign (only fix overflow and reuse shared body).
- Replay-bar behavior changes.
- Keyboard shortcut changes for TTS.

## 5. Current Technical Context

Key files and line references:

- `components/AudioPlayerBar.tsx:10-35` — `AudioPlayerBarProps` (already exposes `isSummarized`, `hasOriginalVersion`, `canSummarize`, `onToggleSummarized`, `onRequestSummarize`).
- `components/AudioPlayerBar.tsx:50-161` — `AudioSettingsContent` (desktop popover + mobile sheet body). To be extracted and shared.
- `components/AudioPlayerBar.tsx:229-246` — bar container + "Summarized" badge; amber background/border swap lives here.
- `components/AudioPlayerBar.tsx:272-304` — scrub + time display (root of overflow: `flex-1` without `min-w-0`, time has `min-w-[5rem]`).
- `components/AudioPlayerBar.tsx:318-334` — audio/volume button (the one getting clipped).
- `components/MessagesPane.tsx:59-164` — duplicate `AudioPopoverContent`; must be replaced.
- `components/MessagesPane.tsx:183-426` — per-message playback state (`playbackModes`, `summarizingIds`, `summarizeErrors`).
- `components/Workspace.tsx:1138-1220` — AudioPlayerBar mounting site, handlers for toggle and summarize.
- `components/settings/TtsSettingsSection.tsx:397-418` — reference for level options (Light / Moderate / Heavy).
- `lib/api.ts:1026-1040` — `getTTSSummarizeConfig` / `updateTTSSummarizeConfig`; `TTSSummarizeConfig.level` is `"light" | "moderate" | "heavy"`.
- `components/__tests__/AudioPlayerBar.test.tsx` — existing test coverage.

## 6. Target End State

1. **One component** (`AudioSettingsContent`) is the source of truth for the settings popover body, imported by both `AudioPlayerBar` and `MessagesPane`.
2. **Summarized mode de-escalates to one primary visual signal**: the scrub accent color. No background swap, no border swap, no volume-icon color change. The inline mode control (see #3) doubles as the textual/iconographic indicator.
3. **New inline mode control on the bar** — to the left of the play/pause button, a compact segmented/dropdown control showing the current mode:
   - When original is playing and a summary exists: shows `Original ▾` with dropdown containing `Original`, `Light`, `Moderate`, `Heavy`.
   - When summarized is playing: shows `Summarized ▾` with same dropdown options; the currently selected level is checked.
   - When no summary exists and `canSummarize`: shows a compact `Summarize ▾` dropdown that lets the user pick a level and summarize.
   - When no summary exists and `!canSummarize`: control is hidden.
   - Selecting a level different from the current one calls `updateTTSSummarizeConfig({ level })` and triggers `onRequestSummarize()` (which re-summarizes and replays the active event).
   - Selecting `Original` calls `onToggleSummarized(false)`.
   - Selecting the current mode's level re-summarizes if the level changed, no-op if identical.
4. **No viewport overflow on the web-console's narrowest supported pane** (~320px logical width). Volume/settings button is always reachable.
5. **The old mode toggle inside the settings popover is removed** — replaced entirely by the inline control. The popover keeps volume slider + "Summarize for playback" (only when no inline control is shown, i.e., fallback path).
6. **Per-message controls in `MessagesPane` continue to work** and render the shared component so UX stays consistent.

## 7. Implementation Strategy (Phased)

### Phase 1 — Extract shared `AudioSettingsContent`

1. Create `components/tts/AudioSettingsContent.tsx` containing the existing body (volume slider + summarized/original toggle + summarize button).
2. Parameterize with a `testIdPrefix` prop so `AudioPlayerBar` and `MessagesPane` can keep their distinct `data-testid` namespaces (`tts-*` vs `msg-*-<eventId>`).
3. Import in `AudioPlayerBar.tsx`; delete the inline `AudioSettingsContent`.
4. Import in `MessagesPane.tsx`; delete `AudioPopoverContent` and update callsite.
5. Confirm both existing test files still pass without behavior change.

### Phase 2 — Fix viewport overflow

1. In `AudioPlayerBar.tsx` bar container (line 229), ensure the scrub `<input>` has `min-w-0` so `flex-1` can actually shrink.
2. Remove `min-w-[5rem]` from the time display (line 302); format stays `m:ss / m:ss` which fits in ~6–7ch and uses `tabular-nums`. Let it shrink naturally.
3. Hide the speed button (line 307-316) at viewport widths below the `sm` breakpoint (use `hidden sm:inline-flex` or equivalent tailwind pattern). Speed stays accessible from the settings popover if needed; if not already there, add it (see Phase 3 — this is the right home for it).
4. Add a wrapping container around transport controls so the bar degrades to `[mode-control] [play] [stop] [time] [settings]` when narrow and `[mode-control] [play] [stop] [scrub] [time] [speed] [settings]` when wide.

### Phase 3 — Move speed into the settings popover (de-clutter)

1. Remove the standalone speed button from the bar.
2. Add speed selector (same six presets `[0.5, 0.75, 1, 1.25, 1.5, 2]`) to the shared `AudioSettingsContent` as a segmented control above the volume slider, gated on `capabilities.canAdjustSpeed`.
3. Extend the shared component's props with `playbackRate` and `onSetPlaybackRate`; pass through from both callsites.

### Phase 4 — De-escalate summarized-mode styling

1. In `AudioPlayerBar.tsx` bar container (line 229), remove the conditional `border-amber-500/30 bg-amber-950/30`. Keep a single `border-wc-default bg-wc-surface-raised` style.
2. Remove the amber color on the volume icon button (line 324-329) — button uses the default `hover:bg-wc-accent/10` in all modes.
3. Remove the standalone "Summarized" pill badge (line 239-246) — the inline mode control replaces it.
4. Keep the scrub accent swap (line 284, `accent-amber-400` when summarized) — this is the single retained visual signal.
5. Keep the shared `AudioSettingsContent`'s amber styling on volume/toggle for consistency inside the popover; this is fine because it's a contained surface.

### Phase 5 — Add inline `PlaybackModeControl`

1. Create `components/tts/PlaybackModeControl.tsx` — a compact button + popover/dropdown anchored by the button, similar to how `AudioPlayerBar` uses `createPortal` for its popover.
2. Props: `isSummarized`, `hasOriginalVersion`, `canSummarize`, `isSummarizing`, `currentLevel` (`"light" | "moderate" | "heavy"`), `onToggleSummarized`, `onChangeLevel: (level) => void` (replaces `onRequestSummarize` — see Contract Decisions).
3. Render logic (inclusive of all states — loading, error, empty, success):
   - `isSummarized` → button label `Summarized`.
   - `!isSummarized && hasOriginalVersion` → button label `Original`.
   - `!hasOriginalVersion && canSummarize` → button label `Summarize`.
   - `!hasOriginalVersion && !canSummarize` → render nothing.
   - `isSummarizing` → button disabled with a small spinner; keep label.
4. Dropdown items (only shown when `hasOriginalVersion || canSummarize`): `Original`, separator, `Light`, `Moderate`, `Heavy`. The active level is checked; the active `Original` is checked when `!isSummarized`.
5. Selecting an item:
   - `Original` → `onToggleSummarized(false)`.
   - A level → if `!isSummarized || currentLevel !== level` → `onChangeLevel(level)` (parent handles config update + re-summarize + playback swap). If already summarized at that level → close dropdown, no-op.
6. Place the component as the leftmost child in the bar container in `AudioPlayerBar.tsx`, before play/pause.

### Phase 6 — Wire summarization level through `Workspace.tsx`

1. In `Workspace.tsx`, add state or selector for `summarizeConfig.level` — pull from `getTTSSummarizeConfig()` on mount, store in local component state or a dedicated zustand slice.
2. Pass `currentLevel` to `AudioPlayerBar`.
3. Replace `onRequestSummarize` with `onChangeLevel(level)`:
   - Calls `updateTTSSummarizeConfig({ level })` (only when level actually changed; skip API call otherwise).
   - Then invokes the existing summarize flow (`summarizeEvent(...)` + store update + replay) exactly as today.
4. `AudioPlayerBar` internally forwards `onChangeLevel` to `PlaybackModeControl`.

### Phase 7 — Cleanup & verification

1. Run TypeScript (`cd scenarios/web-console/ui && npx tsc --noEmit`) and fix every error, including pre-existing ones in files touched.
2. Run linter (`npx eslint src/components/AudioPlayerBar.tsx src/components/MessagesPane.tsx src/components/Workspace.tsx src/components/tts/*.tsx src/components/__tests__/AudioPlayerBar.test.tsx`) and fix every warning in modified files, including pre-existing ones.
3. Run the UI test suite for affected files (`npx vitest run src/components/__tests__/AudioPlayerBar.test.tsx`) and fix every failure.
4. Do **not** restart the web-console scenario. Leave the dev server running; the user restarts or refreshes manually.

## 8. Contract Decisions

- **Summarization level is global, not per-event.** Reason: matches today's `TtsSummarizeConfig` surface and avoids a new per-event schema. A level change applies to the active event immediately (re-summarize + replay) and to all future summarizations.
- **Inline mode control replaces the popover toggle** — there is no longer two places to change mode. The settings popover loses its `Playback version` section entirely. The "Summarize for playback" section inside the popover is also removed (now in the inline control as `Summarize ▾`).
- **`onRequestSummarize` is replaced by `onChangeLevel(level: "light" | "moderate" | "heavy")`.** The old prop is deleted; no compat alias.
- **Speed selector moves into the settings popover.** Rationale: reduces bar clutter, frees horizontal space, keeps power-user control one click away.
- **No `min-w` on time display.** Format is `m:ss / m:ss`; `tabular-nums` keeps width visually stable. If a stable width is later needed, use `ch` units, not rems.
- **`PlaybackModeControl` is a peer of `AudioSettingsContent`**, not nested inside it. Lives under `components/tts/`.
- **Testids** follow existing conventions:
  - `tts-mode-control` for the inline button
  - `tts-mode-option-original`
  - `tts-mode-option-light` / `-moderate` / `-heavy`
  - `tts-speed-preset-<rate>` for in-popover speed buttons
  - All other existing testids unchanged

## 9. Testing Plan

Automated tests are the primary verification. Manual smoke is a secondary sanity check only.

### Unit tests (vitest + testing-library)

Add to `components/__tests__/AudioPlayerBar.test.tsx`:

- **Overflow regression**: Render bar at 320px wide with `isSummarized=true`, assert volume button is within the visible container bounds (check `scrollWidth <= clientWidth` on the bar).
- **Mode control visibility**:
  - Renders `Summarized` label when `isSummarized=true`.
  - Renders `Original` label when `isSummarized=false` and `hasOriginalVersion=true`.
  - Renders `Summarize` label when `!hasOriginalVersion && canSummarize`.
  - Renders nothing when `!hasOriginalVersion && !canSummarize`.
- **Dropdown interactions**:
  - Clicking `Original` calls `onToggleSummarized(false)`.
  - Clicking a different level calls `onChangeLevel(level)`.
  - Clicking the current level is a no-op (does not call `onChangeLevel`).
- **Disabled state during summarization**: `isSummarizing=true` disables the control and shows spinner.
- **Styling de-escalation**: In summarized mode, bar container does NOT have `bg-amber-950/30`. Scrub DOES have `accent-amber-400`.
- **Speed in popover**: Opening settings popover reveals speed presets; clicking one calls `onSetPlaybackRate(rate)`.

New test file `components/__tests__/AudioSettingsContent.test.tsx`:

- Renders volume slider with correct value.
- Renders speed presets and active one is highlighted.
- Does NOT render a summarized/original toggle (that moved to `PlaybackModeControl`).

New test file `components/__tests__/PlaybackModeControl.test.tsx`:

- Dropdown menu structure (Original + Light/Moderate/Heavy).
- Keyboard navigation (arrow keys, Enter, Escape).
- Currently active level has a visible checkmark.
- Clicking outside closes the dropdown.

Update `MessagesPane` tests (if any exist) to confirm the shared `AudioSettingsContent` renders with `testIdPrefix="msg"` and all `msg-*-<eventId>` testids continue to resolve.

### Manual sanity (post-implementation, done by user)

- Mid-playback level change triggers re-summarize and audible re-play.
- On a narrow pane (<400px), all bar controls are reachable in both modes.

## 10. Rollout / Validation Checklist

1. All TypeScript errors resolved (modified files and any pre-existing errors in those files).
2. All ESLint warnings resolved in modified files.
3. All new and existing unit tests pass (`npx vitest run`).
4. No `AudioPopoverContent` symbol remains in `MessagesPane.tsx`.
5. No `AudioSettingsContent` symbol remains inline in `AudioPlayerBar.tsx`.
6. `rg "bg-amber-950"` in `scenarios/web-console/ui/src` returns no results (confirms background swap removed).
7. Manual visual check by user: narrow viewport, summarized mode, volume button clickable.

## 11. Risks + Mitigations

| Risk | Mitigation |
|---|---|
| Removing `min-w-[5rem]` on time causes visual jitter as seconds tick | `tabular-nums` is already applied; confirm by test at 320px. If jitter appears, use `min-w-[3ch]` per side instead of removing entirely. |
| Mid-playback level change feels abrupt (audio stops + re-summarize delay) | Acceptable — same behavior as clicking "Summarize for playback" today. A future enhancement could precompute alternative levels; not in this plan's scope. |
| Extracted shared component couples the two surfaces too tightly | `testIdPrefix` + explicit props keep the coupling minimal. Both surfaces already use identical underlying state; divergence was the accident, not the reuse. |
| Speed button moving into popover makes speed changes slower | Speed-cycle-on-click pattern can be preserved inside the popover via the same SPEED_PRESETS array; no functional regression. If the user relies on bar-level speed access, restore it in a follow-up. |
| Global `updateTTSSummarizeConfig` on every level change is chatty | Call API only when the level actually differs from current. Single POST per user-initiated change is fine. |

## 12. Non-goals / Prohibited Patterns

- **Greenfield only.** Do not add compat shims, do not keep `onRequestSummarize` as an alias, do not add `// removed` comments, do not export unused types, do not rename variables to `_unused`.
- **No mode toggle in two places.** The popover toggle must be fully removed, not hidden behind a feature flag.
- **No restart of the web-console scenario from within Claude Code.** Write code to disk only; the user restarts manually.
- **No per-event level override schema.** Level stays a single global value this iteration.
- **No new global store** for summarization level — read via existing `getTTSSummarizeConfig` API and cache in component state or in an existing TTS-related store if one already exists.
- **No mass-replace scripts** for testid or prop renames — update callsites individually.

## 13. Definition of Done

- `components/tts/AudioSettingsContent.tsx` exists and is the sole definition of the settings popover body.
- `components/tts/PlaybackModeControl.tsx` exists and is rendered as the leftmost child inside `AudioPlayerBar`.
- `AudioPlayerBar.tsx` no longer contains `bg-amber-950/30`, no longer renders a standalone "Summarized" pill, no longer renders a standalone speed button, and does not define `AudioSettingsContent` inline.
- `MessagesPane.tsx` no longer defines `AudioPopoverContent`; imports and renders the shared component.
- Summarized/original mode, and summarization level (Light/Moderate/Heavy), are switchable in one click + one dropdown-item click from the bar itself.
- On a 320px-wide viewport in summarized mode, the volume/settings button is fully visible and clickable.
- TypeScript passes with zero errors in `scenarios/web-console/ui`.
- ESLint passes with zero warnings in all modified files.
- All unit tests in `scenarios/web-console/ui/src/components/__tests__/` pass.
- No mention of backwards compatibility, migration, or deprecated shims anywhere in the diff.
