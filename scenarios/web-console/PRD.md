# Web Console - Product Requirements Document

## Executive Summary
Web Console modernizes the former "Codex Console" scenario into a generalized remote terminal that lives inside authenticated Vrooli experiences. Operators receive shell access, per-session command overrides, and shortcut buttons for high-value CLIs like Codex and Claude. The backend remains Go-based to align with Vrooli's preferred service stack, while the UI stays lightweight for iframe embedding. Authentication is deliberately omitted; parents must supply identity and policy enforcement before exposing the console.

**Revenue Opportunity:** $35K–$85K annual value by delivering mobile-ready shell access, faster on-call remediation, and reusable shortcut workflows (Codex, Claude, platform status) across the ecosystem.

## 🎯 Capability Definition

### Core Capability
Lifecycle-managed PTY sessions streamed over WebSockets with configurable default commands (shell by default) and UI shortcuts that inject predefined commands. Other scenarios can embed the console via iframe to surface terminal control without bespoke engineering.

### Intelligence Amplification
One-click access to Codex, Claude, and platform diagnostics keeps human operators and agents in tight feedback loops. Sessions persist transcripts so Vrooli memory captures the context needed to iterate on future fixes.

### Recursive Value
Every remote session becomes a reusable capability. Maintenance orchestrators, ecosystem dashboards, and future meta-scenarios can summon shell access or launch helper CLIs instantly, compounding the platform's ability to respond to issues.

## Problem Statement
- On-call engineers rely on ad-hoc SSH or screen sharing, which does not integrate with Vrooli lifecycle controls or audit systems.
- Remote operators need the flexibility of a normal shell, not just the Codex CLI.
- Launching helper tools (Codex, Claude, platform status) requires manual typing, slowing response time on mobile devices.
- Vrooli still lacks a Go-native console that exposes shortcut-driven workflows while maintaining proxy guardrails and transcript retention.

## Target Users & Scenarios
- Operators inside authenticated dashboards (Ecosystem Manager, Maintenance Orchestrator, etc.) who need shell-level access with auditable transcripts.
- Support engineers escalating to Codex or Claude mid-incident.
- Future scenarios that want to offer “press-to-launch” automation against local CLIs without building bespoke terminals.

## 📊 Success Metrics

### Functional Requirements
- **Must Have (P0)**
  - [ ] Go API providing session lifecycle endpoints, WebSocket streaming, and transcript persistence.
  - [ ] Configurable default command (`WEB_CONSOLE_DEFAULT_COMMAND`) plus per-session overrides via API.
  - [ ] Browser UI (vanilla JS + xterm.js) with shortcut buttons for Codex, Claude, and `vrooli status` that queue commands until the PTY is ready.
  - [ ] iframe bridge that publishes status/metadata via `postMessage` and responds to transcript/log/screenshot requests.
  - [ ] Proxy guard enforcement that rejects requests lacking authenticated forwarding headers.
  - [ ] Documentation emphasizing proxy-only deployment and shortcut behavior.

- **Should Have (P1)**
  - [ ] Concurrent session handling with queueing when capacity is exhausted.
  - [ ] Prometheus metrics endpoint for session counters and failure modes.
  - [ ] Panic-stop endpoint/UI control to terminate runaway commands.
  - [ ] Mobile-first UI with reconnect/resume flows and quick command access.

- **Nice to Have (P2)**
  - [ ] Configurable shortcut palette provided by parent scenario or service config.
  - [ ] Redis/Postgres adapters for distributed transcript storage.
  - [ ] Voice dictation/TTS hooks that leverage shared resources.
  - [ ] Parent callbacks for automated post-session archiving or compliance workflows.

### Performance Criteria
| Metric | Target | Measurement |
|--------|--------|-------------|
| Session bootstrap time | < 4s | API timing logs |
| WebSocket latency | < 500ms p95 | Heartbeat telemetry |
| Concurrent sessions | Baseline 4 | Supervisor metrics |
| Shortcut dispatch delay | < 1s from PTY ready | UI event timestamps |
| Transcript durability | 100% persisted | Integration tests & crash sims |
| Memory usage | < 250MB per session | `psutil`/cgroup statistics |

