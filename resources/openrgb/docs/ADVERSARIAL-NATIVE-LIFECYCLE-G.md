# Integrated native lifecycle review G

2026-11-11. Source workspace: `/home/matthalloran8/Vrooli-native-lighting`.

**Finding-bearing whole integrated native Lighting source pass.** G1 finds Windows scheduler quiescence/cancellation evidence gaps across exact per-role Tasks and the ordinary boot supervisor. G2 finds scoped manual Off wrongly disables global Off controls for still-lit controllers. Both were sent to the integration owner before any product edits. G does not count as a clean pass. F remains a historical single clean pass; the consecutive-clean source requirement is not met. Native execution, physical lighting and full delivery SPEC acceptance remain open.

## Scope and applicability

Read MAIN delivery CHARTER, SPEC, STATUS, FEEDBACK, native gap audit and implementation checkpoint, then the isolated native design, release profiles, whole E/F reports and final E1/E2 repair/aftercheck evidence. Applied scenario-work-ladder to the established W3 implementation boundaries in this commissioned source review, harness-goal-authoring to fresh facet briefs, and writing-standards to evidence/reporting. No unmeasured W0–W2 or full-scenario gate is claimed. Recall searched nine providers; agent-manager semantic retrieval degraded while lexical results remained available. No conclusion assumes complete historical discovery.

The [entry source manifest](review-native-g/source-manifest.json) binds the same1,169 production, tests, schemas/generated clients, modules/locks, approval and provenance inputs as F, independently remeasured with zero entry drift. Manifest SHA256 `36f62dbf23e8d4103512024cb5290edd04a6e949128e970988d9a7a68846a589`. [Final comparison](review-native-g/source-changes.json) records zero changed bound inputs. Final log receipts are recorded below. These manifests bound applicability; a digest does not make a mutable workspace immutable or imply line-by-line review of every file.

Fixed private Proto artifact: `f9e80ce7cb5759aee0b8e0973c883955dedc4c1e9d1056b0cadab2e7f7b478a0`; source: `fa918a265f8cf0587de1e460d6cf758be25aafa5108abfacb09244a24c86c1b7`; aggregate descriptor: `ba614bd6f10f031b697fdc703f5c0fab4820a573d82b392f8001f0cf37b56420`. The canonical private `make verify-committed-gen` exited0. G performed no generation/publication and did not chase later unrelated MAIN descriptor changes. Module/approval identities are applicability inputs, not a new SDA dependency receipt. Historical accidental shared Proto publications, canonical recovery and unknown historical consumer effects remain disclosed in the integration checkpoint.

Every source-aware test/build explicitly supplied:

```sh
VROOLI_ROOT=/home/matthalloran8/Vrooli-native-lighting
VROOLI_SOURCE_ROOT=/home/matthalloran8/Vrooli-native-lighting
VROOLI_PROTO_ARTIFACT_ROOT=/home/matthalloran8/Vrooli-native-lighting/.vrooli/work/proto-artifacts
```

Private fixture home/state/resource/intent/fence/database roots only. SDK checks use private loopback byte fixtures. MAIN remained read-only. No root/account/OS service/Task/Job, vendor executable, physical device/SDK, target/grant, primary product API, Git mutation or dependency installation occurred. Parent retains actual Linux human permission retry/integration/effects and journal ownership.

## G1 — Windows Task absence is accepted without authoritative scheduler state

Given an exact native role whose task cancellation was requested, when a fresh snapshot has no exposed instances but reports `TASK_STATE_UNKNOWN` or an invalid enum, retirement must retain uncertainty. The current `stopWindowsOwnedTask` instead returns success for every empty-instance state except2(queued) and4(running). `readWindowsTaskSnapshot` checks the VARIANT type but accepts any integer state. The current `deleteWindowsOwnedTask` ignores state completely once disabled/no instances is observed, so a fresh unknown, queued or running state can precede deletion.

Normal `stopOwnedOperation` revokes admission and calls this stop owner before returning NativeQuiescent; `releaseOwnedOperation` calls stop again, checks Job absence, deletes the Task, then deletes its custody record. Empty/absent Job evidence does not establish extinction of a not-yet-joined queued wrapper. These are false Task quiescence/metadata-retirement source receipts if the native API exposes the modeled state. Durable resource/capsule revocation independently prevents an unclaimed vendor payload from acquiring execution authority; no actual vendor execution, hardware escape or native occurrence is claimed.

The same family reaches ordinary boot supervision: `awaitWindowsTaskQuiescent` accepts any state when GetInstances is empty. `stopWindowsOperatorTask` cancels only enumerated instances and skips a registered-task cancellation when the snapshot is queued-without-instance. Ordinary stop/restart/removal can therefore acknowledge this modeled dispatch without canceling it; restart proceeds to another Run. Repair must cover ordinary boot and exact per-role owners together while preserving exact definition/incarnation checks, no redispatch on uncertainty, read/execute versus setup authority and legitimate empty Task absence. An extra human grant, parallel process roster or broad cancellation is not indicated.

