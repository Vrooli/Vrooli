# Progress — Experience Manager

Lifecycle log for meaningful scenario changes. Future agents read this
file to understand what changed without reconstructing history from git.

This file ships empty in newly generated scenarios. Append entries when
work lands, not while work is still speculative.

## Progress Log

| Date | Author | Status | Summary |
|---|---|---|---|
| 2026-07-04 | claude (design session with operator) | done | Scenario scaffolded (react-vite 1.4.0 + vrooli-default kit), detemplated, dependencies declared (BAS optional/try_start, image-tools optional/on_demand). PRD + 13 requirements authored via business-health wizard (validates green, 0 findings; all statuses honestly `planned`; auto-sync off pending real refs). Full design captured: claim-based open-world spec schema with enforcement tiers, `experience/` folder placement, ui-health/workflow-health boundaries, zero-ML v1 sequencing — see DECISIONS.md (18 durable decisions) and PROBLEMS.md (spike gate + BAS prerequisite). No implementation code yet. |
| 2026-07-04 | claude (design session with operator) | done | Shipped `scenario-experience-spec/v1` (`.vrooli/schemas/scenario-experience-spec.schema.json`, JSON Schema draft 2020-12) encoding the locked doctrine: claim-based + open-world, tiers with custom-never-machine enforced in-schema, ARIA role vocabulary, intent/bindings split, non-normative sketch, `x-` extension points. Authored the OT-P0-005 self-spec: `experience/` (index + 5 draft pages — fleet, scenario-explorer, evidence, studio, findings — + 2 journeys), 39 claims (27 machine / 7 manual / 5 aspirational), every element bound, L3+ depth by construction. Verified via scratch validator: metaschema check, all 8 documents schema-valid, cross-file references green (claim→element/state, bindings→elements, journey steps→page/state, prd_refs→PRD OTs). Spike gate 1/3 done (Studio); Matrix + web-console remain. |
| 2026-07-04 | claude (design session with operator) | done | **Spike gate CLOSED 3/3.** Authored intent-level spec seeds in two more scenarios' root `experience/` folders, grounded in their real UIs (selectors registries, a11y roles, state handling): business-health Matrix (`status: draft`, 16 claims — 8 expected reconciliation FAILURES per operator verdict "directionally correct but unpolished", forming the detection-calibration list in its `x-spike` block: missing verdict headline, unproven count computed-but-never-rendered, bookkeeping ordered before problems, drawer not announced as dialog, status chips indistinguishable) and web-console workspace (`status: active`, 16 claims expected to PASS — the hostile case: routerless single-surface SPA, `x-terminal` custom role, no DESIGN.md, display modes via `x-display-mode` scoping). Vocabulary held on both with ZERO schema changes. Scratch validator green across all three scenarios (12 docs: metaschema, schema-valid, index parity, cross-refs, prd_refs; experience-manager's 8 docs revalidated as regression). Two learnings appended to DECISIONS.md. |
| 2026-07-04 | claude (BAS a11y-capture session) | done | **BAS prerequisite RESOLVED** (see PROBLEMS.md entry). Browser-automation-studio now ships `CAPTURE_TYPE_ACCESSIBILITY` end-to-end: CDP `Accessibility.getFullAXTree` + DOMSnapshot geometry/`data-testid` join, normalized to the frozen `bas-accessibility-snapshot/v1` contract (`accessibility.json`; `inline_accessibility` returns it inline). Live-validated against a Vrooli UI (355 nodes, roles+names+bounds, real testids). Single-location capture only in v1; per-step timeline attachment deferred (slot reserved as `ARTIFACT_TYPE_ACCESSIBILITY_SNAPSHOT`). OT-P0-003 structure reconciliation is unblocked on the contract; it can consume the snapshot from a BAS capture. Contract seam: BAS `docs/SEAMS.md` §30. |

## Entry Template

Use this table shape when appending entries.

```markdown
| YYYY-MM-DD | author | done | Concise summary of the completed change |
```

## Cross-references

- [`PROBLEMS.md`](PROBLEMS.md) — known issues, tech debt, and deferred work
- [`DECISIONS.md`](DECISIONS.md) — durable decisions and tradeoffs
- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — system map
