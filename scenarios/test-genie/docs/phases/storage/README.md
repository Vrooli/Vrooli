# Storage Phase

**ID**: `storage`
**Timeout**: 120 seconds
**Optional**: No
**Source**: validation-provider (`storage-manager`)

The storage phase owns Test Genie's storage judgment and — most importantly —
its **test-isolation safety gate**. It runs immediately before the `workflow`
phase so that its verdict can decide whether destructive end-to-end flows are
allowed to run at all.

Test Genie does not analyze storage natively. The phase is **delegated to the
[`storage-manager`](../../../../storage-manager/) scenario** through the shared
`ScenarioValidationService` contract (the same delegation model used by
structure-health, unit-health, security-health, and the other provider-backed
phases). Test Genie calls `storage-manager validate scenario <scenario>`, maps
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

The phase aggregates six per-capability ladders; the safety-critical one is
`isolation_safety` (its L2 verdict gates whether destructive workflow flows may
run). The rungs are monotone.

`declaration_accountability` is target-accountability evidence. The per-owner
markers move the ladder, while `STORAGE_OWNER_GATE_FAILED` is the aggregate
gating finding when an owner is unsafe. Its rungs are:

| Rung | Meaning | Blocked by |
|---|---|---|
| L0 Unknown | The owner declares no `storage.entries`, so its durable surface cannot be evaluated. | `STORAGE_ACCOUNTABILITY_UNDECLARED` (INFO) |
| L1 Declared | Storage intent is present, but reconciliation is incomplete. | `STORAGE_ACCOUNTABILITY_UNRECONCILED` (WARNING) |
| L2 Reconciled | Writers, observed paths, sidecars, and ceilings agree. | `STORAGE_ACCOUNTABILITY_UNGOVERNED` (WARNING) |
| L3 Governed | Every non-sidecar entry carries a budget or a reclaim command. | — |

These three markers are emitted by the `storage.accountability` post-pass, which
runs after every analyzer because the rung is a function of their combined
output. It always runs: the maturity engine starts each capability at its top
level and lowers it only on a blocking finding, so an owner that declares
nothing produces no findings and would otherwise score "governed end to end".
Only the lowest blocked rung is reported, so an owner is never told to fix
governance while reconciliation is still open.

The per-entry findings below explain *why* a rung is blocked; they are advisory
and do not move the ladder themselves. `STORAGE_TOKEN_SUPERSEDABLE` is
deliberately excluded from the reconciliation set: a repeated `byOS` map is
verbose, not wrong.

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
| `STORAGE_ACCOUNTABILITY_UNDECLARED` | declaration_accountability | L1 | INFO | No |
| `STORAGE_ACCOUNTABILITY_UNRECONCILED` | declaration_accountability | L2 | WARNING | No |
| `STORAGE_ACCOUNTABILITY_UNGOVERNED` | declaration_accountability | L3 | WARNING | No |
| `STORAGE_BUDGET_BELOW_OBSERVED` | declaration_accountability | L2 | ERROR | Yes |

The remaining accountability codes are advisory detail: they explain which rung
marker fired but do not move the ladder on their own. All are WARNING and none
fail the phase. Those in the **reconciliation set** hold the ladder at L1 via
`STORAGE_ACCOUNTABILITY_UNRECONCILED`:

| Code | In reconciliation set? |
|---|---|
| `STORAGE_ENTRY_NO_WRITER` | Yes |
| `STORAGE_ENTRY_CLASS_CONFLICT` | Yes |
| `STORAGE_SQLITE_SIDECAR_UNDECLARED` | Yes |
| `STORAGE_RETENTION_CONFLICT` | Yes |
| `STORAGE_PATH_NOT_PORTABLE` | Yes |
| `STORAGE_PATH_PLATFORM_MISMATCH` | Yes |
| `STORAGE_PATH_BRANCH_MISSING` | Yes |
| `STORAGE_PATH_UNACCOUNTED` | Yes (census-only; not reachable from this phase) |
| `STORAGE_PATH_ORPHANED` | Yes (census-only; not reachable from this phase) |
| `STORAGE_ENTRY_UNRECLAIMABLE` | No — it is a governance gap, reported at L3 |
| `STORAGE_TOKEN_SUPERSEDABLE` | No — verbose, not wrong |

The full finding inventory (schema layout, cross-domain FKs, per-domain
ownership, handle capture, namespace hardcoding, direct SQL in handlers) is
declared in the descriptor's `maturity.findings` block.

