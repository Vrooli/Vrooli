# Vrooli Assistant

**Real-time issue capture and agent spawning overlay for rapid Vrooli iteration**

## 🚀 Overview

Vrooli Assistant is a meta-scenario that provides a persistent desktop overlay for capturing issues and spawning agents to fix them in real-time. It's designed to create a zero-friction feedback loop that accelerates development and improvement of all Vrooli scenarios.

## ✨ Key Features

- **Global Hotkey Activation**: `Cmd+Shift+Space` (Mac) / `Ctrl+Shift+Space` (Linux/Windows)
- **Screenshot Capture**: Automatically captures visual context of issues
- **Context Preservation**: Captures URL, scenario name, DOM state, console errors
- **Agent Spawning**: Instantly spawns Claude Code or Agent S2 with full context
- **Task Tracking**: Automatically creates tasks in the backlog system
- **Issue History**: Track all captured issues and their resolution status
- **Pattern Recognition**: Identifies similar issues to prevent duplicates

## 🎯 Purpose

This scenario adds a **permanent meta-capability** to Vrooli:
- **10x faster iteration speed** on scenario improvements
- **Zero context switching** when reporting issues
- **Automatic agent assignment** with rich context
- **Pattern analysis** to identify systemic problems

## 🔧 Architecture

### Components
- **Electron Overlay**: Cross-platform desktop app with global hotkey
- **Go API**: Handles issue storage, agent coordination, and pattern analysis
- **PostgreSQL**: Stores issues, agent sessions, and patterns
- **CLI Tool**: Command-line interface for daemon control and manual capture

### Technology Stack
- Electron (Desktop overlay)
- Go (API server)
- PostgreSQL (Data persistence)
- Node.js (CLI and build tools)

## 📦 Installation

```bash
# Using Makefile (recommended)
make setup

# Or using Vrooli CLI
vrooli scenario setup vrooli-assistant

# Or manually
cd cli && ./install.sh
cd ../api && go build -o vrooli-assistant-api main.go
cd ../ui/electron && npm install
```

## 🎮 Usage

### Start the Assistant
```bash
# Using Makefile (recommended)
make start

# Or using Vrooli CLI
vrooli scenario start vrooli-assistant

# Or using vrooli-assistant CLI
vrooli-assistant start --daemon  # Start in background
vrooli-assistant start            # Start in foreground
```

### Preview in the Browser (App Monitor friendly)

The Go API now serves a browser-based dashboard alongside the API itself. After
starting the scenario you can open the assistant UI directly:

```
http://localhost:${API_PORT}
```

This UI mirrors the core overlay capabilities (capture, history, agent
handoff) and is wired for the App Monitor via the iframe bridge. No additional
processes are required—the assets are embedded in the API binary.

### Capture an Issue
1. Press `Cmd+Shift+Space` (or configured hotkey)
2. The overlay appears - screenshot is captured automatically
3. Describe the issue
4. Select agent type (Claude Code, Agent S2, or none)
5. Press Submit or `Cmd+Enter`

### CLI Commands
```bash
# Check status
vrooli-assistant status

# Manual capture
vrooli-assistant capture "Description of issue"

# View history
vrooli-assistant history

# Stop daemon
vrooli-assistant stop
```

## 🧪 Testing

```bash
# Run all tests (recommended)
make test

# Run specific test phases
test/phases/test-structure.sh    # Verify file structure
test/phases/test-dependencies.sh # Check dependencies
test/phases/test-unit.sh         # Unit tests
test/phases/test-integration.sh  # Integration tests (includes CLI BATS)
test/phases/test-business.sh     # Business logic tests
test/phases/test-performance.sh  # Performance benchmarks

# Run CLI BATS tests only
bats cli/vrooli-assistant.bats

# Run all phased tests via Go orchestrator
vrooli scenario test vrooli-assistant
```

## 🔌 API Endpoints

- `GET /health` - Health check
- `GET /api/v1/assistant/status` - Operational status
- `POST /api/v1/assistant/capture` - Capture new issue
- `POST /api/v1/assistant/spawn-agent` - Spawn agent for issue
- `GET /api/v1/assistant/history` - Get recent issues
- `GET /api/v1/assistant/issues/{id}` - Get issue details
- `PUT /api/v1/assistant/issues/{id}/status` - Update issue status

## 🧬 Evolution Path

### Current (v1.0)
- Basic overlay with hotkey activation
- Screenshot capture and context gathering
- Agent spawning with issue context
- PostgreSQL storage

### Planned (v2.0)
- Voice input for issue description
- Pattern recognition and duplicate detection
- Multi-agent orchestration
- CI/CD integration for automatic validation

### Future Vision
- Autonomous issue detection
- Predictive issue prevention
- Cross-scenario learning
- Natural language programming

## 🎨 UX Design

The overlay follows a **minimal, unobtrusive** design philosophy:
- Dark theme with subtle transparency
- Floating window that doesn't block work
- Keyboard-first navigation
- Fast capture with smart defaults
- Auto-hide on completion

## 🔗 Integration Points

### Provides To
- **All Scenarios**: Real-time issue reporting and fixing
- **Agent Dashboard**: Issue tracking data for performance metrics
- **System Monitor**: Development iteration metrics

### Consumes From
- **Task Planner**: Task creation and tracking
- **Agent Dashboard**: Agent session monitoring

## 🚨 Known Limitations

- Hotkey conflicts possible with other applications
- Screenshot may capture sensitive information (use blur tools)
- Electron app requires ~100MB memory when running

## 📊 Performance Targets

- Hotkey activation: < 100ms
- Screenshot capture: < 500ms
- Agent spawn: < 2s
- Memory usage: < 100MB idle
- API response: < 500ms

## 🔒 Security Considerations

- Screenshots stored locally with encryption
- Auto-deletion after 30 days
- Only local user can trigger capture
- All actions logged with timestamps

## 📚 References

- [Electron Documentation](https://www.electronjs.org/)
- [Global Shortcuts API](https://www.electronjs.org/docs/api/global-shortcut)
- [Screenshot Desktop](https://github.com/bencevans/screenshot-desktop)

---

**This scenario is a meta-tool that enhances all other scenarios by enabling rapid iteration and continuous improvement.**
