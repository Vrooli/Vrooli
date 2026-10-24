# Research and experiment preservation

The original research is retained **in place**, not moved or replaced:

`~/.vrooli/evidence/device-presentation-2026-10-18`

[Original inventory](original-inventory.json) pins 3,495 files / 2,039,219,332 bytes with SHA-256 hashes before this foundation. [Snapshot manifest](snapshot-manifest.json) maps byte-identical portable text copies to their source paths. Large PNG/MP4/GLB, source model packages, native scene files, tools and receipts remain in the original folder. These snapshots are history, not a second product specification. Absolute historical paths resolve on the originating host; on another machine restore the full archive separately and verify it against the inventory. No independent backup or remote copy is claimed.

## Evidence map

| Stage | What it establishes | Important limit |
|---|---|---|
| Baseline and v1–v2 | Original capture/device experiments, model sourcing and broad domain inventory | Exploratory implementations, not production modules |
| v3–v4 | Device material editing, screen safe areas/status chrome and browser/OS presentation | Preserve exact provenance, source pixels and device-specific masking |
| v5 | Geometric desk/table environment experiments | Appearance was rejected by the user; keep as a regression comparison |
| v6 | Accepted photographic environment direction with authentic screen composition | Generated scene plus screen composite; not proof of the exact GLB placed in a reconstructed room |
| v7 | Source rights/dimensions, generalized devices, strategy/renderer/integration research | Recommendations were pending; later accepted decisions take precedence |
| v8 | Seven accepted decisions and desktop/monetization/identity follow-up | Integration gaps and synthetic probes are not deployed behavior |
| v9/foundation | Generation, preservation, document authoring and review evidence | No product implementation or craft qualification |

## Current authority

Read [decisions](../internal/DECISIONS.md), [PRD](../../PRD.md), [architecture](../concepts/ARCHITECTURE.md), [acceptance](../internal/ACCEPTANCE.md) and [known gaps](../internal/PROBLEMS.md). Preserve rejected outputs as well as accepted ones. New experiments append a new evidence version; do not overwrite the original inventory or silently replace source artifacts. Model licenses govern distribution independently of a free download; no model bytes were bundled into this scaffold.

## Portable source documents

- [v7/research/README.md](snapshots/v7__research__README.md.txt)
- [v7/research/brief.md](snapshots/v7__research__brief.md.txt)
- [v7/research/device-contract.md](snapshots/v7__research__device-contract.md.txt)
- [v7/research/device-domain-input.md](snapshots/v7__research__device-domain-input.md.txt)
- [v7/research/integrations.md](snapshots/v7__research__integrations.md.txt)
- [v7/research/product-validation.md](snapshots/v7__research__product-validation.md.txt)
- [v7/research/rendering.md](snapshots/v7__research__rendering.md.txt)
- [v7/research/source-inventory.json](snapshots/v7__research__source-inventory.json.txt)
- [v8/research/README.md](snapshots/v8__research__README.md.txt)
- [v8/research/brief.md](snapshots/v8__research__brief.md.txt)
- [v8/research/catalog-follow-up.md](snapshots/v8__research__catalog-follow-up.md.txt)
- [v8/research/decisions.json](snapshots/v8__research__decisions.json.txt)
- [v8/research/desktop-rendering.md](snapshots/v8__research__desktop-rendering.md.txt)
- [v8/research/device-domain-input.md](snapshots/v8__research__device-domain-input.md.txt)
- [v8/research/identity-credits.md](snapshots/v8__research__identity-credits.md.txt)
- [v8/research/implementation-approach.md](snapshots/v8__research__implementation-approach.md.txt)
- [v8/research/interaction-contract.md](snapshots/v8__research__interaction-contract.md.txt)
- [v8/research/monetization.md](snapshots/v8__research__monetization.md.txt)

## Source control and local delivery evidence

Canonical requirements, design, domain contracts, curated research snapshots, shipped renderer adapters and starter asset manifests belong in Git. Local delivery evidence under `docs/internal/goal/evidence/` is ignored: it contains captures, decoded/rendered frames, copied source trees and descriptors, compiled workers, native-tool archives, private routed-storage copies and run receipts. Keep that directory on the originating host; an ordinary clone does not contain it. Do not use Git cleanup commands to reclaim preserved research. Product code or required starter assets must be promoted into their owning source/asset directories rather than force-added from evidence.

On2026-10-20,805 previously tracked evidence files (74,708,818bytes) were removed from the Git index after all working bytes were copied and SHA-256 verified at `/home/matthalloran8/.vrooli/evidence/device-studio-delivery-2026-10-20/git-evidence-retirement/manifest.json`. The working evidence and original research folder remain in place. This is a same-host preservation copy, not an off-host backup. Goal ledgers retain scope/results and point to exact local evidence; the canonical domain contracts retain implemented behavior.


## Additional free model acquisition — 2026-10-20

Two E4 research candidates were acquired from SceneView's public model hosting.
Their embedded credits, SceneView's catalogue and original creator APIs agree on
CC BY 4.0. These are retained source bytes, not qualified or shipped Library
assets. Dimensions, screen mapping, renderer compatibility and any upstream
normalization still need review.

| Candidate | Creator and original listing | Retained GLB identity |
|---|---|---|
| Apple Watch Series 7 | [SDC PERFORMANCE creator API](https://api.sketchfab.com/v3/models/cf15ecbf159f42a28df5e56bdfa666bb) | 5,904,504 bytes; SHA-256 `ac3c33a32bc27c562e6655245e20327d17c77c988f11aab3016496d851e4e5e3` |
| iPad mini 6 (2021) | [DatSketch creator API](https://api.sketchfab.com/v3/models/f363c2b1c74a462588db184bd4fb024e) | 1,986,244 bytes; SHA-256 `a462088b7bd84b7752802c109beee04110047c3b1a4f355624c863f790e93d5e` |

[SceneView credits](https://sceneview.github.io/credits.html) distinguish model
licences from the SDK licence. Catalogue provenance is pinned to SceneView commit
`05943c35f07e8029bacd41780e951618e40dde5c`; the separately hosted model bytes are
pinned by the hashes above. This does not claim that those files were committed
in that Git tree. Local metadata, source API records and GLBs are retained under
`docs/internal/goal/evidence/E4/free-models/sceneview-research/`.
