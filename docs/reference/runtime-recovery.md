# Runtime recovery policy

The runtime registry distinguishes observed workload state from desired recovery
state. A dead or expired lease is evidence only: it never authorizes a restart.
Automatic recovery is disabled until an operator persists an explicit policy.

## Policy

Use the control plane to declare a workload eligible for a future recovery
controller:

```bash
vrooli runtime recovery policy set system-monitor \
  --critical --enabled --tier 0 --retry-budget 2
```

The declaration is durable in the runtime registry. `--opt-out` wins over every
other setting; omitting `--critical` or `--enabled` leaves a fail-closed policy.
Policies are per scenario variant; use `--variant <name>` for a non-live
instance.

Disable an automatic policy immediately with `--opt-out`; restore it later by
writing the complete enabled declaration with `--clear-opt-out`.

```bash
vrooli runtime recovery policy list
vrooli runtime supervisor status
```

The supervisor status exposes policies and retained pressure epochs so operators
can distinguish observed pressure, a recovery gate, queued work, restoration,
skips, and terminal failures as the controller is enabled in later rollout
stages.

## Safety contract

- Declared recovery is disabled by default.
- A recovery decision is deduplicated by a durable idempotency key.
- Pressure epochs and decisions survive supervisor restart.
- No policy is inferred from a runtime lease, process ID, or prior start.
- Policy changes do not launch processes; lifecycle operations remain the only
  permitted future launcher.

## Recovery tunables

`VROOLI_RUNTIME_RECOVERY_QUIET_PERIOD` requires continuously clear pressure
evidence before dispatch. `VROOLI_RUNTIME_RECOVERY_COOLDOWN` prevents immediate
retry after lifecycle failure. `VROOLI_RUNTIME_RECOVERY_CONCURRENCY` bounds
dispatch within the first eligible dependency tier; the safe default is one.
`VROOLI_RUNTIME_PRESSURE_SOME_AVG10` sets the Linux memory PSI `some.avg10`
threshold that creates or regresses an epoch (default `10`). The supervisor
also treats an advancing kernel `oom_kill` counter as pressure. When either
`/proc/pressure/memory` or `/proc/vmstat` is unavailable or malformed, the
host provider reports degraded evidence: the controller records no clear state
and starts nothing.

See [CODE: internal/scenarioruntime/recovery.go] and
[CODE: internal/runtimesupervisor/service.go].

## Staged rollout and rollback

Promote recovery in order: observe pressure and timelines only; enable
system-monitor liveness plus retention; enforce Test Genie admission limits;
declare one tier-0 critical workload; then add later dependency tiers after the
previous stage has stable evidence. Before each promotion, inspect
`vrooli runtime recovery inspect --json`, `/api/v1/admission`, and the
system-monitor pressure/process timelines.

Rollback is immediate and non-destructive: set the workload policy to
`--opt-out`, or set `--enabled=false` while retaining its declaration. This
prevents future automatic lifecycle starts without stopping a healthy running
workload. If host pressure evidence is degraded, recovery is already fail-closed.

## Agent-spawn recovery boundary

Agent-spawn recovery is owned by the control plane's `internal/recovery` broker.
Autoheal and Prompt Manager submit a scenario, reason, and requester; they do
not implement a private spawn ladder. The broker tries governed attachment,
fresh agent run, and native runner execution in order, then records operator
escalation when the budget is exhausted. Every attempt persists the reached
tier, remaining per-scenario budget, and outcome.

The broker derives child environments through the envkit `DelegatedAgent`
boundary. A run identity token is therefore not ambient process state: it is
kept only for an explicitly delegated agent child and is removed from foreign
scenario and resource boundaries.

## Functional health

An API may be live and return HTTP 200 from `/health` while its primary work
is unavailable. Such services expose a `functional` object with `healthy` and
an operator-facing `reason`. A false functional value changes an otherwise
healthy response to `degraded`; services that omit the object retain their
existing health semantics. Scenario status consumes this existing health
value, so functional refusal is visible to lifecycle and autoheal callers.

## The supervisor unit and boot recovery

The supervisor runs from a native user unit
(`vrooli-runtime-supervisor.service`, `com.vrooli.runtime-supervisor`) rendered
from `platformgo.RuntimeSupervisorDefinition`, the same
[service definition seam](native-service-definitions.md) as the autoheal loop
and the emergency watchdog. The `runtime_supervisor` safeguard converges it on
every `vrooli setup` and re-inspects it in the readiness phase, recording the
native validator's verdict and the unit's `NRestarts` and `Result` as evidence.
`vrooli runtime supervisor install --user` calls the same converge path.

At start the supervisor retires any predecessor session whose PID is dead
before it claims its own, and under the native unit it takes over a live peer
rather than exiting. Until 2026-09-02 a dead predecessor's unexpired lease made
the unit exit 1 and restart every five seconds until the lease lapsed, with the
reason visible only in `~/.vrooli/logs/runtime-supervisor.log`.
