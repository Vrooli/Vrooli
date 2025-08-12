# Node-RED Integration Examples

This document provides detailed examples of Node-RED integration patterns and use cases.

## 🔧 Integration Examples

### IoT Sensor Monitoring
```bash
# Create a flow that:
# 1. Reads temperature from MQTT sensor
# 2. Stores data in database
# 3. Displays on dashboard
# 4. Sends alerts if temperature exceeds threshold

# Import example IoT flow
./manage.sh --action flow-import --flow-file examples/sensor-monitoring.json
```

### API Integration Hub
```bash
# Create a flow that:
# 1. Polls REST API every 5 minutes
# 2. Transforms data format
# 3. Forwards to webhook
# 4. Logs all activities

# Import example API integration flow
./manage.sh --action flow-import --flow-file examples/api-integration.json
```

### Home Automation Controller
```bash
# Create a flow that:
# 1. Responds to motion sensor
# 2. Controls smart lights
# 3. Sends notifications
# 4. Provides manual override via dashboard

# Import example home automation flow
./manage.sh --action flow-import --flow-file examples/home-automation.json
```

## 📊 Management Commands

### Service Control
```bash
./manage.sh --action start           # Start Node-RED
./manage.sh --action stop            # Stop Node-RED
./manage.sh --action restart         # Restart Node-RED
./manage.sh --action status          # Show detailed status
./manage.sh --action logs            # View logs
./manage.sh --action health          # Check health status
```

### Flow Management
```bash
./manage.sh --action flow-list                                    # List all flows
./manage.sh --action flow-export --output my-flows.json           # Export flows
./manage.sh --action flow-import --flow-file flows.json           # Import flows
./manage.sh --action flow-execute --endpoint /test --data '{}'     # Execute HTTP endpoint
```

### Backup & Recovery
```bash
./manage.sh --action backup                              # Create backup
./manage.sh --action restore --backup-path backup.tar    # Restore from backup
```

### Monitoring
```bash
./manage.sh --action monitor --interval 30              # Monitor with 30s intervals
./manage.sh --action metrics                            # Show JSON metrics
./manage.sh --action stress-test --duration 60          # Run stress test
```

## 🔌 Integration with Other Resources

### With PostgreSQL
- Store sensor data and flow results
- Query databases from flows
- Trigger flows based on database events

### With MQTT Broker
- Connect IoT devices and sensors
- Publish and subscribe to topics
- Real-time device communication

### With Grafana
- Visualize data collected by Node-RED
- Create comprehensive dashboards
- Historical data analysis

### With Ollama (AI)
- Add AI capabilities to flows
- Process natural language in flows
- Intelligent decision making

### With Browserless
- Automate web interactions
- Screen scraping and testing
- Generate web-based reports

## 🧪 Testing & Examples

### Run Integration Tests
```bash
# Full test suite
./test/integration-test.sh

# Specific test categories
./manage.sh --action test
```

### Example Flows
All example flows are located in the `examples/` directory:

- **`default-flows.json`**: Basic starter flows with examples
- **`sensor-monitoring.json`**: IoT sensor data collection
- **`api-integration.json`**: REST API polling and transformation
- **`home-automation.json`**: Smart home control flows
- **`dashboard-examples.json`**: Interactive dashboard components

### Testing Your Flows
```bash
# Test HTTP endpoints created in flows
curl -X POST http://localhost:1880/test-endpoint -d '{"test": "data"}'

# View flow execution logs
./manage.sh --action logs --follow yes

# Check flow health
curl http://localhost:1880/flows
```

## 📚 Advanced Usage

### Custom Dockerfile
The resource includes a custom Dockerfile with additional tools and host access capabilities:

```dockerfile
FROM nodered/node-red:latest
USER root
RUN apk add --no-cache curl jq docker
USER node-red
```

### Host System Access
With custom image enabled, flows can:
- Execute shell commands on the host
- Access Docker socket for container management  
- Read/write host filesystem
- Monitor system resources

### API Programming
```javascript
// Example function node for advanced processing
const result = msg.payload.data
  .filter(item => item.temperature > 25)
  .map(item => ({
    sensor: item.id,
    reading: item.temperature,
    timestamp: new Date().toISOString()
  }));

msg.payload = {
  count: result.length,
  readings: result
};

return msg;
```