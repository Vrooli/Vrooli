# Independent native artifact review M

2026-11-11. Reviewer: `/root/native_lighting_portability/native_artifact_profiles/artifact_review_m`.

## Decision

No material source finding remains in the reviewed reopened complete-Mach-loader and OpenRGB artifact component. This is a fresh clean pass of the current derived/operator source, not reuse of K/L and not qualification of native execution or the full lighting destination. No production source changed in this review.

Read the main delivery CHARTER, SPEC, STATUS, FEEDBACK and native gap audit before substantive work. The full destination and operator stance remain applicable. Read the four assigned native artifact/headless documents, `docs/TESTING.md`, and the writing-standards skill. Recall succeeded with `source-ledger.scopes` degraded by its request deadline; that provider's missing results were not treated as a complete inventory. No native target, vendor process, device, SDK, installation, acquisition, service or Git mutation was performed.

## Source identity and applicability

Retained evidence: `.vrooli/work/artifact-review-m/`. `source-before.json` captures the initial 257-file component view. `fullscope-source.json` captures 751 files across internal/resources, platform-go, binaryfetch, the resource schema, MSVC tool and OpenRGB resource. `production-source.json` retains the 741 non-documentation files with canonical-map SHA256 `6f7c85667366d646dbfd1bbb2d20b91e134789d1972e645e55db38be532f8fba`. Rechecking each retained production file found no changed bytes. The owner updated MACOS-COMPLETE-LOADER-PROOF.md and NATIVE-ARTIFACT-PROGRESS.md during this review; the current loader-proof text was reread. These documentation updates do not change the tested production inputs. This is a mutable-workspace observation, not a claim that hashing isolates test inputs.

Source-resolving commands used both absolute isolated roots. Some early literal file reads used no root override; those commands only opened explicitly named local files and did not resolve a project root or invoke a capability owner. All capability/Go executions explicitly set both roots to `/home/matthalloran8/Vrooli-native-lighting`.

## Adversarial traces

| Boundary | Trace and judgment |
| --- | --- |
| Shared held-reader parser | `native_owned_macho.go` owns binary decoding. The explicit extent is enforced by SectionReader, command bounds/alignment and exact command schemas; unknown commands refuse. All five dylib forms, alternate dylib-use flags, install identity, dylinker, runpaths and embedded environment become neutral facts. Resource policy remains separate from the host broker's system-only policy. Resource ABI validation calls the same parser on an opened file and additionally checks exact native CPU/subtype/byte order, file kind and minimum OS. |
| Private graph and cache | `openrgb_macho_loader.go` admits the exact system dyld and no embedded dyld environment. Every declared system load, including weak/reexport/upward/lazy forms, reaches cache admission. Private targets must canonicalize to inventoried images inside the bundle. An rpath request needs a usable self/main path; every other private candidate must agree on the canonical target. Foreign runpaths cannot reach an rpath-dependent descendant, including through cycles. This deliberately overapproximates ancestor candidates rather than emulating dyld search order. |
| Bounded runpath exceptions | Only six exact hash/path pairs admit the three C libraries' Cellar runpaths. Actual-byte tests check imports, loader-name bytes and initializer sections/commands, and graph traversal excludes descendant rpath requests. No arbitrary dlopen or arbitrary-code confinement guarantee follows. A Qt caller loading a plugin does not make these libraries' unrelated build paths providers in the declared graph. The 22 Qt plugin escape paths resolve only to the exact stage-parent/lib absence; the stage inventory and immutable ancestry must preserve that absence. |
| Original versus derived identity | `deriveServingTree` verifies the complete original tree and native ABI before the fixed framework omission, then verifies the entire separately pinned derived tree. The original adopted installation is copied and preserved. A matching payload cannot admit an already-derived or incomplete original source. Recipe JSON, source identity and selected envelope/payload identity are bound in read-only metadata. The recipe hash participates in the stage address. |
| Publication, recovery and old stages | `PrepareOpenRGBAppImageStage` and `prepareOpenRGBAppImageAt` validate the derived tree, retained source, metadata and exact stage layout before reuse. Pending extraction/derivation stays unpublished; exclusive publication separates durable acknowledgement from visibility. An acknowledgement failure recovers the verified stage without extracting again. A legacy complete original at the new address cannot be relabeled using valid new metadata, and recovery refuses rather than rewriting it. |
| Initial, update and rollback | `verifyOpenRGBRoleArtifact` always selects the current recipe-addressed stage and invokes the full serving gate. Initial preparation/observation, frozen update role loading, target preexec and retained rollback envelope opening all pass through that gate. Retained original source SHA alone does not authorize executing a legacy original app. Frozen source, catalog, configuration, physical grant and native claim remain independently checked. |
| Default opt-out and adoption | Checked-in service configuration leaves OpenRGB disabled. Public native actions require persisted enablement/source/policy before artifact observation and launch. Explicit host source selection fingerprints offline through the OpenRGB-specific branch before state persistence and bypasses the generic version-execution branch. Startup stages/acquires nothing. Native profile selection is metadata, not enablement. |
| Constructor and platform limits | XDG_CONFIG_HOME and APPDATA are set to private configuration before vendor construction; saved remote clients and automatic profiles remain disabled. The server branch's lack of QApplication construction is source evidence. Cocoa-only QPA contents, early Intel macUSPCIO opening, transport denial and native WindowServer/HID behavior remain explicitly separate. Windows retains the approved fixed six-PawnIO-module omission and unchanged 16-PE serving closure. Windows ARM remains unsupported. The exact child working directory is the sealed app directory; no mutable-runtime-CWD claim was used to reject the optional GPU DLL route. Native optional DLL resolution is still unqualified. |
| Signature truth | The Mac omission changes CodeResources coverage. Current documentation explicitly disclaims unchanged whole-app resource signature, notarization and Gatekeeper qualification. Unchanged remaining Mach-O bytes do not prove native execution acceptance. No signing bypass or binary patch is introduced. |

