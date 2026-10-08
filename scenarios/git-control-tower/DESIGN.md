# Rigel workspace design

Git Control Tower is a diff-first repository workspace. Advisory results are
secondary interpretations of one explicitly identified subject; they never
silently select a staged set, checkout a branch, run hooks, or publish to a
host.

## Information hierarchy

1. Subject bar: repository identity, scenario/file scope, source standing, and
   freshness or stale warnings.
2. Changes and diff: the selected evidence remains central on desktop.
3. Inspector: summary, findings, drafts, evidence, and provenance open beside
   the diff on wide screens and as full-width views on mobile.
4. Human handoff: any stage, commit, merge, undo, comment, PR, or release
   action shows exact destination, revision, preconditions, and verified human
   authority before it can be attempted.

## State contract

Every advisory surface renders loading, empty, partial, unavailable, failed,
and stale states. “No findings” means only that examined evidence produced no
finding; it never means omitted or unavailable evidence passed. Host capability
states are shown per capability, not collapsed into one connected/disconnected
badge.

## Responsive behavior

Desktop preserves the file list and diff as the primary workspace with an
optional inspector. Mobile uses the existing tab/panel navigation and full
width detail views. Subject identity and scroll/focus state remain attached to
the selected operation when moving between views.

## Draft safety

Commit, pull-request, and release text is editable and revisioned separately
from repository state. Regeneration displays a comparison and preserves human
edits. Trailer chips distinguish owner-resolved, legacy, ambiguous, and
unresolved references. A draft button cannot stage, commit, checkout, fetch,
push, publish, or grant authority.

## Proposal review delivery specification — approved 2026-10-08

### Authority and durable context

The operator approved the reviewed API correctness, full-workspace proposal UI,
explicit grouping/closure, trailer integration, validation and debt reduction.
They requested durable context and explicitly prohibited Plan Manager. This is
a bounded delivery task: this document owns design/acceptance; the existing
`docs/internal/PROBLEMS.md` owns checkpoints, evidence and remaining work.
Scope is this scenario and its owned proto contracts/generated artifacts.
Necessary shared-owner extensions must be recorded. Agents must never commit,
stage, push or rewrite the real shared repository. Use existing isolated test
seams and authorized lifecycle commands for verification/deployment.

Reviewed baseline: desktop review occupies the narrow commit sidebar while the
diff area stays empty; mobile truncates distinguishing filenames. Withdrawal
only closes a database record but does not explain preservation of code/index
or expose its reason. File links use current staged/unstaged diffs, omitting
staged changes when a file also has unstaged changes. Index verification occurs
before hooks; a rewritten commit can be detected only after publication.
Anchors/proposals contain hashes rather than retained bytes, edits overwrite
revision payloads, and automatic folding adopts current contents silently.
Proposal trailers exist, but history reads subjects and Continue only increments
lowercase pN. Owner resolvers for effort/epoch/run/plan/backlog are absent.
At review time nine proposals existed, none committed; the open E28–E33 package
contained thirteen files, eight mixed-prior warnings and seventeen exclusions.
These counts are observations, not continuing assertions about runtime state.

### Content, approval and evidence decisions

1. Retain immutable proposal revision payloads and exact candidate bytes/blob
   IDs/modes, base HEAD, branch, message, membership and provenance. Lifecycle
   activity can advance separately. Old revisions remain addressable.
2. Diff the captured candidate against captured base content; never substitute
   live files. Cover additions/deletions, executable/symlink changes and bounded
   binary/large-file standing. Legacy missing captures remain explicit and
   require a deliberate current-content revision before approval.
3. Refresh status automatically, never review contents. Adopting current bytes
   creates a revision, shows changed paths and invalidates approval. Acceptance
   receipts remain attributed to their source revision; newer contents must not
   inherit an invented passing gate result.
4. Freshness is identity, not authorship. Explain mixed-prior/no-anchor/outside
   scope in plain language. Inclusion of uncertain prior work requires explicit
   human acknowledgment bound to the reviewed revision/intent.
5. Hooks/checks finish before the exact tree/message publication decision.
   A changed candidate refuses before repository ref publication. Use isolated
   candidate/index processing and explicit branch/HEAD publication preconditions;
   do not leave private hook changes in the operator's working tree or index.
6. Preserve human-only authority, single-use revision-bound intents, foreign
   staged refusal, operator authorship and audit. Commit exactly reviewed blobs,
   modes, paths and message. Approval never implies push. Reuse common safety
   primitives with ordinary commits rather than create competing writer logic.
