# Workspace Sandbox Auditability Contract

This document is the canonical, implementation-co-located mirror of the auditability contract authored in `scenarios/swarm-manager/research/agent-sandbox-auditability-contract/conclusion.md`. The conclusion is the source of truth; this file is for discoverability inside the workspace-sandbox surface.

## Purpose

The primary purpose of `workspace-sandbox` when used as the default execution path for `agent-manager` coding runs is **auditability**: every coding run produces a durable, per-run provenance record correlating repository changes back to the run, conversation, cost, and execution context that produced them. Containment and protection are deferred to a separate `protected` mode.

## Locked defaults

| Lever | Default |
|---|---|
| `mode` | `tracking` (`protected` reserved; requesting it errors until implemented) |
| `manualReview` | `false` (opt-in only) |
| `autoApply` | `true` |
| `applyOnFailure` | `true` |
| `lock` | `false` |
| `networkMode` | `localhost` (supports `none|localhost|full`) |
| `sandboxCreation` | eager (created at run start regardless of writes) |
| Agent-side awareness | none (no prompt or behavior changes based on sandbox mode) |

## Apply-timing state machine

- **`manualReview=false` (default)**: in-acceptance changes auto-apply at run end; out-of-acceptance changes persist as `state=pending-review` provenance and remain in the sandbox.
- **`manualReview=true` (opt-in)**: no apply at run end; all changes persist as `state=pending-review`. The sandbox persists beyond run end until the operator approves or denies. Approval can come from any of three surfaces (git-control-tower AI Changes, agent-manager run-detail diff, workspace-sandbox sandbox-detail diff); the originating surface is recorded on the resulting state transition for audit.
- Apply behavior is identical regardless of run outcome. `runOutcome` ∈ {`success`, `failure`, `cancelled`, `timeout`} is captured on the provenance record but does not gate apply.

## Provenance schema additions

- `runOutcome` ∈ {`success`, `failure`, `cancelled`, `timeout`} on `ProvenanceRunGroup` (run-level).
- `state` ∈ {`applied`, `pending-review`, `denied`} on `ProvenanceFile` (per-file). Per-file granularity lets a single run mix in-acceptance applied files with out-of-acceptance pending-review files without splitting the run group.
- The provenance query API supports filtering by `state=pending-review` so the AI Changes review queue is the existing endpoint filtered, not a parallel surface.

Workspace-sandbox owns state transitions; git-control-tower reads them.

## Locking and acceptance are orthogonal

`NoLock` controls only mutual exclusion. Acceptance allow/deny rules are evaluated against every candidate apply regardless of `NoLock`. Multiple concurrent sandboxes over the same scope can coexist (per `lock=false` default) and acceptance still gates each independently.

The historical `noLock`-implies-accept-all shortcut in `service.go` is removed by `fix/workspace-sandbox-lock-and-acceptance-semantics`.

## Pending-to-committed lifecycle

Git-control-tower auto-promotes a pending provenance record to committed when it detects a commit whose changed files overlap the pending record's file set.

## Source-of-truth interaction matrix

| Surface | Role |
|---|---|
| `agent-manager` run executor | Run lifecycle, eager sandbox creation, env injection (`VROOLI_SANDBOX_ID`, `VROOLI_SANDBOX_MERGED`, `VROOLI_SANDBOX_SCOPE`), apply at run end when `manualReview=false` |
| `agent-manager` UI | Run-detail diff view; approval surface for `pending-review` provenance |
| `workspace-sandbox` service | Overlay creation, mutation tracking, acceptance evaluation, apply, teardown hooks, persistence beyond run end when `manualReview=true`, owns state transitions |
| `workspace-sandbox` UI | Sandbox-detail diff view; approval surface |
| `workspace-sandbox` toolregistry | Tool-level `RequiresApproval` for canonical-repo-modifying tools that bypass the sandbox (e.g. direct git commit) |
| `internal/scenario` + `internal/scenarioexec` + `internal/cli/vroolicli` | Sandbox-aware scenario restart; scope-narrowed redirect using `VROOLI_SANDBOX_*` env vars |
| `packages/cli-core/cliutil/sandbox.go` + `cmd/sandbox-resolve` | Path resolution for arbitrary CLIs |
| `test-genie` CLI | Sandbox-aware test execution |
| `workspace-sandbox` `TeardownHooks` | Invokes Go-based `vrooli scenario heal-from-sandbox` on teardown |
| `git-control-tower` API | Provenance-by-run query, state-filtered query for review queue, commit linkage |
| `git-control-tower` UI | AI Changes tab, review queue via `state=pending-review`, approval surface |

## Validation matrix

The contract requires the nine behaviors in Finding 5 of the source-of-truth conclusion to pass on the agent-manager UI and swarm-manager queue spawn surfaces before the default flips. See `execute/sandbox-runtime-e2e-verification` for the readiness checklist.

## Source of truth

`scenarios/swarm-manager/research/agent-sandbox-auditability-contract/conclusion.md` (Findings 1–6).
