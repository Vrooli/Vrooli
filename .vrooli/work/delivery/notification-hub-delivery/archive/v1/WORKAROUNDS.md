# Shared infrastructure findings

## E16 GCT admission anchor — host pressure

- Attempted at 2026-10-21T04:36Z: `git-control-tower proposal anchor --effort effort:notification-hub-redesign --epoch E16 --scope 'scenarios/notification-hub/**' --json`.
- Exact result: `Runtime error: CLI build admission deny (host memory PSI some.avg10 13.36% is at or above setpoint substrate/SB20 maximum 10.00%); retry when host capacity is available`; GCT CLI refresh was not admitted/completed, exit status 1. No anchor receipt was produced and no E16 worker was admitted.
- This is a shared host/tool admission refusal, not a product failure or operator decision. Do not bypass lifecycle/build admission or repeat the unchanged command. Return condition: changed host-capacity evidence or owner signal showing GCT CLI refresh can be admitted; then run the normal anchor and proceed with E16 only if it succeeds.

## HUB-RUNTIME-20261020 — managed test lifecycle denied by host pressure

- Failed owner operation: Test Genie server-owned run `20261020-135108-e4402898`, requested phases `ui-health,api,unit`, attempted to start the managed `notification-hub` scenario.
- Evidence: zero phases started. Lifecycle had to prepare stale dependency artifacts (`tunnel-manager` API `go.mod`; shared proto-types UI generation; scenario-authenticator rebuild-only). Notification Hub API build was denied because host memory PSI `some.avg10 11.89%` was at/above control-plane setpoint `substrate/SB20` maximum `10.00%`; exact message: `retry when capacity is available`. Retained lifecycle log `/home/matthalloran8/.vrooli/state/lifecycle-runs/notification-hub/20261020-135227.360904381-600353.stage.setup.log`.
- Consumer effect: current-tree Test Genie validation did not run; the managed reviewed build and operated journey remain unverified. No Notification Hub product source change from lifecycle startup is claimed; the governed Test Genie Proto preflight did alter generated output, as listed below. The existing service/dependency state after failed preparation was not independently rechecked.
- Updated next action (supervisor feedback seq 3, acknowledged 2026-10-20): do not require a separate PSI/admission observation from `vrooli status`; the lifecycle control plane reads fresh Linux PSI and acquires the authoritative SB20 lease. Changed supervisor diagnostics (`some.avg10` and `full.avg10` 0.06% at 14:35Z) support one normal governed retry of the previously unstarted `ui-health,api,unit` phases, which has been launched. Let lifecycle admission grant or deny; retain its exact outcome. No force, bar weakening, blind repeat, or certification claim.
- Return condition: this one run completes or records the new exact lifecycle refusal; do not rerun without changed evidence or a native retry recommendation. No time-based reset was provided.

The Agent Manager child subsequently ended after `thread/resume` reported `no rollout found`; its report also has 127 rate-limit observations and no reliable final output. Same-child continuation is being reconciled. This runner failure does not establish a product gate or host recovery.

## Test Genie shared Proto preflight side effect

Before any phase ran, Test Genie detected stale shared proto-types and ran its governed generation preflight (`go run -mod=mod ./cmd/protogen generate --changed --scenario notification-hub`). The pre-attempt workspace was already broadly dirty, including generated Proto state; E1 did not capture these out-of-scope paths, so this post-run inventory cannot attribute each individual difference to the preflight or to another writer. No generated output was hand-edited or restored. The exact current `git status --short packages/proto/gen` inventory is:

