# Storage Phase

**ID**: `storage`
**Timeout**: 120 seconds
**Optional**: No
**Source**: validation-provider (`storage-health`)

The storage phase owns Test Genie's storage judgment and — most importantly —
its **test-isolation safety gate**. It runs immediately before the `workflow`
phase so that its verdict can decide whether destructive end-to-end flows are
allowed to run at all.

Test Genie does not analyze storage natively. The phase is **delegated to the
[`storage-health`](../../../../storage-health/) scenario** through the shared
`ScenarioValidationService` contract (the same delegation model used by
structure-health, unit-health, security-health, and the other provider-backed
phases). Test Genie calls `storage-health validate scenario <scenario>`, maps
the returned `MaturityAssessment` findings into the `FINDING_SOURCE_STORAGE`
channel, and gates the phase on finding severity.

This phase declares a [Phase Capability Contract](../../concepts/phase-capability-contract.md); the sections below follow the required remediation-doc skeleton.

## North Star

Every storage-owning scenario is **isolation-safe and persistence-clean**:
destructive end-to-end flows are statically routed to isolated, variant-aware
test storage (never production data), embedded schemas are per-domain,
idempotent, `ALTER`-free, and wired through the supported seam, all persistence
goes through the routed repository seam with no raw `sql.Open`, unclosed rows, or
SQLite single-connection deadlocks, and backup/migration posture is clean or
intentionally not applicable. At maximum maturity the safety throughline —
`isolation_safety` — is L3 (clean) and the supporting ladders
(`target_classification`, `schema_substrate`, `persistence_hygiene`,
`operational_readiness`) are each at their top rung, so the `workflow` phase can
run mutating flows safely.

## The rungs and their gates

The phase aggregates five per-capability ladders; the safety-critical one is
`isolation_safety` (its L2 verdict gates whether destructive workflow flows may
run). The rungs are monotone.

| Rung (isolation_safety) | Gate | Next unlock |
|---|---|---|
| L0 Unknown | Storage isolation cannot be evaluated. | Routed storage seams can be inspected. |
| L1 Inspectable | Isolation seams and namespace usage can be inspected. | Prove routed seams and variant-aware namespaces. |
| L2 Safe | Destructive E2E playbooks are statically routed to isolated test storage. | Clear the remaining isolation findings. |
| L3 Clean | Storage isolation checks are clean. | Maximum isolation-safety maturity reached. |

The other capabilities share the same L0→top shape: schema must be coherent
(`schema_substrate`), persistence hygienic (`persistence_hygiene` — the fallback
capability), the target classifiable (`target_classification`), and backup/migration
posture visible (`operational_readiness`).

## What each finding means

Each finding caps the capability it names at a rung; only ERROR/BLOCKER
severities fail the phase, so many hygiene/advisory findings are honest,
non-failing debt.

| Code | Capability | Caps at | Severity | Fails phase? |
|---|---|---|---|---|
| `ROUTED_SEAMS_UNWIRED` | isolation_safety | L2 | ERROR | Yes |
| `STORAGE_ISOLATION_UNVERIFIED` | isolation_safety | L2 | WARNING | No |
| `SCHEMA_CENTRALIZED` | schema_substrate | L2 | ERROR | Yes |
| `SCHEMA_NOT_IDEMPOTENT` | schema_substrate | L2 | ERROR | Yes |
| `RAW_SQL_OPEN` | persistence_hygiene | L2 | ERROR | Yes |
| `ROUTED_DRIVER_IMPORT` | persistence_hygiene | L2 | ERROR | Yes |
| `DB_ROWS_NOT_CLOSED` | persistence_hygiene | L2 | ERROR | Yes |
| `SQLITE_POOL_DEADLOCK` | persistence_hygiene | L2 | ERROR | Yes |
| `BACKUP_TARGET_MISSING` | operational_readiness | L1 | INFO | No |
| `MIGRATION_DEBT` | operational_readiness | L1 | INFO | No |

The full finding inventory (schema layout, cross-domain FKs, per-domain
ownership, handle capture, namespace hardcoding, direct SQL in handlers) is
declared in the descriptor's `maturity.findings` block.

## The canonical fix

