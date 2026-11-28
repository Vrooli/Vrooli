# Native retained-launcher combined maturation review

2026-11-11. Reviewer: `native_launcher_maturation`. Workspace:
`/home/matthalloran8/Vrooli-native-lighting`. This is a fresh independent review
of the current optional proof and its Darwin/Windows consumers after the prior
Mach-O closure and Windows reverse-projection repairs.

## Verdict and applicability

No material source finding was identified in this bounded pass. The inspected
implementation preserves the distinction between exact retained current-role
selection by the resource and machine-published executable admission by the
platform. The published collection does not become a historical-version chooser.

This verdict covers the current typed NEW same-compatible-host service submission
seam. It is a source/fixture maturation result, not full-destination acceptance.
Native Task Scheduler/S4U/Job/ACL behavior, macOS SDK/CGO/launchd/sandbox behavior,
headless operation and physical lighting remain unverified. The full delivery
CHARTER/SPEC remain the denominator; no full-goal closure or native qualification
is claimed.

The charter, SPEC, STATUS, FEEDBACK, native gap audit, docs/TESTING.md, current
`native_owned_launcher.md` and prior independent review were read. Search Hub
recall returned relevant retained-launcher records with an explicit
`source-ledger.scopes` timeout; no capability absence was inferred. The
writing-standards skill was read for report preparation.

## Source review

The reviewed flow starts at `OwnedServiceSpec.processSpec` and
`NewServiceIntent`. Proof is included in the canonical specification digest.
The optional JSON tag preserves old nil-proof service/operation encodings and
canonical digests. Mutating a pointed-to proof after intent creation changes
the digest and is rejected at validation; pointer identity is not authority.

Darwin durable decoding independently validates the proof-bearing specification,
account, intent digest, closed invocation shape and plist hash. The launch guard
uses the publication branch only for nonnil proof. It cannot fall back to the
installed boot-host branch after publication failure. The backend retains
admit-before-dispatch, one dispatch marker, uncertain-submission lookup and
revocation/retirement behavior. The publication branch does not change those
state transitions or create a root account pivot.

Windows Start validates the closed capsule shape, same operator security and
boot, then opens the exact proof-selected image before durable task reservation
and registration. `validateWindowsPayload` now preserves proof in its reverse
service projection. The strict JSON ledger decoder used by native lookup and
early bootstrap validates the complete frozen intent/specification. Early task
bootstrap independently reopens the proof-selected host, verifies its mapped
image and exact task action, and acquires named Job custody before owner IPC or
vendor execution. The action remains the waiting wrapper. There is no direct
caller CreateProcess fallback. Task Run remains create-only, on-demand,
least-privilege S4U under the same admitted operator.

The public opener checks profile/version/lowercase digest and exact canonical
collection/digest/executable path before native opening. Darwin checks complete
sealed ancestry and named/opened/final vnode equality, checksum and Mach-O
closure. Windows pins the approved ancestors, refuses reparse points and foreign
write ACLs, and holds the final image without write/delete sharing while its
checksum and PE imports are parsed. Common return handling rewinds the verified
file and closes it for cancellation or failed admission. Machine root remains a
trusted publisher, as required by the accepted threat model.

The Mach-O metadata reader bounds the explicit reader extent, fat table,
nonoverlapping architecture slices, command count/bytes, individual forms and
strings. Maintained-host policy is applied separately to every slice, including
nonselected slices. It requires native executable architecture and system
dependencies/interpreter/rpaths, refuses embedded dyld environment and unknown
load forms, and does not reuse the vendor Qt dependency policy. The PE reader
uses held-file raw-backed RVA mappings, validates every descriptor/name until
the bounded terminator, and refuses foreign ABI, DLL images, unqualified DLL
names, delay imports and CLR entry. It does not use Go's unimplemented
ImportedLibraries method.

Resource integration was inspected in `openrgb_native_images.go`,
`managed_update_prepare_launch.go` and `managed_update_launch.go`: current role
selection requires the retained native reference, consumed claim, exact account,
resource namespace and frozen selected-device binding before returning its
launcher. NEW successors receive the optional profile; old stored runtime specs
keep their nil-proof digest. Parent-owned resource integration was still active
during this review. This pass does not substitute for that owner's complete
initial/update/restart/cleanup journey proof, and expected in-flight work was not
classified as a defect.

The implementation uses the existing publisher collections and owned lifecycle.
It adds no mutable launcher registry, historical selector or general executable
entrypoint. The shared Mach-O metadata parser serves both consumers without
sharing their distinct trust policies. Portable injection seams remain private;
native production constructors supply fixed path/ACL/image effects. No material
avoidable duplication or new responsibility split was found in this seam.

## Fresh independent desired-behavior probes

The retained overlay fixture is independent of the producer's image builders.
It constructs raw Mach-O and PE bytes directly and tests these expectations:

- Given a universal image whose second slice has a foreign dependency, admission
  rejects it for either selected architecture and all five dylib load forms.
  Replacing only that dependency with a canonical system dependency admits both
  architectures. This checks that nonselected slices are not ignored.
- Given dependency paths with system-like prefixes, parent traversal, repeated
  separators or `@rpath`, maintained-host admission rejects all five load forms.
- Given a PE import table with both a qualified and a foreign dependency,
  admission rejects either ordering, including path, alternate-stream and
  whitespace forms. Two qualified imports remain accepted.
- Given a native opener that returns a descriptor and an error, the public
  boundary preserves the failure and closes the partial descriptor.

Reproduce from `packages/platform-go`:

```sh
go test -race -count=1 -overlay docs/native-launcher-maturation/overlay.json -run TestMaturation -v .
```

All four independent tests passed (`native-launcher-maturation/independent.log`,
race 1.016s). A combined affected regression selection passed 52 top-level tests,
134 test entries, race 1.064s (`combined.log`). The retained complete log records
the actual selected names, including proof/legacy encoding, Darwin launch and
Mach-O closure, Windows launcher/payload/task/submission/custody coverage.

The combined command was:

```sh
go test -race -count=1 -overlay docs/native-launcher-maturation/overlay.json -run 'Test(Maturation|NativeLauncher|PublishedNativeLauncher|Darwin.*(Launch|MachO)|Windows.*(Launcher|Payload|Task|Submission|Custody))' -v .
```

Reviewed production source identities are retained in
`native-launcher-maturation/reviewed-source.sha256`. They record observed bytes;
they do not prove a mutable shared workspace was frozen throughout execution.
No relevant source mutation was observed during this pass. Existing prior
finding-bearing passes remain finding-bearing; this clean bounded review does
not relabel them or count as a full-destination clean pass.

## Changed, verified, remaining and unverified

Changed: this report and independent fixtures/overlay/logs/source hashes under
`packages/platform-go/docs` only. No production edit, Git operation, dependency
operation, root/service/account/token/native task/Job/device effect or dispatch
was performed. No service or model resource was started; test processes exited.

Verified: current proof dataflow and source invariants described above, fresh
independent multi-slice/multi-dependency/path/cleanup probes, and selected affected
portable race regressions. No new material source finding was identified.

Remaining: parent-owned complete resource retained-current-role integration and
fresh full-destination adversarial maturation after the required operated proof.

Unverified: real native image loader/ACL/account/Task/S4U/Job/launchd/SDK/CGO and
sandbox behavior; actual update/restart after boot-definition removal; logged-out
boot/resume/reconnect and controller/physical lighting. Passing portable fixtures
and compilation cannot establish those outcomes.
