# GridLAB-D Resource

GridLAB-D is a powerful distribution-level power system simulator for smart grid technologies, enabling detailed analysis of power distribution networks, renewable energy integration, and grid resilience.

## Overview

GridLAB-D provides comprehensive power system simulation capabilities:
- **Power Flow Analysis**: Unbalanced three-phase power flow calculations
- **Smart Grid Modeling**: Advanced metering, demand response, distributed generation
- **Market Simulation**: Transactive energy and real-time pricing models
- **Resilience Planning**: Outage analysis and restoration strategies
- **DER Integration**: Solar PV, batteries, electric vehicles, microgrids
- **Time-series Integration**: Export results to QuestDB and Redis for analysis
- **Real-time Simulation**: Support for real-time and faster-than-real-time modes
- **REST API**: Full-featured API with Flask for programmatic access

## Installation

```bash
# Install GridLAB-D resource
vrooli resource gridlabd manage install

# Verify installation
vrooli resource gridlabd status
```

## Quick Start

```bash
# Start GridLAB-D service
vrooli resource gridlabd manage start

# Run example simulation
vrooli resource gridlabd content execute --model ieee13

# Check simulation results
vrooli resource gridlabd content list

# Stop service
vrooli resource gridlabd manage stop
```

## Usage Examples

### Basic Power Flow Analysis
```bash
# Execute a simple residential feeder model
vrooli resource gridlabd content add residential_feeder.glm
vrooli resource gridlabd content execute --model residential_feeder

# Get results
vrooli resource gridlabd content get results/residential_feeder_output.csv
```

### Solar PV Integration Study
```bash
# Run solar hosting capacity analysis
vrooli resource gridlabd content execute --model examples/solar_integration.glm

# Visualize voltage profiles
vrooli resource gridlabd content get plots/voltage_profile.png
```

### Market Simulation
```bash
# Execute transactive energy market simulation
vrooli resource gridlabd content execute \
  --model examples/market_clearing.glm \
  --duration 24h \
  --timestep 5min
```

## API Endpoints

The GridLAB-D resource exposes a REST API on port 9511:

- `GET /health` - Service health check
- `GET /version` - GridLAB-D version information
- `POST /simulate` - Execute simulation model
- `POST /powerflow` - Run power flow analysis
- `GET /results/{id}` - Retrieve simulation results
- `POST /validate` - Validate GLM model syntax

## Configuration

Environment variables:
- `GRIDLABD_PORT` - API service port (default: 9511)
- `GRIDLABD_DATA_DIR` - Data directory for models and results
- `GRIDLABD_LOG_LEVEL` - Logging level (DEBUG/INFO/WARNING/ERROR)
- `GRIDLABD_MAX_THREADS` - Maximum simulation threads
- `GRIDLABD_MEMORY_LIMIT` - Memory limit for simulations

## Model Examples

The resource includes several example models:

- **IEEE Test Feeders**: Standard IEEE 4, 13, 34, 37, 123 bus test systems
- **Residential Models**: Typical residential distribution feeders
- **Commercial Models**: Commercial building and campus grids
- **Microgrid Models**: Islanded and grid-connected microgrids
- **Market Models**: Transactive energy and demand response

## Integration with Vrooli

GridLAB-D integrates with other Vrooli resources:

- **QuestDB**: Store time-series simulation results
- **Qdrant**: Index and search simulation scenarios
- **Grafana**: Visualize grid metrics and power flows
- **N8n**: Automate simulation workflows
- **Judge0**: Execute custom analysis scripts

## Testing

```bash
# Run smoke tests (basic health check)
vrooli resource gridlabd test smoke

# Run integration tests
vrooli resource gridlabd test integration

# Run all tests
vrooli resource gridlabd test all
```

## Troubleshooting

### Common Issues

1. **Simulation fails to converge**
   - Check model for inconsistencies
   - Adjust solver settings
   - Reduce timestep size

2. **Out of memory errors**
   - Limit parallel simulations
   - Reduce model complexity
   - Increase memory allocation

3. **Slow performance**
   - Enable multi-threading
   - Use simplified models for initial analysis
   - Optimize recorder outputs

## Documentation

- [GridLAB-D Official Documentation](http://gridlab-d.shoutwiki.com/)
- [Power Flow Theory](docs/POWER_FLOW.md)
- [Model Development Guide](docs/MODEL_GUIDE.md)
- [API Reference](docs/API.md)

## Support

For issues or questions:
- Check the [troubleshooting guide](docs/TROUBLESHOOTING.md)
- Review [example models](examples/)
- Submit issues to the Vrooli ecosystem manager