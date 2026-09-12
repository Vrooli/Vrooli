# Progress Log

Historical execution details are preserved beneath the protected runtime home:
`plan-artifacts/docs-html-progress-20260908/scenarios/vrooli-autoheal/docs/internal/PROGRESS.md`.
The archive retains exact entries, validation receipts, and unresolved qualifications;
relocation does not resolve a finding or establish current readiness.
The table retains selected dated milestones. Other milestones and their limitations
remain in the archive; consult it before resuming historical work.

| Date | Author | Status Snapshot | Notes |
|------|--------|-----------------|-------|
| 2025-12-03 | Improver Agent | Check-Level Docs | Added individual documentation for each health check with "Learn more" button on check cards for direct navigation |
| 2025-12-04 | Claude | Auto-Heal Verification | Added post-action verification to all recovery actions - restarts now confirm success by re-running checks |
| 2026-02-18 | Codex | Seam Enforcement (Watchdog Installer) | Routed watchdog install/uninstall side effects through `detectorProbe` (commands, privileged writes, fs mutations, temp-file writes) and added seam-focused unit tests |
| 2026-02-18 | Codex | Seam Enforcement (User Config I/O) | Added `configIO` seam for `userconfig.Manager` file/home-dir interactions (load/save/schema/default-path), plus seam-focused tests for missing-config behavior, temp-file cleanup on atomic-save failure, and default-path fallback when home resolution fails |
| 2026-08-25 | Codex | Host capability coupling and notification reach | Added invariant-backed host verdicts, transition-only incident facts, live notification approval verification, one-time remediation authorization records, evidence-backed incident disposition, and remediation/delivery reach sensors. |

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
- `GET /api/v1/checks/{id}/history`: Get historical results for a check
- `GET /api/v1/docs/manifest`: Documentation navigation structure
- `GET /api/v1/docs/content`: Fetch markdown content by path

### React Dashboard (UI-HEALTH-*, UI-REFRESH-*)
- Status overview with color-coded indicators (green/amber/red)
- Summary cards showing ok/warning/critical counts
- Health check list grouped by severity
- Platform capabilities display
- Auto-refresh every 30 seconds (toggleable)
- Run Tick button for manual execution
- Enhanced check cards with:
  - Description from check metadata
  - Interval indicator (e.g., "30s", "1m")
  - Relative timestamp (e.g., "5m ago")
  - Expandable details section with clear expand/collapse indicator

### Database Schema (PERSIST-*)
- `health_results` table for check history
- `action_logs` is the sole action ledger; the never-written
  `autoheal_actions` duplicate was retired in 2026-08
- `autoheal_config` table for settings
- Cleanup function for 24-hour retention

### Events Timeline (UI-EVENTS-*)
- Aggregated timeline of all health check events
- Filter to show only issues (warnings/critical)
- Shows relative timestamps with expandable details
- Auto-refreshes every 30 seconds

### Per-Check History (PERSIST-HISTORY-*)
- Click "History" on any check card to see recent history
- Lazy-loaded to minimize API calls
- Shows status transitions and messages over time

### Uptime Statistics (PERSIST-HISTORY-*)
- 24-hour uptime percentage in sidebar
- Color-coded status (green >= 95%, amber >= 80%, red < 80%)
- Breakdown of ok/warning/critical event counts
- Progress bar visualization

### Documentation Tab
- In-app documentation browser accessible via Docs tab
- Markdown rendering with syntax highlighting
- Mermaid diagram support with dark theme styling
- Searchable sidebar navigation with collapsible sections
- 18 documentation files covering:
  - Getting Started (QUICKSTART, GLOSSARY)
  - Concepts (architecture, health-check-flow, watchdog-design)
  - Guides (dashboard, adding-checks, watchdog-installation, troubleshooting)
  - Reference (api-endpoints, cli-commands, check-catalog)
  - Health Check Details (infra-network, infra-dns, infra-docker, infra-cloudflared, infra-rdp, resource-check, scenario-check)
  - Project Status (PROGRESS, PROBLEMS, SEAMS, RESEARCH)

### Check-Level Documentation
- Individual documentation file for each health check type
- "Learn more" button on each check card navigates to relevant docs
- Deep-link support via URL hash (e.g., `#docs?path=reference/checks/infra-network.md`)
- Documentation includes:
  - What it monitors and why
  - Status meanings (OK/Warning/Critical)
  - Common failure causes with commands to diagnose
  - Step-by-step troubleshooting guide
  - Configuration options
  - Related checks and auto-heal actions

