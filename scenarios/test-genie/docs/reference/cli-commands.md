# Test Genie CLI Reference

Complete reference for the `test-genie` command-line interface. Commands are designed for both interactive use and CI/CD scripting.

## Installation

```bash
# Install globally (recommended)
cd scenarios/test-genie/cli
./install.sh

# Or use directly without installation
./cli/test-genie [command]

# Verify installation
test-genie --version
```

## Global Options

These options apply to all commands:

| Option | Short | Description |
|--------|-------|-------------|
| `--help` | `-h` | Show help for any command |
| `--verbose` | `-v` | Enable verbose output |
| `--json` | | Output in JSON format (scriptable) |
| `--timeout <seconds>` | `-t` | Override default timeout |
| `--quiet` | `-q` | Suppress non-essential output |

## Commands Overview

| Command | Description |
|---------|-------------|
| `generate` | Generate test suites for a scenario |
| `execute` | Execute a test suite |
| `run-tests` | Trigger the scenario-local phased runner |
| `ui-smoke` | Run the shared UI smoke harness |
| `registry` | Manage playbook registries |
| `requirements` | Inspect and sync requirements evidence |
| `provider-contract` | Validate provider-backed maturity assessment contracts |
| `playbooks-seed` | Apply or clean up playbooks seed data |
| `storage` | Run storage maintenance tasks |
| `status` | Check test-genie operational status |
| `health` | Show Test Genie self-health: catalog, provider conformance, reliability ledger |
| `fleet` | Fleet-wide health aggregated over stored runs (`fleet status`) |

---

## generate

Generate test suites for a scenario. Tests are created based on code analysis, PRD, and requirements.

```bash
test-genie generate <scenario-name> [options]
```

### Options

| Option | Default | Description |
|--------|---------|-------------|
| `--types <types>` | `unit` | Comma-separated test types |
| `--coverage <percent>` | `90` | Target coverage percentage (1-100) |
| `--parallel` | `false` | Enable parallel generation |
| `--dry-run` | `false` | Preview without executing |
| `--priority <level>` | `normal` | Queue priority |
| `--output <dir>` | auto | Output directory for generated tests |
| `--fallback` | `false` | Use local templates instead of delegation |

### Test Types

| Type | Description |
|------|-------------|
| `unit` | Unit tests for functions/methods |
| `integration` | Integration tests for component interaction |
| `performance` | Performance benchmarks |
| `vault` | Lifecycle validation tests |
| `regression` | Regression test suites |

### Priority Levels

| Priority | Description |
|----------|-------------|
| `low` | Background processing |
| `normal` | Standard queue position |
| `high` | Prioritized processing |
| `urgent` | Immediate processing |

### Examples

```bash
# Generate unit tests (default)
test-genie generate my-scenario

# Generate unit and integration tests
test-genie generate my-scenario --types unit,integration

# Comprehensive suite with high coverage target
test-genie generate my-scenario --types unit,integration,performance --coverage 95

# Dry run to preview
test-genie generate my-scenario --types unit --dry-run

# High priority generation
test-genie generate my-scenario --types unit --priority high

# Use local templates (fallback mode)
test-genie generate my-scenario --types unit --fallback
```

### Output

```
Analyzing scenario: my-scenario
  Found 12 requirements in requirements/*.json
  Identified coverage gaps: 8 untested functions
  Delegated generation to App Issue Tracker
    Issue ID: #12345
    Estimated completion: 15-30 minutes

Track progress:
  vrooli issue status 12345

View generated tests:
  git status  # After issue completion
```

---

## run-tests

Trigger the scenario-local runner through the Test Genie API.

```bash
test-genie run-tests <scenario-name> [options]
```

### Options

| Option | Default | Description |
|--------|---------|-------------|
| `--type <runner>` | `phased` | Scenario-local runner type |
| `--path <path>` | none | Restrict to one or more files/directories |
| `--playbook <path>` | none | Restrict playbooks to one or more workflow files |
| `--filter <text>` | none | Forward a text filter to the runner |
| `--json` | `false` | Output machine-readable JSON |

