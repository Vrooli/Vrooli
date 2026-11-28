# Native artifact adversarial pass H — 2026-11-11

## Verdict

**Material finding H1: incomplete Windows Microsoft runtime prerequisite closure.** This pass is finding-bearing even after repair. No second material artifact defect is established by the completed examination. This verdict covers source and retained official bytes; it does not qualify native runtime, security APIs, installation, service accounts, hardware or the complete product.

Reviewer: `artifact_review_h`. Source: `/home/matthalloran8/Vrooli-native-lighting`. Read the delivery CHARTER, SPEC, STATUS, FEEDBACK, native gap audit, resource native release profiles/progress/G and `docs/TESTING.md`. Applied writing-standards. Recall searched nine corpora and reported source-ledger degradation; that provider's outage is not a claim of complete historical recall. Every source-aware command explicitly exported both VROOLI_ROOT and VROOLI_SOURCE_ROOT to the isolated root. The shared source remained writable by other owners; results describe observed inputs, not an atomic snapshot. Concurrent initial/update/public installed-supervisor caller integration remains separately owned and is not accepted by this artifact review.

## H1 — two imported MSVC providers escape prerequisite checks

At the finding point, `internal/tools/msvc-runtime/inspect_windows.go:34` checked only `msvcp140.dll`, `msvcp140_codecvt_ids.dll`, `vcruntime140.dll` and `vcruntime140_1.dll`. `imports.go:15` opened the same four export providers; its line47 condition silently ignored imports from every other DLL.

Independent PE import-table parsing of the pinned official Windows bundle found:

| Actual retained image | Required import absent from both check lists |
| --- | --- |
| Qt6Core.dll, Qt6Gui.dll, Qt6Widgets.dll | `_Aligned_get_default_resource:MSVCP140_1.dll` |
| Qt6Gui.dll | `__std_smf_hypot3f:MSVCP140_2.dll` |

Neither extension DLL is bundled. A native machine can possess all four checked, signed/version-compatible providers and UCRT while either extension is absent, incompatible or lacks its named export. The existing preflight then reports success before the native loader fails on the Qt dependency. This is a source completeness defect established from the actual release, independently of unavailable target runtime evidence.

The private overlay regression `/tmp/openrgb-review-h-counterexample_test.go` builds finite synthetic x64 PE export providers for every named import from the four checked libraries, omits both extension DLLs, and checks the real retained Windows bundle through production `verifyBundleImports`. Expected behavior is refusal. The test failed because production returned nil, reproducing the gap in0.109s. Exact log: `/tmp/openrgb-review-h-counterexample.log`; overlay: `/tmp/openrgb-review-h-overlay.json`; actual imported-symbol output: `/tmp/openrgb-review-h-msvc.log`. The synthetic providers are parser fixtures, never executed or installed. No product test/source file was added by this reviewer.

Routed promptly to the artifact owner with exact source lines, symbols, reproduction and required repair: cover both extensions with the same native System32 ABI/version/Microsoft signer/named-export checks; prove refusal of missing providers and missing exports. Repair and independent recheck are recorded below. Native signature-cache/loader execution remains a separate unqualified gate.

## Independent examination beyond H1

The fixed profile matrix admits Linux amd64/arm64, Darwin Intel/ARM and Windows x64 and rejects Windows ARM. Current-host checks separately reject Windows emulation and Rosetta. Pure offline profile lookup cannot substitute for actual host admission. Whole-tree and payload hashes remain exact, with ELF64 little-endian machine, Mach-O64 little-endian subtype/macOS minimum and PE32+ process checks. The retained four-profile measurement exercised full tree/payload/all recognized native-image ABI, Windows RT_MANIFEST, and envelope/payload adoption. All passed. No vendor executable ran.

I independently recomputed each of the four new official envelope sizes/SHA256 and compared the governed repeat receipts: all match. Provenance is observed official HTTPS content, with no upstream signature claim. Source acquisition reuses canonical SDA approval, fixed official asset names, bounded bytes/redirects, private exclusive output and context cancellation. Production native target/provenance facts use OS/architecture without generic GPU or ambient preference callbacks. Default disabled intent does not acquire or probe.

An additional standalone standard-library import inventory examined81 ARM ELF images,78 images in each Mac bundle and16 Windows PE images. It exposed H1 rather than relying on profile-test green results. Mac external imports were OS framework or `/usr/lib` paths. Linux external-provider resolution and version metadata remain explicit and conservative. Bundle counts/sizes independently fit production limits: ARM277entries/104258583bytes; IntelMac440/117363789; ARMMac444/112768484; Windows26/50790712. Thus the128MiB/512-entry verifier does not accidentally reject these retained closures. The initial private import script used Go PE ImportedLibraries, which does not enumerate these imports; I corrected it to ImportedSymbols before interpreting Windows results. Only the corrected `/tmp/openrgb-review-h-imports.log` supports this inventory.