### Recovery Actions (HEAL-ACTION-*)
- All recovery actions execute and log results to database
- **Verified actions** - start/restart actions verify success by re-running health check:
  - Resources: start, restart (waits 3s, verifies running)
  - Scenarios: start, restart, restart-clean (waits 5s, verifies running)
  - Infrastructure: Docker, Cloudflared, NTP, Resolved, DNS (waits 2-5s, verifies)
  - System: Zombie reap (signals parents, verifies cleanup)
- Action results include verification status in output
- Actions logged with success/failure based on actual verification
- API endpoints for listing and executing actions per check

## Next Steps

1. **Core bootstrap logic** (04-core-bootstrap requirements)
   - Bootstrap Vrooli from cold state
   - Start required resources and scenarios

2. **OS watchdog installers** (07-os-watchdog requirements)
   - Linux systemd service generator
   - Windows scheduled task/service
   - macOS launchd plist

3. ~~**Auto-heal actions** (existing check failures should trigger restarts)~~ ✅ **COMPLETE**
   - ~~Restart failed resources~~ ✅ Implemented with verification
   - ~~Restart failed scenarios~~ ✅ Implemented with verification
   - All recovery actions now verify success by re-running health checks

4. **P1 infrastructure checks** (09-infrastructure-checks) ✅ **COMPLETE**
   - ~~Disk space, swap, zombie processes~~ ✅ All implemented
   - ~~Time synchronization~~ ✅ NTP check implemented

5. ~~**History and trends** (08-persistence)~~ ✅ **COMPLETE**
   - ~~Query historical data~~ ✅ Check history API
   - ~~Dashboard timeline view~~ ✅ Events timeline implemented

### Remaining Work

1. **Core bootstrap logic** - Bootstrap Vrooli from cold state
2. **OS watchdog installers** - systemd/launchd/Windows service generators

> **Note**: Legacy `scripts/maintenance/` was removed December 2024. This scenario is now the production autoheal implementation.

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

### RDP Service Detection (`infra/rdp_service.go`)
- `detectRDPService(ctx)` → Detects the configured RDP provider using the shared executor seam
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

## Failure Handling

### Error Response Structure (FAIL-SAFE-001)

All API errors return structured JSON responses:
```json
{
  "success": false,
  "error": "DATABASE_ERROR",
  "message": "Failed to retrieve events",
  "requestId": "123456",
  "timestamp": "2025-12-03T12:00:00Z"
}
```

**Error Codes:**
- `DATABASE_ERROR` - Database operation failed (500, retryable)
- `NOT_FOUND` - Resource not found (404, not retryable)
- `TIMEOUT` - Operation timed out (504, retryable)
- `SERVICE_UNAVAILABLE` - Dependency unavailable (503, retryable)
- `INTERNAL_ERROR` - Unexpected error (500, retryable)
- `NETWORK_ERROR` - API unreachable (UI-only, retryable)

### Graceful Degradation Patterns

1. **Tick endpoint** - Persistence failures are logged but don't fail the tick
   - Results are still returned to the client
   - `warnings` array indicates persistence issues

2. **Timeline/Uptime endpoints** - Return empty arrays on database errors
   - UI shows error state with retry button
   - Auto-retry with exponential backoff

3. **UI error recovery**
   - All data-fetching components have retry buttons
   - `APIError` class provides user-friendly messages
   - Request IDs enable log correlation for debugging

### Observability

All errors are logged with structured format:
```
[ERROR] request=123456 component=timeline code=DATABASE_ERROR message="Failed to retrieve events" cause=connection refused
[WARN] component=tick operation=save_result:infra-network error=deadline exceeded
```

## Test Infrastructure

### Test Suite Summary
- **Go Unit Tests**: 111 tests (69.6% coverage)
- **UI Tests**: 38 tests (vitest with jest-dom)
- **CLI Tests**: Go tests under `cli/` (app + loop), `[REQ:xxx]` tagged
- **Total**: 149 tests

### Test Files by Type
- `api/**/*_test.go` - Go unit tests for platform, registry, checks
- `cli/**/*_test.go` - Go tests for CLI commands and loop
- `ui/src/**/*.test.{ts,tsx}` - React component tests

### Coverage Tracking
- UI tests use `[REQ:xxx]` tags in test descriptions for auto-sync
- Go tests use `[REQ:xxx]` tags in test names/comments for tracking

### Completeness Score: 17/100
- 12/74 requirements passing (16%)
- 1/25 operational targets passing (4%)
- Primary gaps: multi-layer validation, API endpoint coverage

## Implementation Notes

- Go API uses `vrooli resource status` and `vrooli scenario status` CLI for health checks
- Platform detection correctly identifies WSL (Linux with WSL indicators)
- Health check results are persisted to PostgreSQL for history
- CLI auto-detects API port via `vrooli scenario port` command
