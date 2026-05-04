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

## CLI–API parity gaps

Several API surfaces don't have CLI commands yet (e.g. visual capture,
workflow capture, agent runs, tidiness, rules, SSH key management,
credentials). These are intentionally UI-first today; surface them in
the CLI when an agent flow needs them.
