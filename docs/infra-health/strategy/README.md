# Infra Health Strategy — Retired Folder

> **Retired 2026-08-20.** This folder held the team's reliability targets before
> the team had an instrument. Targets are now the checked-in operator setpoint at
> [`scenarios/infrastructure-manager/setpoint/reliability-setpoint.json`](../../../scenarios/infrastructure-manager/setpoint/reliability-setpoint.json),
> graded live against owner-authored reliability spaces. Read them with
> `infrastructure-manager coverage status` and `coverage show <projection>`.

## Strategy Documents

This folder is **empty**. `RELIABILITY_TARGETS.md` was retired and then deleted on
2026-08-20, once the out-of-folder links its deletion waited on were repointed at the
surfaces that own each concept: the honesty-flag vocabulary at the instrument's
`SETPOINT-MODEL.md`, the open-loop rule at its `COVERAGE-MODEL.md`, and per-team
reliability targets at the checked-in setpoint.

A folder that exists only to say "look elsewhere" is the next instance of the pattern
this cycle removed three times, so retiring the folder itself is the remaining step; it
is tracked in the plan of record's Future-PoR-work list.

## Extension Rules

Do not add documents to this folder. A target that can be graded belongs in the
instrument's setpoint, where it is compared against a live reading and cannot
drift out of agreement with the grid beside it. A judgment that cannot be graded
belongs in [`../operating/`](../operating/README.md) or
[`../evidence/`](../evidence/README.md).
