# Decisions — Template Manager

This document records durable decisions and tradeoffs future agents
should not accidentally relitigate.

## Purpose Of This Document

Use this document when a choice:

- affects multiple files or future agents,
- rejects a plausible alternative,
- changes architecture, deployment, data, security, monetization, or
  testing direction,
- needs a revisit trigger.

Routine implementation log entries belong in [`PROGRESS.md`](PROGRESS.md).
Known unresolved issues belong in [`PROBLEMS.md`](PROBLEMS.md).

## Decision Log

| Date | Decision | Context | Consequences | Revisit Trigger |
|---|---|---|---|---|
| 2026-09-01 | Backfilled the `ui/server.js` port guards into all 79 existing scenarios by **surgical transform**, not by overwriting from the template. | A census found 68 distinct structural shapes across the 79 files — extra imports, `__dirname` resolution, `scenarioConfig` blocks, `apiHost` overrides, `createScenarioServer` vs `startScenarioServer`. Rendering the template over them would have destroyed real per-scenario customisation. | A scripted transform rewrote each file's own `process.env.UI_PORT`/`API_PORT` reads to the guarded consts, collapsed matching object keys to shorthand, and inserted the guard after that file's last top-level import, matching each file's semicolon style. Verified per file: `node --check` on all 79, exactly two remaining raw reads each (the guard's own), and 0 detector violations across all 81 files including both templates. | Revisit if a scenario ever needs a port read that is legitimately unguarded. |
| 2026-09-01 | Generated `ui/server.js` validates `UI_PORT` and `API_PORT` at the boundary, and **throws** for both. | `PROFILE_ENV_VALIDATION` flagged the raw `process.env` reads in every generated scenario. The first fix warned on `API_PORT` instead of throwing, on the belief that the UI could still serve static assets and `/health` in a degraded mode. Running the old and new files side by side disproved it: `createScenarioServer` already throws `Invalid API_PORT configuration` at startup (`template.ts:492`), so no such mode exists and the warning would have described behaviour that never happens. | Behaviour is **identical** to the previous template — the process still refuses to start on either missing port. Only the message changes, from `Invalid UI_PORT configuration` (the symptom) to a sentence naming the cause and the fix. Verified empirically in all three cases: UI_PORT missing, API_PORT missing, and both present. | Revisit if `api-base` ever makes `API_PORT` genuinely optional, which would make a warning correct. |
| 2026-07-09 | Use the generated `react-vite` scenario documentation contract. | Scenario scaffold was generated from the template. | Docs start with stubs and maturity metadata in `docs/manifest.json`. | Revisit when scenario adopts a different template or doc contract. |

## Superseded Decisions

| Date | Superseded Decision | Replacement | Details |
|---|---|---|---|
| None yet. | n/a | n/a | Add when a durable decision is replaced. |

## Cross-References

- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — system decisions
- [`PROBLEMS.md`](PROBLEMS.md) — unresolved drift and debt
- [`PROGRESS.md`](PROGRESS.md) — completed work history
