# Glossary

Key terminology used throughout the Autoheal documentation.

## Core Concepts

### Health Check
A single diagnostic that tests one aspect of system health. Each check has:
- **ID**: Unique identifier (e.g., `infra-network`, `resource-postgres`)
- **Description**: Human-readable explanation
- **Interval**: How often it should run (in seconds)
- **Platform Filter**: Which OS platforms it applies to

### Status
The result of a health check. One of:
- **OK** (green): System is healthy
- **Warning** (amber): Degraded but functional
- **Critical** (red): Requires immediate attention

### Tick
A single execution cycle that runs all eligible health checks. Checks are skipped if their interval hasn't elapsed since the last run.

### Registry
The central collection of all registered health checks. Manages:
- Check registration
- Platform filtering
- Interval enforcement
- Result caching

### Watchdog
An OS-level service that ensures autoheal stays running. Survives process crashes and system reboots.

## Platform Concepts

### Platform
The detected operating system: `linux`, `windows`, `macos`, or `other`.

### Platform Capabilities
Boolean flags indicating what the current system supports:
- `supportsSystemd` - Linux systemd service manager
- `supportsLaunchd` - macOS launchd service manager
- `supportsWindowsServices` - Windows Service Control Manager
- `supportsRdp` - Remote desktop (xrdp on Linux, RDP on Windows)
- `hasDocker` - Docker daemon is available
- `supportsCloudflared` - Cloudflare tunnel available
- `isWsl` - Running in Windows Subsystem for Linux
- `isHeadlessServer` - No display manager detected

## Check Categories

### Infrastructure Checks
Low-level system health:
- Network connectivity
- DNS resolution
- Docker daemon
- Remote access (RDP/xrdp)
- Cloudflared tunnels

### Resource Checks
Vrooli resource health:
- PostgreSQL database
- Redis cache
- Qdrant vector store
- Ollama AI runtime

### Scenario Checks
Vrooli scenario health:
- API responsiveness
- Critical scenario status

### System Checks
OS-level health:
- Disk space
- Swap usage
- Zombie processes
- Port exhaustion

## UI Terms

### Summary Card
Dashboard widget showing count of checks in each status category.

### Check Card
Detailed display of a single health check result with status, message, and timing.

### Events Timeline
Chronological view of recent health events, state changes, and auto-heal actions.

### Trends
Historical view of health metrics over time (24h, 7d, 30d).

## Persistence

### Health Result
A stored check result with:
- Check ID
- Status
- Message
- Timestamp
- Duration (how long the check took)

### Auto-heal Action
A logged recovery action (e.g., "restarted postgres resource").

### Retention
Health results are kept for 24 hours by default, then cleaned up.
