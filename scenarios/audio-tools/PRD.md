# Product Requirements Document (PRD)

> **Template Version**: 2.0
> **Canonical Reference**: `scenarios/business-health/docs/reference/canonical-prd-template.md`
> **Validation**: Enforced by `business-health` (`validate scenario audio-tools`)
> **Policy**: Generated once and treated as read-only (checkboxes may auto-update)

## 🎯 Overview

Purpose: Audio Tools provides dependable local speech-to-text and dictation for people and scenarios that need privacy-aware audio capture, provider choice, recovery, and evidence-backed quality. Users include end users recording dictation, operators selecting local engines, and consuming scenarios such as Web Console. Deployment surfaces include the Audio Tools UI, Connect and WebSocket APIs, CLI, shared browser capture adapters, and local Whisper and Kyutai resources.

## 🎯 Operational Targets

> Checkboxes auto-update from requirements sync; do not hand-edit them.

### 🔴 P0 – Must ship for viability

- [ ] OT-P0-001 | Long-form dictation trust | Every captured audio interval is processed, retained for replay, or ends in an explicit recoverable failure across interruption, reconnect, slow-consumer, restart, and provider-contention paths.
- [ ] OT-P0-002 | Provider-parity stable engines | Whisper and Kyutai can be selected only after each passes the same no-loss, durability, recovery, policy, quality, and product-path trust floor.
- [ ] OT-P0-003 | Explicit speaker-policy safety | Requested extraction and verification always report an explicit applied or degraded result, and required filter policy fails closed by default.

### 🟠 P1 – Should have post-launch

- [x] OT-P1-001 | Provider-neutral evidence | Dictation Studio compares provider, strategy, policy, fault, and replay cells using persisted provider-neutral metrics and promotion verdicts.
- [x] OT-P1-002 | Mobile recovery and diagnostics | Mobile users can inspect coverage, recover retained audio, and export metadata-only diagnostics without developer tools.

### 🟢 P2 – Future / expansion

- [ ] OT-P2-001 | Expanded device qualification | Maintain recorded manual qualification evidence across iOS Safari, installed PWA, Android Chrome, and desktop microphone environments.

## 🧱 Tech Direction Snapshot

Preferred: Go owns STT session business logic, protobuf defines transport contracts, and TypeScript browser adapters own capture presentation only. Canonical PCM and recognition spans are provider-independent; browser and server retention are bounded and private. Integrations include Whisper, Kyutai, speaker-verification, Web Console, Swarm Manager, and Browser Automation Studio. Non-goals: new speech models, unrelated terminal behavior, and hosted multi-tenant GPU scaling.

## 🤝 Dependencies & Launch Plan

Resources: Whisper, Kyutai, ffmpeg, and optional speaker verification. Scenario consumers: Web Console, Swarm Manager, Browser Automation Studio, and Audio Tools. Launch sequencing: establish requirements and trust rubric, ship protocol and ledger, add admission and capture recovery, qualify provider-neutral experiments and product paths, then promote only with complete evidence. Risks: browser persistence constraints, single-model contention, device variation, and sensitive retained audio.

## 🎨 UX & Branding

Accessibility: Dictation feedback must use clear domain states, keyboard-accessible controls, readable recovery guidance, and status that does not rely only on color or developer-console logs. The voice should be calm, direct, privacy-conscious, and explicit about active engine, policy, coverage, queueing, and recovery.
