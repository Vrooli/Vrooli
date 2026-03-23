# Video Studio — Refined Summary

## Overview

Video Studio is a new Vrooli scenario that unifies browser recording (via BAS dependency), desktop recording (custom FFmpeg+Xvfb implementation), and AI-driven scripting into a single pipeline for automated video production. It enables marketing agents and other scenarios to produce professional video content from text briefs without human intervention during production.

## Key Architecture Decisions

### Recording Sources
1. **Browser recording**: Runtime dependency on BAS. BAS already has a mature pipeline — Playwright viewport capture → ReplayMovieSpec → FFmpeg encode (libx264, CRF 21, yuv420p). Video Studio calls BAS's HTTP API to request recordings, not import its packages.
2. **Desktop recording**: Custom implementation using `resource-ffmpeg screen-capture` CLI for capture (same underlying tool as S2D, but with Video Studio's own scene/timing control layer) and direct FFmpeg for compositing. S2D's Xvfb management pattern (`:99`, 1920x1080x24, 5s readiness poll) is proven and can be replicated.
3. **Compositing**: Direct FFmpeg invocation for stitching segments, transitions, overlays, and format conversion. Independent of recording source.

### Intermediate Representation
BAS's ReplayMovieSpec is a strong candidate for a universal video description format. It already handles frames, cursor motion, theme, watermarks, intro/outro cards, and playback settings. Extending it (or creating a compatible superset) for desktop recordings and composited videos would unify the render pipeline.

### API Surface (P0)
- `POST /api/v1/videos` — Create video from brief (async job)
- `POST /api/v1/videos/record/browser` — Record browser session via BAS
- `POST /api/v1/videos/record/desktop` — Record desktop session
- `POST /api/v1/videos/composite` — Stitch segments together
- `GET /api/v1/videos/{id}` — Get video status/result
- `GET /api/v1/videos/{id}/artifacts` — Stream video file

### Integration Points
- **Marketing crew**: Content-creator logs video opportunities in `decisions.jsonl`. P0: manual invocation from logged opportunities. P1: direct API endpoint for programmatic requests.
- **Agent-Manager**: Spawns AI scripting agents that interpret text briefs and produce recording scripts.
- **resource-ffmpeg**: Desktop capture start/stop + post-processing encode operations.

## Clarifications Applied
- BAS integration is runtime (API calls), not build-time (package imports) — follows the Vrooli scenario dependency pattern.
- Desktop recording reuses `resource-ffmpeg` for capture but uses FFmpeg directly for compositing (resource-ffmpeg doesn't cover concat/transitions).
- P0 targets fully autonomous operation (brief in → video out) with human review of final output only.

## Existing Infrastructure Assessment

### BAS (Strong reuse potential)
- Full Playwright recordVideo + export pipeline with ReplayMovieSpec format
- FFmpegEncoder with proven settings (libx264, CRF 21, high profile, level 4.1)
- Artifact storage (MinIO/S3 + file-based), content-type detection, cache headers
- Viewport stabilization script (scrollbar flicker fix)
- Archive ingestion (ZIP import)

### S2D (Pattern reuse, not dependency)
- Xvfb display management: `:99`, 1920x1080x24, GLX+render, 5s readiness poll
- `resource-ffmpeg screen-capture start/stop` CLI contract
- Recording config: `Enabled`, `DisplayWidth`, `DisplayHeight`, `FPS`, `MaxDurationSec`
- Result tracking: `Recorded`, `VideoPath`, `DurationMs`, `FileSizeBytes`, `Error`

### Marketing Crew (Ready for integration)
- Content-creator already logs video opportunities as decisions
- Video content listed as "planned" capability in TEAM.md
- 5-step content workflow (identify → research → create → edit → approve) ready to incorporate video

## Phased Implementation

### P0 — Core Recording (Target: MVP)
1. Scaffold scenario (API + CLI + UI)
2. Implement BAS API client for browser recording requests
3. Implement desktop recording with Xvfb + resource-ffmpeg (scene-level start/stop)
4. Implement AI scripting agent (text brief → recording script)
5. Implement FFmpeg compositing (concat segments, basic transitions)
6. Artifact storage layer with manifest tracking
7. Async job system with status polling

### P1 — Polish & Integration
1. Marketing crew direct API integration
2. Platform-specific output formats (landscape/square/vertical)
3. Text overlays and TTS narration
4. Template system for common video patterns
5. Webhook/callback for async completion notifications

### P2 — Motion Graphics
1. Evaluate Remotion vs Rendervid
2. Programmatic intro/outro/animation generation
3. Composite pipeline (motion graphics + live recording)
4. Brand consistency system

## Success Criteria
- Agent produces finished video from text brief without human process intervention
- Browser demos recorded cleanly at target resolution (no artifacts, no flickering)
- Desktop recordings support scripted multi-step workflows with scene control
- Output covers major social media platforms (landscape, square, vertical)
- Marketing crew can request videos through existing workflow
- Video production time measured in minutes, not hours

## Readiness Gate
- [ ] Q2 answer received (video brief format: free-form, structured, or hybrid)
- [ ] Architecture decision: adopt ReplayMovieSpec as universal IR or create new format
- [ ] BAS API stability confirmed (which endpoints are stable for external consumption)

## Staging Artifacts Produced
- `enhance/prd-context.md` — Context brief for PRD generation
