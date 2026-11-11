# Integrated native lifecycle review E

2026-11-11. Source workspace: `/home/matthalloran8/Vrooli-native-lighting`.

**Finding-bearing whole-source pass.** E1 exposes false Darwin retirement of an uncertain rootless submission. E2 loses persisted public controller inventory while the backend is unavailable. Both findings are accepted by the integration owner for repair. E cannot count toward consecutive clean passes, including after repairs pass their checks. Actual native execution, visible lighting output and full delivery acceptance remain open.

## Scope and applicability

Read the complete MAIN delivery CHARTER, SPEC, STATUS, FEEDBACK, native gap audit and current implementation checkpoint, then isolated native design and B/C/D reports. Applied scenario-work-ladder to this source implementation review and writing-standards to the report. This is W3 source integration scrutiny under the supplied destination; it does not claim unmeasured W0–W2 gates or all H/W/R/L/Q acceptance.

The [initial manifest](review-native-e/source-manifest.json) binds1,031 source, test, schema, generated-client, dependency and provenance inputs, SHA256 `e723317f68080f577953d24060ea93716322437e94759092381638875f369e32`. It remeasures the D post-repair path set at E entry. The fixed private Proto dependency point remains artifact `f9e80ce7cb5759aee0b8e0973c883955dedc4c1e9d1056b0cadab2e7f7b478a0`, source `fa918a265f8cf0587de1e460d6cf758be25aafa5108abfacb09244a24c86c1b7`, aggregate descriptor `ba614bd6f10f031b697fdc703f5c0fab4820a573d82b392f8001f0cf37b56420`. Unrelated later MAIN aggregate churn does not restart these domain proofs. Approval/module/lock identities are applicability inputs, not a dependency-phase receipt. No generation, publication or verifier-driven mutation occurred. The [report-checkpoint manifest](review-native-e/final-source-manifest.json) adds the two E tests, binds1,033inputs and has SHA256 `8ac5518e950799228fb0e651d7ebea9034c795299a881526c322711b7b3fb4e0`; [comparison](review-native-e/source-changes.json) records zero drift in the original1,031inputs before owner repair acceptance. This manifest does not cover later repair changes, which require a separate followup identity.

All source-aware checks used these three absolute roots:

```sh
VROOLI_ROOT=/home/matthalloran8/Vrooli-native-lighting
VROOLI_SOURCE_ROOT=/home/matthalloran8/Vrooli-native-lighting
VROOLI_PROTO_ARTIFACT_ROOT=/home/matthalloran8/Vrooli-native-lighting/.vrooli/work/proto-artifacts
```

Fixture owners retain private HOME/state/resource/contract roots. The OS facet uses its own private HOME/state and overlay. MAIN was read only. There were no vendor executions, physical SDK/device requests, native service/Task/Job/account operations, root changes, target dispatches, grant changes, Git mutations or dependency installs. Read-only discovery contacted existing search/Prompt Manager services; some providers degraded, and the consumer facet reports incidental automatic search-hub/Prompt Manager CLI shim rebuilds. Discovery was not treated as complete history.

## E1 — uncertain Darwin rootless submission loses its extinction fence

Given a same-boot durably prepared bootstrap whose reply is lost, when native manager association is absent but its submitted wrapper remains alive, the owner must retain uncertainty until exact extinction is proven. The independent OS overlay invokes the actual `darwinOwnedBackend` and actual file store with the existing modeled native edge. Bootstrap launches modeled PID42, returns an uncertain reply, then manager association disappears while that process remains alive. Lookup returns RootPID0. Actual Stop reports NativeQuiescent=true, Release succeeds, and Released reports true while the modeled wrapper remains alive.

`claimAlive` treats zero PID as false. Stop and Release therefore conflate an unobserved submitted root with an extinct root. This bypasses the same-boot conservative uncertainty already encoded by the backend's `absent` method. The retained rooted-orphan tests establish that absent-manager/live-root is an admitted state of this abstraction; E adds its missing rootless counterpart.

Normal resource cleanup reaches it: `retireOpenRGBInitialWith` revokes the fence, looks up the role and, before a durable Claim/NativeRef exists, has no rooted comparison to reject the rootless reference. It accepts Stop's quiescence, persists that receipt, releases native metadata and marks the attempt stopped. Thus the defect is a false trusted-wrapper extinction receipt and discarded uncertainty.

This is a source-contract counterexample with modeled OS leaves. Actual modern launchd reachability remains unqualified. The capsule independently refuses a rootless lookup before vendor preparation/claim; E1 does not prove unclaimed vendor execution, hardware access or a same-user sandbox escape. The fix belongs in the existing native owner, preserving no-dispatch and changed-boot recovery without a UID-wide search, parallel roster or invented account/broker.

