# Product Requirements Document (PRD)

> **Scenario**: web-console
> **Version**: 2.0.0
> **Last Updated**: 2025-11-19
> **Status**: Active
> **Template**: Canonical PRD v2.0.0

## 🎯 Overview

**Purpose**: Web Console delivers lifecycle-managed PTY sessions streamed over WebSockets, enabling authenticated remote shell access with configurable commands and UI shortcuts. The capability transforms ad-hoc SSH workflows into auditable, iframe-embeddable terminal experiences integrated with Vrooli's resource ecosystem.

**Primary Users**:
- On-call engineers and operators inside authenticated dashboards (Ecosystem Manager, Maintenance Orchestrator)
- Support teams escalating to Codex or Claude mid-incident
- Future scenarios requiring "press-to-launch" automation against local CLIs

**Deployment Surfaces**:
- Go API (REST + WebSocket) for session lifecycle and streaming
- Browser UI (Vite + vanilla JS + xterm.js) with shortcut palette
- iframe bridge for parent scenario integration via `postMessage`
- CLI for local testing and development

**Revenue Opportunity**: $35K–$85K annual value through mobile-ready shell access, faster on-call remediation, and reusable shortcut workflows across the ecosystem.

## 🎯 Operational Targets

### 🔴 P0 – Must ship for viability
- [x] OT-P0-001 | Session Lifecycle API | Go API providing session lifecycle endpoints, WebSocket streaming, and transcript persistence
- [x] OT-P0-002 | Configurable Default Commands | Configurable default command (`WEB_CONSOLE_DEFAULT_COMMAND`) plus per-session overrides via API
- [x] OT-P0-003 | Browser UI with Shortcuts | Browser UI (vanilla JS + xterm.js) with shortcut buttons for Codex, Claude, and `vrooli status` that queue commands until the PTY is ready
- [x] OT-P0-004 | Iframe Bridge Protocol | iframe bridge that publishes status/metadata via `postMessage` and responds to transcript/log/screenshot requests
- [x] OT-P0-005 | Proxy Guard Enforcement | Proxy guard enforcement that rejects requests lacking authenticated forwarding headers
- [x] OT-P0-006 | Security Documentation | Documentation emphasizing proxy-only deployment and shortcut behavior

### 🟠 P1 – Should have post-launch
- [ ] OT-P1-001 | Concurrent Session Handling | Concurrent session handling with queueing when capacity is exhausted
- [ ] OT-P1-002 | Prometheus Metrics | Prometheus metrics endpoint for session counters and failure modes
- [ ] OT-P1-003 | Panic-Stop Endpoint | Panic-stop endpoint/UI control to terminate runaway commands
- [x] OT-P1-004 | Mobile-First UI | Mobile-first UI with reconnect/resume flows and quick command access

### 🟢 P2 – Future / expansion
- [ ] OT-P2-001 | Configurable Shortcut Palette | Configurable shortcut palette provided by parent scenario or service config
- [ ] OT-P2-002 | Distributed Transcript Storage | Redis/Postgres adapters for distributed transcript storage
- [ ] OT-P2-003 | Voice Dictation/TTS Integration | Voice dictation/TTS hooks that leverage shared resources
- [ ] OT-P2-004 | Parent Callbacks | Parent callbacks for automated post-session archiving or compliance workflows

## 🧱 Tech Direction Snapshot

**Backend**: Go 1.21+ for API service, PTY management, and session supervision. Aligns with Vrooli's preferred service stack.

**Frontend**: Vite build pipeline with vanilla JS + xterm.js. Lightweight for iframe embedding; no heavy frameworks. Vendor dependencies stored locally (`ui/public/lib/`) for offline-first reliability.

**Data Storage**: Local filesystem for transcript persistence (NDJSON format). Future adapters for Redis/Postgres enable distributed storage.

**Integration Strategy**:
- Designed for iframe embedding within authenticated parent scenarios
- WebSocket-based streaming with heartbeat keepalive
- `postMessage` protocol for parent communication
- Shortcut commands queue until PTY readiness to prevent race conditions

**Security Model**: Proxy-only deployment enforced via header validation. No direct auth—parent scenarios provide identity and policy gates.

