# Independent native artifact review B

2026-11-11. Reviewer: `/root/native_lighting_portability/artifact_native_review_b`. Scope: the isolated `Vrooli-native-lighting` native artifact/acquisition/adoption/staging, private configuration, MSVC prerequisite and tool-schema source. Read the main delivery CHARTER, SPEC, STATUS and FEEDBACK, native gap audit, native release profiles, artifact checkpoint and review A. Review A was evidence, not the threat model for this pass.

## Result

**Two material source findings require repair and independent recheck.** The artifact owner acknowledged both and is implementing the fixes. This is not a finding-free pass. The reviewed profile, retained-byte, adoption and private-config fixtures pass; these do not establish native runtime or full delivery acceptance.

### B1 — Darwin artifact sealing ignores extended ACL authority

At review time, `openrgb_appimage_stage_posix.go` checked ancestor ownership and mode bits in `verifyOpenRGBAppDirAncestry` (line 29), and checked artifact owner, nonwritable mode, type and link count in `verifyOpenRGBAppDirEntries` (line 48). `sealOpenRGBStage` in `openrgb_appimage_prepare_posix.go` changed modes and flushed files. None inspected or removed Darwin extended ACL entries on the sealed files/directories or ancestry. Darwin's `verifyOpenRGBFileCapabilities` is intentionally a no-op; it supplies no ACL check.

Counterexample: retain the exact payload/tree bytes and recorded operator UID, with mode 0555, but give a different local principal an extended allow ACL containing write-data on a payload/dependency or delete-child on a containing directory. These permissions are independent of the inspected POSIX mode bits. The current predicates accept the metadata, and the content hash also remains valid until a write occurs. That principal can then mutate the closure or replace a dependency after content verification. The stated seal guarantee that other accounts cannot replace the path/dependencies is therefore not proved. This is a foreign-principal permission defect, distinct from the documented ability of the authorized operator to deliberately replace their own files.

Required repair: inspect ACL authority on native artifacts and replacement-capable ancestors through the maintained native ACL owner; configure a finite artifact ACL with operator ownership and runtime read/execute, and refuse foreign write, delete, ownership or security-modification authority. Preserve exact bundle symlinks and avoid applying a private-0700 policy to the deliberately readable/executable bundle. The existing config ACL mechanism is reusable but its private-mode invariant is insufficient as an artifact policy.

This is a source counterexample, not an executed Darwin exploit. Native ACL setup/refusal, inherited ACL behavior and actual daemon traversal require a Darwin fixture/native run. No host ACL was changed in this review.

### B2 — Windows artifact ancestry accepts foreign ownership

At review time, `verifyOpenRGBStageWindowsACL` in `openrgb_appimage_prepare_windows.go` compared the owner SID only when `exact` was true (line 111). Ancestry verification calls it with `exact=false`. That branch scanned allow ACE masks for explicit replacement rights (lines 161–166) but ignored the actual owner SID.

Counterexample: a pre-existing ancestor owned by a foreign ordinary principal, with a protected DACL granting the operator full control, SYSTEM full control, and the foreign principal only read/execute. The explicit ACE scan accepts this state: the operator and SYSTEM are allowed exceptions, and the foreign ACE has no tested replacement bits. On Windows, ownership itself normally grants implicit READ_CONTROL/WRITE_DAC. The foreign owner can rewrite that DACL to grant deletion/replacement and replace a descendant despite the sealed descendant's protected exact DACL. Exact child ACL validation cannot compensate for mutable ancestry.

Required repair: bind each ancestor's owner to a bounded trusted identity set appropriate to the native machine path, in addition to checking explicit replacement rights. Do not trust the dedicated runtime SID as an ancestor owner. Account for OS-owned ancestors without permitting arbitrary owners, and retain the existing reparse-point and hard-link refusal. Native fixtures must exercise a foreign-owner/read-only-ACE counterexample, trusted OS ancestry, and normal operator-owned publication.

This is a source counterexample. No Windows owner/DACL or native filesystem operation ran here. The implementation owner acknowledged B1/B2 directly; their repairs are outside this reviewer's write scope and require a subsequent independent recheck.

## Independently checked

The review followed acquisition selection through exact source admission, retained envelope snapshot, offline extraction, whole-tree and payload proof, native-header/closure verification, stage publication/reuse, private constructor environment and optional prerequisite setup. It inspected native artifact ACL/path code, normal artifact retention/source implications, and the explicit unsupported Windows ARM64 case. Capsule lifecycle integration remains separately owned; incomplete runtime coupling is not misreported here as an artifact defect.

