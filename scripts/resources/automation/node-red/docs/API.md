# Node-RED API Reference

Node-RED provides both built-in APIs and custom endpoints for flow management and system integration. This guide covers the complete API surface including manage.sh integration.

> **💡 Recommended Usage**: Use the `manage.sh` script for most operations. This guide shows direct API usage for advanced integration scenarios.

## Base Access

- **Editor Interface**: `http://localhost:1880`
- **Dashboard UI**: `http://localhost:1880/ui`
- **Recommended Management**: `./manage.sh --action [action]`
- **Runtime API**: `http://localhost:1880/admin/*`
- **Flow Endpoints**: Custom endpoints defined in your flows

## Management API (via manage.sh)

### Service Management

**Recommended Method:**
```bash
# Check service status with comprehensive information
./manage.sh --action status

# Start/stop/restart service
./manage.sh --action start
./manage.sh --action stop  
./manage.sh --action restart

# View service logs and metrics
./manage.sh --action logs
./manage.sh --action metrics
```

### Flow Management

**Recommended Method:**
```bash
# List all flows with details
./manage.sh --action flow-list

# Export flows to file
./manage.sh --action flow-export --output ./my-flows.json

# Import flows from file
./manage.sh --action flow-import --flow-file ./my-flows.json

# Execute specific flow endpoint
./manage.sh --action flow-execute --endpoint "/test/hello"

# Execute flow with JSON data
./manage.sh --action flow-execute --endpoint "/api/resource-check" --data '{"resource": "ollama"}'
```

### Testing and Validation

**Recommended Method:**
```bash
# Run complete test suite
./manage.sh --action test

# Test host command access
./manage.sh --action validate-host

# Test Docker socket access
./manage.sh --action validate-docker
```

## Node-RED Runtime API

**Direct API (for integrations):**

### Flow Management

#### Get All Flows
```http
GET /flows
```

**Response:**
```json
{
  "flows": [...],
  "credentials": {...}
}
```

#### Deploy Flows
```http
POST /flows
Content-Type: application/json

{
  "flows": [...],
  "credentials": {...}
}
```

#### Get Single Flow
```http
GET /flow/{id}
```

### Node Management

#### Get All Nodes
```http
GET /nodes
```

#### Get Node Info
```http
GET /nodes/{module}/{type}
```

#### Install Node Module
```http
POST /nodes
Content-Type: application/json

{
  "module": "node-red-contrib-example"
}
```

### Context API

#### Get Context
```http
GET /context/{scope}/{store?}
```

**Scopes**: `global`, `flow:{id}`, `node:{id}`

#### Set Context
```http
PUT /context/{scope}/{store?}/{key}
Content-Type: application/json

{
  "value": "data"
}
```

## Custom HTTP Endpoints

### Vrooli Resource Monitoring

**Example Endpoints Created by Flows:**

#### Resource Health Check
```http
GET /api/resources/health
```

**Response:**
```json
{
  "timestamp": "2024-01-01T12:00:00Z",
  "resources": {
    "ollama": {"status": "healthy", "port": 11434},
    "n8n": {"status": "healthy", "port": 5678}
  },
  "overall": "healthy"
}
```

#### System Command Execution
```http
POST /api/exec
Content-Type: application/json

{
  "command": "ls -la /workspace",
  "timeout": 10000
}
```

**Response:**
```json
{
  "stdout": "total 24\ndrwxr-xr-x ...",
  "stderr": "",
  "exitCode": 0,
  "duration": 45
}
```

#### Docker Container Management
```http
POST /api/docker/{action}
Content-Type: application/json

{
  "container": "ollama",
  "options": {}
}
```

**Actions**: `start`, `stop`, `restart`, `status`

### Dashboard API

#### Dashboard UI Access
```http
GET /ui
```

Returns the Node-RED dashboard interface.

#### Dashboard Socket.IO
```
ws://localhost:1880/comms
```

Real-time dashboard updates via Socket.IO.

## WebSocket API

### Admin Socket
```
ws://localhost:1880/comms
```

Administrative WebSocket for editor communication.

### Custom Flow WebSockets
```
ws://localhost:1880/ws/{endpoint}
```

Custom WebSocket endpoints defined in flows.

## Message Patterns

### Standard Message Structure
```json
{
  "payload": {...},          // Main data content
  "topic": "resource-name",  // Message routing/identification
  "url": "http://...",       // Dynamic HTTP endpoints
  "resource": {...},         // Custom metadata
  "statusCode": 200,         // HTTP response codes
  "responseTime": 45         // Performance metrics
}
```

### Resource Monitoring Message
```json
{
  "payload": {
    "resource": "ollama",
    "status": "healthy",
    "metrics": {
      "cpu": "15%",
      "memory": "2.1GB",
      "uptime": "4d 12h"
    }
  },
  "topic": "resource-health",
  "timestamp": "2024-01-01T12:00:00Z"
}
```

