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

## Staged commit messages — approved 2026-10-21

### Authority, work shape and ownership

The operator approved full implementation, validation and a dedicated subsequent
refactoring/debt-reduction pass after reviewing the evidence-first mobile
composer concept. This is a bounded delivery. This section owns the target and
acceptance; `docs/internal/PROBLEMS.md` owns checkpoints and validation receipts.
Native sub-agents implement separate boundaries and independently review them.
Do not stage, commit, push, reset or rewrite the real shared repository while
implementing or validating this feature. Tests use isolated fixtures and routed
test storage. Existing unrelated dirty files belong to their current authors.

The source mockups are the approved `flow.html`/`flow-storyboard.png` concept in
`/home/matthalloran8/.codex/artifacts/gct-staged-context-design/`. They illustrate
the interaction, not a backend or a promise about production latency.

### Desired behavior

1. Ordinary manual staging is the primary input. A staged selection may contain
   zero, many or partial proposals/runs and changes without agent evidence.
   The feature MUST work without proposal membership or known authorship.
2. The Commit panel MUST expose Title, optional Description and an optional
   trailer disclosure. The sparkle Suggest button MUST have an accessible name.
   Existing message text and unknown trailers MUST survive parsing, navigation,
   regeneration and comparison. Conventional validation MUST inspect the title
   separately from the body/trailers. Commit authority remains unchanged.
   History Continue actions merge continuation trailers into existing editor text
   and prefill the continuation message only when the editor is empty. AI adoption
   and continuation selection preserve each other's references in either order.
3. Suggest MUST first collect bounded evidence programmatically with no model
   or agent call. The UI MUST show gathering state and allow cancellation.
   The collector MUST read staged Git objects, never substitute working files.
4. Suitable retained messages MAY be reused or combined deterministically only
   for supported staged scope. Partial suggestions MUST disclose unmatched
   edits/files and MAY use a broad suffix such as "and related updates".
   Selecting a partial suggestion MUST NOT force agent invocation.
5. When no reusable message exists, the UI MUST say so and offer Start drafting
   agent and Cancel. Related context without a reusable message is not a ready
   suggestion. No stage of collection MAY automatically start an agent.
6. Starting an agent is explicit. Its job MUST be message-only/read-only, bounded
   to the immutable staged subject plus authorized context, and MUST NOT create
   a commit proposal, stage files, commit, publish or alter agent grants.
7. Queued/running jobs MUST show authoritative status, elapsed time and Cancel
   agent. Hiding a panel or reconnecting MUST retain the same job. Cancellation
   MUST remain pending until the execution owner confirms its disposition.
8. A completed result MUST appear as a suggestion with Use and Discard actions.
   Arrival MUST NOT overwrite human text. Concurrent edits require comparison.
   Accepting a suggestion fills editor fields only; commit is a separate action.
   A confirmed successful UI commit retires that repository's completed suggestion session
   and saved recovery handle, including when the composer is closed. Failed or
   refused commits preserve the draft. Late reads or launch replies from a retired
   session MUST NOT restore it. Newer sessions and unresolved/active job handles
   remain available for reconnect/cancellation. Durable server job history remains unchanged.
9. Staged-content drift MUST retain existing text/results and mark applicability
   stale. Working-tree-only edits MUST NOT stale a staged-content draft. Changed
   staging MUST NOT silently restart inference or adopt a new input subject.
10. Empty staging, partial/unavailable evidence, failed/cancelled agent jobs,
    binary/large-file omissions and source privacy MUST have truthful states.

### Content and context contracts

The staged subject identifies repository, base HEAD and sorted changed index
entries (old/new paths, modes and blob IDs). Read object bytes by immutable IDs;
check index/base identity before and after capture and before applying a result.
Unmerged entries are explicit errors. Every bounded omission carries a reason.

Context relationships are many-to-many and scoped to matching edits. Preserve
content correspondence, relevance and authorship as separate facts. Path overlap,
matching text, timestamps and an operator-selected explanation MUST NOT become
automatic authorship claims or inferred attribution trailers. Retained proposal
before/after content can support partial matching; a final digest alone cannot
prove a subset. Missing/legacy captures remain explicit.

