# Progress — Token Economy

Lifecycle log for meaningful scenario changes. Future agents read this
file to understand what changed without reconstructing history from git.

Append entries when work lands, not while work is still speculative.

> **Reading note.** The two `codex` rows dated 2026-07-07 are inherited from the
> `react-vite` template's own authoring history and describe changes to the
> *template*, not to this scenario. This scenario's own history begins at the
> 2026-08-18 initialization row.

## Progress Log

| Date | Author | Status | Summary |
|---|---|---|---|
| 2026-08-18 | claude | done | **Scenario initialized from `react-vite` v1.6.5 with the `vrooli-default` design kit.** Replaced an abandoned stub directory (one 8-byte `.gitignore`, no salvageable content) with a fresh generation. Completed orientation gates 0–5b: scaffold health (`make setup` clean), charter (PRD authored through the `business-health` wizard — 14 P0, 10 P1, 7 P2 operational targets), requirements registry (31 requirements across three modules, EARS-worded with RFC 2119 keywords, `vrooli scenario requirements validate` PASSED with zero findings), domain map (seven product domains recorded in `DOMAINS.md` with `DATA.md`, `FLOWS.md` and `INTEGRATIONS.md` filled), dependency decisions (`scenario-authenticator` required and fail-closed; `notification-hub` and `agent-manager` optional and degrading; zero external resources, each candidate rejected with a reason), design adaptation (two-audience note recorded in `DESIGN.md`), experience contract (12 real pages plus 3 journeys; `experience-manager spec validate` PASSED), and the business/operations/internal document set. Orientation reports 7/9. The two outstanding gates are `example-domain-removed` (code work, correctly not attempted) and `scaffold-health` — the latter because **the generated scaffold does not pass its own suite**, established by the first post-generation run failing before any authoring. Filed upstream as scenario-qa `knw-1787091113421895908` (baseline failure + the `make orient` gate that reports it green on submission rather than verdict) and `knw-1787090742498513070` (44px tap-target floor violated by the `notes` example). **No product code was written.** |
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
