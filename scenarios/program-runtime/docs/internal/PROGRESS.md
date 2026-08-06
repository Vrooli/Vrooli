# Progress — Program Runtime

Lifecycle log for meaningful scenario changes. Future agents read this
file to understand what changed without reconstructing history from git.

This file ships empty in newly generated scenarios. Append entries when
work lands, not while work is still speculative.

## Progress Log

| Date | Author | Status | Summary |
|---|---|---|---|
| 2026-07-07 | codex | partial | Continued Phase 5 floor-proof work: generated experience pages are now active instead of draft, the template ships dashboard/notes/settings BAS observer cases with `spec_entry_id` labels, `bas/registry.json` is regenerated with those cases, the perf example uses `@selector/layout.shell`, and the routed DB proof declares required mutating safety labels. Validation: shallow template validation passes and deep validation no longer reports registry-stale or selector-bypass findings. Remaining blocker: retained deep run `template-validation-react-vite-deep-20260707-041314-6ce71066` still cannot prove active floors because Test Genie runs the generated scenario by `--scenario-path`/logical placement without registering runtime ports for BAS/experience-manager capture; filed scenario-qa bug `knw-1783397791178031359`. |
| 2026-07-07 | codex | partial | Experience floors/component-canon Phase 5 slice: bumped template to 1.6.0, seeded generated scenarios with adopted-provenance UI primitives, reworked AppShell to min-h-dvh with fixed safe-area BottomNav and Settings-owned locale switching, converted starter dashboard/notes/settings surfaces to governed components, added DataTable sorting/searching for the notes example, and updated generated docs to steer adopt-not-hand-roll UI growth. Validation: shallow template validation and generator tests passed; deep quick validation still fails on broader pre-existing template gates, with the slice-specific scattered-keydown warning addressed and component coverage improved. |
| 2026-08-06 | codex | shipped | Implemented the governed programmatic projection: descriptor/manifest bindings with pre-flight validation and grants, live Act RPC and meta-optimization client, persistent session manager with ceilings, Python kernel sidecar with bounded handles, program corpus/provenance, typed telemetry, Connect API/CLI surfaces, and an operator binding registry UI. Focused API, CLI, kernel, and UI tests pass; full scenario validation remains in progress. |
| 2026-08-06 | codex | shipped | Closed the Act projection audit and runtime integration gaps: every 28-cell Act denominator row now receives a live registry verdict with partial confidence, typed submission/invocation/failure events reach `vrooli-events`, the registry UI validation is earned, and `vrooli.agent.run` now uses a session-gated Go bridge to start/wait/collect agent-manager workflow evidence as a handle. A lifecycle-managed live call against agent-manager's empty catalog returned the expected explicit workflow-not-found error. Typed inference remains fail-closed under its recorded ai-gateway promotion waiver; post-render P1 budget targets carry explicit scope waivers. |
| 2026-08-06 | codex | validated | Final hardening and validation: added the typed capabilities protojson contract, declared and registered the optional `vrooli-events` dependency, normalized API/CLI Go modules and generated TypeScript protobuf packaging, and corrected requirement metadata so earned telemetry/sandbox work is checked while inference and spend ceilings remain explicit planned waivers. The authoritative server-owned suite passed 19/20 phases with 0 failures; Template Manager remains the sole skipped phase because its own setup is blocked by go.mod tidy drift. |

## Entry Template

Use this table shape when appending entries.

```markdown
| YYYY-MM-DD | author | done | Concise summary of the completed change |
```

## Cross-references

- [`PROBLEMS.md`](PROBLEMS.md) — known issues, tech debt, and deferred work
- [`DECISIONS.md`](DECISIONS.md) — durable decisions and tradeoffs
- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — system map
