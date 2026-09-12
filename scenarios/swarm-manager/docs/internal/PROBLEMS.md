# Known Problems & Technical Debt

## Open issues

No cutover-specific open issue is known.

### Execution modes supersede execution strategies — 2026-09-11

Audit: <https://claude.ai/code/artifact/9a603669-8080-419d-a662-3f47ffe4f2f9>.
Scope: <https://claude.ai/code/artifact/9343ee6a-89c7-4d3f-ab0f-57d801b46e73>.

- The goal route as a workflow (`goal-session` strategy,
  `swarm-manager/goal-session-drain`) is superseded by goal mode as one Agent
  Manager run with `/goal <finish line>` installed. The three stored
  `execution_strategy` values collapse to `execution_mode: sliced | goal`, with
  plan shape (`phased` | `mandate`) carried on the Plan Manager plan. The
  interim workflow and the `swarm-manager-workflow-goal-session` prompt retire
  with it.
- Continuation keyed on `budget_exhausted` never applies. The typed apply for
  `plan.execute` requires a succeeded workflow, so a `budget_exhausted` result
  never reaches the sweeper that would create the continuation child. The
  target design resumes only involuntary interruptions (usage window, timeout,
  crash, session lost) under `continuation: until-allowance`; agent-decided
  verdicts go to finalization.
- The Run dialog crashes when the first runner in the execution catalog has no
  model list. The target dialog reads the model list per selected runner and
  tolerates its absence.
- The edit dialog neither loads nor submits the execution fields (mode, limits,
  continuation, scope policy); it shows defaults and relies on sparse patch
  semantics to leave the stored values untouched.

### Owner-grant boundary — 2026-09-09

- Agent Manager now accepts and durably stores `WorkflowEngagementGrant` on a
  top-level workflow execution. Admission rejects grants that widen the pinned
  declaration, advancement reapplies the grant after catalog reload, and an
  idempotency replay cannot replace the original grant.
- The operator projection exposes the retained grant. Swarm's development
  coordinator reserves before dispatch, carries the exact token/wall envelope
  into the owner request, binds the returned execution identity, and settles
  only measured terminal usage. Unknown usage leaves the reservation held.
- Native-goal dispatch is intentionally fail-closed. The coordinator is not
  composed into a launch transition yet; provider cadence, restart routing,
  native/fallback parity, and production evidence adapters remain open. No
  Audio item, approval, goal, or agent was created.
- Focused `-race` validation passes for `internal/development`,
  `internal/agentmanager`, and `internal/transitionrunner`. Agent Manager's
  workflow-runtime race suite and focused handler projection/start tests also
  pass. Broader Test Genie failures recorded below remain separate and are not
  reclassified as evidence for this owner-boundary change.

## Contract-driven development readiness — 2026-09-08

