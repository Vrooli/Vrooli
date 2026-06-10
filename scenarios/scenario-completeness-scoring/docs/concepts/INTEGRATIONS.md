# Integrations — Scenario Completeness Scoring

This document is the canonical dependency contract for resources,
other scenarios, and third-party services used by the scenario.

## Purpose Of This Document

Use this document to answer:

- What does the scenario depend on?
- Which dependencies are required versus optional?
- Which domain uses each dependency?
- What is the failure or degradation behavior?
- Where is the dependency declared or configured?

## Dependency Inventory

| Dependency | Type | Required? | Used By | Contract | Failure Behavior |
|---|---|---|---|---|---|
| SQLite | embedded storage | yes | API, notes reference | `SQLITE_PATH` lifecycle env var | API reports unhealthy if unreachable. |
| Vrooli lifecycle | local platform | yes | API, UI, CLI | `.vrooli/service.json`, Makefile targets | Scenario should be started through lifecycle commands. |

## Vrooli Resources

The generated template does not declare external Vrooli resources. Add
resources to `.vrooli/service.json` only when a real scenario domain
requires them.

| Resource | Status | Reason | Revisit Trigger |
|---|---|---|---|
| None yet. | not-applicable | SQLite is embedded by default. | Add when PRD/requirements demand shared resource behavior. |

## Scenario Dependencies

The core score path is deliberately zero-network: every signal is read from
the target scenario's cached files. The two scenario dependencies below are
optional enrichment only and must never gate or slow the core path.

| Scenario | Status | Reason | Contract |
|---|---|---|---|
| scenario-dependency-analyzer | optional / best-effort (P1, deferred to the importance pass) | Importance line: reverse-dependency centrality + distance-to-core. | `graph centrality` Connect endpoint; shares a hard 1s combined budget with swarm-manager; on miss the importance line is silently omitted. |
| swarm-manager | optional / best-effort (P1, deferred to the importance pass) | Importance line: recent-activity window per scenario. | `GET /api/v1/operations?window=...`; same 1s combined budget and silent omission. |
| test-genie | upstream producer (no runtime call) | Writes the artifacts this scenario reads: `coverage/phase-results/*.json`, `coverage/runs.index.json`. | Read-side contract pinned by `packages/freshness-go` (runindex types) and the phase-results decoder fixtures. test-genie is never invoked by this scenario. |

## Shared Packages (code, not services)

| Package | Why | Contract |
|---|---|---|
| `packages/maturity-go` | Dimension vocabulary + R0–R4 ladder gate predicates shared with ecosystem-manager so rung answers agree by construction. | Pure logic; no service calls. |
| `packages/freshness-go` | Treedigest + run-index read + fresh/stale/unknown verdict core shared with test-genie. | Pure logic; digest spec frozen (byte-identical). |

## Third-Party Services

| Service | Status | Reason | Contract |
|---|---|---|---|
| None yet. | not-applicable | Generated scenario has no third-party dependency. | Add when PRD/requirements require external APIs, webhooks, auth, payments, or data feeds. |

## Failure Modes

| Dependency | Failure Signal | Expected Behavior | Tests |
|---|---|---|---|
| SQLite | `PingContext` error | `/health` returns unhealthy dependency status. | health handler tests |

## Cross-References

- [`ARCHITECTURE.md`](ARCHITECTURE.md) — system boundaries
- [`DATA.md`](DATA.md) — storage ownership
- [`../reference/configuration.md`](../reference/configuration.md) — environment and service manifest
- [`../operations/DEPLOYMENT.md`](../operations/DEPLOYMENT.md) — deployment readiness
