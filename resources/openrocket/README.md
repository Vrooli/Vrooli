# OpenRocket Launch Vehicle Design Studio

## Overview

OpenRocket provides professional-grade model rocket design and flight simulation capabilities for Vrooli's aerospace scenarios. It enables rapid prototyping of launch vehicles with accurate trajectory analysis and integration points for CFD and visualization pipelines.

## Features

- **Rocket Design Management**: Import/export .ork design files
- **Flight Simulation**: Physics-based trajectory calculations
- **Atmosphere Modeling**: ISA standard atmosphere for realistic dynamics
- **Telemetry Export**: CSV and JSON output for downstream analytics
- **API Access**: REST endpoints for programmatic control
- **Integration Ready**: Works with MinIO, PostgreSQL, QuestDB

## Quick Start

### Installation
```bash
# Install OpenRocket
vrooli resource openrocket manage install

# Start the service
vrooli resource openrocket manage start --wait

# Verify health
vrooli resource openrocket status
```

### Basic Usage
```bash
# List available designs
vrooli resource openrocket content list

# Run a simulation with the example rocket
vrooli resource openrocket content execute simulate alpha-iii

# View simulation logs
vrooli resource openrocket logs
```

## API Endpoints

- `GET http://localhost:9513/health` - Health check
- `GET http://localhost:9513/api/designs` - List designs
- `POST http://localhost:9513/api/designs/{name}` - Upload design
- `POST http://localhost:9513/api/simulate` - Run simulation

## Rocket Design Examples

### Alpha III Model Rocket
Classic Estes model rocket included as example:
- Ogive nosecone for optimal aerodynamics
- 3-fin configuration for stability
- A8-3 motor specification
- Recovery system modeling

### Custom Designs
Import your own .ork files:
```bash
vrooli resource openrocket content add my-rocket.ork
```

## Simulation Parameters

Configure simulation settings via environment variables:
- `OPENROCKET_MAX_ALTITUDE`: Maximum altitude (default: 10000m)
- `OPENROCKET_TIME_STEP`: Integration step (default: 0.01s)
- `OPENROCKET_WIND_SPEED`: Wind conditions (default: 5 m/s)

## Integration Examples

### With MinIO Storage
```bash
# Designs and results automatically stored if MinIO is running
vrooli resource minio manage start
export OPENROCKET_ENABLE_MINIO=true
```

### With QuestDB Time Series
```bash
# Telemetry data streams to QuestDB if available
vrooli resource questdb manage start
export OPENROCKET_ENABLE_QUESTDB=true
```

### With Blender Visualization
```bash
# Export trajectory for 3D rendering
vrooli resource openrocket content execute simulate my-rocket
# Results available for Blender import
```

## Testing

Run comprehensive test suite:
```bash
# All tests
vrooli resource openrocket test all

# Quick health check
vrooli resource openrocket test smoke

# Integration tests
vrooli resource openrocket test integration
```

## Configuration

Default settings in `config/defaults.sh`:
- Port: 9513
- Data directory: ~/.openrocket
- Container: Docker with Java 11
- Memory limit: 2GB
- CPU limit: 2 cores

## Troubleshooting

### Container Won't Start
- Check Docker is installed: `docker --version`
- Verify port 9513 is available: `lsof -i :9513`
- Review logs: `vrooli resource openrocket logs`

### Simulation Fails
- Validate design file format (.ork)
- Check memory limits for complex rockets
- Review atmosphere model configuration

### API Not Responding
- Confirm service is running: `docker ps | grep openrocket`
- Test health endpoint: `curl http://localhost:9513/health`
- Check firewall settings

## Advanced Features

### Parameter Sweeps
Optimize rocket designs by varying parameters:
```python
# Example sweep configuration
parameters = {
    "fin_count": [3, 4, 5],
    "motor": ["A8-3", "B6-4", "C6-5"],
    "recovery_delay": [3, 4, 5]
}
```

### CFD Pipeline Integration
Export geometry for high-fidelity analysis:
1. Design in OpenRocket
2. Export to SU2 for CFD
3. Visualize in Blender
4. Import results back

### Mission Planning
Combine with other Vrooli resources:
- OpenTripPlanner for launch site selection
- GeoNode for terrain analysis
- Traccar for real-time tracking comparison

## Performance Optimization

- Use container resource limits to prevent overconsumption
- Enable result caching for repeated simulations
- Batch parameter sweeps for efficiency
- Configure appropriate simulation time steps

## Security Considerations

- Designs validated before processing
- Simulations run in sandboxed container
- No external network access during computation
- Input size limits enforced

## Contributing

OpenRocket integration follows Vrooli's v2.0 universal contract. Improvements should maintain compatibility with:
- Standard CLI interface
- Health monitoring
- Test coverage
- Documentation standards

For more information, see the [OpenRocket official documentation](https://openrocket.info).