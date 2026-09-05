# Vrooli CLI Commands

`vrooli capacity list` shows the live ledger by default. Use
`vrooli capacity list --all` when terminal claim history is required.
This document summarizes the current root CLI surface for the project-level platform.

Use `vrooli help` and subcommand `--help` output as the final authority for exact flags and any newly added commands.

## Prompt-manager skill governance

```bash
prompt-manager skill import --help
prompt-manager skill review-import --help
prompt-manager skill import-staleness --help
VROOLI_SKILL_PROJECTION_DIRS="$HOME/.claude/skills:$HOME/.codex/skills:$HOME/.config/opencode/skills:$HOME/.gemini/skills:$HOME/.grok/skills" prompt-manager skill topology --json
```

`skill import` requires a local pinned source directory, immutable commit,
SPDX license, exact `sha256:` checksum, importer, and skill ID. Imported skills
land inactive in `packs/vendor` until `review-import` records a verdict from a
different reviewer. `import-staleness` compares the recorded upstream version
with a current version supplied by the operator.

`skill topology` reports every indexed skill's pack and whether its generated
projection is resident in each configured target, including a resident
metadata-token estimate. It accepts either `--projection-dir` or the
path-list environment variables `VROOLI_SKILL_PROJECTION_DIR` and
`VROOLI_SKILL_PROJECTION_DIRS`.

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
from `.vrooli/operator-state.json` `core.seed` and canonical scenario
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
vrooli scenario restart <name>
vrooli scenario wait <name> [--timeout <seconds>] [--json]
vrooli scenario stop <name>
vrooli scenario stop-all
vrooli scenario test <name>
vrooli scenario logs <name>
vrooli scenario open <name>
vrooli scenario port <name>
vrooli scenario ui-smoke <name>
vrooli scenario requirements
template-manager registry list --kind scenario
template-manager generate <template> --id <slug> --display-name <name> --description <text>
vrooli scenario completeness <name>
```

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
vrooli resource acquisition prune
vrooli resource acquisition prune "<name>"
vrooli resource uninstall <name>
vrooli resource start <name>
vrooli resource restart <name>
vrooli resource start-all
vrooli resource stop <name>
vrooli resource stop-all
vrooli resource logs <name>
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

## Common Make Targets

The project root also exposes a few important Make targets:

```bash
make setup
make build
make install
make test
make hygiene
make validate-package-governance
```

These targets are confirmed in the current root `Makefile`.

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
