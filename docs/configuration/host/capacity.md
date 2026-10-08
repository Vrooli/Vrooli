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

Use `vrooli capacity fit --json` when another program must consume the result.
Fit is read-only. It does not change operator state or stop a resource.

## Select a posture

Set `capacity_posture` in `~/.vrooli/operator-state.json` through an
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

Claims describe current reservations and activity. Footprints persist measured
high-water marks across terminal-claim garbage collection. Recommendations can
report both over-reservation and measured over-consumption. A configuration
without a measured sample does not receive a right-sizing recommendation.

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
