# Independent native storage/device review A

Review scope: isolated `/home/matthalloran8/Vrooli-native-lighting`, 2026-11-11. Read the founding CHARTER, SPEC, STATUS, FEEDBACK, native gap audit, NATIVE-PORTABILITY and NATIVE-DEVICE-PROGRESS. No product repair, Git mutation, host/root/device/service operation, target dispatch, dependency acquisition or credential effect was performed by this reviewer. Discovery returned degraded semantic/skill/record providers; direct charter and source reads establish this report's scope.

## Decision

**Do not count this as a clean adversarial closure pass.** This review found material storage boundary defects while implementation was active. The implementation owner repaired three during review. One directory-creation provenance race remains to resolve or disprove. Actual native compilation/runtime/HID and lifecycle acceptance remain unqualified.

## Findings and dispositions

1. **Darwin privileged ownership/ACL path substitution — addressed in source during review.** Original `ConfigurePrivateDirectoryAccess` checked path metadata and then used path-based `os.Chown`/`acl_set_file`. A substitution could redirect mutation after inspection. Latest source uses no-follow component traversal (`privatepath_darwin_handles.go`) and descriptor `Chown`/`acl_set_fd_np`. Windows setup similarly pins ancestors, rejects reparse points and mutates security through the identity-checked handle. This is a source correction, not executed native race proof.

2. **Read-only runtime accepted as actual metadata owner — addressed in source during review.** Both native verifiers originally accepted runtime ownership even under RuntimeReadOnly. The owning runtime could change mode/DACL and defeat the nominal read-only grant. Latest Darwin and Windows verifiers explicitly reject runtime-owned read-only/traverse metadata. The separate descriptor check also binds its asserted management owner to actual OS ownership; runtime identity resolves to the provisioned account. Native adversarial ACL fixtures are still needed.

3. **Privileged temporary-file owner assignment reopened a different inode — addressed in source during review.** Original writer retained the real temporary descriptor but assigned ownership by independently reopening `tmp.Name()`. Directory/regular-file substitution could chown another object before a later publication check detected mismatch. Latest writer passes the original `*os.File`: Darwin uses `file.Chown`; Windows binds its pinned reopen to the original descriptor's `Stat` identity before owner mutation. No claim of native runtime qualification follows.

4. **Created-directory provenance is lost between Mkdirat and Openat — OPEN, medium correctness/security-contract concern.** `internal/config/privatepath_darwin_handles.go:35` records successful Mkdirat, then independently opens the name at line43. A writer of the parent can rename the created directory and replace it with an existing ordinary directory between those calls. O_NOFOLLOW rejects links but accepts this replacement. `ensureNativePrivateDirectory` treats the returned `createdFinal=true` as authority to replace inherited ACLs before verifying the returned object. Thus the claimed rule “only a newly created, still empty directory may replace inherited grants” is not established for the object receiving the ACL. This does not establish an arbitrary-root-file escalation; it establishes an unproven creation/normalization invariant in privileged setup. Bind the created object/provenance before normalization, or refuse unknown existing grants without relying on that boolean. Add a deterministic native substitution regression. Routed directly to native_device_policy.

## Positive boundary observations

The native policy account is stable and exact, separate from the reviewing operator. Metadata selection does not claim daemon account access; the final witness requires access under the actual runtime principal and re-observes context/generation. The finite ASUS HID scope refuses extra current matching attachments before access and rejects missing/ambiguous generation. Windows uses effective token identity and fails elevated tokens; Darwin refuses root HID access. The source does not install a privileged vendor fallback. These observations do not prove pinned upstream HID ABI applicability or native account access.

The owner descriptor verifies actual file ownership, strict JSON, provisioned runtime account identity and exact resource subtree. Root/launch metadata receives runtime read access, state remains management-owned, vendor runtime is a distinct subtree, and parent namespace creation avoids widening HOME. Current source has an explicit setup method, but `rg PrepareOpenRGBNativeDeviceSetup --glob '*.go'` found its declaration only at review time. Normal governed setup routing must be integrated and tested by the lifecycle owner; active missing integration is an outstanding delivery item, not a new closed-pass claim.

## Verification and limits

Executed in the isolated checkout:

- `go test ./internal/config ./internal/hostreqkit -run 'TestPrivateWriter|TestNativeHIDPolicy' -count=1`: PASS, config 0.019s; hostreqkit 0.003s.
- An earlier pattern accidentally selected no hostreqkit tests; it is not evidence. The corrected command above selected the four NativeHIDPolicy fixtures.
- Read current Darwin C ACL/IOKit and Windows security/SetupAPI/HID calls, selector/account validation, owner descriptor, native setup and private writer source. Source changed concurrently; this report records the latest reread dispositions rather than attributing fixes to this reviewer.

Linux generic writer fixtures and injected policy fixtures do not execute Darwin CGO ACL/IOKit or Windows native calls. No macOS SDK compile/link, Windows execution, actual effective-account rights, HID handles, ACL durability, native crash/recovery, normal setup/disable/update/rollback, logged-out boot/resume or visible light output was verified. Broader source integration and physical acceptance remain the parent owner's denominator.

Changed: this report only. Verified: bounded source properties and the listed hermetic Linux tests. Remaining: open finding4, independently rerun affected native negative fixtures after repair, integrate normal setup route and finish full lifecycle acceptance. Unverified: all actual native runtime/hardware behavior. No service/process/model was started by this review.
