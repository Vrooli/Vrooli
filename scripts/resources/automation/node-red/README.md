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

## 🔗 Documentation

- **[Integration Examples & Commands](docs/EXAMPLES.md)** - Detailed usage examples and management commands
- **[Troubleshooting Guide](docs/TROUBLESHOOTING.md)** - Solutions for common issues and problems
- **[Node-RED Official Documentation](https://nodered.org/docs/)** - Complete Node-RED reference
- **[Node-RED Flows Library](https://flows.nodered.org/)** - Community flow repository
- **[Node-RED Cookbook](https://cookbook.nodered.org/)** - Practical recipes and patterns

---

**Node-RED Resource** - Part of the Vrooli automation platform ecosystem. For support and contributions, see the main Vrooli documentation.