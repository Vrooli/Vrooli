# SimPy - Discrete Event Simulation

SimPy is a discrete event simulation framework for modeling complex systems, workflows, and processes. It's perfect for scenario planning, optimization, and understanding system dynamics.

## Quick Start

```bash
# Install SimPy
vrooli resource simpy install

# Check status
vrooli resource simpy status

# Run example simulation
vrooli resource simpy run examples/basic_queue.py
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

## Documentation

- [Installation Guide](docs/installation.md)
- [API Reference](docs/api.md)
- [Examples](examples/)

## Integration with Vrooli

SimPy integrates with Vrooli to:
- Model scenario execution patterns
- Optimize resource allocation
- Predict system performance
- Simulate complex workflows before implementation