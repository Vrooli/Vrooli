# Plan: decision-accept ergonomics for option-less ("accept as proposed") decisions

## Purpose

Eliminate the awkward `--selected=__other__ --freeform="accept as proposed"` workaround for decisions that have no A/B/C options (single-proposal "Action as proposed" decisions). The intent in those flows is "yes, accept the proposed action" — `__other__` semantically means "none of the above" and is the wrong primitive.

## Required Reading

```bash
prompt-manager skill read scientific-debugging api-steer cli-steer seam-discovery-and-enforcement utils-unification
```

Initiative context (single call):
```bash
swarm-manager initiatives context --name prompt-manager-decision-workflow-polish
```

## Problem Statement

Decisions in prompt-manager can be created in two shapes:
1. **Multi-option:** explicit A/B/C options — operator picks one with `--selected=A`.
2. **Single-proposal ("Action as proposed"):** no A/B/C options — the decision *is* the proposal; the operator either accepts or rejects.

For shape (2), the current `decision-accept` API requires `--selected` and validates against the (empty) options list. The only path that works is `--selected=__other__ --freeform="accept as proposed"`, which:
- Is semantically wrong: `__other__` means "none of the above," not "yes, the proposed one."
- Adds noise to every accept (operator must invent freeform text).
- Drags morning vision walks by ~10 min/session and creates ambiguity about whether the tool supports the operator's intent.

Surfaced 2026-04-24 walk while accepting 3 of 4 meta-opt decisions, all of which had no A/B/C options.

## Scope

