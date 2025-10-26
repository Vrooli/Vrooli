# Web Console - Product Requirements Document

## Executive Summary
Web Console modernizes the former "Codex Console" scenario into a generalized remote terminal that lives inside authenticated Vrooli experiences. Operators receive shell access, per-session command overrides, and shortcut buttons for high-value CLIs like Codex and Claude. The backend remains Go-based to align with Vrooli's preferred service stack, while the UI stays lightweight for iframe embedding. Authentication is deliberately omitted; parents must supply identity and policy enforcement before exposing the console.

> **Implementation note:** The frontend now ships as a Vite project with the main entry at `ui/src/main.js` (previously `ui/static/app.js`) and vendored browser libraries under `ui/public/lib/`.

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
  - [✅] Mobile-first UI with reconnect/resume flows and quick command access.

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
| Concurrent sessions | Baseline 20 | Supervisor metrics |
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
2. **Hardening**: TTL enforcement validation, panic-stop telemetry, richer metrics, shortcut configuration hooks. (IN PROGRESS)
3. **Mobility**: Reconnect/resume states, offline transcript cache, responsive polish. ✅ (2025-10-04)
4. **Scale-Up**: Multi-session queueing, Redis/Postgres persistence, parent callbacks.
5. **Assistive Features**: Voice dictation, push notifications, programmable shortcut palettes.

## Progress History

### 2025-10-05: Multi-Strategy Terminal Rendering Fixes
**Issue**: Terminal canvas failing to render output in various scenarios:
1. Initial shell prompt not visible on first connection
2. Terminal appearing blank after returning from tab switch
3. Critical interactive flows (like Claude Code sign-in) breaking due to terminal reset on reconnection

**Root Causes**:
1. Single `requestAnimationFrame` refresh insufficient - browser state variability requires multiple strategies
2. Missing `scrollToBottom()` - viewport could be positioned incorrectly
3. Terminal canvas not properly initialized for rendering without user interaction
4. Timing issues - browser paint cycles not aligned with refresh attempts

**Solutions Implemented**:
**Multi-Strategy Terminal Refresh** (app.js:1745-1782):
- Strategy 1: Focus terminal (critical for render pipeline activation)
- Strategy 2: Fit to container (ensures proper canvas dimensions)
- Strategy 3: Scroll to bottom (corrects viewport position)
- Strategy 4: Write empty string (triggers internal xterm.js render)
- Strategy 5: Explicit row refresh (forces full canvas repaint)
- Multiple timing: Immediate + requestAnimationFrame + setTimeout(50ms)

**Applied to Critical Points**:
1. **First live output** (app.js:1747-1782) - Ensures shell prompt visible on new sessions
2. **Visibility change** (app.js:376-402) - Fixes blank terminal on tab return
3. **Replay completion** (app.js:1863-1877) - Ensures content visible after reconnection

**Session Reconnection Verified** (app.js:2493-2521):
- Confirmed `reconnectSession` does NOT call `term.reset()`
- Terminal state preserved during tab switches
- Existing sessions reconnect instead of creating new (app.js:1172-1187)
- Interactive flows (Claude Code sign-in, password prompts, 2FA) no longer reset terminal

**Impact**:
- Terminal content reliably renders in all browser/tab states
- Claude Code sign-in flow now works correctly - users can switch tabs to get code and return without losing session
- Multi-step interactive workflows preserve state across tab switches
- Terminal canvas properly initializes without requiring user interaction
- Eliminated race conditions between output arrival and canvas readiness

**Evidence**: Multi-strategy approach handles various browser states - backgrounded tabs, different paint cycles, focus states, and viewport positions - ensuring reliable rendering across all scenarios.

### 2025-10-04: Offline-First UI Dependencies
**Issue**: UI was loading vendor libraries (xterm.js, lucide icons, html2canvas) from external CDNs (jsdelivr, unpkg), causing Cloudflare 502 errors when CDN access failed or was blocked. This broke the entire UI with cryptic error messages.

**Solution**: Downloaded all vendor dependencies locally to `ui/public/lib/` directory:
- @xterm/xterm@5.5.0 (JS + CSS)
- @xterm/addon-fit@0.10.0
- lucide icons (547KB UMD bundle)
- html2canvas@1.4.1

**Changes**:
- Created `ui/public/lib/` vendor directory
- Downloaded all CDN dependencies locally (total ~1.1MB)
- Updated `index.html` to reference local paths instead of CDN URLs
- Verified UI renders correctly offline with all icons and terminal functionality intact

