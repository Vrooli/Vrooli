# Progress — Template Manager

Lifecycle log for meaningful scenario changes. Future agents read this
file to understand what changed without reconstructing history from git.

Append entries when work lands, not while work is still speculative.

## Progress Log

| Date | Author | Status | Summary |
|---|---|---|---|
| 2026-07-07 | codex | partial | Continued Phase 5 floor-proof work: generated experience pages are now active instead of draft, the template ships dashboard/notes/settings BAS observer cases with `spec_entry_id` labels, `bas/registry.json` is regenerated with those cases, the perf example uses `@selector/layout.shell`, and the routed DB proof declares required mutating safety labels. Validation: shallow template validation passes and deep validation no longer reports registry-stale or selector-bypass findings. Remaining blocker: retained deep run `template-validation-react-vite-deep-20260707-041314-6ce71066` still cannot prove active floors because Test Genie runs the generated scenario by `--scenario-path`/logical placement without registering runtime ports for BAS/experience-manager capture; filed scenario-qa bug `knw-1783397791178031359`. |
| 2026-07-07 | codex | partial | Experience floors/component-canon Phase 5 slice: bumped template to 1.6.0, seeded generated scenarios with adopted-provenance UI primitives, reworked AppShell to min-h-dvh with fixed safe-area BottomNav and Settings-owned locale switching, converted starter dashboard/notes/settings surfaces to governed components, added DataTable sorting/searching for the notes example, and updated generated docs to steer adopt-not-hand-roll UI growth. Validation: shallow template validation and generator tests passed; deep quick validation still fails on broader pre-existing template gates, with the slice-specific scattered-keydown warning addressed and component coverage improved. |
| 2026-07-09 | codex | done | Template Manager owns registry, validation runs, templates phase, factory docs/search, dashboard measures, and recurring monitor slices through Phase 7 of the template-domain plan. Focused API/CLI/UI/proto/provider/measures validations passed; comprehensive maturity gates still track broader hardening. |
| 2026-07-11 | codex | done | Hardened deep-validation evidence and debt lifecycle: Test Genie protocol, startup, phase-results, and warning failures now have stable keys; each newer terminal deep run resolves only superseded Test Genie summary entries. Fresh retained evidence: Template Manager run `validation-0f1b4951-ed84-458a-ab3b-cd588eddf306`, Test Genie run `20260711-041544-256fe0b7`, workspace `/tmp/vrooli-template-deep-459889675`. All 20 phases executed (13 passed, 7 failed); react-vite remains correctly quarantined. |
| 2026-07-11 | codex | partial | Released react-vite 1.6.5 validation hardening: temporary deep workspaces link canonical schemas, generated routers opt into compatible React Router v7 behavior, starter requirements cover both P0 targets, and the CLI testutil guard reads Go's production package graph. Shallow validation passed; retained deep run `validation-b157ccf9-3a64-4617-935b-0d685876479c` executed all 20 phases with 17 passing. React-vite remains quarantined only for dependency, workflow-port, and proto implementation-discovery provider-boundary failures. |

## Entry Template

Use this table shape when appending entries.

```markdown
| YYYY-MM-DD | author | done | Concise summary of the completed change |
```

## Cross-references

- [`PROBLEMS.md`](PROBLEMS.md) — known issues, tech debt, and deferred work
- [`DECISIONS.md`](DECISIONS.md) — durable decisions and tradeoffs
- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — system map