### Examples

```bash
test-genie run-tests my-scenario
test-genie run-tests my-scenario --path api --filter UserService
test-genie run-tests my-scenario --playbook bas/cases/smoke/dashboard-loads.json
```

---

## UI Runtime Validation

Runtime UI validation is owned by the `ui-health` phase. It embeds the scenario
UI in a host iframe shell through **Browser Automation Studio (BAS)**, waits for
the iframe-bridge handshake, captures browser artifacts, and delegates generic
visual judgment to ui-health.

```bash
test-genie execute <scenario-name> --phases ui-health
ui-health validate scenario <scenario-name> --json
```

### Examples

```bash
test-genie ui-smoke my-scenario
test-genie ui-smoke my-scenario --json
```

---

## registry

Manage `bas/registry.json` generation and validation.

```bash
test-genie registry --help
```

---

## requirements

Inspect, validate, and sync requirement evidence without the legacy bash tooling.

```bash
test-genie requirements --help
```

---

## provider-contract

Restart a provider through Vrooli lifecycle and validate that its programmatic output includes a valid shared maturity assessment.

```bash
test-genie provider-contract check <phase|provider> <scenario-name> [--json]
```

Use `--no-restart` only for diagnostics when lifecycle freshness is already guaranteed.

Supported probes include CLI-backed providers (`cli-health`, `ui-health`, `quality-health`, `scenario-dependency-analyzer`, `security-health`, `measures-health`, `proto-health`, `scenario-auditor`, `architecture-cartographer`), the `knowledge-observatory` DocHealth RPC wrapper, and the `tidiness-manager` HTTP tidiness scan endpoint discovered through lifecycle.

### Examples

```bash
test-genie provider-contract check contracts my-scenario
test-genie provider-contract check cli-health my-scenario --json
test-genie provider-contract check standards my-scenario
test-genie provider-contract check architecture my-scenario --json
test-genie provider-contract check docs my-scenario
test-genie provider-contract check tidiness-manager my-scenario --json
```

### scan

Sweep every delegated provider phase in the catalog and score how correctly each one adopts the shared contract. Unlike `check` (one provider, restart-and-probe), `scan` is a read-only fleet probe.

```bash
test-genie provider-contract scan [--json] [--target <fixture-scenario>] [--timeout <dur>]
```

Per provider it reports `reachable`, `contract_valid`, `identity_ok`, `spec_valid`, `metrics_adopted` (advisory), a `violations[]` list, and an `adoption_score`. The default target is `test-genie` with `include_execution=false` to bound probe cost. The command exits non-zero only for a mis-specified provider (`spec_valid=false`) or a contract/identity break among reachable providers; unreachability and missing metrics never fail the gate.

```bash
test-genie provider-contract scan --json
```

---

## playbooks-seed

Apply or clean up `bas/__seeds` lifecycle data for playbook execution.

```bash
test-genie playbooks-seed --help
```

---

## storage

Run one-time operational storage maintenance tasks.

```bash
test-genie storage --help
```

---

## execute

Execute a test suite for a scenario. Supports presets for common configurations.

```bash
test-genie execute <scenario-name> [options]
```

### Durable, cancel-survivable runs

The run is owned by the **test-genie server**, not this command, so it is
decoupled from the request/process lifecycle:

- The run id and a re-attach command are printed **up front** (first lines).
- If this command is interrupted (Ctrl-C, a tool timeout, a dropped
  connection), the run **keeps going** — interrupting only detaches the viewer.
  It does **not** abort the run. To actually stop a run, use `runs abort`.
- A run whose estimated duration is at or above the auto-background threshold
  (`TEST_GENIE_AUTOBACKGROUND_SECONDS`, default 60s; `0` disables) is launched
  in the background so your shell returns immediately, with a re-attach command.
  Unknown-ETA runs also background by default because first runs can be long
  (`TEST_GENIE_AUTOBACKGROUND_ON_UNKNOWN_ETA=0` disables that behavior).
  Shorter known-ETA runs are followed inline (still cancel-survivable).
