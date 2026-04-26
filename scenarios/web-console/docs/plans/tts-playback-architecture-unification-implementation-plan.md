# TTS Playback Architecture Unification Implementation Plan

## 1. Purpose

Refactor the web console's text-to-speech playback system into a single, session-scoped architecture that owns playback state, playback queue state, summarization mode state, and playback UX behavior for both the messages view and the bottom playback bar.

This plan covers:

- eliminating duplicated TTS UI state and behavior
- fixing pause/resume reliability issues
- making summarized/original playback selection deterministic and persistent
- simplifying the message-row UX and playback-bar UX
- adding the missing execution seam needed for trustworthy "current message" queue context
- validating the refactor with comprehensive unit, component, and scenario-level tests

## 2. Hard Rule: Greenfield Constraint

This implementation is **greenfield only**.

The execution must:

- remove replaced code paths rather than preserving them
- avoid compatibility shims, alias props, transitional stores, or fallback legacy UI state
- avoid dead code, deprecated helpers, or hidden dual-write periods
- land on one canonical playback architecture with one authoritative state model

Definition of done is not met if any legacy playback-state path remains live.

## 3. Required Reading

Before implementation, read:

```bash
prompt-manager skill read cli-steer api-steer utils-unification seam-discovery-and-enforcement
```

Read these repository artifacts for local context:

```bash
sed -n '1,260p' scenarios/web-console/docs/plans/tts-playback-bar-unification-plan.md
sed -n '1,260p' scenarios/web-console/docs/internal/SEAMS.md
sed -n '1,260p' scenarios/web-console/docs/internal/INVARIANTS.md
sed -n '1,260p' scenarios/web-console/docs/internal/COHERENCE-NOTES.md
sed -n '1,260p' scenarios/web-console/docs/internal/UTILS_UNIFICATION_NOTES.md
```

Inspect current implementation surfaces:

```bash
sed -n '1,760p' scenarios/web-console/ui/src/components/MessagesPane.tsx
sed -n '500,1410p' scenarios/web-console/ui/src/components/Workspace.tsx
sed -n '1,520p' scenarios/web-console/ui/src/components/TerminalPane.tsx
sed -n '1,700p' scenarios/web-console/ui/src/hooks/useTextToSpeech.ts
sed -n '1,260p' scenarios/web-console/ui/src/components/AudioPlayerBar.tsx
sed -n '1,240p' scenarios/web-console/ui/src/components/tts/PlaybackModeControl.tsx
sed -n '1,260p' scenarios/web-console/ui/src/stores/useConversationStore.ts
```

## 4. Problem Statement

The current web-console TTS system has drifted into three partially-overlapping control layers:

1. `TerminalPane` owns the real TTS engine through `useTextToSpeech`.
2. `Workspace` reconstructs active playback context, replay-bar behavior, summarize level, and playback version state.
3. `MessagesPane` separately tracks per-event summarized/original selection, per-message summarization requests, and per-message audio popover state.

This violates the intended single-source-of-truth design and creates user-visible failures:

- pause is unreliable in some runtime paths
- stop is overloaded as both transport-stop and bar-dismiss
- summarized/original selection does not stay in sync between message rows and the bottom bar
- the selected playback version is not durably persisted as the user's working preference
- the selected summarization level is only partially globalized
- the per-message audio settings UI is not actually wired to the real TTS engine
- "read from here" does not expose trustworthy per-event progress to the bottom bar
- the message row header contains redundant, non-actionable metadata that consumes limited space

## 5. Scope

### In scope

- `scenarios/web-console/ui/src/components/Workspace.tsx`
- `scenarios/web-console/ui/src/components/MessagesPane.tsx`
- `scenarios/web-console/ui/src/components/AudioPlayerBar.tsx`
- `scenarios/web-console/ui/src/components/TerminalPane.tsx`
- `scenarios/web-console/ui/src/hooks/useTextToSpeech.ts`
- `scenarios/web-console/ui/src/stores/useConversationStore.ts`
- new playback-domain modules, hooks, selectors, and tests under `scenarios/web-console/ui/src`
- test coverage updates for TTS playback, summary selection, replay behavior, and queue progress
- scenario-local documentation updates for seams and invariants if implementation changes them

### Out of scope

- backend TTS synthesis model changes
- backend summarization algorithm changes
- non-web-console scenarios
- new cross-scenario APIs
- cosmetic redesign beyond the playback surfaces and message-row cleanup necessary for this work

## 6. Current Technical Context

### Canonical playback engine