Collect current filesystem mtime and nonzero cached index-stat mtime as labeled
hints, with content comparison to the staged version. Partial staging, deletion,
touches, refresh, preserved timestamps and multiple edits can invalidate or omit
these hints. Search retained message/tool activity in bounded nearby windows;
do not use staging time or treat a resumed session's full lifetime as activity.
Use paths, rare changed identifiers and patch fragments to narrow/strengthen
candidates. Preserve native versus synthesized/import timestamps and unknown
scope/freshness. No matches in available evidence does not prove no agent work.

Agent Manager owns policy-filtered session/event search and execution. Improve
imported scope/timestamp/freshness at that owner where required for this path;
do not bypass its visibility/authentication boundary with raw transcript scans.
Code Facts already owns file observation infrastructure; a new general historical
filesystem journal is not required to deliver the approved staged composer.

### Large staged selections — correction approved 2026-10-22

The operator reported that 256 files excludes most real commits and requested
8–16 times that capacity. Support 4,096 changed index entries. This corrects the
existing staged-message behavior without changing commit authority.

- The complete sorted index manifest MUST remain the immutable subject. Every
  accepted file participates in its digest and drift check, even when detailed
  bytes are omitted. Never narrow the commit or silently ignore tail entries.
- Manifest/storage limits and model-context limits MUST be separate. A valid
  4,096-file subject MUST NOT subsequently fail merely because full paths, blob
  IDs, coverage arrays or repeated source metadata exceed the old prompt bound.
- Collect representative detail with bounded subprocess and byte work. Prefer
  breadth across affected scopes and meaningful source over generated/lockfile
  volume. Exact retained blob/mode correspondence does not require patch sampling.
- The agent MAY receive scope summaries, stable file references and selected
  patches rather than thousands of full paths. Actual AM workflow input MUST
  remain below its existing 64 KiB structured-value ceiling; full evidence stays
  with GCT's existing durable job owner. No generic owner limit expansion.
- Detailed inspection MUST map back to the complete canonical manifest.
  Unexamined files remain uninspected; summaries and sampling MUST NOT imply
  full detailed review. Message scope is separate: a whole-selection synthesis
  may use inventory and selected patches. Do not echo thousands of file names.
- Existing jobs/results and request keys MUST remain interpretable across a
  workflow revision. Collection and recovery MUST NOT silently launch new work.
- UI counts MUST describe the whole selection. Large evidence lists MUST render
  bounded previews with deliberate expansion. Evidence detail never filters the
  files being committed.

Acceptance covers 4,096-file collection/admission/results with realistic long
paths, bounded representative work/prompt/output, a clear 4,097-file limit,
omitted-tail drift, exact retained matches without sampled patches, legacy jobs,
strict coverage and responsive large lists. Use isolated Git/fake-owner fixtures;
preserve the operator's real staging, drafts and jobs during activation.

### Commit drawer follow-up — operator feedback 2026-10-22

IMG_1829.png shows a dense mobile drawer and a failed workflow admission
(`unauthenticated: 401 Unauthorized`). The operator reports that pressing Start
appears to do nothing and asks for polish, testing and cleanup. This is a bounded
correction to the existing utility drawer, under the authorized implementation.

Use a focused drafting workspace: compact staged-selection summary, prominent
current step and result, optional context, then one clear next action. Put
low-level evidence counts and technical explanations behind a single evidence
drill-in. Keep full selection and partial coverage honest without repeating
warnings. Desktop may expose the supporting evidence beside the draft; mobile
prioritizes progress/result and keeps the primary action in its footer.

Admission begins with immediate visible feedback before the network resolves.
Queued, running, cancellation, reattachment and failure must each have explicit
copy and actions. A failed launch must say whether an owner job exists; do not
claim an agent ran when admission failed. Elapsed time and last reported phase
remain visible for active work. Preserve request-key recovery and staged-content
identity. Errors belong next to the status and primary action, not beneath
several evidence sections. Show actionable human wording with technical detail
available separately. No raw credential disclosure or authorization widening.

Trace the 401 through browser → GCT → Agent Manager using existing actor/grant
propagation. Repair that owner boundary when defective; do not bypass it with
anonymous retries, fabricated tokens, or broader permissions. Confirm the real
launch failure before changing authorization behavior. Validate regressions,
desktop/mobile captures and asynchronous launch states; then independently review
correctness and simplify the touched code.

