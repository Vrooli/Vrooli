# Decision Stream And Auto-Advance UX Implementation Plan

## 1. Purpose

Repair the Swarm Manager Decision Stream so it becomes a trustworthy, high-throughput operator workflow instead of a secondary surface that feels riskier than answering decisions in backlog details. This plan covers the full implementation and validation path for:

- deterministic Decision Stream ordering
- removal of stale/already-answered questions from the stream
- a professional responsive navigator
- a dedicated completion / next-step state after the final answer
- shared countdown / cancel visibility for deferred workshop auto-advance
- parity between the Decision Stream and backlog-details workshop surfaces

The intended outcome is simple: the operator should be able to enter Command Post, answer many decisions quickly, understand exactly what will happen next, and either let the next workshop/finalize action proceed or cancel it without feeling rushed.

## 2. Required Reading

Run before implementation:

```bash
prompt-manager skill read cli-steer api-steer utils-unification seam-discovery-and-enforcement
```

Recommended additional reading:

```bash
prompt-manager skill read implementation-plan-authoring
```

Primary repo context:

- `scenarios/swarm-manager/ui/src/components/command-post/CommandPostOverlay.tsx`
- `scenarios/swarm-manager/ui/src/components/command-post/DecisionStreamView.tsx`
- `scenarios/swarm-manager/ui/src/components/command-post/ScenarioNavigatorPopover.tsx`
- `scenarios/swarm-manager/ui/src/hooks/useDecisionStreamLogic.ts`
- `scenarios/swarm-manager/ui/src/lib/command-post-utils.ts`
- `scenarios/swarm-manager/ui/src/components/backlog/inline-question-stepper.tsx`
- `scenarios/swarm-manager/ui/src/components/backlog/auto-advance-countdown.tsx`
- `scenarios/swarm-manager/ui/src/components/backlog/workshop-panel.tsx`
- `scenarios/swarm-manager/ui/src/components/backlog/backlog-notes-panel.tsx`
- `scenarios/swarm-manager/ui/src/components/backlog/header-primary-action.tsx`
- `scenarios/swarm-manager/ui/src/hooks/useBacklogDetailData.ts`
- `scenarios/swarm-manager/ui/src/hooks/useBacklogCRUDHandlers.ts`
- `scenarios/swarm-manager/api/internal/backlog/backlog_summary.go`
- `scenarios/swarm-manager/api/internal/backlog/pending_questions.go`
- `scenarios/swarm-manager/api/internal/backlog/workshop_save.go`
- `scenarios/swarm-manager/ui/src/components/agents/AgentsDropdown.tsx`
- `scenarios/swarm-manager/ui/src/components/ui/drawer.tsx`
- `scenarios/swarm-manager/ui/src/components/ui/bottom-sheet.tsx`

## 3. Problem Statement

The current Decision Stream is functionally close to useful, but not reliable enough to become the operator’s default path.

Observed and code-grounded issues:

1. Question ordering is not owned as a stable contract for the Decision Stream.
   - Within one backlog item, unanswered workshop decisions preserve latest-round file order.
   - Across backlog items, `CommandPostOverlay` currently consumes `backlog-summary`, whose `pending_questions` list is built from `LoadAll(nil)` order in `api/internal/backlog/backlog_summary.go` rather than the ranked order already available in `api/internal/backlog/pending_questions.go`.
   - Result: parent-item ordering is effectively filesystem/load-order instead of an explicit “most important next” sequence.

2. Already-answered questions can remain visible in the Decision Stream and in its count.
   - `CommandPostOverlay` reads `backlog-summary` through React Query with `staleTime: 60_000`.
   - `useDecisionStreamLogic` filters `activeQuestions` only by snoozed/deleted state, not by successful local answer persistence.
   - Result: stale cache plus no local pruning can surface already-answered questions, which makes the stream feel untrustworthy.

3. The Decision Stream completion model is more aggressive and less transparent than backlog-details workflows.
   - `useDecisionStreamLogic` saves answers, then on all-resolved state performs `workshopSave` and evaluates auto-advance for each affected parent item.
   - There is no dedicated completion surface showing “what happens next,” only a brief “Saving answers and checking auto-advance...” state and then return to summary.
   - Result: operators cannot calmly review the final state, explicitly launch the next step, or clearly cancel a pending deferred auto-advance from inside the Decision Stream.

