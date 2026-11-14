# Integrated native lifecycle review D

2026-11-11. Workspace: `/home/matthalloran8/Vrooli-native-lighting`.

**Finding-bearing pass.** D1 is an in-scope validation defect in the normal existing-installation setup journey. Production correctly refuses the fixture's unqualified executable; its old success oracle was not migrated with exact offline adoption. No additional material production counterexample was established. This pass cannot count toward consecutive clean integrated passes, even after D1 is repaired. Native execution, physical output and full SPEC acceptance remain open.

## Applicability

Read the full MAIN delivery CHARTER, SPEC, STATUS, FEEDBACK, native gap audit and implementation checkpoint, then the isolated native design, B/C reports and exact Mac signature facts. The destination remains portable, opt-in, headless lighting with durable Off and actual hardware proof. This is a whole source-integration challenge under that destination, not certification of all H/W/R/L/Q rows. The localized D1 issue is W3 test integration; no unmeasured W0–W2 gate is declared green.

The [initial manifest](review-native-d/source-manifest.json) binds1,030 existing source/test/schema/dependency inputs, SHA256 `ea86a510d526fe1c0348ba48ec8cde833a198185e291a253d557bd410b28753f`. The [final manifest](review-native-d/final-source-manifest.json) adds the new D regression file,1,031 inputs, SHA256 `4655da206e16afd35ab8a5a5bb81e6d58ecfcb77ef983ddccb87bba9a86eb10a`. [End comparison](review-native-d/source-changes.json) reports zero drift in preexisting inputs. This bounded identity is applicability evidence, not a claim of line-by-line review of every manifest entry.

The fixed private Proto selection is `f9e80ce7cb5759aee0b8e0973c883955dedc4c1e9d1056b0cadab2e7f7b478a0`, source digest `fa918a265f8cf0587de1e460d6cf758be25aafa5108abfacb09244a24c86c1b7`; reviewed aggregate descriptor SHA256 `ba614bd6f10f031b697fdc703f5c0fab4820a573d82b392f8001f0cf37b56420`. Approval input SHA256 is `a051ed784291ed9557b24602fe4d7dc69016321322abdb9c2da1288419186473`. Module/lock and Agent Manager refresh provenance are in the manifest. These identities do not constitute a dependency-phase receipt or a generated-freshness verification. No generation was needed or performed; no generated-symbol compile failure occurred. Runtime compatibility stamp is separately excluded selection state, not transferred source.

Every test command used all three explicit variables:

```sh
VROOLI_ROOT=/home/matthalloran8/Vrooli-native-lighting
VROOLI_SOURCE_ROOT=/home/matthalloran8/Vrooli-native-lighting
VROOLI_PROTO_ARTIFACT_ROOT=/home/matthalloran8/Vrooli-native-lighting/.vrooli/work/proto-artifacts
```

Fixture owners isolate home/state/storage; the OS facet additionally uses private HOME/state/GOCACHE. No MAIN writes, vendor execution, SDK/hardware operation, native service/Task/Job/account operation, root install, target dispatch, grant change, dependency install, Git mutation or artifact publication occurred. Supported read-only skill discovery incidentally refreshed the Prompt Manager CLI, reported by its shim. The historical three shared Proto publications and unknown historical consumer effects remain disclosed in the integration checkpoint; D performs no publication or recovery and does not erase those facts.

## D1 — stale success oracle at existing-source setup

The complete `internal/app/runtime` race suite fails:

```text
--- FAIL: TestExistingInstallationSetupSeparatesMetadataFromMissingPolicy
    lighting_commands_test.go:87: adopted OpenRGB is not the exact qualified native release payload or envelope
```

The fixture writes an arbitrary shell script that would print `OpenRGB 1.1`, places it on private PATH and expects ordinary `LightingActions.Setup(EXISTING)` to return `Installed=true`. Current normal setup calls `SetManagedServiceSource(host)`, whose OpenRGB adoption verifies the file digest against the qualified envelope/payload before any execution. `inspectOpenRGBAdoptedSourceForProfile` correctly rejects the script. A version string cannot establish the qualified dependency/ABI/config/constructor profile.

This failure is validation debt caused by a stale desired-behavior oracle, not evidence that production admission should be weakened. It invalidates an otherwise green claim for the complete runtime package. The old fixture's subsequent metadata/access separation assertions are unreachable. The proper repair must retain exact-source rejection/no execution at the public setup boundary and preserve meaningful coverage that installation metadata, selected policy and live access are separate facts. Existing resource-level exact-profile/copy/no-execution fixtures and disabled-status preparation tests are relevant complementary evidence, not a passing result for this broken journey.