```text
packages/proto/gen/.vrooli-proto-artifact.json
packages/proto/gen/descriptor/image.binpb
packages/proto/gen/go/agent-manager/v1/api/service.pb.go
packages/proto/gen/go/agent-manager/v1/domain/events.pb.go
packages/proto/gen/go/agent-manager/v1/domain/run.pb.go
packages/proto/gen/go/dev-routing/v1/routing/routing.pb.go
packages/proto/gen/go/dev-routing/v1/routing/routing_v1connect/routing.connect.go
packages/proto/gen/go/nutrition-planner/v1/planning/planning.pb.go
packages/proto/gen/go/react-component-library/v1/versions/lifecycle.pb.go
packages/proto/gen/go/react-component-library/v1/versions/versions_v1connect/lifecycle.connect.go
packages/proto/gen/manifests/agent-manager.lock.json
packages/proto/gen/manifests/cli.lock.json
packages/proto/gen/manifests/dev-routing.lock.json
packages/proto/gen/manifests/nutrition-planner.lock.json
packages/proto/gen/manifests/react-component-library.lock.json
packages/proto/gen/python/agent_manager/v1/api/service_pb2.py
packages/proto/gen/python/agent_manager/v1/api/service_pb2.pyi
packages/proto/gen/python/agent_manager/v1/domain/events_pb2.py
packages/proto/gen/python/agent_manager/v1/domain/run_pb2.py
packages/proto/gen/python/dev_routing/v1/routing/routing_pb2.py
packages/proto/gen/python/dev_routing/v1/routing/routing_pb2.pyi
packages/proto/gen/python/nutrition_planner/v1/planning/planning_pb2.py
packages/proto/gen/python/nutrition_planner/v1/planning/planning_pb2.pyi
packages/proto/gen/python/react_component_library/v1/versions/lifecycle_pb2.py
packages/proto/gen/python/react_component_library/v1/versions/lifecycle_pb2.pyi
packages/proto/gen/typescript/agent-manager/v1/api/service_pb.ts
packages/proto/gen/typescript/agent-manager/v1/domain/events_pb.ts
packages/proto/gen/typescript/agent-manager/v1/domain/run_pb.ts
packages/proto/gen/typescript/dev-routing/v1/routing/routing_pb.ts
packages/proto/gen/typescript/nutrition-planner/v1/planning/planning_pb.ts
packages/proto/gen/typescript/react-component-library/v1/versions/lifecycle_pb.ts
packages/proto/gen/go/cli/v1/renderer_profile.pb.go
packages/proto/gen/go/device-studio/
packages/proto/gen/manifests/device-studio.lock.json
packages/proto/gen/python/cli/v1/renderer_profile_pb2.py
packages/proto/gen/python/cli/v1/renderer_profile_pb2.pyi
packages/proto/gen/python/device_studio/
packages/proto/gen/typescript/cli/v1/renderer_profile_pb.ts
packages/proto/gen/typescript/device-studio/
```

This is a concurrent/out-of-scope unknown-attribution report, not part of the E1 allowed write set or source line totals. Leave this state for the Proto owner to verify through its native owner command if needed.

## E1 managed journey/visual fixture hold

- Current app state: `vrooli scenario status notification-hub --json` reports the managed process healthy on build `sha256:52a6bdfab4d482f06f4d4f79eeaf9a8a39524dea94d5ea68e5d82e5097ea006a`, after restart operation `startop-c254fd95acfc30e0179dcd3638e19f65` at `2026-10-20T14:06:37Z` (unknown attribution). After supervisor feedback removed the unsupported extra-PSI prerequisite, one normal governed retry `20261020-144357-5f2c90e0` was admitted and completed. Its detailed phase results are below; no unchanged retry was made.
- Safe fixture inventory: `scenarios/notification-hub/bas/README.md`, `bas/registry.json`, and `bas/cases/**` contain only observer smoke cases (`dashboard`, `settings`); there are no `bas/seeds/`, fixture actions, or managed test-recipient fixture. The only proven isolated ask fixture remains the in-memory `databasetest.NewSQLite(t)` handler/service test. Using the current managed app's ordinary read would retrieve configured-recipient asks and may expose personal data; creating a real ask can send a real notification. Neither is authorized for this journey.
- Next dependency owners: Notification Hub product owner for a disposable managed recipient and authorized fixture configuration; shared storage owner (`storage-manager` / routed test-storage) for a verified leased database and test-mode-header propagation through the actual client. No named engineering run/task or verified fixture currently exists. The earlier request for a separate host PSI observation is withdrawn: managed lifecycle owns fresh admission and decides it at action time.
- Return condition: identify the supported fixture owner/route and verify isolation without personal-data reads, notification delivery, or answer mutation. Once fixture exists, operate the managed route and capture desktop/phone dark/light on the exact served build against selected originals. Let a normal governed lifecycle request obtain its admission decision; no separate PSI precheck is required. Until fixture return passes, in-memory and component tests remain the available behavior evidence; no synthetic answer is operator approval.

## HUB-SUP-AUDIT-E1-CAPACITY-FIXTURE-20261020 — admitted retry and routed fixture boundary