[OS facet report](../../../.vrooli/work/reviews/native-lifecycle-e-os/RESULT.md) retains full command,25-file source manifest and normal cleanup trace. The final failing log SHA256 is `be78ccbcb57ed1d4b9cbdb2cfa26064d1d736616bd343d4d944ce6e344e0ee12`; private overlay fixture SHA256 is `572cce68212d2fbd128e8c1735108205192cb13e622fabb9dfdfad32de246f1e`. Parent assigned `native_owned_services` the coherent repair and retained E for independent aftercheck.

## E2 — offline enrolled inventory disappears at the public consumer

Given an enrolled persisted controller/zone and durable manual Off, when native reconciliation has quiesced its actual `BackendBinding` or SDK discovery returns a transport error, `ListControllers(IncludeUnavailable=true)` must preserve the persisted unavailable inventory without presenting it as freshly connected or writing the device.

The new `native_e_consumer_inventory_test.go` crosses the actual public Connect handler and SQLite roster. Both subcases return an empty controller list although the retained enrolled identity/zones/Off still exist. The UI's ordinary `readDesk` explicitly requests IncludeUnavailable, so backend absence removes controller inspection/configuration from the public screen. Nil-backend and successful-but-absent discovery already retain unavailable inventory; failure on a nonnil native binding takes a different early return.

A repair must retain unavailable/error status, load the existing roster only when requested, avoid `SyncControllers` on failed discovery and preserve strict write admission. It must not invent fresh capability or physical observations. The before log and [facet source/consumer trace](../../../.vrooli/work/reviews/native-lifecycle-e-consumer/RESULT.md) are retained under `.vrooli/work/reviews/native-lifecycle-e-consumer`. Its209-input source manifest SHA256 is `aa53c597ef2323891ad166f8d4214237e582fff7d353deb4a1a9e075c799f2a5`. Parent assigned `native_lighting_zone_configuration` the repair and retained E for independent aftercheck.

## Whole integration challenges

| Boundary | Independent challenge and outcome |
| --- | --- |
| Default opt-out/acquire/adopt | Read current-host exact release profiles, OS/arch-only bounded raw acquisition and public existing-source setup. Acquisition and content publication remain explicit; startup cannot download/upgrade. Adoption verifies original envelope or complete original native tree/ABI before derivation, rather than accepting a version string. The repaired public negative and approved offline positive setup cases both pass in the complete runtime check. |
| Complete native artifacts | Read five profile/ABI pins and original-versus-serving derivation, extraction/adoption copy and immutable publication. Windows retains its16-PE safe-HID tree after fixed6PawnIO omissions; Mac77-image serving bundles explicitly lack the original whole-app resource-seal guarantee. Byte/loader proof does not establish AMFI/Gatekeeper/exec acceptance. Prior actual artifact proofs retain their own applicability; E's optional offline artifact cases skipped are not new measured-byte proof. |
| Normal preparation/start | Read real enabled intent/source/selected-device/catalog checks, namespace/account/private locks and shared wrapper spec. Enabled target prepare is content-only; no dispatch authority is acquired through staging. Intent lock precedes lifecycle ownership; ordinary start cannot advance an admitted update. |
| Capsule and once claim | Read exact self image/invocation/environment/cwd lookup, material/transport generation, repeated wrapper observation, durable claim before config/output, Windows suspended-child claim before resume and Darwin process-only verifier. Uncertain claim retains the consumed fence; no new direct-process fallback exists. |
| Update/rollback/lost caller | Read durable role reconstruction, unchanged uncertain invocation, fresh restoration, source/lighting-selection resumption checks and policy/serving publication under intent/resource locks. Existing integrated public journeys and D changed-source/publication-failure cases pass. Retained code uses current-role launcher plus typed proof, not historical path guessing. |
| Disable/retirement/boot | New E public test interrupts native V2 update after claim, fails cleanup, checks durable Disabling and consumed revoked fence, refuses late Claim and update, then retries ordinary Disable to terminal Disabled without another dispatch/claim. Darwin/Windows variants pass. E1 finds the separate actual platform rootless retirement gap. C1's real private-writer durability path remains covered. |
| Namespace/ownership/listener | Read actual operator home/account binding, retained namespace after boot-definition removal, strict native IPC frames, current incarnation and listener ownership using the Windows serving child. A sole literal127.0.0.1 endpoint is required; unknown/foreign/wildcard/ambiguous listeners cannot qualify readiness. |
| Devices/private config | Read native complete matching HID route enrollment, current generation/access at claim, finite detector catalog/private constructor APPDATA/XDG isolation and safe one-shot HID settings. No privileged bus driver or broad administrator fallback is implied. Hostreqkit's34 selected OpenRGB/native HID tests all pass. |
| Public/API/CLI/UI/policy | Independent consumer facet traces actual handler/service/repository/private binding and current UI loading/manual/config flows, stable zone identities, capability count/resize/Off ordering, metric health/activity/freshness rules and auth. E2 is its material source finding. Successful SDK acknowledgements never become physical-light claims. |
| Linux compatibility and debt | Read native tagged extensions versus legacy omitted fields, shared Linux lifecycle/update owners, source/retained claim binding and default durable dispatch. The review adds boundary regressions without a production shim, parallel state machine or capability roster. E1/E2 corrections must remain at their existing owners. |

