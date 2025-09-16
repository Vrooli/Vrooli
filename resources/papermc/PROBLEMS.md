# PaperMC Resource - Known Problems

## Health Service Port Conflict

### Problem
The original health service port (11459) was conflicting with a system-level health monitor that continuously spawned `nc` processes on that port, preventing the PaperMC health service from starting.

### Solution
Changed the default health port from 11459 to 11460 in `config/defaults.sh`.

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

## Limited Plugin Repository

### Problem
The built-in plugin repository only supports a few common plugins (EssentialsX, Vault, WorldEdit). Users need to know direct download URLs for other plugins.

### Solution
Users can provide direct download URLs for any plugin:
```bash
vrooli resource papermc content add-plugin https://example.com/plugin.jar
```

### Future Enhancement
Consider integrating with Spigot/Bukkit/Paper plugin repositories API for automatic plugin discovery.

## Docker Permissions

### Problem
When running in Docker mode, file permissions between the container and host can sometimes cause issues with world saves or plugin installations.

### Solution
The itzg/minecraft-server Docker image handles most permission issues automatically by running as UID 1000. If you encounter permission issues, ensure your user has UID 1000 or adjust the Docker configuration.

## RCON Command Limitations

### Problem
Some Minecraft commands may not work properly through RCON, especially those requiring player context or interactive responses.

### Solution
Most standard server commands work fine. For advanced operations, consider using plugins that provide RCON-friendly alternatives.

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