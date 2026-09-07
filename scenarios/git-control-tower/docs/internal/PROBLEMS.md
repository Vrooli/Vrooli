# Problems: git-control-tower

Known issues, tech debt, and deferred work for the git-control-tower scenario.

## Open Issues

### Scenario-level

- The scenario has been re-scaffolded from the `react-vite` template; OT-P0-001
  has initial implementation + tests, but most operational targets remain
  unimplemented.
- Git operations are inherently risky; design must enforce repo-root path
  validation, explicit allowlists, and safe defaults for any mutating endpoint.
- Repository size and file counts can cause slow status/diff calls; plan
  pagination and caching before claiming "production-ready".
- Some `test-genie` phases (smoke/performance) require a running Browser
  Automation Studio (BAS) workflow engine; when BAS is unreachable, smoke
  routes through the runnability resolver and SKIPs rather than hard-failing.
- `vrooli scenario status git-control-tower` currently fails its optional
  `test-genie` structure probe due to an unsupported `-no-record` flag; the
  core scenario test lifecycle still uses `test-genie execute`.

### React stability internals

- ESLint React stability config added with safety-critical rules and import
  cycle detection.
- TypeScript safety rules enforced (strict + noUncheckedIndexedAccess) with
  protective comments.
- Guarded array indexing and optional values in App selection logic,
  FileList grouping, GitHistory scopes, file search helpers, mobile search
  selection, and bottom sheet touch handling.
- No remaining lint or type-check failures after `pnpm lint` and
  `pnpm type-check` (2026-02-04).

### Proto-first transport convergence (2026-09-06)

The scenario is greenfield and must converge on Connect-RPC for all typed UI,
CLI, and inter-scenario operations. The current inventory is 38 hand-wired
REST registrations in `api/routes.go`, 9 mounted Connect services, 9
production UI `api-*.ts` modules with 31 direct REST calls, and no remaining
repo-domain REST callsites in the CLI. `repo status`, `repo stage`, `repo
groups`, `repo diff`, `repo sync-status`, `repo unstage`, `repo commit`, and
all four `branch` commands now use generated Connect clients. The UI repository
file tree, directory listing, related-file lookup, content search, credentials,
remote URL, SSH keys, grouping rules, gitignore health/remediation, tracked-binary
health/remediation, file save, and path delete now use the typed `RepoService` as
well. Discard, ignore,
push, pull, upstream actions, and precommit configuration/execution now use the
same typed service and exact intent path where they write repository state.

REST is retained during convergence for read-only UI/CLI surfaces and documented
transport exceptions (`ops_probe`, `multipart_upload`, `webhook_receiver`, or
`third_party_shape`). Independent legacy writers must not receive incremental
authorization patches; they must be migrated to Connect-RPC/generated clients or
removed, with transport-independent domain logic beneath the boundary. Evidence: plan execution
decision `3db32287-012a-4a9f-a1b8-974b6aec97ca` and note
`73c794a8-8aff-4612-aa54-0d6278b4b4ec`.

The latest slices also migrated repository history, approved-change previews,
provenance, provenance search, and evidence-bound advisory drafts to generated
Connect methods (`RepoService` methods
`GetRepoHistory`, `GetApprovedChanges`, `GetProvenance`, and
`SearchProvenance`, plus `AdvisoryService.Draft`,
`AuditorService.StartCheck`, `GetJobStatus`, `ListRules`, `ListViolations`,
`PreviewFix`, `ApplyFix`, and `ReviewService.Start`);
their REST registrations
and UI fetches are retired. The remaining 38 REST registrations are not
completion evidence: each typed
operation still needs a domain-aligned Connect contract, or an explicit
`multipart_upload`, `webhook_receiver`, `third_party_shape`, or `ops_probe`
exception.

## Engagement overwrite (resolved 2026-09-02)

`baseline start` wrote the engagement manifest unconditionally, so a second
session starting an engagement for the same scenario and slug silently took
over the first session's restore point. The control plane's
`recovery.WriteEngagement` now refuses to overwrite a live (unexpired)
engagement and names the holder (scenario, slug, mode, created and touched
times); `baseline start --replace` and `vrooli recovery write --replace` are
the explicit override. Guard: `TestEngagementStartRefusesLiveEngagement`
(`internal/app/recovery`).

## Worktree Domain Follow-ups (2026-05-16)

The worktree domain shipped Tiers 1 + 2 against the WorktreeService /
RepoService Connect-RPC surface. Deferred from this work and tracked
here:

- **UI worktree feature surface.** The `WorktreeSidebar`, `WorktreeRow`,
  `CreateWorktreeModal` components and the status-header worktree badge
  are not yet implemented. CLI is the agent-facing surface for now. When
  the UI lands, it MUST consume the generated Connect-Web client at
  `packages/proto/gen/typescript/git-control-tower/v1/worktree/` — no
  hand-written interfaces.
- **Branch UI tooltip for claimed worktrees.** `BranchSelector.tsx`
  should disable the switch action and show a tooltip when
  `checked_out_in_worktree` is non-empty. The typed BranchService shape already
  carries the field.
