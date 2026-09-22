# Browser rehabilitation progress

This file owns the current checkpoint and append-only execution history for the
file-based continuous goal. Follow [TESTING.md](TESTING.md); use
[PROBLEMS.md](../PROBLEMS.md) as the only defect register. There is no active plan,
phase progression or external execution-log dependency. Maintain this small
current-state section, then append dated records without erasing prior evidence.

## Current checkpoint — implementation, 2026-09-22 UTC

Active unlimited continuous goal BAS-FB-008. File tracking only; no plans or
external journal. Green checks trigger fresh investigation, never completion.

- Live BAS through040 healthy2026-09-22T18:16:54.021829+00:00, build
  9c7c5eef94430e9f37d9a5b9bb14fb7fb01710053704586fdcc447d480651df7;
  API17116/driver24485/UI21794. Fresh18:12:42 gate2885historyrows/five exact
  unchanged historicalRUNNING,0sessions/0recordings. Managed start76820 consumed;
  profile33967 consumed. APIprofile metadata equals039; three complete profile
  reads preserve original rollback identity. No pending lifecycle operation.
-033 RF002/RF022 recording delivery is deployed and qualified for named cases. Browser stable UUID and matching delivery acknowledgement; one bounded
  pending queue, one driver entry/delivery/receipt owner; retained failed stop,
  reset and close; explicit pull ACK only after Go journal commit. One tracked
  page/frame activation owner uses WindowProxy control acknowledgements and joins
  activation before stop. Default Rebrowser mode preserved.
-033 native14/14, activation23/23, owner29/29, affected Go races/types/API+UI+driver
  builds pass. Scoped UnitHealthuh-20260922-145457-7e793a0764f101413651f7c3ea6d386d
  passed393.839s:1549tests/125suites pass,2tests/1suite skipped. Tool27922 consumed.
  Shared RF069 CLI timeout fixed using existing authenticated long-RPC helper;
  actual delayed-response regression/all CLI race packages pass. Runtime0line/
  function/cyclomatic change at shared owner; installed CLI autorebuilt.
-033 runtime22files12922->12286lines(-636). Go7files5567/270/847 ->5572/273/851
  lines/functions/cyclomatic (+5/+3/+4); BAS cumulative-242 plus shared-12 gives
  affectednet-254. Browser queues2->1, driver maps4->1, callback circuit/destructive
  read and duplicate page-activation policies deleted. No unsupported TS complexity
  claim. Matched10000-entry read+serialization median4000.795->3917.874us, within
  noise; no speed claim. Initial allocating implementation regression corrected.
-034 RF068 structured AI response contract is deployed. One JSON schema governs
  generation/validation; empty inputs need no provider call. Final scoped API
  33.130s/focused race1.092s/build pass. Runtime+11lines/-1function/+2Go cyclo;
  cumulative BAS-240/shared-12 => net-252. Mocked request median4562->9967ns
  supplies previously missing validation, no inference speed claim.
- Latest034 tidiness1115findings/103long/387complexity/607dup/17coupling,
  duplication debt33540(-2063expandedoriginal). Ratchets/floors unchanged.
  Full UI coverage remains28.59/30.34/65.15/28.59vs85. RF064 TS AST/JS duplication
  observation gaps persist. Last board03417pending/productfalse.
-035 RF004/RF030/RF070/RF072 deployed healthy16:32:02UTC, build
  4cbb4572edd4546c5aa0ba64bf53754fd51d659294ab109f8da33d89e677f1f9.
  Canonical Rebrowser patch installed viaSDA; frame identity/document targeting,
  context acknowledgement and new-download effects qualified in native tests.
  Full scoped driver uh-20260922-162102-ce8fafecad1445ee49c7f58ab339b894 passes
  351.446s:1549tests/124suites,2tests/1suite skipped. First failing run retained;
  mandatory mock state and delayed binding-ack handling repaired. Audio passes.
  Types/builds pass. Quiet16:28:16 gate2555historyrows/five historicalRUNNING,
  0sessions/0recordings; start92545/profile13486 consumed. Original full profile
  and API metadata preserved. No pending035operations.
-035 runtime21files5172->4724lines(-448); SDK1879->1890(+11); net-437.
  Same runtime hashes at qualification/publication. Cold main-document context
  median3924.92->2582.95us(-34.19%); warm347.40->348.58us(noise). Not a whole
  journey benchmark. No Go changes; cumulative affectedGo cyclomatic remains-252.
  Tidiness1116findings/103long/387complexity/607dup/18coupling/debt33540; one
  extra coupling finding is the expanded native integration test. Floors intact.
  Board17pending/productfalse. Full frame record/replay and broader stealth remain
  unqualified; details in frame-targeting-2026-09-22.json.
-036 RF073 confirmed: arbitrary evaluation retry repeats a committed external
  POST after navigation. Temporary native server fixture and maintained regression
  both observe2effects for1instruction; returned error remainsretryabletrue.
  Single-dispatch repair applied;82focused cases/5suites pass15.235s, types/build
  pass. Full driver uh-20260922-163345-07f0358027780d17a733b9ef0ad01760
  passed307.798s:1552tests/124suites;2tests/1suite skipped.73224 consumed.
  Tidiness20260922-163350-8274aa55 failed6s with unchanged1116findings/debt33540;
  one quiet wait consumed. Board81240 consumed:17pending/productfalse. Live035
  unchanged; no pending lifecycle operation.
-037 RF071/RF074/RF075 source qualification complete. SDA shared build
  6034b18af0b24f9b4ffc33a3a1fcb88c7c6b3f6945965338a77194764504fe2b is healthy.
  Owner API12.241s and four package races pass. SDA restart43426 consumed.
  Its approved install operations updated both BAS graphs to js-yaml4.3.2 and
  removed superseded overrides; no unrelated lock resolution changed. Native
  merge-budget probe now rejects correctly and ordinary YAML remains valid.
  Security Health20.186s passes0errors/357warnings/406infos. Exact reviewed hash
  exceptions preserve default rules and detect three counterexamples. API/UIbuild
  passes; package governance44existingwarnings. BAS published in combined038.
-038 RF072 URL download falsely fails on Chromium's expected ERR_ABORTED although
  correct attachment bytes arrive. Native red1.320s reproduced; narrow abort
  handling implemented while event/save remain mandatory. Native URL success,
  abort/no-attachment and unrelated-error controls pass. Focused33-case run had
  one evaluation fixture failure withzeroeffects (32passed); failure context
  was not printed. Added diagnostic assertion; five targeted native cases pass
  1.985s. The40-context probe found32failures and isolated immediate binding cleanup
  as a cause. One session registration/global with unique receipt tokens and
  a bounded internal absent-binding handshake now passes40effects/40next-page
  reads plus20matrixrows. Canonical SDK patch installed throughSDA; final
  focused36/4 pass22.394s; types8184 consumed. No arbitrary script retries restored.
  Types71356/driverbuild pass. Tidiness20260922-165657-106c4f59 terminalfailed,
  admission33748 and one quiet wait consumed; board90943 consumed:17pending/productfalse.
  Final full driver uh-20260922-171531-588ec4e8e9386bbcc9675231167fbaca passes:
  1559tests/124suites,2tests/1suite skipped,366.709s Jest; waiter20434 consumed.
  SecurityHealth5.060s passes0errors/357warnings/406infos; types/build pass.
  Finaltidiness20260922-171537-45b14dd7 failed, one quiet wait consumed:
  1117findings/104long/387complexity/607dup/18coupling/debt33540. Native test
  expansion adds one long-file finding; no floor/threshold relaxed.
  All six recorded source hashes unchanged before combined036–038 publication.
  Gate/stop59912, start31992 and profile84003 consumed. No pending validation.
  Full product still unqualified: durability/crash/overflow/fullrecordreplay,
  native OS/soaks, RF038mutation leases and RF055historical orphans remain open.
-039 RF038 reset lease admission implemented across GoSession/client/driver route.
  Native3red cases3.204s -> focused9/2pass4.248s with open-handle inspection;
  Go driver/session/engine races pass. Runtime3files -77lines. Owner qualifications
  driveruh-20260922-173337-68e0fa749361097f6a4aa0c9df53bc49 passes342.144s:
  1565tests/124suites,2tests/1suite skipped;38932 consumed. APIuh-20260922-173337-7982c7926e5e83a27ee25627d788697f
  passed36.661s;30665 consumed. Tidiness20260922-173343-048f3e9f failed, quiet
  wait consumed:1118/104long/387complexity/608dup/18coupling/debt32643. Whole
  dirty-scenario change; no sole039 attribution. Types/build4732 and API/UIbuild
  70677 pass. Board50728 consumed:17pending/productfalse. Live038 unchanged; no lifecycle
  published039 after fresh quiet gate; stop55720/start29526/profile86231 consumed. RF076 current-owner clean reset still fails onopaqueabout:blank and
  retains old-origin storage; do not claim storage repair. Next consume receipts
  once, retain pending IDs, then continue coherent reset ownership/storage work.
-040 RF076 clean reset repair deployed; details in reset-storage-2026-09-22.json. Context tracks imported/
  visited origins; reset retains primary page/context, clears persistent storage
  through public CDP and per-tab storage via intercepted empty origin visits.
  Manager owns in-flight reset joining, reserves phase before flush, permits
  explicit failed retry and joins before close. Route duplicate map removed.
  Native initial multi-origin red3.166s -> green3.375s; focused114/5pass6.948s,
  types pass. Expanded59-case run57pass/2fail40.85s: storage matrix timeout after
  adding cross-site iframe/cache-only cases, and close-test startup-probe baseline
  counted as reset disposal. Baseline assertion corrected and gate released in
  finally. Diagnostic12939 failed30.946s; test setup suppressed console markers.
  Visible diagnostic7594 failed30.782s and isolates setup tosetContent of a
  cross-site iframe after service-worker registration, before reset admission.
  Native fixture now serves real frame-host/frame-oracle URLs; markers removed.
  Expanded2run87066 passed59/2in6.477s. File-origin adversarial test then fails
 3.109s: WHATWG URL.origin reportsnull while Chromium exposesfile:// storage.
  Tracker now retains that explicit origin. Final focused149tests/6suites pass
  9.609s, including primary-page capture, file origins, partial retry and successful/
  failed reset-close coordination. Types/build pass; no diagnostics remain.
  Full scoped driveruh-20260922-180536-251d1c74ffe41f505e06d9d89f73826c
  passes332.571s (Jest327.966s):1573tests/124suites,2tests/1suite skipped.89109 consumed.
  Tidiness20260922-180542-e4db78b1 failed; admission26369 and one quiet wait consumed.
  Metrics unchanged1118findings/104long/387complexity/608dup/18coupling/debt32643.
  Boardprog_8ded6dad-b2fc-4e13-9b5b-2b84432c3a9b
  remains17pending/productfalse;97945 consumed. Public native oracle confirms
  stale404/missing400 preserve state; current200 clears storage and returnsready.
  Runtime5files net+41lines; combined039–040 net-36, Go cumulative-246 unchanged.
  Open-handle warning still unqualified. Live039 unchanged; no lifecycle operation.
-041 RF077 readiness lifetime repair in source, not deployed. Native losing timer1->0;
  closed-page CDP attempts40->0 and settlement2019.62->0.387ms (single discriminator).
  Normal live-page control preserved; closed readiness rejects explicitly. Focused
  100tests/6suites pass19.47s with no handle warning, types pass. Final scoped driver
  52477 consumed: owneruh-20260922-182410-0f03f688f5567b49d15aa763664bc239 failed
  84.444s on another incomplete inline Page mock (page.off absent). Reset route
  fixture now reuses createMockPage; final98/6pass13.931s. Second owner33499 failed deadline-test contamination; corrected manager fixture
  and third owner2810 passed1579tests/124suites356.381s. No pending driver check.
  Tidiness20260922-182416-a733ef44 failed; admission85469 and one
  quiet wait consumed. Types/build53057 pass. No lifecycle operation. Receiptreadiness-lifetime-
  2026-09-22.json. Runtime2files net-13lines; Go cumulative-246 unchanged.
- OriginalHEAD7d1c7531d057c061c5ad5a67fbecb97ebe2eb194; isolated121probes33met/88failed;
  expandedoriginal debt35603. Detailed prior receipts and measurements remain in
  history and internal/evidence/rehabilitation/. RF061 duplicateRF023 is retired.

## Record format

Use a stable record ID and timestamp. Keep a record proportional to the work.

```text
Record/date; BAS-RF, journey and outcome IDs:
Question/hypothesis and independent expected behavior:
Source/build identity, concurrent-tree limits and affected owner/paths:
Decision or boundary extension and rationale:
Changes; converted callers; removed policies/code/adapters:
Checks, operation IDs, receipt/artifact refs and interpretation:
Comparable before -> after: performance, complexity, duplication, coupling:
Baseline -> cumulative delta; useful behavior preserved:
Rejected hypotheses and remaining uncertainty:
Unavailable check: cause, attempts, evidence, limitation, next-check trigger:
Pending operations and next useful action/review angle:
Feedback addressed with receipt:
```

For a review, record new cases and adversarial questions, concrete observations,
new BAS-RF IDs and their dispositions. Two clean reviews do not close this goal.
If runtime time is short, update the current checkpoint before starting another
operation. Keep the operation identity when work outlives the current turn.

## History

### BAS-WORK-000 — 2026-09-22 UTC — preparation correction

The operator rejected the previous work shape. The mistakenly created
`bas-browser-rehabilitation` record (948897c4-bc18-4426-b798-f8bd2d25421e) was
archived through its owner; it is historical and must not be resumed. The
supported owner operation is soft archive, so an audit record still exists.
No execution was started. No other historical BAS plan was changed.

The active target, scope, goal, feedback, checkpoints and proof requirements
now live in BAS files. Continuous investigation replaces automatic completion
after two clean reviews. Complexity, duplication, ownership clarity and removal
of obsolete code are first-class outcomes alongside browser correctness and UX.
The operator's unavailable-validation rule is retained. Existing production
source changes and original baseline observations are preserved.

Correction receipts: eight preparation regression tests and the contract
consistency check passed; goal length is 2,044 characters; current handoff links,
local evidence hash and archived plan status were verified. Test Genie
20260922-030931-9015f688 passed skill-set and failed docs. Correcting 19 preparation
manifest entries and registering four artifacts reduced documentation errors
from 42 to 4 in docs rerun 20260922-031112-03ee86df. Remaining errors and reference
warnings are retained under BAS-RF-050, with 389 warnings and 20 infos in that
rerun. This is a failing docs result, not unavailable validation or a product
pass. The direct preparation checks are narrower and passed.

Feedback BAS-FB-001 through BAS-FB-007 is resolved for preparation wording and
tracking with references in OPERATOR_FEEDBACK.md. Product repairs, performance
improvements and cleanup remain implementation work; none is claimed complete.

### BAS-WORK-001 — 2026-09-22 UTC — profile save receipts and recovery (in progress)

W3 localized repair, BAS-RF-003 / J06 / profile-durability. Prior turn was
preparation progress, not a running validation wait; no pending operation was
recorded. Current HEAD is `7d1c7531d057c061c5ad5a67fbecb97ebe2eb194` in a shared
dirty tree. Recall found the retained profile investigation; its current probe
reproduces all seven profile failures (plus unrelated replay failures).

Hypotheses: (1) handler swallowing Save errors explains false persisted/closed
receipts; (2) repository corruption alone explains them. Injected SaveErr reproduces
HTTP 200 and association loss without disk I/O, confirming (1). Repository
faults independently reproduce RF-025/027 and remain follow-up work.

Decision: use the existing aggregate PersistSessionState owner for manual saves
and close; capture both storage and tabs before writing, propagate every failure,
and retain the browser plus association until a successful save and close.
This removes duplicate handler snapshot/save policy. No scope extension or data
format conversion. File-repository crash atomicity and overlapping aggregate
updates are still open; this change must not claim to fix them.

Validation intended: maintained HTTP regressions (save/capture/close failures,
retry, empty tabs, combined snapshot); profile/service regressions; scoped Test
Genie unit/workflow/tidiness; contract, inventory and rehabilitation board.

RF-003 caller review: RecordModeView ignored HTTP errors and navigated away on
network failure, with duplicate close code in workflow completion. Extended the
same repair within BAS to both paths, retained the workspace on failure, admitted
one pending close, and removed forced full-page reload after router navigation.
Five of six UI behavioral tests fail before the repair; the ordinary success
control passes. Initial test setup exposed different router majors in the shared
render helper and BAS; the test uses the helper's supported withoutRouter option
and the BAS router. No dependency change or production adapter was introduced.
Relevant source changed while the broad owner run was active; direct tests will
verify final inputs, and that owner's receipt cannot imply an isolated snapshot.

### BAS-WORK-002 — 2026-09-22 UTC — fail-closed profile recovery (in progress)

Feedback reread; W3 / BAS-RF-027 / J01,J14 / profile-durability. The current
retained probe shows missing protected bytes restored as empty state and a wrong
key producing an empty listing plus a newly created default identity. Hypothesis:
swallowed repository read failures cause the replacement, not default selection
ordering. Fault the real file-backed service with missing/corrupt state and a
wrong key; Get/List/default selection must fail visibly, leave bytes unchanged,
and recover the original identity when the fault clears.

Remove the missing-state success branch and skip-unreadable listing policy.
Do not infer intentional empty state from missing ciphertext: Create always
commits encrypted state, including an empty state. Recovery reports the affected
profile identifier and an error, never authenticated contents. Native/browser
recovery and a richer per-profile metadata recovery UI remain unverified.

BAS-WORK-001/002 completed scoped checks: handler 11-case regression and focused
Go package checks passed, handler race check passed, UI 6-case regression passed,
UI ESLint/typecheck passed, all profile/persistence/session-profile RPC package
tests passed under `-race`. Contract preparation remains valid, not qualified.
Board receipts `prog_d6665b51-2d4d-422c-bba7-ab4c6fdbf941` and
`prog_9d792154-251b-4c9b-956c-abfc733a660f` both read 0/17 qualified outcomes.
Detailed reproducible commands, failure/success output, source hashes, before/after
probes and inventories: [profile boundary evidence](evidence/rehabilitation/profile-boundaries-2026-09-22.json).

Comparable deltas: server snapshot/save implementations 2 → 1, client close
implementations 2 → 1, aggregate commits per snapshot 2 → 1, swallowed capture/save
errors 5 → 0 across the repaired persist/close paths. Handler cyclomatic sum
184 → 170 over the same 23 functions; close 16 → 6; persist helper 10 → 6.
Repository function sum 75 → 73 over 14 functions; missing-state success and
unreadable-list skip policies removed. No new runtime module or import dependency.
Runtime source: API 94,165 → 94,101 lines (-64), UI 131,532 → 131,518 (-14);
Go tests +228 lines, UI test +93 lines. These are substantive local reductions,
not a claim of completed domain-wide rehabilitation. Broader duplication/coupling
remain owed by the active tidiness receipt. Record-mode.go remains >500 lines;
its unrelated handlers are still a cohesion hotspot rather than excused by this
repair. Existing tests and budgets were not weakened.

Retained profile/replay checks: 3/18 → 7/18 expected behaviors; 11 still fail,
none unavailable. The missing-state probe now permits a nil returned profile on
error instead of dereferencing it; its expected behavior is unchanged. Original
baseline artifacts remain immutable. File-backed recovery tests restore synthetic
bytes only, never user profiles. Native key provisioning, metadata recovery UX,
crash persistence and full browser journeys remain unverified; recheck when their
producer/owner work changes. Feedback BAS-FB-008 remains active.

### BAS-WORK-003 — 2026-09-22 UTC — execution sink lifetime (in progress)

Feedback reread. W3 / BAS-RF-021 / J07,J18 / resource-budget and evidence-completeness.
Hypotheses: the UX decorator hides lifecycle cleanup (24 retained queue workers
in the original composition probe); independently, queue close discards accepted
terminal events (1/2 delivered when the downstream hub is held). Current source
retains both defects. A healthy-state response cannot disprove either.

Decision: make existing CloseExecution part of the sink contract, require wrapper
propagation and defer it from workflow ownership on every exit path. Drain accepted
events before hub close, reject late publishes and atomically check closed state
when admitting queues. No new service/framework or compatibility fallback.
The hub's contextless blocking broadcast remains a separate shutdown/backpressure
limitation; these changes do not by themselves qualify bounded shutdown against a
permanently stopped hub or attribute the entire historical 10 GiB heap.

Scoped repair verified: events, collector and workflow packages pass `-race`.
Maintained tests hold the hub while accepting terminal events (three terminal
kinds), exercise completed/failed/cancelled/compile-failed workflow exits, preserve
another execution's buffers, and reject late events before UX persistence. The
late-event test first failed on persisted evidence, then passed after delegate
admission moved before collection. Retained probes: leaked workers 24 → 0, hub
close receipts 0 → 24, accepted terminal deliveries 1/2 → 2/2. Recording API
probe expectations 1/9 → 3/9; the six remaining failures are still reported.
Standalone execution probe sink implements the now-required close contract.

Lifecycle owner: workflow → Sink contract → decorator → queue worker → hub.
Removed the concrete type assertion, separate closed-state check and queue-clear
drop path. No compatibility fallback or new runtime module. API runtime +3 lines
for required lifecycle/rejection semantics, tests +197 lines. Cyclomatic sum over
53 affected functions 317 → 319; this slice improves ownership/correctness, not
branch count. Cumulative touched Go sum is down 14 across cycles 001–003. Whole
API runtime 94,165 → 94,104 (-61), UI -14. The workflow orchestration functions
remain complexity 56/29 and >500-line cohesion debt; no wrapper extraction credit.
Evidence: [event lifetime receipt](evidence/rehabilitation/event-lifetime-2026-09-22.json).

Owner validation `20260922-032049-4b5c680b` returned failed after one wait. Full
findings artifact `artifact_b401e30fb1b458c052a049e71f8a48f2` identifies actual
program assertions (4 failures/7 errors), video.webm detected as audio/webm,
UI coverage 28.42% statements/30.24% functions/64.92% branches/28.42% lines
against unchanged 85%, and a CLI companion reimplementation. These are repair
work under RF-014, not unavailable checks. Workflow provider was recovered via
`workflow-health validate get ec1304ed-fef0-4f37-91ff-85eeb3bceae2 --json` and is
canceled, cancellation_requested=true at the 900s deadline. No executable browser
result exists in that receipt. Recheck after provider execution diagnosis, not
an identical immediate retry. Host swap-pressure warning is retained; do not
infer the assertion failures from host pressure.

Tidiness same 1,104 findings, duplication 35,603 → 35,585 (-18), still above
35,302 budget. Relevant sources changed during this run, so no isolated final
snapshot claim. Board cycle 003 succeeds but still has 17/17 pending outcomes.
Feedback BAS-FB-008 remains active. Full-source compile initially overlapped a
test import edit and failed importing errors; package tests subsequently pass.
Final stable-input compile is pending; retain both observations.

### BAS-WORK-004 — 2026-09-22 UTC — unit-receipt failures (in progress)

Feedback reread. RF-014 / evidence-completeness and known-flow-reliability.
Fix the owner-reported assertions without reducing coverage floors or skipping
tests. First hypothesis: host MIME registration takes precedence over the
export source's known video formats; canonical MP4/WebM types must be stable
across hosts. Then diagnose the governed workflow-program failures from their
full focused output. Shared tree remains dirty; preserve unrelated work.

Results: the 45 Python program cases pass from the root but fail 4 assertions
and 7 errors through the Go launcher. Root cause is lexical parent traversal
before resolving `..` in `__file__`, pointing the real kernel import at the wrong
directory. Resolving that path restores all 45 cases through the actual launcher;
no product verifier or expected result changed. WebM canonical video typing now
precedes OS registrations; tests force audio/webm registration and retain mixed
case/empty-path cases. The recorded-video handler uses that existing owner, deleting
its duplicate sniffing policy. Generic mixed-media stores retain normal detection.

Adversarial caller review found `executeResumedWorkflowAsync` constructing another
unclosed sink. Expanded the existing lifecycle test to fresh/resumed × four exits;
all resumed cases failed before repair. Resumed execution now defers cleanup and
uses the same existing `executionOutcome` policy as fresh execution, deleting its
cancel-substring/failed-status branch. Race checks for workflow, export source,
workflow handlers and all handlers pass; eight lifecycle cases pass. This amends
the previous regular-execution-only scope of BAS-WORK-003.

CLI unit finding was a real duplicate helper. Both parity callers now import
`cli-core/cliapptest.ReadManifest`; deleted `cli/internal/testutil/manifest.go`
(20 lines), no shim or dependency addition. Entitlement/workflows/testutil package
tests pass. Stable final API compilation passed after the overlapping-import
observation from cycle 003. Durable proof: [unit boundary receipt](evidence/rehabilitation/unit-boundaries-2026-09-22.json).

Comparable cyclomatic sum over 24 functions 221 → 215; policy owners reduced
for video export and execution outcome. API runtime -20 this cycle, cumulative
94,165 → 94,084 (-81), UI -14. Go tests +45 net including deletion of the CLI
helper. No source budgets or coverage floors changed. Touched export handler's
64/20-complexity functions remain substantial debt; this small repair does not
establish structural qualification. Board `prog_0a0625c8-06cf-42e6-8a8b-fd437f63091b`
completed with 17 pending outcomes. Full unit/tidiness rerun admitted after these
changes; no workflow rerun while its unchanged deadline cause is unresolved.

Final-source owner run `20260922-035031-d39d8044` completed in 100s and was
attached once. Go command failure and companion reimplementation no longer
appear. The remaining unit command failure is the UI coverage floor: statements
28.57%, functions 30.34%, branches 64.96%, lines 28.57% against unchanged 85%.
Tidiness reports 1,105 findings (1 error, 236 warnings, 868 infos), selected
ratchet failure long_files=82 against recorded 81. Its long-file location set
matches the preceding run; no new location in this cycle. Final-source metrics
must be read from that owner, not inferred from the earlier duplication total.
Findings artifact: `artifact_d1c395176a0b65936557555c29869976`.

