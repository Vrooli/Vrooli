# Vrooli CLI Commands

This document summarizes the current root CLI surface for the project-level platform.

Use `vrooli help` and subcommand `--help` output as the final authority for exact flags and any newly added commands.

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
```

These root commands are confirmed by the current CLI help surface.

## Orientation

```bash
vrooli help
vrooli status
```

- `vrooli help` shows the current command tree.
- `vrooli status` gives a health and status overview.

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
vrooli scenario template
vrooli scenario generate <template> --id <slug> --display-name <name> --description <text>
vrooli scenario completeness <name>
```

Notes:

- `run` is an alias of `start`
- for day-to-day work on one scenario, prefer the scenario-local `make start|test|logs|stop` flow when available

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
  `start_operation` with the current step, dependency n/m, elapsed, an
  honest ETA (`eta_known=false` when there is no recorded phase history —
  never a fabricated number), and `recommended_next_check_seconds`
  (0 terminal — stop checking; 30 unknown ETA; else remaining clamped to
  [5, 60]).
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
vrooli resource info <name>
vrooli resource status
vrooli resource validate
vrooli resource install <name>
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
vrooli resource template
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
vrooli package validate
vrooli package build <name>
vrooli package generate <name>
vrooli package refresh <name>
vrooli package audit
```

`vrooli package audit --all --json` includes `audit.scan_stats`, which reports
bounded docs-drift scan counts, skipped categories, bytes scanned, and whether a
text-scan budget was exceeded. Runtime data and generated output skips are
normal for live development trees.

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
