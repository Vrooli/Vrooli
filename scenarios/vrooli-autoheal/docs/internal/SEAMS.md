# Seams and Responsibility Boundaries

This document describes the architectural seams and responsibility boundaries in the vrooli-autoheal scenario. Understanding these boundaries helps maintainers know where to add or modify behavior.

## Overview

vrooli-autoheal follows a layered architecture with clear responsibility separation:

```
┌─────────────────────────────────────────────────────────────────┐
│                     Entry / Presentation                        │
│  main.go (server lifecycle) │ handlers/*.go (HTTP) │ UI (React)│
├─────────────────────────────────────────────────────────────────┤
│                     Orchestration / Wiring                      │
│              bootstrap/checks.go (check registration)           │
├─────────────────────────────────────────────────────────────────┤
│                        Domain Rules                             │
│  checks/types.go (interfaces, status logic) │ checks/registry.go│
├─────────────────────────────────────────────────────────────────┤
│                   Domain Implementations                        │
│     checks/infra/*.go (infrastructure) │ checks/vrooli/*.go     │
├─────────────────────────────────────────────────────────────────┤
│                   Integrations / Infrastructure                 │
│  persistence/store.go │ platform/platform.go │ config/config.go │
└─────────────────────────────────────────────────────────────────┘
```

---

## API Responsibility Zones

### 1. Entry / Presentation Layer

**Location:** `api/main.go`, `api/internal/handlers/`

**Responsibility:**
- HTTP server lifecycle (startup, graceful shutdown)
- Route registration and middleware
- Request/response serialization (JSON encoding)
- Lifecycle enforcement (reject direct execution)

**Boundaries:**
- Does NOT contain business logic or domain rules
- Does NOT directly access storage (delegates to Store)
- Does NOT create health check instances (delegates to bootstrap)

**Where to add:**
- New HTTP endpoints → `handlers/handlers.go`
- New middleware → `main.go:setupRouter()`
- Server configuration → `main.go:startServer()`

### 2. Orchestration / Wiring Layer

**Location:** `api/internal/bootstrap/`

**Responsibility:**
- Deciding which checks get registered at startup
- Providing operational defaults (target hosts, domains)
- Wiring platform capabilities to checks that need them

**Boundaries:**
- Does NOT implement check logic (delegates to check packages)
- Does NOT handle HTTP (that's the handlers' job)
- Owns operational constants (DefaultNetworkTarget, DefaultDNSDomain)

**Where to add:**
- New default checks → `bootstrap/checks.go:RegisterDefaultChecks()`
- Operational defaults → constants in `bootstrap/checks.go`

### 3. Domain Rules Layer

**Location:** `api/internal/checks/types.go`, `api/internal/checks/registry.go`

**Responsibility:**
- Core types: `Status`, `Result`, `Check` interface, `Summary`
- Pure domain logic: `WorstStatus()`, `AggregateStatus()`, `ComputeSummary()`
- Registry behavior: registration, interval filtering, platform filtering

**Boundaries:**
- Contains NO I/O operations
- Does NOT depend on persistence or HTTP
- Platform capabilities are injected, not detected internally

**Where to add:**
- New status types or result fields → `types.go`
- Status calculation logic → `types.go`
- Registry behavior changes → `registry.go`

### 4. Domain Implementations (Health Checks)

**Location:** `api/internal/checks/infra/`, `api/internal/checks/vrooli/`

**Responsibility:**
- Implementing specific health checks (network, DNS, Docker, etc.)
- Defining check metadata (ID, description, interval, platform filter)
- Running the actual check and returning results

**Boundaries:**
- Each check is self-contained with a single responsibility
- Checks receive dependencies via constructor injection (platform caps)
- Checks do NOT embed operational defaults (those come from bootstrap)

**Where to add:**
- New infrastructure checks → `checks/infra/`
- New Vrooli-specific checks → `checks/vrooli/`
- Platform-specific behavior → inject `*platform.Capabilities`

### 5. Integrations / Infrastructure Layer

**Location:** `api/internal/persistence/`, `api/internal/platform/`, `api/internal/config/`

**Responsibility:**
- `persistence/store.go`: Database CRUD for health results
- `platform/platform.go`: OS/capability detection (cached)
- `config/config.go`: Environment variable loading

**Boundaries:**
- These are adapters to external systems
- Domain logic does NOT live here
- Platform detection is cached; detection logic is isolated