### BAS-WORK-005 — 2026-09-22 UTC — atomic profile commit (design/investigation)

Feedback reread. RF-025 / J06,J14 / profile-durability. The two-file writer
cannot implement one atomic aggregate commit: protected bytes are destructive
before metadata rename. Consider one encrypted profile envelope, keeping the
same repository contract and public profile shape. Reuse api-core's atomic-file
writer instead of a second temporary-file implementation. Reject runtime
conversion/fallback and keep any required personal-data conversion outside the
repo with byte-preserving rollback and synthetic round-trip/fault proof.

Storage-steer and required portability guidance read; storage-manager validation
retained at /tmp/bas-storage-before-005.json (failed, existing direct-writer and
isolation findings). No engine/dependency addition is justified for this bounded
profile aggregate. Operator explicitly authorizes offline data preservation and
forbids runtime migrations; file-only tracking overrides visited-tracker logging.
First inventory actual storage/key availability without exposing credentials;
then decide conversion and publication boundary before changing the runtime format.

Investigation: canonical storage resolver reports one `.json` profile with
plaintext storage at `~/.vrooli/data/vrooli/browser-automation-studio/session-profiles`.
No `.protected` file exists. The live process lacks BAS_SESSION_STORE_KEY.
Metadata-only credential status shows the encrypted-file authority available and
`vrooli/browser-automation-studio:session-profile-keyring` unconfigured. No user
state was modified or secret printed. Therefore format work is a candidate only
until offline conversion and key recovery are verified; do not restart yet.

Candidate single-document implementation now passes profile/handler race tests,
including rejected write/rename preserving the prior snapshot, one protected
commit document, 24 concurrent complete saves with no mixed generations, foreign
identity refusal and old-format read refusal without mutation. Shared atomic
writer fault tests pass (write/sync/close/chmod/rename). Retained profile expectations
7/18 → 9/18; concurrent field updates and eight replay cases still fail. The
staging fault probe now targets the atomic writer seam, with actual file mechanics
covered by api-core tests; original dated fault evidence is unchanged. Existing
recovery fixtures now remove the envelope's sealed field rather than an obsolete
sidecar. Production crypto still uses the old environment key at this checkpoint.

Necessary adjacent boundary: RF-012 key provisioning/recovery, using the existing
credential-authority-go owner (no scenario key store or environment injection).
Declare a generated destructive encryption-ring credential in BAS's manifest,
resolve through the authority with a data-owned loss witness, and retain old
versions for reads. Dependency gateway preview approves the existing repository
module; apply through that gateway. This extension remains within BAS and its
supported shared credential boundary, with authority-fault/missing-key/rotation
tests. No real-account action or credential-loss override is authorized or needed.

Credential implementation removes BAS_SESSION_STORE_KEY and implicit `.test` keys.
Explicit synthetic authority injection keeps host credentials out of test runs.
Rotation retains old versions; missing active/historical keys, malformed rings,
provider failure, and credential loss with profile-only/witness-only/history all
fail visibly. Persistent history is restored when data and keys are recovered
without their witness. `go test -race` passes profile, handlers and all testutil
packages; full API compile passes. Dependency approved validation passed with
preexisting governance warnings; security-health dependency status is available
(index_ready=true, ecosystem vulnerability counts are not a BAS security verdict).

Offline personal helper `/tmp/browser-automation-studio/migrate-profile.go` passes
synthetic encrypted roundtrip, byte-identical rollback after publication failure,
changed-source refusal and unknown-field refusal. It accepts only the inspected
plaintext format, fails on unknown data, and never enters runtime source. Actual
staging uses the platform authority without printing values. Driver health reports
zero sessions before this operation; API healthy. Original live data remains
unchanged pending staged verification and lifecycle stop.

Publication completed after `make stop` returned success. One profile preserved
field-for-field through fresh repository reads; old bytes remain in
`~/.vrooli/data/vrooli/browser-automation-studio/session-profiles.rollback-20260922`.
`make build` passed. `make start` operation `startop-61bbc2e53728c37501c85b5678b795e7`
is progressing through normal dependency freshness and setup (session 62836).
The sampled 1,000 recent executions were terminal; driver had zero live sessions.
No saved profile was retired and no browser was navigated to a real account.

Final source cleanup removes obsolete mock Rename/WriteFile methods as well as
the old production writer. Final race checks pass. Changed-file union Go cyclomatic
sum vs HEAD: 147 functions / 853 → 147 / 840 (-13); this includes new credential
validation and the removed unused mock paths. Source inventory digest
`b21afc9aa40d50cd14cf4cbd56598760e39dba570f8d92e697bac1658d167e8e`; API runtime
94,102 (-63 from original 94,165), UI 131,518 (-14); this cycle API +18 against
BAS-WORK-004. Source size is not a performance claim. Shared worktree caveat applies.
Board `prog_1f53d16a-40b5-41d7-96bf-1c60c1080cc2`: status ok, 17 pending telemetry,
product_qualified=false. Contract preparation check passes. Retained profile
expectations remain 9/18 met (field-update loss and eight replay failures remain).
Test Genie `20260922-042038-b5b05137` unit/tidiness admitted once; attached once
with its recommended 175s timeout. Do not re-admit or poll while wait is active.

Lifecycle restart completed healthy, build identity
`sha256:ad61ab7fb5444f3de0c8143db5918afef1755e3b8bb53951551e1d569b2f52ae`.
Live profile metadata API returns exactly the original identity and name, omits
protected storage, and reports one profile. Repository comparison preserves every
field. Observed fresh-process credential/read latency was 1,230ms cold, then 0.40ms
and 0.30ms warm; small shared-host samples, not a latency qualification or baseline
comparison. Off-host recovery and real-browser sign-in continuity remain unverified.

Owner run `20260922-042038-b5b05137` terminal failed in 97s: API/CLI command failures
absent; UI coverage command still below 85% (owner reports 28.6% workspace coverage).
Tidiness: 1,109 findings, 1 error / 239 warnings / 869 infos; long_files=82 versus
baseline81. Four added HIGH_COMPLEXITY findings are manual assertion boilerplate
in the new fault tests. Replaced repeated error branches with the existing testify
require library, retaining/strengthening all fault expectations; focused race test
passes. This removes assertion boilerplate without changing production policy or
thresholds. A tidiness-only rerun is justified by that changed test source.

Final tidiness-only rerun `20260922-042610-4dbb9896` completed inline in 4s, failed
on unchanged long-file ratchet (82 vs81). Findings returned to 1,105: 1 error,
236 warnings, 868 infos. All four newly introduced test-complexity findings are
removed, with every fault case retained. No owner operation is pending. Final
BAS-WORK-005 evidence: `evidence/rehabilitation/profile-atomicity-2026-09-22.json`.

### BAS-WORK-006 — 2026-09-22 UTC — concurrent profile field updates (investigation)

Feedback reread. RF-026 / J01,J06,J14. Current probe still proves successful tab
and storage updates can lose the storage update through a legal Get/Save
interleaving. Candidate cause is the service's repeated read/modify/full-save
policy, independently of now-atomic document publication. Write a deterministic
real-file interleaving regression across separate services/repository instances.
Expected: both successful updates survive, and delete cannot be undone by a
stale field update.

Recall found existing profile evidence and the shared platform-go native file
lock owner (Unix flock, Windows LockFileEx, bounded/cancellable wait). Consider
a repository-owned atomic Update operation using one stable, bounded store lock;
all Create/Save/Delete writers must honor it. Convert every service mutator and
handler-owned cookie/tab read-modify-save caller; do not add a mutex only around
the two reproduced methods. Keep whole-snapshot replacement distinct from field
mutation. Remove repeated Get/Save policy and centralize timestamp/error handling
without adding a runtime data migration. No new implementation is committed to
yet; lock wait budget, nil/not-found semantics, caller cancellation and all
writer coverage must be settled from tests and owner interfaces.

Deterministic real-file regression is red in both cases: separate service/repository
instances lose the acknowledged storage update after a blocked tab read, and a
stale tab save resurrects an acknowledged deletion. `concurrency_test.go` uses
channels at the read/publication/native-lock seams, not sleeps for ordering.
Red output: `/tmp/bas-concurrent-red-006.txt`; no user data involved. The selected
boundary is documented in ARCHITECTURE.md before implementation. Initial native
lock admission is bounded to five seconds; the existing context-free service API
still does not propagate caller cancellation.

Candidate repository Update now owns read/modify/publication under a stable
`.profiles.lock`; Create, explicit Save and Delete participate. It reuses
platform-go native locking with five-second admission timeout. All service field
mutators now use the transaction; repeated Get/Save and duplicate session-state
application are removed. The two deterministic real-file regressions now pass
under `-race`, as do existing profile tests. Source is not deployed yet (live
binary remains BAS-WORK-005).

Recording cookie/localStorage and individual-tab handlers have been converted
to mutate within the service transaction; storage mutation retains its LastUsedAt
policy through UpdateStorageState. Handler tests/race checks are in progress
(session 46084, `/tmp/bas-update-boundaries-006.txt`). Remaining proof: handler
interleavings, rejected callback/write/lock invariants, retained probes, full API
compile, debt inventory/board and scoped Test Genie. Caller cancellation and
exclusive browser-session ownership remain broader gaps, not claimed repaired
by serializing a persisted field update.

All profile, recording, session-profile handler and general handler race suites
pass. One intermediate failure was the recording receipt test's counter still
overriding the removed Save interface; it now counts Update commits with the
same one-snapshot/one-commit expectation. New handler tests prove cookie and tab
edits retain a competing committed addition; Go overlay against the preceding
handler source fails both cases without changing the shared worktree. Repository
tests also preserve bytes after rejected callback, identity mutation, lock admission
and commit failures. Retained expectations: 10/18 met, only eight replay cases fail.

Changed-file union vs HEAD: 220 functions / cyclomatic sum 1,105 → 229 / 1,088
(-17). New ownership operations add functions while removing repeated update
policy; this is not counted as a function-count reduction. Current API runtime
94,111 (+9 this cycle, -54 from original); UI remains -14. Source digest
`9c6ba30e654ab865ea50dbf8c21ed0d4bc5221ca681a90ae459fb6284f114997`.
Board `prog_00ae62a1-beb5-4993-a191-77fd22812feb` succeeded with 17 pending rows
and product_qualified=false. Test Genie `20260922-043957-3e0e9df4` admitted once,
175s quiet wait attached once. Build is running; no deployment claim for cycle006.

BAS-WORK-006 deployed through successful make build/restart after observing zero
live driver sessions/recordings. Health build identity
`sha256:010f4c391123888043b276ec2756d5c45f75e826d15f6efe00e2b92f8fbd4b0a`.
Original metadata identity/name and all underlying profile fields remain equal
to the preserved rollback copy. Owner run terminal failed in 97s: only UI coverage
command failure, tidiness 1,105 findings (1 error, 236 warnings, 868 infos), same
long-file ratchet82 vs81. One service duplication finding removed; one five-line
channel-wait duplication finding added in the deterministic concurrency test.
No threshold or assertion changed to improve the score. Evidence:
`evidence/rehabilitation/profile-concurrency-2026-09-22.json`. No operation pending.

### BAS-WORK-007 — 2026-09-22 UTC — opaque authentication preservation (investigation)

New adversarial angle after profile transaction proof: targeted cookie deletion
round-trips the whole storage object through a partial typed projection. A synthetic
HTTP fixture proves the operation acknowledges success while dropping unrelated
origin IndexedDB data and another cookie's partitionKey. No live profile was edited.
Go overlay keeps this initial experiment out of the stable cycle006 source.
Red proof: `/tmp/bas-profile-preservation-red-007.txt`; overlay source
`/tmp/browser-automation-studio/profile-preservation-adversarial-007_test.go`.
Expected: change only the requested cookie/localStorage subtree and preserve all
other authentication fields, including unsupported future fields; preserve visible
errors for malformed relevant state. Register RF-051 and repair the projection
ownership before claiming this storage-edit journey reliable.

RF-051 candidate now preserves raw JSON outside the targeted subtree. Cookie
deletion preserves partitionKey, IndexedDB and unknown fields; all/origin/item
localStorage edits retain IndexedDB and origins containing opaque state. Large
integers stay raw (no float64 conversion); malformed targeted arrays/items reject
without replacing the snapshot. Focused race tests pass. Original narrower
structs remain only as test assertion decoders, not a runtime serializer.

Boundary cleanup found the exported FileRepository.Save and MockRepository.Save
entrypoints now have no runtime callers: the repository interface and all service
writers use Create/Update/Delete. Remove the obsolete unconditional replacement
path and exercise atomic snapshots through the actual transaction API. Delete two
obsolete mock-temporary-filename tests superseded by real-file generation/failure
regressions and shared atomic-writer fault tests; preserve every useful behavior
assertion. This finishes the previous replacement, not a new compatibility layer.

RF-051 candidate verification: cookie deletion retains partitionKey and origin
IndexedDB; all/origin/item localStorage edits retain unrelated cookies, IndexedDB,
opaque origins and fields, including exact integers beyond float64 precision.
Malformed cookie/origin arrays and null/invalid selected items reject without
changing saved bytes. Existing recording/profile/service race checks pass; all
API packages compile. Native callback transaction tests pass after replacing the
removed unconditional Save calls with Create or Update. An intermediate fixture
mechanically attempted Create for its 24 updates; corrected to Update, preserving
the all-writes-succeed/complete-snapshot expectation.

Current source digest `da4e0c43391b145aaffc11ae5c7b593c22548a522ea230189c54387aa4715e69`.
Runtime API94,115 (+4 cycle007, -50 original), UI131,518 (-14 original). Changed-file
Go union: 220 functions/1,105 cyclomatic total before → 231/1,099 after (-6);
new preservation/error branches are included. editOriginLocalStorage is 11, so
expect a new complexity warning rather than suppressing or fragmenting its logic
for a score. Full owner unit/tidiness and make build are running. Do not deploy
source until their concrete results are retained; existing profile/rollback remain
unchanged.

Pending owner run is `20260922-045614-b2c1b7c5` (unit/tidiness), with one quiet
175s wait attached. Build process session40744 is also pending. Board
`prog_d04a7899-d6f3-4d3e-a3b4-0a721df5a8c4` succeeded, 17 pending telemetry rows,
product_qualified=false. Contract preparation passes; retained profile probe
remains 10/18 met, with eight replay failures. Next after cycle007 publication:
RF-011 driver snapshot still calls storageState() without its IndexedDB option;
inspect installed binding and prove synthetic browser continuity before changing it.
Driver manager.ts already carries unrelated changes; preserve them.

Cycle007 owner run `20260922-045614-b2c1b7c5` terminal failed in137s. Unit failure
is still UI coverage; selected tidiness ratchet now duplication_line_debt35,311
versus35,303 baseline (+8). Full build passed. Restart was initiated after a
health read showed seven driver sessions and zero active recordings. This was
an operator error by the agent: the restart should have waited for session
activity classification/drain. Preserve this fact; investigate execution receipts
for interruption, do not invent an all-idle or no-impact claim. Restart process
session67309 is active. No user-profile mutation was intended.

Restart mitigation: runtime was still on cycle006 while dependency checks ran.
Sent SIGINT then SIGTERM to this agent's restart initiator PID1060384 before the
stop step. The command exited143 (make exited2). API build identity remained
cycle006 and driver uptime increased continuously (1,019,446 →1,092,041ms);
sessions naturally drained10 →2 →0. No BAS target restart interruption was
observed. Cycle007 is NOT deployed. Further deployment must classify/drain active
sessions first and recheck immediately before admission. Preserve the admission
error and mitigation; do not rewrite it as a clean deployment.

Cycle007 deployment completed after an immediate zero-session/zero-recording and
terminal-execution gate, explicit make stop, then make start. API PID was not
changed by the canceled first attempt. Successful deployment build identity
`sha256:75fc64c4ec22e76c9c3332ec4d407531580b6ae2c3b7f028cb783e23ff04c1d8`; API
healthy, original profile identity/name and all stored fields preserved. No owner
run or lifecycle wait pending. Owner result: 1,107 tidiness findings (1 error,
237 warnings, 869 infos), duplication35,311 (initial35,603, delta-292), still
above recorded baseline35,303 and target35,302. New warnings include targeted
edit wrappers and editOriginLocalStorage complexity11. No suppression. Evidence:
`evidence/rehabilitation/profile-preservation-2026-09-22.json`.

### BAS-WORK-008 — 2026-09-22 UTC — browser authentication snapshot (investigation)

Feedback reread. RF-011 / J01,J14. Installed rebrowser-playwright1.52.0's own
BrowserContext types document storageState({indexedDB:true}); SessionManager
currently calls storageState() and the local SessionSpec repeats a narrower
cookie/localStorage-only schema. Hypothesis: capture omits IndexedDB while the
restoration path can already accept it. Prove using a real browser and local
synthetic identity stored independently in cookie, localStorage and IndexedDB;
close/reopen one profile and keep a second identity distinct. No real sites or
accounts. Only then enable the supported option and derive the storage type from
the provider instead of maintaining a second partial schema. Preserve unrelated
preexisting edits in manager.ts and types/session.ts.

Cycle008: real Chromium fixture failed on missing IndexedDB identity (cookie and
localStorage remained alpha). With indexedDB:true, 59 session/context/browser
checks pass and TypeScript compilation passes. Provider-derived storage type
removes 15 runtime lines. Fixed a cycle006 error-mapping regression: missing tab
now retains its own NotFound message; strengthened HTTP assertion failed before
and recording-handler race suite passes after. Full build passed. Owner run
20260922-051505-58a226c1 unit/tidiness has one 239s wait attached, pending. Driver
is still outside owner discovery, so this run cannot qualify its browser fixture.

### BAS-WORK-009 — 2026-09-22 UTC — required driver validation (investigation)

Feedback reread. RF-014. Unit Health plan-only returned only api/cli/ui, and a
fresh Code Facts surfaces query omits the service.json-declared playwright-driver
sidecar. Installed Unit Health already supports node-jest. Necessary shared-owner
scope extension: Code Facts api/internal/facts/discovery.go and maintained tests,
plus its architecture docs; discover declared component roots instead of imposing
new BAS directory layout or implementing a private inventory in Unit Health.
Then add BAS's required driver role using the observed surface identity and
existing Jest adapter. Validate fixture discovery, owner package tests and live
plan inclusion before claiming owner execution. Existing thresholds stay intact.
Unit Health skill recall scope was unregistered; read-only plan remains usable.
No external log is written under this file-only engagement.

Cycle008 owner terminal failed96s: UI coverage command and tidiness (1,107
findings, 1error/237warnings/869infos). Direct driver tests are green; routine
owner coverage still excludes driver. No pending wait. Initial Jest reported
an exit-delay warning; detectOpenHandles rerun passed59 with no handle report
and no forced exit. Source digestd359400b95700b200108a938d90a24b1b26a3618b05ebe96323851d510c2fa4b.
API94,119 (+4cycle/-46original), driver55,552 (-15cycle), UI131,518 (-14original).
Go mapping adds one branch to restore prior user-visible error semantics; no
claim of a complexity reduction for that repair. Board remains17pending and
product_qualified=false. Evidence profile-indexeddb-2026-09-22.json. Deployment
will accompany the validation-owner repair after a fresh drain check.

Cycle009 scope refinement: Unit Health's Node workspace resolver applies React
Vitest advice to all TypeScript surfaces, including declared sidecars. Extend
necessary owner repair to unit-health/api/internal/adapters/workspace.go and its
existing tests: recognized non-UI TypeScript server roles can use Jest; actual
React/UI rules remain. Prove the distinction with a sidecar fixture and retained
UI regression. BAS Jest maxWorkers=1 projects its existing owner runner cap,
without changing coverage floors or excluding tests. Code Facts focused surface
regression is red before/green after; full facts -race passed. Owner unit/structure
run passed15s but reports unit L0/TEST_MISCONFIGURATION; keep that qualification
limit instead of claiming all Code Facts tests were admitted. Full build passed.

Code Facts cache.go analyzer identity advances to components-v1 with the changed
surface semantics; otherwise a seven-day report cache could keep omitting the
driver after deployment. This is within the necessary discovery-owner boundary.
Existing read-only source fingerprints do not identify analyzer implementation.

Live owner repair deployed healthy: Code Facts startop-a75c2365bea6d3567b03842e444cf828,
Unit Health startop-71869221859dfe0fbc35f136ad5a9521. Fresh cached describe reports
playwright-driver as KNOWN with analyzer code-facts.components-v1. Unit Health
now plans four workspaces and driver node-jest is ready, with test/coverage
commands and no driver policy error. Shared checks: code-facts facts -race green;
unit-health adapters/discovery/validation -race green; owner unit/structure passed
for both (code-facts20260922-052102-e449ec3c; unit-health20260922-052234-d7bef24a).
Shared-owner changed Go boundary grows2391→2453lines and cyclomatic529→546
(109→110functions). This closes omitted execution coverage, not a source-size
reduction; include the additional discovery safety/policy branches honestly.

### BAS-WORK-010 — 2026-09-22 UTC — profile path boundary (investigation)

Feedback reread; repository source admits any nonempty ID and joins it to a
filename. Disposable sibling-file probe confirms Delete("../outside") removes
an unrelated .json file and returns nil. It touched only a newly created temp
fixture, never operator data. SessionProfiles Delete RPC validates UUIDs, so no
claim of that endpoint being exploitable. Repository boundary and less strict
recording read/edit paths still lack this invariant. Put validation at the
path-owning repository boundary, reject invalid IDs before I/O, retain UUID and
existing safe fixture identifiers, and test all CRUD paths with independent
outside-file preservation. Owner run009 is still pending; no new production
profile mutation yet. Record as RF-052 after maintained reproduction.

Cycle009 owner terminal failed233s: driver suite now admitted and ran123 passing
suites/1,442 passing tests, with2failed suites/26failed tests and1suite/2tests
skipped. It then retained a handle and hit the60s no-output watchdog (130s driver
command total). Owner excerpt shows injector-selectors importing transitive
playwright instead of the deployed rebrowser-playwright provider and demanding
an unprovisioned Chromium1200; failing launch also breaks teardown. Fix the
fixture binding and unconditional cleanup, then capture the second failure and
retained-handle owner with a bounded detectOpenHandles run. Do not lower the
watchdog, skip the test or force-exit Jest. UI coverage still fails; tidiness
still35,311 (+8ratchet, -292original). No wait pending. Source edits for010 began
after009's API command; its receipt does not cover that later repository change.

Cycle009 driver diagnostic was bounded at180s and exited124 before completing,
not a pass. Using rebrowser exposed setContent() on a blank document stalling;
an independent local browser probe times out setContent then succeeds on actual
data-document navigation and evaluate. Selector fixture now navigates to its
synthetic document, preserving every selector expectation. Audio-capability's
remaining assertion concatenated all files, including separately imported
PipeWire qualification code. Keep the platform-neutral decision obligation and
check the transitive imports/exports of audio/index.ts instead, using installed
TypeScript resolution; also prohibit child-process imports. No host-device code
or runtime audio decision changed. Diagnostic additionally observed a transient
synthetic-audio-fidelity Execution context was destroyed after permissions were
granted following navigation; investigate that exact operation ordering rather
than retrying the assertion away. Native audio qualification remains unverified.

Cycle010 implementation: centralized profilePath validation now returns a checked
path to the write transaction, and private snapshot publication consumes that
path. Removed repeated empty-ID checks. Rejects both platform separators, colon,
NUL and path-only dot identifiers. Forty real-file cases, profile/service/recording
race suites, actual recording InvalidArgument HTTP test and full API compile pass.
Build passed. Runtime API94,130 (-35original, +11cycle008→010), driver55,552
(-15cycle008), UI131,518 (-14original). Repository210→215lines, cyclomatic44→47;
cumulative changed BAS Go1105→1104 (-1). Necessary shared-owner009 adds17 outside
that BAS sum; do not claim aggregate complexity fell across the wider boundary.
Retained profile probe stays10/18 with eight replay failures. Board010 returned17
pending outcomes, product_qualified=false. Build010 not deployed yet.

Cycle009 focused selector/audio-decision fixtures now pass33 with detectOpenHandles
and clean exit in1.93s. Selector expectations are unchanged. Synthetic microphone
permission now precedes document navigation; this matches preconfigured-context
capture setup and avoids changing permissions during evaluation. This ordering
change alone is not proof the previously observed transient is eliminated. Full
150s bounded driver diagnostic with coverage/open-handle detection and typecheck
are now running; do not promote them before terminal results.

Cycle009 full driver diagnostic passed125suites/1,468tests (one suite/two tests
explicitly skipped) in76.433s, but process did not exit. detectOpenHandles identifies
AIGatewayVisionClient.analyze's timeout allocated before buildTurns rejects an
invalid screenshot. Body validation throws outside the cleanup try/finally,
leaving the long timer alive. This is a runtime resource leak, not a runner
budget issue. Strengthen the invalid-image regression to require zero timers,
then allocate request resources only after local body validation. No forced exit,
watchdog relaxation or test suppression. Track as RF-053. Synthetic microphone
fixture passed this full run; one pass does not establish a flake rate.

Final owner run for009/010 is20260922-054223-196bad55; one407s wait attached.
RF-053 resource allocation reorder adds no runtime lines or branches;43focused
vision checks pass with detectOpenHandles and clean exit. Previous full driver
diagnostic's exit124 is retained separately from1,468passing assertions. Current
source changes are stable for the final scoped run; earlier009's Go receipt does
not qualify010. Evidence: driver-validation-2026-09-22.json and
profile-paths-2026-09-22.json. No live BAS restart yet.