[Independent OS facet](../../../.vrooli/work/reviews/native-lifecycle-g-os/RESULT.md) retains exact source bodies with only native leaves substituted, desired-behavior failing overlays, official Task-state semantics, normal callers and source/log identities. Darwin unknown-kernel/reused-root/changed-boot probes pass. Parent was informed before any production repair. G remains finding-bearing after future afterchecks.

## G2 — scoped manual Off disables the normal global Off actions

Given enrolled controllers A and B, with a durable manual Off scoped only to A and nonzero desired output on B, Dashboard global Off and Controller B Keep Off must remain available. Both current controls disable on `manualOverride.active` alone. Their actions call the all-enrolled `setOff()` operation; Presets already tests `active && allEnrolled` correctly.

Fresh private Vitest desired-behavior cases fail at the expected-enabled assertions for Dashboard and Controller B. An actual public Connect-handler/SQLite fixture independently establishes that scoped A Off preserves B's nonzero desired output. Thus these normal screens prevent the operator extending Off to all enrolled controllers without clearing an existing Off or changing pages. Dashboard Inspector also receives a scope-blind offActive flag; scope-aware status/indication belongs in the same coherent repair. The report does not claim the API rejects a global Off request or that Presets is unusable.

[Independent consumer facet](../../../.vrooli/work/reviews/native-lifecycle-g-consumer/RESULT.md) retains exact UI/source traces, failing desired-behavior fixture, public handler/SQLite scope proof, broader new successful API/SDK probes and logs. No product edits were made by G. Parent was informed for the retained zone/UI owner.

## Additional fresh desired-behavior challenge

Given two completed native replacements and a third publication whose durable write succeeds but acknowledgement is lost, when cleanup of one old completed role fails, ordinary Disable must attempt every retained role, preserve durable Disabling and revoke every consumed claim. A later normal retry must finish Disabled without another dispatch/claim, writer binding or recovery restart.

The new [private resource overlay](review-native-g/native_g_multi_history_test.go.txt) exercises actual public Advance/admission/explicit rollback, three persisted ledger records, exec stores and Lighting desired intent. Both Darwin and Windows variants pass under race28.711s. The late Claim checks hit actual durable revocation; the disabled Advance/Resume/Reconcile checks cannot revive uncertain publication. Native manager/HID/SDK execution leaves are modeled, separate from source state-machine/persistence proof.

## Whole integration assessment

| Boundary | Independent review and result |
| --- | --- |
| Default opt-out/acquire/adopt | Constructor/status/disabled reconciliation retain no acquisition/process/probe path. Explicit setup stages through governed exact bounded OS/architecture release profiles. Startup inspects retained/staged bytes without silent acquisition/upgrade. Full runtime executed both exact approved offline adoption positive and version-only no-execution negative. |
| Artifacts/platform prerequisites | Reviewed all five profiles, archive/payload/derived serving identities, complete ABI/loader graphs, immutable ancestry and source-preserving adoption. Windows derives16PE images by the exact6PawnIO omissions; Mac derives77images with explicit invalid original whole-resource-seal caveat. Byte/loader verification is not Security.framework, AMFI/Gatekeeper or actual exec acceptance. Optional offline checks skipped below retain earlier measured-byte applicability rather than new G credit. |
| Setup/prepare/start | Reviewed selected source/catalog/policy/account/machine checks and current namespace/boot-definition/retained-launcher proof. Enabled prepare is content-only, disabled setup owns provisioning; resource commands take durable intent lock before lifecycle lock. No alternate broker/account/capacity owner was introduced. |
| Once-claim and image/context | Reviewed immutable capsule/self-image, repeated exact wrapper observation, argv/env/cwd/material bindings and current access/generation/transport scope. Windows suspended exact child is claimed before resume; Darwin process-only proof precedes replacement. Unknown/replayed claims cannot become another execution admission. |
| Update/rollback/recovery | Reviewed frozen original/target/rollback policy, retained current-role launcher and actual ledger reconstruction after caller loss. New three-record public history challenge extends coverage beyond one pending role; cleanup errors retain durable opt-out and revoke all historical claims. No path guessing or version-only adoption substitutes for exact image/contract evidence. |
| Stop/disable/retire/boot | E1 shared extinction proof retains Darwin dispatched/rootless uncertainty and metadata until exact recovery/removal. G1 exposes Windows scheduler state/cancel gaps, including ordinary boot supervisor, so integrated retirement source maturity is not clean. Existing C1 private durable-writer/Unix sync behavior remains covered. |
| Listener/private config/HID | Reviewed literal127.0.0.1-only ownership with serving-child identity on Windows, bracketed process/readiness checks and constructor APPDATA/XDG isolation before vendor startup; remotes/autoprofiles remain disabled. Finite Aura motherboard/ROG Terminal HID selection requires complete matching routes/current generation/access. No admin bus/driver fallback or generic vendor wildcard is admitted. |
| Public consumers/policy/security | Fresh consumer crosses real handler/SQLite/SDK byte boundaries, auth/private binding, enrollment/count/resize/readback, manual Off and stale metrics. E2 read-only unavailable roster keeps old LastSeen/capability warning without fresh Sync. G2 shows normal Off controls still conflate scoped/global overrides. API/CLI/UI passing baseline checks do not erase this failing desired-behavior counterexample. |
| Linux compatibility/debt | Reviewed omitted legacy native fields/nil-launcher-proof compatibility and existing shared initial/update/reconciliation owners. No duplicate process/controller roster, new account/broker or execution shim was established. G findings should repair existing scheduler and UI scope owners rather than add parallel state. |

