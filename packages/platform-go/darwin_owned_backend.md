# Darwin operator jobs

This native lane runs the ordinary RuntimeSupervisor as the existing operator.
Root setup publishes one immutable executable/descriptor and one system
LaunchDaemon with `UserName` set to that operator. It does not create an OpenRGB
account, run an enduring root broker, write operator state or create root-owned
mutable logs. The supervisor uses the operator's home and ordinary lifecycle.

This is a closed maintained OpenRGB capsule capability. Admission requires one
fixed capsule command and a bounded canonical absolute capsule.json path matching
launch/AttemptID, empty environment, launcher working directory and a
resource-initial/resource-update intent. The maintained early reader binds custom
resource roots to the frozen namespace/account/intent before SDK execution. The
guard runs before persistence or manager effects. Arbitrary native executable
services and extensible registration are refused.

Legacy specs with no NativeLauncher proof retain their installed-supervisor
approval branch: identical separately staged bytes are valid. New successor
specs can carry a digest-bound, versioned maintained-launcher proof. That branch
opens only the exact root-published
`/Library/PrivilegedHelperTools/com.vrooli.openrgb/launchers/<sha>/vrooli` image and
has no boot-definition, manager, setup-journal or newest-pointer dependency.
It requires root/ACL-sealed ancestors, immutable executable mode, the exact FD
SHA256, Mach-O architecture/system-loader closure and stable pathname/FD identity
before and after observation. Public proof/path validation is shared with the
other native owner. No-CGO ACL proof remains explicitly unsupported.

Resource policy derives this new proof only from an exact retained current-role
fence/ref/policy or a fresh approved installed publication. This is not an
arbitrary historical-version selector or new registration service. Old runtime
specs remain nil and retain their exact digest. Removing a boot definition alone
does not prevent a newly admitted successor from using the exact retained
published current host; the resource's owner and frozen admission still govern
whether that successor may run.

Each owned role has one deterministic, invocation-specific launchd label in
`user/<UID>`, independent of the GUI domain. Intent, frozen spec, origin boot and
plist digest are durable before bootstrap. Bootstrap occurs once; start waits
for a positive kernel root birth under the caller deadline and a 30-second
maximum. Timeout remains uncertain with durable intent available to lookup. Plists and records are 0600 under
`<operator-home>/.vrooli/native-owned-launchd` (0700). No user LaunchAgents folder
or desktop login is needed by this source design. Current installed macOS SDK
and before-any-login execution remain required native qualification.

The ownership record and native identity are version 2. A native identity binds
user domain, exact label, invocation, actual `uid:<UID>`, boot UUID, kernel unique
birth and process start time. The changing kernel exec version protects each
observation; it is not frozen into a receipt that must survive intentional
same-PID exec. The never-operated version-1 dedicated-account candidate is
removed rather than given fictional migration compatibility. Linux persisted
identity bytes are outside this change.

## First-boot domain readiness

A missing user domain is not absence proof. If the initial typed query is
unknown, the owner first persists the complete intent and private plist, then
runs the documented domain-only `launchctl bootstrap user/<UID>` operation.
A fresh typed query must prove exact-label absence before role bootstrap. A
lost/already-existing domain response is acceptable only with that new query;
a failed query retains unknown state and causes no role dispatch. A pre-existing
label is refused and only the never-dispatched local files are removed.

The record's DispatchPrepared phase becomes durable immediately before one
exact role submission. A domain-only intent cannot adopt a later manager root.
This phase describes prepared dispatch, not proof that launchctl executed or
that a vendor process is healthy; uncertain submission is recovered by lookup.

