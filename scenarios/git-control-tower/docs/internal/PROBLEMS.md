# Problems: git-control-tower

Known issues, tech debt, and deferred work for the git-control-tower scenario.

## Corrected — committed suggestion retained for the next staged selection (2026-10-22)

Operator feedback: “I accepted its proposal and committed with that. But then the
next staged suggestion still showed it there.” Screenshot: IMG_1836.jpeg.

- Rung: localized W3 session lifecycle defect under DESIGN.md's staged-message
  acceptance/commit contract. No maturity or unrelated contract gate claimed.
- Recall found related staged-message work; no evidence this repeats the prior
  budget/auth failures. Search run reranking and source-ledger recall degraded.
- H1 confirmed: UI commit success clears the editor but never retires the
  indefinitely cached suggestion session or browser job handle. The next Suggest
  reopens it; its staged-drift warning is correct. H2 (server returning an old
  result to a fresh collection) rejected for this path: hasWork bypasses Collect.
- Desired behavior: acceptance remains editable; confirmed UI success plus hash
  retires the completed session. Refusal, transport uncertainty, missing hash and
  ordinary staged drift preserve it. Active/unknown execution handles remain for
  cancellation/reconnect; server job history is not deleted or rewritten.
- Shared session ownership extracted from useMessageSuggestions into
  messageSession.ts. useCommit captures repository/session generation at dispatch,
  and retires only that generation. Both ordinary and skip-precommit commit paths
  already use this hook. New collection/reset generations guard late evidence,
  verification, adoption and persistence writes; retiring closes the drawer.
- RED: accepted owner result -> confirmed commit still displays suggestion
  standing, `/tmp/gct-1836-red.log`. Final focused regression covers navigation,
  fresh-client reload, failed/uncertain commits, unmounted composer, other repo,
  deferred collection/verification, newer collection and active/unknown handles.
- Independent review identified a pending-mutation repository-switch race and
  a default repository key normalization error. Fixed by capturing the original
  raw repository ID alongside the session, using it for cache invalidation and
  only normalizing at the feature session boundary. Added a deferred switch test
  and undefined/null/empty-ID table. Independent final review found no further
  material issue or justified additional refactor. The shared extraction removes
  duplicate session/storage ownership without a new persistence format.
- Final focused 122 tests pass across five affected files; lint, type-check,
  build and proto freshness pass. Evidence:
  `/tmp/gct-1836-final-focused.log`, `/tmp/gct-1836-{lint,types,build,proto-freshness}.log`.
- Scoped Test Genie unit `20261023-002435-524db039` remains failed at the existing
  global coverage threshold: lines/statements 64.79%, functions 61.37%, branches
  70.17% versus 85%. Owner evidence reports 68 test files / 641 tests passed;
  that summary alone is not a complete execution receipt. Findings payload
  `/tmp/gct-1836-unit-findings.json` is 925240 bytes, sha256
  `6b797a142d305a30ec7b6d9499f2cfee37d7dffdb1ce0a8bce289b5d7e7802ce`.
  The focused final checks include the last normalization correction; the shared
  checkout suite is not claimed to certify an isolated exact source snapshot.
- Activated with canonical make restart; API healthy/readiness true and UI
  serves index-CxE6pYqi.js. `/tmp/gct-1836-restart.log` and served HTML retain
  activation evidence. Index before/after comparison is unchanged. No real
  commit, model execution, staging or historical job mutation used for testing.
- Limits: this browser's successful UI commit lifecycle; external commits and
  cross-tab synchronization are not added. No new browser/BAS certification
  claimed for this state-only fix; regression uses the real composer and mutation
  hook with mocked owner responses.

## Corrected — completed investigation rejected without a visible cause (2026-10-22)

Operator reports repeated manual verification after IMG_1835.jpeg shows "No
completed suggestion" and the previous limited draft without a visible reason.
This is a scoped W3 correction under DESIGN.md's completed-child acceptance
target. Earlier isolated owner and mocked browser tests did not prove the whole
budget-settlement and message-acceptance path.

- Recall reused the previous identity correction; search-hub returned related
  staged-message skills/docs, with run reranking and source-ledger degraded.
- H1: the previous authorization failure recurred. Rejected by 15 real evidence
  receipts: complete 285-file inventory and 10 additional complete patches.
- H2: the workflow rejected an otherwise completed child. Confirmed: job
  `message-136f0b281360289d0b281031e9ed10c1`, execution
  `03731f67-87aa-4f77-be8d-c0b06011dce6`, child
  `2b6fbc48-d6f2-4e67-beaf-56611778c87f`. Child completed in ~73s;
  workflow reports budget_exhausted(tokens), 523731 > 262144.
- H3: failure diagnostics were lost. Confirmed: AM terminal reason contains
  code and budget_name but no message; GCT copied only message. Drawer puts
  diagnostics below the lengthy previous draft and uses generic failure copy.
- Secondary acceptance defect: completed output cites numeric file IDs as
  source_ids despite empty selected contexts. Strict decoding must continue to
  reject unsupported references; fix the generating contract, not the validator.
- Evidence: `/tmp/gct-third-failure-{execution,runs,trace,run,events}.json`.
  Backend, presentation and independent validation lanes own reproduction,
  fixes and cross-owner acceptance. Preserve staging, existing jobs and authority.
- Canonical accounting confirms 523731 = 94163 non-cache + 429568 cache-read
  tokens, versus 86988 context tokens; accounting is correct. Quick observed
  usage was 26563. The fixed declarations are quick 2.0.1 / 262144 tokens and
  deep 1.1.1 / 2000000 tokens. Existing $1/$2, time, call and child caps stay.
  No claim that every possible 4096-file investigation fits this finite budget.
- Independent predeployment catalog validation rejected an initially proposed
  4 Mi token ceiling with the unchanged $2 cap (`charge_budget_ratio`). Corrected
  to 2M within the existing charge ceiling. The new behavioral regression now
  resolves the real skill and calls the actual catalog validator before using
  a test-owned revision, so the engine helper cannot conceal this boundary.
- Budget regression failed on real completion behavior before the fix, then
  accepted observed usage and bounded followups while preserving genuine token
  exhaustion and complete typed end output. Terminal projection regression
  failed on the empty message and now preserves owner messages or typed budget
  meaning, including missing reason. Logs: `/tmp/gct-1835-budget-behavior-red.log`,
  `/tmp/gct-1835-reason-{red,green}.log`, and
  `/tmp/gct-1835-independent-final-owner-contracts.log`.
- Both generating skills and result schemas explicitly separate selected
  `contexts[].id` from stable integer file IDs and require empty source_ids when
  no context was selected. Strict normalization is unchanged. Added a permanent
  sanitized 285-file regression retaining the actual 26 prior / 10 newly read /
  29 claimed ID sets. Invalid source references, missing inventory and invented
  inspection fail; valid source-free whole synthesis passes with 29/285 detailed
  inspection. Exact retained replay found no further acceptance defect. Logs:
  `/tmp/gct-1835-retained-replay.log`,
  `/tmp/gct-1835-independent-final-replay-reason.log`.
- Cleanup deliberately removed a proposed redundant source-ID projection and
  feature flag. Existing contexts remain the single allowlist; retained input
  and lost-ack retry byte shape stay unchanged. Semantic JSON assertions replace
  string-format fallback logic. Two independent source reviews found no further
  justified production refactor. Prompt instructions are not model guarantees.
- UI has 81 focused passing tests and full lint/types/build. Served bundle is
  `index-B0CsW0IY.js`. Failure cause and technical detail now precede long prior
  drafts, with truthful failed/cancelled copy and an explicit legacy no-reason
  fallback. Mobile first fixture exposed its own missing successful-parent
  snapshot; restored the real owner response shape and enabled native fail-fast
  node settings, preserving all earlier checks. Corrected selected run
  `37b33751-22d1-40d4-bd67-ad40e860166f`, BAS
  `42824e2c-1c72-4cd0-b92b-b81f7c24a37f`, passed all 110 steps. SQL/file lease
  cleared with empty installation/heartbeat/cleanup errors and all four routing
  counters zero. Actual pixels reviewed. Browser transport remains fully mocked.
- Full affected GCT API/domain/policy checks passed; final full AM
  workflowruntime/catalog packages passed. Proto verifier passed. Scoped Test
  Genie unit `20261022-225532-8bbd164b` retains the existing UI test:coverage
  failure, 64.7% versus 85%; no full unit-green claim. Receipts:
  `/tmp/gct-1835-{gct-full,final-owner-packages,proto-freshness,unit-owner}.log`.
- Live model validation cannot be launched from this session: canonical own
  identity renewal reports identity_refresh_unavailable, its retained attached
  root is needs_review with expired owner proof, and identity-binding returns
  lease_unavailable. No credentials borrowed or stripped. Separate defect filed
  as `knw-1792709401363861686`; no authority-model change belongs in this fix.
