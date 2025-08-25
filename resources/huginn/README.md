# Huginn Workflow Automation Platform

Huginn is an agent-based workflow automation platform that helps you build automated workflows using connected components.

## Quick Start

```bash
# Install Huginn
./manage.sh --action install

# Check status
./manage.sh --action status

# List agents
./manage.sh --action agents --operation list

# View recent events
./manage.sh --action events --operation recent
```

## Web Interface

- **URL:** http://localhost:4111
- **Username:** admin
- **Password:** vrooli_huginn_secure_2025

## Key Concepts

- **Agents:** Individual workflow components (RSS readers, website monitors, webhooks, etc.)
- **Scenarios:** Collections of connected agents forming complete workflows
- **Events:** Data flowing between agents in the system
- **Links:** Connections between agents that define data flow

## Available Actions

### System Management
- `install` - Install and set up Huginn
- `uninstall` - Remove Huginn and optionally clean up data
- `start/stop/restart` - Control container lifecycle
- `status` - Show comprehensive system status
- `health` - Run health checks
- `logs` - View container logs

### Agent Management
- `agents --operation list` - List all agents
- `agents --operation show --agent-id <id>` - Show agent details
- `agents --operation run --agent-id <id>` - Run agent manually
- `agents --operation types` - List available agent types

### Scenario Management
- `scenarios --operation list` - List all scenarios
- `scenarios --operation show --scenario-id <id>` - Show scenario details

### Event Monitoring
- `events --operation recent --count <n>` - Show recent events
- `events --operation agent --agent-id <id>` - Show events for specific agent

### System Operations
- `backup` - Create system backup
- `integration` - Check integration with other Vrooli resources
- `monitor --interval <seconds>` - Real-time monitoring

## Examples

### Basic System Check
```bash
# Check if Huginn is running and healthy
./manage.sh --action status
./manage.sh --action health
```

### Agent Operations
```bash
# List all agents with their status
./manage.sh --action agents --operation list

# Show detailed information for agent ID 5
./manage.sh --action agents --operation show --agent-id 5

# Run agent ID 10 manually
./manage.sh --action agents --operation run --agent-id 10
```

### Monitoring
```bash
# View recent system activity
./manage.sh --action events --operation recent --count 20

# Monitor system in real-time (30-second intervals)
./manage.sh --action monitor --interval 30

# View application logs
./manage.sh --action logs --container app --lines 100
```

### Integration
```bash
# Check integration with other Vrooli resources
./manage.sh --action integration

# Create system backup
./manage.sh --action backup
```

## Directory Structure

```
resources/huginn/
├── config/                  # Configuration files
│   ├── defaults.sh         # Default settings and constants
│   └── messages.sh         # User-facing messages
├── lib/                    # Modular libraries
│   ├── common.sh          # Core utilities
│   ├── docker.sh          # Container management
│   ├── install.sh         # Installation logic
│   ├── status.sh          # Status and health checks
│   └── api.sh             # Agent/scenario operations
├── examples/              # Sample configurations
│   ├── agents/           # Sample agent configs
│   ├── scenarios/        # Sample scenario configs
│   └── workflows/        # Complete workflow examples
├── templates/            # Reusable templates
├── docker/              # Docker configurations
└── manage.sh            # Main management script
```

## Integration with Vrooli

Huginn integrates seamlessly with other Vrooli resources:

- **MinIO:** Store monitoring data and workflow artifacts
- **Redis:** Publish events to Vrooli's event bus
- **Node-RED:** Chain workflows between automation platforms
- **Ollama:** AI-powered content analysis and filtering

## Troubleshooting

### Common Issues

1. **Port conflicts:** Check if port 4111 is available
2. **Container not starting:** Check Docker daemon and permissions
3. **Database connection:** Verify PostgreSQL container is healthy
4. **Low memory:** Huginn requires at least 2GB RAM for optimal performance

### Getting Help

```bash
# View detailed system information
./manage.sh --action info

# Check system health
./manage.sh --action health

# View logs for troubleshooting
./manage.sh --action logs --container app --lines 200

# Test integration status
./manage.sh --action integration
```

## Architecture

Huginn follows Vrooli's resource management patterns:

- **Modular Design:** Separate libraries for different functionality
- **Configuration Management:** Centralized defaults and settings
- **Resource Integration:** Standard integration with Vrooli ecosystem
- **Health Monitoring:** Comprehensive status and health checks
- **Error Handling:** Graceful failure handling and recovery

For more information about Vrooli's resource system, see the main resource documentation.