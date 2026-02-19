# Requirements Context: Web Console

> Prepared for `prd-control-tower requirements generate` consumption.
> Source: archive/requirements/index.json + clarification decisions.

## Validation Approach

### Testing Strategy

Requirements use `[REQ:WC-...]` tags for traceability. Each requirement maps to a PRD operational target via `prd_ref`.

**Test Phases:**
- **Unit tests**: Core logic isolation (session policy, provider selection, API routing)
- **Integration tests**: Component interaction (WebSocket streams, PTY resize, reconnect flows, AI provider fallback sequencing)
- **E2E / Manual**: Full interactive validation (Claude Code flows, Codex flows, mobile toolbar usability)

**Key Test Areas by Requirement:**

| Requirement | Test Type | Focus |
|-------------|-----------|-------|
| WC-P0-001 (Pane Layout) | Integration | Responsive breakpoint behavior, simultaneous pane rendering |
| WC-P0-002 (Terminal Fidelity) | Integration + E2E | Binary/text frame handling, resize, cursor reporting, interactive CLI flows |
| WC-P0-003 (Session Durability) | Unit + Integration | Never-expire default, reconnect continuity, transcript hydration/replay |
| WC-P0-004 (Networking) | Unit | API and WebSocket URL resolution under proxied embedding |
| WC-P0-005 (AI Fallback) | Integration | Ollama → OpenRouter failover sequencing, timeout handling, response normalization |
| WC-P0-006 (Launcher) | Integration | Launcher choice flow, default shortcut entries present |
| WC-P0-007 (Mobile Toolbar) | Integration | Key injection mappings, toolbar modes |
| WC-P0-008 (Drawer) | Integration | Drawer state, control actions, terminal continuity not disrupted |

## Technical Constraints

### Storage
- **SQLite only** — no Redis or Postgres dependencies (per clarification Q3)
- Must follow `api-core/storage` patterns and storage-steer conventions
- Scenario-isolated database (no cross-scenario pollution)
- Idempotent schema initialization
- WAL mode recommended for read/write parallelism under single-user load

### Networking
- All traffic via `@vrooli/api-base` — no direct-origin URL construction
- WebSocket upgrade must work through parent proxy layers
- iframe embedding uses `postMessage` bridge (not direct DOM access)

### AI Provider
- Context depth: user prompt + last N lines of terminal output (N is configurable)
- No shell environment variables or working directory sent to LLM
- Provider routing must be explicit and observable (not silent fallback)
- Timeout-based failover: Ollama timeout triggers OpenRouter attempt

### User Model
- Single-user only (per clarification Q6) — no session multiplexing or user isolation
- No authentication implementation required — delegated to api-base

### Mobile
- Mobile is P0 critical (per clarification Q4) — floating toolbar is not deferrable
- Must work on phone browsers for real terminal operations

## Dependency Relationships

```
web-console (scenario)
├── packages/api-base (required) — HTTP/WS routing, proxy compat, auth
├── resource-ollama (required) — primary AI provider
├── resource-openrouter (optional) — fallback AI provider
├── api-core/storage (required) — SQLite abstraction
├── Go toolchain 1.21+ (required) — API compilation
├── POSIX runtime (required) — PTY allocation
└── xterm.js (required) — terminal rendering
```

## Operational Target Elaboration

### WC-P0-003: Session Durability Details
- Default expiration policy: `never` (sessions persist indefinitely until explicit deletion)
- Reconnect behavior: client reconnects → server replays buffered output → session resumes
- Transcript storage: SQLite table with session_id, timestamp, output chunks
- Offline output: all PTY output is captured to SQLite while client is disconnected

### WC-P0-005: AI Provider Fallback Details
- Provider chain: [Ollama, OpenRouter]
- On Ollama timeout/error → attempt OpenRouter with same prompt
- Context assembly: last N lines of visible terminal output (configurable, suggest default N=50)
- Response format: plain text command suggestion displayed in UI input area
- Provider status must be visible in drawer/sidebar

### WC-P0-006: Launcher Shortcut Details
- Default shortcut entries (shipped with scenario):
  - `claude --dangerously-skip-permissions` — Claude Code autonomous mode
  - `codex --yolo` — Codex autonomous mode
- Shortcuts are configurable (add/remove/reorder)
- Launcher UI: modal or inline selector offering empty shell + shortcut list
