# PRD Context Brief — Video Studio

## Product Name
Video Studio – Unified AI-Driven Video Creation Scenario

## Problem
Vrooli's marketing and content capabilities are text-only. Video content drives higher engagement for product demos, tutorials, and promotional material, but there's no unified tool. Two recording capabilities exist (BAS for browser, S2D for desktop) but are tightly coupled to their own scenarios and not reusable.

## Target Users
1. **Marketing agents** (prompt-manager marketing-crew) — request and review marketing videos
2. **Other Vrooli scenarios** — programmatically request video artifacts (demos, tutorials)
3. **Human operators** — review final video output, configure templates, manage video jobs

## Existing Capabilities to Leverage
- **BAS**: Playwright recordVideo → ReplayMovieSpec → FFmpeg (libx264, CRF 21). Full export pipeline with artifact storage. Mature, production-tested.
- **S2D**: FFmpeg + Xvfb desktop recording via `resource-ffmpeg screen-capture` CLI. 1920x1080/15fps. Recording config + result structs.
- **resource-ffmpeg**: Vrooli resource for video capture and encoding.
- **Agent-Manager**: Spawns AI agents for scripting and orchestration.
- **Marketing Crew**: Already logging video opportunities; video content listed as planned capability.

## Core Requirements (P0)
1. Browser recording via BAS API (runtime dependency)
2. Desktop recording via custom Xvfb + resource-ffmpeg (not S2D dependency)
3. AI-driven scripting: text brief → structured recording script → executed recording
4. FFmpeg compositing: concat segments, transitions, timing adjustments
5. Async job management with artifact storage
6. Standard Vrooli scenario structure (API + CLI + UI)

## Integration Requirements (P1)
1. Marketing crew direct API for programmatic video requests
2. Platform-specific output (landscape/square/vertical)
3. Narration/annotation overlays (text + TTS)
4. Reusable video templates

## Advanced Capabilities (P2)
1. Programmatic motion graphics (Remotion or Rendervid — evaluate at implementation time)
2. Full composite pipeline (motion graphics + live recording)
3. Brand consistency system

## Key Constraints
- BAS is a runtime dependency (API calls), not a build-time dependency
- Desktop recording must be independent of S2D (S2D's recording is smoke-test-coupled)
- Video production must be fully autonomous (human reviews output only)
- Must integrate with Vrooli's scenario lifecycle system (Makefile, service.json)

## Success Metrics
- End-to-end: text brief → finished video without human process intervention
- Video quality: clean recordings at target resolution, no artifacts
- Production time: minutes, not hours
- Platform coverage: landscape, square, vertical output formats
- Integration: marketing crew can request videos through existing workflow
