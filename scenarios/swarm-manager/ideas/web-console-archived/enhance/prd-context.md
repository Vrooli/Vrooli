# PRD Context Brief: Web Console

> Prepared for `prd-control-tower prd generate` consumption.
> All clarification decisions are incorporated as definitive statements.

## Overview & Value Proposition

Web Console delivers a zero-compromise browser terminal experience for operators running interactive CLI workflows on personal Vrooli servers. It provides pane-based multi-terminal workspaces, durable sessions that survive disconnects, AI-assisted command generation, and mobile-first usability — all behind authenticated proxy embedding via api-base.

**Primary Users:**
- Solo operators and engineers running interactive CLI workflows (Claude Code, Codex, diagnostics) on their personal Vrooli server
- Parent scenarios embedding terminal capability via iframe
- Mobile operators needing practical terminal controls without SSH clients

**Deployment Surface:**
- Standalone Vrooli scenario with Go API + Vite/xterm.js UI
- iframe embeddable by any parent scenario via postMessage bridge
- Single-user design (no multi-tenant requirements)

## P0 Operational Targets (Must Ship)

1. **Pane-Based Terminal Workspace** — Multi-pane layout with 2-column desktop / 1-column mobile. Simultaneous active terminals with responsive layout transitions.

2. **Production-Grade Web Terminal Fidelity** — PTY-backed streams with binary-safe I/O, resize handling, cursor reporting, and reconnect-safe input sequencing. Must support Claude Code and Codex interactive flows.

3. **Durable Session Continuity** — Default session expiration: never. Reconnect restores live state. Output generated while client was offline is replayed on reconnect. Persistence via SQLite only (no Redis/Postgres).

4. **Proxy-Correct Networking** — All HTTP/WebSocket traffic through shared `@vrooli/api-base`. No direct-origin assumptions. Authentication handled by api-base adoption — no custom auth implementation needed.

5. **AI Command Generation with Fallback** — Ollama primary provider, OpenRouter fallback. Deterministic failover behavior. Context sent to LLM: user prompt + last N lines of terminal output. No shell environment or working directory injection.

6. **Terminal Launcher with Shortcuts** — New-terminal flow offers empty shell and configurable shortcut entries. Default shortcuts: `claude --dangerously-skip-permissions` and `codex --yolo`.

7. **Mobile Terminal Toolbar** — Floating keyboard toolbar with common terminal keys/chords for practical mobile usage. This is critical priority — operators actively use phones for terminal access.

8. **Sidebar/Drawer Controls** — Session/workspace status, controls, and diagnostics in a drawer that does not block terminal workflow.

## P1 Operational Targets (Should Have Post-Launch)

1. **Session Policy Controls** — Per-workspace/session expiration controls (never/preset TTL/custom) with explicit persistence behavior.

2. **Shortcut Profile Management** — Configuration-driven shortcut catalog loaded from service/workspace/parent context rather than hardcoded.

3. **AI Provider Policy Controls** — Configurable provider priority, timeout, and fallback ordering with surfaced provider health status.

4. **Operational Observability** — Metrics and structured events covering session lifecycle, reconnects, pane actions, and AI provider failovers.

## P2 Operational Targets (Future)

1. **Collaborative Session Modes** — Optional observer/shared-view for paired operations.
2. **Persisted Workspace Presets** — Save/load named pane + shortcut presets.

## Tech Direction Snapshot

- **API**: Go 1.21+, session lifecycle REST endpoints, WebSocket terminal streaming
- **UI**: Vite + xterm.js, vanilla JS approach (per original tags), pane layout system
- **Storage**: SQLite exclusively, via `api-core/storage` patterns. Scenario-isolated database. No Redis, no Postgres.
- **Networking**: `@vrooli/api-base` for all HTTP/WebSocket routing. iframe embedding via postMessage bridge.
- **AI**: Ollama primary → OpenRouter fallback. Bounded context (prompt + last N terminal lines).
- **Non-goals**: Public auth surface, direct internet exposure, multi-user support, replacing parent RBAC

## Dependencies & Launch Plan

**Required:**
- Go toolchain (1.21+)
- POSIX shell runtime
- `resource-ollama`
- `packages/api-base`
- SQLite (embedded)

**Optional:**
- `resource-openrouter` (fallback AI provider)

**Launch Sequence:**
1. Ship P0: terminal fidelity, session durability, networking, AI assist, mobile toolbar, launcher, drawer
2. Ship P1: policy controls, shortcut profiles, provider config, observability
3. Ship P2: collaboration modes, workspace presets

**Operational Risks:**
- Session loss regressions during reconnect flows
- Interactive CLI incompatibilities (PTY edge cases)
- AI provider outages affecting command generation
- Mobile input friction on diverse device/browser combinations

## UX & Branding

- Utility-first terminal workspace with pane density and explicit state signaling
- WCAG 2.1 AA target: keyboard-only support, readable contrast, labeled controls
- Operational and direct voice; prioritize clarity around session state, provider status, proxy boundaries
- Keep critical UI dependencies locally available for offline/restricted-network reliability
