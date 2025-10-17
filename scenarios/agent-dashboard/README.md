# Agent Dashboard

## Purpose
Cyberpunk-themed monitoring dashboard for AI agents within Vrooli resources. Provides real-time visibility into agent health, logs, and performance metrics.

## Status
**P0 Complete (8/8), P1 Complete (6/6), P2 Complete (5/5)** (2025-09-27): All features working and validated.
- ✅ Agent discovery across resources
- ✅ Real-time status monitoring
- ✅ Cyberpunk UI with radar visualization
- ✅ Log viewer functionality
- ✅ Basic agent control (stop)
- ✅ Agent search and filtering
- ✅ Hover tooltips on radar
- ✅ Log export/download
- ✅ Keyboard shortcuts (press ? for help)
- ✅ Responsive design for different screens
- ✅ Agent performance history graphs
- ✅ Multiple log viewers support
- ✅ Agent grouping by resource type
- ✅ Agent capability matching/discovery
- ✅ Custom radar themes (cyberpunk, matrix, military, neon)

## Quick Start
```bash
# Start the dashboard
make run

# Access interfaces
# Dashboard: http://localhost:{UI_PORT}
# API: http://localhost:{API_PORT}/api/v1
# CLI: agent-dashboard help

# Launch a Codex task from the CLI
agent-dashboard start --task "Audit agent registry state" --mode auto

# Replay the most recent task for an agent
agent-dashboard restart codex:abcd1234
```

## Codex Integration Highlights
- 🔁 **Real Codex agents** - the dashboard now launches and tracks codex-powered tasks instead of static stubs.
- 🚀 **One-click launches** - use the new "Launch Codex Agent" button in the UI or `agent-dashboard start` from the CLI to run investigations, remediations, or health checks.
- 🧠 **Task replay** - restart any historical run with `agent-dashboard restart <id>` to re-run the stored Codex task.
- 📡 **Live metrics & logs** - CPU, memory, uptime, and streaming logs are sourced directly from the managed Codex processes.

## Key Features
- **Agent Discovery**: Automatically finds agents across claude-code, crewai, ollama, and other resources
- **Real-time Monitoring**: Live status updates with 30-second auto-refresh
- **Cyberpunk UI**: Animated radar view with agent visualizations and hover tooltips
- **Log Streaming**: Real-time log viewing for any agent with export capability
- **CLI Interface**: Full control from command line
- **Search & Filter**: Find agents by name, type, status with real-time filtering
- **Sort Options**: Multiple sort options including name, type, status, uptime, memory usage

## API Endpoints
- `GET /health` - Service health check
- `GET /api/v1/agents` - List all discovered agents
- `POST /api/v1/agents/{id}/stop` - Stop specific agent
- `GET /api/v1/agents/{id}/logs` - Get agent logs
- `POST /api/v1/agents/scan` - Trigger immediate discovery
- `GET /api/v1/capabilities` - Get all available agent capabilities
- `GET /api/v1/agents/search?capability=X` - Search agents by capability

## CLI Commands
```bash
# List all agents
agent-dashboard list

# View agent status
agent-dashboard status <agent-name>

# View agent logs
agent-dashboard logs <agent-name> --lines 50

# Stop an agent
agent-dashboard stop <agent-name>

# Trigger immediate agent scan
agent-dashboard scan
```

## Dependencies
- **Resources**: None (monitors existing resources)
- **Optional**: Any Vrooli resource with agent support

## UX Style
Cyberpunk-themed dashboard with animated radar, scan lines, and Matrix-inspired aesthetics. Features cyan/magenta/yellow color palette with smooth animations.

## Testing
```bash
# Run comprehensive test suite
make test

# All tests pass:
# ✅ Unit tests
# ✅ Integration tests
# ✅ Structure tests
# ✅ Dependencies tests
# ✅ Business tests
# ✅ Performance tests
```