Final010 build and driver typecheck completed successfully. Awaiting existing
owner run (one waiter, session98097); do not re-admit. Next selected investigation
after deployment is RF-028/RF-030 recorded-action conversion: eight retained
profile-probe replay expectations still fail for modifiers, double-click count,
horizontal scroll, unknown blur/drag handling and page/frame context. Read-only
recall/source inspection has begun; no conversion source changed yet.

Final owner20260922-054223-196bad55 terminal failed167s. API/CLI/driver/typecheck
commands pass; driver naturally exits71.543s with125suites/1,468tests passed,
1suite/2tests skipped. Only UI coverage command fails. Tidiness still duplication
35,311 vsbaseline35,303 (-292frominitial),1,107findings. No owner wait remains.
Preparing drain-gated deployment. Inspection found ListExecutions status argument
is ignored, so do not use a status-filtered read as a drain oracle; inspect actual
returned statuses across bounded pagination and recheck driver health just before
stop. This separate API contract defect needs a maintained reproduction/repair.

Drain gate first refused without invoking stop: full pagination found five
RUNNING snapshots older than the current process, unlike the earlier1000-row
scan. Inspected all2,803 rows:2,448completed/350failed/5running, all five dated
September7–16. Current API beganSeptember22 and driver has no sessions/recordings.
Source confirms both fresh start and ResumeExecution allocate a new execution ID;
these old IDs cannot be active runners in this process. Classify them as orphaned
historical records, preserve their bytes, and admit deployment only if a fresh
scan finds no nonterminal execution created during the current API lifetime and
an immediate driver check remains zero. Do not claim their journal state is fixed.
RF-054 separately proves ignored status filter (PENDING→COMPLETED); RF-055 tracks
orphaned RUNNING history. No lifecycle stop has yet been admitted.

### BAS-WORK-011 — 2026-09-22 UTC — semantic workflow conversion

Feedback008 reread. Recall completed via search-hub (recording conversion query).
RF-028 hypothesis: discarded config plus repeated V1/V2 mapping causes modifier,
click-count, horizontal-scroll and blur/drag corruption. Replace with existing
compiler/typeconv owner, typed flow through service/handler, no compatibility shim.
Necessary same-scenario boundary includes compiler action builder and parameter
builders (drag support and Go/JSON string list equivalence). Source before copies
in/tmp/bas-before-011; original issue/probe evidence retained. RF-030 receives an
explicit unsupported-context boundary pending proper lifecycle reconstruction;
no claim of working multi-tab/frame replay. RF-004 merging remains separate.
Prove semantics through service-generated typed candidates, compiler roundtrip,
unknown/context failures and drag phases, then owner unit/tidiness checks.

Deployment008–010 completed healthy via make stop/start after fresh drain gate.
Build3c0e62a11289d56a838a06f72e452c7b9b49e4466c051e29ebf9a9e2ea6de6aa;
read-only credential-backed verification preserved every original profile field;
live metadata ID/name match and no storage state is exposed by List. Cold1218ms,
warm0.36/0.31ms are observations, not comparative performance qualification.
Owner/evidence files008–010 now include final verdict and deployment. No pending
owner or lifecycle operation. Cycle011 semantic corpus red on previous generator;
implementation in progress, not built/deployed or qualified yet.

Cycle011 scope refinement: typed input explicitly replaces the whole value; the
existing merge implementation would still concatenate full snapshots and mutate
raw payloads before typed conversion. Repair that same deriver ownership boundary
now (RF-004 subset): final snapshot wins only within identical page/driver-page/
frame/URL/selector, submit boundaries remain distinct, no map mutation. Existing
fragment-based fixtures were corrected to actual full-value capture semantics;
new replacement/empty/target/immutability assertions cover expected behavior.

Cycle011 adversarial read during owner validation found a downstream execution
boundary still discarding the newly preserved fields: InteractionHandler.handleClick
calls page.click(selector,{timeout}) only; KeyboardHandler logs modifiers but never
uses them; InputParams submit/clearFirst/delayMs also ignored. Registered RF-056.
This invalidates any whole-replay claim for011; typed-candidate guarantees remain
valid. Next cycle012 will use independent Chromium event/state oracles through real
handlers, repair the semantic execution owner, then revalidate combined source.
No driver source changed while current owner run is in flight.

Cycle011 owner terminal failed172s, API/CLI/driver/typecheck pass. Driver exits
naturally with125suites/1468tests; UI28.6% below85% remains. Tidiness1099 findings
(1error/234warning/864info), duplication35311→34640 (-671this cycle,-963original),
longfiles82→81, coupling12 unchanged. Duplication now satisfies its ratchet; the
remaining selected budget error is complexity385 vsbaseline363. Added context/drag
validation functions explain the count+1 despite total touched cyclomatic667→590
(-77). Retain a cohesive explicit recorded-action switch (23 branches), since its
cases are the small boundary adaptation and labels, not duplicated proto policy.
Review context-identity representation next for a simpler exact binding contract.
Build011 passes but not deployed: RF-056 requires downstream browser semantics
repair and combined validation before treating replay as delivered.

### BAS-WORK-012 — 2026-09-22 UTC — driver action semantics (investigation)

Feedback008 reread; reuse011 browser-semantic recall. RF-056 source evidence:
click options and keyboard modifiers are discarded, input append/submit/delay and
scroll selector/delta options are ignored. Add real Chromium DOM event/state
oracles through production handlers; red before edits. Fix the same execution
ownership boundary and remove divergent option interpretation where practical.
No profile, account, external website or live session is used by these fixtures.

Cycle012 real Chromium fixture red:5failed/1passed; after click/keyboard/input and
scroll-owner repairs6passed. A stepped-scroll unit oracle caught and corrected a
new interpolation overshoot (missing division by step count); retain its failed
receipt. Typecheck found an ElementHandle union overload mismatch, repaired with
an explicit target type. Additional smooth/stepped/viewport oracles added.

RF-057 instrument defect proved: with Babel coverage the scroll fixture fails
with `elementHandle.evaluate: ReferenceError: cov_ieri02bia is not defined`; the
same six actual browser assertions pass without instrumentation. Do not work around
this with production string-eval, discarded source files or lowered thresholds.
Test installed Jest V8 coverage provider, which does not rewrite browser-bound
functions. A provider change breaks direct coverage percentage comparability;
record a new producer identity and preserve all existing thresholds/source scope.

Cycle012 expanded focused suite passes41tests, including10real-browser cases,
with V8 coverage enabled and unchanged floors. Driver typecheck and final Go
context-boundary race check pass. Multiline append was red (insertion at first
line end), then repaired via DOM selection before typing. Scroll unit oracle also
caught the implementation's temporary interpolation error; both failures remain
in evidence. Owner unit/tidiness admission and build are now pending; do not deploy
until their exact results are inspected. No claims of full recording, native OS,
shortcut-composition or concurrent-input qualification.

Cycle013 read-only selection: RF-002/RF-024 share four cache-first journal writes
and two competing query authorities. Recall complete. Inspect repository append,
sequence allocation, actual acknowledgement callers and all broadcast paths before
choosing the storage repair. No013 runtime edits yet; retain pending012 IDs above.

### BAS-WORK-013 — 2026-09-22 UTC — journal schema ownership (investigation)

RF-058 / J06, profile-durability and structural-debt. While tracing RF-002/024,
source search found timeline_entries only in repository SQL and a private test
schema, absent from every embedded production schema. Hypothesis: a real BAS
bootstrap cannot append a journal entry; private test DDL conceals the failure.
Target docs updated first. Repair the existing internal/recording domain schema,
replace private test DDL with its provider, and extend the existing routed-pool
regression to append/read events and verify primary/test isolation. No live data
rewrite, new storage engine or runtime migration. Existing recording_actions
rows stay untouched. RF-002 false success and RF-024 cache pagination remain
separate repair work after this bootstrap prerequisite; no durability claim yet.
Pending012 driver-owner diagnostic session65852; no lifecycle operation pending.

BAS-WORK-013 result: canonical repository fixture and actual routed production
bootstrap both failed with `no such table: timeline_entries` before repair.
Existing recording domain SQL now creates the journal and its two query indexes.
Private DDL removed from repository tests. Routed regression writes/reads only
the leased pool, confirms primary remains empty, and preserves the entry across
reapplying schema. `go test -race ./database ./services/recording/persistence`
passes (10.711s/1.160s); make build passes. Contract and board run
prog_a76a4436-ef0a-4d18-9cc8-bfd867c24b90 complete:17pending/productfalse.
Evidence: evidence/rehabilitation/recording-schema-2026-09-22.json. No runtime Go
branch/function added; removes private SQL ownership instead of introducing a
second bootstrap path. Live deployment pending combined011–013 drain gate.

012 validation diagnosis: driver-only Unit Health CLI also hit its client deadline
without native verdict. One direct bounded600s diagnostic of the same full Jest
coverage command is attached(session48229) to obtain actual failing assertions
or completed timing. No threshold/floor/exclusion change, no Test Genie run
pending. Aggregate changed BAS Go cyclomatic is now1623→1538 (-85), with shared
owner009 +17 giving wider affected net-68; source files8616vs9248 (-632).

### BAS-WORK-014 — 2026-09-22 UTC — durable journal authority (investigation)

RF-002/024, J06 and recording fidelity. After013 supplies the missing table,
four service writers still append to mutable cache before storage and suppress
storage errors. Only two are called by production; the Unified interfaces exist
only in tests, while HTTP unconditionally sends200 and broadcasts. Target:
repository-owned append identity/order and full-history queries, error propagation
to the actual HTTP boundary, one production writer pair, no alternate cache or
unused Unified API. Tests will use the canonical SQLite schema, save faults,
1,001+ events, reopen, concurrent repositories and retries with independent
expected IDs/counts. Preserve journal data and raw observed actions, including
two distinct navigations to the same URL. No live data mutation or destructive
migration is required. Remaining callbacks/page lifecycle side effects are to be
traced before claiming end-to-end durable acknowledgment.

012 full diagnostic completed:126suites/1,476tests pass, 1suite/2tests skipped;
349.969s natural exit under V8 coverage. No product assertion failure. RF-059
records the substantial producer slowdown versus preceding Babel71.543s; coverage
values across those producers are not comparable improvement. Existing300s unit
phase could not return native output. Raise only that phase deadline to600s,
matching the bounded runner ceiling and measured serial aggregate; all coverage
floors, source inclusions, tests and ratchets stay unchanged. Next combined owner
run follows014 compile/tests; no run currently pending. Evidence raw diagnostic
/tmp/bas-driver-full-diagnostic-012.txt. Both prior deadline attempts retained.

BAS-WORK-014 result before owner validation: actual HTTP red returned200 and
broadcast on storage failure/unknown session; repaired to503/404 with no broadcast.
Real journal red falsely published2 uncommitted events, reported501of1001 with
page starting902, and erased a distinct same-URL navigation. Repository now owns
atomic sequence/identity, immutable retry comparison and all history queries.
Service has no cache. Removed recorder.go, unused Unified writers and batch
persistence; all constructor, handler/interface/mock, wire and retained probe
callers converted. Page-event and navigation HTTP failures propagate too, with
explicit browser-effect wording. Failed session registration releases the opened
browser; fake-driver regression confirms close and zero retained ownership.

Canonical disk tests cover reopen, two independent writers, exact1001→1041
history, offsets1/101, identical retry, conflicting ID, large JSON numeric
precision, corrupt JSON and commit fault. Actual handler regression covers
success/error/unknown session. Scoped race suites pass. A temporary fixture
mistake used a raw explicitDSN without shared tuning, exposedSQLITE_BUSY, and was
corrected using storage.SQLiteDSNAt (same production authority, no weaker custom
mode). Journal race timing41.502s→2.733s is a fixture-configuration observation,
not a measured live performance claim. All expected outcomes unchanged.

Measured ten-file pre014 owner union4,600→3,778lines,108→96functions,
560→505cyclomatic before final formatting; exact refreshed values in the evidence
receipt. Runtime source inventory API92,712 versus original94,165 before final
formatting. Contract and board prog_fb5a98ac-4d49-4184-8144-05a70d14073d pass
producer checks,17pending/productfalse. Build passed. Evidence:
evidence/rehabilitation/recording-journal-2026-09-22.json.
Owner20260922-070615-366d3617 pending one533s wait(session67863).
Broader raw-context preservation, callback-gap UI, driver reconnect and native
power-loss remain unqualified. No claims of full recording readiness.

BAS-WORK-014 owner result:20260922-070615-366d3617 terminal failed379s;
API14.978s,CLI0.895s,driver283.766s andUItypecheck12.019s pass. Driver126suites/
1,476tests naturalexit, no retained-handle failure. UI coverage28.57%vs85% remains.
Tidiness1,077findings (vs1,098), longfiles80(vs81), complexity380(vs384),
duplication604groups(vs620), lineDebt35,064(vs34,640), coupling12. The duplication
increase is retained honestly;408of424 anchors to the repeated back/forward/
reload handlers. Cumulative line debt stays539below original35,603; the unchanged
complexity ratchet still fails. Next owner simplification should consolidate
actual navigation commit/notification behavior instead of chasing detector output.

Fresh deployment gate07:14:23UTC:2,924execution rows, no driver session/recording,
five exact unchanged historic orphan RUNNING rows only. make stop completed;
make start admitted07:14:56UTC pending session50204. No history deleted/reclassified.
Read-only production database before stop had no journal table and zero recording
session/action rows. Post-start health, schema and original-profile read checks
remain required. All four011–014 receipts now include final owner result/catalog.

Deployment014 completed. make start returned0; API/driver/UI healthy at07:17:18UTC,
new build e55e1a548d5c0f27774977035f8894db9d09e27f02089ebca18c45eab7962992.
All original saved profile fields match on3read-only reads; live List identity/name
and one-profile count match original, with no storage state in metadata.
Actual startup-log DSN is scenario/data/browser-automation-studio.db; journal table
exists, session/action/journal counts0. Correction: the earlier default-root read
was an unused database, not the running authority. Its receipt is marked invalid
as live evidence; pre-start actual journal presence was not measured. Bootstrap
repair remains supported by real canonical-schema red/green tests. No historical
execution rows or saved profile bytes rewritten. All011–014 receipts include this
correction and deployment proof. No operation pending; continuous work proceeds.

### BAS-WORK-015 — 2026-09-22 UTC — navigation journal ownership (investigation)

RF-002 and structural-debt. Owner014 attributes408of424added duplication debt to
three navigation handlers. They repeat action construction/commit/broadcast and
skip recording silently when API session tracking is missing. Consolidate that
real post-effect owner, preserving each route, payload, browser result and page
notification semantics. Add route-level success/save-failure/missing-session
assertions before repair. Adversarial journal review also found a decoder that
accepts a valid JSON prefix followed by garbage; add the full-document regression
and reject that corruption. Prior broader recording recall reused; feedback008
reread. Native and callback-recovery qualifications remain open. No operation
pending before this cycle; live014 build and saved profile verified.

BAS-WORK-015 result: red proved missing-session false200, stale reload metadata
and trailing JSON corruption accepted. Shared navigation owner removes134lines
from record_mode.go, preserving all route payloads and page notifications.
Common strict journal decoder covers action/page events with exact numeric JSON.
Focused recording/persistence/handler races2.817/1.182/1.198s pass; make build passes.
Final owner20260922-072645-a721922d terminalfailed:complex380 unchanged,
dupDebt33,521 vs35,064, group607vs604; grouping changes are not behavioral claims.
Interimrun20260922-072400-60d89a29 retained; two new complexity findings prompted
simplifying the same nine test cases and consolidating whole-JSON policy.
Three-file comparison1,787→1,699lines,34→36functions,246→247cyclomatic. Service
formatting adds10readability-only lines outside that union. No local net complexity
claim. Contract passes; board01517pending/productfalse. Evidence:
evidence/rehabilitation/navigation-journal-2026-09-22.json. No pending operations.
Live014 unchanged; deploy015 with next coherent repair after fresh drain check.

### BAS-WORK-016 — 2026-09-22 UTC — frame measurement ownership

RF-002/022 callback investigation: HTTP failures and circuit-open are swallowed;
stop does not join deliveries; FIFO can evict uncommitted entries. Close/reset
suppress recording/cleanup errors and delete buffers/ownership (RF-036). A safe
repair must span delivery identity/ack, pending retention, stop/restart and close,
not just add a retry loop. No callback code changed. Recall
/tmp/bas-callback-recall-016.txt; provider results do not supply a BAS replacement.

Select RF-009 bounded frame measurement repair while that boundary is understood.
The existing collector divides retained bytes by lifetime skips/time and calls
blocking socket wait processing latency. Desired: one bounded ring/window owner,
window-local counts/bytes and monotonic observation interval; lifetime count only
for broadcast cadence. Processing durations start after a full frame arrives.
Existing wire names remain compatible but descriptions/UI must not claim measured
network or input-to-paint latency. Independent clock/window fixtures and actual
WebSocket delayed-arrival test will discriminate. Positive capacity invariants,
empty/reset, skipped samples, wrap/recent order and concurrent access required.
Feedback008 reread. No deployment or test operation pending.

016 source inspection expands the same measurement repair to driver PerfCollector,
which repeats lifetime/window mixing, never resets its start time, and can divide
by zero. Include both collectors and their shared UI diagnostic descriptions;
monotonic intervals exclude wall-clock jumps. No new dependency or runtime owner.

016 focused green: Go performance/handlers race1.025/1.293s; driver performance
unit+real Chromium capture14tests2suites3.094s; driver and UI typechecks; make build.
Retained recording-api now9/9. Real WebSocket red100.241ms idle falsely reported
as receive time; repaired omission preserves processing measurement. First test
fixture had to retain collector before handler cleanup; nil after disconnect was
not treated as product failure. Both collectors now use bounded observation rings,
window-local totals and detached chronological recent frames. Driver reset renews
monotonic epoch; wall-clock jumps do not change elapsed time. Four UI consumers
converted to the existing frame-streaming types; duplicate server declaration
removed from usePerfStats. Legacy JSON names retained with honest processing label.
Board prog_72b48fe9-3355-4d87-962f-96231dbb91d5 completed17pending/productfalse.
Owner20260922-074813-d12be3cc unit,tidiness admitted07:48:13UTC; exactly one quiet
663s wait attached in tool session59914; resume that session, never re-admit/poll.
Live014 remains healthy;015–016 await deployment after owner evidence/drain check.

016 measurement: comparable Go3-file union1,578→1,526lines,32functions unchanged,
180→178cyclomatic; driver collector378→323lines. Existing UI canonical types
replace79duplicated declaration lines in usePerfStats (2importlines added).
Affected Go union vs original37files15,845→14,258lines,541→523functions,
2,406→2,266cyclomatic (-140); sharedowner009+17 gives wideraffectednet-123.
Three sequential microbench trials: Go Record amortized allocation270→0bytes/frame;
median36.08→42.92ns/op, a small measured cost increase for monotonic observation,
not a speed improvement. Shared-host conditions limit timing attribution.
Initial concurrent benchmark retained but not treated as speed evidence.
A cwd mistake ran root make build (completed0, no install); that log is explicitly
excluded as BAS evidence. Correct final BAS make build now pending.
Evidence evidence/rehabilitation/frame-window-2026-09-22.json; owner wait59914
remains pending, do not poll/re-admit.

RF-036 pre-repair inspection during016 owner wait: driver teardown catches every
recording/tracing/context failure; manager catch/finally removes session ownership
and concurrent close returns empty success. Go closeSessionWithArtifacts also
returns void after close/artifact-write failure, and resolveArtifactPath can return
a missing source path after failed download. These are one finalization boundary,
with capture/teardown evidence and repeat-safe ownership required. Existing test
expects a synthetic already-closed-page error to succeed; distinguish actual
closed resource from unverified generic close failure when repairing. No017
runtime edits started while016 owner source is being validated.
Final BAS make build completed0; initial root build excluded.

016 final owner result:20260922-074813-d12be3cc failed (unit438s); API/CLI/driver
andUItypes pass. Driver126suites1,478tests naturally exit345.608s, command346.324s.
UIcoverage still28.57/30.34/64.96/28.57vs85. Tidiness1,080findings,80long/380complex/
607dup/12coupling, debt33,697 (+176cycle; -1,906original). No weakened threshold.
One quiet wait59914 completed; artifact catalog retained in015/016 receipts.
Fresh deployment gate stopped before lifecycle mutation due new RUNNING rows;
07:57:39UTC observation10newRUNNING/10driver sessions over3,046historyrows.
Do not interrupt or reclassify active work. Runtime014 remains;015–016 source/build
verified but not deployed. Recheck only after current workload settles.
Continue RF-036 finalization boundary investigation/repair; feedback008 reread.

### BAS-WORK-017 — 2026-09-22 UTC — executor finalization authority

RF-036/J08/J18. Start with the Go executor's complete finalization boundary:
Execute currently returns nil after engine Close/CloseWithArtifacts or artifact
writer failure; missing remote downloads become requested paths. Use public
Execute fault cases (ordinary close, artifact close, write, missing download,
cancellation context) before repair. Return combined errors and preserve original
action cause, use bounded WithoutCancel context retaining routed storage values,
and consolidate video/trace/HAR handling. Preserve source evidence on failed
import; no repeated browser action. Driver teardown suppression/recovery remains
a connected next owner, not claimed repaired by Go propagation. Shared dirty
executor code inspected; no unrelated edits reverted. No test run currently
pending; live workload prevents015–016 deployment. Feedback008 reread.

017 first Go executor repair passes public failure/cancellation/local+remote bytes
tests and executor/engine/workflow races1.253/1.078/1.067s. Retained execution-api
close-failure probe now passes;5other failures remain. Invalid --case execution
invocation corrected to execution-api and not treated as product evidence.
SimpleExecutor1,899→1,866lines,373→369cyclomatic;close helper17→13.
Tidiness interim20260922-080702-56cc1e9d terminalfailed; quiet wait completed once.

017 boundary extension within BAS: actual execution-writer/external_artifacts.go
suppresses stat/read/sanitize/storage errors. Trace metadata gets a digest but
neither inline bytes nor durable storage, so downloaded trace temp cleanup loses
the only local copy. Include this owner and its tests before claiming Go
finalization repaired. Preserve sanitized HAR policy and source files. Success
requires committed bytes or supported sanitized inline bytes; failed stores and
missing sources propagate, valid sibling artifacts can persist with explicit
combined failure. Nil/empty store receipts cannot establish success.
The driver teardown suppression and callback pending recovery remain next owners.

017 actual writer red reproduced seven false-success cases and missing stored
trace bytes. Repaired with one per-artifact preparation transaction and batch
error accumulation. Missing/malformed/nonregular sources, failed/empty/incomplete/
wrong-size receipts fail explicitly; valid sibling artifacts persist. Trace/video
bytes require storage. HAR stores sanitized derivative only (small derivative can
still use the existing inline policy without a configured backend). Sources remain
unchanged. Removed video-only storage gate and repeated log-and-continue branches.
Two older trace metadata tests now supply the same memory backend and retain all
metadata/path assertions; new test deletes the source and retrieves matching
trace bytes/hash through storage. Real writer failure also reaches public Execute.

Cohesion review: prepareExternalArtifact27 versus former RecordExecutionArtifacts37.
One read/sanitize/store/receipt transaction; batch accumulation separate. Branches
are explicit failure/disclosure decisions. No extraction-only complexity claim;
keep this hotspot visible in the owner scan. Final focused race suites pass;
final unit,tidiness owner just admitted (identity recorded after admission output).

017 final focused result: writer/executor/engine/workflow races2.012/1.297/1.081/
1.074s; final make build passes. Two-file runtime2,042→2,007lines,52functions
unchanged,414→403cyclomatic (-11). Cumulative affected Go original→current
39files17,887→16,265lines,593→575functions,2,820→2,669cyclomatic (-151),
sharedowner009+17 yields wideraffectednet-134. prepareExternalArtifact27 remains
visible; aggregate reduction comes from deleted error suppression/repeated import
policy, not extraction. Contract/board017 producer checks pass17pending/productfalse.
Final owner20260922-081559-47cf38fc admitted08:15:59UTC; one784squiet wait attached
in tool session56392. No other owner/build/lifecycle operation pending.
Evidence evidence/rehabilitation/execution-finalization-2026-09-22.json.
Runtime014 unchanged because fresh drain016 found active workload. No live data
changed. Do not mark goal complete/blocked; continue after owner result.

017 final owner20260922-081559-47cf38fc failed401s (unit397s); API/CLI/driver
and UI types pass; same UI coverage floor and complexity ratchet fail.
Tidiness1,078findings,long80/complex380/dup605/coupling12,debt33,679,
-18cycle/-1,924original. Native receipts and catalog retained with017 receipt.
Quiet wait56392 completed exactly once; no build/owner pending.

### BAS-WORK-018 — 2026-09-22 UTC — instruction admission and start retries

RF-031/037,J07. Feedback008 reread. Driver finalization investigation shows
retained closing state is unsafe until the run route respects failed phase
transitions. Same-execution start also resets active phases, so include the
manager and session-decisions owner with the route. Hypothesis: delayed effects,
reset/close and retried starts can allow competing browser effects. Discriminate
with actual manager + route admission, delayed body/action/cleanup tests.
Same-execution start must observe ownership without reset/recovery. Admission
must occur after awaited body parsing and before dispatch; all exits release only
the reservation they still own. Preserve recording phase and audio additions.
Lease wire and payload/invocation cache identity stay explicitly unresolved;
no full fencing claim. Record source before edits, run focused driver checks,
retained probes, typecheck and scoped owner tidiness. No new dependency.

