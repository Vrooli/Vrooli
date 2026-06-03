# API Endpoints — Data Backup Manager

Human-readable reference for the API. The machine-readable source of
truth is [`.vrooli/endpoints.json`](../../.vrooli/endpoints.json), which
is regenerated from registered API modules by `make endpoints`.

Wire shapes live in
`packages/proto/schemas/data-backup-manager/v1/<domain>/`. Product
operations use generated Connect-RPC handlers and clients; `/health` is
the operational REST probe.

## System

| Method | Path | Purpose | CLI |
|---|---|---|---|
| `GET` | `/health` | Service readiness plus database and backup-posture dependency state. | `data-backup-manager status` |

## Targets

Connect service:
`/vrooli.data_backup_manager.v1.targets.TargetsService/<method>`

| Method | Purpose | CLI |
|---|---|---|
| `RegisterTarget` | Idempotently upsert a target keyed by owner and name. | `targets register` |
| `DeregisterTarget` | Remove a target by owner and name. | `targets deregister` |
| `GetTarget` | Fetch a target by id. | `targets get` |
| `ListTargets` | List registered backup targets. | `targets list` |

## Destinations

Connect service:
`/vrooli.data_backup_manager.v1.destinations.DestinationsService/<method>`

| Method | Purpose | CLI |
|---|---|---|
| `CreateDestination` | Create an encrypted kopia destination. | `destinations create` |
| `GetDestination` | Fetch a destination by id. | `destinations get` |
| `ListDestinations` | List configured destinations. | `destinations list` |
| `UpdateDestination` | Update cap bytes or cap policy. | `destinations update` |
| `DeleteDestination` | Delete the catalog record; v1 does not delete the underlying repo. | `destinations delete` |
| `GetDestinationUsage` | Read current usage from kopia stats. | `destinations usage` |

## Discovery

Connect service:
`/vrooli.data_backup_manager.v1.discovery.DiscoveryService/<method>`

| Method | Purpose | CLI |
|---|---|---|
| `ListTargetSuggestions` | Suggest local runtime state worth protecting. | `discovery targets` |
| `ListDestinationSuggestions` | Suggest safe destination volumes. | `discovery destinations` |
| `DismissSuggestion` | Hide a suggestion permanently. | `discovery dismiss` |

## Coverage

Connect service:
`/vrooli.data_backup_manager.v1.coverage.CoverageService/<method>`

| Method | Purpose | CLI |
|---|---|---|
| `GetCoverageReport` | Live first-backup readiness: registered vs recommended vs sensitive targets, plus planned/backed-up/verified posture. | `coverage report` |
| `AcceptDefaultTargets` | Bulk-register non-sensitive discovered durable targets (`include_sensitive` opts in credential/token targets; `dry_run` registers nothing). | `coverage accept-defaults` |

Coverage composes the discovery, targets, plans, runs and restores seams; it
reads no file contents and persists nothing of its own. `AcceptDefaultTargets`
returns per-item `accepted` / `skipped_sensitive` / `failed` lists so partial
failures are never swallowed.

## Plans

Connect service:
`/vrooli.data_backup_manager.v1.plans.PlansService/<method>`

| Method | Purpose | CLI |
|---|---|---|
| `CreatePlan` | Bind targets to destinations with schedule and retention. Blocked with `failed_precondition` when non-sensitive recommended targets are unregistered unless `allow_incomplete_coverage` is set. | `plans create` |
| `GetPlan` | Fetch a plan by id. | `plans get` |
| `ListPlans` | List backup plans. | `plans list` |
| `UpdatePlan` | Replace plan fields and membership. Same coverage guard as create via `allow_incomplete_coverage`. | `plans update` |
| `DeletePlan` | Delete a plan. | `plans delete` |

## Runs

Connect service:
`/vrooli.data_backup_manager.v1.runs.RunsService/<method>`

| Method | Purpose | CLI |
|---|---|---|
| `TriggerRun` | Execute a plan immediately. | `runs trigger` |
| `GetRun` | Fetch a run by id with outcomes. | `runs get` |
| `ListRuns` | List run history, optionally by plan. | `runs list` |
| `ListTargetStatus` | Show last-success and last-run status by target. | `runs status` |
| `BrowseSnapshot` | Browse snapshot contents. Requires a matching `resource-kopia` browse surface. | `runs browse` |

## Restores

Connect service:
`/vrooli.data_backup_manager.v1.restores.RestoresService/<method>`

| Method | Purpose | CLI |
|---|---|---|
| `RestoreTarget` | Restore a snapshot to a target location. | `restores restore` |
| `VerifyTarget` | Test-restore to scratch, verify, checksum, and record evidence. | `restores verify` |
| `GetRestore` | Fetch a restore/verify record by id. | `restores get` |
| `ListRestores` | List restore/verify records, optionally by target. | `restores list` |

## Updating This File

After endpoint changes, run:

```bash
make endpoints
jq '.endpoints[] | {method, path, summary}' .vrooli/endpoints.json
```

Keep this page as a concise operator/developer map. The generated
manifest and proto schemas remain the detailed contract.
