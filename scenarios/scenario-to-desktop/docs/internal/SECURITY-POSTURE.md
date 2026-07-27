# Security Posture

## Last Updated
2026-07-26

## Deployment Boundary

scenario-to-desktop is a trusted-local-host deployment tool. Its build,
signing, task, and live-desktop APIs intentionally do not implement user or
network authentication because the API is operated by the local Vrooli control
plane. This does not authorize LAN or public exposure. A future LAN/public
posture requires an explicit authentication and authorization design.

The live-desktop backend is more restrictive than the API: x11vnc and
websockify bind only to `127.0.0.1`, and browser access is mediated by the API
VNC proxy. This prevents an unauthenticated VNC listener from being reachable
on the host network.

## Defense Layers

### HTTP Security

| Mechanism | Location | Status |
|-----------|----------|--------|
| CORS middleware | [CODE: api/shared/http/middleware.go] | Active — restricts to localhost origins by default |
| Baseline security headers | `SecurityHeaders` middleware | Active — nosniff, DENY framing, XSS protection disabled, and HSTS stamped on REST and Connect responses |
| Request ID propagation | `RequestIDMiddleware` | Active — `X-Request-ID` header on all responses |
| Structured logging | `StructuredLoggingMiddleware` | Active — all requests logged with slog |
| Recovery middleware | `RecoveryMiddleware` | Active — panic recovery prevents crash |

**CORS Configuration**: Origins derived from `ALLOWED_ORIGIN` env var or computed from `UI_PORT`. Falls back to `http://localhost:{UI_PORT}` + `http://127.0.0.1:{UI_PORT}` if misconfigured. Intentionally permissive for tool protocol handlers (`HTTP-002` CORS wildcards documented as acceptable).

### Electron Security (Templates)

| Setting | Value | Purpose |
|---------|-------|---------|
| `contextIsolation` | `true` | Isolates preload from renderer |
| `nodeIntegration` | `false` | Prevents renderer access to Node.js APIs |
| `sandbox` | default (enabled) | OS-level process sandboxing |
| Preload bridge | [CODE: templates/vanilla/preload.ts] | Controlled API surface via `contextBridge` |

**Exception**: Secret prompt modals temporarily use `nodeIntegration: true` for ephemeral data URI windows that are destroyed immediately after credential collection.

### Path Traversal Prevention

- Build cleanup operations validate paths contain `platforms/{framework}` before deletion
- Template path resolution uses `filepath.Clean()` and validates against scenario root
- **Audit finding AUTH-002**: False positive — `signing/generation/electron_builder.go:24` is a template string for env var reference, not a hardcoded credential

### Code Signing

- Signing infrastructure in [CODE: api/signing/config.go] supports certificate generation and platform-specific signing workflows
- Certificates are runtime-generated, not bundled
- P1 roadmap: Automated notarization for macOS and Windows

### Scanner-reviewed credential boundaries

The default `APPLE_ID_PASSWORD` string in the notarization generator names an
environment variable; it is never a credential value. Vault recovery material
is loaded from platform secure storage and is neither logged nor emitted to the
UI. The small network adapter deliberately centralizes runtime dialing so
callers can be tested and destination policy stays visible at the boundary.
Security scanner annotations at these locations are narrow explanations of
those facts, rather than rule-wide exclusions.

## Known Gaps

| Gap | Severity | Mitigation | Tracking |
|-----|----------|------------|----------|
| No rate limiting on API endpoints | Low | Trusted-local-host API; do not expose to LAN/public networks | P2 |
| CORS wildcards on tool protocol handlers | Low | Intentional for tool execution protocol | Documented as acceptable |
| No CSP headers on API responses | Low | API serves JSON only, no HTML rendering | — |
| Electron auto-updater signature verification | Medium | electron-updater supports signatures; needs certificate setup | OT-P1-001 |
