# MEEP Electromagnetic Simulation Engine

## Overview

MEEP (MIT Electromagnetic Equation Propagation) is an open-source finite-difference time-domain (FDTD) software for electromagnetic simulations. This resource provides MEEP with Python bindings, MPI support, and a REST API for integration with the Vrooli ecosystem.

## Features

- **FDTD Solver**: Full-featured electromagnetic field solver
- **Python API**: Complete scriptability via Python
- **MPI Support**: Parallel computing for large simulations  
- **REST Interface**: HTTP API for remote simulation control
- **HDF5 Export**: Industry-standard data format
- **Templates**: Pre-configured simulation examples

## Quick Start

```bash
# Install MEEP
vrooli resource meep manage install

# Start the service
vrooli resource meep manage start --wait

# Run a waveguide simulation
vrooli resource meep content execute waveguide

# Check service status
vrooli resource meep status
```

## Usage

### Basic Commands

```bash
# Lifecycle management
vrooli resource meep manage install      # Install MEEP
vrooli resource meep manage start        # Start service
vrooli resource meep manage stop         # Stop service
vrooli resource meep manage restart      # Restart service

# Testing
vrooli resource meep test smoke          # Quick health check
vrooli resource meep test integration    # Full functionality test
vrooli resource meep test all            # Run all tests

# Content management
vrooli resource meep content list        # List templates
vrooli resource meep content execute waveguide  # Run simulation
```

### Configuration

Environment variables for customization:

```bash
export MEEP_PORT=8193                    # API port
export MEEP_MPI_PROCESSES=4              # Number of MPI processes
export MEEP_DEFAULT_RESOLUTION=50        # Default grid resolution
export MEEP_GPU_ENABLED=false            # Enable GPU acceleration
```

### API Endpoints

The REST API is available at `http://localhost:8193`:

- `GET /health` - Health check
- `GET /templates` - List simulation templates
- `POST /simulation/create` - Create new simulation
- `POST /simulation/{id}/run` - Execute simulation
- `GET /simulation/{id}/status` - Check simulation status
- `GET /simulation/{id}/fields` - Download field data (HDF5)
- `GET /simulation/{id}/spectra` - Get transmission/reflection spectra
- `POST /batch/sweep` - Run parameter sweep

### Simulation Templates

Built-in templates include:

1. **Waveguide**: 2D waveguide with 90-degree bend
2. **Ring Resonator**: Coupled ring resonator
3. **Photonic Crystal**: 2D photonic crystal structure

Example using the API:

```python
import requests

# Create simulation
response = requests.post(
    "http://localhost:8193/simulation/create",
    json={"template": "waveguide", "resolution": 50}
)
sim_id = response.json()["simulation_id"]

# Run simulation
requests.post(f"http://localhost:8193/simulation/{sim_id}/run")

# Get results
spectra = requests.get(f"http://localhost:8193/simulation/{sim_id}/spectra").json()
```

## Integration with Other Resources

### Storage
- **MinIO**: Store HDF5 field data files
- **PostgreSQL**: Simulation metadata and parameters
- **QuestDB**: Time-series simulation data
- **Qdrant**: Pattern search and knowledge base

### Co-simulation
- **Elmer-FEM**: Coupled electromagnetic-thermal analysis
- **OpenEMS**: Circuit-level EM simulations
- **Blender**: 3D visualization of field data
- **FreeCAD**: Geometry generation for simulations

### Analysis
- **Pandas-AI**: Automated data analysis
- **Superset**: Dashboard visualization
- **Windmill**: Workflow automation

## Performance

- **2D Simulations**: ~1000x1000 grid in 5 minutes
- **3D Simulations**: ~100x100x100 grid in 30 minutes
- **MPI Scaling**: 80%+ parallel efficiency
- **Memory**: ~2GB per million grid points

## Troubleshooting

### Service won't start
```bash
# Check if port is in use
lsof -i :8193

# Check Docker logs
docker logs meep-server

# Verify Docker is running
docker ps
```

### Simulations run slowly
- Reduce resolution: Lower grid points per wavelength
- Enable MPI: Increase `MEEP_MPI_PROCESSES`
- Use 2D instead of 3D when possible
- Consider GPU acceleration (if available)

### Out of memory errors
- Reduce simulation size
- Increase Docker memory limit: `MEEP_MEMORY_LIMIT=8g`
- Use domain decomposition with MPI

## Advanced Usage

### Custom Simulations

Create your own simulation script:

```python
import meep as mp

# Define geometry
geometry = [
    mp.Block(
        mp.Vector3(mp.inf, 1, mp.inf),
        center=mp.Vector3(),
        material=mp.Medium(epsilon=12)
    )
]

# Define sources
sources = [
    mp.Source(
        mp.GaussianSource(0.15, fwidth=0.1),
        component=mp.Ez,
        center=mp.Vector3(-7, 0)
    )
]

# Create simulation
sim = mp.Simulation(
    cell_size=mp.Vector3(16, 8, 0),
    boundary_layers=[mp.PML(1.0)],
    geometry=geometry,
    sources=sources,
    resolution=50
)

# Run
sim.run(until=200)
```

### Parameter Sweeps

Automate parameter studies:

```bash
curl -X POST http://localhost:8193/batch/sweep \
  -H "Content-Type: application/json" \
  -d '{
    "template": "waveguide",
    "parameter": "resolution",
    "values": [20, 30, 40, 50]
  }'
```

## References

- [MEEP Documentation](https://meep.readthedocs.io/)
- [MEEP GitHub](https://github.com/NanoComp/meep)
- [FDTD Theory](https://en.wikipedia.org/wiki/Finite-difference_time-domain_method)

## Support

For issues or questions:
- Check the troubleshooting section above
- Review test output: `vrooli resource meep test all`
- Examine logs: `vrooli resource meep logs`
- Consult the PRD.md for requirements and specifications