Fleet-wide validation of resources, tools, safeguards, and scenarios is a
storage-manager product gate (`storage-manager validate fleet`). This phase
additionally reports on non-scenario owners inside the `storage-manager`
scenario run: `storage` declares `targets.kinds` of `scenario`, `resource`,
`tool`, and `safeguard`, and every finding about an owner other than the run's
own target carries a `subject` naming it.

**A foreign subject never moves this run's ladder.** The maturity engine scores
a capability only from findings whose subject is the run's own target, so an
undeclared safeguard cannot pull a scenario's `declaration_accountability` down.
The aggregate `STORAGE_OWNER_GATE_FAILED` finding is what carries fleet state
into the run's verdict. Before that scoping existed, one safeguard with no
`storage.entries` reported `storage-manager` itself at L0 while its own storage
was fully governed.

## The canonical fix

- **Isolation** (`ROUTED_SEAMS_UNWIRED`) → wire the four routed test-DB seams (`database.Open`→`*RoutedDB`, `database.EnsureSchemas`, `apihttp.TestModeMiddleware`, `devrouting.Register`); `STORAGE_ISOLATION_UNVERIFIED` on a non-Go API means porting to the Go seams or adding an equivalent verifiable mechanism.
- **Schema** (`SCHEMA_CENTRALIZED`, `SCHEMA_NOT_IDEMPOTENT`, `SCHEMA_HAS_ALTER`, `ENSURE_SCHEMAS_NOT_WIRED`) → split into per-domain `schema.sql`, make DDL idempotent (`IF NOT EXISTS`), relocate `ALTER` into a migration, and wire schemas through the supported seam. Several are auto-fixable.
- **Persistence** (`RAW_SQL_OPEN`, `ROUTED_DRIVER_IMPORT`, `SQL_DB_HANDLE_CAPTURE`, `DB_ROWS_NOT_CLOSED`, `DIRECT_SQL_IN_HANDLERS`, `SQLITE_POOL_DEADLOCK`) → route access through the `api-core/database` repository seam, close rows, and restructure `MaxOpenConns:1` nested-query flows. `DB_ROWS_NOT_CLOSED` is auto-fixable.
- **Operational** (`BACKUP_TARGET_MISSING`, `MIGRATION_DEBT`) → advisory: register a data-backup-manager target or author a stage-appropriate migration; these never fail the phase.

Many findings are auto-fixable — `storage-manager fix preview|apply <scenario>`
previews/applies format-preserving repairs (dry-run by default).

## How to verify

```bash
# See the current rung, gaps, and next move for every capability:
storage-manager validate scenario <scenario>

# Or drive it through Test Genie and read the per-phase scorecard:
test-genie execute <scenario> --phases storage
test-genie runs findings --scenario <scenario>
```

## What Gets Validated

storage-manager runs static, Go-first analyzers across four concerns:

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

Many findings are **auto-fixable** — `storage-manager fix preview|apply
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

Storage runs before workflow, so it depends on the `storage-manager` provider
being reachable. Test Genie's delegated-phase path treats an unreachable
provider as an environmental failure (`storage-manager` is a non-optional safety
provider; start it via `vrooli scenario start storage-manager`).

## Boundary with storage-manager product acceptance

The static phase is intentionally not a host census. It checks isolation,
schema, persistence, and migration hygiene quickly for the target scenario.
The storage-manager product surface owns the slower, host-state-dependent
checks: owner inventory across scenarios/resources/tools/safeguards, census
accounting, retention parsing, placement resolution, adoption suggestions, and
degraded-confidence API behavior. Those checks are exercised by the focused Go
and UI suites and by the comprehensive `vrooli scenario test storage-manager`
run. Keeping the boundary explicit prevents a green static phase from being
misread as proof of fleet-wide byte accounting.

## Related Documentation

- [storage-manager scenario](../../../../storage-manager/) — the provider that
  backs this phase (analyzers, maturity ladder, auto-fix, fleet intelligence)
- [Workflow Phase](../workflow/README.md) — the destructive E2E phase this
  gate protects
- [Routed Test-Storage Contract](../../../../docs/agent-system/routed-test-db.md) —
  the canonical routed database and file-storage contract

## See Also

- [Phases Overview](../README.md) - All phases
- [Workflow Phase](../workflow/README.md) - Next phase