4. Deferred auto-advance visibility is inconsistent across surfaces.
   - The backend already returns `autoAdvance.pending`, `advanceAt`, `delaySeconds`, and `nextMode`.
   - `AutoAdvanceCountdown` already exists and supports cancellation.
   - That state is surfaced in backlog-card transitional UI, but not prominently in the backlog-details workshop header/panel where the operator actually expects to control the next step, and not in the Decision Stream completion path at all.

5. The Decision Stream navigator is not production-grade.
   - `ScenarioNavigatorPopover` is a custom absolute-positioned popover with no viewport clamping and no mobile-specific presentation.
   - The repo already has better responsive patterns via `AgentsDropdown`, `BottomSheet`, and `Drawer`.
   - Result: the navigator feels visually rough, can overflow the viewport, and does not adapt to mobile ergonomics.

6. The UX contract between backlog details and Decision Stream is inconsistent.
   - The inline backlog-card stepper is explicit: answer, then `Next`/`Finish`.
   - The Decision Stream also advances only on explicit navigation, but because it does more hidden completion work afterward, the operator experiences it as less predictable.
   - Result: the “bulk answer” workflow feels riskier than the single-item workflow, which is backwards.

## 4. Scope

### In scope

- Decision Stream queue correctness and deterministic ordering
- local pruning of answered questions from the active Decision Stream session
- explicit Decision Stream completion / next-step UI
- shared auto-advance countdown and cancel visibility in Decision Stream and backlog details
- responsive replacement or substantial rewrite of the Decision Stream navigator
- query invalidation / refresh behavior needed to remove stale pending-question surfaces
- automated coverage for queue semantics, completion state, countdown behavior, and responsive navigator behavior

### Out of scope

- New question-level answer API endpoints
- Rewriting workshop round persistence away from the existing fetch-patch-save plus `/workshop/save` contract
- General Command Post redesign beyond the decision-specific surfaces
- Review-flow redesign unrelated to pending workshop decisions
- New backlog ranking semantics beyond choosing and standardizing one existing server-side source of truth
- Large visual redesign of backlog-details pages unrelated to workshop/decision control

## 5. Current Technical Context

### Existing contracts and seams

- `api/internal/backlog/workshop_save.go` already owns workshop auto-advance decision-making and emits:
  - `triggered`
  - `pending`
  - `advanceAt`
  - `delaySeconds`
  - `nextMode`
- `ui/src/components/backlog/auto-advance-countdown.tsx` already renders a countdown and cancel button backed by `backlogService.workshopCancelPendingAdvance(...)`.
- `ui/src/components/backlog/inline-question-stepper.tsx` already uses the calm interaction model desired here:
  - answer selection does not itself advance
  - the operator advances with `Next` / `Finish`
- `ui/src/components/agents/AgentsDropdown.tsx` already demonstrates the desired responsive navigator pattern:
  - clamped desktop popover
  - mobile bottom-sheet
- `ui/src/components/ui/drawer.tsx` and `ui/src/components/ui/bottom-sheet.tsx` already provide reusable mobile-friendly shell components.

### Current Decision Stream data flow

- `CommandPostOverlay` queries `["backlog-summary"]` and derives cross-item questions from `summaryQuery.data?.pending_questions?.items`.
- `aggregateCrossItemQuestions(...)` in `ui/src/lib/command-post-utils.ts` flattens grouped item questions into a single list while preserving parent-item order as supplied by the server.
- `useDecisionStreamLogic` owns local session state: current index, local answers, skip set, deleted questions, snoozed parents, and completion behavior.

### Current backlog-details wiring

- `BacklogNotesPanel` renders `WorkshopPanel` for full backlog-detail workshop interaction.
- `HeaderPrimaryAction` and `BacklogActionButtons` expose workshop/finalize/run CTAs, but they do not currently surface deferred auto-advance countdown/cancel state.
- `useBacklogCRUDHandlers` reacts to `workshopSave` success for activity refresh but does not elevate pending auto-advance into a persistent detail-page UI model.

