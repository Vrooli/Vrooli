# Pre-baseline safety admission report

Status: **safe fixture gate green; comprehensive admission still degraded**  
Measured: 2026-09-06  
Schema: `gct.safety-admission.v1`

## Decision

The scenario is not yet eligible for an automated baseline or full suite. A
temporary directory is a real Git repository and does not make mutation safe.
The admission gate therefore rejects any test path that invokes a repository
mutator, even when the path is cleaned up by the test framework.

The machine-readable classifier is implemented by
`api/internal/safetyadmission`. It is pure and receives command inventories;
it does not execute processes or touch a repository. Read-only Git commands
are eligible. Repository mutations, external writes, and unknown commands are
not eligible.

The API test-tree gate is `TestSafetyAdmissionGate`. It scans all API test
sources for Git repository setup and mutator helper calls and currently passes.
This is narrower than full scenario admission: delegated Agent Manager
bootstrap and existing BAS/API writer expectations still require separate
review.

## Current findings

The original inventory found these areas. The commit, staging, repository
status, path-snapshot, file-content, diff, and baseline handler fixtures have
now been converted to immutable records, filesystem-only fixtures, or
recording fakes. The remaining precondition is a static/audited gate over the
whole API test tree plus any delegated bootstrap path.

| Area | Observed effect | Required disposition |
|---|---|---|
| `api/repo_status_service_test.go` | `init`, branch creation, `add`, `commit`, `mv` | **Converted** to porcelain/status and diff records supplied to `FakeGitRunner`. |
| `api/precommit_hook_test.go` | `init`, `config` | **Converted** to controlled `.git/hooks` filesystem fixtures and an injected read-only config reader. |
| `api/internal/baseline/path_snapshot_test.go` | `init` | **Converted** to an explicit immutable candidate-path seam. |
| `api/handlers/baseline/connect_handler_test.go` | `init` | **Converted** to an injected source-evidence fixture. |
| `api/file_content_service_test.go` | `SetupTestRepo` (transitive `init`) | **Converted** to filesystem-only temporary directories. |
| `api/diff_service_test.go` | `init`, `add`, `commit` (via integration fixtures) | **Converted** to `FakeGitRunner` diff records. |
| `api/commit_service_test.go` | `add` | **Converted** to `FakeGitRunner`. |
| `api/staging_service_test.go` | direct read after staging | **Converted** to fake state/call assertions. |

The production `ExecGitRunner` remains the runtime adapter for human-scoped
operations. This report does not claim that adapter is safe for agent
actuation; Phase 2 must enforce that boundary in the transport and domain
layers.

## Admission rule

An automated validation run is admitted only when every resolved subprocess
command is classified as `pure_read`, and every writer-facing test uses a fake
that records the intended decision without invoking a writer. The current
test-tree census has no Git subprocess call sites outside the production
adapter and no calls to the removed mutation fixture helpers. Missing
historical evidence is recorded as `unknown` or `degraded`; it is never
reinterpreted as a pass.

## Explicit pre-change evidence standing

No pre-change baseline is claimed. The current worktree contains concurrent
changes outside this scenario, and the safety admission gate is not yet green.
After the test paths above are converted, the owner workflow may request a
degraded anchor. Until then, no baseline capture or comprehensive scenario
suite should be started.
