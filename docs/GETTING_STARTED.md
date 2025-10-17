# Getting Started with Vrooli

Welcome to Vrooli - a platform where business applications run **directly from source** with zero build steps. Create $10K-50K revenue applications by orchestrating 30+ local resources.

## What is Vrooli?

Vrooli is a **resource orchestration platform** that enables instant business application deployment through direct scenario execution. No compilation, no build artifacts, no conversion - just immediate execution from source.

**Key Innovation**: Scenarios run directly from their `scenarios/` folders - edit and run instantly!

## 5-Minute Quick Start

### 1. Setup
```bash
# Clone and setup
git clone https://github.com/Vrooli/Vrooli.git
cd Vrooli
./scripts/setup.sh  # or: vrooli setup
```

### 2. Start Resources
```bash
# Start essential resources
vrooli resource start-all

# Verify they're running
vrooli resource status
```

### 3. Run Your First Scenario
```bash
# List available scenarios
vrooli scenario list

# Run a scenario DIRECTLY from source
vrooli scenario run research-assistant

# Access at http://localhost:3000 (or configured port)
```

**✅ Congratulations!** You have a $15K research assistant application running!

## Core Concepts (2 Minutes)

### Resources = Capabilities
Local services that provide specific functionality:
- **AI**: Ollama, Whisper, ComfyUI
- **Storage**: PostgreSQL, Redis, Qdrant
- **Automation**: N8n, Windmill, Node-RED
- **Agents**: Agent-S2, Browserless

### Scenarios = Business Applications  
Complete applications that orchestrate resources:
- Each scenario worth $10K-50K when deployed
- Run directly from `scenarios/<name>/` folder
- Serve as both integration test AND production app

### Direct Execution = No Build Steps
```
Traditional: Code → Build → Package → Deploy → Run ❌
Vrooli:      scenarios/my-app/ → Run! ✅
```

## Choose Your Path

### 🚀 Path 1: Use Existing Scenarios
**For:** Users wanting to run business applications  
**Time:** Immediate

```bash
vrooli scenario run invoice-generator
vrooli scenario run customer-portal
```

Each scenario is a complete, revenue-generating application ready to deploy.

### 🛠️ Path 2: Create New Scenarios  
**For:** Builders creating custom business applications  
**Time:** 30-45 minutes to first scenario

**→ Continue with [Scenario Creator's Guide](scenarios/getting-started.md)**

Learn to build your own $10K-50K applications from templates.

### 🔧 Path 3: Develop Vrooli Core
**For:** Contributors improving the platform itself  
**Time:** 2-4 hours setup

```bash
vrooli develop  # Start development environment
vrooli test     # Run test suite
```

**→ See [Contributing Guide](CONTRIBUTING.md)**

## Essential Commands

```bash
# Scenarios (Direct Execution)
vrooli scenario list         # List available
vrooli scenario run <name>   # Run from source
vrooli scenario test <name>  # Test integration

# Resources  
vrooli resource list         # List available
vrooli resource start-all    # Start all
vrooli resource status       # Check health

# System
vrooli status               # System health
vrooli stop                 # Stop everything
```

## Quick Troubleshooting

| Issue | Solution |
|-------|----------|
| Scenario won't start | Check `vrooli resource status` |
| Port conflicts | See `~/.vrooli/port-registry.json` |
| Resource unavailable | Run `vrooli resource start-all` |

**→ Full troubleshooting: [Troubleshooting Guide](devops/troubleshooting.md)**

## Next Steps

**Immediate Actions:**
1. Run `vrooli scenario list` to explore applications
2. Try 2-3 different scenarios to see the variety
3. Choose your path above based on your goals

**Documentation:**
- [Architecture Overview](ARCHITECTURE_OVERVIEW.md) - System design
- [Resource Documentation](resources/README.md) - Available resources
- [Deployment Guide](deployment/README.md) - Production deployment

---

**Welcome to Vrooli!** Build and deploy business applications instantly with direct execution.

📚 [Main Documentation](README.md) | 💬 [Discord](https://discord.gg/vrooli) | 🐛 [Issues](https://github.com/Vrooli/Vrooli/issues)