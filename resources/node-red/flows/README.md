# Node-RED Flows for Vrooli

This directory contains Node-RED flow definitions for Vrooli resource monitoring and automation.

## 📁 Flow Files

### Core Flows

#### `vrooli-resource-monitor.json`
**Purpose**: Real-time monitoring of all Vrooli resources
- **Schedule**: Runs every 30 seconds automatically
- **API Endpoint**: `GET /api/resources/status`
- **Features**:
  - Monitors 9 Vrooli resources (AI, automation, agents, storage)
  - Health checks via HTTP requests
  - Error handling and status aggregation
  - Global context storage for dashboard access
  - Docker network-aware (uses gateway IP 172.21.0.1)

**Resources Monitored**:
- **AI**: Ollama, Whisper, ComfyUI
- **Automation**: n8n, Node-RED, Huginn  
- **Agents**: Agent-S2, Browserless
- **Storage**: MinIO

#### `vrooli-resource-dashboard.json`
**Purpose**: Visual dashboard for resource monitoring
- **Dashboard URL**: `http://localhost:1880/ui/`
- **Update Rate**: Every 10 seconds
- **Features**:
  - Summary statistics (healthy/unhealthy counts, percentages)
  - Individual resource cards with status indicators
  - Real-time status updates
  - Color-coded health indicators (green/red)
  - Error details and timestamps

### Test Flows

#### `test-basic.json`
Basic connectivity test flow
- **Endpoint**: `GET /test/hello`
- **Purpose**: Verify Node-RED is responding

#### `test-exec.json`
Command execution test flow
- **Endpoint**: `POST /test/exec`
- **Purpose**: Test host command execution capabilities
- **Allowed Commands**: ls, pwd, date, whoami, echo, cat, grep, find

#### `test-docker.json`
Docker integration test flow
- **Endpoint**: `GET /test/docker`
- **Purpose**: Verify Docker socket access
- **Response**: List of running containers

#### `default-flows.json`
Combined test flows for installation

## 🚀 Usage Instructions

### Importing Flows

```bash
# Import individual flows
./manage.sh --action flow-import --flow-file flows/vrooli-resource-monitor.json

# Import all flows
./manage.sh --action flow-import --flow-file flows/default-flows.json
```

### Accessing Monitoring

**API Access**:
```bash
# Get JSON status of all resources
curl http://localhost:1880/api/resources/status | jq .

# Example response structure:
{
  "timestamp": "2025-07-28T02:53:20.206Z",
  "totalResources": 9,
  "healthy": 7,
  "unhealthy": 2,
  "healthPercentage": 78,
  "averageResponseTime": 0,
  "resources": [
    {
      "name": "ollama",
      "category": "ai",
      "status": "healthy|unhealthy",
      "url": "http://172.21.0.1:11434/api/tags",
      "port": 11434,
      "description": "Local LLM inference",
      "responseTime": 0,
      "statusCode": 200,
      "lastChecked": "2025-07-28T02:53:20.195Z",
      "timestamp": 1753671200195,
      "response": "truncated response...",
      "error": "error message if failed"
    }
  ],
  "status": "all-healthy|partial|all-down"
}
```

**Dashboard Access**:
- Visit: http://localhost:1880/ui/
- Navigate to "Vrooli Monitor" tab
- View real-time resource status cards

**Node-RED Editor**:
- Visit: http://localhost:1880/
- View flows, debug output, and trigger manual checks

## 🔧 Flow Architecture

### Resource Monitoring Flow Structure

```
Timer (30s) → Split Resources → Set URL → HTTP Request → Process Result → Aggregate → API Response
                                                      ↓
                                                  Error Handler
```

**Key Components**:

1. **Timer Trigger**: Inject node fires every 30 seconds + manual trigger
2. **Resource Splitter**: Function node defines all resources and splits into individual messages
3. **URL Setter**: Sets dynamic URL from resource data (preserves metadata)
4. **HTTP Request**: Makes health check requests with 5s timeout
5. **Success/Error Processing**: Handles responses and creates standardized result objects
6. **Aggregation**: Join node collects all results with 10s timeout
7. **Summary Formatter**: Creates final summary with statistics
8. **API Endpoint**: HTTP response node serves results as JSON

### Dashboard Flow Structure

```
Timer (10s) → Get Global Status → Split → Resource Cards
                     ↓
               Summary Display
```

**Key Components**:

1. **Dashboard Timer**: Updates every 10 seconds
2. **Status Retrieval**: Gets monitoring data from global context
3. **UI Templates**: Angular-based templates for summary and resource cards
4. **Groups & Tabs**: Organized dashboard layout

## 🌐 Docker Networking Considerations