- GCT canonical restart activated the API/UI, but independent readback correctly
  found old AM revisions and stale prompts. Supported explicit reconciliation
  returned 401 verified agent caller required. After a bounded owner read showed
  zero pending/starting/running children, canonical AM restart was chosen to run
  its existing startup declaration reconciliation. Final independent readback
  at 23:02:07 UTC confirms quick 2.0.1 / 262144 and deep 1.1.1 / 2000000,
  unchanged $1/$2 caps, exact source hashes and prompt_stale=false. Both APIs
  report healthy/readiness=true and UI serves B0CsW0IY. Receipt:
  `/tmp/gct-1835-final-live-activation-receipt.json`. AM also reports a startup
  workflow_accounting context deadline timeout while remaining ready; this is
  separate from successful declaration activation and must not be hidden behind
  the lifecycle's optional-resource warning. The same observable signature is
  already recorded in `docs/internal/delivery-reliability-research/follow-through/evidence/continuation-admission/am-workflow-accounting-timeout-20261020/README.md`;
  that is not proof of identical cause or completed recovery. Staged fingerprint
  is unchanged.
  No fresh live model output was obtained; that validation gap remains explicit.

## Resolved correction — missing workflow identity and misleading selection heading (2026-10-22)

Operator feedback (verbatim): 'It says "For all 285 staged files" at the top,
but then gives conflicting info later' (IMG_1832.jpeg).

- Screenshot explicitly reports evidence denied because it requires the owning
  verified workflow child. No new evidence; prior draft retained as LIMITED.
  This is a second real execution boundary failure, not merely count wording.
- H1: signed child identity lacks workflow_execution_id despite the run being
  workflow-owned. Trace latest retained execution, identity issuance metadata,
  request projection and exact evidence denial before changing security.
- H2: CLI transport omits or substitutes the correct verified child token.
  Compare retained invocation/environment ownership without printing secrets.
- H3: 'For all N staged files' is an unconditional selection label mistaken for
  verified draft scope. Reproduce LIMITED/blocked state and make selection
  heading neutral; preserve distinct whole/limited result standing.
- Expected: the actual owning child can read only its immutable inventory and
  selected patches. Unsigned/foreign/insufficient identities remain denied.
  No policy widening, fake execution coordinates, or borrowed credentials.
- H1 confirmed: latest job `message-dd3a5652f20602ebb1da2ee813d343dc`,
  execution `1e4ea155-75dd-476a-85d2-3890fa102dc5`, child
  `53c57a31-7ea0-4035-8681-6c9b81d152f6`, attempt
  `da4539e9-5528-42b5-9ecf-53ca2cd7f441`. The inventory command failed at
  21:35:09 UTC with the screenshot's exact child-identity denial; zero receipts.
  Saved child environment has the correct workflow coordinates. H2 rejected:
  verified read capability reached the handler, and issuer/REST/CLI metadata
  retain the coordinates. The shared provenance layer alone expected snake_case
  while canonical AM issuance uses workflowExecutionId/workflowNodeId.
- Validation gap: positive GCT fixtures fabricated the expected snake_case
  verified-claims map, while separate AM tests asserted camelCase issuance.
  Neither exercised their join. The new regression starts a real workflow,
  captures its issued runner credential, crosses owner verification HTTP and
  CLIUtilVerifier, then checks downstream verified provenance and read capability.
- Related trust repair is necessary at that same boundary: public VROOLI_
  environment labels previously entered signed workflow metadata without a
  durable membership check. Verification must reject unproven workflow ownership
  while preserving valid base authority. A private, persisted admission binding
  covers the first tool call before launch acknowledgement; historical genuine
  children use the retained attempt's exact run identity. No new persistence
  owner or authority policy is introduced.
- UI header is neutral and LIMITED reason prominent; 79 focused cases,
  lint/types/build pass. Browser desktop passed; first updated mobile fixture
  reached the correct limited state but retained an obsolete footer expectation
  from its earlier failed-job variant. Corrected to adopt the explicit previous
  draft button; mobile rerun `211c0d7e-2ac1-45f2-9ce5-a055b2eabb88`
  passed (BAS execution `097a3cfd-28f2-40b5-ac5c-c088922973b3`). This was a
  fixture expectation, not product failure. Both SQL/file test leases clear
  cleanly. Actual mobile pixels show the neutral heading, 26/285 detailed
  inspection, clear limited reason and reachable footer together.
- Identity correction passes real issuance→SQLite persistence→owner HTTP
  verification→CLIUtilVerifier→provenance under race. A genuine workflow child
  is admitted before launcher acknowledgement; missing/wrong private receipt,
  task, key, attempt and execution are denied. Historical foreign signed labels
  retain base read authority without workflow ownership. Acknowledged historical
  children and the actual continuation mint path remain accepted. Baseline and
  legacy grant verification use the same membership projection.
- Broad AM orchestration/domain run initially exposed two fixture assumptions:
  delegation asserted unbound labels in verified ownership, and a maintenance
  fence fixture used now-reserved workflow environment input. Updated fixtures
  preserve raw child-versus-parent label isolation, verified-proof requirements,
  Force rejection, scope/expiry/revocation, and the closed maintenance fence.
  Final full owner run passes (91.151s/.012s). No production workaround or
  weakened fence was introduced.
- Late pre-handle review found a second adapter mismatch: ordinary AM Get
  intentionally omits Input, but GCT used it to bind the immutable job before
  saving the execution handle. GCT now uses the existing explicit payload RPC,
  which supports running executions and the same narrow child read route.
  Corrected fake metadata responses reproduce this defect before the fix.
  Actual handler/routing tests and exact owner/key/input/digest binding checks
  pass. No additional API, scope, profile or persistence owner was added.
- Full final GCT API/domain checks pass; shared provenance/authn/identity and
  independent configured GCT authentication tests pass. Exact receipts:
  `/tmp/gct-workflow-identity-{race,full-owner-final,gct-final}.log`,
  `/tmp/gct-child-independent-realjoin.log`,
  `/tmp/gct-child-independent-prehandle-payload.log`, and
  `/tmp/gct-child-independent-actual-payload-handler.log`.
- Scoped Test Genie: GCT `20261022-214907-8c8ca6c7` dependencies passed; unit
  retains the same existing UI coverage failure (64.7% versus 85%). AM
  `20261022-214936-93c71af8` dependencies passed. No full-scenario green claim.
- Source is frozen after independent review and cleanup. Canonical AM and GCT
  restarts completed; both health endpoints report healthy. UI serves
  `index-CiM8-mRS.js`; deep workflow 1.1.0 remains active with matching source
  hash and current resolved prompt. AM's optional-resource degraded summary is
  pre-existing and separate from these healthy API/readiness checks.
- Final staging fingerprint is unchanged. Existing jobs and prior drafts remain
  intact. No new paid/live model invocation was made. Validation establishes
  real issuance, verified ownership and payload-access boundaries through joined
  and actual-handler tests; it does not claim new model semantic output.

## Resolved correction — whole-selection synthesis versus inspection (2026-10-22)

Operator feedback (verbatim): "I'm confused. I pressed the button to investigate the remaining changes and it looks like it just made an investigation that is the same number of files so like it shows the previous suggestion and then a new one and like I can see the mirror Rita showing like you know previous suggestions for the same you know staged files but it didn't actually like investigate the rest. It looks like an investigated either the same amount or you know like the same number of files but that doesn't make any sense.  And it says like 26 out of like however like 200 or something Files I'm red so it definitely looks like an agent didn't you figure out what's going on with the whole thing like I wanted to but also I don't want to necessarily be required to read all the files. We should be smart about how it's prompted like for example you know we could just give it you know the commands to see what files or stage or whatever but we can tell it to try to be smart about it and like you know like not you look at the whole thing if it's like over a certain size you know like if it you know, just look at the important files or you know things like that or like files a lot of changes or whatever you know that that could be like a strategy there I don't I don't think we need to be too prescriptive here but like like if it had to read all the changes every time you know that would be a lot of tokens. I'm gonna take a lot of time so I definitely want to be able to create. You know the suggestion for the whole thing which it still seems to be bugged about it even though I've been pushing you like a few times to get this working but at the same time, I still wanna emphasize that it you know has to be smart about this."

- Scope: bounded W3 correction under existing staged-message target; DESIGN.md
  whole-selection synthesis correction is authoritative for this feedback.
- Prior art: related implementation is present; search-hub returned advisory docs,
  deep skill and programs. Run reranking/source-ledger retrieval degraded.
- H1: the recent deep run's evidence tool failed or never obtained new evidence.
  Test retained job/execution/tool events rather than mocked results.
- H2: complete-patch inspection is incorrectly required for whole-selection
  message scope. Test prompt, result normalization and drawer semantics.
- H3: deep input hides full inventory and only offers sequential patch traversal.
  Test compact input and targeted access on large immutable fixtures.
- H1 confirmed from retained job `message-06cd680d8eb96b2a5c863fc39b0a12af`,
  execution `01cf8825-9fe2-4f86-9544-5aad6ab249b3`, child run
  `199a2e27-549a-4d21-a6a6-fe6b3ae82c9b`: the exact evidence CLI failed
  `permission_denied: verified staged-evidence read capability required`.
  Zero receipts, no new patches, prior title/body and 26 IDs repeated. Saved
  owner/profile scope ceilings included GCT read before the run; no scope
  expansion is required. Retained evidence: `/tmp/gct-reported-deep-*`.
