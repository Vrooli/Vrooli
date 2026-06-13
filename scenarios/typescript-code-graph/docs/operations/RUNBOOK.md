# Runbook — TypeScript Code Graph

This document records operator procedures for running, diagnosing, recovering, and maintaining the scenario.

## Purpose Of This Document

Use this document to answer:

- How do I start, stop, and inspect the scenario?
- What checks should I run during an incident?
- How do I back up or restore state?
- Where should operational issues be recorded?

## Start / Stop / Status

Use lifecycle-managed commands from the scenario directory:

```bash
make setup            # one-time after generation or after dependency changes (also installs sidecar deps)
make start
make status
make logs
make stop
make test
```

Do not start API/UI/sidecar binaries directly. The lifecycle owns process naming, ports, health checks, and logs. The Node sidecar is spawned as a child of the API process; it does not have its own lifecycle entry.

For agent and CLI use:

```bash
vrooli scenario start typescript-code-graph
vrooli scenario status typescript-code-graph
vrooli scenario port typescript-code-graph API_PORT
vrooli scenario logs typescript-code-graph
vrooli scenario restart typescript-code-graph   # preferred over stop+start
```

## Common Incidents

| Symptom | Checks | Fix | Escalation |
|---|---|---|---|
| Scenario does not start | `make status`, `make logs` | `make restart`, then inspect lifecycle logs | Record recurring failures in [`../internal/PROBLEMS.md`](../internal/PROBLEMS.md). |
| API unhealthy with `sidecar_unavailable` | `/health` returns `sidecar: unhealthy` | Wait for supervisor to restart (5 attempts in 60s with backoff). If `permanently_unhealthy`, run `make restart`. | Inspect `make logs` for sidecar crash cause. Common: `ts-morph` install drift, Node version too old. |
| `Extract` returns `ErrSidecarTimeout` | Recent extraction durations in logs | Check the diagnostics page's sidecar status. If sidecar is healthy but slow, the project is genuinely large or unusual. | If timeouts recur on the same project, capture a sidecar profile and open a perf entry in `PERFORMANCE.md`. |
| `Extract` returns `ErrNoTsconfigFound` | Verify the path. `find <path> -maxdepth 1 -name tsconfig.json` | Point `Extract` at the directory containing `tsconfig.json`. | If a consumer is consistently passing the wrong path, file a bug at the consumer. |
| `Extract` returns `ErrWorkspaceUnsupported` | Check whether the selected project directory itself contains a multi-project `pnpm-workspace.yaml` | Point at a specific child project root or explicit `tsconfig.json`. Workspace-root support is P2 (OT-P2-005). | Promote OT-P2-005 if a real consumer needs workspace-root mode. |
| Leading comments missing from graph response | `Extract` succeeded but `leading_comments` is empty for declarations that have JSDoc | Confirm the fixture is current. The `bas/fixtures/ts-jsdoc-tags/` test should be green. | This is a contract violation — file as a bug. The contract is load-bearing for rcl. |
| `Rewrite apply` returned `ErrApplyPartial` | Inspect `failed_op`; check `git status` in the target project | Operator: `cd <project> && git restore .` to reset. Then re-plan from scratch (do not retry with the same `plan_id`). | If partial failures are recurring, file a bug. |
| `Rewrite apply` returned `ErrPlanContentMismatch` | Source changed between plan and apply | Re-run `RewritePlan` to get a fresh `plan_id`, then apply. | None — expected when consumers don't lock the working tree. |
| Sidecar crashes repeatedly during `Extract` | `make logs` shows Node child exiting | Check Node version (`node --version` ≥ 20.x). Check that `pnpm install` succeeded for the sidecar's `package.json`. | If crashes correlate with a specific input project, capture the crash log and fixture-isolate. |
| Sidecar permanently unhealthy | Supervisor exhausted restart budget | `make restart`. If still failing, inspect the build artifact `sidecar/dist/index.js`. | Re-run `pnpm install && pnpm build` inside `sidecar/`. |
| Concurrent `Extract` calls appear to hang | Per-path mutex blocking | Expected behavior. If "hang" exceeds the SLA, the first call is genuinely slow; inspect sidecar logs. | None unless duration exceeds SLA. |
| UI sidecar status panel shows red | API `/health` reports sidecar issue | Same as the API-unhealthy entry above. | The UI panel is the most discoverable sidecar diagnostic. |
| UI blank or stale | UI port, browser console, `ui/dist` freshness | `make setup` then `make restart` | Add troubleshooting entry if recurring. |
| CLI talks to old API | `typescript-code-graph status`, configured API base | Reinstall via `make setup` | Update CLI reference if command changed. |

## Backup / Restore

The scenario is **stateless in v1**. There is nothing to back up unless the Operation Log (P1) is enabled.

| Data | Backup Procedure | Restore Procedure | Status |
|---|---|---|---|
| Operation Log SQLite database (P1 only) | Copy the `$SQLITE_PATH` file | Replace the `$SQLITE_PATH` file | Define schema migration policy before deployment. |
| Plan registry | n/a — in-process and ephemeral | n/a | By design — plans expire on restart. |
| Sidecar process state | n/a — ephemeral by design | n/a | By design — sidecar holds no durable state. |
| Recent-calls telemetry | n/a — in-memory ring buffer | n/a | By design. |

## Maintenance Tasks

| Task | Frequency | Command / Procedure |
|---|---|---|
| Validate tests | before handoff | `make test` |
| Validate determinism | before handoff | `make test` (includes `bas/fixtures/` golden comparison) |
| Validate performance | before handoff | `go test ./api/internal/graph -bench=. -benchtime=5x` |
| Validate sidecar chaos resilience | before handoff | `go test ./api/internal/sidecar -tags=chaos` |
| Inspect logs | as needed | `make logs` (includes sidecar stderr) |
| Regenerate endpoints | after API endpoint changes | `make endpoints` |
| Regenerate proto | after `packages/proto/schemas/typescript-code-graph/` or `common/v1/code_graph.proto` changes | `cd packages/proto && make generate` |
| Rebuild sidecar | after `sidecar/src/` changes | `cd sidecar && pnpm build` (lifecycle wires this into `make setup`) |
| Regenerate UI strings | after i18n changes | `cd ui && pnpm strings:gen` |

## Escalation

Record known operational issues in [`../internal/PROBLEMS.md`](../internal/PROBLEMS.md). Append meaningful completed work to [`../internal/PROGRESS.md`](../internal/PROGRESS.md).

For consumer scenarios reporting bugs, the first checks are:
1. Is the consumer pointing `Extract` at a valid single-project TS source tree?
2. Is the sidecar healthy? Check `/health`.
3. Is the consumer holding a stale `plan_id`? (5-minute TTL.)
4. Is the consumer using the latest proto from `common/v1/code_graph.proto`?

For `react-component-library` migration issues specifically: verify the leading-comment contract is intact by running the `bas/fixtures/ts-jsdoc-tags/` test. If that test passes but rcl's migration sees missing comments, the bug is in rcl's adapter, not in this scenario.

## Cross-References

- [`DEPLOYMENT.md`](DEPLOYMENT.md) — deployment tiers and release checklist
- [`OBSERVABILITY.md`](OBSERVABILITY.md) — logs, metrics, and health signals
- [`../guides/troubleshooting.md`](../guides/troubleshooting.md) — common fixes
- [`../reference/configuration.md`](../reference/configuration.md) — runtime configuration
- [`../internal/PROBLEMS.md`](../internal/PROBLEMS.md) — known scenario-specific issues