ZIP extraction rejects path/root/type/case aliases and bounded-size violations; regular writes and all link-parent preparation precede symlink creation, with actual-directory checks at every parent. The queued-alias counterexample regression and actual archives passed. Adopted payloads require complete exact enclosing bundles; copying rechecks destination content, preserves the original and never executes a version probe. Constructor APPDATA/XDG_CONFIG_HOME isolation, disabled saved clients and automatic profile hooks remain enforced before vendor startup.

Darwin uses a separate root-owned code store with native immutable-path/extended-ACL verification, confined framework links and immutable real dependencies. Windows creates fresh BA/SYSTEM-owned objects with exact operator/service readers; inherited/broad authority, reparse points and multiple links are refused. Publication is exclusive, cleanup checks its owner, and uncertain durability remains an explicit error. Read-only admitted native publication locks avoid generic user-owner chmod/chown. I checked the actual native launcher wrapper and underlying supervisor-image APIs: they admit the caller's current manager slot and mapped immutable image, bind the installed digest and recheck the manager slot; no caller pathname or generic Windows process-image fallback exists. Public caller integration and platform API execution remain separate proof.

Windows PawnIO receipt handling is a fixed six-leaf exception. The owner verifies complete byte content, fixes hash/size plus FileID128/volume64/timestamps, and seals the authenticated capsule. The runtime proves exact identity, exact metadata-only ACL and actual access denial; every other leaf is hashed and the canonical tree reconstructed. No public arbitrary skipped-path/hash parameter exists. The retained six-leaf measurement passed; actual native ACL denial/upstream startup remains unobserved.

Linux loader/system symbol-version checks, Mac15/dyld-cache metadata checks and Windows system runtime checks have explicit native limits. H1 demonstrates that the Windows list was incomplete despite passing maintained tests. No unavailable native proof is converted into a source defect or a green qualification. Descriptor/schema `unqualified` support matches the maintained loader; opt-out/configuration/recovery/identity regressions passed. No additional material complexity/duplication defect is established.

## Executed evidence

| Check | Observed result |
| --- | --- |
| Scoped resource run `go test -v ./internal/resources -run 'Test(OpenRGB\|NativeOpenRGB)' -count=1` with explicit retained profile/module/adoption inputs | PASS25.483s;169 top-level passes;3 declared skips |
| Retained four-profile whole tree/payload/all-image ABI/startup manifest/envelope+payload adoption | PASS within the resource run |
| Offline ARM production extractor and six blocked-module byte measurement | PASS within the resource run |
| Resource manifest + MSVC maintained packages | PASS0.233s/0.003s;60 total top-level passes |
| SDA releasequalification/installgateway/dependencygovernance | PASS0.018s/0.616s/0.611s;136 top-level passes;fresh acquisition skipped |
| Platform native broker/identity/render/transport fixture subset | PASS2.621s;31 top-level passes |
| CGO0 resource test binaries: Windowsamd64, Darwinamd64+arm64, Linuxarm64 | All compiled; fail-fast loop completed |
| Independent official envelope receipt comparison | Four exact size/hash matches |
| Independent desired-refusal H1 overlay | FAIL0.109s before repair, proving false acceptance |

The resource run explicitly skipped the separately gated pinned AppImage-stage fixture, runtime-normalization fixture and live Linux prerequisite check. No skip is counted as proof. Cross-builds are compilation only. SDA did not download release bytes. Paths: `/tmp/openrgb-review-h-{resources,contracts,sda,platform,crossbuild,imports,msvc,counterexample}.log`; binaries: `/tmp/openrgb-review-h-{windows-amd64,darwin-amd64,darwin-arm64,linux-arm64}.test`. These temporary artifacts are corroboration, not immutable evidence-service receipts.

Resource-log SHA256 `e7b7bb71d9c0166c7ddabc99f36b274072f01f91150620ac39c72e41a9ec2910`; contracts `c242bfc00dd03cd2615d64cf183bbfd07d71925bd393e81621c35bb4e84adb1a`; SDA `f6d43a2c6395ffa1eb154750419e607a8a704c2b98f676507545de5cbfcbb5a3`; platform `345af589626f8b9775f096f9540c9f04a869232359eb762e2dc0c0716991cbfa`; pre-repair counterexample `80ca02a06d74d5a928f9302b5fe180cf53c42557388071d31f9dc45376a4a968`.

