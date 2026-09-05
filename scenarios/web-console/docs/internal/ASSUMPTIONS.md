# Documented Assumptions

## Last Updated
2026-02-19

## Data Shape Assumptions
- **PTY output is binary-safe**: WebSocket framing preserves arbitrary byte sequences from PTY stdout. If a program emits raw binary (e.g., `cat /dev/urandom`), the output loop must not corrupt it. Made in [CODE: api/terminal_ws.go] and [CODE: ui/src/hooks/useTerminalSocket.ts].
- **Session IDs are UUIDs**: All session references assume `uuid.New().String()` format. No validation currently enforces this on the client side — the API generates IDs server-side in [CODE: api/session.go].
- **Shortcut profiles have unique IDs**: The `ShortcutStore.Upsert` method assumes the caller provides a globally unique ID. Collision behavior is "last write wins" (upsert semantics). Made in [CODE: api/shortcut_profiles.go] and [CODE: api/shortcut_profiles_pg.go].

## Behavioral Assumptions
- **Single operator per server**: No concurrent multi-user access. Session isolation, RBAC, and resource quotas are out of scope. This assumption is documented in the PRD and permeates the API (no auth middleware).
- **Parent scenario handles authentication**: Web console trusts that the embedding parent has already authenticated the user. No token validation occurs in [CODE: api/main.go].
- **Ollama is locally available**: AI generation assumes Ollama runs on `localhost:11434`. If unavailable, it falls back to OpenRouter (if configured). Made in [CODE: api/ai_generate.go].

## Timing Assumptions
- **WebSocket keepalive interval < proxy timeout**: The 30-second ping interval must be shorter than any reverse proxy's idle timeout (typically 60s). Made in [CODE: api/terminal_ws.go].
- **Schema initialization completes before first request**: `initSchema()` runs synchronously in `main()` before the HTTP server starts. No request can race with schema creation. Made in [CODE: api/main.go].
- **Expiration sweeper interval (60s) is acceptable resolution**: Session cleanup is not instant on TTL expiry — there's up to 60 seconds of drift. Made in [CODE: api/session_policy.go].

## Environment Assumptions
- **Platform-specific PTY seam**: Unix builds use `creack/pty` and Windows
  builds use the native ConPTY adapter. Persistent tmux sessions are Unix-only;
  the platform capability matrix in `.vrooli/service.json` is authoritative.
- **SQLite available via `api-core/database`**: The database connection string is resolved by the `api-core` library from environment variables. Made in [CODE: api/main.go].
- **SQL files exist relative to binary**: `initSchema()` reads `../api/internal/<domain>/schema.sql` relative to `os.Executable()`. Breaks if binary is moved without its sibling directories. Made in [CODE: api/main.go].

## Hardening Status
| Assumption | Status | Moved to INVARIANTS |
|------------|--------|---------------------|
| Single operator | implicit | no |
| Parent handles auth | implicit | no |
| PTY binary safety | validated (tests) | no |
| Schema before requests | hardened (sync init) | yes |
| Keepalive < proxy timeout | implicit | no |
| SQL files relative to binary | implicit | no |
