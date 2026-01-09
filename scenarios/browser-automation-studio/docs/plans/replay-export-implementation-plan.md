Goal
Deliver a unified replay/export pipeline that supports UI customization, API-driven video exports (recorded video or replay frames), removes Node scripts from the CLI, and keeps exports fast and high quality.

Scope
- Browser Automation Studio only (`scenarios/browser-automation-studio`).
- Replay/export pipeline: UI customization, API render endpoints, Playwright driver capabilities, storage of video artifacts, and removal of CLI Node scripts.

Current State (facts)
- Replay video export is API-driven and uses Playwright capture + ffmpeg by loading the UI export page (`/export/composer.html`) and taking repeated screenshots.
  - API renderer: `scenarios/browser-automation-studio/api/services/replay/renderer.go`
  - Playwright capture loop: `scenarios/browser-automation-studio/api/services/replay/playwright_client.go`
  - Export page (CSS/JS styling): `scenarios/browser-automation-studio/ui/src/export/ReplayExportPage.tsx`
- HTML replay bundle export is API-driven (zip bundle) and CLI `execution render` now downloads/extracts it.
  - HTML bundle builder: `scenarios/browser-automation-studio/api/services/export/html_bundle.go`
  - Bash CLI download/extract: `scenarios/browser-automation-studio/cli/browser-automation-studio`
- Playwright driver supports native video recording (recordVideo) but it is only enabled when:
  - `required_capabilities.video` is true AND
  - `required_capabilities.video=true` in the session spec.
  - Driver config: `scenarios/browser-automation-studio/playwright-driver/src/config.ts`
  - Context creation: `scenarios/browser-automation-studio/playwright-driver/src/session/context-builder.ts`
- Video artifacts exist in the engine metadata path, but are not surfaced for export or storage yet.
  - Storage gap called out: `scenarios/browser-automation-studio/api/automation/README.md`
  - Video metadata persisted as artifacts: `scenarios/browser-automation-studio/api/automation/executor/simple_executor.go`
  - Execution writer handling: `scenarios/browser-automation-studio/api/automation/execution-writer/file_writer.go`
- Replay customization is stored in UI localStorage and injected into export requests, not persisted in API.
  - Customization state: `scenarios/browser-automation-studio/ui/src/domains/executions/viewer/useReplayCustomization.tsx`
  - Replay settings store: `scenarios/browser-automation-studio/ui/src/stores/settingsStore.ts`
  - Export request builds overrides/movie_spec: `scenarios/browser-automation-studio/ui/src/domains/executions/viewer/useExecutionExport.ts`
- API export endpoint accepts `format`, `movie_spec`, and `overrides`, but no HTML bundle generation.
  - Export request types: `scenarios/browser-automation-studio/api/handlers/export/types.go`
  - Handler logic: `scenarios/browser-automation-studio/api/handlers/executions.go`

Gaps vs desired behavior
- Native video recording exists but is not fully wired into export flows; there is no storage/URL surfacing for video artifacts.
- Replay export pipeline only supports screenshot-based render when recorded video is missing; no persisted recorded-video source surfaced in execution metadata.
- Export settings are stored in `exports` table but not treated as canonical render config.

Ideal Target Architecture
1) Replay config persisted in API and referenced by UI and exports.
   - Server stores a canonical `ReplayConfig` (theme, cursor, intro/outro, watermark, motion).
   - UI reads/writes config via API; localStorage becomes fallback only.
   - Export generation uses `ReplayConfig` as default and allows request overrides.

2) Two render sources supported under one API export endpoint.
   - Render source A: recorded video artifact (Playwright recordVideo).
   - Render source B: replay frames (existing export page + capture + ffmpeg).
   - API selects source based on availability or explicit request; falls back to replay frames.

3) HTML export moved into API.
   - New `format=html` path generates a static bundle and returns URL/blob.
   - Removes `cli/render-export.js` dependency and unblocks Go-only CLI.

4) High-quality, performant output.
   - Recorded video: best for smooth site animations.
   - Replay frames: best for stylized exports (cursor path control, timing, intro/outro).
   - Ensure dimension and capture interval controls remain in API config.

Plan of Action
Phase 0: Decisions (blocking)
- Confirm target API contract:
  - New `render_source` field (e.g., `recorded_video` | `replay_frames` | `auto`).
  - Whether HTML export returns zip, storage URL, or stream.
- Confirm where `ReplayConfig` is scoped (user vs project vs workflow).

Phase 1: Persisted Replay Configuration
- Add `ReplayConfig` model in API (DB + repo + handlers).
- Add API endpoints for CRUD.
- Update UI to read/write via API:
  - Replace localStorage-only customization with server config.
  - Keep localStorage as a last-resort cache.
- Update export request building to rely on server config unless user overrides.

Phase 2: Recorded Video Artifact Support
- Ensure executions can request native video capture:
  - Mark `requiresVideo` in plan metadata or settings.
  - Ensure the execution spec sets `required_capabilities.video=true` for runs that need recordings.
- Persist video artifacts:
  - Implement storage/upload for large video artifacts.
  - Persist URLs in execution artifacts (not just local paths).
- Surface recorded video in API execution metadata for export decisions.

Phase 3: Unified Export Endpoint (two modes)
- Extend `/api/v1/executions/{id}/export` to accept `render_source`.
- If `render_source=recorded_video` and video exists, return it directly.
- If `render_source=replay_frames`, run existing replay renderer path.
- If `render_source=auto`, choose recorded video first, fallback to replay frames.

