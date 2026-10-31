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

## Start and restart operation ceilings

The scenario control-plane RPC accepts `timeout_seconds` for start and restart.
A positive value supplies a deadline to the shared lifecycle operation; zero
leaves the default unbounded context, and negative values are rejected before
work begins. The shared application service also applies this ceiling to
library requests, preserving any earlier owning-context deadline or
cancellation. Cancellation is cooperative through lifecycle operations.
Deadline expiry returns the RPC `deadline_exceeded` code.

An RPC client disconnect does not itself acquire cancellation authority over
the operation. Use an explicit operation ceiling to bound server work. This
ceiling is separate from a demand lease, which controls scenario retention
after startup.

The start/restart RPC forwards path, best-effort, stale cleanup, forced setup,
credential-loss acknowledgement, and demand-managed supervision to the same
lifecycle options used by the CLI. Setup also forwards its custom path.
Browser opening and node/instance connection selection remain client-side
controls and are declared local-only in the CLI manifest. They are not silently
converted into server lifecycle options. Demand-managed callers must retain
renewable leases for their target throughout use.

## Bounded log snapshots

The log RPC supports named step logs, previous step backups, runtime logs,
and the default lifecycle log. CLI and RPC share source selection: runtime
selection takes precedence over a named step, then lifecycle is the default.
The source scan is bounded to 4,096 directory entries and 128 selected files;
aggregate RPC output, including source labels, shares the 1 MiB limit.
Streaming (`follow`, `force-follow`) and orphan cleanup (`clean`) are client-only
CLI controls. Program bindings reject explicitly supplied client-only controls
rather than silently discarding them. This also applies to browser opening and
connection selection on lifecycle bindings.

The control-plane log RPC and app-log HTTP endpoint read a bounded file suffix,
not the full log. The default is 50 lines, with a maximum request of 10,000 lines
and a 1 MiB output limit. The RPC uses `tail_lines=0` for the default; explicit
HTTP line counts must be positive. Invalid counts return an argument error.

If the requested tail cannot fit, the RPC returns `resource_exhausted` and the
HTTP endpoint returns 413. Neither returns a partial first line as a complete
tail. Request fewer lines or use the local scenario log command for larger
inspection. Snapshot reads use one opened file and reject observed truncation;
they do not provide a transactional snapshot of in-place log rewrites.
Special files are rejected before reading. Unix opens are nonblocking so that
a log replaced by a named pipe cannot wait indefinitely for a writer; the
opened descriptor is checked again before reading its contents. These checks
do not impose a deadline on regular-file I/O on an unresponsive filesystem.

## Demand leases

Committed lease acquisition, renewal, release and expiry, explicit retention,
and demand-stop transitions have transactional audit records. Inspect recent
evidence with `vrooli scenario demand history --scenario <name> --variant live
--limit 100 --json`. The registry retains the newest 1,000 demand transitions
per scenario across variants; reads filter one exact variant and return newest
first. Records exclude consumer metadata and are bounded to 8 KiB each.
Repeated release or signaling retries do not duplicate completed transitions.
Failure to persist the audit rolls back the associated state change.

This is bounded recent transition history: an empty result is not proof of no
historical demand. It excludes rejected requests and unchanged reconciliation
decisions, and it is not an indefinite audit archive. Unrelated runtime events
are outside its retention policy.

Process heartbeat leases answer whether a lifecycle owner is alive. Demand
leases answer whether a caller still wants a scenario available, so the two
signals must not be conflated. The runtime registry exposes bounded,
renewable demand leases with a scenario, variant, consumer, purpose kind, and
expiry. Acquisition is idempotent for a stable lease ID; renewal after expiry
fails with a typed error; expiry is an explicit transition that can be audited.
Durations default to ten minutes and are capped at 24 hours so a caller cannot
create an unbounded hold by accident.

Releasing a program hold or a transient dependency hold ends caller ownership
immediately and leaves a ten-minute warm interval for that scenario variant.
The interval slides from the latest release; calls do not accumulate idle
credit. Replaying a release does not extend it. Schema 11 records last use and
the deadline in `runtime_demand_idle_windows`, independently of active holds.
Lifecycle dependency, job, core, and explicit releases do not add warm time.