018 initial regression15failures reproduced. Removing start phase recovery and
checking admission after body parsing fixes these. Follow-up actual delayed
reset/close tests pass, but reset completion while an older action is unsettled
still admits another action (200vs409). Add one per-session in-flight reservation
until the promise settles; pool reuse and idle cleanup must respect it too.
This is the same session ownership boundary, with the types file added explicitly.
A cwd-only test-edit attempt failed before writes, then retried with absolute paths.
Earlier unmodified route check passed; not used as repair evidence.

018 focused72tests/4suites2.574s pass with natural exit under detectOpenHandles.
Earlier default-mode Jest printed its one-second advisory but exited; no hang
claim. Driver typecheck passes. Existing browser semantics suite extended with
actual run route/executor/manager/InteractionHandler and a post-effect gate:
first click independently observed, start retry preserves ownership, competing
click409, next click executes after settlement. 11tests3.218s pass. Delayed
actual reset/close and reset-completed-before-action-settled cases also pass.
One prior clean-retry test expected destructive reset of the same owner; replaced
with explicit released-lease clean reuse by a new owner, retaining cookie-reset
proof. Unsafe recovery-decision test replaced by actual lifecycle/route tests;
no preserved capability assertion removed.
Retained execution-driver7/14pass (four phase guards pass;7otherfailures unchanged).
Reuse probe6/17pass unchanged profile/capture reuse gaps. Board01817pendingfalse;
contract preparation passes. Local4runtimefiles2,456→2,265lines (-191); deleted
three unsafe/unneeded decision functions and unused result types; one shared
finally replaces four releases. Existing cumulative Go metrics unchanged.
Owner20260922-083535-a68d66dc admitted08:35:35UTC; quiet784s wait once(session73730).
Last workload observation08:35:32UTC3,088rows,5historicorphans,driver0sessions/
0recordings. Fresh gate required before eventual lifecycle change.
Code Facts before snapshot lacked manifest (unsupported); relative after target
resolved under service cwd (invalid); corrected to absolute and copied existing
manifest to snapshot. Unsupported 'complexity' fact family rejected. These are
producer limits, not passing complexity evidence; after/all read26721 pending.

018 Code Facts all-families read26721 hit client deadline; bounded symbols-only
reads completed (before97306/after71999), retaining native evidence. They measure
function/type removal, not cyclomatic complexity; exact counts in018receipt.
No additional Code Facts read pending. Retained session-frame probe confirms
active-instruction start retry nowpasses; teardown false-success stillreproduces.
Go session wrapper also marks closed before driver receipt; a concurrent Close or
Release currently returns nil before the first request resolves. This must join
one terminal operation result before complete finalization can be claimed.

018 first full owner20260922-083535-a68d66dc terminalfailed437s; quiet wait73730
done. API12.497s/CLI0.816s/UItypes10.697s pass. Driver126executedsuites,
1,497pass/2fail/2skip in348.649s: two older idempotency tests explicitly demand
same-owner cookie reset and unsafe executing→ready start recovery. Align their
expectations to desired ownership and retained new-owner clean-reset capability;
parameterize all phases and assert lease/phase/admission preservation. These were
not caught by initial four focused suites. Independent actual-browser delayed-click
proof already demonstrates why the old behavior is unsafe. No runtime change.
UI coverage unchanged below85. Tidiness unchanged1,078/80/380/605/12/debt33,679.
After focused idempotency check, rerun unit owner; reuse completed tidiness proof.
No deployment until this correction has verified driver evidence.

018 corrected focused90tests5suites2.282s pass. Final unit owner
20260922-084539-e7702029 admitted08:45:39UTC; exactly one778s quiet wait attached.
No runtime edits while owner validates. 019 Go terminal-operation repair can be
prepared/validated through a temporary Go overlay, then applied after018 owner.

### BAS-WORK-019 — 2026-09-22 UTC — joined terminal operations

RF-036/J07/J08. Feedback008 reread. Public Go Session Close/Release wrappers
share a closed marker but return nil while the actual driver request is pending.
Four real HTTP fault cases in a temporary test overlay reproduce canceled joining
callers receiving nil for close-close/close-release/release-close/release-release.
Original source remains unchanged during018 owner. Boundary: automation/session
Session finalization and its tests; consolidate release/close ownership, one
in-flight result and completion signal. All waiters observe original operation
outcome or their own cancellation; canceled waiters must not cancel the owner.
Failures retain session ownership and permit explicit later retry; successful
terminal notification fires once. Keep existing absent-driver404 terminal policy
explicitly limited to resource ownership, not proof of retained artifacts.
Driver teardown suppression and capture policies remain connected next owners.

018 final unit20260922-084539-e7702029 terminalfailed371s, wait81548doneonce.
API12.221s/CLI0.920s/driver288.389s/UItypes10.440s pass. Driver126suites1,502tests
287.528s naturalexit,2skipped. UI coverageunchangedbelow85; no production regression
assertion remains in driver run. First018tidiness receipt reused unchanged.
Fresh deploy gate rejected before make stop:08:52:48UTC3,095rows,1newRUNNING and
1driversession. No lifecycle mutation, live014 preserved. Do not reclassify active
work as oldorphans. Continue019 while this workload owns the browser.

019 overlay green: session/executor/engine/driver races1.054/1.282/1.068/1.035s.
The complete terminal operation replaces separate closed/artifact flags; shared
pending/result signal and retained success have one owner. Concurrent callers
join the same receipt/error, canceled waiters leave owner running, failed attempts
retain ownership for explicit retry, successful callback fires once. Existing
closed-session guard fixtures now use an acknowledged terminal signal and retain
all assertions. Test success/failure controller uses existing approved testify
assertions for readable oracles, not extraction of production branches.
Local runtime407→408lines,36functionsunchanged,76→77cyclomatic: explicit+1cost
for concurrent completion, not a local complexity reduction. No caller/shimadded.
Ready to apply overlay after018owner completed; no production019edit beforethen.

019 boundary extension: driver.Client.CloseSessionWithLease and ReleaseSessionLease
ignore the typed success flag in HTTP200 bodies. Include these two client methods
and public Session real-HTTP tests: missing/negative acknowledgment must fail and
retain ownership; partial close artifact metadata may accompany the explicit error.
This is necessary for the shared terminal result to represent an acknowledged
operation. Preserve existing404 policy. No retry engine or new protocol added.

019 acknowledgment red: all6HTTP200 missing/false cases returnednil. Driver client
now rejects missing/negative success, retains partial close metadata with error.
Actual public Session tests prove ownership retained, explicit valid retry ends
it once, and partial metadata is preserved. Final four Go race packages pass
1.076/1.309/1.076/1.042s. Final make build passed. Contract/board019 producerchecks
pass;17pending/productfalse. Scope validation: changed Go session/driver/engine/
executor packages plus BASbuild and owner tidiness;018full driver/UI proof reused,
not claimed full candidate unit qualification. No thresholds/denominators changed.
Final runtime union2files1,560→1,569lines,98functionsunchanged,282→286cyclomatic
(+4 for explicit synchronization/acknowledgment checks). Cumulative41Go files
19,447→17,834lines,691→673functions,3,102→2,955cyclomatic(-147); shared009+17
meanswideraffectednet-130. Earlier standalone session count408 included one dead
closed field missed by a spacing-sensitive edit; final review removed it before
source application. Final session407linesunchanged,36functionsunchanged,76→77.
Owner20260922-085824-c5e34b26 tidinessfailed4s; quiet120s waitcompletedonce.
1,077findings,long80/complex380/dup604/coupling12,debt33,648(-31cycle/-1,955original).
Fresh019 deployment gate again rejected currentnonterminalwork beforemake stop;
no active execution interrupted. Runtime014 remains; noowner/build/lifecyclepending.

### BAS-WORK-020 — 2026-09-22 UTC — driver close recovery ownership

RF-036/J07/J08/J18. Feedback008 reread. Driver retains neither failed cleanup
nor completed-step state; concurrent close immediately returns empty success.
Now that admission respects closing and Go honors errors/acknowledgments, repair
the driver close owner: manager + session-teardown + session types and public
route/manager tests. Keep the same Session in closing after failure, join one
in-flight close result, and retain successful teardown stages/artifact paths for
explicit retry. Failed pre-close recording/audio/SW/Playwright-trace steps must
not dispose the context or recording buffer. Verify artifact references against
actual files; video rename fallback requires a readable original source.
Page/context/CDP failure remains owned; external targets must not have their
pages/context/process closed. Idle cleanup/shutdown must attempt all sessions
without falsely reporting completion. Do not weaken the admin close boundary.
Accessibility/performance collectors' internal best-effort failure policy and
recording callback pending durability remain explicit connected RF-013/002 work;
this cycle cannot claim those unreported inner failures repaired. Per-session
recovery state is bounded by existing session capacity; process-restart recovery
and indefinitely hung Playwright promises remain unqualified.

020 red: actual HTTP trace/context faults returned200 and concurrentclose returned
before cleanup. Initial green run exposed fixture problems: host-audio probe uses
same mocked handles (reset counters after creation), and body delivery is timer-
based (wait for second route admission before releasing firstclose). These were
fixture corrections, not weakened product assertions.
Teardown now retains successful stages, validates captures, rejects failures,
keeps buffer until success, and manager keeps same closing Session/lease onfailure.
Reset refuses closing; idle/shutdown attempt every requested session and report
aggregate errors. Generic already-closed-page test now models page.isClosed=true
instead of expecting an unexplained close exception to disappear.

020 native video/trace/HAR case first returned500 without error-body assertion;
focused and full reruns passed. Cause of that initial500 not established. Added
error-body assertion and explicit double-animation-frame fixture paint. Separate
source inspection of installed Playwright Video.path shows it returns only the
initialized destination. Controlled writer createsvideo onlyon contextclose:
red ENOENT beforecontextclose proves the independent ordering defect. Repair moves
video onlyaftercontextflush; finalboundary revalidates everypublishedreference.
Real Chromium output has WebMEBML/ZIPPK/HAR1.2 signatures and0remaining sessions.

020 late-close review: successful teardown removes session before asynchronous
shared-device cleanup completes; lease guard then returned404 to overlapping
close. Newregression reproduces this; closing owner now retains Session pointer
pluspending result, so matchinglease joins andwronglease stillfails. No authrelax.
Six focused suites102tests passed before thislastcase; final103test/typechecks
pending tool sessions26173/46851. Noowner020/build/lifecycle operation pending.
Retained probes updated onlysynthetic filesystem/page seams andfailedHTTP receipt;
normalclosecontrol usesindependent manager so retainedfailure isnotmiscounted.
Original datedreceipts preserved; teardownfailure nowpasses, othergapsremain.

020 final focused103tests6suites14.755s naturalexit withdetectOpenHandles; driver
TypeScriptcheckpasses. Retained session-frame7/16,reuse7/17; knownremainingfailures
preserved. Board020prog_43cbe165-1a86-4748-996e-3d5718ecee66:17pending/productfalse;
contractpreparationpasses. Full unit,tidiness owner admitted; recover identityfrom
/tmp/bas-owner-admit-020.txt andattachquietwaitonce. No more sourcechanges duringrun.

020 full owner20260922-092430-2c566001 admitted09:24:30UTC; exactly one784s quiet
wait attached to /tmp/bas-owner-terminal-020.json. No further source edits while
it validates. Remaining work in parallel is read-only next-owner investigation.

020 full owner completed failed424s (unit420s). It exposed two stale fixtures:
Go journal-failure cleanup mocked HTTP200 with `{}` as successful close;019 now
correctly requires success=true and retains unacknowledged ownership. Driver
reset route's fake Page omitted isClosed/video;020 now exercises those real APIs.
Repair fixture fidelity and add negative close-ack control; preserve assertions
about successful cleanup and retained failed ownership. API20.387s failedonecase;
driver125suites/1515tests pass,onecase failed,2skip,323.836s and open-handle warning
after failed shutdown. CLI/UItypes passed; UI coverage28.57%vs85 remainsfailed.
Tidiness1077,long80/complex380/dup604/coupling12,debt33648 unchanged. Noownerpending.
RF040 graph/linear persistence investigation remains read-only until020revalidated.

020 fixture corrections pass: Go live-capture race1.045s, driver reset2tests2.115s
naturalexit. Explicit success=true releasesfailedjournalbrowser; missing/false
ack preservesmanagerownership andjoinederror. Finalunit20260922-093435-960e5a77
admitted09:34:35UTC; one649squietwait session80028 attached. No source edits until
terminal;021 prepared onlyin /tmp overlay ifuseful. Tidiness020 unchanged33648.

### BAS-WORK-021 — 2026-09-22 UTC — cancellation outcome persistence

RF040/J18. Feedback008 reread; recall found existing investigation/probe, source-
ledger recall degraded withdeadline exceeded; no new workflow needed. Target
owner remains Go executor. Graph ordinary/subflow/loop/set-variable persistence
uses canceled ctx; linear/terminated persistence discards cancellation without
a deadline. Consolidate outcome persistence/event payload/progress policy into
one bounded cancellation-independent owner retaining request routing values.
Keep linear-only checkpoint semantics (graph resume is not qualified). Join
storage failure with action/cancellation cause. Prove public Execute with real
filewriter and context-sensitive routed root, faulting disk, graph/linear and
nested subflow/loop controls. Preserve success, event identity and artifacts.
Scope: simple_executor.go, flow_executor.go, existingexecutor tests, retained
probe/docs. Requiredcapture failures, graphresume and transportidentity remain
separate issues. First stage is /tmp Gooverlay while020unit runs.

021 real-writer red expands necessary owner boundary: RecordStepOutcome logs
and swallows proto timeline, result manifest and index failures. Even repaired
executor cannot report unavailable evidence. Include execution-writer/file_writer.go
required-write returns and native writer tests; preserve optionalartifactpolicy
as separateRF013. Overlay onlyuntil020terminal. Eight cancellation/disk cases
reproduce missingbudget/canceledgraph and hidden diskfailure. Root observer first
included telemetry calls; corrected to observe actual RecordStepOutcome context
while retaining real filesystem writer, not weakening boundedoutcome assertion.

020 corrected finalunit20260922-093435-960e5a77 completed. API12.880s/CLI0.779s/
driver297.000s/UItypes10.587s pass. Driver126suites1516tests296.155s naturalexit;
2skip. UImergedcoverage28.57stmts/lines,30.34functions,64.96branchesvs85 remains
unitfailure; no weakening. Catalog /tmp/bas-artifacts-final-020.json andnative
/tmp/bas-unit-native-final-020.json. Bothquietwaitscompletedonce. No ownerpending.
Fresh020 deploymentgate running(session86508); source021 still only/tmpoverlay.

020 deployment completed: gate09:42:09UTC scanned3135rows(2693completed/437failed/
5exacthistoricRUNNING),driver0/recording0. Managedstop/start healthy09:46:11UTC,
builddc7fae94ed05ef12aa56d1f5a9fd02ffdb2db096ae2714bb98acdea9d6dbc5fd. API/driver/UI
healthy;oneoriginalprofilemetadata present andalloriginalsavedfields compareequal
tooriginalrollback. /tmp/bas-health-deployed-020.json,profile-api-final-020.json,
profile-verify-final-020.txt. Gate isobservational,notatomicadmissionlock;sharedtree
changes limitcandidateexactness.021remainedoverlayuntilstartfinished.

021 additional publicsyntheticcase red: pre-canceledlinearset_variable returned
success andcreatedbrowser. Cancellation guard nowbeforelinearadmission andowned
at executePlanStep forordinarygraphandloopbody. Syntheticlinear successnilGraph
nowusescommonprogresswithoutpanic. Loopparentfailurealsopersists; childfailure
cause retained. Requiredwritefaults timeline/manifest/index allred(nilerrors)
thenpasswithrealfilesystem preservation.16newcases inclcontext-awareevents,
4writercases. Overlayfourpackage races1.830/2.350/cached/1.059s pass.

021 reviewed overlay applied after020owner/lifecycle completed; bytecomparison
confirmed all6existingpaths unchanged sincebeforecopy, preserving unrelatedwork.
Callerconversion leaves intPtr unused; removeprivatehelper fromflow_utils.go
(necessaryboundaryextension, no survivingcaller). No paralleloutcomepathsremain.

021 finalsource validation passes: executorrace1.646s/writerenginecached/workflow
1.060s;previousoverlaywriter2.350s. AllAPI go test ./... andBASmakebuild pass.
Retainedexecution-api4/8(graphcancelrepaired);no historicalprobe expectationedited.
Contractpreparationpasses;boardprog_13cd5954-74b3-43af-8bfa-2f1722b2588b17pendingfalse.
Owner20260922-094905-f7781095 tidinessfailed4s,quietwaitdoneonce;1076findings,
long80/complex379/dup604/coupling12,debt33649(+1cycle/-1954original). Local4runtime
4607→4485lines,130funcunchanged,934→918cyclomatic(-16). Cumulative45unionpaths
(44currentfiles;deletedrecorderstillinbaseline),22188→20453lines,772→754func,
3667→3504cyclomatic(-163);shared009+17wideraffectednet-146. No cutoffchanges.
No owner/buildpending;live020healthy. Multi-fileatomicity/exactlyonceretry and
forcedtimeoutsfornoncooperativefilesystem remainunknown;contextbudgetisnotproof
ofinterruptibleI/O. Scope excludesoptionalcaptureerrors andgraphresume.

### BAS-WORK-022 — 2026-09-22 UTC — explicit screenshot failure verdict

RF013/J08. Feedback008 reread. Scopedsearch matchednoproviders; requiredfallback
prompt-managerdiscover returned screenshotusageactions butno applicable repair
workflow. Existingactualexecutorprobe stillfails: shouldIgnoreFailure unconditionally
turns everyfailed screenshot into Success=true, clearsfailure, skipsdeclaredretry.
Delete that policy/helper; useexisting declaredretry andcontinueOnError owner.
Explicitcontinue may allowworkflowcompletion butmustpreservefailedstep evidence.
Scope: simple_executor.go andexistingtelemetry_directive_test.go publicExecute
withrealfilewriter/storage, typed screenshot result/error, retry, optionalcontrol.
Unreportedinnercapture/storagefailures remain a separateconnectedRF013 boundary;
this cycle repairs verdicts for failures the engine actually reports. No capture
capability/productqualificationclaim. Live020remains;021readybutnotyetdeployed.

022 red24casepublicExecute matrix: requiredfailurereturnsnil,optionalworkflow
losesfailedstep,declaredretrydispatchesonlyonce. Removedunconditionalscreenshot
successoverrideandshouldIgnoreFailure; existingretry/continueowner handlesall.
24casesgreenwithrealfilewriterandmemorystoredPNG;fourpackage races3.397/cached/
cached/1.059s,makebuildpass. Boardprog_85f1c2fa-8151-41aa-87e2-13f661fa8875
17pendingfalse;contractpass,execution-api5/8(explicitscreenshotrepaired).
Firstowner02220260922-095454-de285146 tidinessfailed4s;1078findings,long80/
complex380/dup604/coupling13,debt33649. Newtestharnessitself had17cyclomatic,
21imports. Rewrite policy-name conditionals as explicitexpected-outcome table;
preserve24cases/assertions. Importsnecessaryfortypedpublicexecutor/realstorage
andPNGoracle remain; no importshuffle togamecoupling. Finaltidinessadmitted
aftertablechange;recoverIDfrom/tmp/bas-owner-admit-final-022.txt andwaitonce.

022 finaltablefixture executor race3.704s pass,24casesunchanged,cyclomatic17→7.
Finalowner20260922-095731-37153b89 tidinessfailed4s;1077,long80/complex379/
dup604/coupling13,debt33649. Bothquietwaitsdoneonce. Newcouplingfinding21imports
inexistingtestfileisdocumented,notshuffledaway. Runtime1850→1820lines,50→49func,
371→363cyclomatic(-8);cumulative44currentGofiles22188→20423lines,772→753func,
3667→3496cyclomatic(-171),shared009+17net-154. Innercapture/storagefalseack
remainRF013nextboundary;noactualbrowserfaultqualificationclaim. Live020healthy,
021/022sourcebuiltbutnotyetdeployed. No owner/build/lifecyclepending.

022 deployment gate rejected beforestop: full-history offsetpagination repeated
anexecutionidentity (/tmp/bas-deployment-gate-022.txt). This cannotestablishsafe
drain; do notinferactivecounts orrestart. Live020continues. Recheckafterusefulwork.

### BAS-WORK-023 — 2026-09-22 UTC — screenshot bytes and storage receipt

RF013/J08 plusnewRF060. Feedback008 stillactiveandpreviousverbatimreread. Recall
findscheckpointprofilework: preserveProfileNone'sexplicitdiscardpolicy; no passive
capturepolicyoverride. Readonlyinspection: sanitizer slicescompressedimagebytes
tothebudget andmutatescaller'sScreenshot/Notes; writerlogsstorefailureandaccepts
nil/incompletereceipt. Targetownerexecution-writer/file_writer.go plusexisting
writer/publicexecutor tests. Screenshotbytesareimmutable;oversizeimagesareomitted
withreason,neverbyte-truncated. Requiredexplicitscreenshotwithcollectionenabled
mustretainvalidbytesandcompleteobject/URL/size receipt orpersistfailedoutcomeand
returnerror. Passiveimagesremainoptionalwithreason;ProfileNone stilldiscardsall.
Retainbothartifactfailureandrequiredmanifest/index errors. No newstoragebackend,
imagequalityreduction,temporaryframeworkorsecondwritepath. Sourcepreservation,
PNGdecode/hash andfaultreceiptsdefineoracles. Scopedraces/allAPI/build/tidiness.

023 initial22writer receiptcases allpassafterrepair. Addedtruncated-PNGinput
showedDecodeConfigacceptsbrokenstream; fullimageDecode nowrejects beforestorage.
24writercasesplus2publicExecute storagefaultcases planned. Bench realwriter+
memoryimageStore+realmanifestfiles,1280x720 fixedfixture,3x10iterations:before
31.761/32.071/32.570ms,~113KB/op;fullvalidation37.572/38.079/38.766ms,~3.863MB/op.
Measured~18.7%medianlatencycost/+3.75MB/op; not a performancegain. Header-only
validation rejectedbecauseknowntruncatedimagepasses. No weakenedfidelity/workload.
Completecodecverification costsremainmeasureddebt; wholebrowserjourneyimpact
unmeasured. Sourcepreservationandrequiredcaptureintegritytakeprecedence over
falseacknowledgment; optimizeownerI/O/validation laterwithoutremovingoracle.

023 finalvalidation:27newcases,writer/executor/workflow/storage racespass;allAPI
tests passedbeforefinaldiagnosticrefinement andchangedpackagesrerun;finalBAS
buildpass. Owner20260922-101405-34d1a561 tidinessfailed4s;bothwaitsdoneonce.
1078findings,long80/complex380/dup604/coupling13,debt33649 unchanged. Board023
prog_df6c3a1e-8e7b-4513-ac85-2a9b2bcf4072:17pending/productfalse. Contractpasses.
Localruntime1504→1539lines,35→36functions,304→314cyclomatic(+10explicitreceipt/
bytechecks);cumulative44currentfiles22188→20458lines,772→754func,3667→3506cyclo
(-161),shared009+17widernet-144. No owner/build/lifecyclepending. Live020remains;
022restartgatepaginationinconsistent,noactive-countclaim. Next memoryreview
foundFileWriter.ForgetExecution dropsonlyconfig;results/timelinesretained. Workflow
servicealreadydefersForget;archiveingestionpersistentwriternevercallsit.

023 deploymentcompleted: gate10:16:47UTC coherent3110rows(2676completed/429failed/
5unchangedhistoricRUNNING),driver0/recording0. Managedstop/start healthy2026-09-22T10:19:19Z;
sha256:cac7d388ab175e7832912adf5bb2395834de952f92e7a64e6264979d5407f243. API/driver/UIhealthy;oneprofileandalloriginalsaved
fields verifiedagain againstoriginalrollback. Receipts health-deployed-023.json,
profile-api-final-023.json,profile-verify-final-023.txt under/tmp. Rowcountdiffers
from020observation; no deletions performedhere and no attribution inferred.
No owner/build/lifecyclepending. Cycles001–023deployed,sharedtree limitationretained.

### BAS-WORK-024 — 2026-09-22 UTC — completed execution memory ownership

NewRF061/J07. Feedback008 reread. RecallfoundBASusage/improve andpriorrefactor
records;source-ledgerrecallagainpartiallydegraded. FileWriter ownsresults/timelines/
perExecConfig maps; ForgetExecution dropsonlyconfiguration. Workflowservicealready
defersForgetafterallwrites;archiveingestionhasonepersistFrames writerpathandno
release. Extendexistingterminalowner toreleaseallthree perexecutionaccumulators;
adddeferredForgetinarchivepersistFrames(success/failure/cancel). Reuseexisting
interface, no newcache/registry/tombstone. Scopeartifact_config.go/interface.go,
archivepersistence.go,existingtests. Preservecompletedfiles andotheractiveexecution
state. Measure retainedheap withrealwriter+filesystemstore andsamefixedcohort,
not a private-map assertion. Go weak-pointer eventualcollectionisnotguaranteed;
do notuseGC-timingassertions asunit gates. Activeexecutionbudget/lateinvalidwrites
remainseparateunknown; terminalcallsite mustfollowalllegitimatewrites.

024 initialnative memorycohort16finishedexecutions x3,eachuniqueDOMwithsame
shapeandfilesystemstorage:before674606/674650/674616 retainedB/execution; after
1402/1144/1240. ScopeprocessheapafterGCwithwriterkeptalive;notGC-timingunitgate.
Archiveexit3casered(noForget)→green. ExistingForgetnowdeletesresults/timelines/
configusingexistingstringkey;archivepersistFrames deferssameowner. Completed
manifestbytes andotherliveexecution's twoentries preserved. Writer/import/workflow
races3.090/1.043/cached pass. Local3runtimefiles287→292lines,8func/43cyclomatic
unchanged. Larger64cohortbefore/aftereach3trials running(sessionoutputnext).
Firsteditassert usedwrongkeytype andmadezerochanges;failedtargetedruninspected,
canonicalstringkeypreserved inactualrepair. No benchmarkfromthatno-op counted.

