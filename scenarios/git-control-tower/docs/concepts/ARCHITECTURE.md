# Architecture: git-control-tower

System overview, component boundaries, and the planned operational
targets that drive future work.

## Purpose

git-control-tower is a **single-repo control plane** for git: agents and
humans get a stable, audited HTTP/CLI/UI surface for status, diff,
history, staging, committing, branching, and review operations against
the local Vrooli working tree. The scenario is intentionally not a git
server — it wraps `git` as a subprocess and exposes safe, structured
endpoints that don't require parsing raw CLI output.

## Components

```
┌─────────────────────────────────────────────────────────────────┐
│                           USER / AGENT                          │
│                                                                 │
│            ┌──────────────┬──────────────┐                      │
│            ▼              ▼              ▼                      │
│         ┌─────┐       ┌─────┐       ┌─────┐                     │
│         │ CLI │       │ UI  │       │HTTP │                     │
│         └──┬──┘       └──┬──┘       └──┬──┘                     │
│            └─────────────┼─────────────┘                        │
│                          ▼                                      │
│                   ┌──────────────┐                              │
│                   │   API (Go)   │  ← all business logic        │
│                   └──┬───────────┘                              │
│                      │                                          │
│              ┌───────┴───────┐                                  │
│              ▼               ▼                                  │
│          ┌───────┐       ┌────────┐                             │
│          │ git   │       │ SQLite │  ← audit log                │
│          │subproc│       │  + FS  │                             │
│          └───────┘       └────────┘                             │
└─────────────────────────────────────────────────────────────────┘
```

| Component | Path | Purpose |
| --- | --- | --- |
| API | [CODE: api/routes.go]                                   | Routes (92 endpoints), business logic, git-subprocess wrappers |
| CLI | [CODE: cli/domains/domains.go]                          | Thin command wrappers; all business logic lives in API |
| UI  | [CODE: ui/src/App.tsx]                                  | 3-pane React app: file-list / diff / history |
| Audit log | [CODE: api/audit_logger.go]                       | SQLite-persisted log of all mutating operations |
| Auditor | [CODE: api/auditor_client.go]                       | Quality / scenario-review pipelines |

## Operational targets

The scenario implements the P0 targets and surfaces P1/P2 work as
planned operational targets. Each target is referenced inline in the
relevant component's documentation.

### Implemented (P0)

See [PRD.md](../../PRD.md) for the full list of P0 targets (OT-P0-001
through OT-P0-008). All P0 items have at least an initial implementation
+ tests.

### Planned post-launch

- [REQ: OT-P1-002] **AI-assisted commit messages** — generate suggestions
  via Ollama / OpenRouter with rule-based fallback. Surface: API
  endpoint + UI affordance in the commit panel.
- [REQ: OT-P1-003] **Conflict detection & reporting** — detect active
  merge conflicts plus potential conflicts (local changes + behind
  remote). Surface: status endpoint extension + UI status header.
- [REQ: OT-P1-004] **Change preview** — preview commit impact with LOC
  analysis, affected scenarios/resources, deployment risk assessment.
  Surface: dedicated preview endpoint + UI panel.

### Future / expansion

- [REQ: OT-P2-002] **Worktree management** — Tiers 1 + 2 SHIPPED as the
  first proto+Connect-RPC domain in GCT (2026-05-16). See
  [Worktree Connect surface](#worktree-connect-surface) below.
- [REQ: OT-P2-003] **Stash operations** — save, apply, drop stashes
  programmatically.

See [docs/internal/PROBLEMS.md](../internal/PROBLEMS.md#deferred-ideas)
for additional deferred ideas (multi-repo support, real-time WebSocket
updates).

## Worktree Connect surface

The worktree domain is the first proto+Connect-RPC domain in GCT. It
ships alongside (not replacing) the existing flat-package REST API and
acts as the seed pattern for future incremental migration of other
domains.

| Domain | Service | Methods | Transport | Maturity |
|---|---|---|---|---|
| `worktree` | `WorktreeService` | `ListWorktrees`, `GetWorktree`, `CreateWorktree`, `RemoveWorktree`, `LockWorktree`, `UnlockWorktree`, `MoveWorktree`, `PruneWorktrees` | Connect-RPC | L3 (proto+Connect end to end with dry-run support) |
| `repo` | `RepoService` | `GetRepoStatus` (Tier-1 worktree identity fields only) | Connect-RPC | L3 (greenfield Connect; legacy REST `/api/v1/repo/status` continues to serve other consumers) |

Source paths:

- proto: `packages/proto/schemas/git-control-tower/v1/{worktree,repo}/`
- API: `scenarios/git-control-tower/api/handlers/{worktree,repo}/`, `scenarios/git-control-tower/api/internal/{worktree,repo}/`
- CLI: `scenarios/git-control-tower/cli/domains/worktree/`
- Mount: `scenarios/git-control-tower/api/connect_wiring.go::mountConnectHandlers`

Branch cross-link: REST `GET /api/v1/repo/branches` now includes a
`checked_out_in_worktree` field per local branch. Field is populated via
the `claimedBranchesFn` test seam in `api/branch_handler.go`, which
defaults to `worktree.Inspector.ClaimedBranches` in production and is
override-able in tests so the existing branch flow never hits real git.

## Cross-references

- API endpoint catalog: [docs/reference/api-endpoints.md](../reference/api-endpoints.md)
- CLI command reference: [docs/reference/cli-commands.md](../reference/cli-commands.md)
- Configuration: [docs/reference/configuration.md](../reference/configuration.md)
- Integration boundaries: [docs/internal/SEAMS.md](../internal/SEAMS.md)
- Performance audits: [docs/perf/2026-05-03-after-fixes.md](../perf/2026-05-03-after-fixes.md)