- Root cause: PersonalLocalProvider rejected verified agent provenance with
  FailureInvalid. Shared authn intentionally projects agent authority only when
  human providers abstain with FailureMissing. The earlier transport fixture
  supplied a synthetic human provider that abstained and missed this deployed
  chain. Actual buildAuthenticationConfig plus typed evidence transport now
  reproduces the exact failure before the fix (`/tmp/gct-whole-auth-red.log`).
  The provider now abstains for verified agents; OS-human authority is never
  supplied to an agent. Invalid credentials, missing capability, unsigned
  workflow coordinates and foreign executions remain rejected.
- H2/H3 confirmed in source: whole-patch inspection was required for full
  message coverage, while deep input omitted the full inventory and forced
  sequential whole-patch retrieval. These are product and evidence-access
  defects independent of the live authentication failure.
- Implementation and activation complete. The configured-provider regression
  passes under race (`/tmp/gct-whole-auth-final.log`); invalid browser proof
  cannot be bypassed by a valid child. UI has 76 focused passing cases and
  full lint/typecheck/build; served bundle is `index-C1lfIoRa.js`.
- Canonical lifecycle restarts activated the matching deep workflow 1.1.0,
  active source hash matches the declaration and prompt_stale=false. AM's
  asynchronous scenario sweep completed without failed declarations; its
  pre-existing optional resource degraded standing remains separate.
- Selected Workflow Health run `8c4e7ac0-b3bd-4e08-a05e-7d8977b2d5f6`
  passed 2/2 served-browser fixtures, including whole-selection message scope
  with 34/285 detailed inspection, a mocked eight-file read count, no partial-message
  warning and explicit adoption. Desktop and mobile cancellation/failure/prior
  fallback remain covered. SQL/file lease cleared cleanly with all four routing
  counters zero (transport fully mocked). Actual desktop result pixels reviewed.
- Scoped Test Genie run `20261022-212147-ebd6b8c5`: dependencies passed; unit
  still fails its existing UI test:coverage threshold (64.7% versus 85%).
  This is not a green full-scenario unit gate. Evidence:
  `/tmp/gct-whole-testgenie-wait.json`, `/tmp/gct-whole-unit-owner.log`.
- Constrained 390×450 served-browser execution
  `b106b258-0c72-44ac-883f-9a0b308c8ba9` completed all 82 entries. Actual
  780×900 pixels confirm the footer action remains reachable while content
  scrolls; no native physical-keyboard claim. Its separate paired lease cleared
  with zero routed writes/primary bypasses and no heartbeat errors. The first
  attempt stopped before BAS admission because CLI rebuild was denied by the
  host memory PSI gate; after the canonical CLI refresh completed, this run
  passed. Evidence: `/tmp/gct-whole-keyboard-{execution,isolation,timeline}.json`.
- Final independent acceptance found no remaining scoped source or layout
  issue. Staged identity stayed unchanged. No new paid/model execution was
  launched. The original live failure was traced, while corrected authorization
  and provider contracts are proved in regression tests; semantic message
  quality is not certified by transport receipts or mocked browser results.

- Backend format 4 separates typed whole-selection/limited message scope from
  detailed inspection coverage. Whole scope requires complete immutable
  inventory delivery to the exact owning execution; it never requires all
  patches. Legacy formats 0/2/3 and retained same-key retries keep their meaning.
- The existing evidence RPC/CLI supports metadata-only inventory and targeted
  stable-ID patches. Inventory omits blob IDs and patches, performs no blob
  reads, and permits up to 256 entries within the requested byte bound. Absent
  or zero edit counts mean unmeasured. Patch requests remain at most 64 IDs,
  permit important inherited-ID rereads and retry only unreturned target IDs.
- Regression first: the prior format-3 continuation failed selective synthesis.
  Passing 285/4,096-file fixtures now use 2/18 inventory pages, maximum
  16,307/16,346 bytes, and exactly two targeted patch reads. Whole-message scope
  coexists with truthful 28-file inspection (26 inherited). Missing inventory,
  invented patch inspection and invalid targets are rejected. No-progress
  limited explanations preserve previous text; bounded target retries and
  persisted format-3 admission retries pass.
- Full API/domain/store tests pass (11.032s/6.744s/0.017s), final focused cases
  and full CLI tests pass, and CLI primitive evidence was owner-regenerated.
  Proto verifier passes artifact
  `bc6a6825b471a3bf6d2cc2cddaa12100adc155ac4da4f7fbecb0fa6109fbf8b6`.
  Receipts: `/tmp/gct-synthesis-final-focused.log`,
  `/tmp/gct-synthesis-final-api-domain.log`, `/tmp/gct-synthesis-final-cli.log`,
  `/tmp/gct-synthesis-final-proto-verifier.log`.
- Separate post-green functional and maintainability reviews corrected unknown
  edit counts, target completion semantics, neutral inspection facts appearing
  as uncertainty, and exact supported-format validation. Independent frozen
  backend review found no remaining material finding. Receipts prove delivery,
  not model understanding or message quality. No paid execution or lifecycle
  mutation was performed by this lane; root owns deployment/browser acceptance.

## Prior correction — partial result continuation and action icons (2026-10-22)

Operator feedback (verbatim): "Thanks this is definitely better. So from what I
saw, it said there wasn’t any evidence gathered so I pressed the thing to spawn
the agent and it looked like the agent was going for a little bit and then it
gave me this partial result. But there’s no course of action for getting a full
result. I can either just accept the partial result or discarded. I guess I don’t
see why it doesn’t let me spawn an agent to get the full result. Am I missing
something? Also, it looks like the primary buttons and all the text buttons and
stuff. They probably should have icons with them to just make it easier at a
glance to you understand what they do."

- Scoped W3 correction under GCT-OT-P1-002; target is the partial drafting
  follow-up in DESIGN.md. IMG_1831.png shows an agent draft covering only
  26/285 staged files with a single Use partial suggestion primary action.
- Prior art: reuse this session's bounded large-selection implementation and
  drawer acceptance. Search-hub found the staged-message workflow/skill and
  advisory operations; agent-run retrieval remains degraded, not proof of no
  prior runs. Preserve staging, existing drafts and durable jobs.
- H1: the UI hides continuation for successful agent results. Reproduce with
  a succeeded partial job; reject if a visible action starts expanded work.
- H2: representative input and compact output enforce a permanent partial
  ceiling. Inspect evidence packaging, skill decision table and schema; reject
  if all staged patches can be reached and fully covered in the current run.
- H3: no related source is being mistaken for no staged evidence. Compare
  collection copy with snapshot details; staged diffs remain primary evidence.
- H1/H2 confirmed in source and failing regressions: successful agent output
  hides the completion action; quick packaging and workflow output cap complete
  coverage at 64 file IDs. The initial 26/285 draft is not evidence that all
  remaining changes were investigated. A second identical Start reuses the
  original sampled admission rather than providing deeper evidence.
- Design choice: explicit parent-linked deep workflow and typed paged immutable
  evidence reads through the existing advisory owner. No new multi-run job
  orchestrator, unbounded prompt, coverage inflation or paid validation run.
- Independent hook review found empty-collection continuation reloads could
  lose frozen retry intent, and an unknown child status could allow a new run.
  Both require regressions before acceptance; retain original parent and exact
  continuation key while admission remains uncertain.
- CLI integration: parent-only `message start --continue-from` requires an
  explicit new key and snapshot; contradictory collection/context/note overrides
  are rejected before transport. `message evidence` returns one bounded typed
  page without following its cursor or launching work. Six focused HTTP tests
  and the complete CLI module pass. Primitive evidence regenerated through its
  existing owning test, never edited by hand.
- CLI review A checked explicit launch authority, frozen parent/digest/key,
  invalid argument rejection and typed caller transport. Review B checked
  bounded cursor/byte parsing, complete-page versus complete-patch semantics,
  JSON receipts and absence of automatic follow-up requests. No remaining
  material finding; the domain's stale expected command count was updated for
  the new evidence operation after the broader CLI suite exposed it.

- Backend continuation inherits the original note/context and full immutable
  subject. A new format-3 workflow pages recorded old/new Git objects through
  the existing Advisory owner. Its prior job remains independently usable.
  Whole-patch receipts are bounded checks in the same job row, atomically added
  without changing the lifecycle revision or invalidating Start's handle CAS.
- Agent reads require `git-control-tower:read` and the exact owning verified
  workflow. Before the execution handle is saved, AM's retained execution must
  match the owner, deep workflow key, original idempotency key, job and input
  digest. Human inspection creates no agent coverage receipts. Deep result
  coverage accepts only inherited verified IDs or whole patches supplied to
  that exact execution; binary, oversized, unissued or truncated content remains
  uncovered. Existing format-0/2 inputs and quick workflow remain unchanged.
- Fake-runner integration supplied 259/4,070 new whole patches for 285/4,096-file
  subjects before Start returned; same-key retries start one execution and old
  note/context/draft survive. Real scratch-Git regression changes staging after
  pinning old/new OIDs, returns the original patch and preserves the index.
  Binary, oversized-page and mode/rename metadata cases are covered.
- Full GCT API/domain/store tests passed (11.959s/5.711s/0.021s), plus independent
  evidence authorization transport under `-race`. Actual AM catalog/profile
  parsers and narrow-child read gate passed. Its real structured-value validator
  accepted the 4,096-ID maximum ASCII-field output at 38,933 bytes. Receipts:
  `/tmp/gct-deep-final-api-domain-store.log`,
  `/tmp/gct-deep-am-owner-validation.log`, and
  `/tmp/gct-partial-evidence-auth-test.log`.
