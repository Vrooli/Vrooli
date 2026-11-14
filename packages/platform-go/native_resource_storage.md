# Enrolled native resource storage

The existing protected native boot descriptor optionally stores
`GlobalResourceStorageRoot`. It freezes the whole `VROOLI_RESOURCE_STORAGE_ROOT`
override selected during explicit initial native setup. The path is canonical,
absolute, bounded, and contains no control separators; empty means the maintained
default. The field is omitted when empty and does not add a descriptor version,
registry, per-class path mapping, or migration framework.

`EnrolledNativeOperatorResourceStorageRoot(ctx, home)` returns the frozen value
and whether enrollment exists. An enrolled default is `("", true, nil)`; no fixed
native boot unit/task AND no retained protected ownership evidence is
`("", false, nil)`. Boot removal alone does not erase namespace ownership.
The accessor validates protected descriptor authority and the selected account/home.
Darwin permits root publication or the actual selected UID. Windows requires the
actual selected effective-token SID, including its same-account elevated token.
When a native definition exists it must remain exact and its executable keeps
full image proof. When absent, Darwin reads its existing protected journal and
Windows reads bounded existing immutable descriptors. A missing/corrupt retained
descriptor, denied query, foreign account/home, or changed observation is an error.
Readonly namespace fallback does not hash or require old executable bytes; it
checks protected descriptor authority and canonical options/path binding. Full
Image proof remains mandatory for executable-opening/boot APIs. Windows initial
native-task query and downstream plan validation are separate so their missing
codes cannot be confused.

The manager applies this frozen value only inside the native adapter's verified
actual-operator callback. It sets a custom override or removes an inherited
ambient override for an enrolled default. The elevated setup process does not
receive a forged operator HOME or a process-wide shadow resource environment.

The resource owner validates current storage against enrollment. High-level
lighting actions call this validator before intent writes, scenario quiescence,
permission changes, or cleanup. Supervisor installation/removal also checks it.
Switching root A to B cannot falsely disable a still-running A backend by seeing
empty B state. The error instructs restoring the enrolled value, or unsetting the
override for an enrolled default. Initial preparation may select its root before
enrollment; moving an existing enrollment is not implemented implicitly.

Source verification on 2026-11-11: full platform module normal tests PASS4.467s;
focused storage/enrollment race fixtures PASS1.156s; high-level lifecycle/source
race fixtures PASS (app/runtime1.480s, cmd/vrooli1.039s). Fixtures cover absent,
default/custom, malformed/foreign/query failure, native unit/task replacement,
existing task with missing descriptor, frozen bootstrap env, and A→B refusal
without intent/quiesce/backend changes followed by successful restored-A cleanup.
Native CLI crossbuilds pass Windows amd64/arm64 and Darwin amd64/arm64 (CGO0).
Platform test binaries crosscompile Windows arm64 and Darwin amd64. Offline Go
modules and -buildvcs=false were used; all OS effects are injected.

These results do not qualify real native boot, logon, definition ACLs, device
permissions, physical lighting, or runtime recovery. No services, accounts,
tokens, logons, devices, or host configuration were operated in this source lane.

Post-uninstall recovery uses no new pointer or registry. Darwin's root-owned
existing journal remains0444, its metadata-only directory0755, and lock0600.
Windows enumeration is bounded to128 version directories,512 aggregate files,
256 descriptors and16MiB aggregate descriptor bytes; every candidate must agree
on actual operator/home and namespace. Unknown names, malformed/foreign or
inaccessible candidates, reparse objects, and conflicting roots report unknown.
Exact root-protected `.stage-<32 lowercase hex>` publisher residue is validated
as unpublished metadata and never counted as ownership authority. Native task
absence and descriptor lists are rechecked after observation.

Initial elevation instructions explicitly transfer only the selected global
resource root: a quoted sudo assignment on Darwin, a same-account elevated
PowerShell Process-environment assignment with restoration on Windows. Empty
means an explicit default, not inheritance from an elevated shell. No general
environment preservation is introduced. Readonly acquired-artifact inspection
precedes boot publication, so a wrong account or missing acquisition cannot
publish a boot descriptor before discovering that failure.

Retained-recovery checkpoint: full platform module PASS4.919s; focused retained
namespace/residue race fixtures PASS1.216s; high-level native lighting/elevation
race fixtures PASS (see /tmp/native-retained-highlevel-final.log). Fresh Windows
amd64/arm64 and Darwin amd64/arm64 CGO0 CLI builds pass; Windows arm64 platform
test binary also crosscompiles. Uninstall/image-cleanup/duplicate-pointer proofs
use injected native effects and temporary fixture files, not live native units.
The explicit elevation rendering tests cover custom/default root and quoted path
characters; missing acquired bytes are rejected before the installation callback.
Native runtime and hardware qualification remain unverified.
