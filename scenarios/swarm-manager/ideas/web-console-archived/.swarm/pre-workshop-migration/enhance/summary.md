# Enhanced Plan: Web Console

## Overview

Web Console is a standalone Vrooli scenario delivering a full-fidelity browser terminal experience. It provides pane-based terminal workspaces with durable sessions, AI-assisted command generation, configurable launch shortcuts, and mobile-first usability. The scenario will be revived from its archived state as a standalone scenario (not merged into agent-manager or app-monitor). Storage uses SQLite exclusively — no Redis or Postgres dependencies. Authentication and proxy networking are handled entirely by `packages/api-base`, requiring no custom auth implementation.

## Clarifications Applied

| Question | Answer | Impact |
|----------|--------|--------|
| Revive vs. merge vs. keep archived? | Revive as standalone web-console scenario | Scenario gets its own directory, Makefile lifecycle, independent deployment |
| Parent embedding auth contract? | Handled by packages/api-base adoption | No custom auth headers or proxy middleware needed; api-base resolves routing |
| MVP persistence backend? | SQLite only; remove Redis/Postgres from spec | Simplifies dependencies; uses storage-steer patterns for SQLite isolation |
| Mobile toolbar priority? | Mobile is critical (P0) | Floating keyboard toolbar stays in MVP scope |
| AI command context depth? | User prompt + last N lines of terminal output | Stateful but bounded context; no shell env or cwd injection |
| Multi-user support? | Single-user only (personal server) | No auth/session multiplexing; simplifies session model |
| Performance targets? | No specific targets — just feels responsive | No latency SLAs; focus on perceived responsiveness |

## Suggestions Integrated

### Accepted

No suggestions phase was executed for this item.

### Not Accepted

N/A — no suggestions to reject.

## Refined Scope

### Included (Must Have — P0)
- Pane-based terminal workspace: 2-column desktop, 1-column mobile, simultaneous terminals
- Production-grade PTY fidelity: resize, cursor reporting, binary-safe I/O, reconnect-safe input
- Durable session continuity: default never-expire policy, reconnect restores state, offline output replay
- Proxy-correct networking via shared api-base (HTTP and WebSocket)
- AI command generation: Ollama primary, OpenRouter fallback, deterministic failover
- New terminal launcher with configurable shortcuts (defaults: `claude --dangerously-skip-permissions`, `codex --yolo`)
- Mobile floating keyboard toolbar for practical terminal use
- Sidebar/drawer for session status, workspace controls, diagnostics

### Included (Should Have — P1)
- Per-session/workspace expiration policy controls (never/preset/custom)
- Configuration-driven shortcut profiles (not hardcoded)
- AI provider policy controls with health visibility
- Operational observability: metrics and structured events for lifecycle, reconnects, pane actions, failovers

### Excluded (Out of Scope)
- Multi-user / multi-tenant support — single-user personal server only
- Redis / Postgres persistence adapters — SQLite is the only storage backend
- Built-in public auth surface — auth is delegated to parent via api-base
- Direct internet exposure — always behind authenticated proxy
- Collaborative / shared terminal sessions — deferred to P2
- Persisted workspace presets — deferred to P2

### Deferred (Future — P2)
- Collaborative session modes (observer/shared-view) — Target: P2
- Persisted workspace presets (save/load named layouts) — Target: P2

## Implementation Notes

### Technical Approach
- **API**: Go service with session lifecycle endpoints (create/list/delete/inspect), WebSocket terminal streaming, SQLite-backed session/transcript persistence, AI provider orchestration with fallback policy
- **UI**: Vite + xterm.js, pane-based workspace layout, launcher UI, floating mobile toolbar, drawer controls
- **Storage**: SQLite via `api-core/storage` patterns per storage-steer conventions. Schema initialization is idempotent. Database is scenario-isolated.
- **AI Provider**: Ollama first → OpenRouter fallback. Context sent to LLM: user prompt + last N lines of terminal output (configurable N). Provider routing is explicit and observable.
- **Networking**: All HTTP/WebSocket traffic through shared `@vrooli/api-base` helpers. No direct-origin assumptions. iframe embedding uses `postMessage` bridge for status/events.

### Integration Points
- `packages/api-base`: HTTP/WebSocket routing, proxy compatibility, auth delegation
- `resource-ollama`: Primary AI provider for command generation
- `resource-openrouter`: Fallback AI provider
- Parent scenarios: iframe embedding via postMessage bridge contract
- `api-core/storage`: SQLite filesystem storage abstraction

### Dependencies
- Go toolchain (1.21+)
- POSIX shell runtime (PTY allocation)
- `resource-ollama` (required — primary AI provider)
- `resource-openrouter` (optional — fallback AI provider)
- `packages/api-base` (required — networking and auth)
- SQLite (embedded, no external service)

### Risks & Mitigations
| Risk | Mitigation |
|------|------------|
| Interactive CLI fidelity regressions (Claude Code, Codex) | Dedicated e2e test suite for interactive flows; manual validation checklist |
| Session loss on reconnect | SQLite-backed transcript persistence with replay; integration tests for reconnect path |
| Mobile input friction | P0 floating toolbar with configurable key/chord mappings; user testing on real devices |
| AI provider outages | Deterministic Ollama → OpenRouter fallback; clear provider health signaling in UI |
| SQLite concurrency under load | Single-user assumption bounds concurrency; WAL mode for read/write parallelism |

## Success Criteria
- [ ] Interactive CLIs (Claude Code, Codex) run correctly with full PTY fidelity
- [ ] Page refresh restores session state with no output loss
- [ ] AI command generation works with Ollama and falls back to OpenRouter
- [ ] Mobile toolbar provides usable terminal experience on phone browsers
- [ ] Launcher offers empty shell and configurable shortcut entries
- [ ] All HTTP/WebSocket traffic routes correctly under parent proxy embedding
- [ ] SQLite persistence survives scenario restart without data loss
- [ ] Drawer provides session status without blocking terminal workflow

## Readiness Gate
- [x] All critical questions answered — all 7 questions have definitive answers
- [x] Scope clearly defined — included/excluded/deferred sections complete
- [x] Technical approach validated — no research showstoppers flagged
- [x] Dependencies available — Go, Ollama, api-base all exist in Vrooli ecosystem
- [x] Success criteria measurable — 8 concrete criteria defined
- [x] Archive materials incorporated into staging artifacts — PRD, README, requirements all synthesized

**Ready for processing:** Yes

## Staging Artifacts Produced
- `enhance/prd-context.md` — Synthesized PRD context brief incorporating all clarification decisions, ready for prd-control-tower consumption
- `enhance/requirements-context.md` — Requirements context with validation approach, technical constraints, and operational target details
- `enhance/doc-outlines.md` — Documentation outlines for README, PROBLEMS.md, and PROGRESS.md initial entries