Routed to the integration owner with the exact retained failure log. No reviewer production edit or test deletion was performed. At the original reviewed source point D1 was open. The post-repair followup below records the owner correction and its independent check. D remains finding-bearing after repair.

## Integrated challenges

| Boundary | Challenge and conclusion |
| --- | --- |
| Default opt-out and setup | Read normal explicit acquisition/adoption, current-host profile, immutable native publication and disabled/enabled prepare composition. Startup cannot acquire or update. Disabled state does not install/probe/launch. Existing-installation public setup exposed D1 above. |
| Official image and loader closure | Read exact release selection, bounded raw download planning, offline ZIP extraction/link ordering, original-source adoption/copy and pinned source-versus-derived serving trees. Windows excludes fixed PawnIO files only after complete original proof; Mac preserves page bytes while explicitly losing the original whole-resource seal. Native acceptance remains unknown. |
| Native boot/role ownership | Independent OS facet traced actual-operator machine LaunchDaemon and per-role user-domain/process-only owner; Windows boot S4U/LUA and exact create-only role Task with durable RunIssued and nonbreakaway Job. Unknown submit cannot become redispatch permission; old boot/PID/manager observations do not authorize successor termination. |
| Initial/update admission | Read actual capsule material/intent/image/config/access freezing, exact wrapper reconfirmation and authenticated private claim exchange. Configuration/output follows the durable once-only claim. Retained runtime uses the shared native wrapper spec and exact typed launcher proof. No parallel supervisor, controller roster or permanent broker was introduced. |
| Lost caller and changed source | New public regression cancels after durable prepare and before first modeled Run, changes the persisted operator source, then resumes through `AdvanceManagedServiceUpdate`. It requires conflict, byte-equivalent durable admission, zero dispatch/claims and preserved newer source. Subsequent explicit disable retires without restoring that stale source. Darwin and Windows V2 records both pass with modeled native/SDK leaves. |
| Claim/publication failure | New regression enters the actual native capsule executor with a modeled suspended Windows child, successfully consumes the claim, then fails private configuration publication or cancels immediately after it. The child must abort without resume/wait; claim stays consumed; replay cannot republish configuration or resume. Both cases pass. |
| Update/rollback/disable | Read intent-before-resource lock order, frozen physical-grant identity, source/selection resumption checks, durable role reconstruction, exact original/target/restoration ownership and all-role revoke-before-cleanup. Existing public recovery, lost acknowledgements, rollback, boot retirement and C1 durable disable regressions pass in the broad run. |
| Namespace/config/HID/listener | Read retained namespace evidence and launcher/account/home binding; private APPDATA/XDG constructor roots and finite enabled detector configuration; all matching HID routes enrolled before launch and fresh generation at claim/readiness. Listener identity uses the claimed Windows child and rejects foreign/wildcard/ambiguous endpoints. Serial-less SDK controllers remain unenrollable rather than gaining guessed identity; actual controller serial/access applicability remains target qualification. |
| Public API/CLI/UI and policy | Read private scenario bind/drain, public capability admission, stable controller/zone resolution, persisted count configuration/Off-first ordering, fresh resize ACK/readback and full-roster controller effects. Reviewed shared UI count/capability/brightness editor and setup consumer. Policy separates health from utilization, source freshness, activity, manual priority and expiry. Scenario control/policy/source/auth/private-binding race packages pass; previous full189-test UI evidence remains separate and was not rerun or relabeled. |
| Compatibility and debt | Read native tagged policy union and immutable identity, retained claims, Linux omitted legacy fields and shared owner composition. No new production abstraction, compatibility shim or duplicated state machine was added by D. The stale application fixture is the concrete remaining integration debt. |

## Verification

Exact log hashes/byte counts are in [log receipts](review-native-d/log-receipts.json). Commands ran with the three-variable environment above and `-count=1`.

| Check | Result |
| --- | --- |
| Initial new D regressions, race, `./internal/resources -run '^TestNativeD' -v` | PASS8.719s. This loose initial prefix also selected the preexisting `TestNativeDataArtifactStagesVendoredBytesAndChecksum`; no claim that it was a new D test. |
| Resource integration, race, `./internal/resources -run '^(TestNativeD(Prepared\|Claimed)\|TestOpenRGB\|TestNativeOpenRGB)' -v` | PASS165.286s;174 top-level tests,0 failures. Exact actual shell pattern is recorded below. |
| Complete `./internal/lightingruntime ./internal/app/runtime ./internal/operatorstate ./internal/config`, race | Overall FAIL: D1 in runtime22.706s. Other packages PASS3.892s/3.382s/2.503s respectively. |
| Lighting API `./internal/control ./internal/policy/... ./internal/apiauth ./handlers/runtimebinding`, race | All PASS:7.484s,1.307s,1.027s,11.637s,1.024s. |
| Independent platform fixture facet | PASS3.124s; exact filter and private environment in OS report. No actual native OS execution. |

