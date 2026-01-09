# Research Documentation

## Uniqueness Check

**Query**: `rg -l 'autoheal' scenarios/`

**Result**: This is the only scenario implementing autoheal functionality. The legacy bash scripts (`scripts/maintenance/`) were removed in December 2024 after this scenario reached production-ready status.

## Related Scenarios

### maintenance-orchestrator
- **Purpose**: Control plane for maintenance scenarios (activate/deactivate)
- **Relationship**: Complementary - autoheal monitors health, maintenance-orchestrator controls maintenance windows
- **Overlap**: None - different concerns (health vs activation)

### system-monitor
- **Purpose**: Real-time metrics, anomaly detection, AI investigation
- **Relationship**: Complementary - system-monitor provides detailed metrics and anomaly detection; autoheal focuses on health checks and auto-recovery
- **Overlap**: Minimal - both check system health, but system-monitor does metrics/anomalies while autoheal does binary health/recovery

### scenario-completeness-scoring
- **Purpose**: Score scenario quality and completeness
- **Relationship**: Unrelated - different domain (quality scoring vs infrastructure health)

## Legacy Autoheal Implementation (Removed)

> **Note**: The legacy bash scripts were removed in December 2024. This section is preserved for historical reference.

**Former Location**: `scripts/maintenance/autoheal/` (deleted)

### Previous Architecture (Bash)
```
vrooli-autoheal.sh (main orchestrator)
├── autoheal/config.sh (configuration)
├── autoheal/utils.sh (logging, locking, helpers)
├── autoheal/checks/
│   ├── infrastructure.sh (network, DNS, time)
│   ├── services.sh (cloudflared, display, docker)
│   ├── system.sh (disk, swap, zombies, ports)
│   ├── api.sh (Vrooli API health)
│   ├── resources.sh (postgres, redis, etc.)
│   └── scenarios.sh (scenario health)
└── autoheal/recovery/
    └── cleanup.sh (zombie reaping, port cleanup)
```

### Limitations of Bash Approach
1. **Linux-only**: Heavy use of systemctl, journalctl, /proc filesystem
2. **No UI**: Configuration only via environment variables
3. **Cron-based**: No daemon mode, relies on cron scheduler
4. **No persistence**: No history, no trend analysis
5. **Limited error handling**: Bash makes complex error handling difficult

### Features to Port
- [x] Network connectivity check (ping)
- [x] DNS resolution check (getent)
- [x] Time synchronization check (timedatectl)
- [x] Cloudflared service monitoring
- [x] Display manager monitoring (P2)
- [x] Docker daemon health
- [x] Disk space/inode checks
- [x] Swap usage monitoring
- [x] Zombie process detection
- [x] Port exhaustion check
- [x] Certificate expiration (P2)
- [x] Resource health checks
- [x] Scenario health checks
- [x] Lock file mechanism (prevent concurrent runs)
- [x] Configurable resources/scenarios list

### New Features (Not in Bash)
- Cross-platform support (Windows, macOS)
- OS-level watchdog installation
- Web UI dashboard
- Health history persistence
- Per-check interval scheduling
- Graceful signal handling

## External References

### systemd Service Files
- [systemd.service man page](https://www.freedesktop.org/software/systemd/man/systemd.service.html)
- Key settings: `Restart=always`, `RestartSec=10`, `WantedBy=multi-user.target`

### macOS launchd
- [launchd.plist man page](https://developer.apple.com/library/archive/documentation/MacOSX/Conceptual/BPSystemStartup/Chapters/CreatingLaunchdJobs.html)
- Key settings: `KeepAlive=true`, `RunAtLoad=true`

### Windows Task Scheduler
- [Task Scheduler API](https://docs.microsoft.com/en-us/windows/win32/taskschd/task-scheduler-start-page)
- Alternative: Windows Service via NSSM or native Go service

### Cross-Platform Detection
- Go `runtime.GOOS` returns: `linux`, `windows`, `darwin`, `freebsd`, etc.
- WSL detection: Check `/proc/version` for "Microsoft" or "WSL"
- Docker detection: Check if `docker info` succeeds

## Technology Choices

### Why Go for API/CLI?
1. **Cross-platform binaries**: Single binary, no runtime dependencies
2. **Low overhead**: Important for a watchdog that runs continuously
3. **System access**: Good stdlib support for process management, networking
4. **Consistency**: Matches other Vrooli scenarios (maintenance-orchestrator, system-monitor)

### Why React for UI?
1. **Consistency**: Matches other Vrooli scenario UIs
2. **Template**: react-vite template provides iframe-bridge, api-base integration
3. **Ecosystem**: Rich component libraries for dashboards
