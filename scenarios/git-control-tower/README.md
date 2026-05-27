# Git Control Tower (Scaffold)

Git Control Tower is being re-built from scratch using the modern `react-vite` scenario template.

This directory currently contains **scaffolding only** (PRD + requirements + lifecycle wiring). The operational targets are defined in `PRD.md` and tracked via `requirements/`, but business logic is intentionally not implemented yet.

## What This Scenario Will Become
- REST API for structured git operations (status, diff, stage/unstage, commit, branches, conflicts, previews)
- CLI wrapper for agent-friendly invocation
- Web UI dashboard for interactive review (diff viewer, staging, health)

## Baselines — the primary review primitive

A **baseline** captures a scenario's review surfaces (workflows, tests,
structure, visuals, rules) at a point in time so you can answer *"did my change
cause this failure, or was it already failing?"* without touching the working
tree — the regression-diagnosis replacement for `git stash`.

- **UI**: the **Baselines tab** is the cross-surface management view (set,
  compare, edit, delete). The Tests/Rules tabs show a "vs. baseline" diff, and
  the Workflows tab is a focused view of test-genie playbooks runs (with video).
- **CLI** (agent surface): `git-control-tower baseline {snapshot,diff,list,show,delete,create,edit}`.

A baseline owns no artifacts — it is a manifest of pointers into surfaces that
already store their own results (test-genie runs; GCT-local snapshots). See
[`docs/baseline-model.md`](docs/baseline-model.md) for the architecture,
branch-scoping, adapters, and concurrent-agent safety.

## Operational Targets
See `scenarios/git-control-tower/PRD.md:1`.

## Requirements Tracking
`requirements/` mirrors the PRD’s operational targets. Add tests tagged with `[REQ:<id>]` to automatically sync status.

Useful commands:
- `vrooli scenario requirements lint-prd git-control-tower`
- `vrooli scenario requirements report git-control-tower --format markdown`
- `vrooli scenario requirements sync git-control-tower`

## Running (Lifecycle Only)
Use the scenario Makefile:
- `make start`
- `make stop`
- `make logs`
- `make test`

## Notes
- No packages were installed as part of re-scaffolding; run lifecycle `setup` when you’re ready to install/build.
- The legacy implementation was moved to `/tmp` on this machine for reference/backups.