- Test Genie run `20261020-144357-5f2c90e0` ran phases `ui-health,api,unit` from `2026-10-20T14:43:57Z` to `14:44:27Z`; lifecycle admission succeeded. API passed. UI-health failed with 5 errors, 33 warnings and 9 infos. Findings include `standard_shell_ownership`, four `visual_offscreen_interactive` errors, `visual_native_screen_unverified` on browser-viewport captures, and `visual_safe_area_not_exercised` (0 CSS px safe-area insets). The shell finding reflects a real source-graph gap: `main.tsx` dynamically imported `App`, whose `AppRouter` mounts `AppShell`; the entrypoint has since been corrected to import `App` statically. This source correction is not a rebuilt/served UI-health pass. The four offscreen rectangles share bounds `(8,-56 117x32)`; those bounds are consistent with the component library's focus-only skip link (`translateY(-200%)`, revealed on `:focus-visible`). The artifact reports only the generic selector `a`, so leave these findings open and preserve the skip-link behavior. Native-screen and safe-area items are INFO capture limitations and do not prove selected visual fidelity.
- Unit failed with `TEST_EXECUTION_FAILURE`: UI workspace `pnpm run test:coverage` exited 1 after 28 test files / 150 tests passed. Global coverage was functions 78.69% and branches 83.53%, below both configured 85% thresholds (lines/statements were 86.70%). Detailed unit phase artifact: `artifact_86d5182c43e09689e804270019c7c155`; run findings artifact: `artifact_55a3466ec91cf70780c0a94c3d2eb658`; command log: `artifact_7f8de3633909a69a9a22671ee39f8461`. The producer receipt was inspected once; do not rerun unchanged.
- Fixture-route evidence: `test-genie eligibility check notification-hub --json` returns `{"routed":true}`. `storage-manager validate scenario notification-hub --json` returns `passed` with routed-seam/isolation analyzers, but reports two `SQL_DB_HANDLE_CAPTURE` findings at `api/cmd/schema-preflight/main.go:44` and `api/internal/conversations/identity_preflight.go:17`, plus duplicate retention-authority warnings. Neither result proves an active leased DB or a disposable recipient. `api/main.go` wires `database.RoutedDB` and `apihttp.TestModeMiddleware`; with no installed lease, RoutedDB uses primary, so the header alone is not isolation evidence.
- The Notification Hub Connect client now adds `X-Vrooli-Test-Mode: 1` only for explicit `?vrooli_test_mode=1` and preserves that opt-in across shell, queue, History and ask navigation. A focused test captures the real `identityClient` request with and without the opt-in. This verifies client header propagation logic only; no active lease was installed and no live ask, notification send, answer, or fixture write was attempted.
- Owner/return: product owner is the Notification Hub delivery orchestrator run `503e3320-4190-4335-85e2-f28318a620dd`. Attempt 1 task/run `3c22bdb9-3b52-494e-bbfb-b27f10666dc7` / `99e7be42-82ef-4b6a-9e41-e0cceca829e0` is COMPLETE; successor task/run `79e72930-ff9d-435d-b9d6-8ad0410cfe4d` / `50848796-3690-45dc-b154-c3b593fe4e88` is RUNNING. Shared leased-workflow/testpool and isolation counters are owned by existing workflow-health/Test Genie mechanisms; sequence-5 supervisor direction requires adopting those mechanisms and Hub’s schema initializer, rather than waiting for a product-policy decision. The prior supported `agent-manager run recover` returned `403: run identity cannot perform lifecycle operation`; no alternate task/run was created. Return after the product owner identifies an isolated recipient/configuration and storage owner verifies an active leased DB plus actual client header routing isolated from primary.


## HUB-SUP-AUDIT-E1-REACTIVATION-FINDINGS-20261020 — seq 4 follow-through