- `--wait` always blocks to completion inline regardless of ETA (used by CI and
  callers that need the real exit code inline).
- Background and busy-run output includes an **Agent wait protocol** block with
  the exact quiet `runs wait --json --timeout=<seconds>` command, the expected
  duration, the recommended wait timeout, and interruption recovery commands.

Re-attach / inspect a run by handle with the `runs` verbs below
(`test-genie runs wait <scenario> <run-id>`, etc.).

### Options

| Option | Default | Description |
|--------|---------|-------------|
| `--preset <name>` | `comprehensive` | Preset configuration |
| `--phases <phases>` | all | Comma-separated phases to run |
| `--skip <phases>` | none | Phases to skip |
| `--fail-fast` | `false` | Stop on first failure |
| `--wait` | `false` | Block to completion inline; never auto-background (CI / scripted use) |
| `--json` | `false` | Block to verdict and emit the full result blob |
| `--jsonl` | `false` | Stream canonical newline-delimited phase events (run_started…run_completed) for TUIs |
| `--request-id <uuid>` | | Link to queued suite request |
| `--scenario-path <path>` | resolved from scenario name | Absolute physical scenario directory to read and write |
| `--logical-repo-root <path>` | none | Absolute repo root to use for repo-relative validation |
| `--logical-scenario-relpath <path>` | none | Scenario directory relative to `--logical-repo-root` |

`--scenario-path` and the logical placement flags intentionally describe
different concepts. Use only `--scenario-path` when the scenario truly lives at
that external path. Add `--logical-repo-root` and
`--logical-scenario-relpath` when the physical scenario is an ephemeral copy
that should be validated as if it lived under a repo, such as template deep
validation.

### Presets

| Preset | Scope | Use Case |
|--------|-------|----------|
| `quick` | Fast structural and unit-oriented feedback | Local iteration |
| `smoke` | Lightweight validation before broader suites | Pre-push / pre-commit |
| `comprehensive` | Full enabled phase plan for the scenario | CI, release, audit runs |

### Phase Names

| Phase | Description |
|-------|-------------|
| `structure` | Validates scenario layout, manifests, and JSON health |
| `standards` | Runs scenario-auditor standards enforcement |
| `dependencies` | Delegates dependency health validation to scenario-dependency-analyzer |
| `quality` | Delegates static quality contracts to quality-health |
| `docs` | Validates markdown, mermaid, and links |
| `smoke` | Performs fast runtime / UI handshake checks |
| `unit` | Executes language-specific unit tests |
| `integration` | Runs CLI runtime, API health, and WebSocket connectivity checks |
| `playbooks` | Executes BAS browser automation workflows |
| `business` | Audits requirement coverage and business validation |
| `performance` | Builds binaries and checks performance budgets (optional) |

### Timing Preview

Before execution starts, the CLI asks the API to resolve the real phase plan for the scenario and prints:

- `Estimate` — scenario-aware runtime guidance based on recent per-phase history
- `Timeout budget` — the configured runtime ceiling after scenario overrides are applied

Estimate selection is phase-based:

- Scenario + phase history first when enough recent runs exist
- Blended scenario/global phase history when local evidence is sparse
- Global phase history as the fallback
- Timeout budget only when no useful history exists

This means the CLI no longer treats timeout budgets as runtime estimates.

### Examples

```bash
# Run comprehensive tests (default)
test-genie execute my-scenario

# Quick smoke test
test-genie execute my-scenario --preset quick

# Specific preset
test-genie execute my-scenario --preset smoke

# Fail fast for CI
test-genie execute my-scenario --preset smoke --fail-fast

# Run specific phases
test-genie execute my-scenario --phases structure,dependencies,unit

# Skip optional phase
test-genie execute my-scenario --skip performance

# Live monitoring
test-genie execute my-scenario --preset comprehensive --watch

# Link to suite request
test-genie execute my-scenario --request-id 550e8400-e29b-41d4-a716-446655440000

# Test a generated temp scenario while resolving repo-relative docs from Vrooli
test-genie execute template-validation-react-vite-deep \
  --scenario-path /tmp/vrooli-template-deep-123/scenarios/template-validation-react-vite-deep \
  --logical-repo-root /home/matthalloran8/Vrooli \
  --logical-scenario-relpath scenarios/template-validation-react-vite-deep \
  --preset quick

# Trigger requirements sync
test-genie execute my-scenario --preset comprehensive --sync
```

