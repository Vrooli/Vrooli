# Research Notes — from research/command-center-architecture

This file records guidance from `research/command-center-architecture/conclusion.md` that the kiosk-ux execution must incorporate.

## Controller input is already solved — consume, don't build (Action 5)

**Do not write Gamepad API plumbing.** The react-vite template (`templates/scenarios/react-vite/ui`) already depends on `@vrooli/iframe-bridge` and ships three ready-to-use React hooks:

- `useGamepad` — raw gamepad input with `navigator.getGamepads()` polling, connect/disconnect events, W3C Standard Gamepad mapping, dead-zone filtering.
- `useSpatialNav` — initializes `SpatialNavManager` on mount; 2-D directional focus (up/down/left/right) with geometric nearest-neighbor scoring; modes: `spatial`, `passthrough`, `grid`, `modal`.
- `SpatialGroup` — component wrapper that registers a focus group with the spatial navigator, including wrap-around, modal scope stack, and automatic focus-ring styling.

**What this execution owes:**

1. Call `useSpatialNav()` at the app root.
2. Wrap navigation sections in `<SpatialGroup>`.
3. Verify at scaffold time that `@vrooli/iframe-bridge` is present in `scenarios/command-center/ui/package.json`.
4. Add an **Xbox Edge smoke test** of spatial navigation to this item's verification section — the library has been verified by source inspection but has not been exercised on Xbox Edge specifically.

No polling loop, no keycode mapping, no `navigator.getGamepads()` plumbing is needed in this scenario. The controller story is the same across Xbox Edge, smart-TV remotes with gamepad profiles, and desktop keyboards (spatial nav falls back to arrow keys).

Underlying engine (for reference only): `packages/iframe-bridge/src/gamepadInput.ts`, `spatialNav.ts`, `spatialNavBridge.ts`.

See conclusion Finding 16.

## Staleness UX: subtle, per-widget, never a takeover (Finding 17)

When an upstream source is unavailable, widgets continue to render the **last-cached value** with a small timestamp/badge. **No global banner, no full-screen takeover.** Widgets that have never received data fall back to the existing gap-mode rendering (Finding 6). The staleness signal is driven by the cache envelope's `staleness_ts` field.

The `/api/v1/gaps` endpoint is **not** used for transient outages — it remains the source of truth for structural gaps (metrics that have never shipped), not runtime outages.

Per-theme staleness badge visuals are specified in `execute/command-center-theming-engine`.

## Fullscreen and Wake Lock hooks

These hooks are established at the scaffold level (see `execute/command-center-scenario-scaffold/research-notes.md`) and adapted for command-center's auto-fullscreen-on-load, wake-lock-always-on posture. This item consumes them rather than redefining them. Source hooks: `scenarios/web-console/ui/src/hooks/useWakeLock.ts`, `scenarios/landing-manager/ui/src/hooks/useFullscreenMode.ts`, `scenarios/tech-tree-designer/ui/src/hooks/useFullscreenManager.ts` (Finding 12).

## Auto-cycle transitions

Auto-cycle between dashboards uses **fade-through-black transitions** (Finding 11) — current page fades to black (~0.5s), outgoing R3F Canvas unmounts, incoming Canvas mounts, then fades in (~0.5s). This masks both the Canvas initialization flash and the ~100–300ms code-split dynamic-import window (Finding 15). Crossfade/slide/portal alternatives were rejected as either too complex (dual WebGL contexts) or too disruptive for a sit-back TV experience.
