# PaperMC Minecraft Server Resource PRD

## Executive Summary
**What**: PaperMC Minecraft server resource enabling automated server deployment and management
**Why**: Enable mcrcon resource to have servers to control, facilitate educational scenarios, virtual workspaces, and creative environments
**Who**: Educational institutions, creative teams, developers building Minecraft-based applications
**Value**: Enables $40K+ in server hosting, educational platforms, and creative workspace scenarios
**Priority**: P2 - Gaming/Creative infrastructure

## Memory Search Results
1. **Exact matches found**: No existing PaperMC implementations
2. **Similar implementations**: mcrcon client for controlling servers, Godot for game development
3. **Reusable components**: Standard v2.0 resource patterns, Docker deployment patterns from browserless
4. **Known failures**: None found for Minecraft servers
5. **Best templates**: v2.0 universal contract, browserless Docker patterns

## Research Findings
- **Similar Work**: mcrcon (client only), Godot (different game engine)
- **Template Selected**: v2.0 universal resource contract, browserless Docker patterns
- **Unique Value**: First Minecraft server hosting capability, completes the mcrcon ecosystem
- **External References**: 
  - https://papermc.io/software/paper (Official PaperMC)
  - https://docker-minecraft-server.readthedocs.io/ (Docker image docs)
  - https://github.com/itzg/docker-minecraft-server (Popular Docker image)
  - https://minecraft.wiki/w/Server.properties (Server configuration)
  - https://minecraft.wiki/w/Commands (Server commands reference)
- **Security Notes**: RCON password management, port exposure control, resource limits

## P0 Requirements (Must Have)
- [x] **v2.0 Contract Compliance**: Full implementation of universal.yaml requirements
- [x] **Server Installation**: Install PaperMC or use Docker image itzg/minecraft-server
- [x] **RCON Configuration**: Enable RCON with secure password on port 25575
- [x] **Local Port Exposure**: Expose RCON port 25575 locally only (not to internet)
- [x] **mcrcon Connectivity**: Verify mcrcon can connect and execute commands (say, list)

## P1 Requirements (Should Have)
- [x] **Systemd/Docker Management**: Auto-restart capability via systemd or Docker
- [x] **EULA Acceptance**: Automated EULA acceptance during setup
- [x] **JVM Configuration**: Performance tuning with configurable memory (2G-4G default)
- [x] **Graceful Shutdown**: Save-all and stop commands via RCON before termination
- [x] **Firewall Rules**: Restrict RCON access to localhost only

## P2 Requirements (Nice to Have)
- [x] **Configuration Management**: Expose server.properties for AI editing
- [x] **Plugin Management**: Automated Paper plugin download/update system
- [x] **Backup System**: World and config backups before upgrades
- [x] **Health Monitoring**: TPS reports and player count monitoring
- [x] **Log Analysis**: Parse server logs for metrics and events

## Technical Specifications

### Architecture
```
┌─────────────────────────────────────────────────┐
│                PaperMC Resource                  │
├─────────────────────────────────────────────────┤
│  CLI Interface (./cli.sh)                       │
│  ├─ manage (install/start/stop)                 │
│  ├─ test (smoke/integration/unit)               │
│  ├─ content (execute/backup/configure)          │
│  └─ status/logs/help                            │
├─────────────────────────────────────────────────┤
│  Core Libraries (lib/)                          │
│  ├─ core.sh - Server lifecycle management       │
│  ├─ test.sh - Test implementations              │
│  ├─ rcon.sh - RCON connectivity                 │
│  └─ config.sh - Configuration management        │
├─────────────────────────────────────────────────┤
│  Docker Support (docker/)                       │
│  ├─ docker-compose.yml - Container config       │
│  └─ Dockerfile - Custom image (if needed)       │
├─────────────────────────────────────────────────┤
│  Configuration (config/)                        │
│  ├─ defaults.sh - Default settings              │
│  ├─ runtime.json - Dependencies & startup       │
│  └─ schema.json - Configuration schema          │
└─────────────────────────────────────────────────┘
```

### Dependencies
- Docker or Java 17+ for native installation
- mcrcon resource for command execution
- curl for health checks
- jq for JSON processing

### Port Allocation
- `25565` - Minecraft server port (game connections)
- `25575` - RCON port (remote console)
- `11459` - Health check endpoint