Phase 4: API HTML Export (remove Node scripts)
- Implement `format=html` in export handler:
  - Reuse export page or server-rendered bundle.
  - Download assets and package into a static HTML bundle.
- Update CLI to call API `format=html`.
- Remove `scenarios/browser-automation-studio/cli/render-export.js`.

Phase 5: UI Export UX + Settings
- Add export mode selector in UI:
  - "Recorded video (fast, native)" vs "Replay render (stylized)".
- Make all replay options configurable in UI and persisted:
  - Watermark/intro/outro settings available for export render.
  - Ensure export page supports these settings from `movie_spec` or config.

Testing Strategy
- API unit tests:
  - Replay config CRUD.
  - Export endpoint: render source selection behavior.
  - HTML export bundle creation.
- Driver tests:
  - Verify `recordVideo` path on session close and returned `video_paths`.
- Integration tests:
  - Execution -> recorded video artifact -> export fetch.
  - Execution -> replay frames -> export video via renderer.
- UI tests:
  - Export dialog respects render source selection and config persistence.

Migration / Backward Compatibility
- Keep `movie_spec` + `overrides` support for existing clients.
- If no replay config exists, fall back to defaults (current behavior).
- If recorded video unavailable, auto mode falls back to replay render.
- Maintain existing export formats (`mp4`, `gif`, `json`, `html`).

Risks / Mitigations
- Video artifacts are large: require storage lifecycle policies and cleanup.
- Two sources may diverge: ensure export metadata labels source type.
- Performance: recorded video best for animations; replay render still required for stylized cursor paths.

Key Files to Touch (initial list)
- API export handler: `scenarios/browser-automation-studio/api/handlers/executions.go`
- Export service: `scenarios/browser-automation-studio/api/services/export/*`
- Replay renderer: `scenarios/browser-automation-studio/api/services/replay/*`
- Driver video capture: `scenarios/browser-automation-studio/playwright-driver/src/session/context-builder.ts`
- Driver config: `scenarios/browser-automation-studio/playwright-driver/src/config.ts`
- Execution writer: `scenarios/browser-automation-studio/api/automation/execution-writer/file_writer.go`
- UI export flow: `scenarios/browser-automation-studio/ui/src/domains/executions/viewer/useExecutionExport.ts`
- UI export page: `scenarios/browser-automation-studio/ui/src/export/ReplayExportPage.tsx`
- HTML bundle export: `scenarios/browser-automation-studio/api/services/export/html_bundle.go`
- CLI render updates: `scenarios/browser-automation-studio/cli/browser-automation-studio`

Open Questions
- Should HTML export be a zip bundle or a single HTML file with inlined assets?
- Where should replay config be stored (workflow vs project vs user)?
- Do we want stylized overlays on recorded video, or do we treat recorded video as "unstyled"?

Progress Notes
- 2025-12-20: Added API replay config endpoints (`/api/v1/replay-config`) backed by the settings table.
- 2025-12-20: UI replay customization now loads/saves the replay config via API with localStorage fallback.
- 2025-12-20: Export handler applies persisted replay config overrides before request overrides.
- 2025-12-20: Replay config now includes cursor motion + branding (watermark/intro/outro) and is applied in export specs.
- 2025-12-20: Settings view syncs replay settings to API; export composer uses watermark/intro/outro in playback.
- 2025-12-20: Decision: `render_source=auto` should prefer recorded video and fall back to replay frames.
- 2025-12-20: Decision: recorded video artifacts should be persisted in the execution result file (same place as console/network artifacts).
- 2025-12-20: Export handler now supports `render_source` with recorded-video passthrough/transcode (mp4/gif) and auto fallback to replay render.
- 2025-12-20: API now generates HTML replay bundles (`format=html`) and CLI `execution render` downloads/extracts them; Node scripts removed.
- 2025-12-20: Recorded video artifacts are stored via the screenshot storage backend and served under `/api/v1/artifacts/*`.
- 2025-12-20: Added `/api/v1/executions/{id}/recorded-videos` to surface recorded video artifacts.
- 2025-12-20: UI export dialog supports render source selection (auto/recorded/replay) persisted via replay config.
- 2025-12-20: Added HTML bundle zip unit test coverage; resolved a helper name conflict in the HTML bundle generator.
- 2025-12-20: Added handler tests for recorded videos endpoint and render-source webm guard; documented video capture requirements in ENVIRONMENT.md.
- 2025-12-20: Ran Go tests for handlers, execution-writer, and export packages; adjusted HTML export font stack to avoid Inter defaults.
- 2025-12-20: Fixed UI export typing issues (focused element bounding box, replay config branding mappings) to unblock UI build.

Progress Assessment (current)
- Phase 1 complete: persisted replay config (API + UI), export handler applies config, and composer renders branding settings.
- Phase 2 complete: recorded video artifacts stored via the storage backend and surfaced via API.
- Phase 3 complete: export handler supports `render_source` and recorded-video passthrough/transcode with fallback to replay render.
- Phase 4 complete: API HTML export + CLI render updated; Node scripts removed.
- Phase 5 complete: UI export mode selector shipped with recorded video availability hints, and execution start requests can opt into recorded video capture.
- Remaining gaps: none blocking; recorded video capture depends on `required_capabilities.video` in the session spec.

Immediate Next Steps (recommended order)
1) Optional: add service-level test coverage around execution video artifacts in `services/workflow` (result.json parsing).
2) Optional: add handler coverage for recorded-video passthrough (mp4/gif) once a lightweight repo stub is available.
