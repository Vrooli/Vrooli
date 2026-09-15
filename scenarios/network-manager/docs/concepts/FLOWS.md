# Flows — Network Manager

## Purpose Of This Document

This document captures Network Manager's user/system workflows, lifecycle states, retries, cancellation points, and stale-completion risks. Data details live in [`DATA.md`](DATA.md); integration details live in [`INTEGRATIONS.md`](INTEGRATIONS.md).

## Flow Inventory

| Flow | Status | Requirements | Notes |
|---|---|---|---|
| Health snapshot run | Planned P0 | `NM-P0-001` | Read-only measurement flow. |
| Resolver setup/status | Planned P0 | `NM-P0-002` | AdGuard Home first. |
| Filtering change preview/apply/rollback | Planned P0 | `NM-P0-003` | Persistent changes require approval. |
| Device inventory refresh | Planned P0 | `NM-P0-004` | Identity confidence must be recorded. |
| Optimization experiment | Planned P0 | `NM-P0-005` | Baseline/candidate/after with scoring. |
| Device Control action/event | Planned P0 | `NM-P0-007` | Consumed through the generalized device-control integration. |
| Household schedule evaluation | Advisory P1 slice | `NM-P1-001`, `NM-P1-002` | Persisted device/group policy intent and manual-required schedule evaluation; automatic enforcement awaits resolver/router capability. |
| Continuous monitoring | Advisory P1 slice | `NM-P1-007` | Stored schedules, operator-triggered checks, recurring snapshot intent, and regression alerts. |

## Flow Details

### Health snapshot run

1. Operator or automation requests a snapshot.
2. Adapter registry reports available probes.
3. Snapshot domain runs supported probes with timeouts.
4. Unsupported probes are recorded as unavailable, not failed.
5. Report is persisted and returned to UI/CLI/API consumers.

### Filtering change preview/apply/rollback

1. Operator drafts a policy change.
2. Resolver adapter produces a change plan and risk summary.
3. UI/CLI displays affected devices/groups and rollback availability.
4. Operator approves.
5. Resolver adapter applies the change.
6. Rollback handle and audit entry are persisted.
7. A rollback operation can restore the prior state if supported.

### Household schedule evaluation

1. Operator stores a named profile with a device group, filtering strength, schedule, and override behavior.
2. Policy service validates and persists the profile intent.
3. Operator or UI/CLI requests schedule evaluation for a profile and target.
4. Service evaluates the current schedule window and returns `manual_required` effects rather than mutating resolver/router state.
5. Future resolver/router adapters may turn the same profile intent into scheduled enforcement after capability and rollback support are proven.

### Encrypted DNS guidance

1. Operator requests IPv6/encrypted-DNS bypass guidance or endpoint/browser DoH guidance.
2. Policy service generates a read-only report with checks, evidence, manual steps, adapter-preview actions, and guardrails.
3. The report stays `manual_required` or `guidance_only` unless a future adapter can prove safe mutation and rollback support.
4. Network Manager never uses TLS interception, hidden monitoring, or query-level surveillance to generate the guidance.

### Continuous monitoring

1. Operator captures or selects a baseline snapshot.
2. Operator stores a monitoring schedule with profile, interval, baseline snapshot, and alert thresholds.
3. Operator, UI, CLI, or future scheduler requests a monitoring check for that schedule.
4. Monitoring runs a fresh read-only snapshot through the snapshot service.
5. Monitoring compares DNS latency, unavailable/failed probe count, and critical resolver/WAN probe status against the baseline.
6. Regressions are persisted as open alerts; non-regression runs remain evidence without mutating resolver/router state.
7. Current production behavior is advisory/operator-triggered. A future background scheduler can consume the stored schedule contract.

### Optimization experiment

1. Baseline snapshot is captured.
2. Candidate configurations are generated from supported capabilities.
3. Each candidate is applied temporarily or simulated where safe.
4. Candidate snapshot is captured after stabilization.
5. Candidate is rolled back or left pending approval.
6. Scorer ranks candidates by reliability-first metrics.
7. Operator chooses whether to apply the winning persistent change.

## State Machines

### Optimization run

`draft -> baseline_running -> candidates_running -> scored -> awaiting_approval -> applied -> verified`

Failure states:

- `aborted`: operator cancels before persistent apply.
- `failed`: probe or adapter error blocks comparison.
- `rolled_back`: applied candidate was reverted.
- `manual_required`: adapter cannot apply but can produce instructions.

### Policy change

`draft -> previewed -> approved -> applying -> applied -> rollback_available`

Failure states:

- `rejected`
- `apply_failed`
- `rollback_failed`
- `unsupported`

## Maturity Ladder

- L0: Ad hoc commands with no comparable report.
- L1: Read-only snapshots and resolver status.
- L2: Previewable filtering changes with rollback records.
- L3: Optimization experiments with controlled candidates and scoring.
- L4: Scheduled monitoring and policy workflows with regression detection.

## Production Shape

Production flows must be resumable or explicitly terminal. Long-running operations should write an operation row before starting, update progress after every phase, and expose cancellation where cancellation is safe.

## Deferred / Unmodeled Flows

- Router write flows are deferred until a P1 router adapter is selected.
- Roaming-device filtering is deferred to P2.
- Advanced packet capture is deferred to P2 and requires explicit consent.

## Cross-References

- [`ARCHITECTURE.md`](ARCHITECTURE.md)
- [`DOMAINS.md`](DOMAINS.md)
- [`DATA.md`](DATA.md)
- [`../internal/SEAMS.md`](../internal/SEAMS.md)