### Network Configuration
- **Node-RED Container**: Runs on `vrooli-network` (172.21.0.0/16)
- **Gateway IP**: `172.21.0.1` (Docker host)
- **Container IP**: `172.21.0.2` (Node-RED)

### Resource Access Patterns

**Host Services** (use gateway IP):
```javascript
// Services running on host (not in Docker)
"http://172.21.0.1:11434"  // Ollama
"http://172.21.0.1:8090"   // Whisper
"http://172.21.0.1:4113"   // Agent-S2
```

**Container Services** (use container names):
```javascript
// Services in same Docker network
"http://comfyui:8188"      // ComfyUI container
"http://n8n:5678"          // n8n container
```

**Self-Reference**:
```javascript
"http://localhost:1880"    // Node-RED monitoring itself
```

## 🛠️ Customization Guide

### Adding New Resources

Edit `vrooli-resource-monitor.json`, find the `split-to-resources` function:

```javascript
const resources = [
    // ... existing resources ...
    {
        name: "new-service",
        category: "automation",
        url: "http://172.21.0.1:8080/health",
        port: 8080,
        description: "New service description"
    }
];
```

### Modifying Check Intervals

**Monitoring Frequency** (default: 30s):
- Edit the `inject-timer` node `repeat` property

**Dashboard Updates** (default: 10s):
- Edit the `dashboard-timer` node `repeat` property

### Custom Health Endpoints

Different services may need different health check URLs:
- `/health` - Standard health endpoint
- `/api/tags` - Ollama-specific
- `/pressure` - Browserless system status
- `/minio/health/live` - MinIO health check

## 📊 Integration with Vrooli

### ResourceRegistry Integration

The monitoring data can be integrated with Vrooli's ResourceRegistry:

```typescript
// In Vrooli server code
const nodeRedHealth = await fetch('http://localhost:1880/api/resources/status');
const healthData = await nodeRedHealth.json();

// Update ResourceRegistry with real-time health data
for (const resource of healthData.resources) {
    await resourceRegistry.updateHealthStatus(resource.name, {
        status: resource.status,
        lastChecked: resource.lastChecked,
        responseTime: resource.responseTime
    });
}
```

### Event Streaming

Node-RED can stream health events to Vrooli's event bus:

```javascript
// In Node-RED function node
const eventData = {
    type: 'resource-health-change',
    resource: resource.name,
    oldStatus: previousStatus,
    newStatus: resource.status,
    timestamp: new Date().toISOString()
};

// Send to Redis event stream
// (requires Redis integration in Node-RED)
```

## 🔍 Debugging and Troubleshooting

### Debug Output
- Enable debug nodes in flows to see real-time data
- Check Node-RED editor sidebar for debug messages
- Use browser developer tools on dashboard for UI issues

### Common Issues

**ECONNREFUSED Errors**:
- Check if target service is running: `docker ps | grep service-name`
- Verify network connectivity: `docker exec node-red curl http://172.21.0.1:port`
- Confirm port mappings are correct

**404 Not Found**:
- Verify health endpoint URLs are correct
- Check service documentation for proper health check paths

**Dashboard Not Loading**:
- Ensure ui_template nodes are properly configured
- Check for JavaScript errors in browser console
- Verify Node-RED dashboard module is installed

### Testing Commands

```bash
# Test monitoring API
curl -s http://localhost:1880/api/resources/status | jq '.healthy'

# Check specific resource
curl -s http://localhost:1880/api/resources/status | jq '.resources[] | select(.name=="ollama")'

# Test from within Node-RED container
docker exec node-red curl http://172.21.0.1:11434/api/tags
```

## 📈 Performance Considerations

### Resource Usage
- **Memory**: Each HTTP request creates temporary objects
- **Network**: 9 requests every 30 seconds = ~1 request/3.3s average
- **CPU**: Minimal for HTTP requests and JSON processing

### Optimization Tips
- Increase check intervals for less critical resources
- Use connection pooling for high-frequency requests
- Implement caching for stable resources
- Consider batch health check endpoints

## 🔮 Future Enhancements

### Planned Features
1. **Alerting System**: Email/Slack notifications for resource failures
2. **Historical Data**: Store health trends in database
3. **Performance Metrics**: Response time trending and analysis
4. **Auto-Recovery**: Automatic service restart for known issues
5. **Custom Dashboards**: Resource-specific monitoring views
6. **Integration APIs**: Webhooks for external monitoring systems

### Advanced Workflows
1. **Predictive Monitoring**: Use AI to predict resource failures
2. **Load Balancing**: Route traffic based on resource health
3. **Dependency Mapping**: Understand service interdependencies
4. **Capacity Planning**: Monitor resource utilization trends