**Where to add:**
- Database operations → `persistence/store.go`
- New platform capabilities → `platform/platform.go`
- Environment configuration → `config/config.go`

---

## UI Responsibility Zones

### 1. Entry / Presentation

**Location:** `ui/src/main.tsx`, `ui/src/App.tsx`, `ui/src/components/`

**Responsibility:**
- React component rendering
- User interaction handling
- Visual state (loading, error, success)

**Boundaries:**
- Does NOT contain business logic
- Data fetching delegated to api client

### 2. API Integration

**Location:** `ui/src/lib/api.ts`

**Responsibility:**
- HTTP client wrapper
- Type definitions matching API responses
- Request/response handling

**Boundaries:**
- Does NOT render UI
- Does NOT contain React hooks (just async functions)

### 3. Cross-Cutting

**Location:** `ui/src/consts/selectors.ts`

**Responsibility:**
- Test selector constants for UI testing

---

## Key Design Decisions

### `vrooli` CLI Invocation Seam — `repo-contract-go/cliinvoke`

`github.com/vrooli/repo-contract-go/cliinvoke` is the only place that
resolves, runs and classifies a `vrooli` subprocess, for the whole repository.
Inside autoheal, `api/internal/checks/executor.go` (`RealExecutor`) routes
every `name == "vrooli"` invocation through it, and the loop's `invokeVrooli`
wrapper does the same; the argv comes from the `cliinvoke` catalog
(`ScenarioLifecycle`, `ScenarioStatusJSON`, `ScenarioPort`, `AgentRecover`,
`Setup`, `SetupStatusReadiness`, `DiagnosePort`, `VersionJSON`).

Why: the CLI's argv surface is a contract every supervisor consumes. On
2026-09-02 a retired global flag left every supervisor unable to call the CLI
after a reboot. The seam passes no global flags, resolves the binary one way
(explicit path, `VROOLI_BIN`, the runtime home's bin entry, `PATH`), applies
the deadline and `WaitDelay` discipline from the 2026-08-01 inherited-pipe
outage, and classifies failures (`usage`, `binary-missing`, `timeout`,
`refusal`, `lifecycle`) so a caller never retries a usage error identically.

How to apply: build the argv with a catalog function and run it through
`cliinvoke.Run` (or the `CommandExecutor` interface inside the API). Register
new argv producers in `internal/cli/rootcli/invokers`; the conformance test in
`cliinvoke` fails on any direct `exec.Command(..., "vrooli", ...)` under
`scenarios/vrooli-autoheal/`, and the registry test fails when a registered
argv no longer parses or relies on the retired-globals tolerance table.

Reference: `docs/reference/cli-invokers.md` (generated registry page) and
`docs/reference/staleness-and-rebuild.md` (why no stale-check flag exists).

### Loop Freshness Seam — the lifecycle engine builds, the safeguard verifies

The boot-recovery loop (`cli/loop`, a nested Go module) is declared as the
`loop` component in `.vrooli/service.json` (`build.kind: go_module`,
`build.dir: cli/loop`, `build.output: cli/vrooli-autoheal-loop`). `vrooli
scenario setup vrooli-autoheal` builds it with the same engine that builds the
API and stamps `cli/vrooli-autoheal-loop.freshness.json` next to it; the
component's `go list` closure (the loop module, `langrecover`,
`packages/repo-contract-go`, `packages/envkit-go`) is the recorded input set.

`internal/safeguards/autoheal-watchdog` (project control plane) never builds
the loop. `loopBinaryVerdict` reads that manifest through the engine's public
reader (`cli-core/cliutil.ReadFreshnessManifest` + `EvaluateFreshness`) and
reports `fresh`, `stale` (manifest disagrees with the sources, or the binary is
missing) or `unknown` (no manifest: the engine never built this binary). Stale
and unknown are both not-applied; `Apply` then runs `vrooli scenario setup
vrooli-autoheal` through `repo-contract-go/cliinvoke` and restarts the unit
when the binary's sha256 changed. `Inspect` also compares the unit's main
process executable (`/proc/<MainPID>/exe` on Linux) with the binary on disk
and reports "process older than binary" until the unit is restarted; evidence
lands in `loop_freshness` and `process_identity` of `vrooli setup status
--json --phase readiness`.

