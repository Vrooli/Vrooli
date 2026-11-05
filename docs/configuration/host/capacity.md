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

Set `capacity_posture` in `~/.vrooli/state/vrooli/vrooli-onboarding/operator-state.json` through an
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

This derives the request from the resource manifest and refuses startup unless
that admission establishes the manifest's full preferred budget without
displacing another owner. A lower-rung recommendation cannot launch an unchanged
resource workload. Do not create a manual
resource claim first or copy its profile, idle policy or GPU defaults into a
second request. Ordinary starts retain advisory behavior. The admission constraint
does not change the resulting claim's future yield policy. This admits the
manifest's declared resource kind; separate per-operation RAM/CPU estimates still
need their own admission.

Enforced admission succeeds only when required reclaim callbacks complete and
release enough capacity. A callback failure or partial reclaim returns a denied
execution verdict; the stored claim remains observable until its owner releases
it or its lease expires. Advisory admission describes a placement decision and
does not itself execute reclaim or prove model execution.

Inference owners bind the admitted physical GPU in the worker process, keep the
lease alive while work runs, reap the worker before release, and surface cleanup
failure with the claim identity. Image Tools uses a ten-minute lease and bounded
cleanup independent of the request's cancelled context.

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
