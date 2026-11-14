# Native retained-launcher fresh review C

Reviewed 2026-11-11 in `/home/matthalloran8/Vrooli-native-lighting` by
`native_launcher_fresh_c`. This is the next independent full source/fixture pass
after `NATIVE-LAUNCHER-MATURATION.md`.

## Verdict

No new material source issue was identified in the retained maintained-launcher
seam. This pass can count as a second consecutive clean **component source**
review after the prior repaired findings and the clean maturation review. It
does not count as acceptance of the complete lighting destination or native
runtime qualification. The parent owns the count and integration boundary.

The delivery CHARTER, SPEC, STATUS, FEEDBACK, native gap audit, docs/TESTING.md,
`native_owned_launcher.md`, current producer source and previous review were
read. Search Hub recall returned relevant prior findings and implementations.
The writing-standards skill was read for the report. The accepted destination,
including logged-out operation, actual devices and physical output, remains
unchanged.

## Independent source assessment

The proof remains part of `OwnedServiceSpec.processSpec`, canonical operation
digest, durable Darwin record and Windows payload. A proof pointer is data,
not retained authority: later mutation breaks the intent digest. Both strict
durable readers validate the resolved specification. Optional omitted fields
preserve old nil-proof encodings and digests. Resource runtime projection adds
the profile only to NEW native successors; the old runtime is not upgraded in
place. The closed account vocabulary and actual native account/security checks
remain separate requirements.

Darwin validates its closed capsule shape and proof-selected image before
durable admission or native dispatch. The persisted record binds the exact
intent, account/UID/GID, boot, specification and generated plist. The consumed
dispatch marker, lookup after uncertain submission, revocation before removal,
root-incarnation observation and exact retirement are not bypassed by retained
publication authority. A nonnil proof has no fallback to installed-boot image
selection. After old ledger removal, a new compatible invocation can still
admit the same sealed publication without reviving the old invocation.

Windows Start and early task bootstrap both call the same fixed proof-selected
host opener. The registry reader validates the complete proof-bearing frozen
payload and exact intent before bootstrap checks revocation, issued marker,
boot, canonical same operator, actual argv/cwd, mapped image and exact task
action. The wrapper obtains and retains its own named Job custody before
vendor execution. The Job is not inherited from the submitting caller, and
the vendor receives its image handle rather than the Job handle. The task is
on demand, S4U and least privilege; no relay or general executable registration
was introduced. Native Task Scheduler/Job behavior was inspected as source
only and was not executed.

The public publication boundary admits only its version/profile/lowercase
SHA-256 vocabulary and the exact canonical publisher/digest/executable path.
It checks cancellation, rejects alternate paths before native opening, closes
partial failures, and returns only a rewound descriptor. Darwin checks sealed
ancestry, root metadata/ACL, alias restrictions, initial/open/final inode
identity and checksum/ABI/loader closure from the held descriptor. Windows
pins root-approved ancestry and opens the final non-reparse image without
write/delete sharing while checksum and PE imports consume that held image.
The trusted machine publisher and same operator remain the accepted threat
model; this review did not invent a malicious-root or malicious-owner model.

Mach-O parsing bounds the explicit reader extent, universal table and slices,
load-command counts/bytes, command-specific lengths and strings. It reports
all five dylib import forms, modern dylib-use flags, interpreter, rpaths and
embedded environment. Maintained system closure is checked on every slice;
vendor Qt policy is separate. PE closure uses real raw-backed bounded RVA
and import-descriptor/name parsing, validates every dependency through its
terminator, and refuses unsupported ABI, delay-import and CLR forms. Unknown
or malformed inputs produce errors. No new parser omission was found.

Resource selection was read at `openrgb_native_images.go`,
`managed_update_prepare_launch.go`, `managed_update_launch.go`,
`openrgb_initial_exec.go` and publication staging. It requires the exact
retained claim/reference, consumed fence, account, state/home namespace and
frozen selected-device policy before selecting that one current role's
launcher. Publication does not enumerate history or select a newer host.
Metadata observation after uninstall is distinct from executable admission;
execution still requires the full image proof. The parent is concurrently
finishing the resource journey; expected pending work was not reported as a
defect. No source-level responsibility duplication requiring refactoring was
identified in this bounded capability.

## Fresh desired-behavior evidence

The retained overlay fixture adds three independent probes:

- Given a raw independently built Mach-O and removed installed boot authority,
  an uncertain Darwin submission retains the exact proof and consumed dispatch
  marker. Lookup recovers its native birth, replay cannot dispatch again,
  stop persists revocation, and release removes the old ledger. The same
  publication remains admissible after retirement, with zero installed-host
  lookups. The fixture uses the production Darwin file store and lifecycle,
  real held-file checksum/Mach-O parser, and a simulated native manager/path
  seal; it does not establish launchd or ACL behavior.
- Given a valid proof-bearing payload, both actual production durable decoders
  accept it and reject omission or substitution of profile, version, checksum,
  account and entrypoint without a replacement frozen intent. The Darwin
  probe recomputes the plist hash, so rejection cannot rely on an accidentally
  stale plist hash alone. The Windows probe uses the strict registry codec,
  rather than retired base64 transport.
- Given a returned descriptor that cannot be rewound, the common boundary
  returns an error and closes it, including this final admission failure path.

Reproduce from `packages/platform-go`:

```sh
go test -race -count=1 -overlay docs/native-launcher-fresh-c/overlay.json -run TestFreshC -v .
go test -race -count=1 -overlay docs/native-launcher-fresh-c/overlay.json -run 'Test(FreshC|NativeLauncher|PublishedNativeLauncher|Darwin.*(Published|Retirement|Uncertain|Launch|MachO)|Windows.*(Launcher|Payload|Task|Submission|Custody))' -v .
```

Both commands exited 0. The three fresh tests passed under race in 1.113s
(`native-launcher-fresh-c/independent.log`). The combined affected selection
passed 56 top-level tests, 152 pass entries, in 1.596s
(`native-launcher-fresh-c/combined.log`). Full logs retain actual names and
subtests. No test skipped. No opt-in native journey, live Task/Job, service,
device, SDK or target-dispatch test was selected.

`native-launcher-fresh-c/reviewed-source.sha256` records observed production
bytes for the 29 reviewed implementation files. It is an observation manifest,
not proof that the concurrent workspace was immutable during the review.
No production edit was made by this reviewer. No protobuf consumer or generated
artifact failure occurred; no regeneration or dependency operation was needed.

## Changed, verified, remaining and unverified

Changed: this report and retained independent fixture, overlay, logs and hashes
under `packages/platform-go/docs` only. No main-checkout change, Git operation,
dependency change, root/elevation/account/token/service/native task/Job/device
effect or target dispatch. All test processes exited. No long-lived resource,
service or model was started.

Verified: the source/dataflow/lifecycle boundaries described above, three fresh
counterexample groups, and focused portable race regressions. No material source
finding remains from this pass. Prior finding-bearing passes remain so.

Remaining: parent-owned complete retained-current-role resource journey and
integration acceptance, plus full-destination maturation after operated proof.
The parent should include this bounded outcome in its required completion work
record; this reviewer was authorized to write documentation/fixtures only.

Unverified: native Darwin SDK/CGO compile/link and launchd/dyld/sandbox behavior;
Windows real Task/S4U/Job/ACL/loader behavior; actual boot-definition removal and
successor launch on either OS; headless startup/logout/boot/resume/reconnect;
selected-controller access, SDK behavior and physical lighting. Component source
cleanliness does not establish any of these facts or full CHARTER/SPEC closure.
