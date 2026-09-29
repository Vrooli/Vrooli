# Workspace Sandbox Auditability Contract

This document is the canonical, implementation-co-located mirror of the auditability contract authored in `scenarios/swarm-manager/research/agent-sandbox-auditability-contract/conclusion.md`. The conclusion is the source of truth; this file is for discoverability inside the workspace-sandbox surface.

## Purpose

The primary purpose of `workspace-sandbox` when used as the default execution path for `agent-manager` coding runs is **auditability**: every coding run produces a durable, per-run provenance record correlating repository changes back to the run, conversation, cost, and execution context that produced them. Protected mode (`mode=protected`) is now the production default and adds containment on top of auditability — bwrap isolation, network mode enforcement, and git-verb allowlist are enforced on the agent process tree itself, not just on its file output.

## Locked defaults

| Lever | Default |
|---|---|
| `mode` | `protected` (default, post-Slice 4 of `execute/protected-sandbox-agent-launch`). The agent process tree itself runs inside workspace-sandbox `/processes`; agent-manager wires a `SandboxLauncherFactory` for every coding-agent runner. `tracking` is the explicit operator opt-out for runs that legitimately need full host capability (e.g., `git push` after review, scraping a remote URL). See [Protected-mode enforcement](#protected-mode-enforcement) below. |
| `manualReview` | `false` (opt-in only) |
| `autoApply` | `true` |
| `applyOnFailure` | `true` |
| `lock` | `false` |
| `networkMode` | `localhost` (supports `none|localhost|full`) |
| `sandboxCreation` | eager (created at run start regardless of writes) |
| Agent-side awareness | none (no prompt or behavior changes based on sandbox mode) |

## Apply-timing state machine

- **`manualReview=false` (default)**: in-acceptance changes auto-apply at turn end through `/turn-checkpoint`; out-of-acceptance changes persist as `state=pending-review` provenance and remain in the sandbox. The sandbox is then unmounted and marked `checkpointed` so the same logical sandbox can be resumed for a follow-up turn.
- **`manualReview=true` (opt-in)**: no apply at run end; all changes persist as `state=pending-review`. The sandbox persists beyond run end until the operator approves or denies. Approval can come from any of three surfaces (git-control-tower AI Changes, agent-manager run-detail diff, workspace-sandbox sandbox-detail diff); the originating surface is recorded on the resulting state transition for audit.
- Apply behavior is identical regardless of run outcome. `runOutcome` ∈ {`success`, `failure`, `cancelled`, `timeout`} is captured on the provenance record but does not gate apply.

Regular binary additions, modifications and deletions carry Git binary patches
with original and resulting blob identities. The same payload serves review,
application and archival capture. Application verifies the original bytes;
conflicting binary content fails without replacing it. `git apply` handles these
payloads for both Git repositories and ordinary workspace directories. The
`TestBinaryChangesApplyExactBytes` regression covers all three operations,
scope-prefixed paths with spaces, executable modes and conflicting originals.

Symlink evidence stores the link target and link mode, not the referenced file's
contents. Dangling and external links remain reviewable, patchable and archivable
without reading the referent. Diff previews and archive capture use root-scoped
reads that refuse parent-directory escapes. Deleting a sandbox preserves link
evidence before teardown; a dangling link does not require manual removal.

## Provenance schema additions

- `runOutcome` ∈ {`success`, `failure`, `cancelled`, `timeout`} on `ProvenanceRunGroup` (run-level).
- `state` ∈ {`applied`, `pending-review`, `denied`} on `ProvenanceFile` (per-file). Per-file granularity lets a single run mix in-acceptance applied files with out-of-acceptance pending-review files without splitting the run group.
- The provenance query API supports filtering by `state=pending-review` so the AI Changes review queue is the existing endpoint filtered, not a parallel surface.

Workspace-sandbox owns state transitions; git-control-tower reads them.

## Locking and acceptance are orthogonal

`NoLock` controls only mutual exclusion. Acceptance allow/deny rules are evaluated against every candidate apply regardless of `NoLock`. Multiple concurrent sandboxes over the same scope can coexist (per `lock=false` default) and acceptance still gates each independently.

The historical `noLock`-implies-accept-all shortcut in `service.go` is removed by `fix/workspace-sandbox-lock-and-acceptance-semantics`.

## Pending-to-committed lifecycle

Each `AppliedChange` also preserves an optional SHA-256 content digest and the
revision known when the evidence was recorded. The digest is computed from the
canonical file after apply when the file remains readable; deleted or
unreadable files retain an explicit empty digest rather than a guessed value.
Readers may use these fields for content-aware joins, but file path overlap or
an applied timestamp alone never establishes line authorship.

Git-control-tower auto-promotes a pending provenance record to committed when it detects a commit whose changed files overlap the pending record's file set.

### Capture and closure invariants

Capture uses one shared changed-file walker for every driver. Git internals and
gitignored, untracked artifacts are excluded before a `FileChange` is created;
tracked dot-prefixed files remain eligible for both apply and provenance. The
driver-specific overlay artifacts are separately excluded, and a contract test
pins equivalent output across overlay and copy drivers.

Commit reconciliation is bounded by `CommitResolutionBatchLimit`. An unresolved
row is retried while its path may still gain a commit. Once it is older than
`CommitResolutionHorizon` and Git reports it untracked, it is stamped
`unresolvable_at` exactly once and no longer selected. Retention deletes only
those retired, uncommitted rows after `UnresolvedProvenanceRetention`; a row
with a real commit hash is never eligible for that deletion.

## Source-of-truth interaction matrix

| Surface | Role |
|---|---|
| `agent-manager` run executor | Run lifecycle, eager sandbox creation, env injection (`VROOLI_SANDBOX_ID`, `VROOLI_SANDBOX_MERGED`, `VROOLI_SANDBOX_SCOPE`), apply at run end when `manualReview=false` |
| `agent-manager` UI | Run-detail diff view; approval surface for `pending-review` provenance |
| `workspace-sandbox` service | Overlay creation, mutation tracking, acceptance evaluation, apply, teardown hooks, persistence beyond run end when `manualReview=true`, owns state transitions |
| `workspace-sandbox` UI | Sandbox-detail diff view; approval surface |
| `internal/scenario` + `internal/lifecycle` + `internal/shell` + `internal/cli/vroolicli` | Sandbox-aware scenario restart; scope-narrowed redirect using `VROOLI_SANDBOX_*` env vars |
| `packages/cli-core/cliutil/sandbox.go` + `cmd/sandbox-resolve` | Path resolution for arbitrary CLIs |
| `test-genie` CLI | Sandbox-aware test execution |
| `workspace-sandbox` `TeardownHooks` | Invokes Go-based `vrooli scenario heal-from-sandbox` on teardown |
| `git-control-tower` API | Provenance-by-run query, state-filtered query for review queue, commit linkage |
| `git-control-tower` UI | AI Changes tab, review queue via `state=pending-review`, approval surface |

## Validation matrix

The contract required the nine behaviors in Finding 5 of the source-of-truth conclusion to pass on the agent-manager UI and swarm-manager queue spawn surfaces before the default flipped. They did, and Slice 4 of `execute/protected-sandbox-agent-launch` flipped the default to `protected`. See `execute/sandbox-runtime-e2e-verification` for the original readiness checklist; ongoing parity is enforced by the per-runner protected-mode coverage documented in [`agent-manager/docs/PROTECTED_MODE_RUNNERS.md`](../../agent-manager/docs/PROTECTED_MODE_RUNNERS.md).

## Protected-mode enforcement

### Runtime workspace write policy

`behavior.writePolicy` is persisted with the sandbox. Agent Manager supplies it
from `sandboxConfig.writePolicy`. It is separate from apply-time acceptance.

- Omitted policy preserves an unrestricted writable workspace. Present policy
  with `paths: []` makes the workspace read-only.
- Each entry grants an existing literal workspace-relative file or directory.
  Directory grants include descendants. Root, globs, traversal, overlapping
  entries, missing paths and symlink components in a grant are rejected.
- Linux required containment mounts every workspace alias read-only, then
  mounts only the granted paths writable. This includes `/workspace`, the
  merged host path and the project path, even when ordinary mirroring is off.
- Before launch, the backend scans granted trees and refuses existing regular
  files with multiple hardlinks. Otherwise a writable alias could change an
  owner file despite its read-only path. The scan does not follow symlinks.
- Shared host PID namespaces and writable profile binds overlapping a workspace
  alias are refused. Unsupported containment modes/platforms refuse the policy;
  they never run it as advisory. Prefer directory grants for editor atomic-save
  behavior; an individual file bind permits in-place writes, not replacement
  of that mountpoint.

The owner must select grants that exclude its controls and must not concurrently
modify their mount topology or inode aliases during launch. This is a workspace
write boundary, not a network, API-authority or host-read restriction. The
`workspace-write-policy` capability advertises backend support; consumers must
also check the actual sandbox's persisted policy. The live no-model regression
is `TestLiveWorkspaceWritePolicy` with `-args -live-bwrap-aliases` in
`api/internal/driver/exec`; it checks allowed writes and denied control edits,
creation, deletion, rename, symlink escapes and Git metadata across all aliases.

### Read-only launch policy files

`/exec` and `/processes` accept `policyFiles`, each with absolute `source`,
absolute `target`, and a hexadecimal `sha256`. Sources must be regular files
below registered roots. Required Linux containment verifies content identity
and mounts both paths read-only. It refuses symlink sources, multiply linked
inodes, shared PID namespaces, overlapping policy aliases and collisions with
workspace, writable-profile or masked paths. Other containment modes and
platforms refuse these requests. Owners must keep source bytes and aliases stable
through launch; this is not isolation from a concurrently mutating host owner.

Consumers must require `read-only-policy-files` before sending policy-bearing
requests. An older provider can ignore an unknown JSON field. The policy source
belongs outside writable runtime directories. Workspace Sandbox does not compile
runner-specific policy; the caller owns its content and consumer path. The
no-model `TestLivePolicyFiles` check uses `-args -live-bwrap-aliases` and proves
reads succeed while writes and deletion fail at both aliases.

### Process isolation

When a run's `SandboxConfig.Mode == protected`, agent-manager's runner launches the agent process tree through workspace-sandbox `/processes` instead of `os/exec` on the host. The same workspace that backs tracking-mode auditability now also enforces:

- **Bwrap isolation** — the agent process inherits the sandbox's namespace boundaries.
- **Network mode** — the sandbox's `NetworkMode` (`none` / `localhost` / `full`) is translated to an isolation profile and applied at the OS level (`agent-manager/api/internal/adapters/sandbox/workspace_sandbox.go`). Enforcement is binary today: `none` denies all network, while `localhost` and `full` both grant unrestricted network — no backend enforces loopback-only (`network-loopback-only` in the containment vocabulary; tracked as `knw-1784006975589682125`). Agent-manager's launcher selector warns on the run timeline whenever that enforcement is missing.
- **Git-verb allowlist** — `Behavior.Protected.GitAllowlist` is consulted on **both** `/exec` and `/processes` ingress. Without `/processes` enforcement, an agent launched via the protected-mode path could call `git push` directly and bypass the `/exec` guard. The handler returns a structured 403 (`{"error": "git_verb_blocked", "verb": "...", "message": "..."}`) which agent-manager surfaces as a typed `*sandbox.LaunchBlocked` run-level error rather than a hang or corrupted run state.

The symmetric `/exec` and `/processes` allowlist enforcement is tested in:

- `scenarios/workspace-sandbox/api/internal/handlers/process_git_allowlist_test.go` — `/exec` (4 tests, pre-existing)
- `scenarios/workspace-sandbox/api/internal/handlers/process_start_git_allowlist_test.go` — `/processes` (4 tests, added by `execute/protected-sandbox-agent-launch`)

The runner-side counterpart (the `runner.Launcher` seam that picks `HostLauncher` vs `SandboxLauncher` per run) is documented in `scenarios/agent-manager/docs/SEAMS.md` § "Process Launcher" and the per-runner capability matrix in `scenarios/agent-manager/docs/PROTECTED_MODE_RUNNERS.md`.

## Source of truth

`scenarios/swarm-manager/research/agent-sandbox-auditability-contract/conclusion.md` (Findings 1–6).
