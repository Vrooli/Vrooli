# Native artifact adversarial pass D — 2026-11-11

## Verdict

**Finding-bearing pass. D1 identified an archive containment defect in queued symlink creation; its owner repair was independently rechecked.** This pass cannot count toward consecutive clean passes. The production source verifies a fixed envelope checksum before extraction, and the retained official archives do not contain the counterexample. D1 therefore establishes a broken defensive extraction invariant, not a demonstrated way to substitute malicious bytes for an admitted release. No additional material defect was established in the reviewed artifact boundary.

Reviewer: `artifact_review_d`, independently reviewing isolated `/home/matthalloran8/Vrooli-native-lighting`. Read the main delivery CHARTER, SPEC, STATUS, FEEDBACK and native gap audit, plus isolated NATIVE-RELEASE-PROFILES, NATIVE-ARTIFACT-PROGRESS, pass C and docs/TESTING.md. Applied the writing-standards skill. This review covers artifact acquisition, extraction, adoption, config, ABI/prerequisite checks, sealed native artifacts and launchers, locks, denied-module receipts, declarations and SDA ownership. It does not accept the full delivery destination. Source remained shared and actively changing; the checks below establish the inputs actually observed, rather than an immutable repository revision.

## D1 — link publication can traverse an earlier archive link

Location at discovery: `internal/resources/openrgb_release_zip.go:122–126`.

The extractor delays symlink creation until regular writes finish. However, it then creates each queued link with `owner.mkdir(filepath.Dir(l.name))` and `os.Symlink`, without rejecting a parent that an earlier queued link made into a symlink. The lexical target check at lines 76–80 cannot account for that parent substitution.

Concrete private fixture: create an existing directory `outside` beside extraction destination `bundle`, and supply these three ZIP symlinks in order:

| ZIP entry | Target |
| --- | --- |
| `OpenRGB.app/A` | `.` |
| `OpenRGB.app/A/B` | `../outside` |
| `OpenRGB.app/A/B/C` | `.` |

Each target passes its lexical containment check. `A/B` actually becomes `bundle/B -> ../outside`, because `A` already points to the bundle root. Creating `A/B/C` then follows that parent and creates `outside/C`. The extractor returns success. An already existing external directory is sufficient; the test uses only a temporary sibling directory and never writes outside its private fixture root.

Independent reproduction used an extra test injected with Go's `-overlay`, with the physical test source and overlay map under `/tmp`, so the product tree was not edited. `/tmp/openrgb-review-d-chained-links.log` records `extract returned success and created external symlink outside/C=.`. This observation test passes when the defect occurs; it is not a passing rejection regression. The first experiment, with no pre-existing sibling directory, correctly failed at `MkdirAll`; the final fixture supplies the concrete condition needed to traverse the chain.

Expected behavior: Given queued archive links with a symlink ancestor, when the extractor publishes those links, then it must refuse before creating anything through that ancestor. Reject link ancestors with an `Lstat` component walk or an equivalent no-follow construction, retaining support for the actual macOS framework links. Verify the actual release measurement afterward. Full-tree verification after extraction does not undo an external write that already occurred.

Scope and severity: this is a defensive containment/failure-isolation defect and an inaccurate universal extractor guarantee. The production caller first snapshots and hashes the exact pinned release, and the actual-byte measurements passed, so the fixture does not demonstrate a current untrusted public input reaching this path. The adopted-bundle copy also verifies its source tree first; no separate adopted-source escape was established. D1 was routed to the artifact owner with the exact counterexample and reachability limitation before this report was written. Repair is owned by that implementation owner.

Repair independently rechecked: the owner now prepares all queued link parents before creating any links and verifies actual root-to-parent directory components with `Lstat` before link creation. Archive aliases remain leaves. The same ancestry helper protects delayed adopted-bundle links. The original observation fixture now fails its old success expectation at the first attempted alias, before any outside link is created. Independently changing that private fixture to require refusal and an empty outside directory passes (0.008s). The committed `TestOpenRGBZipNeverCreatesLinksThroughQueuedLinkParents`, existing ZIP/adoption checks and all four retained-profile measurements also pass together (1.382s; measurement 1.36s). This preserves the actual framework-link layouts while rejecting the counterexample. Logs: `/tmp/openrgb-review-d-{chained-links-after,rejection,repair}.log`. Pass D remains finding-bearing after this repair.

## Source boundary assessment

The profile matrix closes over exactly Linux amd64/arm64, Darwin amd64/arm64 and Windows amd64. Windows ARM64 remains unsupported. The new host-admission layer preserves pure cross-target measurement while requiring native Windows x64 via `IsWow64Process2` and rejecting Darwin Rosetta via `sysctl.proc_translated`; native API behavior remains unobserved here. The generic download-size helper now rejects unrelated resource names before consulting OpenRGB native host metadata.

Acquisition has exact byte bounds and digest identities, frozen manifest admission and source-install governance. Offline extraction does not execute an AppImage or installer. Whole-tree, payload, image ABI and Windows startup-manifest proof remain separate from version strings. Explicit host adoption verifies the complete enclosing bundle before copying and preserves the original. Constructor XDG_CONFIG_HOME and APPDATA isolation precede CLI config selection, with fixed detector/remote-client/profile policy and ambient-config preservation tests.

Darwin and Windows code stores remain separate from mutable capsule/admission state. Darwin uses the canonical root-owned PrivilegedHelperTools namespace and native ACL verification. Windows uses ProgramFiles, atomic machine-owned constructors, protected exact operator/service reader roles, reparse/hardlink rejection and closure-wide checks. Both initial and update launcher paths call the sealed launcher publisher. Native code-store locks open admitted read-only handles and reverify identity/seal after locking; they do not invoke the generic per-user chmod/chown lock preparation. These are source findings and cross-compilation evidence, not observed OS enforcement.

