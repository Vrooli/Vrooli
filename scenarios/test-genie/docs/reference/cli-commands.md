# Test Genie CLI Reference

Test Genie is an execution and remediation control surface. Validation targets
are either a legacy bare scenario slug or an explicit repository target in
`kind:id` form. It does not accept arbitrary source-file lists or generic agent
prompts.

## Global options

```text
test-genie [--api-base <url>] [--instance <name>] [--auto-start] [--dry-run] \
  [--color|--no-color] <command>
```

Run `test-genie --help` for the current command inventory and
`test-genie <command> --help` for command-specific flags. The CLI is installed
by the Test Genie lifecycle; start it with `make start` when needed.

## Primary commands

| Command | Purpose |
|---|---|
| `execute <target>` | Start a descriptor-planned, server-owned execution for a scenario or repository target. |
| `runs` | Wait for, follow, inspect, abort, compare, cost, or browse durable executions. |
| `remediate <scenario>` | Create one evidence-bound remediation job from a completed execution. |
| `phases` | Inspect the live descriptor-backed phase catalog and execution plan. |
| `requirements` | Inspect and synchronize requirement evidence. |
| `health` / `status` | Inspect Test Genie self-health or API liveness. |
| `registry` | Build or validate Browser Automation Studio workflow registry data. |
| `provider-contract` | Validate provider maturity-assessment contracts. |
| `fleet` | Summarize stored execution health across scenarios. |

## Execute and wait

```bash
test-genie execute my-scenario --preset comprehensive
# The same command accepts first-class non-scenario targets.
test-genie execute package:api-core --preset quick
```

If admission rejects the request as saturated, no run was started. Human output
shows the saturated capacity and retry hint. A `--json` start-error object
includes `retryable: true` and the typed `admissionSaturation` detail; wait for
capacity before submitting once again.

Executions are server-owned. If the initiating terminal disconnects, the run
continues. The command prints a run ID and one re-attach command; use it once
instead of polling:

```bash
test-genie runs wait --json --timeout=840 my-scenario <run-id>
```

`--json` writes an immediate attachment receipt to stderr and reserves stdout
for its one terminal JSON snapshot. If a coding-tool session detaches after the
receipt without terminal JSON, the wait process or server-owned run may still
be active; do not infer a failure. Read `runs status --json` once and use its
typed next action.

Use `test-genie runs abort my-scenario <run-id>` only when you intend to cancel
the execution. See the [server-owned execution guide](../guides/sync-execution.md)
for abort wait and reattachment behavior. The `runs` command also provides `status`, `follow`, `list`,
`show`, `compare`, `cost`, and freshness/history operations; refer to `test-genie runs
--help` for exact syntax.

For a pending run, `runs status` is a nonblocking snapshot that prints one
canonical `runs wait --json --timeout=…` command. Its JSON response adds
`nextAction` with `kind=wait`, the exact command and timeout, and
`doNotPoll=true`. Terminal status responses omit that action. Any recommended
next-check interval is dashboard backoff metadata, not the agent completion
workflow. The `vrooli scenario test status` proxy follows the same contract.

## Run artifacts

Every `runs` command addresses a run as `<scenario> <run-id>`, the same order
`runs wait` and `runs status` use; `--scenario <scenario>` is accepted in place
of the first argument (cli-core accepts any positional by its name).

