# Integrated native lifecycle review B

Review date: 2026-11-11. Workspace: `/home/matthalloran8/Vrooli-native-lighting`.

This is an independent challenge of the combined adopted lifecycle, not another
launcher-only review. No material production counterexample was found. Two new
boundary regressions were added without changing production behavior. Native
runtime and hardware qualification remain open; this result does not close the
delivery SPEC or certify a physical controller.

## Applicability and boundaries

The reviewer reread the delivery charter, SPEC, feedback and current
`NATIVE-PORTABILITY.md`. Both `VROOLI_ROOT` and `VROOLI_SOURCE_ROOT` were explicitly
set to the isolated workspace for every Go verification reported below. Fixtures
use private controller/operator-state/home/storage roots. No vendor executable,
scenario, service, Task, Job, account, grant or device was operated. No dependency
installation, generation, Git mutation or main-checkout source write occurred.

The exact reviewed source identity is retained in
[source-manifest.json](review-native-b/source-manifest.json): 777 files, including
the shared platform module, resource and host device owners, runtime/supervisor,
early CLI dispatch, scenario API, persistence classifier, module metadata and
current source-copy/proto refresh receipts. Its SHA256 is
`1ae2a9a762bdaba748b8515906a7a6884e9436d0cf9a0515a70fce9efb51ea80`.
An end-of-pass comparison checks these recorded paths for changes. This is a
bounded source identity, not an assertion that every file in a concurrently
maintained repository was reviewed or that native code was executed.

The refreshed proto receipt is recorded as source provenance only. The parent
disclosed that its earlier generation omitted a private proto artifact root and
published a snapshot to the shared proto runtime store, then reported supported
rollback to the exact previous selection with MAIN verification before/after and
all5,206 outputs matching. This reviewer did not generate or change that store;
isolated test source provenance does not erase that separate operating-history
fact. After this clarification, further checks also set
`VROOLI_PROTO_ARTIFACT_ROOT` to the private isolated
`.vrooli/work/native-review-b/proto-artifacts` namespace; no generation was run.

## Counterexamples examined

| Boundary | Challenge and conclusion |
| --- | --- |
| Acquire and adopt | Original source identity remains distinct from derived serving identity. Managed/adopted paths verify the complete pinned envelope before deriving Windows PawnIO-free and Mac closed-loader trees. Adoption preserves the original; a derived tree cannot stand in for original source qualification. Tested managed/adopted recovery and preservation paths; no vendor execution. |
| Setup and policy publication | Pending setup recovery survives permission application until durable policy publication. Runtime preparation does not imply enablement. Stale catalog/profile/account selection fails rather than broadening detector scope. Source and setup regressions cover partial-failure and prior qualified policy publication. |
| External controller to real config | New regression persists an initial motherboard policy, runs the ordinary `writeManagedServiceConfig`, persists ROG AURA Terminal `0b05:1889:00`, then runs the same constructor again. Exactly Terminal is enabled in the complete pinned detector map; the old motherboard and every peer are disabled. Dynamic registration remains disabled. Rejected stale-catalog publication leaves the valid config byte-for-byte unchanged. This is actual persisted Linux policy/config routing, not native HID qualification; native tagged selection/freeze is separately checked by existing union fixtures. |
| Initial start and claim | Capsule preparation binds immutable material before root birth; the dynamic claim digest is independently supplied after exact ref lookup. Image, explicit invocation, selected generation/access, transport exclusion and exact wrapper incarnation precede the once-only durable claim. Windows suspended child identity is independently checked before resume. Darwin verifies inherited no-child proof before same-PID replacement. |
| Caller lifetime | Windows submission is a per-role Task action rather than an inherited child of an arbitrary CLI/API Job. The early wrapper establishes exact Task/Job custody before vendor or device work. Durable RunIssued and singleton task/action identity prevent uncertain submission from becoming replay permission. This pass reread the actual submission and capsule wiring; native Task/Job behavior remains unqualified. |
| Replacement after prepare | New public journey stops after durable role/fence preparation, before the modeled first Run, then creates a new controller and new operator-state service for the same private persisted namespace. An unacknowledged submission conservatively ends in one fresh admitted restoration, retains the uncertain primary intent unchanged and never replays it. The alternate disable journey revokes the prepared role with zero dispatches/claims. Both Darwin and Windows V2 records are exercised with explicitly modeled OS/SDK leaves. |
| Reconnect and boot-definition removal | Current consumed role ancestry, frozen launcher path and SHA select retained code; no historical version enumeration or global host reconfiguration is used. Exact revoked ancestry remains available for a separately admitted successor. NativeLauncherProof is optional on old records and is inserted only on newly prepared runtimes/specs, preserving old digest bytes. |
| Update, rollback and lost acknowledgements | Public durable state-machine journeys cover update/explicit rollback, target failure and restoration, interrupted readiness, lost start/publication acknowledgements, changed desired selection, and opt-out. Production dispatch reconstructs the exact admitted untouched role from the durable ledger and reconfirms its revision before effects; it does not depend on an ephemeral prepare/start map. |
| Disable and retirement | Durable Disabling precedes quiescence/cleanup. Cleanup continues across retained-role failures, and unknown native absence retains its fence. Completed or disabled roles cannot use their old intent as dispatch authority. Initial, update and restoration role absence remain distinct proofs. |
| Cold boot | Old claim provenance is preserved while exact current-boot native roots are separately captured, revoked, stopped and released. A fresh capsule has a fresh admitted invocation; old Task/capsule execution is not adopted as new execution. Source fixtures include another reboot during interrupted cleanup. |
| Serving listener and device identity | Sole literal `127.0.0.1` listener ownership uses the serving process, including the typed Windows child. Foreign, wildcard, IPv6/mapped, duplicate and ambiguous listeners fail. Native HID selection freezes complete matching routes and generation; transient paths or source/controller names alone do not establish access. |
| Compatibility and opt-out | Legacy Linux omission and opaque service-start fixture semantics remain intact. Native V2 records reject mixed containment/account/manager identities. Default-disabled paths do not acquire, prepare, probe, dispatch or create a native boot owner; explicit restart/update cannot bypass durable enablement or selection. |

