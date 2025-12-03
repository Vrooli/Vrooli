# SEAMS · scenario-to-desktop

## Current boundaries
- **API integration** now lives in `ui/src/lib/api.ts`. UI components call typed helpers for scenario inventory, quick-generate, build status, Wine checks, telemetry upload, and deletion instead of constructing fetches inline.
- **Presentation components** (`GeneratorForm`, `ScenarioInventory`, `GenerateDesktopButton`, `RegenerateButton`, `BuildDesktopButton`, `TelemetryUploadCard`, `DeleteButton`) focus on UX state and rendering; they do not assemble URLs or juggle transport concerns directly.
- **Control surface** stays narrow: components depend on `lib/api` contracts plus shared domain helpers (deployment modes, proxy requirements), keeping orchestration logic out of view rendering.

## Observations and next seams to tighten
- Wine installer dialog still performs its own fetches; it could share the `lib/api` helpers to keep all system checks in one place.
- Download links (platform chips/cards) still build URLs directly; a small download helper would fully isolate URL assembly from presentation.
- Runtime/build telemetry display depends on control API responses; keep that layer dedicated to consuming `lib/api` to avoid reintroducing transport details in UI components.
