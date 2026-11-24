# Host capacity

Vrooli records finite host capacity in a local SQLite ledger. The control plane
uses the ledger to explain resource placement and recommend changes. The
default policy is advisory and does not apply a recommendation automatically.

## Inspect the enabled set

Run the fit command before you enable or resize GPU-backed resources:

```bash
vrooli capacity fit
```

The command reports one of these verdicts:

- `fits`: The selected rungs and transient reserve fit this machine.
- `fits_with_changes`: The set fits after the listed rung or tunable changes.
- `over_subscribed`: The set does not fit. Review the listed offenders and
  proposal before you start more workloads.

`over_subscribed` is a completed evaluation, not a transient command failure.
The CLI exits with status 1 for any verdict other than `fits`, so shell callers
must preserve and inspect the report instead of retrying the unchanged request.

Use `vrooli capacity fit --json` when another program must consume the result.
Fit is read-only. It does not change operator state or stop a resource.

## Select a posture

Set `capacity_posture` in `operator state` through an
operator-state surface. Supported values are `responsive`, `balanced`,
`throughput`, and `minimal`. The posture selects default priority tiers, rungs,
and transient headroom.

Set `accel_preference` to `auto`, `prefer_gpu`, or `force_cpu` to override
artifact acquisition. When this field is absent, `minimal` derives
`force_cpu`; all other postures derive `auto`.

An acquisition target must still provide a hash-pinned closure for the selected
variant. `force_cpu` does not make a CUDA lockfile into a CPU closure.

## Inspect claims and evidence

```bash
vrooli capacity list
vrooli capacity footprint list
vrooli capacity recommend
vrooli capacity policy get enforce
```

Memory claim amounts use bytes (suffixes such as `GiB` are accepted). CPU
claim amounts use millicores: `--resource-kind cpu --preferred 4000 --floor 4000`
requests four logical cores. A value of `4` requests only 0.004 cores. The
legacy JSON field names ending in `_bytes` also carry millicores for CPU claims.
These reservations are advisory; they do not configure worker thread counts.
Admission measures used CPU from real utilization (the busy share of a short
`/proc/stat` window), not the load average, which counts tasks blocked on disk
and overstates use under I/O or swap pressure. Where utilization cannot be
measured, the load average is used and the verdict carries a warning.

Claims describe current reservations and activity. Footprints persist measured
high-water marks across terminal-claim garbage collection. Recommendations can
report both over-reservation and measured over-consumption. A configuration
without a measured sample does not receive a right-sizing recommendation.

Capacity reads are point-in-time snapshots. Use a claim or mutation response for
its current claim ID and generation. Read `capacity list` once before a decision
or after a real state transition; if a mutation reports a stale generation,
read `vrooli capacity list --owner <owner-id> --active` once before deciding
whether to retry. When waiting for a worker to release its claim, wait on that
worker or lifecycle operation instead of polling the full ledger.

## Bound transient inference

Use `vrooli capacity claim ... --no-reclaim` for work that may use already-free
capacity but must not displace another owner. This policy is independent of
priority: it also excludes idle, yield-enabled claims from potential reclaim.
It does not rewrite the new claim's priority or yielding policy.

For a resource experiment, use the resource lifecycle directly:

```bash
vrooli resource start songgen --no-reclaim
```

A resource owner may pass `--capacity-claim-id <id>` with
`vrooli resource start <name> --no-reclaim` to reuse an active claim that it
already admitted. The lifecycle verifies the claim belongs to that resource
and still follows its manifest priority and idle policy before it starts. The
owner can include `--capacity-claim-min-bytes <bytes>` to reject a claim that
was resized below the amount previously confirmed.

This derives the request from the resource manifest and refuses the lifecycle
operation unless that admission establishes the manifest's full preferred
budget without displacing another owner. A lower-rung recommendation cannot
launch an unchanged resource workload. Do not create a manual
resource claim first or copy its profile, idle policy or GPU defaults into a
second request. Ordinary starts retain advisory behavior. The admission constraint
does not change the resulting claim's future yield policy. This admits the
manifest's declared resource kind; separate per-operation RAM/CPU estimates still
need their own admission.

Enforced admission succeeds only when required reclaim callbacks complete and
release enough capacity. A callback failure or partial reclaim returns a denied
execution verdict; the stored claim remains observable until its owner releases
it or its resource-claim heartbeat policy expires. Advisory admission describes a
placement decision and does not itself execute reclaim or prove model execution.

Inference owners bind the admitted physical GPU in the worker process, keep the
lease alive while work runs, reap the worker before release, and surface cleanup
failure with the claim identity. Image Tools uses a ten-minute lease and bounded
cleanup independent of the request's cancelled context.

## Resource idle policy comes from the manifest

