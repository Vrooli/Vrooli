# CLI Reference: git-control-tower

Command groups. Source of truth: [CODE: cli/domains/domains.go].
Each subcommand is a thin wrapper over the API; business logic lives in
the API.

Run `git-control-tower <group> --help` or `git-control-tower <group>
<subcommand> --help` for full flag listings.

## `repo` — Inspect and change repository state

| Subcommand | Description |
| --- | --- |
| `status`       | Show repository status (branch + changed files) |
| `groups`       | Show resolved change groups (manual, contract, and `Other`) |
| `diff`         | Show git diff (`--path=FILE --staged`) |
| `blame`        | Show bounded native Git attribution and optional historical evidence (`FILE...`, `--revision=REV`, `--start=N --end=N`, `--enrich`, `--json`) |
| `stage`        | Stage files (`FILE...` or `--scope=scenario:name`) |
| `unstage`      | Unstage files (`FILE...` or `--scope=scenario:name`) |
| `commit`       | Create a commit (`-m MESSAGE [--conventional]`) |
| `sync-status`  | Check push/pull status (`[--fetch] [--remote=NAME]`) |

`repo groups` is read-only. Manual rules take precedence over contract targets;
unmatched paths are placed in `Other`. Contract groups are derived from the
active repository's `.vrooli/repo-contract.json` and are never persisted as
grouping rules.

## `branch` — Manage repository branches

| Subcommand | Description |
| --- | --- |
| `list`     | List branches (with ahead/behind counts) |
| `create`   | Create branch `NAME [--from=BASE] [--no-checkout] [--allow-dirty]` |
| `switch`   | Switch branch `NAME [--allow-dirty] [--track-remote]` |
| `publish`  | Publish current branch (`[--remote=NAME] [--branch=NAME] [--fetch]`) |

## `proposal` — Commit proposals bound to exact content

A proposal is an editable commit draft: exact files (sha256 and git blob ID per
path), per-file flags, excluded paths with reasons, and a message whose trailers
link the effort, epoch and runs. Every subcommand except `approve` writes only
Git Control Tower's database, so orchestrators may call them. `approve` is
human-only.

| Subcommand | Description |
| --- | --- |
| `anchor`   | Fingerprint files already dirty in scope at epoch admission (`--effort REF --epoch E<n> --scope GLOB...`) |
| `create`   | Create a proposal from listed paths (`--request-file F` or `--effort --epoch --subject [--body-file] --path P... [--run ID...] [--plan REF...] [--trailer K=V...] [--gate D...]`; `--validate-only` stores nothing) |
| `list`     | List proposals with live freshness (`[--state open\|committed\|withdrawn\|superseded\|all] [--effort REF]`) |
| `show`     | Show files, flags, exclusions, message, trailers and history (`ID [--revision N] [--message]`) |
| `diff`     | Review captured contents (`ID --revision N --path P [--compare-current] [--json]`); comparison never updates the proposal |
| `combine`  | Explicitly combine 2–20 sources (`--source ID@REV... --subject S [--body-file F]`); keeps code and source revision history |
| `edit`     | New revision with a changed subject, body or trailers, or dropped files (`ID --revision N [--subject] [--body-file] [--replace-trailers --trailer K=V...] [--remove-path P...]`) |
| `approve`  | Human only: print files, flags and message, ask for `confirm`, then prepare, confirm and apply one `repo.apply_proposal` intent (`ID [--yes] [--acknowledge-prior-work] [--keep-later-changes] [--skip-precommit]`) |
| `withdraw` | Close an open proposal; all code, staged changes and history are kept (`ID [--reason R] [--replacement ID] [--revision N]`) |
| `refresh`  | Re-hash current content into a new revision and recompute flags (`ID [--revision N]`) |

Delivery orchestrators run `anchor` when they admit an epoch and `create` when
they accept it:

```bash
git-control-tower proposal anchor --effort effort:<slug> --epoch E<n> \
  --scope 'scenarios/<scenario>/**' --scope 'packages/proto/**' --json
git-control-tower proposal create --effort effort:<slug> --epoch E<n> \
  --subject '<scenario>: <outcome> (E<n>)' --body-file <body.txt> \
  --path <file> [--path <file>...] --run <worker-run-id> --run <orchestrator-run-id> --json
```

`create` takes the newest anchor for the effort and epoch. A listed path that
did not change since the anchor, or that is unsafe or ignored, is refused;
changed paths in the anchor scope that are not listed are recorded in
`excluded`. A file dirty at the anchor and changed since is flagged
`mixed_prior_uncommitted`. Shared paths (`other_open_proposal`), Workspace
Sandbox pending paths (`sandbox_pending`) and `already_staged` are computed when
read. Each submission preserves older proposals and their review contents.
Final contents identical to HEAD are excluded as `matches_commit_base`; an
index-only difference is not a contribution to a commit.

`combine` requires exact retained source revisions, the same repository, base
and branch, and identical captured versions for shared files. It creates one
review package and atomically supersedes its sources, preserving their exact
revision manifests, evidence and relationships. Conflicting versions require
an operator decision and a newly captured revision; they are never replaced
with live bytes. `withdraw --replacement` records a relationship and omitted
paths; it does not combine content. Prior submissions remain addressable.

Agent-created messages must pass trailer grammar;
an agent `Co-Authored-By` is refused because the operator is always the author.

