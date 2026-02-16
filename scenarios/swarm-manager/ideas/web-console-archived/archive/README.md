# Web Console (web-console scenario)

Web Console is a rewrite-focused scenario that targets a no-compromise browser terminal experience for authenticated parent scenarios. This README defines the intended architecture and delivery contract for the upcoming rebuild.

## Scope Status

- This document describes the **target rewrite scope**.
- Current implementation may not match every requirement yet.
- Product contract is defined by `PRD.md` + `requirements/index.json`.

## Product Goals

1. Pane-based terminal workspace showing multiple terminals simultaneously.
2. Full interactive CLI fidelity in browser (Claude Code/Codex-class flows).
3. Durable sessions with default expiration policy set to never.
4. AI command input with Ollama primary and OpenRouter fallback.
5. Mobile usability through floating terminal keyboard controls.
6. Configurable new-terminal launcher with empty shell and shortcut options.
7. Sidebar/drawer for operational controls and status.

## Target UX and Behavior

### Pane Layout
- Desktop: two-column pane layout by default.
- Mobile: single-column stacked pane layout.
- Layout should preserve active session continuity while resizing.

### Terminal Fidelity
- PTY-backed stream handling with binary-safe input/output.
- Reliable resize and cursor-report handling.
- Reconnect-safe input sequencing.
- Interactive CLIs must remain usable after tab visibility changes and reconnects.

### Session Continuity
- Default policy: never expire.
- Refreshing page or reconnecting should restore session context.
- Missed output while disconnected must be replayed and visible.

### AI Input
- Command-generation UI should be available from main shell workspace.
- Provider policy:
  1. Try Ollama first.
  2. Fall back to OpenRouter if Ollama fails or times out.
- Provider routing must be explicit and observable.

### Launcher and Shortcuts
- New-terminal action presents:
  - Empty terminal
  - Configured shortcut entries
- Default shortcut entries:
  - `claude --dangerously-skip-permissions`
  - `codex --yolo`

### Mobile Toolbar
- Floating keyboard toolbar should expose keys/chords needed for terminal workflows.
- Toolbar mode should be configurable per workspace/session context.

### Drawer/Sidebar
- Provides session summary, workspace controls, policy controls, and diagnostics.
- Must not block core terminal operations.

## Architecture Direction

### API Layer (Go)
- Session lifecycle endpoints for create/list/delete/inspect.
- WebSocket streaming for terminal IO.
- Durable transcript/session persistence model.
- Session policy controls (never/preset/custom expiration).
- AI provider orchestration endpoint with fallback policy.

### UI Layer (Vite + xterm.js)
- Pane-based terminal workspace.
- Launcher flow and shortcut management UI.
- Drawer with status and controls.
- Floating keyboard toolbar.
- Shared `@vrooli/api-base` for all HTTP/WebSocket routing.

### Integration Layer
- Parent embedding via iframe.
- `postMessage` bridge for status/events and parent operations.
- Auth/proxy boundaries enforced by parent stack.

## Dependencies

### Required
- Go toolchain
- POSIX shell runtime
- `resource-ollama`
- Authenticated reverse proxy headers
- Scenario-managed local storage path

### Optional
- `resource-openrouter` (fallback provider)
- Redis/Postgres adapters for future persistence extensions

## Configuration Model (Target)

- Session policy should support default `never` expiration.
- Shortcut catalog should be configuration-driven.
- AI provider policy should define provider order and timeout.
- Routing should rely on shared api-base resolution behavior.

## Validation and Testing Expectations

- Requirements coverage should be tracked in `requirements/index.json`.
- Tests should include `[REQ:WC-...]` tags for stable sync.
- Acceptance coverage should include:
  1. Interactive CLI fidelity (Claude Code-class flows)
  2. Reconnect and offline output replay
  3. Responsive pane layout behavior
  4. AI provider fallback behavior
  5. Launcher + shortcut configuration behavior

## Lifecycle Commands

```bash
cd scenarios/web-console
make setup
make run
make test
make logs
make stop
```

## Notes

- Avoid direct process execution outside scenario lifecycle commands.
- Keep docs and requirements in lockstep as rewrite decisions evolve.