- Acknowledged in the E1 slice log. Rechecked exact child/task `99e7be42-82ef-4b6a-9e41-e0cceca829e0` / `3c22bdb9-3b52-494e-bbfb-b27f10666dc7`; `agent-manager run get` reports RUNNING, with the prior failed result retained from the 13:52 invocation. The local `agent-manager run identity --json` call returned `identity_refresh_unavailable`; the same-child `run continue ... --idempotency-key notification-hub-e1-seq4-findings-steer-20261020` attempt returned `401: verified agent caller required`. This exact caller refusal applies to this invocation; no RecoverRun, task/run creation, successor or alternate route was attempted.
- The unit producer's detailed error remains coverage-only: 28 files / 150 tests passed, then functions 78.69% and branches 83.53% failed the configured 85% floors. Added recipient answer readback, NotFound non-disclosure, expired/default timing and post-expiry status regressions in the changed E1 boundary. Focused nine-file UI command passes 45 tests. A filtered AskPage coverage run passes 16 tests but still reports 93.41% statements/lines, 82.82% branches and 90% functions; it exits nonzero because that module does not meet the unchanged 85% branch floor. Do not treat this filtered sample as the whole suite or lower the floor.
- The browser entry now imports `App` statically so the checked import graph reaches `AppRouter` and `AppShell`. `type-check`, strings/selector freshness, and focused R24 API handler/service suites pass. Managed UI-health and the complete Test Genie unit phase remain unverified after this source/test change. No unchanged managed build was retried.
- Fixture return is still open under the existing product owner run `503e3320-4190-4335-85e2-f28318a620dd`, task/run `3c22bdb9-3b52-494e-bbfb-b27f10666dc7` / `99e7be42-82ef-4b6a-9e41-e0cceca829e0`; routed storage is owned by Test Genie/storage-manager, with no named engineering task or identified disposable recipient/configuration. `docs/agent-system/routed-test-db.md` requires an active verified lease and header propagation. Existing `routed: true`, analyzer pass with handle-capture findings, and the actual-client header test do not prove active lease or primary-store non-effect. No live ask, notification send, answer or managed fixture write was used.


## HUB-SUP-AUDIT-E1-TERMINAL-FIXTURE-20261020 — seq 5 follow-through

- Acknowledged by the orchestrator on the sequence-5 wake. Canonical signal log reports latestSeq 5; this feedback does not establish product acceptance. Existing verified successor `50848796-3690-45dc-b154-c3b593fe4e88` remains the owner. A bounded same-session `run continue` delivered the fixture direction; transport accepted and provider consumption is unknown.
- Reconciled route: existing workflow-health execution/service and isolation/counter paths own routed mutating workflow leases. Supervisor source evidence says Hub applies `modules.AllSchemas()` to Primary but has not installed the existing schema initializer for a fresh routed testpool; the BAS registry has only observer cases and no disposable recipient fixture. Successor is directed to inspect current APIs, wire/adopt the existing hook within `scenarios/notification-hub/**`, create a deterministic disposable recipient and narrowly labeled mutating journey, and prove active lease identity, real client header routing, and zero primary requests/writes before mutation.
- No manual devrouting, new lease framework, personal recipient reads, real notification, fake answer, or admission bypass. Once the fixture is isolated, run one governed Notification Hub workflow phase and retain testpool/primary counters and exact receipt. UI-health/coverage failures, exact-build visual proof, and independent review remain open.
- Return if the existing seam refuses: retain the precise refusal, API/owner, and unmet condition. This work remains inside the approved E1 goal and does not depend on new operator policy.

## HUB-SUP-AUDIT-UIHEALTH-ISOLATION-RETURN-20261020 — seq 6 follow-through

- Acknowledged by the orchestrator after canonical FEEDBACK and sequence 6 were reread. Active successor task/run remains 79e72930-ff9d-435d-b9d6-8ad0410cfe4d / 50848796-3690-45dc-b154-c3b593fe4e88. A same-session directive was delivered about the new validator and unisolated workflow result; transport accepted, consumption unknown. A first message had shell interpolation of inline backticks; a corrective exact-value message was sent under a new idempotency key.
- Successor Test Genie run 20261020-154237-0247c7b3 passed API and unit (global floors met), but UI-health failed with four offscreen-anchor errors. Affected workflow execution a132aea9-015f-4297-acb9-b618e42f471f records isolation.installed=false, empty lease and TestPoolRequests=0. Empty primary counters do not establish isolation; the temporary workflow case was removed. No unisolated browser journey should be repeated.
- Shared UIHealth resolver/focus provider changed and was adopted on normal restart 15:45:55Z, healthy build sha256:6c149d7fc4bd869282c3b36926c2bdac389b89c8434ad551e7c8a783f3e9272b; selected source hashes and package tests passed. One relevant current-tree UI-health recheck can use this changed validator. Require its per-document same WeakMap node, actual Tab/focus-visible path and full unclipped viewport evidence; missing/mismatched/reloaded/clipped evidence stays ERROR. This is not visual acceptance and does not establish whether the earlier anchor was a skip link.
- Route the fail-closed lease/route issue through existing workflow-health owner. Required return before another browser workflow: nonempty verified lease identity, TestPoolRequests greater than zero, PrimaryDuringTestModeRequests equal to zero, declared route and required file-root counters. Preserve no personal reads, real delivery, fake human answer, manual devrouting, new lease framework or bypass. Selected original visual comparison and independent E1 review remain open.

