# Progress Log

| Date | Author | Status Snapshot | Notes |
|------|--------|-----------------|-------|
| 2025-12-03 | Generator Agent | Initialization complete | Scenario scaffold + PRD seeded |
| 2025-12-03 | Generator Agent | Requirements seeded | 11 requirement modules with 60+ requirements mapped to PRD targets |
| 2025-12-03 | Generator Agent | Documentation complete | README, RESEARCH, PROBLEMS created |
| 2025-12-03 | Improver Agent | Scenario runnable | Fixed go.mod, built API & UI, scenario starts successfully with health endpoints responding |
| 2025-12-03 | Improver Agent | Core implementation | Platform detection, health registry, tick/loop/status CLI, React dashboard |
| 2025-12-03 | Improver Agent | DB + Test Infra | Fixed database schema (health_results, autoheal_actions, autoheal_config tables), installed jsdom/vitest-reporter, created selectors.manifest.json |
| 2025-12-03 | Improver Agent | Test Suite | Added 48 Go unit tests (platform detection, health registry, checks), 36 API integration tests, 10 UI tests |
| 2025-12-03 | Improver Agent | Decision Boundary Extraction | Extracted CLI status classifier, RDP/Cloudflared decision functions, UI status helpers; Added 18 new tests |
| 2025-12-03 | Improver Agent | Architecture Audit | Screaming Architecture refactor - split monolithic main.go (354→155 lines), organized checks by domain (infra/, vrooli/), added config/handlers/persistence packages |

## Completed Features

### Platform Detection (PLAT-DETECT-*)
- Detects Linux/Windows/macOS/other platforms
- Identifies capabilities: Docker, systemd, launchd, RDP, cloudflared, WSL
- Cached detection for performance

### Health Check Registry (HEALTH-REGISTRY-*)
- Extensible registry pattern for health checks
- Platform-filtered execution (checks only run on compatible platforms)
- Interval-based scheduling (checks skip if interval not elapsed)
- Stored results with timestamps

### Core Health Checks
- `infra-network`: TCP connectivity check
- `infra-dns`: DNS resolution check
- `infra-docker`: Docker daemon health
- `infra-cloudflared`: Cloudflared service status
- `infra-rdp`: xrdp/RDP service status
- `resource-postgres`: PostgreSQL resource status via vrooli CLI
- `resource-redis`: Redis resource status via vrooli CLI

### CLI Commands (CLI-TICK-*, CLI-LOOP-*, CLI-STATUS-*)
- `vrooli-autoheal tick [--force] [--json]`: Run single health cycle
- `vrooli-autoheal loop [--interval-seconds=N]`: Continuous monitoring
- `vrooli-autoheal status [--json]`: Show health summary
- `vrooli-autoheal platform`: Show detected platform capabilities
- `vrooli-autoheal checks`: List registered health checks

### API Endpoints
- `GET /health`: Lifecycle health check
- `GET /api/v1/status`: Current health summary
- `POST /api/v1/tick`: Run health check cycle
- `GET /api/v1/platform`: Platform capabilities
- `GET /api/v1/checks`: List registered checks
- `GET /api/v1/checks/{id}`: Get specific check result

### React Dashboard (UI-HEALTH-*, UI-REFRESH-*)
- Status overview with color-coded indicators (green/amber/red)
- Summary cards showing ok/warning/critical counts
- Health check list grouped by severity
- Platform capabilities display
- Auto-refresh every 30 seconds (toggleable)
- Run Tick button for manual execution

### Database Schema (PERSIST-*)
- `health_results` table for check history
- `autoheal_actions` table for auto-heal log
- `autoheal_config` table for settings
- Cleanup function for 24-hour retention

## Next Steps

1. **Core bootstrap logic** (04-core-bootstrap requirements)
   - Bootstrap Vrooli from cold state
   - Start required resources and scenarios

2. **OS watchdog installers** (07-os-watchdog requirements)
   - Linux systemd service generator
   - Windows scheduled task/service
   - macOS launchd plist

3. **Auto-heal actions** (existing check failures should trigger restarts)
   - Restart failed resources
   - Restart failed scenarios

4. **P1 infrastructure checks** (09-infrastructure-checks)
   - Disk space, swap, zombie processes
   - Time synchronization

5. **History and trends** (08-persistence)
   - Query historical data
   - Dashboard timeline view

## Architecture

The codebase follows "Screaming Architecture" - the structure clearly communicates the domain:

```
api/internal/
├── checks/           # Core domain - health check system
│   ├── infra/        # Infrastructure checks (network, DNS, Docker, etc.)
│   ├── vrooli/       # Vrooli-specific checks (resources, scenarios)
│   ├── registry.go   # Registry pattern for managing checks
│   └── types.go      # Domain types (Result, Status, Summary)
├── config/           # Configuration loading
├── handlers/         # HTTP request handlers
├── persistence/      # Database operations
└── platform/         # Platform detection (Linux/Windows/macOS)
```

**Key design decisions:**
- Checks organized by domain (infra vs vrooli) not by technical concern
- Registry pattern allows extensible health check registration
- Handlers are thin - delegate to registry for business logic
- Persistence is isolated behind Store interface

## Decision Boundaries

Key decision points are extracted into named, testable functions:

### CLI Status Classification (`vrooli/status_classifier.go`)
- `ClassifyCLIOutput(output)` → Determines if CLI output indicates healthy/stopped/unclear
- `CLIStatusToCheckStatus(cliStatus, isCritical)` → Maps CLI status to health check status
- Stopped indicators checked first (handles "not running" containing "running")

### RDP Service Selection (`infra/rdp.go`)
- `SelectRDPService(caps)` → Decides which RDP service to check based on platform
- Linux with systemd → xrdp service
- Windows → TermService
- Other → not checkable

### Cloudflared Verification (`infra/cloudflared.go`)
- `DetectCloudflaredInstall()` → Checks if binary exists
- `SelectCloudflaredVerifyMethod(caps)` → Chooses verification method (systemd or none)
- Returns warning if installed but can't verify running status

### UI Status Grouping (`lib/api.ts`)
- `groupChecksByStatus(checks)` → Groups checks by severity
- `statusToEmoji(status)` → Maps status to display emoji
- `STATUS_SEVERITY` → Defines severity ordering for sorting

## Implementation Notes

- Go API uses `vrooli resource status` and `vrooli scenario status` CLI for health checks
- Platform detection correctly identifies WSL (Linux with WSL indicators)
- Health check results are persisted to PostgreSQL for history
- CLI auto-detects API port via `vrooli scenario port` command