- `TerminalPane` owns the real TTS engine via `useTextToSpeech`: `ui/src/components/TerminalPane.tsx` around the TTS hook setup
- imperative playback methods are exposed through `TerminalPaneHandle`: `ui/src/components/TerminalPane.tsx` around the imperative handle contract

### Duplicated UI state

- `Workspace` owns `activePlaybackVersion`, `summarizeLevel`, `isSummarizing`, replay-bar visibility, and the active replay event context: `ui/src/components/Workspace.tsx`
- `MessagesPane` separately owns `playbackModes`, `summarizingIds`, and per-message audio popover state: `ui/src/components/MessagesPane.tsx`

### Provider-routing reliability gap

- runtime fallback to a separate browser provider uses `fallbackProviderRef`: `ui/src/hooks/useTextToSpeech.ts`
- `stop()` targets both providers, but `pause()`, `resume()`, `seek()`, and `getPlaybackState()` only target `providerRef.current`: `ui/src/hooks/useTextToSpeech.ts`

### Sequence progress gap

- `Workspace` expects per-event progress updates for read-from-here playback: `ui/src/components/Workspace.tsx`
- `TerminalPane` currently collapses all sequence chunks into one speak call and only invokes `onProgress(0)` once: `ui/src/components/TerminalPane.tsx`

### Message-row clutter

- the message header currently renders the source label and a summarized/original badge that duplicates control state: `ui/src/components/MessagesPane.tsx`

## 7. Target End State

After implementation:

1. A single session-scoped playback controller owns all playback UI state for one terminal session.
2. Both `MessagesPane` and `AudioPlayerBar` render off the same authoritative playback controller state.
3. The controller tracks:
   - active queue
   - current queue position
   - active event id
   - replay anchor event id
   - playback lifecycle state
   - playback version selection
   - summarization level
   - summarization request state
   - engine capabilities
   - real volume, mute, and playback rate
4. Pause/resume/seek/state inspection control the actual speaking provider, including runtime fallback paths.
5. Stop and dismiss are separate UX actions with separate semantics.
6. Playback version selection is deterministic:
   - selecting original always plays original when available
   - selecting a summary level always plays summarized output at that level
   - the last selected summarization level persists through settings and reloads
7. Message rows no longer maintain private playback-version state.
8. Per-message audio settings either use the canonical playback controller or are removed if not semantically valid per message.
9. Read-from-here playback exposes truthful queue position so the bottom bar can display current message context.
10. The message header row is reduced to actionable controls and essential context only.

## 8. Architecture Direction

### 8.1 Screaming architecture

Create a dedicated playback domain instead of scattering TTS orchestration across presentation components.

Preferred UI structure:

```text
ui/src/
  domains/
    tts-playback/
      controller.ts
      state.ts
      selectors.ts
      queue.ts
      summarize.ts
      contracts.ts
      __tests__/
  components/
    tts/
      AudioPlayerBar.tsx
      PlaybackModeControl.tsx
      PlaybackQueueControl.tsx
      AudioSettingsContent.tsx
```

If the repo already has a stronger local convention, adapt the folder names but keep the domain boundary explicit.

### 8.2 Responsibility boundaries

- `useTextToSpeech`:
  - provider resolution
  - provider lifecycle
  - playback transport commands
  - playback state snapshots
  - no message-specific or summarization-specific UI policy

- `TerminalPane`:
  - bridge between terminal session and playback domain
  - translate incoming assistant events into queue-ready playback events
  - expose imperative transport adapter to the playback domain
  - no duplicate policy state for message-level playback UX

- new playback controller:
  - authoritative state machine for session playback behavior
  - queue semantics
  - summarized/original selection rules
  - summarization request orchestration
  - replay-bar visibility and dismissal state
  - persistence of user playback preferences

- `MessagesPane`:
  - present message actions from controller state
  - no private playback-mode store
  - no fake per-message volume state

- `AudioPlayerBar`:
  - pure transport presentation
  - receives canonical playback model from the controller
  - no hidden business logic

## 9. Contract Decisions

1. **One authoritative playback controller per session.**
   `MessagesPane` and the bottom bar must not maintain independent playback-mode state.

2. **Playback version is session playback state, not view-local state.**
   The selected version for the currently targeted event belongs to the playback controller.

3. **Summarization level is globally persistent preference, not per-event preference.**
   It is stored through the existing summarize-config endpoint and mirrored in the controller.

4. **Transport stop and bar dismiss are different actions.**
   Stop cancels audio or queue execution.
   Dismiss hides replay UI without altering persistent summarize-level preference.

5. **Per-message popover controls must either be real or absent.**
   No local fake volume or speed controls that do not touch the actual engine.

6. **Queue progress is event-aware.**
   "Read from here" must expose event index progression, not only flat chunk playback.

