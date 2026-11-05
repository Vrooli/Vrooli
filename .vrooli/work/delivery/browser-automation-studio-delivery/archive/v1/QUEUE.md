# BAS queue

Full history through 2026-10-06 (handoffs, censuses, candidate evidence, the
long Forecast) is in `archive/QUEUE-through-2026-10-06.md`.

## Forecast

- Destination: 25 Linux preservation journeys pass daily; current: 23/25
  available Linux journeys pass in the latest `bas-goal` preservation set
  (23/23). J09 and J11 remain on outside return conditions. The earlier 22/22 suite
  includes E27's J16/J19/J20, E28's J05/J06/J22,
  and E30's J10/J12/J14.
- Recent five accepted epochs, actual/estimate work units: E41 4/5, E42 5/4,
  E43 3/5, E44 4/4, E45 3/3 (19/21 combined, 0.90 actual/estimate). The goal
  gap remains two Linux journeys plus E22 intended-player review and macOS
  qualification; these test-support slices do not close those gaps. Runtime
  inventory is 250,616; the line-count phase remains closed under BAS-FB-064.
  E46's test-only fixture cleanup cannot affect runtime inventory; the worker's
  latest prescribed G5 result is 251,599 (+983). No runtime-reduction slice
  meeting the existing >=1,000-line estimate rule is currently evidenced.
- E30 grouped J10/J12/J14 into one session-readiness and profile-continuity
  slice. J09/J11 remain parked on outside conditions; J21 is in scope under
  BAS-FB-067, with physical desktop/Android qualification separately waiting on
  an authorized target.
- Linux: 23/23 on 2026-10-08 incl. J21 (E34 receipt `/home/matthalloran8/.vrooli/evidence/bas-goal/journeys-2026-10-08-e34-orchestrator-g2.json`). E30/E31 transient failures passed isolated and final reruns (details below). macOS arm64 occasional; x64 and Windows unavailable.

## Needs operator

- **E46 G5 disposition** — E46's admitted scope changes only
  `api/handlers/artifacts_test.go`, which the inventory classifies as test source;
  its prescribed G5 result is therefore 251,599 against 250,616 (+983), with
  zero runtime-line delta. The exact slice cannot pass G5 without unrelated
  runtime changes. Recommendation: keep E46 unaccepted and parked; return when
  the setpoint owner either supplies a new measured, in-scope runtime reduction
  slice meeting the >=1,000-line admission estimate or explicitly revises the
  G5 ceiling/metric against a fresh accepted baseline. No successor is admitted
  and no unrelated runtime deletion is authorized by this disposition.


- **Remaining Linux journey slots (J09, J11)** — either provide the authorized
  desktop/Android target and the shadow-safe AI gateway injection seam, or keep
  both slices parked until those external conditions arrive. Recommendation:
  keep them parked; neither has a safe Linux substitute. The available Linux
  preservation set now passes 23/23, including E34's J21 managed-session case.

## Next

- E46 — retained task `2546080f-7c56-4f85-94cb-640b552c9250`; child
  `37282362-007c-41d1-b499-3fc0b2ad6a91` is CANCELLED. The 43-line fixture-only
  change is present at source hash `e66202f7…c0e681`; fixture names are absent.
  Worker-reported G1–G4 pass; G5 remains 251,599 vs 250,616 (+983). Inventory
  source excludes `_test.go` from runtime counts, so this exact slice has no
  qualifying runtime reduction. Preserve the child evidence and keep E46
  unaccepted; no build/test rerun or successor dispatch under the reliability
  heavy-lane hold.
  (`epochs/E46.md`)

## Waiting on others

These never hold acceptance or other slices.

- **E46 G5 and completion** — return only when the setpoint owner supplies a
  new measured in-scope runtime reduction slice meeting the >=1,000-line
  admission estimate, or explicitly revises G5's ceiling/metric against a fresh
  accepted baseline. Then reconcile the same E46 task/source and require every
  original gate before acceptance. Reliability's heavy-lane hold remains in
  force for builds/tests until its owner releases it.

- **E22-R1** — intended-player review of capture
  `8ed37203-2ff7-49d7-9e1f-f953e1f0edc2`. Returns when QA bug
  `knw-1791087389154714359` (ad-hoc executions cannot open in review UI) is fixed
  or the capture gets a persisted workflow/project association.
