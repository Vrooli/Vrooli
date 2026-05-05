# Plan A: Initiative Feedback Rescoping UX

## 1. Purpose

Close the rescoping gap inside swarm-manager's existing backlog-item-level execution mode. Two findings drive this plan:

- **Asymmetric ops**: the proposals layer supports `split_item` but has no `merge_items` counterpart — the agent can break an item apart but cannot consolidate two coupled items into one.
- **Zero UI surface for the LLM-judgment cases**: the FeedbackDialog is a free-form text box. Operators have no visible affordance for what kinds of investigation feedback can drive — the supported-ops table lives only in the agent's prompt, invisible to the operator. Worse, the most common reason operators *use* feedback (post-completion gap-and-drift sweeps to find missing work) has no surfaced path at all.

The result is a system where rescoping is *largely possible but not discoverable*, which manifests as the operator believing rescoping is unsupported (it mostly is) and either avoiding it or escaping the harness entirely.

This plan adds the missing `merge_items` op, audits and surfaces the operator-facing op set in the skill prompt, and replaces the current free-form-only dialog with a **selection-driven Quick Actions surface** scoped to operations that genuinely need LLM judgment (the rest belong on direct item / graph UIs and are explicitly out of scope here).

This is a **companion** to Plan B (`path:docs/plans/swarm-manager-initiative-operating-mode-implementation.md`), not a replacement. Plan A improves backlog-item-level mode for cases where the items are correctly chosen as the unit of execution but were initially mis-scoped or have drifted from code reality. Plan B introduces a second execution mode for cases where backlog-items are not the right unit at all. The conceptual framing for both lives in [`path:scenarios/swarm-manager/docs/concepts/EXECUTION-MODES.md`](../../scenarios/swarm-manager/docs/concepts/EXECUTION-MODES.md).

## 2. Required Reading

```bash
prompt-manager skill read cli-steer api-steer utils-unification seam-discovery-and-enforcement
prompt-manager skill read implementation-plan-authoring
prompt-manager skill read swarm-manager-initiative-feedback
prompt-manager skill read swarm-manager-initiative-context
```

Required file reads (no chat context required to follow this plan):