- Post-green review tightened exact source-set comparison, original-note
  inheritance, final page-completion byte accounting, typed denial codes and
  exact lifecycle response handles. The deep contract skill now starts with
  16 KiB pages and requires smaller same-cursor retries after tool truncation.
  A server receipt proves CLI delivery, not model visibility or understanding.
  Ninety-six calls and token/time/cost bounds may still leave changes partial.
- The deep profile retains protected empty workspace writes and no autoapply;
  its only declared API scope is GCT read. Native localhost networking remains
  advisory where the existing runner lacks loopback enforcement. Live policy
  inspection found the configured AM Cloudflare owner has AM write but no GCT
  read in its delegated scope ceiling; canonical identity issuance would strip
  the new child read scope. Activation must resolve that owner-policy boundary;
  no ambient token substitution, grant widening or live paid run was performed
  by this lane.
- Root deployment correction: the existing exact Cloudflare operator binding
  scoped to `resources=["agent-manager"]` now also contains
  `git-control-tower:read` (16→17 capabilities). All subjects, resources, other
  scopes and other bindings remain unchanged; no write capability or audience
  was added. This necessary feature deployment is covered by the operator's
  authorization for GCT/AM changes. A private 0600 preimage backup was retained,
  and the candidate and installed policy passed `authn.FromEnvironment` and
  canonical scope resolution before lifecycle reload. Human-only consent
  issuance remains untouched. Post-restart serving-process configuration confirms
  its sole selected Cloudflare owner can delegate the evidence read.
- Final UI acceptance: 72 focused cases, full UI lint/typecheck/build pass.
  Bundle `index-CcsuLl4Z.js` is served. Workflow Health selected run
  `3af65b3f-2d7a-49c0-aa96-1989929ecfea` passed both cases: desktop 81 entries,
  mobile 92 entries, every entry successful. These prove adjacent partial-use
  and investigation actions, distinct parent-linked admission keys, full result
  adoption, cancellation/failure fallback, and safe previous-draft adoption.
  Actual partial/footer, cancelled/previous and desktop full-result captures
  were inspected. Executions: `93c24821-e2b3-4d6d-8e77-ef6b3d973e31` and
  `1e33b865-9111-4f2d-aa11-a2cc693fc7bf`. SQL/file isolation installed,
  heartbeated and cleared without error; all four counters zero because all
  POSTs are fail-closed mocked. No real drafting request reached either owner.
- The first browser attempt failed because repeated mock registrations are
  intentionally idempotent, retaining the earlier failed response. Distinct
  equivalent patterns fixed fixture setup without weakening assertions or
  changing product behavior. Whole-catalog workflow schema/observer-label debt
  remains separate from the passing selected execution.
- Scoped Test Genie `20261022-202944-dac1a5cb`: dependencies passed; unit failed
  on the existing UI coverage command (64.7% against 85%). Focused feature,
  API/domain/store, CLI and race checks pass; the full scenario gate is not
  represented as green. No threshold was weakened.
- Canonical AM and GCT restarts completed. The active deep workflow is 1.0.0,
  source hash `d36833a8efbed025e98080fc9b736670a45c9d934a77187e3a666d717c30ad6d`,
  matching the authored declaration. AM's startup owner scan performed activation;
  a direct declarations CLI attempt correctly rejected this session's expired
  identity. No borrowed credential, private signer or owner-session substitution
  was used. No live paid model execution or physical-phone qualification is claimed.
- First constrained-height Chromium run `207089c5-2e6e-4710-942c-8cd392c96d39`
  proved both footer actions fit at 390×450, but one asynchronous fixture wait
  missed the brief Stopping state after cancellation had already completed;
  91/92 entries passed. The fixture now holds nonterminal responses until the
  acknowledged Stopping check, then releases the terminal response. Assertions
  were retained, the registry regenerated, and no product code changed.
- Live post-restart reads also confirm the declared deep profile's source hash,
  sole GCT read scope, protected/noapply settings and resolved 16 KiB-page prompt
  match the authored assets. Runtime policy inspection verifies the reloaded
  exact owner binding's delegated read capability; it does not stand in for a
  paid live child-run or physical-phone execution.
- Corrected constrained-height run
  `5b5fdb0a-b8f3-44e0-ad0c-218ac411eb86` completed all 93 entries successfully
  at 390×450. Captures are 780×900 at scale 2; both primary and investigation
  buttons remain reachable while content scrolls. The fresh paired lease
  installed, heartbeated and cleared successfully with all counters zero.
  This also executes the final mobile fixture's acknowledgement-controlled
  cancellation sequence. Source UI/production bundle is unchanged from the
  passing desktop/390×844 runs; no physical keyboard claim is made.
- Closing independent review after browser acceptance found no remaining
  material UI/adoption/identity/coverage issue. Source formatting and scoped
  whitespace checks pass. The real raw staged-index identity is unchanged across
  all browser checks and lifecycle activation. All browser model responses are
  fixtures; no paid drafting execution or repository mutation was used.

## Active correction — drawer clarity and drafting admission (2026-10-22)

Operator feedback (verbatim): "The new drawer doesn’t look that great I feel like
it could look a lot more professional and polished right now it’s a bit messy.
Also I pressed the button to spawn an agent and it looks like I didn’t do anything
or if it did do something there is no indication of that from the UI. So seems
like you know just needs more polishing and testing and cleanup and stuff."

- Evidence: IMG_1829.png shows the 285-file subject, repeated explanatory blocks,
  531 ungrouped omission records, and a failed terminal job with
  `start message workflow: unauthenticated: 401 Unauthorized`. The image proves
  admission failed, not that a drafting agent ran. Prior work explicitly did not
  exercise live admission; this is a newly observed boundary failure.
- Scoped W3 correction under GCT-OT-P1-002, target in DESIGN.md. Work shape:
  bounded convergence task. Preserve real staging, message drafts, owner jobs and
  request keys; do not broaden grants or bypass authentication.
- H1: outbound GCT service credentials are stale or fail renewal. Test: trace the
  actual owner client and runtime identity standing, then an expired-credential
  fixture through the canonical refresh path. Reject if valid identity reaches AM.
- H2: browser operator authority is dropped or confused with service identity.
  Test: capture safe actor/grant metadata at GCT→AM boundaries and owner denial;
  compare an authorized browser launch with fixture propagation. Reject if owner
  receives and accepts the expected actor/covered grant.
- H3: launch progress is below evidence content or absent while Start awaits.
  Test: hold Start unresolved in a component/browser fixture and require immediate
  visible admission status, then queued/running/failure/cancel recovery. A green
  request alone does not prove visible feedback.
- Related prior art: large-selection checkpoint below; search-hub recall found
  staged-message docs and skills. Agent-run and memory corpora were degraded by
  open reranker circuits, so search did not establish complete historical absence.
- Root coordinates integration/docs/lifecycle; backend lane owns authentication;
  UI lane owns composition; independent interaction lane owns hook regressions
  and post-green review. Final acceptance requires desktop/mobile captures and
  explicit asynchronous states, followed by cleanup and two fresh review passes.
- Confirmed trace: GCT Start returned a durable job at 15:11:37 EDT while the
  corresponding AM StartWorkflowExecution returned 401 immediately. A context
  search at 15:11:28 also returned 401. No AM execution was admitted. Runtime
  log paths had been deleted; registry-owned process output descriptors retained
  the relevant logs. Normal logs CLI and scenario-triage both reported unavailable
  logs, not an absent or disconnected scenario.
- H2 confirmed as a configuration defect: personal-local configuration replaced
  shared configuration (dropping OwnerBindings) and omitted GCT's managed
  Cloudflare provider. Explicit browser credentials could be replaced by a local
  OS principal, which has no portable proof to forward. Tunnel Manager's typed
  authentication binding read confirms GCT and AM use the same team/audience;
  audience expansion is unnecessary. Signed-credential regression fixtures
  reproduce and cover the boundary; the actual phone credential was not inspected.
- Direct local OS identity is intentionally not a cross-service bearer. Do not
  reuse Bridge-only OS1 enrollment/session keys. Portable signed browser proof
  remains required at AM. A future direct-local delegation service belongs in
  the control plane with short-lived, target-bound issuance, not a private GCT
  key or an ambient loopback-human bypass.
- UI review finding: point-in-time Get NotFound after an ambiguous Start is not
  proof the original admission cannot complete. Preserve its key and frozen
  intent; reconnect or retry that same request instead of allowing a new key.
  This is part of the launch-state correction, not an extra workflow.
- Live negative reproduction: the old served GetAuthorityStatus returned HTTP
  200, authenticated personal_local, even with an explicit invalid Cloudflare
  assertion. `/tmp/gct-drawer-invalid-proof-before.json` records this read-only
  probe. After activation the same probe must reject the credential rather than
  silently substituting OS identity.
- Authentication repair tests now exercise configured GCT middleware, actual
  Advisory Collect/Start, durable store and typed AM HTTP transport. The test AM
  independently checks signed human identity and its required capability. Valid
  proof reaches an execution handle; expired/wrong-audience proof and missing
  destination capability are rejected. Agent proof is never upgraded to human.
  Proofless local starts receive a precise sign-in precondition before contacting
  AM; no new private signer, enrollment-key reuse or ambient server token.