7. Failure reports must be truthful about restoration and unexpected commits;
   never report a green verified success for a mismatched published commit.
8. Automatic hunk attribution is out of this delivery: anchor comparisons cannot
   identify concurrent writers. Future selective patches require retained applied
   change evidence and conflict-aware construction. Whole-file inclusion must
   remain visibly reviewed and explicitly acknowledged when attribution is mixed.

### Lifecycle and relationship decisions

- **Close proposal:** changes only lifecycle state; optional reason and replacement
  proposal link. Copy: "Removes this proposal from the inbox. Your code and staged
  changes stay untouched." Retain activity and closed records.
- **Exclude file:** changes proposal membership only. Explain that the file stays
  uncommitted; if every file is excluded, offer Close instead of an empty commit.
- **Related:** same effort/work reference, navigable without modifying contents.
- **Replaced by:** explicit revision-bound relationship; age/epoch number does
  not prove coverage. Disclose omitted/different files before disposition.
- **Combined:** explicit union of selected revisions. Shared files appear once;
  different captured versions require a visible decision rather than newest-byte
  selection. Retain all source proposal/revision/effort/epoch/run/evidence links.
  Supersede sources only after durable creation. Do not silently truncate provenance.
- Stop automatic content folding. Existing folded records stay readable. New
  submissions may be grouped automatically for navigation but cannot silently
  change another review candidate or discard operator wording.
- A combined package supports a single explicit commit. Independent proposals
  remain independent; do not promise atomic batches of multiple commits.

### Experience and visual concept

Concept: a calm repository review desk — scope, change, decision. Reuse Rigel's
palette/primitives and diff renderer. Prefer readable titles, restrained status,
consistent typography and one clear primary action over nested cards/badge walls.

Desktop proposal mode fills the workspace like history mode:

| Region | Responsibility |
| --- | --- |
| Subject bar | Readable title, repo/branch, effort/epochs, readiness, Back to working changes |
| File pane | Candidate files only, basename-first directory context, inclusion and risk |
| Main pane | Exact captured diff; first file automatically selected |
| Inspector | Summary/evidence, editable message, related/source links; technical disclosures |
| Action area | Operator/destination, blocker/acknowledgment, Create commit; secondary Close |

The inbox is a first-class entry with open/closed filters, effort grouping,
plain-language readiness and selection for Combine. URL links survive reload.
Never expose working-tree stage/discard controls inside proposal review.
Preserve operator drafts and selected file/focus across navigation/refetch.
Mobile composes overview → files → diff with retained identity, explicit back
navigation, reachable bottom actions, safe areas and keyboard-aware editing.
Long common prefixes must not hide distinguishing filenames; full paths remain
available. Evidence, activity, exclusions and raw trailers use disclosures.
Mockup desktop/mobile before product UI work and inspect live captures against
the concept. The user's approval authorizes these recommendations and routine
design choices; do not introduce extra permission gates for implementation.

### Trailers and attribution

Share parser/renderer/vocabulary across proposal/manual composition and history.
Preserve unknown trailers. History exposes full body, parsed trailers and resolved
links/uncertainty without rewriting historical commits. Continue retains work
links and adds `Vrooli-Continues: <previous hash>`. Keep P/pN as a legacy display
convention, not the authoritative relationship. Combined candidates retain source
manifests via proposal links and every source work/evidence relationship.
Wire owning effort/epoch/run/plan/backlog reads with truthful unavailable/private/
deleted states. A supplied run ID is not a verified creator identity.

### Sequence, cleanup and acceptance

1. Write this specification/checkpoint; inspect generation/testing owners.
2. Add retained revisions/content and exact bounded diff contract.
3. Harden isolated candidate checks/publication and prior-work acknowledgment.
4. Implement explicit combining/replacement/removal/reasons; retire automatic folds.
5. Complete history/manual trailer continuation and resolver integration.
6. Build feature-owned inbox/workspace and desktop/mobile composition; remove
   obsolete detail/mount paths and extract genuinely shared App orchestration.
7. Validate focused Go/CLI/UI regressions, scoped Test Genie phases, proto
   owner verification, UI lint/type-check/build and BAS observer journeys.
8. Rebuild/restart via lifecycle owner; update docs/requirements, inspect scoped
   changes and record receipts/work journal. No real commit as a test.

Prefer existing owners; extract boundaries that remove duplication or improve
independent tests. Remove replaced behavior and stale docs; do not merely lower
line counts. No shims, hand-edited generated code or ungoverned dependencies.

