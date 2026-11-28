# Integrated native lifecycle review C

2026-11-11. Workspace: `/home/matthalloran8/Vrooli-native-lighting`.

**Finding-bearing pass.** One material Windows opt-out defect was found and
repaired by the implementation owner during this review. This pass cannot count
as a clean pass, including after its affected regression checks pass. No other
material source counterexample was established in the reviewed integrated paths.
Native execution, hardware and full delivery acceptance remain open.

## Scope and identity

Read the MAIN delivery CHARTER, SPEC, STATUS, FEEDBACK and native gap audit, then
isolated NATIVE-PORTABILITY and clean integrated B. The approved portable,
headless, persistent and physical destination remains the denominator. This is a
source integration review under that destination, not certification of every
SPEC row. The localized defect is W3; no unmeasured W0–W2 gate is claimed.

The initial [source manifest](review-native-c/source-manifest.json) binds1,002
files and has SHA256 `d43a1e8c3a3ddc40cb046bd76831af13dde67d40d8cecd8ef9ad5cec36821ca9`.
The [final source manifest](review-native-c/final-source-manifest.json) binds1,029
files and has SHA256 `e86fa44f488201244b1cf27db10c85e5eaf9242e583f00e7058af1634ec35c83`.
The final manifest additionally records the new regression, actual Lighting
schemas/generated Go/TypeScript/descriptor, UI package/lock and dependency
approval inputs. This is a bounded identity for applicability; it does not claim
line-by-line review of every manifest entry. Dependency input identity is not a
passing dependency-validation receipt.

[Source changes](review-native-c/source-changes.json) records exactly four
changed preexisting inputs: `openrgb_headless.go`, `openrgb_device_policy.go`,
`openrgb_device_policy_native.go` and `openrgb_device_setup.go`. These are the
owner's C1 correction and helper reuse. Reviewer production edits: none. The
reviewer added only `internal/resources/openrgb_native_disable_durability_c_test.go`
and review evidence. Independent native OS facet evidence is retained in
`.vrooli/work/reviews/native-lifecycle-c-os/RESULT.md`.

## C1 — Windows disable cannot reach retained-role cleanup

