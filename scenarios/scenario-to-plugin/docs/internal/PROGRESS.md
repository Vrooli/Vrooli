# Progress — Scenario to Plugin

Lifecycle log for meaningful scenario changes. Future agents read this
file to understand what changed without reconstructing history from git.

Append entries when work lands, not while work is still speculative.

## Progress Log

| Date | Author | Status | Summary |
|---|---|---|---|
| 2026-08-19 | claude | done | Generated the scenario from `react-vite` 1.6.5 with the `vrooli-default` design kit. Authored the full product contract: `PRD.md` to the canonical template (8 P0 / 5 P1 / 4 P2 operational targets), and a `requirements/` registry of 40 requirements across six domain modules with every operational target covered. Validation: `vrooli scenario requirements validate scenario-to-plugin --json` reports `PASSED` with zero findings. Every requirement is `planned` and every validation is a `manual` entry pointing at `docs/internal/TESTING.md` — no code exists, so naming a not-yet-written test path would have been a broken proof path. |
| 2026-08-19 | claude | done | Authored the concept, business, operations, and internal documentation set against the real capability: `DOMAINS.md` (six pipeline domains plus `health`, with the acyclic stage chain and its two enforcement rules), `DATA.md` (records-and-references split; capture store holds bytes), `INTEGRATIONS.md` (seven scenario dependencies, reconciled with `.vrooli/service.json`), `FLOWS.md` (six stateful flows, state machines, and two cross-flow invariants), `SECURITY.md` (12-threat model mapped to requirements), `DECISIONS.md` (13 durable decisions), plus `MONETIZATION.md`, `GO-TO-MARKET.md`, `DEPLOYMENT.md`, `RUNBOOK.md`, `OBSERVABILITY.md`, and `PERFORMANCE.md`. Template-durable sections (`ARCHITECTURE.md` extension rules, `FLOWS.md` maturity ladder and production shape, `SEAMS.md`, `troubleshooting.md`) were preserved rather than rewritten, and every `EXAMPLE-DOMAIN` fence is intact so `detemplate` still works. |
| 2026-08-19 | claude | partial | Declared seven scenario dependencies in `.vrooli/service.json` with per-edge `startup_policy` and `degraded_behavior`, matching `docs/concepts/INTEGRATIONS.md`. Not yet done: no code, no proto, no domain implementation. Gate 0 (`make setup` / `make start` / `make test`) has not been run, and Gates 6–7 (first vertical slice, example-domain removal) are untouched. |
| 2026-11-24 | PT2 owner | partial | Added the missing scenario UI Input primitive and stabilized the AppShell accessibility test with API mocks. Focused tests (7), UI type-check, and UI coverage suite (72) pass. Local requirements validation still reports stale business evidence, orientation hit a Template Manager 500, and Test Genie was blocked in shared proto preflight; no release readiness claim. See `evidence/PT2-SCENARIO-TO-PLUGIN-LOCAL-UI-REQUIREMENTS-GATES.md`. |
| 2026-11-24 | PT2 owner | partial | Ran the Test Genie business phase; it passes but the requirements snapshot is stale against that run, with all 40 requirements still planned and 36 live statuses pending. The active Settings page still has three unavailable captures because normal Scenario-to-Plugin start is blocked by Workspace Sandbox's dirty failed rebuild. Production UI build passes; static UI health remains not-evaluated with slot/component findings. No release-readiness claim. See `evidence/PT2-SCENARIO-TO-PLUGIN-REQUIREMENTS-EXPERIENCE-GATES.md`. |

| 2026-11-24 | PT2 owner | partial | Reproduced the package-retention gap: Compose persisted package trees without any age-based cleanup. Composition now prunes unpublished package rows and their owned trees after 90 days when a new eligible package is composed; published/withdrawn packages, recent packages and paths outside the owned store are preserved/refused. Focused SQLite lifecycle tests pass. Idle-time scheduling, rehearsal logs, readiness history, declaration snapshots and install attributions remain uncovered; retention finding stays open. See the PT2 evidence. |

## Entry Template

Use this table shape when appending entries.

```markdown
| YYYY-MM-DD | author | done | Concise summary of the completed change |
```

## Cross-references

- [`PROBLEMS.md`](PROBLEMS.md) — known issues, tech debt, and deferred work
- [`DECISIONS.md`](DECISIONS.md) — durable decisions and tradeoffs
- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — system map
