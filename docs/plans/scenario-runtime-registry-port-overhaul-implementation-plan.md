# Scenario Runtime Registry and Port Overhaul Implementation Plan

## Phase Progress Ledger

Use this ledger as the durable handoff surface. Each phase should be completed, validated, and recorded before starting the next phase unless the user explicitly approves parallel work.

| Phase | Status | Owner | Evidence / Notes |
| --- | --- | --- | --- |
| 0. Baseline inventory and decision freeze | Completed 2026-05-08 | Codex | Contracts confirmed through `vrooli help` and per-command help. Test inventory and baseline recorded in [Phase 0 Completion Record](#phase-0-completion-record). No runtime behavior changed. |
| 1. Runtime registry package | Completed 2026-05-08 | Codex | Added `internal/scenarioruntime` domain types, repository contracts, SQLite schema/store, explicit clock/path seams, and focused unit tests. See [Phase 1 Completion Record](#phase-1-completion-record). No lifecycle, CLI, port allocation, maintenance, or scenario runtime behavior is wired to the registry yet. |
| 2. Lease, heartbeat, and health snapshots | Completed 2026-05-08 | Codex | Added explicit lease/heartbeat/expiry/stop APIs, heartbeat TTL policy, fixed-width timestamp ordering for SQLite deadline comparisons, health probe adapter, and focused tests. See [Phase 2 Completion Record](#phase-2-completion-record). No lifecycle, CLI, port allocation, maintenance, or scenario runtime behavior is wired to the registry yet. |
| 3. Lifecycle dual-write | Completed 2026-05-08 | Codex | Added opt-in lifecycle registry writes behind `VROOLI_RUNTIME_REGISTRY=dual|prefer|strict`, including start leases, reserved/bound claims, process refs, health snapshots, failed-start compensation, and stop release/stop transitions. See [Phase 3 Completion Record](#phase-3-completion-record). Legacy records and lock files remain production read authority. |
| 4. Registry-backed read side | Completed 2026-05-08 | Codex | Added opt-in registry reads behind `VROOLI_RUNTIME_REGISTRY=prefer|strict` for inventory/status/detail/port resolution, centralized migration-mode parsing, and covered prefer/strict/fallback behavior with unit tests. See [Phase 4 Completion Record](#phase-4-completion-record). Default empty/off and `dual` modes remain legacy-read-authoritative. |
| 4H. Phase 0-4 hardening pass | Completed 2026-05-08 | Codex | Consolidated runtime port request resolution into the scenario runtime model, aligned app-layer specific-port lookup with orchestrator running-state policy, added migration-mode tests, and revalidated internal/API-core packages. See [Phase 0-4 Hardening Completion Record](#phase-0-4-hardening-completion-record). No scenario was stopped or restarted. |
| 5. Registry-backed port allocation and claims | Completed 2026-05-08 | Codex | Moved opt-in lifecycle allocation to reserve registry claims inside `internal/ports` before legacy lock compatibility writes, added reserved-claim expiry and rollback, cleared claim expiry on bind, and covered race/stale/socket conflict behavior with focused tests. See [Phase 5 Completion Record](#phase-5-completion-record). Default empty/off mode remains legacy-only; no scenarios were stopped or restarted. |
| 5H. Phase 0-5 hardening pass | Completed 2026-05-08 | Codex | Centralized runtime claim URL construction, made latest-registry-instance selection independent of store ordering, clarified abandoned-reservation cleanup boundaries, added focused tests, and revalidated internal/API-core packages. See [Phase 0-5 Hardening Completion Record](#phase-0-5-hardening-completion-record). No scenario was stopped or restarted. |
| 5H2. Phase 0-5 hardening pass 2 | Completed 2026-05-08 | Codex | Hardened allocator-owned rollback for partial multi-port registry allocation failures, abandoned compatibility locks for newly-created rolled-back claims, centralized active runtime status policy in `internal/scenarioruntime`, added focused regression tests, and revalidated internal/API-core packages. See [Phase 0-5 Hardening Round 2 Completion Record](#phase-0-5-hardening-round-2-completion-record). No scenario was stopped or restarted. |
| 6. Maintenance, diagnostics, and autoheal update | Completed 2026-05-08 | Codex | Added registry-aware maintenance diagnostics, cleanup of abandoned startup leases/reserved claims, registry process refs in host snapshots, CLI/JSON rendering for registry claims, and autoheal's CLI-backed lock/claim reader with filesystem fallback. See [Phase 6 Completion Record](#phase-6-completion-record). No scenario was stopped or restarted. |
| 7. Browser automation studio discovery cleanup | Completed 2026-05-08 | Codex | Removed BAS `SCENARIO_REGISTRY` production discovery path, routed runtime port/URL resolution only through the shared scenario CLI/API-core discovery seam, updated docs, and added regression tests proving registry env overrides are ignored. See [Phase 7 Completion Record](#phase-7-completion-record). No scenario was stopped or restarted. |
| 8. Sandbox validation | Completed 2026-05-08 | Codex | Added strict-registry sandbox-like regression coverage, validated workspace-sandbox through lifecycle with registry writes enabled, confirmed strict registry `scenario status`/`scenario port` resolution and API `/health` reachability, and kept web-console running. See [Phase 8 Completion Record](#phase-8-completion-record). |
| 8A. Allowlist rollout guardrail | Completed 2026-05-08 | Codex | Added `VROOLI_RUNTIME_REGISTRY_ALLOWLIST` to scope `dual`/`prefer`/`strict` migration behavior to selected scenarios during soak, with lifecycle write gating, orchestrator read gating, focused tests, and non-invasive live status checks. See [Phase 8A Completion Record](#phase-8a-completion-record). No scenario was stopped or restarted. |
| 8B. Crash/reboot and sudden-stop reconciliation | Completed 2026-05-09 | Codex | Added host boot/session metadata, schema v2 migration, domain reconciliation, strict/prefer read-side enforcement, maintenance diagnostics/cleanup, and crash-regression tests. Adoption is explicitly deferred; strict fails closed until a scenario is restarted under registry-enabled lifecycle. See [Phase 8B Completion Record](#phase-8b-completion-record). |
| 8C. Registry authority refactor and hardening | Completed 2026-05-09 | Codex | Tightened the reconciliation boundary so claim-level authority lives in `internal/scenarioruntime`, centralized host evidence construction and runtime/claim status policies, and prevented reserved registry claims from surfacing as runtime ports. See [Phase 8C Completion Record](#phase-8c-completion-record). |
| 8D. Runtime supervisor and heartbeat authority | Completed 2026-05-09 | Codex | Added the central runtime supervisor, schema v3, supervisor sessions, lifecycle auto/on ensure, strict stale supervised-lease enforcement, diagnostics, bounded health probe execution, Linux systemd user-service install/uninstall, and live allowlist soak for workspace-sandbox, agent-manager, and swarm-manager. `vrooli locks --json` reports zero authoritative registry claims with `lease_fresh=false`. See [Phase 8D Completion Record](#phase-8d-completion-record) and `docs/plans/runtime-supervisor-heartbeat-authority-implementation-plan.md`. |
| 9. Legacy cleanup | Not started |  |  |

## Purpose

Replace Vrooli's current PID/proc/file-lock-centered scenario runtime and port discovery model with a platform-neutral runtime registry that works from host processes, sandboxes, and future non-Linux environments. The end state must be more reliable, more observable, easier to test, and easier for agents to reason about than the current implementation.

This is critical infrastructure. Do not implement it as a single hard cutover. The intended migration is dual-write, validate, switch reads, validate again, then remove legacy authority in a final cleanup phase.

## Required Reading

Future agents must run these commands before changing code for this plan:

```bash
prompt-manager skill read implementation-plan-authoring screaming-architecture-audit utils-unification seam-discovery-and-enforcement test boundary-of-responsibility-enforcement decision-boundary-extraction
prompt-manager skill read cli-steer api-steer utils-unification seam-discovery-and-enforcement
```

Also review:

- `docs/reference/port-allocation.md`
- `docs/operations/troubleshooting.md`
- `internal/process/`
- `internal/ports/`
- `internal/lifecycle/`
- `internal/orchestrator/`
- `internal/maintenance/`
- `internal/scenario/runtime_state.go`
- `packages/api-core/discovery/`
- `packages/api-core/health/`
- `scenarios/vrooli-autoheal/api/internal/checks/`
- `scenarios/vrooli-autoheal/api/internal/healing/`
- `scenarios/browser-automation-studio/api/internal/scenarioport/`

## Problem Statement

The current runtime and port discovery model treats host process liveness as the source of truth:

- `internal/process.LiveRecords` filters records by PID liveness.
- Linux liveness depends on `/proc` and `signal(0)`.
- Non-Linux liveness currently reports false for every process.
- Runtime records live as JSON and PID files under `$HOME/.vrooli/processes/scenarios/<scenario>/`.
- Port locks are plain files under `$HOME/.vrooli/state/scenarios/.port_<port>.lock`.
- Scenario list/status/port resolution eventually depends on those records being visible and PID-probable from the current execution context.

That model is not suitable for workspace sandboxes. A sandbox can need to resolve host-running scenario ports through the same CLI/API path as the host, but host PIDs and `/proc` are not reliably visible inside the sandbox. The result is that valid running scenarios can appear dead, causing API-core discovery, agent-manager spawned agents, and sandboxed workflows to fail.

The current setup also leaves too much correctness to cleanup paths:

- stale lock files can survive crashes or lock-before-bind windows;
- orphan listeners can remain after unclean termination;
- autoheal and maintenance commands need to inspect and heal those states;
- `vrooli diagnose-port` currently reports locks, listeners, and host orphans, but not a durable registry claim model.

The redesign must retain orphan/stale detection as operational diagnostics while removing them as the normal source of truth for discovery.

## Scope

In scope:

- New runtime registry package and storage schema.
- Registry-owned scenario instance lifecycle, port claims, heartbeats, and health snapshots.
- Registry-backed `vrooli scenario list`, `vrooli scenario status`, and `vrooli scenario port`.
- Dual-write and fallback migration from legacy JSON records and plain lock files.
- Registry-aware `vrooli locks`, `vrooli cleanup locks`, `vrooli orphans`, `vrooli cleanup orphans`, and `vrooli diagnose-port`.
- Autoheal checks and healing strategies that currently read legacy Vrooli state.
- Browser automation studio cleanup so it uses `api-core/discovery` only, without `SCENARIO_REGISTRY` as a runtime escape hatch.
- Tests for race prevention, stale claims, orphan detection, sandbox visibility, and health/lease semantics.

Out of scope:

- Changing scenario manifest port bands except where tests expose a direct conflict with the registry model.
- Removing the canonical port allocation policy in `docs/reference/port-allocation.md`.
- Replacing all process diagnostics. OS process inspection should remain available for cleanup and operator visibility.
- Scenario-specific environment-variable escape hatches for discovery.
- Direct scenario execution paths that bypass lifecycle management.

## Current Technical Context

Current authority paths:

- `internal/process/process.go`
  - Defines `Record`.
  - Writes process JSON and PID files.
  - `LiveRecords` returns only records whose PID is live.
  - `SummarizeScenario` and `DiscoverRunningScenarios` derive running state from live records.
  - `ReadEnvironmentPorts` reads process environments for port variables.
- `internal/process/process_linux.go`
  - Uses `signal(0)`, `/proc/<pid>/stat`, and `/proc/<pid>/environ`.
- `internal/process/process_other.go`
  - Reports every process as not live and returns no process environment.
- `internal/ports/ports.go`
  - Uses `.port_<port>.lock` files and guard files.
  - `CleanStaleLocks` prunes locks based on PID liveness.
  - `BuildEnvironment` and port ownership paths call `process.LiveRecords`.
- `internal/lifecycle/lifecycle.go`
  - Starts scenarios by checking existing live records, preparing ports, launching tracked processes, waiting for health, and confirming fixed port locks.
  - Stop/cleanup kills process groups, removes process records, kills fixed-port orphans, verifies release, and removes locks.
- `internal/lifecycle/phases.go`
  - `startTrackedProcess` sets lifecycle environment variables and writes process records.
- `internal/scenario/runtime_state.go`
  - `DescribeRuntime` sets status `running` only when live record count is greater than zero.
  - `RuntimePortBindings` derives port bindings from records and process environments.
- `internal/orchestrator/detail.go`
  - Inventory, lookup, and port resolution depend on process summaries and runtime state.
- `packages/api-core/discovery/resolve.go`
  - Resolves scenario ports by shelling out to `vrooli scenario port`.
- `scenarios/browser-automation-studio/api/internal/scenarioport/scenarioport.go`
  - Has a `SCENARIO_REGISTRY` escape hatch before falling back to API-core discovery. This must be removed as a runtime discovery path.
- `internal/maintenance/maintenance.go`
  - Lists locks, prunes stale locks, lists/kills orphans, cleans stale records, and diagnoses ports.
- `internal/maintenance/process_snapshot.go`
  - Builds process health from tracked process records plus host process table heuristics.
- `scenarios/vrooli-autoheal/api/internal/checks/vrooli_state.go`
  - Reads legacy tracked process JSON and lock files directly.
- `scenarios/vrooli-autoheal/api/internal/healing/strategies/vrooli.go`
  - Runs `vrooli cleanup locks` and `vrooli cleanup orphans` during clean restart flows.

Health context:

- `.vrooli/schemas/service.schema.json` standardizes `/health` for API and UI lifecycle orchestration.
- `internal/scenario/scenario.go` defines `HealthConfig`, `HealthCheck`, `EvaluateHealth`, and `PerformHealthCheck`.
- `internal/lifecycle/health.go` waits for manifest-declared health checks, but strict health still begins by requiring `process.LiveRecords(records)` to be non-empty.
- `packages/api-core/health/health.go` defines the scenario health response schema with `status`, `service`, `timestamp`, `readiness`, dependencies, metrics, and structured errors.

## Target End State

The architecture should make the important boundaries obvious from package names and contracts:

- `internal/scenarioruntime`
  - Owns scenario runtime instance records, lifecycle state, leases/heartbeats, health snapshots, port bindings, and query APIs.
  - This is the source of truth for scenario status and port discovery.
- `internal/portalloc` or a narrowed `internal/ports`
  - Owns allocation policy, port claim acquisition, port claim release, socket bind probing, and port conflict reporting.
  - It does not decide whether a scenario is running.
- `internal/process`
  - Provides optional OS process diagnostics and cleanup support only.
  - It is not authoritative for scenario running state.
- `internal/lifecycle`
  - Coordinates start/stop/restart and writes lifecycle transitions to the registry.
  - It does not expose raw process records as the public runtime model.
- `internal/orchestrator`
  - Reads scenario inventory, status, and port data through a registry interface.
- `internal/maintenance`
  - Provides registry-aware diagnostics and healing: expired leases, stale claims, orphan listeners, dead process refs, and legacy files during migration.
- `packages/api-core/discovery`
  - Continues to resolve via the CLI/API discovery contract; callers do not use bespoke environment registries.

The core semantic split:

- **Runtime lease**: whether Vrooli's control plane still owns a scenario instance record.
- **Runner/process diagnostics**: optional host-level evidence about child processes.
- **Port claim**: whether a scenario instance owns a declared port.
- **Socket reachability**: whether a TCP listener exists and can be probed.
- **Health readiness**: whether the scenario's standardized health checks report healthy, degraded, or unhealthy.

No single field should conflate these.

## Heartbeat and Health Recommendation

Use both heartbeats and standardized health endpoints. They answer different questions.

Recommended contract:

- Heartbeat is a control-plane lease renewal written to the registry by lifecycle management or a future supervisor.
- Health is a data-plane readiness probe against the scenario's declared health checks and standardized `/health` schema.
- A scenario can have a fresh lease but be unhealthy while starting or degraded due to a dependency.
- A scenario can have a stale lease while a socket listener still exists; that is an orphan/stale-state diagnostic, not a valid running scenario.
- A health endpoint can be temporarily unreachable because of network profile, bind host, startup timing, or probe failure; that should not by itself delete the instance record.

The registry should store both:

- `last_heartbeat_at` and `heartbeat_deadline_at` for lease freshness.
- `health_status`, `readiness`, `last_health_probe_at`, `health_latency_ms`, and structured health metadata for readiness.

Updated recommendation:

- Use registry lease freshness as the authoritative liveness signal for Vrooli-managed runtime records.
- Use standardized health endpoints to populate health snapshots and to gate readiness-sensitive flows.
- Keep socket listener checks and OS process inspection as diagnostics, cleanup inputs, and conflict confirmation.

This is more robust than using health alone because some lifecycle phases are valid before health exists, and because the sandbox networking question must be diagnosable separately from registry discovery.

## Data Model Draft

Storage recommendation: SQLite at `$HOME/.vrooli/state/runtime.db`.

Use one writer transaction per state transition. Prefer small repository methods with injected clock and storage seams for unit tests.

Suggested tables:

### `runtime_instances`

- `instance_id` text primary key
- `scenario` text not null
- `generation` integer not null
- `scope_path` text
- `sandbox_id` text nullable
- `status` text not null: `starting`, `running`, `stopping`, `stopped`, `failed`, `expired`
- `phase` text
- `started_at` datetime
- `updated_at` datetime not null
- `last_heartbeat_at` datetime
- `heartbeat_deadline_at` datetime
- `stopped_at` datetime
- `stop_reason` text
- `owner_kind` text: `lifecycle`, `supervisor`, `manual`
- `owner_pid` integer nullable for diagnostics only
- `working_dir` text
- `schema_version` integer not null

### `runtime_port_claims`

- `claim_id` text primary key
- `instance_id` text not null
- `scenario` text not null
- `port_name` text not null
- `env_var` text not null
- `port` integer not null
- `bind_host` text not null default `127.0.0.1`
- `url` text nullable
- `status` text not null: `reserved`, `bound`, `released`, `expired`
- `created_at` datetime not null
- `updated_at` datetime not null
- `expires_at` datetime nullable
- `last_bound_at` datetime nullable
- unique active claim constraint for `(port, bind_host)` where status is `reserved` or `bound`

### `runtime_health_snapshots`

- `instance_id` text primary key
- `scenario` text not null
- `status` text not null: `healthy`, `degraded`, `unhealthy`, `unknown`, `not_configured`
- `readiness` boolean
- `checked_at` datetime
- `latency_ms` integer nullable
- `error` text nullable
- `response_json` text nullable, bounded in size
- `schema_valid` boolean nullable

### `runtime_process_refs`

Diagnostic only. This table must not be required to answer "is scenario running?".

- `ref_id` text primary key
- `instance_id` text not null
- `pid` integer nullable
- `pgid` integer nullable
- `process_id` text
- `step` text
- `command` text
- `log_file` text
- `status` text
- `started_at` datetime
- `ended_at` datetime nullable

### `runtime_events`

Useful for diagnosing races and future autoheal explanations.

- `event_id` text primary key
- `instance_id` text nullable
- `scenario` text
- `event_type` text not null
- `created_at` datetime not null
- `details_json` text

## Contract Decisions

Accepted decisions:

1. Use SQLite as the registry store.
2. Use dual-write and fallback during migration.
3. Prefer registry claims first and socket bind/listener probes second for port conflict authority.
4. Keep orphan and stale-state detection, but move it to diagnostics and remediation instead of primary discovery.
5. Remove browser automation studio's `SCENARIO_REGISTRY` runtime escape hatch and require API-core discovery.
6. Keep a dedicated final cleanup phase for legacy process records and lock files after validation.

CLI/API behavior:

- `vrooli scenario list`: read active registry instances first; during migration, fall back to legacy live records.
- `vrooli scenario status <name>`: report runtime lease status, health status, port claims, and diagnostics separately.
- `vrooli scenario port <name> <port-key>`: resolve from active registry port claims; during migration, fall back to legacy runtime bindings.
- `vrooli locks`: during migration, show both registry port claims and legacy lock files. Final state should report registry claims/leases rather than plain lock files.
- `vrooli cleanup locks`: during migration, clean expired registry claims and stale legacy lock files. Final state should clean expired claims.
- `vrooli orphans`: continue host process-table diagnostics. Add registry context so orphan means "Vrooli-looking process not referenced by active registry process refs or legacy records during migration."
- `vrooli cleanup orphans`: continue guarded kill behavior, with registry-aware revalidation immediately before signaling.
- `vrooli diagnose-port <port>`: report registry claim, socket listener inspection, health snapshot, process diagnostics, legacy lock file if present, port policy, and recommendations.

Forbidden contracts:

- Do not add per-scenario env var registries for discovery.
- Do not make PID visibility mandatory for scenario status or port resolution.
- Do not make health endpoint success the only proof of running state.
- Do not let process inspection decide port ownership before registry claims.
- Do not directly execute scenario binaries or dev scripts in tests or docs.

## Phase 0 Completion Record

Completed on 2026-05-08. This phase made no production code or runtime behavior changes.

Confirmed current command contracts:

- `vrooli scenario list [options]`
  - Options: `--json`, `--include-ports`, `--help`.
- `vrooli scenario status [scenario-name] [options]`
  - Options: `--json`, `--help`.
- `vrooli scenario port <scenario-name> [port-name] [options]`
  - Options: `--json`, `--help`.
- `vrooli locks [action] [options]`
  - Options: `--json`, `--help`.
  - Current help text still describes stale port lock files; future phases must broaden this to registry claims plus legacy lock files during migration.
- `vrooli cleanup orphans [--dry-run]`
  - Current behavior: kill orphaned Vrooli processes.
- `vrooli cleanup locks`
  - Current behavior: remove stale lock files.
- `vrooli orphans [action] [options]`
  - Options: `--json`, `--dry-run` for `kill`, `--help`.
- `vrooli diagnose-port <port> [scenario] [options]`
  - Options: `--json`, `--help`.

Confirmed current test inventory:

- Core runtime/port/lifecycle/maintenance areas:
  - `internal/process`: `process_test.go`, `process_linux_test.go`.
  - `internal/ports`: `ports_test.go`, `lock_confirm_test.go`.
  - `internal/lifecycle`: `lifecycle_test.go`, `lifecycle_smoke_test.go`, `stepsink_test.go`, `progressf_test.go`.
  - `internal/orchestrator`: `orchestrator_test.go`, `sandbox_test.go`, `logger_test.go`.
  - `internal/maintenance`: `maintenance_test.go`.
  - CLI/API surfaces adjacent to this work: `internal/cli/scenariocli`, `internal/cli/projectcli`, `internal/app/project`, `internal/api`.
- API-core is a separate Go module under `packages/api-core`; run tests from that directory.
  - Relevant packages include `discovery`, `health`, `scenario`, `scenariocli`, `server`, `storage`, and shared utility packages.
- Browser automation studio has focused coverage in `scenarios/browser-automation-studio/api/internal/scenarioport/scenarioport_test.go`.
  - Current tests intentionally cover `SCENARIO_REGISTRY`; Phase 7 must replace these with seam-based discovery tests.
- Vrooli autoheal has coverage under `scenarios/vrooli-autoheal/api/internal/checks/...` and `scenarios/vrooli-autoheal/api/internal/healing/...`.
  - `checks/vrooli_state.go` still reads legacy process records and port lock files directly.
  - `healing/strategies/vrooli.go` delegates cleanup through `vrooli cleanup locks` and `vrooli cleanup orphans`, which is the correct long-term direction.

Baseline validation results:

```bash
vrooli help
# passed

go test ./internal/process ./internal/ports ./internal/lifecycle ./internal/orchestrator ./internal/maintenance
# passed

go test ./internal/api
# passed

go test ./internal/cli/scenariocli ./internal/cli/projectcli ./internal/app/project
# passed

go test ./packages/api-core/...
# failed from repo root because packages/api-core is a separate Go module:
# pattern ./packages/api-core/...: main module (github.com/vrooli/vrooli) does not contain package ...

cd packages/api-core && go test ./...
# passed

cd scenarios/browser-automation-studio/api && go test ./internal/scenarioport
# passed

cd scenarios/vrooli-autoheal/api && go test ./internal/checks/... ./internal/healing/...
# failed in internal/checks/infra display-manager tests only:
# - TestDisplayManagerCheckRunGDMActive
# - TestDisplayManagerCheckRunWithX11
# - TestDisplayManagerCheckExecuteActionRestart
# - TestDisplayManagerCheckGnomeRDPHealthy
# - TestDisplayManagerCheckRDPPortNotListening
# Adjacent packages passed, including internal/checks, internal/checks/host,
# internal/checks/system, internal/checks/vrooli, internal/healing,
# internal/healing/healers, and internal/healing/strategies.
```

Phase 0 decision freeze:

- Keep the migration flag name as `VROOLI_RUNTIME_REGISTRY` with planned values `off`, `dual`, `prefer`, and `strict`.
- Treat `off` as legacy-only.
- Treat `dual` as legacy-read-authoritative with registry writes and reconciliation diagnostics.
- Treat `prefer` as registry-read-first with legacy fallback.
- Treat `strict` as registry-only for production discovery and port ownership; legacy data remains diagnostic until Phase 9 removes old authority.
- Do not restart scenarios for Phases 1 or 2. They should be package-level implementation and unit-test phases only.
- Do not stop or restart `web-console` during this overhaul unless the user explicitly approves it in a later phase.

## Phase 1 Completion Record

Completed on 2026-05-08. This phase introduced the registry package only; it did not wire registry reads or writes into lifecycle, CLI, port allocation, orchestrator, maintenance, autoheal, browser automation studio, or running scenarios.

Files added or changed:

- `internal/scenarioruntime/types.go`
  - Defines registry domain types: `Instance`, `PortClaim`, `HealthSnapshot`, `ProcessRef`, and `Event`.
  - Defines lifecycle, port claim, health, and cleanup repository seams.
  - Defines explicit status constants and sentinel errors for not found, stale generation, and active claim conflict.
- `internal/scenarioruntime/schema.go`
  - Defines SQLite schema version `1`.
  - Creates `runtime_instances`, `runtime_port_claims`, `runtime_health_snapshots`, `runtime_process_refs`, and `runtime_events`.
  - Adds a partial unique index for active `(port, bind_host)` claims where status is `reserved` or `bound`.
- `internal/scenarioruntime/sqlite.go`
  - Opens `$HOME/.vrooli/state/runtime.db` by default, or an explicit test path.
  - Uses injected `Clock`.
  - Applies portable SQLite pragmas: foreign keys, WAL, busy timeout, synchronous normal, temp store memory.
  - Implements create/update/query instance operations and atomic acquire/release/expire port claim operations.
  - Implements basic health snapshot upsert/query.
- `internal/scenarioruntime/scan.go`
  - Centralizes SQL row scanning and nullable time/bool/int conversion.
- `internal/scenarioruntime/sqlite_test.go`
  - Covers instance create/update/query, generation stale-writer protection, active claim uniqueness, expired-claim query behavior, and explicit temp-path use.
- `go.mod` / `go.sum`
  - Adds `modernc.org/sqlite v1.43.0` as the root module's pure-Go SQLite driver dependency.
  - Keeps the root module `go` directive at `1.24.0`. Newer `modernc.org/sqlite` versions were intentionally avoided because they require Go 1.25.

Validation:

```bash
go test ./internal/scenarioruntime -count=1
# passed

go test ./internal/process ./internal/ports ./internal/lifecycle ./internal/orchestrator ./internal/maintenance
# passed
```

Phase 1 handoff notes, resolved by Phase 2:

- Phase 2 added explicit lease/heartbeat methods and health-probe behavior.
- The current store intentionally still does not conflate runtime liveness and readiness. That separation must remain intact.
- `ProcessRef` and `Event` tables remain present for upcoming lifecycle diagnostics and audit history; Phase 3 can add write methods as it wires lifecycle dual-write.
- No scenario was stopped or restarted during Phase 1 or Phase 2.

## Phase 2 Completion Record

Completed on 2026-05-08. This phase remained additive inside `internal/scenarioruntime`; it did not wire registry reads or writes into lifecycle, CLI, port allocation, orchestrator, maintenance, autoheal, browser automation studio, or running scenarios.

Files added or changed:

- `internal/scenarioruntime/types.go`
  - Adds repository contract methods for `CreateLease`, `HeartbeatLease`, `ExpireStaleLeases`, and `StopLease`.
  - Adds centralized defaults for heartbeat TTL and bounded health response storage.
- `internal/scenarioruntime/lease.go`
  - Adds explicit lease lifecycle methods.
  - `CreateLease` initializes `last_heartbeat_at` and `heartbeat_deadline_at`.
  - `HeartbeatLease` refreshes active `starting`/`running` leases without changing health or readiness.
  - `ExpireStaleLeases` marks stale `starting`/`running` leases as `expired` without inspecting PIDs or socket listeners.
  - `StopLease` records `stopped_at` and `stop_reason`.
- `internal/scenarioruntime/healthprobe.go`
  - Adds a small health probing adapter that reuses `internal/scenario` health check configuration.
  - Recognizes the standardized `github.com/vrooli/api-core/health.Response` shape for HTTP health responses.
  - Stores bounded response JSON, schema-valid metadata, readiness, latency, and bounded structured diagnostics.
  - Keeps health results separate from runtime lease state; unhealthy health does not delete or stop an instance.
- `internal/scenarioruntime/sqlite.go`
  - Changes persisted registry timestamps to fixed-width UTC nanosecond strings so SQLite text comparisons are stable for heartbeat and claim deadlines.
- `internal/scenarioruntime/lease_test.go`
  - Covers lease create/heartbeat/stop, fresh lease active with unknown health, stale lease expiry with a bound port claim preserved as diagnostic evidence, and stale generation rejection.
- `internal/scenarioruntime/healthprobe_test.go`
  - Covers no configured checks, standardized health response recognition, schema-invalid health response diagnostics, and unhealthy snapshot persistence without mutating the runtime instance.

Validation:

```bash
go test ./internal/scenarioruntime -count=1
# passed

go test ./internal/process ./internal/ports ./internal/lifecycle ./internal/orchestrator ./internal/maintenance -count=1
# passed

go test ./internal/api -count=1
# passed

go test ./internal/cli/scenariocli ./internal/cli/projectcli ./internal/app/project -count=1
# passed
```

Important findings:

- Phase 2 tests exposed that `time.RFC3339Nano` produces variable-width timestamp strings, which are unsafe for direct SQLite text ordering when subsecond values differ. The registry now writes fixed-width UTC nanosecond timestamps. Existing parsers still accept the value through `time.RFC3339Nano`.
- Expiring a lease intentionally does not expire or release bound port claims yet. That distinction is important for later maintenance and diagnose-port work: a stale lease plus bound claim/listener is diagnostic evidence for Phase 6, not proof that normal discovery should still treat the scenario as active.

Notes for Phase 3:

- Lifecycle dual-write should call `CreateLease` at start planning time and `HeartbeatLease` while lifecycle owns startup supervision.
- If there is no long-lived supervisor yet, Phase 3 must make the renewal owner explicit and conservative instead of pretending there is a permanent heartbeat source.
- Lifecycle should upsert health snapshots from the new probe adapter after manifest health checks run, but it must not let health failures overwrite lease lifecycle state.
- Production behavior must remain legacy-authoritative in Phase 3; this registry data is for reconciliation and future read-side migration only.

## Phase 3 Completion Record

Completed on 2026-05-08. This phase wires lifecycle into the runtime registry only when `VROOLI_RUNTIME_REGISTRY` is set to `dual`, `prefer`, or `strict`. The default empty/off mode remains legacy-only. No read side was switched, and no running scenario was stopped or restarted during implementation.

Files added or changed:

- `internal/lifecycle/runtime_registry.go`
  - Adds the lifecycle-side registry adapter and migration-mode decision boundary.
  - Uses `VROOLI_RUNTIME_REGISTRY=off|dual|prefer|strict`; empty is treated as `off`.
  - Creates runtime leases for fresh starts, records conservative heartbeats during supervised setup/develop phases, reserves claims from legacy-allocated ports, binds claims after health/listener confirmation, records process refs for background lifecycle steps, records health snapshots, marks successful starts `running`, marks failed starts `failed`, and releases active claims on failure.
  - Stop cleanup marks active leases `stopping`, then releases claims and records `stopped` after legacy process/lock cleanup completes.
  - Rollback cleanup after a failed start intentionally remains registry-stop-disabled so failed starts stay `failed` for diagnosis instead of being overwritten as `stopped`.
- `internal/lifecycle/lifecycle.go`
  - Adds the registry dependency seam to `lifecycleDeps`.
  - Wires opt-in registry start/stop transitions around the existing legacy lifecycle path.
  - Leaves legacy process records and lock files authoritative in this phase.
- `internal/lifecycle/phases.go`
  - Writes diagnostic registry process refs for background lifecycle steps after the legacy process record and lock are written.
  - If opt-in registry process-ref writing fails, the just-started child is killed and the legacy record/lock are rolled back so dual-write failures are not hidden.
- `internal/scenarioruntime/types.go`
  - Extends repository contracts with `BindPortClaim`, `ReleaseActivePortClaimsForInstance`, process-ref APIs, and event recording.
- `internal/scenarioruntime/sqlite.go`
  - Implements bound-claim transitions, active claim release by instance, process-ref write/update/list methods, and event recording.
- `internal/scenarioruntime/scan.go`
  - Adds process-ref scanning.
- `internal/scenarioruntime/sqlite_test.go`
  - Adds focused coverage for bind/release claim transitions and process-ref round trips.
- `internal/lifecycle/lifecycle_test.go`
  - Adds opt-in dual-write integration coverage for successful start, stop, and health-failure rollback.

Validation:

```bash
go test ./internal/scenarioruntime -count=1
# passed

go test ./internal/lifecycle -count=1
# passed

go test ./internal/process ./internal/ports ./internal/lifecycle ./internal/orchestrator ./internal/maintenance ./internal/api ./internal/cli/scenariocli ./internal/cli/projectcli ./internal/app/project -count=1
# passed
```

Important findings:

- Startup rollback needed a registry-aware distinction between failed-start compensation and operator stop. The first implementation correctly cleaned legacy state but then the rollback stop path overwrote the failed registry lease as `stopped`. Phase 3 now keeps failed starts as `failed` and releases their claims, while normal `vrooli scenario stop` still records `stopped`.
- Existing lifecycle health accepts plain HTTP 200 responses, while the new standardized health probe can identify non-schema `/health` responses. Phase 3 stores schema diagnostics where available but preserves lifecycle's accepted health result for compatibility; later phases can decide whether stricter standardized health should become operator-facing.
- The current heartbeat owner is explicitly the lifecycle process during supervised startup only. There is still no long-lived supervisor heartbeat after start; Phase 4 reads must account for this, and a future supervisor remains the correct long-term owner for continuous renewal.

Notes for Phase 4:

- Do not switch discovery reads globally yet. Add registry-read behavior behind `VROOLI_RUNTIME_REGISTRY=prefer|strict`, with legacy fallback in `prefer`.
- Because Phase 3 defaults to `off`, validation of Phase 4 should include an explicit lifecycle start under `VROOLI_RUNTIME_REGISTRY=dual` before expecting registry-backed `scenario port` reads to exist.
- Registry health snapshots may contain `schema_valid=false` while status remains `healthy` when legacy lifecycle accepted a plain HTTP 200 health response. Operator output should expose that as a health schema diagnostic, not as proof the scenario failed to start.
- Keep web-console running. If live validation is needed, use a selected non-console scenario such as `workspace-sandbox`.

## Phase 4 Completion Record

Completed on 2026-05-08. This phase switches read-side scenario discovery to the registry only when `VROOLI_RUNTIME_REGISTRY` is set to `prefer` or `strict`. The default empty/off mode and `dual` mode remain legacy-read-authoritative. No scenarios were stopped or restarted during implementation.

Files added or changed:

- `internal/scenarioruntime/mode.go`
  - Centralizes `VROOLI_RUNTIME_REGISTRY=off|dual|prefer|strict` parsing and read/write policy helpers.
  - `dual`, `prefer`, and `strict` still enable registry writes.
  - Only `prefer` and `strict` enable registry reads.
- `internal/lifecycle/runtime_registry.go`
  - Reuses the centralized migration-mode decision boundary while keeping existing lifecycle test aliases intact.
- `internal/orchestrator/runtime_registry_read.go`
  - Adds the orchestrator registry read adapter.
  - Builds `Detail` values from active registry instances, active port claims, health snapshots, and diagnostic process refs.
  - Treats process refs as diagnostics only; runtime status comes from the registry instance status.
- `internal/orchestrator/detail.go`
  - Reads registry details for `InventoryReport`, `Lookup`, start/restart follow-up detail, and `ResolvePort` when read mode is enabled.
  - `prefer` mode uses registry data first and falls back to legacy process records when registry data is missing.
  - `strict` mode does not use legacy process records for runtime status or port resolution.
  - `ResolvePort` now checks `detail.Details.Status == "running"` instead of requiring a positive PID-derived process count.
- `internal/orchestrator/orchestrator.go`
  - Adds the registry query-store seam.
  - `Running()` now follows runtime status instead of PID-derived process count.
- `internal/app/scenario/service.go`
  - Scenario `info`, `status`, and `port` output now use `Detail.Details` so registry-backed details are not accidentally recomputed from legacy process records.
  - Port-list errors now refer to runtime ports rather than process records.
- `internal/app/scenario/types.go`
  - Adds `BuildRuntimeDataFromDetail` / `BuildRuntimeDataFromDetails` so callers can preserve registry-built runtime details.
- `internal/orchestrator/orchestrator_test.go`
  - Adds coverage for `prefer` registry reads without process records.
  - Adds coverage that `strict` ignores legacy process records.
  - Adds coverage that `prefer` falls back to legacy records when registry data is missing.

Validation:

```bash
vrooli help
# passed

go test ./internal/scenarioruntime ./internal/orchestrator ./internal/app/scenario -count=1
# passed

go test ./internal/lifecycle -count=1
# passed

go test ./internal/process ./internal/ports ./internal/lifecycle ./internal/orchestrator ./internal/maintenance ./internal/api ./internal/cli/scenariocli ./internal/cli/projectcli ./internal/app/project ./internal/app/scenario -count=1
# passed

cd packages/api-core && go test ./... -count=1
# passed
```

Important findings:

- `scenario port` and application-level port listing had a hidden legacy dependency: they checked `ProcessCount` or rebuilt ports from `Runtime.Records`. Phase 4 moved these read paths to `Detail.Details`, which is the correct seam for both legacy and registry-backed runtime views.
- Because there is no long-lived supervisor heartbeat after start yet, Phase 4 deliberately does not expire leases during normal read calls. Active registry reads currently mean latest `starting` or `running` registry instance. Lease expiry and cleanup remain Phase 6 maintenance responsibilities until a continuous heartbeat owner exists.
- Registry-backed read behavior is still opt-in. `dual` is intentionally write-only/read-legacy so teams can collect registry data without changing discovery behavior.

Notes for Phase 5:

- Port ownership is still legacy-lock authoritative unless registry reads are enabled and registry claims already exist from lifecycle dual-write.
- Phase 5 should move allocation authority into a registry-backed port claim service and keep legacy lock files as migration compatibility artifacts only.
- Add concurrent allocator tests before enabling registry-backed allocation beyond controlled migration modes.
- Do not restart web-console. Live validation should still use a selected non-console scenario such as `workspace-sandbox`.

## Phase 0-4 Hardening Completion Record

Completed on 2026-05-08. This pass deliberately did not start Phase 5. It hardened the already-completed registry package, lifecycle dual-write, and registry-backed read-side work while keeping existing runtime behavior additive and migration-controlled. No scenarios were stopped or restarted.

Files changed:

- `internal/scenario/runtime_state.go`
  - Added `RuntimePortResolution` and `ResolveRuntimePort` so runtime port request normalization is owned by the scenario runtime model rather than duplicated in orchestrator and app-layer code.
  - Kept UI-to-API fallback out of the shared helper because that fallback is an orchestrator/operator policy, not a core runtime binding rule.
- `internal/orchestrator/detail.go`
  - Replaced its local request-normalization helper with `scenario.ResolveRuntimePort`.
  - Preserved existing behavior: `ResolvePort` still requires runtime status `running`, still falls back from `UI_PORT` to `API_PORT`, and still falls back to the only available port when exactly one runtime port exists.
- `internal/app/scenario/service.go`
  - Replaced its local request-normalization helper with `scenario.ResolveRuntimePort`.
  - Fixed a responsibility leak where `scenario port <name> <port>` could return a specific runtime port from `Detail.Details.Ports` even when `Detail.Details.Status` was not `running`. List-all-port behavior already had this guard; specific-port behavior now matches it.
- `internal/scenario/runtime_state_test.go`
  - Added focused tests for runtime port request normalization and missing-port rejection.
- `internal/app/scenario/service_test.go`
  - Added tests that specific-port CLI app behavior rejects non-running runtime states and resolves running runtime details through the shared runtime helper.
- `internal/scenarioruntime/mode_test.go`
  - Added explicit coverage for `VROOLI_RUNTIME_REGISTRY` normalization and read/write/strict classification.

Validation:

```bash
go test ./internal/scenario ./internal/app/scenario ./internal/orchestrator ./internal/scenarioruntime -count=1
# passed

go test ./internal/process ./internal/ports ./internal/lifecycle ./internal/orchestrator ./internal/maintenance ./internal/api ./internal/cli/scenariocli ./internal/cli/projectcli ./internal/app/project ./internal/app/scenario ./internal/scenario ./internal/scenarioruntime -count=1
# passed

cd packages/api-core && go test ./... -count=1
# passed
```

Hardening findings:

- Runtime port lookup had drifted into two slightly different helpers. The shared helper now lives beside `RuntimeDetails`, `RuntimePortBinding`, and runtime endpoint construction, which is the correct boundary for this rule.
- The app layer should present and format scenario port results, not decide differently from orchestrator whether a non-running runtime may expose a specific port. That inconsistency is now covered by tests.
- Migration-mode parsing was central but previously lacked direct unit tests. The new tests protect the critical `off`/`dual`/`prefer`/`strict` decision boundary before Phase 5 adds more write-side responsibility.

Notes for Phase 5:

- Preserve the Phase 4H invariant: public port resolution must not expose reserved or starting-only claims as usable runtime ports.
- When registry-backed allocation begins, add tests for reserved claims appearing in status/detail output while `scenario port` remains gated on `running`.
- Continue dual-write/fallback behavior. Do not restart `web-console`; live validation should use a selected non-console scenario only after unit/integration coverage is strong.

## Phase 5 Completion Record

Completed on 2026-05-08. This phase moves port allocation authority to registry claims only when lifecycle registry writes are enabled through `VROOLI_RUNTIME_REGISTRY=dual|prefer|strict`. Default empty/off mode remains legacy-only. Legacy lock files are still written during migration for compatibility, but in enabled modes the allocator now checks/acquires a registry claim before it writes the legacy lock. No scenarios were stopped or restarted.

Files changed:

- `internal/ports/ports.go`
  - Added `RuntimeClaimOptions` and `RuntimeClaimStore` as the allocator seam.
  - Added `BuildEnvironmentWithRuntimeClaims`, preserving existing `BuildEnvironment` as legacy-compatible default behavior.
  - Added `Environment.RuntimeClaims` so lifecycle can adopt the claims acquired during allocation instead of reserving them later.
  - Acquires registry claims before legacy lock compatibility writes when runtime claims are enabled.
  - Uses reserved -> bound -> released/expired semantics:
    - new allocations create `reserved` claims with a TTL;
    - failed legacy/socket checks release newly-created claims;
    - abandoned expired `reserved` claims are expired before a replacement allocation;
    - `bound` claims are not expired by the allocator cleanup path.
  - Keeps legacy lock files as migration compatibility artifacts and conflict evidence only after registry claim acquisition succeeds.
- `internal/lifecycle/lifecycle.go`
  - Passes the current runtime registry session into scenario environment preparation.
  - Uses `BuildEnvironmentWithRuntimeClaims` so lifecycle start allocation claims ports inside the allocator.
- `internal/lifecycle/runtime_registry.go`
  - Exposes the lifecycle runtime session as `ports.RuntimeClaimOptions`.
  - Replaces post-allocation reservation-only behavior with adoption of allocator-created claims plus fallback reservation for any missing claim.
  - Extends the lifecycle registry store seam with cleanup operations needed for claim expiry.
- `internal/lifecycle/phases.go`
  - Keeps ad-hoc phase execution on the disabled runtime-registry path so phase commands do not accidentally claim registry ports.
- `internal/scenarioruntime/sqlite.go`
  - `BindPortClaim` now clears `expires_at` so a successfully bound claim is not later treated as an abandoned startup reservation.
- `internal/ports/ports_test.go`
  - Added coverage that concurrent allocators cannot both claim the same fixed port.
  - Added coverage that expired reserved claims are expired and replaced without changing the abandoned instance status.
  - Added coverage that bound claims survive expired-reservation cleanup and clear their expiry.
  - Added coverage that socket conflicts release newly-created registry claims.
  - Added coverage that a stale legacy lock cannot override an active registry claim.

Validation:

```bash
go test ./internal/ports -count=1
# passed

go test ./internal/scenarioruntime -count=1
# passed

go test ./internal/lifecycle -count=1
# passed

go test ./internal/process ./internal/ports ./internal/lifecycle ./internal/orchestrator ./internal/maintenance ./internal/api ./internal/cli/scenariocli ./internal/cli/projectcli ./internal/app/project ./internal/app/scenario ./internal/scenario ./internal/scenarioruntime -count=1
# passed

cd packages/api-core && go test ./... -count=1
# passed
```

Important findings:

- Phase 3 had useful registry claim writes, but they happened after legacy allocation had already selected and lock-claimed a port. That did not make the registry the first ownership gate. Phase 5 moves claim acquisition into `internal/ports` so the SQLite active-claim uniqueness constraint is now the first authority in enabled migration modes.
- Successful bind needed to clear the reserved-claim deadline. Without that, an old `bound` claim with a past `expires_at` would keep appearing in expired-active queries. The allocator filters expiry to `reserved` claims, and `BindPortClaim` now clears the deadline for newly bound claims.
- Legacy lock files still matter during the migration window as compatibility artifacts and diagnostics. They are deliberately not removed from the normal write path yet.

Notes for Phase 6:

- `internal/maintenance` still reports and cleans legacy lock files first. Phase 6 should make `diagnose-port`, `locks`, and `cleanup locks` registry-aware, using active/expired registry claims as first-class evidence.
- Autoheal still has direct legacy readers under `scenarios/vrooli-autoheal/api/internal/checks/vrooli_state.go`. Phase 6 should move that surface to a CLI/API-backed seam or a registry-aware reader while keeping cleanup delegated to `vrooli cleanup locks` and `vrooli cleanup orphans`.
- `scenario port` remains gated by running runtime status from Phase 4H, so newly reserved claims from a starting instance are not exposed as usable ports until lifecycle binds claims and marks the instance running.
- Live validation is still deferred. The next phase should remain unit/integration-test focused first; if a live scenario is needed later, use a selected non-console scenario such as `workspace-sandbox` and do not stop `web-console`.

## Phase 0-5 Hardening Completion Record

Completed on 2026-05-08. This pass deliberately did not start Phase 6. It hardened the completed registry package, registry-backed read side, lifecycle dual-write, and registry-backed allocation path while keeping the migration additive and opt-in. No scenarios were stopped or restarted.

Files changed:

- `internal/scenarioruntime/types.go`
  - Added `LocalPortURL` as the single runtime-claim URL construction rule for local loopback URLs stored in registry claims.
  - This removes duplicated URL policy from lifecycle and port allocation code and keeps claim URL behavior in the runtime registry domain.
- `internal/scenarioruntime/mode_test.go`
  - Added focused coverage that only known HTTP port kinds are advertised through `LocalPortURL`.
- `internal/lifecycle/runtime_registry.go`
  - Replaced its local runtime claim URL helper implementation with the centralized scenario runtime helper.
- `internal/ports/ports.go`
  - Replaced its local runtime claim URL helper implementation with the centralized scenario runtime helper.
  - Split expired active-claim discovery from reserved-claim expiry so the allocator boundary is explicit: it expires abandoned `reserved` claims only, while `bound` claims remain ownership evidence for later maintenance/diagnostics.
- `internal/orchestrator/runtime_registry_read.go`
  - Added explicit latest-runtime-instance selection by generation, then `updated_at`, then instance id.
  - Both inventory and single-detail registry reads now use the same selection decision instead of depending on repository ordering.
- `internal/orchestrator/orchestrator_test.go`
  - Added coverage that latest instance selection is stable even when the store returns instances out of order.

Validation:

```bash
go test ./internal/scenarioruntime ./internal/ports ./internal/lifecycle ./internal/orchestrator ./internal/app/scenario ./internal/scenario -count=1
# passed

go test ./internal/process ./internal/ports ./internal/lifecycle ./internal/orchestrator ./internal/maintenance ./internal/api ./internal/cli/scenariocli ./internal/cli/projectcli ./internal/app/project ./internal/app/scenario ./internal/scenario ./internal/scenarioruntime -count=1
# passed

cd packages/api-core && go test ./... -count=1
# passed
```

Hardening findings:

- Runtime claim URL construction had become duplicated between lifecycle and port allocation. That was a small but real boundary drift: the registry claim model owns claim metadata, so claim URL construction now has one home.
- Registry detail lookup relied on SQLite ordering indirectly. The store still returns ordered data, but the orchestrator seam no longer requires every future repository implementation or test fake to preserve that ordering for correctness.
- Allocator cleanup intentionally expires only abandoned startup reservations. Bound claims with stale-looking timestamps must survive as ownership/diagnostic evidence until maintenance logic decides how to report or remediate them in Phase 6.

Recommended next steps:

- Start Phase 6 only after reviewing `internal/maintenance` and autoheal call sites with the Phase 5H invariants in mind.
- In Phase 6, make registry claims first-class in `vrooli locks`, `vrooli cleanup locks`, and `vrooli diagnose-port`, but keep legacy lock files visible during the migration period.
- Do not restart `web-console`. Continue to prefer unit/integration validation before any selected live scenario validation.

## Phase 0-5 Hardening Round 2 Completion Record

Completed on 2026-05-08. This pass deliberately did not start Phase 6. It hardened the completed registry package, registry-backed read side, and registry-backed allocation path while keeping the migration additive and opt-in. No scenarios were stopped or restarted.

Files changed:

- `internal/ports/ports.go`
  - Tightened allocator responsibility for partial multi-port failures. If a registry-enabled allocation successfully creates one or more claims and then a later port allocation fails, the allocator now releases newly-created registry claims and abandons the corresponding legacy compatibility locks.
  - Keeps reused existing claims out of allocator-owned rollback, so repeated allocation attempts for the same runtime instance do not release ownership they did not create in the current call.
  - Preserves lifecycle-level failure compensation as a second safety net, but the `internal/ports` seam no longer depends on lifecycle cleanup to avoid leaking earlier claims from the same allocation call.
- `internal/ports/ports_test.go`
  - Added a regression test for a two-port scenario where the first fixed port is claimed successfully and the second fixed port fails socket probing. The test asserts both registry claims are released and the first legacy lock is abandoned.
- `internal/scenarioruntime/types.go`
  - Added `ActiveInstanceStatuses` and `IsActiveInstanceStatus` so the scenario runtime domain owns the active-runtime status policy.
- `internal/scenarioruntime/mode_test.go`
  - Added coverage that active runtime statuses are centralized, immutable to callers, and limited to `starting` and `running`.
- `internal/orchestrator/runtime_registry_read.go`
  - Replaced orchestrator-local active status constants with the scenario runtime domain helper.

Validation:

```bash
go test ./internal/ports -run 'TestBuildEnvironmentWithRuntimeClaims|TestRuntimeClaim|TestLegacyStaleLock' -count=1
# passed

go test ./internal/scenarioruntime ./internal/ports ./internal/orchestrator -count=1
# passed

go test ./internal/process ./internal/ports ./internal/lifecycle ./internal/orchestrator ./internal/maintenance ./internal/api ./internal/cli/scenariocli ./internal/cli/projectcli ./internal/app/project ./internal/app/scenario ./internal/scenario ./internal/scenarioruntime -count=1
# passed

cd packages/api-core && go test ./... -count=1
# passed
```

Hardening findings:

- The registry allocator already released the claim for the port that failed, and lifecycle failure compensation released active claims for failed starts. That was not enough ownership clarity: `internal/ports` can be called directly and should not leak the first claim in a multi-port allocation when a later claim fails.
- Legacy compatibility locks are still written during migration. When the allocator rolls back newly-created registry claims from the same call, it should also abandon the matching compatibility locks so Phase 6 diagnostics do not inherit avoidable stale lock evidence.
- Active runtime status policy was duplicated as an orchestrator-local slice. Moving it into `internal/scenarioruntime` gives Phase 6 maintenance/diagnostics code a single policy seam before it starts reasoning about active leases, expired leases, and stale claims.

Recommended next steps:

- Phase 6 should now build maintenance and autoheal behavior on `scenarioruntime.ActiveInstanceStatuses()` rather than re-declaring active lease states.
- In Phase 6, keep the same rollback discipline: cleanup commands may expire or release claims they own, but diagnostics must not silently delete bound ownership evidence.
- Continue avoiding live scenario restarts until registry-aware maintenance has strong unit coverage. If live validation becomes necessary, use a selected non-console scenario such as `workspace-sandbox`; do not stop `web-console`.

## Phase 6 Completion Record

Completed on 2026-05-08. This phase made maintenance, diagnostics, and autoheal registry-aware while preserving legacy lock/process behavior during the migration. No scenarios were stopped or restarted.

Files changed:

- `internal/scenarioruntime/types.go`
  - Added `ExpireStaleStartingLeases` to the cleanup repository seam.
  - This intentionally expires abandoned startup leases only. Running leases are preserved until a long-lived supervisor heartbeat exists, so cleanup does not break migration-mode discovery for scenarios that started before continuous renewal is available.
- `internal/scenarioruntime/sqlite.go`
  - Implements `ExpireStaleStartingLeases` transactionally.
- `internal/scenarioruntime/lease_test.go`
  - Adds coverage that stale `starting` leases expire while stale-looking `running` leases remain running during the supervisor migration window.
- `internal/maintenance/runtime_registry.go`
  - Adds the maintenance-side registry adapter.
  - Opens `$HOME/.vrooli/state/runtime.db` only when it already exists, so maintenance commands do not create an empty registry as a side effect.
  - Builds registry claim diagnostics with instance status, lease freshness, health status/readiness, and process ref evidence.
  - Provides registry process refs for host snapshot tracking.
- `internal/maintenance/maintenance.go`
  - Adds `RuntimeClaimInfo` and `RuntimeProcessRefInfo`.
  - Extends `PortDiagnostic` with `registry_claims` and `registry_processes`.
  - Adds `ListRuntimeClaims`.
  - Updates `CleanStaleLocks` to expire abandoned `reserved` registry claims and stale startup leases, then prune stale legacy lock files.
  - Adds registry-aware diagnose-port recommendations while keeping listener, legacy lock, port-policy, and orphan evidence.
- `internal/maintenance/process_snapshot.go`
  - Counts active registry process refs alongside legacy process records when classifying tracked processes and orphans.
  - Preserves guarded orphan detection and PID/session protections.
- `internal/app/project/service.go`
  - Includes registry claims in locks responses while preserving legacy lock lists.
- `internal/cli/projectcli/projectcli.go`
  - Renders `vrooli locks --json` with both `locks` and `registry_claims`.
  - Renders human `vrooli locks` output in separate "Registry claims" and "Legacy lock files" sections.
  - Renders registry claim/process evidence in human `vrooli diagnose-port`.
- `internal/cli/vroolicli/runtime.go`
  - Passes registry claims through the top-level locks command response.
- `scenarios/vrooli-autoheal/api/internal/checks/vrooli_state.go`
  - Adds a CLI-backed state reader that parses `vrooli locks --json` registry claims and legacy locks through the public maintenance CLI contract.
  - Keeps a filesystem fallback for older local installs or CLI failures.
  - Preserves direct legacy file parsing as fallback only, not the preferred production path.
- `scenarios/vrooli-autoheal/api/internal/checks/vrooli_state_test.go`
  - Covers registry claim parsing from the CLI contract, filesystem fallback, and delegation of registry cleanup to `vrooli cleanup locks`.

Validation:

```bash
go test ./internal/scenarioruntime ./internal/maintenance ./internal/app/project ./internal/cli/projectcli ./internal/cli/vroolicli -count=1
# passed

go test ./internal/process ./internal/ports ./internal/lifecycle ./internal/orchestrator ./internal/maintenance ./internal/api ./internal/cli/scenariocli ./internal/cli/projectcli ./internal/cli/vroolicli ./internal/app/project ./internal/app/scenario ./internal/scenario ./internal/scenarioruntime -count=1
# passed

cd packages/api-core && go test ./... -count=1
# passed

cd scenarios/vrooli-autoheal/api && go test ./internal/checks/... ./internal/healing/... -count=1
# passed
```

Important findings:

- The existing heartbeat owner is still startup-scoped, not a long-lived supervisor. Because of that, cleanup must not call the broad `ExpireStaleLeases` behavior on running leases yet. Phase 6 uses a narrower startup-only expiry path so abandoned starts can be cleaned without invalidating real running scenarios during migration.
- Registry diagnostics now make stale/orphan concepts explicit without promoting process inspection back into discovery authority. A stale lease, expired reservation, legacy lock, live listener, and orphan process are reported as separate facts.
- Autoheal should continue delegating cleanup to core `vrooli cleanup locks` and `vrooli cleanup orphans`; it now has a CLI-backed seam for reading registry claim evidence without importing root `internal` packages or reimplementing SQLite policy.

Recommended next steps:

- Proceed to Phase 7: remove browser automation studio's `SCENARIO_REGISTRY` escape hatch and force discovery through `packages/api-core/discovery`.
- Keep live scenario restarts deferred. Phase 8 should perform targeted sandbox validation later, preferably with `workspace-sandbox`, and must not stop `web-console`.
- Before introducing a long-lived heartbeat supervisor, keep cleanup semantics conservative: expire abandoned startup reservations, but treat stale `running` leases as diagnostic evidence rather than automatic deletion.

## Implementation Strategy

### Phase 0: Baseline Inventory and Decision Freeze

Goal: lock the target contracts before implementation.

Tasks:

- Add or update a short architecture note if the team wants a separate design doc, but keep this plan as the execution source.
- Confirm command contracts for `scenario list/status/port`, `locks`, `cleanup locks`, `orphans`, `cleanup orphans`, and `diagnose-port`.
- Inventory current tests around `internal/process`, `internal/ports`, `internal/lifecycle`, `internal/orchestrator`, `internal/maintenance`, `api-core/discovery`, browser automation studio scenario port resolution, and autoheal Vrooli state checks.
- Identify feature flag names for migration, for example `VROOLI_RUNTIME_REGISTRY=off|dual|prefer|strict`.

Validation:

```bash
vrooli help
go test ./internal/process ./internal/ports ./internal/lifecycle ./internal/orchestrator ./internal/maintenance
go test ./internal/api
go test ./internal/cli/scenariocli ./internal/cli/projectcli ./internal/app/project
cd packages/api-core && go test ./...
cd scenarios/browser-automation-studio/api && go test ./internal/scenarioport
cd scenarios/vrooli-autoheal/api && go test ./internal/checks/... ./internal/healing/...
```

Exit criteria:

- Contracts above are still accepted or updated in this plan.
- Test baseline is recorded in this ledger.
- No behavior changes are made.

### Phase 1: Runtime Registry Package

Goal: introduce the durable registry behind a narrow seam with unit tests only.

Tasks:

- Create `internal/scenarioruntime`.
- Define domain types: `Instance`, `PortClaim`, `HealthSnapshot`, `ProcessRef`, `Event`.
- Define repository interfaces for lifecycle, query, port claim, health, and cleanup operations.
- Implement SQLite storage under `$HOME/.vrooli/state/runtime.db`.
- Add schema initialization and versioning.
- Add injected clock and temp DB test seams.
- Add transaction helpers for atomic claim acquire/release and lifecycle transitions.

Tests:

- Creating, updating, and querying an instance.
- Generation increments prevent stale writers from overwriting newer instances.
- Active claim uniqueness prevents two scenarios from claiming the same host/port.
- Expired claims are queryable separately from active claims.
- Repository works with temp home/state paths and does not touch real user state.

Exit criteria:

- `go test ./internal/scenarioruntime` passes.
- No CLI or lifecycle reads use the registry yet.

### Phase 2: Lease, Heartbeat, and Health Snapshots

Goal: encode the core liveness/readiness split before wiring production lifecycle.

Tasks:

- Add lease methods: create lease, heartbeat lease, expire stale leases, stop lease.
- Define heartbeat TTL defaults and make them configurable in one internal location.
- Add health snapshot methods that can store schema-valid `/health` responses and manifest health check results.
- Add a small health probing adapter that can reuse `internal/scenario` health configuration and recognize `packages/api-core/health.Response` shape when available.
- Keep health probe errors structured and bounded.

Tests:

- Fresh lease reports active even when health is unknown.
- Stale lease reports expired even if a port listener exists.
- Unhealthy health snapshot does not delete the runtime instance.
- Schema-invalid health response is captured as diagnostic metadata.
- Time-dependent tests use injected clock, not sleeps.

Exit criteria:

- Registry can represent `starting`, `running`, `degraded`, `unhealthy`, `expired`, and `stopped` without PID inspection.

### Phase 3: Lifecycle Dual-Write

Goal: write registry state from scenario start/stop while legacy records remain authoritative.

Tasks:

- In `internal/lifecycle`, create a registry instance at start planning time.
- Write port reservations/claims as ports are allocated.
- Write process refs when `startTrackedProcess` launches children, but keep them diagnostic.
- Confirm claims as `bound` after health/listener confirmation.
- Heartbeat the active instance while lifecycle is supervising startup. If there is no long-lived supervisor yet, define the lease renewal owner explicitly and conservatively.
- On stop, mark instance stopping/stopped and release claims before or during legacy cleanup in a transactionally safe order.
- Add compensation paths so DB write failures cannot leave mount/process/claim state silently split.
- Keep legacy JSON records and plain lock files as the production read authority in this phase.

Tests:

- Successful start writes registry instance, process ref, claim, health snapshot, and legacy files.
- Start failure marks instance failed and releases reserved claims.
- Stop marks stopped and releases claims even when one cleanup sub-step fails.
- Registry write failure is surfaced and does not get reported as "scenario running" by the new registry.
- Legacy behavior remains compatible.

Exit criteria:

- Existing lifecycle commands behave as before.
- Registry data can be inspected and reconciled against legacy records.

### Phase 4: Registry-Backed Read Side

Goal: switch discovery reads to registry-first behavior behind migration controls.

Tasks:

- Add query interface in `internal/orchestrator` that reads from `internal/scenarioruntime`.
- Update `scenario list`, `scenario status`, and `scenario port` to prefer active registry records when the migration flag enables it.
- Keep legacy fallback for missing registry data during the migration window.
- Update `internal/scenario/runtime_state.go` so runtime status can be built from registry data rather than live PID count.
- Ensure `packages/api-core/discovery` still shells out through the same public CLI contract.
- Add explicit output fields where useful so operators can distinguish runtime lease, health, and diagnostics.

Tests:

- Sandbox-like test with hidden/unavailable PID diagnostics still resolves a registry-backed scenario port.
- Registry active + health degraded returns running/degraded semantics instead of not running.
- Registry expired + socket listener present reports stale/orphan diagnostic, not a valid active scenario.
- Legacy fallback works when registry is absent.
- API-core discovery does not learn any scenario-specific escape hatch.

Exit criteria:

- `vrooli scenario port <name> <port-key>` can succeed without `/proc` visibility when registry data is present.

### Phase 5: Registry-Backed Port Allocation and Claims

Goal: move port ownership authority from plain lock files to registry claims.

Tasks:

- Introduce a dedicated port claim service in `internal/ports` or `internal/portalloc`.
- Allocate by manifest policy and registry active claims.
- Confirm availability with socket bind/listener probing after checking registry claims.
- During migration, dual-write legacy lock files for compatibility.
- During migration, read legacy locks as fallback/conflict evidence only.
- Replace lock-before-bind race-prone behavior with reserved -> bound -> released claim transitions.
- Add cleanup for expired reserved claims.

Tests:

- Two concurrent allocators cannot claim the same port.
- Reserved claim expires if startup never binds.
- Bound claim survives transient health failure while lease is fresh.
- Socket bind failure with no registry claim is reported as external listener conflict.
- Legacy lock file stale state does not override an active registry claim.
- Claims are released on normal stop and expired on abandoned startup.

Exit criteria:

- Port ownership is registry-first under migration flag.
- Lock files are no longer necessary for correctness when both writer and reader use the registry.

### Phase 6: Maintenance, Diagnostics, and Autoheal Update

Goal: make orphan/stale handling stronger under the new model.

Tasks:

- Update `internal/maintenance` to read registry instances and claims.
- Extend `PortDiagnostic` with registry claim, lease status, health snapshot, and process ref fields.
- Update `CleanStaleLocks` into a registry-aware cleanup that expires stale claims and, during migration, still prunes legacy lock files.
- Update `Snapshot` so tracked process counts can use registry process refs plus legacy records during migration.
- Keep guarded host process orphan detection; revalidate registry/process identity immediately before killing.
- Update autoheal `VrooliStateReader` or replace it with a CLI/API-backed seam that can read registry state.
- Update autoheal stale lock and orphan checks to distinguish:
  - expired registry claim,
  - stale legacy lock,
  - live listener without active claim,
  - process ref whose PID no longer exists,
  - Vrooli-looking process with no active registry or legacy ownership.
- Keep `vrooli cleanup locks` and `vrooli cleanup orphans` as the healing commands so autoheal does not duplicate low-level cleanup policy.

Tests:

- Diagnose-port shows active registry owner.
- Diagnose-port shows expired claim plus live external listener.
- Cleanup locks expires stale registry claims and removes stale legacy locks during migration.
- Orphan cleanup does not kill protected control-plane APIs, CLI invocations, or unrelated processes.
- Autoheal reports registry stale claims and orphan listeners with actionable recovery actions.
- Autoheal clean restart still runs the appropriate core cleanup commands.

Exit criteria:

- Autoheal understands the registry model.
- Orphan and stale concepts remain operationally visible, but not part of normal discovery authority.

### Phase 7: Browser Automation Studio Discovery Cleanup

Goal: remove the browser automation studio runtime registry escape hatch.

Tasks:

- Remove `SCENARIO_REGISTRY` as a production discovery path in `scenarios/browser-automation-studio/api/internal/scenarioport`.
- Keep tests by injecting a port lookup seam or static resolver, not by using environment variables that bypass the public discovery contract.
- Confirm `DefaultScenarioCLI.LookupPort` goes through `packages/api-core/discovery`.
- Add regression tests proving BAS uses the same CLI/API discovery path as other scenarios.

Tests:

```bash
cd scenarios/browser-automation-studio/api && go test ./internal/scenarioport
```

Exit criteria:

- BAS has no special runtime discovery path.
- Tests still have a clean seam.

## Phase 7 Completion Record

Completed on 2026-05-08. This phase was limited to browser automation studio discovery cleanup. No scenario was stopped or restarted.

Files changed:

- `scenarios/browser-automation-studio/api/internal/scenarioport/registry.go`
  - Deleted the BAS-specific `SCENARIO_REGISTRY` parser, cache, URL combiner, and port lookup path.
- `scenarios/browser-automation-studio/api/internal/scenarioport/scenarioport.go`
  - Removed registry-first resolution from `ResolvePort` and `ResolveURL`.
  - Removed the legacy `portLookupFunc` test seam so runtime resolution goes through the `ScenarioCLI` boundary.
  - Kept `ScenarioCLI` as the package-level seam for tests and callers.
  - Made `DefaultScenarioCLI.LookupPort` delegate through an injectable `scenarioPortResolver`, defaulting to `discovery.NewResolver(discovery.ResolverConfig{})`, so the production path remains `packages/api-core/discovery`.
- `scenarios/browser-automation-studio/api/internal/scenarioport/scenarioport_test.go`
  - Replaced registry-override tests with seam-based tests.
  - Added regression coverage that setting `SCENARIO_REGISTRY` does not affect BAS URL or port resolution.
  - Added coverage that default CLI lookup delegates to an API-core-style resolver seam.
- `scenarios/browser-automation-studio/README.md`
  - Replaced the scenario registry override documentation with scenario discovery guidance that points to shared `api-core/discovery` and `vrooli scenario port`.

Validation:

```bash
cd scenarios/browser-automation-studio/api && go test ./internal/scenarioport -count=1
# passed

rg -n "SCENARIO_REGISTRY|lookupRegistryEntry|resetRegistryCacheForTests|registryEnvVar|scenarioRegistry|SetPortLookupFuncForTests|portLookupFunc" scenarios/browser-automation-studio -g'*.go' -g'*.md'
# only the regression tests still mention SCENARIO_REGISTRY, to prove it is ignored
```

Important findings:

- The production BAS runtime discovery path no longer has a scenario-local registry, file parser, or env-var override. Runtime port and URL resolution now flows through the same scenario CLI/API-core discovery contract as the rest of Vrooli.
- The remaining `SCENARIO_REGISTRY` mentions are intentionally confined to tests that set the variable and assert it no longer wins over the shared discovery seam.
- Phase 7 did not validate live sandbox behavior. That remains Phase 8 and should start with unit/integration confidence, then a selected non-console scenario such as `workspace-sandbox`.

### Phase 8: Sandbox Validation

Goal: prove the actual blocker is fixed through the same public discovery path used by agents.

Tasks:

- Start a representative host scenario through lifecycle only.
- From a workspace sandbox or sandbox-like test harness, run `vrooli scenario status` and `vrooli scenario port`.
- Verify API-core discovery works from inside the sandbox without host PID visibility.
- Separately validate loopback/network profile reachability. If registry lookup succeeds but TCP connection fails, record it as sandbox networking, not discovery.
- Validate agent-manager spawned agents can resolve required scenario ports by default.

Suggested scenario tests:

```bash
vrooli scenario test workspace-sandbox
vrooli scenario test agent-manager
vrooli scenario test browser-automation-studio
vrooli scenario test vrooli-autoheal
```

Exit criteria:

- Sandboxed agents can discover running scenarios through `api-core/discovery`.
- Any remaining failure is classified as network reachability, health, or lifecycle, not runtime discovery.

## Phase 8 Completion Record

Completed on 2026-05-08. This phase validated the registry-backed discovery path without changing the default/global migration mode and without stopping `web-console`.

Files changed:

- `internal/orchestrator/orchestrator_test.go`
  - Added `TestStrictRegistryModeResolvesPortWithoutPIDVisibility`.
  - The test creates a registry-backed running instance and a stale/unusable legacy process record, enables `VROOLI_RUNTIME_REGISTRY=strict`, and verifies `ResolvePort` and `Status` use the registry claim rather than PID-visible process records.

Validation:

```bash
go test ./internal/orchestrator -run 'TestStrictRegistryModeResolvesPortWithoutPIDVisibility|TestResolvePortUsesRegistryInPreferModeWithoutProcessRecords|TestStrictRegistryModeDoesNotUseLegacyProcessRecords' -count=1
# passed

go test ./internal/scenarioruntime ./internal/orchestrator ./internal/app/scenario ./internal/cli/scenariocli ./internal/maintenance ./internal/ports ./internal/lifecycle -count=1
# passed

cd packages/api-core && go test ./... -count=1
# passed

cd scenarios/browser-automation-studio/api && go test ./internal/scenarioport -count=1
# passed

cd scenarios/vrooli-autoheal/api && go test ./internal/checks/... ./internal/healing/... -count=1
# passed

vrooli scenario status workspace-sandbox --json
# initially running through legacy/default discovery

VROOLI_RUNTIME_REGISTRY=strict vrooli scenario status workspace-sandbox --json
# initially reported stopped because the existing live instance had been started before registry writes were enabled

vrooli scenario test workspace-sandbox
# failed in phase-standards only:
# 10 passed, 1 failed, highest standards finding critical
# discovery/runtime phases did not identify a runtime-registry blocker

VROOLI_RUNTIME_REGISTRY=prefer vrooli scenario start workspace-sandbox
# passed; workspace-sandbox restarted through lifecycle with registry writes enabled

VROOLI_RUNTIME_REGISTRY=strict vrooli scenario status workspace-sandbox --json
# passed; status running, runtime registry, API_PORT=15120, UI_PORT=21239, WS_PORT=28836

VROOLI_RUNTIME_REGISTRY=strict vrooli scenario port workspace-sandbox API_PORT
# passed; 15120

VROOLI_RUNTIME_REGISTRY=strict vrooli scenario port workspace-sandbox UI_PORT
# passed; 21239

VROOLI_RUNTIME_REGISTRY=strict vrooli scenario port workspace-sandbox WS_PORT
# passed; 28836

curl -fsS http://localhost:15120/health
# passed; standardized healthy/readiness=true response from workspace-sandbox API

vrooli scenario stop browser-automation-studio
# passed; restored browser-automation-studio to its pre-validation stopped state after workspace-sandbox testing started it as a dependency

vrooli scenario status web-console --json
# passed; web-console remained running and was not stopped or restarted
```

Important findings:

- The core sandbox blocker is fixed for registry-backed instances: strict registry mode can resolve a running scenario port without relying on PID visibility or legacy process liveness.
- Existing running scenarios that were started before registry writes were enabled do not magically become strict-registry discoverable. This is expected during migration. They remain visible in default/off mode and in prefer mode through fallback until they are restarted under `VROOLI_RUNTIME_REGISTRY=dual|prefer|strict`.
- The workspace-sandbox scenario test currently has an unrelated quality gate failure in `phase-standards`: 10 phases passed and 1 failed because standards violations exceed `fail_on=high` with highest severity `critical`. That should be handled as a workspace-sandbox standards cleanup task, not as a runtime discovery blocker.
- `vrooli locks --json` now reports bound registry claims for workspace-sandbox. The claims show `lease_fresh=false` after startup because there is still no long-lived supervisor heartbeat. That is the known Phase 6 migration constraint; strict read-side discovery intentionally still treats active `running` registry instances as discoverable until a supervisor owns continuous renewal.
- The live validation briefly started browser-automation-studio as a dependency of workspace-sandbox testing. It was stopped afterward to restore the pre-validation state. `web-console` was not stopped or restarted.

Recommended next steps:

- Do not start Phase 9 cleanup yet. First run a short migration soak with a small allowlist of non-console scenarios started under `VROOLI_RUNTIME_REGISTRY=prefer`, beginning with workspace-sandbox.
- Add a follow-up task for the workspace-sandbox standards failure if that scenario needs a green full scenario test before broader rollout.
- Before removing legacy authority, decide and implement the long-lived heartbeat/supervisor owner. The current registry is good enough for migration-mode discovery validation, but cleanup semantics should stay conservative until continuous renewal exists.

### Phase 8A: Allowlist Rollout Guardrail

Goal: make the short migration soak safe by allowing operators and agents to enable registry migration mode for a small set of scenarios without changing read/write semantics for the rest of the running system.

Tasks:

- Add a scenario allowlist env var for migration-mode scoping.
- Keep empty allowlist behavior backward compatible: the selected migration mode applies globally, as it did before this phase.
- When the allowlist is non-empty, apply registry writes only to allowlisted scenarios.
- When the allowlist is non-empty, apply registry reads and strict-read behavior only to allowlisted scenarios.
- Preserve legacy/off behavior for non-allowlisted scenarios, even when the global mode is `prefer` or `strict`.
- Add tests for mode parsing, lifecycle write gating, and orchestrator read gating.
- Perform non-invasive live checks only. Do not stop or restart `web-console`.

Exit criteria:

- `VROOLI_RUNTIME_REGISTRY=prefer VROOLI_RUNTIME_REGISTRY_ALLOWLIST=workspace-sandbox` uses registry reads for `workspace-sandbox`.
- `VROOLI_RUNTIME_REGISTRY=strict VROOLI_RUNTIME_REGISTRY_ALLOWLIST=workspace-sandbox` does not force non-allowlisted scenarios such as `web-console` into strict registry-only reads.
- Broad affected package tests pass.

## Phase 8A Completion Record

Completed on 2026-05-08. This phase adds the rollout guardrail needed before a short allowlist soak. It does not start Phase 9 cleanup and does not remove legacy authority.

Files changed:

- `internal/scenarioruntime/mode.go`
  - Added `VROOLI_RUNTIME_REGISTRY_ALLOWLIST`.
  - Added allowlist parsing and scenario-scoped read/write/strict classification helpers.
  - Empty allowlist remains global, preserving existing behavior for already-written tests and explicit full-mode validation.
- `internal/scenarioruntime/mode_test.go`
  - Added tests for empty allowlist, normalized scenario matching, and scenario-scoped mode classification.
- `internal/lifecycle/runtime_registry.go`
  - Gated runtime registry start/stop/process-ref writes by scenario allowlist.
  - Non-allowlisted scenarios remain legacy-write-only even if `VROOLI_RUNTIME_REGISTRY=dual|prefer|strict`.
- `internal/lifecycle/lifecycle_test.go`
  - Added coverage that a non-allowlisted scenario start under `dual` does not create the runtime registry DB.
- `internal/orchestrator/runtime_registry_read.go`
  - Gated registry detail reads by scenario allowlist.
- `internal/orchestrator/detail.go`
  - Made inventory/status/detail reads evaluate strict/prefer behavior per scenario when an allowlist is present.
  - Non-allowlisted scenarios keep legacy fallback behavior under global `strict`.
- `internal/orchestrator/orchestrator_test.go`
  - Added coverage that `prefer` and `strict` modes apply only to allowlisted scenarios.

Validation:

```bash
go test ./internal/scenarioruntime -run 'TestModeFromString|TestScenarioAllowed|TestScenarioScopedModeClassification|TestLocalPortURL|TestActiveInstanceStatuses' -count=1
# passed

go test ./internal/orchestrator -run 'TestRegistryAllowlist|TestResolvePortUsesRegistry|TestStrictRegistryMode|TestPreferRegistryModeFallsBack' -count=1
# passed

go test ./internal/lifecycle -run 'TestRunnerStartHonorsRuntimeRegistryAllowlist|TestRunnerStartDualWritesScenarioRuntimeRegistry|TestRunnerStopDualWritesStoppedRuntimeRegistry' -count=1
# passed

go test ./internal/scenarioruntime ./internal/orchestrator ./internal/lifecycle ./internal/ports ./internal/maintenance -count=1
# passed

VROOLI_RUNTIME_REGISTRY=prefer VROOLI_RUNTIME_REGISTRY_ALLOWLIST=workspace-sandbox vrooli scenario status workspace-sandbox --json
# passed; workspace-sandbox reported running from runtime="registry" with API_PORT=15120, UI_PORT=21239, WS_PORT=28836

VROOLI_RUNTIME_REGISTRY=strict VROOLI_RUNTIME_REGISTRY_ALLOWLIST=workspace-sandbox vrooli scenario status web-console --json
# passed; web-console remained running through legacy runtime discovery, proving non-allowlisted scenarios are not forced into strict registry-only reads

vrooli locks --json
# passed; reported workspace-sandbox registry claims and legacy lock files

vrooli diagnose-port 28836 --json
# passed; reported no listener and no host orphan, a stale legacy lock for workspace-sandbox WS_PORT, and a bound registry claim for the current workspace-sandbox instance
```

Important findings:

- The allowlist gives the project a safer soak mechanism than setting `VROOLI_RUNTIME_REGISTRY=prefer` globally.
- `workspace-sandbox` is a good first allowlisted scenario because it already has registry-backed running state from Phase 8.
- `web-console` was not stopped or restarted. The strict-with-allowlist status check confirmed it stays on legacy discovery when not allowlisted.
- The stale legacy lock on `28836` is exactly the kind of legacy artifact expected during migration. Registry diagnostics correctly separate it from the current bound registry claim. It was not cleaned during this phase.
- `lease_fresh=false` remains expected until a long-lived supervisor/heartbeat owner exists. Do not use lease freshness alone as a cleanup trigger for running migration instances yet.

Recommended next steps:

- Run a short soak with `VROOLI_RUNTIME_REGISTRY=prefer VROOLI_RUNTIME_REGISTRY_ALLOWLIST=workspace-sandbox` for agent-manager spawned agents that need sandbox scenario discovery.
- Monitor `vrooli scenario status workspace-sandbox --json`, `vrooli scenario port workspace-sandbox API_PORT`, `vrooli locks --json`, and `vrooli diagnose-port <workspace-sandbox ports> --json` during the soak.
- Add one or two more non-console scenarios to the allowlist only after workspace-sandbox remains stable.
- Do not start Phase 9 cleanup until the allowlist soak is stable and the long-lived heartbeat/supervisor decision is implemented or explicitly deferred with conservative cleanup rules.

## Phase 8A Soak Start Record

Started on 2026-05-08 at 18:43 America/New_York. This is an operational soak, not a new implementation phase.

Soak configuration:

```bash
VROOLI_RUNTIME_REGISTRY=prefer
VROOLI_RUNTIME_REGISTRY_ALLOWLIST=workspace-sandbox
```

Background monitor:

- PID file: `/home/matthalloran8/.vrooli/logs/runtime-registry-soak/workspace-sandbox-20260508.pid`
- Current PID at start: `2268858`
- Log file: `/home/matthalloran8/.vrooli/logs/runtime-registry-soak/workspace-sandbox-20260508.log`
- Cadence: every 300 seconds

Read-only commands captured each cycle:

```bash
VROOLI_RUNTIME_REGISTRY=prefer VROOLI_RUNTIME_REGISTRY_ALLOWLIST=workspace-sandbox vrooli scenario status workspace-sandbox --json
VROOLI_RUNTIME_REGISTRY=prefer VROOLI_RUNTIME_REGISTRY_ALLOWLIST=workspace-sandbox vrooli scenario port workspace-sandbox API_PORT
VROOLI_RUNTIME_REGISTRY=strict VROOLI_RUNTIME_REGISTRY_ALLOWLIST=workspace-sandbox vrooli scenario status web-console --json
vrooli locks --json
vrooli diagnose-port 15120 --json
vrooli diagnose-port 28836 --json
```

Baseline at soak start:

- `workspace-sandbox` status passed through registry discovery: running, healthy, API_PORT `15120`, UI_PORT `21239`, WS_PORT `28836`.
- `web-console` status passed through legacy discovery while non-allowlisted, proving the allowlist guard does not force it into strict registry reads.
- `workspace-sandbox` API port resolution returned `15120`.
- Existing stale legacy WS lock on port `28836` remained present; registry claim was bound to the current workspace-sandbox instance. This was already known from Phase 8A validation and was not cleaned.

When checking the soak later:

```bash
PID="$(cat /home/matthalloran8/.vrooli/logs/runtime-registry-soak/workspace-sandbox-20260508.pid)"
ps -p "$PID" -o pid,ppid,sid,stat,etime,cmd
tail -n 200 /home/matthalloran8/.vrooli/logs/runtime-registry-soak/workspace-sandbox-20260508.log
```

### Phase 8A Soak Checkpoint 1

Checked on 2026-05-08 at 21:43 America/New_York.

Monitor status:

- PID `2268858` was still running.
- Elapsed monitor time was about 3 hours.
- Log had 37 completed cycles.
- Latest cycle timestamp was `2026-05-08T21:43:35-04:00`.

Observed state:

- Fresh `workspace-sandbox` status with `VROOLI_RUNTIME_REGISTRY=prefer VROOLI_RUNTIME_REGISTRY_ALLOWLIST=workspace-sandbox` passed.
- `workspace-sandbox` remained `running`, `healthy`, and registry-backed.
- `workspace-sandbox` API port resolution still returned `15120`.
- Fresh `web-console` status with `VROOLI_RUNTIME_REGISTRY=strict VROOLI_RUNTIME_REGISTRY_ALLOWLIST=workspace-sandbox` passed and stayed on legacy runtime discovery, as intended for a non-allowlisted scenario.
- The log contained no `"success": false` entries and no `scenario_not_running` entries.
- The known stale legacy WS lock for `workspace-sandbox` port `28836` remained present with no listener/orphan and a bound registry claim.

Decision:

- Continue the soak. Do not add more scenarios yet and do not start Phase 9 cleanup.

### Phase 8A Soak Scope Expansion

Started on 2026-05-08 at 21:50 America/New_York.

The original workspace-only monitor was stopped cleanly:

- Old PID: `2268858`

Expanded allowlist:

```bash
VROOLI_RUNTIME_REGISTRY_ALLOWLIST=workspace-sandbox,agent-manager,swarm-manager
```

Replacement monitor:

- PID file: `/home/matthalloran8/.vrooli/logs/runtime-registry-soak/workspace-agent-swarm-20260508.pid`
- Current PID at start: `3363470`
- Log file: `/home/matthalloran8/.vrooli/logs/runtime-registry-soak/workspace-agent-swarm-20260508.log`
- Cadence: every 300 seconds

Important baseline caveats:

- `workspace-sandbox` remains the only currently registry-backed running scenario in this allowlist.
- `agent-manager` is running and healthy, but it is still legacy-backed under `prefer` because it was started before registry writes were enabled. It is included for read-path/fallback monitoring only until a later explicit restart under `VROOLI_RUNTIME_REGISTRY=prefer`.
- `swarm-manager` is currently stopped. It is included for status monitoring only until a later explicit lifecycle start under `VROOLI_RUNTIME_REGISTRY=prefer`.
- No scenario was stopped, restarted, or started during this scope expansion.
- `web-console` remains outside the allowlist and is still monitored as the non-allowlisted guard.

Expanded read-only commands captured each cycle:

```bash
ALLOWLIST=workspace-sandbox,agent-manager,swarm-manager
VROOLI_RUNTIME_REGISTRY=prefer VROOLI_RUNTIME_REGISTRY_ALLOWLIST="$ALLOWLIST" vrooli scenario status workspace-sandbox --json
VROOLI_RUNTIME_REGISTRY=prefer VROOLI_RUNTIME_REGISTRY_ALLOWLIST="$ALLOWLIST" vrooli scenario port workspace-sandbox API_PORT
VROOLI_RUNTIME_REGISTRY=prefer VROOLI_RUNTIME_REGISTRY_ALLOWLIST="$ALLOWLIST" vrooli scenario status agent-manager --json
VROOLI_RUNTIME_REGISTRY=prefer VROOLI_RUNTIME_REGISTRY_ALLOWLIST="$ALLOWLIST" vrooli scenario port agent-manager API_PORT
VROOLI_RUNTIME_REGISTRY=prefer VROOLI_RUNTIME_REGISTRY_ALLOWLIST="$ALLOWLIST" vrooli scenario status swarm-manager --json
VROOLI_RUNTIME_REGISTRY=strict VROOLI_RUNTIME_REGISTRY_ALLOWLIST="$ALLOWLIST" vrooli scenario status web-console --json
vrooli locks --json
vrooli diagnose-port 15120 --json
vrooli diagnose-port 28836 --json
vrooli diagnose-port 18800 --json
```

When checking the expanded soak later:

```bash
PID="$(cat /home/matthalloran8/.vrooli/logs/runtime-registry-soak/workspace-agent-swarm-20260508.pid)"
ps -p "$PID" -o pid,ppid,sid,stat,etime,cmd
tail -n 240 /home/matthalloran8/.vrooli/logs/runtime-registry-soak/workspace-agent-swarm-20260508.log
```

### Phase 8A Crash Checkpoint

Checked on 2026-05-08 at about 22:43 America/New_York after a machine crash/reboot.

Crash/reboot evidence:

- `uptime -s` reported boot time `2026-05-08 21:55:50`.
- The expanded soak monitor PID `3363470` was gone after reboot.
- The expanded soak log only had one cycle from `2026-05-08T21:50:20-04:00`, so the crash happened shortly after the scope expansion.

Post-crash current state:

- `workspace-sandbox` came back up automatically in default/legacy discovery mode with new post-boot PIDs:
  - API PID `30689`, API_PORT `15120`
  - UI PID `30734`, UI_PORT `21239`
- `agent-manager` came back up automatically in default/legacy discovery mode:
  - API PID `34055`, API_PORT `18800`
  - UI PID `34154`, UI_PORT `21238`
- `web-console` came back up automatically in default/legacy discovery mode:
  - API PID `67599`, API_PORT `16382`
  - UI PID `67644`, UI_PORT `21233`
- `swarm-manager` remained stopped.

What the legacy side did well:

- `vrooli locks --json` reported no stale legacy lock files after the reboot.
- Legacy lock files for restarted scenarios were updated to post-boot PIDs.
- `vrooli orphans --json` reported no host orphans.
- `workspace-sandbox` API health endpoint on `15120` returned healthy/readiness true.

What the registry side did not catch:

- The runtime registry still contained only the pre-crash `workspace-sandbox` instance:
  - instance `inst-f7d8e1c476ddee8f8a4ea0f5db88a36d`
  - status `running`
  - last heartbeat `2026-05-08T22:28:37.137225690Z`
  - heartbeat deadline `2026-05-08T22:29:07.137225690Z`
  - owner PID `2174272`
- Registry process refs still pointed at dead pre-crash PIDs:
  - API PID `2175061`, `pid_running=false`
  - UI PID `2175123`, `pid_running=false`
- Registry claims remained `bound` and `instance_status=running` for ports `15120`, `21239`, and `28836`, with `lease_fresh=false`.
- `VROOLI_RUNTIME_REGISTRY=strict ... vrooli scenario status workspace-sandbox --json` still reported `workspace-sandbox` as registry-backed running from the stale pre-crash instance.
- `VROOLI_RUNTIME_REGISTRY=strict ... vrooli scenario port workspace-sandbox WS_PORT` returned `28836` even though `vrooli diagnose-port 28836 --json` reported `in_use=false`.
- `VROOLI_RUNTIME_REGISTRY=strict ... vrooli scenario status agent-manager --json` reported stopped because `agent-manager` had no registry rows, even though default/prefer fallback saw it running from legacy records.

Conclusion:

- The crash produced valuable evidence. The current registry migration model is not yet safe for strict read authority across reboots.
- Legacy startup/lock recovery behaved better than the registry after this reboot.
- The missing piece is a boot/session-aware registry reconciliation step before strict reads are trusted after host restart.

Required follow-up before Phase 9:

- Add a Phase 8B hardening task for reboot/crash reconciliation.
- Store enough owner/session metadata to distinguish pre-boot registry instances from post-boot runtime.
- On startup or first registry read after boot, mark active registry instances with stale heartbeat plus dead owner/process refs as `expired` or otherwise exclude them from strict discovery.
- Reconcile registry claims against actual listeners and health:
  - API/UI claims with matching listeners may be adoptable only if the post-boot lifecycle path explicitly writes or adopts them.
  - Claims without listeners, such as `workspace-sandbox` WS_PORT `28836`, must not remain authoritative in strict discovery.
- Keep prefer-mode fallback for non-registry restarted scenarios until their lifecycle start path writes fresh registry rows.

Post-crash monitor:

- Started a new read-only monitor focused on comparing prefer/default recovery with strict-registry behavior.
- PID file: `/home/matthalloran8/.vrooli/logs/runtime-registry-soak/post-crash-workspace-agent-swarm-20260509.pid`
- Current PID at start: `296089`
- Log file: `/home/matthalloran8/.vrooli/logs/runtime-registry-soak/post-crash-workspace-agent-swarm-20260509.log`
- Cadence: every 300 seconds

When checking the post-crash monitor:

```bash
PID="$(cat /home/matthalloran8/.vrooli/logs/runtime-registry-soak/post-crash-workspace-agent-swarm-20260509.pid)"
ps -p "$PID" -o pid,ppid,sid,stat,etime,cmd
tail -n 260 /home/matthalloran8/.vrooli/logs/runtime-registry-soak/post-crash-workspace-agent-swarm-20260509.log
```

### Phase 8B: Crash/Reboot and Sudden-Stop Reconciliation

Goal: make the runtime registry professionally safe as a long-term source of truth when the host reboots, the machine crashes, or a scenario process exits outside normal `vrooli scenario stop/restart` lifecycle commands.

This phase is mandatory before Phase 9. The crash checkpoint proved that the current registry can preserve stale authority after reboot: a pre-crash `workspace-sandbox` instance and bound claims remained discoverable in strict mode even though its process refs were dead and its WS claim had no listener. Legacy records recovered better after reboot. The registry must be hardened until strict-mode discovery is at least as conservative as legacy recovery under crash and sudden-stop conditions.

Required reading before implementation:

```bash
prompt-manager skill read plan-skill-discovery implementation-plan-authoring screaming-architecture-audit seam-discovery-and-enforcement boundary-of-responsibility-enforcement
prompt-manager skill read progress-continuity-interruption-resilience error-semantics-recovery-path-design failure-topography-and-graceful-degradation temporal-flow-audit idempotency-replay-safety-hardening
```

Discovery evidence behind this phase:

```bash
prompt-manager discover "runtime registry crash recovery" "boot session reconciliation" "scenario lifecycle sudden stop" "port claim stale cleanup" --complexity architectural
# found recovery, temporal-flow, idempotency, failure-topography, and progress-continuity skills within budget
```

#### Problem Statement

The registry currently records active instance state, process refs, bound port claims, health snapshots, and heartbeat deadlines. Those fields are not sufficient after reboot or external termination because numeric PIDs can die or be reused, a stale `running` row can outlive the boot/session that created it, a bound claim can outlive its listener, health snapshots describe a past process, and `prefer`/`strict` reads currently select active registry rows before proving they still match current host reality.

The target behavior is not "delete anything stale immediately." The target behavior is safer:

- stale evidence must stop being authoritative for discovery;
- diagnostics must explain the stale condition;
- cleanup/adoption must be explicit, idempotent, and revalidated immediately before destructive action.

#### Scope

In scope:

- Host boot/session identity for runtime registry instances and process refs.
- Reconciliation of active registry instances before registry-backed read authority is used.
- Safe classification of stale, orphaned, adoptable, and healthy runtime states.
- Claim-level reconciliation against listener evidence and health readiness.
- Conservative maintenance cleanup for stale registry rows and claims.
- Tests for reboot, sudden stop, PID reuse, no-listener claims, listener-with-stale-registry, and allowlist strict/prefer behavior.
- Operator-facing diagnostics in `vrooli locks`, `vrooli diagnose-port`, `vrooli cleanup locks`, and scenario status/port behavior.

Out of scope:

- Removing legacy process records or lock-file fallback.
- Killing host processes solely because a registry row is stale.
- Treating health alone as liveness.
- Broad scenario restarts, especially `web-console`, unless explicitly approved.
- Global strict-mode rollout.

#### Target End State

The runtime registry must distinguish at least these concepts:

| Concept | Meaning | Discovery authority |
| --- | --- | --- |
| Fresh managed runtime | Current boot/session, active lifecycle/supervisor ownership, valid claims, expected listeners/health where applicable | Can answer strict/prefer registry discovery |
| Stale registry instance | Active row from an old boot/session, stale heartbeat, or dead owner/process refs with no adoptable evidence | Must not answer strict discovery |
| Stale claim | Reserved/bound claim whose instance is stale or whose listener is absent when listener evidence is required | Must not answer `scenario port`; eligible for conservative cleanup |
| Orphan listener | Listener exists but is not referenced by current registry or legacy lifecycle state | Diagnostic/remediation only; not automatically adopted |
| Adoptable runtime | Listener and legacy/current lifecycle evidence match an expected scenario after reboot, but registry row is absent/stale | Requires explicit adoption path before registry authority |
| Unknown/unverified | Listener inspection unavailable or evidence incomplete | Prefer fallback may use legacy when allowed; strict should fail closed with a structured diagnostic |

Strict mode must fail closed. If registry evidence cannot be reconciled, `scenario status` should report stopped/expired/unverified rather than running, and `scenario port` must not return a stale claim as usable.

#### Contract Decisions

1. **Boot/session identity is required.**
   Add a host boot identity seam under a platform-owned package, likely `internal/hostsession` or `internal/scenarioruntime/hostsession`. Linux should use `/proc/sys/kernel/random/boot_id`. Non-Linux should use a Vrooli-managed host-session token under `$HOME/.vrooli/state/` with clear limitations. Store boot/session metadata on `runtime_instances` and `runtime_process_refs`; optionally add a `runtime_host_sessions` table for richer observability.

2. **Reconciliation is a domain workflow, not ad hoc filtering.**
   Add a runtime reconciliation service/model in `internal/scenarioruntime` that classifies instances and claims from durable registry state plus injected host evidence. Keep side effects behind seams: process inspection, listener inspection, health probing, clock, and store writes. The pure classification rules should be table-tested as temporal workflow logic.

3. **Registry reads must pass through reconciliation.**
   `internal/orchestrator/runtime_registry_read.go` must not directly select latest active instances as discovery authority. Registry-backed status/port reads should consume reconciled candidates only. `prefer` may fall back to legacy if the registry candidate is stale/unverified. `strict` must not fall back to legacy for allowlisted scenarios, but it also must not return stale registry data.

4. **Bound claims need listener-aware semantics.**
   A bound claim with a stale instance and no listener is stale, not running. A bound claim with a listener but old boot/session is not automatically current authority; it is an orphan/adoption candidate until the lifecycle/supervisor writes or adopts it. WS or optional ports without listeners must be handled explicitly by manifest/port semantics, not guessed from claim status alone.

5. **Cleanup and adoption must be idempotent.**
   Re-running reconciliation or cleanup should not create duplicate instances, duplicate claims, or contradictory status transitions. Use stable instance/generation guards and transaction boundaries. Prefer marking old rows `expired` and claims `expired` over deleting rows during migration.

6. **No destructive action without fresh revalidation.**
   Maintenance may mark registry rows non-authoritative based on stale evidence. Killing host processes still requires fresh process/listener revalidation and existing protected-process rules.

#### Implementation Strategy

Phase 8B.1: Host/session metadata foundation

- Add a boot/session identity seam with unit tests for Linux boot id, fallback behavior, and error cases.
- Add schema migration support from registry schema version 1 to 2.
- Extend `Instance` and `ProcessRef` with host/session fields.
- Write current boot/session metadata during lifecycle start and process-ref creation.
- Add backward-compatible scanning for old rows so existing registry databases can be reconciled rather than discarded.

Phase 8B.2: Reconciliation model

- Add explicit classification types such as `VerifiedRunning`, `StaleInstance`, `StaleClaim`, `OrphanListener`, `AdoptionCandidate`, and `Unverified`.
- Implement pure rules that consume instance status, heartbeat deadline, stored boot/session id, owner/process PID liveness and boot/session match, active claims, listener inspection, and health snapshot/probe freshness.
- Add matrix tests for reboot, external kill, partial API/UI death, PID reuse, no listener inspection, stale heartbeat, old schema rows, and mixed legacy/registry evidence.

Phase 8B.3: Read-side enforcement

- Route `registryDetailsByScenario`, `registryDetail`, and `ResolvePort` registry paths through reconciliation.
- Ensure stale/unverified instances cannot populate `RuntimeDetails.Ports`.
- Preserve allowlist semantics:
  - allowlisted `prefer`: reconciled registry first, legacy fallback if registry is stale/missing/unverified;
  - allowlisted `strict`: reconciled registry only, fail closed;
  - non-allowlisted scenarios: legacy behavior unchanged even under global strict.
- Add regression tests using the observed crash case: pre-crash `workspace-sandbox` running instance, old dead process refs, stale heartbeat, bound API/UI/WS claims, different current boot, no WS listener, and strict mode must not return WS_PORT `28836` as authoritative.

Phase 8B.4: Maintenance and diagnostics

- Extend `vrooli locks --json` and `vrooli diagnose-port --json` to include reconciliation classification and recommended recovery action.
- Extend `vrooli cleanup locks` to expire stale registry instances/claims that are non-authoritative after reconciliation.
- Keep legacy lock cleanup behavior.
- Add a dedicated dry-run report if the cleanup surface becomes too ambiguous.
- Make recommendations specific: expire stale registry claim, restart scenario under registry-enabled lifecycle, inspect orphan listener, or withhold strict discovery because listener inspection is unavailable.

Phase 8B.5: Adoption path, if needed for boot recovery

- Decide whether adoption belongs in lifecycle start, a maintenance command, or a future supervisor.
- If implemented now, adoption must require strong evidence: current boot/session, live process records or listener PIDs that match expected Vrooli process identity, declared manifest ports, and health readiness for readiness-sensitive surfaces.
- Adoption must create a new generation or mark a new current instance rather than reviving a pre-crash generation.
- If adoption is deferred, document that post-reboot running scenarios must be restarted under registry-enabled lifecycle before strict registry authority is expected.

Phase 8B.6: Soak restart

- Stop any stale read-only monitor processes if present; do not stop scenarios.
- Restart a read-only monitor only after Phase 8B tests pass.
- Begin with `workspace-sandbox` only.
- Do not expand to `agent-manager` or `swarm-manager` until the stale pre-boot instance case is proven fixed.

#### Testing Plan

Focused unit tests:

```bash
go test ./internal/scenarioruntime -run 'Test.*Reconcile|Test.*HostSession|Test.*Boot|Test.*Stale|Test.*Adopt' -count=1
go test ./internal/orchestrator -run 'Test.*Registry.*Reboot|Test.*Registry.*Strict|Test.*Registry.*Allowlist|Test.*ResolvePort' -count=1
go test ./internal/maintenance -run 'Test.*Registry.*Cleanup|Test.*Diagnose.*Registry|Test.*Stale' -count=1
go test ./internal/lifecycle -run 'Test.*RuntimeRegistry.*Session|Test.*RuntimeRegistry.*ProcessRef' -count=1
```

Broad package validation:

```bash
go test ./internal/scenarioruntime ./internal/orchestrator ./internal/lifecycle ./internal/ports ./internal/maintenance ./internal/app/scenario ./internal/cli/scenariocli -count=1
cd packages/api-core && go test ./... -count=1
```

Live validation must be non-invasive unless explicitly approved:

```bash
VROOLI_RUNTIME_REGISTRY=strict VROOLI_RUNTIME_REGISTRY_ALLOWLIST=workspace-sandbox vrooli scenario status workspace-sandbox --json
VROOLI_RUNTIME_REGISTRY=strict VROOLI_RUNTIME_REGISTRY_ALLOWLIST=workspace-sandbox vrooli scenario port workspace-sandbox WS_PORT
vrooli diagnose-port 28836 --json
vrooli locks --json
```

Expected live result before any restart/adoption: stale pre-crash registry evidence is classified as stale/non-authoritative, not running. If a current legacy-backed scenario is running, `prefer` may fall back to legacy; `strict` must fail closed until the scenario is restarted or explicitly adopted under the registry model.

#### Acceptance Criteria

- A stale registry instance from a previous boot/session cannot make strict-mode `scenario status` report `running`.
- A stale bound registry claim without a listener cannot make strict-mode `scenario port` return a port.
- A stale bound registry claim with a listener is classified as orphan/adoptable, not silently trusted.
- `prefer` mode remains operational during migration by falling back to legacy when the registry candidate is stale or unverified.
- `web-console` remains protected by allowlist behavior and is not forced into registry reads while non-allowlisted.
- Maintenance commands can expire stale registry claims/instances without deleting useful diagnostic history.
- All reconciliation decisions are covered by table or matrix tests, not just one-off happy-path tests.
- The plan records whether adoption was implemented or explicitly deferred before the soak restarts.

#### Prohibited Shortcuts

- Do not solve this by globally disabling strict mode.
- Do not delete `runtime.db` as the recovery strategy.
- Do not trust PID existence without boot/session or process identity checks.
- Do not trust old health snapshots as current readiness.
- Do not expose reserved/bound claims as runtime ports until reconciliation says the owning runtime is current.
- Do not use listener presence alone to adopt a scenario into the registry.
- Do not start Phase 9 while any allowlisted strict-mode scenario can resolve stale pre-crash claims.

#### Phase 8B Completion Record

Completed on 2026-05-09.

Files added or changed:

- `internal/hostsession/`
  - Added the host-session seam. Linux uses `/proc/sys/kernel/random/boot_id`; non-Linux falls back to a Vrooli-managed persistent token with explicit limitations.
- `internal/scenarioruntime/`
  - Bumped the registry schema to version 2.
  - Added `host_boot_id` / `host_session_id` to instances and `host_boot_id` to process refs.
  - Added v1-to-v2 migration support without discarding existing `runtime.db` history.
  - Added `ReconcileRuntime`, which classifies active registry state as verified, stale, or unverified before reads expose it.
- `internal/lifecycle/`
  - Lifecycle registry writes now stamp host/session metadata on runtime instances and process refs.
- `internal/orchestrator/`
  - Registry-backed status/detail/port reads now pass through reconciliation.
  - `prefer` falls back to legacy when registry evidence is stale or unverified.
  - `strict` fails closed and does not expose stale/unverified registry claims.
- `internal/maintenance/`
  - `vrooli locks --json` and `vrooli diagnose-port --json` now expose reconciliation classification, reason, and authority.
  - `vrooli cleanup locks` can expire stale non-authoritative registry instances and claims. It intentionally does not expire `unverified` old-schema rows automatically; operators should restart those scenarios under registry-enabled lifecycle or decide on explicit adoption later.

Adoption decision:

- Automatic adoption is deferred. A stale listener or old registry row is not enough evidence to revive an old generation.
- Post-reboot strict registry authority requires either a fresh registry-enabled lifecycle restart or a future explicit adoption command/supervisor flow with stronger evidence.

Validation:

```bash
go test ./internal/hostsession ./internal/scenarioruntime -run 'Test.*Reconcile|Test.*HostSession|Test.*Boot|Test.*Stale|Test.*Adopt' -count=1
# passed

go test ./internal/orchestrator -run 'Test.*Registry.*Reboot|Test.*Registry.*Strict|Test.*Registry.*Allowlist|Test.*ResolvePort|Test.*PreviousBoot' -count=1
# passed

go test ./internal/maintenance -run 'Test.*Registry.*Cleanup|Test.*Diagnose.*Registry|Test.*Stale|Test.*PreviousBoot' -count=1
# passed

go test ./internal/lifecycle -run 'Test.*RuntimeRegistry.*Session|Test.*RuntimeRegistry.*ProcessRef|TestRunnerStartDualWritesScenarioRuntimeRegistry' -count=1
# passed

go test ./internal/hostsession ./internal/scenarioruntime ./internal/orchestrator ./internal/lifecycle ./internal/ports ./internal/maintenance ./internal/app/scenario ./internal/cli/scenariocli -count=1
# passed

cd packages/api-core && go test ./... -count=1
# passed
```

Live non-invasive validation:

```bash
VROOLI_RUNTIME_REGISTRY=strict VROOLI_RUNTIME_REGISTRY_ALLOWLIST=workspace-sandbox vrooli scenario status workspace-sandbox --json
# passed; reports stopped/empty ports instead of stale registry running state

VROOLI_RUNTIME_REGISTRY=strict VROOLI_RUNTIME_REGISTRY_ALLOWLIST=workspace-sandbox vrooli scenario port workspace-sandbox WS_PORT
# passed; fails closed with no running runtime ports instead of returning stale 28836

VROOLI_RUNTIME_REGISTRY=prefer VROOLI_RUNTIME_REGISTRY_ALLOWLIST=workspace-sandbox vrooli scenario status workspace-sandbox --json
# passed; falls back to current legacy runtime while registry evidence is unverified

vrooli diagnose-port 28836 --json
# passed; reports the old registry claim as non-authoritative/unverified with recommendation to restart under registry-enabled lifecycle before strict discovery

vrooli locks --json
# passed; includes registry claim reconciliation fields
```

### Phase 8C: Registry Authority Refactor and Hardening

Goal: make the Phase 8B implementation structurally safer before any Phase 9 legacy-authority removal. This pass applies the architecture, responsibility-boundary, seam, temporal-flow, utility-unification, and test-quality lenses to the runtime registry authority path.

Required pre-read:

```bash
prompt-manager skill read screaming-architecture-audit boundary-of-responsibility-enforcement decision-boundary-extraction utils-unification change-axis-and-evolution-resilience-audit seam-discovery-and-enforcement test temporal-flow-audit
```

Refactor decisions:

- `internal/scenarioruntime` must own the difference between instance authority and claim authority.
- Registry readers may fetch reserved and bound claims for diagnostics, but only reconciled bound claims may become runtime ports.
- Process and listener evidence construction should use domain-owned helper seams so orchestrator and maintenance do not independently re-create evidence shape or map keys.
- `internal/orchestrator` remains a consumer of reconciled runtime state. It should not decide that an active instance plus active claim is enough to expose a port.
- `internal/maintenance` may expire stale non-authoritative state and report diagnostics, but it should consume the same reconciliation semantics as reads.

#### Phase 8C Completion Record

Completed on 2026-05-09.

Files changed:

- `internal/scenarioruntime/reconcile.go`
  - Added domain-owned evidence helper seams for process refs and listener claims.
  - Changed claim reconciliation so a runtime instance can be authoritative while a reserved claim remains non-authoritative.
  - Reserved claims now classify as stale/non-authoritative for discovery instead of being exposed as ports.
- `internal/scenarioruntime/types.go`
  - Added centralized active claim, discoverable claim, and stop-candidate instance status policy helpers.
  - Kept returned status slices immutable by callers.
- `internal/orchestrator/runtime_registry_read.go`
  - Removed duplicate process/listener evidence construction and routed host evidence through `internal/scenarioruntime`.
  - Registry-backed details still fetch reserved claims for diagnostics, but `authoritativeClaims` can now only pass claims approved by reconciliation.
- `internal/maintenance/runtime_registry.go`
  - Routed maintenance reconciliation through the same process/listener evidence helper seams as orchestrator.
- `internal/ports/ports.go`
  - Uses the runtime-domain active claim status policy when looking for existing runtime claims.
- `internal/lifecycle/runtime_registry.go`
  - Uses the runtime-domain stop-candidate instance status policy when collecting registry rows for stop cleanup.
- `internal/scenarioruntime/reconcile_test.go`
  - Added tests proving reserved claims are not authoritative runtime ports.
  - Added tests for the evidence helper seam behavior.
- `internal/scenarioruntime/mode_test.go`
  - Added immutability and classification coverage for active claim, discoverable claim, and stop-candidate instance status policies.
- `internal/orchestrator/orchestrator_test.go`
  - Added strict-mode coverage proving a running registry instance with only a reserved claim reports no runtime port and `scenario port` fails with `scenario_port_not_found`.

Validation:

```bash
go test ./internal/scenarioruntime -run 'Test.*Reconcile|TestRuntimeEvidence' -count=1
# passed

go test ./internal/orchestrator -run 'Test.*Registry|TestStrictRegistry|TestPreferRegistry|TestResolvePortUsesRegistry' -count=1
# passed

go test ./internal/maintenance -run 'Test.*Registry|TestCleanStaleLocks|TestDiagnose' -count=1
# passed

go test ./internal/hostsession ./internal/scenarioruntime ./internal/orchestrator ./internal/lifecycle ./internal/ports ./internal/maintenance ./internal/app/scenario ./internal/cli/scenariocli -count=1
# passed

cd packages/api-core && go test ./... -count=1
# passed
```

Remaining recommendations before Phase 9:

- Preserve the full 8B/8C/8D work in source control before Phase 9; the worktree contains unrelated changes and untracked files.
- Use the Phase 8D completion evidence in `docs/plans/runtime-supervisor-heartbeat-authority-implementation-plan.md` as the starting point for Phase 9.
- Do not remove legacy read authority until the current supervised allowlist soak remains stable for the operator's desired observation window.
- Consider a future follow-up that makes reconciliation classifications a typed enum throughout JSON/Go boundaries if more callers start switching on those values.

### Phase 9: Legacy Cleanup

Goal: remove old authority after the migration window proves stable.

Tasks:

- Remove read authority from legacy JSON process records.
- Remove read authority from plain `.port_<port>.lock` files.
- Keep a one-time cleanup/migration command if needed to prune old records and locks.
- Delete or simplify obsolete PID/proc-based status code.
- Keep OS process inspection under diagnostics and cleanup only.
- Update docs so operators learn registry claims/leases, not lock files, as the normal model.
- Remove migration flags after strict mode has run long enough without fallback.

Tests:

- Full Go package tests for touched packages.
- Scenario tests for autoheal, workspace-sandbox, agent-manager, and browser automation studio.
- Manual smoke: start, status, port, diagnose-port, stop, cleanup locks, cleanup orphans.

Exit criteria:

- No production discovery path depends on PID visibility or legacy lock files.
- Legacy cleanup is complete and documented.

## Testing Plan

Unit test priorities:

- Registry repository transactions, lease expiry, generation safety, and claim uniqueness.
- Port allocation race handling with concurrent goroutines and temp DBs.
- Health snapshot parsing and timeout/error storage.
- Orchestrator read semantics for active, expired, failed, degraded, and missing registry states.
- Maintenance diagnostics for registry claim, legacy lock, listener, and orphan combinations.
- Autoheal Vrooli state reader seam with registry-backed fixtures.
- BAS scenarioport seam without `SCENARIO_REGISTRY`.

Integration test priorities:

- Start/stop dual-write compatibility.
- Registry-first `scenario port` works without PID liveness.
- Expired claim cleanup does not remove active claims.
- Orphan listener is diagnosed separately from active runtime claim.
- Health failure does not block basic port discovery when runtime lease is active, but is visible in status.

Validation commands, adjusted as packages evolve:

```bash
go test ./internal/scenarioruntime
go test ./internal/process ./internal/ports ./internal/lifecycle ./internal/orchestrator ./internal/maintenance ./internal/api
go test ./internal/cli/scenariocli ./internal/cli/projectcli ./internal/app/project
cd packages/api-core && go test ./...
cd scenarios/browser-automation-studio/api && go test ./internal/scenarioport
cd scenarios/vrooli-autoheal/api && go test ./internal/checks/... ./internal/healing/...
vrooli scenario test workspace-sandbox
vrooli scenario test agent-manager
vrooli scenario test browser-automation-studio
vrooli scenario test vrooli-autoheal
```

Do not run scenario binaries or dev scripts directly. Use scenario Makefiles or `vrooli scenario ...`.

## Phase 8D Completion Record

Completed 2026-05-09 by Codex.

Implemented the dedicated runtime supervisor plan in `docs/plans/runtime-supervisor-heartbeat-authority-implementation-plan.md`:

- Greenfield runtime registry schema v3 and supervisor session/domain APIs.
- Central `internal/runtimesupervisor` service with lifecycle auto/on adoption, session heartbeat, reconciliation-gated lease renewal, stale supervised strict-mode enforcement, and supervisor-aware diagnostics.
- Bounded health probe execution with `VROOLI_RUNTIME_SUPERVISOR_MAX_HEALTH_CONCURRENCY` and health snapshot cadence based on health snapshot timestamps, not lease timestamps.
- Linux systemd user-service install/uninstall through `vrooli runtime supervisor install --user` and `vrooli runtime supervisor uninstall --user`.
- Recovery path for expired prior-supervisor rows: strict reads fail closed while stale, but a live supervisor can revalidate process/listener evidence, take over supervision, and renew.

Validation completed:

```bash
go test ./internal/runtimesupervisor ./internal/scenarioruntime ./internal/cli/vroolicli ./internal/cli/topcli -count=1
go test ./internal/runtimesupervisor ./internal/scenarioruntime ./internal/orchestrator ./internal/lifecycle ./internal/maintenance ./internal/app/scenario ./internal/app/project ./internal/cli/projectcli ./internal/cli/scenariocli ./internal/cli/topcli ./internal/cli/vroolicli -count=1
cd packages/api-core && go test ./... -count=1
go run ./cmd/vrooli --no-stale-check locks --json | jq '[.registry_claims[] | select(.authoritative==true and .lease_fresh==false)] | length'
```

Live allowlist soak:

- `workspace-sandbox`, `agent-manager`, and `swarm-manager` were stopped and restarted through lifecycle commands with registry writes and `VROOLI_RUNTIME_SUPERVISOR=auto`.
- Strict registry status resolved for all three scenarios.
- `vrooli locks --json` reported zero authoritative registry claims with `lease_fresh=false`.
- API/UI claims for the three soak scenarios showed fresh supervised leases, fresh supervisor sessions, healthy snapshots, and `verified_running` reconciliation.

Known caveat before Phase 9: some old expired/stopped registry rows remain visible as non-authoritative diagnostics. They are not discovery authority and can be handled by the existing cleanup path or Phase 9 legacy cleanup.

## Rollout and Validation Checklist

- Registry schema can be created on a clean machine.
- Registry schema can be opened repeatedly without corrupting data.
- Dual-write does not break existing lifecycle commands.
- Registry data agrees with legacy records during migration.
- Registry-first reads can be enabled behind a migration flag.
- Registry-first reads can be disabled quickly if needed.
- Autoheal can diagnose both legacy and registry state during migration.
- `diagnose-port` explains conflicts using registry, socket, process, health, and port policy signals.
- Sandbox agents resolve ports through the same discovery path as host agents.
- Legacy cleanup is intentionally scheduled and not forgotten.

## Risks and Mitigations

- Risk: a stale registry claim blocks a valid start.
  - Mitigation: TTL, generation checks, explicit expired-claim cleanup, and socket bind probe.
- Risk: health endpoint failure is misread as process death.
  - Mitigation: separate lease status from health status.
- Risk: orphan cleanup kills unrelated or protected processes.
  - Mitigation: keep conservative process classification, registry-aware revalidation, and protected executable lists.
- Risk: dual-write creates split-brain state.
  - Mitigation: registry events, reconciliation diagnostics, migration flags, and compensation paths.
- Risk: SQLite contention during concurrent starts.
  - Mitigation: short transactions, busy timeout, unique active claim constraints, and claim acquisition tests.
- Risk: sandbox networking is confused with discovery.
  - Mitigation: explicitly test registry lookup separately from TCP reachability.
- Risk: autoheal duplicates core cleanup logic.
  - Mitigation: keep autoheal using core CLI/API cleanup commands; do not reimplement low-level policy in the scenario.

## Non-Goals and Prohibited Patterns

- No per-scenario discovery environment variables.
- No `SCENARIO_REGISTRY` escape hatch for browser automation studio.
- No direct scenario process execution in tests or docs.
- No new PID/proc dependency for authoritative status.
- No string parsing of SQLite data where a structured repository method is appropriate.
- No broad unrelated refactors while touching lifecycle and port code.
- No final state where legacy locks remain the source of truth.

## Definition of Done

This overhaul is done when:

- Scenario runtime status and port discovery work without host PID visibility.
- `vrooli scenario port` reads active registry claims and no longer requires `/proc`.
- Port allocation is atomic, race-safe, and registry-backed.
- Health/readiness is visible but separate from runtime lease liveness.
- Orphan and stale-state handling remains available as diagnostics and cleanup, not normal discovery.
- Autoheal is registry-aware and still delegates cleanup to core Vrooli commands.
- Browser automation studio uses API-core discovery like other scenarios.
- Sandbox agent-manager workflows can resolve running scenario ports by default.
- Legacy JSON process records and plain lock files are no longer production authority.
- Tests cover race conditions, stale claims, orphan listeners, health failures, sandbox discovery, and migration fallback.