The resource scans the fixed release's thin little-endian Mach-O images. That is not a general arbitrary/fat-image inventory claim: the whole original/derived tree digests, exact ABI and current measured image inventory constrain the admitted release. No new bytes can enter through that scan without changing the pinned tree identity.

## Independent desired-behavior counterexamples

Review-only fixture `counterexamples_test.go` runs through Go's retained `overlay.json`; no production/test source path was edited. `counterexamples.log` records four top-level tests and 13 passing events with zero skips, PASS10.859s:

- Dylib-use flags 1, 2, 4, 8 and 15 cannot waive missing system-provider admission; unknown flag16 refuses.
- A present native dependency reached through an external symlink refuses for both direct loader-relative and rpath-relative loads.
- Both actual official Mac archives admit complete original payload adoption, then refuse the unchanged payload as original adoption after approved derivation; a second derivation also refuses.
- Both actual complete Mac originals placed at a serving-stage address with correctly bound new recipe metadata refuse recovery without extraction or mutation. The retained original tree remains byte-identical.

## Reexecuted proof

All logs retain verbose names and final package verdicts. No scenario suite or native effect was needed for this read-only source review.

| Evidence | Observed result |
| --- | --- |
| `resource-suite.log`: `go test ./internal/resources -run 'OpenRGB|WindowsHID|MSVC|ManagedServiceAcquisition' -count=1 -v` with all three assigned actual-byte environments | PASS58.600s; 348 pass events, zero failures, three explicit skips. Actual four new source profiles, both derived Mac trees/32 system-image graphs, original/adopted recovery, Windows derivation, MSVC imports, opt-out/configuration and read-only Linux adoption were selected. |
| `platform-parser.log`: platform-go MachO/Mach/DarwinBroker selection | PASS0.192s; 22 pass events, zero skips. Includes every load form, malformed/unknown commands, universal slices, exact reader extent and alternate dylib-use encoding. |
| `manifest-schema.log`: manifest Schema/ExclusiveStopStart/PersistentState/MigrationParity selection | PASS0.109s; 35 pass events, zero skips. |

The broad suite's explicit skips were the optional Linux pinned-stage extraction fixture, pinned runtime-normalization fixture, and live Linux OS-library metadata check. Their separate opt-ins were not set. They are not counted as proof. No native Mac/Windows runtime observation is inferred from any of these tests. The parent separately retains cross-target compile and prior baseline evidence; this review does not convert that evidence into executed-platform qualification.

## Changed, verified, remaining and unverified

Changed: this report and review-only evidence/overlay fixtures. Verified: current source traces, new independent counterexamples, actual retained artifact metadata/derivation/adoption/recovery, focused shared parser and manifest tests, and stable production-file fingerprints. No material finding needs repair from pass M.

Remaining: integrate the current component through the parent and preserve its native qualification matrix. Full product acceptance still requires actual target execution, native CGO/ACL/locking/signature behavior, boot/logout/resume, selected HID access, private SDK operation, physical Off/static/presets and recovery. Those missing observations are not newly discovered source defects, and source-clean M cannot close R1–R3/L1–L4 or the full CHARTER/SPEC. No process or service started by this review remains running.

Work-record journal: `3a6acc3d-2da5-4380-b33d-b5080c045eba`.
