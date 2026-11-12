# Retained maintained launcher authority

An exact current OpenRGB role can retain its published maintained host after the
boot definition is removed. Its owner already freezes Launcher, launcher SHA256,
native receipt, consumed execution fence and device policy. A new update or
restart can reuse that exact compatible image without selecting another code
version. Durable lighting disable remains the domain owner's launch interlock.

`NativeLauncherProof` identifies version1 of the closed
`openrgb-maintained-launcher-v1` publication profile and its exact SHA256.
`OwnedServiceSpec.NativeLauncher` and `OperationSpec.NativeLauncher` are optional
and participate in the resolved specification digest. Existing stored runtimes
must retain nil proof; adding a proof to an old specification changes its digest.
The resource owner sets its profile only on newly prepared successor runtimes.

The profile accepts these existing machine-published paths:

| Platform | Exact publication |
| --- | --- |
| Darwin | `/Library/PrivilegedHelperTools/com.vrooli.openrgb/launchers/<sha256>/vrooli` |
| Windows | Known Program Files directory + `Vrooli/OpenRGB/launchers/<sha256>/vrooli.exe` |

Explicit native setup publishes approved maintained host bytes into these
content-addressed directories and seals the complete ancestry. The canonical
publisher namespace supplies durable code authority after an old operation's
ledger is retired. Alternate artifact roots remain fixture/research locations.

`OpenPublishedNativeLauncher(ctx, path, proof)` verifies that exact path, machine
owner/ACL and immutable ancestry, aliases, file checksum, native architecture and
loader closure. It returns the verified file at offset0 and closes it on failed
or canceled admission. `NativeLauncherPublisherRoot(profile)` returns the
launcher collection directory and creates nothing. Selecting which role's image
may be reused belongs to the resource: callers must verify the exact retained
current-role provenance or fresh installed publication before supplying proof.
The getter does not enumerate historical images or establish device permissions.

Both native starts retain the closed maintained capsule entrypoint contract.
Darwin's launch guard and Windows start/task bootstrap independently consume the
same frozen proof. A nonnil proof has no installed-boot lookup fallback; nil
retains the preexisting installed-host approval branch. The process account,
once-only dispatch, native birth/Job identity, pre-vendor witness/claim and exact
retirement checks remain mandatory. Linux rejects this native publication mode;
omitted proof preserves the original Linux JSON bytes and specification digest.

The Windows verifier reads real PE import descriptors from the same held file as
the checksum, including strict RVA/name/bounds checks and refusal of unsupported
delay-load/managed entry forms. Go's `debug/pe.ImportedLibraries` stub is not
closure evidence. The Darwin verifier reads thin/universal Mach-O loader metadata
including weak, reexport, upward, lazy and modern dylib-use forms. The pure
`ReadMachOLoaderMetadata` API reports dependency/interpreter/rpath/environment
facts; each consumer owns policy. The maintained host requires the qualified
system-loader closure. Vendor Qt/framework policy belongs to the artifact owner.

Publication authority does not guarantee an arbitrary older host understands a
new capsule protocol or selected target. The owner must retain a compatible host
or report the concrete preparation dependency. Source fixtures and crossbuilds
do not qualify native loader execution, Windows task/S4U/Job behavior, Apple
SDK/CGO/sandbox behavior, devices or physical lighting. Those proofs remain open.