### Output

```
Executing test suite for: my-scenario
  Preset: comprehensive
  Estimate: 3m 12s
  Timeout budget: 29m 0s
  Planned phases: structure, standards, dependencies, quality, docs, ui-health, unit, integration, workflow, business, performance

Plan:
  structure     estimate 4s    timeout 15m  high confidence from 12 runs
  workflow      estimate 15m   timeout 15m  timeout fallback

[1/11] structure     PASSED  (5s)
[2/11] standards     PASSED  (2s)
[3/11] dependencies  PASSED  (12s)
[4/11] quality       PASSED  (9s)
[5/11] docs          PASSED  (3s)
[6/11] ui-health     PASSED  (6s)
[7/11] unit          PASSED  (45s)
[8/11] integration   PASSED  (30s)
[9/11] workflow      PASSED  (14s)
[10/11] business     PASSED  (8s)
[11/11] performance  PASSED  (20s)

Summary:
  Status: PASSED
  Duration: 2m 14s
  Estimate: 3m 12s
  Timeout budget: 29m 0s
  Phases: 11 passed, 0 failed
  Execution ID: 660e8400-e29b-41d4-a716-446655440001

Log: /tmp/test-genie/logs/my-scenario-comprehensive.log
```

---

## runs

Inspect and control the durable, server-owned run history for a scenario. The
append-only history verbs (`list`/`show`/`delete`/`pin`/`unpin`/`compare`/
`freshness`) are documented under [run history](#history); the verbs below
operate on a single run **by handle** and are the anti-polling re-attach
surface.

```bash
test-genie runs <verb> <scenario> <run-id> [options]
```

| Verb | Description |
|------|-------------|
| `wait <scenario> <run-id> [--timeout <seconds>] [--json]` | Block until the run is terminal. Exit `0` passed, `1` failed/aborted, `124` if `--timeout` elapses first. |
| `follow <scenario> <run-id>` | Stream the run's canonical events to completion (replays history for a finished run). |
| `status <scenario> <run-id> [--json]` | Live snapshot: status, active phase, elapsed, estimated remaining, and `recommended_next_check_seconds` (the backoff to use if you must poll). |
| `abort <scenario> <run-id> [--json]` | Cancel a running run (transitions it to `aborted`). This is the **only** way to destroy a run — interrupting `execute`/`wait`/`follow` merely detaches. |

These verbs are also surfaced from the root CLI as
`vrooli scenario test wait|status|follow|abort <scenario> <run-id>` (and `logs`
as an alias for `follow`), which proxy to `test-genie runs …`.

### Anti-polling recipe

Do **not** loop with repeated "still waiting" checks. If a run backgrounds or
your tool times out, block once with the printed re-attach command:

```bash
test-genie runs wait --json --timeout=840 my-scenario 20251208-151044-abcd1234
# exits with the suite's real code; 124 if a --timeout window elapses
```

For coding-agent tool execution, give that single wait command at least the
printed recommended timeout and do not poll with short output checks. If a wait
process was already started and then the tool turn was interrupted, attach to
the existing local process instead of starting another wait:

```bash
pgrep -af 'test-genie runs wait --json .* my-scenario 20251208-151044-abcd1234'
tail --pid=<pid> -f /dev/null
```

When a bounded wait times out while the run is still active, Test Genie records
that timestamp locally. A second bounded wait for the same run within the
eagerness window prints a warning and the same attach guidance on stderr, while
keeping `--json` stdout parseable.

## coverage

Analyze test coverage for a scenario. Identifies gaps and suggests improvements.

```bash
test-genie coverage <scenario-name> [options]
```

### Options

| Option | Default | Description |
|--------|---------|-------------|
| `--depth <level>` | `standard` | Analysis depth |
| `--threshold <percent>` | `80` | Minimum coverage threshold |
| `--report <file>` | | Output file for report |
| `--format <type>` | `text` | Output format |
| `--show-gaps` | `false` | Show only uncovered areas |
| `--by-requirement` | `false` | Group by requirement |

### Depth Levels

| Depth | Description | Time |
|-------|-------------|------|
| `quick` | Basic line coverage | ~10s |
| `standard` | Line + branch coverage | ~30s |
| `deep` | Full analysis + gap identification | ~60s |
| `comprehensive` | Deep + requirement mapping | ~120s |

### Output Formats

| Format | Description |
|--------|-------------|
| `text` | Human-readable summary |
| `json` | Machine-readable JSON |
| `markdown` | Markdown report |
| `html` | Interactive HTML report |

## See Also

- [API Reference](api-endpoints.md)
- [Execution Configuration](configuration.md)
- [Test Presets](presets.md)

### Examples

```bash
# Standard coverage check
test-genie coverage my-scenario

# Deep analysis with report
test-genie coverage my-scenario --depth deep --report coverage.json

# Show only gaps
test-genie coverage my-scenario --show-gaps

# Coverage by requirement
test-genie coverage my-scenario --by-requirement

# Fail if below threshold
test-genie coverage my-scenario --threshold 90

# Markdown report
test-genie coverage my-scenario --format markdown --report coverage.md
```

### Output

```
Coverage Report: my-scenario
==============================

Overall Coverage: 87.3%
  Go API:     89.2% (target: 80%)  PASSED
  Node UI:    85.1% (target: 80%)  PASSED
  CLI:        82.4% (target: 70%)  PASSED

Requirement Coverage: 20/25 (80%)
  P0:  8/8  (100%)  PASSED
  P1: 10/12 ( 83%)  PASSED
  P2:  2/5  ( 40%)  OK

Uncovered Areas:
  api/handlers/projects.go:45-62      ProjectDelete handler
  api/handlers/workflows.go:120-145   WorkflowExecute error paths
  ui/src/stores/authStore.ts:89-102   Token refresh logic

Recommendations:
  1. Add tests for ProjectDelete handler
  2. Cover WorkflowExecute error scenarios
  3. Test token refresh edge cases
```

---

## vault

Create and run vault tests that validate scenarios through their complete lifecycle.

```bash
test-genie vault <scenario-name> [options]
```

### Options

| Option | Default | Description |
|--------|---------|-------------|
| `--phases <phases>` | all | Phases to include |
| `--criteria <file>` | | Path to success criteria YAML |
| `--timeout <seconds>` | `1800` | Per-phase timeout |
| `--continue-on-failure` | `false` | Don't abort on phase failure |
| `--report <file>` | | Output report file |

### Vault Phases

| Phase | Description |
|-------|-------------|
| `setup` | Initialize environment and resources |
| `develop` | Build and configure application |
| `test` | Run test suites |
| `deploy` | Deploy to target environment |
| `monitor` | Validate operational health |

### Examples

```bash
# Run all vault phases
test-genie vault my-scenario

# Specific phases
test-genie vault my-scenario --phases setup,develop,test

# With custom criteria
test-genie vault my-scenario --criteria ./vault-criteria.yaml

# Extended timeout
test-genie vault my-scenario --timeout 3600

# Generate report
test-genie vault my-scenario --report vault-report.json
```

### Success Criteria Format

```yaml
# vault-criteria.yaml
name: my-scenario-vault
phases:
  setup:
    timeout: 300
    requirements:
      - resource_availability: true
      - database_schema: loaded

  develop:
    timeout: 600
    requirements:
      - api_endpoints: functional
      - business_logic: implemented

  test:
    timeout: 900
    requirements:
      - unit_tests: passing
      - integration_tests: passing
      - unit_coverage_minimum_percent: 80
```

### Output

```
Vault Execution: my-scenario
=============================

[1/5] setup       PASSED  (45s)
  Resources initialized
  Database schema loaded

[2/5] develop     PASSED  (120s)
  API built successfully
  Endpoints verified

[3/5] test        PASSED  (180s)
  Unit tests: 45 passed
  Integration: 12 passed
  Coverage: 87%

[4/5] deploy      PASSED  (90s)
  Deployed to staging
  Health check passed

[5/5] monitor     PASSED  (60s)
  Latency: 45ms (target: <100ms)
  Error rate: 0% (target: <1%)

Vault Result: PASSED
Total Duration: 8m 15s
```

---

## health

Show **Test Genie's own** self-health — distinct from `status` (which checks
liveness). `health` is a thin client over `RunsService.GetSelfHealth` and stitches
three reads into one payload: the phase catalog summary, per-provider conformance
(the same scorecard `provider-contract scan` produces, probed live + time-boxed),
and the reliability ledger over a recent window (suite availability, run-level
terminal-outcome histogram, and per-phase/per-provider availability %, failure
rate, degraded counts, skip-reason + classification histograms, duration
p50/p95/min/max/avg, and worst-scenarios-per-phase). Compute-on-read and
advisory — nothing here gates a build.

