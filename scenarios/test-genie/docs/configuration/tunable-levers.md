# Test Genie Configuration Levers

This is the operator-facing configuration reference for Test Genie. These levers are the intended control surface for tuning behavior without editing code.

## Core runtime

These are normally provided by the Vrooli lifecycle system:

| Variable | Scope | Default | Purpose |
|----------|-------|---------|---------|
| `API_PORT` | API runtime | lifecycle | Port the Test Genie API listens on |
| `TEST_GENIE_SQLITE_PATH` | API runtime | `${SCENARIO_DATA_DIR}/test-genie.db` | Scenario-local SQLite database path |
| `SQLITE_PATH` / `SQLITE_DB` | API runtime | unset | Generic SQLite path override used by maintenance tooling |
| `SCENARIO_DATA_DIR` | API runtime | lifecycle | Default root for embedded persistent state |
| `SCENARIOS_ROOT` | API runtime | inferred from cwd | Root directory for scenario discovery |
| `VROOLI_ROOT` | API + CLI | environment | Repo root for docs, scenario lookup, and path resolution |

## High-value Test Genie levers

| Variable | Scope | Default | Purpose |
|----------|-------|---------|---------|
| `TEST_GENIE_EXECUTION_TIMEOUT` | CLI `execute` | `900` seconds | Blocking timeout for synchronous suite execution |
| `TEST_GENIE_MAX_CONCURRENT_RUNS` | Run manager | `2` | GLOBAL cap on suites executing at once across ALL scenarios, shared by manually-started runs and the background fleet sweep. Requests beyond the cap are admitted as `queued` and promoted FIFO as slots free (not rejected). Floor 1. |
| `TEST_GENIE_MAX_RUNS_PER_SCENARIO` | Run manager | `1` | Per-scenario in-progress cap. `1` is a correctness invariant (one live instance per scenario); raising it is documented-unsafe until per-run isolation lands. |
| `TEST_GENIE_PLAYBOOKS_RETAIN` | Playbooks phase | `0` | Keep temporary isolated Postgres/Redis/SQLite resources alive after the phase for debugging |
| `TEST_GENIE_QUEUE_STALE_AFTER` | Queue telemetry | `24h` | How long queued/delegated requests remain part of active queue counts before they are reported as stale |
| `TEST_GENIE_SKIP_PLAYBOOKS` | Playbooks phase | unset | Hard-disable playbooks execution for debugging or constrained environments |
| `TEST_GENIE_STANDARDS_FAIL_ON` | Standards phase | phase default | Minimum severity that fails the phase |
| `TEST_GENIE_STANDARDS_MIN_SEVERITY` | Standards phase | phase default | Minimum severity shown in standards output |
| `TEST_GENIE_STANDARDS_LIMIT` | Standards phase | phase default | Maximum number of standards findings displayed |
| `TEST_GENIE_DOCS_DIR` | Docs handlers | scenario default | Override docs directory served by the API |

## Queue telemetry

`TEST_GENIE_QUEUE_STALE_AFTER` is the main queue hygiene lever.

- Format: Go duration string, for example `30m`, `6h`, or `24h`
- A queued or delegated request older than this threshold is excluded from active `pending`, `queued`, and `delegated` counts
- Stale rows are still counted separately as `stale` in health/status output so operators can see cleanup drift instead of silently losing visibility

Example:

```bash
TEST_GENIE_QUEUE_STALE_AFTER=6h test-genie status
```

## Playbooks

### Debug retained isolation

```bash
TEST_GENIE_PLAYBOOKS_RETAIN=1 test-genie execute my-scenario --phases playbooks
```

When retention is enabled, the playbooks phase leaves its temporary isolated resources alive and prints inspection commands in the observations.

### Skip playbooks entirely

```bash
TEST_GENIE_SKIP_PLAYBOOKS=1 test-genie execute my-scenario --phases playbooks
```

This is intended for debugging and constrained environments. It is not a substitute for fixing a broken playbooks setup.

## Standards tuning

These levers tune the standards phase without modifying scenario config:

```bash
TEST_GENIE_STANDARDS_FAIL_ON=critical \
TEST_GENIE_STANDARDS_MIN_SEVERITY=high \
TEST_GENIE_STANDARDS_LIMIT=20 \
test-genie execute my-scenario --phases standards
```

## CLI timeout tuning

Long suites can exceed the default synchronous timeout. Extend it when running comprehensive or slow playbook-heavy suites:

```bash
TEST_GENIE_EXECUTION_TIMEOUT=1800 test-genie execute my-scenario --preset comprehensive
```

## Playbooks registry metadata

Some behavior is expressed in `bas/registry.json` rather than environment variables:

| Field | Purpose |
|-------|---------|
| `metadata.execution_mode` | Marks a registry as observer-only when every playbook is read-only |
| `playbooks[].reset` | Declares whether the next workflow needs fresh seed state |
| `playbooks[].requirements` | Requirement IDs attached to the workflow |

The registry should be generated via `test-genie registry build`, not hand-maintained.

## Dependencies Phase

The dependencies phase delegates to Scenario Dependency Analyzer:

```bash
scenario-dependency-analyzer health <scenario> --json
```

Dependency readiness, runtime dependency policy, graph drift, governance, release-age policy, and degraded Security Health dependency-index status are configured and interpreted by SDA. Test Genie does not read `.vrooli/testing.json` dependency knobs for this phase.

Common remediation:

| Finding | Typical command |
|---------|-----------------|
| `dependency.readiness.*` | Run the remediation reported by `scenario-dependency-analyzer health <scenario> --json` |
| `dependency.runtime.*` | Start or restart the reported resource/scenario dependency |
| `dependency.graph.*` | Update `.vrooli/service.json` or remove stale dependency usage |
| `dependency.governance.*` | Use SDA governance verbs (never hand-edit the JSON): `scenario-dependency-analyzer deps approved explain <ecosystem>/<pkg>`, then `approve-observed --apply` / `widen-range` / `deny-vulnerable` |
| `dependency.release_age.*` | Add/raise pnpm `minimumReleaseAge` or record an approved exclusion |
| `dependency.security.*` | Check `security-health deps status --json` |

## Advanced infrastructure levers

Test Genie also participates in broader Vrooli infrastructure concerns:

- `AGENT_MANAGER_ENABLED`
- `AGENT_MANAGER_PROFILE_KEY`
- `CONTAINMENT_*`

Those are advanced deployment controls rather than primary Test Genie workflow levers. Use them when you are changing agent-execution or containment policy, not when debugging normal suite execution.

### Realtime note

Agent run streaming comes from `agent-manager`, not from a native Test Genie WebSocket endpoint. Test Genie exposes `/api/v1/agents/ws-url` so the UI can discover that external socket when agent workflows are enabled. Because that channel is optional infrastructure, the scenario's core integration phase should not treat it as a required runtime surface.