### Current evidence that must drive the plan

- `DecisionStreamView` does not auto-advance on option click; progression is already explicit.
- The stale-question bug therefore should be treated as queue/data-state corruption, not answer-click behavior.
- `backlog_summary.go` and `pending_questions.go` currently expose two different ordering stories for pending questions.
- The Command Post uses the weaker one today.

## 6. Target End State

After implementation:

1. The Decision Stream always presents unresolved questions only.
   - Successfully answered questions disappear from the active queue immediately.
   - Counts, progress, and navigator rows reflect unresolved questions only.

2. Parent backlog items in the Decision Stream are deterministically ordered by a single server-owned ranking contract.
   - The stream never falls back to filesystem/load-order semantics.

3. The operator experience is explicit and calm.
   - Selecting an answer never advances automatically.
   - `Next` / `Done` remain the only progression actions.
   - After the final answer, the Decision Stream transitions into a completion state rather than bouncing back to summary.

4. The completion state clearly communicates the next action.
   - It shows whether the next step is `workshop` or `finalize`.
   - If auto-advance is pending, it shows the countdown and cancel affordance.
   - If auto-advance is not pending, it offers centered explicit next-step buttons plus a skip/exit option.

5. Backlog details and Decision Stream share the same auto-advance mental model.
   - When a deferred next step exists, both surfaces show a countdown, clear “what will happen next,” and cancellation.

6. The Decision Stream navigator looks intentional and works on all screen sizes.
   - Desktop: anchored and clamped to viewport bounds.
   - Mobile: presented as a bottom sheet or drawer-style sheet using shared components.

## 7. Implementation Strategy

Phases are dependency-ordered. Each phase should land in a way that keeps the system working and simplifies the next phase.

### Phase 1 — Canonicalize pending-question ordering for Command Post

Establish one server-owned source of truth for Decision Stream ordering before changing UI behavior.

Deliverables:

- Choose one canonical API path for Command Post pending decisions.
  - Recommended: keep using the combined `backlog-summary` endpoint for round-trip efficiency, but update `buildPendingQuestions(...)` in `api/internal/backlog/backlog_summary.go` to apply the same ranked parent-item ordering used by `PendingQuestions(...)` in `api/internal/backlog/pending_questions.go`.
  - Alternative only if necessary: teach `CommandPostOverlay` to call the dedicated pending-questions endpoint instead of summary.
- Extract or reuse the ranking seam already represented by `rankPendingQuestionItems(...)` so summary and dedicated endpoint cannot drift.
- Preserve within-item question order exactly as it appears in the latest workshop round.

Why this order:

- The stream cannot feel stable until ranking is stable.
- This also avoids baking any UI-only “sort again just in case” workaround into the Command Post layer.

Acceptance criteria for Phase 1:

- Two backlog items with different dependency/priority rank always appear in the same ranked order in Command Post.
- Multiple unanswered questions under one backlog item remain in latest-round order.
- The same parent-item ordering is produced by summary-backed and pending-questions-backed tests.

### Phase 2 — Fix Decision Stream queue correctness and stale-answer removal

Repair the local session model so already-answered questions cannot remain visible or counted.

Deliverables:

- Update `CommandPostOverlay` query behavior on Decision Stream entry:
  - force a fresh refetch or explicit invalidation before the stream is shown
  - do not rely on a 60-second cached summary when entering a decision-focused workflow
- Update `useDecisionStreamLogic` so `activeQuestions` excludes questions that have already been successfully answered in the current session.
  - answered questions should be pruned from the active queue after save succeeds
  - counts and progress should recompute from unresolved questions
- Keep skipped questions and snoozed parents explicit and separate from answered questions.
- Ensure deleted questions are removed everywhere: question area, counter, navigator, and completion logic.

Recommended seam:

- Introduce a small “decision stream queue reducer” or pure helper module under `ui/src/lib/` or `ui/src/components/command-post/lib/` that computes:
  - unresolved questions
  - current index normalization
  - parent grouping
  - completion readiness

This should not stay entangled inside the hook as hand-rolled array filtering.

Acceptance criteria for Phase 2:

