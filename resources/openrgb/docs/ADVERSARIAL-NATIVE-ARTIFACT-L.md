# Independent adversarial native artifact review L

Current actual-operator model; 2026-11-11. Reviewer `/root/native_lighting_portability/native_artifact_profiles/artifact_review_l`. Source root `/home/matthalloran8/Vrooli-native-lighting`.

## Decision and applicability

No material source finding in the reviewed artifact/profile/acquisition/adoption/constructor/prepare/stage/launcher boundary. This is an independent source-only maturation pass of the current actual-operator and Windows module-free HID derivation model. It does not certify native execution, hardware, the full lighting product or the full destination. I did not read K's review or use its conclusions.

Read the main CHARTER, SPEC, STATUS, FEEDBACK and product-resumption/NATIVE-LIGHTING-GAP-AUDIT-20261111 before source work, then the current artifact checkpoint and authoritative NATIVE-RELEASE-PROFILES. The full cross-platform/cross-machine destination remains the denominator. The ordinary non-elevated actual operator is the runtime identity; privileged initial immutable publication is separate. Retired dedicated-account/restricting-SID/read-denial designs were not evaluated as current requirements.

Source fingerprint: `.vrooli/work/native-artifact-review-l/source-before.json` and `source-after.json` contain 210 exact source/test/manifest/recipe/schema entries. Their bytes match before and after validation; manifest SHA256 `a2e6c1ba54af72602a621ec794b914bd5488c51768013ecfe187ec7063e6213a`. Supplemental shared-platform boot/image handoff and tool/schema sources are hashed in `supplemental-source.json` at final inspection; they were not included in the before/after stability claim. No Git identity or dirty-tree status substitutes for these bytes. Matching endpoints do not prove isolation from an undetected intermediate concurrent edit.

## Adversarial source assessment

- **Official source versus derived serving authority:** `openrgb_release_profile.go`, `openrgb_serving_derivation.go` and `openrgb_appimage_prepare.go` require the complete original Windows22-file tree before omission. Exactly six fixed PawnIO modules are hash/size checked and removed; unchanged16-PE tree and recipe identities are independently pinned. Recipe participates in stage namespace. `serving-profile.json` binds the recipe and selected original envelope/adopted-payload identity. A payload hash alone does not admit a partial original bundle. Existing wrong/legacy stages fail without in-place rewriting.
- **Adoption and mutable sources:** `resources.go` and `managed_service_artifact_owner.go` route OpenRGB through offline exact content/ABI checks before the generic vendor-version branch. Adopted native payloads need their complete enclosing original closure. `openrgb_adopted_bundle.go` copies into separate storage, then verifies all copied bytes; the installed original is preserved. `openrgb_source_file.go` and Unix helper pin the observed regular opened object and use nonblocking opens to reject FIFO replacement before reading. Post-copy hashes remain mandatory.
- **Publication and recovery:** stage preparation copies a verified retained source, extracts, derives, writes metadata, seals, verifies, then publishes exclusively. Stage lock precedes pending cleanup. Cancellation before publication leaves no candidate authority. Lost durable-publication acknowledgement remains `ErrOpenRGBStageCommitUnknown`; retry verifies the complete retained stage instead of rebuilding it. Existing active/rollback stages are separate from candidates. Artifact preparation does not itself switch lifecycle roles.
- **Native immutable paths:** Darwin roots and code closure require privileged ownership, safe ancestry and native ACL verification. Windows publishers are restricted to the canonical machine code root, create machine-owned objects before content writes, and seal the complete readable closure with finite SYSTEM/Administrators/actual-operator rights. Reparse/multiple-link entries are rejected. Read-only existing-stage reuse precedes elevation requirements. Windows verification derives the current-token principal, not ambient global-root owner lookup. Native locks use admitted read-only handles, avoiding per-user lock chmod/chown of sealed ancestors. These are source findings; native API/ACL enforcement remains unobserved here.
- **Installed launcher image:** `PrepareOpenRGBNativeLauncher` accepts no caller source path/PID. `OpenInstalledNativeSupervisorImage` selects the installed protected boot definition, verifies its image and rechecks boot identity. Staging copies the opened image, verifies bounded exact bytes and native ABI, seals and publishes exclusively. Generic Windows process image lookup is not substituted for this source authority.
- **Closure/ABI/prerequisites:** complete tree hashes cover libraries/plugins/frameworks, not only payloads. Native ELF/Mach-O/PE headers and Apple platform/minimum commands are parsed offline. Linux prerequisite traversal covers each native image's external providers and symbol versions; safe system-root/provider ancestry is checked. Darwin requires macOS15 and active-cache metadata, while no-CGO remains explicitly unsupported. Windows requires native x64/build17763+, all six System32 signed Microsoft v14.44 providers and exact imported named exports, plus OS UCRT. ARM Windows/emulation remain unsupported. Runtime loaders and native ABI behavior remain qualification obligations.
- **Opt-out/configuration:** native fact selection avoids GPU/operator preference inventory. Acquisition/source selection and preparation remain explicit; preparation does not enable/start/acquire/probe. OpenRGB adoption does not run arbitrary candidates. Constructor roots bind both APPDATA and XDG_CONFIG_HOME before CLI configuration; auto profiles/remotes are disabled. Windows PE manifest languages require asInvoker/uiAccess=false. Native resource statuses remain unqualified, rather than converting source/build evidence into runtime support.
- **Archive and canonical hashing:** ZIP path/root/alias/type/size checks precede writes; symlinks are delayed and every destination parent is checked as an actual directory, including queued-link cases. Tree hashing reads every regular leaf and safe in-tree link; its canonical leaf serializer is private and has no unreadable-leaf substitution route. Malicious archive input still cannot publish without fixed closure checks.

