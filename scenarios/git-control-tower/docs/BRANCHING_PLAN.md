# Branching Controls Plan (Git Control Tower)

## Context Snapshot
- Scenario: `git-control-tower` (API + CLI + UI) with existing status/diff/stage/commit/push/pull.
- Current branch support: read-only display via `RepoStatus.branch` in UI header.
- Gap: no branch list/switch/create/publish workflows, limiting usability.
- Architectural seams: `GitRunner` interface, service-layer deps structs, pure parsers, and test fakes (`FakeGitRunner`) documented in `docs/SEAMS.md`.

## Objectives
- Provide safe, intuitive branch controls (list, create, switch, publish).
- Preserve strict boundaries: UI/CLI call API; API service enforces safety rules; GitRunner is the sole side-effect surface.
- Add test seams for unit, integration, parser, and UI tests.

## UX Proposal (Safe + Intuitive)
- Branch chip in header becomes a dropdown trigger (desktop + mobile).
- Dropdown layout:
  - Search input (filters by name).
  - Sections: "Current", "Local", "Remote".
  - Each row shows: branch name, upstream (if any), ahead/behind badges, last commit OID (short).
  - Current branch highlighted with "Current" badge.
- Actions:
  - Switch: on row click for local branch.
  - Create: inline "New branch" action (name + optional base).
  - Publish: visible only when upstream missing or mismatched; uses `push -u`.
- Safety behavior:
  - Switching checks for dirty working tree; default is block with clear warning.
  - "Switch anyway" requires explicit confirmation (checkbox in modal).
  - Publish checks sync status; if behind remote or fetch stale, require fetch/recheck.
- Messaging:
  - Warnings use precise, actionable text (no fluff).
  - Buttons disabled with reason tooltips.

## Architecture Plan (Screaming + Seams)
### New Domain Slice: Branch Management
- `api/branch_model.go` for request/response types.
- `api/branch_parser.go` for pure parsing of `git for-each-ref` output.
- `api/branch_service.go` for business rules and safety checks.
- `api/branch_handler.go` for HTTP transport and validation.

### API Endpoints (Proposed)
- `GET /api/v1/repo/branches`
  - Returns local + remote branch list with upstream and ahead/behind.
- `POST /api/v1/repo/branch/create`
  - Body: `{ name, from?, checkout? }`
- `POST /api/v1/repo/branch/switch`
  - Body: `{ name, allow_dirty?: boolean }`
- `POST /api/v1/repo/branch/publish`
  - Body: `{ remote?, branch?, set_upstream?: true }`

### GitRunner Extensions (Seam Surface)
- `Branches(ctx, repoDir string) ([]byte, error)` using stable format:
  - `git for-each-ref --format="%(refname:short)|%(upstream:short)|%(objectname:short)|%(committerdate:iso8601)" refs/heads refs/remotes`
- `CreateBranch(ctx, repoDir, name, from string) error`
- `CheckoutBranch(ctx, repoDir, name string) error`
- `TrackRemoteBranch(ctx, repoDir, remote, name string) error` (optional; remote-only branch)

### Service-Layer Safety Rules
- Switching:
  - If dirty and `allow_dirty` is false: return a typed warning and block.
  - If target is remote-only: create local branch tracking remote, then checkout.
- Creating:
  - Validate name and check branch existence before creating.
  - Optionally checkout immediately if `checkout` is true (with dirty check).
- Publishing:
  - If no upstream: push with `set_upstream`.
  - If behind remote: return warning and block unless explicitly overridden.

## Data Shapes (UI/CLI)
### Branch List Response (Example)
```
{
  "current": "feature/branch",
  "locals": [
    {
      "name": "feature/branch",
      "upstream": "origin/feature/branch",
      "ahead": 2,
      "behind": 0,
      "oid": "a1b2c3d",
      "last_commit_at": "2025-01-01T12:34:56Z"
    }
  ],
  "remotes": [
    {
      "name": "origin/other-branch",
      "oid": "d4e5f6a",
      "last_commit_at": "2025-01-01T10:00:00Z"
    }
  ],
  "timestamp": "2025-01-01T12:35:00Z"
}
```

## Testing Plan (Seams + Coverage)
- Parser tests:
  - `ParseBranches` handles local + remote, missing upstream, and detached states.
- Service tests (FakeGitRunner):
  - Switch blocked on dirty when `allow_dirty` false.
  - Create with existing branch returns error.
  - Publish with no upstream uses `Push(set_upstream=true)`.
- Integration tests (real git):
  - Create and switch branch on a real repo.
  - Publish sets upstream and reports success.
- UI tests:
  - Branch dropdown shows current branch.
  - Switch blocked with dirty warning.
  - Publish button visible for unpublished branches.
- BAS tests:
  - API workflow coverage for branch list/switch/create/publish.

## Phased Checklist (Comprehensive)
### Phase 0: Discovery + Alignment
- [ ] Confirm desired behavior for dirty-switch overrides.
- [ ] Confirm publish semantics (always push -u or optional).
- [ ] Confirm branch list format and UI copy.
- [ ] Map audit logging expectations for branch operations.

### Phase 1: API Foundation
- [ ] Add branch models (`branch_model.go`) with validation.
- [ ] Extend `GitRunner` with branch methods.
- [ ] Implement `ExecGitRunner` branch commands.
- [ ] Extend `FakeGitRunner` to simulate branches + errors.
- [ ] Build branch parser and tests.

### Phase 2: Service Layer
- [ ] `branch_service.go` enforcing safety rules.
- [ ] Integrate dirty checks via `GetRepoStatus`.
- [ ] Implement publish flow using existing push logic.
- [ ] Add audit logging for create/switch/publish.

### Phase 3: API Routes
- [ ] Add branch handlers and route wiring in `api/main.go`.
- [ ] Return structured errors for safety violations.
- [ ] Add API tests for new endpoints.

### Phase 4: CLI
- [ ] Add `branch list`, `branch create`, `branch switch`, `branch publish`.
- [ ] Ensure CLI outputs structured, human-safe messages.
- [ ] Add CLI tests if present.

### Phase 5: UI
- [ ] Build `BranchSelector` component.
- [ ] Add dropdown search + grouped list.
- [ ] Add create branch modal.
- [ ] Add publish affordance with safety warnings.
- [ ] Mobile header parity.
- [ ] Add UI tests and data-testid hooks.

### Phase 6: Scenario Validation
- [ ] BAS cases for branch endpoints.
- [ ] Update `requirements/09-branch-operations`.
- [ ] Update `docs/PROGRESS.md` with completed milestones.

## Notes / Risks
- Detached HEAD states require special handling in list/switch UX.
- Large repos may need pagination for branch list; keep response bounded.
- Remote-only branches must be tracked explicitly before checkout.

## Decision Log (Clarifications Applied)
- Dirty switch: block by default, allow override with explicit confirmation.
- Publish semantics: use last-used remote if available, fallback to origin.
- Branch list scope: local + remote branches (no tags).
- Remote-only switch: prompt to track and switch.
- Detached HEAD: prompt to create a branch, then switch.
- Ahead/behind: use sync status for current branch only (initial).
- Validation: `git check-ref-format` for branch names.
- Audit logging: create/switch/publish.
- UI placement: branch dropdown in header.
- CLI parity: list/create/switch/publish.