The lifecycle maintenance sweep and runtime supervisor tick now perform the
automatic stopping integration. They expire due holds, atomically claim only instances explicitly
started with the demand-managed supervision policy, and stop them after
process references identify the exact runtime instance, scenario, and variant.
In-flight `starting` instances remain under heartbeat and stale-start handling
until they are running.
Acquisition and renewal reject a variant already claimed for stopping. An
explicit operator start of a running demand-managed instance atomically
promotes it to managed supervision; a demand caller cannot demote it. A stop
that wins first must finish before a subsequent start can retain that instance.
Ordinary operator, core, manual, and autoheal-supervised instances are not
eligible. Before signaling begins, a failed identity or host-evidence check
restores the instance to running and records a failed maintenance result. After
signaling begins, any failure retains the stopping state and blocks new demand
until recovery finishes. A running process
reference without a valid PID is incomplete evidence and prevents stopping;
it is never interpreted as an exited process. Cancellation before teardown
restores the stopping claim on a fresh, bounded cleanup context. Once signaling
begins, teardown bookkeeping uses a separate ten-second context so caller
cancellation cannot interrupt the remaining registry updates. Forced signals
are dispatched to all surviving members before a shared two-second exit wait.

The supervisor integration regression uses a disposable real TCP listener and
an isolated registry. It verifies retention across lease renewal, process exit
and port release after expiry, and preservation of a process whose variant
does not match the claimed instance. Linux demand shutdown opens a stable
process handle before identity inspection and checks that the same incarnation
is still alive after inspection. Both graceful and forced signals use that
handle; PID reuse cannot redirect those signals. Hosts without stable-handle
support fail preflight instead of falling back to numeric PID signaling.

Recorded process groups are expanded into individually inspected workers, with
a maximum of 1024 process references per attempt. Every live member must identify
the exact instance, scenario, and variant before any member is signaled. Foreign
members preserve the whole group and its port claims. Real-process regressions
cover untracked workers and forced shutdown of members that ignore SIGTERM.
A final group check retains port claims if a member appeared during teardown.
This does not establish containment of workers that escape their recorded group.

Newly launched background services use the variant-aware record slug for
best-effort scope placement. Scope names retain a readable prefix plus a digest
of the original slug and step, so sanitization, tuple boundaries, and truncation
do not collapse distinct identities. Existing running scopes are not renamed.
Scope placement remains best-effort and does not supply a durable containment
proof for demand teardown.

Schema 12 persists a demand stop intent in the same transaction that claims an
idle instance. Maintenance and supervisor ticks discover these pending intents,
including attempts interrupted after process termination. A registry-local file
lock serializes teardown owners across processes and resolves database symlink
aliases to the same lock. It is held through finalization and has no timed
ownership takeover; the kernel releases it when its owner exits. Lock waits
are bounded, and a busy owner defers the rest of that sweep's demand stops.
The adjacent `.demand-stop.lock` file must not be removed while controllers run.

The owner records the signaling boundary durably before sending any signal.
Recovery repeats identity checks, but cannot restore a partially stopped
instance to running. Port release, the stopped state, and removal of the intent
commit atomically. Generic expiry, stuck-instance reapers, and ordinary claim
release cannot bypass an active demand stop intent. Regression evidence covers
an owner process killed while holding the lock, an injected persistence failure,
a new controller resuming a completed process shutdown, and atomic rollback of
port release when finalization fails.

The 11-to-12 upgrade preserves existing runtime and demand data. It does not
infer ownership of older stopping rows from free-form stop reasons: legacy
rows without an intent require operator inspection. Older binaries reject the
new schema, so deploy the matching control-plane and supervisor builds together.

Program Runtime and
opt-in API clients acquire and renew leases explicitly. Callers that also have
startup capability may use the demand-aware resolver: it acquires the lease
before starting a stopped target, passes `--demand-managed` to lifecycle, and
retries discovery after startup. A direct demand-managed start still requires
a caller to hold and renew a lease. The public control-plane transport is:

```bash
vrooli scenario demand acquire --scenario search-hub --consumer session-1 \
  --kind program --lease-id request-1 --request-id request-1 --json
vrooli scenario demand renew --lease-id request-1 --json
vrooli scenario demand release --lease-id request-1 --reason completed --json
```

Program Runtime uses this transport around governed binding calls. Discovery
callers can opt in with `ResolveScenarioURLWithDemand`; the ordinary resolver
helpers remain read-only so health probes do not keep scenarios alive.

The supervisor status exposes policies and retained pressure epochs so operators
can distinguish observed pressure, a recovery gate, queued work, restoration,
skips, and terminal failures as the controller is enabled in later rollout
stages.

## Safety contract

- Declared recovery is disabled by default.
- A recovery decision is deduplicated by a durable idempotency key.
- Pressure epochs and decisions survive supervisor restart.
- No policy is inferred from a runtime lease, process ID, or prior start.
- Demand-based stopping requires the explicit `demand` supervision policy and
  an atomic no-active-demand claim; a stale heartbeat alone cannot authorize
  a stop.