Retained for provenance. The contract-development route this section
qualified is retired in the target design; a plan-backed item now carries its
own grant and runs in sliced or goal mode. See
[Retired route: contract-development](../concepts/ARCHITECTURE.md#retired-route-contract-development)
and [SCENARIO_DEVELOPMENT.md](../../../../docs/agent-system/SCENARIO_DEVELOPMENT.md#implementation-status)
for what still exists in code. Do not create new items on this route.

Latest continuation — adopted budget policy (supersedes the policy question below):

- New reviews explicitly bind `metered-cancellation` (default) or `hard-ceiling`
  to the goal, fingerprint and immutable approval. The drawer reviews this choice
  without remembering authority. Historical approvals keep their prior semantics.
- Swarm retains metered token overshoot and subtracts it from the next reservation.
  Aggregate exhaustion prevents more work. Acceptance still requires owner evidence;
  metered token overshoot alone does not invalidate an otherwise proven outcome.
  Wall-time overruns and operator revocation retain strict disposition.
- Agent Manager has a limited opt-in sequential fresh-run meter with durable stop
  intent, restart retry, terminal usage settlement and cancellation accounting.
  Unsupported graphs and hard ceilings fail admission. No live workflow declaration
  has been opted in, and no Agent Manager process was restarted for this change.
- The complete engagement loop remains unavailable. Required next work includes
  per-engagement grant admission/fencing, lost-dispatch reconciliation, native versus
  fresh-run fallback continuation, provider cadence qualification, production
  outcome adapters, new-item shape selection and completion/acceptance controls.
  The interpreter meter does not authorize Swarm launch. No Audio item or agent was
  created. Numeric pilot budgets and acceptance bands remain unset.
- Protocol generation was refreshed for Swarm. A focused UI test caught an installed
  `file:` package still using the old schema; the governed dependency installer
  refreshed it. This was a real serialization failure, not a waived test.

Validation for the budget-policy continuation: Swarm development tests pass with
race detection; Agent Manager's full workflow-runtime and workflow-catalog packages
pass with race detection, as do focused orchestration metering checks. Swarm's
main-package development/transition regressions and full trimpath CLI suite pass.
The three development UI files pass 13 cases, TypeScript checking and focused lint.
These are source-level checks; neither shared service was restarted and no live
provider or Audio product run was used.

Scoped Test Genie unit runs both returned FAIL:

- Swarm `20260909-021934-27d5afc9`: terminal persistence degraded with `run not found
  in index`; artifact retrieval also fails. This repeats the recorded owner issue
  `knw-1788901003359886533`, not a new passing certificate or duplicate bug report.
- Agent Manager `20260909-021934-8c499131`: API coverage command timed out, UI
  coverage command failed, and architecture/coverage findings remain. The retained
  findings artifact is `artifact_e6e2b75716f0791c82db1296e10a3b77`. No attribution
  of these broad failures to unrelated code is claimed. Final focused tests include
  edits made during that broad run; the broad observation is not exact-input proof
  for the final mutable worktree.

Continuation checkpoint — goal configuration and execution admission:

- Added the goal-review drawer, server-owned fingerprinted guidance, protected
  outcome/artifact editors, remembered non-authority defaults, stale-preview
  protection and amendment field/artifact comparison. Bound development items
  now route to contract review instead of plan authoring/retry. New-item shape
  selection, runtime launch, completion submission and acceptance UI remain open.
- Final boundary review found that the older transition guard only covered
  backlog subjects, while plan execution uses an execution-record subject.
  Added an execution-owned guard for direct/automatic/forced queue admission,
  pending start and plan/correction/follow-up/spec-sync input builders. The
  regression verifies refusal without queue status changes or agent dispatch.
  Full execution package tests pass with `-race`. Atomic cross-store admission
  and shape selection remain part of future launch qualification.
- Agent Manager regression tests reproduced known-spend schema-repair bypass,
  overlarge child turn/time requests and another agent admitted at exact token
  exhaustion. The sequential interpreter now checks these boundaries before
  dispatch/repair and clips supported child limits. Full workflow-runtime tests,
  including race checks, pass. This is not a hard in-flight token ceiling or
  qualification of concurrent/subworkflow reservations.
- Hard token ceiling versus metered cancellation with possible in-flight
  overshoot remains an explicit operator policy question. No new policy has
  been adopted. Keep launch unavailable; do not infer a guarantee from prompts.
- Swarm development/backlog/next-action packages pass with `-race`; focused
  main-package development tests and the complete `go test -trimpath ./...`
  CLI run pass. UI checks are recorded at the final checkpoint below.
- Scoped Test Genie unit run `20260909-001739-53bf19fc` returned FAIL, including
  API/CLI command failures and unit-policy findings. The subsequent CLI run
  passed, but does not explain or erase that receipt. Artifact retrieval returned
  `run not found in index`, matching the previously recorded owner persistence
  issue (`knw-1788901003359886533`); detailed attribution remains limited.
- Audio skill-set validation passes. Its 15 evidence-board tests initially failed
  because the program adopted `program.inputs/classify` while its test fixture
  still injected old globals. The fixture now injects that interface and uses
  the real shared error classifier; all 15 tests pass without changing evidence
  expectations. This does not certify a live kernel session or audio quality.
- The Audio proposal now includes explicit proposed engineering guidance. Its
  budgets remain unset and its SLO/corpus decisions remain proposals. No Audio
  backlog item, approval, harness goal or development agent was created.

Final focused checkpoint: 20 UI cases pass with type-check and focused lint;
Swarm execution/development/backlog/next-action packages pass with race detection;
the main-package development tests and complete CLI trimpath suite pass. Agent
Manager's scoped Test Genie unit run `20260909-002357-03cecc07` returned FAIL
with execution-readiness and architecture/coverage findings. Neither scenario
has a passing broad unit certificate from this continuation. The generated live
Audio preview resolves 16 artifacts, retains `budget_required` and all runtime
blockers, and has fingerprint
`22c7a1ac6bb6b84fef885568361a928e81c789db26ef101e11ccb02d2091eb5f`.
The packet remains non-authorizing. No live browser interaction or positive
operator-authentication acceptance flow was qualified by the component fixtures.

Managed Swarm restart completed healthy at 2026-09-08 20:37 EDT, serving API
port 16421 and UI port 21234. Lifecycle refreshed dependencies through their
owners: Agent Manager was rebuilt without restarting its shared process; the
earlier start refreshed Plan Manager through its managed lifecycle. No private
process start, host repair, paid inference or Audio development launch was used.

Status: retained decisions, accounting core and review panel implemented;
general mandate execution is not ready. Do not queue the Audio Tools pilot or equate the review preview with an
approval receipt. The operator requested review before any development-agent
launch.

| Gap | Owning repair and required proof |
| --- | --- |
| W0/W1 contract | Repaired through Business Health's generated PRD preview. OT-P0-002/004 distinguish plan and contract-development shapes; OT/SWM-P0-015–017 declare continuity, evidence-bound acceptance and review surfaces. Existing requirement evidence was preserved; new targets remain planned. |
| Admission and snapshots | Implemented in `internal/development`: resolve an existing idle item without a plan, retain exact approved bytes and verified human attribution, atomically compare-and-swap one aggregate. Next actions consult retained shape; explicit selection at backlog creation remains open. |
| Execution adapters | The declared `contract-development` transition and typed input/apply adapters are registered in `internal/transitionrunner`. Native harness-goal selection remains fail-closed and the governed fallback still needs live owner qualification. Do not rename `phased-plan-drain` or silently grant its slices broader authority. |
| Accounting and continuation | Token/wall reservations, known-usage settlement, checkpoint persistence, restart recovery, overrun recording and amendment preservation are implemented and tested at the domain boundary. Owner dispatch/usage adapters, monetary and delegated-child accounting, cancellation propagation and native/fallback continuation remain unqualified. Swarm cannot implement hard owner limits by putting them in a prompt. |
| Evidence-bound disposition | Acceptance predicate and negative controls are implemented. Production outcome resolvers and tested product/build/cohort identity are not configured. Audio's v1 setpoint reader deliberately has no acceptance-eligible outcome rows. Do not wrap its `status=ok` as a product receipt. |
| Operator surface | Development contract panel and goal drawer, typed API and CLI expose review/retained bytes, approval history, configurable goal, budgets and blockers. Retained-item board routing and amendment comparison are implemented; completion-submission/acceptance controls remain. New decisions require the configured Scenario Authenticator provider and `swarm-manager:write`; owner dispatch, provider qualification and the bounded Q3 grant remain unqualified. |
| Audio adoption | Review the [pilot decision sheet](../../../audio-tools/docs/internal/TESTING.md#pilot-decision-sheet-and-safe-first-slice). Numeric SLOs, corpus/device commitments and billing policy remain proposals. The initial local slice requires no paid inference. |

Acceptance qualification must demonstrate two successive authorized repairs
under one item, interruption recovery without renewed allowance, rejected target
weakening, stale/unknown evidence refusal, and honest budget exhaustion. The
fresh-agent proof is deliberately deferred until the operator reviews the goal
and authorizes that launch. Tech Tree Designer draft bundles are a later
enabler, not a prerequisite for read-only preparation.

Latest implementation validation checkpoint:

- Focused development/transition transport and work-shape guard tests pass with
  `-race`, including two successive repairs under one approval, repository
  reopen, concurrent decisions, unknown usage, cancellation, overrun and stale
  or mismatched receipt rejection. The final check used a fresh test-only
  `GOCACHE` after shared-cache failures; it passed all three selected packages.
  These use owner fixtures, not live agents.
- The complete CLI `go test -trimpath ./...` passes after adding the new command
  group to the expected surface and refreshing primitive evidence. Five review
  panel tests and two transition client tests pass; UI type-check and focused
  lint pass. Audio Tools' existing 15 evidence-board tests pass.
- Test Genie run `20260908-204608-e3b2ccba` returned FAIL during implementation.
  It reported API/CLI execution failures and existing unit-policy/architecture
  findings. The CLI group mismatch and an intermediate new-authentication
  scope mismatch were fixed and rechecked locally; the suite is not claimed
  green. The receipt also reports `canonical terminal persistence unavailable:
  run not found in index`; filed as `knw-1788901003359886533`.
- A later whole-API `go test -trimpath ./...` encountered missing Go cache
  archives and `TestSearchJSONMapsToValidDescriptor` resolving a relative
  `.vrooli/search.json`. It is not a passing whole-API certificate. The cache
  symptom also interrupted governed package refresh; recorded in
  `knw-1788900812734663930`. Do not add a private scenario host-repair script.
  The trimpath descriptor failure is recorded in `knw-1788901354074452382`.
- Package refresh stopped Swarm and failed its setup build. `make start`
  restored a healthy service; a final managed rebuild served the current API.
  Live negative controls returned not-found for the proposed Audio engagement
  and refused an anonymous development revocation before any write. These do
  not qualify a configured human authentication provider or a launched engagement.
- The real Audio proposal still resolves 16 artifacts and returns
  `budget_required`. Its refreshed live review fingerprint is
  `65db4974eecaad97e7eac200e323243b4b77cf4b223b61758f7c5d23ab9608a8`;
  the existing review packet and generated goal draft were refreshed, not approved.
  No Audio backlog item, mandate approval, harness goal,
  paid-provider call or development agent was created for this implementation.

Storage audit before this change reported 108 findings, including undeclared
durable paths and legacy direct file writers. This pass declares the existing
event database and WAL/SHM sidecars and adds domain-owned tables without changing
existing columns. It does not certify the other stores or perform a fleet
storage migration. Review `ARCHITECTURE.md` for the remaining Agent Manager
budget/continuation and evidence-owner work before removing launch blockers.

Earlier preparation checkpoint (retained for provenance):

- The real Audio proposal resolves 16 selected artifacts through the compiled
  review handler. The unset budget remains a finding; launch readiness stays
  false. The API/CLI preview does not create the proposed work item.
- Focused development-domain, transport and repository-pilot tests pass with
  `-race` and `-trimpath`. CLI preview and primitive-evidence tests pass; the
  generated CLI evidence was refreshed after the new command was added.
- Test Genie unit run `20260908-195739-7a220766` failed. Its result included API
  and CLI execution failures and unit-policy projection drift. This is not a
  green scenario certificate or a clean baseline comparison.
- A subsequent full API check isolated
  `TestEveryDeclaredWorkflowIsReachableFromATransition`: the separately declared
  `swarm-manager/readiness-review` workflow has no transition/child binding or
  recorded exception. Focused reproduction fails at `registry_test.go:288`.
  Reported through scenario-qa as `knw-1788898117660187174`; the declaration was
  preserved, not deleted or exempted to pass the gate.
- The live service was not restarted over this unresolved declaration. The new
  RPC is implemented and tested in source, but is not claimed to be served by
  the running Swarm process. No development agent or harness goal was started.

## Measures projection convergence

The operator-facing Stats projection is canonical: it retains the incremental
event-log replay, goal-scoped snapshot, trends, ETA, review, operating-mode,
record, and session analysis. The `measures` API/CLI remains a programmatic
contract, but its individual computations have not yet all been refactored to
delegate to that projection. Do not retire a Stats field or its UI until the
measure contract has a behaviorally equivalent, projection-backed adapter and
parity tests. Sandbox adoption remains correctly owned by
`agent-manager metrics-sandbox-adoption`.

## Remaining Connect migration

The 15-command `measures` group is fully Connect-backed and is the completed
rank-3 migration slice. The typed services that already preserve a command's
full behavior are likewise Connect-backed. The remaining local bindings are
intentional until their contracts cover the complete existing operation:
backlog (rank 1), execution and review (rank 2), sessions and captures (rank
4), scenarios and settings (rank 5), operations/agent-manager/portfolio (rank
6), and the greenfield records, proposals, search, prompts, and queue surfaces
(ranks 7–10). In particular, `backlog list`, `backlog get`, and
`backlog delete` are now typed, while current
`BacklogService.CreateItem` remains a cross-scenario triage contract rather
than a replacement for attachment-aware operator creation; the seam is
documented in `SEAMS.md`.

## Interop follow-ups

- Lifecycle state fields in `scenario.proto`, `backlog.proto`, and
  `execution.proto` remain strings with validation constraints. Migrating them
  to enums needs a compatibility and deprecation plan.
- The Agent Manager client intentionally emits lowerCamelCase JSON while local
  Swarm Manager contracts use canonical proto JSON names. Keep this behavior
  covered by a contract test.
- File-content endpoints intentionally use raw or streamed responses. Do not
  wrap them in proto envelopes without preserving streaming and attachment
  behavior.
- The ecosystem client still uses a hand-written JSON task shape because no
  behavior-equivalent Swarm Manager proto exists. Revisit this only when that
  proto contract is available.
- Inter-scenario clients currently fail fast after one attempt. Bounded retry
  and recovery tests remain a resilience opportunity, not a reason to change
  the current dependency contract.

## UX issues

- **Resolved 2026-07-22 — detail-view consistency:** Goal and backlog detail tabs now share `CompactTabBar`; cross-lens actions render after the tabs and only from Overview/Info. Goal files use the same full-width editable workspace as backlog files.
- **Resolved 2026-07-22 — nested surfaces:** Milestones render as individual persisted disclosures with flat dividers rather than a card surrounding a grid of cards.
- **Resolved 2026-07-22 — plan empty state:** The absent-plan state provides a direct `plan.author` launch action and takes the operator to Activity to follow the execution.
- **Resolved 2026-07-23 — mobile detail deep links:** Direct mobile links retained the persisted open desktop sidebar, which is a full-screen overlay below 768px and hid the detail pane. `AppShell` now collapses the sidebar on entering the mobile breakpoint.
- **Intentional integrity guard:** `goal.json` is viewable but protected from raw file mutations because it is canonical goal graph state. All goal artifacts, including migrated material, remain editable through the file workspace.

## E2E verification

- 2026-07-22: BAS execution `3b484aad-32c9-4622-942a-45ffb99126b7` completed the read-only Plan Workshop policy journey in
  [`bas/cases/plan-workshop/open-policy.json`](../../bas/cases/plan-workshop/open-policy.json). It opened the Graph workspace
  settings drawer, selected Plan Workshop, and verified the explicit-review policy states that there are no automatic rounds
  or readiness controls. Screenshot evidence is stored at
  `/home/matthalloran8/.vrooli/data/vrooli/browser-automation-studio/recordings/3b484aad-32c9-4622-942a-45ffb99126b7/artifacts/screenshots/00001--adhoc--open-graph-workspace--ACTION_TYPE_NAVIGATE.png`.
- 2026-07-23: BAS execution `46403ad3-a14e-42e9-8c36-bf6cd4ac2b3a` captured `/goals/ai-image-generation-foundation` at 390x844 after a 5-second settle. The goal detail is visible with the sidebar collapsed and no console output. Desktop confirmation is `c57632af-5608-4527-835f-69199e1d7428` at 1440x900.

## Historical notes

Prior operating-mode and agent-operation implementation notes were removed with
the runtime. Their persisted data is retained as read-only provenance, and
migration evidence is kept under
[operations/migration](../operations/migration/).

## Work ladder

### 2026-09-10 run-sheet clarity

- Rung: W3 (implementation)
- Evidence: The supplied mobile screenshot showed the backlog run sheet presenting
  strategy descriptions, a slice slider, and raw execution limits without enough
  plain-language hierarchy to explain what the operator was choosing or what the
  primary action would do. The contract and requirements were not changed.
- Outcome: The run sheet now makes readiness, execution approach, run scope, and
  approved guardrails distinct; grouped limits explain budget versus safety rails;
  the primary action says `Start run`; focused UI tests, type-check, and lint pass.
  The run scope now defines a slice in place, and Budget/Safety rails each have
  an accessible RCL Popover with plain-language details. Budget guidance
  distinguishes the approved spend ceiling (`max charge ÷ 1,000,000`) from the
  strategy estimate (`cost per turn × configured maximum turns`) and explains
  that actual spend comes from measured provider receipts.
  A first live mobile check then exposed a separate overlay defect: the new
  Popover content opened and had the correct geometry in the DOM, but its
  `Presence` wrapper formed an unpromoted stacking context, so the drawer
  painted over it. The governed RCL `Popover@1.2.9` successor now promotes
  that portalled presence layer to the menu layer; both info buttons are
  visible above the drawer and remain dismissible through the shared outside
  interaction behavior.
  The custom drawer implementation was reduced to a compatibility adapter over
  the governed RCL `ResponsiveDialog` adoption, so all existing drawer call sites
  now receive the shared desktop dialog/mobile sheet, safe-area footer padding,
  and grabber swipe dismissal. The three native range inputs were replaced with
  the governed RCL `Slider` adoption. Adoption obligations pass; preflight still
  reports the library's host/runtime viewport tokens and Slider's computed
  percentage token as unsatisfied static tokens, although `BaseStyles` and the
  components provide those values at runtime. The Popover draft validator was
  initially held by stale catalog projections and was later published as
  `Popover@1.2.9`; the consumer test and live mobile capture pass after the
  package rebuild and managed scenario restart.
- Measured: 2026-09-10

### 2026-09-09 contract-development pilot execution

- Rung: W3 (implementation)
- Evidence: W0 comparison found the execution target consistent with the
  governing development contract and Swarm OT-P0-002/004/015-017; `business-health
  validate scenario swarm-manager` and `vrooli scenario requirements validate
  swarm-manager` both pass; `go test -race ./internal/archtest` passes. The
  current owner-grant, transition, evidence, and board records still identify
  unqualified runtime behavior, including live owner qualification, provider
  reconciliation, production evidence resolvers, and complete work-shape
  routing.
- Blocker: The implementation qualification work in this plan remains open;
  no Audio pilot launch is authorized.
- Measured: 2026-09-09

### 2026-08-19 session recall and resolution plan

- Rung: W3 (implementation)
- Evidence: Goal `swarm-manager-quality-gates` requires first-class meta-orchestration sessions, and `OT-P1-004` requires conversational agent sessions; the proposed query-conditioned recall, job-scoped briefs, durable terminal resolutions, draft re-kinding, doctrine calibration, and token-contract adoption support that contract without contradicting another named Swarm Manager goal or the session architecture decisions. `business-health validate scenario swarm-manager` and `vrooli scenario requirements validate swarm-manager` both pass.
- Blocker: The implementation described by `swarm-manager-sessions-query-conditioned-recall-job-scoped` remains to be completed and validated.
- Measured: 2026-08-19

- Rung: W0 (goal/problem contract comparison)
- Evidence: Search found multiple swarm-manager goals, but none directly represents the user-supplied audio-tools reliability plan. Swarm-manager is in scope only for the thin shared-package adapter and focused validation evidence.
- Constraint: Broad pre-existing swarm-manager health findings remain separate from the audio capture contract.
- Measured: 2026-08-03.

### 2026-08-11 mobile composer follow-up

- Rung: W3 (implementation)
- Evidence: The mobile composer’s fixed expansion mode was nested inside the fixed footer’s backdrop-filter containing block, so the editor could be positioned below the safe area; the implementation now uses a bounded 2–6 row textarea with scrolling after the cap.
- Blocker: Resolved in the composer and auto-resize implementation; scenario-owned test-genie validation remains unavailable because its CLI build requests `go mod tidy` before running.
- Measured: 2026-08-11.

### 2026-08-12 mobile mic/composer follow-up

- Rung: W3 (implementation)
- Evidence: The supplied mobile screenshots showed `VoiceMicButton` rendering `Voice input unavailable (discovery_failed)` as a visible alert inside the composer. That expanded the mic flex item and starved the textarea; the host label is now removed, unavailable voice remains represented by the button’s accessible state, and the adopted React component-library version is refreshed from 4.0.0 to 4.1.0.
- Blocker: Resolved in the mic host, composer layering, and component-library adoption. Focused UI validation and the component contract suite pass; scenario-owned test-genie validation remains unavailable because its CLI build requests `go mod tidy` before running.
- Measured: 2026-08-12.