### Configuration Schema
```json
{
  "server": {
    "type": "docker|native",
    "version": "latest",
    "memory": "2G",
    "max_memory": "4G"
  },
  "rcon": {
    "enabled": true,
    "port": 25575,
    "password": "${PAPERMC_RCON_PASSWORD}"
  },
  "eula": {
    "accept": true
  },
  "world": {
    "name": "world",
    "seed": "",
    "gamemode": "creative"
  }
}
```

## Success Metrics

### Completion Targets
- P0: 100% (5/5 requirements) ✅
- P1: 100% (5/5 requirements) ✅
- P2: 100% (5/5 requirements) ✅
- Overall: 100% weighted completion

### Quality Metrics
- Server startup time <30 seconds
- RCON response time <100ms
- Memory usage within configured limits
- Zero crashes during normal operation

### Performance Targets
- Support 20+ concurrent players
- TPS (ticks per second) maintained at 20
- World save time <5 seconds
- Plugin load time <10 seconds total

## Implementation Status

### Progress History
- 2025-01-15: Initial PRD creation and implementation by generator (80%)
  - ✅ v2.0 contract compliant structure created
  - ✅ Docker and native installation support implemented
  - ✅ RCON configuration with secure local-only access
  - ✅ Full lifecycle management (install/start/stop/restart/uninstall)
  - ✅ Health monitoring service with HTTP endpoint
  - ✅ Integration with mcrcon resource for command execution
  - ✅ EULA acceptance and JVM configuration
  - ✅ Graceful shutdown with save-all support
  - ✅ World backup functionality
  - ✅ Comprehensive test suite (smoke/unit/integration)
  - ✅ Server.properties generation and management

- 2025-09-15: Validation and completion by generator (85%)
  - ✅ Fixed health service implementation with proper JSON responses
  - ✅ Verified Docker installation and server startup
  - ✅ Confirmed RCON connectivity with mcrcon resource
  - ✅ Tested command execution (list, say commands)
  - ✅ Validated graceful shutdown process

- 2025-09-16: Full P2 implementation by improver (100%)
  - ✅ Fixed health service port conflict (changed to port 11460)
  - ✅ Implemented plugin management (add/remove/list plugins)
  - ✅ Added health monitoring with TPS and player count metrics
  - ✅ Created comprehensive log analysis with event tracking
  - ✅ Fixed log parsing issues for Docker environments
  - ✅ All tests passing (smoke, unit, integration)

### Current Sprint
- [x] Create v2.0 compliant structure
- [x] Implement Docker-based server deployment
- [x] Configure RCON for mcrcon connectivity
- [x] Create comprehensive test suite
- [x] Validate RCON integration with mcrcon

### Known Issues
- Port 11459 conflict with system health monitor (resolved by using port 11460)
- Health service requires Python 3 installed
- Plugin repository is limited to a few common plugins (can use direct URLs for any plugin)

### Test Commands
```bash
# Installation and lifecycle
vrooli resource papermc manage install
vrooli resource papermc manage start --wait
vrooli resource papermc status

# Configure mcrcon for the server
vrooli resource mcrcon content add localhost localhost 25575 changeme123

# Verify RCON connectivity (requires environment variables)
MCRCON_DEFAULT_SERVER=localhost MCRCON_PASSWORD=changeme123 \
  vrooli resource mcrcon content execute "list"
  
MCRCON_DEFAULT_SERVER=localhost MCRCON_PASSWORD=changeme123 \
  vrooli resource mcrcon content execute "say PaperMC server is running!"

# Stop the server
vrooli resource papermc manage stop

# Testing
vrooli resource papermc test smoke
vrooli resource papermc test all
```

## Revenue Justification

### Direct Revenue Opportunities
1. **Server Hosting Services** ($20K): Managed Minecraft server hosting with automation
2. **Educational Platforms** ($10K): Schools using Minecraft for STEM education
3. **Virtual Event Spaces** ($10K): Corporate team building and virtual conferences

### Indirect Value Creation
- Completes the mcrcon ecosystem for full automation
- Enables complex multi-server scenarios
- Foundation for Minecraft-based applications

### Market Validation
- Minecraft server hosting is a $500M+ market
- Growing demand for educational Minecraft usage
- Enterprises exploring virtual worlds for collaboration
- PaperMC is the most popular performance-optimized server

Total Estimated Value: $40K+ in direct scenario revenue