- **E22-R2** — timed cursor samples and edit map in the movie spec. Returns on
  the generated movie-spec contract owner's decision.
- **DET-SIGNIN-R1** — real sign-ins with persistent profiles. Returns when the
  operator supplies dedicated, authorized test-only sites and accounts.
- **J11 shadow journey** — returns when the stub AI gateway can be injected
  without restarting the shadow.
- **J09 target journey** — returns when an authorized desktop or Android target
  is available for owner-controlled attach, action and detach qualification.
- **J21 native target qualification** — the Linux managed-session part is in
  E34; physical desktop/Android admission and session qualification return when
  an authorized desktop or Android target is available for owner-controlled
  released-session attach and validation-context checks.

## Censuses

- 2026-10-09T05:33:45Z | Remaining production Mock/Fake names are in established shared internal testutil packages; FakeMediaConfig is production behavior used by session management. No additional low-risk deletion candidate found in this sweep. | admissible=none
- 2026-10-09T05:34:00Z | MockStorageWithInfo is 36 implementation lines plus a 5-line banner in handlers/artifacts_test.go; repository-wide Go search finds only its declaration, constructor and methods, with no test callers. Delete the unused fixture and imports. | admissible=E46

## Qualification findings

- 2026-10-08 E30 acceptance G2 attempt: 21/22; J04 navigation request timed out at the API fetch. Shadow stayed healthy; isolated J04 passed 1/1 and a subsequent complete run passed 22/22. Retain as an intermittent qualification observation; acceptance uses the passing final run. Receipt: `/home/matthalloran8/.vrooli/evidence/bas-goal/journeys-2026-10-08-e30-acceptance.json`.
- 2026-10-08 E31 G3 first attempt: 21/22; J02 workflow execution returned HTTP 500 `context deadline exceeded`. Isolated J02 and the final complete run passed 22/22; post-suite driver session count was zero. Retain as an intermittent qualification observation; acceptance uses the final full run.
- 2026-10-09 E35 motion: three five-minute runs missed the ≥9,001 frame threshold (8,195/300,015, 8,786/300,004, 8,618/300,000). Fixture stayed near 30/s; frame-age/decode passed. Per TARGETS.md this soak target is nonblocking; amended 13-spec G2 passed and no later motion run started.

## Done

- E45 — removed MockScenarioCLI and its constructor from production; compiler tests
  use a local test-only fixture. Parent G1–G5 pass; runtime −56, test/support +25;
  3/3 work units. Proposal `gctp-35830f51b2ef` is fresh/open. (`epochs/E45.md`)

- E44 — moved MockAutomationRunner and MockRunCall into test-only compilation;
  parent G1–G5 pass, runtime −61/test +99, 4/4. Proposal
  `gctp-5e14f5bfdf5c` r2 is fresh/open. (`epochs/E44.md`)

- E43 — render/credits fakes test-only; −146 runtime/+139 test; 3/5; proposal
  `gctp-0d690f7fbd7e` (`epochs/E43.md`).
- E42 — workflow sync fake test-only; −170/+178; 5/4; proposal
  `gctp-ba27a956573b` (`epochs/E42.md`).
- E41 — sidecar cleanup and MockProcess test-only; −247/+144; 4/5; proposal
  `gctp-4cddb976d697` (`epochs/E41.md`).
- E40 — uxmetrics fake shared test support; −178/+178; 3/4; proposal
  `gctp-abb5b7b93450` (`epochs/E40.md`).
- E39 — recording repository fake shared test support; −376/+411; 4/4; proposal
  `gctp-09dda90082e7`; read-only Git call disclosed (`epochs/E39.md`).
- E38 — deleted unused AI/credits mocks; −345 runtime; 3/3; proposal
  `gctp-2a7edc531801` (`epochs/E38.md`).

- E37 — moved the 259-line filesystem fake to `_test.go` and relocated the
  174-line session-profile repository fake to `internal/testutil/sessionprofilemocks`;
  updated four test callers. G1/G2 tests and build, G3 package-file checks, and
  G4 production-reference review passed on orchestrator rerun. Runtime inventory
  fell 433 (252,628→252,195), test/support lines rose 441; 4/4 work units.
  Proposal `gctp-d3438138671b` is fresh/open for operator review.
  (`epochs/E37.md`)