- A question answered in the current session never reappears later in that same session.
- A question answered elsewhere and then reopened via Command Post disappears after stream entry refresh.
- The counter never includes answered questions.
- The navigator unresolved counts match the visible unresolved queue.

### Phase 3 — Add a dedicated Decision Stream completion state

Replace the current “save and return” behavior with a proper completion surface.

Deliverables:

- Extend `DecisionStreamView` / `useDecisionStreamLogic` to support a new post-answer phase, separate from:
  - `answering`
  - `completing`
- The new phase should render a centered completion panel with:
  - summary of answered / skipped / snoozed work
  - next-mode CTA(s): `Run next workshop round` or `Finalize`
  - `Back to Command Post` or `Skip for now`
  - unlocked-items list when multiple parents are affected
- Preserve the existing `workshopSave`-backed auto-advance evaluation, but expose its result instead of hiding it.

Important contract decision:

- The completion view must not independently recompute workshop readiness or next mode in the UI.
- It must consume the backend `autoAdvance` / `nextMode` contract from the existing `workshopSave` response.

Acceptance criteria for Phase 3:

- Finishing the last question no longer jumps directly back to summary.
- If no next step is available, the completion state explains that clearly.
- If one or more parent items unlocked a next step, the operator can launch or skip it from the completion surface.

### Phase 4 — Promote shared auto-advance state into a first-class seam

Unify the “what happens next?” UX across Decision Stream and backlog details.

Deliverables:

- Create a shared UI seam for workshop transition state instead of keeping it implicit inside backlog-card-only transitional rendering.
  - Recommended extraction: a focused component and lightweight state adapter around:
    - auto-advance pending countdown
    - triggered next-step spinner
    - explicit next-step readiness message
    - cancel action
- Reuse `AutoAdvanceCountdown`, or refactor it into a slightly broader transition-state component if needed.
- Add a detail-page state holder so workshop-save results can be rendered in:
  - `WorkshopPanel`
  - header primary action area when appropriate
- The same seam should be consumable by the Decision Stream completion state.

Files most likely involved:

- `ui/src/components/backlog/auto-advance-countdown.tsx`
- `ui/src/components/backlog/workshop-panel.tsx`
- `ui/src/components/backlog/backlog-notes-panel.tsx`
- `ui/src/components/backlog/header-primary-action.tsx`
- `ui/src/hooks/useBacklogCRUDHandlers.ts`
- `ui/src/hooks/useBacklogDetailData.ts`

Acceptance criteria for Phase 4:

- If `autoAdvance.pending` is returned after a detail-page decision save, the detail page shows countdown + cancel without requiring the operator to infer it from agent activity alone.
- If `autoAdvance.pending` is returned from a Decision Stream completion path, the same countdown + cancel model appears there.
- Cancelling a pending advance updates the UI immediately and prevents the pending run from firing.

### Phase 5 — Rebuild the navigator using shared responsive patterns

Replace the hand-rolled navigator popover with a polished responsive implementation.

Deliverables:

- Desktop behavior:
  - keep anchored trigger behavior
  - add viewport clamping comparable to `AgentsDropdown`
- Mobile behavior:
  - open the navigator as a `BottomSheet` or `Drawer`-based sheet
  - preserve quick jump and per-parent snooze controls
- Upgrade row presentation:
  - cleaner parent title hierarchy
  - clearer unresolved counts
  - stronger current-parent highlighting

Implementation note:

- Do not add another bespoke mobile modal pattern.
- Use existing `BottomSheet` / `Drawer` infrastructure unless a narrowly-scoped extension is genuinely required.

Acceptance criteria for Phase 5:

- Desktop navigator never renders off-screen horizontally or vertically.
- Mobile navigator is presented as a sheet, not a tiny anchored popover.
- Navigation and snooze actions behave identically in both presentations.

### Phase 6 — Detail-page workshop CTA visibility and cancellation clarity

Make the backlog-details page match the Decision Stream’s improved mental model.

Deliverables:

- Surface pending auto-advance state near the workshop/finalize CTA area in backlog details.
- If a deferred workshop/finalize step is armed:
  - show countdown
  - show exact next mode
  - show cancel
