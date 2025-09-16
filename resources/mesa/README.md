# Mesa Agent-Based Simulation Framework

Mesa is a Python-based agent-based modeling framework for building, analyzing, and visualizing complex adaptive systems. This resource provides Mesa as a service with REST API access for running simulations, parameter sweeps, and exporting results.

## Quick Start

```bash
# Install Mesa
vrooli resource mesa manage install

# Start the service
vrooli resource mesa manage start --wait

# Check status
vrooli resource mesa status

# Run example simulation
vrooli resource mesa content execute schelling

# Run tests
vrooli resource mesa test all
```

## Architecture

Mesa runs as a FastAPI-based REST service providing:

- **Agent-Based Modeling**: Build complex systems with autonomous agents
- **Canonical Examples**: Simple demo model (always available), plus Schelling/virus models when Mesa installed
- **Batch Simulation**: Parameter sweeps for experimentation with parallel runs
- **Deterministic Execution**: Reproducible results with seed control
- **Metrics Export**: JSON export of simulation data with automatic file storage
- **State Snapshots**: Capture simulation state at intervals
- **Redis Integration**: Automatic result storage when Redis is available
- **Graceful Degradation**: Works in simulation mode without full Mesa installation

## Core Capabilities

### Available Models

1. **Simple Demonstration Model** (Always Available)
   - Basic agent happiness simulation
   - Tests framework functionality
   - Parameters: n_agents, width, height

2. **Schelling Segregation** (Requires Mesa)
   - Classic model of residential segregation
   - Shows how mild preferences lead to strong segregation
   - Parameters: density, minority_fraction, homophily

3. **Virus on Network** (Requires Mesa)
   - Epidemiological model of disease spread
   - Network-based transmission dynamics
   - Parameters: nodes, infection_rate, recovery_rate

4. **Forest Fire** (Future)
   - Fire spread through forest grid
   - Environmental simulation

5. **Wealth Distribution** (Future)
   - Economic inequality emergence
   - Trading and resource distribution

### API Endpoints

- `GET /health` - Service health check
- `GET /models` - List available models
- `POST /simulate` - Run single simulation
- `POST /batch` - Run parameter sweep
- `GET /results` - List simulation results
- `GET /metrics/latest` - Get latest metrics

### Example Usage

```bash
# Run Schelling model with custom parameters
curl -X POST http://localhost:9512/simulate \
  -H "Content-Type: application/json" \
  -d '{
    "model": "schelling",
    "steps": 100,
    "seed": 42,
    "parameters": {
      "density": 0.8,
      "homophily": 3
    }
  }'

# Run batch simulation with parameter sweep
curl -X POST http://localhost:9512/batch \
  -H "Content-Type: application/json" \
  -d '{
    "model": "virus",
    "parameters": {
      "infection_rate": [0.2, 0.3, 0.4],
      "recovery_rate": [0.1, 0.2]
    },
    "steps": 50,
    "runs": 3
  }'
```

## Configuration

Mesa uses environment variables for configuration:

- `MESA_PORT` - API port (default: 9512)
- `MESA_DEFAULT_STEPS` - Default simulation steps (100)
- `MESA_DEFAULT_SEED` - Default random seed (42)
- `MESA_MAX_AGENTS` - Maximum agents per simulation (10000)
- `MESA_EXPORT_PATH` - Path for exported data
- `MESA_METRICS_PATH` - Path for metrics output

## Integration with Vrooli

Mesa integrates with other Vrooli resources:

- **Redis**: Store simulation results for caching (✅ Implemented, auto-detects)
- **Qdrant**: Save simulation patterns for reuse (Future)
- **PostgreSQL**: Persist experiment data (Future)
- **Open MCT**: Visualize real-time simulations (Future)
- **ElectionGuard**: Model voting behavior (Future)
- **OpenTripPlanner**: Simulate mobility patterns (Future)

## Testing

```bash
# Quick health check
vrooli resource mesa test smoke

# Full functionality test
vrooli resource mesa test integration

# Library unit tests
vrooli resource mesa test unit

# Run all tests
vrooli resource mesa test all
```

## Troubleshooting

### Service Won't Start
- Check port 9512 is available
- Verify Python 3.9+ is installed
- Check virtual environment creation

### Simulations Fail
- Verify model parameters are valid
- Check memory limits aren't exceeded
- Review logs: `vrooli resource mesa logs`

### Non-Deterministic Results
- Ensure seed parameter is set
- Verify no external randomness sources
- Check model implementation

## Performance

- Startup time: ~20-30s
- Health check: <500ms
- Small simulation (100 agents, 100 steps): <5s
- Large simulation (10000 agents, 1000 steps): <60s
- Memory usage: ~200MB idle, up to 512MB active

## Development

### Adding Custom Models

1. Create model file in `examples/`
2. Implement Mesa Model and Agent classes
3. Register in API models dictionary
4. Add tests for deterministic execution

### API Extension

The API can be extended with:
- WebSocket support for live updates
- GPU acceleration for large populations
- Distributed simulation across nodes
- Advanced visualization endpoints

## Security

- No authentication required (local use only)
- Input validation on all parameters
- Resource limits enforced
- Sandboxed execution environment

## License

Mesa is licensed under Apache 2.0. This resource wrapper is part of Vrooli.

## Support

For issues or questions:
- Mesa documentation: https://mesa.readthedocs.io
- Vrooli support: See main documentation