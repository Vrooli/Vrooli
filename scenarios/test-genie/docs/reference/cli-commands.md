# Test Genie CLI Reference

Test Genie is an execution and remediation control surface. It does not accept
coverage targets, test-type generation requests, arbitrary source-file lists,
or generic agent prompts.

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
| `execute <scenario>` | Start a descriptor-planned, server-owned test execution. |
| `runs` | Wait for, follow, inspect, abort, compare, or browse durable executions. |
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
```

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
the execution. The `runs` command also provides `status`, `follow`, `list`,
`show`, `compare`, and freshness/history operations; refer to `test-genie runs
--help` for exact syntax.

For a pending run, `runs status` is a nonblocking snapshot that prints one
canonical `runs wait --json --timeout=…` command. Its JSON response adds
`nextAction` with `kind=wait`, the exact command and timeout, and
`doNotPoll=true`. Terminal status responses omit that action. Any recommended
next-check interval is dashboard backoff metadata, not the agent completion
workflow. The `vrooli scenario test status` proxy follows the same contract.

## Remediate findings

Create a job only from stable finding IDs emitted by a completed execution.
Agent Manager owns workspace protection, policy, tools, network, review, and
application behavior. Test Genie constructs the immutable evidence packet and
records Agent Manager provenance.

```bash
test-genie remediate my-scenario \
  --execution <execution-id> \
  --findings <stable-finding-id,...> \
  --requirements <requirement-id,...> \
  --role code.default \
  --context "Keep the existing public API intact."
```

`--requirements` and `--context` are optional. A completed agent task is
provisional: the job becomes verified only after Test Genie records a new
server-owned execution and compares stable finding IDs.

## Live descriptor planning

Use the current descriptor registry instead of copying phase names into scripts
or documentation:

```bash
test-genie phases --help
test-genie execute my-scenario --preset quick
test-genie execute my-scenario --phases workflow
```

The available phases, applicability, documentation links, runnability, and
timeouts are calculated by the server for the selected scenario.

## Requirements and workflow registry

```bash
test-genie requirements --help
test-genie registry build
test-genie registry validate
```

Requirement evidence remains readable independently of remediation. Select
requirement IDs when creating a remediation job to preserve their validation
context in the same evidence packet.

## Provider and operational inspection

```bash
test-genie health
test-genie status
test-genie provider-contract scan --json
test-genie fleet status --json
```

These commands report existing evidence and provider posture; they do not
generate tests or enforce Agent Manager security policy.

## Automation example

```bash
test-genie execute my-scenario --preset comprehensive
# Copy the printed run ID, then block once until its terminal verdict.
test-genie runs wait --json --timeout=840 my-scenario <run-id>
```

For the HTTP contract and remediation endpoints, see
[API Endpoints](api-endpoints.md). For the operator flow, see
[Evidence-Driven Remediation](../guides/test-generation.md).
