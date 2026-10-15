# Problems: git-control-tower

Known issues, tech debt, and deferred work for the git-control-tower scenario.

## Active delivery — proposal review (2026-10-08)

Current status: delivered and feature acceptance verified. Final API/UI/CLI checks,
proto owner verification, lifecycle rebuild, and desktop/mobile observer evidence
are recorded in the final checkpoint below. Wider scenario certification remains
failed on separately reported policy/catalog/experience debt. No real proposal or
operator Git history/index was mutated; no Plan Manager was used.

The foundation list below is historical; latest receipts and corrections appear
in the appended delivery checkpoints.

- Authority: operator approved the findings/recommendations, full implementation,
  cleanup and validation; explicitly prohibited Plan Manager.
- Shape: bounded delivery; canonical context/decisions/sequence/acceptance and
  harness handoff live in `../../DESIGN.md#proposal-review-delivery-specification--approved-2026-10-08`.
- Rung: W3 scoped implementation/UX under existing diff-first/human-authority
  targets; baseline evidence is source/test inspection and desktop/mobile BAS
  executions b526cb81-7e34-4422-ab64-3a0fb2515627 and
  3fc1bd3b-fd44-4ffe-a21b-505a4ccca19b. No approval or withdrawal was exercised.
- Completed foundation: immutable revision/content tables and transactional
  retention; bounded canonical capture (8 MiB/file, 64 MiB/package) into scenario
  storage using temporary Git objects; captured diff RPC reusing diff parsing;
  explicit Combine RPC with exact revisions and atomic source closure; Close
  reason/replacement/revision support; no automatic content folding; source
  manifests include source revisions/evidence; prior-work acknowledgment and
  branch/content/mode checks; candidate prepare/verify/publish seam and short
  SQLite publication lease; isolated candidate implementation. Product UI/CLI
  and history integration have NOT been updated yet. No real proposal mutated.
- Canonical schema extensions: GetProposalDiff, CombineProposals, captured
  standing, revision reads, source revision/evidence, close revision/replacement,
  prior-work acknowledgment. Generated only through proto owner. Vrooli-Run max
  raised to 400 for explicit source unions; overflow errors instead of truncation.
- Removed: dead automatic fold implementation. Replaced obsolete fold tests
  with explicit union/conflict/stale-source/retention/acknowledgment/edit-during-
  checks regressions. Existing submission now preserves older operator review.
- Validation so far: pre-change `make verify-committed-gen` passed; scoped owner
  generation published artifact e0297678ffff507050b615dcb696e5dc30db76f91046d4841dd70fef8656ad6f
  (closure common/git-control-tower/test-genie). All proposals/trailers domain
  tests passed (`go test ./internal/proposals ./internal/trailers -v`). Five
  focused API transport/digest tests passed with in-memory repository fixtures
  (`go test . -run 'Test(Proposal|ApplyProposal|CommitWriteTimeout)' -v`).
- Next safety cleanup: finish lock-time source-index validation in candidate
  publication; remove now-dead markCommitted retry path; truthful restore error
  handling; environment/check portability and canonical schema ownership;
  production candidate/capture/diff regressions (safe seams, never real Git writes).
- Next product work: UI API acknowledgment context `;prior-work=acknowledged`,
  captured-diff mapping/hooks, feature-owned full workspace/inbox with URL state,
  close/remove/refresh comparisons/evidence, explicit combine/replacement,
  mobile composition/mockups; CLI parity; full-message history/trailer Continue;
  owner resolution integration; lifecycle rebuild/observer QA and scoped Test Genie.
- Key files: api/internal/proposals/{model,store,service,review,apply,digest,memory_git}.go;
  api/proposal_{capture,candidate,review_connect,connect,git}.go;
  packages/proto/schemas/git-control-tower/v1/proposals/proposals.proto. UI still
  uses old ProposalsSection/ProposalDetail and must be replaced/extracted, not
  left as a competing review surface. App has desktop/mobile mount duplication.
- Validation remaining: proto reverify after final schemas; UI/CLI regressions;
  scoped Test Genie structure/unit/lint/integration/appropriate UI gate; desktop
  and mobile live BAS observer assertions/captures; final scoped review and work
  record. Never claim full acceptance from these partial passes.
- Checkpoints: append decisions and receipts here as slices land; this entry is
  the continuation source after compaction. No separate session audit document.

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

