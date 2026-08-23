# Performance — Unit Health

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
| UI build | 5-10 minutes accepted for current Vite module graph | lifecycle/test-genie build logs | inherited |
| API health | responsive under lifecycle health timeout | `/health` check | active |
| UI health | responsive under lifecycle health timeout | `/health` check | active |

## Current Measurements

| Measurement | Value | Source | Date |
|---|---|---|---|
| Web Console representative Vitest file | 5 repeated warm samples: median/p90 1.44/1.81 s wall, 1.83/2.21 s user CPU, 145,048/150,108 KiB peak RSS | `/usr/bin/time` + `vitest run src/stores/voiceSilenceDefaults.test.ts` | 2026-08-21 |
| Swarm Manager representative Vitest file | 5 repeated warm samples: median/p90 4.55/4.62 s wall, 1.94/2.01 s user CPU, 149,472/153,308 KiB peak RSS | `/usr/bin/time` + `vitest run src/stores/agent-session-store.test.ts` | 2026-08-21 |
| React Component Library representative Vitest file | 5 repeated warm samples: median/p90 1.66/1.74 s wall, 1.93/1.98 s user CPU, 177,984/180,524 KiB peak RSS | `/usr/bin/time` + `vitest run src/components/adopted-foundations.test.tsx` | 2026-08-21 |

These are five repeated warm targeted smoke samples, not the required five
cold/five warm full-suite measurements or the three-suite geometric-mean
comparison. They establish repeatable behavior for representative migrated
tests, but do not prove the Phase 11 or Phase 13 performance acceptance
thresholds.

The executor now uses typed, shell-free commands and does not install
dependencies while validating a workspace. Dependency readiness is an
observational input owned by Scenario Dependency Analyzer. This prevents a
validation run from paying package-manager startup cost or mutating the
workspace.

Unit evidence keys include source/configuration and lockfile digests,
toolchain, adapter and policy versions, runner profile, host OS/architecture,
relevant environment, coverage mode, and artifact schema. Complete records are
written by atomic rename, integrity checked on read, quarantined when corrupt,
and evicted deterministically under byte and age budgets. A cache miss is the
safe result for incomplete, stale, or damaged evidence. Responses expose cache
hit/miss state, invalidated dimensions, saved wall-time, and retained bytes;
the executor also records portable process CPU time and peak RSS for reuse and
resource-envelope accounting.

Concurrent requests for the same exact key are coalesced behind one in-flight
validation. Waiters re-check the evidence store after the owner commits, so an
unchanged burst starts one runner child instead of one child per request; failed
or non-cacheable owners release the flight without manufacturing reusable evidence.

An exact warm-cache hit still performs discovery, workspace planning, and governed
dependency readiness before reuse, then skips source analyzers and execution. This
keeps the fast path inexpensive while ensuring a dependency that became missing or
stale cannot be hidden by an older successful record.

Callers that need lightweight test evidence can set `fast_test_only`; that mode
plans the adapter's test command, omits coverage artifacts, and uses a distinct
cache key from the default coverage-capable execution. The default remains
coverage-capable for existing Test Genie and CLI callers.

Workspace-scoped requests are filtered immediately after discovery, before plan
construction, static analysis, cache-key construction, or execution. An unknown
selector produces an explicit absent-surface result rather than silently paying
the cost of validating every discovered workspace.

Command planning is owned by versioned adapters (`go`, `react-vitest`,
`node-jest`, `python-pytest`, `rust-cargo`, `bash-bats`, and
`powershell-pester`). The validation kernel consumes their typed plans and
only owns universal scheduling, containment, findings, and evidence rules.
Adapters also declare bounded artifact kinds and paths; coverage parsing is
performed from those declarations rather than by a language switch in the
kernel. The generic policy leaves package-manager and projection vocabularies
open for future adapters.
Unsupported host/framework matches fail closed before a runner child starts.

The three known slow UI suites now declare named bounded profiles: Web
Console uses `bounded-batches`, Swarm Manager uses `serial-isolation`, and
React Component Library uses `coverage-isolation`. Each is an explicit
one-worker profile with a declared 2 GiB memory ceiling and larger
no-output/total-time bounds appropriate to its current isolation evidence; the
default fleet profile declares a 512 MiB ceiling. The profile is the policy source, while the
scenario scripts remain responsible only for their runner-specific batching or
coverage behavior. Parallelism should increase only after native leak, handle,
RSS, and shuffled-order evidence is captured.

Dependency readiness is a typed boundary: missing, stale, unavailable, and
unsupported requirements carry their source and governed remediation. A
production validation request checks Scenario Dependency Analyzer's O(1)
`DescribeProvider` contract before execution; a non-ready report produces a
dependency finding and starts no runner children. Hermetic policies that this host cannot enforce
(for example open-handle
observation, or shuffled-order execution) fail closed as unsupported rather
than silently running with weaker guarantees. Temporary-root execution and
parent-environment isolation are enforced directly by the typed executor.

The executor exposes this distinction through `HostHermeticCapabilities`: the
portable guarantees currently are temporary-root projection and parent
environment restoration; native network denial and workspace read-only mounts
are additionally enabled when the host sandbox probe succeeds. Linux hosts
with `/proc` process-group metadata also support descendant-leak observation.
Declared egress, open-handle observation, and order-independent replay remain
unsupported until a dedicated platform-specific adapter is installed.

Focused validation evidence captured 2026-08-21: the Unit Health adapter,
executor, readiness, evidence, and validation packages passed targeted Go tests
and `go vet`; executor test binaries compiled for Linux, macOS, and Windows.
The native CI matrix now uploads a per-OS `unit-health-native-evidence-*`
artifact containing JSON executor test events and a runner/toolchain manifest.
The matrix also runs the native `packages/platform-go` process-substrate tests;
their outcome is recorded as `outcomes.platformGo` in the manifest.
The full scenario suites and historical performance baseline were intentionally
not run during this implementation pass; the uploaded native artifacts remain
the authoritative place to record those OS-specific results when CI executes.

### Native CI Runbook

The authoritative native gate is job `unit-health-executor` in
`.github/workflows/test.yml`. It runs on `ubuntu-latest`, `macos-latest`, and
`windows-latest`, and can be started with the workflow's manual dispatch after
the changes are published. Each matrix leg uploads
`unit-health-native-evidence-<runner>` (for example,
`unit-health-native-evidence-ubuntu-latest`); inspect its
`native-evidence/manifest.json` for step outcomes and
`native-evidence/executor-tests.json` for the JSON test events. A local
cross-compile or focused test run is useful evidence, but it does not replace
these hosted native results.

## Known Constraints

- Vite production builds may process thousands of modules and take
  several minutes.
- Performance budgets for real product workflows must be defined after
  domains and UX flows are known.

## Regression Procedure

1. Run the focused API package tests:
   `cd api && go test ./internal/evidence ./internal/executor`.
2. Capture relevant API/UI command timing without installing dependencies from
   Unit Health.
3. For UI interaction regressions, use `ui/perf/README.md` and the
   provided capture template.
4. Record persistent findings in this document or
   [`PROBLEMS.md`](PROBLEMS.md) depending on whether they are accepted
   constraints or unresolved debt.

## Cross-References

- [`../operations/OBSERVABILITY.md`](../operations/OBSERVABILITY.md) — signals and telemetry
- [`../operations/DEPLOYMENT.md`](../operations/DEPLOYMENT.md) — release checklist
- [`TESTING.md`](TESTING.md) — coverage and test expectations
- [`PROBLEMS.md`](PROBLEMS.md) — unresolved performance debt
