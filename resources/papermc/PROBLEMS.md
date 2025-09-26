# PaperMC Resource - Known Problems

Updated: 2025-09-26

## Health Service Port Conflict

### Problem
The original health service port (11459) was conflicting with a system-level health monitor that continuously spawned `nc` processes on that port, preventing the PaperMC health service from starting.

### Solution
Changed the default health port to 11461, properly allocated in `scripts/resources/port_registry.sh`. Updated all configuration files (`config/defaults.sh`, `config/runtime.json`, `lib/core.sh`) to use the new port consistently.

### Workaround
If you encounter port conflicts, you can override the health port:
```bash
export PAPERMC_HEALTH_PORT=11461
vrooli resource papermc manage start
```

## Log Analysis Output Duplication

### Problem
When using Docker logs with `grep -c`, the output was being duplicated, causing syntax errors in bash comparisons.

### Solution
Fixed by properly capturing grep output into variables before echoing, and ensuring Docker logs are written to temp file without tee output.

## Health Service Python Dependency

### Problem
The health service requires Python 3 to be installed on the system, which may not be available in minimal environments.

### Solution
Consider implementing a simple bash-based health check using netcat as a fallback when Python is not available.

## Plugin Download URLs

### Problem
Plugin download URLs change frequently as new versions are released. GitHub release URLs need exact version numbers. Some plugins (WorldEdit) use different distribution methods like CurseForge which require authentication.

### Solution
Updated plugin URLs to use specific working versions:
- EssentialsX: v2.21.2 (working)
- Vault: v1.7.3 (working)
- CoreProtect: v21.2 (latest with release assets, v23.0 has no JAR)
- ProtocolLib: v5.3.0 (working)
- LuckPerms: Requires manual download (URLs change frequently)
- WorldEdit: Requires manual download (CurseForge authentication)

For plugins that require manual download, get the JAR file from the official site and use:
```bash
vrooli resource papermc content add-plugin https://example.com/plugin.jar
```

### Future Enhancement
Consider implementing dynamic plugin URL resolution using GitHub API or plugin repository APIs.

## Docker Permissions

### Problem
When running in Docker mode, file permissions between the container and host can sometimes cause issues with world saves or plugin installations.

### Solution
The itzg/minecraft-server Docker image handles most permission issues automatically by running as UID 1000. If you encounter permission issues, ensure your user has UID 1000 or adjust the Docker configuration.

## RCON Command Limitations

### Problem
Some Minecraft commands may not work properly through RCON, especially those requiring player context or interactive responses. The `memory` command is not recognized by PaperMC server.

### Solution
Most standard server commands work fine. For memory usage, use Docker container stats instead of RCON commands. For advanced operations, consider using plugins that provide RCON-friendly alternatives.

## Memory Configuration

### Problem
The default memory allocation (2G-4G) may be insufficient for larger servers or modded gameplay.

### Solution
Adjust memory settings via environment variables:
```bash
export PAPERMC_MEMORY=4G
export PAPERMC_MAX_MEMORY=8G
vrooli resource papermc manage start
```

## Startup Time

### Problem
PaperMC servers can take 30-60 seconds to fully start, especially on first run when generating world data.

### Solution
Use the `--wait` flag when starting to wait for the server to be ready:
```bash
vrooli resource papermc manage start --wait
```

The health check endpoint will show "starting" status during initialization.

## Docker Compose Version Warning

### Problem
Docker Compose showed a warning about the `version` attribute being obsolete in the compose file.

### Solution
Fixed by removing the `version` field from the docker-compose.yml generation in core.sh. The compose file now starts directly with `services:` which is compatible with all modern Docker Compose versions.

## Plugin List Command Partial Output

### Problem
The `./cli.sh content list-plugins` command was showing only one plugin due to an issue with arithmetic operations in bash strict mode (`set -euo pipefail`) where `((plugin_count++))` returns exit code 1 when plugin_count is 0.

### Solution
Fixed by replacing `((plugin_count++))` with `plugin_count=$((plugin_count + 1))` which correctly handles the increment without triggering error exit. The command now properly lists all installed plugins.