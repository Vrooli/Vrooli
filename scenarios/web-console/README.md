# Web Console (web-console scenario)

Web Console provides remote-first terminal access for Vrooli scenarios. The service exposes a secure-by-design shell bridge that runs behind authenticated parents while providing one-click launches for common operator workflows (Codex, Claude, and `vrooli status`). The backend remains Go-based for ecosystem reuse and supervision, and the UI stays dependency-light so it can be embedded anywhere an iframe is allowed.

## 🎯 Purpose & Vision
The console keeps shell access, agent CLIs, and status tooling available wherever operators happen to be. Sessions default to a POSIX shell, but any executable can be targeted per launch. Shortcut buttons fire opinionated commands (`codex --yolo`, `claude --dangerously-skip-permissions`, `vrooli status --verbose`), and the UI will auto-boot a fresh shell if one is not active. Everything still respects the lifecycle system: no direct binaries, no unmanaged processes.

## 🏗️ Architecture

### Go API & Supervisor
- REST + WebSocket endpoints under `/api/v1` managing PTY-backed sessions.
- Supervisor spawns `WEB_CONSOLE_DEFAULT_COMMAND` (defaults to `/bin/bash`) with optional per-session overrides.
- Transcript writer streams newline-delimited JSON records to scenario-owned storage.
- **Session buffer**: Keeps last 500 chunks (up to 1MB) for replay on reconnection.
- Metrics endpoint surfaces session counters (`web_console_*`) alongside panic-stop and timeout data.

### Browser UI (Vanilla JS + Xterm)
- **Offline-first design**: All vendor libraries (xterm.js, lucide icons, html2canvas) served from `ui/static/lib/` - no CDN dependencies.
- Static assets served as plain files; no build tooling required.
- Quick command panel issues pre-defined commands into the active terminal and queues them while a session boots.
- **Automatic reconnection**: Detects WebSocket disconnects and reconnects to running sessions automatically.
- **Heartbeat keepalive**: Sends heartbeat every 30 seconds to maintain connection when tab is inactive.
- **Visibility detection**: Reconnects to sessions when browser tab becomes visible after being backgrounded.
- `proxyToApi` helper maintains lifecycle-compliant networking.
- `postMessage` bridge mirrors session status and responds to parent requests for transcripts, screenshots, and logs.

**Vendor Libraries** (1.1MB total, all local):
- xterm@5.3.0 - Terminal emulator
- xterm-addon-fit@0.7.0 - Terminal size fitting
- lucide icons - UI icons
- html2canvas@1.4.1 - Screenshot capture

### Embedding Model
1. Parent scenario authenticates the operator, then loads the console via iframe.
2. Operators can start a fresh shell or tap a shortcut; the UI will open a session automatically when needed.
3. Terminal output and lifecycle updates stream over WebSocket, while transcripts persist locally under the scenario storage path.
4. Parents own long-term storage, auditing, and compliance; the console never runs with public exposure.

## 🔌 Quick Commands
| Action | Command | Behavior |
|--------|---------|----------|
| Launch Codex | `codex --yolo` | Queues command inside the active shell or boots a new session first. |
| Launch Claude | `claude --dangerously-skip-permissions` | Same pattern; optimized for fast escalation to Claude. |
| Check Status | `vrooli status --verbose` | Surfaces platform diagnostics without deeper shell navigation. |

Commands queue if the shell is not ready yet and execute as soon as the PTY handshake completes.

## 🚫 Deployment Warning
- Never expose the service directly to the internet; it ships with **no authentication**.
- Always proxy through an authenticated parent scenario (Traefik/Nginx/Caddy/etc.).
- Treat missing `X-Forwarded-*` headers as deployment blockers; proxy guard enforcement rejects such requests.

## 🚀 Getting Started
```bash
cd scenarios/web-console
make setup
```
This builds the Go API binary and creates the transcript directory. The UI is already static.

### Development Lifecycle
```bash
make run     # start API + UI via lifecycle
make logs    # tail aggregated logs
make stop    # stop everything cleanly
```

## ⚙️ Configuration
Environment variables (managed in `.vrooli/service.json`):
- `WEB_CONSOLE_DEFAULT_COMMAND` — executable to launch (defaults to `/bin/bash`).
- `WEB_CONSOLE_DEFAULT_ARGS` — space-separated default arguments for the command.
- `WEB_CONSOLE_SESSION_TTL` — session lifetime (default `30m`).
- `WEB_CONSOLE_IDLE_TIMEOUT` — idle termination window (default `5m`).
- `WEB_CONSOLE_STORAGE_PATH` — transcript output directory (`data/sessions`).
- `WEB_CONSOLE_EXPECT_PROXY` — set `false` to disable proxy guard (not recommended).
- Optional adapters: `WEB_CONSOLE_REDIS_URL`, `WEB_CONSOLE_POSTGRES_URL` for external transcript sinks.

Per-session overrides can send `command` and `args` in `POST /api/v1/sessions` to target alternative binaries.

## 📡 iframe Bridge Contract
Channel: `web-console`
- **Accepted commands**: `init-session`, `end-session`, `request-transcript`, `request-screenshot`, `request-logs`.
- **Emitted events**: `bridge-initialized`, `ready`, `session-started`, `session-update`, `session-ended`, `operator-updated`, `error`.
- Messages are JSON with `type` and `payload`. Parent scenarios are responsible for validation and storage.

## 🧪 Testing
```bash
make test  # runs go vet + targeted checks
```
(Extend with integration/UI tests once a CLI stub or mock PTY is available.)

## 📡 Monitoring
- `GET /healthz` → readiness/liveness (notes proxy requirements).
- `GET /metrics` → Prometheus counters for sessions, panic stops, TTL expirations, and upgrade failures.
- Structured JSON logs integrate with existing platform collectors.

## 🛣️ Roadmap Thoughts
1. Harden multi-session queueing and resource pools (Redis/Postgres adapters, rate limits).
2. Offline-first UX: reconnect/resume, local transcript cache, better mobile latency handling.
3. Additional shortcuts configurable via service config or parent handshake.
4. Optional RBAC hooks so parent scenarios can gate which commands/operators are allowed.

## 🤝 Contributing Guidelines
- Format Go with `gofumpt`, lint with `golangci-lint` before submitting patches.
- Keep UI dependency-free (vanilla JS + CDN assets) unless there is a compelling reason otherwise.
- Update documentation when shortcut commands or iframe contract changes.
- Preserve the lifecycle guardrails: no direct binary launches outside `make`/`vrooli` wrappers.

**Remember:** this console is infrastructure for trusted scenarios only. Always run it behind an authenticated proxy and treat shortcut commands as privileged operations.
