# Security Posture

## Last Updated
2026-02-19

## Trust Model
Web console is designed for single-operator use on a personal Vrooli server. It runs behind an authenticated parent scenario's proxy — the console itself performs **no authentication or authorization**. This is by design ([CODE: api/main.go], PRD non-goals).

## Hardening Status by Category

### Secrets Management
- [x] No hardcoded secrets in source code
- [x] AI provider API keys passed via environment variables only
- [x] Database credentials resolved via `api-core/database` (env-based)
- [ ] No secret rotation support (single-operator, acceptable risk)
Status: **hardened** (for single-operator model)

### Authentication & Authorization
- [ ] No auth middleware — delegated to parent proxy
- [ ] No session ownership enforcement (single operator)
- [ ] No RBAC or permission checks
Status: **by-design** — auth is parent's responsibility. If web-console is ever exposed directly, auth must be added.

### Input Validation
- [x] All JSON request bodies decoded with size limits (1MB default via api-core)
- [x] Session creation validates shell path, cols/rows ranges
- [x] Shortcut profile validation (required fields, valid scopes)
- [x] AI config validation (timeout 1-120s, retries 0-5)
- [x] Policy validation (valid duration strings, clamp to bounds)
- [x] Path parameters extracted via gorilla/mux (no manual parsing)
- [x] SQL parameterized queries in PG stores (no string interpolation)
Status: **hardened**

### Error Response Security
- [x] Structured error responses with no stack traces or internal paths
- [x] Error catalog maps codes to safe user-facing messages
- [x] PTY errors mapped to generic "session_error" (no leaking shell output in errors)
Status: **hardened**

### WebSocket Security
- [x] Connection upgrade validates session existence before accepting
- [x] Ping/pong keepalive with configurable intervals
- [x] Read size limits on WebSocket frames
- [ ] No per-connection rate limiting (single operator, acceptable)
Status: **hardened** (for single-operator model)

### SQL Injection Prevention
- [x] All queries use parameterized statements (`$1`, `$2`, etc.)
- [x] No string concatenation in SQL queries
- [x] Schema initialization uses static SQL files (no user input)
Status: **hardened** — verified in [CODE: api/shortcut_profiles_pg.go] and [CODE: api/ai_provider_config_pg.go]

### PTY / Shell Execution
- [x] Shell path configurable via environment variable (not user input)
- [x] PTY spawned with standard terminal dimensions (clamped)
- [ ] No command injection surface — user input goes to PTY stdin, not shell exec
Status: **hardened** — PTY is the intended execution surface, not a vulnerability

## Known Vulnerabilities
None identified. The primary security boundary is the parent proxy — if bypassed, an unauthenticated user could create terminal sessions on the host.

## Priority Hardening Areas
1. **If direct exposure is ever needed**: Add JWT or session-based auth middleware
2. **Rate limiting**: Consider per-IP connection limits if multi-user support is added
3. **Audit logging**: Session creation/deletion events could be logged to PostgreSQL for forensic review