Why: the safeguard used to rebuild from an mtime walk and its own `go build`,
a second freshness engine that reported "Already present" over a binary months
older than its source (2026-09-01), and a rebuilt file never reached the
running process without a restart. One engine, one manifest, one verdict.

Declaration note: the engine skips building a component whose `run.condition`
is unmet and launches every component that has no `supervised_by`, so the loop
is declared `supervised_by: api` to be built-but-not-launched. The API does
not supervise it; the native scheduler unit does. A build-only component kind
is the missing contract (recorded in the boot-recovery plan's p09 evidence).

How to apply: change loop sources, run `make loop-build` (which is `vrooli
scenario setup vrooli-autoheal`), never `go build` the loop by hand; the next
`vrooli setup` restarts the unit onto the new binary.

### Recovery Floor Command Seam — `selfHealRunner`

`cli/loop/selfheal.go` routes every command the dependency-drift recovery
floor executes (`go mod tidy`, `pnpm install`, the recovery scripts in
`langrecover`) through `selfHealRunner`, a `langrecover.Runner`. Production
uses `langrecover.DefaultRunner`; tests inject a recording runner and assert
the exact argv, working directory and ordering the floor would run, without
touching a real module. Recovery decisions (`decideFromSources`) are pure over
the failure output and the signatures, so a wrong recovery is a unit-test failure,
not a host mutation.

### System Event Timeline Collection Seam

`api/internal/systemevents/` owns host-event normalization for the forensic
Timeline surface. It is the only place that should parse OS package logs,
kernel journal lines, boot history, or platform-specific event sources for the
system-event timeline.

Production collection goes through injected `checks.CommandExecutor`,
`journal.Reader`, and file/glob functions. Tests should exercise parser and
collector behavior with injected content rather than reading host log paths,
`journalctl`, or platform event logs directly.

The `/api/v1/timeline` endpoint remains the health-check-result timeline. New
host-level event work belongs behind `/api/v1/system-events` and the
`system_events` SQLite table so it can dedupe repeated ingestion and survive
process restarts.

### Watchdog Detection System Probe Seam

`api/internal/watchdog/watchdog.go` now routes environment/runtime interactions through a dedicated `detectorProbe` boundary. This seam isolates:

- OS and platform branching (`GOOS`)
- command execution (`tasklist`, `pgrep`, `systemctl`, `launchctl`, `schtasks`, `loginctl`)
- process inspection fallback (`/proc` directory and cmdline reads)
- file existence checks for service artifacts
- user, home-directory, and environment lookups used by detection and template rendering

Production uses `realDetectorProbe`; tests can inject a fake probe to exercise watchdog detection without real process tables, systemd state, launchctl, or Windows task scheduler.

This avoids environment-coupled tests and keeps watchdog status logic testable as pure decision flow over probe outputs.

### Cleanup Manager Handoff Seam

Broad disk reclaim is delegated to storage-manager. Autoheal may observe disk
pressure, surface diagnostics, start stopped services, and gather logs, but it
must not execute host cleanup such as `docker system prune`, journal vacuum,
package-cache cleanup, or arbitrary file deletion as a recovery action.

The Docker check keeps a `prune` recovery action only as a compatibility
discovery surface. Executing it returns a storage-manager planning hint
(`storage-manager cleanup plan --json`) and performs no command invocation. The
actual cleanup flow is: storage-manager estimates candidates, records a plan,
requires policy/provider-version/approval gates, and audits any apply attempt.

Tests should protect this seam by asserting broad-cleanup handoff actions make
zero `CommandExecutor` calls.

### Watchdog Compatibility Seam (Read-Only)

`api/internal/watchdog/installer.go` retains the historical install, uninstall,
and lingering method signatures for API compatibility, but those methods now
return setup guidance and perform no host mutation. Native scheduler rendering,
installation, enablement, and boot-policy changes are owned by
`internal/safeguards/autoheal-watchdog` in the project control plane.

The scenario watchdog detector remains an observation surface: it reports
native service state and can render a diagnostic template, but it has no file,
process, scheduler, or privilege-mutation seam.

### User Config Filesystem + Home-Dir Seam

`api/internal/userconfig/manager.go` now routes filesystem and home-directory behavior through a dedicated `configIO` seam (`api/internal/userconfig/config_io.go`):

- config file presence checks (`Stat`)
- config and schema reads (`ReadFile`)
- directory creation and atomic save writes (`MkdirAll`, `WriteFile`, `Rename`, `Remove`)
- default config path home-directory lookup (`UserHomeDir`)

Production uses `realConfigIO`. Tests can inject fakes to exercise failure paths and fallback behavior without touching host filesystems or user home directories.

This keeps config merge/validation logic focused on domain behavior while isolating OS-dependent side effects behind one explicit boundary.

### Dependency Injection for Platform Capabilities

**Before (leaked responsibility):**
```go
func (c *CloudflaredCheck) Run(ctx context.Context) checks.Result {
    caps := platform.Detect() // Hidden dependency!
    ...
}
```

**After (injected dependency):**
```go
func NewCloudflaredCheck(caps *platform.Capabilities) *CloudflaredCheck {
    return &CloudflaredCheck{caps: caps}
}
```

This change:
- Makes dependencies explicit and testable
- Allows testing with mock platform capabilities
- Removes hidden coupling to global state

### Operational Defaults in Bootstrap Layer

**Before (embedded defaults):**
```go
func NewNetworkCheck(target string) *NetworkCheck {
    if target == "" {
        target = "8.8.8.8:53" // Embedded default
    }
    ...
}
```

**After (explicit in bootstrap):**
```go
// In bootstrap/checks.go
const DefaultNetworkTarget = "8.8.8.8:53"

func RegisterDefaultChecks(registry *checks.Registry, caps *platform.Capabilities) {
    registry.Register(infra.NewNetworkCheck(DefaultNetworkTarget))
}
```

This change:
- Keeps check implementations pure
- Centralizes operational configuration
- Makes defaults visible and changeable in one place

### Remaining Weak Points

- Some checks still rely on wall-clock `time.Now()` internally instead of injected clocks; those paths are testable but less deterministic for edge timing cases.
- Some Vrooli check action flows (for example parts of `api/internal/checks/vrooli/api.go`) still perform direct environment, home-directory, and filesystem lookups in action logic instead of a single injected boundary.

---

## Adding New Features

### Adding a New Health Check

Follow `docs/guides/adding-checks.md`; the steps are:

1. Add the check file under `api/internal/checks/<category>/` with its desired-behavior tests.
2. Add it to the appropriate `DefaultCheckFactory` slice in `api/internal/bootstrap/`.
3. If it heals, add its action to the allowlists in `checks/registry.go`. A check that only reports (for example `system-boot-recovery-readiness`) declares no actions.
4. Add or update the corresponding space-doc cell.
5. Add a setpoint bar with a unit and threshold.
6. Add the check to `docs/reference/check-catalog.md`, write its `docs/reference/checks/<id>.md` page, and register that page in `docs/manifest.json`.

### Adding a New API Endpoint

1. Add handler method in `handlers/handlers.go`
2. Register route in `main.go:setupRouter()`
3. Add corresponding client function in `ui/src/lib/api.ts` if needed

### Adding Platform Detection

1. Add field to `platform.Capabilities` struct
2. Add detection function in `platform/platform.go`
3. Call detection in `detect()` function
4. Update tests in `platform_test.go`

---

## Testing Boundaries

| Layer | Test Location | What to Test |
|-------|---------------|--------------|
| Handlers | `handlers/` (future) | HTTP status codes, JSON structure |
| Registry | `checks/registry_test.go` | Registration, filtering, execution |
| Types | `checks/types_test.go` | Status aggregation, summary computation |
| Checks | `checks/infra/*_test.go` | Check interface compliance, result structure |
| Platform | `platform/platform_test.go` | Detection accuracy, caching |
| UI | `ui/src/*.test.tsx` | Component rendering, user interaction |

### Typed readiness and healing-episode read

| | |
|---|---|
| **Seam** | Condition/coverage consumers ↔ autoheal startup and recovery evidence |
| **Module** | `packages/proto/schemas/vrooli-autoheal/v1/healing/healing.proto`, implemented by `handlers.typedHealing.GetReadiness` |
| **Production wiring** | Autoheal derives first healthy probes after process start and joins persisted failing probes, recovery actions, and subsequent healthy probes. It reports an explicit unknown starter when supervisor attribution is unavailable. Infrastructure-manager reads the generated Connect client through `sources.AutohealReader`. |
| **Test fake** | Handler tests substitute `StoreInterface` and provide persisted check/action timestamps; source tests substitute discovery and HTTP clients. |
| **Why it exists** | A3 and R3 require measured timestamps, not a hand-written coverage claim or action duration reused as episode duration. |
