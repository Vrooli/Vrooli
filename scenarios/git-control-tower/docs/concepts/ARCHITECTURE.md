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

## Embedded mobile shell

The mobile workspace is an iframe-safe percentage-height shell. `html`, `body`,
and `#root` form the height chain; `App` owns three structural rows: mobile
header, one min-height-zero active-panel region, and mobile navigation. The
navigation participates in that flex layout rather than overlaying the panel.
Each panel owns its own scrolling. In particular, Changes stores and restores
only `changes-scroll-region` after layout settles; its selection controls are
part of that scrolling content, not sticky chrome. This prevents tab changes
from moving the embedding document and keeps safe-area ownership explicit.

## Descriptor-aware review evidence

The scenario review UI has exactly eight workflow tabs: Overview, Baselines,
Metrics, Screenshots, Workflows, Tests, AI Changes, and Agent. Rules and Code
Quality are not routable or persisted tabs; their underlying providers remain
available through Test Genie phase findings and other reusable review checks.

Every retained evidence view crosses one programmatic boundary:
`EvidenceService`. GCT forwards Test Genie's canonical `RunInfo`, frozen
descriptor snapshot, phase results, applicability/maturity summaries, and open
`ArtifactRef` records without translating phase keys. Tests filters descriptor
metadata, while Screenshots and Workflows select artifact kinds regardless of
producer phase. Agent attachments retain the exact run/phase/artifact identity.
Artifact bodies use run-scoped opaque IDs through the same-origin generic byte
route; filesystem paths never enter the UI contract.

Tests and Baselines are outcome-first projections over that boundary. Tests
groups failed, skipped, unavailable, and finding-bearing phases ahead of clean
phases; filters use only captured descriptor fields, historical runs are
selectable, and expansion lazily requests the run's typed evidence catalog.
Clean catalogs render incrementally in bounded batches. Baseline comparisons
show exact base/current run and Git identity, dirty/stale state, catalog
additions and retirements, aggregate outcome counts, and Test Genie comparison
reasons. Phase keys and evidence kinds remain open data throughout; the UI has
no Test Genie phase registry or surface-to-phase mapping.

Artifact presentation is a separate, evidence-kind capability. A small renderer
registry recognizes images and visual diffs, recordings, text/log output,
JSON/findings reports, coverage, and trace/HAR/network/console evidence. It is
never keyed by a producer phase. Unknown kinds always use the generic renderer,
which exposes safe metadata, catalog or legacy provenance, relationships, and
the authorized opaque artifact action without guessing from a filename.
Screenshots groups captures and advisory comparisons by run and capture context;
Workflows groups recordings and related operational evidence by run. Both tabs
deduplicate stable artifact IDs, page metadata in bounded increments, and defer
all image/video byte requests until the operator opens Preview. Evidence cards
can attach the exact opaque identity to Agent or navigate to the exact captured
run and producer phase in Tests.

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

## Policy Gate

Mutating Connect-RPC methods on the GCT API are gated by an
agent-access policy: `allow | warn | confirm | deny`. The default is
`confirm` — agents must pass an explicit override flag (
`--i-was-explicitly-authorized` / `VROOLI_GCT_AUTHORIZED=true`) or the
call is refused with a strong rejection message redirecting the work
back to the human.

The gate is enforced in two layers:

- **CLI (point of intent):** `cli/internal/callerheader.New()` is
  installed as a Connect client interceptor; it stamps every outbound
  request with `X-Vrooli-Caller: <kind>` (from `cliutil.DetectCallerKind()`)
  and `X-Vrooli-Authorized: true` when the agent-override env var is
  set.
- **API (defense in depth):** `api/internal/policygate.NewInterceptor()`
  is installed as a Connect server interceptor on the worktree + repo
  handlers. It reads the headers, falls back to its own
  `DetectCallerKind()` if the header is missing, applies the
  `policygate.Decide()` matrix, and emits a structured audit-log line
  for every gate decision.

Read-only Connect methods bypass the gate. Per-command policy
granularity is intentionally deferred — see PROBLEMS.md.

Operator surface: `scenarios/git-control-tower/.vrooli/config.json`
`policy` block. See `api/internal/config/config.go` for the schema,
`docs/internal/SEAMS.md` for the loader/interceptor seams, and
`packages/cli-core/docs/reference/agent-detection-signals.md` for the
detection-signal catalog the gate relies on.

## Cross-references

- API endpoint catalog: [docs/reference/api-endpoints.md](../reference/api-endpoints.md)
- CLI command reference: [docs/reference/cli-commands.md](../reference/cli-commands.md)
- Configuration: [docs/reference/configuration.md](../reference/configuration.md)
- Integration boundaries: [docs/internal/SEAMS.md](../internal/SEAMS.md)
- Performance audits: [docs/perf/2026-05-03-after-fixes.md](../perf/2026-05-03-after-fixes.md)
