# Service objectives and alerting contract

Frozen 2026-09-09 by phase 23 of plan
`scenario-to-cloud-professional-vps-delivery-certification`. Every number
here is the same number as in `certification/budgets.json` (`phase_23`);
`api/perfbudget` loads and validates that file and refuses a document where a
phase-23 number drifts from its phase-1 owner. The budgets apply to the
declared machine profile (2 vCPU, 4 GiB RAM, 40 GiB disk, one public IPv4)
and to the certified fixture profiles. They are not a claim about any other
machine or workload.

Rule (from `budgets.json`): a weaker budget may not be introduced after a
failed measurement solely to make a report pass. Adjust only before
execution, with workload size, capacity and justification recorded.

## Service level objectives

### Read latency and errors (management API, 20 concurrent readers)

| Profile | p50 | p95 | p99 | Error rate | Target CPU | Target RSS | Target disk |
|---|---|---|---|---|---|---|---|
| `stateless-web` | ≤ 150 ms | ≤ 500 ms | ≤ 1000 ms | ≤ 0.5 % | ≤ 60 % | ≤ 1024 MiB | ≤ 12 GiB |
| `headless-api` | ≤ 150 ms | ≤ 500 ms | ≤ 1000 ms | ≤ 0.5 % | ≤ 60 % | ≤ 1024 MiB | ≤ 12 GiB |
| `sql-uploads` | ≤ 200 ms | ≤ 500 ms | ≤ 1500 ms | ≤ 0.5 % | ≤ 70 % | ≤ 2048 MiB | ≤ 20 GiB |

Read endpoints per profile are listed in `budgets.json`
`phase_23.fixture_profiles.<profile>.read_endpoints`. Percentiles are only
claimed over at least 100 samples per phase; fewer samples are reported as
`insufficient_samples` with the observed values attached
(`perfbudget.Compare`). A measurement has four phases: baseline (quiet),
sustained (20 readers), peak (40 readers), post-cleanup (quiet).

### Management process

| Phase | CPU | RSS | Goroutines | Open fds |
|---|---|---|---|---|
| Idle | ≤ 5 % | ≤ 300 MiB | ≤ 500 | ≤ 256 |
| Active operation | ≤ 50 % | ≤ 768 MiB | ≤ 2000 | ≤ 1024 |
| Post-cleanup growth vs. baseline | – | ≤ 64 MiB | ≤ 8 | ≤ 8 |

Growth beyond the post-cleanup deltas after a load window is unexplained
growth and fails P23-A02.

### Operations: queue, concurrency, retries, timeouts

| Bound | Value | Where enforced |
|---|---|---|
| Effectful operations per deployment | 1 | `persistence.AcquireWorker` (RUN-08) |
| Effectful operations per host | 4 | `persistence.AcquireWorker` across deployments sharing a target (`SetHostConcurrencyLimit`) |
| Queue depth | 32 | budget; `operations.Service` pool queue |
| Queue delay | ≤ 120 s | `operations.Config.QueueTimeout` |
| Worker pool | 2 | `operations.DefaultConfig` |
| Execution / transport / observer | 45 min / 60 s / 5 min | `operations.DefaultConfig` |
| Lease / heartbeat / reconcile | 90 s / 30 s / 90 s | `operations.DefaultConfig` |

Retry ceilings per `api/execplan` retry class (attempts include the first;
backoff is exponential with a cap; beyond the ceiling the operation parks in
`reconciling` with a typed next action):

| Class | Max attempts | Initial backoff | Multiplier | Cap |
|---|---|---|---|---|
| `safe_replay` | 3 | 5 s | 2 | 60 s |
| `observe_then_replay` | 3 | 15 s | 2 | 120 s |
| `recover` | 1 | – | – | – |
| `transport_receipt_read` | 3 | 2 s | 2 | 30 s |
| `alert_delivery` | 5 | 10 s | 2 | 300 s |

`perfbudget.RetryBudget.Backoff(attempt)` is the arithmetic; the test
`TestEveryRetryClassHasAFiniteCeiling` proves every class is finite.

### Retention

| Family | Keep | Window | Protected (never deleted) |
|---|---|---|---|
| Release artifacts (target bundle cache) | active + predecessor (2 per scenario) | 30 d | active, rollback predecessor, any release a non-terminal operation names (`bundle.PlanVPSBundleGC` + `bundle.ReleaseProtectedBundleSHA256s`) |
| Logs | – | 14 d | attachments ≤ 32 MiB each |
| Operation receipts | – | 90 d | non-terminal operations; operations an open incident references |
| Recovery points | last 7 | 30 d | held by an active or retained release, a non-terminal operation, or an operator pin (`backup.Protection`) |
| Health observations | – | 7 d | – |
| Fixture leases after terminal | – | 24 h | – |

### Recovery and detection

