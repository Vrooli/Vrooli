# Device Studio

## What You Get

Device Studio is the planned reusable studio for showing authentic app recordings on realistic devices. It combines devices, scenes, presentation profiles and motion into saved shots, using photographic composition for eligible fixed views and full 3D for physical movement.

**Current state: working local project, photographic rendering and initial physical rendering workflows; full release remains in development.** Create, reopen and revision-safely edit projects through UI/API/CLI, import original images/recordings, and save immutable shots. The integrated photographic workflow has bounded still/video, cancellation, restart and verified-download evidence. Public physical phone stills and a ten-second 1080p rotation with a fixed screen image have passed current-host checks. Recording textures, articulated laptop motion, the complete qualified library, creation workflows and standalone desktop delivery remain unfinished. Draft physical previews work but still miss the agreed ten-second gate. Follow [the delivery queue](docs/internal/goal/QUEUE.md) for exact evidence and remaining work.

Accepted scope: eight representative device groups; phone rotation/laptop lid motion; guided and advanced editing; 1080p 24/30 fps SDR landscape/portrait video and source-supported stills; reference-based AI creation with review; local/BYOK/Vrooli-credit choices; Windows/macOS/Linux desktop architecture. Free model sourcing and external-app-media opt-in remain binding.

## Documentation Map

Start with [docs/START-HERE.md](docs/START-HERE.md). Product intent lives in [PRD.md](PRD.md), measurable obligations in [requirements/](requirements/README.md), intended UX in [DESIGN.md](DESIGN.md) and [experience/](experience/README.md). [Architecture](docs/concepts/ARCHITECTURE.md), [data](docs/concepts/DATA.md), [domains](docs/concepts/DOMAINS.md) and [integrations](docs/concepts/INTEGRATIONS.md) define ownership. [Testing](docs/internal/TESTING.md), [acceptance](docs/internal/ACCEPTANCE.md) and [problems](docs/internal/PROBLEMS.md) define the true frontier.

The original research folder is preserved: [research index](docs/research/README.md). Portable source snapshots and a hash inventory preserve the accepted decisions even when the original machine is unavailable. Canonical Plan Manager foundation plan: `device-studio-foundation`; use `plan-manager plans render device-studio-foundation` for its review view.

## Customize Safely

Run lifecycle operations through `make setup|start|status|logs|stop`. Use scoped Test Genie phases for changes. All package work goes through SDA. Public contracts are proto-owned; generated artifacts are never hand-edited. Preserve library shell/i18n/accessibility and domain-first structure. Keep claims scoped to their evidence. Do not delete research, silently introduce paid calls or upload real media without project opt-in. The Notes template was removed through its owner; retained experiments and baselines remain preserved. Generated delivery evidence, runtime state and caches are ignored by Git, while curated research, source assets and documentation remain tracked.
