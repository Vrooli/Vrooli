# E7 — NH13.4/.6 test integration and current-build proof

- Outcome: complete the still-open E6 acceptance evidence for the four selected NH13.4/NH13.6 phone states after the E6 UI implementation.
- Kind: fix.
- Status: returned COMPLETE, E7 remains unaccepted; task `81284216-f343-4857-8726-1903eb5e628d`, run `7b278054-81fd-49a6-bdc0-b37724d401e7` ended 2026-10-20T23:18:46Z. G1 passed; G2/G3 unverified; parent G4 review passed for the bounded test/evidence delta only. E7 does not accept E6 or E4.
- Estimate: 4–6 meaningful work units; report actual units.
- Parent run: `11175250-c50d-44e9-9af5-cd6eba79bab2`.
- Admission authority: active `effort:notification-hub-redesign`, revision `sha256:ee51f1db548652434a8bba3c5a301439abf2c316fef04e1652191e46b9332dd0` (recheck immediately before admission). Profile owner already resolved `prompt-manager/delivery-epoch-worker`, profile `4c19aef1-36a2-4cc5-8961-42648692883b`, `gpt-6-luna`, medium, interactive.
- Prior E6 task/run: `e801c005-06ba-4454-9ba6-6cafe73bdce5` / `39226203-f710-4046-9cce-83f22fa35d43`, terminal COMPLETE but E6 gate status UNMET. Do not retry its Test Genie commands unchanged.
- Complete E6 requirements and selected references remain in `epochs/E6-nh13-empty-and-read-failure.md`; selected original image hashes and NH13.4/.6 semantics there govern. Read that epoch and this continuation, the selected cases, applicable supervisor feedback IDs HUB-SUP-AUDIT-NH13-VISUAL-IMPLEMENTATION-20261020 and HUB-SUP-AUDIT-SKIP-FOCUS-CAUSAL-PROBE-20261020, the worker card, and only the necessary tail of the E6 slice log. Do not read FEEDBACK.md.

## Returned evidence and reason for this successor

E6 implemented bounded empty/read-error UI, locale parity, 200 cap, read-only retry, and stale-row retention. Focused Asks tests (31), locale checks (5), type-check, strings check and lint passed. Test Genie unit result `20261020-222708-c3817c57` was 181/183: the two failures are stale `AppShell.a11y.test.tsx` expectations for `No decisions are waiting for you`; its 85% coverage floors passed (lines/statements 90.15%, functions 85.39%, branches 87.53%). Fix those two affected expectations so shell accessibility/navigation assertions reach their intended checks.

The E6 workflow run `notification-hub/20261020-221520-61329d0e` failed on four selected-state assertions using superseded copy and two unregistered `[role="alert"]` selectors. Its archived BAS screenshots were from pre-final build `sha256:951f47a3011fef3d636bc2f96d24b6d90dc06247607a0747468a7abf443cb2bd`, while current managed UI build at handoff was `sha256:b714e695f87451e3d8dad5c3b50a2015ed028c5772b9e431502b329dd6f2dcbb`. The four old dark/light files are pairwise identical across theme. They are diagnostic evidence only. G3 has no completed served retry/Settings trace. Preserve all exact results and do not claim acceptance.

## Scope and baseline

Parent anchor: `gcta-8c74017e19c8`, HEAD `52097258290d1bff0191319242d4948e214af9e3`, scope fingerprint `scenarios/notification-hub/**`. This is an existing dirty workspace; the anchor records existing authored files, not E7 authorship. Before any write, re-read the exact files and current managed build identity. Stop if an affected file differs from the recorded anchor/current source in a way that makes the evidence obsolete; report exact paths and hashes.

Allowed authored paths:
- `scenarios/notification-hub/ui/src/layout/AppShell.a11y.test.tsx`
- `scenarios/notification-hub/bas/cases/e4/asks-empty-dark.json`
- `scenarios/notification-hub/bas/cases/e4/asks-empty-light.json`
- `scenarios/notification-hub/bas/cases/e4/asks-read-failure-dark.json`
- `scenarios/notification-hub/bas/cases/e4/asks-read-failure-light.json`
- evidence under `scenarios/notification-hub/docs/internal/goal/evidence/E7-nh13-proof/`

Original file hashes at admission preparation:
- `AppShell.a11y.test.tsx`: `1c24e3140d86f88f62c27bdf57b2aee329b61a3707060de1b2dbd4b575e0e3ab`
- `asks-empty-dark.json`: `0366efa7069254ab95095525feb0d91ba5802c0b9bd2444fc68724c588c4729a`
- `asks-empty-light.json`: `144b6bfcbe494cb88d3ce562e9df2b340a981ca4b31d4eff19ae7839066960fb`
- `asks-read-failure-dark.json`: `1c625ddeaa27fdb2cf14cbaef2a2c44aeeaa4d8ce5b499dedd8230fe6f0e0f69`
- `asks-read-failure-light.json`: `b07b1d9639116ed038bde1f01aacbd4ad0aacccd0315d4fc6fa09d10790f75a3`

No Notification Hub product source, API, storage/auth/identity, shared package, schema, generated artifact, selector registry, dependency, or live-data changes are in scope. If achieving a safe controlled journey requires another path or a missing fixture capability, stop at that boundary and return the precise owner/path/evidence; do not widen the assignment.

## Required work

