# Node-RED Resource

Node-RED is a flow-based programming tool for wiring together hardware devices, APIs, and online services. This resource provides automated installation, configuration, and management of Node-RED for the Vrooli project with enhanced host system access.

## Overview

- **Category**: Automation
- **Default Port**: 1880
- **Service Type**: Docker Container
- **Container Name**: node-red
- **Documentation**: [nodered.org](https://nodered.org)

## Features

- 🔄 **Flow-based Programming**: Visual programming using flows and nodes
- 💻 **Host Command Access**: Run system commands directly from Node-RED flows
- 🐳 **Docker Integration**: Control other Docker containers from within Node-RED
- 📁 **Workspace Access**: Direct access to the Vrooli project files
- 🔧 **Enhanced Tools**: Pre-installed nodes for common automation tasks
- 📊 **Dashboard Support**: Built-in dashboard for creating UIs
- 🌐 **HTTP Endpoints**: Create REST APIs and webhooks
- 📈 **Real-time Processing**: Event-driven flows with continuous execution

## Prerequisites

- Docker installed and running
- 512MB+ RAM available
- 1GB+ free disk space
- Port 1880 available (or custom port)
- (Optional) Docker socket access for container management

## Node-RED vs n8n

### When to Use Node-RED
- **Real-time data processing**: Continuous streams, MQTT, WebSockets
- **IoT and hardware integration**: GPIO, sensors, serial communications
- **Event-driven automation**: Immediate response to system events
- **Dashboard creation**: Built-in UI components for monitoring
- **Protocol bridging**: Convert between different communication protocols

### When to Use n8n
- **Business workflow automation**: Scheduled tasks, API orchestrations
- **SaaS integrations**: Pre-built connectors for cloud services
- **Complex data transformations**: Advanced JSON manipulation
- **Audit and replay**: Detailed execution history and debugging

### Complementary Usage
Node-RED and n8n work excellently together:
- **Node-RED**: Real-time monitoring, alerts, IoT data collection
- **n8n**: Business process automation, scheduled workflows, data pipelines
- **Integration**: Share data via webhooks, message queues, or shared storage

## Installation

### Quick Start
```bash
# Install with custom image (recommended)
./manage.sh --action install --build-image yes

# Install with standard Node-RED image
./manage.sh --action install --build-image no
```

### Installation Options
```bash
./manage.sh --action install \
  --build-image yes \           # Build custom image with host access
  --force yes                   # Force reinstall if already exists
```

## Usage

### Management Commands
```bash
# Check status
./manage.sh --action status

# View logs
./manage.sh --action logs

# Stop Node-RED
./manage.sh --action stop

# Start Node-RED
./manage.sh --action start

# Restart Node-RED
./manage.sh --action restart

# Uninstall Node-RED
./manage.sh --action uninstall

# Show resource metrics
./manage.sh --action metrics
```

### Flow Management
```bash
# List all flows
./manage.sh --action flow-list

# Export flows to file
./manage.sh --action flow-export --output ./my-flows.json

# Import flows from file
./manage.sh --action flow-import --flow-file ./my-flows.json

# Execute flow via HTTP endpoint
./manage.sh --action flow-execute --endpoint "/test/hello"

# Execute flow with JSON data
./manage.sh --action flow-execute --endpoint "/test/exec" --data '{"command": "ls -la"}'
```

### Testing and Validation
```bash
# Run complete test suite
./manage.sh --action test

# Test host command access
./manage.sh --action validate-host

# Test Docker socket access
./manage.sh --action validate-docker

# Show performance metrics
./manage.sh --action metrics
```

## Custom Setup Details

### Custom Docker Image
- Based on official Node-RED image with additional tools
- Seamless host command execution (no absolute paths needed)
- Pre-installed: bash, curl, jq, python3, docker-cli, development tools
- Smart PATH handling for transparent host binary access

### Volume Mounts
- `/data`: Node-RED user data and flows
- `/data/flows`: Flow configuration files
- `/workspace`: Direct access to Vrooli project
- `/var/run/docker.sock`: Docker control capabilities (optional)
- `/host/usr/bin`, `/host/bin`: Read-only access to system binaries

### Pre-installed Nodes
- **node-red-contrib-exec**: Enhanced command execution
- **node-red-contrib-dockerode**: Docker API integration
- **node-red-contrib-fs**: File system operations
- **node-red-contrib-cpu**: System monitoring
- **node-red-dashboard**: Dashboard UI components
- **node-red-contrib-string**: String manipulation utilities

### Security Note
This setup prioritizes functionality over isolation. It's designed for development and trusted environments where Node-RED needs full system access.

## Built-in Test Flows

### Basic Connectivity Test
- **Endpoint**: `GET /test/hello`
- **Purpose**: Verify Node-RED is responding
- **Response**: JSON with status and Node-RED version

```bash
curl http://localhost:1880/test/hello
```

### Command Execution Test
- **Endpoint**: `POST /test/exec`
- **Purpose**: Test host command execution with validation
- **Allowed Commands**: ls, pwd, date, whoami, echo, cat, grep, find

```bash
curl -X POST http://localhost:1880/test/exec \
     -H "Content-Type: application/json" \
     -d '{"command": "ls -la /workspace"}'
```

### Docker Access Test
- **Endpoint**: `GET /test/docker`
- **Purpose**: Verify Docker socket access
- **Response**: List of running containers

```bash
curl http://localhost:1880/test/docker
```

## Flow Development Examples

### System Monitoring Flow
```javascript
// Function node to get system stats
const os = require('os');
msg.payload = {
    hostname: os.hostname(),
    uptime: os.uptime(),
    loadavg: os.loadavg(),
    memory: {
        total: os.totalmem(),
        free: os.freemem()
    },
    timestamp: new Date().toISOString()
};
return msg;
```

### File Processing Flow
```javascript
// Function node to process files
const fs = require('fs');
const path = '/workspace/logs';

try {
    const files = fs.readdirSync(path);
    const logFiles = files.filter(f => f.endsWith('.log'));
    
    msg.payload = {
        directory: path,
        totalFiles: files.length,
        logFiles: logFiles,
        timestamp: new Date().toISOString()
    };
} catch (error) {
    msg.payload = {
        error: error.message,
        timestamp: new Date().toISOString()
    };
}
return msg;
```

### Docker Container Management
```javascript
// Function node to manage containers
const exec = require('child_process').exec;

const command = `docker ${msg.payload.action} ${msg.payload.container}`;

exec(command, (error, stdout, stderr) => {
    if (error) {
        msg.payload = {
            status: 'error',
            error: error.message,
            stderr: stderr
        };
    } else {
        msg.payload = {
            status: 'success',
            output: stdout,
            stderr: stderr
        };
    }
    
    // Send the message to the next node
    node.send(msg);
});
```

## Advanced Features

### Dashboard Creation
Node-RED includes a powerful dashboard module for creating real-time monitoring interfaces:

1. Install dashboard nodes (pre-installed in custom image)
2. Create dashboard flows with UI nodes
3. Access dashboard at: `http://localhost:1880/ui`

### MQTT Integration
```javascript
// MQTT subscriber flow
[
    {
        "type": "mqtt in",
        "topic": "sensor/temperature",
        "broker": "mqtt-broker-config"
    },
    {
        "type": "function",
        "func": "msg.payload = JSON.parse(msg.payload);\nreturn msg;"
    },
    {
        "type": "debug",
        "name": "Temperature Data"
    }
]
```

### WebSocket Real-time Communication
```javascript
// WebSocket flow for real-time updates
[
    {
        "type": "websocket listener",
        "path": "/ws/updates"
    },
    {
        "type": "function",
        "func": "// Process incoming WebSocket data\nmsg.payload = {\n    type: 'update',\n    data: msg.payload,\n    timestamp: Date.now()\n};\nreturn msg;"
    },
    {
        "type": "websocket out"
    }
]
```

## Integration with Vrooli

### Resource Configuration
Node-RED is automatically configured in `~/.vrooli/resources.local.json`:

```json
{
  "services": {
    "automation": {
      "node-red": {
        "enabled": true,
        "baseUrl": "http://localhost:1880",
        "adminUrl": "http://localhost:1880/admin",
        "healthCheck": {
          "endpoint": "/",
          "intervalMs": 60000,
          "timeoutMs": 5000
        },
        "flows": {
          "directory": "/data/flows",
          "autoBackup": true,
          "backupInterval": "1h"
        }
      }
    }
  }
}
```

### Health Monitoring
- **Endpoint**: `http://localhost:1880/`
- **Method**: GET
- **Expected**: 200 OK with Node-RED editor
- **Timeout**: 5 seconds
- **Interval**: 60 seconds

### Flow Backup Strategy
- Flows are automatically exported during installation
- Manual backup: `./manage.sh --action flow-export`
- Flows stored in `/data/flows/` directory
- Version control friendly JSON format

## Troubleshooting

### Node-RED Won't Start
- Check logs: `./manage.sh --action logs`
- Verify port availability: `ss -tlnp | grep 1880`
- Check Docker: `docker ps | grep node-red`

### Commands Not Found
- Verify host directories are mounted: `docker exec node-red ls /host/bin`
- Test command resolution: `docker exec node-red which ls`
- Check PATH setup: `docker exec node-red echo $PATH`

### Flow Execution Errors
- Check flow validation: Node-RED editor shows errors
- Verify node dependencies: Check if required nodes are installed
- Review debug output: Use debug nodes to trace flow execution

### Performance Issues
- Monitor memory usage: `./manage.sh --action metrics`
- Check flow complexity: Simplify resource-intensive flows
- Review debug output volume: Limit debug node outputs

### Docker Access Issues
- Verify socket mount: `docker exec node-red ls -la /var/run/docker.sock`
- Check permissions: `docker exec node-red docker ps`
- Alternative: Remove Docker socket mount if not needed

## Architecture

### Directory Structure
```
node-red/
├── manage.sh                    # Main management script
├── Dockerfile                   # Custom Node-RED image
├── docker-entrypoint.sh         # Enhanced entrypoint for host access
├── settings.js                  # Node-RED configuration
├── README.md                    # This documentation
└── flows/                       # Flow definitions
    ├── default-flows.json       # Combined test flows
    ├── test-basic.json          # Basic connectivity test
    ├── test-exec.json           # Command execution test
    └── test-docker.json         # Docker integration test
```

### How It Works

1. **Custom Entrypoint**: The `docker-entrypoint.sh` script:
   - Adds host directories to PATH
   - Sets up command fallback handling
   - Preserves Node-RED's original functionality

2. **Smart Command Resolution**:
   - Container binaries are preferred
   - Falls back to host binaries if not found
   - Transparent command execution for flows

3. **Volume Strategy**:
   - Read-only mounts for system directories
   - Read-write for workspace and data
   - Conditional Docker socket access

4. **Flow Management**:
   - JSON-based flow definitions
   - Hot-reload capabilities
   - Version control integration

## Best Practices

### Flow Design
1. **Error Handling**: Always include error branches in flows
2. **Resource Management**: Limit concurrent executions
3. **Debugging**: Use debug nodes during development
4. **Validation**: Validate inputs in function nodes
5. **Documentation**: Use info panels to document flows

### Security
1. **Command Validation**: Whitelist allowed commands
2. **Input Sanitization**: Validate all external inputs
3. **Resource Limits**: Set timeouts and buffer limits
4. **Access Control**: Limit Docker socket access if not needed

### Performance
1. **Flow Optimization**: Minimize complex operations
2. **Memory Management**: Monitor context data usage
3. **Batch Processing**: Group related operations
4. **Async Operations**: Use non-blocking operations where possible

### Maintenance
1. **Regular Backups**: Export flows regularly
2. **Version Control**: Store flows in git repository
3. **Monitoring**: Set up health checks and alerts
4. **Updates**: Keep Node-RED and nodes updated

## FAQ

**Q: How is Node-RED different from n8n?**
A: Node-RED is event-driven and real-time focused, while n8n is workflow-oriented with better business integrations. They complement each other well.

**Q: Can I use Node-RED for IoT projects?**
A: Yes! Node-RED excels at IoT integration with MQTT, WebSockets, and hardware protocols.

**Q: How do I create custom nodes?**
A: You can create custom nodes using JavaScript. Place them in the `/data/nodes` directory.

**Q: Can I access the dashboard from outside localhost?**
A: The dashboard is available at `/ui` endpoint. For external access, configure proper network settings.

**Q: How do I persist data between flows?**
A: Use Node-RED's context storage (flow, global) or external databases/files.

**Q: Can I schedule flows like cron jobs?**
A: Yes, use the inject node with scheduling options or cron-plus node for advanced scheduling.

## Configuration

### Environment Variables
- `NODE_RED_PORT`: Override default port (default: 1880)
- `NODE_RED_FLOW_FILE`: Flow file name (default: flows.json)
- `NODE_RED_CREDENTIAL_SECRET`: Encryption key for credentials
- `TZ`: Timezone setting

### Advanced Settings
Edit `settings.js` for advanced configuration:
- Function node global context
- HTTP timeouts and limits
- Logging levels and formats
- Editor theme and projects

## Support

For issues related to:
- This custom setup: Check this README and manage.sh script
- Node-RED itself: Visit https://nodered.org/docs
- Vrooli integration: See Vrooli documentation

## Development

### Building Custom Image Manually
```bash
cd scripts/resources/automation/node-red
docker build -t node-red-vrooli:latest .
```

### Testing Commands
```bash
# Test command resolution
docker exec node-red which curl
docker exec node-red curl --version

# Test workspace access
docker exec node-red ls /workspace

# Test Docker access
docker exec node-red docker ps
```

### Extending the Image
Edit `Dockerfile` to add more tools:
```dockerfile
RUN npm install --no-audit --no-update-notifier \
    your-node-package-here
```

Then rebuild: `./manage.sh --action install --build-image yes --force yes`