Acceptance: immutable old revisions/candidate diffs survive drift and closure;
staged+unstaged/binary/deletion/mode cases are truthful; hook changes, stale
revisions, foreign staging, wrong destinations, agents/replays and missing risk
acknowledgment cannot publish; closure/exclusion preserves code/index; combining
is explicit and retains source evidence; desktop/mobile review begins with useful
content, exposes evidence and supports URL/draft/navigation; new trailer links
work while legacy history remains usable. Missing provider evidence stays unknown.

Proof follows `docs/TESTING.md`: focused regressions with selected names; relevant
`vrooli scenario test git-control-tower --phases ...`, then block once with
`test-genie runs wait --json git-control-tower <run-id>`; governed proto generation
and verification; UI lint/type-check/build; BAS desktop/mobile observer captures
inspected against mockups. Compilation/screenshots alone do not prove acceptance.

### Harness handoff

Work until this approved specification is implemented and verified across API,
CLI and UI. Read this document, existing problems and testing guidance first.
Do not use Plan Manager. Preserve the shared dirty tree and human-only authority;
use admitted isolated/in-memory fixtures, never real commits for validation.
Repair understood blocking defects and retire obsolete duplication without
weakening tests. Blocked means a decision, credential, or approval you lack.
Name it. Record changed files, decisions, proofs, failures and next steps in the
existing problems document before compaction/interruption, then resume from it.
Finish only when acceptance is verified; final report: changed, verified,
remaining, unverified. No agent commit, push or history rewrite.

### Delivered behavior and practical limits

Implementation is feature-owned in `ui/src/features/proposals/`; retained review
and grouping live in `api/internal/proposals`, production candidate adapters in
`api/proposal_{capture,candidate,environment}.go`, shared diff/history boundaries
in `repo_{diff,read}_connect.go`, and trailer ownership in `trailer_owners.go`.
Canonical typed schemas/generated clients are owned by packages/proto. Draft
storage has embedded declarative domain schema through api-core EnsureSchemas.

Commit candidates are checked in a private checkout at current compatible HEAD,
with exactly the retained files/modes applied. Portable configured checks and
pre-commit/message hooks execute there; hard-coded source-root commands or outside
working directories refuse. Environment roots are mapped to the candidate and
Git storage overrides removed. Post-commit hooks do not run during this flow.
Git branch/index locks plus short proposal revision lease protect publication.
A lost ref acknowledgment is resolved before restoring an index; an uncertain
outcome names its candidate and requires history review instead of a green result.

Limits are explicit: 2,000 paths/package, 8 MiB/file, 64 MiB retained base/candidate
content per package, 2 MiB diff output, 2–20 selected combination sources, and 400
run links without truncation. Metadata owner reads share an 800 ms response budget.
The inbox currently shows the first 200 records and names that boundary; CLI filters
can inspect older state/effort records. It does not promise cross-page batching.
Whole-file granularity remains deliberate. Acknowledgment approves carried earlier
work; it does not recover concurrent author attribution or isolate epoch hunks.

Existing legacy records retain their actual payloads/history but have no invented
original bytes or phantom revisions. Open legacy reviews require explicit Adopt
current contents and review before approval. Closed legacy reviews are historical.
No human-authored production approval/commit was performed by this delivery;
publication was verified only in temporary repositories and in-memory fixtures.
Live BAS proves navigation/preservation explanations; browser-local API fixtures
prove captured-diff layout only. Wider scenario failures remain separately reported
and do not become passing receipts by virtue of these focused results. Final
receipts, failed gate causes and bug IDs live in `docs/internal/PROBLEMS.md`.


## Proposal review follow-up — interactive guidance and later edits (2026-10-08)

Operator feedback: technical saved-version terminology is unclear; disabled commit
buttons must open an explanation with actionable next steps. Proposals should remain
committable after later working edits. This supersedes the earlier requirement to
refuse every working-tree drift, only for an explicitly authorized saved-version mode.

- Use plain language: “proposal version”, “latest working files”, “Update proposal”.
  Contextual help explains exactly what is saved, committed, or left uncommitted.
- A visually disabled primary action remains focusable/clickable via a guarded wrapper;
  it opens a device-appropriate explanation dialog and offers specific next steps.
  Loading/actions in progress also explain their standing; they never trigger a write.
- The review can commit the saved proposal version while leaving subsequent working
  edits unchanged. “Use latest working files” remains an explicit new revision/review.
