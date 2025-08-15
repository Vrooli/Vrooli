# Vrooli Firewall Management

## Overview

This module provides automatic firewall configuration for Docker-to-host connectivity. It solves the common Linux issue where Docker containers on custom networks cannot reach services running natively on the host.

## The Problem

On Linux, Docker's iptables rules block connections from containers to the host by default. This affects services like:
- **Ollama** (runs natively for GPU access)
- **Claude Code** (CLI tool)
- Any other service you choose to run natively instead of in Docker

## The Solution

The firewall module automatically:
1. Detects which services are running natively vs in Docker
2. Creates appropriate iptables rules for native services
3. Manages rules dynamically as services start/stop
4. Cleans up rules when no longer needed

## Key Features

- **Automatic Detection**: Determines if a service runs natively or in Docker
- **Dynamic Management**: Updates rules as services change
- **Port Registry Integration**: Uses centralized port definitions
- **Safe & Idempotent**: Can be run multiple times without issues
- **Clean Removal**: Removes all rules cleanly when needed

## Usage

### Manual Usage

```bash
# Setup firewall rules for all native services
sudo ./scripts/lib/firewall/firewall.sh setup

# Check current firewall status
sudo ./scripts/lib/firewall/firewall.sh status

# Remove all Vrooli firewall rules
sudo ./scripts/lib/firewall/firewall.sh clean
```

### Automatic Integration

The firewall is automatically managed when:
- Starting resources via `vrooli resource <name> start`
- Running `vrooli develop` or `vrooli setup`
- Services change from Docker to native or vice versa

## How It Works

1. **Chain Creation**: Creates a custom `VROOLI-DOCKER` iptables chain
2. **Service Detection**: Checks each enabled service to see if it's:
   - Not running (no rule needed)
   - Running in Docker (no rule needed - uses Docker networking)
   - Running natively (rule needed for Docker access)
3. **Rule Management**: Adds/removes rules based on service state
4. **Port Mapping**: Uses the central port registry for consistency

## Security Considerations

- Rules are specific to individual ports (not blanket access)
- Only enabled services get rules
- Rules are automatically removed when services stop
- Custom chain keeps rules organized and removable

## Troubleshooting

### Permission Denied
- The firewall script requires sudo/root access
- Run with: `sudo ./scripts/lib/firewall/firewall.sh setup`

### Service Not Accessible
1. Check if service is running: `netstat -tln | grep <port>`
2. Check if firewall rule exists: `sudo ./scripts/lib/firewall/firewall.sh status`
3. Ensure service is listening on all interfaces (0.0.0.0) not just localhost

### Rules Not Persisting
- Add to system startup scripts if needed
- Or integrate with your system's firewall management (ufw, firewalld, etc.)

## Integration with Resources

Each resource can specify its network mode in its configuration:
- `network_mode: "docker"` - Runs in Docker (default)
- `network_mode: "native"` - Runs on host (needs firewall rule)

The firewall module respects these settings and configures accordingly.