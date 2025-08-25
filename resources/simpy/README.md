# SimPy - Discrete Event Simulation

SimPy is a discrete event simulation framework for modeling complex systems, workflows, and processes. It's perfect for scenario planning, optimization, and understanding system dynamics.

## Quick Start

```bash
# Install SimPy
vrooli resource simpy install

# Check status
vrooli resource simpy status

# Run example simulation
vrooli resource simpy run examples/bank_queue.py

# List available examples
vrooli resource simpy list-examples

# Run resource pool simulation to analyze bottlenecks
resource-simpy run examples/resource_pool.py

# Use the API to run custom simulations
curl -X POST http://localhost:9510/simulate \
  -H "Content-Type: application/json" \
  -d '{"name": "bank_queue", "parameters": {"customers": 20}}'
```

## Architecture

SimPy runs as a lightweight Python service that:
- Provides discrete event simulation capabilities
- Models complex workflows and resource allocation
- Supports process interactions and queuing systems
- Enables what-if scenario analysis

## Use Cases

- **Workflow Optimization**: Model and optimize automation workflows
- **Resource Planning**: Simulate resource utilization and bottlenecks
- **Scenario Analysis**: Test different configurations before deployment
- **Queue Management**: Model and optimize queuing systems
- **Performance Prediction**: Predict system behavior under various loads

## Available Examples

- **bank_queue.py**: Simulates customer service at a bank with multiple tellers
- **machine_shop.py**: Models a production system with machines and maintenance
- **resource_pool.py**: Models shared resource allocation (agents, servers, GPUs) with bottleneck analysis
- **basic_queue.py**: Simple queuing system demonstration

## API Endpoints

- `GET /health` - Service health check
- `POST /simulate` - Run a simulation with custom code
- `GET /examples` - List available example simulations

## Documentation

- [Installation Guide](docs/installation.md)
- [API Reference](docs/api.md)
- [Examples](examples/)

## Testing

Integration tests are available in the `test/` directory:

```bash
# Run integration tests
bats resources/simpy/test/integration.bats
```

## Integration with Vrooli

SimPy integrates with Vrooli to:
- Model scenario execution patterns
- Optimize resource allocation
- Predict system performance
- Simulate complex workflows before implementation