An independent offline measurement rehashed the four already retained official assets and extracted the three ZIPs into private test directories. Linux arm64's already extracted research tree was read without running unsquashfs or any downloaded program. Linux arm64, Darwin Intel/ARM and Windows x64 all matched their frozen envelope, payload and whole-tree digests, and native ABI parsing passed. Opposite-architecture ELF/Mach-O rejection passed. This verifies agreement with the retained official-byte evidence; the review did not reacquire bytes or independently establish a publisher signature. The documented provenance is observed official HTTPS content, not an upstream signature.

No additional material source finding was identified in exact pre-execution adoption, delayed-symlink ZIP extraction, selected-source snapshot refusal, constructor XDG_CONFIG_HOME/APPDATA isolation, platform-profile closure, or explicit optional SDA prerequisite routing. The Windows prerequisite source uses System32 rather than PATH, checks native x64/v14>=14.44, cached trust and actual signer identity, and compares the bundle's named MSVC imports with native export names. Those Windows APIs and actual DLL exports were not executed here. The optional handler remains source/build evidence, not qualified native installation.

Private checks run in the isolated checkout:

```text
go test ./internal/resources ./internal/tools/msvc-runtime \
  -run 'Test(OpenRGB(NativeProfile|WindowsStage|Zip|Constructor|Adopt|StagePreparation|NativeConstructors|FirstLaunch)|MicrosoftRuntime)' -count=1
PASS: resources 0.866s; msvc-runtime 0.003s.

VROOLI_OPENRGB_PROFILE_MEASUREMENT_ROOT=<isolated>/.vrooli/work/openrgb-official-1.1 \
go test ./internal/resources \
  -run 'Test(OpenRGBReleaseProfileMeasurement|OpenRGBExactDownloadProfileHasClosedByteBound|ManagedServiceArtifactSnapshotRejectsChangedManifestBeforeAcquisition)' -count=1 -v
PASS: resources 1.500s; all four retained release measurements passed.

go test ./internal/runtime \
  -run 'Test(ToolManifestsReferenceRegisteredHandlers|LoadToolsReportsUnknownHandlerWithoutPanic|LoadToolsReportsMalformedManifestWithoutPanic)' -count=1
PASS: 0.010s.

go test ./internal/runtime -run 'TestEmbeddedToolManifestsValidateAgainstSchema' -count=1
PASS: 0.019s, including the commandless MSVC handler and squashfs-tools metadata.
```

These checks use the package's private test storage and fixtures. No full-package/heavy baseline was rerun; its separately documented sparse-copy/source-integration limitations remain. No product source or generated artifact was edited by this reviewer.

## Changed, verified, remaining, unverified

Changed: this report only. Verified: retained four-target content/ABI agreement, focused adoption/config/staging/profile guards and tool schema/handler registration. B1/B2 were source-reviewed and accepted by the implementation owner, with repairs pending independent review at report time.

Remaining: repair and independently recheck B1/B2; complete the separately owned native capsule/lifecycle integration; exercise selected-source acquire/adopt/setup/start/update/rollback/disable through that final implementation. Ordinary opt-out must continue to avoid acquisition, process launch and hardware probing; profile availability alone cannot enable the resource or qualify a native platform. Exact retained source/closure revalidation must survive normal update, rollback and restart.

Unverified: Darwin native ACL/exclusive publication/loading; Windows native DACL, trusted ancestry, virtual-service SID, System32 signer/export/cache checks, actual Winget prerequisite installation and Session0 execution; Linux arm64 libc/loading; all hardware access, SDK/device output, visible Off/static, logout/boot/resume and physical qualification. No native target, root, SDK, hardware, service, grant, credential or dependency acquisition operation was performed. No long-running process/service was started. Full CHARTER/SPEC acceptance remains open.

Recall used `search-hub query 'review OpenRGB native artifact acquisition adoption release profile' --type library,record,skill,doc --limit 5`. Search returned usable results with degraded agent-manager semantic retrieval, prompt-manager skill, source-ledger and swarm-manager providers; it was not treated as a complete inventory. The supported CLI shims incidentally rebuilt their local search-hub/prompt-manager CLI cache. The writing-standards skill was read for this report. No Git mutation or main-checkout product edit occurred.