7. **No legacy compatibility props.**
   Remove replaced props and callbacks instead of preserving aliases.

8. **No direct `useConversationStore.setState(...)` writes from multiple unrelated UI surfaces for playback policy.**
   Event updates can still land there, but playback policy must route through the playback domain.

## 10. Implementation Strategy

### Phase 0: Baseline capture and execution guardrails

1. Inventory all current TTS tests and identify the gaps around:
   - fallback-provider pause/resume
   - bar/message mode sync
   - read-from-here progress
   - replay-bar dismiss semantics
   - summarize-level persistence
2. Capture a short architecture note in the plan implementation PR or commit notes describing the new canonical seam.
3. Confirm no existing `AI_CHECK` tracking comments exist in affected files. At time of planning, none were found under `scenarios/web-console/ui/src`.

### Phase 1: Create the canonical playback domain seam

1. Introduce a session-scoped playback controller module with:
   - serializable state model
   - command methods
   - selectors for `MessagesPane` and `AudioPlayerBar`
2. Make the controller the only owner of:
   - active event id
   - replay event id
   - queue state
   - playback version
   - summarize level cache
   - summarize request state
   - dismissal state
3. Move presentation-agnostic queue logic and selection rules into pure utilities under the same domain.
4. Write unit tests for the controller before wiring the UI.

### Phase 2: Repair the playback transport seam

1. Refactor `useTextToSpeech` so the active speaking provider is explicit.
2. Ensure `pause`, `resume`, `seek`, `setPlaybackRate`, `setVolume`, `setMuted`, and `getPlaybackState` target the actual active provider.
3. Remove any ambiguity between the configured provider and the runtime fallback provider.
4. Add transport-focused tests for:
   - Kokoro happy path
   - browser explicit backend
   - Kokoro runtime fallback to browser in auto mode
   - pause/resume after runtime fallback
   - stop after runtime fallback

### Phase 3: Replace ad hoc `Workspace` playback reconstruction

1. Remove playback-policy state from `Workspace` that belongs in the playback controller.
2. Keep `Workspace` as the composition root that wires:
   - session id
   - terminal transport adapter
   - conversation events
   - settings-backed summarize level bootstrap
3. Replace current inline replay-bar logic with controller selectors and controller commands.
4. Keep `Workspace` focused on layout and surface composition.

### Phase 4: Replace `MessagesPane` private playback state

1. Delete `playbackModes`, local summarize orchestration state, and fake per-message audio volume state from `MessagesPane`.
2. Drive message-row controls entirely from playback controller selectors and controller commands.
3. Clean the message-row header:
   - keep copy
   - keep speak/read control
   - keep summarize/original mode control
   - keep sequence number
   - remove source label unless it becomes actionable elsewhere
   - remove summarized/original badge because it duplicates the mode control
4. Decide on the per-message audio button:
   - if it opens real settings scoped to current session playback, keep it
   - if it is only pretending to be per-message state, remove it

### Phase 5: Fix event-aware queue semantics

1. Replace the current flat sequence progress contract with an event-aware queue contract.
2. The controller should track both:
   - logical event index in queue
   - low-level transport playback state
3. `TerminalPane` read-from-here execution must report event progression truthfully rather than only a single `onProgress(0)`.
4. Add a compact queue indicator in the bottom bar once the contract is real:
   - current event sequence number or compact `n / total`
   - optional jump affordance if it can remain space-efficient
5. Add a subtle indicator for whether more queued playback remains after the current event.

### Phase 6: Redesign bottom-bar actions

1. Remove the stop button from the primary transport row.
2. Introduce a secondary dismiss action, likely an `X`, with reduced visual weight.
3. Keep pause/play as the primary transport action.
4. If an explicit stop/cancel-queue action is still needed, place it in secondary UI with distinct wording and tests.
5. Preserve narrow-layout support and avoid regressing the prior overflow fix.

### Phase 7: Persistence and settings consolidation

1. Treat summarize level as the single persisted playback summarization preference.
2. On controller initialization:
   - load summarize config
   - hydrate controller preference state
3. On user summarize-level change:
   - update controller state optimistically
   - persist through `updateTTSSummarizeConfig`
   - re-summarize active event if required
4. Ensure new messages and replay behavior respect the persisted preference without view-local override drift.

### Phase 8: Documentation and seam enforcement

1. Update `docs/internal/SEAMS.md` to describe the new playback seam.
2. Update `docs/internal/INVARIANTS.md` with explicit rules:
   - one playback controller per session
   - one active playback version
   - message rows do not own playback policy
3. Update `docs/internal/UTILS_UNIFICATION_NOTES.md` if utilities are consolidated or removed.