- Recovery cleanup uses one frozen admission request (collection, digest,
  selected IDs, note, key), persisted before Start. Nine focused hook cases
  pass, including late admission after 404, immutable retries, reload recovery
  and cancellation races.
- Post-green review caught and closed two more races. An old Wait cannot revive
  a cancelled job or clear acknowledged cancellation. A lost owner acknowledgement
  followed by a new authentication denial retains the original nonterminal job.
  Every handle-less dispatch now takes an exact CAS claim receipt; a concurrent
  retry invalidates an older refusal's response update. No new schema, lock
  registry or replacement execution API was introduced. Domain/store suites and
  focused race regressions pass, including deterministic first-refusal/retry
  interleaving. First definitive refusal remains terminal when uncontested.
- UI final feature checks: 55 focused cases pass; lint/typecheck/build pass.
  Final copy-only regression rerun passes all 41 composer cases. Build is
  `index-jHprPPCD.js`. Mockup and two source review passes are retained in
  `docs/mockups/commit-drawer.html`. Browser fit remains an acceptance check,
  not a claim from the mockup alone.
- Live owner-policy read: one configured AM Cloudflare owner binding exists and
  authorizes agent-manager:write. Same team/audience and owner policy are verified
  without exposing subjects or credentials. The phone's exact credential channel
  remains unobserved. Generic native-password GCT sign-in is not proven to be an
  AM audience; no AM audience policy was widened.
- Scoped Test Genie unit receipt `20261022-193842-886b986d` remains failed on
  the UI coverage command: 64.6% against the existing 85% threshold. Focused
  feature checks and backend/race regressions pass; the scenario gate is not
  represented as green and no coverage threshold was weakened.
- Canonical lifecycle activation completed healthy. The served entry is
  `index-jHprPPCD.js`. The invalid-proof live probe now reports an unauthenticated
  Cloudflare error instead of authenticated personal-local fallback. The passive
  authority-status endpoint intentionally returns HTTP 200 for this observation.
  Signed HTTP integration fixtures prove positive forwarding and destination
  capability checks; no paid live agent or physical-phone positive start was used.
- Served-browser acceptance: Workflow Health run
  `aa51503e-ccf9-4dc8-917d-2f8fd0278cf5` executed both selected drawer cases,
  passed 2/2, with all 56 timeline entries successful in each case. Desktop
  1440×900 and mobile 390×844 cover immediate starting, queued, running,
  immediate cancellation, stopping, cancelled, and a second failed launch.
  Actual running/prelaunch/failure screenshots were visually inspected. BAS
  executions are `16817bf6-1647-4fe2-840d-d62a7607782f` and
  `b9d5c84d-2060-4062-bd68-4be97d270fbd`. All POSTs are intercepted behind a
  fail-closed catchall; no real drafting, stage or commit request is sent.
  Routed database/file lease installed, heartbeated and cleared without errors;
  all four owner routing/write counters are zero, as expected for these mocks.
- Earlier browser attempts exposed assertion timing mistakes: BAS assertion
  timeout waits for the element, not its eventual text. Explicit heading waits
  now precede asynchronous assertions; immediate-feedback checks still follow
  clicks directly. One earlier run was interrupted by an unrelated lifecycle
  restart and is not counted as acceptance; its stale runtime lease was cleared
  through the shared owner. The final run completed without that interruption.
- Whole-catalog Workflow Health remains failed on existing legacy schema and
  observer-label findings, plus `/health` bootstrap route declaration findings.
  Selected execution success above is separate from whole-catalog certification.
  These Chromium fixtures do not qualify physical iOS keyboard behavior.
- Constrained-height acceptance: reused the original mobile flow with an
  ExecutionParameters fingerprint override of 390×450. BAS
  `cb5f6199-7962-45d9-8363-fcedd814e2ff` completed all 56 entries successfully;
  780×900 captures at device scale 2 confirm the override took effect. Inspected
  prelaunch and failure captures: content scrolls while the primary footer stays
  reachable. A fresh shared-owner paired lease was confirmed before execution,
  heartbeated and cleared successfully with zero routing/write counters. Only
  its own temporary database directory was removed after confirmed clearance.
  This proves constrained Chromium geometry, not a physical keyboard interaction.
- Closing hygiene: scoped whitespace check passes and the raw staged-index hash
  is unchanged across browser acceptance. No real commit, staging, paid model
  execution, grant expansion or Agent Manager production change was required.

## Active correction — large staged selections (2026-10-22)

- Rung: scoped W3 under GCT-OT-P1-002. Operator screenshot IMG_1818.png shows
  `[failed_precondition] staged selection exceeds 256 files; narrow selection`.
  The operator frequently stages more and authorized 8–16 times the capacity
  with intelligent bounded agent evidence. Target: 4,096 complete staged files.
- Spec: DESIGN.md large-selection correction. Backend owns full manifests,
  representative capture, retained matching and persistence; adapter owns
  compact prompt/output and job compatibility; UI owns whole-selection
  disclosure and bounded lists. Parent owns integration and validation.
- Initial bottlenecks: maxFiles256, durable draft192KiB, collection2MiB,
  workflow coverage arrays256 and repeated full-path output. Capture performs
  per-file Git calls for all entries. Actual AM structured input is capped at
  64KiB, so merely widening the advertised prompt budget would still fail.
- Decision: full subject stays in existing GCT durable storage, bounded compact
  AM input stays below the existing owner ceiling; private provider reads can
  receive the already-loaded canonical input for strict result expansion.
- Preserve real staging, commit history, proposals and existing jobs. Tests
  use scratch repositories and fake owners; no model launch is needed.
- Implemented: 4,096 full identities; 64 scope-balanced detail candidates;
  fair raw reads and 96 KiB of complete patches. Canonical collection/draft
  storage has independent 16 MiB bounds. The workflow receives at most 48 KiB
  context/56 KiB envelope and returns compact stable IDs. Unspecified files
  remain uncovered. Older jobs retain their subject and owner key.
- Review cleanup: whole-patch-first packing avoids clipping every medium patch;
  large lexical edge files cannot exclude all useful detail. Removed the obsolete
  runtime 192 KiB cap, made collection excerpt clipping UTF-8 safe, and changed
  coverage copy to count fully described files rather than implying known hunk
  counts for omitted details. Long UI lists use expandable previews.
- Deterministic evidence: actual 4,096-file scratch Git snapshot, 4,097 rejection,
  omitted-tail drift, exact retained identities without blob reads, long-path
  collection/admission/result expansion, legacy retry and strict coverage pass.
  Typed Connect exercised collect/start/get/wait with 64 covered/4,032 uncovered.
  Snapshot capture used 199 Git processes rather than work proportional to all
  4,096 file contents. Domain race suite passed. Final UI checks: 34 focused
  tests, lint and typecheck pass. Adapter tests and real AM structured-value
  validation preserve its existing 64 KiB ceiling.
- Scoped Test Genie unit receipt `20261022-171506-eaa2f9e2`: failed UI coverage
  command; UI coverage remains 64.6% against the existing 85% threshold. The
  scenario gate is not green. Shared generated owner preflight was fresh
  (`b1eeee66291d79e13690be22d7a15b79e1a194938b628781f359a70531bd9197`).
- Real-selection acceptance: the previously rejected 285-file staged index now
  collects successfully in 3.938 seconds with 39 complete patch details. Raw
  index identity was identical before and after collection. No drafting agent
  was started. Served UI is `index-D8GhTTJm.js`; canonical GCT restart completed
  healthy. AM startup's noncritical declaration sweep activated the new workflow
  at 17:21:51 UTC. Final read-only owner GET confirms version 2.0.0, revision
  `a76f1f14-63d8-4084-9796-f41449416a02`, and source hash
  `85b2334b11336b508068c4f1569861da5dfc838fa5b413ca196a7573f788483c`
  matching the declaration. The pinned prompt contains the new compact-ID
  contract and conservative coverage rules; its content hash is
  `sha256:c975af0840607d0e6fdb6c3e57d35646fa99e1a1e4ea327dcf8db08fecb6d92c`.
  Numeric skill provenance remains revision 1 and is not a file revision counter.
  Live Check confirms the collected snapshot still applies. Paid live drafting
  was not exercised; fake-owner and real owner schema checks cover that boundary.
- Work-record append was attempted with `vrooli-memory journal note --kind
  work-record`; the owner returned `unavailable: unexpected EOF`. Outcome is
  unknown, so the immutable append was not repeated. This checkpoint retains
  the trigger, approach, evidence and outcome independently of that outage.

## iOS focused editor displacement repair (2026-10-22)

- Follow-up `IMG_1817.png` at 12:35 EDT falsified the first repair's
  end-to-end conclusion. Native observations confirm build `index-DPblYM-6.js`
  loaded and scale now stays 1, but document focus scrolling still moves the
  app upward. Observation `0cbdd54cffe21b1b77fbb3250b7b8d4a` reports
  inner/visual height 449, scroll/offset 344, shell top -344 and bottom 105.
  A second sample (`e77e1e13925605ddedbe59c1483d8ac4`) has the keyboard
  established and still leaves the shell at -213..236. This rejects stale
  client code and remaining zoom as explanations for this follow-up.
  Owner response and bounded diagnosis:
  `/tmp/gct-ios-observations-20261022-followup.json` and
  `/tmp/gct-ios-diagnosis-20261022-1235.md`.
