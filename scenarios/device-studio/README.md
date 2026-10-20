# Device Studio

## What You Get

Device Studio is the planned reusable studio for showing authentic app recordings on realistic devices. It combines devices, scenes, presentation profiles and motion into saved shots, using photographic composition for eligible fixed views and full 3D for physical movement.

**Current state: working project workspace and image intake; saved-shot rendering is in development.** Create, list, reopen and revision-safely rename local projects through UI/API/CLI. Import and inspect original PNG/JPEG media. These journeys have scoped tests and isolated browser evidence. A private photographic still worker and shared process owner have focused qualification; they are not yet connected to the editor. The full library, physical motion, AI workflow and standalone desktop release remain pending. Follow [the delivery queue](docs/internal/goal/QUEUE.md) for the current implementation boundary.

Accepted scope: eight representative device groups; phone rotation/laptop lid motion; guided and advanced editing; 1080p 24/30 fps SDR landscape/portrait video and source-supported stills; reference-based AI creation with review; local/BYOK/Vrooli-credit choices; Windows/macOS/Linux desktop architecture. Free model sourcing and external-app-media opt-in remain binding.

## Documentation Map

Start with [docs/START-HERE.md](docs/START-HERE.md). Product intent lives in [PRD.md](PRD.md), measurable obligations in [requirements/](requirements/README.md), intended UX in [DESIGN.md](DESIGN.md) and [experience/](experience/README.md). [Architecture](docs/concepts/ARCHITECTURE.md), [data](docs/concepts/DATA.md), [domains](docs/concepts/DOMAINS.md) and [integrations](docs/concepts/INTEGRATIONS.md) define ownership. [Testing](docs/internal/TESTING.md), [acceptance](docs/internal/ACCEPTANCE.md) and [problems](docs/internal/PROBLEMS.md) define the true frontier.

The original research folder is preserved: [research index](docs/research/README.md). Portable source snapshots and a hash inventory preserve the accepted decisions even when the original machine is unavailable. Canonical Plan Manager foundation plan: `device-studio-foundation`; use `plan-manager plans render device-studio-foundation` for its review view.

## Customize Safely

Run lifecycle operations through `make setup|start|status|logs|stop`. Use scoped Test Genie phases for changes. All package work goes through SDA. Public contracts are proto-owned; generated artifacts are never hand-edited. Preserve library shell/i18n/accessibility and domain-first structure. Keep claims scoped to their evidence. Do not delete research, silently introduce paid calls or upload real media without project opt-in. Notes removal is pending an evidence-preserving correction to the template owner's detemplate scan; do not run destructive detemplate against preserved baseline copies.
