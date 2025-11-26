# Vrooli Autoheal - Modular Architecture

This directory contains the modular components of the Vrooli autoheal system.

## Structure

```
autoheal/
├── config.sh                # Configuration and environment variables
├── utils.sh                 # Logging, locking, and helper utilities
├── checks/                  # Health check modules
│   ├── infrastructure.sh    # Network, DNS, time sync
│   ├── services.sh          # Cloudflared, GDM, Docker, systemd-resolved
│   ├── system.sh            # Disk, swap, zombies, ports, certificates
│   ├── api.sh               # Vrooli API health
│   ├── resources.sh         # Resource monitoring (postgres, redis, etc.)
│   └── scenarios.sh         # Scenario/application monitoring
└── recovery/                # Recovery and cleanup functions
    └── cleanup.sh           # Zombie reaping and port cleanup
```

## Design Principles

1. **Separation of Concerns**: Each module handles one category of checks
2. **Single Responsibility**: Files are focused (~100-200 lines each)
3. **Testability**: Individual modules can be sourced and tested independently
4. **Maintainability**: Easy to find, understand, and modify specific functionality
5. **Extensibility**: New checks can be added without touching existing code

## Module Descriptions

### config.sh
- All environment variable defaults
- Feature flags and thresholds
- No logic, just configuration

### utils.sh
- Logging functions (`log`, `log_stream`, `log_section`)
- Lock file management (`acquire_lock`, `cleanup_lock`)
- Helper utilities (`parse_list`, `run_with_timeout_capture`)
- Prerequisites checking (`ensure_prereqs`)

### checks/infrastructure.sh
- Network connectivity (ping test)
- DNS resolution (with auto-recovery)
- Time synchronization (NTP)

### checks/services.sh
- Cloudflare tunnel monitoring (service status + end-to-end testing)
- Display manager monitoring (GDM/lightdm/sddm)
- systemd-resolved health
- Docker daemon health

### checks/system.sh
- Disk space usage
- Inode exhaustion detection
- Swap usage monitoring
- Zombie process detection
- Port exhaustion monitoring
- Certificate expiration warnings

### checks/api.sh
- Vrooli orchestration API health check
- Optional recovery command execution

### checks/resources.sh
- Resource health monitoring (postgres, redis, qdrant, ollama, etc.)
- Automatic restart on failure

### checks/scenarios.sh
- Scenario/application monitoring
- Port cleanup before restart
- Detailed logging on failure

### recovery/cleanup.sh
- Zombie process cleanup (SIGCHLD to parents)
- Port listener force-kill
- Scenario port cleanup

## Adding New Checks

To add a new check:

1. **Choose the right module** based on the check category
2. **Add configuration** to `config.sh`
3. **Implement the check function** in the appropriate module
4. **Call the function** from the main orchestrator (`../vrooli-autoheal.sh`)

Example:

```bash
# In config.sh
CHECK_MY_SERVICE="${VROOLI_AUTOHEAL_CHECK_MY_SERVICE:-true}"

# In checks/services.sh
check_my_service() {
    [[ "$CHECK_MY_SERVICE" != "true" ]] && return 0

    log "DEBUG" "Checking my service"

    if systemctl is-active my-service >/dev/null 2>&1; then
        log "DEBUG" "My service is healthy"
        return 0
    else
        log "ERROR" "My service is not running"
        restart_my_service
        return $?
    fi
}

# In ../vrooli-autoheal.sh (main orchestrator)
# Add to Phase 2:
check_my_service || true
```

## Testing Individual Modules

You can test individual modules by sourcing them:

```bash
# Source dependencies
source autoheal/config.sh
source autoheal/utils.sh

# Source the module you want to test
source autoheal/checks/infrastructure.sh

# Call specific functions
check_network_connectivity
check_dns_resolution
```

## Benefits of Modular Design

- **Easier onboarding**: New developers can understand one module at a time
- **Parallel development**: Multiple people can work on different modules
- **Better testing**: Can test infrastructure checks without running scenario checks
- **Reduced merge conflicts**: Changes isolated to specific modules
- **Clearer git history**: Commits naturally scoped to specific functionality
- **Professional structure**: Follows Unix philosophy and best practices

## Migration from Monolithic

The previous monolithic script (`vrooli-autoheal-enhanced.sh`, ~1000 lines) has been refactored into this modular architecture with:
- Main orchestrator: ~100 lines
- Individual modules: ~100-200 lines each
- Total same functionality, much better organization

All functionality preserved, zero breaking changes.