### Security & Reliability Controls
- Must operate only behind authenticated parent scenarios; proxy guard enforces `X-Forwarded-*` headers.
- Commands run as non-privileged users with TTL, idle timeout, and panic-stop controls.
- Shortcut execution queues until the session is ready, preventing dropped commands during bootstrap.
- Automated cleanup of PTYs, temp files, and transcripts at session termination.

## 🔗 Dependencies

| Type | Dependency | Notes |
|------|------------|-------|
| Mandatory | Go 1.21+ | API + supervisor implementation |
| Mandatory | POSIX shell (default `/bin/bash`) | Default command | 
| Mandatory | Local filesystem | Transcript storage under scenario-managed path |
| Optional | Codex CLI | Triggered via shortcut (`codex --yolo`) |
| Optional | Claude CLI | Triggered via shortcut (`claude --dangerously-skip-permissions`) |
| Optional | Redis/Postgres | External transcript storage |
| Optional | Authenticated proxy/parent scenario | Required front door |

## 🏗️ Architecture Overview

### Session Flow
1. Parent scenario loads the console inside an authenticated iframe.
2. Operator starts a shell or taps a shortcut; the UI queues shortcut commands until the PTY is ready.
3. UI calls `POST /api/v1/sessions` (with optional `command`/`args`) via `proxyToApi`.
4. Go supervisor spawns the command inside a PTY, records transcripts, and enforces TTL/idle policies.
5. UI upgrades to WebSocket (`/api/v1/sessions/{id}/stream`) for bidirectional messaging.
6. Session ends via stop request, TTL expiry, idle timeout, or panic-stop.

### Components
- **Go API Service**: REST + WebSocket, metrics, structured logging.
- **Session Supervisor**: PTY management, command overrides, resource limits, transcript writer.
- **Shortcut Dispatcher**: UI queue that replays queued commands once the WebSocket is open.
- **Vanilla JS UI**: xterm.js terminal, quick command panel, iframe bridge support.
- **Iframe Bridge**: `postMessage` protocol for parent scenarios to consume status, transcripts, screenshots, and logs.

### Data Flow
- WebSocket transmits JSON envelopes for terminal output, status events, and heartbeats.
- Transcripts stored as NDJSON for replay/audit use.
- Metrics exposed via Prometheus endpoint and consumable by ecosystem observability tooling.

## UX Principles
- Mobile-first layout with sticky input bar and easily tappable shortcuts.
- Clear status badge showing terminal/connection state.
- Event feed summarizing lifecycle events, shortcut dispatches, and suppression counters.
- Prominent safety notice reminding operators the console must remain behind authenticated proxies.

## Implementation Phases
1. **Reorientation**: Default shell command, shortcut queue, updated UI copy, docs overhaul. ✅
2. **Hardening**: TTL enforcement validation, panic-stop telemetry, richer metrics, shortcut configuration hooks.
3. **Mobility**: Reconnect/resume states, offline transcript cache, responsive polish.
4. **Scale-Up**: Multi-session queueing, Redis/Postgres persistence, parent callbacks.
5. **Assistive Features**: Voice dictation, push notifications, programmable shortcut palettes.

## Risk Assessment
- **Security Misconfiguration (High)**: Deploying without proxy/auth. Mitigation: proxy guard rejection, docs, runtime warnings.
- **Runaway Commands (Medium)**: Long-lived shells or processes. Mitigation: TTL, idle timeout, panic-stop endpoint.
- **Shortcut Drift (Medium)**: Commands become stale or unsafe. Mitigation: configurable palette, versioned docs, parent overrides.
- **Transcript Sensitivity (Low)**: Sensitive data captured. Mitigation: retention policies, redaction hooks.

## Definition of Done
- All P0 requirements implemented with automated coverage where feasible.
- Shortcut buttons dispatch commands reliably whether or not a session is already running.
- iframe bridge docs current; parent integration example tested.
- Proxy guard enabled by default; warnings documented for bypass.
- Metrics endpoint validated; panic-stop and timeout counters increase under test.

## Open Questions
- Should parent scenarios be able to provide custom shortcuts at runtime via iframe handshake?
- Do we need policy gates (RBAC) on which commands may execute per operator?
- What transcript retention defaults best align with platform compliance requirements?
- Should we surface richer command metadata (exit codes, runtime) in the event feed for analytics?