Before repair, normal `OpenRGBHeadless.DisableOpenRGB` called `persistDisabled`
and returned its error before `completeOpenRGBDisableCleanup`. After successfully
persisting `enabled=false`, `persistDisabled` opened the state parent using
`os.Open` and unconditionally called `dir.Sync()`. Windows Go opens that handle
with GENERIC_READ; `File.Sync` reaches `syscall.Fsync` and `FlushFileBuffers`.
[Microsoft requires GENERIC_WRITE for FlushFileBuffers](https://learn.microsoft.com/en-us/windows/win32/api/fileapi/nf-fileapi-flushfilebuffers).
Therefore this call cannot satisfy the documented API contract. Every retry
repeats the failure before initial/update/serving-role retirement. Disabled state
alone does not stop an existing backend. This is a concrete source/API
counterexample, independently confirmed by the OS reviewer; no Windows runtime
failure is falsely claimed as observed.

The preexisting Linux headless fixture skips Windows, so its successful disable
retries did not exercise this boundary. The new regression uses real private
operator-state publication and actual `persistDisabled`, without release,
manager, SDK or hardware effects. Its second test exercises the default resource
controller composition on Darwin/Windows, rather than an injected operator
service, and first checks that the resolved state remains under its private
logical home. That native-only test is compiled here and skipped on Linux.

The owner replaced the duplicate flush with existing shared
`confirmOpenRGBPublicationDurability`, also used by the device-policy/pending
writers. Unix retains actual directory fsync and propagated errors. Windows
relies on the already completed private durable writer rather than attempting a
POSIX directory flush. Independent tracing verified the normal production path:
`Controller.operatorStateService` sets `LogicalOperatorHome=c.Home` on native
OSes; Apply reaches `writeStateFile`, `nativePrivateStorage`,
`writeNativeStateFile`, then `config.WritePrivateFileDurable`; its temporary file
is synced before publication, and Windows publication uses
`MoveFileEx(REPLACE_EXISTING|WRITE_THROUGH)` followed by file identity and private
access checks. Disabled-state readback remains required. No broad error ignoring,
new privileged broker, separate account or operator-state API was introduced.

The affected Linux regression/policy/setup checks pass after repair. Actual
Windows filesystem, disable and retirement execution remains unqualified; a
cross-compile is not a before-fail/after-pass native execution result.

## Other integrated challenges

| Boundary | Independent challenge and result |
| --- | --- |
| Acquisition/adoption and profile | Traced OS/architecture-only bounded acquisition, explicit setup/publication, original-source versus derived serving identity and complete profile closure. Native profiles do not implicitly enable lighting; startup cannot acquire/update. Mac signature acceptance and Windows prerequisites remain explicit target gates. No additional source counterexample established. |
| Setup, prepare and normal command | Traced current operator source/enablement, selected device/catalog binding, immutable role preparation and public start/restart routing through the same intent-before-lifecycle lock. Preparation is not execution permission; unknown/pending setup cannot qualify launch. |
| Once-only execution | Read actual capsule lookup/image/environment/material/transport sequence, independent ownership reconfirmation, durable Claim before configuration and payload handoff, Windows suspended child claim before resume, and Darwin no-child verifier before replacement. Uncertain Claim remains terminal for that invocation. Existing real fence/transport and modeled public journeys pass. |
| Recovery, update and rollback | Traced durable role reconstruction after caller loss, conservative unknown-submit recovery, unchanged primary identity, fresh restoration admission, retained code and typed proof, and publication under resource plus lighting locks. Existing actual persisted journey fixtures cover target failure, interrupted readiness, lost start/publication acknowledgements and changed selection. |
| Disable, boot and namespace | Challenged the actual normal disable path rather than only modeled leaf owners, finding C1. Also traced durable Disabling/quiescence ordering, revoke-before-cleanup, all-role cleanup despite errors, retained fencing on unknown absence, old-boot claim preservation and current-boot exact retirement, retained namespace/launcher after boot-definition removal and stopped policy replacement. |
| Native OS ownership | Independent facet reviewed machine LaunchDaemon/UserName and user-domain process-only roles; Windows S4U/LUA create-only Task, durable RunIssued, exact action/engine birth and nonbreakaway Job, suspended child/image/token/claim, native listener and storage/security adapters. No further material source defect established. Modern launchd routing, S4U credentials and actual Job/ACL behavior remain unqualified. |
| Device/configuration scope | Traced safe HID one-shot detection and complete matching selected routes; private constructor APPDATA/XDG roots, finite motherboard/Terminal catalog selection and dynamic-registration exclusion. Reviewed real external-controller config fixture and current native generation/access witness separation. No broad privileged-bus or physical-light claim follows from this source. |
| Full consumer path | Read persistent intent/resource update and native binding owners through scenario BackendBinding acquire/drain, stable controller/zone identity, saved configuration and initial Off, manual override, brightness cap, SDK fresh handles/resize ACK/readback/capability checks and truthful partial-write reporting. Read shared editor/API controls and current final zone A/B evidence. Control/auth/server, CLI and UI tests pass; no new roster/writer or direct unauthenticated SDK surface was introduced. |
| Compatibility/debt | Reviewed legacy omission versus native tagged identity, retained role/profile semantics and persistent-state classifier coverage. The C1 repair removes a duplicated incompatible durability operation by reusing its existing owner. Windows ARM lighting stays explicitly unsupported while inventory remains separate. |

## Verification and limits

Every source-aware test command set all three absolute isolated roots:
`VROOLI_ROOT`, `VROOLI_SOURCE_ROOT` and `VROOLI_PROTO_ARTIFACT_ROOT`, the last to
`/home/matthalloran8/Vrooli-native-lighting/.vrooli/work/proto-artifacts`.
Resource TestMain and individual fixtures retain private HOME/resource/artifact
and contract storage. No generation or publication ran.

[Verification index](review-native-c/verification-index.json) binds exact logs.
All Go runs used `-count=1`.

| Check | Observed result |
| --- | --- |
| New real private-writer disable retry, Linux race before repair | PASS1.209s; Linux does not expose Windows API defect |
| Entire lightingruntime/operatorstate/config, race | PASS7.231s /15.671s /2.990s |
| Integrated resource native public journeys/capsule/transport/boot/retained namespace/external config/setup/access/acquisition, race | PASS107.169s; compiled before C1 repair |
| Post-repair disable retry plus policy/setup/disable paths, race | PASS3.341s; native default-owner case skipped on Linux |
| Full control/apiauth/server, race | PASS10.337s /13.318s /1.021s |
| All Lighting CLI packages | PASS |
| Lighting lights/page UI | PASS82tests,2.40s |
| Persistent codec/state/contract classifier selection | PASS1.077s |
| Windows amd64 resource test binary with new native regression and final C1 source | COMPILE PASS, CGO0; no execution |

Two initial Go selectors named nonexistent `internal/transport` and
`internal/auth`; their commands failed setup while existing selected packages
passed. Corrected `control/apiauth/server` selection passed. An initial UI filter
matched no files; corrected lights/pages passed82tests. These command-selection
errors are retained in the index and are not product failures or green suites.
No generated-consumer compile failure occurred. No scenario Test Genie run or
live browser/native journey was launched under this source-only assignment.
Parent owns dependency/scenario-server receipt applicability and adoption.

## Changed, verified, remaining and unverified

Changed by reviewer: two desired-behavior cases in one new test file, manifests,
logs, this report and private checkpoints. Changed by implementation owner:
four source files for C1. Verified: the concrete Windows Go/API mismatch, actual
default private-writer reachability, bounded final correction review and scoped
source/fixture/build checks above. C remains finding-bearing; fresh independent
clean passes on the repaired final integrated source are still required.

MAIN was read only. Discovery/skill reads contacted existing shared search and
Prompt Manager services; recall reported degraded embedding, knowledge,
source-ledger and swarm providers, so its results were not treated as complete.
No shared service/runtime state, dependency, vendor executable, physical SDK device, account,
privilege, target, approval or grant was operated; no Git mutation or source
transfer occurred. No ongoing service/resource was started. Parent workflow owns
the shared work-record journal; this assignment did not write shared runtime.

The earlier isolated-generation incident remains disclosed in native source
provenance: three unintended shared proto publications were restored by the
canonical owner to the MAIN-matching selection; unknown consumer effects during
those windows remain unknown. This review neither republishes nor erases that
history.

Unverified: native Darwin CGO/link/runtime and derived signature/AMFI acceptance;
logged-out user-domain boot and vendor no-fork compatibility; Windows S4U/Task/
Job/token/image/listener/credential/filesystem behavior; Linux ARM runtime;
actual selected HID access, SDK/controller behavior, visible Off/static/effects,
reboot/logout/resume/device return and actual update/rollback; this host's
operator-owned lighting and radio qualification; complete delivery SPEC closure.
The source matrix must keep all such rows unqualified. Source repair and these
passing checks do not establish physical success or a full-destination challenger
NONE.
