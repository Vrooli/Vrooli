# Flows — Quality Health

## Purpose Of This Document

This document records workflows with meaningful state, retries, cancellation, stale completion, or formal-model needs.

## Flow Inventory

| Flow | Domain | Trigger | Outcome | Statefulness | Validation |
|---|---|---|---|---|---|
| Live audit | audit | API/CLI/UI requests an audit. | Surfaces, contracts, findings, command results, maturity, and next steps are returned. | Multi-step orchestration with degraded and failed states. | Phase 2 unit/integration tests with fake Code Facts and command executor. |
| Config autofix | autofix | Caller requests dry-run or apply. | Deterministic config edit preview or explicit applied result. | Safety-critical because dry-run must not mutate and apply must be bounded. | Phase 2 temp-directory tests. |

## Flow Details

### Live audit

1. Resolve target scenario or path.
2. Request surface facts from Code Facts.
3. Match contracts to surfaces.
4. Evaluate config/source contracts.
5. Optionally run bounded lint/type commands.
6. Optionally compute autofix candidates.
7. Normalize findings and maturity.
8. Return `passed`, `failed`, `degraded`, or `error`.

If Code Facts is unavailable, the audit becomes degraded and must not claim a clean pass.

### Config autofix

1. Resolve target and requested rule set.
2. Evaluate supported fix candidates.
3. Produce preview hunks.
4. If `apply` is explicit, write only supported config edits.
5. Return applied/skipped status and warnings.

Dry-run is the default and must leave files unchanged.

## State Machines

| Domain/Flow | States | Illegal Transitions | Enforcement |
|---|---|---|---|
| audit / live audit | requested, discovering, evaluating, running_commands, summarizing, passed, failed, degraded, error | clean pass after discovery failure; command result without bounded execution metadata | Unit tests and future flow model if lifecycle complexity grows. |
| autofix / config fix | requested, planning, previewed, applying, applied, skipped, failed | mutation during dry-run; unsupported source edit in v1 | Temp-directory tests and explicit apply flag. |

## Maturity Ladder

| Level | Name | What exists |
|---|---|---|
| 0 | Unmodeled risk | Workflow behavior is implicit in handlers. |
| 1 | Inventory | Workflow is listed here with owner and risk. |
| 2 | Workflow model | State/status values and transition checks live beside the domain. |
| 3 | Matrix + traces | Tests cover state/event pairs and representative traces. |
| 4 | Declarative contract | A domain-local `*.flow.json` declares states, events, transitions, and invariants. |
| 5 | Checked formal model | Flow verifier checks generated formal artifacts and production replay. |

## Production Shape

Phase 2 can start with ordinary unit/integration tests. Add flow-verifier contracts only if audit/autofix orchestration grows enough that stale completion, cancellation, or retry ordering becomes a production risk.

## Deferred / Unmodeled Flows

| Flow | Risk | Next Step |
|---|---|---|
| UI audit run history | Stale run display once history exists. | Model when persistence is implemented. |
| Test Genie provider call | Provider timeout and partial findings. | Model in Phase 4 if delegation has retry/cancel states. |

## Cross-References

- [DOMAINS.md](DOMAINS.md)
- [DATA.md](DATA.md)
- [SEAMS.md](../internal/SEAMS.md)
- [TESTING.md](../internal/TESTING.md)
