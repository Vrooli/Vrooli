# Web Console (web-console scenario)

Web Console delivers a full-fidelity terminal experience in the browser with pane-based workflows, durable sessions, and AI-assisted input generation for authenticated parent scenarios.

## Platform support

Linux is supported and covered by PTY/tmux integration tests. macOS uses the
Unix PTY path with Darwin termios support. Windows uses the native ConPTY
adapter; the Windows build is cross-compiled in validation, while tmux
persistence remains Unix-only. Platform claims and evidence live in
`.vrooli/service.json`.

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
8. Non-destructive session archive with cross-session transcript retrieval and bounded retention.

## Single-User Design

Web Console is designed for personal server use — a single operator running their own Vrooli instance. There is no multi-user session isolation, RBAC, or user management. Auth is delegated entirely to `packages/api-base`.

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
- Closing a pane archives it and preserves its transcript. Permanent deletion is a separate confirmed action.
- Archived sessions expose `Reopenable`, `Read-only`, or `Nothing to restore` before the operator acts.

### Groups, Roles, and Handoffs

- A **group** is one piece of work, named for the task. The new-session dialog
  states which group it will join on every open, and can create one by name
  without leaving the dialog.
- A **role** is a named position inside a group. A role is *running* when it
  holds a session and *waiting* when it holds a command and no session. A
  waiting role costs no process and no PTY, survives a restart, and renders as
  a dashed placeholder. Roles are optional: dragging a session into a group
  keeps working and creates none.
- A **group template** is a saved role list. Creating a group from one creates
  the group and its roles in a single action; only a role marked *starts now*
  spends a process.
- A **handoff** sends a message from one session to one or more targets in the
  same group, from the pane header, the file viewer, or a sidebar role. It
  starts a waiting target first, reports a per-target `sent`, `queued`, or
  `failed` result, and is never dropped silently. The payload is a file path, a
  passage, or nothing — the console never classifies it.
- A **capture rule** decides when a handoff is *suggested*. A rule never sends
  anything: a match is a dismissible chip that opens the same composer a button
  opens.
- A group with no sessions and no waiting roles closes itself, reversibly. The
  waiting-role exemption is what keeps a half-started group alive.

Shipped templates and rules are ordinary editable rows. Deleting them leaves
every capability working. Design record:
`docs/internal/ROLES-AND-HANDOFFS-UX.md`.

### Conversation Messages

- Messages use a bounded 500-event window. Opening a Messages pane reads the newest page; approaching the top loads the preceding page without shifting the visible content.
- `ConversationService.Get` accepts `limit` and `before_sequence`; callers that omit both retain the legacy complete-history response.
- Per-session whole-history search is server-side and debounced in the UI. Cross-session archive search uses SQLite FTS5 and returns sequence-addressable message excerpts across retained lineages.
- Message jump loads the page containing an off-window search result. Export Select All uses the server range endpoint, capped at 5,000 events, so it does not silently omit older history.
- Virtualized message and navigator lists keep first paint bounded even for multi-thousand-event sessions. Inactive Messages panes unmount while terminal panes remain warm.

### AI Input
- Command-generation UI should be available from main shell workspace.
- Provider policy:
  1. Try Ollama first.
  2. Fall back to OpenRouter if Ollama fails or times out.
- Provider routing must be explicit and observable.
- Context: user prompt + last N lines of terminal output (no environment injection).

### Launcher and Shortcuts
- New-terminal action presents:
  - Empty terminal
  - Configured shortcut entries
- Default shortcut entries:
  - `vrooli agent launch --runner claude --arg=--dangerously-skip-permissions`
  - `codex --yolo`

### Mobile Toolbar
- Floating keyboard toolbar should expose keys/chords needed for terminal workflows.
- Toolbar mode should be configurable per workspace/session context.
- Mobile is P0 — operators actively use phones for terminal access.

### Drawer/Sidebar
- Provides session summary, workspace controls, policy controls, and diagnostics.
- Must not block core terminal operations.

## Architecture Direction

### API Layer (Go)
- Session lifecycle endpoints for create/list/delete/inspect.
- WebSocket streaming for terminal IO.
- SQLite-backed transcript and session persistence.
- Session policy controls (never/preset/custom expiration).
- AI provider orchestration endpoint with fallback policy.

### UI Layer (Vite + xterm.js)
- Pane-based terminal workspace.
- Launcher flow and shortcut management UI.
- Drawer with status and controls.
- Floating keyboard toolbar.
- Shared `packages/api-base` for all HTTP/WebSocket routing.

### Integration Layer
- Parent embedding via iframe.
- `postMessage` bridge for status/events and parent operations.
- Auth/proxy boundaries enforced by parent stack via api-base.

## Storage Architecture

- **Backend**: SQLite (WAL mode) — no Redis or Postgres.
- **Schema**: Initialized on first API start; migrations managed via embedded SQL.
- **Isolation**: Database file scoped to scenario data directory.
- **Rationale**: Single-user design bounds concurrency; SQLite simplifies deployment and eliminates external resource dependencies.

## Dependencies

### Required
- Go toolchain
- POSIX shell runtime
- `resource-ollama`
- `packages/api-base`
- Scenario-managed local storage path

### Optional
- `resource-openrouter` (fallback AI provider)

## Configuration Model

- Session policy: default `never` expiration, configurable per workspace.
- Shortcut catalog: configuration-driven (service/workspace/parent context).
- AI provider policy: provider order and timeout configurable.
- SQLite database path: configurable via environment variable.
- Routing: relies on shared api-base resolution behavior.

## Validation and Testing Expectations

- Requirements coverage tracked in `requirements/index.json`.
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
make start
make test
make logs
make stop
```

## Notes

- Avoid direct process execution outside scenario lifecycle commands.
- Keep docs and requirements in lockstep as rewrite decisions evolve.
