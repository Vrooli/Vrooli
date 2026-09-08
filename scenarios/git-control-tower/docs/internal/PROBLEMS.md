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

## Identity and desktop entitlement work ladder (2026-09-07)

- Rung: W3
- Evidence: the identity/authentication plan requires a documented
  `personal_local` default, explicit four-mode desktop support, resource-specific
  audiences, and browser credentials that are not readable by JavaScript. The
  current GCT manifest declares `shared_provider` as its default and only
  advertises two modes; several relying parties still fall back to
  `scenario-authenticator:default`; and the UI writes the privileged access
  token with `document.cookie`.
- Blocker: implementation contract drift in scenario-owned authentication
  wiring. This is repairable in the worktree; it is not an authority blocker.
- Measured: 2026-09-07

## Work ladder — outgoing history safety (2026-09-07)

- Rung: W3, scoped implementation under OT-P0-006, OT-P0-011, and OT-P0-014.
- Evidence: the reported GitHub rejection names a 440820652-byte Vault executable introduced in the oldest of four outgoing commits. The push path had no historical blob-size preflight. User authorized detection, isolated preparation, and safe seam-based validation before touching this real history.
- Repair: typed outgoing-history inspection and isolated recovery preparation, exact preview intent binding, durable artifact status, and UI explanations. Application to the active checkout and automated LFS migration remain explicitly unavailable; they require a separate coordinated workspace-checkpoint contract.
- Measured: focused adapter tests restore an original bundle independently, repair four outgoing commits (including a later deletion and empty checkpoint), and preserve source refs, index bytes, staged/unstaged changes, and untracked files. Fault injection covers preparation commands and storage failures. Full W0–W2 readiness was not asserted.
- Tooling finding: `vrooli package refresh proto git-control-tower --no-restart` failed parsing template Go replace placeholders. Reported as `knw-1788817702005618488`; generated UI dependencies were refreshed through the scoped SDA install gateway.

### Validation outcome for outgoing-history safety

- Focused checks: 26 UI tests passed; TypeScript and scoped ESLint passed. API, pushsafety, policy gate, transport, and complete CLI package tests passed. Race-enabled safety tests and the API-core strict file-routing tests passed.
- Test Genie `20260907-215822-eb6c07ef`: structure and proto passed; unit failed. The new CLI command inventory expectation was corrected. The subsequent unit run `20260907-220346-80421731` still failed on existing UI test-policy drift and UI execution failures.
- Full UI execution: 430 passed, 9 failed across FileList (8) and DiffViewer (1). These files were not edited by this work; observed causes require separate investigation. Scenario QA entries: `knw-1788818622060875232` (UI failures), `knw-1788818622384056243` (policy drift).
- Production service was not restarted. No live branch rewrite, index change, checkout, stash, cleanup, or push was performed. Shared-worktree evidence does not certify an immutable full-scenario revision.

### Recovery hardening follow-up (2026-09-07)

- Rung: W3, scoped repair of reviewed reattachment and verification gaps.
- Changes: repository-bound latest-operation discovery independent of HEAD;
  exact operation-ID lookup; SHA-256 evidence and rechecks; independent candidate
  bundle restore and reference verification; explicit damaged/unverified/stale
  status; read-only current-remote reinspection; UI consent and status boundaries.
- Safety: production application, rollback and publication were not added or
  invoked. Actual four-commit HEAD remained `baaa1c5dd9e231c963b65d728c02b017c7ac4d69`.
  Preparation tests use temporary repositories and leased temporary artifacts.
- Measured: focused recovery/domain/HTTP tests with race detection; complete CLI
  package tests; 18 UI/API tests; TypeScript and scoped ESLint. Real-browser
  capture `b46a1da9-b9d1-4c47-8cd3-0387c0f1aa39` completed consent, preparation,
  reopening and stale-operation discovery through the production dialog.
  The outer fixture verified refs, index bytes, staged/unstaged changes,
  ignored/untracked content, local deletion and symlink preservation afterward.
- Browser limits: authentication and remote inspection are controlled seams;
  actual GitHub authentication and production identity-provider wiring are not
  certified. Earlier browser fixture build/routing errors were corrected before
  the passing capture. An adhoc BAS wait returned no receipt within 120 seconds;
  the bounded page-capture path supplied the final evidence instead.