The Windows six-module scheme fixes release names, sizes and digests. The owner rehashes complete bytes before recording native file/volume/time identity in authenticated capsule material. The exact child runtime must verify those identities, exact metadata-only access, actual FILE_READ_DATA denial, every readable leaf and the canonical reconstructed tree. Arbitrary caller hashes or omitted-file lists do not satisfy this path. The complete official module table independently matched retained bytes. This does not establish actual OpenRGB compatibility with denied PawnIO modules or native security API correctness.

Runtime preflight remains separate from artifact admission: Linux checks the native interpreter/system version providers across the closure; Darwin checks OS baseline and active dyld cache metadata, with no-CGO refusal; Windows checks native System32 MSVC ABI/version/signature identity/imports and separate UCRT. The maintained SDA system-tool exception remains restricted to the named msvc-runtime handler, rather than arbitrary special handlers. No dependency acquisition or installation was performed during this review.

## Independent checks

Every test, cross-build and discovery command exported both VROOLI_ROOT and VROOLI_SOURCE_ROOT to the isolated source root. Source reads used that explicit worktree. No native executable from a retained release was run.

- Resources artifact/config/capsule selection PASS, 2.853s. It includes actual retained-profile measurement (1.41s), exact six denied-module byte checks, stage publication/recovery/cancellation/tamper refusal, adopted-source preservation, config isolation/recovery, exact capsule material/launcher mismatch refusal, one-claim/uncertainty handling and transport-confinement refusal. Native Linux system-library inspection deliberately skipped because its explicit live flag was absent.
- Actual-byte profile measurement passed all four new profiles: Linux ARM64 retained extracted tree, both freshly offline-extracted Darwin ZIPs and Windows ZIP. Complete tree/payload/image ABI checks and Windows embedded startup manifest passed; each payload also contains the exact selected ASUS Aura Motherboard registration. Existing Linux amd64 release constants were source-reviewed; this four-new-profile measurement does not remeasure its archive.
- Additional resources selection PASS, 0.298s: complete closure seal/tamper checks, staged serving/root/invocation proof, disabled-detector pruning, confined update execution, stale manifest refusal, exact download bound and first-launch config isolation.
- Full resource manifest package PASS, 0.250s, including the repaired truthful `unqualified` declaration path. Full MSVC runtime package PASS, 0.003s. This Linux execution does not exercise Windows signature/API implementation.
- SDA releasequalification PASS, 0.022s; installgateway PASS, 0.726s; dependencygovernance PASS, 0.610s. Live official acquisition remains an explicit skipped opt-in, not a fresh download performed here.
- Resource test binaries cross-compiled successfully for Windows amd64, Darwin amd64/arm64 and Linux arm64 with CGO disabled, to private `/tmp/openrgb-review-d-*` outputs. These are compilation checks only; Darwin CGO/SDK code was not compiled or linked here.
- The D1 private overlay observation reproduced the external symlink creation in 0.008s. No root, native OS, service, device or target-dispatch operation was needed.

Local corroboration logs: resources SHA256 `88f460baa825918206a47d4e3a21db4a39998e013756b5c11d3e9ccad4766449`; extra checks `c826c62794a72d0dfc91bf02efbe342a9d0f0a6de8705f4b1edf17c7700f8e93`; SDA `210a6f861a60fc3122512f76067c4b40a8e0c1ac7733f3b002771ddc24deec4e`. Paths are `/tmp/openrgb-review-d-{resources,extra,sda}.log`; these temporary files are corroboration, not a durable evidence service.

Source fingerprints at finding time: release profile `4d65042f10a374f557e02d91bc8d95ba2f8e20764d146f21a8b6b1d8a5c65969`; ZIP extractor `d50a389bcbd270a1c54b189188f9aeb7f7a325200e83e0fb8fe7c0e3c9837459`; blocked-module Windows verifier `46f93a26a6aa1e59555594c5eed66c78e32a82e83f0d3c1ca7042fea17d7eb1a`; stage lock `9b747c722843174eeba9d54ea31221c087086d67d25c7fea995ed086324ffff0`; native launcher publisher `bad14d44fb28366e8a83a6fe68a667441cf50a6e827390f26117f1d8c0a9954e`.

Recall discovery succeeded with degraded providers: agent-manager semantic retrieval, docs providers, prompt-manager skill search, source-ledger and swarm records timed out. Returned hits did not add artifact-specific evidence. The direct maintained writing-standards skill read succeeded. No conclusion about capability absence was drawn from this degraded search.

## Changed, verified, remaining and unverified

Changed in the product tree: this report only. Tests created private fixtures, temporary logs/overlay source and four cross-compiled test binaries. All test/build processes completed. No Git mutation, dependency installation, root adoption, service/device/installer action, target dispatch or grant change occurred. No model or persistent service was started.

Verified: D1's exact helper counterexample, current retained byte/profile consistency, scoped desired-behavior fixtures, manifest acceptance and four native target cross-compilations listed above. No second material artifact defect was established.

Remaining: fresh independent passes after the independently rechecked D1 repair. Repairing D1 cannot retroactively make pass D clean. Full destination acceptance and its fixed challenger remain separate.

Unverified: native Darwin CGO/SDK/link and Intel/ARM execution; Windows ACL/FileID/signature-cache/import behavior on target; elevated publisher and native lock/launcher failure recovery; actual governed prerequisite installation; denied-PawnIO startup compatibility; native service account/permissions, Session0/logged-out boot/resume/device return; physical Off/static/effects; complete Linux real-host lighting and wireless acceptance. Source checks and authentic artifacts do not discharge those obligations.
