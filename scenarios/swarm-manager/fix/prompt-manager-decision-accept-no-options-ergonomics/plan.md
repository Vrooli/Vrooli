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
- Tests covering: (a) accept with no `--selected`, (b) accept with explicit `--selected=proposed` sugar if adopted, (c) reject still works, (d) multi-option decisions still require `--selected`

**Out of scope:**
- Changes to deferral primitive (sibling: `execute/prompt-manager-decision-deferral-primitive`, completed)
- decision-show option-block display (sibling fix in same initiative)
- decision-add changes for multi-option decisions
- Changes to the `modifications` contract (sibling: `execute/prompt-manager-decision-partial-accept-with-modifications`, completed)

## Greenfield Constraint

This is greenfield work on an internal tool with a single operator. No backwards-compatibility shims, no `__other__ + freeform="accept as proposed"` alias preserved "just in case." Old invocations should fail loudly with a helpful error pointing to the new pattern.

## Current Technical Context

**Source root:** `scenarios/prompt-manager/`

Key areas the executing agent will need to locate (paths approximate; verify via `grep -r "decision" scenarios/prompt-manager/`):
- API decision handler / validation (likely under `scenarios/prompt-manager/api/`)
- CLI `decision-accept` command (likely under `scenarios/prompt-manager/cli/`)
- Decision domain model — particularly the `options` and `selected` fields
- `decision-add` flow for understanding how single-proposal decisions are stored (empty `options[]` vs. populated)
- UI surface (web-console or equivalent decision view)

The executing agent should use `scientific-debugging` to first reproduce the awkward path end-to-end, then locate the validation rule that requires `selected` regardless of options-list size.

## Target End State

A single-proposal decision (one created with no A/B/C options) can be accepted with:

```bash
prompt-manager decision-accept --id <id>
```

— no `--selected`, no `--freeform`. The API accepts the request, marks the decision accepted, and stores no spurious `selected` value (or stores a normalized sentinel like `"proposed"` if a non-null value is structurally required by storage). The CLI human output confirms acceptance unambiguously.

Multi-option decisions (`options[]` non-empty) continue to require `--selected` and reject requests missing it with a clear error.

## Implementation Strategy

**Approach: B — API-side normalization (per operator preference, recorded in spec).**

### Phase 1: API contract change
1. In the decision-accept endpoint, branch on `len(decision.options)`:
   - **Empty options:** `selected` is optional. If provided, must be the empty string, `null`, or the sentinel `"proposed"` (if we adopt one) — anything else is a 400. `freeform` is also optional in this branch.
   - **Non-empty options:** existing behavior unchanged. `selected` required, must match an option key or be `__other__` with `freeform`.
2. Persist a clear marker that the decision was accepted-as-proposed (e.g., `accepted_as_proposed: true` on the decision record, OR `selected: "proposed"` if storage prefers a non-null discriminator). Decision noted in Contract Decisions below.
3. Update the API response schema docs / OpenAPI if applicable.

