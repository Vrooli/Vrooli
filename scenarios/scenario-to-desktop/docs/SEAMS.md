# SEAMS · scenario-to-desktop

## Current boundaries
- **API integration** now lives in `ui/src/lib/api.ts`. UI components call typed helpers for scenario inventory, quick-generate, build status, Wine checks, telemetry upload, and deletion instead of constructing fetches inline.
- **Presentation components** (`GeneratorForm`, `ScenarioInventory`, `GenerateDesktopButton`, `RegenerateButton`, `BuildDesktopButton`, `TelemetryUploadCard`, `DeleteButton`) focus on UX state and rendering; they do not assemble URLs or juggle transport concerns directly.
- **Control surface** stays narrow: components depend on `lib/api` contracts plus shared domain helpers (deployment modes, proxy requirements), keeping orchestration logic out of view rendering.
- **Download/installer actions** and **Wine install flows** are routed through `lib/api`, so renderer components no longer rebuild URLs or fetch wiring; API responses/types are centralized for tests and future backend changes.
- **Bundled packaging** now routes through a `bundlePackager` seam that owns manifest validation, staging, and runtime builds. The runtime builder and resolver are injected, so tests and future packagers can stub Go builds instead of invoking toolchains directly.

## Observations and next seams to tighten
- Runtime/build telemetry display depends on control API responses; keep that layer dedicated to consuming `lib/api` to avoid reintroducing transport details in UI components.