Provider-backed catalog phases report `ExecutionMetrics`, so `metricsAdopted`
and duration columns are populated from delegated phase results. The ledger's
phase rollup is scoped to live catalog phases — legacy pseudo-phases
(`coverage`, `lint`, `playbooks`) from historical runs are excluded. The same
payload backs the dashboard's **Self-Health** tab.

```bash
test-genie health [--json] [--window-days N] [--skip-conformance]
```

### Options

| Option | Default | Description |
|--------|---------|-------------|
| `--json` | `false` | Emit the full self-health payload as proto JSON |
| `--window-days <N>` | `0` (server default = 30) | Reliability-ledger look-back window |
| `--skip-conformance` | `false` | Skip the live per-provider conformance scan (faster) |

### Examples

```bash
# Human summary (catalog, conformance, worst phases)
test-genie health

# Full machine-readable payload
test-genie health --json

# Reliability ledger only, last 7 days, skip the live conformance probes
test-genie health --window-days 7 --skip-conformance
```

Availability is denominator-correct because every terminal run — including
catastrophic aborts/timeouts/engine-errors that produce no result — is persisted
with a `terminal_outcome` classification. See
[Health Maturity Assessments → Test Genie Self-Health](../../../../docs/reference/health-maturity-assessments.md).

---

## fleet