024 largercohort64×3:before689404/689498/689379,after270.4/285.2/279 retained
bytes perexecution. Bothcohorts agree finishedpayloadretention removed; no speed
or liveheap attributionclaim. Scopedraces/build/contractpass;board02417pendingfalse.
Owner20260922-103012-c25ab689 failed4s,quietwaitoncecompleted. Cumulative48union/
47currentfiles22475→20750lines,780→762functions,3710→3549cyclomatic(-161),
shared009+17widernet-144. No owner/buildpending;live023healthy,024notdeployed.
RF061 discovered duplicate oforiginalRF023; consolidated issue andreceiptunder
RF023 without erasinginvestigationhistory. Currentcheckpoint condensed; dated
records preserved. Native tidiness counts in execution-memory receipt.

### BAS-WORK-025 — 2026-09-22 UTC — structured execution evidence

RF049/J24. Feedback008 reread; recall found prior BAS domain-outcome work and
existing usage programs, no applicable repair workflow. Writer's private generic
converter renders structs/pointers with fmt.Sprintf, losing raw outcome structure.
Existing typeconv owns value conversion but rounds fallback integers through
float64, unwraps data-shaped maps, and silently drops unsupported members. Extend
that existing owner with strict raw-value conversion/error propagation; keep
serialized-proto interpretation only in the existing convenience input adapter.
Delete the writer's duplicate converter and use strict conversion for artifacts,
assertions and extracted values. Typed outcome retains version/attempt/failure
and JSON field names; unsupported evidence returns an error, never debug strings
or a successful partial projection. Scope writer and typeconv existing files/
tests; public writer→disk proto→typed outcome tests plus shared conversion controls,
compiler/protoconv/executor/workflow checks. Saved malformed historical strings
remain strings and will not be fabricated into reconstructable evidence.

025 boundary extension: first test compile exposed typeconv→execution-writer
import cycle. Only two unused artifact conversion functions create that dependency;
repository-wide Go caller search found their definitions only. Remove those dead
functions/import from typeconv/timeline.go, retain its actively used data types and
retry converters. This enables writer→shared primitive owner without new package.
Initial compile failure is not behavioral red evidence; rerun after this deletion.

025 first correction passes writer/typeconv/protoconv/compiler/executor/telemetry/
workflow races (4.620/1.024/1.044/1.047/5.244/1.017/1.056s). Raw JSON oracle uses
UseNumber independently, preserves9,007,199,254,740,993 and schema/attempt/failure.
Additional nil map/list controls reproduced null→empty loss and now preserveboth.
Cycle/function/nonfinite/overflow evidence rejects before accumulator mutation.

025 measured first writer benchmark20×3:old31.84/39.93/45.79ms and~92.8KB/op;
structured59.99–61.12ms/~373KB. Focused races overlapped this initial cohort, so
latency causality is not established. Later isolated CPU sample30iterations ran
~0.66s total, mostlysyscalls; timing variability must remain visible. Allocation
increase is real for thisfixture. Necessary boundaryextension result_manifest.go:
it serializes all raw artifacts then JSON-decodes/discards them. Clone the proto
snapshot under its lock, omit artifacts before serialization, preserve full
timeline and manifest semantics. Remove unused FileWriter forwardingwrapper.
Validate output/unchanged source and samefixture projection allocation/time.

025 finalsource APIalltests/sevenpackage races/build/contractpass. Owner
20260922-104555-6dcd0052 tidinessfailed4s,quietwaitcompletedonce.1080findings,
long80/complex379/dup606/coupling14,debt33642(-7cycle/-1961original). Newtest
coupling retainedhonestly. Boardprog_3e865214-cf16-41fe-aa27-cf7f3b4d0024 all17
pending/productfalse.4runtimefiles2260→2220lines,62→60functions,515→502cyclo
(-13).51unionpaths/50currentGo:23196→21431lines,806→786func,3911→3737cyclo
(-174),shared009+17widernet-157. No threshold/source-denominator changes.

025 manifestfixture10entries×~557KB each:3×10trials old24.47/25.01/25.23ms,
33.47MB/op;new25.34/28.82/36.93µs,19.3KB/op. Fullsourceequalafterprojection.
Wholewriteproperrepeat old20.19/20.57/21.43ms,new20.70/21.65/20.70ms; no speed
claim. Allocations~93→323KB/op remaincostofstructuredfidelity. Hybridoldconverter/
newmanifest repeat preservedseparately,excludedfromoriginalcomparison.

025 heapfollowup16×3=44152/128870/44408 retainedB/execution;64×3=11182/32261/
32138. Bothcohorts totaldelta~0.7–2.1MB, supportsboundedresidual. Postbenchmark
profilecontainsmostlyruntime/descriptors butextraGCchangedobservation; liveat-
measurement profilepending session6183. No immediateleak/causeclaimfromthatprofile.

025 deploymentgate10:46:32UTC admitted3125rows(2691completed/429failed/5exact
unchangedhistoricRUNNING),driver0/recording0. Managedstopcompleted. make start
pending session37745; completeitandverifyhealth/profile next. No ownerwaitpending.
Cycles024–025 sourcebuilt; do notclaim deployeduntilhealthy. Receiptstructured-
outcome-2026-09-22.json recordscurrentlimits andallbenchmarks.

025 deployment completed healthy2026-09-22T10:50:33Z;buildsha256:10a20446d1a26b20842b967c0ed11451ee8e01815e11ca066dcfafc87dd38f2f.
API/driver/UIhealthy;oneoriginalprofilemetadataequalandallfieldscompareequal
againstoriginalrollback. Two initialprofilechecks usedwrongRPC names(404);actual
browser_automation_studio.v1.session_profiles.SessionProfilesService/List passes.
No productfailure inferredfromwrongmethod. Alltoolwaitscompleted,includingstart
37745,profile2552,liveprofile6183andsettledprofile61167. No pendingoperation.

025 memoryforensics: measurement-time inuseprofile2638.89KB includes1322KB
bytes.growSlice and672KBencoding/json.unquote, withwriteralive. AdditionalGC
same64cohortdiagnostic removesbothsites;60.38B/execution delta. Ordinarysame-GC
measurementsremainrecorded, not replacedbysettlednumber. Supportsboundedserializer
residue, not continuedaccumulatorpayloadretention. No runtimeGCworkaroundadded.
RF023andRF049receipts updated; liveproductionsoak remainsunqualified.

### BAS-WORK-026 — 2026-09-22 UTC — instruction lease admission

RF038/J17. Feedback008 reread; recall found prior BAS operation work, source-ledger
partially degraded with deadline exceeded. Public GoSession.Run sends neither
execution nor lease; driver run route ignores both and may return prior-owner
cached evidence or execute effects. Scope canonical RunInstruction/GoSession,
driver SessionManager lease lookup and actual run route before phase/cache/effect.
Missing credentials reject; stale owner/token or released lease rejects without
refreshing activity. Delayed body is checked after parse against current owner.
Preserve valid execution/recording controls and earlierphase reservation. Prove
real GoHTTP packet and actual driver route plus real Chromium effect counter.
Update driver API example; retain owner fields in fixtures. Full RF038 remains
open for other mutating routes; RF039 dynamic invocation/idempotency is next
separate boundary, not solved by lease validation alone.

Read-only review also finds unused automation/driver/playwright adapter and
RunInstructions plural sending obsolete instruction-array schema; repository-wide
imports found no production caller of that package. Do not preserve an unsigned
path in active RunInstruction for that dead adapter. Remove/replace obsolete
adapter onlyafter separate bounded caller review. E2Erecord-mode script also
uses obsolete untyped instructions and missingstartidentity; qualification not
claimed from thatscript. No source change made for either discovery yet.

026 red: GoHTTP packet lacks execution_id; six actualdriver route cases returned
200 for missing/stale/released lease or ownershiphandoff while body arrives.
Initial focused Jest command also evaluated global coverage with onlyonesuite;
behavioral failures retained, focused reruns explicitlycoveragefalse. No native
floor/configchange. First correction:38driver/nativebrowsercases pass/typespass,
Go session/driver/executor races pass; engine run fixtures failbecause their
synthetic start receipts omit lease_id. Extendexistingengine test fixture responses
with real lease metadata, preserve all response/error assertions. Scope remains
instruction ownership, not wider mutation/identity qualification.

026 finalfocused39driver/nativecases2suites4.322s naturalexit/typespass. FourGo
racespass (session1.064/driver1.029/engine1.062/executorcached). Retainedpublic
execution-api6/8:leasewire nowpasses;retry/loopidentity remainfailed. Nativeeffect
counterprovesold/releasedownership adds0clicks andvalidhandoff permitsnewclick.
Board026prog_f48ebf2a-2772-4686-9c1a-5c904a61e887 admitted;buildsession32829 and
board9816 outputpendingconsumption. Fullunit+tidiness owner20260922-110144-e89cb9f0
admitted11:01:44UTC; one742squietwait attached (recovertoolsessionfromnextrecord).
No production source edits whilethisrunvalidates. Live025healthy;026notdeployed.

026 operations:quietwait76816 active exactlyonce, admission66771 stillattached;
build32829/board9816 completedconsumed. Sourcefrozen. LocalGo1569→1574lines,
98funcunchanged,286→288cyclo;driver1444→1460lines. Cumulative50currentGo
23196→21436lines,806→786func,3911→3739cyclo(-172),shared+17widernet-155.

Read-only RF039 review: session node:index replay and separate global5minTTL
cache collide with intentional retries/loops and permit repeats aftereviction.
Potential replacement: lease-bound monotonic invocation sequence, boundedresponses
plus retained high-water mark, payload digest, terminal uncertain-effect receipt.
Start retry must preserve/restore sequence ownership (GoManager currentlyreplaces
same-session wrappers), reset mustnoterasehighwaterwithinlease, newlease mayreset.
Unexpectedpost-effectthrow mustproduce nonretryable outcome:executor alreadyhonors
Failure.Retryable=false. This isdesignhypothesis, no implementation/scopeswitch.

RF039 additional read-only constraint: executor normalizes a transport error as
retryable unless driver returns an explicit nonretryable StepFailure. Merely adding
a new invocation ID on each GoSession.Run would let a lost response repeat an
uncertain effect as a new logical attempt. Repair must classify ambiguous HTTP/
decode failures as nonretryable or reconcile the same operation before a newintent.
Existing normalizeOutcome preserves supplied Failure; declared retry respects
Retryable=false. Global idempotency cache has no other productionconsumer beyond
run/close/reset; one replacement owner can remove its timer/TTL/cache duplication.
Do not implement justUUIDs as a superficial retryfix.

026 fullownerterminal456s(unit452s):API24.336/CLI0.866/driver351.922/UItypes11.336s
pass;driver126suites1523tests351.097s naturalexit,2skip. UImergedcoverage28.57/
30.34/64.96/28.57vs85remainsfailed. Tidiness1081,long80/complex379/dup607/
coupling14,debt33632(-10cycle/-1971original). Quietwait76816completedonce;
no ownerpending. Currentretaineddriverprobe fixture nowcarriesvalidlease onnormal
requests, seedsownerfromrealGo packets, andcachehandoffcaseusescurrentnewowner
credentials so itstilltestsoldcachepayload ratherthansimplyrejectionofstalelease.
Expectedbehaviors unchanged; syntheticmiddleware scopeexplicitlyretained.

026 retaineddriver8/14 afterfixtureupdate; allremainingfailures preserved. Receipt
instruction-lease-2026-09-22.json recordsfullunit/tidiness/local+cumulativecost.
Freshdeploymentgate11:12:14UTC:2998historyrows(2560completed/433failed/5unchanged
historicRUNNING),driver0/recording0. Managedstopcompleted11:12:17UTC. make start
nowpending (toolIDrecordnext). No userdata/historydeletion performedhere; differing
historycounts notattributed. No TestGenieownerpending; do not re-admit026.

Next RF039 design constraints (read-only, not yet implemented): wire logical
invocation+attempt separately from node/index and transportsequence. Executor
allocates invocationonceperrunWithRetries; loopiterations thereforediffer and
declaredattempts incrementwithinthatscope. GoSession owns monotonictransportseq
perlease; samelease startretry/recreatedclient mustnotresetitscounter (driverstart
receiptcanreturnhighwater; GoManager shouldreusematchingactivewrapper). One driver
lease-owned boundedresponsemap+highwater can rejectevictedoldseq withoutunbounded
tombstones; resetclearsresponses butnotleasehighwater, handoffnewleaseresetsboth.
Bind admittedseq to payload and invocation/attempt. Cacheuncertainthrown outcome
asnonretryable, preserveit ontransportretry. Classify ambiguousclientHTTP/decode
failure asnonretryable instead ofmintingnewintentviaautomaticexecutor retry.
Do notsilentlyignoreoldidempotencyheaders; settleexplicitinternalprotocol and
convertallactiveclients/tests/docs when replacingglobalTTLcache. Bytebounds and
processrestart applicability needexplicitlimits; no in-memoryexactlyonceclaim.
No sourcechange forRF039 yet. Continueafter026health/profile verification.

026 deploymentcompletedhealthy2026-09-22T11:16:23Z;buildsha256:99b935baabc594de2fbc627c8a6a1509db4d5f8b16c8f5f596f5feeeee68bb2d.
API/driver/UIhealthy,driversessions0. Oneoriginalprofilemetadataequal;alloriginal
fields preserved inthree repositoryreads. Originalrollbackuntouched. Admission,
quietwait,build,lifecyclestart15541andprofile84696 completed; no pendingtool/owner.
Next RF039researchalreadyrecalled with026, no repeatdiscoveryneeded.

### BAS-WORK-027 — 2026-09-22 UTC — invocation and transport ownership

RF039/J17 withRF041 uncertain-handler boundary. Feedback008 reread. Recall026
already covers thisowner; no applicable implementationprogram. Baseline remains
two failingpublicGo retry/loopidentity probes and sixdriveridentity/evidence gaps.
Replace node:index and globalTTL replay policies with one lease-owned bounded
response map and monotonic transporthighwater. Executor allocates invocation per
logical runWithRetries call andattempt per declared retry; GoSession sequences
each transportoperation andtransportsitsmetadata. A repeated same sequence/payload
returns its retained outcome; changed payload conflicts; an evicted/reset response
never re-executes oldsequence. Rejectedbusy/stale operations cannotadvancehighwater.
Newlease resetssequence; samelease startretry/Go wrapper preserves/restoresit.

Wire protocol becomes explicit: positive safe-integer operation_sequence plus
invocation_id andattempt. Optional X-Idempotency-Key mustequal lease_id:sequence;
canonicalGo sender suppliesit. Obsolete arbitrary-header identity is rejected
explicitly, not silentlyignored. All activecallers/examples/probes converted.
Source contracts carry runtimeattempt metadata outside saved workflowdefinition
(JSONexcluded); no new userworkflowformat/runtime migration. Step outcome retains
correlation andactualattempt. Ambiguous transport/decode failure is nonretryable;
post-admission handler/telemetry exception yields retained nonretryable uncertain
outcome. Declared retry of known returned transientfailure remains allowed.

Scope: Go contracts/executor/session/manager/client/types andexisting tests; driver
run/start/leasehandoff/reset/type boundaries; delete global idempotency cache and
node-key helper/exports/calls. Replace tests of obsolete private-map mechanics
with publicroute/native effects, preserve desired duplicate-suppression controls.
Do notdelete activecapabilities or weaken thresholds. Keep phaseadmission policy:
inflight duplicate mayreceive409 andmustnot createanewintent automatically.
Boundedcacheeviction denies replay whenreceiptunavailable; no processrestart/
exactlyonce claim. Saveddata preserved. Beforecopies:/tmp/bas-before-027.

This scope is now selected; no027runtimeedits yet. Unit026/deployment/profile
verification completed, no priorowner pending. RealHTTP/publicexecutor packet
oracles plus nativeChromium counters, conflicts/eviction/reset/handoff/uncertain
effects and naturalexit checks precede ownerfullsuite. Performance/debt measured
against beforecopies; capture unknowns ratherthan narrowingtests.

027 implementation checkpoint: Go runtime6files +63lines/+1function/+12cyclomatic
(4644/172/737 ->4707/173/749). Driver affected runtime3837->3366lines (-471),
including full335line globalTTLcache deletion and node-key helper deletion;
no TSAST cyclomatic claim. Initial snapshots omitted proto/instruction.ts and
outcome/outcome-builder.ts; their before copies were reconstructed by exact
inverse of this cycle's additive runtime metadata and attempt/notes edits.
Cumulative expanded54-path union (53original and53currentfiles)24446lines/
831functions/3997cyclomatic ->22749/812/3837 (-160); shared009+17 yieldsnet-143.
No thresholds/floors/config denominators changed. Removed15 tests that asserted
literal keys, copied Map mechanics or the obsolete node-key helper; replaced
behavior with real route/native controls, including >1000-effect eviction.

Evidence: /tmp/bas-operation-go-red-027.txt and native-red-027 demonstrate
pre-repair identity/loop suppression and missing uncertainty policy. Initial
focused driver121tests5suites9.331s; expanded110tests5suites13.737s naturalexit.
Final Go races session1.061/driver1.023/engine1.058/executor3.551s. Publicexecutor
four realHTTP cases prove retry metadata, loop visits, malformed response and
lost-response no-new-operation behavior. Initial one-shot-loss fixture exited1;
final ambiguous-loss fixture keeps every retransmission lost, because an HTTP
transport may safely recover by repeating the identical packet. Original first
fixture log was overwritten; do not claim its exact assertion as retained proof.
Native Chromium post-click handler/audio-decoration throws retain same uncertain
receipt and one effect; attempt2 and invocation/operation correlation preserved.

Retained probes API8/8;driver13/14. Synthetic middleware maps all exceptions to500,
so header-lease binding probe now checks exact rejection message and unchanged
effect count; real route tests establish400. Original arbitrary global header
protocol intentionally removed; current valid headers identifylease:number.
Historical receipts remain unchanged. RF041 thrown-handler evidence gap still
fails: zero console/capture despite retained uncertain outcome. No qualification
for restart/exactlyonce, activebytebudget, diagnostic preservation orwholejourney.

Build/contract passed; board02717pendingfalse. Full unit+tidiness admittedonce
20260922-114025-ba3c12e3; quietwaitsession80099 pending. Othertool sessions consumed.
027 notdeployed; live026build/profile receipt unchanged. Next: finish owner,
sequential synthetic overhead trials, record debt/currentreceipt, freshdrain gate,
lifecycledeploy/profilecompare; then repair RF041 diagnostic loss or confirm
unlinkedlegacyadaptercallers before deleting. No external worklog/plan operation.

027 ownerterminal20260922-114025-ba3c12e3 failed399s(unit395): API31.951/CLI0.936/
driver291.276/UItypes10.730s passed. Driver125passedsuites/1526passedtests290.522s,
1suite/2testsskip; naturalexit. UI mergedcoverage remains28.57/30.34/64.96/28.57
vs85. The partialVitesttable83.62isnotmergedfloorproof. Native tidiness1085findings,
long80/complex380/dup609/coupling15,debt33638 (+6cycle/-1965original). PublicHTTP
regression adds15complexity test andexistingfile22imports; not moved/split togame
thresholds. Localnewcode canincrease these readouts despite removal ofsecondcache
policy; cumulativeGo/owner/netline reductions reported independently.

Corrected equalserialization syntheticbenchmark (5000distinctops,3trials;5000effects
andrepeatcontrol allpass):before15.493/12.416/12.817microseconds/op;after19.490/
16.630/16.084. Median+3.813microseconds (~29.7%); added canonicalpayloadsort/hash
andimmutable receipt handling are new required bindingwork. No browserlatencygain
or bandrelaxation. Initialbaseline skippedresponseJSONserialization,excluded;
firstcorrection lackedbaseline res.end andfailed; finalbefore/after-027json are
comparable. NegativeheapdeltasreflectVM/compilationgarbage; no qualifiedretained
memoryclaim. Sourcecopies/scripts kept /tmp forreproduction, notnewproductioncode.

Gate0272026-09-22T11:49:50.977513Z:2998rows,2560completed/433failed/5exactunchanged
historicalRUNNING,driver0/recording0; freshgenerationchecked,observationalnotatomic.
make stopcompleted;make startsession82886 pending. Oneoriginalprofileuntouched;
postdeployhealth/profilecompare next. Ownerquietwait80099 consumed, no ownerpending.
Recall028 /tmp/bas-failure-evidence-recall-028.txt:70hits,priorcapturepolicywork,
no matchinghandlerfailure implementationprogram; source-ledgeragent-memory/scopes
requests timedout butothercorporaavailable. Preservepassivetelemetrypolicy.
Read-only RF041 confirms catchdisposes/rethrowsbeforefailurecapture; telemetry
collect/rejectedmetrics also lackswhole-pipelinefinally. No028sourceedits.

027 health/profileverificationcomplete2026-09-22T11:52:38Z;build
sha256:dca16c2b2fb6f2d38c09ae2df27f8ec84f430bbb8b7ec2b263c84dd73d08cfc5.
API/driver/UI healthy. ProfileAPI metadataequal026; everyoriginalsavedfield equals
untouchedrollback inthree repositoryreads (firstkeyringread1215ms; then0.35/0.29ms,
notacomparablelatencytrial). Start82886/profile1572 consumed. No pendingowner.

Read-only reviewfoundRF062: native027zero driver-filefindings althoughlargeTS
modulesexist. ActualsharedLanguageDetector scansonlyapi/ui/src/cli; usedbylight
scanner,detailedmetrics,handlers. RecordedasRF062, notRF061retiredduplicate023.
Native027debtincrease/overallreduction honestbutdoesnotmeasureTSdriverremoval.
Nextboundedownerinvestigation/necessaryextension willrepairinventoryusingexisting
sharedsurfaceauthority andretainexpandedbefore/currentdenominators. No028edits.


### BAS-WORK-028 — 2026-09-22 UTC — complete maintainability inventory

RF062/W2 measurement gap; bounded necessary shared-owner repair underFB008.
Feedback008 reread. Recall /tmp/bas-tidiness-inventory-recall-028.txt:68hits,
no applicable sourceinventoryrepair program; two source-ledgercorpora timeouts,
otherprovidersavailable. FirstCodeFactscall used unprefixedtarget andfailedpath
resolution; corrected scenario:browser-automation-studio returnsdeclaredsurface
receipt /tmp/bas-driver-surfaces-028.json. No capabilityabsenceinferred.

Two causes: shared api-core/pathfilter excludes literalplaywright-driver asdata;
TidinessManager LanguageDetector separatelyhardcodesapi/ui/src/cli. Its existing
filemetrics inventory alreadywalkstargetrootwithsharedfilters. Earlierintentto
addCodeFactsdiscoverytoruntime isrevised: CodeFacts confirmscomponentownership,
butmaintainabilityalso scanssupportingsourceoutsidecomponents. Reuseoneexisting
localinventoryforlanguagegroups andallmetricfamilies, remove duplicatewalks;
no new remote dependency or privatehardcodedsidecarlist. Targetdocsupdatedfirst.

Scopeextension: packages/api-core/pathfilter/{pathfilter.go,pathfilter_test.go};
scenarios/tidiness-manager/api/{language_detector.go,language_detector_test.go,
light_scanner.go,handlers.go,validation_connect_test.go} andownerdocs. Clean at
entry; beforecopies /tmp/bas-before-028. No dependency/schema/productdatachange.
Redgate: publicnativevalidation mustreportlength/couplingforPlaywrightandcustom
workerTS sources, preservevendor/build/dataexclusion; detectorusesexactsource
inventory. Sharedfilter source-directory regression. Nativeexpandedmetricswill
rise; preserveoldreceipts/ratchets andremeasureequalbefore/currentcoverage.
Then focusedownerchecks,scopedTestGenieownerphases, lifecycleownerdeployment,
BASnative-tidiness receipt andboard. No028runtimeeditsyet. RF041 remainspending.

028 redconfirmed: sharedSkipDirwronglytrue; nativevalidation missesbothdriverlength/
coupling andcustomworkercoupling; detectorreturnsnilTS. Focusedfirstrepairpasses
7.889s. During review, initial singleinventoryrefactor narrowed incremental-mode
language metrics to changedfiles. Preserve existingfulllanguage semantics by
collectingonefullinventory andfilteringonly returnedincrementalfilemetrics;
removeits duplicatewalk. Add realPG publicScanWithOptions regression inexisting
light_scanner_integration_test.go (beforecopyextended), andupdatepathfilterdoc.go.
No candidatepublished; this is correctionofownin-progressrefactor, notnewissue.

028 implementation: removedsharedplaywright-driverdirectoryexclusion; language
classification derivesfromexistingfilteredfilemetrics; nativefindingsreuseits
filelistsratherthanrescanningperlanguage. Incrementalmode deriveschangedoutputs
fromonecompleteinventory whilewholelanguageanalysis remainscomplete. Deleted
secondincrementalwalk, perlanguagefileslookup, excluded-fileforwarder androot/ext
wrappers. Shared5Go files1909lines/63functions/318cyclomatic ->1710/58/288
(-199lines/-5functions/-30cyclomatic). Includespathfilterdoc.go; noexporteddebt.
BASGo cumulative -160, prior shared009+17 and028-30 => affectedwidernet-173.
Sourcebefore /tmp/bas-before-028;metrics /tmp/bas-complexity-shared-028.json.

