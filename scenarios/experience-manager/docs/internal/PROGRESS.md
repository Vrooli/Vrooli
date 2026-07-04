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

## Entry Template

Use this table shape when appending entries.

```markdown
| YYYY-MM-DD | author | done | Concise summary of the completed change |
```

## Cross-references

- [`PROBLEMS.md`](PROBLEMS.md) — known issues, tech debt, and deferred work
- [`DECISIONS.md`](DECISIONS.md) — durable decisions and tradeoffs
- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — system map
