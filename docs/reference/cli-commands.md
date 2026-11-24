# Vrooli CLI Commands

Open the command group for the task at hand:
[project lifecycle](#project-lifecycle), [scenarios](#scenario-commands),
[resources](#resource-commands), [package governance](#package-governance),
or [agent sessions](#agent-sessions).

Use the relevant command's `--help` when you need to confirm flags. Use
`vrooli help` to discover a command group you do not already know.

Scenario and resource CLIs built on cli-core share one command-resolution
contract: singular/plural names dispatch, unknown names list the closest real
commands with their usage lines, positionals may be passed as `--<name>`, and
usage errors print the usage line and nearest flag. The contract lives in
[`packages/cli-core/README.md`](../../packages/cli-core/README.md#command-resolution-and-usage-errors).

## Resource health observations

```bash
vrooli resource status --health-only --json
vrooli resource status reranker --health-only --no-fast --json
```

`--health-only` reports process ownership and operational health without reading
every byte of installed runtime trees. Its managed-service records explicitly
report `artifact_verification: not_evaluated`. `--no-fast` still requests the
resource's readiness/liveness probes. Omit `--health-only` for a current artifact
checksum audit; all lifecycle launch gates continue to verify artifact bytes.

## Agent Manager execution selection

```bash
agent-manager runner execution-options --role <role> [--json]
agent-manager workflow start --owner <scenario> --key <workflow> --input-file <file> \
  --idempotency-key <key> [--preferred-runner <runner>] [--model <model>] [--effort <level>]
```

Execution preferences are hints, not pins. The server validates the runner and
effort, applies preferences to fresh workflow run nodes, and records the
actual selection and reason in the run policy snapshot.

## Git Control Tower staged messages

```bash
git-control-tower message collect [--repo-id ID] --json
git-control-tower message check --snapshot DIGEST [--repo-id ID] --json
git-control-tower message start --collection COLLECTION --snapshot DIGEST \
  --request-key KEY [--context-id CONTEXT ...] [--note TEXT] [--repo-id ID] --json
git-control-tower message status JOB [--repo-id ID] --json
git-control-tower message wait JOB [--timeout 45] [--repo-id ID] --json
git-control-tower message cancel JOB [--repo-id ID] --json
```

Collection gathers bounded evidence and retained messages without starting an
agent. Only `message start` explicitly admits drafting against the selected
immutable snapshot; reusing its request key resumes the same admission. Status,
wait and cancel also accept `--request-key KEY` instead of `JOB` after a lost
response. Wait makes one server-owned call (1–55 seconds), without polling or
starting another job. Cancellation stays pending until its owner confirms it.
These commands never stage or commit. Omitting `--repo-id` uses the active
registered repository; use an explicit ID when reconnecting across selections.

## Prompt-manager skill governance

```bash
prompt-manager skill import --help
prompt-manager skill review-import --help
prompt-manager skill import-staleness --help
VROOLI_SKILL_PROJECTION_DIRS="$HOME/.claude/skills:$HOME/.codex/skills:$HOME/.config/opencode/skills:$HOME/.gemini/config/skills:$HOME/.grok/skills" prompt-manager skill topology --json
```

`skill import` requires a local pinned source directory, immutable commit,
SPDX license, exact `sha256:` checksum of `SKILL.md`, importer, and skill ID.
Imported skills land inactive in `packs/vendor` until `review-import` records a
verdict from a different reviewer. `import-staleness` compares the recorded
upstream version with a current version supplied by the operator.

An imported skill is a **third-party skill**. Its rules:

- The import copies the whole skill directory (`SKILL.md` plus `references/`,
  `scripts/`, `assets/`). Symlinks and nested `SKILL.md` or `skill.json` files
  are refused. The origin records a `tree_checksum` over every copied file.
- The `packs/vendor` tree is git-ignored. A third-party skill is local to the
  host. It is not part of the Vrooli repository.
- `--external-tool 'name|url|purpose'` (repeatable) declares a tool the skill
  needs that Vrooli does not provide or manage, such as a CLI fetched with
  `npx`. External tools are recorded in `skill.json` as `externalTools`, apart
  from the Vrooli-owned `requires` block, and `skill-requirements` sync skips
  the vendor pack.
- The API returns `origin` and `externalTools` on every skill. `skill read`
  prints a third-party header with the source, license, commit, review verdict,
  skill directory, and external tools before the content. The UI marks the
  skill as "Third-party" and shows its attribution.
- A passed review activates the `vendor` pack at the lowest precedence, so a
  third-party skill never shadows an authored skill with the same ID. The review
  verdict, reviewer, and time are written to the origin block in both
  `skill.json` and the `SKILL.md` frontmatter.
- The upstream bytes are immutable. Local changes go in the skill's
  `overlays/` directory.

`skill topology` reports every indexed skill's pack and whether its generated
projection is resident in each configured target, including a resident
metadata-token estimate. It accepts either `--projection-dir` or the
path-list environment variables `VROOLI_SKILL_PROJECTION_DIR` and
`VROOLI_SKILL_PROJECTION_DIRS`.

### Editing projected skills

Edit the canonical `SKILL.md` in the owning scenario or Prompt Manager's
`store/skills/packs/<pack>/<id>/` tree. Files under native skill directories
such as `.codex/skills` with the Prompt Manager generated marker are disposable
projections. Preserve local differences before replacement.

Four facts are distinct:

| Fact | Owner evidence |
| --- | --- |
| Selected for native installation | Prompt Manager's `store/skills/_base-pack.json`; not every registry skill is selected. |
| Available through the registry | `prompt-manager skill read <id>`; `skill sync` does not refresh native files. |
| Resident and current on disk | `skill topology` reports residency; `skill refresh` reports content identities and freshness. |
| Actually consumed by an agent | The run's skill-read/provenance evidence; fresh files do not prove an existing conversation reloaded them. |

Use scoped preview and apply after canonical edits:

```bash
prompt-manager skill refresh --runtime codex --skills scenario-work-ladder --json
prompt-manager skill refresh --runtime codex --skills scenario-work-ladder --apply --expected-digest '<preview digest>' --json
```

No restart is required once the server and CLI support `RefreshProjection`.
Runtime names come from the server's configured resource targets; a single private
`VROOLI_SKILL_PROJECTION_DIR` uses `configured`. Empty selectors inspect the configured
targets and governed base pack, never the entire corpus. Clients cannot supply a
destination path. The digest binds selection, paths, adoption policy, source,
installed content, and receipt identities. Changed inputs require another preview.

`current`, `missing`, and `outdated` rows can apply. `recoverable` means an earlier
receipt was written but its replacement did not finish. `modified`, `unmanaged`,
and `conflict` rows preserve the installed file. `legacy` means the generated marker
exists but no baseline proves whether differences are local edits. Compare the
native file with canonical source, excluding only the generated marker. If the
differences are approved, preview again with `--adopt-legacy`, then apply that digest
with the same flag. There is no force-overwrite flag for modified tracked files.

The writer retains replaced bytes under the target's
`.prompt-manager/backups/<skill>/<hash>.md` and a before/after receipt at
`.prompt-manager/<skill>.json`. Apply results name recovery copies. For recovery,
inspect that copy and the current file; preserve newer edits before an operator-
authorized restoration. A retry uses a new preview and skips unchanged bytes.
Partial results are per runtime and skill; CLI apply exits nonzero if any selected
row did not apply. An API response alone is not an all-target success verdict.

Startup uses the same conflict-safe writer without legacy adoption. It no longer
deletes unselected skill directories. Removing a base-pack entry stops updates;
retiring an installed copy is a separate reviewed operation. Symlinked destinations
or ancestors are refused. This is cooperative local publication, not isolation
against a hostile process racing filesystem operations. Refresh currently projects
`SKILL.md`, not supporting assets. Recovery receipts do not establish which revision
an agent consumed. Existing conversations must explicitly reread canonical guidance
or start a fresh run before claiming the revised guidance governed their behavior.

## Root Commands

```bash
vrooli help
vrooli setup
vrooli develop
vrooli build
vrooli clean
vrooli status
vrooli stop
vrooli backup
vrooli restore
vrooli cleanup
vrooli doctor
vrooli orphans
vrooli locks
vrooli diagnose-port
vrooli contract
vrooli package
vrooli resource
vrooli scenario
vrooli supervision-set
vrooli uninstall
```

These root commands are confirmed by the current CLI help surface.

## Program Runtime portfolio commands

```bash
program-runtime programs portfolio --window-days 30 --json
program-runtime library run program-runtime.portfolio-audit --input min_rung=S3
program-runtime library run program-runtime.usage-triage --input window_days=30
```

`programs portfolio` reports bounded usage, latency, provenance, caller
attribution, and never-executed contracts. `portfolio-audit` grades declared
contracts against deterministic schema and portfolio dimensions. `usage-triage`
selects a bounded attributed cohort and spends one governed batch classification.
These commands measure the corpus; they do not authorize edits to another
scenario.

## Program Runtime learning and dependency commands

```bash
program-runtime library run program-runtime.fragment-promote --input step_key=<step-key>,min_verified=5 --json
program-runtime bindings list --include-program-bindings --json
```

`fragment-promote` reads a verified `learn.act` fragment and returns a bounded,
reviewable crystallization candidate. It does not replace scenario source
without the improve-cycle decision. `--include-program-bindings` adds the
program-runtime contract graph to dependency inspection; Scenario Dependency
Analyzer consumes the same graph and keeps the scenario manifest authoritative
for lifecycle startup and packaging.

## Orientation

```bash
vrooli help
vrooli status
```

- `vrooli help` shows the current command tree.
- `vrooli status` gives a health and status overview.

## Supervision Set

```bash
vrooli supervision-set
vrooli supervision-set --kind scenario
vrooli supervision-set --kind resource --json
```

`vrooli supervision-set` is the control-plane authority for the live set of
scenarios and resources that the platform supervises. It computes the answer
from `operator state` → `core.seed` and canonical scenario
dependency manifests; it does not read a database and does not own another
configuration file. Each member reports its kind, effective supervision
intent, and an attribution chain back to the seed that included it.

The JSON contract is `vrooli.cli.v1.SupervisionSetResponse`. `--kind` filters
the returned member rows to `scenario` or `resource`. The in-process
`internal/app/supervision.Service.Read` API returns the same typed member model
for scenario consumers that must not shell out.

`scenario-dependency-analyzer core-set` remains a compatibility presentation
over the same database-free computation. New consumers should use the
control-plane command or the in-process API; the compatibility verb does not
own a separate list.

## Project Lifecycle

Use `vrooli capacity list` for live claims and `vrooli capacity list --all`
when you need terminal claim history.

```bash
vrooli setup
vrooli develop
vrooli build
vrooli clean
vrooli stop
```

- `setup` initializes the development environment
- `develop` starts the project development workflow
- `build` builds project-level binaries
- `clean` removes build artifacts or runs the project clean lifecycle
- `stop` stops all or selected components

### Setup progress and diagnosis

Setup writes an immediate diagnostic progress line before host validation, then
reports ordered phases and only emits a heartbeat when a phase remains active
for a meaningful period. The first heartbeat is emitted after 10 seconds and
subsequent heartbeats every 30 seconds; an operation change is reported
immediately. Setup uses semantic labels rather than a percentage because
requirements and safeguards are conditional and often indeterminate.

Human progress is written to the diagnostic stream and is safe for terminals,
SSH, pipes, redirected files, and CI: it does not require raw terminal mode,
cursor control, or a full-screen UI. The existing `--result-file` remains the
terminal JSON contract; progress is not a replacement for that result.

For automation or log ingestion, select newline-delimited structured progress:

```bash
VROOLI_SETUP_PROGRESS_FORMAT=json vrooli setup --dry-run --resources none --scenarios none
```

Use `VROOLI_SETUP_PROGRESS_FORMAT=quiet` when a caller wants setup behavior and
terminal results without human progress. Progress output is best effort and
never changes setup success or failure. Operation labels are redacted before
human, structured, or durable publication.

The latest active/terminal setup record is kept in project-scoped Vrooli state.
`vrooli setup status` includes the last known run and phase when available. A
running record is called possibly stale only after its age exceeds the stale
threshold and its recorded host/PID identity cannot be confirmed; an old
timestamp alone is not treated as proof of failure.

## Scenario Commands

Inspect the current surface with:

```bash
vrooli scenario --help
```

Common commands:

```bash
vrooli scenario list
vrooli scenario info <name>
vrooli scenario status <name>
vrooli scenario validate-env <name>
vrooli scenario timings [--scenario <name>]
vrooli scenario start <name>
vrooli scenario start-all
vrooli scenario run <name>
vrooli scenario setup <name>
vrooli scenario restart <name> [--deploy-report]
vrooli scenario deploy-report <name> [--json]
vrooli scenario wait <name> [--timeout <seconds>] [--json]
vrooli scenario stop <name>
vrooli scenario stop-all
vrooli scenario test <name>
vrooli scenario logs <name>
vrooli scenario profile <name> --seconds 20 --out /tmp/scenario.cpu.pprof --json
vrooli scenario open <name>
vrooli scenario port <name>
vrooli scenario ui-smoke <name>
vrooli scenario requirements
template-manager registry list --kind scenario
template-manager generate <template> --id <slug> --display-name <name> --description <text>
vrooli scenario completeness <name>
```

### Authentication mode and the scenario environment

`vrooli scenario start|restart` never takes an authentication mode, provider
list, credential, or agent identity from the caller's environment. A managed
scenario runs in its manifest `authentication.default_mode`, and its processes
inherit only a documented host allowlist. `vrooli scenario status <name>`
prints an `Authentication:` section (`authentication` in `--json`) with the
configured mode, its source, the mode each running process carries, and a
mismatch warning when they differ. Contract:
[Configuring Scenarios](../configuration/scenarios.md#scenario-process-environment).

### Local CPU profiles

`vrooli scenario profile` collects a bounded CPU profile from the exact recorded
component launch (default `--step start-api`). Use a new output path; existing
files are never overwritten. Durations are 1–30 seconds, and output is limited to
16 MiB with owner-only permissions. Inspect it with `go tool pprof -top <path>`.
The shared standard API runtime serves this through an owner-only Unix socket,
enabled after a normal rebuild/restart with the current control plane. No sudo
or public debug endpoint is required. Named local instances are supported;
remote node profiling is explicitly unsupported. See the
[runtime diagnostics contract](../../packages/api-core/runtimediagnostics/README.md).

The CPU samples describe the selected Go process. Account for subprocesses
separately when assessing a whole component; a quiet API profile does not prove
that its child commands are idle. On Linux, `/proc/<pid>/io` also includes
waited-for children, so those counters have a different scope from API-only CPU
samples ([proc_pid_io(5)](https://www.man7.org/linux/man-pages/man5/proc_pid_io.5.html)).

### Node-axis addresses

Scenario addresses use the grammar `[node/]scenario[@variant]`:

```bash
vrooli scenario status web-search
vrooli scenario status web-search@shadow
vrooli scenario status minimouse/web-search
vrooli scenario status minimouse/web-search@shadow
```

The explicit `node/` prefix selects a Bridge node and wins over the global
`--node <name>` option. An explicit `@variant` wins over the ambient shadow
selection. With neither a node prefix nor `--node`, resolution stays local;
there is no environment variable or ambient list that promotes a call to
another machine. Remote scenario status and other admitted typed calls use
the node's outbound Bridge channel, not an invented inbound scenario URL.

Notes:

- `run` is an alias of `start`
- for day-to-day work on one scenario, prefer the scenario-local `make start|test|logs|stop` flow when available

`vrooli scenario timings` reads the retained terminal start-operation records
and renders count, mean, p50, p90, total, and share for each operation step.
Without `--scenario`, it includes per-scenario rows and a `fleet` aggregate;
`--scenario <name>` limits the report to one scenario. The report is a retained
tail, not an exhaustive history: the runtime currently keeps the newest five
terminal records per scenario and variant. Use `--json` for the typed timing
rows (`scenario`, `operation`, `step`, `count`, `mean_ms`, `p50_ms`, `p90_ms`,
`total_ms`, and `share`).

`vrooli scenario restart <name>` reuses fresh artifacts and therefore skips
unchanged setup work. Pass `--force` to rebuild unconditionally. The flag is
also accepted by `scenario start`; it never changes the production artifact
served by the scenario.

For a deliberate restart that must proceed without an owner lifecycle proof,
pass `--force-lifecycle --lifecycle-override-reason '<reason>'`. This is an
explicit risk acceptance: active requests or detached owner work may be
interrupted. The lifecycle log records the reason and the missing or overridden
proof. When the provider's drain is held by work that never finishes, the forced
restart waits a bounded 60 seconds, then replaces the provider and logs which
requests or work were cut off. It does not allow replacement when the control plane cannot establish
which process owns the target scenario.

`vrooli scenario deploy-report <name>` lists, without git, the source files
and new environment switches that changed since each running build, so a
coordinator can see other sessions' unfinished work before a restart ships
it. `restart --deploy-report` prints the same report to stderr first. See
[staleness-and-rebuild.md](staleness-and-rebuild.md#pre-deploy-report).

Once a start, stop, or restart holds the scenario's lifecycle lock, it
finishes even if whoever reads its output goes away (a `| head` that has its
lines, a closed terminal): while the lock is held, a write to a broken stdout
or stderr fails instead of ending the CLI, and the lifecycle log still records
every line. Outside that window a broken pipe ends the CLI as usual. Read the
outcome from `vrooli scenario freshness <name>` or the lifecycle log rather
than from truncated output.

### Scenario start wait contract (anti-polling recipe)

`vrooli scenario start|restart` are blocking and every top-level invocation
writes a durable **start-operation record** to the runtime registry: per-step
state (stop → dependencies → setup → develop → health), the current
dependency (n of m), initiator pid, and the terminal verdict. That record is
what makes the start introspectable, attachable, and resumable:

- **Block once — never poll.** To wait on an in-flight start (yours or
  another process's), run `vrooli scenario wait <name> --json [--timeout N]`.
  It attaches to the in-flight operation, returns ONE JSON document
  (`vrooli.cli.v1.ScenarioWaitResponse`), and exits with the verdict. With no
  start in flight it evaluates current runtime health and returns
  immediately. Do **not** loop `vrooli scenario status`.
- **Exit codes** (`start`/`restart --json`, `wait`): `0` healthy (or
  `running` for scenarios with no health checks); `1` failed / not running /
  abandoned; `2` degraded-after-timeout success (usable, but non-critical
  checks failing — `--json` mode only for `start`/`restart`; text mode keeps
  exit 0 for legacy callers); `124` `--timeout` ceiling elapsed.
- **`--timeout` is a ceiling, not the expected wait.** Size it as the ETA
  plus ~75% buffer. On expiry `wait` detaches with exit 124 and the awaited
  start is untouched. On a `start`/`restart` the expiry stops the
  orchestration with the CLI process — unlike test-genie runs there is no
  server owning it — but the operation record stays honest (dead initiator ⇒
  reported `abandoned`) and the next `start` resumes/takes over.
- **Concurrent starts attach.** A second `vrooli scenario start` on a
  scenario whose start is already in flight waits for the owner's verdict
  instead of failing with "scenario is busy"; if the owner dies mid-flight
  the second caller takes the start over. Busy errors remain only for true
  conflicts (e.g. a concurrent stop).
- **Introspection.** `vrooli scenario status <name> --json` includes
  `start_operation` with the current step, dependency n/m, elapsed,
  `initiator_pid`, an honest ETA (`eta_known=false` when there is no recorded
  phase history — never a fabricated number), and
  `recommended_next_check_seconds` (0 terminal — stop checking; 30 unknown
  ETA; else remaining clamped to [5, 60]).
- **Runtime status is not activity.** Mid-start a scenario reports
  `Status: stopped` — it has not registered processes yet — which reads
  identically to a scenario nobody is touching. The human `vrooli scenario
  status <name>` therefore prints a `Lifecycle:` line whenever an operation
  is in flight, naming the same initiator pid a competing lifecycle call
  reports as the lock holder:

  ```
  Status: stopped
  Lifecycle: start in progress (pid 3941220) — setup step, 4s elapsed
  ```

  No `Lifecycle:` line means no operation is in flight: a terminal record is
  history, and a `running` record whose initiator is dead evaluates to
  `abandoned` rather than being advertised as activity.
- **Ctrl-C semantics.** Interrupting `wait` detaches (exit 0, re-attach
  guidance on stderr); interrupting the owning `start` records the operation
  `abandoned` so status stays honest.
- **Agent-manager runs park.** Inside an agent-manager run,
  `vrooli scenario wait` parks the run instead of blocking (producer key
  `lifecycle`); agent-manager performs the wait and wakes the run with the
  JSON verdict at zero token cost.
- **JSON purity.** In `--json` mode stdout carries exactly one JSON
  document; progress lines and every hint go to stderr.

## Resource Commands

Managed-service JSON status separates current declaration, verified staged bytes,
and the owned running image using `declared_artifact_version`,
`staged_artifact_version` and `running_artifact_version` in the resource status
record. `artifact_update_available` does not mean the service is down.
`artifact_staging_error` describes the desired artifact independently of running
health, so a missing newer release cannot hide a verified retained service.

Inspect the current surface with:

```bash
vrooli resource --help
```

Common commands:

```bash
vrooli resource list
vrooli resource census
vrooli resource cli-sync [--dry-run]
vrooli resource scaffold --name <name> --driver <archetype>
vrooli resource info <name>
vrooli resource status
vrooli resource validate
vrooli resource install <name>
vrooli resource acquisition explain "<name>"
# Check installed Python imports before model/weight preparation; new receipt directory required.
vrooli resource acquisition runtime-probe "<name>" --imports numpy --out "<new-receipt-directory>"
# Reuse verified compose parts; refuse network acquisition on any miss.
vrooli resource acquisition compose-candidate "<name>" --out "<new-candidate-directory>" --offline
vrooli resource acquisition prune
vrooli resource acquisition prune "<name>"
vrooli resource uninstall <name>
vrooli resource start <name>
vrooli resource restart <name>
vrooli resource start-all
vrooli resource stop <name>
vrooli resource stop-all
vrooli resource logs <name> [--tail <lines>]
vrooli resource enable <name>
vrooli resource disable <name>
vrooli resource deprecate <name>
vrooli resource list-deprecated
vrooli resource archive-to-blueprint <name>
vrooli resource list-blueprint-archived
vrooli resource restore <name>
vrooli resource restore-blueprint <name>
vrooli resource archive
vrooli resource blueprint
vrooli resource schema
```

These commands are part of the current CLI help surface, though some of them are more administrative than day-to-day.

Single-resource lifecycle commands preserve the CLI operation context through
resource dispatch, transient CLI commands, readiness probes and lifecycle-lock
waiting. Ctrl+C ends that operation's wait or transient subprocess; a cancelled
start/restart readiness wait does not itself stop an already supervised service.
Use `vrooli resource stop <name>` for owned shutdown. A cancelled queued lock
waiter cannot release or bypass the active lifecycle owner. Cancellation and
startup deadlines retain distinct causes plus the last readiness diagnostic.
Authored startup budgets and normal stop escalation policy are unchanged.
This does not extend cancellation to the bulk `start-all`/`stop-all` wrappers.

## Package Governance

Inspect the current surface with:

```bash
vrooli package --help
```

Common commands:

```bash
vrooli package list
vrooli package info <name>
vrooli package dependents <name>
vrooli package build <name>
vrooli package generate <name>
vrooli package refresh <name>
```

Structural validation is provided by Structure Health:

```bash
structure-health rules list --json
structure-health rules coverage
```

## Repo Contract

Inspect the current surface with:

```bash
vrooli contract --help
```

Common commands:

```bash
vrooli contract validate
vrooli contract show
vrooli contract resolve
vrooli contract match-glob
```

## Agent sessions

```bash
vrooli agent launch --runner claude [--claim <path>]...
vrooli agent launch --runner codex --prompt "Investigate the scenario" --timeout 10m
vrooli agent recover list [--json]
vrooli agent recover reconcile [--limit 32] [--json]
vrooli agent recover investigate --scenario web-console --reason "API remains unhealthy" [--json]
vrooli agent list [--json] [--idle-after 168h]
vrooli agent list --session-id <harness-session-id> [--json]
vrooli agent reap --orphaned --dry-run [--json]
vrooli agent reap --orphaned [--older-than 1h] [--json]
vrooli agent reap --orphaned --exact-scope <scope-unit> --dry-run --json
vrooli agent thaw <session-or-scope>
```

`agent launch --prompt` selects the runner's headless form: Claude, Grok and
Antigravity use `-p`, Codex uses `exec`, and OpenCode uses `run`. Without a
prompt, the runner keeps its normal interactive invocation. `--timeout` bounds
the waiting invocation and preserves the wrapper so cancellation is not lost
through process replacement. Cancellation alone does not prove that every
descendant has stopped.

`agent recover list` includes both lifecycle recovery and coding-agent
investigation receipts. JSON separates these into `lifecycle` and
`coding_agents`. Coding-agent receipts include the selected runner, creation
time, deadline, outcome and error. A `succeeded` investigation means its runner
returned successfully; fresh scenario health checks establish recovery.
`scheduled`, `running` and `unconfirmed` receipts block another investigation
for that scenario even after the hourly admission window. A lost owner or an
expired deadline is not proof that the previous agent stopped. These receipts
need reconciliation with native session ownership before retry; do not delete
the ledger to bypass this protection.
New ambiguous executions record `observation_ended_at` and leave `finished_at`
unset. Observer cancellation is not agent completion.

`agent recover reconcile` changes bounded historical or expired execution
claims without a retained native identity into `unconfirmed`. It records when
that uncertainty was established. It does not claim exit, stop or launch work,
set a completion time, or release duplicate suppression. Fresh observations of
native execution remain necessary before an unresolved attempt can be retired.

`agent recover investigate` admits a bounded coding-agent investigation through
the same native broker used by Autoheal, including its hourly scenario window,
unresolved-work suppression, concurrency limit and ten-minute native deadline.
It requires a resolved repository root and explicit failure reason. It returns
the durable admission identity without waiting for the model. An ambiguous
dispatch still prints that identity and exits with an error; inspect the receipt
instead of submitting a replacement. The native job survives the calling CLI
and Autoheal API. Fresh health evidence is required to claim a repair.

`agent list` shows each session's state (`live`, `idle`, `orphaned`,
`agentless`, `unrecorded`, `ghost`, `undetermined`) and its owner
(`agent-manager-run`, `web-console-session`, `desktop-app`, `terminal`,
`unknown`) with whether the owner still exists. `agent reap --orphaned` stops
only sessions whose owner is confirmed gone, through the platform containment
seam, with one audit row per stop; `--dry-run` lists what would stop with the
owner evidence and stops nothing. The unattended reap in the runtime
supervisor is off unless `VROOLI_TUNING_AGENT_ORPHAN_REAP_INTERVAL` is set.
Owner kinds, states and the reap policy: [Agent sessions](agent-sessions.md).

## Maintenance

```bash
vrooli cleanup
vrooli doctor
vrooli orphans
vrooli locks
vrooli diagnose-port <port>
```

These commands help inspect registry port claims, orphaned processes, diagnostics, and port conflicts.

## Project uninstall

`vrooli uninstall` is the project-level removal command. It is deliberately
not a prompt-manager runnable action. Use a plan/apply pair and select the
smallest scope that meets the need:

```bash
vrooli uninstall --plan --scope agent --confirm-target <hostname>
vrooli uninstall --apply <plan-id> --confirm-target <hostname> --break-glass-token <token>
```

The scopes are `agent` (node agent and state), `runtime` (checkout, binaries,
and supervisor), and `all` (both). Apply discovers nothing: it executes only
the frozen inventory and refuses if the record, paths, symlink boundaries, or
disk fingerprints drift. Four independent controls protect a destructive
apply: the command is `run_eligible: false`, the live hostname must match
`--confirm-target`, a short-lived per-machine break-glass credential is
required, and the plan/apply inventory is frozen. Never use a development host
as the target; inspect the emitted plan before applying it.

Practical guidance:

- `vrooli locks` shows the runtime registry's port claims (expired claims hidden unless `--all`; `--json` always includes everything)
- `vrooli cleanup locks` expires stale leases and non-authoritative registry claims
- `vrooli orphans` inspects or terminates orphaned Vrooli-managed processes
- `vrooli diagnose-port <port>` is the targeted tool for a fixed-port startup failure after lifecycle has already attempted automatic cleanup

`vrooli orphans kill --dry-run --json` previews generic orphan candidates.
Execution pins a stable process handle before a fresh complete ownership check
and verifies exit through that handle; PID reuse cannot transfer a signal to a
replacement. Unknown handles or native membership refuse cleanup. Candidates
that disappear or acquire an owner are reported as observations, not stop
effects. Dedicated Vrooli service and operation scopes remain with their native
lifecycle owners, including daemon children left after a root exits.
Stale process-record cleanup compares OS incarnation when available and removes
only the complete observed record under the same lock as launch registration.
A new generation and its PID sidecar survive a stale cleanup snapshot.

`vrooli agent reap --orphaned --dry-run --json` is the separate owner-aware
session preview. Unknown membership is undetermined and cannot enter its reap
plan. An operator terminal, unknown owner, or Web Console terminal whose origin
is unobserved remains protected even when its former session is absent. Age,
idle state and an agentless scope name do not authorize termination.

## Cloud target (target-local deployment owner)

`vrooli cloud-target` runs on a cloud deployment target inside the native
`vrooli` binary the deploying side uploads. It is the control-plane owner of
release staging, activation, receipts and host repair for that target; the
scenario-to-cloud API drives it over its reach layer with argv, never with a
shell string. Package: `internal/cloudtarget`.

```bash
vrooli cloud-target receipt get --deployment <id> --operation <id> --step <id>
vrooli cloud-target release verify --deployment <id> --release-manifest <path> --archive <path>
vrooli cloud-target release stage --deployment <id> --operation <id> --step <id> --fence <n> --release-manifest <path> --archive <path>
vrooli cloud-target release activate --deployment <id> --operation <id> --step <id> --fence <n> --release <digest> [--strategy side_by_side|maintenance] [--scenario <id>]... [--port <name>=<n>]... [--data-binding <id>=<scenario>/<path>]... [--legacy-carry <scenario>/<path>]... [--legacy-root <workdir>] [--restart]
vrooli cloud-target release rollback --deployment <id> --operation <id> --step <id> --fence <n> --to <digest>
vrooli cloud-target release list --deployment <id>
vrooli cloud-target data inventory --deployment <id> --workdir <path> --scenario <id> [--bindings '[{"id":"uploads","path":"data/uploads"}]']
vrooli cloud-target host repair --action <privilege-broker action> --subject '<json>' [--deployment --operation --step --fence]
# JSON-shaped flags (--subject, --bindings, --binding, --spec) also accept b64:<base64url JSON>, the spelling the cloud side uses on argv-only transports.
# release activate pins listener ports as <NAME>_PORT for the lifecycle owner, binds --data-binding paths to <deployment>/persistent-data/<id> (adopting legacy content by rename, never copying or deleting), carries --legacy-carry paths under persistent-data/legacy/, reports every unmapped heuristic directory as legacy_unmapped and leaves it in place; --restart re-runs the activator for the already active release (start/resume).
vrooli cloud-target credential ingest --deployment <id> --binding <id> --version <n> --operation <id> --step <id> --fence <n> [--grant <id> --from-env <VAR> --logical-id <id> --field <f> --content-ref <ref>]   # sealed payload on stdin
vrooli cloud-target credential acknowledge --deployment <id> --binding <id> --version <n> --consumer <scenario:id|resource:id> --operation <id> --step <id> --fence <n>
vrooli cloud-target credential revoke --deployment <id> --binding <id> --version <n> --operation <id> --step <id> --fence <n>
```

State lives beneath `~/.vrooli/cloud/deployments/<deployment-id>/`:
`releases/<release-digest>/` (complete, immutable), `releases/<digest>.staging-<op>/`
(in flight, never activatable), `active-release.json` (active and retained
predecessor), `activation-intent.json` (present only while a switch is in
flight), `operations/<operation-id>/<step>.json` (receipts) and `fence.json`.

Contract for every effectful verb (`stage`, `activate`, `rollback`,
`credential ingest|acknowledge|revoke`, and `host repair` when `--operation`
is given):

- The fence is refused when it is below the highest fence the deployment has
  accepted (`{"error":{"code":"fence_stale"}}`, exit 2); an accepted fence is
  recorded before the verb runs.
- One receipt exists per (operation, step). A re-run returns that receipt
  unchanged with `"replayed": true` and never re-executes; a re-run with
  different inputs is refused with `receipt_input_mismatch`. To retry a failed
  step, use a new step id or a new operation.
- Exit codes: 0 ok, 1 failed (receipt outcome `failed`), 2 refused.

Credential verbs (`internal/cloudtarget/credential.go`): `ingest` reads the
sealed JSON payload `{schema_version:1, deployment_id, binding_id, logical_id,
field, version, content_ref, value}` from standard input (or, for a Bridge
dispatch, the value from the environment variable `--from-env` names) and
writes it into the node credential authority; the receipt input digest covers
metadata only and a value never appears in argv, the receipt or the ledger
(`credentials/<binding-slug>.json`). A locked store is refused with
`credential_store_locked` before any write; a version below the held one is
`credential_version_stale`; a revoked version can never be re-ingested
(`credential_version_revoked`). `acknowledge` proves the held version matches
and the authority still answers for it. `revoke` deletes the active version
and marks it revoked; its receipt lists what was proven, what is unproven and
the limitation that a compromised host's memory cannot be proven cleared.

Release trust: `verify` (and `stage` before it writes anything) checks the
archive sha256 against `bundle_sha256`, the running binary's sha256 and
GOOS/GOARCH against `native_cli`, and `release_digest` against the canonical
identity `{"bundle_sha256","closure_digest","configuration_digest","native_cli":{"goarch","goos","sha256"}}`
encoded as sorted-key compact JSON and hashed with sha256. It then walks the
archive without writing, enforcing no absolute paths or `..`, no escaping
symlink or hardlink, no device/fifo entries, and the manifest's
`limits.max_entries`, `max_expanded_bytes`, `max_entry_bytes` plus a time
budget. Reason codes: `release_digest_mismatch`, `bundle_sha256_mismatch`,
`native_cli_mismatch`, `archive_traversal`, `archive_symlink_escape`,
`archive_too_large`, `archive_entry_limit`, `archive_unsupported_entry`,
`archive_time_budget`.

Activation is delegated to the lifecycle owner: the default activator runs
`vrooli scenario restart <id> --path <release-dir>/scenarios/<id>` for every
scenario in the release (after `vrooli scenario stop` under the `maintenance`
strategy). The active pointer is committed only after the runtime owner
succeeds; a crash in between leaves the prior release active and an
`interrupted_activation` entry in `release list` for reconciliation.
`rollback --to` accepts only the retained predecessor.

`host repair` accepts only privilege-broker actions with their typed subject
(`{"apt":{"packages":[...]}}`, `{"edge":{"port":80}}`,
`{"process":{"scenario":"<id>","workdir":"<abs>"}}`, and the existing
bridge/volume/storage families). `process.stop.scoped` is executed through
`vrooli scenario stop <id>` as the workload owner; every other action goes to
the broker socket and fails closed when the broker is absent. There is no
generic shell verb.

## Plan Manager Commands

Use the `plan-manager` scenario CLI for new implementation plan authoring. The
authoring flow owns structured plan setup, context discovery, code references,
validation, and finalization.

```bash
plan-manager --auto-start author start --title "<title>"
plan-manager author continue <session>
plan-manager author context-discover <session> --concepts "<concepts>"
plan-manager author context-accept <session> <candidate-id>
plan-manager author suggest-references <session>
plan-manager author reference-accept <session> <candidate-id>
plan-manager author preview <session>
plan-manager author validate <session>
plan-manager author finalize <session>
plan-manager plans list --workspace <path>
plan-manager plans get <id-or-slug> --workspace <path>
plan-manager plans render <id-or-slug>
```

Project-level plan hygiene remains under `vrooli hygiene --plans` and
`vrooli hygiene --plans-only`; it delegates reconciliation to Plan Manager.
Plan lifecycle, inspection, render, import, archive, and authoring workflows
belong on `plan-manager` directly.

## Notification Hub decisions

Use the `notification-hub` scenario CLI to ask the operator a decision and wait
for the answer. Write the ask by `prompt-manager skill read notification-hub`;
the packet model (attachments, answer form, context requests, packet check) is
in `scenarios/notification-hub/docs/concepts/DECISION_PACKETS.md`.

```bash
# the minimum: a question, a menu, and a subject key
notification-hub conversations ask --yes-no --subject-key <situation> --question "..."
# a fuller ask; deadline 24h, sensitivity private, urgency normal and the idempotency key are defaults
notification-hub conversations ask --question "..." --option a=Approve --option b=Reject --subject-key <situation> \
  --explainer why.html --attach plan.md=Plan --attach screenshots/ --form form.yaml
notification-hub conversations ask --body-file ask.md --yes-no --subject-key <situation> --dry-run   # check and preview, send nothing
notification-hub conversations ask --yes-no --subject-key <situation> --question "..." --and-wait    # send and wait
notification-hub conversations wait <id>   # --until-answer ignores context requests; default deadline 30m
notification-hub conversations get <id>
notification-hub conversations wait --subject-key <situation>   # resume: re-derive the key, no id needed
notification-hub conversations request-context --ask-id <id> --question "..."   # operator
notification-hub conversations answer --ask-id <id> --answer a --responses-file answers.json   # operator
```

- Exit status of `wait` and `ask --and-wait`: `0` an answer is available (answered,
  or the default applied), `3` the operator asked for more context, `4` the deadline
  passed with the ask still open, `5` the ask closed without an answer (withdrawn,
  dismissed or expired). `wait` carries on across the Hub's ten-minute limit on one
  wait and across dropped connections until its own deadline. `ask` output ends with
  `ask_id=... state=... outcome=... next: ...`; `--json` is `{ask_id, state, outcome,
  warnings[]}`. `--body-file FILE|-` reads the question (first paragraph),
  `## Background` and `## Details` from markdown. `--yes-no` is the menu yes/no.
  With no `--idempotency-key` the CLI derives one from the terms (the same terms give
  the same ask, changed terms a revision) and prints it.
- A revision (`ask --subject-key K` while that ask is open) says only what changes.
  The CLI sends only the flags the command gave, and the Hub keeps the open ask's
  question, answers, big picture, technical details, recommendation, default
  answer, urgency, context link, lapse window, attachments, explainer and form
  for everything left out; giving a field an empty value clears it. Naming
  `--option` changes the menu, which replaces the ask and drops the old
  recommendation and default. A sent attachment replaces the one with the same
  name, a sent question the one with its id; `--clear-attachments` and
  `--clear-questions` drop every current part first. A keyed ask gets a fresh
  idempotency key on each send, so returning to earlier wording still revises.
- `--attach PATH[=LABEL]` (repeatable) is a file or a folder. A folder is one
  attachment of at most 500 files in path order; `.git`, `.hg`, `.svn`,
  `node_modules`, `__pycache__`, `.venv`, `.mypy_cache`, `.pytest_cache`,
  `.cache`, `.idea`, `.DS_Store` and `Thumbs.db` are skipped and symlinks inside
  it are never followed. The attachment name is the sanitized base name
  (`a.png`, `a-2.png`). A file over 64 MiB, or one that would take the total past
  256 MiB, is left out from its size alone, without being read. The CLI hashes the files, asks the Hub which
  blobs it lacks, and uploads only those.
- Packet parts are optional, so a part that cannot be used never stops the ask:
  a missing, unreadable or oversize attachment, a form file that cannot be
  read (all questions left out), a broken question (only that one left out), a
  broken recommendation, an upload failure (the Hub then leaves that
  attachment out). Each is named on stderr as "Left out ..." and the ask is
  sent. Only the cover (question, options, deadline, ...) can fail the command.
- `--explainer PATH` is one self-contained HTML file (role explainer), sent
  first.
- `--form FILE` is YAML or JSON: a `questions:` list (or a bare list) of
  `{id, type, prompt, help, required, choices, sections, sections_from,
  recommended, recommendation_reason, unanswered_verdict}` with the proto's snake_case keys.
  Types are `single_choice`, `multi_choice`, `variant`, `section_review` and
  `text`. `recommended` is a choice key, a list of keys, a text, or a full
  response mapping. `unanswered_verdict` (section_review) says what a section the
  operator leaves alone means: `approve` (default), `change` or `drop`. `sections_from: <attachment name or path>` turns the `##`
  and `###` headings of an attached Markdown file into review sections (key and
  anchor are the GitHub-style heading slug). A question missing its id, prompt or type, with a
  duplicate id or key, or with `sections_from` on something not attached is
  left out with a warning; a choice or section whose attachment is not
  attached loses only that link. `sections_from` keeps the first 40 headings and
  cuts keys (64 characters), titles and anchors (200) to the Hub's limits; other
  counts and lengths are the Hub's to enforce.
- `answer --responses-file` takes a JSON list of responses, or an object
  `{"responses": [...], "attachment_notes": [{"attachment", "path", "note"}]}`
  (`path` names a file inside a folder attachment). `get` and `wait` show the
  operator's notes on attachments.
- `get`, `list` and `wait` read asks, and the Hub shows an ask only to the owner,
  to a verified agent run for the asks of its conversation, or to the service that
  raised it as an event. The CLI sends the shell's own credential (the agent
  identity token, else `NOTIFICATION_HUB_API_TOKEN`); a shell with neither is
  refused with a message saying what to do. A run's conversation is shared by the
  runs recovered from it, spawned by it and re-attached to the same harness
  session, so a recovered run reads, lists and waits on its predecessor's asks.
  `list` shows only that lineage. An ask outside it answers "not found", except
  that `get --subject-key K` and `wait --subject-key K` take the exact subject key
  instead of an id: a run of the same owner may name it (the Hub never lists by
  key). That is the resume path for an agent that lost its run: re-derive the key,
  then `wait --subject-key K`. A wait by key follows the key when the asker
  replaces the ask. `withdraw` follows the same rule.
- The Hub's packet check prints to stderr after the ask (what it left out, then
  advice), beside the writing-guide reminder; `--json` stdout stays the plain
  response.
- Exit status `3` (the operator asked for more context) is offered only for keyed
  asks: `ask` sets `accepts_context_requests` whenever `--subject-key` is given, and
  `--no-context-requests` opts out. Stdout then shows their question and stderr says
  how to answer it: run `ask --subject-key K` with the answer in
  `--technical-details`, `--big-picture` or a new attachment (everything else stays),
  then `wait` again. An answered `wait` prints the plain-text answer summary.

## Common Make Targets

The project root also exposes a few important Make targets:

```bash
make setup
make build
make install
make test
make hygiene
make test-packages
```

These targets are confirmed in the current root `Makefile`.

Root CLI installation freezes the candidate bytes, verifies its Go package and
native target, and requires a stamped identity canary before publication.
`vrooli version --json` reports that executable's `build_identity`.
`vrooli runtime supervisor status --json` reports the running owner's identity.
After installation, use `vrooli runtime supervisor install --user` for native
adoption and inspect status again. Artifact preparation and fallback byte
retention leave independent activation rollback as a separate obligation.

`vrooli runtime instance inspect <instance-id> [--json]` reads one exact
scenario-instance owner record from the canonical runtime registry in read-only
mode. It reports the scenario, variant, generation, status, start time and any
recorded stop completion time; it does not reconcile lifecycle state. A stopped
instance is owner-liveness evidence and does not by itself prove that its
process descendants are gone.

Use `vrooli setup broker --sudo-mode=ask` to apply a focused broker upgrade;
inspect with `vrooli setup broker --dry-run --json`. Scope and elevation behavior:
[privilege broker setup convergence](../architecture/PRIVILEGE_BROKER.md#setup-convergence).

`vrooli hygiene` is the root hygiene aggregator. Internally, hygiene checks are
moving to a registered-provider pattern so scenario- or domain-specific checks
can plug in without expanding one monolithic service. The plans provider is the
first migrated provider: it asks Plan Manager to dry-run or execute
`ReconcilePlans` when available, and falls back to a read-only static scan when
Plan Manager is down. JSON separates `fixes_applied` (actual mutations such as
imported, mirror repaired, or source removed) from
`plan_reconcile_outcomes` (including no-ops such as skipped duplicate, already
canonical, parse failures, conflicts, source untouched, source retirement planned,
source removed, mirror status, and item errors). `vrooli hygiene --fix-safe
--plans` asks Plan Manager to repair mirrors, canonicalize misplaced markdown, and remove
source files only after they are proven canonical/imported/duplicate.
Parse failures and conflicts are reported as invalid plan sources with a
guided remediation path; safe-fix intentionally leaves them in place until an
agent repairs them into importable plan markdown or moves non-plan notes out of
plan source locations. Use `vrooli hygiene --plans-only --details` to see the
affected paths and Plan Manager parser/conflict reason before rerunning
`vrooli hygiene --fix-safe --plans`.
Human output mirrors that split with a separate "Plan reconcile results"
section.

For individual scenarios, the preferred lifecycle remains:

```bash
cd scenarios/<scenario-name>
make start
make test
make logs
make stop
```

## Shared Go build cache

`vrooli host go-cache` owns the per-user Go build cache (`GOCACHE`, normally
`~/.cache/go-build`). It keeps the cache inside the `build_cache` budget in
`internal/tools/go/tool.json` and never removes an output that a running build
still needs. Run it as the user whose builds it serves, not as root.

```bash
vrooli host go-cache status [--json]          # activation, live sessions, go-command audit, last reclaim
vrooli host go-cache activate [--json]        # self-test the helper, then set GOCACHEPROG in the go env file
vrooli host go-cache plan [--fresh] [--json]  # budget, store size, reclaimable bytes, plan token
vrooli host go-cache reclaim [--token <t>] [--max-bytes <n>] [--time-budget 40s] [--json]
vrooli host go-cache deactivate [--json]      # remove GOCACHEPROG; running builds keep their sessions
```

`activate` installs nothing. It requires `~/.vrooli/bin/vrooli-gocacheprog`,
which `make install` provides. It proves the helper with a private self-test
before writing `GOCACHEPROG` with `go env -w`, and it refuses to replace another
cache program. Go commands started before activation run outside the protocol,
so reclaim refuses until they exit. `plan` and `reclaim` refuse while any go
command uses the cache outside the protocol, and name the process. Examples are
`GOENV=off`, another `GOCACHEPROG`, and Go older than 1.24. `reclaim` without
`--token` plans afresh and applies that plan. `--cache <dir>` selects another
store; Storage Manager passes its declared root this way.

Storage Manager's `go-build-cache` provider invokes `plan` and `reclaim` during
pressure recovery. If the helper binary is ever removed while the protocol is
active, every go command fails until you reinstall the helper or run
`vrooli host go-cache deactivate` (or `go env -u GOCACHEPROG`). The design and
its residual risk are in `internal/gocache/doc.go` and the Storage Manager
[recovery ledger](../../scenarios/storage-manager/docs/reference/recovery-ledger.md#shared-go-build-cache-safety).

## Reviewed host wireless transactions

`vrooli host wireless` keeps unsigned preparation, signed public artifacts,
root transaction publication, radio effects and recovery evidence separate.
Use an exact `--profile noble-mt7927` (Wi-Fi) or `--profile noble-mt6639`
(Bluetooth). Native eligibility remains specific to the reviewed host profile.

All effect actions require `--apply`. Omitting it performs a read-only preview.
`status`, `activation-plan`, `activation-preflight` and `verify-signed-copy`
reject `--apply`. Signing and native root operations require an explicitly
elevated normal command, for example `sudo vrooli host wireless sign ...`;
the command never invokes generic sudo reexecution itself. Public publication
from existing signed copies can run unprivileged.

```bash
vrooli host wireless sign --profile <profile> --bundle <unsigned-generation> \
  --cert <public-certificate> --json
sudo vrooli host wireless sign --profile <profile> --bundle <unsigned-generation> \
  --cert <public-certificate> --receipt-sha256 <reviewed-unsigned-receipt-sha256> --apply --json
vrooli host wireless publish-signed --profile <profile> --bundle <unsigned-generation> \
  --signed-dir <signed-module-directory> --cert <public-certificate> --apply --json
vrooli host wireless record-recovery --profile <profile> \
  --recovery-kernel <distinct-stock-kernel> --recovery-entry <exact-menu-entry> --json
vrooli host wireless stage --profile <profile> --bundle <unsigned-generation> \
  --signed-bundle <signed-receipt-sha256> --receipt-sha256 <reviewed-unsigned-receipt-sha256> \
  --recovery-kernel <distinct-stock-kernel> --recovery-entry <exact-menu-entry> --json
vrooli host wireless status --profile <profile> --transaction <transaction-id> --json
vrooli host wireless activate --profile <profile> --transaction <transaction-id> --json
vrooli host wireless recover --profile <profile> --transaction <transaction-id> --json
vrooli host wireless rollback --profile <profile> --transaction <transaction-id> --json
```

Add `--apply` to a reviewed `record-recovery`, `stage`, `activate`, `recover` or
`rollback` request only when intending that effect. `stage`, `activate`,
`recover` and `rollback` also accept `--maintenance-window` for coordinated
maintenance; native checks still own admission. Stage application requires the signed bundle identity and the
independently retained unsigned receipt digest. A mutable artifact pathname or
its current self-reported digest is insufficient for privileged signing.

`publish-signed` without `--apply` authenticates existing signed copies without
retaining an artifact. Successful publication returns its content-addressed
signed receipt, module directory and retained public certificate paths. Neither
a verified signature nor enrollment proves running kernel acceptance, recovery
bootability or functioning radio hardware. Inspect each operation's checks and
state, and retain the exact transaction ID for status, recovery or rollback.
The durable native boundaries and qualification limits are documented in
[wireless activation and recovery](../../internal/hostwireless/ACTIVATION-DESIGN.md).
