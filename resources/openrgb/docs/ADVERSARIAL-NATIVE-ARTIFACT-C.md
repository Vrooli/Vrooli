# Native artifact adversarial pass C — 2026-11-11

## Verdict

**Material finding C1 discovered and repair rechecked. This is not a clean pass for the consecutive-clean-pass requirement.** No additional material source defect was established in the reviewed artifact boundary. Native OS execution, native ACL/API behavior, service startup, hardware access and physical output remain unqualified. This report does not accept the full CHARTER/SPEC destination.

Independent reviewer: `artifact_review_c`. Scope: isolated `/home/matthalloran8/Vrooli-native-lighting`, source and retained official bytes. Read main delivery CHARTER, SPEC, STATUS, FEEDBACK and `product-resumption/NATIVE-LIGHTING-GAP-AUDIT-20261111.md`, plus isolated `docs/TESTING.md`. A/B reports were not used as acceptance evidence. The shared isolated source was actively changing during this review; tests prove the inputs they observed, not an immutable whole-tree revision.

## Material finding and repair

**C1 — native platform declarations rejected by the maintained resource loader (high impact, repaired).** `resources/openrgb/resource.json:34` and `:35` use `unqualified`. The canonical schema's `common.schema.json#/definitions/platformStatusToken` permits this truthful state, but the original `internal/resources/manifest/manifest.go:62` allowed only `supported`, `partial`, and `unsupported`. Loading the actual OpenRGB manifest therefore failed before native capsule, config and lifecycle behavior could run. This also affected Linux loading the same resource manifest; an unsupported native runtime was not the cause.

Concrete reproduction: the first scoped resources execution failed all selected `TestOpenRGBNativeCapsule*` fixtures with `validate resource manifest .../resources/openrgb/resource.json: platform support value "unqualified" is invalid`. That same execution independently passed actual-byte release measurement, separating declaration rejection from archive correctness. Evidence: `/tmp/openrgb-review-c-tests.log`, SHA256 `0a6e53f74fd0699b13c9c1f4aeb790519c2f474ba5107ceadfd68cae13be206b`.

Routed promptly to the artifact owner. The owner corrected the loader's explicit allowed-state list (current line 62) and added `TestPlatformUnqualifiedRetainsSourceQualificationPath`. I independently observed that regression PASS and subsequently all selected capsule/config behavior PASS against the real manifest. A temporary intermediate compile failure naming two not-yet-landed observer helpers was disclosed by the active owner; the helpers subsequently appeared and the successful retry compiled and exercised the package. It is not retained as a separate artifact design defect.

## Trust-boundary assessment

Reviewed acquisition/plan/adoption, archive extraction, complete release profiles, payload/bundle ABI checks, Windows embedded startup manifest parsing, native prerequisites, constructor settings, sealed roots, pending publication/recovery, launcher staging, privileged lock handling and denied-module capsule reconstruction.

The current source keeps native executable authority separate from mutable operator state: Darwin `/Library/PrivilegedHelperTools/com.vrooli.openrgb`, Windows known-folder ProgramFiles `Vrooli/OpenRGB`. Production publishers reject alternate fixture namespaces. Windows stage creation delegates atomic owner/DACL creation to `platform.PrepareImmutableNativeDirectory` and `WriteImmutableNativeFile`, rather than first creating operator-owned code and later chowning it. Canonical ancestry verification, reparse-point rejection, single-link leaf checks and finite exact operator/service read roles are present. The launcher follows the same publication and verification path. This assessment is source inspection plus cross-compilation, not observed native filesystem enforcement.

For Windows PawnIO, `openrgb_blocked_modules.go` fixes six names, SHA256 values and sizes; authenticated native material binds their FileID128, volume64, creation/write times and lengths. The owner hashes the complete closure before witness capture. `openrgb_blocked_modules_windows.go` accepts reconstruction only for the exact dedicated runtime principal, checks native identities, rechecks the stage seal and requires actual `FILE_READ_DATA` denial on each blocked leaf. Readable leaves are hashed and combined with fixed authenticated hashes into the exact pinned tree. Operator/SYSTEM uses full byte proof. This prevents a caller-selected digest or arbitrary omitted leaf from substituting for full-closure qualification. No bypass was established under the stated privileged publisher trust model. This does not prove that actual OpenRGB execution on Windows tolerates denied PawnIO bytes or that native security APIs behave as expected.

The native lock path requires publication authority, checks the canonical root/parent and opens a read-only admitted lock handle; it avoids the per-user artifact lock's ownership/mode rewrite. Reverification follows lock acquisition. No second generic staging implementation was found that bypasses these native owner decisions. ZIP extraction delays symlinks until regular-file writes complete, rejects alias/escape paths and enforces bounded sizes. Adopted trees are copied into staging and rehashed; payload identity alone does not admit an arbitrary bundle.