1. Update the two stale shell-test expectations to assert the current authoritative empty-state wording or stable semantic selector. Preserve their original axe and navigation assertions. Do not weaken, skip, or delete either test.
2. Repair the four existing E4 BAS cases in place: use current selected copy and registered selectors only; ensure each light/dark case actually applies its named theme and fresh selected-state result. Preserve the existing routed-isolation labels, disposable fixtures, route, viewport, and ownership. Do not add a parallel case owner or change unrelated E4 cases.
3. Extend the existing NH13.6 failure case(s) only if supported by current BAS actions and the existing routed failure fixture, to exercise Retry then Settings after selected-state capture. Retain a trace proving the retry made only another `ListAsks` read and caused no answer/clear/delete/primary write. If the native workflow cannot expose trustworthy client-call proof inside these paths, record the exact limitation and leave G3 unmet.
4. Run the affected focused AppShell test and the unchanged-floor Test Genie `unit` and `workflow` phases using the scenario's normal owned routes. For each Test Genie run, follow `docs/TESTING.md` wait-once protocol and retain actual phase output/artifacts. No unchanged retries after a failure.
5. Use the managed scenario lifecycle route to confirm the served build before captures. Capture all four exact NH13.4/NH13.6 light/dark states at 390×844 CSS px (DPR 2), retain original/capture hashes, theme check and build identity. Do not infer theme or current-build proof from historical E6 captures.
6. Return code/test delta with explicit consolidation/deletion review, preservation of stale rows and ordinary History/filter access, accessibility/localization impact, exact validation commands and outputs, current build identity, route/client-call evidence and remaining blockers. Do not accept E6 or E4.

## Exit gates

- G1 test: shell a11y tests pass; current `unit` phase passes all tests and preserves configured 85% floors.
- G2 custom: Reason: exact visual fidelity must be judged from matched current-build captures against the selected originals, not from code assertions alone. Four retained captures come from one identified current managed build, named light/dark themes differ as expected, exact viewport matches, and selected hierarchy/content matches the four originals with documented adaptations only.
- G3 journey: controlled served NH13.6 failure → retry → Settings path records the actual client calls and proves no primary write/answer/clear/delete. Otherwise state the specific blocker and leave this gate unverified.
- G4 review: parent separately reviews privacy, boundary, accessibility, test protection, source attribution and maintainability; this review is not delegated to the worker.

## Worker card and handoff

Read only `scenarios/prompt-manager/store/skills/packs/core/large-effort-orchestration/references/worker.md`. Exact parent run: `11175250-c50d-44e9-9af5-cd6eba79bab2`. Task `81284216-f343-4857-8726-1903eb5e628d`, run `7b278054-81fd-49a6-bdc0-b37724d401e7`. Preserve the existing E6 implementation and unrelated dirty work. No code-quality gate or deadline may be waived. E7 can supply evidence toward E6; all ten BENCHMARK gates remain open until native independent review and whole-scope acceptance.


## Parent reconciliation — 2026-10-20T23:22Z

`agent-manager run get` confirms E7 terminal COMPLETE; selected terminal output is a truthful blocked handoff, not acceptance. Independently reviewed the exact assigned delta: the AppShell test updates only replace stale copy expectations, keep the axe and navigation checks, and use an isolated notifications mock; the four E4 BAS cases retain routed-isolation/confirmation labels and disposable fixtures, explicitly choose their named theme, check 390×844 DPR 2, use registered selectors, and capture only after assertion. No product source, API, storage, identity, generated, or shared package changed; no tests or controls were deleted or weakened. No consolidation/deletion was needed. This review is G4 for E7 only, not E6/E4 or whole-product acceptance.

The current evidence supports G1 pass: focused AppShell 2/2 and Test Genie unit `20261020-223746-2fa4746f` PASS (one phase, zero errors; 46 unit-health warning findings retained, with no coverage percentage claim). G2 is unverified: no four captures completed on a current managed build. Test Genie workflows `20261020-225349-d721f391` and `20261020-230600-6218419e` terminally failed; both durable providers are RECOVERY_FAILED. Selected provider `ecf3f054-f6bf-404b-9f77-9b95e4ebc3aa` selected four cases, executed zero and refused four; install error records historical active lease `9a666086-3f9c-4565-a649-1ebb0c69257c` and mismatched cleanup request. This does not establish a parent-run requirement or current lease. G3 is unverified: no served retry→Settings client-call trace. Its current list Retry has no registered selector; Workflow Health’s fixed `validation` BAS artifact profile disables network events, and case JSON cannot enable them. Do not infer no answer/clear/delete/primary write. Seq28 was sent to the child through normal same-child continuation (transport accepted, consumption unknown); it returned with exact safe handoff and did not launch another check. Search-Hub and fallback Prompt Manager discovery found no current-lease read route; no manual install or clear was attempted.

## Slice log

2026-10-20T23:22:00Z | Parent returned-run reconciliation: exact task/run COMPLETE 23:18:46Z; evidence/status in `evidence/E7-nh13-proof/status.md`, unit manifest/log, workflow manifests/results, and selected provider manifest. Parent G4 review passes for the E7-only test/evidence delta; no whole-product gate closes. | gate G1=pass focused AppShell 2/2 and Test Genie unit `20261020-223746-2fa4746f` PASS, 0 errors; unit-health warnings retained, no percentage claim | gate G2=unverified no current-build four-state captures; workflow runs terminal FAIL before usable capture evidence | gate G3=unverified no completed served retry/Settings call trace; selector/artifact-profile limitations documented | gate G4=pass parent independent privacy, boundary, accessibility, test-protection, attribution and maintainability review of E7 delta; does not accept E6/E4 | exit metric=E7 partial: G1/G4 pass, G2/G3 unverified, E6/E4 not accepted | actual units=6 | ack=HUB-SUP-AUDIT-COHERENT-SCOPE-SELF-VALIDATION-20261020,HUB-SUP-AUDIT-FOCUSED-WORKFLOW-DEBUG-20261020,HUB-SUP-AUDIT-E7-LEASE-PRODUCER-CORRECTION-20261020
