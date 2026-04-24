# Implementation Plan: Partial-Accept with Structured Modifications on Decisions

## Purpose

Add a structured `modifications` field to the prompt-manager decision-accept workflow so that when an operator accepts an option but wants to scope out (or add to) parts of its rationale, the exception is captured as **machine-readable data** that downstream consumers (workshop agents, plan generators, auto-create flows) can read directly — instead of being buried in `--notes` prose.

## Required Reading

```bash
prompt-manager skill read api-steer cli-steer interoperability-steer seam-discovery-and-enforcement
```

## Problem Statement

- Decision-accept today has two fields for operator intent: `selected` (which option) and `notes` (free-form prose).
- When a single option's rationale conflates multiple concerns — e.g., *"create initiative X **and** relocate items Y into it"* — the operator may agree with the first concern but disagree with the second.
- Current workarounds:
  - Reject the option and re-surface a custom alternative (loses the operator's intent to mostly-accept).
  - Accept the option and bury the exception in `--notes` prose (no structured consumer; easy to miss).
- Surfaced on 2026-04-24 vision walk with `dec-1776982737575948642`: operator accepted option A but rejected the "relocate items" portion; modification had to be duplicated in prose in both `--notes` and `orchestration-summary.md`, with no structured handoff.

## Greenfield Constraint

This is **greenfield** work. The existing `notes` field stays (for unstructured commentary), but the new `modifications` field is net-new across API + CLI + UI with no compatibility shim. No wrapper that bridges `notes` ↔ `modifications` — each has a distinct purpose.

## Scope

### In scope
- New `modifications` field on decision records, populated via the decision-accept path.
- API schema extension (proto + handler): accept and persist `modifications`; expose in `decision-get` / `decision-show` / decision history.
- CLI flag(s) on `prompt-manager decision-accept` for providing modifications.
- UI decision-accept flow: capture modifications alongside option selection.
- Rendering modifications in decision-history / decision-show default human output.
- Contract documentation for downstream consumers (workshop agents, plan generators, auto-create flows).

### Out of scope
- Changing the shape of `notes` or deprecating it.
- Retroactive migration of decisions already accepted with prose-in-notes modifications.
- Changes to reject / defer paths (deferral is covered by sibling item `execute/prompt-manager-decision-deferral-primitive`).
- Display fixes to `decision-show` default output (covered by sibling `fix/prompt-manager-decision-show-options-default-output` — but this plan must ensure that item's eventual changes account for rendering `modifications`).
- Auto-create initiative flow (covered by sibling `execute/prompt-manager-decision-accept-initiative-proposal-auto-create` — per orchestration-summary, F14 sequences **after** F15 so it can carry modifications forward).

## Cross-Initiative Implications

This plan must align with sibling items in initiative `prompt-manager-decision-workflow-polish`:

| Sibling | Relationship |
|---|---|
| `fix/prompt-manager-decision-show-options-default-output` (F9) | Should render `modifications` in default output once this item lands. Flag for that item's workshop. |
| `execute/prompt-manager-decision-accept-initiative-proposal-auto-create` (F14) | Sequenced after this item. Auto-create flow must propagate `modifications` into the generated initiative's metadata / member items. |
| `fix/prompt-manager-decision-accept-no-options-ergonomics` (F10) | If `--selected=proposed` (or API normalization) ships, ensure `modifications` is accepted on option-less decisions too. |

Orchestrator should confirm sequencing when queuing: F9 → (F10) → F15 (this item) → F14.

## Current Technical Context

<!-- TBD — requires codebase investigation in round 2 -->

- Decision record schema location (proto + Go struct).
- Decision-accept API handler (request validation, persistence).
- CLI `decision-accept` command implementation and flag parsing.
- UI decision-accept view and decision-history view.
- Existing `notes` field plumbing end-to-end (use as a shape reference).

Action for round 2: locate each of the five above in `scenarios/prompt-manager/` and note file paths.

## Target End State

An operator can run:

```bash
prompt-manager decision-accept --id dec-123 --selected=A \
  --modifications='{"excluded_clauses":["relocate existing items"],"rationale":"items stay in their current initiative for now"}'
```

…and the resulting decision record exposes `modifications` as structured JSON. `decision-show` renders it in a distinct block (not merged into `notes`). Workshop agents and plan generators read `modifications` directly — no prose parsing.

Same capability present in the UI: after selecting an option, a modifications affordance lets the operator add structured exceptions per clause (or globally), visible in decision history.

## Key Decisions (round 1 — pending user input)

1. **Modifications data shape** — pending (see `workshop/round-001.json` decision `d1`).
2. **CLI input UX** — pending (`d2`).
3. **UI affordance** — pending (`d3`).
4. **Downstream consumer contract** — pending (`d4`).
5. **Default / storage behavior** — pending (`d5`).

Once resolved, these become commitments in the Implementation Strategy section.

## Implementation Strategy (phased)

<!-- Phases depend on decisions d1-d5. Scaffold below will be refined in round 2. -->

### Phase 1 — API schema and handler
- Extend proto for decision record and decision-accept request with `modifications` field (shape per d1).
- Update decision-accept handler: validate, persist.
- Update decision-get / decision-list to surface the field.
- Update error model: invalid `modifications` payload returns a structured field-violation error.

### Phase 2 — CLI
- Add flag(s) per d2 to `prompt-manager decision-accept`.
- Add parse/validation at CLI boundary; pass through typed.
- Update `decision-show` default output to render modifications block (coordinate with sibling F9).
- Update `decision-history` / `decision-list` output.

### Phase 3 — UI
- Extend decision-accept form per d3.
- Render modifications in decision-history / decision-detail views.
- Ensure three-surface parity per initiative's non-negotiable rule.

### Phase 4 — Downstream consumer integration
- Update contract doc per d4.
- Update workshop agent prompt/skill so it reads `modifications` as a first-class input distinct from `notes`.
- Update plan generator (if applicable) to honor modifications when synthesizing plan content from a resolved decision.

### Final: Cleanup & Verification
- Run type checking and fix ALL errors, even pre-existing.
- Run linter and fix ALL warnings in modified files.
- Run unit tests and fix any failures.
- `vrooli scenario restart prompt-manager`
- Verify health check + UI loads + CLI round-trip: accept with modifications → show → see structured block.

## Contract Decisions

- **New field name:** `modifications` (distinct from `notes` in semantics).
- **Three-surface parity:** API + CLI + UI all ship together.
- **Shape:** TBD per d1.
- **Nullability:** TBD per d5.
- **Relationship to `notes`:** `notes` remains free-form commentary. `modifications` is structured. Neither supersedes the other; both can be present.

## Testing Plan

<!-- TBD — filled after d1-d5 resolve. Initial coverage targets: -->

- API: unit tests on handler (valid payload, invalid shape, empty, nil), persistence round-trip, backward-read of decisions written before this change (should show `modifications` as empty/null).
- CLI: integration test per flag style (d2) — round-trip `decision-accept` → `decision-show` surfaces the block.
- UI: component test for form, view test for history.
- Contract: golden fixture showing a decision record with `modifications` populated, consumed by a workshop-agent-like harness.

## Rollout / Validation Checklist

<!-- TBD — filled in round 2 or 3. -->

- [ ] API test suite green.
- [ ] CLI round-trip demonstrated on a real `dec-*` id.
- [ ] UI manual test: accept with modifications, verify history renders block.
- [ ] Downstream consumer doc updated.
- [ ] Sibling F9 updated to render modifications in default human output.
- [ ] Sibling F14 workshop notified of new contract before it runs.

## Risks & Mitigations

| Risk | Mitigation |
|---|---|
| Shape churn: if `modifications` proves too narrow, we'll need to extend or widen post-ship. | Pick the shape with the clearest domain intent (d1). Prefer additive extension if extra fields are needed later. |
| Operators continue to put modifications in `--notes` by habit. | `decision-show` visibly separates the two blocks; document the contract in CLI help text. |
| Downstream consumers (workshop agents) don't read the new field. | Phase 4 explicitly updates the consumer; include a contract fixture test. |
| UI parity lags API/CLI ship. | Three-surface parity is non-negotiable per initiative; do not merge API-only. |
| Invalid JSON via CLI flag. | Parse at CLI boundary with a clear structured error; support both single-JSON and repeated-key forms if d2 picks hybrid. |

## Non-goals / Prohibited Patterns

- Do **not** deprecate or repurpose `notes`.
- Do **not** build a `notes` → `modifications` auto-parser.
- Do **not** ship API-only; three-surface parity is required.
- Do **not** introduce a compatibility shim or migration layer for pre-existing decisions.

## Definition of Done

1. `modifications` field exists on the decision record, populated via decision-accept, across API + CLI + UI.
2. `decision-show` and `decision-history` render it as a distinct block.
3. A workshop agent (or documented consumer) can read `modifications` as structured input.
4. Sibling items F9 (display), F14 (auto-create propagation) are notified/coordinated.
5. Cleanup & Verification step passes: typecheck, lint, tests, restart, health.
6. Greenfield constraint honored: no shims, no legacy compat paths.
