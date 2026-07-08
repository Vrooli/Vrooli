# System Monitor

## Purpose
Real-time server monitoring with threshold-based anomaly detection, AI-driven investigation via agent-manager, and automated reporting. Features a Matrix-themed cyberpunk dashboard.

## Features
- **Real-time Metrics**: CPU, memory, disk, network, GPU, process monitoring via 6 pluggable collectors
- **Threshold-based Anomaly Detection**: Configurable warning/critical thresholds with 5 auto-fix triggers
- **AI Investigation**: Automated investigation via agent-manager integration (spawns AI agents)
- **Investigation Scripts**: 30 ready-to-use investigation scripts (CPU, memory, network, container, process analysis)
- **Process Management**: Zombie detection, high-thread monitoring, memory leak candidates, process kill
- **Infrastructure Monitoring**: Database pools, HTTP pools, message queues, storage I/O
- **Automated Reports**: Daily/weekly report generation with executive summaries and trend analysis
- **Dark Cyberpunk UI**: Matrix-themed React dashboard with animated grid backgrounds, neon green styling

## Architecture

```
┌──────────────────────────────────────────────────────────────────┐
│  UI (React + Vite)          │  CLI (Bash)                       │
│  Port: 36232 (lifecycle)    │  system-monitor / vrooli-system-  │
│  HTTP polling (5s/60s/4s)   │  monitor (Bash CLI)               │
└──────────┬──────────────────┴──────────┬────────────────────────┘
           │                             │
           ▼                             ▼
┌──────────────────────────────────────────────────────────────────┐
│  API (Go)   Port: dynamic (lifecycle)                            │
│  Handlers → Services → Repository (in-memory / PostgreSQL)       │
│  6 Metric Collectors │ Agent-Manager Integration                 │
│  Settings Manager                                            │
└──────┬──────┬────────┬──────┬────────┬──────────────────────────┘
       │      │        │      │        │
       ▼      ▼        ▼      ▼        ▼
   Postgres  QuestDB  Redis  Ollama  agent-manager
```

## Dependencies

### Resources (active runtime)
| Resource | Purpose | Config |
|----------|---------|--------|
| PostgreSQL | Metrics, thresholds, investigations, reports, system health (6 tables) | `initialization/postgres/schema.sql` |
| QuestDB | Time-series metrics storage (configured; API defaults to in-memory) | `initialization/questdb/server.conf` |
| Redis | Real-time alerts queue, metrics caching, session data | `initialization/redis/redis.conf` |
| Ollama | AI analysis model (llama3.2:3b) | Pulled during setup |

### Scenario Dependencies
| Scenario | Purpose |
|----------|---------|
| agent-manager | Orchestrates AI-driven investigations |

### Historical Prototypes
- `initialization/node-red/` contains speculative Node-RED flow prototypes from earlier planning. They are not wired into the current Go API or scenario lifecycle and should be treated as blueprint material, not active runtime dependencies.

## Components

### API (Go)
REST API with 40+ endpoints across these groups:
- **Health**: `/health`, `/api/v1/health`
- **Metrics**: current, detailed, processes, infrastructure
- **Investigations**: CRUD, agent spawn/stop/status, cooldown, triggers, scripts
- **Reports**: generate (daily/weekly), list, get by ID
- **Settings**: get/update/reset
- **Maintenance**: state get/set
- **Agent Config**: config, runners, status

### UI (React + Vite + TypeScript)
Matrix-themed dashboard with 7 routes:
| Route | View |
|-------|------|
| `/` | Main dashboard with all monitoring panels |
| `/metrics/cpu` | CPU detail view |
| `/metrics/memory` | Memory detail view |
| `/metrics/network` | Network detail view |
| `/metrics/disk` | Disk detail view |
| `/metrics/gpu` | GPU detail view |
| `/scripts` | Investigation script browser and executor |

Key features: MetricsGrid (5-column), sparkline charts (recharts), process monitor with kill dialog, infrastructure monitor, investigation agent management, script editor/executor, report generation panel.

Styling: "Share Tech Mono" font, `#5cff95` primary green, `#020b07` background, animated grid, glow effects, backdrop blur.

### CLI (Bash, v2.0.0)
Entry points: `cli/system-monitor`, `cli/vrooli-system-monitor`