## HUB-SUP-AUDIT-READONLY-UIHEALTH-CLARIFICATION-20261020 — governed read-only check timed out

- Trigger: seq 8 clarified that R24 permits the ordinary read-only UI-health layout/keyboard probe against the changed serving validator. One Test Genie run was submitted after confirming no in-progress Notification Hub run: `20261020-160055-10006e04`, command `vrooli scenario test notification-hub --phases ui-health --json`.
- Result: FAIL before any phase (`phaseSummary` total/passed/failed/skipped all zero); managed lifecycle start timed out after 2m with `context deadline exceeded`. This is not a UI-health result and does not prove a product defect or pass. No unchanged retry.
- Shared preflight did perform governed Proto generation: artifact `1a1558127cff621693b77d09fc4ab3725ae620b31d3eeaa4f9c19ca0a00105a`, source digest `sha256:c6e26c164a0235ab5b8c4e10af05355d09777794059383239a4659adf504b1c0`, published scope `device-studio`. This is outside the Hub allowed set; shared workspace attribution is unknown. Leave it for the Proto owner; do not restore or hand-edit generated outputs.
- Fallback/return: continue the independently admitted E2 slice. Retry the one affected read-only UI-health phase only after changed lifecycle availability/capacity evidence, using the normal Test Genie owner. Preserve R24 and avoid any mutation or repeat of the earlier unisolated custom workflow. E1 operated fixture, selected visual evidence and independent acceptance remain open.

## Current routed-fixture return after seq 8

- Parent reread canonical feedback and the effort signal log through seq 8. Seq 3's separate PSI prerequisite was removed and its one normal governed retry is already recorded above; seq 3/5's isolated-fixture obligation remains open.
- Current bounded Hub owner is E3 task `a0b3c1cc-2881-42bc-b7b6-71adb9d6b5e1` / run `b8132adc-54f4-4015-8c66-cc9e3d332a2d`, admitted under verified active effort reference and queued at the last status read. Scope is Hub-only adoption of the existing Test Genie/workflow-health routed testpool mechanism.
- Return proof remains: declared routed-isolation label, `isolation.installed=true`, nonempty verified lease, `TestPoolRequests > 0`, `PrimaryDuringTestModeRequests = 0`, required file-root counters, and an actual explicit test-mode request routed to that lease. If the existing owner denies or lacks the supported route, preserve its exact refusal and owner; no new lease system or alternate path.
- E1 UI-health run `20261020-160055-10006e04` completed before all phases (0/0) on a two-minute lifecycle-start timeout. Do not rerun that run or repeat its command. This is not a UI-health verdict. E2's later healthy lifecycle operation is recorded in `evidence/E2-attempt1/handoff.md` but supplies no isolated journey or UI-health result.
- Task-authority refusal (seq 10 correction): `UpdateTask` targeted the already-started E3 child task `a0b3c1cc-2881-42bc-b7b6-71adb9d6b5e1` and returned exact `403: agent authority does not include global management or approval`; it did not target the leader's task. This is historical. Do not retry, seek broader edit authority, or treat it as a human hold. Run creation captured the accepted brief; use the current epoch and the previously accepted same-child directive for context.
- Seq 9 supervisor route update: the reviewed Workflow Health routed-observer guard is adopted and healthy on its recorded managed build. E3 now uses the canonical `metadata.labels.routed_isolation=true` declaration and preserves exact case bytes. A valid isolated fixture may exercise application answering only with legitimate fixture authorization; no real-operator impersonation, real decision, consent/grant creation, real notification or external effect is permitted. A separate in-session child directive was accepted by transport after verifying this parent identity and direct-child target; provider consumption remains unknown.