Runtime prerequisites remain read-only before launch: Darwin checks macOS baseline/shared-cache framework presence; Windows checks system x64 MSVC provenance/version and required exports. SDA owns the finite official-release measurement route and declared system-tool installation route. The Windows special-handler exception is restricted to matching `msvc-runtime` name/handler rather than arbitrary custom tool handlers. No installation was performed here.

## Independent verification

Every source-aware test/build exported both `VROOLI_ROOT` and `VROOLI_SOURCE_ROOT` to the isolated root. No native target execution, live acquisition, dependency installation, Git mutation, root action, service launch or device operation occurred.

- Retained actual-byte `TestOpenRGBReleaseProfileMeasurement` PASS: Linux arm64 ELF, Darwin Intel and ARM Mach-O bundles and Windows x64 PE bundle; offline ZIP extraction, envelope/payload/tree hashes, complete bundle ABI and Windows startup manifest checks. All four hashes matched the compile-time profiles. Existing Linux arm64 extracted bytes were verified rather than re-extracting that AppImage in this test.
- `TestOpenRGBBlockedModuleLeafTableMatchesOfficialBundle` PASS with the explicit retained Windows bundle path; all six fixed denied-module byte identities matched.
- Final resources selection PASS, 2.400s: 22 top-level tests including adoption refusal/source preservation, fixed/complete denied-module receipts, config preservation/isolation/recovery, authenticated capsule material/wrapper mismatch, uncertain claims, child claim-before-resume and abort behavior, owner authority separation and transport-confinement refusal. No selected test skipped.
- Additional resources selection PASS, 1.360s: exact acquisition bounds, stale manifest refusal, stage publication/retry/cancellation/tamper/alias recovery, adoption, first-launch config isolation, closed native target matrix and ZIP escapes/link ordering. No selected test skipped.
- Manifest `TestPlatformUnqualifiedRetainsSourceQualificationPath` PASS, 0.007s.
- MSVC runtime package PASS, 0.003s: non-Windows refusal and closed explicit winget-source install fixture. This is not execution of Windows runtime inspection.
- SDA releasequalification/installgateway scoped selection PASS (0.026s/0.024s); exact/short/long download fixture and manifest/path freezing. Live official acquisition test deliberately SKIP. Separate `TestSystemTool*` PASS, 0.012s, including changed-resolution and manual/release-owned refusal.
- Windows amd64, CGO0 `go test -c ./internal/resources` PASS to `/tmp/openrgb-review-c-windows.test.exe`. Compilation only; no native SDK/service/security/hardware qualification.

Retained local test logs: `/tmp/openrgb-review-c-final-tests.log` SHA256 `b42817fe29b0a332b6e218b9e486b1368f2d14f68359b5778a7b9925d013245e`; `/tmp/openrgb-review-c-additional.log` SHA256 `8b5b378a324c65a6239f5bf7b965195fdbf80fd4e4fe372751a57f6bc806fcf8`; `/tmp/openrgb-review-c-sda.log` SHA256 `a3a6badc7418215c44f6b5e4969828c7918326a6b9449cdf4c44313898d97b83`. These temporary logs are local corroboration, not a promised durable evidence service.

Reviewed trust-boundary source fingerprints: `openrgb_release_profile.go` `4d65042f10a374f557e02d91bc8d95ba2f8e20764d146f21a8b6b1d8a5c65969`; `openrgb_blocked_modules.go` `33f0da92b997367f2aabafc6cf6e7e3812d005e908632fcfb21b12cabbc8148c`; `openrgb_blocked_modules_windows.go` `28a6537741d21c91a3089a4d5697a2d041ba67ead514674a570a47f9c0edaa2b`; `openrgb_appimage_prepare_windows.go` `4a6b35504224568e5cd267195a13d6b96a253e5b25ee5205b04373f1ffaef0ac`; `openrgb_stage_lock.go` `9b747c722843174eeba9d54ea31221c087086d67d25c7fea995ed086324ffff0`.

## Changed, verified, remaining and unverified

Changed by this reviewer: this report only in the product tree. Tests created isolated temporary fixtures/logs and one cross-compiled test executable; all started test processes completed. Product repairs were made by their existing owners.

Verified: C1 reproduction and repair recheck; retained byte/profile consistency; the scoped fixture assertions and Windows cross-compilation listed above; source trust boundaries within the reviewed outcome.

Remaining: a fresh independent pass after repaired C1 is required before counting a clean pass; the full effort still requires its consecutive clean passes and full-destination challenger. Preserve all existing hardware/operator recovery restrictions.

Unverified: native Darwin SDK/CGO and Intel/ARM execution; Windows ACL and FileID/volume observation on target; real MSVC signature/export inspection, winget installation and offline trust behavior; actual PawnIO denial startup compatibility, native launcher/lock/publication recovery, service-account permissions, logged-out boot, device return/resume, physical Off/static/effects and complete current-host wireless/lighting acceptance. Cross-compilation, authentic release bytes and passing fixtures do not discharge these obligations.