- The first fix's editing exemption was wrong for this internally scrolling
  app: native document focus panning must be recovered while unzoomed, even
  during editing. Its tests asserted positive shell height and no forced
  scroll without asserting shell position or focused-field visibility. The
  displaced native shell satisfied those weak checks. A new served-browser
  replay models the measured WebKit displacement using actual document scroll
  and reproduces shell -344..105 on the first repair build. Position, nav,
  keyboard establishment and subsequent document-only scroll checks fail:
  `/tmp/gct-ios-recurrence/before.json`. This is a synthetic WebKit geometry
  model in Chromium, not a post-repair physical iPhone observation.
- Follow-up correction observes actual window scroll as well as shared viewport
  changes, coalesces recovery per animation frame, and restores the document
  origin at scale 1 even while editing. Visual offset alone is not sufficient
  to issue a scroll reset; current native scale protects zoom before shared
  state publication. CommitPanel explicitly marks its scroll owner, allowing
  focused controls to be revealed with bounded inner `scrollTop` changes after
  page recovery and keyboard resize. This avoids scrolling ancestor pages or
  moving an already visible field. A separate post-green review simplified the
  reveal branches without adding another abstraction or viewport owner.
- Scoped owner run `uh-20261022-164256-9a363c9e201cda44232cee2a7f484018`
  executed all 578 UI tests across 67 files and typecheck successfully with
  zero blocking findings. After the readability cleanup, all 61 focused tests
  (15 viewport, 26 composer, 20 CommitPanel), lint with zero warnings and
  typecheck passed again: `/tmp/gct-ios2-{focused,lint,types}-final.log`.
  Aggregate status is degraded because the Code Facts database was locked
  (`SQLITE_BUSY`); this is not a passing scenario-wide certification. Raw
  response: `/tmp/gct-ios-recurrence/unit-workspace.json`.
- The related UI Health detection gap (a keyboard-visible shell at -213..236
  produced no finding) was filed separately through report-bug as
  `knw-1792687347225213154`; no analyzer implementation changed here.
- Closing follow-up production build `index-DJtw-_Wh.js` passed
  (`/tmp/gct-ios2-build-final.log`). The served-browser replay now restores
  scroll 344 to 0, shell 0..449, nav 398..449 and trailer 323..389 within
  its owned scroller 171..397. Title, description and trailer focus changes
  each keep the entire field visible and document at zero. A later
  document-only scroll of 213 is recovered, pinch pan of 120 is preserved,
  and keyboard dismissal restores the 852px canvas. All assertions pass;
  `/tmp/gct-ios-recurrence/after.json`, `after.png` and `replay.py` retain
  the evidence, including synthetic/native limitations and blocked server
  mutation requests. Both before/after screenshots were inspected. Physical
  iPhone confirmation of this second repair remains pending.
- Trigger: native recording `ScreenRecording_10-22-2026_09-09-18_1.mp4`
  shows the app disappear when References and trailers receives focus. This is
  scoped W3: editing must retain a visible application above the keyboard.
- Prior-art lookup found related viewport/overlay work, including swarm record
  `rec-d89e886c90d70b1c`; that record concerns overlay variable ownership and does
  not establish the cause of this full-shell symptom. Shared RCL stays unchanged.
- Competing hypotheses were input zoom plus forced pan recovery, a zero-height
  keyboard canvas, and an overlay-only fault. Native observation
  `b22c8546f82a9b8966331b8b542883d3` at 2026-10-22T13:09:22Z on build
  `index-D426SCUO.js` reports scale 1.33333337, viewport offset top 443.75/left
  31.25 and document scroll 444/31. Shell/body top -444, height 408, bottom -36
  establish displacement above the visible viewport, rather than zero height.
  No dialog was open. The 4/3 scale also matches the trailer's 12px font zooming
  to 16px; title/description were 14px. Owner response retained at
  `/tmp/gct-ios-observations-20261022.json`.
- Separate Chromium viewport replay confirmed the GCT hook forced `scrollTo(0,0)`
  during focused, zoomed viewport panning. Focused tests reproduced that override
  and the same conflict during scale-one keyboard reveal before the repair.
- Repair: commit text fields, optional agent note/select and related author
  inputs use at least 16px, including landscape phones and tablets. A scoped
  compact override requires width at least 768px, hover and a fine pointer.
  GCT recovers page overscroll only while unzoomed and no editable field has
  focus. Browser focus/pinch panning remains browser-owned. Normal established
  keyboard height, installed frame restoration and idle overscroll recovery
  remain covered; no guessed minimum-height or disabled user zoom was added.
- Focused receipt: 53 passing tests across 3 files (7 viewport, 26 composer,
  20 CommitPanel), `/tmp/gct-ios-focused-final.log`. The two earlier regression
  failures are retained in `/tmp/gct-ios-viewport-red.log` and
  `/tmp/gct-ios-focused-pan-red.log`. Lint, typecheck and production build pass
  (`/tmp/gct-ios-{lint,types,build}.log`, build `index-DPblYM-6.js`).
- Closing browser replay uses the served build: portrait 393px, touch landscape
  852px and touch tablet 820px compute all three editor fields at 16px; 1280px
  fine-pointer desktop retains 14/14/12px. Each focused zoom trace now records
  zero forced scroll resets, compared with one before repair. The portrait
  integration trace establishes keyboard height 449px and dismissal restoration
  to 852px. Replay inputs are synthetic visual-viewport events, not native OS
  keyboard pixels; potentially mutating server requests are blocked in the
  isolated contexts. Initial wider replay assertions incorrectly assumed the
  mobile-only shell existed at desktop breakpoints; the harness now measures the
  actual application root and waits for initial navigation requests to settle.
  Receipts: `/tmp/gct-keyboard-video/*-replay.json` and `replay.py`.
- Scoped Unit Health UI execution
  `uh-20261022-131928-d6cea519933ca885bc1b53063ccc34c0` passed all 570 tests
  across 67 files plus typecheck, with zero blocking findings. This used the
  documented `--workspace ui --execution --fast-test-only` iteration path;
  coverage and scenario-wide certification were not rerun or represented as
  passing. Independent final CSS/viewport review found no remaining issue.
- BAS production mobile capture `065a4b3f-9c92-4bd4-a93f-c228abc1d033` reached
  explicit editor readiness; screenshot inspected. It proves served layout,
  while replay/tests prove the repaired code paths. A fresh physical iPhone
  interaction on the repaired build remains unverified; the native observations
  cited above establish the original failure, not a post-fix device verdict.

## Active delivery — staged commit messages (2026-10-21)

- Authority: operator approved the evidence-first composer and full implementation,
  validation, orchestration through sub-agents and a dedicated post-validation
  refactoring/debt pass. Related GCT/Agent Manager/shared contract changes are
  authorized. No real repository stage/commit/push is part of this delivery.
- Shape: bounded delivery; target and acceptance live in
  `../../DESIGN.md#staged-commit-messages--approved-2026-10-21`.
- Rung: scoped W3 under OT-P1-002 and existing advisory/human-authority targets;
  existing planned requirement will be reconciled with concrete passing evidence.
- Initial evidence: deterministic draft formatter has no generation; proposal
  capture retains bytes; ordinary composer is a single multiline text field;
  conventional validator incorrectly applies a single-line pattern to all text;
  staged full-content display may read worktree bytes; AM time-window message/tool
  search and durable generic workflows exist. Research and approved mockups are
  in the conversation and the local artifact directory referenced by DESIGN.
- Preflight: `cd packages/proto && make verify-committed-gen` passed before edits.
- Work lanes: GCT domain/typed transport/persistence; AM context/execution owner
  and adapters; UI composer/feature state. Orchestrator owns integration,
  validation receipts, independent review/refactoring and final checkpoint.
- Status: scoped feature implemented, activated and reviewed. Functional proofs,
  final browser checks and the dedicated subsequent cleanup pass are complete.
  Broader scenario certification remains failed on the coverage and existing
  contract issues recorded below. Earlier checkpoint counts are historical.
- Integration decisions: shared immutable input budget, complete index manifest
  with explicit patch omissions; lost admission acknowledgments reconcile through
  the original request key; cancellation races retain authoritative standing;
  trusted registered repository roots reach the generic AM workflow boundary.
  Existing native-command/outer-model-network separation preserves configured
  model access while protected execution cannot publish repository changes.
- Generated consumer refresh: canonical proto owner generation completed; stale
  UI file dependency refreshed through SDA `deps install` (approved, installed),
  after which UI typecheck passed. Dependency validation passed in advisory mode
  with existing scenario-wide warnings; no new external dependency was added.
