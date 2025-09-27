# Immersive Engine – Developer Notes

This document tracks the work-in-progress Experience engine that will replace the legacy React room.

## State Bridge
- `ui/src/state/store.js` uses Zustand + `subscribeWithSelector` so Three.js systems and React components read the same source of truth. Key domains:
  - **Viewport + World**: `timeOfDay`, `cameraFocus`, loader progress.
  - **UX**: generator/reader modals, prototype/developer toggles.
  - **Stories**: selected story drives emissive accents and camera choreography.
- When adding new modules prefer selectors over full-store subscriptions to avoid needless renders.

## Experience Skeleton
- `ui/src/three/Experience.js` mirrors Bruno’s architecture: `Sizes`, `Time`, `Camera`, `Renderer`, `World`, `CameraRig`, `HotspotRegistry`, `LoaderBridge`.
- Store subscription fans out to:
  - Scene colour lerp (day/evening/night baseline).
  - `World.setStoreSnapshot` for emissive/ground blends.
  - `CameraRig.updateFocus` for cinematic framing.

## Camera Rig
- `ui/src/three/CameraRig.js` exposes preset orbits. Use `setCameraFocus(id)` in the store to tween towards a preset.
- Presets are placeholder values; once the GLB blockout lands, update vectors to match actual anchors.

## Hotspots
- `ui/src/three/HotspotRegistry.js` pre-registers focus points in state. World modules can read `hotspots` and attach interaction meshes later on.

## Loader Bridge
- Real loading now flows through `ResourceLoader` (Three LoadingManager). `LoaderBridge` remains available as a fallback but is currently unused.

## Developer Console
- Toggling “Developer Mode” in the UI enables the HUD overlay and unlocks in-scene tooling. The overlay is temporary—target implementation is a diegetic holographic panel driven by the same store keys (`developerMode`, `developerConsoleOpen`).
- Camera buttons call `setCameraFocus`, providing a quick smoke test for nav rails before OrbitControls are removed.

## Next Actions
1. Replace fallback GLB placeholders with exported blockout (`/prototype/room_blockout.glb`) and assign baked textures per time-of-day.
2. Implement `HotspotController` that maps store entries to actual mesh hit areas and pointer/touch events.
3. Move developer console contents into the Experience scene (Tweakpane successor) while keeping DOM fallback.
4. Expand asset manifest to include per-module entries (e.g., screens, props) and connect to loader events for incremental updates.

Keep this file updated as modules mature so new agents can ramp quickly.

## Bedroom Suite Assets
- Added `public/experience/bedroom/bedroomScene.glb`, sourced from the Minimalistic Modern Bedroom by Dylan Heyes (CC BY 4.0, https://skfb.ly/oCnNx).
- Attribution is recorded in `public/experience/bedroom/ATTRIBUTION.txt`. When shipping or presenting the scenario, keep the credit visible in user-facing materials.
- The immersive experience now supports switching between the legacy studio and the new bedroom. Loader manifests and navigation presets are room-aware, so future rooms can extend `ROOM_ASSET_MANIFEST` and `ROOM_PRESETS` without reworking the core systems.