Print a completed run's phase command logs with `test-genie runs logs
<scenario> <run-id> [--phase <phase>]`; several phases print one after another
under a `==> phase <name>: …` header. List a run's typed artifact catalog with
`test-genie runs artifacts <scenario> <run-id>`. Inspect one entry with
`test-genie runs artifact <scenario> <run-id> <artifact-id>`. The CLI exposes an
opaque artifact ID and routes byte transfer through its download command; it
never prints the internal HTTP route. Read an artifact's bytes on stdout, or
publish them to a file:

```bash
test-genie runs download-artifact my-scenario <run-id> <artifact-id> | jq .
test-genie runs download-artifact my-scenario <run-id> <artifact-id> --out ./evidence.json
```

The terminal `runs wait --json` object carries `runHandle.logs` and
`runHandle.artifacts` with these commands filled in for the run.

When a phase producer persists command logs or raw output, the catalog kinds
are `command.log` and `command.output`; availability depends on the producer and
run. Filter to them with
`test-genie runs artifacts my-scenario <run-id> --kinds command.log,command.output`,
then download the selected artifact by ID; `runs logs` does this for every
`command.log` in one call. This is the supported log retrieval path for
completed runs; `runs follow` is for live progress. For Unit Health,
the unfiltered `findings.report` also preserves the typed provider response,
including each command's bounded stdout/stderr excerpts. `test-genie runs
findings` is a compact phase-standing summary; it does not replace that report.
A missing `command.output` artifact therefore does not imply that command-level
failure details are absent.

With `--out`, the destination must be in an existing directory. Existing files are preserved
unless `--overwrite` is supplied. The command writes through a same-directory
temporary file and publishes using the host filesystem's rename semantics. If
the host cannot replace an existing file by rename, the command fails and keeps
the old destination. It checks the byte count against artifact metadata and
reports the downloaded bytes' SHA-256 digest; it does not compare that digest
with an expected content hash. Use this
command for logs, JSON reports, screenshots, and other cataloged payloads
instead of copying the opaque path into an ad hoc HTTP request.

## Remediate findings

Create a job only from stable finding IDs emitted by a completed execution.
Agent Manager owns workspace protection, policy, tools, network, review, and
application behavior. Test Genie constructs the immutable evidence packet and
records Agent Manager provenance.

```bash
test-genie remediate create my-scenario --execution "<execution>" --findings "<findings>" --requirements "<requirements>" --role code.default --context "Keep the existing public API intact."
```

`--requirements` and `--context` are optional. A completed agent task is
provisional: the job becomes verified only after Test Genie records a new
server-owned execution and compares stable finding IDs.

## Live descriptor planning

Use the current descriptor registry instead of copying phase names into scripts
or documentation:

```bash
test-genie phases list --json
test-genie execute my-scenario --preset quick
test-genie execute my-scenario --phases workflow
```

The available phases, applicability, documentation links, runnability, and
timeouts are calculated by the server for the selected scenario.

## Requirements and workflow registry

```bash
test-genie requirements sync --dir scenarios/test-genie --scenario test-genie
test-genie registry build
```

The local registry command supports `build`; it has no `validate` subcommand.
Requirement evidence remains readable independently of remediation. Select
requirement IDs when creating a remediation job to preserve their validation
context in the same evidence packet.

## Provider and operational inspection

```bash
test-genie health
test-genie status
test-genie provider-contract scan --json
test-genie fleet status --json
test-genie runs cost --scenario my-scenario --window 604800 --compare-window 604800 --json
```

These commands report existing evidence and provider posture; they do not
generate tests or enforce Agent Manager security policy.

Fleet JSON is wrapped in `fleetHealth`. Its run counts and alerts summarize
stored server-owned runs in the selected look-back window; they are not a live
test or a statement that the latest run is still active. Use each scenario's
`lastOutcome` and `ageDays` to distinguish a recent failure from an older one.
Add `--roster` to include `neverTestedInWindow` coverage gaps, then summarize the
payload without printing every alert:

```bash
test-genie fleet status --json --roster
```

The JSON fields `windowDays`, `capturedAt`, `scenariosTested`, `scenariosTotal`,
`totalRuns`, `failedPhaseObservations`, and `neverTestedInWindow` summarize the
selected window. The `scenarios` array contains the most-errored scenarios.

## Automation example

```bash
test-genie execute package:api-core --preset comprehensive
# Copy the printed run ID, then block once until its terminal verdict.
test-genie runs wait --json --timeout=840 package:api-core <run-id>
```

For the HTTP contract and remediation endpoints, see
[API Endpoints](api-endpoints.md). For the operator flow, see
[Evidence-Driven Remediation](../guides/test-generation.md).