- **Isolation** (`ROUTED_SEAMS_UNWIRED`) → wire the four routed test-DB seams (`database.Open`→`*RoutedDB`, `database.EnsureSchemas`, `apihttp.TestModeMiddleware`, `devrouting.Register`); `STORAGE_ISOLATION_UNVERIFIED` on a non-Go API means porting to the Go seams or adding an equivalent verifiable mechanism.
- **Schema** (`SCHEMA_CENTRALIZED`, `SCHEMA_NOT_IDEMPOTENT`, `SCHEMA_HAS_ALTER`, `ENSURE_SCHEMAS_NOT_WIRED`) → split into per-domain `schema.sql`, make DDL idempotent (`IF NOT EXISTS`), relocate `ALTER` into a migration, and wire schemas through the supported seam. Several are auto-fixable.
- **Persistence** (`RAW_SQL_OPEN`, `ROUTED_DRIVER_IMPORT`, `SQL_DB_HANDLE_CAPTURE`, `DB_ROWS_NOT_CLOSED`, `DIRECT_SQL_IN_HANDLERS`, `SQLITE_POOL_DEADLOCK`) → route access through the `api-core/database` repository seam, close rows, and restructure `MaxOpenConns:1` nested-query flows. `DB_ROWS_NOT_CLOSED` is auto-fixable.
- **Operational** (`BACKUP_TARGET_MISSING`, `MIGRATION_DEBT`) → advisory: register a data-backup-manager target or author a stage-appropriate migration; these never fail the phase.

Many findings are auto-fixable — `storage-health fix preview|apply <scenario>`
previews/applies format-preserving repairs (dry-run by default).

## How to verify

```bash
# See the current rung, gaps, and next move for every capability:
storage-health validate scenario <scenario>

# Or drive it through Test Genie and read the per-phase scorecard:
test-genie execute <scenario> --phases storage
test-genie runs findings --scenario <scenario>
```

## What Gets Validated

storage-health runs static, Go-first analyzers across four concerns:

- **Schema structure & layout** — per-domain `schema.sql`, embedded +
  idempotent schemas, no `ALTER` in schema files, no centralized schema, no
  cross-domain hard foreign keys.
- **Isolation & shadow-safety (the safety throughline)** — the four routed
  test-DB seams (`database.Open`→`*RoutedDB`, `database.EnsureSchemas`,
  `apihttp.TestModeMiddleware`, `devrouting.Register`) are statically wired, and
  storage namespaces use the variant-aware helpers. A non-Go API that cannot be
  statically verified surfaces an explicit `STORAGE_ISOLATION_UNVERIFIED`
  fail-safe instead of a silent pass.
- **Persistence hygiene** — no raw `sql.Open`, routed driver imports, raw
  `*sql.DB` handle capture, unclosed `rows`, direct SQL in handlers, or the
  SQLite single-connection nested-query deadlock pattern.
- **Operational readiness** — backup/restore-target registration for
  data-persisting scenarios (advisory).

Many findings are **auto-fixable** — `storage-health fix preview|apply
<scenario>` previews/applies format-preserving repairs (dry-run by default).

## Severity, Gating & the Isolation Precondition

Findings carry a severity; `ERROR`/`BLOCKER` findings fail the phase. The
isolation rung (L2) is the safety core: `ROUTED_SEAMS_UNWIRED` is an `ERROR`, so
a Go scenario whose routed-DB seams are not wired **fails the storage phase**.

The workflow phase keys mutating execution off the same L2 verdict through
workflow-health. When isolation is statically proven, workflow-health can run
destructive flows safely. When it cannot be proven — unwired seams, or a non-Go
API that cannot be verified — workflow-health refuses those flows before BAS is
called. There is no restart-based fallback: the unverified restart path was
deleted in favour of this static, fail-closed gate.

## Resilience

Storage runs before workflow, so it depends on the `storage-health` provider
being reachable. Test Genie's delegated-phase path treats an unreachable
provider as an environmental failure (`storage-health` is a non-optional safety
provider; start it via `vrooli scenario start storage-health`).

## Related Documentation

- [storage-health scenario](../../../../storage-health/) — the provider that
  backs this phase (analyzers, maturity ladder, auto-fix, fleet intelligence)
- [Workflow Phase](../workflow/README.md) — the destructive E2E phase this
  gate protects
- [Routed Test-Storage Contract](../../../../docs/agent-system/routed-test-db.md) —
  the canonical routed database and file-storage contract

## See Also

- [Phases Overview](../README.md) - All phases
- [Workflow Phase](../workflow/README.md) - Next phase
