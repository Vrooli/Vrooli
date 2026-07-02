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
- [Test-Isolation Contract](../../../../storage-health/docs/concepts/test-isolation-contract.md) —
  the canonical routed test-DB contract owned by storage-health

## See Also

- [Phases Overview](../README.md) - All phases
- [Workflow Phase](../workflow/README.md) - Next phase
