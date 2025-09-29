# Codex Console - Product Requirements Document

## Executive Summary
Codex Console enables secure remote telepresence for the Codex CLI through a browser-based console that can be embedded inside authenticated Vrooli scenarios. All backend services are written in Go so the scenario stays consistent with Vrooli’s preferred service stack and can be leveraged directly by other Go-first scenarios. The service intentionally omits authentication and must only be surfaced through a parent scenario operating behind an authenticated proxy; exposing Codex Console directly to the public internet is expressly disallowed.

**Revenue Opportunity:** $30K–$80K annual value by unlocking mobile-ready operator workflows, faster incident response, and reusable remote Codex sessions across the Vrooli ecosystem.

## 🎯 Capability Definition

### Core Capability
Managed, session-scoped Codex CLI streaming over WebSockets delivered by a Go backend and lightweight vanilla JavaScript UI that other scenarios can embed via iframe.

### Intelligence Amplification
Ensures Codex remains available to operators away from their workstation, preserving transcripts and context so the wider Vrooli platform compounds knowledge across devices and agents.

### Recursive Value
Enables maintenance orchestrators, ecosystem management tools, and future meta-scenarios to escalate work to a human+Codex pair without custom engineering; the iframe-ready UI component becomes reusable infrastructure for remote operations.

## Problem Statement
- Operators currently rely on ad-hoc SSH or screen sharing to supervise Codex; none integrate with Vrooli’s lifecycle or memory systems.
- Mobile devices cannot easily attach to Codex CLI streams, limiting responsiveness during incidents or travel.
- Exposing a terminal without hard guardrails is dangerous; we need a controlled, auditable bridge that still fits the make-driven scenario lifecycle.
- Vrooli lacks a Go-native, iframe-friendly console scenario that can be safely proxied through existing authenticated portals.

## Target Users & Scenarios
- Operators using authenticated dashboards (e.g., Ecosystem Manager, Maintenance Orchestrator) who need to consult Codex from a phone.
- Support engineers triaging production incidents via secure proxy.
- Future scenarios that require real-time Codex collaboration while retaining an audit trail.

## 📊 Success Metrics

### Functional Requirements
- **Must Have (P0)**
  - [ ] Go-based API service providing session lifecycle endpoints and WebSocket streaming.
  - [ ] Session supervisor that launches Codex CLI inside a PTY with enforced TTL and idle timeout.
  - [ ] Browser UI (vanilla JS + xterm.js) that talks to the API via a `proxyToApi` helper to obey scenario networking conventions.
  - [ ] iframe bridge that exposes session status, screenshots, and transcript metadata to the parent scenario through `postMessage`.
  - [ ] Transcript persistence to rotating files with configurable retention and redaction hooks.
  - [ ] Explicit documentation warning against direct exposure and outlining required parent-scenario safeguards.

- **Should Have (P1)**
  - [ ] Multiple concurrent sessions with queueing when supervisor slots are exhausted.
  - [ ] Structured metrics endpoint (`/metrics`) for Prometheus-compatible scraping.
  - [ ] Panic-stop endpoint and UI control to terminate runaway Codex sessions.
  - [ ] Mobile-first UI with reconnect/resume flow and offline transcript viewing.

- **Nice to Have (P2)**
  - [ ] Optional Redis/Postgres adapters for distributed transcript storage.
  - [ ] Voice dictation and text-to-speech bridge leveraging existing Vrooli resources.
  - [ ] Parent scenario callback hooks for automated log capture or compliance workflows.

### Performance Criteria
| Metric | Target | Measurement |
|--------|--------|-------------|
| Session bootstrap time | < 4s | API timing logs |
| WebSocket latency | < 500ms p95 | Heartbeat telemetry |
| Concurrent sessions | 4 baseline | Supervisor metrics |
| Transcript durability | 100% persisted | Integration tests & crash simulations |
| Memory usage | < 250MB per session | `psutil`/cgroup statistics |

### Security & Reliability Controls
- No built-in auth; only operable when embedded by trusted parent scenario behind secure proxy.
- Codex worker runs as non-privileged Unix user with optional chroot and cgroup resource limits.
- Command rate limiting and max session quota enforced server-side.
- iframe sandbox + strict CSP documented for parent scenario implementation.
- Automated cleanup of stale PTYs, temp files, and transcripts when TTL expires.