Actual broad shell argument (the table escapes pipes for Markdown):

```sh
go test -race ./internal/resources -run '^(TestNativeD(Prepared|Claimed)|TestOpenRGB|TestNativeOpenRGB)' -count=1 -v
```

The OS facet's [report](../../../.vrooli/work/reviews/native-lifecycle-d-os/RESULT.md) and55-file reviewed hash manifest are retained privately; its manifest SHA256 is `bc4c096af8d24d2196a509b7b9efbdff539ab35d58b358c86815dc4800eda708`. No additional material source defect was established there. These focused fixtures do not replace target platform execution. No Test Genie phase, dependency gate, six-target crossbuild, full UI suite or actual controller output was newly run by D.

## Changed, remaining and unverified

Changed: the uniquely named two-test resource regression file, this report and private review evidence only. Production source unchanged. Verified: the named source boundaries and scoped fixture/race results above, including all174 selected resource tests; complete runtime package is explicitly failed at D1. D1 is repaired and independently rechecked below. Remaining: fresh clean integrated passes; D itself remains finding-bearing.

Unverified: actual Darwin CGO compile/link, user-domain availability at logged-out boot, process-only inheritance across Qt/HID, AMFI/signature/resource-seal/direct-exec acceptance and native listener ABI; actual Windows S4U/profile/storage/task/action/Job/custody/registry/suspended child/token/ACL behavior. All supported targets still need real selected HID/access/SDK/visible Off/static/effects, device return, logout/boot/resume and update/rollback/recovery proof. Windows ARM lighting remains explicitly unsupported for lack of an official native profile, independently of inventory support. Current Linux physical operations, wireless restoration, full delivery acceptance and fixed challenger NONE remain parent-owned and open.

## Post-repair D1 followup

The integration owner repaired only `internal/app/runtime/lighting_commands_test.go`; production admission is unchanged. Its SHA256 changed from `d5c423fbaecade30f639aad261e1d89368c2fb054519f8b51368ad3bafcb5ff3` to `88bd0e173aa066316cc89205cca813be9cdd07e04926fa80eb673731514c8900`. Independent reread confirms both intended proof obligations remain:

- Mandatory negative: actual public Setup sees a version-only shell program with a private execution-marker path, rejects it before execution, and retains no enabled selection/preparation or resource-owner verification/start. It uses ordinary private test directories.
- Positive: the preexisting explicit `VROOLI_OPENRGB_STAGE_FIXTURE` contract supplies approved exact offline AppImage bytes. The fixture copies those bytes into private PATH, uses actual source adoption/sealed staging and preserves metadata-versus-policy/access assertions. No fake qualification seam or new production API was added. Without an explicit archive it skips honestly.

For positive staging the fixture creates one private MkdirTemp directory below the original operator home, then sets its private HOME. It does not chmod the shared temporary root or existing ancestor. Cleanup traverses only that newly created subtree without following symlink directories, makes retained readonly directories removable and removes the subtree. The stage never executes vendor code.

Independent race rerun with all three required root/proto variables and explicit approved archive `/home/matthalloran8/.vrooli/artifacts/openrgb-sealed/1.1/d556c2273b0ef863dbee7ffdb438304b382add5d685c93c60758b42de3b0268d/OpenRGB.AppImage` passed both exact test names: negative0.05s, positive2.44s, package3.538s. [Independent log](review-native-d/d1-independent-repair.log). After the run, no `.vrooli-openrgb-setup-test-*` directory remained under the operator home; its observed mode remained0750. No vendor, SDK or device effect occurred.

The owner's retained complete runtime rerun is PASS23.758s,56 top-level passes,0 failures and1 unrelated build-integration opt-in skip. Both repaired adoption tests actually passed without skip. Reviewer read the JSONL test events and independently verified log SHA256 `ae42ae369eabb239699f1c854540d6682d90104dd6b00562cf8bd27fccb5cf6b` against its receipt in `.vrooli/work/native-portability-history/verification-final/runtime-full-after-adoption-test-repair-receipt.json`. This is owner-produced complete-gate evidence, separately identified from the independent focused rerun. The original failed log remains retained.

[Post-repair source identity](review-native-d/post-repair-source-manifest.json) binds the same1,031 inputs, SHA256 `f1791dad129b66ebf382067a16762a5fbdb1134687dafe29b97819767a7adc0c`; its [comparison](review-native-d/post-repair-source-changes.json) contains exactly the test-only repair above. Production/schema/dependency inputs did not change in this comparison. D1 is closed as validation debt; D remains a finding-bearing historical pass and does not gain a clean-pass count. All native/physical/full-destination qualification limits above remain.