Red: /tmp/bas-filter-red-028.txt; /tmp/bas-inventory-red-028.txt. Incremental
correctionactualred /tmp/bas-inventory-incremental-red-final-028.txt showszero
changedfilesbutmissinglanguagecoverage. First attemptusedwrongcwd; nextfixture
storedlocaltimeintotimestamp-without-timezoneandincorrectlymarkedfileschanged;
correctedfixtureusesUTCfuturetimestamp. No productiontimestampchange. Final
focusedrace owner7.322s andsharedfilter1.013s pass. Beforecopyextendedfor existing
incrementaltest andpathfilterdoc. OriginalHEADBASarchive retained at
/tmp/bas-expanded-original-028/scenarios/browser-automation-studio (2199tracked
files/19MB); not yet scannedbyrepairedprovider. TypeScriptcomplexity analyzer
explicitlyunsupported; applicablelength/duplication/coupling coverage is repaired,
not a TSASTcomplexityclaim. Existinganalyzererror/reportlimitsremainunqualified.

Fullownerunit+tidiness20260922-120534-16d6f5c6 admittedonce. QuietwaitONCE pending
(/tmp/bas-inventory-owner-wait-028.json; keep returned toolsession); admission
session16380 maystillstream. Build16368completedlog buttoolnotyetconsumed;
board71199 pendingconsumption. Noownerlifecyclepublicationyet; BAS remainshealthy
027. Nextextractownerreceipt, deploytidinessownerthroughmake, scanexpandedoriginal
andcurrentusingfreshnativeprovider, keepoldnarrowmetrics andunchangedratchets.

028 owner run is terminal (failed, 70 s); admission/wait/build/board sessions
consumed. Unit native: 29 findings, 11 blocking. API exceeded unchanged 60 s
no-output watchdog; CLI/UI tests/UI types passed. Ten other blockers are existing
Tidiness UI coverage/import-policy projection gaps. Tidiness phase passed using
the older live provider and cannot certify the new inventory.

The diagnostic coverage command completed naturally in 91.909 s, coverage 72.0%,
with eight trimpath-broken fixture failures and one repository executable-budget
failure (864 observed, 472 declared). No SIGQUIT stack exists; the process had
finished before the attempted signal. Earlier inference of startup/init hang was
wrong: multi-package Go output was buffered. Do not classify this as a crash.

Necessary owner extension BAS-RF-063: Unit Health's Go evidence adapter rejects
the absolute executable path supplied by its planner, omitting existing streaming
JSON/fresh-execution instrumentation. Fix that adapter and exercise the actual
resolved command with its native evidence test; no timeout/floor changes. Also
repair Tidiness's existing liveRepoContract fixture to resolve from the package
working directory, which survives -trimpath. Before copies retained in028.
Repository-wide executable debt and unrelated Tidiness UI policy drift remain
visible; this cycle does not authorize deleting hundreds of other owners' outputs.

028 extension validation: resolved-executable native regression failed before
repair (canonical Go command unsupported); after repair gotest and validation
races pass in1.270/2.821s. Preserve JSON/-count=1 instrumentation, selected binary,
coverage flags; accept native go/go.exe basenames. Owner builds pass. Original
pathfilter overlay also observes864executables vs472, proving that failure
predates028. Keep the budget untouched. Fixture repair removes runtime.Caller
source-path dependence; full JSON diagnostic running with only executable-budget
failure observed so far.

028 shared-owner lifecycle gate: no campaigns, no API child work processes;
only established connections for ports16792/15317 belong to runtime supervisor
health checks (PID3159738). Existing server shutdown drains requests. Gate is
observational, not atomic. Restart Tidiness and Unit Health through make after
focused/build validation; no BAS restart or product-data mutation is required.

028 full streaming API diagnostic terminal90.429s:718passes,14skips,1failure
(repository executable budget, reproduced unchanged with original filter).
No remaining fixture assertion failures. Evidence /tmp/bas-inventory-api-final-028.jsonl.
Restart commands are still active; 12:17:50 health reads showed old uptimes/builds
and therefore do not establish publication. Preserve tool sessions71385/62980.

028 shared restarts completed healthy: Tidiness12:19:39Z build3ffde178...;
UnitHealth12:20:09Z build2971b6b6.... CodeFacts rebuild-only preserved running
shared service; UnitHealth lifecycle also refreshed QualityHealth dependency.
No BAS restart. All restart sessions consumed. Expanded native scans retained
in /tmp/bas-expanded-{original,current}-028.json: original1131findings,
105long/384complexity/628duplication/13coupling/debt35603; current1121,
103/387/613/17/debt33666. Each has23driver long and1driver coupling finding.
Observed scan wall times5.266/4.764s are single reads, not a performance claim.

RF064 discovered: JS duplication adapter ignores inventory and hardcodesui/src;
malformed stdout becomesempty, and skipped/error analyzers are not coverage
receipts. Expanded totals therefore qualify supported observations only, not
whole-codebase TScomplexity/duplication. File tracking owns this out-of-slice
finding underFB008; no external journal/bug write. No dependency install attempted.

028 final Tidiness owner run20260922-122015-cbf7d8e5 admitted; quietwaitONCE
active tool81628. UnitHealth admission was rejected by caller preview capacity,
so no UnitHealth run exists; retry only after the current owner run completes.
No duplicate admission. Extension execution.go169lines/3functions/54cyclomatic
->170/3/55 (+1). Shared028 net -198lines/-5functions/-29cyclomatic;
cumulative BAS -160 +shared00917 +shared028(-29) = -172 affected net.

028 completed validation: Tidiness20260922-122015-cbf7d8e5 failedunit121s,
passedtidiness2s. API113.115s now reports test_failure; CLI0.863/UI4.178/
UItypes2.131spass. UnitHealth20260922-122233-b1f4bc2d passedunit12s with0findings
(API4.306/CLI0.625/UIcoverage3.891/UItypes2.767), failedtidiness2s. BASnative
20260922-122311-0fd78914 failed with expanded1121findings/debt33666, matching
directcurrent. All admitted quietwaits consumed; admission rejected before
UnitHealthrun was retried only after Tidiness terminal. Board final028:
prog_112922f0-8ab9-44bd-a6e9-fab640816076,17pending/productfalse. Receipt
maintainability-inventory-2026-09-22.json retains hashes and scope limitations.
No pending validation/lifecycle. RF064 remains open; next BAS behavior workRF041.

### BAS-WORK-029 — 2026-09-22 UTC — instruction failure diagnostics

RF041/J18 bounded BAS behavior repair. FB008 reread; recall already retained at
/tmp/bas-failure-evidence-recall-028.txt, no applicable existing repair workflow.
Hypothesis: handler throws bypass capture and dispose buffered diagnostics;
metrics/capture/build failures also bypass the sole success-path disposal.
Target architecture updated first. Before copies /tmp/bas-before-029 retain
executor/orchestrator and existing unit/native test files. No product data or
schema change. Add native real-click failure evidence/replay oracles and cleanup
fault controls, then simplify to one final ownership release. Preserve optional
capture rules and existing native timeout behavior; do not introduce a racing
background capture or claim hang qualification. Shared028 remains deployed.

029 initial red /tmp/bas-failure-evidence-red-029.txt:7failed/29passed across3suites.
Implementation: one executor finally owns disposal; handler exceptions become
nonretryable uncertain results before captures. Capture channels fail independently
and retain errors in outcome notes; metrics observers use existing safeInvoke.
Known handler success plus unexpected capture error becomes nonretryable evidence
failure; preexisting handler failure retains its cause. Typecheck passes. First
focused after:35passed/1nativeconsolefailure. Screenshots/DOM exist in thatfailure;
page-evaluate andmainworldonclick logs both absent under default Rebrowser.
Diagnostic REBROWSER_PATCHES_RUNTIME_FIX_MODE=0 passes same2nativefailure cases
(/tmp/bas-failure-native-runtime-control-029.txt). Installed crPage.js:429 guards
Runtime.enable onthatsetting. RF065 recorded; no production env/dependencychange.
Retained execution-driver producer nowpasses14/14 (/tmp/bas-failure-retained-029.json).
Pending expansion to console ownership; beforecopies /tmp/bas-before-029.

029 necessary RF065 extension: telemetry/collector.ts and existing collector
unit tests; synthetic execution probe must model acknowledged CDP console events.
Before copies extended. Existing browser owner only launches/connects Chromium;
no alternate-engine capability is removed. Use a dedicated per-instruction CDP
console source, awaited startup/detach, no page-script wrapper or process-wide
patch override. Runtime observation is necessarily enabled during requested
console capture; anti-detection/worker/OOPIF implications are unqualified and must
remain explicit. Recall /tmp/bas-console-recall-029.txt returned63hits; no existing
console-provider repair program. No dependency or native browser patch edited.

029 default-mode native console repair passes: dedicated CDP listener consumes
Runtime.consoleAPICalled, filters pre-window history, preserves primitive text
without evaluating remote object getters, and detaches after initialization or
execution failures. Startup/cleanup are awaited; delayed attachment disposal
remains owned. Old Page console bridge removed, not retained as a fallback.
Native fixture proves real onclick console + PNG/DOM, same immutable failed
receipt on retransmission, one click, and detached CDP session rejecting later
commands. Focused205tests/16suites9.221s naturalexit, types pass. One test-edit
mistake made describe async; fixed before final expanded run, no production fault.
Retained driver producer updated only its synthetic CDP interface and nowpasses
14/14 with zero remaining console sessions/listeners. Beforecopiesextended.
Affected3runtimeTS files1012->1016lines(+4); no TSAST complexity or latency gain
claim. Existing capture helpers can still return undefined for native capture
failures; worker/OOPIF/transport hangs and anti-detection impact are unqualified.
Fullownerunit+tidiness admission and makebuild pending; BAS029 not deployed.

029 build passed; contract preparation valid/productfalse. Board
prog_f1cd54e3-0e1d-460e-a246-04074d7c7956:17pending/productfalse. Fullowner
20260922-124226-6f1b08ac admitted once, quietwaitONCE tool27351 pending;
admission24111 still available. No BAS lifecycle publication yet.

Native collector setup+acknowledged-dispose benchmark:3trials x50samples after3
warmups, original medians0.11755/0.05047/0.05171ms; current0.87231/0.84458/
0.77651ms. Median-of-medians delta+0.79287ms. This is expected added attachment/
Runtime-enable/detach work, not equal functionality (original default has no
console events) or whole-workflow latency. TypeScript modules are actual before/
current with identical constant/normalizer seams; no events emitted. Firstbench
assumed per-module dist files, but build is bundled; that failed import is excluded.
Final /tmp/bas-console-cost-final-029.json; script /tmp/browser-automation-studio/
console-cost-029.cjs. No threshold or timeout changed; no speedup claim.

029 owner terminal440s (unit435/tidiness5):API31.564/CLI1.987/driver332.436/
UItypes10.353spass;125driversuites1532tests331.72s naturalexit,1suite2testsskip.
UI56.113sfailsunchangedmergedcoverage28.57/30.34/64.96/28.57vs85. Native expanded
1121findings,103long/387complexity/613duplication/17coupling/debt33666 unchanged
from028. Fullownerreceipt /tmp/bas-owner-findings-029.json; no assertion masked.
All owner/admission/build/board tool sessions consumed.

029 deployment gate12:50:59UTC rejectedstop:2834historyrows,
2433completed/395failed/6RUNNING, including new currentexecution
630342bc-f422-4760-b2b4-79d677abb02e plus5unchangedhistoricalorphans. Driver1session,
0recordings. No stop was performed; livebuildremains027dca16c2b.... Historycount
changed since027; no attribution inferred and no history/profiledatawritten here.
Receipt /tmp/bas-drain-deferred-029.json. Recheck only after useful independent
work; never kill current execution to publish. Cycle029candidate evidence updated.

Read-only next-slice review: legacyGoDriver adapters have no repository Goimports;
38qualifiedreferences allinside adapters/tests. Alias-aware audit found only
Point tokens outside them, which inspection resolves to contracts.Point (not
interface.go Point). /tmp/bas-legacy-driver-alias-audit-030.json and
/tmp/bas-legacy-driver-symbols-030.txt retain evidence. No030sourceeditsyet.

### BAS-WORK-030 — 2026-09-22 UTC — retire unused Go driver model

RF066 bounded maintenance task while029publication waits for active browser work.
FB008 reread; recall /tmp/bas-legacy-driver-recall-030.txt. No applicable replacement
workflow. Targetdocs updated before source. Full repository search:52imports of
maintained driver package, none of oldadapterpackages; two outsideAPI are retained
profile probes and do not use retired symbols. Alias-aware reference audit's only
apparent non-adapter Point uses are actually contracts.Point. All38qualified
legacyinterface references are inadapter/stub/tests. Plural transport and request
wrapper have onlyadaptercallers; typed request wrapper also unused. Beforecopies
/tmp/bas-before-030 preserve current client/types including prior cycle edits.
Retire these unused packages/types/transport and7mechanicaltests, with active
GoSession/client/executor/recording and navigator behavior unchanged. No new test
needed to mirror deleted dead declarations; use existing public behavioral tests,
Go races/build and scoped native owner validation. No product-data mutation.
Cycle029remainsvalidatedbutnotdeployed; no active testowner before030edits.

030 removal implemented:3unused runtime files +their7mechanicaltests retired;
obsolete plural transport andbothunused request wrappers removed frommaintained
client/types. No caller conversion needed; complete symbol/importinventoryshows
no active caller. Five runtimepaths2736lines/92functions/301cyclomatic ->
2files1803/63/215: -933lines/-29functions/-86cyclomatic. Expanded57-pathGo union:
original56files25362lines/859functions/4082cyclomatic ->current53files22732/811/
3836, BAS -246; prior shared009+17/028-29 yieldsnet-258. No exporteddebt.
Metrics /tmp/bas-complexity-{local,cumulative}-030.json; originalHEAD retained.
Focused driver/session/engine/executor races1.024/1.067/1.060/3.695spass.

030 validation uses discovered UnitHealth workspace filter: canonical execution
withcoverage forapi+cli, preserving029fullpasseddriver result because driver
source is unchanged. UnitHealth tool84972; nativeTestGenietidiness admission74462;
makebuild67472. No changedfloor, fast-test-only switch or weakenedrolepolicy.
This is a scoped receipt, not a fresh whole-product unit verdict. BAS remains027
live until a fresh quiet gate permits publication of029+030 together.

030 verification terminal: canonical scoped API36.090s andCLI2.254s passwith
coverage. Provideroverallreports2falsemissingroles (excludeddriver/UI), RF067.
No role waiver/floor change. Tidinessrun20260922-125645-c641df05 terminalfailed:
1117findings,103long/386complexity/610duplication/17coupling/debt33616
(-50cycle/-1987expandedoriginal). Buildpasses. All030tools consumed; board
receipt /tmp/bas-board-030.txt. SourceprogramNo nativecapabilitytestsremoved;
only7mechanicaltestsofthedeadadapterswereeliminated.

030 deployment gate rejected beforestop because offsetpagination repeated anID
whilehistorychanged (/tmp/bas-drain-stop-attempt-030.txt). No stopperformed;
follow-updriverhealth shows10sessions,0recordings, confirming activework. Do not
relax the consistency guard or terminateothers' browsers. 029+030candidate remains
readyforquietpublication, with UIcoverage/debt limits explicit. Next independent
repairRF067 belongsinUnitHealth; no pendingTestGenie/build/lifecycle atthispoint.

## 031 — required-role evidence under workspace selection (2026-09-22 UTC)

RF067 necessary scope extension: Unit Health owns the false missing-role findings
in BAS030's canonical API/CLI validation. Target architecture is updated before
source. Affected owner paths: api/internal/validation/{service,plan}.go and their
existing tests. Preserve complete discovery for global policy and selected
inventory for planning/analysis; do not weaken roles, floors, or waivers.
Hypothesis: the selector erased observed UI/driver roles before policy resolution.
Discriminating checks: present-but-excluded role causes no missing-role finding;
actually missing role still fails; unknown selector schedules no commands.
Recall /tmp/bas-workspace-policy-recall-031.txt returned results with degraded
source-ledger providers; no suitable program repairs this implementation boundary.
Before copies /tmp/bas-before-031 retain shared-tree source. Focused race checks,
owner unit/tidiness phases, and native BAS API/CLI scoped validation will qualify
the change. BAS029/030 publication remains deferred while browser work is active.

031 result: red regression reproduced excluded CLI/UI as missing; complete
discovery fix passes validation race2.282s and makebuild. Unit Health published
healthy13:11:24UTC. Owner20260922-131130-e1ccd657: unit13s passed/zero findings,
tidiness2s failed existing budget. Canonical BAS API35.493s/CLI2.069s pass with
coverage, zero required-role findings,55warnings; no floor or profile change.
Runtime two files1639->1641lines,39functions/272cyclomatic unchanged; affected
cumulativeGo delta remains-258. Board17pending/productfalse.
BAS029/030 finally published after gate13:10:32UTC found zero sessions/recordings
and exact historical-five orphans; clean stop/start healthy13:12:35UTC. Metadata
and complete original saved-profile identity preserved. Both deployments and all
validation sessions consumed. Receipt scoped-unit-policy-2026-09-22.json.

## 032 — execution-history filtering and pagination (2026-09-22 UTC)

RF054: handler ignores status, total is page length, has_more guesses from page
fullness. Repair the repository/workflow/handler boundary and migrate all list
callers to one query, removing duplicate status-list methods and retention's
optional fallback. Count/page share the routed transaction; preserve existing
retention selection budgets, execution history and saved results. Public bounds
follow the existing proto contract. Target architecture updated before source.
Recall /tmp/bas-execution-filter-recall-032.txt returned BAS usage programs, no
query-repair workflow; source-ledger providers degraded. Before snapshots are in
/tmp/bas-before-032. Real temporary SQLite plus public Connect requests will
check intersecting filters, equal-time ordering, middle/final/empty pages, total,
has_more, default/bounded limits and invalid requests. Existing retention and
recovery regressions remain required; no live history mutations are authorized
by this read-path repair.

032 caller audit found the global UI asks for200 rows and workflow histories
expect an unbounded list when limit is omitted. Necessary boundary extension to
existing ui/src/domains/executions/services/executionApi.{ts,test.ts}: aggregate
bounded public pages, preserve exportability and caller limits, reject broken
non-advancing/duplicate pagination rather than return incomplete history. No
new view/state owner. Initial total bounds the aggregate under growing history;
multiple requests still do not provide a snapshot. Add routed-pool read checks
to the existing database connection regression before publishing.

032 result so far: six race packages pass; focusedUI8/8 and types pass. Canonical
API39.743s/CLI2.297s pass; UI55.562s fails unchanged85% coverage requirement
(28.59/30.34/65.15/28.59); UItypes10.626s pass. Makebuild passes. Tidiness
20260922-132426-85e2b06f terminalfailed, quietwait consumed; debt33524(-92cycle).
Final metric includes compiled Go test helpers: -47lines/-4functions/0cyclomatic
plus UI+25lines. Expanded65original/62currentfiles30110/27433lines and
4952/4706cyclomatic; BAS-246 + shared-12 => affectednet-258. Domain-only
-7cyclomatic is offset by mock updates; do not report it as overall cycle gain.
Quiet matched-size query medians181.390->253.381us (+71.991/~39.69%) after
coverage completed. Extra count+read transaction+tie sort supply previously
missing behavior; no whole-journey speed claim. Initial test fixture pointer typo
was fixed before valid red receipt. Gate13:27:14UTC admitted cleanstop; start
session30071 pending at checkpoint. Boardprog_94bdf9ed-0f67-45c8-8581-6e0b383f6b16
17pending/productfalse. Receipt execution-query-2026-09-22.json.

032 publication complete: healthy build27b9cd6d7734e44874e92bb6e3213ffb3dc83873696d10f9433f34e642a7c13b
at13:31:09UTC. Native read verifies pending0/running5, exact historical IDs,
total2833 for page1 and empty-offset page, default50. Saved profile metadata
equals031; three complete identity reads preserve original rollback fields.
No pending lifecycle/build/test waits. Recall033 finished; no033source edits.

## 033 — recording acknowledgement boundary, investigation (2026-09-22 UTC)

RF002 persists beyond the committed Go journal repair014/015. Source audit finds:
browser send/retry treats any resolved fetch as success, removes pending entries
by timestamp, drops old/overflow/retried entries, and clears storage during
recovery/reinjection; raw events lack stable identity across retransmission.
Driver event-route invokes a void callback and returns200; pipeline fire-and-
forgets async entry delivery; callback circuit skips/failures return success;
stop transitions ready without joining deliveries; local buffer evicts and clear
reads erase before delivery. Current Go ingress commits before200, so it is not
the remaining false-ack source. Fix must include stable delivery identity,
acknowledgement propagation, pending retention and stop/close/reset ownership,
not just retry HTTP. First add actual Chromium fault regressions to existing
tests/integration/pipeline-e2e.test.ts for thrown delivery and delayed delivery
at stop. No033runtime source edits yet; live032 remains healthy. Recall finished
/tmp/bas-recording-delivery-recall-033.txt; no replacement workflow found.

033 native investigation result: all three new desired-behavior tests fail under
actual Chromium in1.550s with natural exit (seven unrelated cases filtered).
Raw click has no retry identity; rejected async callback still yields HTTP200;
stop reports success before controlled commit release. Initial two-case1.408s
receipt retained. No runtime changes; live032 remains healthy. Target architecture
now records delivery/identity/retention invariants before implementation. Receipt
recording-delivery-investigation-2026-09-22.json. All test/tool sessions consumed;
no pending operations. Next implement the complete acknowledgement owner boundary
without weakening the three native assertions. Retain no-callback/pull mode, and
keep driver/browser-crash guarantees unknown until directly qualified.

033 implementation boundary prepared: existing browser script/init generator,
recording buffer/event route/initializer/pipeline, raw-event conversion, callback
and recording routes, session reset/teardown, plus Go driver/pull ingress where
explicit acknowledgement must follow persistence. Before copies under
/tmp/bas-before-033 preserve shared audio/session work. Planned replacement uses
one browser pending queue (mirrored to sessionStorage), one driver entry owner
with stable identity/sequence and joined delivery, and no silent circuit skips.
Current-frame stop handshake must flush input and wait for delivery; failed
stop retains ownership for retry. Pull mode needs an explicit acknowledgement
after the caller commits, with Go ingress using the existing journal. No runtime
replacement is deployable until all three native fault cases and affected
retention/close/reset/pull cases pass.

033 interim edits (not deployable): browser now retains one serial pending queue
with UUIDs, checks HTTP/application acknowledgement, and starts dormant until an
acknowledged control handshake. Native MessageChannel probe passed on installed
default Rebrowser. The browser retry case now passes; async-route and stop cases
still fail (interim1pass/2fail,2.635s). Replaced buffer's four maps with one bounded
entry/delivery/receipt owner; pending entries cannot be evicted, cleared or
removed. Buffer integration, raw identity/sequence reuse, async route/callback
propagation, stop/reset/close flush, pull acknowledgement, and tests are still
required. New buffer APIs are not yet wired. No pending tool/test processes.

033 joined-delivery checkpoint: driver admission now awaits one ordered buffer drain, retains stable browser ID/driver sequence, and propagates callback errors. The silent callback circuit and duplicate route buffer ownership are removed. Native fault cases now 3/3 pass (7 filtered, 3.233s; /tmp/bas-recording-ack-wired-033.txt). Stop joins the browser control handshake and driver delivery; failures retain ownership. Not deployable: pull acknowledgements, reset/close guards and affected-suite migration remain. Typecheck reports only unused handleError pending error-observer wiring. Original native 3-failure receipt remains. No pending runs or lifecycle operations.

033 integration checkpoint: native fault3/3 pass; recording suites163/163 and guard suites148/148 pass. Explicit pull ACK follows Go journal commit; reset/close retain pending entries. Stop joins an admitted start and retains terminal count/time. New native rejection/stop retry, final debounce flush and overlapping start-stop cases3/3 pass1.226s. The first retry test asserted the inner journal text instead of propagated HTTP500, left its fixture pending and contaminated later cases (121.63s retained); corrected assertion and finally recovery yield natural exit. Go driver/session/handlers/live-capture/recording/persistence/sidecar race tests and driver types pass. Source still not deployed. Canonical Test Genie unit,tidiness admitted via tool71346, log /tmp/bas-recording-owner-admission-033.txt; must consume admission and attach one quiet wait. Runtime delta currently about450 fewer lines; measured Go+4cyclomatic/+3functions, cumulative BAS-242/shared-12 = net-254. Browser/driver crash durability and full journeys remain unverified. Next inspect owner regressions, measure final costs/deltas, critique bounded receipt/overflow semantics and gate publication.

033 canonical owner run20260922-142106-ee94c0d4 admitted14:21:06UTC; admission tool71346 consumed. One quiet wait launched with timeout798; stdout /tmp/bas-recording-owner-033.json. No polling or re-admission.

033 wait owner: tool95803 (pid4160848) is attached once to Test Genie20260922-142106-ee94c0d4. Do not re-admit or create another waiter; resume that session. Setpoint read tool20038 is pending. Source audit still does not qualify browser overflow recovery, cross-origin navigation during outage, process death, or replay after committed receipt eviction; retain these as explicit next fault probes under RF002/RF022.

033 adversarial finding while canonical suites run: a new native slow-click/cross-origin-navigation case fails0.991s; stop succeeds but the queued target navigation is absent. Serial browser dispatch stranded that observation in the old origin. Retained /tmp/bas-recording-cross-origin-red-033.txt. Replacing only browser dispatch with one bounded pending owner and per-entry joined transmission; driver retains ordered commit. This source change postdates canonical admission, so that run alone cannot qualify the amended case. Driver/process crash and undispatched page-unload evidence remain distinct unknowns. Publication deferred for this demonstrated assertion, not stopped goal.