Where `health` is Test Genie looking at **itself**, `fleet` is Test Genie looking
at the **whole fleet**. `fleet status` is a thin client over
`RunsService.GetFleetHealth`, which aggregates the **stored runs of every
scenario** into a fleet-wide health snapshot: per-scenario run counts,
availability, failure rate, issue counts (failed phase observations), and the
newest run's age + outcome; a **most-errored-first** ranking; finding-source
clustering across the fleet; and — with `--roster` — the scenarios that exist on
disk but have **no run in the window** (an explicit coverage gap, never a silent
zero).

This is **compute-on-read over runs that already executed** — it does NOT launch
a fleet run. Every datum is **as-of stamped** (`captured_at` for the whole
rollup; `last_run_at` + `age_days` per scenario) so stale data can never read as
fresh. Keeping those stored runs reasonably fresh is the job of the **default-OFF
background fleet scheduler** (see below); `fleet status` simply reads whatever is
there and labels its age honestly.

```bash
test-genie fleet status [--json] [--window-days N] [--roster]
```

### Options

| Option | Default | Description |
|--------|---------|-------------|
| `--json` | `false` | Emit the full fleet-health payload as proto JSON |
| `--window-days <N>` | `0` (server default = 30) | Fleet aggregation look-back window |
| `--roster` | `false` | Cross-reference the on-disk fleet roster to list never-tested-in-window scenarios |

### Examples

```bash
# Human summary: most-errored scenarios, top finding sources, coverage gaps
test-genie fleet status --roster

# Full machine-readable payload (consumed by the meta-optimization loop)
test-genie fleet status --json --roster
```