Finding-point source SHA256: MSVC imports `cebb8ecd65e0d6308d30e40450bd967d5f2d843f6364d0ef386dba158ead467e`; Windows inspector `c6918fdb3a0ed335e1e81bf3aa7dfe4a887d891abcc8d9c945990137aefb47c5`. Reviewed profile `ee8508701e4b0caf72a1cfbbcf8d671ff83101735c9ce09b29f034bab581e2c5`; ZIP `2b75b7491ba0e9c20937d2e08e3f2762696e0dbf188eaa3de4e7dd1310bf13f2`; adoption `61385eb2f0c530369e75e74f037c760577667b7037f3838e196d65101de89caf`; launcher `238df262adbf0dffb730df3458681a90fd89c4c7cde01016feee7d7535bfe126`; denied-module verifier `46f93a26a6aa1e59555594c5eed66c78e32a82e83f0d3c1ca7042fea17d7eb1a`.

## H1 repair and independent recheck

The artifact owner replaced both separate four-library lists with canonical `requiredRuntimeLibraries()`, containing all six measured external Microsoft providers. Both native provenance/version checks and import-export checks use that owner. Unknown MSVC-family providers now refuse admission rather than silently escaping the verified set. The owner added always-on metadata fixtures for each extension's absence, wrong export and complete export, plus unknown-family refusal and a retained actual-bundle inventory test. This is a small owner-boundary repair; no new DLL acquisition route or runtime bypass was introduced.

I reread the repair and ran the original independent private overlay unchanged: PASS0.056s, now correctly refusing the absent extensions. The maintained MSVC module passed0.123s with all five top-level tests, both extension subtests and the actual retained-bundle inventory enabled. A refreshed Windowsamd64 CGO0 resource test binary compiled successfully after the repair. Logs: `/tmp/openrgb-review-h-counterexample-after.log`, `/tmp/openrgb-review-h-msvc-after.log`, `/tmp/openrgb-review-h-windows-after-build.log`. Counterexample-after SHA256 `5e17dab0c2d8757c8bbb1304d9d7639a9fbe90d19814c8d816330205c71bfb1b`; MSVC-after `bfe65c98fb652467d143613556766857d7e05f305c0ed42e9d3706c6955d5ecb`. Repaired source SHA256: imports `395a77d6218db3a528c7220700fc311243c0fe4510a97b9a617fb9a14dcb662c`; native inspector `60a6bbadaf5603d90ab46b06b6aed1b399983be33f0bcb428658f4a85c937cfa`; new behavior tests `1cf9ce77afe55b349bdbe0af6001f0890030afbca4a75b5e3b2a2b3b3f034f5a`.

Separately, the owner's contemporaneous OS-baseline audit added the Qt6.8 Windows10 1809/build17763 minimum to the exact profile and the pre-effect current-host gate. I checked the [official Qt6.8 Windows support table](https://doc.qt.io/qt-6.8/windows.html), reread the source, and independently ran the exact baseline boundary/profile-matrix/four-retained-profile tests: PASS1.545s, all three selected tests executed. `/tmp/openrgb-review-h-osbaseline.log`, SHA256 `2b35b64af3ea58ed22ffec63f44d078c9b05e62089ba4781cb4782e311b759dc`. An initial selector missed the baseline test and skipped the unset measurement opt-in; the corrected command above replaced that evidence and is the only baseline result counted. No native Windows OS query was executed here.

H1 is source-repaired and independently rechecked. H remains finding-bearing; another clean pass must come from a fresh reviewer. Native signature/API/runtime proof remains unqualified.

## Changed, verified, remaining and unverified

Changed: this report only in the product tree; temporary private fixtures/logs/binaries are outside it. Parent workflow owns delivery journaling. No Git, root, service/device lifecycle, target dispatch, SDK execution, dependency acquisition/install, grants or permission operations occurred. All started commands completed; no resource/service/model was started.

Verified: retained bytes/provenance agreement and offline profiles/extraction/adoption; the reviewed source trust boundaries and focused tests/builds; H1's concrete pre-repair false acceptance and the independently rechecked repair. Passing maintained tests did not cover H1.

Remaining: fresh independent passes are required for clean-pass counting after the independently rechecked H1 repair. Public installed-supervisor provisioning, final host-image binding and concurrent initial/update integration remain with their named owners. This report cannot close full SPEC or substitute for the fixed full-destination challenger.

Unverified: native Darwin SDK/CGO/link/ACL/dyld execution; Windows System32/signature-cache/import/SCM/FileID/DACL/publisher/lock APIs and installer behavior; native crash recovery and process launch; denied-PawnIO runtime behavior; native accounts/device access, SDK, visible Off/static/effects and logged-out boot/resume/device return. Linux ARM permission plan/apply/rollback is device-owner scope with separately routed retained-amd64 gate; artifact support never qualifies that route. Current-host lighting/wireless and full product acceptance remain open.