- E36 — moved the unchanged 1,408-line handler mock-services source out of
  production compilation. G1 handler tests, G2 handler build and G4 package
  file-set classification passed on orchestrator rerun; G3 is unverified under
  A1 because the inventory skips the basename before and after. Runtime delta
  0; +1,408 test-only lines; 5/3 work units. Proposal `gctp-4e91c3d0e3db` is
  fresh/open for operator review. (`epochs/E36.md`)

- E35 — consolidated Chromium lifecycle ownership across 14 Playwright
  integration specs. G1 2/2, amended G2 13/13 (142 passed, 1 skipped), G3
  typecheck, G4 runtime 252,628 and G5 cohort 7,453/7,456 pass. Runtime delta 0;
  test/support delta −53 lines (7,506→7,453), 10/24 work units. The five-minute
  motion soak remains below threshold in three retained observations and is
  unverified per TARGETS.md; D3 excluded it from the blocking gate. Proposal
  `gctp-07020a97e979` is fresh for operator review; all 16 files are flagged
  `no_anchor`, with exact before/after source in `e35-prewrite` and
  `e35-final2`. (`epochs/E35.md`)

- E34 — Linux managed-session coverage for BAS-RH-J21's released target and
  validation-context admission boundaries. Independent G1–G4: J21 1/1, G2
  23/23, typecheck passed, runtime 252628 ≤ 252925. Actual 4/16 work units,
  +3 runtime lines and +175 test lines. Physical desktop/Android qualification
  remains parked; GCT proposal validation was unavailable under the workaround
  documented above. (`epochs/E34.md`)

- E33 — regenerated the BAS Electron shell from the canonical processed template
  and added a full output comparison regression. G1–G4 pass; G5 remains
  unverified under amendment A1. Runtime delta 0, test delta +187, 6/8 work
  units. Proposal `gctp-95f0216fa761` r2 folds E28–E32 and is open/fresh for
  operator review. (`epochs/E33.md`)
- E27 — BAS-RH J16/J19/J20 journey coverage. Aggregate suite verification is
  verified by the orchestrator's 2026-10-08 full 18/18 G2 run. (`epochs/E27.md`)
- E28 — BAS-RH J05/J06/J22 reliability journeys; G1 3/3, full G2 18/18, G3
  typecheck and G4 252,625 runtime lines all pass. Proposal `gctp-be5278ab7a08`
  is folded into `gctp-95f0216fa761`. (`epochs/E28.md`)
- E29 — BAS-RH J24 typed assertion/outcome journey; G1 1/1, full G2 19/19, G3
  typecheck and G4 252,625 runtime lines all pass. Proposal `gctp-ae9858ee9b33`
  is folded into `gctp-95f0216fa761`. (`epochs/E29.md`)
- E30 — J10/J12/J14 session readiness and profile continuity; G1 3/3, final G2
  22/22, G3 typecheck and G4 252,625 runtime lines all pass. Runtime delta 0,
  test delta +245, 14/24 work units. Proposal `gctp-01c94d3dcfd1` is folded
  into `gctp-95f0216fa761`.
  (`epochs/E30.md`)
- E31 — one cleanup owner for test-created resources in J02/J06/J12/J14; G1 2/2,
  focused G2 4/4, full G3 22/22, G4 typecheck and G5 252,625 runtime lines pass.
  Runtime delta 0; scoped test/support lines +101; 4/12 work units. Proposal
  `gctp-f87d89f02dc7` is folded into `gctp-95f0216fa761`. (`epochs/E31.md`)
- E32 — shared cleanup ownership across the remaining nine journey resource
  paths; G1 2/2, focused G2 9/9, full G3 22/22, G4 typecheck and G5 252,625
  runtime lines pass. Runtime delta 0; scoped lines −8; 2/17 work units.
  Proposal `gctp-bfd4bae0c676` is folded into `gctp-95f0216fa761`.
  (`epochs/E32.md`)
