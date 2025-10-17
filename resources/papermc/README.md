# PaperMC Minecraft Server Resource

A Vrooli resource for deploying and managing PaperMC Minecraft servers with RCON support.

## Overview

The PaperMC resource provides automated deployment and management of Minecraft servers using the PaperMC performance-optimized server software. It enables:

- **Automated server deployment** via Docker or native Java
- **RCON remote console** for programmatic command execution
- **Integration with mcrcon** resource for advanced automation
- **World backup** and configuration management
- **Health monitoring** and lifecycle management

## Quick Start

```bash
# Install the resource
vrooli resource papermc manage install

# Start the server
vrooli resource papermc manage start --wait

# Check server status
vrooli resource papermc status

# Execute commands via RCON (requires mcrcon resource)
vrooli resource mcrcon content execute "say Hello from Vrooli!"
vrooli resource mcrcon content execute "list"

# Or use PaperMC's built-in command execution
vrooli resource papermc content execute "gamemode creative @a"
```

## Features

### P0 - Core Functionality
- ✅ v2.0 contract compliant resource structure
- ✅ Docker and native Java deployment options
- ✅ RCON enabled by default on port 25575
- ✅ Secure local-only port exposure
- ✅ Full mcrcon integration support

### P1 - Production Features
- ✅ Systemd/Docker auto-restart capability
- ✅ Automated EULA acceptance
- ✅ JVM performance tuning
- ✅ Graceful shutdown with world save
- ✅ Localhost-only firewall rules

### P2 - Advanced Features
- ✅ Server.properties management
- ✅ Plugin installation system
- ✅ Automated backup system
- ✅ Health monitoring and metrics
- ✅ Log parsing and analysis

## Configuration

### Environment Variables

```bash
# Server type (docker or native)
PAPERMC_SERVER_TYPE=docker

# Memory allocation
PAPERMC_MEMORY=2G
PAPERMC_MAX_MEMORY=4G

# RCON configuration
PAPERMC_RCON_PORT=25575
PAPERMC_RCON_PASSWORD=changeme123

# Game configuration
PAPERMC_GAMEMODE=creative
PAPERMC_MAX_PLAYERS=20
```

### Ports

- `25565` - Minecraft game server (players connect here)
- `25575` - RCON remote console (mcrcon connects here)
- `11461` - Health check endpoint (allocated in port_registry.sh)

## Commands

### Lifecycle Management

```bash
# Install server
vrooli resource papermc manage install

# Start server
vrooli resource papermc manage start [--wait]

# Stop server
vrooli resource papermc manage stop

# Restart server
vrooli resource papermc manage restart

# Uninstall (optionally keep data)
vrooli resource papermc manage uninstall [--keep-data]
```

### Content Operations

```bash
# Execute RCON command
vrooli resource papermc content execute "command"

# Backup world and config
vrooli resource papermc content backup

# List installed plugins
vrooli resource papermc content list-plugins

# Add plugin (by name or URL)
vrooli resource papermc content add-plugin essentialsx
vrooli resource papermc content add-plugin https://example.com/plugin.jar

# Remove plugin
vrooli resource papermc content remove-plugin essentialsx

# Show server health metrics
vrooli resource papermc content health

# Analyze server logs
vrooli resource papermc content analyze-logs [lines]

# Configure server
vrooli resource papermc content configure
```

### Testing

```bash
# Run smoke tests (quick health check)
vrooli resource papermc test smoke

# Run integration tests (full lifecycle)
vrooli resource papermc test integration

# Run all tests
vrooli resource papermc test all
```

## Plugin Management

The PaperMC resource includes a built-in plugin management system:

### Supported Plugins
- **EssentialsX** - Core server utilities (`essentialsx`)
- **Vault** - Economy and permissions API (`vault`)
- **WorldEdit** - World editing tools (`worldedit`) - requires direct URL
- **LuckPerms** - Advanced permissions system (`luckperms`)
- **CoreProtect** - Block logging and rollback (`coreprotect`)
- **ProtocolLib** - Packet modification API (`protocollib`)

### Installing Plugins

```bash
# Install from known repository
vrooli resource papermc content add-plugin essentialsx

# Install from direct URL
vrooli resource papermc content add-plugin https://github.com/EssentialsX/Essentials/releases/latest/download/EssentialsX.jar

# List installed plugins
vrooli resource papermc content list-plugins

# Remove a plugin
vrooli resource papermc content remove-plugin essentialsx
```

## Server Monitoring

### Health Metrics
```bash
# Get server health and performance metrics
vrooli resource papermc content health
```

This shows:
- Server status (running/stopped)
- TPS (Ticks Per Second) - should be 20 for optimal performance
- Player count and list
- Memory usage
- Docker/process statistics

### Log Analysis
```bash
# Analyze recent server logs
vrooli resource papermc content analyze-logs 100
```

This provides:
- Event summary (joins, leaves, deaths, chat)
- Issue tracking (warnings, errors)
- Performance monitoring (lag warnings)
- Plugin error detection

## Integration with mcrcon

The PaperMC resource is designed to work seamlessly with the mcrcon resource:

```bash
# First, ensure both resources are installed
vrooli resource papermc manage install
vrooli resource mcrcon manage install

# Start the Minecraft server
vrooli resource papermc manage start --wait

# Use mcrcon to control the server
vrooli resource mcrcon content execute "op YourUsername"
vrooli resource mcrcon content execute "gamemode creative @a"
vrooli resource mcrcon content execute "time set day"
vrooli resource mcrcon content execute "weather clear"
```

## Docker vs Native Installation

### Docker (Recommended)
- Easier setup and management
- Consistent environment
- Automatic dependency handling
- Better resource isolation

### Native Java
- Lower overhead
- Direct file system access
- Requires Java 17+ installed
- More control over JVM options

## Security Notes

- RCON is bound to localhost only by default
- Use strong passwords for RCON authentication
- Consider firewall rules for production deployments
- Regular backups are recommended

## Troubleshooting

### Server Won't Start
- Check Docker is installed (for Docker mode)
- Check Java 17+ is installed (for native mode)
- Verify ports 25565 and 25575 are available
- Check logs: `vrooli resource papermc logs`

### RCON Connection Failed
- Ensure server is fully started (can take 30-60 seconds)
- Verify RCON password is correct
- Check RCON is enabled in server.properties
- Test with: `nc -zv localhost 25575`

### Performance Issues
- Increase memory allocation in configuration
- Adjust JVM options for your workload
- Monitor TPS (ticks per second) via RCON
- Consider using Docker for better isolation

## Support

For issues or questions, see the [PRD.md](PRD.md) for detailed requirements and specifications.