### Partial drafting follow-up — operator feedback 2026-10-22

An agent-produced partial message must offer an explicit action to investigate
the remaining staged changes. Repeating the same representative sample is not
that investigation. Reuse the existing durable drafting owner and immutable
staged subject; expand access to staged evidence in bounded chunks and preserve
coverage honesty. Keep the previous draft available if further investigation
fails, is cancelled, or remains partial. Never promise complete coverage before
the evidence supports it. New work starts only on an explicit operator action,
with visible progress and the existing cancellation/reconnect behavior.

The primary result action and the further-investigation action must be easy to
find together on mobile. Add consistent, meaningful icons beside primary and
text actions in the commit drafting surface, retaining visible text and
accessible names. Explain the distinction between unavailable related-run
context and staged-diff evidence; no related run does not mean no evidence.

Deep investigation is one explicit durable Agent Manager workflow. Its compact
input references the existing GCT job and immutable subject. Typed advisory reads
supply bounded inventory and selected patches from recorded Git blob identities;
current checkout/index content cannot substitute for that evidence. Persist
bounded supplied-file receipts with the existing job owner, independently of
admission and cancellation revision claims. Validate stable-ID detailed inspection
against supplied complete patches. Inventory and representative evidence can
support a whole-selection synthesis without exhaustive inspection. Keep the
provider's existing structured-input ceiling and avoid a private multi-run orchestrator.
The continuation request binds its successful partial parent, subject and new
request key; ambiguous retries preserve that exact new request. Prior durable
job references provide fallback without copying megabytes into browser storage.

### Whole-selection synthesis correction — operator feedback 2026-10-22

A useful draft for the complete selection must not require reading every file
or every complete patch. Per-file coverage measures detailed evidence inspection,
not commit-message scope. Do not exhaust patch pages as a prerequisite to writing
a whole-selection message. Preserve truthful inspected-file accounting separately.

Give investigation a bounded inventory of the complete immutable selection and
let it choose important, representative and high-change details. Provide targeted
immutable evidence access rather than a mandatory sequential full-patch crawl.
A draft may summarize every affected theme from inventory and selected evidence
without claiming every file was read. Distinguish whole-selection synthesis,
limited synthesis with substantive unknown themes, and detailed inspection counts.
Missing full patches alone must not force a partial-message warning; actual
unsupported themes and uncertainty must remain visible. Legacy results keep their
old meaning. A continuation must improve the previous draft using new evidence
or clearly explain why investigation could not progress, rather than silently
presenting an unchanged sampled result as a successful deeper investigation.

Acceptance: trace the reported retained execution, prove whole-selection input
and selective immutable reads for a large selection, preserve honest inspection
counts, support legacy/retry/cancel behavior, and validate actual provider/tool
boundaries as well as drawer presentation. Preserve staging and existing drafts.

### Scope wording and workflow identity correction — 2026-10-22

Use a neutral "Staged selection: N files" heading for the pinned subject. Only
an explicitly supported whole-selection draft may claim to summarize the whole
selection. A LIMITED result after an evidence access failure must explain that
investigation made no progress and preserve the previous draft.

Verify the real workflow child identity end to end: workflow-created run,
canonical identity issuance, CLI transport, provenance verification and exact
GCT evidence binding. Workflow ownership coordinates must come from verified
issuer metadata, not caller headers or a prompt-supplied execution ID. Validate
the cross-owner issuance path, not only a fabricated verified-claims fixture.

### Completed child and workflow acceptance correction — 2026-10-22

A completed agent child is not yet an accepted message. Validate the complete
path through workflow budget settlement, structured-result binding and GCT
normalization before declaring the feature fixed. Replay retained production
failure data through these owners; browser fixtures prove presentation only.
Keep genuine budget failures authoritative and visible, including when the
structured terminal reason has no free-text message. Display the failure before
the preserved previous draft, with an explicit fallback if no cause was supplied.

Size the bounded investigation for cumulative model usage across tool calls,
not context-window size. Preserve time, charge, call and child limits. Selected
context source IDs and immutable file IDs are distinct namespaces; the prompt
and result schema must explain that source_ids is empty when no related context
was selected. Keep strict result validation and truthful evidence receipts.

### Durable execution and interfaces