## 11. Testing Plan

Validation must be layered.

### 11.1 Pure unit tests

Create focused tests for pure playback-domain utilities and controller logic:

- queue building from conversation events
- active event selection
- replay anchor behavior
- summarized/original selection rules
- summarize-level persistence rules
- dismiss vs stop semantics

### 11.2 Hook and provider tests

Extend TTS hook/provider coverage for:

- runtime fallback provider selection
- pause/resume/stop correctness on fallback provider
- playback state snapshot correctness on fallback provider
- mute/volume behavior when fallback provider is active

### 11.3 Component tests

Update or add tests for:

- `AudioPlayerBar`
- `MessagesPane`
- playback mode control
- queue indicator control
- dismiss action behavior

Required behaviors:

- changing mode in message row updates bar state immediately
- changing mode in bar updates message row immediately
- changing summarize level persists and replays correct version
- removing source label and summary badge does not lose essential affordances
- bar can show current message context during queued playback

### 11.4 Integration tests at `Workspace` composition level

Add or extend tests verifying:

- replay-bar state across start, pause, resume, stop, dismiss
- auto-play catch-up after tab refocus
- summarize failure surfacing
- active event tracking during read-from-here playback
- replay after switching between original and summarized versions

### 11.5 Scenario validation

Run the relevant scenario/UI test suites after implementation. Minimum commands:

```bash
cd scenarios/web-console && make test
cd scenarios/web-console/ui && npx vitest run
cd scenarios/web-console/ui && npx tsc --noEmit
cd scenarios/web-console/ui && npx eslint src/components src/hooks src/stores src/domains
```

If the full suite is too broad, document the narrower commands actually run and why.

## 12. Rollout / Validation Checklist

- `MessagesPane` no longer owns playback-mode state.
- `Workspace` no longer reconstructs playback policy that belongs in the playback domain.
- `useTextToSpeech` controls the actual active provider for pause/resume/seek/state.
- runtime browser fallback is covered by automated tests.
- read-from-here playback reports real event progression.
- the bottom bar can indicate current queue position.
- stop and dismiss are separate actions.
- summarize-level persistence is deterministic across reload and new messages.
- source label and summary badge are removed from message-row header unless reintroduced as explicitly actionable controls.
- no compatibility shims or dead playback state remain.
- all modified files are documented where needed and have passing tests.

## 13. Risks + Mitigations

### Risk: transport refactor and UI refactor entangle and become hard to land

Mitigation:

- complete provider-routing repair first
- keep the playback controller pure and test-driven
- wire presentation only after controller tests pass

### Risk: read-from-here event progress is more complex than expected

Mitigation:

- formalize a queue contract before changing the bar UI
- separate logical event progression from transport chunk progression

### Risk: summarize persistence interacts poorly with optimistic UI

Mitigation:

- define one explicit optimistic update policy in the controller
- test failed persistence and retry paths

### Risk: refactor drifts into another layer of indirection

Mitigation:

- keep one playback domain
- keep one terminal transport seam
- avoid parallel stores or wrapper wrappers

## 14. Non-goals / Prohibited Patterns

- no compatibility layer between old and new playback-state models
- no duplicate source-of-truth stores
- no view-local playback mode state in `MessagesPane`
- no fake per-message volume or speed state
- no reintroduction of overloaded stop-as-dismiss semantics
- no broad `utils.ts` dumping ground; extracted utilities must live in the playback domain or established shared tiers
- no direct implementation of business policy inside presentational components
- no dead props, deprecated callbacks, or temporary alias APIs

## 15. Definition of Done

This work is done only when all of the following are true:

1. The web-console UI has one canonical playback controller per session.
2. Both the messages view and the bottom playback bar consume the same playback-policy state.
3. Runtime fallback playback can be paused, resumed, stopped, and inspected reliably.
4. Summarized/original selection is synchronized across surfaces and behaves deterministically.
5. Summarization level persists and becomes the effective default after reload and for new messages.
6. Read-from-here playback exposes current event context suitable for compact bar display.
7. The bottom bar uses separate dismiss semantics instead of an overloaded stop button.
8. The message-row header has been cleaned of redundant non-actionable indicators.
9. No compatibility code or dead legacy playback state remains in the modified codebase.
10. Typecheck, lint, unit tests, integration tests, and scenario tests all pass for the modified scope.

## 16. Suggested Plan Filename Rationale

This plan intentionally uses `tts-playback-architecture-unification-implementation-plan.md` rather than extending the earlier playback-bar plan because the problem is broader than bar UX. The real work is playback architecture unification, not only bar polish.