- `path:scenarios/swarm-manager/docs/concepts/EXECUTION-MODES.md` — framework framing this plan operates within.
- `path:scenarios/swarm-manager/api/internal/proposals/types.go` — current ops, mutation shape, validation envelope.
- `path:scenarios/swarm-manager/api/internal/proposals/apply.go` and `apply_test.go` — current apply flow including `split_item`'s atomic create-then-archive pattern that `merge_items` will mirror.
- `path:scenarios/swarm-manager/api/internal/proposals/validate.go` — per-op validation (in-progress gating used by `validateInterrupt` is the precedent for merge's strict-rejection rule).
- `path:scenarios/swarm-manager/api/routes_feedback.go` — wiring of the feedback agent and event-emit shape (`feedbackEventEmitter`, `proposalEventTarget`).
- `path:scenarios/swarm-manager/api/internal/eventlog/types.go` — `ProposalAppliedPayload`; merge gains optional `Sources []string`.
- `path:scenarios/swarm-manager/ui/src/components/initiative/feedback-dialog.tsx` — current dialog (free-form text + image attachments + round-type chips).
- `path:scenarios/swarm-manager/ui/src/components/initiative/feedback-panel.tsx` and `feedback-round-card.tsx` — current proposal-rendering surfaces (downstream of dialog).
- `path:scenarios/swarm-manager/ui/src/types/feedback.ts` — wire types.
- `path:scenarios/swarm-manager/ui/src/consts/selectors.ts` — centralized selector registry; new affordances register here.
- `path:scenarios/prompt-manager/store/skills/packs/core/swarm-manager-initiative-feedback/SKILL.md` — agent prompt; receives merge row, intent-mapping table, well-scoped-item examples, requested-actions interpretation guidance, and a fix to the existing-but-incorrect split-retargeting claim on line 106.

## 3. Problem Statement

### Op-level asymmetry

`path:api/internal/proposals/types.go` declares 10 ops: `add_item`, `update_item`, `change_status`, `change_priority`, `add_edge`, `remove_edge`, `move_initiative`, `archive_item`, `interrupt_in_progress`, `split_item`. Of these, `split_item` lets one item become N. There is no `merge_items` op letting M items become one. Today, an agent attempting to express "merge execute/foo and execute/bar into execute/foobar" must emit (a) an `add_item` for the new merged item, (b) `archive_item` for each source, and (c) `add_edge`/`remove_edge` mutations to retarget every dependent of every source. This is correct but not atomic — partial application leaves the graph in a broken intermediate state — and is harder for the agent to author correctly than `split_item`.

### Operator-side discoverability

`path:ui/src/components/initiative/feedback-dialog.tsx` exposes a free-form textarea, image attachments, and three round-type chips (`feedback`, `research` (disabled), `note`). It does not surface what kinds of feedback are productive, what operations the agent can investigate, or any path for selecting the items the operator wants to talk about. The operator must already know what feedback can do to direct it usefully.

### The original use case has no surfaced path

The most common operator use of feedback today is *gap and drift analysis*: an initiative's items are mostly done, but the operator wants the agent to investigate the initiative's *actual current code state* and propose new items for missing work (tests not yet written, greenfield cleanup, follow-ups) and updates to items that have drifted from what the code now does. This is a high-value LLM-judgment task, and the dialog gives it no surfaced affordance whatsoever.

### Skill-prompt accuracy and completeness

`path:scenarios/prompt-manager/store/skills/packs/core/swarm-manager-initiative-feedback/SKILL.md` lists supported ops in a table. The table is correct but does not connect ops to *operator-phrased intents* — there is no guidance like "if the user says 'combine these items', use `merge_items`". This makes the agent's mapping from feedback text to mutations less reliable than it could be.

Additionally, the existing `split_item` row (line 106) currently claims "Dependents of the source repoint to the first new item automatically; emit add_edge / remove_edge for the rest." This is **false** — `apply.go::applySplit` explicitly does *not* retarget dependents (see the OpSplitItem comment in `types.go`). The skill prompt has been giving the agent a wrong rule. This plan corrects it as part of the same edit.

The skill prompt also does not contain examples of *well-scoped items*, which the redesigned UX depends on (see §7 Phase 3): non-prescriptive templates trust the agent to apply guidelines, and those guidelines have to exist in the prompt.

### Why this matters now

The two sandboxing initiatives (`agent-sandbox-audit-foundation`, `protected-agent-sandboxing`) completed 2026-04-27 surfaced a separate failure mode (coupled items at scale; addressed in Plan B). But during walk #5 it became clear that even *within* backlog-item-level mode, the operator's mental model of "feedback can only reorder items and add notes" is wrong — and the existing UX accidentally reinforces that mental model by giving zero affordances for the things the operator does most. Closing that gap unblocks rescoping and gap-analysis as routine parts of working with swarm-manager rather than workarounds.

## 4. Scope

### In scope

- Add `merge_items` op to the proposals package (types, apply, validation), with edge auto-retargeting to the merged item and strict in-progress-source validation rejection.
- Update the swarm-manager-initiative-feedback skill prompt to:
  - Add the `merge_items` row.
  - **Fix** the incorrect dependents-retargeting claim on the existing `split_item` row.
  - Add an intent-phrase mapping for common operator language (covers the free-prose path).
  - Add a `<requested_actions>` interpretation section explaining how to read the new XML envelope the dialog can emit.
  - Add a well-scoped-item criteria section the agent can anchor on when proposing splits / merges / new items.
- Replace the current free-form-only FeedbackDialog with a **selection-driven Quick Actions surface**:
  - Item-selection picker (collapsed by default, with `Select all` / `Select none` toggles).
  - Five Quick action buttons gated by selection count and combinability rules.
  - XML envelope assembly when one or more actions are selected (raw textarea passthrough when zero actions are selected).
- Add a "What can feedback do?" help block listing the agent's full mutation surface in plain language.
- Add `Sources []string` field to `eventlog.ProposalAppliedPayload` so merge audits record source items.
- Cover the new op and UX with automated tests (Go unit/integration for proposals; Vitest/RTL for the dialog).
- Update the api-endpoints reference doc to reflect the new op.

### Out of scope

- Implementation of initiative-level mode (Plan B).
- Changes to the workshop loop or workshop skills (workshop is per-item; feedback is initiative-scoped).
- Any change to the `full_graph` form (target-state submission) — that path remains untouched.
- Any change to the review skill or review apply path.
- Any change to attachment handling, IndexedDB persistence, or lock semantics.
- Any direct-action affordance for archive / change-priority / single-edge add-or-remove / move-initiative / change-status / interrupt-in-progress / single-item update-title-or-description. These do not require LLM judgment; their right home is direct UI on the item or graph surfaces. They are explicitly *not* added to the feedback dialog as quick actions.
- Decision-rejection / contrarian flows on the meta-optimization side.
- An item-picker autocomplete inside the textarea (the selection picker handles ref capture; freehand `kind/name` typing is not optimized further).
- A `template_used` telemetry field on `feedback_round_created` events (deferred — track later if usage signal becomes load-bearing).

## 5. Current Technical Context

### Proposals package

`path:api/internal/proposals/types.go` defines `Op`, `Mutation`, `ItemSpec`, `ItemPatch`, `Proposal`, and `Source`. `AllOps()` returns the canonical list. The `split_item` op uses the `Into []ItemSpec` field to carry new items, with the source named in `Target`. Apply is atomic: child creates run first, source archive runs last, and any failure rolls back already-created children. Dependents of the source are *not* automatically retargeted.

`path:api/internal/proposals/apply.go` contains the per-op handlers and the orchestration that ensures atomicity. The Applier holds `Store BacklogStore`, `Assigner InitiativeAssigner`, `Creator ItemCreator`, `cancel ExecutionCanceller`, `invalidator GraphInvalidator`, and `events EventEmitter`.

`path:api/internal/proposals/apply_test.go` covers each op including split's rollback behavior. New op tests follow this shape.

`path:api/internal/proposals/validate.go` contains per-op validators. `validateInterrupt` already gates on `state.InProgressRefs` — that is the precedent for merge's strict-rejection-on-in-progress rule.

### Feedback wiring

`path:api/routes_feedback.go::registerFeedbackRoutes` wires:
- `proposals.Applier` over the backlog store + initiatives assigner + execution canceller adapter.
- `feedback.Service` over a spawner that loads the initiative-feedback skill via prompt-manager, hydrates the prompt with current graph + items + prior rounds + attachments, and spawns through agent-manager (with activity tracking).

The `feedbackEventEmitter` records each applied mutation against the affected backlog ref. `proposalEventTarget()` decides which ref the event attaches to per op (current logic: `add_item` uses the new item's ref; `add_edge`/`remove_edge` use `From`; everything else — including `split_item` — uses `Target`).

### Eventlog payload

`eventlog.ProposalAppliedPayload` (in `path:internal/eventlog/types.go`) currently has no `Sources` field. Merge gains an optional `Sources []string` so each source-ref's per-item history records "this item was merged into X" (sources are recorded as a payload field; the event itself attaches to the merged item per the convention below).

### UI

`path:ui/src/components/initiative/feedback-dialog.tsx` is a controlled dialog component: text state, attachment state via `useIndexedDBAttachments`, blocker state from preflight lock query, type state (`feedback` | `research` | `note`). Submission goes through `feedbackService.start`. The dialog already handles draft persistence, lock-conflict UX, and override.

`path:ui/src/components/initiative/feedback-panel.tsx` renders the rounds list and the operator's checklist UI for proposed mutations. It is the *downstream* surface where mutations become accept/reject decisions; this plan does not modify it (the UI changes are in the entry-point dialog, not the proposal review).

`path:ui/src/consts/selectors.ts` centralizes test-id strings under a `feedback` namespace; new affordances register here.

### Skill prompt

`path:scenarios/prompt-manager/store/skills/packs/core/swarm-manager-initiative-feedback/SKILL.md` is rendered by prompt-manager with variable substitution at spawn time. The "Supported ops" table and the "Rules you must follow" sections are the contract surface.

## 6. Target End State

After this plan lands:

- The proposals package has 11 ops; `merge_items` is fully implemented, validated, and tested. Edges from non-source items to/from sources are auto-retargeted to the merged item; intra-source edges are dropped. Merge rejects at validation if any source is `in_progress` (operator must emit `interrupt_in_progress` first).
- The skill prompt advertises `merge_items` alongside the existing ops, includes a phrase-to-op intent-mapping table, includes a `<requested_actions>` interpretation section, includes well-scoped-item examples, and corrects the existing wrong claim about split's edge-retargeting behavior.
- The feedback dialog opens to a selection-driven surface: type chips, an item-selection picker, a Quick actions row of five LLM-judgment buttons, a "What can feedback do?" help block, the existing textarea, and the existing attachment / submit row. Selecting actions composes an XML envelope; selecting no actions falls through to today's raw-text path.
- All new behavior is covered by automated tests (Go unit/integration + Vitest/RTL).
- `path:docs/reference/api-endpoints.md` lists the new op.
- swarm-manager scenario restarts cleanly (`vrooli scenario restart swarm-manager`) and `make test` passes.

## 7. Implementation Strategy

### Phase 1 — Proposals: add `merge_items` op (Go)

1. **types.go**
   - Add `OpMergeItems Op = "merge_items"` to the const block; append to `AllOps()`.
   - Extend `Mutation` with a `Sources []string` field (kind/name refs of items to merge). The merged target is described by `Item *ItemSpec` (reusing the existing field). `Target` is left empty for this op.
2. **validate.go**
   - Add `validateMergeItems`. Enforce: `len(Sources) >= 2`; no duplicate sources; every source resolves to a current member of the initiative (`state.HasNode`); `Item != nil` and passes `validateItemSpec`; `Item.Ref()` does not collide with any existing non-source item or any item already staged by an earlier mutation in the batch; **strict** rejection if any source is in `state.InProgressRefs` (operator must emit `interrupt_in_progress` first).
   - Wire into `validateMutation` switch.
3. **apply.go**
   - Implement `applyMergeItems`:
     a. Create the merged item via the existing `applyAddItem` path so it inherits the `backlog.SourceProposal` event + attribution.
     b. For each source: enumerate its outbound and inbound edges in the current graph (read from the in-memory state already passed to Apply via the per-mutation re-validate path; Apply does not currently load edges, so add a small `currentState` accessor on the Applier or pass `current` through to `applyOne` — choose the smaller patch). Retarget any edge whose other endpoint is *not* among Sources to point to the merged item; drop edges between Sources entirely.
     c. Archive each source via `applyArchive`.
     d. On any failure: archive the merged item if created, restore source-archive state if any source archive completed (re-clearing `ArchivedAt`), restore retargeted edges, and return the original error wrapped — mirroring the rollback discipline in `applySplit`.
   - Wire into the per-op switch in `applyOne`.
   - Note the in-progress concern is handled at validate, not apply: by the time apply runs, no source is in-progress.
4. **proposalEventTarget** (in `routes_feedback.go`)
   - Add a `case proposals.OpMergeItems` returning `m.Item.Ref()` so the audit event attaches to the merged item. Existing split behavior (falls through to `m.Target`) is intentionally left alone — convention: events attach to the *primary new state*, and split's primary new state is N children, so attaching to one ref would be misleading; merge's primary new state is one merged item, so the merged ref is the right anchor.
5. **feedbackEventEmitter** (in `routes_feedback.go`)
   - Populate `payload.Sources = m.Sources` so per-source history queries record "this item was merged into X" (the event itself attaches to the merged item; Sources is a payload field).
6. **eventlog/types.go**
   - Add `Sources []string \`json:"sources,omitempty"\`` to `ProposalAppliedPayload`.

### Phase 2 — Skill prompt updates

In `path:scenarios/prompt-manager/store/skills/packs/core/swarm-manager-initiative-feedback/SKILL.md`:

1. **Fix the wrong split claim**. Edit the `split_item` row of the Supported ops table: dependents of the source are *not* automatically retargeted; if the agent wants to repoint dependents, it must emit explicit `add_edge` / `remove_edge` mutations alongside the split.
2. **Add the `merge_items` row** with example wire shape:
   ```json
   {
     "id": "m1",
     "op": "merge_items",
     "sources": ["execute/sandbox-aware-cli", "execute/sandbox-lifecycle-coord"],
     "item": { "kind": "execute", "name": "sandbox-runtime-coord", "title": "...", "description": "...", "priority": 3, "effort": "M" },
     "rationale": "These items both refactor the workspace-sandbox runtime path; folding them into one item avoids partial-state intermediate executions."
   }
   ```
   Document the apply-layer behavior: edges between sources are dropped; edges from non-sources to/from sources auto-retarget to the merged item; the merged item enters as `backlog`. Document the validation contract: at least two sources, all current initiative members, none in `in_progress` (propose `interrupt_in_progress` first if needed), merged item ref must not collide.
3. **Add an "Intent-to-op mapping" subsection** for the free-prose path:

   | Operator says... | Op |
   |---|---|
   | "split this", "break X apart", "X is two things", "scope is too big" | `split_item` |
   | "merge these", "combine X and Y", "fold X into Y", "consolidate", "this is one item not two" | `merge_items` |
   | "this isn't relevant", "drop X", "remove X" (note: archive, never `remove_item`) | `archive_item` |
   | "X belongs in initiative Y", "this is wrong-initiative" | `move_initiative` |
   | "stop X", "cancel the running X" | `interrupt_in_progress` |
   | "X depends on Y", "Y has to come first" | `add_edge` |
   | "Y doesn't actually depend on X" | `remove_edge` |
   | "rename X", "X's title is wrong", "fix X's description" | `update_item` |
   | "raise/lower X's priority" | `change_priority` |
   | "X is ready" / "X needs more research" (only non-terminal targets) | `change_status` |
   | "we need a new item for Z", "missing work for Z" | `add_item` |

4. **Add a "Requested actions" interpretation section** explaining how to read the optional `<requested_actions>` XML envelope the dialog can emit. Each `<action name="...">` is a *lens* the operator wants the agent to look through, not a prescriptive command. Document the five action names (see Phase 3) and the agent's job under each — emphasize the *non-prescriptive* framing: "The operator told you what to look for, not what to do. Apply the well-scoped-item criteria and propose mutations consistent with them. The agent decides counts, splits, and merges; the operator decides which proposals to accept."
5. **Add a "Well-scoped item criteria" section** the agent anchors on. Short list (each ≤ one line):
   - One agent run can plausibly converge it to `plan.md` in one workshop pass.
   - Acceptance is testable in isolation, ideally with one or two automated tests.
   - The acceptance globs cover one cohesive code area; an item that touches `path:scenarios/foo/**` and `path:scenarios/bar/**` is suspect.
   - Description fits in a paragraph; if it needs sections, it's probably two items.
   - Title names the *change*, not the *area* ("Add merge_items op to proposals", not "proposals work").
   - No internal ordering — if step 1 must complete before step 2 can be designed, those are two items joined by a `depends_on` edge.
6. **Reinforce the existing rule against `remove_item`** (which doesn't exist).
7. **Reinforce: never auto-cancel running executions implicitly.** Always propose `interrupt_in_progress` as its own mutation if the operator's intent requires it.

### Phase 3 — UI: selection-driven Quick Actions in FeedbackDialog

The redesigned dialog layout (top to bottom):

1. Existing intro paragraph and type chips.
2. **Item-selection picker** (new). Collapsed-by-default summary button: `Target items — N of M selected ▾`. Expanded: scrollable checklist of the initiative's items (one row per ref with `kind/name` and short title), `Select all` / `Select none` toggles at the top. Selection state persists per-initiative under the same draft-key namespace as the textarea. Disabled when round type is `Note`.
3. **Quick actions row** (new). Five buttons, each gated by selection count and combinability:
   - **Split oversized items** — enabled when ≥1 item selected and Merge is not selected.
   - **Merge coupled items** — enabled when ≥2 items selected and Split is not selected.
   - **Identify missing work** — enabled with any selection (including zero).
   - **Reconcile with code drift** — enabled with any selection.
   - **Reframe scope** — enabled solo only (selecting it deselects all other Quick actions; selecting any other deselects Reframe).
   Each button is a toggle; multiple non-mutually-exclusive buttons can be active simultaneously. The five action selections persist in the draft-key namespace alongside item selection.
4. **"What can feedback do?" help block** (new). Collapsible, default closed, persisted. Content:
   - Short paragraph: "The agent reads your selection, your text, and any attachments, then proposes a checklist of mutations you accept or reject in the proposal panel. Quick actions are starting points for common investigation tasks — the agent can also propose any of the mutations below from your free-form text."
   - Bulleted list of operator-facing op intents in plain language: split, merge, archive, change priority, change status, add-or-remove edge, move to other initiative, interrupt running execution, update title/description, add new item.
5. Existing active-agent blocker notice.
6. Existing textarea (placeholder text adapts to selection state — e.g., "Add anything else the agent should know…" when an action is selected vs. the existing default when nothing is selected).
7. Existing attachment / submit row.

#### XML envelope assembly

When at least one Quick action is selected, the submission text sent to the agent is wrapped:

```
<selection>
  <item ref="execute/foo" />
  <item ref="execute/bar" />
</selection>

<requested_actions>
  <action name="identify_missing_work" />
  <action name="reconcile_with_code_drift" />
</requested_actions>

<user_note>
{{ free-form text from textarea, may be empty }}
</user_note>
```

When zero Quick actions are selected (and zero items are selected), the submission text is the raw textarea content — exactly as today. When items are selected but no actions are picked, the submission still wraps in the envelope (so the agent knows the operator scoped to specific items) but `<requested_actions>` is empty.

The five action names emitted into the envelope (stable identifiers): `split_oversized`, `merge_coupled`, `identify_missing_work`, `reconcile_with_code_drift`, `reframe_scope`. The skill prompt's Requested-actions section maps these to the agent's job under each.

#### Combinability enforcement (UI)

- `split_oversized` and `merge_coupled` are mutually exclusive — selecting one deselects the other.
- `reframe_scope` is solo — selecting it clears the other four; selecting any other clears it.
- `identify_missing_work` and `reconcile_with_code_drift` stack freely with each other and (subject to the rules above) with split or merge.

#### Note-type interaction

When round type is `Note`, the item-selection picker is disabled, the Quick actions row is hidden, and the help block remains visible (informational). Notes still go through the raw-text path.

#### Selectors

Each new affordance gets a stable `data-testid` registered in `consts/selectors.ts`:
- `selectors.feedback.dialogTargetPicker`, `dialogTargetPickerToggleAll`, `dialogTargetPickerItem`
- `selectors.feedback.dialogQuickActionSplit`, `dialogQuickActionMerge`, `dialogQuickActionIdentifyMissing`, `dialogQuickActionReconcile`, `dialogQuickActionReframe`
- `selectors.feedback.dialogHelpBlock`

### Phase 4 — Tests

1. **Go (proposals package)**:
   - `Applier_MergeItems_HappyPath` — 3 sources merged into 1, edges retargeted, sources archived, merged item created with provided spec.
   - `Applier_MergeItems_RetargetsExternalEdges` — outbound and inbound edges to non-source items all retargeted; verify edge counts and endpoints.
   - `Applier_MergeItems_DropsIntraSourceEdges` — sources have edges among themselves; verify those are not preserved as edges to/from the merged item (no self-loops).
   - `Applier_MergeItems_DeduplicatesRetargetedEdges` — two sources both depend on C; merged item has exactly one depends_on C edge.
   - `Applier_MergeItems_RollbackOnSourceArchiveFailure` — inject failure on second source archive; verify merged item is removed, first source is restored, retargeted edges restored.
   - `Applier_MergeItems_RollbackOnRetargetFailure` — failure during edge retargeting rolls back the merged item.
   - `Validate_MergeItems_RejectsSingleSource` — `len(Sources) < 2` rejected.
   - `Validate_MergeItems_RejectsDuplicateSources` — duplicate ref in Sources rejected.
   - `Validate_MergeItems_RejectsCollidingTarget` — `Item.Ref()` collides with non-source existing item, rejected.
   - `Validate_MergeItems_RejectsCollisionWithBatchStaged` — collision with an item staged earlier in the same proposal, rejected.
   - `Validate_MergeItems_RejectsInProgressSource` — at least one source in `state.InProgressRefs`, rejected with a message directing the agent to emit `interrupt_in_progress` first.
   - `Validate_MergeItems_RejectsNonMemberSource` — source not in `state.Nodes`, rejected.
   - `proposalEventTarget` test extended to cover `OpMergeItems` returning the merged item ref.
   - `feedbackEventEmitter` test extended to cover `Sources` payload field round-trip.
2. **Go E2E** (`path:api/e2e_initiative_feedback_test.go` style): submit a feedback round whose proposal includes a `merge_items` mutation, accept it via the proposal handler, verify the resulting initiative graph state (merged item present, sources archived, edges retargeted).
3. **Vitest/RTL** (`feedback-dialog.test.tsx`):
   - Item-selection picker: open/close persists; `Select all` / `Select none` work; selection persists across re-render.
   - Quick action gating: Split disabled when 0 items selected; Merge disabled when <2 items; Reframe-solo behavior; Split/Merge mutual exclusion; Identify+Reconcile stack.
   - Round-type=Note hides Quick actions and disables item picker.
   - Help block toggles; default closed; state persists.
   - Submission body: with zero actions selected, body equals raw textarea (no XML wrap); with ≥1 action selected, body is the XML envelope with `<selection>` reflecting checked items, `<requested_actions>` listing canonical action names, and `<user_note>` carrying textarea content (possibly empty).
   - All new `data-testid` selectors are stable.
4. **Type-check**: `cd scenarios/swarm-manager/ui && npm run typecheck`.

### Phase 5 — Docs and scenario restart

1. Add the `merge_items` row (with shape, validation rules, and edge-handling semantics) to `path:scenarios/swarm-manager/docs/reference/api-endpoints.md` (proposals/feedback section; create the section if it does not exist).
2. Update `path:scenarios/swarm-manager/docs/internal/SEAMS.md` if the proposals seam is documented there (verify; only edit if a seam entry exists for the apply layer).
3. `vrooli scenario restart swarm-manager` and verify health (`make test` from `path:scenarios/swarm-manager/`). Plan must leave the scenario healthy.

## 8. Contract Decisions

- **Op name is `merge_items` (plural).** Mirrors `split_item` (singular source → multiple targets) by inversion.
- **Wire shape**: `{ id, op: "merge_items", sources: ["kind/a", "kind/b"], item: ItemSpec, rationale }`. `target` is omitted.
- **Edge handling — asymmetric with split, intentionally**: merge auto-retargets external edges to the merged item and drops intra-source edges; split does *not* auto-retarget. Rationale: merge is operationally a graph-cluster collapse where the agent already knows the full source set, so atomic retargeting matches operator intent; split's children are typically intentionally distinct successors and the agent should be explicit about which one inherits which edge.
- **Existing skill-prompt claim about split retargeting is fixed**: the prompt is corrected to match actual apply behavior (no auto-retargeting).
- **Status of merged item**: enters as `StatusBacklog` (consistent with `add_item`).
- **Priority/effort/tags/acceptance globs of merged item**: come from `Item *ItemSpec`. The agent synthesizes; sources' values are not auto-merged.
- **Event emission**: one `EventBacklogProposalApplied` per merge mutation, attached to the merged item's ref. Sources are recorded in `ProposalAppliedPayload.Sources`. Split's existing event-target behavior (attaches to `m.Target` = source) is unchanged.
- **In-progress-source policy**: validation **rejects** merge if any source is in `state.InProgressRefs`. The agent must emit `interrupt_in_progress` as a separate prior mutation if interruption is the operator's intent. No silent auto-cancel in apply.
- **Rollback ordering**: create merged → retarget edges → archive sources, in that order. Rollback reverses: un-archive sources → restore retargeted edges → archive the merged item. Atomicity is best-effort the same way `split_item` is.
- **Quick actions are LLM-judgment only.** Direct ops (archive, change_priority, single edge add/remove, move_initiative, change_status, interrupt_in_progress, single-item update_item) are not surfaced as Quick actions — their right home is direct UI on the item or graph (out of scope here).
- **Quick action set is five**: `split_oversized`, `merge_coupled`, `identify_missing_work`, `reconcile_with_code_drift`, `reframe_scope`. **Investigate dependencies** is intentionally deferred; add later if it earns its slot.
- **Default item selection is none**. Whole-initiative scope is the implicit default; explicit selection narrows the agent's attention.
- **Combinability rules** (UI-enforced and skill-prompt-documented): Split⊥Merge; Reframe is solo; Identify-missing and Reconcile-with-drift stack with each other and with Split or Merge.
- **XML envelope is emitted only when ≥1 Quick action is selected**, or when items are selected without an action (selection alone is a meaningful narrowing). Pure free-prose with zero selection is unchanged from today.
- **Action names in the envelope are stable identifiers** (`split_oversized`, etc.), not human-readable labels. The skill prompt's Requested-actions section maps stable identifiers to agent guidance.
- **Templates are non-prescriptive.** The agent's skill prompt anchors on well-scoped-item criteria; the dialog does not tell the agent how many splits to produce or which item to keep.

## 9. Testing Plan

All verification is automated. No manual test checklists.

### Go (`path:scenarios/swarm-manager/`)

- Unit tests for `proposals.Applier.applyMergeItems` covering happy path, all rollback paths, all edge-handling invariants, and all validation rejections. Run via `cd scenarios/swarm-manager/api && go test ./internal/proposals/... -timeout 120s`.
- Unit test extension for `proposalEventTarget` and `feedbackEventEmitter` (Sources payload).
- E2E test in `path:api/e2e_initiative_feedback_test.go` style covering "submit feedback with merge mutation → accept → graph reflects merge". Run via `cd scenarios/swarm-manager/api && go test ./... -run E2E -timeout 300s`.
- Scenario-wide test pass: `cd scenarios/swarm-manager && make test`.

### TypeScript (`path:scenarios/swarm-manager/ui/`)

- Vitest tests for `feedback-dialog.test.tsx` covering item picker, Quick action gating and combinability, help block, and XML envelope assembly. Run via `cd scenarios/swarm-manager/ui && npm test -- feedback-dialog`.
- Type-check pass: `cd scenarios/swarm-manager/ui && npm run typecheck`.

### Skill validation

- Render the updated `swarm-manager-initiative-feedback` skill via the prompt-manager CLI (verify the canonical command during implementation; `prompt-manager skill read` is a reasonable starting point) and confirm the rendered output contains the new `merge_items` row, the corrected split row, the intent-mapping table, the requested-actions interpretation section, and the well-scoped-item criteria.

### Cross-scenario

- `vrooli scenario restart swarm-manager` succeeds.
- `vrooli scenario test swarm-manager` returns a clean pass.
- Adjacent scenarios (`prompt-manager`, `agent-manager`) baseline pass (no regressions).

## 10. Rollout/Validation Checklist

- [ ] `merge_items` op present in `AllOps()` with all validation rules including in-progress strict rejection.
- [ ] `apply_test.go` tests pass for happy path, all rollback paths, edge auto-retarget, intra-source-edge drop, and edge dedupe.
- [ ] `proposalEventTarget` returns merged item ref for `OpMergeItems`; split's existing behavior unchanged.
- [ ] `ProposalAppliedPayload.Sources` round-trips through the eventlog emitter.
- [ ] Skill prompt advertises `merge_items` with intent-mapping table, requested-actions interpretation, well-scoped-item criteria, and the corrected split row.
- [ ] FeedbackDialog renders item-selection picker, five Quick action buttons with documented gating and combinability, and the help block.
- [ ] XML envelope is emitted when ≥1 action or ≥1 item is selected; raw text passes through otherwise.
- [ ] Vitest tests cover all UI affordances.
- [ ] `path:docs/reference/api-endpoints.md` updated to list `merge_items`.
- [ ] `path:scenarios/swarm-manager` scenario passes `make test`.
- [ ] `vrooli scenario restart swarm-manager` succeeds; the dialog renders the new affordances at runtime.
- [ ] No other scenarios regress (`vrooli scenario test prompt-manager`, `vrooli scenario test agent-manager` baseline pass).

## 11. Risks and Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| `merge_items` rollback path leaves the graph inconsistent under partial filesystem failure | Medium | High | Mirror `split_item`'s tested rollback ordering; add explicit retarget-rollback (record original edges before retargeting); cover every failure injection point with a unit test. |
| Edge auto-retargeting silently drops a dependency the operator wanted preserved | Low | Medium | Intra-source edges are *intentionally* dropped (collapsed into the merged item); external edges retarget atomically. Skill prompt instructs agents to summarize source-item context into the merged item's description so the operator sees what's being collapsed before accepting. |
| Strict in-progress rejection feels like an obstruction in flows where the operator wants merge + cancel together | Medium | Low | The agent can emit `interrupt_in_progress` followed by `merge_items` in one proposal; the user reviews both checklist items before accepting. Skill prompt's intent-mapping table documents the pattern. |
| XML envelope confuses the agent when free-prose feedback was the prior contract | Medium | Medium | Skill prompt's Requested-actions section explicitly documents the envelope. Envelope is only emitted when the user takes an action that asks for it; pure free-prose path is unchanged. |
| Operators ignore the new affordances and continue typing free-form text | Low | Low | Acceptable; structured inputs are additive. Free-prose is reinforced by the intent-mapping table on the agent side. |
| `merge_items` makes it easier to lose context that lived in source items' `plan.md` / workshop history | Medium | Medium | Skill prompt rule: when proposing `merge_items`, the merged item's description must explicitly reference what each source contributed. |
| Skill-prompt growth pushes initiative-feedback agent runs past token budgets | Low | Medium | New sections (intent-mapping, requested-actions, well-scoped-item criteria) are short — net token addition < 600. Verify with a render-size check during implementation. |
| Reframe-scope solo behavior surprises operators who expected to combine it | Low | Low | UI clears other selections visibly when Reframe is picked; help-block text explains it's holistic and incompatible with prescriptive lenses. |
| Item-selection picker becomes unwieldy on initiatives with many items | Low | Low | Picker is collapsed by default; expanded view scrolls; `Select all` / `Select none` keep bulk operations cheap. Filtering / search is deferred until pain shows up. |
| Skill-prompt fix to the split retargeting claim breaks an agent prompt that some downstream test depends on | Low | Medium | Search for tests that pin the exact split-row text and update; the corrected claim matches actual code, so any test asserting the wrong text was already buggy. |

## 12. Non-goals and Prohibited Patterns

- **No `remove_item` op.** Archive remains the way.
- **No skill-prompt rewrite.** This plan adds rows / sections to existing tables; it does not restructure the skill.
- **No changes to the workshop loop.** Workshop is per-item refinement; feedback is initiative-scoped graph edits.
- **No backwards-compatibility shim for old proposals missing `sources`.** Greenfield: the new op is rejected if `sources` is absent; old proposal envelopes never had `merge_items` so there is nothing to migrate.
- **No new wire form.** `mutation_list` and `full_graph` remain the only two forms; `merge_items` is just another op inside `mutation_list`.
- **No "merge into existing item" variant.** Merging M sources into one of the sources (rather than into a brand-new item) is intentionally not supported — it complicates rollback and introduces ambiguity about which source's history "wins". Operators who want this semantic can `update_item` on the chosen survivor and then `archive_item` the others.
- **No partial-merge on validation failure.** All sources must be valid initiative members and not in_progress at submit time, or the whole mutation rejects.
- **No silent auto-cancel of in-progress sources at apply time.** Validation rejects; the agent must emit `interrupt_in_progress` explicitly.
- **No direct-action Quick actions in the feedback dialog.** Archive, priority, single-edge ops, move-initiative, status flips, interrupt, single-item title/description updates — these are direct UI elsewhere, not feedback. The feedback dialog's Quick actions are LLM-judgment investigations only.
- **No item-picker autocomplete in the textarea.** Selection picker handles ref capture; freehand `kind/name` typing is not optimized.
- **No prescriptive templates.** Quick actions describe lenses, not commands; the agent applies well-scoped-item criteria from the skill prompt.
- **No UI work in `feedback-panel.tsx`.** The proposal review UI handles `merge_items` via the same generic checklist row that all ops use; if the row needs to display `sources`, add it to the generic mutation renderer rather than a merge-specific component.
- **No `template_used` telemetry field on `feedback_round_created` events.** Deferred; revisit only if usage signal becomes load-bearing.

## 13. Definition of Done

The plan is done when **all** of the following hold:

1. `OpMergeItems` is implemented in `proposals/types.go`, `proposals/validate.go`, `proposals/apply.go`, and registered in `proposalEventTarget` (in `routes_feedback.go`).
2. `eventlog.ProposalAppliedPayload.Sources` exists and is populated by `feedbackEventEmitter` for merge mutations.
3. All Phase 4 tests are written, present in the repo, and pass.
4. The swarm-manager-initiative-feedback skill advertises `merge_items` with the intent-mapping table, the corrected split row, the requested-actions interpretation section, and the well-scoped-item criteria.
5. `FeedbackDialog` renders the item-selection picker, the five Quick action buttons (with documented gating and combinability), the help block, and assembles the XML envelope on submission per §7 Phase 3 — all covered by automated tests.
6. `path:docs/reference/api-endpoints.md` lists `merge_items`.
7. `cd scenarios/swarm-manager && make test` exits 0.
8. `vrooli scenario restart swarm-manager` succeeds; the dialog at runtime shows the new affordances.
9. No regressions in adjacent scenarios (`prompt-manager`, `agent-manager`) — verified by their respective `vrooli scenario test` runs.
10. The plan does not require manual verification — every check in the Rollout/Validation Checklist is satisfied by an automated test or a CLI exit code.