| Objective | Value |
|---|---|
| Fresh-host restore (RTO) | ≤ 15 min |
| Recovery point (RPO) | ≤ 5 min for the protected-write stream |
| Alert detection after the condition is met | ≤ 60 s |
| Recovery notification after the condition clears | ≤ 120 s |
| Health observation considered stale after | 120 s (`health.DefaultMaxObservationAge`) |
| Certificate expiry warning | ≤ 14 days before `not_after` |
| Soak | ≥ 24 continuous hours, observation every 60 s, one controlled reboot, one safe update |

## Alert conditions

Evaluated by `perfbudget.Evaluate` from observations other owners produce:
the typed health observation (`GET /api/v1/deployments/{id}/health/observation`),
the edge observation (`GET /api/v1/deployments/{id}/edge`, certificate
`not_after`), the recovery-point list, and the credential binding list.

| `check_id` | Condition | Severity | Next action |
|---|---|---|---|
| `service_unavailable` | current observation with status `UNHEALTHY` (`DEGRADED` → warning) | critical | `docs/guides/incident-runbook.md#diagnose` |
| `health_observation_stale` | no observation, freshness ≠ `CURRENT`, `observed_at` older than 120 s on the consumer clock, or an unknown status | critical | `scenario-to-cloud deployment health --deployment "$DEPLOYMENT_ID"` |
| `certificate_expiring` | `not_after` − now ≤ 14 d (critical once expired) | warning | `scenario-to-cloud edge status --deployment "$DEPLOYMENT_ID"` |
| `backup_freshness` | deployment has protected writes and the newest recovery point is older than the RPO (or none exists) | critical | `scenario-to-cloud deployment recovery-points capture --deployment "$DEPLOYMENT_ID"` |
| `credential_rotation_pending` | at least one binding has a pending rotation | warning | `scenario-to-cloud credential list --deployment "$DEPLOYMENT_ID"` |

Invariants:

- A stale or unknown observation never yields "service healthy"
  (`perfbudget.ServiceHealthy`); it raises `health_observation_stale` and
  suppresses nothing else.
- `incident_id` is `<deployment_id>:<check_id>`, stable across re-evaluation,
  so notification-hub de-duplicates repeats and pairs the resolution.
- Only transitions are published (`perfbudget.Transitions`): a newly met
  condition opens; a previously open condition that is no longer met
  publishes `status: resolved` with the same `incident_id` and severity
  `informational`. Steady state publishes nothing.

## Delivery path

```
health/edge/backup/credential observation
   → perfbudget.Evaluate + Transitions (producer: scenario-to-cloud API, P16 owner)
   → eventbus.DomainEvent{Source:"scenario-to-cloud", EventType:"scenario-to-cloud.deployment.alert.v1", Payload: alert.Facts()}
   → notification-hub event webhook (signed; event_id = <incident_id>:<status>:<detected_at>)
   → notification-hub renders copy from facts, applies sensitivity by severity, delivers to the operator recipient
```

Event facts (`perfbudget.Alert.Facts()`; producers send facts only, never
title/body/sensitivity):

```json
{
  "schema_version": 1,
  "check_id": "service_unavailable",
  "severity": "critical",
  "status": "open",
  "incident_id": "3c1c9a1e-…:service_unavailable",
  "deployment_id": "3c1c9a1e-…",
  "target_id": "machine:…",
  "release_digest": "sha256:…",
  "observed_at": "2026-09-09T12:00:00Z",
  "detected_at": "2026-09-09T12:00:30Z",
  "reason": "status_unhealthy",
  "message": "The deployment is observed unhealthy on a current observation.",
  "next_action": {"owner": "scenario-to-cloud", "kind": "doc", "reference": "docs/guides/incident-runbook.md#diagnose", "label": "Open the incident runbook"}
}
```

`check_id`, `severity`, `status`, `message`, `reason` and `incident_id` are
the keys notification-hub's renderer and dedupe reads; the rest are
identity facts for the operator and for receipts. Severity vocabulary is
notification-hub's (`critical`, `warning`, `informational`) so the
server-owned sensitivity mapping applies. Nothing in the payload is a secret;
`reason` and `message` are code-owned strings.

Detection budget accounting: observation interval (≤ 60 s in the soak) +
evaluation (in-process) + publish (bounded by the `alert_delivery` retry
class) must fit the 60 s detection budget, so the scheduled observation
interval is the whole budget; delivery retries beyond it are reported as a
missed budget, not hidden.

## Gaps (recorded, not hidden)

- Alert emission is wired (`api/health/alerts.go`, `Server.alerter()` in
  `handlers_health.go`) and publishes transitions after each observation;
  delivery through notification-hub to an operator recipient is not yet
  exercised (needs a soak target, EXT-09).
- The per-host limit (4) is enforced by `persistence.AcquireWorker` across
  deployments sharing a `target_key` (`Repository.SetHostConcurrencyLimit`,
  loaded from `budgets.json` by `health.LoadHostConcurrencyLimit`).
- Operation-record retention exists (`Repository.PruneTerminalOperations`,
  protected ids and non-terminal records kept) but no scheduler calls it;
  records accumulate on a real target until one is wired.
- CPU percentages of the management process are not sampled by the harness
  (RSS, fds, goroutines and heap are); CPU comes from the target's
  `systemmetrics` collector on a real lane.