- Saved-version mode is bound to the human intent and stages retained blobs/modes,
  never current files. The intent ignores subsequent working edits in this mode, but
  continues to bind exact proposal revision/message, branch/HEAD and source index.
- Refuse different staged versions on proposal paths in saved-version mode before
  changing the index. Explain how to review/unstage them. Matching staged proposal
  bytes are safe. Foreign staging, affected base moves, changed destination, missing
  retained bytes, unknown standing, source index races and prior-work acknowledgment
  keep their guards. Failure restores the index; private checks/exact publication remain.
- Desktop explanatory dialog stays near review; mobile is a reachable scrollable
  sheet/dialog. Reuse owned primitives; split guidance/actions out of ProposalDetail.
- Validate temporary-repository saved-version commits with later modification,
  deletion/recreation, modes, check-time editing and staged refusals; transport intent
  binding; clickable blocked-action/help next steps; desktop/mobile BAS without real
  approval. Preserve operator code, real proposals, and no Plan Manager engagement.

Follow-up operator decision: offer both choices. Recommend the saved proposal
version as the default because it matches the reviewed diff; use latest working
files explicitly creates a new revision. On 2026-10-08 the operator asked to make this choice available and requested
a recommendation. This does not imply permission to apply any real proposal.

Interaction structure: `reviewBlocker` determines one ordered primary blocker
used by both the main action and the dialog next step. Authority, closed review,
comparison, missing original contents, unloaded diff, unsaved message, prior-work
acknowledgment, staged mismatch, affected committed base, unknown status and
foreign staging each have a specific explanation. Busy actions explain without
starting another write. Modal errors remain visible inside the dialog. The native
button remains disabled behind a keyboard/touch/mouse explanation wrapper; the
library owns focus trapping, Escape and the mobile sheet.

Follow-up verification (2026-10-08):

- Full API `go test ./...` passes. Named production regressions commit retained
  contents with later modification, addition, deletion/recreation, working deletion,
  executable-mode changes and editing during checks. Different staged contents and
  staged deletion refuse before staging; matching staged proposal bytes are accepted.
  Affected committed base changes refuse. Index-race regression runs both modes.
- Transport tests accept working edits after saved-mode intent preparation and refuse
  either direction of intent-mode substitution. Strict-mode regressions remain green.
- Full UI: 504 passing tests in 60 files. Proposal regressions: 30 passing tests.
  Mouse/keyboard blocked actions, prior-work decisions, legacy adoption, comparison
  return, staged-change guidance, update cancellation, modal-visible errors and
  read-only caller standing are covered. CLI proposal/worktree packages pass.
- Proto owner generation/verification passes. Initial GCT artifact
  `661767d4a0c5569153cb6b86b98ad32412376021725253e870ecb52672ce70f7`;
  later lifecycle selected the current shared artifact
  `bce914f00545f89becf470dae9d5cf6b0cde813f072f9ef3300135ead1019538`
  after concurrent owner changes. Final owner verifier passes; no generated edits.
- Type-check, ESLint, selector owner generation/check and production build pass.
  Existing main-bundle size warning remains. Lifecycle setup/restart is healthy.
- BAS real-data navigation/help/cancel passes at 1440×900 and 390×844:
  `f97c3431-c81d-4bc6-891c-c24a86485d8c`,
  `781a1911-e200-44e3-add9-8c87d3d79ac2`.
  Browser-local saved-version/later-edit fixtures, with mutation RPCs blocked, pass:
  `d71d47ba-bdbc-4f93-97fd-22fcf4239d18`,
  `8c91c6f7-e057-4622-8c7e-6b4829ab3db1`.
  Screenshots were inspected. Browser evidence proves interaction/layout; temporary
  repository tests prove publication. No real approval, update, closure or grouping.
- Broader Test Genie `20261008-182338-da144e10`: API passes; unit/workflow/quality
  fail on existing projection policy, old BAS schema/observer/selectors and scenario
  quality contracts. This is not a green scenario certification. An earlier run
  `20261008-182150-4dac6d9b` admitted no phases because setup held the lifecycle lock;
  the final run followed successful restart. Local UI/API/CLI evidence is separate.
- Real open proposal `gctp-95f0216fa761` remains r2 with no retained contents; no
  real proposals were committed or changed. No Plan Manager operations were used.

- Follow-up work journal receipt: `3b8a385d-a446-4ba6-ab77-009c5c97b27c`. Final lifecycle restart healthy at 2026-10-08 14:31 EDT (API 18710 / UI 21400).