### Background fleet scheduler (default-OFF)

The fleet scheduler is the **write side** of the fleet backbone: a
priority-weighted background loop that cycles full suites across the fleet so the
stored runs `fleet status` reads stay fresh — **without** anyone paying an
hours-long synchronous fleet run. It is **OFF unless explicitly enabled** and
bounded by explicit budgets; it never weakens the run manager's
one-in-progress-per-scenario invariant (it queues by priority, it does not
parallelize a single scenario). Selection is staleness-weighted priority from
`scenario-completeness-scoring score list` (importance × score gap, scaled by
test age; never-tested scenarios get the strongest pull).

| Env var | Default | Meaning |
|---------|---------|---------|
| `TEST_GENIE_FLEET_SCHEDULER_ENABLED` | `false` | Start the scheduler (required to enable) |
| `TEST_GENIE_FLEET_SCHEDULER_INTERVAL` | `6h` | Tick cadence |
| `TEST_GENIE_FLEET_SCHEDULER_START_JITTER` | `0` | Initial delay before the first cycle |
| `TEST_GENIE_FLEET_SCHEDULER_MAX_CONCURRENT` | `1` | Simultaneously-launched runs (compute cap) |
| `TEST_GENIE_FLEET_SCHEDULER_MAX_PER_CYCLE` | `5` | Runs launched per cycle (count budget) |
| `TEST_GENIE_FLEET_SCHEDULER_CYCLE_BUDGET` | `0` (none) | Per-cycle wall-clock cap |
| `TEST_GENIE_FLEET_SCHEDULER_STALENESS_HORIZON` | `168h` | Window over which test age scales priority |
| `TEST_GENIE_FLEET_SCHEDULER_PRESET` | `comprehensive` | Suite shape the scheduler cycles |

---

## status

Check test-genie operational status and service health.

```bash
test-genie status [options]
```

### Options

| Option | Description |
|--------|-------------|
| `--resources` | Include resource status |
| `--queue` | Show queue depth and pending |
| `--executions` | Show recent executions |
| `--all` | Show all status information |

### Examples

```bash
# Basic status
test-genie status

# Full status
test-genie status --all

# Queue status only
test-genie status --queue

# Recent executions
test-genie status --executions
```

### Output

```
Test Genie Status
=================

Service: healthy
Version: 1.0.0
Uptime: 4h 32m

Database: connected
API: http://localhost:8080

Queue:
  Total: 15
  Pending: 3
  Running: 1
  Completed: 10
  Failed: 1

Recent Executions:
  my-scenario        PASSED  2m ago
  another-scenario   PASSED  15m ago
  test-scenario      FAILED  1h ago
```

---

## scenarios

List and inspect available scenarios.

```bash
test-genie scenarios [subcommand] [options]
```

### Subcommands

| Subcommand | Description |
|------------|-------------|
| `list` | List all scenarios (default) |
| `show <name>` | Show scenario details |
| `test <name>` | Run tests for scenario |

### Examples

```bash
# List all scenarios
test-genie scenarios
test-genie scenarios list

# Show scenario details
test-genie scenarios show my-scenario

# Run scenario tests
test-genie scenarios test my-scenario --type unit
```

### Output (list)

```
Scenarios (25 total)
====================

Name                    Tests   Last Run     Status
----                    -----   --------     ------
my-scenario             Yes     2h ago       passed
another-scenario        Yes     1d ago       passed
new-scenario            No      never        -
browser-automation      Yes     30m ago      failed
```

---

## phases

List available test phases and their configuration.

```bash
test-genie phases [options]
```

### Options

| Option | Description |
|--------|-------------|
| `--detailed` | Show full descriptions |
| `--json` | Output as JSON |

### Examples

```bash
# List phases
test-genie phases

# Detailed view
test-genie phases --detailed
```

### Output

