# CLI Reference: git-control-tower

Four command groups. Source of truth: [CODE: cli/domains/domains.go].
Each subcommand is a thin wrapper over the API; business logic lives in
the API.

Run `git-control-tower <group> --help` or `git-control-tower <group>
<subcommand> --help` for full flag listings.

## `repo` — Inspect and change repository state

| Subcommand | Description |
| --- | --- |
| `status`       | Show repository status (branch + changed files) |
| `diff`         | Show git diff (`--path=FILE --staged`) |
| `stage`        | Stage files (`FILE...` or `--scope=scenario:name`) |
| `unstage`      | Unstage files (`FILE...` or `--scope=scenario:name`) |
| `commit`       | Create a commit (`-m MESSAGE [--conventional]`) |
| `sync-status`  | Check push/pull status (`[--fetch] [--remote=NAME]`) |

## `branch` — Manage repository branches

| Subcommand | Description |
| --- | --- |
| `list`     | List branches (with ahead/behind counts) |
| `create`   | Create branch `NAME [--from=BASE] [--no-checkout] [--allow-dirty]` |
| `switch`   | Switch branch `NAME [--allow-dirty] [--track-remote]` |
| `publish`  | Publish current branch (`[--remote=NAME] [--branch=NAME] [--fetch]`) |

## `review` — Run and inspect scenario readiness reviews

| Subcommand | Description |
| --- | --- |
| `summary`  | Show readiness review for a scenario |
| `run`      | Run readiness checks and show results |
| `status`   | Check status of a review run job |

## `audit` — Query git-control-tower audit logs

| Subcommand | Description |
| --- | --- |
| `list`     | Query audit logs (`[--operation=TYPE] [--limit=N]`) |

## `baseline` — One-run review baselines (replaces `git stash` for diagnosis)

A schema-V2 baseline pins exactly one immutable comprehensive Test Genie run.
It preserves the run's Git/tree identity, captured descriptor catalog, dynamic
phase results, and typed evidence references. `diff` it afterwards to ask "did
my change cause this failure, or was it preexisting?" without touching the
working tree. Backed by the `BaselinesService` Connect-RPC.

| Subcommand | Description |
| --- | --- |
| `snapshot` | Start capture from one durable comprehensive run (`--scenario --name [--branch] [--run]`); `snapshot status --run R [--wait]` reattaches |
| `diff`     | Start a durable descriptor-driven comparison (`--scenario --name [--branch] [--wait]`); `diff status --run R` reads standing and `diff wait --run R` reattaches |
| `list`     | List baselines (`--scenario [--branch] [--all-branches]`) |
| `show`     | Show one baseline (`--scenario --name [--branch]`) |
| `delete`   | Delete a baseline and unpin its single Test Genie run (`--scenario --name [--branch]`) |

All baseline review subcommands accept `--json` for machine output. Terminal
diff exit codes are `0` safe to proceed (clean, or only new/preexisting
failures), `1` regression, `2` not-comparable, and `3` not ready. Snapshot and
diff start calls persist an operation ID before background work begins. Ctrl-C,
transport timeout, or unexpected EOF detaches the caller and never aborts or
restarts the operation; status/wait performs one reattachment by durable ID.

### Dynamic phases and typed evidence

GCT does not map Test Genie phases into local surfaces. Capture starts one
comprehensive run, pins it exactly once, and comparisons preserve every
`PhaseDiff` plus typed reason from `RunsService.CompareRuns`. New, retired,
inapplicable, skipped, unavailable, and unknown phases remain visible without a
GCT code change. Screenshots, recordings, logs, reports, and unknown artifact
kinds are referenced by opaque typed artifact IDs rather than filesystem paths.

Legacy V1 manifests migrate only when every non-empty pointer names the same
Test Genie run. Empty, partial, obsolete local-snapshot, corrupt, or mixed-run
manifests remain diagnostic and require recapture; the CLI never chooses one
pointer heuristically. Migration and pin reconciliation are idempotent, and a
failed unpin leaves the manifest intact so deletion can be retried safely.

## CLI–API parity gaps

Several API surfaces do not have CLI commands yet (for example descriptor-aware
EvidenceService history/search, visual capture, agent runs, tidiness, SSH key
management, and credentials). EvidenceService is intentionally UI-first;
agents use Test Genie's canonical runs CLI and GCT's baseline commands.