033 cross-origin repair: per-entry immediate browser transmission retains the same bounded pending queue and matching receipts; the driver alone serializes commits. The fresh cross-origin case passes with the five prior adverse cases (6pass/7filtered,2.499s; /tmp/bas-recording-cross-origin-033.txt). Test Genie quiet wait95803 remains active; canonical source snapshot is mutable and the amended native case is separately attributed. Setpoint prog_145c2fe8-51db-426b-ac78-8bb2c3adf58f remains17pending/productfalse.

033 activation boundary finding: a dynamic cross-origin frame added after start missed its click (native1fail/13filtered,1.684s; /tmp/bas-recording-dynamic-frame-red-033.txt). Dormant initialization requires an owner for every new document. Replacing separate main-page-load and one-shot new-page-load policies with one tracked page/frame listener owner; stop joins pending activations before deactivating frames so late callbacks cannot reactivate capture. Same current/future-page behavior and frame case require native revalidation. No deployment yet.

033 canonical run20260922-142106-ee94c0d4 failed427s (unit422/tidiness5); wait95803 consumed. UnitHealthuh-20260922-142107-4cbf44497c92efc5087e9749554fb398: API38.272s failed real AI suggestion category assertion, independently reproduced9.564s and registered RF068; CLI2.155s pass; driver307.012s failed35/1550cases across5suites (1513pass/2skip), including opaque-origin script init and changed stop mock contracts; UI61.297s fails unchanged merged coverage28.59/30.34/65.15/28.59 vs85; UItypes10.738s pass. Tidiness1115findings/103long/387complexity/607dup/17coupling/debt33540(+16vs032); source mutability and subsequent frame changes mean this is interim evidence. Raw/native receipts under /tmp/bas-recording-{unit,tidiness}-native-033.json.

033 frame diagnostic: default installed Rebrowser childFrame.evaluate('location.href') returns parent localhost URL while childFrame.url() is127.0.0.1. Underlying default addBinding context discovery is suspect; do not globally disable Runtime fix. Testing the installed alwaysIsolated mode as a supported context-routing alternative, then revalidate native coverage and preserve current explicit user configuration. New unified frame listener owner is source-only. No runtime033 publication yet.

033 frame repair proof: native14/14 pass8.166s and TypeScript clean. WindowProxy traversal sends the acknowledged control directly to every current descendant document from the known page context; no SDK/private frame IDs or Runtime-fix disablement. One tracked page/frame listener owner handles dynamic frames, existing tabs and future pages; stop joins admitted activation work before deactivation. Removed the old main-page load/new-page one-shot policies. Default Rebrowser childFrame.evaluate wrong-document behavior remains a separate general execution limitation under RF004/RF030; alwaysIsolated experiment timed out and was rejected, no environment mode changed. Owner regressions for opaque-origin selector init and retained stop receipt29/29 pass2.044s. Next scoped full driver requalification, cost/delta measurement, and quiet publication; RF068 live AI category assertion will remain explicit until next repair.

033 second owner qualification: scoped UnitHealth driver execution admitted once via tool61116; JSON /tmp/bas-recording-driver-owner-033.json. Do not duplicate this execution. Integration fixtures that directly inject the dormant script now send acknowledged activation; rapid-navigation fixtures now use the actual pipeline lifecycle. Focused activation suites running tool89438, /tmp/bas-recording-fixture-activation-033.txt. Native14/14 and types final2 passed before those fixture updates. Benchmark harness bundling corrected a local proto-export resolution error; no performance value recorded yet.

033 necessary shared-owner extension RF069: direct scoped UnitHealth driver validation tool61116 exited Client.Timeout exceeded without a terminal receipt. No validation/Jest process remains in the process snapshot. UnitHealth validate handler uses the ordinary CLI HTTP deadline for inherently multi-minute synchronous execution; existing cli-core NewConnectHTTPClientWithTimeout(app,0) already encodes the proper owner-bounded contract. Scope: unit-health CLI validate handler and real delayed-response regression in app_test.go, with architecture target updated first. This is necessary to obtain the authorized BAS scoped driver receipt; no dependency or host repair needed. Preserve authentication through the existing client helper. Rebuild normal CLI owner after focused Go regression, then admit one replacement scoped request.

033 RF069 regression: ordinary client deadline reproduced on the real CLI/HTTP boundary. Existing long-RPC helper now removes only that transport timeout; authentication and server-owned command limits remain. Initial green attempt reached the fixture response and exposed a JSON/protobuf content-type mismatch; corrected fixture yields all CLI race packages passing (/tmp/bas-unit-cli-deadline-green2-033.txt). Normal unit-health CLI admitted exactly one replacement driver validation, tool27922; outputs /tmp/bas-recording-driver-owner-final-033.{json,err}. Await that tool, do not re-admit. Previous lost benchmark session recovered as terminal; benchmark allocation refinement now uses one visible-entry traversal. Latest tidiness20260922-144527-149c7c63 terminalfailed/quiet wait consumed:1115findings,103long/387complexity/607dup/17coupling,debt33540. No033deployment yet.

034 RF068 investigation started while the033 scoped driver runs. Recall completed
with source-ledger providers degraded; no suitable replacement program found.
Source confirms resource-ollama gateway already supports --format JSON Schema;
no resource-owner/dependency expansion needed. BAS client currently omits it.
Generator accepts missing required fields and malformed JSON fallback always
returns empty success. Target architecture updated before034 implementation.
Planned scope: existing AI client/generator and deterministic/live tests; preserve
governed gateway, valid array/object capability and DOM-only analysis fallback.
Do not turn validation errors into integration skips. No034runtime edits yet.

033 publication observation14:58:07UTC rejected stop: live history2875rows includes10 new RUNNING beyond the five historical IDs; driver10sessions/0recordings. Receipt /tmp/bas-drain-deferred-033.json. No process stopped or history altered. Recheck only after scoped driver completion or observed live activity change. Continue034 independently if activity persists; publish a combined qualified candidate after a fresh quiet gate. Existing032 build remains live.

033 scoped driver owner completed: uh-20260922-145457-7e793a0764f101413651f7c3ea6d386d passed393.839s;1549tests/125suites pass,1suite/2tests skipped. Tool27922 consumed. RF069 fixed CLI now retains the complete terminal receipt. BAS API/UI makebuild and driverbuild pass. Publication gate previously found live work; no stop performed.034 deterministic AI package race passes1.106s. Initial live structured-output smoke fails11.421s: provider schema converter requires anchored patterns, so unanchored nonblank-action constraint is being corrected without weakening validation. Original failing assertions and errors retained.

034 scoped owneruh-20260922-150329-35524363f0d47303a760c85b534c0497 failed only the empty-element integration assertion; isolated identical test passed1.653s. Owner traceability identifies the case but bounded stdout omitted its assertion text, so exact intermittent failure is not established. Empty elements have no grounded action to infer: architecture now requires a local empty result. New no-provider-call regression fails against current code (provider error propagated), then runtime early empty return repairs the semantic boundary. Requalification required; original owner failure retained. Prior schema-validation adds roughly3.8us and~3.8KiB/81allocations per mocked one-suggestion request; repeated quiet final benchmark pending. No real-inference latency claim.

034 final scopedAPIuh-20260922-150913-3760ae5d039caeeb866c3fd0a3db7907 passed33.130s, final focusedAI race1.092s and APIbuild pass. Source3runtimefiles658/23/79 ->669/22/81 lines/functions/cyclomatic (+11/-1/+2); removed dummy fallback and redundant logger state, one schema governs output generation and validation. ExpandedGo69original/66current31177/1129/5079 ->28516/1079/4839; BAS-240/shared-12 = affectednet-252. Final matched mocked-request median4562->9967ns (+5405ns/+118.48%),3849->7704B,24->105allocations. Cost supplies previously absent validation, no inference speed claim. Source-shape3line empty guard is later than latest native tidiness; explicit final Go counts retained. Fresh quiet publication gate/stop tool44577 launched once; receipt /tmp/bas-drain-gate-034.json or deferred034. Tools77291/92901 consumed.

034 quiet gate15:10:11UTC admitted stop:2668rows (2288completed/375failed/five exact unchanged historical RUNNING),0sessions/0recordings, same032build. Stop completed, make start tool45118 pending (/tmp/bas-start-034.txt). The count differs from2833 at032 and2875 at14:58 observation before any034stop: do not claim unchanged whole history or attribute cause without evidence. Existing retention/concurrent work may account for change; inspect owner logs after startup while preserving all remaining data. No agent-issued deletion.

History-count diagnosis035 read-only: previous runtime logs show automatic owner retention at15:01:12UTC,934recording directories/3.670660666GB removed to the declared20GiB budget; captures480/1.342555037GB to5GiB. Existing owner_retention.go RemoveHook also deletes corresponding terminal execution rows. This supports retention as the explanation for the pre-stop count decrease; individual removed IDs were not retained in the gate, so exact207row reconciliation remains unverified. No new defect established/no retention edits or agent-issued cleanup. Original profile fields pass three complete repository reads against rollback. Lifecycle start tool45118 still owns publication; do not duplicate start.

033/034 published: healthy15:14:42UTC buildce4d775d1ee2c2ba22bd0c8c7ceed449bb2e84cfd1cc7315f869859a12460acc. Driverready Chromium136.0.7103.25/0sessions/0recordings. ProfileAPI metadata equals032 and all original rollback fields survive three repository reads. Start45118 consumed; no pending test/build/lifecycle operations. Source-only gap statements above are now superseded by this deployment receipt. Full product remains unqualified; board17pending. Next035 actual frame-targeting audit under RF004/RF030: source frame handler pushes mainFrame instead of target, compares URLs for identity, and route context currently appears not to forward frameStack; native action-effect proof needed before repair.

035 RF004/RF030 frame execution investigation: read-only history diagnosis found
existing retention consistent with the count change; no new retention defect or
edit. Frame source audit finds route ExecutionContext omits session.frameStack,
FrameHandler pushes mainFrame rather than target, compares URLs as identity, and
DOM handlers use page directly. This is separate from033 recording activation.
Target architecture updated before035 source work. Add an actual public-driver
frame-switch then click fixture with same selectors and same-URL sibling frames;
assert independent frame counters, not success metadata. No035runtime edits yet.

035 native red proof: public frame-switch ENTER and CLICK both return success, but independent fixture reports mainClicks1/childEffects[] instead of main0/left1 (/tmp/bas-frame-native-target-red-035.txt,1.973s,onefailed/15filtered). Earlier2.533s case failed EXIT because the route never retained frameStack;1.545s intermediate only checked missing child effect. Native standalone probe also confirms childFrame.evaluate(location.href) returns parent URL; child locator.evaluate then reports destroyed context, while Frame.click and FrameLocator.click both hit the correct child. All use default installed Rebrowser. Investigate the lowest context owner before choosing a complete DOM-target boundary; no035runtime edits/deployment. Source test before-copy under /tmp/bas-before-035. No pending operations.

035 lowest-owner finding and candidate: Rebrowser1.52.0 main-world discovery
returns undefined contextId for scriptless frames; that silently defaults runtime
evaluation to the session main document. Inline child script avoids the failure;
context.addInitScript (void or property) does not. Reading installed SDK confirms
binding discovery emits context payload without validating its ID. An isolated
copy under /tmp/bas-rebrowser-core-035 now resolves the frame's actual document
through public CDP, materializes its main realm before adding the temporary
binding, then waits for a positive matching binding receipt and releases objects/
listeners/binding. Initial prototype bound before realm creation and timed out;
reordered prototype passes Frame.evaluate, locator.evaluate and both click paths
on a scriptless cross-origin frame, without Runtime.enable/mode changes.
Receipt /tmp/bas-frame-routing-sdk2-probe-035.json; no installed SDK changed.
Necessary dependency-owner extension proposed: BAS driver canonical dependency
patch metadata/lockfile via SDA, with regression in native suite; no runtime
monkeypatch/private SDK access from BAS. Existing governed install dry-run supports
playwright-driver surface but reports rebrowser-playwright unrecorded. Next record
that already-observed dependency through its owner, then validate patch install
path before any mutation. Upstream release page lists1.52.0 as latest published
Rebrowser Playwright line (https://github.com/rebrowser/rebrowser-patches/releases);
no verified newer fixed release. No commercial/external publication or message.

035 SDK qualification: isolated matrix passes default/--site-per-process with
scriptless same-origin/cross-origin/sandbox/nested/identical-URL frames, three
concurrent first evaluations, navigation, detached refusal, worker evaluation
and narrow console-stack-getter probe. Corrected SDK owner session lookup through
existing _sessionForFrame; shared context discovery now joins one request. Removed
swallowed undefined context errors and persistent discovery scripts/listeners.
An attempted document-metadata generation guard falsely rejected OOPIF navigation
because metadata lags the actual realm; rejected it. Explicit CDP object/context
identity remains authoritative. Native maintained tests added to existing typed-
action suite; installed SDK fails both cases1.942s before patch adoption (focused
coverage also below global threshold; full scoped suite will retain thresholds).
Security Health11.351s reports773findings/8errors, no rebrowser finding; RF071
records production js-yaml advisory and untriaged detector claims. SDA approved
existing exact rebrowser-playwright1.52.0 for BAS with explicit limits (owner
receipt /tmp/bas-deps-approval-035.json). Canonical two-file SDK patch and pnpm
metadata prepared; use SDA install to adopt, no raw package manager/registry edit.
Live034 unchanged. No pending owner operation.

035 candidate qualification continues, not published. Installed first SDK patch
passes2 native matrix cases1.773s. BAS initial frame wiring passes18native6.995s;
focused owner145cases has138pass/7old frame-mock failures. Replaced tests that
expected detached-frame silent fallback with refusal/recovery/identity/ancestry
checks. Expanded public-route DOM test finds keyboard went to focused main page
rather than selected child. Native probe confirms focusing another document resets
child activeElement to BODY; the correct oracle is key delivery to the selected
document, not invented restoration of a previous input. Keyboard now focuses the
selected frame element when needed and blurs a selected document's descendant
iframe; physical keyboard owner remains Page. Expanded test pending.

First fresh-browser evaluation of a scripted data document exposed an SDK root
DOM.resolveNode mismatch (focused reproductions twice; full suite had hidden the
cold-first path). Isolated correction materializes the default realm only after
Page.getFrameTree proves the requested frame is that connection's root; children
still resolve their own document object. Five fresh-browser first-document cases
and the full frame matrix pass. Canonical patch updated; SDA reinstallation and
requalification next. No runtime mode override or private SDK access from BAS.
An attempted patch regeneration found pnpm had removed the original package;
that Python step failed before edits, and the following SDA invocation reapplied
the unchanged first patch (59242 consumed). Pristine two-file baseline recovered
by reversing the exact canonical first patch on an isolated copy. No installed
package edited directly. No pending owner/test operation at this checkpoint.

035 RF072 extension: native selected-document storage/drag checks pass, but the
second download returns the first child's cached file (parent.test.txt expected,
child.test.txt received),1.186s. Source confirms handler TTL result reuse bypasses
new operation identity. Necessary scope: download.ts, unused result-cache branch
and singleton in infra/operation-tracker.ts/index.ts, legacy close/reset cleanup
callers, and existing download tests. Target architecture updated first. Route
receipts remain the sole retransmission owner. Preserve upload/tab in-flight
tracking; no speculative replacement cache. Public frame/tab/detached/ambiguity
case already passes8.445s. Full159focused cases passed12.442s before this new case.
SDK root patch installed through SDA63437 (consumed); no publication.

035 final focused candidate: shared session frame path and typed DOM target cover
all selector/evaluate handlers, storage, drag, download and physical keyboard
focus. Public native test qualifies main/sibling/nested effects, parent vs exit,
new-tab page adoption, detached refusal and ambiguous same-URL refusal. Keyboard
oracle observes the actual selected document after external focus resets its
active element; no hidden focus-memory owner added. Fixture-only enum/import and
mock-shape errors were corrected, with earlier receipts retained. Full159cases
pass12.442s before download extension; final native/download/close/reset29cases
pass22.832s. Download cache removal also retires unused generic cache configuration/
methods, deprecated cleanup wrapper, route callers, and eight assertion-free or
wrapper-only tests. Actual repeated-download regression replaces them. UUID paths
prevent clock-time collisions; platform temp directory replaces hardcoded /tmp.

035 full qualification admitted once: UnitHealth driver tool2220; Test Genie
tidiness admission5607; types/build84637; board80061. Source candidate not deployed.
Continue collecting owner receipts, final comparable runtime/debt measures and
SDK performance cost; publish only at a fresh quiet gate. Full product remains
unqualified and the continuous goal remains active after publication.

035 measurement checkpoint16:12UTC: types/build pass. Tidiness20260922-160940-cd080fc7
terminalfailed5s, one quiet wait consumed. Native1116findings/103long/387complexity/
607dup/18coupling/debt33540: unchanged duplication; +1coupling is the expanded
integration fixture's21 imports, retained honestly rather than consolidate paths
to game it. Driver21runtimefiles5172->4724lines(-448), SDK2files1879->1881(+2),
affectednet-446lines. No Go source changes this cycle, prior net-252Go cyclo
remains the last comparable cumulative measure. Removed policies: wrong-frameURL
identity, duplicate EXIT/PARENT logic, handler TTL result-cache owner, persistent
SDK discovery scripts and leaked event listeners; result-cache maps/config/methods
and legacy route cleanup retired. TS AST complexity remains unverified (RF064).
SDA governance reports advisorypass with46existing warnings and no rebrowser
warning; no clean-fleet assertion. Security triage identifies five detector
matches as source SHA hashes and G101 as the non-secret credential field label;
follow owner-supported narrow triage after035 qualification. Historical evidence
has not been rewritten. SDK cost harness prepared outside source; run when full
UnitHealth2220 finishes, not during its load. Only that operation remains pending.

035 full driver failure: uh-20260922-160935-a45ef0d56f9e9bedcbddffeba01e6f85
325.444s,1544pass/4fail/2skip (122pass/2fail/1skip suites). Three recording-pages
fixtures omit mandatory session.frameStack; partial utils mock then masks that
error with undefined instanceof class. Fixtures now supply selected frame state
and assert it is cleared on page switch; preserve actual error exports. Native
audio fails context-binding acknowledgement, while isolated unchanged audio
passes7.293s. The current SDK assumes the binding event precedes the command
response; testing delayed legal delivery deterministically before changing it.
The original failure omitted exception-vs-missing-receipt detail, so actual audio
failure cause is not yet established. Full owner operation consumed; no rerun yet.

035 receipt-order repair: deterministic SDK-owner test delays the matching binding
event until after the evaluation response; old candidate fails0.750s. SDK now
joins that exact receipt with a bounded3s handshake timeout, validates the ID and
cleans its timer/listener/binding. Exceptions now carry the distinction from an
absent receipt. SDA install91918 consumed. Native frame/audio/recording-page
fixtures31cases pass19.940s; types/build pass. This proves delayed delivery support;
it does not retrospectively prove that ordering caused the first full audio failure.

035 quiet SDK cost: three trials per variant,30measured+5warm-up document contexts,
alternating variant order, real Chromium, equal main-document sentinel checked.
First-evaluation median-of-medians3924.92->2582.95us (~34.19% lower); warm already-
acquired context347.40->348.58us (~0.34%, noise). Context creation/navigation are
excluded; no child-frame or whole-journey speed claim. Raw90samples per variant
and warm measurements /tmp/bas-frame-sdk-cost-035.json; harness at
/tmp/browser-automation-studio/frame-context-cost-035.cjs. Original SDK restored
only in isolated baseline copy, installed dependency remains SDA-owned patch.
One fresh full scoped driver request admitted after these repairs and the quiet
benchmark, /tmp/bas-frame-driver-owner-final-035.json. No publication yet.

035 final full-driver qualification, 2026-09-22 UTC: UnitHealth
uh-20260922-162102-ce8fafecad1445ee49c7f58ab339b894 passed;1549tests/124suites,
2tests/1suite skipped; Jest350.649s. SDK acknowledgement ordering and mandatory
frame-state fixtures repaired. Existing audio fidelity passes in this full run.
Receipt /tmp/bas-frame-driver-owner-final-035.json; CLI27696 consumed. No broad
stealth/security or complete record/replay qualification implied. Next: fresh
quiet-work gate, lifecycle restart and protected-profile verification.

BAS-WORK-036 investigation, 2026-09-22 UTC — RF073/J07/J17: recall found the
original navigation-flake retry rationale; native independent HTTP counter
disproves its implicit assumption that evaluate is read-only. One instruction
causes two POSTs before returning NAVIGATION_ERROR/retryabletrue. Temporary
harness /tmp/browser-automation-studio/evaluate-navigation-036.ts bundles the
actual handler and typed instruction factory; two initial harness module-resolution
failures are setup errors, not product evidence. Correctly bundled probe confirms
the defect. Target architecture updated before code. Source035 remains unchanged
until its lifecycle restart finishes. Retire implicit retry helper and replace
its implementation-mirroring tests; preserve extraction and successful expressions.

035 publication in progress: quiet16:28:16 gate2555historyrows/2207completed/
343failed/five unchanged historicalRUNNING,0sessions/0recordings; gate0.489s.
Ownerstop39755 consumed. API/UI makebuild54509 passed; start92545 pending.
History decreased before this stop, as in034; exact retention deletions unverified.
No agent-issued data deletion. Full driver receipt351.446s passes.

035 publication complete16:32:02UTC: build4cbb4572edd4546c5aa0ba64bf53754fd51d659294ab109f8da33d89e677f1f9 healthy; driver0sessions/0recordings. API profilemetadata equals034; full original profile fields survive three reads. All lifecycle operations consumed. Final runtime net-437lines includes SDK+11; no source hash drift across21 runtimefiles before publication. Named defects qualified, product board remains17pending.

036 focused qualification:82cases across5suites pass15.235s; types60651 and
focused12197 consumed; driverbuild passed. Original maintained red2.086s had
three actual assertion failures and a whole-project coverage-floor failure from
selecting only six tests; do not interpret that filtered run as aggregate coverage.
Focused green disables collection only for the selected cases; the full owner73224
retains normal aggregate coverage. Tidiness unchanged1116/debt33540; board17pending.
TestGenie20260922-163350-8274aa55 admission42506 and one quiet wait consumed.
037 RF071 investigation: official js-yaml advisory confirms4.3.2 fixes
GHSA-2883-xcg3-v3hh; installed direct/override4.3.1 needs owner-governed update.
SDA override dry-run saysapproved even though recorded allowedScenarios only
containsvrooli-onboarding; do not rely on this as scope governance proof.
Preserve that existing grant when adding BAS. No dependency mutations admitted
while036full driver consumes the current lockfile.

036 qualified307.798s,1552tests/124suites pass; no pending owner runs. Runtime
extraction handler189->152lines(-37), implicit retry helper/load wait retired.
Publication will combine036with037to avoid redundant lifecycle interruption.

BAS-WORK-037 authority extension, 2026-09-22 UTC — RF071:
SDA InstallDependency has a supported governed npm override operation, but its
SetNpmOverride appends a broad package key without retiring the older qualified
keys. BAS currently pins vulnerable js-yaml4.3.1 with qualified overrides; replacing
that policy must leave one effective package-wide version declaration. Extend to
scenarios/scenario-dependency-analyzer/api/internal/installgateway/gateway{,_test}.go
and owner architecture docs: a broad override replaces only same-package simple
version-qualified keys, preserving unrelated/scoped/parent-qualified policies.
Prove the canonical replacement before repair and validate owner packages plus
actual BAS resolved lockfiles. No hand-edited registry/lockfile or raw installer.

RF071 exact-digest triage proof: installed Gitleaks8.18.1 fixture reports8findings
before and3after the candidate exact-line exceptions. All three synthetic credential
counterexamples remain detected in the same evidence file; the five reviewed
source-digest lines are excluded. No path/directory exclusion and every default
rule retained. Candidate /tmp/bas-gitleaks-candidate-037.toml; receipts
/tmp/bas-gitleaks-fixture-{before,candidate}-037.json. Security Health's safe-fix
preview offers no deterministic fix; use its documented scanner-native exceptions
with this evidence. Original historical source JSON stays byte-for-byte untouched.

037 boundary extension RF074: owner source confirms the approved-scopes mismatch
seen in the BAS preview. Extend the same SDA repair to
api/internal/dependencygovernance/install{,_test}.go. Reuse existing
scenarioExceptionViolation, retain its precedence for explicit denied scenarios,
and reject before SetNpmOverride or package install. Test public handler with
scoped/global grants and explicit denials for normal and override requests.
BAS package approval was widened through owner upsert while retaining onboarding;
no unauthorized-scope package installation occurred.

037 owner-validation checkpoint: SDA focused installgateway/dependencygovernance
races pass1.027/1.080s; makebuild33982 consumed/pass. Scoped API UnitHealth
uh-20260922-164500-7aefb46ceaa5aa04865b6682e1006c6c failed15.803s (70972consumed).
Its bounded JSON excerpt contains no failed test event; full local reproduction
is running to preserve the exact assertion before any owner deployment.
SecurityHealth triage037 drops five exact-digest errors, leaving two js-yaml
advisories and one G101. The original G101 annotation was placed on the field
label; native rule-specific report identifies the preceding public namespace
declaration instead. Annotation moved to that exact declaration, removed from
the unaffected field; native G101 scan now haszeroissues. No claimed whole-scan
pass yet. Original historical JSON unchanged. No SDA/BAS restart pending.

037 required-owner check failure identified (87731consumed): four assertions in
SDA coreset/dependencyhealth. RF075 records trimpath-hostile runtime.Caller and
live-operator-state leakage in temporary portability fixtures. Extend test-only
boundary to internal/coreset/coreset_test.go and
internal/dependencyhealth/connect_test.go; reuse repocontract.ResolveRepoRoot and
route fixture operator storage. Strengthen the previously false-positive
contradiction check to name the expected cause. Production closure remains intact.

