# Independent native artifact review N

2026-11-11. Reviewer: `/root/native_lighting_portability/native_artifact_profiles/artifact_review_n`.

## Decision

No material source finding remains in this reopened complete-Mach-loader artifact component at the reviewed source point. This is the second fresh independent component review, performed from current source without presuming the result of M. It does not close native execution, hardware qualification, or the delivery charter.

A documentation precision issue was found and corrected by the artifact owner during review: only the ARM ZIP has an app-root `Contents/_CodeSignature/CodeResources` referencing the omitted framework; the Intel ZIP has no such root resource envelope. The canonical loader proof now states that distinction. No product code needed a change, and neither architecture gains an unchanged upstream whole-app signature, notarization, or Gatekeeper claim.

## Inputs and source identity

Read the main delivery CHARTER, SPEC, STATUS, FEEDBACK and product-resumption native gap audit, then the four requested current artifact/headless/loader documents. The accepted cross-platform destination and operator stance remain the denominator. Work stayed in `/home/matthalloran8/Vrooli-native-lighting`; source-aware commands explicitly set both root variables to that directory.

The retained 190-file source/config/test manifest is `.vrooli/work/native-artifact-review-n/source-manifest.json`. Its SHA256 identity, calculated over sorted compact JSON, is `40135b8958ef8ed57a15cc84a15024cea5c7783e85ebf72e51f76612b2ac3707`. It covers OpenRGB Go files, selected managed acquisition/adoption files, shared Mach parser/test files and resource config. Every captured file remained byte-identical at final comparison. This is a bounded component fingerprint, not a whole-repository snapshot. The owner made the disclosed canonical documentation correction; documentation was excluded from that production-input fingerprint.

## Independent challenges and evidence

Four new desired-behavior probes were retained privately in `.vrooli/work/native-artifact-review-n/counterexamples_test.go`, with `overlay.json`. They run through a Go source overlay, without modifying production or checked-in tests:

- A held reader with a valid backing byte slice but a shorter declared extent must not consume a command from the suffix. Three truncation boundaries are refused.
- Each challenged alternate weak-dylib flag combination (1, 2, 4, 8 and 15), including all four flags together, must retain its required system-provider cache check. A missing provider is refused for each case.
- A loader-relative dependency symlink to a valid native dylib outside the bundle must be refused.
- An exact three-name stage inventory must still refuse a retained-source symlink to another location.

`go test -count=1 -race -overlay ... ./internal/resources -run '^TestReviewN'` passed in **1.031s**. Log: `.vrooli/work/native-artifact-review-n/counterexamples.log`.

An independent Python ZIP/raw-load-command audit, without extracting or running vendor code, examined both pinned Mac archives. Each contains **78 original images / 77 retained serving images**. All six exact dbus/jasper/jpeg images have no `S_MOD_INIT_FUNC_POINTERS`, `S_INIT_FUNC_OFFSETS` or `LC_ROUTINES_64` initializer. Retained metadata includes each image hash, declared imports, runpaths and command counts in `.vrooli/work/native-artifact-review-n/actual-mac-independent.json`. A scan for `QtCore5Compat` outside its own subtree found only ARM CodeResources; Intel had no external occurrence and no root CodeResources. This supports the bounded omission rationale; it does not prove absence of arbitrary computed dynamic loads.

The current official-byte regression independently reran as part of the broad resource suite. It verifies complete original/derived digests, exact inert-image symbol and loader-name exclusions, all native image ABI metadata, both 77-image serving graphs, every system cache provider, and adjacent-provider refusal. Managed and adopted recovery tests reran with original preservation, unknown publication acknowledgement and exact metadata checks.

`go test -count=1 ./internal/resources -run 'OpenRGB|WindowsHID|MSVC|ManagedServiceAcquisition'` passed in **55.467s**, with both original Mac assets, Windows MSVC bundle and explicitly opted-in read-only existing Linux adoption envelope. Log: `.vrooli/work/native-artifact-review-n/resource-suite.log`. These are package/fixture proofs, not an operated native lifecycle.

Shared platform affected race tests, `go test -count=1 -race -run 'MachO|Mach' ./...` from `packages/platform-go`, passed (**1.223s** for the platform root; other subpackages reported no matching tests). Log: `.vrooli/work/native-artifact-review-n/parser-suite.log`.

## Reviewed boundaries

The shared parser reads a bounded held-reader extent and reports normal, weak, reexport, upward, lazy and alternate-encoding dylib facts. Unknown/malformed forms, custom dyld and embedded environment fail closed under resource policy. Vendor metadata parsing remains separate from the maintained host's system-only executable policy.

The resource graph requires private targets inside the pinned bundle, canonical target agreement across private candidates, and a usable self/main runpath. A foreign runpath cannot reach an `@rpath` descendant, including through cycles. The six Cellar-runpath exceptions bind exact image hashes, paths and arch-specific runpaths; these fixed C-image imports and initialization evidence support declared-loader-graph inertness. They do not establish arbitrary-code or arbitrary-`dlopen` confinement. System dependency forms all participate in the cache predicate.

The 22 plugin sibling paths resolve only to the exact stage-adjacent `lib`, which must be absent. The exact stage inventory plus independently verified immutable ancestry is essential; the graph helper alone is not a writable-filesystem confinement mechanism. Original source proof precedes the fixed omission; derived tree, recipe, retained selected-source metadata and stage namespace remain distinct. No mutable original installation becomes serving authority during adoption.

Publication verifies content before sealing and exclusive rename; unknown durable acknowledgement remains a recoverable error. Recovery rechecks the complete derived closure, exact metadata and stage layout. Native locks use admitted immutable handles rather than the ordinary per-user lock mutator. Launcher preparation opens the protected installed supervisor image and copies verified bytes into immutable code storage; the maintained platform opener retains its own executable policy. Child CWD is the sealed bundle. Windows module absence remains a separately pinned 16-PE serving derivation after complete 22-file original verification; no same-SID denial fiction is reintroduced. Automatic acquisition/start/hardware probing is not introduced by platform profile selection; explicit source and lifecycle owners remain required.

## Changed, verified, remaining and unverified

Changed: this report and private review evidence/overlay only. The artifact owner corrected the architecture-specific signature prose. No production changes, Git mutation, package acquisition/install, root action, service launch, vendor/SDK execution or device effect was performed. All review-started Go commands completed; no service or model was started.

Verified: bounded component source review; four independent negative probes; official-byte Mac metadata/omission audit; broad opted-in resource regressions; affected shared-parser race tests; unchanged captured source inputs. Prior cross-build reports are context and were not rerun by N.

Remaining/unverified: native Mac CGO/cache/loader/AMFI/Gatekeeper/ACL/launchd and Windows ACL/token/task/loader/optional NvAPI/ADL behavior, actual headless constructors, narrow HID access, SDK operation, physical Off/presets, boot/logout/resume, real update/rollback, missing-target observations, and the full H/W/R/L/Q destination. Windows ARM remains unsupported. Existing disclosed shared-package baseline failures are not made green by this focused pass. Source-clean component evidence never substitutes for those obligations.

Work-record journal: `59be2435-a0fd-4764-989c-fcbc653f5132`.