- UI lane functional and separate cleanup checkpoint (2026-10-21): feature-owned
  `ui/src/features/commit-message/` now owns title/description/raw unknown-trailer
  editing, typed deterministic collection, retained-source defaults, optional
  explicit agent start, durable job/request-key reconnect, owner Wait/cancellation,
  elapsed/last-known standing, staged applicability checks and edit comparison.
  Applying only fills the editor. Existing human commit authorization is retained.
  Both actual CommitPanel mounts share this composer; `primary=commit` now reaches
  the mobile Commit tab. Late applicability responses cannot modify another repo's
  editor. Immediate unchanged Wait replies are paced; transient observer failures
  stop until explicit reconnect. Speculative/session sources remain unchecked;
  only first-suggestion verified retained sources are selected by default, at most 20,
  and operators can clear them. No attribution trailers are inferred.
- UI cleanup removed the old all-approved-message button/App/hook wiring, restored
  canonical scaffold/generated coverage exclusions and declared 85% thresholds
  without relaxing policy, and moved unique sidebar-slot allocation into existing
  `AppSelection.ts`. Commit-primary now keeps Changes/History/Diff rather than
  duplicating History; five primary-panel layout regressions preserve default order.
  Canonical selectors use `commitMessage.*`, generated via selector:manifest.
- UI focused receipts: 66 passing tests/5 files (26 composer journeys, 5 message
  serialization, 20 CommitPanel, 6 AppPanels, 9 AppSelection); full output at
  `/tmp/gct-message-ui-final-tests.log`. Final lint/typecheck, selector owner
  generation/check and production build passed after retained-source defaults
  and layout cleanup. The build retains the existing large-chunk warning.
  Conventional title-only validation plus multiline/body/trailer/invalid-title
  Go regressions passed; only the title is validated as conventional syntax.
  Parent owns integrated suites/live observers and final receipts; no live agent,
  real staging, commits, pushes or proposal writes were exercised by this lane.
- Final applicability review reproduced a delayed accepted Check crossing a
  complete stage/unstage cycle. Acceptance now binds repository, staged digest,
  status/index generation and newer rejected standing through editor adoption;
  one bounded fresh check permits worktree-only refreshes. Both regressions were
  observed failing before the fix. Final full UI execution passed all 566 tests
  across 67 files; the coverage gate remains failed against the restored 85%
  thresholds (lines/statements 64.46%, branches 68.02%, functions 60.35%). This is
  a scenario-wide coverage limitation, not a passing certification; no baseline
  attribution of the entire deficit was established. New commit-message files
  measured 94.61% lines/statements, 85.39% branches, 76.78% functions. The first
  full run retained one new regression failure (565 passed), corrected before
  the final run. Full receipt: `/tmp/gct-message-ui-full-coverage-final.log`.
  Final lint (zero warnings), typecheck and production build passed, with logs
  `/tmp/gct-message-ui-{lint,types,build}-final.log`; existing bundle-size and
  Browserslist-age build warnings remain. No coverage threshold was reduced.
- Browser gates: added desktop/mobile composer observer cases and regenerated
  `bas/registry.json` through `test-genie registry build`. Initial selected run
  `8f9108b9-9d2c-4f78-841f-a4d9b2915680` executed both cases and failed composer
  visibility assertions before the final UI build/lifecycle refresh (capture
  still shows the Changes panel).
  Retain this failure, then rerun after deployment. Broader catalog reports
  existing legacy schema/observer-safety defects outside these two cases.
- Browser follow-up: selected run `28b12b1a-8f82-43b0-b3af-1f5d37cdfbe3`
  executed and passed both new observer cases after the UI build (BAS mobile
  `5bf1d070-26c8-4e42-b8a0-7a93b4b8eb16`, desktop
  `c68390de-47d6-4329-85c3-84d5c6d5d015`). Orchestrator inspected the actual
  screenshots. Aggregate workflow verdict still fails on legacy catalog defects;
  the selected browser execution count is 2 passed, 0 failed. Desktop capture
  exposed an existing duplicate History stack selection when Commit is primary;
  that related layout helper is being corrected and revalidated.
- Cleanup checkpoint: UI focused suite grew to 53 passing tests; lint,
  type-check and build passed. Removed obsolete approved-message App wiring;
  restored canonical coverage exclusions/85% thresholds. Independent static
  `unit-health validate scenario git-control-tower --workspace ui` now passes
  with zero findings. Backend removed recursive reconciliation/projection paths,
  made cancellation monotonic against persisted state, bounded Git subprocess
  output while streaming, and limited unreferenced collection retention. AM
  full workflowcatalog/workflowruntime/orchestration package tests passed.
- Final proto owner `make verify-committed-gen` passed after schema generation.
  AM API source is stable; canonical lifecycle restart is in progress. CLI parity,
  final scoped suites, remaining independent review fixes and final receipts are
  still required before completion.
- Checkpoint rule: append concrete decisions, files, proofs, failures and remaining
  work here before compaction or handoff. Preserve prior delivery entries.

### Final staged-message delivery checkpoint — 2026-10-22 UTC

- Delivered: exact-index collection and conservative retained-edit matching;
  bounded path/text/file-mtime event retrieval; independently disclosed context
  relevance and source limitations; separate title/description/trailer editor;
  full/partial/no-message suggestions; explicit protected drafting; durable
  request-key admission and reconnect; confirmed cancellation; strict structured
  result validation; stale-selection and concurrent-editor adoption guards;
  typed browser/AM contracts and CLI collect/check/start/status/wait/cancel.
- Independent review fixed two result-size limits (20 KiB decoder and implicit
  16 KiB workflow terminal binding), preserved independently verified credential
  channels, and reproduced/fixed a completed staging cycle crossing an awaited
  applicability check. Review found no remaining blocker in the feature scope.
- Dedicated post-validation cleanup removed obsolete approved-message plumbing,
  recursive backend reconcile/projection paths and an unused duplicate test
  companion. It centralized cancellation in monotonic SQL, bounded streaming
  Git output and collection retention, restored declared UI coverage policy,
  and fixed duplicate sidebar History allocation when Commit is primary.
- Passing evidence: 566 UI tests/67 files, 66 focused UI tests/5 files, UI lint,
  typecheck, selector checks and production build; 60 race-enabled backend
  domain/store/proposal tests plus 8 race-enabled typed transport/adapter tests;
  all 12 CLI packages and generated primitive evidence; full AM workflowcatalog,
  workflowruntime and orchestration packages; full policygate; actual resolved
  workflow declaration; canonical proto generated-artifact verifier.
- Test Genie AM `20261022-032302-b40db304`: unit and contracts both PASS,
  152 observations. GCT `20261022-033856-aba7b7e1`: structure/API PASS;
  unit/proto/contracts FAIL. Follow-up GCT `20261022-034528-27f18d37` confirms
  only UI test:coverage execution and the existing duplicate CLI binding remain
  blocking in unit/contracts after cleanup; the duplicate helper error is gone.
  All 566 UI assertions pass, but aggregate coverage is below the unchanged 85%
  gate (64.46% lines/statements, 68.02% branches, 60.35% functions).
- Existing contract findings were verified in HEAD
  `150768340d31687e4c982091b83a167ad38ecdfb`: repo reuses
  `proposals.ProposalTrailer`, and repo/diff plus proposal/diff truthfully share
  `RepoService.GetRepoDiff`. No nonexistent proposal RPC or false binding was
  invented. Shared-type failure reported through the report-bug skill as
  `knw-1792640659199380661`. Broader legacy BAS catalog failures remain; neither
  selected browser passing nor generated-artifact freshness is full certification.
- Final browser run `f263360d-9b16-4e82-9557-bd8fef239adc`: 2 executed,
  2 passed, 0 failed. BAS mobile `c6d1c90f-1a70-4854-978b-655bce0752d0`
  and desktop `e39530ee-9f38-4ede-9520-d5b29e83db3a` screenshots were inspected.
  Desktop additionally asserts the retained Diff panel after the layout fix.
  These are browser observations, not physical-device or paid-model evidence.
- AM and GCT were restarted through canonical lifecycle and returned healthy
  durable start verdicts. Supported owner GETs verified active workflow revision
  `5b613828-3c8b-45e0-8baf-2e92a83dba07`, digest
  `sha256:57b70fa2415c2d51f72d900072afb5df5973db0a3f26c025ed25e8be439425e3`,
  and final profile `7beaa052-7716-4366-8139-09ee13f9e426` with source/applied
  hashes matching the files. Final workflow uses explicit 256 KiB terminal
  output, trusted registered-root binding, protected empty write policy, no
  autoapply, no API scopes and separated model/command network settings.
- Limits: no live paid/native runner qualification. This session's expired
  agent credential cannot refresh; live workflow validation/start is denied.
  No credentials, identities or grants were substituted. Context retrieval is
  policy-filtered but global; recorded/unknown repository scope and import
  freshness remain visible retrieval-quality limitations. No source match is
  an authorship claim. No real repository staging, commit, push or proposal
  mutation was performed.
- Requirement GCT-OT-P1-002 now records concrete passing validation references.
  Its formal status remains in_progress and the PRD box remains unchecked:
  business-health requires a run-coupled requirements-sync snapshot to earn
  complete status. A broad sync would rewrite unrelated requirements; that
  certification migration is outside this bounded feature delivery. Feature
  implementation and the scoped validation results above are delivered.
- Final work record: `8bf4d742-413f-442a-835b-fb5fc59eb264` (trigger, approach,
  evidence and outcome). Final scoped whitespace checks passed.

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