| Command | Description | API Endpoint |
|---------|-------------|-------------|
| `health` | Check API health | `GET /health` |
| `metrics` | Current CPU/memory/TCP metrics | `GET /api/v1/metrics/current` |
| `status` | System status (HEALTHY/WARNING/CRITICAL/OFFLINE) | `GET /api/v1/metrics/current` |
| `alerts` | List active alerts based on thresholds | `GET /api/v1/metrics/current` |
| `investigate` | Fetch latest investigation results | `GET /api/v1/investigations/latest` |
| `report <type>` | Generate daily/weekly report | `POST /api/v1/reports/generate` (CLI bug: calls `/api/reports/generate`) |
| `watch` | Live monitoring with ASCII progress bars (2s refresh) | `GET /api/v1/metrics/current` |
| `dashboard` | Open UI in browser | None (xdg-open) |
| `simulate` | Simulate CPU anomaly (test endpoint not implemented) | `GET /api/test/anomaly/cpu` |
| `version` | Show CLI version | None |

Global flags: `--help`, `--version`, `--port <port>`, `--json`, `--quiet`

## Investigations

30 investigation scripts in `investigations/active/` covering:
- **CPU**: high-cpu-analysis, go-busy-loop-detector, cpu-usage-comprehensive
- **Memory**: memory-leak-detector, swap-memory-pressure-analyzer, swap-heavy-processes
- **Process**: enhanced-zombie-analyzer, container-zombie-analyzer, process-genealogy
- **Container**: container-resource-optimizer, container-health-comprehensive, docker-health-analyzer
- **Network**: network-anomaly-detector, service-health-monitor
- **System**: comprehensive-system-analyzer, resource-exhaustion-detector, master-system-sweep
- **Service-specific**: judge0-cpu-investigation, judge0-memory-analyzer, chrome-cpu-analyzer

Auto-fix triggers (configurable in `initialization/configuration/investigation-triggers.json`):
1. High CPU Usage (75%, 60s sustained)
2. Memory Pressure (10% available, 30s sustained)
3. Low Disk Space (90%, 120s sustained)
4. Excessive Network Connections (2000, 30s sustained)
5. Process Anomaly (25 processes, 10s sustained)

## Usage
```bash
# Start (preferred)
cd scenarios/system-monitor && make start

# Alternative
vrooli scenario start system-monitor

# CLI
system-monitor health
system-monitor metrics --json
system-monitor status
system-monitor alerts
system-monitor investigate
system-monitor report daily
system-monitor watch
system-monitor dashboard

# Other Makefile targets
make test      # Run tests via test-genie
make logs      # View logs
make stop      # Stop scenario
make build     # Build API + UI
make fmt       # Format Go code
make lint      # Lint Go code
make check     # Full quality gates (fmt + lint + test)
```

## Configuration

### Environment Variables (`.env`)
| Variable | Default | Description |
|----------|---------|-------------|
| `API_PORT` | 8080 | API server port |
| `UI_PORT` | 3003 | UI dashboard port |
| `DATABASE_URL` | postgres://vrooli@localhost:5433/... | PostgreSQL connection |
| `QUESTDB_URL` | http://localhost:9009 | QuestDB HTTP endpoint |
| `REDIS_URL` | redis://localhost:6380 | Redis connection |
| `ENABLE_CLAUDE_INVESTIGATIONS` | true | Enable AI investigations |
| `CPU_WARNING_THRESHOLD` | 70 | CPU warning % |
| `CPU_CRITICAL_THRESHOLD` | 90 | CPU critical % |
| `MEMORY_WARNING_THRESHOLD` | 80 | Memory warning % |
| `MEMORY_CRITICAL_THRESHOLD` | 95 | Memory critical % |

See `.env` for full configuration including logging, health checks, and agent-manager settings.

## Integration Points
Other scenarios can leverage system-monitor for:
- Real-time system metrics (via API endpoints)
- Anomaly detection and investigation (via investigation triggers)
- System health checks (via `/health` endpoint)
- Process management (via process kill endpoint)
- Infrastructure monitoring (database pools, queues, storage)

## Known Limitations
- **No WebSocket**: UI uses HTTP polling (5s current+detailed metrics, 60s process/infrastructure/investigations, 4s agent status when active), not real-time streaming
- **No Authentication**: API endpoints have no auth middleware
- **CLI JSON Parsing**: Uses regex (grep/cut) instead of jq; fragile
- **CLI report bug**: Calls `/api/reports/generate` (missing `/v1/` prefix) — will 404
- **Storage Default**: API defaults to in-memory; PostgreSQL/QuestDB configured but fallback
- **Missing API endpoint**: UI references `POST /processes/{pid}/kill`, but no process-kill route exists in the API router — process kill silently fails
- **Disk remediation boundary**: Disk detail is read-only; broad cleanup routes through cleanup-manager preview/apply rather than system-monitor deletion paths
- **Script API placeholders**: Script list/get/execute endpoints return empty/404; scripts run via investigation agent, not API
- **Test Coverage**: test/ directory is empty; tests defined via test-genie but not populated
- **simulate command**: References test endpoint that doesn't exist in API
- **CLI --quiet flag**: Parsed but never checked in code (no effect)