- Process termination requires exact runtime identity evidence immediately
  before signaling; recycled, foreign, or unreadable PIDs are never killed.
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

## Dependency ownership

Lifecycle dependency bootstrap uses stable consumer identities derived from the
parent scenario instance and bounded lease identities derived from the
dependency instance. Repeated starts are idempotent. Failed dependency starts
release holds acquired by that branch, and a parent start failure releases the
holds acquired by its recursive start graph. Explicit stops and restarts release
the previous dependency graph before a replacement graph is acquired.

Dependency leases are renewed by the same maintenance path that expires demand
leases. Renewal is keyed to active runtime instances, so a crashed or stopped
consumer cannot keep a dependency demanded indefinitely. A dependency started
for this purpose opts into demand supervision; an existing managed, manual, or
explicit/core instance keeps its stronger supervision policy. Test Genie job
ownership and broad discovery adoption remain separate integration boundaries.

Autoheal now renews a stable `core` lease for every scenario in the fresh
canonical supervision set. The lease is a protection pin for a scenario that
was started through another demand-aware path: it does not launch or stop a
process, and it is released only after a non-degraded supervision refresh
removes the member. A control-plane outage retains the last-known-good checks
and their pins. Lease failures leave the checks active but make the source
check warning-visible, so a transient runtime failure cannot silently weaken
core supervision.

Test Genie suite runs use the same boundary with `job` leases. Before a target
run probes or starts its scenario, it acquires a run-scoped lease and starts a
new target with `--demand-managed`. The shared `demand.AcquireLifetime` hold
keeps long suites alive. Startup and phase work use its cancellation context:
a failed renewal cancels work and records the cause in the terminal failure,
even if a runner ignores cancellation and reports success. Cancellation stops
further phase admission. Cleanup releases the hold for both targets Test Genie started and targets that
were already running. Cleanup does not stop a demand-managed target directly:
the control-plane reconciler checks all remaining demand and current ownership
before stopping it. This preserves consumers and operator starts acquired
during a suite. Self-host and source-only targets do not acquire a hold.

Test Genie also holds provider exclusion locks while validation RPCs run, so
another suite cannot restart the provider mid-call. Readiness and phase-owner
lock acquisition honor the suite context. Cancellation abandons the wait and
releases any earlier provider holds; it never revokes the active owner's lock.
These exclusion locks govern validation concurrency, separately from runtime
demand leases that govern process retention.

During dependency bootstrap the parent runtime row may not exist yet. The
reconciler therefore also recognizes the parent's durable running start
operation as a bounded startup grace window; an abandoned or terminal start
operation does not retain the hold.

Short-lived control-plane delegations use the same ownership boundary. Plan
manager reads and reconciliation, capability ledger reads, structure-health
validation, and tidiness validation acquire a `dependency` lease for the
operation and release it on every return path. Their consumer identities are
stable, while bounded invocation identities distinguish concurrent operations.
Injected or static test clients can leave demand disabled; ordinary discovery
remains read-only unless a caller explicitly opts into demand ownership.

Maintenance renews and releases only the lifecycle-owned dependency identity
namespace. Transient CLI and watcher owners release their own holds; absence
from the scenario registry is not evidence that those callers stopped.
`demand.AcquireLifetime` supplies a renewable hold with a cancellation context
and idempotent, bounded cleanup. Protected work uses `Hold.Context()` and ends
if renewal fails or the last granted expiry arrives. Renewal scheduling follows
the earlier of the requested cadence and the granted remaining lifetime. Each
response must carry an active lease, a future expiry, and the same lease ID;
invalid evidence cancels ownership rather than extending it. An independent
expiry timer cancels work even while a renewal transport is slow, and late
responses cannot revive the cancelled hold. Lease clients must honor context
cancellation so Close can finish its bounded cleanup. `AcquireScoped` is for operations shorter than their lease
TTL and provides bounded, idempotent cleanup without a renewal worker.

The standard API server ties event discovery, polling, and subscription workers
to its shutdown lifetime. After discovering Events it acquires a dependency
hold, renews while watching, and releases on shutdown or renewal failure.
Discovery retries asynchronously, so an absent Events service does not block
business requests or automatically start Events. An explicitly injected Events
URL remains caller-managed. Independent demand-aware resolutions receive
distinct request identities by default; pass a request ID only to retry the
same ownership scope. Variant resolution and lease identity address the same
scenario instance.