## Added proof

- `internal/resources/openrgb_config_external_journey_test.go`:
  `TestOpenRGBExternalSelectionReplacesPriorDetectorThroughConfigWriter`.
- `internal/resources/openrgb_native_pre_dispatch_recovery_test.go`:
  `TestOpenRGBPublicNativeReplacementAfterPrepareBeforeFirstDispatch`, with
  Darwin/Windows restoration and disable cases.

The new replacement fixture initially expected target success after an unknown
submission. Source tracing and the actual recovery result showed that this would
be an incorrect oracle: absence is not permission to reissue that uncertain
invocation. The final assertion requires the existing conservative restoration
behavior and preserved primary identity. A separate initial fixture failure was
caused by forgetting its injected private operator-state root when constructing
the replacement; the replacement now reconstructs the same persisted private
owner. Neither iteration required or justified a production change.

## Verification

All commands below ran with both source-root variables set to the isolated root
and `-count=1`.

| Check | Observed result |
| --- | --- |
| Real external-selection config regression, strengthened to check every pinned catalog key and exact length; final race rerun with private proto artifact root | PASS, 1.346s |
| New public prepare-before-dispatch replacement/disable cases, race | PASS, 17.899s |
| Existing native public journey, capsule/claim transport, boot-retirement, retained launcher and namespace regressions, race | PASS, resources 64.900s |
| Complete `internal/lightingruntime`, race | PASS, 1.565s |
| Complete `packages/platform-go` module, ordinary tests | PASS, platform 5.672s; all subpackages pass |
| Complete platform module, race, excluding the real low-address-space shim test described below | PASS, platform 9.398s; all subpackages pass |
| Persistent codec/whole-source/stored-value classifier checks in `internal/buildinfo` | PASS, 0.841s |
| Full Linux CLI package compile (`go test ./cmd/vrooli -run '^$'`) | PASS, 0.011s; no tests executed |
| Broader config/acquire/adopt/setup/headless/public update and rollback race selection, including both new regressions | PASS, 108.153s |
| Native Terminal selection, consumed-access verifier, child handoff readiness and foreign/wildcard listener counterexamples, race, private proto artifact root | PASS, 2.541s |

The complete platform race command was **not green**: the existing
`rlimitexec.TestRlimitShimAppliesNprocAndAsBeforeExec` applies a real 1 GiB address
space ceiling to a race-instrumented helper, which exits66. A separate exact-test
reproduction with a private GORACE log proves ThreadSanitizer allocation failure
(`failed to allocate 131072 bytes of TracePart`, errno12), rather than a reported
data race. The ordinary real-limit test passes; the rest of the complete race
suite passes with that test excluded. The diagnostic is retained in
[rlimit-race.txt](review-native-b/rlimit-race.txt). This validation limitation is
not a claim that the complete race suite passed and was not repaired or hidden by
loosening the production limit. The platform checkpoint already records this
untouched limitation as `knw-1794401862033614384`; it is not a new lifecycle finding.

## Remaining qualification

No new material production finding is open from this review. The RLIMIT race
harness limitation remains explicitly reported. Native Darwin CGO compile/link,
launchd user-domain availability at logged-out boot, exact sandbox/AMFI behavior,
Windows S4U/Task/Job/security/native listener behavior, native credential access,
actual selected HID access, SDK availability, visible Off/static/effect output,
hardware return and actual reboot/update/rollback remain target qualification
obligations. Linux source and modeled native fixtures cannot satisfy them.
Scenario zone/render work remained with its concurrent owner; this pass made no
zone production edits and does not replace that owner's physical output proof.
