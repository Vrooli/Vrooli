# Web Console — Glossary

## Last Updated
2026-08-18

## Core Terms

| Term | Definition | Where Used |
|------|-----------|------------|
| **Session** | A server-side PTY process with associated metadata (shell, cols, rows, policy). Each session has exactly one PTY and zero or more WebSocket subscribers. | [CODE: api/session.go], [CODE: ui/src/hooks/useSessionManager.ts] |
| **Archive** | The non-destructive result of closing a pane. The session row, pane identity, and conversation remain; `sessions.archived_at` records when it was archived. | [CODE: api/internal/sessionstore/store.go], [CODE: ui/src/components/ArchiveDrawer.tsx] |
| **Reopen** | Create a replacement live session from an archived row by using the replay-safe recovery workflow and linking the old row through `recovered_into`. | [CODE: api/handlers/sessions/adapter.go] |
| **Reopenable** | Restore state where agent identity and the exact session-owned history are present, so Reopen can resume the agent. | [CODE: api/handlers/sessions/adapter.go] |
| **Read-only** | Restore state where the conversation is intact and searchable but agent history is unavailable, so the session cannot resume. | [CODE: api/handlers/sessions/adapter.go] |
| **Nothing to restore** | Restore state for an archive with no resumable agent and no captured messages. | [CODE: api/handlers/sessions/adapter.go] |
| **Pane** | A UI-side terminal viewport bound to one session. The workspace displays panes in a grid layout. | [CODE: ui/src/components/Workspace.tsx], [CODE: ui/src/components/TerminalPane.tsx] |
| **Workspace** | The main terminal view containing one or more panes. Desktop shows 2-column layout; mobile shows 1-column. | [CODE: ui/src/components/Workspace.tsx] |
| **PTY** | Pseudo-terminal — a kernel abstraction providing terminal I/O for a shell process. Abstracted behind the `PTY` interface for testability. | [CODE: api/pty.go] |
| **PTYFactory** | Injection point for creating PTY instances. Default creates real PTYs; tests substitute fakes. | [CODE: api/pty.go#PTYFactory] |
| **Shortcut** | A named command template displayed in the terminal launcher for one-click session creation. | [CODE: ui/src/consts/shortcuts.ts], [CODE: api/shortcut_profiles.go] |
| **Shortcut Profile** | A named set of shortcuts scoped to `service`, `workspace`, or `parent` context. Higher scopes override lower ones via `Effective()`. | [CODE: api/shortcut_profiles.go], [CODE: api/repository.go#ShortcutStore] |
| **Scope** | Priority level for shortcut profiles: `parent` > `workspace` > `service`. The highest available scope provides the effective shortcuts. | [CODE: api/shortcut_profiles.go] |
| **Provider** | An AI service (Ollama or OpenRouter) used for command generation. Providers have config (priority, timeout) and runtime health tracking. | [CODE: api/ai_generate.go], [CODE: api/ai_provider_config.go] |
| **Provider Chain** | The ordered list of enabled providers tried during AI generation. Lower priority number = tried first. | [CODE: api/ai_generate.go] |
| **Policy** | Session expiration configuration. Options: `never`, preset TTL (1h, 8h, 24h), or custom duration. | [CODE: api/session_policy.go], [CODE: ui/src/consts/policy-options.ts] |
| **Sweeper** | Background goroutine that periodically checks sessions against their expiration policy and closes expired ones. | [CODE: api/session_policy.go] |
| **Offline Buffer** | Per-session ring buffer that stores PTY output generated while no WebSocket clients are connected, enabling reconnect continuity. | [CODE: api/session.go] |
| **Idempotency Key** | Client-provided header (`X-Idempotency-Key`) for replay-safe session creation. Cached results prevent duplicate sessions on retry. | [CODE: api/session_handlers.go] |
| **iframe Bridge** | `@vrooli/iframe-bridge` library providing `postMessage`-based coordination between the embedded web-console UI and its parent scenario. | [CODE: ui/src/App.tsx] |
| **api-base** | `@vrooli/api-base` library providing proxy-correct HTTP/WebSocket URL resolution for parent-embedded scenarios. | [CODE: ui/src/lib/api.ts] |
| **Error Catalog** | Server-side registry mapping error codes to structured error responses with category, recovery hints, and retry guidance. | [CODE: api/errors.go] |
| **Metrics** | Operational counters tracking session lifecycle events, WebSocket connections, AI generation attempts, and provider health. | [CODE: api/metrics.go] |