### Command Execution Message
```json
{
  "payload": {
    "command": "docker ps",
    "result": {
      "stdout": "CONTAINER ID   IMAGE...",
      "stderr": "",
      "exitCode": 0
    }
  },
  "topic": "command-exec",
  "responseTime": 120
}
```

## Function Node Patterns

### Host Command Execution
```javascript
// Function node for host command execution
const exec = require('child_process').exec;
const command = msg.payload.command || 'echo "No command provided"';

exec(command, (error, stdout, stderr) => {
    if (error) {
        msg.payload = {
            status: 'error',
            error: error.message,
            exitCode: error.code,
            stderr: stderr
        };
    } else {
        msg.payload = {
            status: 'success',
            stdout: stdout.trim(),
            stderr: stderr,
            exitCode: 0
        };
    }
    node.send(msg);
});
```

### Docker Integration
```javascript
// Function node for Docker management
const exec = require('child_process').exec;
const action = msg.payload.action; // start, stop, restart, ps
const container = msg.payload.container || '';

const command = `docker ${action} ${container}`.trim();

exec(command, (error, stdout, stderr) => {
    msg.payload = {
        command: command,
        success: !error,
        output: stdout,
        error: error ? error.message : null,
        timestamp: new Date().toISOString()
    };
    node.send(msg);
});
```

### Resource Monitoring
```javascript
// Function node for resource health checking
const http = require('http');

const checkResource = (url, name) => {
    return new Promise((resolve) => {
        const req = http.get(url, (res) => {
            resolve({
                name: name,
                status: res.statusCode === 200 ? 'healthy' : 'unhealthy',
                statusCode: res.statusCode,
                responseTime: Date.now() - start
            });
        });
        
        req.on('error', () => {
            resolve({
                name: name,
                status: 'unreachable',
                error: 'Connection failed'
            });
        });
        
        req.setTimeout(5000, () => {
            req.destroy();
            resolve({
                name: name,
                status: 'timeout',
                error: 'Request timeout'
            });
        });
    });
};

const start = Date.now();
const resources = [
    {name: 'ollama', url: 'http://ollama:11434'},
    {name: 'n8n', url: 'http://n8n:5678'}
];

Promise.all(resources.map(r => checkResource(r.url, r.name)))
    .then(results => {
        msg.payload = {
            timestamp: new Date().toISOString(),
            resources: results,
            overall: results.every(r => r.status === 'healthy') ? 'healthy' : 'degraded'
        };
        node.send(msg);
    });
```

## Error Handling

### Standard Error Response
```json
{
  "error": true,
  "message": "Error description",
  "code": "ERROR_CODE",
  "timestamp": "2024-01-01T12:00:00Z",
  "context": {
    "flow": "flow-id",
    "node": "node-id"
  }
}
```

### Common Error Codes
- `FLOW_NOT_FOUND`: Requested flow doesn't exist
- `COMMAND_FAILED`: Host command execution failed
- `DOCKER_UNAVAILABLE`: Docker socket not accessible
- `TIMEOUT`: Operation exceeded timeout limit
- `INVALID_INPUT`: Input validation failed

## Authentication

Node-RED can be configured with authentication in `settings.js`:

```javascript
adminAuth: {
    type: "credentials",
    users: [{
        username: "admin",
        password: "$2a$08$zZWtXTja0fB1pzD4sHCMyOCMYz2Z6dNbM6tl8sJogENOMcxWV9DN.",
        permissions: "*"
    }]
}
```

## Rate Limiting

Configure rate limiting in `settings.js`:

```javascript
httpNodeMiddleware: function(req, res, next) {
    // Custom rate limiting logic
    next();
}
```

## Best Practices

### API Usage
1. **Use manage.sh for operations** - More reliable than direct API calls
2. **Implement proper error handling** - Always handle async operation failures
3. **Set appropriate timeouts** - Prevent hanging operations
4. **Validate inputs** - Sanitize all external inputs
5. **Monitor performance** - Track response times and resource usage

### Flow Design
1. **Modular flows** - Create reusable subflows
2. **Error branches** - Always include error handling paths
3. **Status reporting** - Use status nodes for flow state indication
4. **Documentation** - Use info panels to document flow purpose

### Security
1. **Input validation** - Validate all incoming data
2. **Command whitelisting** - Restrict allowed system commands
3. **Access control** - Implement proper authentication
4. **Audit logging** - Log important operations

## Integration Examples

### With Vrooli Resources
```bash
# Test resource integration
./manage.sh --action flow-execute --endpoint "/api/resources/health"

# Execute system command via flow
./manage.sh --action flow-execute --endpoint "/api/exec" --data '{"command": "docker ps"}'
```

### With External Services
```javascript
// HTTP request to external API
[
    {
        "type": "http request",
        "method": "GET",
        "url": "https://api.example.com/status"
    },
    {
        "type": "function",
        "func": "msg.payload = {service: 'external', status: msg.payload.status};\nreturn msg;"
    }
]
```

See the [flows documentation](../flows/README.md) for complete flow examples and usage patterns.