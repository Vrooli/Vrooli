# Huginn Workflow Automation Platform

Huginn is an agent-based workflow automation platform that helps you build automated workflows using connected components. Now with AI-powered event filtering and performance monitoring!

## Quick Start

```bash
# Install Huginn
vrooli resource huginn manage install

# Start the service
vrooli resource huginn manage start

# Check status
vrooli resource huginn status

# List agents
vrooli resource huginn content list

# View performance metrics
vrooli resource huginn performance dashboard
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

## New Features

### Native API Endpoints
Direct API access without Rails runner for improved performance:
```bash
# Get API status
vrooli resource huginn api status

# List all agents
vrooli resource huginn api agents list

# Get specific agent
vrooli resource huginn api agents get 1

# Create new agent
vrooli resource huginn api agents create "Test Agent" "ManualEventAgent" '{}'

# Run agent
vrooli resource huginn api agents run 1

# List events
vrooli resource huginn api events list

# List scenarios
vrooli resource huginn api scenarios list
```

### Multi-Tenant Support
Isolated workspaces for different users/teams:
```bash
# Create tenant
vrooli resource huginn tenant create demo-user demo@example.com SecurePass123

# List all tenants
vrooli resource huginn tenant list

# Get tenant details
vrooli resource huginn tenant get demo-user

# Check tenant quotas
vrooli resource huginn tenant quota demo-user

# Export tenant data
vrooli resource huginn tenant export demo-user demo-backup.json

# Import tenant data
vrooli resource huginn tenant import demo-backup.json

# View multi-tenant statistics
vrooli resource huginn tenant stats
```

### AI-Powered Event Filtering (Ollama Integration)
```bash
# Test Ollama connectivity
vrooli resource huginn ollama test

# Create an AI filter agent
vrooli resource huginn ollama create-filter "Critical Alert Filter" "only critical severity events"

# List AI filter agents
vrooli resource huginn ollama list-filters

# Analyze event with AI
vrooli resource huginn ollama analyze '{"message":"CPU at 95%"}' "critical events only"
```

### Performance Monitoring
```bash
# View performance dashboard
vrooli resource huginn performance dashboard

# Get metrics as JSON
vrooli resource huginn performance metrics

# Export metrics to file
vrooli resource huginn performance export metrics.json
```

## Examples

### Basic System Check
```bash
# Check if Huginn is running and healthy
vrooli resource huginn status
vrooli resource huginn test smoke
```

### Agent Operations
```bash
# List all agents
vrooli resource huginn content list

# Export agents to JSON
vrooli resource huginn export agents all agents-backup.json

# Import agents from JSON
vrooli resource huginn import agents-backup.json
```

### Monitoring and Backup
```bash
# Real-time monitoring dashboard
vrooli resource huginn monitor

# Create system backup
vrooli resource huginn backup /tmp/huginn-backup

# Restore from backup
vrooli resource huginn restore /tmp/huginn-backup.tar.gz

# View logs
vrooli resource huginn logs
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