Historical Apple launchd source creates a per-user context on lookup and pins
non-root callers to their own UID. Current ServiceManagement internals are not
open, so this design does not assume its dictionary query creates the domain.
The explicit command route still needs current SDK and before-login proof.
[Apple per-user context implementation](https://github.com/apple-oss-distributions/launchd/blob/main/src/core.c),
[Xcode launchctl manual](https://keith.github.io/xcode-man-pages/launchctl.1.html).

Native acceptance case (open): Given a fresh boot with no operator GUI or SSH
login and an enabled lighting policy, when the operator LaunchDaemon reconciles
its first role, then the actual operator user domain becomes ready, the typed
query proves label absence, one role is submitted, the exact unique birth is
claimed, and OpenRGB reaches supported HID/SDK readiness. Record the actual
SDK/OS/image identities, user/session evidence and native operation receipts.
Fixture/bootstrap success alone does not satisfy this case.

## Manager and process observation

Production uses ServiceManagement's typed job-dictionary API, not `launchctl
print` diagnostics. `SMCopyAllJobDictionaries` returning NULL is a query error;
absence means no exact label in a valid bounded array. Duplicate labels,
unexpected types and missing required Label/ProgramArguments are unknown.
The frozen ProgramArguments must match. Apple does not export all configured
plist fields: credentials, mapped image, argv/environment and working directory
are separately proven by native process observers at the boot/resource owners.
`LastExitStatus` never becomes an exact-birth exit receipt.

The API is deprecated and its dictionary contents are not a stable on-disk
schema. This lane requires the exact typed facts it consumes and fails closed
when they are unavailable. It has no print parser fallback. The current SDK,
linkage, actual service-dictionary shape and OS floor remain qualification work.
See Apple's [ServiceManagement API](https://developer.apple.com/documentation/servicemanagement/smcopyalljobdictionaries(_:))
and [launchd job export source](https://github.com/apple-oss-distributions/launchd/blob/main/src/core.c).

The process observer calls `proc_pidinfo` flavor 17 with the exact pinned 56-byte
XNU identity layout and requires a full reply and nonzero unique ID/version.
That flavor is private, with no SDK availability guarantee. Unsupported replies
are unknown. Two atomic unique-ID/version replies bracket the kinfo birth
observation; the combined stamp brackets image mapping, argv, credentials,
working directory and listeners. Birth alone cannot detect an intervening exec.
Pinned [XNU layout](https://github.com/apple-oss-distributions/xnu/blob/f6217f891ac0bb64f3d375211650a4c1ff8ca1ea/bsd/sys/proc_info_private.h).
No-CGO builds retain explicit unsupported native observations.

Private ledger admission checks both POSIX modes and fd-bound extended ACLs on
the record, plist and complete ancestor chain. Only the actual operator and root
may receive ACL mutation rights; foreign user/group and inheritance grants are
refused. The root immutable-image policy still permits only root mutation. ACL
inspection binds the opened inode and refuses metadata changes during the
observation. No-CGO ACL observations are explicitly unsupported.

## Body, stop and retirement

This lane supports a trusted, single-process final body. OpenRGB's existing
transport facet is the one inherited sandbox application: it denies unsupported
macUSPCIO access and adds process-fork denial after image
preparation and before the access witness. The final after-claim boundary and
serving observation independently call `VerifyDarwinNoChildrenProcess`.
The readonly verifier requires native `sandbox_check` denial replies while
bracketing process identity. It never installs a second sandbox. It makes no
owner-home or private-state isolation claim: the pinned vendor is trusted under
the same operator baseline as Linux.

XNU applies its MAC fork check to fork/spawn creation. The closed maintained
capsule and pinned OpenRGB source have no native-job submission caller. This
lane does not assert containment of hypothetical delegated OS job creation or
add speculative XPC restrictions. Current target fork/spawn negative tests and
OpenRGB compatibility remain mandatory.
[Apple process creation](https://github.com/apple-oss-distributions/xnu/blob/f6217f891ac0bb64f3d375211650a4c1ff8ca1ea/bsd/kern/kern_fork.c).

Stop revokes durable activation first and asks launchd to boot out only the exact
role label. It never signals a sampled PID, process group or all operator-owned
processes. Quiescence requires manager absence and extinction of the exact
claimed birth. An unassociated surviving birth is unknown/not quiescent; there
is no numeric-kill fallback. Natural exit remains outcome unknown and can retire.
A revoked-but-running capsule cannot reacquire its frozen execution environment.
A prior boot cannot authorize a current root. Release deletes the ledger only
after those conditions; unknown manager observations never become absence.

No account-wide NPROC limit is installed. `native-advisory` memory/nice policy is
not an aggregate workload quota. General descendant-spawning native workloads
need a different qualified body contract; a launchd label alone does not provide
recursive containment for them.

## Evidence limits

Hermetic tests exercise predispatch durability, uncertain bootstrap/bootout,
PID reuse, exec transitions, natural exit, revocation, old boot, exact stop,
retirement, private file modes, absence, boot immutable publication/recovery and
operator runtime identity. Darwin amd64/arm64 no-CGO test builds prove compilation
only. Native CGO SDK compile/link, boot before login, logout, process negatives,
owner crash, real OpenRGB/HID output and reconnect/resume remain unverified.

## Shared Mach-O loader metadata

`ReadMachOLoaderMetadata(reader, size)` reads a bounded held-reader extent and
returns loader facts for each thin or universal slice. It handles 64-bit
executables, dylibs and bundles; reports load/weak/reexport/upward/lazy dylib
commands (including known dylib-use modifier flags), the dynamic linker,
RPATHs, embedded dyld environment and dylib install
identity. It does not apply dependency-root policy. Vendor bundle owners use
these facts to qualify their own sealed Qt/framework closure.

The maintained-host policy separately requires executable slices, finite
system-library paths, `/usr/lib/dyld`, system RPATHs and no embedded dyld
environment. Every universal slice is checked. Malformed command sizes, offsets,
strings, counts, architecture tables and overlaps are rejected. Unsupported
load-affecting forms and unknown commands return an error; they are never
silently omitted as proof. The parser bounds images to 512 MiB and each slice's
command table to 4 MiB/4096 commands. Known non-path command layouts are explicit.
See [Apple's Mach-O command definitions](https://github.com/apple-oss-distributions/cctools/blob/main/include/mach-o/loader.h).

This replaces the incomplete `debug/macho.ImportedLibraries` convenience method,
which omitted several dependency forms. Static parsing and successful
cross-compilation do not establish native SDK linking or execution readiness.
