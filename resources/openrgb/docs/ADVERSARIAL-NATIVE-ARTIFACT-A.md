# Independent native artifact review A

2026-11-11. Reviewer: `/root/native_lighting_portability/artifact_native_review`. Scope: isolated `Vrooli-native-lighting` source only. Read the founding CHARTER, SPEC, STATUS, FEEDBACK, native gap audit, NATIVE-RELEASE-PROFILES and artifact checkpoint. This is an artifact/configuration source review, not full destination closure.

## Result

One material finding was found and repaired by the implementation owner during this pass. No further material finding was identified in the reviewed artifact/profile/acquisition/adoption/configuration/MSVC source. This pass therefore does **not** count as an originally finding-free maturation pass. The repair has focused local regression evidence; Windows native execution remains open.

### A1 — Windows stage verifier rejected every local drive path

Before repair, `internal/resources/openrgb_appimage_stage.go`, function `verifyOpenRGBReleaseDir` (currently line 66), rejected `:` anywhere in the absolute stage path. Given an ordinary native stage such as `C:\ProgramData\Vrooli\resources\openrgb\artifacts\AppDir`, the verifier returned its canonical-path error before sealing, whole-tree or payload verification. Both newly extracted and existing stages use this verifier. Consequently ordinary Windows setup could never publish successfully, and start/update/rollback could never qualify a native stage. Cross-compilation and envelope parser tests did not expose this behavior.

The owner replaced the unconditional predicate with `openRGBStagePathHasUnsafeCharacters` (currently line 125). It permits exactly the local drive prefix on Windows and continues to reject a subsequent colon (alternate data stream), semicolon, dollar, control character, UNC and relative-drive forms. `TestOpenRGBWindowsStageAcceptsDrivePrefixOnly` exercises actual `C:\ProgramData...` and `D:/...` strings plus refusals. I independently reran it successfully after repair. This proves the platform-independent path predicate; it does not prove Windows filesystem/ACL publication.

## Review and evidence

The source review followed normal paths rather than only profile parsers: manifest platform selection and bounded acquisition; selected host-source fingerprinting before execution; exact payload plus enclosing whole-tree adoption; retained-envelope snapshot and exclusive stage publication; stage reuse and lost-publication acknowledgement; retained-stage launch and rollback; constructor environment isolation; optional native prerequisite handler and offline PE import/export checks. Platform-specific POSIX seal and Windows DACL/reparse/hardlink code were inspected. No acquisition, package installation, device probe, service operation, target dispatch or grant change was performed.

Independent offline measurement used the already retained official bytes in `.vrooli/work/openrgb-official-1.1`. `TestOpenRGBReleaseProfileMeasurement` passed for Linux arm64, macOS Intel, macOS ARM and Windows x64. This recomputed complete extracted-tree and payload digests, checked the frozen envelope digests, and parsed native ABI. Both macOS ZIP framework symlink closures survived extraction and hashing. The measured identities matched all four frozen profiles. No release executable was run.

Commands and results:

```text
go test ./internal/resources ./internal/tools/msvc-runtime \
  -run 'Test(OpenRGBRelease|ExtractOpenRGB|OpenRGBConstructor|OpenRGBAdopt|Native|MSVC)' -count=1
PASS: resources 0.268s; msvc-runtime had no matching tests.

VROOLI_OPENRGB_PROFILE_MEASUREMENT_ROOT=<isolated>/.vrooli/work/openrgb-official-1.1 \
go test ./internal/resources \
  -run 'TestOpenRGB(ReleaseProfileMeasurement|NativeProfileMatrixIsClosed|WindowsStageAcceptsDrivePrefixOnly|Zip|Constructor)' -count=1 -v
PASS: 1.390s. Includes four actual retained release measurements, closed target matrix,
Windows drive-prefix repair, ZIP traversal/foreign-root/backslash refusal and archive-link refusal.

go test ./internal/resources ./internal/tools/msvc-runtime \
  -run 'Test(OpenRGB(NativeConstructorsRemainPrivate|FirstLaunchSeedsOnlyResourceOwnedConfig|StagePreparation)|MicrosoftRuntime)' -count=1 -v
PASS: resources 0.564s; msvc-runtime 0.002s.
```

The final command covered exclusive publication/reuse, lost acknowledgement recovery, cancellation and partial extraction, tampered closure/snapshot refusal, private scratch recovery and aliased scratch refusal, ambient-config preservation, all three constructor environment contracts, and no MSVC installation on other platforms. These are focused package fixtures with private temporary state, not Test Genie scenario or native operated journeys.

The Windows official executable's import table was inspected offline with the pre-existing `objdump`: it imports the four named MSVC DLL families checked by the new owner and bundled Qt/libusb/HID/PawnIO libraries. The MSVC source checks System32 rather than PATH, x64 PE, v14 >=14.44, cached WinVerifyTrust and the actual PKCS7 signer certificate, then named imports across bundle EXE/DLLs. None of those Windows APIs executed in this Linux review. The claimed signer/ABI behavior therefore remains source-reviewed only.

## Remaining and unverified

Changed by reviewer: this report only. Product repair and its regression were written by `native_artifact_profiles`, not the reviewer. No Git mutation, generated-output edits, root command, dependency acquisition, hardware or service effects occurred. No long-running resource remains from the review.

Remaining integration evidence: ordinary native selected-source acquire/adopt/setup/start/update/rollback/disable journeys through the final shared lifecycle implementation; source-manifest platform status must agree with that completed integration. The manifest still declares macOS/Windows unsupported while sibling integration is in progress. The artifact extension alone does not justify promoting that status.

Unverified: native Windows DACL creation/inheritance, virtual-service-SID provisioning, System32 signature cache and named-export availability, Windows loader/Session0 execution; macOS native ACL/exclusive publication and loader behavior; Linux arm64 libc compatibility and execution; all device access, SDK operation, visible lighting output, logout/boot/resume and physical qualification. Existing exact pinned bytes and cross-builds must not substitute for these observations. Full CHARTER/SPEC acceptance remains open.

Recall used `search-hub query "review OpenRGB native artifact acquisition adoption release profile" --type library,record,skill,doc --limit 5`; results reported `source-ledger.scopes` degraded by a timeout. Other providers returned results; the source-ledger outage was not treated as a complete inventory. Writing-standards skill was read for the report.
