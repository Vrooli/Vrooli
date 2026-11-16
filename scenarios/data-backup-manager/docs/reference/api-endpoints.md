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
| `AcceptDefaultTargets` | Bulk-register non-sensitive targets; `include_sensitive=true` is rejected and sensitive targets require a verified decision (`dry_run` registers nothing). | `coverage accept-defaults` |
| `RequestSensitiveReview` | Raise one verified Notification Hub review for current sensitive suggestions; only the signed callback can register approved targets. With the standing rule on it asks nothing: it applies the rule and returns `pending=false`. | `coverage review-sensitive` |
| `GetSensitivePolicy` | The standing rule: on or off, who consented (verified identity), when, through which Hub decision, what it added, what it keeps out, history. | `coverage sensitive-policy` |
| `AdoptSensitiveConsent` | Record the operator's own-words answer on a sensitive coverage decision as the standing consent. The ask is read back from the Hub as DBM's ask-reader principal and must be this scenario's decision, answered by a verified human with Other free text; the record keeps the text verbatim, the ask id, the answer time and the identity. The rule is applied at once and the stale local review closed; nothing is raised at the Hub. | `coverage adopt-sensitive-consent` |
| `RevokeSensitivePolicy` | Turn the standing rule off; what it added stays registered. Recorded with who and when. | `coverage revoke-sensitive-policy` |
| `ExcludeSensitiveTarget` | Keep one owner/name item out of the standing rule. | `coverage exclude-sensitive` |

Coverage composes the discovery, targets, plans, runs and restores seams; it
reads no file contents and persists nothing of its own, apart from the standing
rule for sensitive items (`sensitive-coverage-policy/policy.json` in the state
directory). A SQLite database's `-wal`, `-shm` and `-journal` files are folded
into the database and never listed as items. `AcceptDefaultTargets`
returns per-item `accepted` / `skipped_sensitive` / `failed` lists so partial
failures are never swallowed.

## Plans

Connect service:
`/vrooli.data_backup_manager.v1.plans.PlansService/<method>`

| Method | Purpose | CLI |
|---|---|---|
| `CreatePlan` | Bind targets to destinations with schedule and retention. Targets are an explicit list or, with `target_selector: "registered"`, every registered target except `excluded_target_ids`, resolved on every read. Blocked with `failed_precondition` when non-sensitive recommended targets are unregistered unless `allow_incomplete_coverage` is set. | `plans create` |
| `GetPlan` | Fetch a plan by id. | `plans get` |
| `ListPlans` | List backup plans. | `plans list` |
| `UpdatePlan` | Replace plan fields and membership. Same coverage guard as create via `allow_incomplete_coverage`. | `plans update` |
| `DeletePlan` | Delete a plan. | `plans delete` |

Critical tiers validate explicit critical target classification and destination
separation before persistence. `recovery_drill_schedule` enables independent
scheduled drill units for the plan.

## Recovery drills

Connect service:
`/vrooli.data_backup_manager.v1.drills.RecoveryDrillsService/<method>`

| Method | Purpose | CLI |
|---|---|---|
| `PreviewDrill` | Read-only eligibility and latest successful snapshot preview. | `drills preview` |
| `RunDrill` | Persist and asynchronously execute a scratch-only verified restore. | `drills run` |
| `GetDrill` | Read persisted drill status and linked restore evidence. | `drills get` |
| `ListDrills` | List drill evidence newest first. | `drills list` |

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

## Audits

Connect service:
`/vrooli.data_backup_manager.v1.audits.AuditsService/<method>`

| Method | Purpose | CLI |
|---|---|---|
| `RunSnapshotAudit` | Generic snapshot audit: restore to scratch + capture live + compare by generic inventory (counts, bytes, hashes, SQLite integrity). Async — returns a requested record. | `audits run` |
| `GetAudit` | Fetch a snapshot audit record (status, inventories, comparison) by id. | `audits get` |
| `ListAudits` | List snapshot audit records, optionally by target. | `audits list` |

The audit is scenario-agnostic: it stores relative paths, counts, and
hashes only — never file contents or secrets — and reports a
`live_newer_than_snapshot` drift flag so a mismatch caused by live
changing after the snapshot is distinguishable from corruption.

## Updating This File

After endpoint changes, run:

```bash
make endpoints
jq '.endpoints[] | {method, path, summary}' .vrooli/endpoints.json
```

Keep this page as a concise operator/developer map. The generated
manifest and proto schemas remain the detailed contract.
