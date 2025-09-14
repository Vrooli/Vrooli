# CrewAI Resource

Multi-agent AI framework for building collaborative AI systems within Vrooli.

## Overview

CrewAI enables the creation of autonomous AI agents that work together to accomplish complex tasks. This resource provides a local CrewAI server that manages crews and agents, allowing scenarios to leverage multi-agent collaboration for problem-solving.

## Features

### Current
- **Dual-Mode Operation**: Supports both real CrewAI and mock mode
- **Flask API Server**: Modern REST API with CORS support
- **Health Monitoring**: Standard health endpoint with CrewAI status
- **Crew Management**: Full CRUD operations for AI crews
- **Agent Management**: Full CRUD operations for AI agents
- **Task Execution**: Run tasks with progress tracking
- **Content Injection**: Import crews and agents from external files
- **Auto-Detection**: Automatically detects if CrewAI library is available
- **v2.0 Compliant**: Follows Vrooli resource standards

### Planned
- **Tool Integration**: Agents can use external tools and APIs
- **Memory System**: Persistent memory via Qdrant integration
- **UI Dashboard**: Web interface for visual management
- **Workflow Designer**: Visual tool for creating agent workflows

## Quick Start

### Installation
```bash
# Install CrewAI resource (with real library support)
vrooli resource crewai manage install

# For mock-only mode (no dependencies)
CREWAI_MOCK_MODE=true vrooli resource crewai manage install

# Start the service
vrooli resource crewai manage start --wait

# Check status
vrooli resource crewai status
```

### Basic Usage
```bash
# List available crews
vrooli resource crewai content crews

# List available agents
vrooli resource crewai content agents

# Inject a crew from file
vrooli resource crewai content inject --file my_crew.py --type crew

# Run tests
vrooli resource crewai test smoke
```

## API Endpoints

The CrewAI server provides the following REST API endpoints:

### Core Endpoints
- `GET /` - API info and capabilities (shows CrewAI library status)
- `GET /health` - Health check with active task count

### Agent Management
- `GET /agents` - List all agents with metadata
- `POST /agents` - Create new agent with role/goal/backstory
- `GET /agents/{name}` - Get specific agent details
- `DELETE /agents/{name}` - Delete agent

### Crew Management
- `GET /crews` - List all crews with agents/tasks
- `POST /crews` - Create new crew with agent assignments
- `GET /crews/{name}` - Get specific crew details
- `DELETE /crews/{name}` - Delete crew

### Task Execution
- `POST /execute` - Execute crew with input data
- `GET /tasks` - List all task executions
- `GET /tasks/{id}` - Get task execution status

### Content Management
- `POST /inject` - Inject crew/agent from file

### Example API Calls
```bash
# Check health
curl http://localhost:8084/health

# List crews
curl http://localhost:8084/crews

# Inject a crew
curl -X POST http://localhost:8084/inject \
  -H "Content-Type: application/json" \
  -d '{"file_path": "/path/to/crew.py", "file_type": "crew"}'
```

## Configuration

### Environment Variables
- `CREWAI_PORT` - API server port (default: 8084)
- `CREWAI_DATA_DIR` - Data directory (default: ~/.crewai)
- `CREWAI_MOCK_MODE` - Run in mock mode (default: true)

### Configuration Files
- `config/defaults.sh` - Default configuration values
- `config/schema.json` - Configuration schema
- `config/runtime.json` - Runtime dependencies and startup order

## Directory Structure

```
~/.crewai/
├── crews/          # Crew definitions
├── agents/         # Agent definitions
├── workspace/      # Working directory for crews
├── crewai.log      # Service logs
└── server.py       # Mock server implementation
```

## Testing

Run the comprehensive test suite:

```bash
# All tests
vrooli resource crewai test all

# Individual test phases
vrooli resource crewai test smoke       # Quick health check (<30s)
vrooli resource crewai test integration # End-to-end tests (<120s)
vrooli resource crewai test unit        # Library function tests (<60s)
```

## Troubleshooting

### Service Won't Start
1. Check if port 8084 is available: `ss -tlnp | grep 8084`
2. Check Python installation: `python3 --version`
3. Review logs: `vrooli resource crewai logs`

### Health Check Failing
1. Verify service is running: `vrooli resource crewai status`
2. Check endpoint directly: `curl http://localhost:8084/health`
3. Review server logs: `cat ~/.crewai/crewai.log`

### Crews/Agents Not Loading
1. Check directory permissions: `ls -la ~/.crewai/`
2. Verify file format (must be .py files)
3. Check injection response for errors

## Integration with Other Resources

### Ollama Integration (Planned)
```bash
# Configure CrewAI to use Ollama for LLM
export CREWAI_LLM_PROVIDER=ollama
export CREWAI_LLM_MODEL=llama3
```

### Qdrant Integration (Planned)
```bash
# Enable memory persistence with Qdrant
export CREWAI_MEMORY_ENABLED=true
export CREWAI_QDRANT_HOST=localhost
export CREWAI_QDRANT_PORT=6333
```

## Development

### Mock Mode
Currently runs in mock mode with a Python server that simulates CrewAI functionality. This allows for:
- Testing integration points
- Developing scenarios without full CrewAI
- Validating the resource structure

### Future Development
1. **Phase 1**: Complete v2.0 compliance ✅
2. **Phase 2**: Real CrewAI integration
3. **Phase 3**: Advanced features (UI, tools, memory)

## CLI Commands

```bash
# Lifecycle management
vrooli resource crewai manage install
vrooli resource crewai manage start [--wait]
vrooli resource crewai manage stop
vrooli resource crewai manage restart
vrooli resource crewai manage uninstall

# Content management
vrooli resource crewai content list
vrooli resource crewai content add
vrooli resource crewai content get
vrooli resource crewai content remove
vrooli resource crewai content inject

# Testing
vrooli resource crewai test smoke
vrooli resource crewai test integration
vrooli resource crewai test unit
vrooli resource crewai test all

# Information
vrooli resource crewai help
vrooli resource crewai info
vrooli resource crewai status
vrooli resource crewai logs
```

## Contributing

See [PRD.md](PRD.md) for the product requirements and development roadmap.

## License

Part of the Vrooli ecosystem - see main project license.