# Baseline Model

Git Control Tower's **baseline** subsystem is the regression-diagnosis primitive
that replaces `git stash` for the question every agent and reviewer asks
mid-change: *"did my change cause this failure, or was it already failing?"*

A baseline captures a scenario's review surfaces at a point in time. After you
change code, you diff the baseline against the working tree and each surface is
classified — so a pre-existing failure never gets mistaken for a regression you
introduced.

## Manifest of pointers (Decision 1)

A baseline **owns no artifacts**. It is a manifest of *pointers* into the
surfaces that already store their own results:

```mermaid
flowchart LR
  B[BaselineManifest<br/>data/&lt;repoID&gt;/baselines/&lt;scenario&gt;/&lt;branch&gt;/&lt;name&gt;.json]
  B -->|workflows| TG1[test-genie run<br/>runID]
  B -->|tests| TG2[test-genie run<br/>runID]
  B -->|structure| GS1[gct-local snapshot]
  B -->|visuals| GS2[gct-local snapshot]
  B -->|rules| GS3[gct-local snapshot]
```

- **workflows** and **tests** point at **test-genie runs** (`kind=test-genie-run`,
  `ref=<runID>`). test-genie owns run history; GCT *pins* the run
  (`gct:baseline:<name>`) so retention GC can't reclaim it while the baseline
  references it, and unpins on delete.
- **structure**, **visuals**, **rules** point at **GCT-local snapshots**
  (`kind=gct-local-snapshot`) — issue-set / screenshot snapshots GCT stores
  itself.

Because the manifest only holds pointers, capturing a baseline is cheap and the
same run can back several baselines.

## Branch scoping (Decision 2)

Baselines are stored per branch:

```
data/<repoID>/baselines/<scenario>/<branch>/<name>.json
```

The branch is the scoping axis because regressions are evaluated relative to the
branch you're working on. A detached HEAD scopes to `detached-<sha8>`. Listing
defaults to the current branch; `--all-branches` (CLI) or the UI list shows
every branch.

## Per-surface adapters

Each surface is captured/diffed by an adapter behind a narrow interface
(`scenarios/git-control-tower/api/internal/baseline/adapter_*.go`). To **add a
new surface**:

1. Implement the adapter interface (capture → pointer, diff → `SurfaceDiff`).
2. Register it in the service's surface map.
3. Add it to `BASELINE_SURFACES` in the UI (`ui/src/features/baselines/model.ts`)
   so the modal, chips, and diff routing pick it up.

The diff verdict vocabulary is shared verbatim with test-genie's classifier:
`clean` · `regression` · `new-failure` · `preexisting` · `not-comparable`.
`diff` exit codes: `0` safe (clean/new-failure/preexisting), `1` regression,
`2` not-comparable.

## Staleness

Staleness is computed from the baseline's recorded git sha vs. current HEAD —
commits since, plus files changed (including uncommitted edits and untracked
files). A baseline captured against a dirty tree records `git.dirty` so the UI
and CLI can warn that its surfaces may reflect uncommitted changes rather than
the committed state.

## Concurrent-agent safety

Baseline writes are branch-scoped and guarded by a per-branch file lock
(`flock`), so two agents capturing baselines on different scenarios (or
branches) never collide. Pins on test-genie runs are additive per consumer, so
concurrent baselines referencing the same run are safe. This is why baselines
replace `git stash`, which is process-global and unsafe when multiple agents
share a working tree.

## Surfaces over the wire

```mermaid
flowchart TB
  subgraph UI [GCT UI]
    BT[Baselines tab] --> BC[baselinesClient]
    WT[Workflows tab] --> WC[workflowReplayClient]
  end
  BC -->|Connect| BSVC[BaselinesService]
  WC -->|Connect| WRS[WorkflowReplayService]
  WRS -->|proxy, playbooks-only| RUNS[test-genie RunsService]
  BSVC --> ADP[surface adapters]
  ADP --> RUNS
  ADP --> AUD[scenario-auditor]
  ADP --> VIS[GCT visual capture]
  WT -. video bytes .-> VID[GET /repo/workflow-runs/&lt;id&gt;/video] --> TGART[test-genie artifact route]
```

- `BaselinesService` (Connect) — create/snapshot/get/list/diff/delete/edit.
- `WorkflowReplayService` (Connect) — the Workflows tab's read-only proxy over
  test-genie playbooks runs (Decision 3: the UI never calls test-genie
  directly). Binary video bytes stream over a GCT REST route, not proto.

## Surfaces

| Surface | Pointer kind | Backed by |
|---|---|---|
| workflows | test-genie-run | test-genie playbooks run |
| tests | test-genie-run | test-genie unit/integration/smoke run |
| structure | gct-local-snapshot | scenario-auditor structure scan |
| visuals | gct-local-snapshot | GCT visual capture (screenshots) |
| rules | gct-local-snapshot | scenario-auditor rules scan |

## Surfaces, CLI, and UI

- **CLI** (agent surface): `git-control-tower baseline {snapshot,diff,list,show,delete,create,edit}`.
  Always explicit — every command requires `--name`.
- **UI** (human surface): the **Baselines tab** is the cross-surface management
  view; Tests/Rules tabs read baseline diffs; the Workflows tab is a focused
  view of playbooks runs. A per-device "default baseline" (localStorage) lets
  the read-only tabs know which baseline to diff against; the CLI/API ignore
  this convenience and always require an explicit name (Decision 4).