**In scope:**
- API contract for `decision-accept` (or equivalent endpoint) on single-proposal decisions
- CLI ergonomics for `decision-accept` on the same shape
- UI parity (per the initiative's three-surface rule)
- Tests covering: (a) accept with no `--selected`, (b) reject still works, (c) multi-option decisions still require `--selected`, (d) legacy `__other__ + freeform="accept as proposed"` write is rejected with a helpful error, (e) read-side normalization renders historical legacy-shape decisions as "accepted as proposed".

**Out of scope:**
- Changes to deferral primitive (sibling: `execute/prompt-manager-decision-deferral-primitive`, completed)
- decision-show option-block display (sibling fix in same initiative)
- decision-add changes for multi-option decisions
- An explicit `--single-proposal` flag on `decision-add` (D4 → stay implicit; empty `options[]` continues to encode intent)
- Changes to the `modifications` contract (sibling: `execute/prompt-manager-decision-partial-accept-with-modifications`, completed)

## Greenfield Constraint

This is greenfield work on an internal tool with a single operator. No backwards-compatibility shims, no `__other__ + freeform="accept as proposed"` alias preserved "just in case." Old write invocations should fail loudly with a helpful error pointing to the new pattern. Read-side normalization for already-stored historical decisions is a separate, narrow concern (see Risks).

## Current Technical Context

**Source root:** `scenarios/prompt-manager/`

Key areas the executing agent will need to locate (paths approximate; verify via `grep -r "decision" scenarios/prompt-manager/`):
- API decision handler / validation (likely under `scenarios/prompt-manager/api/`)
- CLI `decision-accept` command (likely under `scenarios/prompt-manager/cli/`)
- Decision domain model — particularly the `options`, `selected`, and (new) `accepted_as_proposed` fields
- `decision-add` flow for understanding how single-proposal decisions are stored (empty `options[]` vs. populated)
- UI surface (web-console or equivalent decision view)

The executing agent should use `scientific-debugging` to first reproduce the awkward path end-to-end, then locate the validation rule that requires `selected` regardless of options-list size.

## Target End State

A single-proposal decision (one created with no A/B/C options) can be accepted with:

```bash
prompt-manager decision-accept --id <id>
```

— no `--selected`, no `--freeform`. The API accepts the request, marks the decision accepted, and persists `accepted_as_proposed: true` on the decision record. `selected` and `freeform` remain null/absent for accept-as-proposed. The CLI human output confirms acceptance unambiguously: `Accepted decision <id> as proposed.`

Multi-option decisions (`options[]` non-empty) continue to require `--selected` and reject requests missing it with a clear error.

`decision-add` continues to infer single-proposal vs multi-option from `options[]` (empty = single-proposal). No new `--single-proposal` flag.

## Implementation Strategy

**Approach: B — API-side normalization (D1=B, per operator preference, recorded in spec).**

### Phase 1: API contract change
1. In the decision-accept endpoint, branch on `len(decision.options)`:
   - **Empty options:** `selected` and `freeform` are optional and should be omitted/null. If `selected` is provided, it is a 400 with a helpful message: `single-proposal decisions are accepted with no --selected; rerun without it`. (D3=A — reject loudly on write.)
   - **Non-empty options:** existing behavior unchanged. `selected` required, must match an option key or be `__other__` with `freeform`.
2. Persist a new boolean field on the decision record: `accepted_as_proposed: true` when an empty-options decision is accepted. (D2=A — explicit boolean, orthogonal to `selected`/`freeform`.) Do NOT use a sentinel string in `selected`; leave `selected` null in this branch.
3. Update the API response schema docs / OpenAPI if applicable to expose `accepted_as_proposed`.

### Phase 2: CLI alignment
1. `decision-accept` no longer requires `--selected` when the decision has no options. The CLI should pre-fetch the decision (or rely on the API's tolerant behavior) so the operator can omit `--selected` even on the very first call.
2. If the operator passes any `--selected` (including `--selected=__other__ --freeform="accept as proposed"`) against a single-proposal decision, the CLI/API rejects with: `single-proposal decisions are accepted with no --selected; rerun without it`. (Greenfield — no silent alias.)
3. Update CLI human output for accept-as-proposed: `Accepted decision <id> as proposed.` (vs. `Accepted decision <id>: option A`). Detect via `accepted_as_proposed: true` from the response.

### Phase 3: UI parity
1. Web UI accept button on a single-proposal decision card sends the new payload shape (no `selected`, no `freeform`).
2. UI confirms with the same "accepted as proposed" copy and reads `accepted_as_proposed` from the decision record for display.

### Phase 4: Cleanup & verification
- Run type checking and linter against `scenarios/prompt-manager/` and fix ALL errors, including pre-existing ones in modified files.
- Run unit tests; fix failures.
- `vrooli scenario restart prompt-manager` and verify health.
- Manually exercise the flow end-to-end using a freshly created single-proposal decision.

## Contract Decisions

All workshop decisions answered in `workshop/round-001.json`.

- **D1 — Approach:** **B — API-side normalization.** Single-proposal decisions don't require `--selected` at the API level; CLI and UI both benefit. Matches the three-surface parity rule.
- **D2 — Storage marker for accepted-as-proposed:** **A — boolean field `accepted_as_proposed: true`.** Explicit, queryable, orthogonal to `selected`/`freeform`. No sentinel string in `selected`; it stays null for accept-as-proposed.
- **D3 — Treatment of legacy `--selected=__other__ --freeform="accept as proposed"` on writes:** **A — reject loudly with a 400 and a helpful migration message.** Greenfield-pure. Reads of historical data are still tolerant via `decision-show` normalization (see Risks).
- **D4 — `decision-add` behavior for single-proposal decisions:** **A — stay implicit.** Empty `options[]` continues to mean single-proposal. No new `--single-proposal` flag, no enum kind field.

## Testing Plan

Unit + integration tests in `scenarios/prompt-manager/`:
1. `accept_single_proposal_no_selected` — POST accept with no `selected`, no `freeform` → 200, decision marked accepted with `accepted_as_proposed: true`, `selected` is null.
2. `accept_single_proposal_rejects_explicit_selected` — POST with any `selected` value (including `__other__ + freeform="accept as proposed"`) on a single-proposal decision → 400 with the helpful migration message. (D3=A — no silent alias.)
3. `accept_multi_option_no_selected` — POST accept on multi-option decision with no `selected` → 400 unchanged.
4. `accept_multi_option_with_selected` — existing behavior unchanged; `accepted_as_proposed` is false/absent.
5. `reject_single_proposal` — reject path still works on single-proposal.
6. `decision_show_legacy_normalization` — historical decision stored with `selected="__other__"`, `freeform~/accept as proposed/i`, `options==[]` renders in `decision-show` as "accepted as proposed".
7. CLI smoke: `prompt-manager decision-accept --id <id>` against a single-proposal decision succeeds without further flags and prints `Accepted decision <id> as proposed.`
8. UI smoke: accept button on single-proposal decision card succeeds and shows "accepted as proposed" copy.

## Rollout / Validation Checklist

- [ ] All tests above pass
- [ ] Lint + typecheck clean on modified files (including pre-existing issues)
- [ ] `vrooli scenario restart prompt-manager` successful, health OK
- [ ] Manually create a single-proposal decision via `decision-add`, accept it without `--selected`, confirm `accepted_as_proposed: true` via `decision-show`
- [ ] UI accept flow exercised in browser
- [ ] OpenAPI / API docs updated if present (new `accepted_as_proposed` field documented)

## Risks & Mitigations

| Risk | Mitigation |
|------|-----------|
| Existing in-flight decisions stored with `selected="__other__"` and `freeform="accept as proposed"` need historical interpretation | One-time read-side normalization in `decision-show`: if `selected=="__other__"` AND `freeform~/accept as proposed/i` AND `options==[]`, render as "accepted as proposed". No data migration; no new field backfill. |
| Other CLI flows (decision-list output, history) may have hardcoded assumptions about `selected` being non-empty | Grep for `selected` references in display code; update to handle null `selected` + `accepted_as_proposed: true`. |
| Three-surface parity rule (per initiative): API + CLI + UI must ship together | Single PR covering all three surfaces; do not partial-ship. |
| Sibling fix `decision-show-options-default-output` may overlap on display formatting | Coordinate: this plan owns the "accepted as proposed" rendering; sibling owns the always-show-options-block change. Surface this overlap in the orchestrator's review. |
| New `accepted_as_proposed` field needs schema/migration if storage is typed | Add the field as optional/nullable boolean defaulting to false/absent; no backfill needed since historical decisions are normalized at read time. |

## Cross-Initiative Implications

- This item is part of `prompt-manager-decision-workflow-polish` (6 members, 2 completed: deferral primitive + partial-accept-with-modifications).
- Sibling `fix/prompt-manager-decision-show-options-default-output` overlaps on `decision-show` output formatting — coordinate so both PRs render single-proposal decisions consistently. **Flag for orchestrator:** the executing agent should sequence this fix and the show-options-default-output fix so display copy stays consistent across both PRs.
- No upstream/downstream initiatives.

## Non-Goals / Prohibited Patterns

- No backwards-compat alias for `--selected=__other__ --freeform="accept as proposed"` on single-proposal decision **writes** (read-side normalization is a separate, narrow concession)
- No sentinel string (`"proposed"`) in the `selected` field — D2=A picked the boolean field instead
- No `--single-proposal` flag on `decision-add` — D4=A keeps it implicit
- No changes to multi-option decision behavior
- No changes to the `modifications` field contract (handled by completed sibling)
- No changes to deferral semantics

## Definition of Done

- API: single-proposal decisions accept without `--selected`; passing any `--selected` on a single-proposal decision returns a clear 400; multi-option behavior unchanged; `accepted_as_proposed: true` persisted on accept-as-proposed
- CLI: `decision-accept --id <id>` works on single-proposal decisions with no other flags; legacy `__other__` pattern returns the documented error; output reads `Accepted decision <id> as proposed.`
- UI: accept button on single-proposal decision works and shows "accepted as proposed" copy
- All tests in Testing Plan pass
- Lint/typecheck clean on modified files (including pre-existing)
- Scenario restarted and verified healthy