A resource's `acceleration.claim` block in `resource.json` is the one source of
its claim's tier, reservation ladder, `yield_when_idle`, `idle_grace_seconds` and
`idle_unload_ttl_seconds`. Lifecycle admission and every `capacity-sync`
companion read that block, so a claim carries the same policy whichever created
it. Companions pass it to `vrooli capacity claim` through `--idle-grace` and
`--idle-unload-ttl`; no resource restates these values in Go, and a companion
that cannot read its manifest refuses to start instead of claiming without them.

No verb edits a claim's idle policy in place. When a granted claim's policy
differs from the manifest, the companion releases it once and claims again with
the declared values, carrying the activity state across. Degraded claims keep
their mode and converge when next created.

`capacity admit` is read-only. A refusal reports the free capacity it counted,
the amount held by claims reclaimable at that priority, and the total if those
claims all went idle. A `--no-reclaim` request counts free capacity only.

## Enforcement safety gate

Keep `enforce` set to `advisory` until all of these checks pass in one recorded
drill:

1. Confirm that work owners have reported non-idle activity.
2. Confirm that `vrooli capacity fit` returns a successful verdict.
3. Degrade and upshift one claim successfully.
4. Confirm that the lowest-priority idle claim changes first.
5. Confirm that no active claim changes.

If a check fails, leave enforcement in advisory mode and repair the owning
layer first.

## Storage

The authoritative ledger is `~/.vrooli/state/capacity.db`. The old
`~/.vrooli/capacity.json` file is not part of the current contract.

## Admit disk jobs

Disk work uses the same host capacity ledger. Reserve every directory that can
receive bytes: final output, download cache, unpack/staging tree, tool/model
cache, and temporary files. Give each path a peak additional-byte estimate. The
capacity owner canonicalizes paths and filesystem identities, deduplicates
canonical aliases, combines each filesystem's peak, and atomically admits the
whole job only when it leaves at least 120 GiB free after all active leases.
Production callers cannot lower that floor. A canonical path that resolves to
a different filesystem after admission is refused during extension or sampling.
A point-in-time `df` or `statfs` reading is not admission.

A job that writes only small metadata, such as publishing a lockfile or manifest
without downloading or linking anything (or a Go install of first-party or already
cached modules, which rewrites go.mod and go.sum), declares `"class": "metadata"` in its
request. The owner admits it against an 8 GiB floor instead of 120 GiB, so a host
that is merely inside the install floor does not stop a lockfile from being
published, while a host that is truly out of space still refuses. The class is
bounded: the job's declared peak may total at most 64 MiB, and `disk-extend` or a
measured peak extension past 64 MiB is refused (reserve an ordinary job instead).
A metadata job is held to its own floor only, not to the stronger floor an active
install promised for its bulk writes; its peak still counts in every other job's
admission and it never lowers the floor any other job is held to. Production
callers cannot lower the 8 GiB floor either. Anything that can write a package
store, `node_modules`, a model, an artifact or any other bulk output is an
ordinary job.

A lease is a heartbeat promise. Renew it with `disk-heartbeat` (or `disk-extend`
/ `disk-sample`) before the 15 minute deadline passes. Keep it from before the
first write through publication and cleanup, and call `disk-extend` before
writing beyond the admitted peak. Call `disk-complete` only after output and
cleanup are verified; the ledger then records each filesystem's observed
free-byte delta against its admission baseline. `actual_delta_basis` identifies
this as the filesystem available-byte decrease: it is a host-wide net
measurement and can include unrelated writes or deletions. Path samples
separately preserve the owner-supplied monotonic byte counts and their
baselines; report both without claiming the host-wide delta is attributable to
one job.

Once the deadline passes the reservation is no longer capacity anyone holds:
admission stops counting it at once, and the next ledger pass (any admission,
`disk-list`, `disk-observe`, or the maintenance sweep) settles it. The job
closes as `reconciled` with `cleanup_verified=false` and a legacy reservation
closes as `expired`. This applies whether or not the owner process is still
alive, because a lease measures the heartbeat, not the process, and a hold that
nobody renews only withholds capacity from every other job. Settling releases
the peak the job had reserved but not yet written; bytes it already wrote stay
visible in the measured free space, so admission remains correct. It does not
claim cleanup happened. An owner whose process exited before the deadline keeps
its lease until the deadline, because detached children may still be writing.

A lapsed lease cannot be revived. `disk-heartbeat`, `disk-extend` and
`disk-sample` refuse it with `capacity: disk lease lapsed after its heartbeat
deadline`; reserve again before writing more. `disk-complete` is still accepted
from the live owner of a settled job, because completing attests that output and
cleanup were verified, and it upgrades the receipt to `completed` with
`cleanup_verified=true`. If the owner crashed or was cancelled, a settled job
needs no operator step; `disk-reconcile` remains the way to attest cleanup
after the fact, once the capacity owner confirms the bound PID/start-time
identity is no longer live.

A peak is the bytes this job will add to that path, never the size of the
directory. A shared cache (`~/.npm`, a pnpm store, `~/.cache`, `/tmp`) is
declared with only this job's growth, measured with a bounded probe before
acquisition the way the dependency gateway measures package growth. A job that
cannot measure it declares what it has evidence for and raises the peak with
`disk-sample --extend-peak` as it grows. Round ceilings (10 GiB on one cache,
3 GiB on each of three more) are paid for by every other job until the lease
ends.