2026-10-07 (O15): that guard ran at the manifest write, after `start` had
already re-captured the restore point. A worker's `baseline start --name
bas-goal` put the working tree under live and then failed with "engagement is
live". `baseline start` now reads the floor before it captures anything. It
refuses an open engagement of the same name (expired ones too) and any second
engagement for the scenario. `--replace` takes over the same name and keeps the
restore point and anchor. Guards: `TestStartRefusesOpenEngagementBeforeCapture`,
`TestStartRefusesSecondEngagementUnderAnotherName` and
`TestStartReplaceTakesOverWithoutRecapturing`. A small window remains between
the check and the capture; `vrooli recovery capture` itself does not refuse an
open engagement.

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

## Work ladder — responsive mutation dialogs (2026-09-08)

- Rung: W3, localized UI behavior under the existing mutation-authorization and push-safety obligations.
- Evidence: the commit authorization surface was mounted only in the desktop App branch, while push recovery used a hand-rolled fixed overlay. On narrow screens, commit state could be pending without a rendered dialog; push overlay presentation was not owned by the shared portal-backed dialog primitive.
- Repair: extracted `CommitAuthorizationDialog`, mounted it in both desktop and mobile App branches, and migrated commit authorization plus push safety review to `react-component-library:ResponsiveDialog`.
- Validation: 27 focused UI tests passed; TypeScript, scoped ESLint, production build, and diff whitespace checks passed. Test Genie unit run `20260908-151507-53396297` remains failed on existing `UNIT_POLICY_PROJECTION_DRIFT` and `TEST_EXECUTION_FAILURE`; no full-scenario pass is claimed.
- Delivery: no Git mutation, recovery preparation, push, or production restart was performed.

## Work ladder — batched stage clicks (2026-09-08)

- Rung: W3, localized staging interaction and mutation-throughput repair under OT-P0-004 and OT-P0-014.
- Evidence: each row action called `stageFiles` independently. React Query serialized those calls per repository, but each queued click still repeated authorization and a Git index write. The existing endpoint already accepts multiple paths, so the UI was leaving a batching seam unused.
- Hypotheses: the dominant cause was per-click request creation; a status-refresh loop was less likely because status reads are disabled while workspace mutations are pending; row rendering was not the owning path because the action handler created one mutation per click.
- Repair: the first stage action remains immediate. Additional stage actions received while it is active are deduplicated and sent as one follow-up path batch. Queued paths keep per-row pending indicators, and the batch is scoped to the active repository.
- Validation: focused stage-batcher and core mutation tests pass (7 tests total), TypeScript passes, changed-file ESLint passes, and `git diff --check` passes. Test Genie unit run `20260908-155346-1ec82ebf` failed on existing `UNIT_POLICY_PROJECTION_DRIFT`, `TEST_EXECUTION_FAILURE`, and architecture findings; it does not certify or invalidate this focused repair.
- Delivery: no service restart, repository staging, commit, recovery preparation, or push was performed.

### Proposal delivery checkpoint — full workspace/history slice

- Full proposal workspace is mounted once in App with deep-linked proposal/file/
  revision state; desktop file/diff/inspector panes and mobile overview/files/diff.
  Inbox groups work and explicitly combines frozen selected revisions. Old inline
  detail mount/live working diff fallback removed. Close optional reason/replacement,
  exclusion, prior-work acknowledgment, retained revision reads and reported source
  evidence are present. Immutable diff reuses RepoService.GetRepoDiff and existing
  diff decoder/renderer; separate proposal diff RPC retired to reduce duplication.
- Added explicit captured-to-current comparison (read only) and confirmation before
  adopting current contents; comparison cannot be approved. Snapshot unavailable
  legacy revisions are honest and require explicit adoption. No real proposals changed.
- History now reads full Git messages and NUL-delimited paths, exposes body/trailers/
  continuation message, and selected history links back to proposals. Shared Continue
  parser retains effort/plan/backlog/work links, adds predecessor hash, excludes old
  proposal/run/epoch/author attestations. P/pN remains usable. Owner read adapters for
  effort/epoch/run/plan/backlog use typed Connect, truthful missing/private states and
  bounded response budgets. No Plan Manager used to manage this engagement.
- Domain schema moved from inline SQL to embedded proposals/schema.sql via api-core
  SchemaProvider/EnsureSchemas. Retired dead automatic fold domain/CLI behavior.
- Production temporary-repository regressions caught manual HEAD.lock conflicting
  with Git's own reflog lock. Publication now uses prepared Git ref transaction plus
  index lease and checks destination after transaction lock acquisition. Successful
  exact staged+unstaged publication, hook tree/message rewrite refusal, binary/
  deletion/mode/symlink retention and full-message/literal newline path history tests
  PASS. Domain/transport/CLI focused tests pass; proposal UI tests updated for new
  expected UX (one save-blocker ordering fix pending rerun).
- Governed dependency owner installed approved genproto/googleapis/api annotations
  dependency required by typed Agent Manager client. Scenario setup succeeded after
  correcting compile issues and refreshed copied UI proto dependency; UI type-check
  passed before final comparison schema. Final schema generation pending completion.
- Remaining: validate final proto/consumer refresh; finish provenance/continuation UI
  tests; workspace/mobile/inbox/conflict regression tests; request-scoped owner auth/
  resolver evidence tests; candidate lock/race/filter/unborn/base-move regressions;
  lint cleanup and bounded list/revision controls; actual BAS desktop/mobile captures
  against mockups; scoped Test Genie phases and owner waits; final docs/work journal.
- Existing whole-UI lint also flags unrelated SourceDistributionPanel typing,
  SettingsModal unused test argument, api-core dead SSE imports and AIProvenanceTab
  memo warning. Fix narrow debt where safe; preserve concurrent dirty work.

### Proposal delivery checkpoint — final correctness and validation

- Actual retained revision list replaces inferred legacy history; exact destination
  name/path displayed, immutable history ignores but preserves current message drafts.
  Async approval preparation cannot reuse a changed review/acknowledgment/draft.
  Diff load failure blocks approval. Closing replacement previews omitted paths.
- Added anchor effective mode, raw-byte drift verification even with canonical-equal
  filtered blobs, and index-only/no-contribution exclusions without touching staging.
  Named destination branches required; restore work has bounded original context.
  Verified proposal commits record precommit receipts; receipt errors remain visible.
- CLI retained diff and exact-revision explicit combination/close paths delivered;
  published-but-failed approval displays recovery commit identity and notes.
- Owner auth forwarding is read-only, response bounded to 800 ms, duplicate reads
  deduplicated. Regression coverage includes owner revision/epoch/missing/privacy states.
- Removed obsolete nested queue, duplicate diff RPC/decoder/SSE parser/union helper,
  inline schema SQL, automatic folding and old post-publication retry. Narrow existing
  UI lint issues and virtualized FileList test fixtures repaired, not relaxed.
- Owner artifact 6eda94b04237575c4c9d7403697e6bcec53bb957dd25d032eed92a9069498a6e;
  final make verify-committed-gen PASSED. Governed lifecycle setup refreshed consumer.
- Full API go test ./... PASSED. Full UI 60 files/493 tests PASSED; additional two
  final review guard/replacement regressions PASSED. UI type check and lint PASSED.
  Proposal CLI + worktree manifest contract tests PASSED; full CLI has three unrelated
  trusted-base/baseline fixture failures reported as knw-1791478030261644073.
- First scoped Test Genie run 20261008-162550-e81e998d: structure/api passed;
  unit/quality/workflow/experience failed. API restore note/UI virtualization failures
  fixed; stale chunk recovery fixed with manual reload preserving drafts. Remaining
  policy projection/architecture/playbook/experience binding debt requires final rerun
  and honest report. No real proposals closed/refreshed/approved, and no operator
  repository staging/commit/push performed. No Plan Manager engagement operations.

### Proposal delivery checkpoint — observer proof and publication recovery

- Final live BAS desktop journey 7a89cd95-328f-4c75-bd75-5e168d0c76b7 and mobile
  4dd6f9f1-67d1-4788-aecb-8691da14d32a completed: deep link, files/diff/mobile
  transitions, close form with explicit code/index preservation, and inbox return.
  Neither confirms closure or applies a commit. Legacy capture-needed view inspected.
- Browser-local captured review fixtures 89878c3f-770d-46aa-a7ce-07883a4e3bac desktop
  and c5f9ee0d-fef7-4446-aba1-166a9043d3e4 mobile completed and captures inspected.
  Fixture heading identifies simulated data; mutation RPCs are blocked. These prove
  rendering/layout only, not production API publication or a human approval.
- Maintained proposal-review/inbox-observer uses owner-registered selectors and V2
  actions. Workflow owner requires snake-case execution_mode=observer; its classifier
  treats all clicks as mutating, so maintained observer uses assertions/navigation
  only. Live back/close-tab journeys remain separate bounded observer receipts.
- Production adapter check regression verifies portable configured precommit against
  private candidate, mapped environment root, exact proposed file and base contents
  for unrelated unfinished files; passing timestamp/stdout retained. PASSED.
- Lost Git ref acknowledgment regression forces publication then suppresses commit
  acknowledgment. Publisher now resolves the destination under a bounded detached
  recovery context before deciding whether index restore is safe. Verified publication
  is recorded; uncertain publication names the candidate and requires recovery review.
  Regression PASSED with temporary Git/SQLite only.
- Final wider run 20261008-172905-8e73d8d5: structure/api PASSED, unit/quality/
  workflow/experience FAILED. Full API/UI execution passed; CLI trusted-base tests,
  policy projection/companion/coverage debt remain. Quality includes existing branch/
  repo interactive boundary, missing Makefile targets and global TS assertion debt.
  Six API playbooks use retired BAS schema; workspace experience bindings/claims and
  mobile chrome measurement fail. One new observer metadata/classifier issue found
  in that run was repaired and rechecked through workflow owner/standalone BAS.
- Original virtualized FileList fixture bug report knw-1791478030259573196 is now
  repaired by this delivery (20/20 pass); published entry remains immutable. The
  trusted-base fixture report remains knw-1791478030261644073. Additional catalog/
  experience observations filed separately; receipt IDs will be added below.

### Final delivery record — 2026-10-08

- Feature acceptance: complete across retained capture/revision/diff, explicit
  combination/closure/exclusion, risk acknowledgment, isolated candidate publication,
  full proposal workspace/mobile navigation/drafts, full-message history/Continue
  and typed owner-backed provenance. Scope limits remain as stated in DESIGN.md.
- Final checks: all API packages PASS (`go test ./...`); all UI 60 test files/496
  tests PASS; UI type check/lint/production build PASS; proposal CLI plus worktree
  manifest tests PASS; proposals/trailers domain tests PASS; governed proto verifier
  PASS with artifact 6eda94b04237575c4c9d7403697e6bcec53bb957dd25d032eed92a9069498a6e.
  Production adapter regressions explicitly cover configured private checks,
  hook tree/message/whitespace rewrites, lost acknowledgment, effective Git mode,
  raw-byte drift with equal filtered blobs, anchor chmod and index-only exclusions.
- Final unit/api/workflow run 20261008-174336-5b7a06fa: API PASS; unit/workflow FAIL.
  Unit's execution failure is CLI trusted-base fixture debt; no API or UI execution
  failure. The new proposal observer has zero findings in the native workflow
  report; broader catalog has 29 findings. Wider six-phase run
  20261008-172905-8e73d8d5 remains the quality/experience/structure receipt.
  These are failed scenario runs, not a production-readiness certificate.
- Maintained read-only observer efc0ddc6-6d77-443b-a801-174e55ca817e PASS; static
  workflow owner recheck has no proposal-review findings. API playbook schema debt
  filed as knw-1791481217193596235; workspace experience reconciliation debt as
  knw-1791481220743111198 through report-bug. Existing CLI trusted-base observation
  remains knw-1791478030261644073; FileList fixture observation was repaired here.
- Final lifecycle setup/restart completed healthy at 13:48 EDT, API 18710/UI 21400.
  Live read-only observer captures and simulated captured-diff evidence are named
  above; mobile concept mockup inspected. Inbox load/completed and legacy capture
  standing inspected. Explicit source revision selection survives review and
  proposal file arrows move focus with selection.
- Ordinary/operator repository commits, staging, push, history rewrite, real
  proposal mutation and human production approval were not performed. The operator
  can enter ?proposal=inbox; the existing open legacy review requires Adopt current
  contents before it becomes approvable. Whole-file risk is explicit, not isolated
  hunk attribution. Large real-repository approval latency and human actuation
  remain unmeasured; safe temporary-repository publication is the behavioral proof.
- Validation logs currently at /tmp/gct-{api-final-full,ui-final-full,cli-final-focused,
  type-final,lint-final,ui-final-build,proto-final-verifier}.log; durable scenario run
  findings/artifacts and BAS execution IDs are the long-lived receipt owners.

- Immutable work journal receipt: 16a0ff35-bc84-42a3-b315-b47558e5bfff.
- Final message verification reads the raw commit object and removes only Git’s
  structural final LF. Additional hook whitespace is a mismatch and refuses before
  publication; the named whitespace regression and full API suite pass.
- Production build succeeds with the existing large-chunk warning (main bundle
  about 1.4 MB before gzip); broad bundle splitting remains separate performance debt.


## Completed follow-up — proposal guidance and saved-version approval

- Authority: operator requests intuitive terminology/help, clickable blocked-action
  explanations with next steps, and approval despite later file changes.
- Rung: scoped W3 behavior extension under the approved exact-reviewed-content target;
  DESIGN.md follow-up supersedes blanket drift refusal only in intent-bound snapshot mode.
- Recall: search-hub found existing GCT documentation; run/memory providers degraded.
  Skills reused: experiential-ui-design, scenario-work-ladder and their prior-loaded guidance.
- Design: retained blobs staged via a new Git seam; no writes to working files. Different
  staged versions require human review/unstaging, not silent replacement. Existing strict
  mode remains for clients that do not opt into saved-version approval. Dialog/help composition
  remains feature-owned; no dependency additions or Plan Manager work.
- Complete: contextual help, blocked-button guidance with specific next steps, and
  explicit saved-version versus latest-files review choices. Shared blocker composition
  keeps explanations/actions aligned and removes duplicated message-save logic.
- Validation: full API passes; 504 UI tests pass; focused CLI proposal/worktree tests pass;
  lint/type-check/build/selector check/proto owner verifier pass. Desktop/mobile BAS
  help/update-cancel passes against real data; browser-local captured/later-edit fixtures
  pass with mutation RPCs blocked. Exact commit behavior is proved only in temp repos.
- Broader run 20261008-182338-da144e10: API passes; unit/workflow/quality retain existing
  policy/schema/quality failures. The earlier concurrent-setup run admitted no phases.
  Feature results are separate from scenario certification. Detailed evidence and BAS
  execution IDs live in DESIGN.md. No real proposal mutation or Plan Manager use.

- Follow-up work journal receipt: `3b8a385d-a446-4ba6-ab77-009c5c97b27c`. Final lifecycle restart healthy at 2026-10-08 14:31 EDT (API 18710 / UI 21400).

### Mobile bottom edge investigation — 2026-10-15

**REOPENED by physical screenshots at 11:45 EDT.** The operator's 1179×2556
screenshots (393×852 CSS pixels at 3×) contradict the previous fix claim.
The 49-step Chromium run below remains historical emulation evidence only.
Its invented 390×903 screen shape did not reproduce the phone and MUST NOT be
used to qualify the installed app. Root cause remains under investigation.

The normal header is 60px high (44px controls plus 16px vertical padding),
independently of the status inset. The photographed navigation begins around
743 CSS px, leaving about 109px to the physical bottom; this exceeds the
51px row plus the assumed 34px inset. Actual client inset and viewport readings
are needed to distinguish oversized env() from excluded native space.

Follow-up acceptance: collect automatic, bounded layout observations from the
actual installed client, including build identity, screen/layout/visual/fixed
geometry, raw/effective insets, nav/header dimensions and open overlay owner.
UI Health owns retention and readback; GCT supplies a same-origin facade.
Geometry observations explicitly do not contain OS pixels. A fabricated
standalone flag or an html2canvas rendering is not physical-device evidence.

Scoped W3 repair plus W2 evidence repair; no full scenario-readiness claim.
Prior-art search found the existing mobile regression oracle and percentage
height shell in docs/concepts/ARCHITECTURE.md. Source-ledger recall timed out;
other search providers returned related work. Existing GCT viewport/CSS edits
and ui-health measured-height rule were already dirty at investigation start.

Hypotheses and falsifiers:
1. Divergent inset owners: GCT publishes unused --gct-safe-bottom while BottomNav
   reads raw env() and library overlays resolve screen-bottom independently.
   Test geometry with nonzero insets and a viewport ending before the screen;
   disprove if all owners reserve the same effective inset in both states.
2. Canvas ownership: custom mobile dialogs bypass library drawer canvas rules.
   Test page background through menu -> Settings -> close and compare an actual
   FullPageDrawer; disprove if custom dialogs and library drawers paint equally.
3. Evidence never exercises the phone shape: mobile dimensions alone produce
   zero env() insets. Initial BAS capture a2ed1402-8212-45ba-940a-f27ab6a775c4
   confirms a zero-height status strip. Driver source includes emulation but
   running BAS dates to Oct 6; lifecycle reports a failed replacement UI build.
   Test governed replacement and observed env() values before accepting proof.

Initial screenshots: /tmp/gct-safe-area-before/a2ed1402-8212-45ba-940a-f27ab6a775c4.
These are zero-inset Chromium evidence, not the operator's physical phone state.

#### Findings, repair and qualification

Operator clarified: **"Installed Rigel app from Home Screen"**. The native
home-indicator strip can be outside the app's reported viewport. Screenshots
of the web document cannot show or certify that physical OS region.

Confirmed causes:
- GCT wrote unused --gct-safe-bottom while BottomNav used env() directly.
  In the simulated installed 390x844 viewport on a 390x903 screen, the shared
  detector correctly resolves the effective inset to zero. BottomNav 1.5.4
  now consumes that same library contract, rather than reserving another 34px.
- The custom menu, Settings and mobile dialogs bypassed the library's page
  canvas rule. Reused BottomSheet/FullPageDrawer for the menu, Settings, metrics,
  history filters, upstream information, discard/delete/stage confirmations,
  baseline creation and scenario picker. Removed private gesture/portal/scroll
  lock code and obsolete local safe-area classes. History/blame headers also
  reserved a second top inset inside TopSafeArea; removed that duplicate.
- BAS accepted its CDP inset override but discarded it across navigation when
  its owner detached. Retained one page-owned CDP session and reapplied the
  profile after completed navigation. Production-session initialization needed
  the override after setup as well. A simple browser-only test was insufficient;
  the production-session integration and an actual BAS execution caught that.
- BAS expands `${...}` in workflow expressions. The old mobile regression used
  JavaScript template literals for selectors and RGB conversion; interpolation
  corrupted those expressions (observed metaRGB was `rgb(, , )`). Replaced those
  literals with concatenation and retained every existing assertion.
- UI Health assumed one color for all four edges and used CSS inset counts as
  screenshot pixel counts. It now accepts a semantic bottom-owner color, scales
  sampling to the screenshot, retains raw versus effective inset measurements,
  and reports zero-inset mobile captures as `visual_safe_area_not_exercised`.

Qualification:
- BAS shadow execution **6f99457c-6cf7-4992-ab0e-381db5e66475**: **49 steps PASS**,
  covering Changes/Diff return, measured 59/34px insets, all five nav controls'
  bounds/descendants/hit tests, exactly one inset, menu and Settings canvas
  changes/restoration, and simulated installed app zero-inset navigation/footer.
- Screenshots, DOM and traces are retained by BAS under that execution. Use
  `browser-automation-studio --instance shadow executions screenshots <id> --json`
  and `executions timeline <id> --json`. Named screenshot nodes and observed
  profile values are retained in experience/fixtures/mobile-regression-oracle.json.
- UI Health live analyzer passed all **six** selected real screenshot states.
  Negative control with expected bottom red FAILED with
  `visual_safe_area_color_mismatch`, measured bottom **#020617**. Both requests
  used the same captured screenshot/DOM bytes; only the expectation changed.
- GCT focused UI: **63 tests PASS**, type-check/lint PASS. BAS context/navigation
  regressions: **47 tests PASS**, including real Chromium and production session
  creation. UI Health visual/runtime packages: **65 tests PASS**. RCL BottomNav
  unit/a11y tests: **4 PASS**, plus owned component contract report
  **ctr_16d50785fb2d68a6 PASS** for release **1.5.4**. Strict RCL package build
  selected verified artifact 5c062f2abe3a65a8e276cec4e840c766b8a6f06bae1208e243d48420f48bb90e.
  Proto owner verifier completed with exit 0; no generated files hand-edited.
- GCT and UI Health were rebuilt/restarted through lifecycle and are healthy.
  BAS shadow was rebuilt/restarted and is healthy. Live BAS intentionally still
  runs the existing bas-goal frozen serving tree; no engagement was promoted,
  abandoned or force-bypassed. Early live restarts therefore tested the frozen
  source, not this repair. The exact qualified target is BAS **shadow**.
- Scoped Test Genie **20261015-151502-3513bec9 FAIL** across unit/ui-health/
  workflow/experience. Relevant debt includes CLI trusted-base fixtures, unit
  policy/companion/coverage findings, UI governance debt, old API workflow JSON
  with unknown `requirements`, observer action classification, and unresolved
  experience bindings. This is not a full scenario certification. Earlier docs
  already record those debt families; no check was weakened to turn them green.
- The unrelated one-asset RCL gate fault on missing collection-page manifest is
  filed as **knw-1792077958624332549** through report-bug. The supported component
  contract runner passed. The attempted UI Health shadow start rejected the
  documented --variant-dependencies flag; this did not affect live analyzer proof.

Review passes:
1. Found secondary top-inset duplication and the scenario-picker canvas path;
   removed duplicate padding and migrated the picker to the shared drawer.
   Scoped keyboard reveal to its owned scroller. Added selection/search coverage.
2. Reviewed the resulting boundary for clipping, double padding, busy-operation
   dismissal, canvas restoration, provenance, and unnecessary code: no additional
   material findings. Confirmations retain guards during mutations.
3. Rechecked code/evidence after focused tests, six-state pixel analysis and the
   negative control: no additional material findings. Physical installed-iPhone
   confirmation remains unverified, as explicitly recorded above.

Reusable case remains the maintained BAS fixture. An Action creation preview
for workflow-health start-selected was structurally valid but unvalidated at its
owner's permission/governance boundary; no unvalidated Action was published.

- Work record: **cac62d1b-66b3-431d-a9d7-5eeab774f5a0**. Documented variant
  flag rejection filed as **knw-1792078526646646813**. Final review removed a
  peripheral, unqualified media-lightbox canvas override; the requested drawer
  paths use the shared library's tested paint ownership.


#### Physical client follow-up — 2026-10-15 12:06 EDT

Automatic UI Health intake received real installed-client observations after the
operator reopened Rigel. Observation **9f5d79f0da36073c97f4f9427936726d**:
393×852 screen, innerHeight/visualViewport/100dvh **793**, measured 100lvh **852**,
59px top inset, 34px raw bottom inset, effective bottom **0**, shell bottom **793**,
nav **742–793**, track **745–793**. Build `index-NsiwAUXo.js`. Thus the earlier
zero-inset workaround worked as written and still left 59px of drawable canvas
unused. The photographed nav height was 51px plus that 59px band, not duplicate
34px padding. The tentative partial-overlap hypothesis above was falsified;
no partial-gap arithmetic was shipped. Full CSS canvas measurement is the owner.

The named Commit/Push surfaces already adopted **ResponsiveDialog**. That
library presentation lacked the canvas rule in FullPageDrawer/BottomSheet;
Settings-only coverage missed it. Both consumers also overrode the panel color
with `!bg-slate-900`, which would disagree with a library-owned canvas. Removed
those overrides and repaired ResponsiveDialog's mobile canvas ownership.

The shared viewport now retains raw layout/visual measurements and exposes a
measured drawable frame only when standalone, top-level, black-translucent,
full-width geometry is corroborated by 100lvh matching the screen and the
status-height deficit. GCT sizes all roots and the shell to that same frame.
The common overlay hook fills it for every presentation; keyboard avoidance
still uses the established keyboard viewport. Library bottom clearance is again
reserved once inside the full frame. Web Console's projection also consumes
the optional frame, preserving fallback behavior for older host snapshots.

Published through draft ownership: useViewportEnvironment **1.1.1**,
useOverlaySurface **1.4.5**, ResponsiveDialog **1.3.2**. Intermediate overlay
release 1.4.4 failed portable declaration emission; it was superseded and
retired through VersionLifecycleService after owner eligibility preflight.
No released source was edited. Strict package build selected verified artifact
**0972fc40be09a85dd9d0bcf7af30c1923f5fbd2a9521b584b55a63f60c67fed7**.

The actual-device reports eliminate manual screenshots for layout diagnosis.
Native screenshot automation is a separate capability: device-control inventory
exposed no iPhone screen transport; scenario-to-ios target probing returned a
bridge inventory authentication failure. That failure does not establish that
no trusted macOS/iPhone route exists. OS pixels remain explicitly unverified
until a device transport supplies them; browser screenshots cannot substitute.


Final phone confirmation: **"Looks great! The top navbar could still use some
 tightening up perhaps, but everything else looks great"**. Automatic actual
client report **2162035af13965de10fdba14daf83e80** (16:20:54Z,
`index--YIPbwyE.js`) now measures innerHeight/dvh **852**, shell/nav bottom **852**,
nav top **767**, one **34px** bottom inset, and all five control hit tests **true**.
Push report **1bae2b98771fe2b47b56528308582bad** measures panel bottom **852** and
panel/body both **rgb(23,32,51)**; nav hit tests are correctly obstructed by the
modal. On close the canvas restores to **rgb(2,6,23)** and nav hits pass again.
The remaining top toolbar is **45px**, below a single measured **59px** status
inset. Tightened its horizontal gutters/gaps while preserving 44px controls.

BAS shadow replay **a89d764a-f725-4e3f-8436-2dbf9c2bb84a** completed all **50**
steps using a 393×852 browser and the actual phone's shortened-report shape.
The earlier failed follow-up replay d2af7167 used the fixture's old 390×844
settings despite requested parameters; corrected the fixture-owned settings.
This remains Chromium evidence, not native OS pixels.

Validation: focused GCT UI **25 PASS** (and final changed-client subset **19
PASS**), type-check/lint PASS; shared frame/overlay/canvas focused **6 PASS**;
Web Console viewport projection **7 PASS**. UI Health visual/runtime/schema/
observation packages and all CLI packages PASS. Negative regression proves
`chrome.requireNativeScreen` rejects emulation. Client intake now retains
receiver-computed findings for unused canvas, unreachable nav controls,
oversized declared headers and overlay/canvas style disagreement.

Broad checks remain separate: GCT Test Genie **20261015-161500-da4eda78 FAIL**
(unit/ui-health/workflow debt); UI Health unit **20261015-161810-262b4407 FAIL**
(parser lookup under unrelated home Git repository plus existing UI provider
policy). Corrected parser tooling fallback after the suite exposed that host
fixture path. RCL story contract **ctr_67148ab6eb5d2f3d FAIL** at BAS screenshot
capture across variants; source/closure/story-contract stages passed. This
report is not represented as passing browser qualification. The actual GCT
Push drawer rendering is independently confirmed by its phone report and the
operator. No full scenario certification is claimed.


Final coherence review resolved an invalid dependency pin in the RCL preview: ResponsiveDialog
1.3.2's immutable lock observed overlay 1.4.4, which had been superseded/retired.
A fresh draft retained that old pin until the governed catalog lock generator
ran. Published **ResponsiveDialog 1.3.3** with overlay **1.4.5**, retired the
unusable intermediate 1.3.2 through the owner, and confirmed preview HTTP **200**.
The selected strict runtime artifact is now
**e4c8ae2954411c68a917e98da5270bf5f571d3ced99c66ebf8f0524838dd2f8f**.
The retired releases were never hand-edited. Final owner browser contract
**ctr_bc3043135ddb973c FAIL** still reports screenshot-capture failures for
several stories and readiness timeouts for others. Source/closure/contract
stages pass. An independent full-viewport capture of the published 1.3.3
sheet, **49d20eb9-5db7-45b8-984b-5304a8706e45**, visibly renders the dialog
without console errors. That capture does not replace the authoritative
contract. Cause remains unestablished; filed through report-bug as
**knw-1792082382843370162**. Browser-contract qualification remains open.

Parser regression coverage now passes every selected shell-ownership case and
the validation handlers. The remaining PWA/default-kit fixture failures are
filed as **knw-1792081948919897572**. Web Console's approved local dependency was
refreshed through Scenario Dependency Analyzer without manifest changes; its
frameHeight freshness errors are gone. Remaining machine/desktop test-fixture
type errors are filed as **knw-1792081948624493753** through report-bug. The seven viewport
projection tests pass; no Web Console runtime restart was required.

Work record: **bcaf815a-11a4-4af9-8dd7-61fd0ccc5227**. Final scoped diff
whitespace check passed. Runtime healthy after the coherent library rebuild.
