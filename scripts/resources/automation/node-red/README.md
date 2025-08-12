# Node-RED - Visual Flow Programming for IoT and Automation

Node-RED is a low-code visual programming tool for wiring together hardware devices, APIs and online services in new and interesting ways. This resource provides enterprise-grade installation, configuration, and management of Node-RED with enhanced host system access and integration capabilities.

## 🎯 Quick Reference

- **Category**: Automation / Visual Programming
- **Port**: 1880 (Editor & Dashboard)
- **Container**: node-red-vrooli
- **Data Directory**: `./data/node-red`
- **API Docs**: [Complete API Reference](docs/API.md)
- **Status**: Production Ready ✅

## 🚀 Quick Start

### Prerequisites
- Docker installed and running
- 512MB+ RAM available  
- Port 1880 available
- (Optional) Docker socket access for advanced container management

### Installation
```bash
# Standard installation
./manage.sh --action install

# With host system access (recommended for advanced flows)
./manage.sh --action install --build-image yes

# Force reinstall if already exists
./manage.sh --action install --force yes

# Check status
./manage.sh --action status
```

### First Access
- **Editor**: http://localhost:1880
- **Dashboard**: http://localhost:1880/ui (if dashboard nodes are installed)
- **Settings**: http://localhost:1880/settings

## 🌟 Core Features

- **🎨 Visual Programming**: Drag-and-drop interface for creating flows
- **🔌 4000+ Nodes**: Extensive library of community-contributed nodes
- **📊 Built-in Dashboard**: Create interactive dashboards without coding  
- **🌐 HTTP/REST API**: Full API for flow management and execution
- **💾 Persistent Storage**: Flows and data automatically saved
- **🔒 Authentication**: Optional basic authentication support
- **🐳 Container Management**: Docker integration for advanced flows
- **📡 Real-time Communication**: WebSocket support for live updates

## 📋 When to Use Node-RED

### ✅ Perfect For:
- **IoT Data Collection**: Connect sensors, devices, and APIs
- **Home Automation**: Smart home control and monitoring  
- **API Integration**: Connect disparate systems and services
- **Prototyping**: Rapid development of automation concepts
- **Dashboard Creation**: Quick data visualization interfaces
- **Message Routing**: Transform and route data between systems

### ⚠️ Consider Alternatives For:
- **High-throughput Streaming**: Use Apache Kafka or similar
- **Complex Business Logic**: Consider n8n or custom applications
- **Enterprise Workflows**: n8n provides better scalability and features
- **Heavy Computational Tasks**: Use dedicated computing resources

## 🏗️ Architecture

### Flow-Based Programming Model
```
Input Nodes → Processing Nodes → Output Nodes
    ↓              ↓                ↓
  Sensors      Transform         APIs
  Timers        Filter        Databases  
  HTTP          Switch         Files
  MQTT         Function       Dashboard
```

### Key Components
- **Flows**: Visual programs composed of connected nodes
- **Nodes**: Individual processing units with specific functions
- **Context**: Shared data storage across flows and nodes
- **Dashboard**: Web-based UI for displaying and controlling flows
- **Runtime**: Node.js engine executing the flows

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

## 📖 Configuration

### Environment Variables
- `NODE_RED_CREDENTIAL_SECRET`: Encryption key for stored credentials
- `TZ`: Timezone setting for the container
- `NODE_RED_ENABLE_PROJECTS`: Enable project management features
- `NODE_RED_ENABLE_SAFE_MODE`: Start in safe mode (flows disabled)

### Data Persistence
- **Flows**: Stored in `/data/flows.json`
- **Credentials**: Encrypted in `/data/flows_cred.json`
- **Settings**: Configuration in `/data/settings.js`
- **Context**: Persistent data in `/data/context/`

### Custom Nodes
```bash
# Install custom nodes in container
docker exec node-red-vrooli npm install node-red-contrib-dashboard
docker exec node-red-vrooli npm install node-red-contrib-influxdb

# Restart to load new nodes
./manage.sh --action restart
```

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

## 🔧 Troubleshooting

### Common Issues

**Node-RED not accessible**
```bash
# Check if container is running
./manage.sh --action status

# Check logs for errors
./manage.sh --action logs --lines 50

# Verify port is accessible
curl http://localhost:1880
```

**Flows not loading**
```bash
# Check flows file
cat data/flows.json

# Validate JSON format
jq . data/flows.json

# Restart with safe mode
docker exec node-red-vrooli touch /data/.config.json
./manage.sh --action restart
```

**Permission errors**
```bash
# Fix data directory permissions
sudo chown -R 1000:1000 data/

# Restart container
./manage.sh --action restart
```

**Memory issues**
```bash
# Check resource usage
./manage.sh --action metrics

# Monitor real-time usage
./manage.sh --action monitor
```

For detailed troubleshooting, see [TROUBLESHOOTING.md](docs/TROUBLESHOOTING.md).

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

## 🔗 Useful Links

- [Node-RED Official Documentation](https://nodered.org/docs/)
- [Node-RED Flows Library](https://flows.nodered.org/)
- [Node-RED Cookbook](https://cookbook.nodered.org/)
- [Community Forum](https://discourse.nodered.org/)

---

**Node-RED Resource** - Part of the Vrooli automation platform ecosystem. For support and contributions, see the main Vrooli documentation.