**Waiting digest (off by default).** With `GCT_PROPOSAL_DIGEST_ENABLED=true`
in the API environment, GCT publishes at most one Vrooli Events domain event per
UTC day, `git_control_tower.proposals.waiting.v1`, listing the open proposals
older than `GCT_PROPOSAL_DIGEST_MIN_AGE` (default `12h`). The facts are
`check_id`, `severity` (`informational`), `message`, `count`, `oldest_hours`
and `proposal_ids`; Notification Hub writes the notification copy. Notification
Hub delivers it to the phone only when its notification pattern includes
`git_control_tower.**` (default `incident.**`).

`approve` defaults to strict mode, refusing working drift in raw bytes, canonical
Git blobs or modes. `--keep-later-changes` instead commits the saved proposal
blobs/modes and leaves subsequent working edits untouched and uncommitted; this
choice is bound to the single-use intent. It refuses proposal paths with different
staged versions before altering staging. Review or unstage those versions first.
Both modes refuse missing retained bytes, a changed destination branch, a base
move touching proposal files, or any foreign staged path. The UI uses saved-version
mode and offers “Use latest working files” to explicitly update and review a new
proposal revision. Mixed-prior/no-anchor/outside-scope
files require `--acknowledge-prior-work` even with `--yes`; the acknowledgment
is bound to the single-use intent. Review `proposal diff` before approving.
The server stages exactly the reviewed paths and runs portable checks and hooks
in a private candidate checkout. It verifies the resulting tree and message
before publishing to the named branch under Git ref/index locks. Failure before
publication restores the index; a failure with a commit ID explicitly reports
that commit and does not reset an index after publication. Approval does not push.
Legacy proposals require an explicit `refresh` and review of the new revision.

History exposes full bodies, trailers and proposal links. Continue keeps work
relationships and adds `Vrooli-Continues` with the predecessor hash; it does not
reuse the old proposal, epoch, run or authorship trailers. P/pN subjects remain
supported. Effort/epoch/run/plan/backlog labels come from their typed owners;
unavailable or private owners do not imply a resolved link.

### Trailer vocabulary v1

Trailers follow `git interpret-trailers`: only the final paragraph is the
trailer block, continuation lines unfold, every key (including
`Co-Authored-By` and `Signed-off-by`) keeps its order, and keys match
case-insensitively. Unknown `Vrooli-*` keys (such as `Vrooli-Initiative`) are
kept as `legacy` and never mapped. A trailer is a work-reference assertion, not
authorship proof. Source: [CODE: api/internal/trailers/registry.go].

| Key | Value | Max |
| --- | --- | --- |
| `Vrooli-Effort`    | `effort:<slug>[@<revision>]` | n |
| `Vrooli-Epoch`     | `effort:<slug>#E<n>` | n |
| `Vrooli-Run`       | Agent Manager run UUID | 400 (explicit source unions; overflow refused) |
| `Vrooli-Plan`      | plan ID or slug, optional `#phase-<n>` | n |
| `Vrooli-Backlog`   | `<kind>/<name>` | n |
| `Vrooli-Continues` | 7–40 hex commit ID | 1 |
| `Vrooli-Proposal`  | `gctp-<hex>` | 1 |
| `Vrooli-Work`      | `<kind> <id>[@<revision>]` for any other kind | n |

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

### Engagements (shadow/live Baseline Modes)

An engagement pairs a restore point (the frozen Baseline) with a candidate in
the working tree. While a shadow engagement is open, live serves the restore
point and `<scenario>@shadow` runs the working tree. The floor owns the state
(`vrooli recovery …`); GCT only sequences it. A scenario has one engagement at
a time, named by `--name` (default `wip`).

| Subcommand | Description |
| --- | --- |
| `start`   | Open an engagement: mode decision, restore point, anchor, shadow (`--scenario [--mode] [--name] [--ttl] [--no-anchor]`). Refuses, before it captures anything, while the scenario has an open engagement; `--replace` takes over the same `--name` and keeps its restore point and anchor |
| `check`   | Diff the candidate against the engagement's anchor and renew the lease (`--scenario [--name]`) |
| `promote` | Keep the work and close the engagement: drain, data snapshot, migrate, re-point and restart live, status probe, optional `--probe-cmd`, auto-rollback on failure, shadow teardown (`--scenario [--name] [--probe-cmd] [--probe-timeout] [--exclude-run] [--drain-timeout] [--no-drain] [--force]`) |
| `cycle`   | `promote`, then a fresh shadow engagement under the same name with no anchor run (`--scenario --name`, plus the promote flags); safe to re-run after any failure |
| `status`  | List open engagements (`--json` includes `CreatedAt`, the last promotion for a cycled engagement) |
| `abandon` | Discard the candidate: restore the Baseline over the working tree and tear down the shadow (`--scenario [--name]`) |
| `gc`      | Reap expired engagements (`--force`: all) |

**Re-creating the shadow after promote.** `promote` closes the engagement.
To keep working in a shadow, open the next one under the **same name**:
`git-control-tower baseline start --scenario <s> --name <name> --mode shadow`
(add `--no-anchor` to skip the comprehensive anchor run). A bare `start` opens
`wip`, and `start` refuses it while another engagement is open. `promote`
prints this command; `baseline cycle --scenario <s> --name <name>` does the
promote and the re-create in one step.

`--probe-cmd` runs through `sh -c` with the scenario removed from
`VROOLI_SHADOW_SCENARIOS`, so it reaches live. A non-zero exit or a timeout
(`--probe-timeout`, default 15m) rolls back like a failed restart.

## CLI–API parity gaps

Several API surfaces do not have CLI commands yet (for example descriptor-aware
EvidenceService history/search, visual capture, agent runs, tidiness, SSH key
management, and credentials). EvidenceService is intentionally UI-first;
agents use Test Genie's canonical runs CLI and GCT's baseline commands.
