# Video Studio — Implementation Plan

## Purpose

Create a new Vrooli scenario ("video-studio") that unifies browser recording (via BAS dependency), desktop recording (custom FFmpeg+Xvfb), and AI-driven scripting into a single pipeline for automated video production. Marketing agents and other scenarios can produce professional video content from text briefs without human intervention during production.

**This is greenfield work.** Do not include compatibility shims, legacy wrappers, dead code, unused re-exports, `// removed` comments, or renamed `_unused` variables.

## Required Reading

```bash
prompt-manager skill read api-steer cli-steer seam-discovery-and-enforcement interoperability-steer
```

Additional context:
- BAS source: `scenarios/browser-automation-studio/` (ReplayMovieSpec, FFmpegEncoder, artifact pipeline)
- S2D recording patterns: `scenarios/scenario-to-desktop/` (Xvfb + resource-ffmpeg screen-capture)
- resource-ffmpeg CLI: `resources/ffmpeg/` (screen-capture start/stop contract)
- Agent-Manager: `scenarios/agent-manager/` (spawning AI agents)

## Problem Statement

Vrooli's marketing crew produces only text-based content. Video content is far more engaging for demos, tutorials, and promotions, but there is no unified tool. Two existing recording capabilities (BAS browser recording, S2D desktop recording) are tightly coupled to their specific scenarios and not reusable for general video production.

## Scope

