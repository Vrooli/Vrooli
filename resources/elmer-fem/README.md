# Elmer FEM Resource

## Overview

Elmer is an open-source multiphysics finite element solver that enables coupled simulations for thermal, fluid, electromagnetic, and structural problems. This resource provides Docker-based access to Elmer's powerful simulation capabilities with MPI support for parallel processing.

## Quick Start

```bash
# Install and start the resource
vrooli resource elmer-fem manage install
vrooli resource elmer-fem manage start --wait

# Run a heat transfer example
vrooli resource elmer-fem content execute --case heat_transfer

# Check status
vrooli resource elmer-fem status
```

## Features

### Core Capabilities
- **Multiphysics Coupling**: Solve coupled thermal-fluid-electromagnetic-structural problems
- **Parallel Processing**: MPI support for distributed solving
- **Multiple Solvers**: Heat, fluid flow, elasticity, electromagnetics, and more
- **Mesh Import**: Support for GMSH, UNV, and native Elmer formats
- **Result Export**: VTK, VTU, CSV formats for visualization

### Integration Points
- **Storage**: MinIO for case files, PostgreSQL for metadata
- **Visualization**: Export to Blender, ParaView, or Superset
- **Co-simulation**: Link with OpenEMS, SimPy, GridLAB-D

## Usage Examples

### Create and Solve a Case
```bash
# Create a new heat transfer case
curl -X POST http://localhost:8192/case/create \
  -H "Content-Type: application/json" \
  -d '{"name": "my_heat_case", "type": "heat_transfer"}'

# Solve the case
curl -X POST http://localhost:8192/case/my_heat_case/solve \
  -H "Content-Type: application/json" \
  -d '{"max_iterations": 1000, "mpi_processes": 4}'

# Get results
curl http://localhost:8192/case/my_heat_case/results
```

### Parameter Sweep
```bash
# Run conductivity sweep
curl -X POST http://localhost:8192/batch/sweep \
  -H "Content-Type: application/json" \
  -d '{
    "case": "heat_transfer",
    "parameter": "conductivity",
    "values": [0.5, 1.0, 1.5, 2.0]
  }'
```

## Configuration

Environment variables:
- `ELMER_FEM_PORT`: API server port (default: 8192)
- `ELMER_MPI_PROCESSES`: Default MPI processes (default: 4)
- `ELMER_MAX_MEMORY`: Memory limit (default: 4G)
- `ELMER_MAX_ITERATIONS`: Max solver iterations (default: 1000)

## Testing

```bash
# Run all tests
vrooli resource elmer-fem test all

# Run specific test phase
vrooli resource elmer-fem test smoke      # Quick health check
vrooli resource elmer-fem test integration # Full functionality
vrooli resource elmer-fem test unit        # Library functions
```

## API Endpoints

- `GET /health` - Service health check
- `GET /version` - Version information
- `GET /cases` - List available cases
- `POST /case/create` - Create new case
- `POST /case/{id}/solve` - Execute solver
- `GET /case/{id}/results` - Get results
- `GET /case/{id}/export` - Export results
- `POST /mesh/import` - Import mesh
- `POST /batch/sweep` - Parameter sweep

## Troubleshooting

### Service won't start
- Check Docker is running: `docker ps`
- Verify port availability: `lsof -i:8192`
- Check logs: `vrooli resource elmer-fem logs`

### Solver fails
- Verify mesh is valid
- Check convergence tolerance settings
- Increase max iterations
- Review case.sif configuration

### Performance issues
- Adjust MPI processes for your system
- Increase memory allocation
- Use coarser mesh for initial tests

## Resources

- [Elmer Documentation](https://www.csc.fi/web/elmer/documentation)
- [Elmer Tutorials](https://www.nic.funet.fi/pub/sci/physics/elmer/doc/)
- [Elmer GitHub](https://github.com/ElmerCSC/elmerfem)