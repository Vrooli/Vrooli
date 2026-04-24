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
- Contract documentation for downstream consumers (workshop agents, plan generators, auto-create flows) — authored as a markdown doc in the prompt-manager scenario's docs/ directory, linked from proto comments (per d4-r2=C).
- Workshop-agent prompt update so it treats `modifications` as first-class input (per d4-r1=A).

### Out of scope
- Changing the shape of `notes` or deprecating it.
- Retroactive migration of decisions already accepted with prose-in-notes modifications.
- Changes to reject / defer paths (deferral is covered by sibling item `execute/prompt-manager-decision-deferral-primitive`).
- Display fixes to `decision-show` default output (covered by sibling `fix/prompt-manager-decision-show-options-default-output` — but this plan must ensure that item's eventual changes account for rendering `modifications`).
- Auto-create initiative flow (covered by sibling `execute/prompt-manager-decision-accept-initiative-proposal-auto-create` — per orchestration-summary, F14 sequences **after** F15 so it can carry modifications forward).
- Updating every other downstream consumer in the same change — per d4-r1=A, F15 ships the primitive + contract doc + workshop-agent prompt, and each other consumer wires up the field when it lands.
- Editing/amending `modifications` after first accept (per d2-r2=A, strictly accept-once; amendment is a separate backlog item if it becomes a need).

## Cross-Initiative Implications

This plan must align with sibling items in initiative `prompt-manager-decision-workflow-polish`:

| Sibling | Relationship |
|---|---|
| `fix/prompt-manager-decision-show-options-default-output` (F9) | Should render `modifications` in default output once this item lands. Flag for that item's workshop. |
| `execute/prompt-manager-decision-accept-initiative-proposal-auto-create` (F14) | Sequenced after this item. Auto-create flow must propagate `modifications` into the generated initiative's metadata / member items. |
| `fix/prompt-manager-decision-accept-no-options-ergonomics` (F10) | If `--selected=proposed` (or API normalization) ships, ensure `modifications` is accepted on option-less decisions too. |

Orchestrator should confirm sequencing when queuing: F9 → (F10) → F15 (this item) → F14.

## Current Technical Context

> Codebase investigation happens at execute-time inside `scenarios/prompt-manager/`. Targets:
- Decision record schema location (proto + Go struct).
- Decision-accept API handler (request validation, persistence).
- CLI `decision-accept` command implementation and flag parsing.
- UI decision-accept view and decision-history view.
- Existing `notes` field plumbing end-to-end (use as the shape reference).

Use the existing `notes` field as the structural template for plumbing `modifications` — same end-to-end path, different validator.

## Target End State

An operator can run:

```bash
prompt-manager decision-accept --id dec-123 --selected=A \
  --modifications='{"excluded_clauses":["relocate existing items"],"additions":[],"rationale":"items stay in their current initiative for now"}'
```

…and the resulting decision record exposes `modifications` as structured JSON. `decision-show` renders it in a distinct block (not merged into `notes`). Workshop agents and plan generators read `modifications` directly — no prose parsing.

Same capability present in the UI: after selecting an option, a structured form (per d3-r1=A) lets the operator add `excluded_clauses`, `additions`, and `rationale`, visible in decision history.

## Settled Decisions (round 1)

| # | Decision | Resolution |
|---|---|---|
| d1-r1 | Modifications data shape | **A — Fixed structured schema:** `{excluded_clauses: string[], additions: string[], rationale: string}` |
| d2-r1 | CLI input UX | **D — JSON flag (`--modifications='{...}'`) + `--modifications-file=<path>` escape hatch** |
| d3-r1 | UI affordance | **A — Structured form below selected option** (tag-entry inputs for arrays, textarea for rationale) |
| d4-r1 | Downstream consumer scope | **A — F15 ships field + storage + contract doc + workshop-agent prompt update;** other consumers wire up at their own item |
| d5-r1 | Default / absent value | **A — Optional field; absent = empty/null on read; no backfill** |

## Settled Decisions (round 2)

| # | Decision | Resolution |
|---|---|---|
| d1-r2 | Empty-payload validation | **A — Reject entirely-empty `modifications` objects** with structured field-violation; operator must omit the flag instead |
| d2-r2 | Edit/amend semantics | **A — Strictly accept-once;** `modifications` is immutable after first accept; amendment deferred to a separate item if ever needed |
| d3-r2 | Server-side clause matching | **A — No server validation** that `excluded_clauses` strings appear in the option's rationale; operator intent is authoritative |
| d4-r2 | Contract documentation location | **C — Markdown doc in the prompt-manager scenario's `docs/` directory; link from proto comments** (not a new steer skill) |

## Implementation Strategy (phased)

### Phase 1 — API schema and handler
- Extend proto for the decision record and decision-accept request with a `modifications` message:
  ```proto
  message DecisionModifications {
    repeated string excluded_clauses = 1;
    repeated string additions = 2;
    string rationale = 3;
  }
  ```
- Field is optional on the request (per d5-r1=A); persisted as-is.
- Update decision-accept handler: reject entirely-empty modifications (per d1-r2=A), validate that arrays don't contain empty strings, that `rationale` length is bounded (e.g., 4 KB).
- `modifications` is **immutable after first accept** (per d2-r2=A) — no `decision-update --modifications=...` path in this item.
- Update decision-get / decision-list / decision-show responses to surface the field.
- Error model: malformed `modifications` returns a structured field-violation error pointing at the offending sub-field.

### Phase 2 — CLI
- Add `--modifications` flag (single JSON string) and `--modifications-file=<path>` flag (mutually exclusive) to `prompt-manager decision-accept`.
- Parse + validate at CLI boundary; pass through typed.
- Help text explicitly distinguishes `--modifications` (structured exceptions) from `--notes` (free-form commentary). Include an inline JSON example.
- Update `decision-show` default output to render a `Modifications:` block (coordinate with sibling F9 — that item now owns the options-block render; F15's display addition is just the modifications block).
- Update `decision-history` / `decision-list` output to indicate when modifications are present.

### Phase 3 — UI
- Extend decision-accept form per d3-r1=A: render the three structured inputs only when an option is selected, collapsed by default behind a "Add modifications" affordance.
- Render modifications in decision-history / decision-detail views as a distinct block, visually separated from `notes`.
- Ensure three-surface parity per the initiative's non-negotiable rule.

### Phase 4 — Downstream consumer integration (scoped per d4-r1=A)
- Author the contract doc as a markdown file in the prompt-manager scenario's `docs/` directory (per d4-r2=C). The doc defines:
  - The shape (`excluded_clauses`, `additions`, `rationale`).
  - Semantic rules (no server-side clause matching per d3-r2=A; operator intent authoritative).
  - Relationship to `notes` (distinct fields, distinct purposes).
  - When consumers should read it and how to interpret absence.
  - Immutability rule (per d2-r2=A).
- Reference the doc from proto comments on the `DecisionModifications` message so schema readers land on it.
- Update the workshop-agent skill/prompt so resolved decisions feed `modifications` to plan synthesis as structured input distinct from `notes`.
- Do **not** update F14 (auto-create) or other consumers in this item — they wire up at their own time using the contract doc.

### Final: Cleanup & Verification
- Run type checking and fix ALL errors, even pre-existing.
- Run linter and fix ALL warnings in modified files.
- Run unit tests and fix any failures.
- `vrooli scenario restart prompt-manager`
- Verify health check + UI loads + CLI round-trip: accept with modifications → show → see structured block.

## Contract Decisions

- **New field name:** `modifications` (distinct from `notes` in semantics).
- **Three-surface parity:** API + CLI + UI all ship together.
- **Shape (d1-r1=A):** `{excluded_clauses: string[], additions: string[], rationale: string}`.
- **Nullability (d5-r1=A):** optional on write; absent / empty on read for decisions written before this change; no migration.
- **CLI input (d2-r1=D):** `--modifications='{...}'` JSON flag; `--modifications-file=<path>` escape hatch for editor-authored payloads; mutually exclusive.
- **Relationship to `notes`:** `notes` remains free-form decision-wide commentary. `modifications` is a structured, scoped exception against the *selected option's rationale*. Both can be present.
- **Where applicable (scope):** ships on the **accept** path only. Reject and defer paths are out of scope.
- **Mutability (d2-r2=A):** `modifications` is immutable after first accept.
- **Semantic authority (d3-r2=A):** server does not validate that `excluded_clauses` strings appear in the option's rationale. Operator intent is authoritative; documented in the contract.
- **Contract doc venue (d4-r2=C):** markdown in the prompt-manager scenario's `docs/` directory, referenced from proto comments.

## Validation Policy

- `excluded_clauses` and `additions` are arrays of non-empty strings; empty arrays allowed (means: nothing excluded / nothing added).
- `rationale` is a UTF-8 string, max 4 KB.
- An entirely empty modifications object (`{}` or `{excluded_clauses:[],additions:[],rationale:""}`) is **rejected** with a clear error (per d1-r2=A) — operator should omit the flag instead. (Prevents noise in history.)
- The server does **not** verify that `excluded_clauses` strings appear in the option's rationale text (per d3-r2=A) — operator intent is authoritative; matching is brittle. Documented in the contract.
- `modifications` cannot be edited after the decision is accepted (per d2-r2=A). Attempts to mutate return an error indicating the field is immutable post-accept.

## Testing Plan

- **API unit tests:**
  - Accept with valid `modifications` → persisted and round-trips.
  - Accept with `excluded_clauses` only / `additions` only / `rationale` only → all valid.
  - Empty-everything `{}` → rejected with field-violation error.
  - Malformed JSON / wrong types → 400 with structured error.
  - Read-back of a decision written before this change → `modifications` is absent/null.
  - Attempt to mutate `modifications` on an already-accepted decision → rejected with immutability error.
- **CLI integration tests:**
  - `decision-accept --modifications='{...}'` → `decision-show` renders block.
  - `decision-accept --modifications-file=fixture.json` → same result.
  - Both flags together → CLI rejects with clear message.
  - Invalid JSON in flag → CLI rejects at boundary, not at server.
- **UI tests:**
  - Component test for the structured form (input → typed payload).
  - View test for the modifications block in decision-detail / decision-history.
- **Contract fixture:**
  - Golden JSON fixture alongside the `docs/` contract doc showing a populated decision record. A workshop-agent-shaped harness reads it and asserts the field is consumable as structured data.

## Rollout / Validation Checklist

- [ ] API test suite green.
- [ ] CLI round-trip demonstrated on a real `dec-*` id.
- [ ] UI manual test: accept with modifications, verify history renders block.
- [ ] Contract doc published in `scenarios/prompt-manager/docs/` and linked from proto comments on `DecisionModifications`.
- [ ] Workshop-agent prompt updated and verified on a synthetic decision.
- [ ] Sibling F9 notified that its display change must include the `Modifications:` block.
- [ ] Sibling F14 workshop notified of the new contract before it runs.
- [ ] Sibling F10 (option-less ergonomics) notified to ensure modifications work on `--selected=proposed`-style paths.

## Risks & Mitigations

| Risk | Mitigation |
|---|---|
| Shape churn: `excluded_clauses`/`additions` may prove too narrow. | Schema is additive; can introduce new optional sub-fields without breaking existing consumers. Document the contract clearly so consumers don't over-couple to current shape. |
| Operators continue to put modifications in `--notes` by habit. | Help text + UI placeholder text steer to `--modifications`. `decision-show` visibly separates the two blocks. |
| Downstream consumers (workshop agents) don't read the new field. | Phase 4 explicitly updates the consumer; contract fixture test guards regressions. |
| UI parity lags API/CLI ship. | Three-surface parity is non-negotiable per initiative; do not merge API-only. |
| Invalid JSON via CLI flag. | Parse at CLI boundary with clear structured error. |
| Workshop-agent prompt update lands but synthesis logic ignores `modifications`. | Add a synthesis-side test that resolves a decision with modifications and asserts the plan reflects them. |
| Operator sets modifications on a decision they later want to amend. | Documented as out-of-scope (per d2-r2=A). If amendment becomes a need, follow-up backlog item should add a structured edit path rather than mutating the existing record. |
| Contract doc in `docs/` is less discoverable to skill-based agents than a steer skill. | Proto comments link to it; workshop-agent prompt update (Phase 4) explicitly cites it. Revisit if downstream authors routinely miss it. |

## Non-goals / Prohibited Patterns

- Do **not** deprecate or repurpose `notes`.
- Do **not** build a `notes` → `modifications` auto-parser.
- Do **not** ship API-only; three-surface parity is required.
- Do **not** introduce a compatibility shim or migration layer for pre-existing decisions.
- Do **not** server-side verify that `excluded_clauses` strings appear in option rationale text.
- Do **not** add modifications to reject/defer paths in this item.
- Do **not** add an edit/amend path for `modifications` in this item.

## Definition of Done

1. `modifications` field exists on the decision record, populated via decision-accept, across API + CLI + UI.
2. Validation policy enforced server-side (including empty-payload rejection and post-accept immutability); CLI fails fast on malformed input.
3. `decision-show` and `decision-history` render it as a distinct block.
4. Contract doc published in `scenarios/prompt-manager/docs/` and linked from proto comments; workshop-agent prompt reads `modifications` as first-class structured input.
5. Contract fixture test asserts structured consumption.
6. Sibling items F9, F10, F14 notified/coordinated.
7. Cleanup & Verification step passes: typecheck, lint, tests, restart, health.
8. Greenfield constraint honored: no shims, no legacy compat paths.
