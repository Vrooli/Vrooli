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

## `baseline` — Cross-surface review baselines (replaces `git stash` for diagnosis)

A baseline is a manifest of pointers into a scenario's review surfaces
(workflows, tests, structure, visuals, rules) captured before a change.
`diff` it afterwards to ask "did my change cause this failure, or was it
preexisting?" without touching the working tree. Backed by the
`BaselinesService` Connect-RPC.

| Subcommand | Description |
| --- | --- |
| `snapshot` | Capture a baseline (`--scenario --name [--branch] [--include w,t,...] [--fast\|--full] [--reason]`) |
| `diff`     | Diff against the working tree (`--scenario --name [--branch] [--surface]`); exit `1` on regression, `2` on not-comparable |
| `list`     | List baselines (`--scenario [--branch] [--all-branches]`) |
| `show`     | Show one baseline (`--scenario --name [--branch]`) |
| `delete`   | Delete a baseline and unpin its test-genie runs (`--scenario --name [--branch]`) |
| `create`   | Create an empty baseline, no capture (`--scenario --name [--branch]`) |
| `edit`     | Re-point a surface at a pinned test-genie run (`--scenario --name --surface --pin-run <runID>`) |

All subcommands accept `--json` for machine output. `diff` exit codes:
`0` safe to proceed (clean, or only new/preexisting failures), `1` regression
(something that passed at baseline fails now), `2` not-comparable.

## CLI–API parity gaps

Several API surfaces don't have CLI commands yet (e.g. visual capture,
workflow capture, agent runs, tidiness, rules, SSH key management,
credentials). These are intentionally UI-first today; surface them in
the CLI when an agent flow needs them.