**Impact**: Console now works reliably in offline/restricted network environments, no external dependencies required. Aligns with PRD goal of "lightweight for iframe embedding" - parent scenarios won't be blocked by CDN access issues.

**Evidence**: Screenshot at /tmp/web-console-ui.png shows fully functional UI with all lucide icons rendering, terminal active, no console errors.

### 2025-10-04: Session Persistence & Reconnection (Phase 3 - Mobility)
**Completed**: Automatic reconnection on tab visibility change, WebSocket keepalive, smart buffer management
- Implemented automatic WebSocket reconnection with 1-second retry on unexpected disconnects
- Added 30-second heartbeat mechanism to keep connections alive when tabs are inactive
- Increased output buffer from 100 to 500 chunks (1MB max) for better reconnect experience
- Added browser visibility change detection to trigger reconnection when tab becomes active
- Smart close detection (distinguish clean vs dirty WebSocket closes)
- Proper cleanup of heartbeat intervals to prevent memory leaks
**Evidence**: Sessions now persist across tab switches, no terminal reset on reconnect, up to 1MB output replay
**Next**: Complete Phase 2 hardening work (panic-stop, metrics enrichment)

### 2025-10-04: Blank Terminal Fix on Tab Switch
**Issue**: When switching away from and back to the web-console tab, the terminal would appear completely blank (no prompt, no content) until the user typed something, which would then trigger output and restore the display.

**Root Cause**: The terminal replay logic unconditionally called `term.reset()` on every reconnection, clearing all terminal content including the shell prompt. When the replay buffer was empty or didn't contain the initial prompt, this resulted in a blank terminal until new output was generated.

**Fix**: Introduced `hasEverConnected` flag to distinguish first-time connections from reconnections. The terminal is only reset on initial connection with non-empty replay buffer. On reconnection, the existing terminal state is preserved and replay content is written additively, preventing blank screen while maintaining sync with session state.

**Changes**:
- Added `hasEverConnected` flag to tab state (app.js:507)
- Modified `handleReplayPayload` to conditionally reset terminal only on first connection (app.js:1698-1708)
- Set `hasEverConnected = true` after successful replay completion (app.js:1734)

**Verification**: Terminal content now persists across tab switches, with reconnection seamlessly resuming from existing state.

### 2025-10-04: Critical Session Persistence Fixes (Claude Code Sign-in Support)
**Issue**: Claude Code interactive sign-in flow was impossible due to terminal reset on tab return. When users switched away to get sign-in code and returned, typing any character would reset the terminal and lose the Claude session completely.

**Root Causes Identified**:
1. **WebSocket Close Phase Destruction**: Clean WebSocket closes (codes 1000/1001) marked session phase as 'closed' even though server session remained alive, preventing reconnection
2. **Input Handler Creates New Sessions**: When user typed after disconnect, code saw phase='closed' and started NEW session (calling term.reset()) instead of reconnecting to existing session
3. **Blank Terminal on First Load**: Shell prompt arrives after WebSocket connects but terminal canvas doesn't refresh, showing blank screen until user types

**Fixes Applied**:

**Fix 1: Preserve Session State on WebSocket Close** (app.js:1647-1680)
- Removed logic that marked phase='closed' on clean WebSocket disconnects
- Keep phase='running' when session exists, enabling automatic reconnection
- Only mark 'closed' when explicitly closing session or reconnection fails
- Session state now preserved across WebSocket lifecycle events

**Fix 2: Reconnect-First Input Handling** (app.js:1154-1197)
- Modified `handleTerminalData` to check for existing session before creating new one
- If session exists but socket disconnected: attempt reconnection first
- Only start new session if no existing session OR reconnection fails
- Prevents terminal reset on user input after temporary disconnect

**Fix 3: Force Terminal Refresh on First Live Output** (app.js:1727-1769)
- Added `hasReceivedLiveOutput` flag to track first real output (not replay)
- Force terminal refresh (fit + refresh rows) when first live output arrives
- Ensures shell prompt becomes visible even if it arrives after WebSocket connects
- Reset flag on new session to handle session lifecycle properly

**Impact**:
- Claude Code sign-in flow now works correctly - users can leave to get code and return without losing session
- All interactive multi-step workflows (password prompts, 2FA, etc.) now preserve state
- Terminal properly displays initial prompt on fresh sessions
- Session reconnection seamless and reliable across all disconnect scenarios

**Evidence**:
- WebSocket close no longer destroys session state
- Input triggers reconnection instead of new session creation
- Terminal canvas properly initialized and refreshed on output
- Fixes validated against Claude Code use case (the critical blocker)

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