## 🔗 Dependencies

| Type | Dependency | Notes |
|------|------------|-------|
| Mandatory | Go 1.21+ | API + supervisor implementation |
| Mandatory | Codex CLI binary | Launched by supervisor |
| Mandatory | Local filesystem | Transcript storage under scenario-managed path |
| Optional | Redis | Shared state for multi-instance deployments |
| Optional | Postgres | Long-term transcript archive |
| Optional | Existing proxy/auth scenario | Required entry point for end users |

## 🏗️ Architecture Overview

### Session Flow
1. Parent scenario obtains access via existing authenticated proxy and loads Codex Console in an iframe.
2. iframe establishes handshake using the iframe bridge; parent can request screenshot/log capture while Codex Console exposes session metadata via `postMessage`.
3. UI calls Go API through `proxyToApi` helper, creating a session (`POST /api/v1/sessions`).
4. Go supervisor spawns Codex CLI inside PTY, records transcripts, and enforces TTL/idle policies.
5. UI upgrades to WebSocket (`/api/v1/sessions/{id}/stream`) for bidirectional messaging.
6. Session ends via stop request, TTL expiry, or idle timeout; supervisor tears down PTY, writes final transcript, and informs parent via iframe bridge.

### Components
- **Go API Service**: REST + WebSocket, metrics, structured logging.
- **Go Session Supervisor**: PTY management, resource limits, transcript writer.
- **Transcript Store**: Rotating file sink with hooks for redaction and optional external persistence.
- **Vanilla JS UI**: Chat-first browser interface with terminal overlay powered by xterm.js. Uses `proxyToApi` for all network calls and registers iframe bridge handlers.
- **Iframe Bridge**: `postMessage` protocol for parent scenario to request screenshots, transcripts, or receive session status updates.

### Data Flow
- WebSocket transmits JSON envelopes containing terminal frames, status updates, and metrics.
- Transcripts stored as newline-delimited JSON plus Markdown render for quick viewing.
- Metrics exposed via Prometheus endpoint and mirrored through iframe bridge when requested.

## UX Principles
- Mobile-first design with sticky composer and reach-friendly controls.
- Split chat/terminal view with xterm.js rendering the raw terminal feed.
- Clear iconography and typography using lightweight SVGs (no heavy component libraries).
- Visual warnings reminding operators that access must be routed through authenticated parent scenarios.

## Implementation Phases
1. **Foundation**: Go API skeleton, single-session PTY supervisor, vanilla JS UI with `proxyToApi`, iframe bridge handshake, transcript writer.
2. **Hardening**: TTL enforcement, panic-stop, metrics endpoint, structured logging, doc updates emphasizing proxy-only deployment.
3. **Mobility**: Reconnect/resume, offline transcript cache, responsive layout polish.
4. **Scale-Up**: Multi-session queueing, Redis/Postgres integration, advanced parent callbacks.
5. **Assistive Extras**: Voice bridge, push notifications, enhanced analytics.

## Risk Assessment
- **Security Misconfiguration (High)**: Deploying without authenticated parent scenario. Mitigation: prominent documentation, runtime warning logs, and health endpoint flag if proxy headers missing.
- **Resource Exhaustion (Medium)**: Runaway Codex process. Mitigation: cgroup limits, panic-stop, TTL, watchdog.
- **Network Instability (Medium)**: Mobile WebSocket drops. Mitigation: resume tokens, replay buffer.
- **Transcript Sensitivity (Low)**: Sensitive data in logs. Mitigation: redaction middleware, retention policies set by parent scenario.

## Definition of Done
- All P0 requirements satisfied and covered by automated tests.
- iframe bridge documented with sample parent integration and verified via automated tests.
- `proxyToApi` usage audited across UI; no direct fetch calls.
- Deployment docs explicitly state “Do not expose Codex Console directly; must be proxied through authenticated scenario.”
- Observability in place (metrics, structured logs, panic-stop alerts).

## Open Questions
- Should supervisor support pluggable codex profiles (different configs per parent scenario)?
- Which transcript retention defaults align with compliance needs?
- Do we need built-in screenshot capture or should parent scenario handle DOM snapshots via iframe bridge callbacks?
- Should we ship example parent integration (e.g., Maintenance Orchestrator plugin) to accelerate adoption?
