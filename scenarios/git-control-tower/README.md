# Git Control Tower

Agent-friendly git repository control plane: a REST/Connect API, CLI, and web UI for structured git operations and change review.

## What This Scenario Provides
- **Git operations API + CLI**: status, diff, stage/unstage, commit, branches/worktrees, history, blame, content search, discard — structured for agent invocation (`git-control-tower --help`).
- **Web UI**: Changes tab (optimistic staging, commit panel), diff viewer, history/blame, branch selector, file search.
- **Baselines**: the primary review primitive — see below.
- **Scenario review panel**: readiness reviews per scenario (`git-control-tower review {run,summary,status}`) running test-genie phases plus standards/tidiness/auditor dimensions.
- **Managed pre-commit hook**: configurable hook kinds (none/gct/user/framework) running exactly one command (default `vrooli hygiene`), with one-shot skip.
- **AI provenance**: the AI Changes tab groups pending changes by the agent-manager run that produced them (foundation built; live validation and committed-history attribution are tracked in the `git-control-tower-ai-provenance` initiative).
- **Credentials**: git credential/SSH resolution for fetch/push operations.

Not yet implemented (tracked as swarm-manager initiatives): merge/conflict resolution (`gct-merge-and-conflicts`), commit trailers linking to initiatives (`gct-commit-initiative-linking`), GitHub PRs/releases (`gct-github-integration`), agent-generated PR descriptions/release notes (`gct-release-pipeline`).

## Baselines — the primary review primitive

A **baseline** anchors one comprehensive Test Genie run at a point in time so
you can answer *"did my change
cause this failure, or was it already failing?"* without touching the working
tree — the regression-diagnosis replacement for `git stash`.

- **UI**: the **Baselines tab** captures, compares, and deletes immutable run
  anchors. Review panels render Test Genie's dynamic phase diffs and select
  typed evidence such as workflow videos by artifact kind.
- **CLI** (agent surface): `git-control-tower baseline {snapshot,diff,list,show,delete}`.

A baseline owns no artifacts. Test Genie owns the anchored run, descriptors,
comparison semantics, evidence catalog, and opaque artifact access. See
[`docs/baseline-model.md`](docs/baseline-model.md) for the architecture,
branch scoping, migration rules, and concurrent-agent safety.

## Operational Targets
See `scenarios/git-control-tower/PRD.md:1`.

## Requirements Tracking
`requirements/` mirrors the PRD’s operational targets. Add tests tagged with `[REQ:<id>]` to automatically sync status.

Useful commands:
- `vrooli scenario requirements lint-prd git-control-tower`
- `vrooli scenario requirements report git-control-tower --format markdown`
- `vrooli scenario requirements sync git-control-tower`

## Running
Use the scenario Makefile:
- `make start`
- `make stop`
- `make logs`
- `make test`
