# Replay Style Architecture

## Purpose
Replay styling is treated as a first-class domain so the execution viewer, export player, and live preview share the same rendering primitives, theme catalog, and persistence logic. This prevents drift between presentation contexts and keeps styling changes predictable.

## Boundaries
- **Model**: Canonical schema + normalization logic lives in `ui/src/domains/replay-style/model.ts`.
- **Catalog**: Theme metadata + decor builders live in `ui/src/domains/replay-style/catalog.tsx`.
- **Renderer Primitives**: Shared UI building blocks live in `ui/src/domains/replay-style/renderer/`.
- **Adapters**: Storage, API, and spec mappings live in `ui/src/domains/replay-style/adapters/`.
- **Hook**: State resolution + persistence lives in `ui/src/domains/replay-style/useReplayStyle.ts`.

## Precedence Rules
1. Explicit overrides (execution/export scoped)
2. User settings/presets
3. System defaults

The merge happens in `resolveReplayStyle()` with normalization applied in `normalizeReplayStyle()`.

## Theme Catalog
Theme metadata and visuals are defined once in `catalog.tsx`.
- Chrome themes (browser frame)
- Background themes (canvas backdrop)
- Cursor themes (pointer visuals)
- Cursor positions and click animations

To add a theme:
1. Add the new theme id in `model.ts`.
2. Add its metadata and preview in `catalog.tsx`.
3. Add its decor builder to `buildBackgroundDecor`, `buildChromeDecor`, or `buildCursorDecor`.

## Renderer Primitives
- `ReplayStyleFrame`: wraps background + chrome framing.
- `ReplayCanvas`: handles canvas sizing, zoom, and screenshot placement.
- `ReplayCursorOverlay`: renders cursor visuals and trails.

These primitives are consumed by `ReplayPresentation` and the replay shell.

Watermark rendering stays in the exports domain: `ReplayStyleFrame` accepts an `overlayNode` so callers can inject `WatermarkOverlay` without coupling the replay-style domain to exports or settings types.

## Presentation + Playback
Replay styling and layout now flow through a single presentation model:
- `useReplayPresentationModel`: resolves the normalized style, theme tokens, and layout for a given canvas + viewport.
- `ReplayPresentation`: renders the styled frame + overlay and accepts a child renderer (screenshots or live content).
- `ReplayPlayer`: a shell that composes playback controls, metadata, and storyboard around the shared presentation.

Live preview and replay preview both use the same presentation path to prevent layout drift.

## Persistence
- **Local storage**: `adapters/storage.ts` stores the canonical config and reads legacy keys.
- **API**: `adapters/api.ts` maps the canonical style config to `/replay-config` and splits non-style fields into `extraConfig`.
- **Spec**: `adapters/spec.ts` maps the style config into `ReplayMovieSpec`.

The API adapter exposes `fetchReplayStylePayload()` to return `{ style, extraConfig }` so callers can keep styling and non-style config in sync without duplicating parsing logic.

## Extension Notes
- Avoid reintroducing styling logic outside the replay-style domain.
- Prefer updating the catalog and primitives over per-consumer tweaks.
- Keep new config fields backwards-compatible with normalization.
