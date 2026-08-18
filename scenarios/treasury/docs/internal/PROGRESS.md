# Progress — Treasury

Lifecycle log for meaningful scenario changes. Future agents read this
file to understand what changed without reconstructing history from git.

This file ships empty in newly generated scenarios. Append entries when
work lands, not while work is still speculative.

## Progress Log

| Date | Author | Status | Summary |
|---|---|---|---|
| 2026-07-07 | codex | partial | Continued Phase 5 floor-proof work: generated experience pages are now active instead of draft, the template ships dashboard/notes/settings BAS observer cases with `spec_entry_id` labels, `bas/registry.json` is regenerated with those cases, the perf example uses `@selector/layout.shell`, and the routed DB proof declares required mutating safety labels. Validation: shallow template validation passes and deep validation no longer reports registry-stale or selector-bypass findings. Remaining blocker: retained deep run `template-validation-react-vite-deep-20260707-041314-6ce71066` still cannot prove active floors because Test Genie runs the generated scenario by `--scenario-path`/logical placement without registering runtime ports for BAS/experience-manager capture; filed scenario-qa bug `knw-1783397791178031359`. |
| 2026-07-07 | codex | partial | Experience floors/component-canon Phase 5 slice: bumped template to 1.6.0, seeded generated scenarios with adopted-provenance UI primitives, reworked AppShell to min-h-dvh with fixed safe-area BottomNav and Settings-owned locale switching, converted starter dashboard/notes/settings surfaces to governed components, added DataTable sorting/searching for the notes example, and updated generated docs to steer adopt-not-hand-roll UI growth. Validation: shallow template validation and generator tests passed; deep quick validation still fails on broader pre-existing template gates, with the slice-specific scattered-keydown warning addressed and component coverage improved. |
| 2026-08-18 | claude | done | **Scenario created and documentation gates 1-5b completed.** Generated from `react-vite` v1.6.5 with the `vrooli-default` design kit. Authored `PRD.md` through the business-health wizard (12 P0, 7 P1, 6 P2 targets, EARS-shaped with tier-matched RFC 2119 keywords); replaced the starter requirements registry with three PRD-derived modules carrying `TRS-` requirements and concrete planned validation strategies; wrote the real domain map (11 product domains plus `health`), data ownership, flow inventory with state machines and maturity targets, and dependency contracts; declared `dependencies` in `.vrooli/service.json`; replaced the design-adaptation marker; rewrote the experience contract with six real pages, DESIGN.md-required UX states, aspirational claims and three journeys; and filled the business, operations and internal stubs. Validation: `vrooli scenario requirements validate treasury` clean except the expected `business_evidence_stale` (no suite run yet); `experience-manager spec validate treasury` clean except four `route_unspecced` warnings (no BAS cases for unbuilt pages); `template-manager lifecycle orient treasury` at 7/9. **Not done:** Gate 0 scaffold health is blocked by an inherited template defect (see PROBLEMS.md, scenario-qa `knw-1787080554065889915`), and Gates 6-7 (first real vertical slice, example-domain removal) are code work that was out of scope for this pass. No implementation code was written. |

## Entry Template

Use this table shape when appending entries.

```markdown
| YYYY-MM-DD | author | done | Concise summary of the completed change |
```

## Cross-references

- [`PROBLEMS.md`](PROBLEMS.md) — known issues, tech debt, and deferred work
- [`DECISIONS.md`](DECISIONS.md) — durable decisions and tradeoffs
- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — system map