- E26 — RD-EXP: one ReplaySpec renderer for standalone HTML replay. (`epochs/E26.md`)
- E25 — RD-REC: one recording/execution workspace timeline controller. (`epochs/E25.md`)
- E24 — RD-REC first tranche: canonical `TimelineEntry` owns UI recording state. (`epochs/E24.md`)
- E23 — RD-WF first tranche: one generated-V2-backed UI workflow codec. (`epochs/E23.md`)
- E22 — REC-FIX: faithful recording and useful review playback. (`epochs/E22.md`)
- E21 — REC-1A: retire unreachable legacy recording UI surfaces. (`epochs/E21.md`)
- E20 — DIAG-REDACT: redact recovery credential from observability output. (`epochs/E20.md`)
- E19 — S13: one replay spec. (`epochs/E19.md`)
- E18 — S12: split the recording workspace controller. (`epochs/E18.md`)
- E17 — S6: one timeline shape. (`epochs/E17.md`)
- E16 — DET local detectability and interactive profile behavior. (`epochs/E16.md`)
- E15 — S10: one config owner. (`epochs/E15.md`)
- E12 — S7: one frame transport. (`epochs/E12.md`)
- E13 — S8: one AI engine. (`epochs/E13.md`)
- E14 — S9: one retention planner. (`epochs/E14.md`)
- E11 — S5: one recording journal. (`epochs/E11.md`)
- E10 — S1: one session broker. (`epochs/E10.md`)
- E9 — JX: journey expansion before session/recording rewrites. (`epochs/E9.md`)
- E8 — S4: schema-driven node editor. (`epochs/E8.md`)
- E6 — S3: remove the V1 workflow compatibility path. (`epochs/E6.md`)
- E7 — S2 remainder: remove handwritten action vocabulary owners. (`epochs/E7.md`)
- E5 — Driver, UI and CLI test consolidation. (`epochs/E5.md`)
- E4 — Further Go test maturation. (`epochs/E4.md`)
- E3 — Action vocabulary generated from proto (first tranche only). (`epochs/E3.md`)
- E2 — Go test consolidation. (`epochs/E2.md`)
- E1 — Dead code and diagnostics ownership. (`epochs/E1.md`)
- E0 — Journey suite and local fixture site. (`epochs/E0.md`)
- E01 — cleanup selectors, scheduler cancellation, AI navigation hook split (pre-epoch work; see the archived queue)

## Handoff

- Changed: reread current FEEDBACK.md. E46's source/evidence disposition remains unchanged: fixture-only scope has zero runtime-line effect; G1–G4 are worker-reported pass and G5 remains 251,599 vs 250,616 (+983), so E46 is unaccepted. The open `OWNER-OVERNIGHT-PAUSE-EXCEPT-HUB-20261021` entry is recorded; the current operator wake directs continuation only within the accepted goal/scope and adds no E46 gate change or product authorization. Signal sequence 2 was requested but could not be read: Agent Manager CLI refresh was refused by the SB20 host-memory admission guard before command execution.
- Verified: `run identity --json` returned this retained run `9871d148-a9b0-418f-98e2-dac5b43bc24a` as `running`; it does not establish a current await handle or park state. `agent-manager run get`, `run await-result`, and `effort signals` all returned the same admission refusal. Existing E46 task `2546080f-7c56-4f85-94cb-640b552c9250`, cancelled child `37282362-007c-41d1-b499-3fc0b2ad6a91`, G5 evidence, and prior qualification limits remain as last recorded; no new owner state or gate was inferred.
- Remaining: signal seq 2 content, current run await condition, and native park receipt are unverified. Source/evidence still shows no in-scope way for this exact E46 slice to meet G5 without unrelated runtime deletion. Preserve J09/J11, E22 review, intended-player review, and platform return conditions.
- Unverified: signal sequence 2, current await handle/status beyond identity's `running` result, current task/run status, parent-side rerun of G1–G5, full unified diff/active-test comparison, and fresh accepted inventory baseline. No builds, tests or lifecycle operations were run.
- Exact next action: after host PSI falls below the 10.00% admission ceiling and Agent Manager CLI refresh is admitted, read signals after seq 1, inspect this same run and its await handle, then park this same run through `agent-manager effort park` if the recorded external wait is still unresolved. Keep E46 unaccepted and do not dispatch a replacement. Reopen only when the setpoint owner supplies a newly measured in-scope runtime slice meeting the >=1,000-line admission estimate, or explicitly revises G5's ceiling/metric against a fresh accepted baseline; after reliability releases the heavy lane, reconcile the same task/source and require every original gate. Tool refusal and fallback are in WORKAROUNDS.md. Open feedback remains as listed in FEEDBACK.md, including `OWNER-OVERNIGHT-PAUSE-EXCEPT-HUB-20261021`.
