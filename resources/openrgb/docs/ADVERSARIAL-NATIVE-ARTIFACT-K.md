# Independent native artifact review K — 2026-11-11

Current actual-operator/module-free Windows source pass. Reviewed from `/home/matthalloran8/Vrooli-native-lighting` after reading the delivery CHARTER, SPEC, STATUS, FEEDBACK, native gap audit and current artifact checkpoint/release-profile design. Previous A–J conclusions were not used as evidence that the migrated source is correct.

**Decision: no material source finding in the reviewed artifact/acquisition/adoption/profile/constructor boundary.** This is source and offline-byte evidence only. It does not qualify Windows or Darwin publication, token/ACL behavior, loader behavior, headless launch, HID access, SDK operation or physical lighting.

## Independent counterexamples

Retained harness: `.vrooli/work/artifact-review-k/openrgb_adversarial_k_review_test.go`; execution log: `.vrooli/work/artifact-review-k/counterexamples.log`. The harness was temporarily compiled inside `internal/resources`, then removed from that production package; no production implementation was edited.

`TestAdversarialKCompleteWindowsDerivation` independently extracts the actual pinned Windows ZIP through the current offline extractor, inventories all22 regular files and checks:

- Each of the22 source leaves is individually corrupted by changing its last byte. Every derivation refuses the changed original before removing any of the six source modules. Bytes are restored between counterexamples.
- A canceled derivation leaves the complete original tree unchanged.
- A valid derivation retains exactly16 PE files, each byte-for-byte equal to the corresponding original file; no library or executable patch is hidden in the transformation.
- Each of the16 retained files is individually corrupted. Every changed derived tree fails the compiled serving-tree identity.
- Each of the six omitted modules is individually restored with its original official bytes. Every restoration fails both explicit module-absence admission and the complete serving-tree identity.
- The resulting restored derived tree still reproduces the pinned serving digest.

`TestAdversarialKRecipeEveryIdentityFieldBound` independently changes the architecture, original envelope, unchanged payload, derived tree and recipe digest. Every changed profile fails validation against the canonical recipe identity.

Result: **PASS1.574s**, all22 source mutation subtests executed. No vendor program, SDK endpoint, hardware probe or native target operation was used.

## Source traces and focused regression evidence

The current source verifies complete original tree → exact fixed omissions → separately pinned derived tree → selected-source-bound metadata → seal → exclusive publication. Recipe identity participates in the stage namespace. Retained original and derived identities remain distinct; a legacy complete original stage cannot become a derived stage by reinterpretation. The original adopted installation is copied and verified rather than modified. Selected envelope versus selected payload identity is explicitly bound in metadata.

Inspection covered `openrgb_serving_derivation.go`, release profiles and native ABI/host guards, ZIP extraction, adopted-bundle copying and regular-file source-open helpers, stage preparation/sealing/locking, native launcher preparation and sealed-root selection, constructor isolation, Windows manifest parsing, Linux/Darwin/Windows runtime prerequisites, six-provider MSVC import/provenance checking, and the selected managed acquisition/adoption owners. The boundary snapshot is `.vrooli/work/artifact-review-k/source-manifest.json`. A post-run drift check found only a concurrent change to `openrgb_native_update_capsule_test.go` (a peer-owned lifecycle fixture); no captured production file changed during K. That fixture change does not alter the reviewed derivation or the independent retained K harness.

The observed safety properties remain coherent:

- Selection/preflight uses bytes and native metadata; OpenRGB adoption returns before generic `--version` execution. Acquisition and provenance facts on macOS/Windows use only explicit OS/architecture. Stage/launcher preparation does not enable/start the backend or acquire a runtime provider.
- Current-host admission excludes translated macOS and emulated/ARM Windows profiles; native OS floors remain explicit. Whole dependency trees accompany payload ABI proof. Windows manifest admission reads RT_MANIFEST name1 and checks every language entry for asInvoker/uiAccess=false.
- Windows keeps all16 PE code images, verifies all six named Microsoft runtime providers from native System32 and separates UCRT. No arbitrary PATH DLL or app-local MSVC substitute is admitted. Linux inspects native interpreter/system provider version definitions; Darwin checks the active dyld cache with an explicit no-CGO unsupported path.
- APPDATA and XDG_CONFIG_HOME are private before construction, with private CLI configuration, disabled automatic profiles/remotes and bounded launch environment.
- ZIP regular writes precede link creation; queued link parents are checked as actual directories, so one queued alias cannot redirect a later link. Source copying pins an observed regular opened object; Unix nonblocking open refuses a FIFO substitution before read.
- New native publication requires the canonical privileged machine code root. Existing sealed stages take a read-only verification path before publisher authorization. Native locks use admitted read-only handles without per-user chmod/chown. Launcher input comes from the protected installed supervisor image, not a caller path.
- Windows readonly serving verification derives the actual token SID rather than rediscovering mutable owner.json through an ambient global-root variable. The stage verifier still checks exact machine-owner/read-only DACLs and disallows reparse/multiple-link files.

Focused existing tests executed against actual retained Windows bytes: managed and adopted derivation, metadata/selected-source mismatch, original-stage refusal, publication/lost-ack recovery, canceled candidate preserving rollback closure, safe ZIP/link topology and regular-file/FIFO source changes. Result **PASS4.334s**, `.vrooli/work/artifact-review-k/focused.log`. Its separately opt-in current-host adoption test initially skipped; the subsequent broad run explicitly supplied that retained Linux envelope.

## Reproduction and evidence boundaries

Every source-aware command used both `VROOLI_ROOT` and `VROOLI_SOURCE_ROOT` set to the isolated checkout. Actual-byte tests used `VROOLI_OPENRGB_PROFILE_MEASUREMENT_ROOT=<isolated>/.vrooli/work/openrgb-official-1.1` and `VROOLI_OPENRGB_MSVC_BUNDLE=<measurement>/OpenRGB_1.1_Windows_64_8e16de2/OpenRGB Windows 64-bit`. No network acquisition, vendor execution, native service, device effect, host installation or Git mutation occurred. Discovery was available and returned relevant source/work records; this review then inspected current source directly.

The broad current resources regression command is `go test ./internal/resources -run 'OpenRGB|WindowsHID|MSVC|ManagedServiceAcquisition' -count=1 -timeout=180s` with the above variables plus `VROOLI_OPENRGB_ADOPTION_ENVELOPE=/home/matthalloran8/.vrooli/artifacts/openrgb/1.1/openrgb.AppImage`; result **PASS20.844s**, retained in `.vrooli/work/artifact-review-k/broad.log`. This broad run completed independently after the new K counterexamples; the actual retained current-host adoption envelope was enabled.

Remaining proof is actual target execution: elevated initial publication and ordinary-token existing reuse, custom-global-root readonly launch verification, native locking/crash behavior, cached signer/runtime ABI observations, actual constructor/headless execution, module-free startup behavior, non-elevated HID access, SDK/visible Off/static/effects and boot/logout/resume/device-return. Cross-compilation and Linux-host PE/Mach-O parsing cannot prove those outcomes. This report does not flag missing native observations as an implementation defect and cannot close R1–R3/L1–L4 or the full destination.

Changed: this report and private review evidence only. Verified: current source traces and the independent actual-byte counterexamples above. Remaining/unverified: target runtime and hardware proof above; no source repair requested by K.

Work-record receipt: `293e3194-cc78-4baa-aeee-984aa0ad35ce`.