### In Scope
- New `scenarios/video-studio/` scenario with API + CLI + UI
- BAS runtime dependency for browser recording (HTTP API calls)
- Custom desktop recording via resource-ffmpeg + Xvfb with scene-level control
- AI scripting agent (text brief → recording script)
- FFmpeg compositing (concat segments, transitions, format conversion)
- Artifact manifest and storage layer (P0, per round 2 decision)
- Async job system with status polling + webhook/callback notifications (per round 2 decision)
- ReplayMovieSpec-compatible intermediate representation (adopt standard, don't extract shared code yet — per round 2 decision)

### Out of Scope
- Importing BAS Go packages directly (runtime API dependency only — per round 1 decision)
- S2D as a dependency (pattern reuse only, custom implementation)
- P1 features (marketing crew direct API, platform-specific formats, narration/TTS, templates)
- P2 features (motion graphics, Remotion/Rendervid evaluation, brand consistency)
- Shared library extraction between BAS and Video Studio

### Acceptance Boundaries
- `acceptance_allow`: `scenarios/video-studio/**`
- `acceptance_deny`: (none)

## Current Technical Context

### BAS Recording Pipeline
- Playwright `recordVideo` → raw WebM frames
- `ReplayMovieSpec` (version 2025-11-07): JSON format describing frames, cursor motion, theme, watermarks, intro/outro cards, playback settings
- `FFmpegEncoder`: libx264, CRF 21, high profile, level 4.1, yuv420p
- Artifact storage: MinIO/S3 + file-based, content-type detection, cache headers
- Viewport stabilization script (scrollbar flicker fix)

### S2D Desktop Recording (Pattern Reference)
- Xvfb display management: `:99`, 1920x1080x24, GLX+render extensions, 5-second readiness poll
- `resource-ffmpeg screen-capture start/stop` CLI contract
- Config: `DisplayWidth`, `DisplayHeight`, `FPS`, `MaxDurationSec`
- Result tracking: `Recorded`, `VideoPath`, `DurationMs`, `FileSizeBytes`, `Error`

### Marketing Crew
- Content-creator already logs video opportunities as decisions in `decisions.jsonl`
- P0: manual invocation from logged opportunities
- P1: direct Video Studio API endpoint for programmatic requests

## Target End State (P0)

An agent can submit a text brief describing a desired video. Video Studio autonomously:
1. Interprets the brief and generates a recording script (scenes, actions, timings)
2. Executes browser recordings via BAS API and/or desktop recordings via Xvfb+FFmpeg
3. Composites segments into a final video with transitions
4. Stores artifacts with manifest tracking
5. Notifies the caller via webhook/callback when complete
6. Human reviews the final output only

## Implementation Strategy

### Phase 1: Scenario Scaffold
- Create `scenarios/video-studio/` with standard structure (API, CLI, UI, `.vrooli/service.json`)
- Define scenario dependencies: BAS (runtime), resource-ffmpeg
- Set up Go API skeleton with health check, router, and config
- Set up async job infrastructure (job queue, status tracking, webhook dispatch)

### Phase 2: Recording Backends
- **Browser recording client**: HTTP client for BAS API — request recording, poll for completion, retrieve artifact
- **Desktop recording engine**: Xvfb lifecycle management + `resource-ffmpeg screen-capture` start/stop, scene-level control (pause/resume between scenes), configurable resolution/FPS

### Phase 3: Intermediate Representation & Compositing
- Adopt ReplayMovieSpec-compatible format for describing video segments
- Desktop recordings emit compatible spec (frames + metadata)
- FFmpeg compositing pipeline: concat segments, crossfade transitions, audio mixing, format conversion
- Output: MP4 (H.264/AAC) as default, configurable

### Phase 4: AI Scripting Agent
- Agent receives text brief, produces structured recording script
- Script defines: scenes (ordered), per-scene recording source (browser/desktop), actions, timing, transitions
- Uses Agent-Manager for spawning the scripting agent
- Brief format: TBD (pending round 1 q2 decision — currently unresolved)

### Phase 5: Artifact Storage & API
- Artifact manifest per video job: raw recordings, intermediate composites, final render
- Storage backend (file-based initially, S3-compatible later)
- REST API endpoints:
  - `POST /api/v1/videos` — Create video from brief (async)
  - `POST /api/v1/videos/record/browser` — Direct browser recording
  - `POST /api/v1/videos/record/desktop` — Direct desktop recording
  - `POST /api/v1/videos/composite` — Stitch segments
  - `GET /api/v1/videos/{id}` — Job status/result
  - `GET /api/v1/videos/{id}/artifacts` — Stream video file
- Webhook/callback on job completion

### Final: Cleanup & Verification
- Run `go build ./...` and fix ALL errors, even pre-existing
- Run `golangci-lint run` and fix ALL warnings in modified files
- Run `go test ./... -timeout 300s` and fix any failures
- `vrooli scenario restart video-studio`
- Verify health: `curl -s http://localhost:<port>/health`

## Contract Decisions

### API Contracts
- All video creation is async — `POST` returns job ID, caller polls or receives webhook
- Webhook payload: `{ "job_id": "...", "status": "completed|failed", "artifact_url": "...", "error": "..." }`
- Recording requests include: target URL (browser) or app command (desktop), resolution, max duration
- Compositing requests include: ordered segment list, transition type, output format

### Data Model
- **VideoJob**: id, status (pending/recording/compositing/completed/failed), brief, script, segments[], artifacts[], created_at, updated_at, callback_url
- **VideoSegment**: id, source (browser/desktop), spec (ReplayMovieSpec-compatible), raw_path, encoded_path, duration_ms
- **VideoArtifact**: id, job_id, type (raw/composite/final), path, size_bytes, content_type, created_at

### CLI Contract
- `video-studio create --brief "..." [--callback URL]` — Submit video job
- `video-studio record browser --url URL [--resolution WxH]` — Direct browser recording
- `video-studio record desktop --command CMD [--resolution WxH]` — Direct desktop recording
- `video-studio status --id JOB_ID` — Check job status
- `video-studio artifacts --id JOB_ID` — List/download artifacts

## Testing Plan

### Unit Tests
- AI script generation: mock LLM responses, verify recording script structure
- FFmpeg compositing: verify command construction for concat, transitions, format conversion
- Artifact manifest: CRUD operations, storage path resolution
- Job state machine: valid transitions (pending→recording→compositing→completed, pending→failed, etc.)

### Integration Tests
- BAS API client: record a test page, verify video file returned (requires BAS running)
- Desktop recording: start Xvfb, record a simple X11 app, verify video file
- End-to-end: submit brief → scripting → recording → compositing → final artifact

### Acceptance Criteria
- [ ] `POST /api/v1/videos` with a text brief returns a job ID
- [ ] Job progresses through states and produces a final MP4 artifact
- [ ] Browser recording via BAS returns a valid video file with no visual artifacts
- [ ] Desktop recording captures Xvfb display content correctly
- [ ] Webhook fires on job completion with correct payload
- [ ] `video-studio status --id X` returns current job state
- [ ] Video production completes within minutes for a simple demo scenario

## Risks & Mitigations

| Risk | Impact | Likelihood | Mitigation |
|------|--------|------------|------------|
| BAS API instability — endpoints may change or not be designed for external consumption | High | Medium | Document which BAS endpoints Video Studio depends on; add integration tests that catch breakage early; consider a BAS API stability contract |
| Xvfb display conflicts — multiple concurrent recordings on same display | Medium | Medium | Use dynamic display allocation (`:99`, `:100`, etc.) with lock files, not hardcoded `:99` |
| FFmpeg compositing complexity — crossfade, audio sync, variable frame rates | Medium | Medium | Start with simple concat (no transitions) for P0; add transitions incrementally |
| AI scripting quality — LLM may generate invalid or impractical recording scripts | High | Medium | Validate scripts against a schema before execution; include retry with feedback loop |
| Resource-ffmpeg CLI limitations — screen-capture may not support all needed features | Low | Low | Falls back to direct FFmpeg invocation for unsupported operations |
| Long video rendering times blocking job queue | Medium | Low | Run compositing in background goroutines; set per-job timeouts |

## Non-Goals / Prohibited Patterns

- Do NOT import BAS Go packages — use HTTP API only
- Do NOT depend on Scenario-to-Desktop scenario — reimplement desktop recording
- Do NOT build a video editor UI — this is agent-driven with human review of output
- Do NOT implement P1/P2 features in P0 (templates, TTS, motion graphics)
- Do NOT extract shared libraries between BAS and Video Studio (per round 2 decision)
- Do NOT use hardcoded Xvfb display numbers — use dynamic allocation

## Definition of Done

1. `scenarios/video-studio/` exists with API, CLI, and service.json
2. An agent can submit a text brief and receive a finished MP4 video
3. Browser recording works via BAS API with clean output
4. Desktop recording works via Xvfb + resource-ffmpeg with scene control
5. FFmpeg compositing stitches segments into final video
6. Artifact manifest tracks all intermediate and final files
7. Webhook/callback fires on job completion
8. All tests pass, lint clean, scenario starts and reports healthy
9. No compatibility shims, no dead code, no imported BAS packages

## Open Items

- [ ] Video brief format decision (q2 from round 1 — free-form, structured, or hybrid) — blocks Phase 4 details
- [ ] BAS API endpoint stability audit — which endpoints are safe for external consumption?
- [ ] Specific LLM/model for AI scripting agent — depends on Agent-Manager capabilities
