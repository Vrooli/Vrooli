# Device Studio

## What You Get

Device Studio is a reusable studio in development for showing authentic app recordings on realistic devices. It combines devices, scenes, presentation profiles and motion into saved shots, using photographic composition for eligible fixed views and full 3D for physical movement.

**Current state: the local workbench works; exhaustive release certification is deferred.** Create, reopen and revision-safely edit projects through UI/API/CLI, import original images/recordings, and save immutable shots. All six saved demonstration projects have direct outputs: photographic phone/laptop stills, recording-textured camera entry, a ten-second phone rotation, a ten-second laptop lid closure, and the remaining-family showcase. The four1080p SDR landscape/portrait24/30fps profiles have verified decode/timing evidence.

Ten scoped reviewed-starter installers have actual local install/replay evidence. The exact-output readability flow has actual low-resolution rejections, adequate-output acceptances, revised-source reviews, and restart/readback evidence. The creation workflow retains imported-model edits, local reference-based model construction/refinement, and generated photographic candidates; successful generation does not imply visual approval. The persisted local creation session has completed an actual UI submission, reload, bounded local inference, contained finite compilation, scope-change refusal, and cancellation preservation. The consolidated current AppImage passed offline import/save/render/download/close/restart/reopen/edit. Its bundled high-resolution CPU 3D export remains unverified after the validation harness's10-minute wait limit; that failed validation is not a confirmed renderer outcome. Focused affected-owner tests,47 physical-workspace/API UI tests, UI typecheck and the scoped Test Genie structure/dependencies/programs phases pass. Follow [the delivery queue](docs/internal/goal/QUEUE.md) for exact receipts.

Current Linux desktop artifact (private unsigned build):
`/home/matthalloran8/Vrooli/scenarios/device-studio/artifacts/desktop/Device-Studio-20261021-linux-x64.AppImage`.
Its adjacent `usable-build-receipt.json` retains the artifact SHA and practical
validation scope. This is a durable exported copy, not a dependency on a staging
cache path.

Run `make start` in this directory and open the UI URL printed by the lifecycle.
Local project/media work does not require sign-in. Select a saved device/scene
setup, import a capture, configure its presentation, save and compile the shot,
then preview or export. AI actions disclose the selected route and require
confirmation; local rendering does not silently use a hosted provider.

Known limits: bundled high-resolution CPU 3D export is not yet verified; CPU
exports can take several minutes, and GPU timings apply only to their measured
profiles. Actual host GPU final stills and ten-second physical-motion exports
are retained in the demonstrations. Some devices are explicitly generic or conceptual,
and reviews cover named poses and motions. Unknown mount/physical fit stays
needs-review; incompatible setups are blocked. Generated photographic scenes
can need rejection or correction. Cross-platform architecture is retained, but
runtime validation here covers this Linux host. Paid-service delivery and
exhaustive P0 certification are not claimed by the usable-build handover.

Accepted scope: eight representative device groups; phone rotation/laptop lid motion; guided and advanced editing; 1080p 24/30 fps SDR landscape/portrait video and source-supported stills; reference-based AI creation with review; local/BYOK/Vrooli-credit choices; Windows/macOS/Linux desktop architecture. Free model sourcing and external-app-media opt-in remain binding.

## Documentation Map

Start with [docs/START-HERE.md](docs/START-HERE.md). Product intent lives in [PRD.md](PRD.md), measurable obligations in [requirements/](requirements/README.md), intended UX in [DESIGN.md](DESIGN.md) and [experience/](experience/README.md). [Architecture](docs/concepts/ARCHITECTURE.md), [data](docs/concepts/DATA.md), [domains](docs/concepts/DOMAINS.md) and [integrations](docs/concepts/INTEGRATIONS.md) define ownership. [Testing](docs/internal/TESTING.md), [acceptance](docs/internal/ACCEPTANCE.md) and [problems](docs/internal/PROBLEMS.md) define the true frontier.

The original research folder is preserved: [research index](docs/research/README.md). Portable source snapshots and a hash inventory preserve the accepted decisions even when the original machine is unavailable. Canonical Plan Manager foundation plan: `device-studio-foundation`; use `plan-manager plans render device-studio-foundation` for its review view.

The fixed vehicle-cabin plate used built-in image generation; its complete
prompt, output hash and limits are retained in
[the generation provenance](examples/demos/assets/vehicle-cabin-v1-provenance.json).
Its CarPlay/Android Auto labels illustrate an app-only capture in a generic
parked cabin; they do not certify native platform integration or vehicle
hardware dimensions.

## Customize Safely

Run lifecycle operations through `make setup|start|status|logs|stop`. Use scoped Test Genie phases for changes. All package work goes through SDA. Public contracts are proto-owned; generated artifacts are never hand-edited. Preserve library shell/i18n/accessibility and domain-first structure. Keep claims scoped to their evidence. Do not delete research, silently introduce paid calls or upload real media without project opt-in. The Notes template was removed through its owner; retained experiments and baselines remain preserved. Generated delivery evidence, runtime state and caches are ignored by Git, while curated research, source assets and documentation remain tracked.