- **Audit events for worktree mutations.** `WorktreeCreated`,
  `WorktreeRemoved`, `WorktreeLocked`, `WorktreeUnlocked`,
  `WorktreeMoved`, `WorktreePruned` event types are not yet wired
  through `audit_logger.go`. The Connect handlers should call the
  `Audit` seam on success (and on dry-run with `dry_run=true`).
- **gen-endpoints / EndpointDescriptor parity for new domain.** The
  worktree and repo Connect services are mounted directly in
  `connect_wiring.go` without `Module.Endpoints` / `EndpointDescriptor`
  metadata. Once a second domain adopts the template's module pattern,
  port both domains to `module.Module` + `validateTransport`.
- **Remaining incremental-migration candidates.** Per
  `feedback_incremental_template_migration`, retrofit one domain at a
  time. Branch is now on BranchService; the remaining REST-backed
  repository writers, review, and audit surfaces still need typed contracts
  or an explicit RESTException record.
- **Tier 3 worktree features (research candidate).** Agent-session ↔
  worktree mapping; worktree-aware commit composer; cross-worktree
  conflict pre-flight; worktree health overview (stale detection);
  "create worktree from this branch" one-click flow aligned with
  Claude Code's `isolation: "worktree"` integration.

## Agent-Access Policy Gate — Deferred Work

The agent-access policy gate landed in 2026-05-21 (see ARCHITECTURE.md
"Policy Gate"). The following items were scoped out of the initial
landing and are tracked here:

- **Per-command policy granularity.** Today the policy is coarse: one
  `agentAccess` value applies to every mutating Connect method.
  Per-procedure overrides (e.g. allow `LockWorktree` for agents but
  deny `RemoveWorktree`) would let operators tune the gate without
  flipping the global lever. Defer until at least one operator asks.

- **REST-backed domain coverage.** The Connect-RPC interceptor now covers
  WorktreeService's context-intent mutations and the handler-managed
  RepoService commit plus BranchService mutations. The remaining REST-backed
  repository writers, review, and audit writes still carry their own paths;
  migrate each to Connect-RPC or document it as a RESTException before
  calling the scenario greenfield-complete.

- **Global `--i-was-explicitly-authorized` CLI flag.** Today the
  override is wired via the `VROOLI_GCT_AUTHORIZED` env var that the
  callerheader interceptor reads. A first-class global flag would
  produce friendlier ergonomics (`git-control-tower repo commit
  --i-was-explicitly-authorized`); plumb it through `cliapp`'s flag
  layer in a follow-up.

- **PROBLEMS.md entry for unverified opencode signal.** Per
  `packages/cli-core/docs/reference/agent-detection-signals.md` the
  opencode `OPENCODE_PID` signal is confirmed via static analysis but
  not yet observationally confirmed inside a real opencode-launched
  shell. Re-verify once agent-manager grows an opencode runner that
  actually spawns opencode (today the runner shells to claude
  internally).

- **Detection for additional runtimes.** Cursor (background agent),
  Antigravity CLI — same verification methodology as codex/opencode/
  grok; deferred until a confirmed self-set tool-shell signal exists for
  each. (Grok CLI graduated 2026-06-28 via its `GROK_AGENT=1` sentinel;
  Antigravity CLI — Google's `agy`, replacing Gemini CLI — graduated
  2026-06-29 via `ANTIGRAVITY_AGENT=1`, binary-confirmed with live `/proc`
  confirmation pending. Cursor remains deferred until a harness exists.)
  Until then, the gate falls through to `CallerKindUnknown` (treated as
  human) for those sessions, recoverable via `VROOLI_CALLER=agent`.

## Deferred Ideas

- Multi-repo support (path switching, isolation, multi-tenant auth).
- Real-time UI updates (fsnotify + WebSocket channel) once baseline endpoints
  exist.
- **Bulk backfill of `// DOC:` cross-references** across the API/CLI codebase.
  The 2026-05-03 docs audit reported 230 exported symbols missing DOC anchors.
  Out of scope for the current docs-health remediation; high-signal anchors
  added at three sites (api/server.go, cli/domains/domains.go, ui/src/App.tsx)
  as exemplars. Decide later whether to invest in full backfill vs. accept
  partial coverage as the long-term equilibrium.
- **FileList row virtualization** — the 2026-05-03 perf audit added a
  `<Profiler id="FileList">` boundary; if comparison-run data shows
  per-commit cost exceeds ~5 ms or row counts grow into the hundreds,
  virtualize via `@tanstack/react-virtual` (already a dep).

## Work ladder

- Rung: W3
- Evidence: the canonical PRD wizard now covers the active maturity capabilities
  while preserving every legacy OT identifier; `business-health validate scenario
  git-control-tower` and `vrooli scenario requirements validate git-control-tower`
  both pass at L3 for contract, registry, linkage, and evidence traceability.
  The W3 scenario-owned run `20260906-182152-dfca5497` is admitted and remains
  terminated with `failed` after reaching architecture; the receipt identified
  an unwired routed database/file-root seam plus inherited health/fixture findings.
- Blocker: no contract or obligation defect remains; a replacement W3 receipt is
  required after the routed isolation repair and targeted regression validation.
- Measured: 2026-09-06