## Verification

All Go commands used `-count=1`; complete JSONL output is retained in [review-native-e](review-native-e/). [Log receipts](review-native-e/log-receipts.json) bind commands, byte counts, hashes and selected tests; receipt SHA256 `ba1e1f320c268f5036ebd0346792ab739993f44c0f1606bbb6fb68bac5249e96`. No new scenario Test Genie/dependency/whole UI/native-target run is attributed to E.

| Check | Observed result before E1/E2 repairs |
| --- | --- |
| `go test -race ./internal/resources -run '^(TestOpenRGB|TestNativeOpenRGB|TestNativeD)' -json` | PASS185.726s;175top-level passes,8explicit optional/offline skips |
| New E failed-disable/public retry boundary, exact selected test, race | PASS9.536s; Darwin/Windows subcases pass |
| Complete `./internal/lightingruntime ./internal/app/runtime ./internal/operatorstate ./internal/config`, race | All PASS2.947s/32.351s/3.731s/2.779s;130top-level passes,2unrelated/native skips |
| `./internal/hostreqkit -run '(OpenRGB|NativeHID)'`, race | PASS3.225s;34top-level passes,0skips |
| Consumer facet policy/sources/handler/auth/private binding, race | All PASS1.402s/1.028s/3.252s/9.969s/1.047s; exact module selectors/logs in facet report |
| Complete consumer UI / CLI / lightingruntime | PASS189tests/32files4.24s; all6CLI packages PASS; separate lightingruntime race PASS1.906s |
| OS actual-backend/private overlay rootless counterexample | FAIL as expected at E1; no native execution |
| Public Connect-handler offline inventory counterexample | Both subcases FAIL at E2; no device execution |

The runtime command explicitly supplied the existing approved offline AppImage fixture path ending `d556c2273b0ef863dbee7ffdb438304b382add5d685c93c60758b42de3b0268d/OpenRGB.AppImage`. Its normal existing-source positive and mandatory version-only negative cases actually ran. The vendor executable was never run.

The first E retry test incorrectly treated successful metadata reconstruction (`loadOpenRGBNativeUpdateDispatch`) as completed execution admission; production still rejects at the following current-intent/claim checks. That reviewer oracle failed in both variants and is retained in `new-regressions.jsonl`. The corrected test checks the actual durable `ClaimOpenRGB` revocation boundary, rejected public Advance and successful public Disable retry; it passes. This is disclosed reviewer test design correction, not another product finding.

## Changed, verified, remaining and unverified

Changed by E: one resource desired-behavior test, one public consumer desired-behavior test, private OS overlay/evidence, manifests/logs/checkpoints and this report. E production source edits: none. Verified: named actual source boundaries, the two source counterexamples, normal resource cleanup/public consumer reachability and passing scoped checks above. Repairs and independent followup are pending at the assigned owners. Parent owns the shared work journal and final integration; E performs no shared journal mutation.

Unverified: Darwin native CGO compile/link, logged-out user-domain boot, no-fork vendor compatibility, derived-bundle AMFI/signature/exec acceptance, native listener ABI; Windows S4U/Task/Job/child/token/ACL/profile/storage/runtime behavior; Linux ARM runtime; selected HID access, SDK/controller/visible Off/static/effects, reconnect/device return/logout/boot/resume and actual update/rollback/recovery on each target. Windows ARM lighting remains explicitly unsupported for lack of an official native asset, separately from portable inventory. Linux actual permission/output, wireless restoration, full SPEC acceptance and fixed full-destination challenger remain parent-owned and open.

The historical three accidental shared Proto publications and unknown historical consumer effects remain disclosed in the integration checkpoint. E neither republishes nor erases that operating history. Its private fixture/source results do not promote any native/physical qualification row green.
