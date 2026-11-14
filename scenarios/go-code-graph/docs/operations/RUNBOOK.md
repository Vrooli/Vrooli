# Runbook — Go Code Graph

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
make setup            # one-time after generation or after dependency changes
make start
make status
make logs
make stop
make test
```

Do not start API/UI binaries directly. The lifecycle owns process naming, ports, health checks, and logs.

For agent and CLI use:

```bash
vrooli scenario start go-code-graph
vrooli scenario status go-code-graph
vrooli scenario port go-code-graph API_PORT
vrooli scenario logs go-code-graph
vrooli scenario restart go-code-graph   # preferred over stop+start
```

## Common Incidents

| Symptom | Checks | Fix | Escalation |
|---|---|---|---|
| Scenario does not start | `make status`, `make logs` | `make restart`, then inspect lifecycle logs | Record recurring failures in [`../internal/PROBLEMS.md`](../internal/PROBLEMS.md). |
| API unhealthy | `/health`, SQLite path (if P1 enabled), API logs | Run `make setup`, verify writable data dir | Check `../concepts/INTEGRATIONS.md` for dependency expectations. |
| `Extract` returns `ErrNoGoModFound` for a known module | Verify the path. `find <path> -maxdepth 1 -name go.mod` | Point `Extract` at the directory containing `go.mod`, not a parent or subdirectory. | If a consumer is consistently passing the wrong path, file a bug at the consumer. |
| `Extract` returns `ErrWorkspaceUnsupported` | Look for `go.work` in the input or its ancestors | Point at a specific module inside the workspace instead. Workspace support is P2 (OT-P2-005). | Promote OT-P2-005 if a real consumer needs workspace mode. |
| `Extract` is slow on a large module | `make logs`, check op duration in recent-calls telemetry | Verify the SLA: <30s for ≤2000 files. If beyond, capture profile via `go test -cpuprofile`. | Open a perf entry in `PERFORMANCE.md` and decide whether to relax SLA or optimize. |
| `Rewrite apply` returned `ErrApplyPartial` | Inspect `failed_op` in the response; check git status in the target module | Operator: `cd <module> && git restore .` to reset the working tree. Then re-plan from scratch (do not retry with the same `plan_id`). | If partial failures are recurring, file a bug — the implementation should be hardened, but the contract intentionally accepts mid-state. |
| `Rewrite apply` returned `ErrPlanContentMismatch` | Source code in the target module changed between plan and apply | Re-run `RewritePlan` to get a fresh `plan_id`, then apply. | None — this is expected when consumers don't lock the working tree. |
| Concurrent `Extract` calls appear to hang | Check the per-path mutex — a second call for the same path blocks until the first completes | Expected behavior. If "hang" exceeds the SLA, the first call is genuinely slow; inspect logs. | None unless duration exceeds SLA. |
| UI blank or stale | UI port, browser console, `ui/dist` freshness | `make setup` then `make restart` | Add troubleshooting entry if recurring. |
| CLI talks to old API | `go-code-graph status`, configured API base | Reinstall via `make setup` | Update CLI reference if command changed. |

## Backup / Restore

The scenario is **stateless in v1**. There is nothing to back up unless the Operation Log (P1) is enabled.

| Data | Backup Procedure | Restore Procedure | Status |
|---|---|---|---|
| Operation Log SQLite database (P1 only) | Copy the scenario database file | Replace the scenario database file | Define schema migration policy before deployment. |
| Plan registry | n/a — in-process and ephemeral | n/a | By design — plans expire on restart. Consumers re-plan. |
| Recent-calls telemetry | n/a — in-memory ring buffer | n/a | By design. |

## Maintenance Tasks

| Task | Frequency | Command / Procedure |
|---|---|---|
| Validate tests | before handoff | `make test` |
| Validate determinism | before handoff | `make test` (includes `bas/fixtures/` golden comparison) |
| Validate performance | before handoff | `go test ./api/internal/graph -bench=. -benchtime=5x` |
| Inspect logs | as needed | `make logs` |
| Regenerate endpoints | after API endpoint changes | `make endpoints` |
| Regenerate proto | after `packages/proto/schemas/go-code-graph/` or `common/v1/code_graph.proto` changes | `cd packages/proto && make generate` |
| Regenerate UI strings | after i18n changes | `cd ui && pnpm strings:gen` |

## Escalation

Record known operational issues in [`../internal/PROBLEMS.md`](../internal/PROBLEMS.md). Append meaningful completed work to [`../internal/PROGRESS.md`](../internal/PROGRESS.md).

For consumer scenarios reporting bugs, the first checks are:
1. Is the consumer pointing `Extract` at a valid single-module Go project? (Most issues are input-shape, not implementation.)
2. Is the consumer holding a stale `plan_id`? (Plans expire after 5 minutes.)
3. Is the consumer using the latest proto from `common/v1/code_graph.proto`?

## Cross-References

- [`DEPLOYMENT.md`](DEPLOYMENT.md) — deployment tiers and release checklist
- [`OBSERVABILITY.md`](OBSERVABILITY.md) — logs, metrics, and health signals
- [`../guides/troubleshooting.md`](../guides/troubleshooting.md) — common fixes
- [`../reference/configuration.md`](../reference/configuration.md) — runtime configuration
- [`../internal/PROBLEMS.md`](../internal/PROBLEMS.md) — known scenario-specific issues