Use typed protobuf/Connect contracts for browser and inter-scenario operations.
Reuse/extend the existing advisory-job owner rather than creating competing job
or proposal persistence. Persist exact input identity, idempotency key, operation
kind, execution handle, status and structured result. Bind selected context to
source revisions. Persist admission before dispatch and reconcile after restart.

Use Agent Manager's existing configured runners and generic workflow execution
with schema-validated title/body/trailers/coverage/source output. No new provider
SDK or API-key configuration belongs in this delivery. Preserve canonical caller
authority and explicitly read-only tool/workspace restrictions. Reuse stable
Start/Get/Wait/Cancel/result operations; never parse a transcript for completion.
Cancellation and result adoption require terminal-state/race guards. A provider
outage/authentication failure remains explicit and must not trigger another run.

The backend lane owns GCT domain/transport/schema/generated contracts and startup
wiring. The Agent Manager lane owns context/execution adapters, workflow assets
and necessary AM fixes. The UI lane owns feature hooks/composer/shared message
formatting and App integration. Publish narrow shared interfaces before coding
across these boundaries. Do not add independent REST writers or duplicate hooks.

### Validation, review and cleanup

Required focused evidence includes staged-versus-working bytes, partial matches,
mtime missing/refresh cases, bounded context and omissions, deterministic no-agent
collection, durable idempotent dispatch, cancel/result races, restart/reconnect,
structured-result rejection, stale selection, human-edit preservation, multiline
conventional messages, unknown trailers and mobile keyboard/safe-area behavior.
Use fake/replay providers for deterministic tests. Any live model smoke must be
tiny, bounded, use the supported execution owner and preserve real repository
content/index. Do not provision credentials or launch broad paid evaluations.

Run the generated owner verifier and governed scoped generation when needed;
focused API/CLI/UI tests; UI lint/type-check/build; applicable Test Genie phases;
and desktop/mobile observer checks against the approved interaction. Use routed
test storage for browser mutations and wait once on server-owned test runs.

After functional validation, dedicate independent review to refactoring and
related debt. Remove superseded message paths, duplicated serialization/state,
unnecessary abstractions and dead code; retain protective checks. Revalidate
affected behavior after changes. Completion requires the approved flow, passing
focused evidence, truthful scoped suite results, independent review and a final
checkpoint/work record. Compilation, screenshots or a single passing fixture
alone do not establish acceptance.

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

## Mobile safe-area repair acceptance — 2026-10-15

The operator reports excessive space below bottom navigation, clipped icons,
bottom canvas color disagreement, and drawers failing to take ownership of
the home-indicator area. Web Console is the reference for viewport ownership.
This is a bounded repair, not a workspace redesign. Existing unrelated working
tree changes must be retained. Scope includes GCT and the shared library and
BAS/ui-health evidence owners where the defect crosses those boundaries.

- The shell fills its allocated viewport. Its header, bounded content scroller,
  and nonshrinking navigation are structural rows. Navigation controls remain
  entirely above the effective bottom inset, reserved exactly once.
- Bottom clearance accounts for the actual overlap of the reported app frame
  with the native safe area. It must not assume every gap equals the status-bar
  height. Navigation and overlays consume the same viewport contract.
- The page canvas matches the resting bottom navigation. An open sheet/drawer
  paints through the bottom edge; closing it restores the resting canvas.
- Evidence distinguishes a zero-inset browser, an emulated notched viewport,
  and a physical installed app. Captures must retain actual observed insets,
  geometry and screenshots together. No emulation is called real iPhone proof.
- Regression assertions must detect absent insets, clipped controls, duplicate
  inset spacing, bottom paint disagreement and drawer open/close transitions.
  A screenshot alone or a nav outer rectangle alone is insufficient.

Work handoff: work until this contract is implemented and its focused regressions
and relevant Test Genie phases have observed results. Use scientific-debugging
and docs/TESTING.md; checkpoint in docs/internal/PROBLEMS.md. Fix understood
blocking owner defects within this boundary; retain unrelated issues for triage.
Blocked means a missing decision, credential or approval; name it. Keep going
through friction, stepping back when an experiment stops providing information.
Before handoff, review the boundary twice for correctness and unnecessary code.
Report changed, verified and unverified behavior without claiming device proof.