## Independent desired-behavior counterexamples

The new review-local Go overlay adds three tests without modifying production/package files. Source: `.vrooli/work/native-artifact-review-l/counterexamples_test.go`; overlay map `overlay.json`; complete Go events `counterexamples.jsonl`. All three tests passed, none skipped, package1.226s.

1. `TestAdversarialLDerivedSourceIsNeverReadoptable`: copy the actual complete official Windows source into private storage, derive the serving closure, then attempt original-source adoption and a second derivation. Both refuse. Rejection leaves the derived digest unchanged. A dangling `LpcIO.bin` alias also fails module-absence proof. The actual installed/research original remains the complete original digest.
2. `TestAdversarialLRecoveryRefusesRetainedIdentitySwap`: publish a private actual-byte serving fixture, replace its retained source snapshot, then retry preparation. Recovery refuses while preserving the valid derived serving digest.
3. `TestAdversarialLMetadataBindsVersionAndUnchangedPayload`: create exact sealed-profile bytes for envelope selection, then request another version and the adopted-payload source identity. Neither can reuse those metadata bytes.

These counterexamples were chosen from the current source boundaries, independently of K, rather than importing another review's findings.

## Validation actually executed

Both VROOLI_ROOT and VROOLI_SOURCE_ROOT pointed to the isolated source root on every source-aware command. Actual offline release tests used the retained isolated official envelopes; MSVC inventory used the retained complete Windows bundle. No network acquisition occurred.

| Check | Observed result | Retained events |
| --- | --- | --- |
| Resources `-run 'OpenRGB|WindowsHID|MSVC|ManagedServiceAcquisition' -count=1 -timeout=180s` | PASS22.693s;301 test/subtest pass events;4 explicit opt-in skips | `resources.jsonl` |
| Independent L overlay `-run '^TestAdversarialL'` | PASS1.226s;3 tests, no skips | `counterexamples.jsonl` |
| Read-only existing Linux envelope adoption | PASS0.075s; exact adoption test ran with explicit `VROOLI_OPENRGB_ADOPTION_ENVELOPE` | `adoption.jsonl` |
| Full binaryfetch module | PASS0.168s;100 test/subtest pass events | `binaryfetch.jsonl` |
| Full resource-deployment module | PASS0.191s;73 test/subtest pass events | `resource-deployment.jsonl` |
| Full MSVC prerequisite module with actual bundle inventory | PASS0.138s;7 tests, no skips | `msvc.jsonl` |
| Resource manifest `-run 'OpenRGB|Platform|Resource'` | PASS0.143s; selected platform/unqualified/schema/update contract tests ran | `manifest.jsonl` |

Paths in the last column are relative to `.vrooli/work/native-artifact-review-l/`. Go's command packages have no tests and are not counted as passing test cases. The four resource skips were existing Linux-envelope adoption, pinned Linux extraction, retained normalization evidence and native Linux prerequisite metadata. Adoption was separately executed successfully as shown. The other three remain skipped in this review and are not claimed here. Profile measurement, Windows original/derived fixtures, all new counterexamples and actual PE import inventory ran. This review did not rerun native cross-compilation or the shared-package baseline; their previously reported results are context only.

## Changed, verified, remaining and unverified

Changed: this report, review-local evidence/overlay artifacts and required work-record journal entry `f008c49e-e077-4fe2-9688-3717471d4cf8` only. No production source, main-worktree file, Git state, dependency, installation, native service/task/daemon, permission, device or SDK/vendor process was changed or executed. No persistent process/service/model was started.

Verified: the source boundaries and concrete private actual-byte counterexamples above, current selected owner suites, offline original/derived provenance, and read-only Linux adopted-envelope identity. No material artifact-source repair is requested from this pass.

Remaining/unverified: actual Darwin/Windows ACL, immutable path/lock, installed-boot image, loader/prerequisite/signature-cache, ordinary-token daemon/task and headless behavior; HID controller/SDK/output; native logout/boot/resume/reconnect and update/rollback effects. Native CGO builds and all target runtime/hardware qualification remain separate evidence. Missing observations are not source bugs, and this clean artifact pass cannot close R1–R3, L1–L4, full Q2 or the full-destination challenger.