- Scenario validation: run `20260907-223410-9c9111b0` passed structure and failed
  unit on `UNIT_POLICY_PROJECTION_DRIFT` and UI `TEST_EXECUTION_FAILURE`.
  Full W0-W2 or full-scenario readiness is not asserted.
- External tooling: BAS schema filter binding mismatch reported as
  `knw-1788820628241203086`. No unrelated scenario changes were made.

- Final validation update: unit run `20260907-224249-a2cf28a9` remains failed.
  It reports the prior UI policy/execution issues, a missing UI role observation,
  and `TEST_DEPENDENCY_MISSING`. The final whole-app TypeScript check reports
  `TS2307` in `ui/src/main.tsx:1` for `@vrooli/iframe-bridge/react`; that file
  changed concurrently at 18:37:58 EDT and was not edited by this work.
  This supersedes the earlier whole-app typecheck pass. Reported as
  `knw-1788821073948240062`. Final 18 focused UI/API tests and race-enabled
  recovery/HTTP tests pass.
- Runtime boundary: this task did not restart production. The live service was
  restarted by another workflow during the task (start record 22:17:43 UTC).
  Final read-only discovery still returned `invalid recovery identity`, so the
  final hardening is not certified as deployed. No real recovery was prepared.

## Work ladder — proactive push-safety UX (2026-09-07)

- Rung: W3, scoped implementation of the user-approved indicators. Existing
  staging/commit and history components had no file-size warnings; Push opened
  the only inspection surface. No W0–W2 readiness claim is made.
- Change: shared read-only reports across desktop/mobile; staged-object checks;
  row and commit warnings; history introduction/inheritance badges; accessible
  recovery review from blocked push controls; explicit unknown/stale snapshots.
- Safety: source history/index and recovery artifacts are not changed by these
  indicators. Local commits retain existing human authorization. Recovery
  preparation is still separate, and application/rollback remain unavailable.
- Tooling limitation: normal scoped protogen failed on an unrelated missing
  template-validation-react-vite schema. Direct scoped buf generation succeeded.
  QA record: knw-1788822252046014472. No unrelated schema was repaired.
- Validation: 51 focused UI tests passed, whole-app TypeScript and scoped ESLint
  passed, and focused backend/domain/recovery tests passed with the race detector.
  A real temporary unborn index retained its exact bytes while the checker used
  a small staged pointer instead of the oversized working copy.
- Test Genie `20260907-230417-99ce55f5`: structure passed; unit remains failed
  with test-policy drift and UI test-execution failures. This does not establish
  full scenario readiness. The new indicator suite uses the canonical renderer;
  broader existing policy and execution failures remain separately recorded.
- Deployment: no live service restart, publication, or real recovery operation
  was performed. These indicators require the updated API and UI together.

## Work ladder — history recovery access and compact tabs (2026-09-07)

- Rung: W3. History mode replaces the normal sync navbar and lacked recovery
  access. Commit Files lacked outgoing-violation attribution.
- Change: desktop and compact history headers open the existing read-only
  recovery dialog; Commit Files displays path/size warnings and identifies
  blockers outside the selected commit's changed-file list. Report paths and
  commit IDs are separate sets, so badges deliberately describe outgoing-history
  association rather than inventing exact path/blob identity in a selected tree.
- Compact styling diagnosis: Tabs@1.3.0 registers baseStyles and tab rules with
  the same stylesheet key; StyleSheet retains the first and drops the latter.
  A focused FileList test reproduces the collision warning. Live browser capture
  e38fd341-90ac-4430-aec4-6c782d68d6d1 reproduces unstyled, unspaced labels.
- Repair: consumer-scoped compact CSS restores spacing, sizing, selection color,
  horizontal overflow and focus. Compact/underline options remain unchanged.
  Remove the bridge after a governed Tabs release repairs stylesheet registration.
- Library boundary: `components draft-begin Tabs` refused a released 1.2.2 hash
  mismatch. No library release or integrity records were edited. QA record
  knw-1788823052680370776 retains the collision and draft failure.
- Validation: 20 focused UI tests pass, including desktop/compact recovery
  actions, path attribution, later deletions, missing reports, and compact tab
  keyboard selection. TypeScript and scoped ESLint pass. Requested Test Genie
  structure run was not admitted (caller preview capacity saturated); no run ID
  was issued. Prior scoped structure evidence remains historical, not a new pass.
