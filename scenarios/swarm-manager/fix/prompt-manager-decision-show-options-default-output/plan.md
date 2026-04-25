# Implementation Plan: decision-show options block in default human output

## Purpose

Fix `prompt-manager team decision-show <team> <id>` so its default (human) output displays the `options` block whenever a decision has options. Today operators must re-run with `--json` to see option keys, breaking the human-first CLI-consumption principle from `skill-principles` and dragging vision walks by ~10s per decision in Phase 5.5.

Per workshop decision **d3 = B**, scope extends to the web UI's decision-detail view if it shares the same gap. The API already exposes options correctly (`--json` proves it) and is out of scope.

This is greenfield work. No compatibility shims, no legacy wrappers — just change the formatter (and UI view if needed).

## Required Reading

```bash
prompt-manager skill read cli-steer skill-principles scientific-debugging
```

## Problem Statement

`prompt-manager team decision-show <team> <id>` currently prints (in human/default mode):
- decision (the proposal/question)
- rationale
- context
- status

It omits the `options` array entirely. Consequences:
- Operator cannot tell whether the decision has options or is option-less without re-running with `--json`.
- Operator cannot see which option keys (A/B/C…) are valid for `--selected` without re-running with `--json`.
- The `recommended: true` flag set by the workshop agent is invisible in the default flow.

The web UI's decision-detail view may share the same gap; this is verified during Phase 1 below.

Surfaced 2026-04-24 vision walk during Phase 5.5 (executing decisions).

## Scope

**In scope:**
- Modify the CLI human formatter for `decision-show` to render the options block when `options` is non-empty.
- Render each option's `key`, `label`, `rationale`, and `recommended` flag.
- Render gracefully (omit the section, no empty header) when `options` is absent or empty.
- Verify the web UI's decision-detail view renders options + recommended flag. If it does not, extend the fix to the UI to honor the initiative's three-surface parity rule.

**Out of scope:**
- Changing the JSON output (already correct).
- Changing the API response shape — already exposes options.
- Changing `decision-list` or `decision-accept` output (separate fix items in this initiative).
- Adding new CLI flags (workshop d1 selected always-on; no flag needed).
- Reworking unrelated formatters or UI views.

## Current Technical Context

The prompt-manager source is not in this workspace (this is the agent-manager repo with only a prompt-manager client). Implementation lands in `scenarios/prompt-manager/**`. Executor must locate:
- The `decision-show` command handler and its human formatter (likely a `formatDecision` / `renderDecision` helper).
- The decision struct/type for the `options` field shape (`key`, `label`, `rationale`, `recommended`).
- The web UI's decision-detail view component (for the d3 = B parity check).
- Existing patterns for option rendering on sibling commands so style matches house conventions.

## Target End State

### CLI

Running `prompt-manager team decision-show <team> <id>` against a decision with options prints (default, no extra flags):

```
Decision:   <topic / proposal>
Status:     pending
Context:    <context paragraph>
Rationale:  <rationale paragraph>

Options:
  A  OAuth with Google         (recommended)
     Lowest effort, covers 90% of users.
  B  JWT with custom auth
     More control, offline support.
  C  Other
     Provide your own approach.
```

Decisions with no options omit the `Options:` section entirely (no empty header, no "(none)").

### UI (if Phase 1 finds a gap)

Decision-detail view renders the same fields: option `key`, `label`, `rationale`, and a visible `recommended` indicator. Visual style follows the existing UI's conventions; this plan does not prescribe pixel-level layout.

## Implementation Strategy

Two phases. Phase 2 is conditional on Phase 1's UI-parity finding.

**Phase 1 — CLI fix (always runs):**
1. Locate the `decision-show` human formatter in `scenarios/prompt-manager/**`.
2. Add an "Options" section to the formatter that iterates `options` and renders, per workshop **d2 = A**:
   - `key` and `label` on one line, with `(recommended)` appended after the label when `recommended: true`.
   - `rationale` indented on the next line.
3. Skip the section when `options` is missing or empty.
4. Add/extend unit/golden tests covering: (a) options with a recommended one, (b) options with none recommended, (c) absent/empty options.
5. Quick manual UI check: open the web UI's decision-detail view on the same decision used for CLI verification. Note whether options + recommended marker are visible.

**Phase 2 — UI parity (only if Phase 1 step 5 reveals a gap):**
1. Locate the decision-detail view component in `scenarios/prompt-manager/**`.
2. Render `options` (key, label, rationale, recommended marker) using the existing UI's component conventions.
3. Add or extend a component test mirroring the CLI's three test cases.
4. If the UI already renders options correctly, skip this phase entirely and note the verification result in the rollout checklist.