```
Test Phases
===========

Name          Optional  Timeout  Description
----          --------  -------  -----------
structure     No        15m      Validates scenario layout and manifests
standards     No        1m       Runs scenario-auditor standards checks
dependencies  No        15m      Confirms commands, runtimes, and resources
quality       No        120s     Delegates static quality contracts to quality-health
docs          No        1m       Validates markdown, mermaid, and links
ui-health     No        5m       Delegates UI validation to ui-health
unit          No        15m      Executes language-specific unit suites
integration   No        15m      Runs CLI, API health, and WebSocket checks
workflow      No        15m      Delegates BAS workflow validation to workflow-health
business      No        3m       Audits requirements and business validation
performance   Yes       1m       Checks performance and duration budgets
```

---

## history

View execution history.

```bash
test-genie history [options]
```

### Options

| Option | Default | Description |
|--------|---------|-------------|
| `--scenario <name>` | | Filter by scenario |
| `--limit <n>` | `10` | Number of results |
| `--status <status>` | | Filter by status |
| `--since <duration>` | | Filter by time (e.g., "24h", "7d") |

### Examples

```bash
# Recent history
test-genie history

# Scenario-specific
test-genie history --scenario my-scenario

# Failed executions
test-genie history --status failed

# Last 24 hours
test-genie history --since 24h

# Combined filters
test-genie history --scenario my-scenario --status passed --limit 5
```

### Output

```
Execution History
=================

ID        Scenario          Preset         Status   Duration  Time
--        --------          ------         ------   --------  ----
a1b2c3    my-scenario       comprehensive  passed   2m 15s    2h ago
d4e5f6    my-scenario       quick          passed   45s       4h ago
g7h8i9    another-scenario  smoke          failed   1m 30s    6h ago
j0k1l2    my-scenario       comprehensive  passed   2m 30s    1d ago
```

---

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | Test failures |
| 2 | Configuration error |
| 3 | Timeout |
| 4 | Infrastructure error |
| 5 | Validation error |

---

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `TEST_GENIE_API_URL` | API endpoint | `http://localhost:8080` |
| `TEST_GENIE_DEBUG` | Enable debug output | `false` |
| `TEST_GENIE_LOG_LEVEL` | Log verbosity | `info` |
| `TEST_GENIE_TIMEOUT` | Default timeout (seconds) | `900` |
| `TEST_GENIE_CONFIG` | Config file path | `~/.test-genie/config.yaml` |

---

## Configuration File

Create `~/.test-genie/config.yaml` for persistent settings:

```yaml
# Default settings
api_url: http://localhost:8080
timeout: 900
log_level: info

# Default options for commands
defaults:
  generate:
    types: ["unit", "integration"]
    coverage: 90
  execute:
    preset: comprehensive
    fail_fast: false
  coverage:
    threshold: 80
    depth: standard
```

---

## Scripting Examples

### CI/CD Pipeline

```bash
#!/bin/bash
set -e

# Quick validation
test-genie execute my-scenario --preset quick --fail-fast

# Full test suite
test-genie execute my-scenario --preset comprehensive --json > results.json

# Check exit code
if [ $? -eq 0 ]; then
    echo "Tests passed"
else
    echo "Tests failed"
    exit 1
fi
```

### Coverage Gate

```bash
#!/bin/bash

# Check coverage meets threshold
test-genie coverage my-scenario --threshold 80 --json > coverage.json

if [ $? -ne 0 ]; then
    echo "Coverage below threshold"
    test-genie coverage my-scenario --show-gaps
    exit 1
fi
```

### Parallel Execution

```bash
#!/bin/bash

# Run tests for multiple scenarios in parallel
scenarios=("scenario-a" "scenario-b" "scenario-c")

for scenario in "${scenarios[@]}"; do
    test-genie execute "$scenario" --preset smoke &
done

wait

echo "All scenarios tested"
```

---

## See Also

- [API Endpoints](api-endpoints.md) - REST API reference
- [Presets](presets.md) - Preset definitions
- [Sync Execution Cheatsheet](sync-execution-cheatsheet.md) - Quick reference
- [Troubleshooting](../guides/troubleshooting.md) - Debug common issues
