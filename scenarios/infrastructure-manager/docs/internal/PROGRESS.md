# Progress — Infrastructure Manager

Lifecycle log for meaningful scenario changes. Future agents read this
file to understand what changed without reconstructing history from git.

Append entries when work lands, not while work is still speculative. The two
2026-07-07 rows below are inherited `react-vite` template history, not this
scenario's work; they are retained because they explain the shape of the
scaffold this scenario started from.

## Progress Log

| Date | Author | Status | Summary |
|---|---|---|---|
| 2026-08-17 | claude | done | **Documentation review pass; four consistency fixes applied before the first vertical slice.** (1) `TRUST-MODEL.md` named `UNTRUSTED` as the closed vocabulary's fallback in four places without defining it — it is now a table row, distinguished from `UNAVAILABLE` (no reading vs. an unverifiable reading), and the "unchecked readings" language in the triple now resolves to the same verdict rather than implying a sixth state. (2) The "never persist a verdict" invariant in `ARCHITECTURE.md` extension rule 3 and `DECISIONS.md` contradicted `DATA.md`, which stores trust verdicts on the reading row. `DATA.md` was substantively right, so the invariant is now scoped to **band** verdicts, with the reason stated: a band verdict describes the target and is recomputable against the current deadband, a trust verdict describes the observation and is not. `DATA.md` gains an explicit non-stored row for band verdicts. (3) `SETPOINT-MODEL.md` declared four axes while `OT-P0-005` measured a fifth — instrument coverage, `sensored / authored` targets, 9 of 14 — which none of the four could express. It is now the first axis, with a section on why the two coverage axes differ (plant-side supervision vs. setpoint-side observability); heading and description updated in `docs/manifest.json`. (4) `infra-health/team.json` still declared `instrument.status: none` with a 2026-08-13 marker naming the `*-health` fleet as the plant — the framing `SETPOINT-MODEL.md` § The Plant explicitly corrects. Now `partial`, naming this scenario, with `coversScenarios` and a dated marker that states the no-code position, the `SKETCH` setpoint confidence and the plant correction. No product code was added or implied by any of these edits. |
| 2026-08-17 | claude | done | **Scenario initialized documentation-first; no domain code exists.** Generated from `react-vite` 1.6.5 (`design=vrooli-default`). Authored the charter through the `business-health` wizard — `PRD.md` validates **PASSED at L3 ("PRD clean")** with zero findings, carrying 5 P0 / 4 P1 / 3 P2 operational targets and 12 `IFM-*` requirements across three modules; the template starter module was removed. Authored two new concept documents that carry the load-bearing modelling: `SETPOINT-MODEL.md` (target model, the two-owner setpoint split, the four axes, denominator-confidence at `SKETCH` with its stated reason) and `TRUST-MODEL.md` (the trust axis, closed verdict vocabulary, the uninstrumented-is-never-valid invariant) — both registered in `docs/manifest.json`. Filled `DOMAINS.md` with four domains (`targets` → `readings` → `supervision` → `focus`, in build order) and four deferred domains with revisit triggers; `INTEGRATIONS.md` with seven read-only dependencies each degrading independently; `DATA.md` with the store-readings-never-verdicts rule and a retention floor derived from the longest target window; `ARCHITECTURE.md` with the product boundary, six scenario-specific extension rules, four intentional deviations and a draft maturity table. Declared those dependencies in `.vrooli/service.json` (all optional, `try_start`, each with `degraded_behavior`). Authored the experience contract ahead of the UI: board / setpoint / readings / settings pages plus a `triage-next-finding` journey, with required claims asserting that no surface offers an edit or action affordance. Cleared every Gate 5b stub — business docs marked `not-applicable` with reasons, operations and internal docs filled. Orientation: **8 of 9 gates**; the two remaining need code. |
| 2026-07-07 | codex | partial | Continued Phase 5 floor-proof work: generated experience pages are now active instead of draft, the template ships dashboard/notes/settings BAS observer cases with `spec_entry_id` labels, `bas/registry.json` is regenerated with those cases, the perf example uses `@selector/layout.shell`, and the routed DB proof declares required mutating safety labels. Validation: shallow template validation passes and deep validation no longer reports registry-stale or selector-bypass findings. Remaining blocker: retained deep run `template-validation-react-vite-deep-20260707-041314-6ce71066` still cannot prove active floors because Test Genie runs the generated scenario by `--scenario-path`/logical placement without registering runtime ports for BAS/experience-manager capture; filed scenario-qa bug `knw-1783397791178031359`. |
| 2026-07-07 | codex | partial | Experience floors/component-canon Phase 5 slice: bumped template to 1.6.0, seeded generated scenarios with adopted-provenance UI primitives, reworked AppShell to min-h-dvh with fixed safe-area BottomNav and Settings-owned locale switching, converted starter dashboard/notes/settings surfaces to governed components, added DataTable sorting/searching for the notes example, and updated generated docs to steer adopt-not-hand-roll UI growth. Validation: shallow template validation and generator tests passed; deep quick validation still fails on broader pre-existing template gates, with the slice-specific scattered-keydown warning addressed and component coverage improved. |

## Entry Template

Use this table shape when appending entries.

```markdown
| YYYY-MM-DD | author | done | Concise summary of the completed change |
```

## Cross-references

- [`PROBLEMS.md`](PROBLEMS.md) — known issues, tech debt, and deferred work
- [`DECISIONS.md`](DECISIONS.md) — durable decisions and tradeoffs
- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — system map