## Verification

[Exact log receipts](review-native-g/log-receipts.json), SHA256 `d8cff7e6297b8992c91450b1c71ec3fabc1798e798ec2cdf7ac36a2f197965a3`, retain complete output, commands/cwd, exit codes and selected test counts. All Go commands use `-count=1`. Facet reports bind their separate logs and exact scope.

| Check | Result |
| --- | --- |
| Canonical private Proto verifier | exit0, no generation |
| New actual public three-record history/rollback/uncertain-publication Disable overlay, race | PASS28.711s, Darwin/Windows; selector also ran one existing NativeGlobalRoot guard, so2top-level passes total |
| Root resources OpenRGB/native/D/E selection, race | PASS160.351s,180top-level passes,8explicit optional/offline skips |
| Complete root application runtime, race, approved offline adoption input supplied | PASS27.244s,56top-level passes,1unrelated opt-in skip |
| Complete Lighting Runtime, race | PASS2.499s,26top-level passes; independent consumer repeat PASS1.737s |
| Hostreqkit device/access selection, race | PASS3.223s,41top-level passes; operatorstate has1native skip; config matched no named tests and supplies compile-only evidence |
| New OS actual-boundary overlays, race | Expected FAIL0.146s:3Windows boundaries fail;2Darwin top-level probes pass |
| Existing focused native platform selection, race | PASS4.735s,161top-level passes/365entries |
| New public Connect/SQLite, SDK and private binding probes, race | All3PASS, including4private-binding contradictory-evidence cases |
| Consumer full API with new Go overlays, race / full CLI | PASS20tested packages,4no-test packages / all6packages |
| Consumer complete UI / TypeScript | PASS192tests/32files3.45s / exit0 |
| New actual-page scoped-Off desired-behavior tests | Expected FAIL2assertions/1file; both normal global actions incorrectly disabled |

The broad resource skips are exact optional offline artifact/adoption/loader/normalization/ARM/prerequisite/measurement tests listed in the receipts; none is claimed as G actual artifact execution. The runtime skip is the unrelated opt-in actual-build contract test. Failing new tests are retained as findings, not concealed as skips. Root broad resource log SHA256 is `838c120214568c83a9bd9fb7c57659826d660b66cf22266f280a375011ea936e`; new resource overlay SHA256 `111c9be23779b34a485abbe0ebc0b1191ecb4cb81fa570dbf3b968a70ef422f8`.

Direct source checks are not new Test Genie scenario/dependency/docs receipts, native target runs or physical/browser operation. Existing official20261111-161603-77afa5cc unit/docs PASS retains its own isolated producer-root and degraded source-identity/advisory limits; G does not relabel that run.

## Changed, verified, remaining and unverified

Changed: this report, private resource/OS/consumer overlays, evidence/checkpoints/manifests/logs only. All test commands completed; fixture sockets/processes closed, no resource/service/model remains running. No product source/test/schema/dependency edit, generation or MAIN write by G.

Verified: G1/G2 source counterexamples, named actual source-boundary challenges and scoped passing checks with exact limitations above. G is finding-bearing; repair/aftercheck and fresh whole clean passes remain. Parent owns coordinated MAIN transfer of owned Lighting schemas/clients followed by canonical destination scoped generation, without isolated runtime pointer/global descriptor overwrite.

Unverified: Darwin CGO/native link/API/LaunchDaemon logged-out user-domain/derived-bundle AMFI-Gatekeeper-signature-exec/no-fork constructor compatibility; Windows S4U/Task/Job/child/token/ACL/profile/private socket/native cancellation and scheduler reachability; Linux ARM runtime/prerequisites. All selected HID access, actual SDK/controller/visible Off/static/effects, reconnect/device return/logout/boot/resume and physical update/rollback/recovery remain open. Windows ARM lighting remains explicitly unsupported without an official native1.1 asset, separate from inventory/readiness. Current Linux actual lighting permission/output and wireless restoration remain parent/operator-owned. No source check, cross-build, successful SDK acknowledgement or clean source review establishes physical light output or full H/W/R/L/Q completion.

Retained probe placement: the private Go-overlay payload is stored as `.go.txt` with its original bytes and digest. It is evidence projected into the owning `internal/resources` package by the recorded overlay, not a standalone product package under this documentation directory. Historical receipts retain their original path names; see `PROBE-ARTIFACT-LOCATION.json` beside the payload for the relocation map.