## Large-change workspace responsiveness — 2026-10-15

- Authority: operator requests investigation, Git-noise cleanup, significant UI
  performance improvement and related UI code-quality repair. Shape: bounded
  convergence task; no Plan Manager or separate Swarm goal required.
- Rung: W3 scoped implementation. Prior art:
  `../perf/2026-09-07-ux-audit.md` measured flat-list virtualization benefits;
  current grouped mode still uses eager FileSection maps. Search-hub recall
  found related work; source-ledger recall timed out and agent/swarm ranking
  was degraded, so recall completeness is unknown.
- Baseline: `git status --porcelain=v1 -uall` reports about 10,804 paths;
  7,204 untracked Secrets Manager desktop paths (7,049 copied resources),
  1,851 untracked delivery-operations paths and 424 external-delivery paths.
  These counts include legitimate source and design documents.
- Expected design: one shared measured virtual list for flat and grouped
  Changes, including bands, group headers, category headers and file rows.
  Preserve both staged and unstaged appearances of partially staged files,
  collapse persistence, filters, all file/group actions, mobile selection and
  reveal/scroll behavior. Keep scrolling state below the FileList model.
  Remove the unreachable legacy flat JSX and duplicate category rendering.
- Hypotheses: (1) expanded grouped DOM grows with total changed files; test with
  real virtualizer and a 10,000-file fixture. (2) group classification drops
  partially staged paths because a single Map category overwrites the first;
  test both action routes. (3) repeated parent/query and push-safety context
  updates amplify work; inspect consumers and measure the governed journey.
- Git hygiene: ignore only reproducible desktop bundle copies and generated
  delivery media/evidence, retaining source, design/operations Markdown and
  scripts. Preserve files in place so live consumers and references keep working.
- Validation: focused regression tests, UI type/lint checks, scoped Test Genie
  unit/lint/performance phases and before/after performance-health capture.
  Record failures and unavailable evidence without treating them as passes.
- Harness handoff: Work until this verified end state is true: the design above
  holds under large-change regressions and the governed performance journey is
  measured or explicitly unavailable. Boundary: git-control-tower UI/docs/BAS,
  existing desktop/docs ignore rules. Extend with a recorded reason for related
  owner fixes. Keep going through friction; step back when progress stalls.
  Blocked means a decision, credential, or approval you lack. Checkpoint here on
  interruption; report changed, verified, remaining and unverified. Before
  handoff, run two fresh reviews for correctness, edge cases, maintainability
  and visual behavior; resolve material findings and retain evidence here.

### Scope extension: grouping polling cost

The grouped-list RPC calls full `GetRepoStatus` every five seconds although it
only consumes changed paths. This repeats binary inspection, diff line counts,
untracked content reads and author resolution on the UI polling path. Extend the
repair boundary to `api/repo_diff_connect.go` and its focused grouping regression:
reuse the existing fresh porcelain-only snapshot and preserve group semantics.
This is the same scenario/behavior and does not change the mutation boundary.


### Implementation and review checkpoint

- Shared virtualization now covers grouped headers/bands and all expanded
  category/file rows. Removed the eager FileSection implementation, its orphan
  props, and unreachable legacy flat JSX (over 500 net lines removed).
- Stable path/category keys preserve measurement identity across inserts and
  collapse. Stable virtualizer callbacks retain its measurement cache on scroll.
  The local list resolves its ancestor on first mount before the ancestor ref
  attaches; real-virtualizer tests caught this previously masked layout edge.
- Both categories of partially staged files are preserved. Selection fallback
  works without a caller-supplied key Set. Navigation opens collapsed target
  groups/categories before scrolling and completion; deferred scroll restoration
  cancels both frames on unmount.
- App file-list callbacks now depend on stable mutation functions. FileList is
  memoized. Push-safety context values remain stable during fresh clock ticks;
  staged badges use one shared path index instead of scanning the report per row.
- Scoped backend regression first failed on DiffNumstat/ConfigGet work. Group
  polling now reads fresh porcelain paths, sorts categories as before, deduplicates
  partially staged membership and propagates Git failures. Six focused Go tests
  pass; proto owner `make verify-committed-gen` passed (artifact selected by
  lifecycle: e60d8071bf7dab60f60255031e8c572de80fea23cd851996a8ecc33111fb389d).
- Read-only live benchmark (three iterations, after ignore cleanup): full
  presentation status 871.6 ms mean; fresh path snapshot 459.2 ms mean (47% less
  for this component). This is not an end-to-end action or browser latency claim.
- Initial regression evidence: grouped partially staged rendering lost one copy;
  eager 10,000-file grouped mount exhausted a 4 GiB Vitest worker. After repair,
  actual TanStack range logic (only jsdom geometry supplied) keeps fewer than 60
  mounted file rows at 10,000 changes and fewer than 30 headers at 1,000 groups.
  Distant-row selection, collapse/reopen and selection restoration pass.
- Full UI suite: 64 files / 525 tests pass. Focused navigation/safety tests,
  TypeScript and touched-file ESLint pass. Later stable-callback work receives a
  final focused rerun; broad Test Genie unit/performance run is server-owned as
  `20261015-192354-8a40dcef`. The current CLI has no lint phase; ESLint is direct.
- Review pass 1 found the first-mount ancestor-ref gap, index-based measurement
  cache invalidation and collapsed navigation; all repaired with focused guards.
  Review pass 2 checks action closure freshness, staged/unstaged preservation,
  generated/source ignore boundaries and bounded scroll work. No remaining
  material defect found in those reviewed paths. Final browser/gate receipts follow.


### Final evidence — 2026-10-15

- After deferring settings/review/proposal modules, production entry JavaScript
  is 1,156,444 bytes (315,644 gzip), down 18.8% from 1,424,087 bytes. Settings
  retention regression proves drafts survive close/reopen after first lazy load.
- Final populated grouped trace: BAS
  `9f19f7c8-a620-4406-a3a6-6b1f32a5b98f/fd062d5e-a51c-4d2a-81e3-3ed56c0f7503`.
  FileList 2.5 ms average / 11.3 ms max; App 4.1 / 26.1 ms;
  browser Layout 23.5 ms total / 5.8 ms max; scroll events 1.0 / 1.7 ms.
  No original-versus-final grouped percentage claim: the initial default trace
  captured only the shell before status arrived. The default flow now waits for
  actual rows. Frame-duration/FPS findings remain unreliable and are reported
  to Scenario QA as `knw-1792092650729110423`.
- Populated desktop BAS capture: `a5cf22cd-7bf1-42b6-b099-ebe9ec240791`;
  populated mobile capture: `23d083ef-5325-49f8-8c95-8e164e43dfbe`. Both inspected
  visually; file rows fit their owned scroller, and mobile navigation stays
  pinned. These are Chromium observations, not physical-device qualification.
  A first screenshot attempt coincided with lifecycle restart and failed
  navigation; a later healthy-runtime capture succeeded without a code workaround.
- Final full UI run: 65 files / 526 tests pass. A subsequent filtered-run-dot
  regression plus the affected large-list/lazy-settings tests pass (29 tests).
  Six Go grouping regressions also pass under `-race`; touched-file lint,
  TypeScript and scoped `git diff --check` pass.
- Review follow-up found two further hot-path/edge issues: group action arrays
  were rebuilt when a visible header rerendered, and precomputed run dots could
  include paths excluded by the active filter. Group summaries are now computed
  with the model and respect its filtered paths; a new regression covers the dots.
- Fresh closing review A: checked staged/unstaged actions, all category
  discard/ignore routes, group confirmation closures, filter summaries and
  persisted navigation; no material issue found.
- Fresh closing review B: checked stable measurement callbacks, offscreen
  reachability, mount/restore cleanup, lazy form retention and source-vs-output
  ignore examples; no material issue found. Neither review claims an independent
  acceptance verdict or full scenario certification.
- Git hygiene examples: generated bundle resource/media/render paths are ignored;
  native-host source, Electron source, delivery START-HERE/verifier and authored
  prototype HTML remain visible. No file relocation, shared-index mutation or
  history rewrite was needed. Already tracked generated payloads remain tracked.
- Wider unit run `20261015-192354-8a40dcef` remains failed on existing UI policy
  projection, companion helper and CLI execution debt. CLI finding filed as
  `knw-1792092503178001899`; no CLI changes were made here. The broad unit gate
  is advisory for this repair and is not represented as passing.

- Performance-only rerun `20261015-192848-a0c426f6` remains FAIL on Lighthouse
  0.66 < 0.75 (previous 0.64). No budget or score threshold was weakened.
  Existing initial-load performance debt remains measurable; the grouped-list
  interaction, DOM bounds, bundle reduction and fresh-path read improvements
  are separately evidenced above. No FPS claim is made from the analyzer.
- The runtime is rebuilt/restarted through the lifecycle after final source
  edits. No real staging, discard, commit, push or proposal mutation was used
  for verification; action semantics are covered in disposable/fake fixtures.

- Final unchanged-code UI receipt: all 65 files / 527 tests pass, including the
  filtered-run-dot regression. Lifecycle restart reported healthy at
  2026-10-15 15:31:32 America/New_York; API 18710, UI 21400. The final scoped
  whitespace check passes. Closing reviews were rechecked against this final
  source with no additional material findings.
