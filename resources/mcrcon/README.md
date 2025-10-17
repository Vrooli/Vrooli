# mcrcon - Minecraft Remote Console Resource

A Minecraft Remote Console (RCON) client resource for Vrooli, enabling programmatic command execution, server management, and player interaction automation.

## Overview

mcrcon provides a command-line interface and library for connecting to Minecraft servers via the RCON protocol. This enables automation of server administration tasks, player management, and integration with other Vrooli resources.

## Features

### Core Features
- **RCON Protocol Support**: Full implementation of Minecraft's remote console protocol
- **Multi-Server Management**: Connect to and manage multiple Minecraft servers simultaneously
- **Server Auto-Discovery**: Automatically detect Minecraft servers on local network
- **Command Execution**: Execute any Minecraft server command programmatically
- **Health Monitoring**: Built-in health check endpoint for service monitoring
- **Error Recovery**: Automatic retry logic with exponential backoff
- **v2.0 Contract Compliant**: Follows Vrooli's universal resource contract

### Player Management
- List online players and get player information
- Teleport players to specific coordinates
- Kick or ban players with custom reasons
- Give items to players

### World Operations
- Save world data on demand
- Create world backups with timestamps
- Get comprehensive world information
- Set world properties (difficulty, gamemode, weather, time)
- Configure world spawn points

### Event Streaming & Monitoring
- Stream server events in real-time
- Monitor specific events (joins, leaves, deaths, achievements)
- Stream chat messages for specified duration
- Tail recent events from event log

### Integration Features
- **Python Library**: Full-featured Python module for scripting
- **Webhook Support**: Forward events to external services
- **Mod Integration**: Execute mod-specific commands and list installed mods/plugins

## Quick Start

### Installation

```bash
# Install mcrcon and its dependencies
vrooli resource mcrcon manage install

# Start the health monitoring service
vrooli resource mcrcon manage start --wait

# Verify installation
vrooli resource mcrcon status
```

### Quick Start with PaperMC

```bash
# If you have PaperMC installed, use the quick-start command
vrooli resource mcrcon integration quick-start

# Or auto-configure for all detected servers
vrooli resource mcrcon integration auto-configure
```

### Configuration

Set your Minecraft server's RCON credentials:

```bash
export MCRCON_HOST="localhost"
export MCRCON_PORT="25575"
export MCRCON_PASSWORD="your_rcon_password"
```

### Basic Usage

```bash
# Execute a Minecraft command
vrooli resource mcrcon content execute "list"

# Say something in the game
vrooli resource mcrcon content execute "say Hello from Vrooli!"

# Give a player items
vrooli resource mcrcon content execute "give PlayerName minecraft:diamond 64"

# Teleport a player
vrooli resource mcrcon content execute "tp PlayerName 0 100 0"
```

### Server Management

```bash
# Auto-discover Minecraft servers on local network
vrooli resource mcrcon content discover

# Test connection to a server
vrooli resource mcrcon content test localhost 25575

# List configured servers
vrooli resource mcrcon content list

# Add a new server configuration
vrooli resource mcrcon content add "survival" "mc.example.com" "25575" "password" "Survival Server"

# Remove a server
vrooli resource mcrcon content remove "survival"

# Execute command on all configured servers
vrooli resource mcrcon content execute-all "say Server maintenance in 5 minutes"
```

### Player Management

```bash
# List online players
vrooli resource mcrcon player list

# Get player information
vrooli resource mcrcon player info PlayerName

# Teleport player to coordinates
vrooli resource mcrcon player teleport PlayerName 100 64 200

# Give items to a player
vrooli resource mcrcon player give PlayerName minecraft:diamond 64

# Kick a player
vrooli resource mcrcon player kick PlayerName "Reason for kick"

# Ban a player
vrooli resource mcrcon player ban PlayerName "Reason for ban"
```

## Testing

```bash
# Quick health check (<30s)
vrooli resource mcrcon test smoke

# Full test suite
vrooli resource mcrcon test all
```

## CLI Commands

| Command | Description |
|---------|-------------|
| `help` | Show comprehensive help |
| `info` | Display runtime configuration |
| `manage install` | Install mcrcon binary |
| `manage start` | Start health service |
| `manage stop` | Stop health service |
| `test smoke` | Quick validation tests |
| `test all` | Run complete test suite |
| `content execute` | Execute RCON command |
| `content execute-all` | Execute on all servers |
| `content list` | List configured servers |
| `content discover` | Auto-discover servers |
| `content test` | Test server connection |
| `player list` | List online players |
| `player info` | Get player information |
| `player teleport` | Teleport player |
| `player kick` | Kick player from server |
| `player ban` | Ban player from server |
| `player give` | Give items to player |
| `status` | Show resource status |
| `credentials` | Display connection info |
| `integration auto-configure` | Auto-detect and configure servers |
| `integration quick-start` | Quick setup with PaperMC |
| `integration detect-papermc` | Detect PaperMC installation |
| `integration check-papermc` | Check PaperMC server status |

## Configuration Files

- `config/defaults.sh` - Default environment variables
- `config/runtime.json` - Runtime configuration and dependencies
- `config/schema.json` - Configuration schema definition
- `~/.mcrcon/servers.json` - Server configurations (created at runtime)

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `MCRCON_HOST` | localhost | Minecraft server hostname |
| `MCRCON_PORT` | 25575 | RCON port |
| `MCRCON_PASSWORD` | (none) | RCON password |
| `MCRCON_TIMEOUT` | 30 | Command timeout in seconds |
| `MCRCON_RETRY_ATTEMPTS` | 3 | Retry attempts for failed commands |
| `MCRCON_DATA_DIR` | ~/.mcrcon | Data directory location |
| `MCRCON_HEALTH_PORT` | 8025 | Health check service port |

## Requirements

- **Minecraft Server**: Must have RCON enabled in server.properties
- **Network Access**: Connectivity to Minecraft server's RCON port
- **Dependencies**: wget, tar, python3, jq, curl

## Minecraft Server Setup

Enable RCON in your Minecraft server's `server.properties`:

```properties
enable-rcon=true
rcon.port=25575
rcon.password=your_secure_password
```

## Security Notes

- Store RCON passwords securely using environment variables
- Never commit passwords to version control
- Use strong, unique passwords for each server
- Consider network security when exposing RCON ports

## Troubleshooting

### Connection Issues
- Verify RCON is enabled on the Minecraft server
- Check firewall rules for port 25575
- Ensure RCON password is correctly set

### Command Failures
- Check server console for error messages
- Verify command syntax matches Minecraft version
- Ensure sufficient permissions for commands

## External References

- [RCON Protocol Specification](https://wiki.vg/RCON)
- [Minecraft Commands Reference](https://minecraft.wiki/w/Commands)
- [Original mcrcon Implementation](https://github.com/Tiiffi/mcrcon)

## License

This resource integrates the mcrcon binary, which is licensed under the zlib License.