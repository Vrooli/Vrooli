# Performance — Security Health

This document records performance budgets, current measurements, known
constraints, and regression procedures.

## Purpose Of This Document

Use this document to answer:

- What performance matters for this scenario?
- What budgets or thresholds apply?
- How are measurements captured?
- What performance risks remain?

## Budgets

| Surface | Budget | Measurement | Status |
|---|---|---|---|
| Unchanged validation scan | `num[target]:500ms` | Connect response `metrics.stages[name=scan].duration_ms` | active |
| Scanner admission capacity | `num[threshold]:3-32` weighted units; default `num[decision]:4` | `SECURITY_HEALTH_SCANNER_CAPACITY` and scanner child-stage gauges | active |
| Scanner child CPU | host quarter, floor 2 | `SECURITY_HEALTH_SCANNER_MAX_PROCS` → child `GOMAXPROCS` | active |
| Idle reconcile cadence | `5m` base, `1h` ceiling | `SECURITY_HEALTH_RECONCILE_MAX_INTERVAL` with reset-on-change | active |
| Advisory freshness | stable per-scenario UTC-hour identity, 25h TTL | fingerprint epoch; not a timer-only cache | active |
| API/UI health | lifecycle health timeout | `/health` checks | active |

## Current Measurements

| Measurement | Value | Source | Date |
|---|---|---|---|
| Pre-change managed validation | `num[sot]:35.752s` scan stage | plan baseline measurement | 2026-08-20 |
| Post-change forced-cold provider call | `num[sot]:28.60s` wall | managed `security-health validate scenario` | 2026-08-20 |
| Post-change unchanged warm provider call | `num[sot]:145ms` wall; `num[sot]:105ms` scan stage | Connect execution metrics | 2026-08-20 |
| Provider speedup | about `num[sot]:197x` cold-to-warm | preceding provider measurements | 2026-08-20 |
| Test Genie security phase | baseline `num[sot]:33s`; security-only candidate `num[sot]:14s`; final warm comprehensive candidate `num[sot]:1s` | runs `20260820-193304-0a8525d1`, `20260820-203812-8ae01d61`, and `20260820-205634-7ca2265c` | 2026-08-20 |
| Concurrent identical gosec miss | `num[sot]:826ms` each; one execution and one coalesced result | managed child-stage gauges after targeted row invalidation | 2026-08-20 |
| Documentation-only mutation | `num[sot]:290ms`; gitleaks executed, every dependency/SAST scanner hit cache | managed child-stage gauges | 2026-08-20 |

The Test Genie phase includes provider discovery, lifecycle, transport, and
result processing; the provider metric is the authoritative measure of scanner
work saved. The final comprehensive Test Genie comparison classified run
`20260820-205634-7ca2265c` as compatible and pre-existing against the immutable
baseline. Git Control Tower operation
`ffb577e3-b778-4b45-9831-8ac3f5ec1d3e:1` independently classified the candidate
clean with only pre-existing failures.

## Incremental Validation Model

Every default scanner implements `IncrementalScanner.EvidencePlan`. The
fingerprint is a SHA-256 digest over a framed format version, scanner identity,
normalization-policy version, scanner executable identity, relevant file paths
and contents, and (for advisory-backed scanners) a stable per-scenario UTC-hour epoch. A cache hit is
valid only for an exact fingerprint and unexpired normalized payload.

| Scanner | Relevant input boundary | Weight | Freshness |
|---|---|---:|---|
| gitleaks | tracked plus non-ignored untracked files, staged into an exact temporary snapshot | 1 | content/tool/policy changes |
| gosec | first-party module trees, excluding generated/vendor directories | 2 | content/tool/policy changes |
| govulncheck | first-party Go module trees | 3 | per-scenario UTC hour plus content/tool/policy changes |
| pnpm-audit | first-party package trees and lockfiles | 2 | per-scenario UTC hour plus content/tool/policy changes |
| osv-scanner | supported manifests/lockfiles plus adjacent `go.sum` resolution input | 2 | per-scenario UTC hour plus content/tool/policy changes |

Gitleaks intentionally gates the commit-eligible inventory. Ignored runtime
databases, installed packages, build output, and local secret files cannot enter
a commit; scanning them previously created noise and caused the cache to
invalidate itself when SQLite changed.

The validation cache stores normalized `Finding` fields through an explicit
allowlist. Raw stdout/stderr, secret matches, source snippets, and scanner-native
objects never cross the persistence boundary. Failed, cancelled, corrupt,
expired, or un-fingerprintable work executes normally and is not reused.

OSV validation and dependency reconciliation share the same raw-report
fingerprint and cache contract. This removes redundant advisory work only where
lockfile, tool, policy, and freshness semantics are identical.

## Known Constraints

- Fingerprinting still reads relevant first-party content on every request; it
  is intentionally content-based rather than path-and-mtime-based.
- Cold scanners remain serial inside one request because the existing metrics
  collector is single-flight. The shared admission budget still bounds and
  coalesces work across concurrent requests.
- The normalized evidence table retains one row per `(scenario, scanner)` and
  therefore grows with fleet membership rather than request count.

## Observability

The public response shape is unchanged. Its existing execution metrics now
contain `scan` children named `scanner:<name>`. Each applicable scanner reports
`available`, `fingerprint_ms`, `weight`, `cache_hit`, `cache_miss`, `coalesced`,
`executed`, `uncached`, `failed`, `cache_error`, `admission_wait_ms`,
`execution_ms`, and `findings`, plus the standard per-stage resource sample.

- `cache_hit=1` and `executed=0` means saved scanner work.
- `coalesced=1` means another identical request produced the shared result.
- `cache_error=1` with `executed=1` is safe degradation, not a skipped check.
- `uncached=1` means no trustworthy identity was available; admission still applied.
- `available=0` means the applicable binary was absent and normal degraded evidence was emitted.

## Safe Invalidation

Normal invalidation is automatic: change relevant content, the scanner binary,
the normalization-policy version, or advance the advisory day. Operators should
not edit fingerprints or payload JSON. For a forced-cold diagnostic, stop the
scenario through its lifecycle, back up the scenario-owned SQLite file, delete
only the target `(scenario, scanner)` row from `validation_evidence_cache`, and
restart through the lifecycle. The derived row repopulates on the next request.

## Regression Procedure

1. Run the race-enabled validation, cache, dependency, and handler package tests.
2. Run `vrooli scenario test security-health security` and retain the run id.
3. Through one managed API process, capture cold and unchanged warm responses;
   compare findings and scanner child-stage gauges.
4. Compare the candidate to the immutable baseline through Git Control Tower.
5. Record persistent findings in this document or
   [`PROBLEMS.md`](PROBLEMS.md) depending on whether they are accepted
   constraints or unresolved debt.

## Cross-References

- [`../operations/OBSERVABILITY.md`](../operations/OBSERVABILITY.md) — signals and telemetry
- [`../operations/DEPLOYMENT.md`](../operations/DEPLOYMENT.md) — release checklist
- [`TESTING.md`](TESTING.md) — coverage and test expectations
- [`PROBLEMS.md`](PROBLEMS.md) — unresolved performance debt