**Phase 3 — finalize:**
1. Run the prompt-manager scenario's lint/type/test suites and fix all issues in modified files (including pre-existing).
2. `vrooli scenario restart prompt-manager` and verify health.

## Contract Decisions

Resolved in workshop round-001:

- **d1 — Display behavior:** **A — always show options when present (no flag).** Matches the human-first CLI principle; option-less decisions pay zero cost (section omitted).
- **d2 — Layout/format:** **A — indented list.** Key + label on one line, `(recommended)` marker after label when set, rationale indented on the next line. Matches the Target End State block above.
- **d3 — Three-surface parity scope:** **B — CLI + UI.** API is already correct (returns options via `--json`). UI is verified during Phase 1; extended into scope only if a gap is found. Effort may grow from XS toward S if the UI fix is needed.

## Testing Plan

**CLI (always):**
Unit/golden tests on the formatter:
- Decision with 3 options, one marked `recommended`: output contains all three keys, all labels, the `(recommended)` marker on the right one, and rationale lines.
- Decision with options but no recommended flag: no `(recommended)` marker appears.
- Decision with `options: []` or absent: no `Options:` header in output.

Manual CLI verification:
- `prompt-manager team decision-show <team> <id-with-options>` shows options block.
- `prompt-manager team decision-show <team> <id-without-options>` matches today's output (no options section).
- `--json` output byte-identical to before (regression check).

**UI (only if Phase 2 runs):**
- Component test covering the same three cases as the CLI.
- Manual: load the decision-detail view and confirm options + recommended marker render.

## Rollout / Validation Checklist

- [ ] Type/lint clean (including pre-existing issues in modified files).
- [ ] CLI unit tests pass.
- [ ] Manual CLI walk on a decision with options matches Target End State.
- [ ] Manual CLI walk on an option-less decision unchanged.
- [ ] `--json` output byte-identical to before.
- [ ] UI parity check performed; result recorded (already-correct vs. fixed).
- [ ] If UI was fixed: component test added, manual UI walk confirmed.
- [ ] `vrooli scenario restart prompt-manager` succeeds; health check passes.

## Risks + Mitigations

| Risk | Mitigation |
|---|---|
| Long rationale text wraps awkwardly in narrow terminals | Indent rationale under its option key; rely on terminal soft-wrap rather than truncation. |
| Decisions with many (>10) options become noisy in default output | Accepted per workshop d1 = A; the human-first principle explicitly favors visibility over compactness. |
| Sibling `decision-list` and `decision-accept` have similar gaps | Out of scope here. Sibling item `fix/prompt-manager-decision-accept-no-options-ergonomics` already tracks one of them. |
| UI gap turns out larger than expected, blowing past XS effort | Phase 2 is bounded: only render `options` array fields. If the UI needs broader rework (theming, layout system), open a follow-up item rather than expanding here. |
| Recently-completed sibling fields (`deferred` status, structured modifications) are also missing from `decision-show` output | Not a regression of this fix — flag any gap as a separate backlog item rather than expanding scope. |

## Initiative Neighborhood Notes

This item is a member of `prompt-manager-decision-workflow-polish` (active, priority 6, no upstream/downstream initiatives). Sibling items:
- `fix/prompt-manager-heartbeat-list-lifecycle-states` — unrelated surface (heartbeats), no overlap.
- `fix/prompt-manager-decision-accept-no-options-ergonomics` — adjacent surface (`decision-accept` for option-less decisions). No code overlap, but the operator-experience story is shared. The output-format style chosen here may inform later `decision-accept` work but not vice-versa.
- `execute/prompt-manager-decision-accept-initiative-proposal-auto-create` — unrelated.
- Two completed members (`decision-deferral-primitive`, `decision-partial-accept-with-modifications`) shipped recently and added new fields to the decision schema. Per workshop info i1: confirm whether `decision-show`'s human output renders those fields; if not, file a separate backlog item rather than absorbing the work here.

**Cross-initiative implications:** none. This initiative has no upstream or downstream initiatives; the fix is self-contained to the `prompt-manager` scenario.

## Non-goals / Prohibited Patterns

- No backwards-compat shims.
- No new CLI flags (workshop d1 selected always-on).
- No changes to JSON output.
- No changes to API response shape.
- Do not refactor unrelated formatters or UI views in the same change.
- Do not absorb sibling-item work (`decision-list`, `decision-accept`, `heartbeat-list`) into this fix.

## Definition of Done

- `decision-show` human output always displays the options block when options exist, in the indented-list layout from Target End State.
- Formatter handles option-less decisions cleanly (no empty `Options:` header).
- UI parity check performed; UI either confirmed-correct or fixed to match.
- Tests cover the three options-shape cases on every surface that was modified.
- Lint/type/tests clean in modified files.
- `--json` output unchanged (regression check passes).
- Scenario restarts and is healthy.
