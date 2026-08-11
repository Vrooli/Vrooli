# Progress — Scenario to iOS

Lifecycle log for meaningful scenario changes. Future agents read this
file to understand what changed without reconstructing history from git.

Append entries when work lands, not while work is still speculative.
Entries below the 2026-08-10 regeneration line are inherited from the
`react-vite` template's own history, not from this scenario.

## Progress Log

| Date | Author | Status | Summary |
|---|---|---|---|
| 2026-08-10 | claude | partial | **Scenario recreated from scratch.** The prior `scenario-to-ios` — a Swift template directory, a Go CLI stub, no `docs/`, no `requirements/`, no `experience/`, and an aspirational PRD promising HealthKit and ARKit — was archived to `/tmp/vrooli-retired-ramps-20260810/` and the scenario regenerated from `react-vite` 1.6.5 with the `vrooli-default` design kit. No proto footprint existed for the old scenario, so the move was clean. Authored the charter through the `business-health` wizard: 12 P0, 7 P1, and 4 P2 operational targets with 23 requirements under the `S2I` prefix. Removed the starter `01-foundation` requirements module. Filled the domain map with seven product domains (`projects`, `targets`, `builds`, `journeys`, `releases`, `distribution`, `readiness`), plus data ownership, flow inventory with state machines, integration contracts, and the four-layer ownership table in `ARCHITECTURE.md`. Declared scenario dependencies in `.vrooli/service.json`. The two decisions that shaped the design: the build host is modelled as a **bridge-node role** rather than a machine, and validation capability is tracked separately from release capability, because the only available macOS node is Intel and Xcode 27 dropped Intel entirely. Validation: `vrooli scenario requirements validate scenario-to-ios` reports PASSED. Not yet done: no code, the `notes` example domain is still present, and `experience/` is still the generated L0 contract. |
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
