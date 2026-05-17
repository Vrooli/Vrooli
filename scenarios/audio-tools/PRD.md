# Product Requirements Document (PRD)

> **Template Version**: 2.0
> **Canonical Reference**: `/scenarios/prd-control-tower/docs/CANONICAL_PRD_TEMPLATE.md`
> **Validation**: Enforced by `prd-control-tower` + `scenario-auditor`
> **Policy**: Generated once and treated as read-only (checkboxes may auto-update)

## Overview
- **Purpose**: Canonical owner of reusable audio capabilities — speech-to-text, text-to-speech, summarization, audio processing — with three-tier provider routing (BYOK → Vrooli/LPBS → Local) and an adoptable React component surface. Replaces the per-scenario reimplementation of audio in web-console (and future consumers like swarm-manager, agent-manager, phone-agent, twilio-voice).
- **Primary users/verticals**: Other Vrooli scenarios that need audio capabilities; operators configuring per-instance routing and BYOK credentials; end-users of consuming applications who interact with voice features.
- **Deployment surfaces**: Connect-RPC API, CLI mirroring the API, React UI for configuration/diagnostics/usage, adoptable embed surface as the `@audio-tools/embed` workspace package, WebSocket browser-voice transport.
- **Value promise**: Audio capabilities become reusable infrastructure. Adding voice to a new scenario goes from porting ~30 files of provider code to wiring a Connect client + dropping in `<VoiceInputButton>`. Provider routing (BYOK/Vrooli/Local) becomes uniform across the platform.

## Operational Targets

### 🔴 P0 – Must ship for viability
- [x] OT-P0-001 | Local STT works end-to-end | Whisper resource transcribes a 30s wav in <3s p95.
- [x] OT-P0-002 | Local TTS works end-to-end | Kokoro resource synthesizes 200 words in <2s p95.
- [x] OT-P0-003 | Local summarization works end-to-end | Ollama summarizes 2kB text in <5s p95.
- [x] OT-P0-004 | Audio transcoding works | mp3/wav/flac/aac/ogg → wav round-trip.
- [~] OT-P0-005 | BYOK tier works for 5 starter adapters | openai-whisper, deepgram, openai-tts, elevenlabs, openrouter. **Status (2026-05-16):** five adapters land in `internal/byok/`; unary path works; only Deepgram has streaming. See PROBLEMS.md "Streaming providers declared but not implemented".
- [x] OT-P0-006 | Per-capability provider precedence enforced | BYOK → Vrooli → Local; ErrInsufficientCredits short-circuits.
- [x] OT-P0-007 | Voice-session pub/sub | ≥3 concurrent observer subscribers per session, no transport interference.
- [x] OT-P0-008 | Barge-in | VAD-detected speech cancels in-flight TTS within 100ms p95.
- [x] OT-P0-009 | Configuration Console | Operator sets defaults, enters BYOK creds, sees live availability matrix.
- [x] OT-P0-010 | Diagnostics Workbench | Manual STT/TTS/summarize/transcode with per-call provider trace + timing.
- [x] OT-P0-011 | Usage Dashboard | Recent operations, charged credits, provider distribution.
- [x] OT-P0-012 | Adoption snippet | `<VoiceInputButton>` + `<AudioPlayerBar>` embed snippets in `docs/reference/adoption.md` copy-paste cleanly into a consumer's react-vite tree.

### 🟠 P1 – Should have post-launch
- [ ] OT-P1-013 | Adoption Management UI | Lists connected scenarios + integration health.
- [~] OT-P1-014 | Streaming WS transport | Partial transcripts, segment finals, barge-in, speaker verification gating end-to-end. **Status (2026-05-16):** proto + chain streaming interface + bidi Connect handler + CLI commands shipped (Phases A/B/C/F-partial/H of audio-tools-web-console-restoration plan). Buffered fallback emits Segment + Done. Native partials (Phase D), BYOK streaming adapters (Phase E), and WS-handler chain rewire (Phase F second half) deferred — see `docs/internal/PROBLEMS.md`.

### 🟢 P2 – Future / expansion
- [ ] OT-P2-001 | Twilio media-stream transport (covered by `execute/audio-tools-twilio-media-stream-transport`).
- [ ] OT-P2-002 | LPBS audio gateway tier flipped on (depends on `execute/lpbs-audio-gateway-endpoints`).
- [ ] OT-P2-003 | swarm-manager + agent-manager + phone-agent adoption.

## Tech Direction Snapshot
- Preferred stacks / frameworks: Go (Connect-RPC API, BAS-style provider chains), React + Vite + Tailwind (UI), gorilla/websocket (browser-voice transport).
- Data + storage expectations: SQLite for settings + usage rows; on-disk content-addressable cache for TTS audio.
- Integration strategy: All proto-owned operations via Connect-RPC; REST exceptions only for multipart audio uploads (`Transcribe`, `Transcode`) and `GET /health`. Browser-voice WS is a `TransportReason: websocket_transport` endpoint, not a REST exception.
- Non-goals / guardrails: No generic unified provider gateway (per-capability typed chains only). No caller-configurable precedence (fixed BYOK→Vrooli→Local). No shared `packages/audio-text-utils` package (wrap-not-use; pure utils stay inside audio-tools). No streaming STT/TTS through the provider chain (WS streaming stays Local-only).

## Dependencies & Launch Plan
- Required resources: whisper, kokoro, ollama (all `required: false` — BYOK-only operation is supported).
- Scenario dependencies: landing-page-business-suite (`required: false`; LPBS tier disabled by `AUDIO_AI_ENABLE_VROOLI=false` until `execute/lpbs-audio-gateway-endpoints` lands).
- Operational risks: Cross-scenario hop latency (mitigated by loopback HTTP); LPBS slip (mitigated by Local+BYOK shipping standalone); browser-voice WS transport-reason template extension (R-PROTO).
- Launch sequencing: A. Scaffold + protos. B. Backend + Local chain. C. BYOK adapters. D. LPBS (flag-off). E. CLI. F. UI archetypes + embed. G. End-to-end audio-tools verification. H. web-console adoption. I. Cross-scenario verification + required-flag flip.

## UX & Branding
- Look & feel: vrooli-default design kit (Tailwind + design-tokens.css). Configuration Console is operator-tooling-dense; Diagnostics Workbench is try-it-now playful; Usage Dashboard is data-grid + lightweight charts.
- Accessibility: WCAG 2.1 AA for the embed surface (`<VoiceInputButton>` must be keyboard-activatable, screen-reader-labeled, and provide a visible mic-readiness state). VAD/barge-in state changes are announced via `aria-live`.
- Voice & messaging: Operator-focused; surfaces concrete provider/model identifiers in traces; never hides which tier handled a call.
- Branding hooks: Adopts vrooli-default tokens; embed components inherit consuming-app tokens via CSS variables.

## Appendix
- Research: `scenarios/swarm-manager/research/audio-provider-routing-contract/conclusion.md` (provider-routing FINAL contract); `scenarios/swarm-manager/research/audio-tools-shared-scenario-contract/spec.json` (voice-session architecture).
- Seed plans: `scenarios/swarm-manager/execute/audio-tools-greenfield-scenario/{spec.json,plan.md}`, `scenarios/swarm-manager/execute/web-console-adopt-audio-tools/spec.json`.
- Active plan: `~/.vrooli/plans/audio-tools-greenfield-scenario-web-console-adoption.md`.
- Source for extraction: `scenarios/web-console/api/internal/{tts,voice,audio,audioports}` + `scenarios/web-console/ui/src/hooks/{voice,tts}` + components.