- Deployment: no live service restart, Git mutation, or recovery preparation.
- Browser verification: isolated production-source fixture capture
  686a6a2f-a7e0-49d9-b970-5ad9acaaef78 reached its assertion selector after
  checking nonzero compact padding, minimum height, and distinct selected text
  color. Screenshot confirms restored compact spacing and selection. The live
  screenshot above is a before-state capture, not evidence of deployment.
- Upstream resolution: Tabs 1.3.1 is now published and installed through SDA.
  The temporary compact CSS bridge has been removed. Published-package browser
  capture aeecc4c3-f17c-4aeb-90b6-2d78542db791 and the consumer compact-tabs
  regression pass without it. The library migration script now preserves
  released sources and refuses to merge multiple injection identities blindly.
  Retained release-source recovery evidence is recorded in the RCL problems log.

### Recovery dialog presentation — 2026-09-07

Scoped W3 repair: user reported the recovery dialog lacked the visual hierarchy of commit authorization. Matched its shell, typography, snapshot cards, disclosure styling, approval card and action footer; preserved recovery consent and backend behavior. Twenty focused UI tests, TypeScript, ESLint and production build passed. BAS observer execution `333ed873-e0d1-4bf9-a72a-cd00ba5213a6` confirmed the live blocked dialog, with screenshot inspected. The requested Test Genie unit admission was rejected as resource_exhausted (shared caller preview capacity); no suite verdict is claimed. UI assets were updated while retaining previous assets; no recovery preparation or Git mutation was performed.

## Work ladder — quiet file-size checks (2026-09-08)

- Rung: W3, localized UI behavior. User requested silent successful checks and fewer repeated file-size messages after recovery.
- Evidence: PushSafetyNotice and StagedSafetyNotice rendered for every report, history rows announced passing and inherited checks, and both file panels repeated summaries.
- Change: successful/loading/stale checks are silent; sync or the replacement history navbar owns the outgoing warning/error summary. Staging keeps a concise commit warning and affected-row badges. History marks introducing commits and affected paths; duplicate file-panel summaries and commit-panel push notices are removed. Full details remain in the review dialog. Backend push checks and recovery behavior are unchanged.
- Validation: 28 focused indicator/dialog/sync tests passed, including clean desktop/mobile history, warning-only files, stale and failed reads, and retained review actions. Whole-app TypeScript, scoped ESLint, production build, and scoped diff whitespace check passed.
- Scoped Test Genie unit run 20260908-070753-8263d554 failed with UNIT_POLICY_PROJECTION_DRIFT and TEST_EXECUTION_FAILURE plus architecture findings. Similar policy/execution limitations are recorded above; this run does not certify the full scenario and failure attribution was not established. Existing QA records knw-1788818622060875232 and knw-1788818622384056243 track the earlier UI execution/policy findings.
- Delivery: production UI assets rebuilt. No service restart, history rewrite, index operation, or push was performed.

## Work ladder — mutation latency (2026-09-08)

- Rung: W3, scoped performance and async UX repair. User reported slow staging, committing, and pushing.
- Confirmed: authorization repeatedly requested full UI status enrichment. Read-only benchmark on the live checkout measured 3.191 seconds for full status versus 0.379 seconds for the lean fresh snapshot. This component timing does not measure complete actions.
- Repair: fresh lightweight authorization/target reads; per-repository serialized writes with concurrent optimistic projections and isolated rollback; one status reconciliation after queue drain; all pending paths tracked; early commit feedback and prompt history refresh; redundant pre-push fetch removed; safety scans paused during writes.
- Evidence: 77 focused UI tests, targeted race-enabled backend tests, TypeScript and scoped lint passed. Test Genie unit run 20260908-072311-87086d30 failed broader UI policy/execution and architecture checks. No full-scenario certification claimed.
- Tooling: shared Go cache references failed to resolve during initial compilation. Isolated GOCACHE allowed tests to pass; QA report knw-1788852252831737666 records the observation without assigning a cause.
- Safety: mutation tests use fakes or temporary repositories. The live benchmark only reads repository state. No stage, commit, recovery, or push was executed against the user's checkout.
- Delivery: lifecycle restart startop-482edfbe966d1ecb6c764f45cb688e0f completed healthy at 2026-09-08T07:27:15Z after setup/build. Updated API and UI are running.