**Non-Goals**:
- Embedded authentication/RBAC (parent's responsibility)
- Multi-user collaborative sessions
- Recording/playback of terminal sessions (transcripts only)

## 🤝 Dependencies & Launch Plan

**Required Local Resources**:
- Go 1.21+ (API + supervisor implementation)
- POSIX shell (default `/bin/bash`) for session execution
- Local filesystem for transcript storage under scenario-managed paths

**Optional Resources**:
- Codex CLI (triggered via shortcut: `codex --yolo`)
- Claude CLI (triggered via shortcut: `claude --dangerously-skip-permissions`)
- Redis/Postgres (future distributed transcript storage)
- Authenticated proxy/parent scenario (required front door)

**Launch Sequencing**:
1. **Phase 1 – Reorientation** (✅ Completed): Default shell command, shortcut queue, updated UI copy, docs overhaul
2. **Phase 2 – Hardening** (In Progress): TTL enforcement validation, panic-stop telemetry, richer metrics, shortcut configuration hooks
3. **Phase 3 – Mobility** (✅ Completed): Reconnect/resume states, offline transcript cache, responsive polish
4. **Phase 4 – Scale-Up** (Pending): Multi-session queueing, Redis/Postgres persistence, parent callbacks
5. **Phase 5 – Assistive Features** (Pending): Voice dictation, push notifications, programmable shortcut palettes

**Risk Mitigation**:
- **Security Misconfiguration (High)**: Deploying without proxy/auth. Mitigation: proxy guard rejection, docs, runtime warnings
- **Runaway Commands (Medium)**: Long-lived shells or processes. Mitigation: TTL, idle timeout, panic-stop endpoint
- **Shortcut Drift (Medium)**: Commands become stale or unsafe. Mitigation: configurable palette, versioned docs, parent overrides
- **Transcript Sensitivity (Low)**: Sensitive data captured. Mitigation: retention policies, redaction hooks

## 🎨 UX & Branding

**Visual Tone**: Mobile-first layout with sticky input bar and easily tappable shortcuts. Clear status badge showing terminal/connection state. Clean, functional terminal aesthetic prioritizing readability over decoration.

**Accessibility**: WCAG 2.1 AA minimum. High-contrast terminal themes, keyboard-navigable shortcuts, screen reader compatibility for status messages.

**Voice/Personality**: Operational and safety-conscious. Prominent notices remind operators the console must remain behind authenticated proxies. Event feed provides transparency into lifecycle events, shortcut dispatches, and suppression counters.

**Branding Guardrails**: Vendor libraries (xterm.js, lucide icons, html2canvas) stored locally for offline reliability. No external CDN dependencies to prevent Cloudflare 502 errors in restricted networks.

## 📎 Appendix

### Performance Criteria
| Metric | Target | Measurement |
|--------|--------|-------------|
| Session bootstrap time | < 4s | API timing logs |
| WebSocket latency | < 500ms p95 | Heartbeat telemetry |
| Concurrent sessions | Baseline 20 | Supervisor metrics |
| Shortcut dispatch delay | < 1s from PTY ready | UI event timestamps |
| Transcript durability | 100% persisted | Integration tests & crash sims |
| Memory usage | < 250MB per session | `psutil`/cgroup statistics |

### Session Flow
1. Parent scenario loads console inside authenticated iframe
2. Operator starts shell or taps shortcut; UI queues shortcut commands until PTY ready
3. UI calls `POST /api/v1/sessions` (with optional `command`/`args`) via `proxyToApi`
4. Go supervisor spawns command inside PTY, records transcripts, enforces TTL/idle policies
5. UI upgrades to WebSocket (`/api/v1/sessions/{id}/stream`) for bidirectional messaging
6. Session ends via stop request, TTL expiry, idle timeout, or panic-stop

### Progress History

For implementation history, see below dated entries (retained for context but not part of canonical PRD structure):

#### 2025-10-05: Multi-Strategy Terminal Rendering Fixes
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

#### 2025-10-04: Offline-First UI Dependencies
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

#### 2025-10-04: Session Persistence & Reconnection (Phase 3 - Mobility)
**Completed**: Automatic reconnection on tab visibility change, WebSocket keepalive, smart buffer management
- Implemented automatic WebSocket reconnection with 1-second retry on unexpected disconnects
- Added 30-second heartbeat mechanism to keep connections alive when tabs are inactive
- Increased output buffer from 100 to 500 chunks (1MB max) for better reconnect experience
- Added browser visibility change detection to trigger reconnection when tab becomes active
- Smart close detection (distinguish clean vs dirty WebSocket closes)
- Proper cleanup of heartbeat intervals to prevent memory leaks
**Evidence**: Sessions now persist across tab switches, no terminal reset on reconnect, up to 1MB output replay
**Next**: Complete Phase 2 hardening work (panic-stop, metrics enrichment)

#### 2025-10-04: Blank Terminal Fix on Tab Switch
**Issue**: When switching away from and back to the web-console tab, the terminal would appear completely blank (no prompt, no content) until the user typed something, which would then trigger output and restore the display.

**Root Cause**: The terminal replay logic unconditionally called `term.reset()` on every reconnection, clearing all terminal content including the shell prompt. When the replay buffer was empty or didn't contain the initial prompt, this resulted in a blank terminal until new output was generated.

**Fix**: Introduced `hasEverConnected` flag to distinguish first-time connections from reconnections. The terminal is only reset on initial connection with non-empty replay buffer. On reconnection, the existing terminal state is preserved and replay content is written additively, preventing blank screen while maintaining sync with session state.

**Changes**:
- Added `hasEverConnected` flag to tab state (app.js:507)
- Modified `handleReplayPayload` to conditionally reset terminal only on first connection (app.js:1698-1708)
- Set `hasEverConnected = true` after successful replay completion (app.js:1734)

**Verification**: Terminal content now persists across tab switches, with reconnection seamlessly resuming from existing state.

#### 2025-10-04: Critical Session Persistence Fixes (Claude Code Sign-in Support)
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