Owners may also record exact path-byte baselines and monotonic checkpoints with
`disk-sample`. Each receipt is bound to the lease credential, process identity,
declared canonical path and current filesystem identity. Use the same counting
method for every sample. A sample above the path's admitted peak is an overrun:
stop writes and request `--extend-peak` before resuming. Sampling and peak
extension share one admission-locked transaction and extension is refused when
combined reservations or the strongest floor would be violated. A rejected
extension writes no sample receipt. The first sample on an already active lease
establishes a new baseline only; it cannot reconstruct growth before that
sample or prove remaining allowance against the lease's original whole-job
peak. An extension request must include at least one path; an empty request is
rejected without returning lease details.

For host paths that are intentionally inaccessible to the CLI user, such as
Docker-managed volume sources and writable upper layers, declare the path with
`docker_container` and exactly one of `docker_destination` or
`docker_upper_dir: true`. The capacity owner verifies the exact source using
the Docker daemon, then checks the filesystem device against the kernel mount
table and an accessible `statfs` probe. It does not accept a caller-supplied
filesystem ID. These Docker identity calls have a bounded 15-second timeout
and fail closed on timeout. `disk-sample` uses matching `--docker-container`
and destination or upper-dir flags to repeat that identity check for the
checkpoint.

To reserve Docker storage before a new volume or container exists, declare the
daemon root itself with `"docker_root_dir": true` and the exact path reported
by `docker info --format '{{.DockerRootDir}}'`. The capacity owner obtains that
path from the daemon and verifies its filesystem through the host mount table;
it requires the reported path to be a real directory and does not require a
running container. Treat this as an aggregate peak
envelope for all planned writes under Docker's root. Keep it active through
provisioning and cleanup. Size this initial envelope for the bounded writes
needed to create the volume/container and discover its exact mount paths. Once
those paths exist, use `disk-extend` to add their post-provision growth peaks
before writing data. This partitions the initial provisioning allowance from
the later per-path allowances. Use the root selector for `disk-sample` only
when the owner can measure the Docker root with a repeatable byte-count method;
directory-specific volume/upperdir samples remain preferable when those paths
exist.

The job owner keeps the secret credential in a private file (mode 0600). A
one-shot CLI command cannot own its own future heartbeat, so reserve requests
must include the long-running producer's PID and process start token. On Linux,
`process_start` is field 22 from `/proc/<pid>/stat` (the start-time ticks). The
CLI's heartbeat process must observe that same PID/start pair before it mutates
the lease.

Example request file:

```json
{
  "owner_id": "image-tools:model-install",
  "job_id": "job-123",
  "process_id": 24680,
  "process_start": "98351722",
  "floor_bytes": 128849018880,
  "paths": [
    {"path": "/home/me/.cache/huggingface", "role": "model-cache", "peak_bytes": 1073741824},
    {"path": "/home/me/.vrooli/image-tools/models", "role": "staging-and-final", "peak_bytes": 2147483648},
    {"path": "/tmp", "role": "temporary", "peak_bytes": 268435456}
  ]
}
```

For pre-provision Docker work, a path can instead bind the daemon root:

```json
{"path":"/var/lib/docker","role":"docker-root-preprovision-envelope","peak_bytes":1073741824,"docker_root_dir":true}
```

Operate the lease with the capability file; reserve prints a receipt but never
prints the capability:

```bash
vrooli capacity disk-reserve --request-file request.json --lease-file lease.json --json
vrooli capacity disk-heartbeat --lease-file lease.json --json
vrooli capacity disk-extend --lease-file lease.json --paths-file extra-paths.json --json
vrooli capacity disk-sample --lease-file lease.json --path /output --bytes 1048576 --json
vrooli capacity disk-sample --lease-file lease.json --path /output --bytes 2097152 --extend-peak 4194304 --json
vrooli capacity disk-sample --lease-file lease.json --path /var/lib/docker --bytes 1048576 --docker-root-dir --json
vrooli capacity disk-complete --lease-file lease.json --cleanup-verified --json
vrooli capacity disk-list --active --json
vrooli capacity disk-list --observe --json
```

`disk-list --observe` refreshes free space on filesystems represented by active
disk leases, sums all live legacy and job peaks, applies the strongest active
floor, and reports the remaining safe additional peak. A negative value means
current free space is already below the reserved peaks plus floor. This view is
read-only and is never an admission decision; only the atomic reserve operation
can admit a new job.

The completed ledger receipt is the authoritative audit record. It includes the
owner/job and process identity, canonical paths and filesystem IDs, admitted
peaks, floor and free-space readings, heartbeat/terminal state, and actual
filesystem delta. Unknown package or model sizes need a bounded metadata probe
before acquisition; fail closed when a safe peak or full write-path inventory
cannot be established.