037 SDA qualification final: UnitHealthuh-20260922-164831-94a58551fb070680f95ca89f3d26aa50
passed12.241s;99496consumed. Fixture races1.102/1.124s,15319consumed. Managed
restart43426pending, /tmp/bas-sda-restart-037.txt. Shared runtime753->769lines
(+16),24functions unchanged, cyclomatic209->213(+4). Cumulative affectedGo
reduction now248, including necessary admission checks. No performance speedup
claim. Bounded js-yaml4.3.1 probe with32empty merge sources and budget8 fails to
reject; ordinary workflow YAML parses. Native adverse input is small and local.

BAS-WORK-038 RF072 investigation: recall searched16results across10corpora, then
native realHTTP attachment probe confirms a directURL action falselyfails with
ERR_ABORTED while one correct download event/file stream arrives. No external
account or retained browser data used. /tmp/bas-download-url-red-038.json and
/tmp/browser-automation-studio/download-url-038.ts preserve the oracle. Target
architecture updated before repair: accept only Chromium's expected attachment
navigation-abort signal; actual event/save are still mandatory. Add native URL
byte proof and error/no-event controls to maintained existing test files.

037 dependency/security completion: all three governed installs succeeded; direct
20309 and UIoverride40807 consumed. Exact lock diff changes onlyjs-yaml and
its overriding selector; canonicalRebrowser patch remains90014f7f... Both native
CPU-budget and ordinary-YAML controls pass4.3.2. SecurityHealth73941 consumed,
20.186spass0errors; warning/info posture remains357/406. BAS makebuild74263passed.

038 validation checkpoint: positive URL regression red1.320s then nativeURL
positive/HTML-no-download/abort-no-download controls green. Source adds one
Chromium-specific failure interpretation; no fallback file or inferred success.
First focused33cases15.995s:32pass, evaluation fixture observeszeroPOSTs. That
assertion omitted actualresult; improved diagnostics now retain its error.
Five targeted native cases pass1.985s; 40fresh-context reproduction47867pending
(/tmp/bas-evaluate-stress-038.json). Types/driverbuild pass; first failure retained
and source qualification remains incomplete until this is understood/rechecked.

038 RF070 diagnosis: installed patch40-context native probe has32failures, not
the preliminary28mentioned in commentary; actual receipt retained. Baseline
unpatched SDK0/40effect failures but internal stderr errors remain. DOMobject
root selection candidate39/40fails and is rejected. Three cleanup variants: keep
registration0/40bad (89022consumed), keepglobal38/40bad (70706consumed), keepboth
3/40bad (21724consumed). Original73743 and objectcandidate73549 consumed.
One registration per CRSession with unique per-request token is under isolated
probe48166 (/tmp/bas-evaluate-stable-038.json), including follow-up-document
reads. Do not restore arbitrary evaluate retries or change global runtime mode.
No canonicalSDKmutation or dependency reinstall for this candidate yet.

038 SDK candidate refined: stable registration with deleted globals gives1/40
effect failures and cannot rediscover some retained frame contexts. Combining
DOMroot selection with stable registration fails37/40 and is rejected. Retaining
one session binding/global, using unique receipt tokens and a bounded internal
absent-binding handshake passes40/40effects plus40next-document reads; native
frame/OOPIF/worker matrix20rows passes. Installed-patch maintained fresh-context
regression fails at trial0 (receipt /tmp/bas-sdk-cleanup-maintained-red-038.txt).
Candidate files /tmp/bas-rebrowser-core-038-stable; canonical two-file patch now
regenerated from pristine1.52.0baseline. Governed install follows; no application
Runtime-mode change or arbitrary expression retry. Bounded retained global is an
explicit stealth qualification limitation, and replaces accumulating bindings.

038 installed candidate focused36tests/4suites pass22.394s, including native
fresh-context corpus, SDK receipt correlation and existing audio. Types pass;
owned server cleanup strengthened to join browser/server closure even on failure.
Matched SDK benchmark (3 alternating trials,30samples each plus5warmups) against
pristine1.52.0: cold median3562.362->2215.312us; warm292.730->320.320us. Report
the warm increase27.590us, not an unsupported speedup or full-journey claim.
Raw /tmp/bas-sdk-context-cost-038.json. Final driver/security/types+build/tidiness
operations admitted once; terminal receipts pending. No lifecycle action yet.

038 final qualification: full driver uh-20260922-171531-588ec4e8e9386bbcc9675231167fbaca
passes1559tests/124suites,2tests/1suite skipped,366.709s Jest. Original admission
was not repeated after tool-ID loss; pidfd waiter20434 joined its CLI and is
consumed. SecurityHealth5.060s passes0errors/357warnings/406infos (owner caches
for unchanged Go scans). Driver types/build pass. Finaltidiness20260922-171537-45b14dd7
failed with1117findings/104long/387complexity/607dup/18coupling/debt33540, one
quiet wait consumed. Added native integration cases cross one long-file threshold;
no threshold or coverage floor changed. Preparation valid/productfalse and board
17pending remain. Recorded source hashes match. Proceed to fresh drain observation
before combined036–038 publication, preserving full original profiles.

### BAS-WORK-039 — 2026-09-22 UTC — reset ownership investigation

Feedback008 remains active. Recall returned89hits across10corpora, including
the existing lease/terminal ownership work and BAS architecture. Source shows
reset ignores request ownership and invokes destructive state changes through
a route-owned pending map. Hypothesis: a delayed old lease can clear a new
owner's cookies/navigation after label handoff. Native temporary HTTP/Chromium
fixture uses only synthetic local state, independent cookie/URL/storage sentinels,
and stale/missing/current owner controls. No039production changes yet;038managed
start31992 remains pending and must finish before further source edits.

038 publication: build8129e9e98743f51dfb61a568315b0662dd69c312ca63beff80197474745a2e80
healthy17:26:01UTC atAPI/UI and driverok with0sessions/0recordings. Fresh gate
17:22:15 saw2618historyrows (2239completed/374failed/five exact unchanged historical
RUNNING); no active browser work. Managed stop59912/start31992 consumed. APIprofile
metadata unchanged and three complete profile reads match original rollback;
profile84003 consumed. Combined036–038 runtime net-7lines including shared SDK/SDA
changes; cumulative affected Go cyclomatic-248. Security0errors still357warnings/
406infos; tidiness1117 and board17pending/productfalse are not a release verdict.

039 native HTTP/Chromium probe confirms stale and missing lease requests both
clear the new owner's cookies and navigate it toabout:blank. Even the authorized
control failsSecurityError readinglocalStorage onabout:blank, strands phase
resetting, and retains the old origin's localStorage. Three independent controls
are retained in/tmp/bas-reset-native-red-039.json. Fixture setup first needed
canonical proto bundling and staged browser scripts; no product effect occurred
in those setup failures. Next repair/reset qualification must cover ownership
and a coherent clean-context boundary rather than suppressing the storage error.

039 bounded implementation choice: first close the independently proven stale-lease
reset admission gap across Go Session→driver client→HTTP route, deleting route
time heuristics and duplicate logging. Existing in-flight joining stays in its
current route owner for this change. Do not imply that this repairsRF076: native
current-owner control remains an explicit failing storage-reset observation.
Scope: existing reset route/tests, Go driver/session and native profile continuity
test (temporary owned context, HTTP route). No new dependency or schema work.
Follow with the coherent context reset repair; do not suppress SecurityError.

039 maintained tests: native stale/missing/released reset controls all fail before
repair3.204s. Go tests fail on absent owner body and false/missing success receipts.
After ownership transport+validation, focused9cases/2suites pass4.248s (including
concurrent shared success/failure and delayed-body handoff); Go driver/session/engine
races pass. The engine happy-path fixture now acknowledges reset instead of emitting
an empty200. Initial focused6/2pass3.499s warned of open handles; explicit fixture
server connection closure and --detectOpenHandles run exits naturally without
reported handles. Driver types pass. Runtime130->42reset route, Go client+11lines,
sessionlinecount unchanged: net-77. Scoped driver38932/API30665 and tidiness
admission49887 admitted once; driver types/build4732 pending. Live038 unchanged.

039 API owneruh-20260922-173337-7982c7926e5e83a27ee25627d788697f passes36.661s;
API/UI and driver builds/types pass. Tidiness20260922-173343-048f3e9f terminalfailed
and onequietwait consumed:1118findings/104long/387complexity/608dup/18coupling,
duplication debt32643. This whole dirty-tree reduction cannot be attributed solely
to039. Go99functions unchanged,290->292cyclomatic; cumulative affectedGo-246.
Boardprog_41d40642-91c9-4e08-9fe0-90a6ffc4d7f4:17pending/productfalse. Full scoped
driver38932 remains active; no lifecycle operation.

040 storage-reset discriminator (no production edits): official public Chromium
CDP DOMStorage.clear rejects origins without a currently loaded frame, including
all old origins afterabout:blank. Storage.clearDataForOrigin(all) clears persisted
localStorage on both origins but leaves per-tab sessionStorage intact. Native
/tmp/bas-reset-{cdp,storage-cdp}-040.json rules out those naive substitutions.
Context recreation would also require preserving recording/capture ownership;
per-origin snapshots omit cache-only origins and copying large IndexedDB merely
to enumerate origins is avoidable overhead. Continue owner design/independent
storage matrix, without suppressing errors or claimingRF076 resolved.

039 full driveruh-20260922-173337-68e0fa749361097f6a4aa0c9df53bc49 passes342.144s
(337.637s Jest):1565tests/124suites,2tests/1suite skipped. Tool38932 consumed.
All3recorded runtime source hashes unchanged. Jest reports a one-second open-handle
warning before natural exit; focused detection did not report a handle, so the
earlier fixture closure is not proof that every asynchronous initializer is joined.
Retain this lifecycle concern for the reset/close owner investigation. No pending
validation. Proceed to fresh quiet gate for the qualified lease admission fix only;
RF076clean storage reset remains separately broken.

040 candidate experiment only: a temporary public-CDP plus intercepted-origin
visit clears cookie/localStorage/sessionStorage/IndexedDB/CacheStorage across
two origins with zero fixture HTTP requests during reset (~39.98ms, one trial).
A separate context on the same origins retains all five sentinels; a non-initial
active page remains and other pages close. Raw/tmp/bas-reset-protocol-candidate-040.json.
This is not production qualification: origin inventory/imported and closed origins,
partitioned iframes, service workers, recording's captured primary Page, failed
stages, and retained video/performance capture need tests/owner design. Avoid
replacing the native reset defect with a partial clear. Managed039start29526
still pending; no040production edits.

039 published buildaa7f6efbe4a4ee3e194865259087c40720dd50b2d2e8412b43c89effe7f3f66e
healthy17:43:49UTC. Fresh17:40:45 drain gate2618rows/five exact historicalRUNNING,
0sessions/0recordings. Managed stop55720/start29526 and profile86231 consumed.
APIprofile metadata equals038 and three full reads preserve original rollback
identity. Lease reset admission is deployed; RF076clean reset remains unrepaired.
No pending tool or lifecycle operation.

040 cross-site iframe experiment extends the candidate: localhost frame under
127.0.0.1 top-level holds separate local/session/IndexedDB/CacheStorage sentinels.
After reset all are empty; separate context remains unchanged. Zero fixture HTTP
requests during the operation;44.73ms single observation. This supports the named
installed-Chromium case, not a universal partitioning claim. No production040
change yet; proof/tmp/bas-reset-partition-candidate-040.json.

040 selected owner repair: preserve context and original primary page; record
visited/imported origin inventory at context construction; clear persistent
storage through public CDP and per-tab storage using intercepted empty origin
visits. No external requests or IndexedDB serialization to enumerate origins.
The two-origin and cross-site-frame prototypes support this approach; extend
maintained tests to imported/closed/cache-only origins, service workers, independent
context, primary-page capture, failure/retry and reset/close fencing. Scope existing
context-builder/SessionState/SessionManager/reset helper/route and their tests.
Move reset joining from the route to SessionManager and delete old swallowed
page/unroute errors; no new dependency. RF042profile import and RF043secondary
capture remain their own evidence boundaries. No040production edit before tests.

040 initial production repair: context-owned imported/visited origin inventory,
public CDP clearing plus intercepted origin visits, retained primary page/context,
error propagation for failed page close/unroute, manager-owned phase/in-flight
reset coordination and close settlement join. Initial native red3.166s -> green
3.375s; focused114/5pass6.948s and types pass. Expanded coverage57/59passes40.85s;
capture/video/trace/HAR and post-reset recording test passes. Added frame/cache-only
case times out30s with no operation location; diagnostic12939 admitted with temporary
RESET040 markers. Close-order test wrongly included constructor capability-probe
context close; reset baseline cleared and release now in finally. Do not lengthen
the timeout or call the expanded suite qualified. Live039 remains unchanged.

040 additional native file-origin discriminator: Chromium permits file:// local
and per-tab storage even though Node's WHATWG URL.origin isnull. The initial
tracker omitted it, and the maintained two-file reset test correctly fails3.109s
with both sentinels retained. PublicCDP file:// clearing plus an interceptedfile://
empty document is supported (one intercepted request, original file not loaded);
tracker now records that origin explicitly. Independent second-context storage
is the preservation control. /tmp/bas-reset-file-*-040.json and maintained red
receipt/tmp/bas-reset-file-maintained-red-040.txt.


### BAS-WORK-040 — 2026-09-22 UTC — clean reset qualification

RF076 source repair now passes the final149-case focused matrix in9.609s and
TypeScript/build checks. Original public synthetic reset reproduction now returns
200ready with no cookie/localStorage sentinel; stale404 and missing400 preserve
new-owner state. Its50.15ms observation includes the post-reset navigation/read
oracle and must not be presented as isolated reset latency. File-origin red is
retained. Final expanded fixture uses real frame-host navigation after service
worker registration; setContent timed out before reset and is not a repaired
product path. No deadline was raised and no temporary markers remain.

Owner driver89109 is the sole pending operation, admitted once. Tidiness run
20260922-180542-e4db78b1 failed; one quiet wait consumed. Types/build93368 passes.
Boardprog_8ded6dad-b2fc-4e13-9b5b-2b84432c3a9b still17pending/productfalse.
Runtime ownership improves through one manager-owned reset reservation/join and
error propagation;5runtime files add41lines, combined039–040 removes36. No new
Go changes and no invented TS cyclomatic reading. Retained primary context/page
keeps capture files and new recording usable; secondary captureRF043 and clean
profile importRF042 remain unqualified. Next consume owner, record metrics, then
fresh quiet gate before managed publication if qualified.

BAS-FB-009 captured both status requests verbatim and answered at overall-goal
level: materially improved, still core correctness work and17unqualified release
rows; no unsupported percentage/date. Continue the existing unlimited goal.

040 final owner qualification: uh-20260922-180536-251d1c74ffe41f505e06d9d89f73826c
passes332.571s (Jest327.966s),1573tests/124suites;2tests/1suite skipped. Tool89109
consumed, one-second handle warning followed by natural exit remains unresolved.
Native tidiness unchanged1118/104long/387complexity/608dup/18coupling/debt32643.
All5runtime hashes match the recorded source. No pending validation. Proceed to
fresh observational quiet gate and managed publication; preserve full profiles.

040 managed stop85193 consumed after fresh18:12:42 quiet gate:2885history rows
(2468completed/412failed/five exact historicalRUNNING), zero sessions/recordings.
Managed start76820 pending (/tmp/bas-start-040.txt). Do not edit production source
until the qualified build completes. Profiles must be rechecked afterward.

041 independent lifecycle hypothesis: early recording verification polls a closed
page and retries after session close has already reported success. Recall69hits
across10corpora (some provider degradation), no conflicting owner result found.
A temporary native manager fixture will count new CDP attempts after page closure
and time pipeline settlement after close, with a ready-before-close control. No
production edits and no claim that this explains every Jest handle warning.

040 publication healthy: build9c7c5eef94430e9f37d9a5b9bb14fb7fb01710053704586fdcc447d480651df7.
API/driver/UI healthy, zero sessions/recordings. Managed76820 consumed. All5
source hashes remain qualified. Profile API metadata unchanged and original full
profile preserved on three reads. Scope is clean reset, not whole product release.

041 native discrimination: ordinary immediate-close and ready-before-close controls
both settle before close with zero post-close CDP calls, rejecting an unconditional
normal-start leak claim. A deliberately deferred readiness promise succeeds but
retains one10000ms deadline timer. A raw native page with no injected script,
closed during readiness verification, still incurs40CDP attempts and2019.62ms
settlement against a2000ms timeout. Receipt/tmp/bas-readiness-lifetime-red-041.log
(FIXTURE_RESULT), tool33230 consumed. These prove bounded but orphaned readiness
work, not every full-suite warning's cause. Repair existing manager wait timer
and verification closed-page boundary; add maintained unit/native controls.
No new dependency, no global cancellation framework, no timeout increase.

041 maintained red: four failures/one pass in2.036s verify three losing-deadline
cases and native closed-page wait. First green attempt exposed missing once/off
methods in the existing Page test double and concurrent mocked startup polling
in the timer-count oracle. Added missing event methods; bounded readiness tests
now complete a mocked initial verification before measuring their own deadline.
No product timeout/floor changes. Six focused cases pass1.853s with no handle
warning. Final native fixture observes protocol-close and inter-poll-close without
checking the implementation's listener choice. Types pass; broader focused driver
checks pending. RF038 delayed stream-start ownership remains a separate boundary.

041 broader focused100tests/6suites pass19.47s, no warning; owner operations52477/
85469/53057 admitted once as in checkpoint. Native deadline retained1->0 and
closed-page protocol calls40->0; settlement0.387ms after closure. Live040 remains
healthy, no041deployment yet. Next consume owner receipts without readmission.

042 board investigation only: the rehabilitation branch hardcodes all17 outcomes
to pending_telemetry even after local receipts exist. Test Genie's governed
runs/list, runs/findings and runs/freshness can supply phase verdicts and current
tree identity. Its freshness verdict means latest *passing* run, so never treat
stale as evidence of product failure. Probe actual protojson and source-stability
fields before selecting applicability. Current structural-debt requires both
unchanged tidiness budgets and wider net-ownership/complexity evidence; passing
one phase cannot qualify the whole row. No board production edits yet. Skill
program-runtime read; its external journal/filing rules are overridden by active
file-only/no-stop authority. Read-only memory recall applied; no memory writes.

041 full owneruh-20260922-182410-0f03f688f5567b49d15aa763664bc239 fails84.444s
on an inline reset-route Page double missing event methods; not a product pass.
Replaced that duplicate literal with the existing createMockPage fixture rather
than adding product fallbacks for an incomplete mock. Final affected controls
include reset, health, manager, idempotency, external detach and native pipeline.

042 native governed probeprog_ab6aacd2-51c9-4184-a944-3518b4f28c23 succeeds but
freshness reports the empty-input SHA256td:e3b0c442... while the recorded tidiness
run has nonemptytd:72241b97... . Source inspection: RunsService.scenarioDir now
returns routed artifact storage, but CheckFreshness hashes that path as source.
No applicability assertion is valid from this read. The scratch session
sess_67c712aa-615e-4257-95ce-e0a50e75e19e was reclaimed. Next reproduce at owning
Test Genie handler with separated source/artifact roots; no board code edits yet.

041 corrected-fixture focused98tests/6suites pass13.931s; one-second warning still
appears before natural exit in the wider mocked cohort. Do not claim every handle
problem repaired. Original owner failure retained; second full owner admitted once
against unchanged runtime and corrected reset-route fixture.

042 necessary owner scope extension, before shared source edits: Test Genie
api/internal/app/runs/{service.go,freshness.go,lifecycle.go,*test.go} and existing
owner architecture docs. RunsService.scenarioDir is now an artifact path resolver,
but CheckFreshness and bare-name FindRun(matchCurrentSource) still hash that path.
This prevents BAS from using authoritative current-source qualification and can
make unrelated artifact bytes stand in for code identity. Preserve routed run
history and explicit target kinds; separate source resolution from artifact
resolution at this owner, remove the ambiguous helper name, convert all callers.
Validate source edits invalidate prior evidence while artifact-only edits do not,
with roots deliberately separated; retain exact existing FindRun shape checks.
No shared freshness-go algorithm or schema/dependency changes anticipated.

042 maintained separate-root tests fail for both freshness and current-source
reuse before repair (package0.360s). The run-query owner now resolves a source/
artifact pair explicitly; all17query calls use the routed artifact field, while
freshness and FindRun hash source. Deleted FindRun's duplicate typed-target source
selection and ambiguous scenarioDir helper name. Admission retains its existing
custom-path support. Focused current-source/non-scenario regressions pass0.368s.
Metrics {"before": {"lines": 1664, "functions": 59, "cyclomatic": 334}, "after": {"lines": 1656, "functions": 59, "cyclomatic": 332}} . Next owner/race/build checks, then live quiet gate
for Test Genie. ListRuns intentionally omits heavyweight terminal provenance;
board applicability must read GetRun for the selected exact ID, not mistake the
compact list's absent sourceStable/configuration fields for canonical failure.

042 operations admitted once: Test Genie owner UnitHealth API63392 (/tmp/bas-
freshness-owner-042.json), runs/shared-runs race5184, managed build5771. No Test
Genie lifecycle action yet. BAS041 second driver33499 remains pending separately.
Shared root repair adds no functions and removes8runtime lines/2Go cyclomatic
units; cumulative affectedGo-248. No board implementation or dependency edit yet.

041 second owner uh-20260922-183038-599e1437c3a424e2bb7cef8ee4e96d56
failed354.417s:1578pass/1failure/2skipped,123passed suites/1failed/1skipped.
The timeout regression sees3fake timers instead of0 when run after the whole
manager cohort. Hypothesis: common Page double never becomes closed or emits
close, so prior sessions keep polling across tests. Repair fixture lifecycle
semantics and retain the zero-timer assertion; no product timer relaxation.

042 runs/shared-runs race passes5.063/1.250s and managed build passes; sessions
5184/5771 consumed. API owner uh-20260922-183359-ac3f3e95bbbcfabddb42534feea5c109
times out104.007s after60s silence, with providerconformance fleet-contract
failures in stdout. Session63392 consumed. No full API pass claimed; affected
run-owner packages pass. Diagnose timeout source sufficiently to distinguish
from source/artifact repair; do not waive or suppress fleet assertions.

041 fixture hypothesis refined: globally emulating Page close would also require
repairing older host-audio fixtures that reuse a single Page for different
contexts. Manager unit tests do not validate script injection. Their existing
readiness stub now applies consistently per test, keeping unrelated background
verification out of deadline accounting; native pipeline tests retain real
verification/close coverage. Full manager coverage58/58passes2.28s, no warning.
Third full owner2810 admitted once; no runtime change since041 qualification.

042 isolated providerconformance diagnostic fails45.296s at its bounded test
deadline; stack locates CodeFacts.DescribeCodeFacts via validateExecutionRunners
inside fleet descriptor validation. This is outside the changed run-query owner;
retain missing full API qualification and recheck when provider-conformance
fixtures/live calls are repaired. No assertions suppressed. Fresh live Test Genie
admission reports running0/queued0/previewInFlight0; recorded source hashes match.
Managed restart58063 admitted after gate, pending. Observation is not an atomic
admission lock. BAS remains on040 pending driver qualification.

041 third owner uh-20260922-184018-126e6e3795f97e882bef37948c355810 passes
356.381s (Jest355.453s):1579tests/124suites pass,2tests/1suite skipped. Tool2810
consumed. No final one-second warning. Scoped readiness repair qualifies; fresh
BAS drain/publication is next. Runtime2files still net-13lines.

042 published Test Genie build25e2b598c0417ea7ac23ab7c501c93f4340e70bf0c0aabef1c5d578cf1029e4f
healthy2026-09-22T18:44:48Z; restart58063 consumed. Governed public probe
prog_9a4fffc3-3433-4d89-bd10-d604d934048d succeeds: source digest now nonempty
td:b2200de9a721ebc2d2e02bbe36a95bd341564a08363535f9851fb90ad11c8d8b. Exact
terminal read retains041sourceStable=true/shared-scoped provenance and1118
findings; its older digest differs after subsequent fixture edits, correctly
unqualified for current source. No empty artifact hash remains. Full owner API
qualification still unavailable as recorded above.

042 board integration remains unimplemented: the contract requires wider
structural improvement beyond tidiness, and the current terminal is historical.
SDA scan retained existing declarations but inferred no scenario edges; no
dependency edit/install was made. Latest board remains17pending/productfalse.
Prioritize core stream ownership next (RF038/RF045) while retaining authoritative
raw phase evidence. A green phase alone cannot qualify the structural-debt row.
043 recall66hits/10corpora (some providers degraded) confirms original RF045:
old async cleanup deletes replacement tracking, pending startup survives stop,
and failures leave a socket open. Investigate existing frame lifecycle owner
and session-close integration; preserve current041 source until publication.

041 publication deferred by actual active browser work: first fresh gate rejects
pagination repeated identity; follow-up health shows10sessions/0recordings on
unchanged040build. No stop executed. Resume publication only after a new quiet
gate. Source041 is qualified and may be composed with later scoped repairs; do
not freeze useful work while another owner is using the browser.

043 bounded next intervention within RF038/RF077: delayed session-start preview
ignores readiness=false and captures a mutable session lookup after waiting.
Prove timeout, closed, released and reassigned leases cannot start a preview;
live current-owner readiness still starts. Bind the page provider to immutable
execution/lease constants using existing getSessionForLease authority. Do not
create a second lease policy. Frame-manager overlapping generation defects
RF045 remain separate and will be addressed with their own disposal oracles.