### Phase 2: CLI alignment
1. `decision-accept` no longer requires `--selected` when the decision has no options. The CLI should pre-fetch the decision (or rely on the API's tolerant behavior) so the operator can omit `--selected` even on the very first call.
2. If the operator passes `--selected=__other__ --freeform="accept as proposed"` against a single-proposal decision, the CLI/API should reject with: `single-proposal decisions are accepted with no --selected; rerun without it`. (Greenfield — no silent alias.)
3. Update CLI human output for accept-as-proposed: `Accepted decision <id> as proposed.` (vs. `Accepted decision <id>: option A`).

### Phase 3: UI parity
1. Web UI accept button on a single-proposal decision card sends the new payload shape (no `selected`).
2. UI confirms with the same "accepted as proposed" copy.

### Phase 4: Cleanup & verification
- Run type checking and linter against `scenarios/prompt-manager/` and fix ALL errors, including pre-existing ones in modified files.
- Run unit tests; fix failures.
- `vrooli scenario restart prompt-manager` and verify health.
- Manually exercise the flow end-to-end using a freshly created single-proposal decision.

## Contract Decisions

(Will be populated/refined by workshop decisions in `workshop/round-NNN.json`. Defaults reflect the operator's stated preference.)

- **D1 — Approach:** Option B (API-side normalization). Recommended by spec.
- **D2 — Storage marker for accepted-as-proposed:** TBD via D2 in round-001.
- **D3 — Sentinel value for `selected` when accepting-as-proposed:** TBD via D3 in round-001.
- **D4 — `decision-add` behavior for single-proposal decisions:** TBD via D4 in round-001.

## Testing Plan

Unit + integration tests in `scenarios/prompt-manager/`:
1. `accept_single_proposal_no_selected` — POST accept with no `selected`, no `freeform` → 200, decision marked accepted.
2. `accept_single_proposal_explicit_proposed` — if sentinel adopted, POST with `selected="proposed"` → 200.
3. `accept_single_proposal_legacy_other` — POST with `selected="__other__" freeform="accept as proposed"` → 400 with helpful error (greenfield: no silent alias).
4. `accept_multi_option_no_selected` — POST accept on multi-option decision with no `selected` → 400 unchanged.
5. `accept_multi_option_with_selected` — existing behavior unchanged.
6. `reject_single_proposal` — reject path still works on single-proposal.
7. CLI smoke: `prompt-manager decision-accept --id <id>` against a single-proposal decision succeeds without further flags.
8. UI smoke: accept button on single-proposal decision card succeeds.

## Rollout / Validation Checklist

- [ ] All tests above pass
- [ ] Lint + typecheck clean on modified files (including pre-existing issues)
- [ ] `vrooli scenario restart prompt-manager` successful, health OK
- [ ] Manually create a single-proposal decision via `decision-add`, accept it without `--selected`, confirm via `decision-show`
- [ ] UI accept flow exercised in browser
- [ ] OpenAPI / API docs updated if present

## Risks & Mitigations

| Risk | Mitigation |
|------|-----------|
| Existing in-flight decisions stored with `selected=__other__ freeform="accept as proposed"` need historical interpretation | One-time read-side normalization in `decision-show`: if `selected=="__other__"` AND `freeform~/accept as proposed/i` AND `options==[]`, render as "accepted as proposed". No data migration. |
| Other CLI flows (decision-list output, history) may have hardcoded assumptions about `selected` being non-empty | Grep for `selected` references in display code; update to handle empty/`"proposed"` |
| Three-surface parity rule (per initiative): API + CLI + UI must ship together | Single PR covering all three surfaces; do not partial-ship |
| Sibling fix `decision-show-options-default-output` may overlap on display formatting | Coordinate: this plan owns the "accepted as proposed" rendering; sibling owns the always-show-options-block change. Surface this overlap in the orchestrator's review. |

## Cross-Initiative Implications

- This item is part of `prompt-manager-decision-workflow-polish` (6 members, 2 completed).
- Sibling `fix/prompt-manager-decision-show-options-default-output` overlaps on `decision-show` output formatting — coordinate so both PRs render single-proposal decisions consistently. Flag for orchestrator.
- No upstream/downstream initiatives.

## Non-Goals / Prohibited Patterns

- No backwards-compat alias for `--selected=__other__ --freeform="accept as proposed"` on single-proposal decisions
- No changes to multi-option decision behavior
- No changes to the `modifications` field contract (handled by completed sibling)
- No changes to deferral semantics

## Definition of Done

- API: single-proposal decisions accept without `--selected`; multi-option behavior unchanged
- CLI: `decision-accept --id <id>` works on single-proposal decisions with no other flags; legacy `__other__` pattern returns a clear error
- UI: accept button on single-proposal decision works and shows "accepted as proposed" copy
- All tests in Testing Plan pass
- Lint/typecheck clean on modified files (including pre-existing)
- Scenario restarted and verified healthy
