# SimPy - Discrete Event Simulation

SimPy is a powerful discrete-event simulation framework for modeling complex systems, physics simulations, and process optimization. It provides essential capabilities for scenario planning, resource allocation, and predictive analytics across Vrooli's ecosystem.

## Quick Start

```bash
# Install SimPy
vrooli resource simpy manage install

# Start the service
vrooli resource simpy manage start --wait

# Check status
vrooli resource simpy status

# Access the visualization dashboard
open http://localhost:9510/dashboard

# Run tests
vrooli resource simpy test all

# Execute example simulations
vrooli resource simpy content execute basic_queue
vrooli resource simpy content execute physics_rigid_body
vrooli resource simpy content execute process_optimization

# List available examples
vrooli resource simpy content list

# Use the API to run custom simulations
curl -X POST http://localhost:9510/simulate \
  -H "Content-Type: application/json" \
  -d '{"code": "import simpy; print(simpy.__version__)"}'
```

## Architecture

SimPy runs as an enhanced Python-based REST API service with:
- **Discrete-Event Simulation**: Models complex workflows and resource allocation
- **Physics Modeling**: Simulates rigid body dynamics, collisions, and physical constraints
- **Process Optimization**: Multi-objective optimization with bottleneck analysis
- **Supply Chain Modeling**: Multi-echelon networks with inventory management
- **Real-time Monitoring**: WebSocket support for live simulation progress tracking
- **Pattern Storage**: Qdrant integration for storing and searching simulation patterns
- **Visualization Dashboard**: Interactive web interface for monitoring and analytics

## Core Capabilities

### Physics Simulations
- Rigid body dynamics with collision detection
- Gravity and force simulation
- Momentum conservation
- Energy tracking and validation
- Configurable material properties

### Process Optimization
- Multi-objective scheduling algorithms
- Resource pool management
- Bottleneck identification
- Cost optimization
- Parallel task execution

### Supply Chain Networks
- Multi-echelon modeling
- Inventory policy optimization
- Demand forecasting integration
- Service level tracking
- Bullwhip effect analysis

## Available Examples

### Basic Simulations
- **basic_queue.py**: Simple M/M/1 queue system with metrics
- **bank_queue.py**: Multi-server queuing with customer service
- **machine_shop.py**: Production system with maintenance scheduling
- **resource_pool.py**: Shared resource allocation and bottleneck analysis

### Advanced Simulations
- **physics_rigid_body.py**: Rigid body physics with collisions and gravity
- **process_optimization.py**: Complex workflow optimization with resource constraints
- **supply_chain_network.py**: Multi-echelon supply chain with inventory management
- **real_time_monitoring.py**: Factory simulation with real-time progress tracking
- **qdrant_integration.py**: Pattern storage and similarity search demonstration

## API Endpoints

### Core Endpoints
- `GET /health` - Service health check with feature status
- `GET /version` - Version information and capabilities
- `POST /simulate` - Run a simulation with custom code
- `GET /examples` - List available example simulations

### Visualization Dashboard
- `GET /dashboard` - Interactive web dashboard for visualization
- `GET /` - Redirects to dashboard

### Monitoring Endpoints
- `GET /simulations` - List active simulations
- `GET /progress/{sim_id}` - Get simulation progress events
- `WS /monitor/{sim_id}` - WebSocket connection for real-time monitoring

### Pattern Storage (when Qdrant available)
- `POST /search` - Search for similar simulation patterns

## Documentation

- [Installation Guide](docs/installation.md)
- [API Reference](docs/api.md)
- [Examples](examples/)

## Testing

SimPy includes comprehensive v2.0 contract-compliant tests:

```bash
# Run all tests
vrooli resource simpy test all

# Run specific test phases
vrooli resource simpy test smoke      # Quick health check (<30s)
vrooli resource simpy test integration # End-to-end testing (<120s)
vrooli resource simpy test unit        # Unit tests (<60s)
```

Test coverage includes:
- Health endpoint validation
- Simulation execution
- Content management
- API error handling
- Resource lifecycle

## Integration with Vrooli

SimPy integrates with Vrooli to:
- Model scenario execution patterns
- Optimize resource allocation
- Predict system performance
- Simulate complex workflows before implementation