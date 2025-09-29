# Codex Console

Secure remote telepresence for the Codex CLI designed for iframe embedding inside authenticated Vrooli scenarios. The backend and supervisor are implemented entirely in Go to maximize reuse across the ecosystem. **This scenario deliberately ships without authentication. Never expose it directly to the public internet. Always front it with an authenticated parent scenario and secure proxy.**

## 🎯 Purpose & Vision
Codex Console keeps Codex conversations and terminal interactions available to operators on any device while preserving the guardrails of Vrooli’s lifecycle system. A Go API manages PTY-backed Codex sessions and a lightweight vanilla JavaScript UI streams the terminal into any authenticated scenario via iframe.

## 🏗️ Architecture

### Go API & Supervisor
- REST + WebSocket endpoints under `/api/v1` implemented in Go.
- Session supervisor launches the Codex CLI inside a PTY, enforcing TTL, idle timeout, and panic-stop controls.
- Transcript writer stores newline-delimited JSON snapshots under scenario storage paths.
- `/metrics` endpoint exposes Prometheus-compatible statistics.

### Browser UI (Vanilla JS + Xterm)
- Static assets served from `ui/static`; no bundler or build step required to boot the scenario.
- `proxyToApi` helper ensures all network calls obey lifecycle proxies.
- xterm.js (loaded from CDN) renders the Codex stream, while browser-native code manages transcripts, event feeds, and inputs.
- `postMessage` bridge lets parent scenarios request screenshots, transcripts, or logs and receive live status updates.

### Embedding Model
1. Parent scenario authenticates the operator and loads Codex Console inside an iframe that passes through the secure proxy.
2. Parent sends control tokens/config via `postMessage` to initialize the session.
3. Codex Console streams terminal output, exposes logs/transcripts, and signals lifecycle events back to the parent.
4. Parent scenario owns storage, auditing, and compliance responsibilities; Codex Console remains an internal-only capability.

## 🔧 Key Features
- Go-based session lifecycle APIs (`POST /api/v1/sessions`, `DELETE /api/v1/sessions/{id}`) with bi-directional WebSocket streaming.
- iframe bridge events (`session-started`, `session-update`, `session-ended`, `operator-updated`, `error`) and commands (`init-session`, `end-session`, `request-transcript`, `request-screenshot`, `request-logs`).
- Screenshot capture for audit trails using `html2canvas` and panic-stop controls to kill runaway Codex processes.
- Transcript export as newline-delimited JSON plus event feed for quick diagnostics.
- Health (`/healthz`) and metrics endpoints for lifecycle integration.

## 🚫 Deployment Warning
- Do **not** assign public DNS or expose ports directly.
- Always run Codex Console behind an authenticated scenario and secure proxy (Traefik, Nginx, Caddy, etc.).
- Runtime guardrails log warnings when expected proxy headers are missing—treat those as deployment blockers.

## 🚀 Getting Started

### Prerequisites
- Go 1.21+
- Node 18+ (only required to run the static asset server)
- Codex CLI installed on host (e.g., `/usr/local/bin/codex`)
- `make` and standard Vrooli toolchain

### Setup
```bash
cd scenarios/codex-console
make setup
```
This builds the Go API binary and prepares the transcript directory. No UI build step is required.

### Development
```bash
make run     # starts Go API and static UI server through the lifecycle
make logs    # tails aggregated logs
make stop    # stops all services
```

### Configuration
Environment variables managed via `.vrooli/service.json`:
- `CODEX_CONSOLE_CLI_PATH=/usr/local/bin/codex`
- `CODEX_CONSOLE_SESSION_TTL=30m`
- `CODEX_CONSOLE_IDLE_TIMEOUT=5m`
- `CODEX_CONSOLE_STORAGE_PATH=/var/lib/codex-console`
- `CODEX_CONSOLE_UPSTREAM_PROXY_EXPECTED=true`
- Optional: `CODEX_CONSOLE_REDIS_URL`, `CODEX_CONSOLE_POSTGRES_URL`

## 🔌 iframe Bridge Contract
Codex Console communicates on the `codex-console` channel:
- **Commands Accepted**: `init-session`, `end-session`, `request-transcript`, `request-screenshot`, `request-logs`.
- **Events Emitted**: `bridge-initialized`, `session-started`, `session-update`, `session-ended`, `operator-updated`, `error`.
- Payloads are JSON with `type` + `payload`. Parents must validate responses, handle storage, and manage compliance.

Sample parent integration (pseudo-code):
```ts
iframeEl.contentWindow?.postMessage({
  channel: 'codex-console',
  type: 'init-session',
  payload: { operator: currentUser }
}, iframeOrigin);

window.addEventListener('message', (event) => {
  if (event.origin !== iframeOrigin) return;
  const message = event.data;
  if (message?.channel !== 'codex-console') return;
  if (message.type === 'screenshot') {
    persistImage(message.payload.image);
  }
});
```

## 🧪 Testing
```bash
make test   # runs go vet via lifecycle
```
(Additional integration/UI tests can be layered on once a stable mock Codex binary is available.)

## 📡 Monitoring
- `GET /healthz` for readiness/liveness probes.
- `GET /metrics` exports session counts, latency histograms, and panic events.
- JSON logs emitted for ingestion by parent scenario logging stacks.

## 📂 Suggested Structure
```
scenarios/codex-console/
├── api/                # Go REST/WebSocket service
├── ui/                 # Static UI assets + proxy server
├── PRD.md
├── README.md
└── Makefile
```

## 🧭 Roadmap
1. MVP: single-session flow, iframe bridge, proxy guardrails, transcripts ✅
2. Hardening: multi-session queue, metrics dashboard, panic-stop automation, parent alert hooks
3. Mobility: offline transcript cache, reconnect/resume, haptics
4. Assistive features: voice bridge, push notifications, advanced analytics

## 🤝 Contributing
- Format Go code with `gofumpt` and run `go vet` before submitting changes.
- Update the iframe bridge documentation when message contracts change.
- Record maintenance updates using `// AI_CHECK` comments as needed.
- Coordinate security reviews before widening deployment; the scenario must remain proxy-only.

**Remember:** Codex Console is an internal capability. Route all traffic through an authenticated parent scenario and secure proxy—never operate it as a standalone public endpoint.