- Ensure the existing CTA labels do not conflict with pending auto-advance state.
  - Example: if “Next Round in 8s” is armed, the workshop area should not simultaneously look like a normal idle “Next Round” action with no indication of pending auto-run.
- If the operator manually triggers the next step while a deferred auto-advance is pending, rely on the existing backend cancellation semantics already described in `api/internal/backlog/research.go` and `api/internal/backlog/workshop_save.go`.

Acceptance criteria for Phase 6:

- The detail page clearly indicates when the next workshop/finalize action is pending automatically.
- The operator can cancel that pending action from the detail page.
- The operator no longer has to race the countdown without any UI explanation.

### Phase 7 — Validation closure and regression protection

Finish by converting the bug report into durable automated coverage.

Deliverables:

- Unit tests for queue semantics:
  - local answered-question pruning
  - within-parent ordering preserved
  - parent-item ordering canonicalized
  - safe index normalization after deletions/snoozes/answers
- Decision Stream component tests:
  - completion state rendering
  - no bounce-back after last answer
  - countdown / cancel rendering when auto-advance is pending
  - explicit exit/skip controls
- Navigator tests:
  - desktop clamping behavior where practical
  - mobile sheet rendering path
- Detail-page tests:
  - workshop panel/header show pending auto-advance state
  - cancel action clears the pending state
- Integration-style UI tests where practical:
  - answer all questions for one item, see completion state, cancel pending advance
  - answer across multiple items, ensure counts shrink and resolved items disappear

## 8. Contract Decisions

### 8.1 Ordering ownership

Ordering is server-owned.

- The UI may continue to preserve within-item order and do defensive rendering checks.
- The UI must not own a second ranking system for parent backlog items in the Decision Stream.

### 8.2 Answer progression

Answer selection never implies navigation.

- Choosing an option updates local state only.
- `Next` / `Done` remains the only progression control.
- Keyboard shortcuts may still invoke explicit progression, but must not be conflated with option selection.

### 8.3 Persistence path

Do not create a new answer API in this slice.

Keep:

1. fetch round content
2. patch selected decision
3. save through existing `/workshop/save`

Rationale:

- `workshopSave` is already the contract that evaluates auto-advance and next-step metadata.
- Replacing it here would create unnecessary contract sprawl.

### 8.4 Completion-state ownership

The Decision Stream completion state is a UI concern built on backend auto-advance metadata.

- It may aggregate per-parent results for display.
- It must not invent new rules for when next steps are allowed.

### 8.5 Shared transition seam

Countdown / cancel / “next step pending” behavior should live behind a shared UI seam.

- Do not duplicate countdown logic in Decision Stream and backlog details separately.
- Reuse the existing cancel endpoint and transition metadata.

### 8.6 Responsive navigator behavior

Desktop and mobile may use different shells, but they must share one interaction contract:

- current parent highlighting
- unresolved count display
- jump-to-parent
- snooze-parent

## 9. Testing Plan

All validation should be automated. Manual checking is useful during development but not sufficient as the close-out criteria.

### UI unit / component coverage

- `ui/src/lib/command-post-utils.test.ts`
  - extend ordering tests to cover canonical parent ordering assumptions
- new tests for a queue helper/reducer if extracted
  - unresolved filtering after local answer success
  - deleted/snoozed/skipped interactions
- `ui/src/components/command-post/DecisionStreamView.test.tsx`
  - entering completion state after last answer
  - completion-state CTA rendering
  - no already-answered question in visible queue after save
  - pending auto-advance countdown / cancel in completion state
  - empty state after all questions resolved without bounce-back
- `ui/src/components/command-post/ScenarioNavigatorPopover.test.tsx` or replacement tests
  - desktop presentation
  - mobile sheet presentation
  - jump and snooze behavior
- backlog-detail tests
  - `WorkshopPanel` / `BacklogNotesPanel` / `HeaderPrimaryAction` behavior when auto-advance pending exists

### API / server coverage

- `api/internal/backlog/backlog_summary` tests
  - pending-question parent groups are ranked deterministically
  - latest-round unanswered decision extraction still preserves in-file order
- if ranking logic is extracted/shared, add focused tests for the shared seam to guarantee summary and pending-questions endpoint stay aligned

### Recommended commands

