# Setpoint

This directory holds **the bar** — the deadband every instrumented cell is
graded against.

It is a checked-in declarative file, parsed at query time by the `coverage`
domain. **No API path writes it. No table mirrors it. No migration creates
one.** That absence is not an oversight or a phase-one simplification — it is
the mechanism that prevents deviation `D6`, an observer writing its own
reference model. See [`../docs/concepts/SETPOINT-MODEL.md`](../docs/concepts/SETPOINT-MODEL.md).

## Changing a bar

1. Edit `reliability-setpoint.json`.
2. Set `decision_ref` on the changed entry to the approved
   `reliability-target-update` decision id.
3. Open the change for review like any other diff.

Tightening requires 30+ consecutive in-band days of `measured` data.
Loosening requires sustained out-of-band `measured` data with a named
non-temporary cause. Both are operator decisions; this scenario proposes
neither.

## What must not appear here

- **A cell that does not exist in any owner's space.** Bars grade cells; a bar
  with no cell is an integrity finding.
- **A bar equal to the reading it was authored against.** That is a dead
  deadband: it reports in-band while the defect stands and can only ever detect
  growth.
- **A hand-set `honesty_flag`.** Honesty is derived from coverage status.
- **Anything describing *which* cells exist.** That half of the denominator
  belongs to each control layer's `docs/spaces/<projection>-space.md`.