```bash
cd scenarios/swarm-manager && make test
vrooli scenario test swarm-manager
cd scenarios/swarm-manager/ui && pnpm test -- DecisionStreamView
cd scenarios/swarm-manager/ui && pnpm test -- command-post-utils
cd scenarios/swarm-manager/api && go test ./... -timeout 300s
```

## 10. Rollout / Validation Checklist

- [ ] Decision Stream parent-item ordering is deterministic and no longer depends on `LoadAll(nil)` traversal order.
- [ ] Multiple unanswered decisions within one backlog item remain in latest-round order.
- [ ] Entering Decision Stream forces sufficiently fresh data to prevent stale answered questions from appearing.
- [ ] Successfully answered questions are pruned from the active Decision Stream queue immediately.
- [ ] Counter, progress bar, and navigator counts reflect unresolved questions only.
- [ ] Final answer transitions into a dedicated completion state instead of returning directly to summary.
- [ ] Completion state shows explicit next-step controls and/or pending countdown.
- [ ] Pending auto-advance can be cancelled from Decision Stream when present.
- [ ] Pending auto-advance can be cancelled from backlog details when present.
- [ ] Decision Stream navigator is clamped on desktop and rendered as a sheet on mobile.
- [ ] Existing explicit `Next` / `Done` progression behavior remains intact.
- [ ] No new answer API endpoint or duplicated ranking logic was introduced.
- [ ] Full Swarm Manager tests pass.

## 11. Risks And Mitigations

### Risk 1 — Queue fixes cause index or completion regressions

Why it matters:

- The current hook mixes navigation, persistence, and completion state.
- Naive pruning can create off-by-one errors or accidental premature completion.

Mitigation:

- Extract queue computation into a pure helper with dedicated tests.
- Normalize current index after every queue-shape mutation.

### Risk 2 — Summary and pending-questions ordering drift again later

Why it matters:

- There are already two server paths capable of describing pending questions.

Mitigation:

- Share ranking code between the two handlers instead of duplicating sort logic.
- Add tests that assert alignment.

### Risk 3 — Detail-page pending-state UI fights existing CTA logic

Why it matters:

- Header CTA, workshop panel buttons, and agent-running labels already have their own state rules.

Mitigation:

- Introduce one transition-state seam and make CTA components consume it instead of layering ad hoc conditionals.

### Risk 4 — Mobile navigator shell rewrite introduces accessibility regressions

Why it matters:

- The current popover is simple but limited; the replacement must preserve keyboard/dismiss/focus behavior.

Mitigation:

- Reuse `BottomSheet` / `Drawer` rather than inventing a new shell.
- Add tests for close behavior and action execution.

### Risk 5 — Over-scoping into general Command Post redesign

Why it matters:

- This work can easily drift into unrelated visual or architecture cleanup.

Mitigation:

- Keep scope anchored to decision answering, next-step clarity, and navigator responsiveness only.

## 12. Non-goals / Prohibited Patterns

- Do not introduce a new question-answer API endpoint in this slice.
- Do not add a second UI-only ranking algorithm for parent-item ordering.
- Do not leave stale-cache behavior “fixed” only by reducing `staleTime` without session-level queue pruning.
- Do not duplicate countdown / cancel logic in multiple components.
- Do not add another custom mobile modal/panel implementation for the navigator when shared sheet/drawer components already exist.
- Do not silently auto-trigger progression from option selection.
- Do not mix broad Command Post aesthetic redesign into this implementation.

## 13. Definition Of Done

This work is done only when all of the following are true:

- The Decision Stream is trustworthy: no already-answered questions remain visible or counted once resolved.
- Parent backlog items appear in deterministic ranked order, and within-item decision order is preserved.
- The final-answer experience is explicit, with a completion state that clearly communicates and controls the next step.
- Deferred auto-advance is visible and cancellable in both Decision Stream and backlog details.
- The navigator is professional on desktop and mobile, using shared responsive UI patterns.
- The implementation reuses the existing workshop-save / auto-advance contract rather than bypassing it.
- Automated tests cover the original bug class and the new completion / countdown behavior.
- `make test` and scenario-level tests for `swarm-manager` pass without regressions.
