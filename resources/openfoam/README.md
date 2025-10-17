# OpenFOAM CFD Simulation Platform

Open-source computational fluid dynamics (CFD) platform for complex fluid flow, heat transfer, and multiphysics simulation.

## Overview

OpenFOAM (Open Field Operation and Manipulation) is a comprehensive CFD toolbox that enables scenarios to perform:
- Fluid dynamics simulation (incompressible/compressible flow)
- Heat transfer and thermodynamics analysis
- Multiphase flow modeling
- Chemical reactions and combustion
- Turbulence modeling
- Mesh generation and manipulation

## Quick Start

```bash
# Install and start OpenFOAM
vrooli resource openfoam manage install
vrooli resource openfoam manage start --wait

# Create a simple cavity flow case
vrooli resource openfoam content add cavity

# Run the simulation
vrooli resource openfoam content execute cavity

# Check status
vrooli resource openfoam status
```

## Features

### Supported Solvers
- **simpleFoam**: Steady-state incompressible flow
- **pimpleFoam**: Transient incompressible flow
- **buoyantFoam**: Buoyant, turbulent flow of compressible fluids
- **interFoam**: Two incompressible, isothermal immiscible fluids
- **reactingFoam**: Chemical reactions and combustion

### Mesh Generation
- **blockMesh**: Simple geometries with hexahedral cells
- **snappyHexMesh**: Complex geometries from STL/OBJ files
- **foamyHexMesh**: Conformal Voronoi mesh generation

### Export Formats
- VTK for ParaView visualization
- EnSight format
- CSV for data analysis
- STL for geometry export

## API Endpoints

```bash
# Health check
curl http://localhost:8090/health

# Create simulation case
curl -X POST http://localhost:8090/api/case/create \
  -H "Content-Type: application/json" \
  -d '{"name": "test_case", "type": "cavity"}'

# Generate mesh
curl -X POST http://localhost:8090/api/mesh/generate \
  -H "Content-Type: application/json" \
  -d '{"case": "test_case"}'

# Run solver
curl -X POST http://localhost:8090/api/solver/run \
  -H "Content-Type: application/json" \
  -d '{"case": "test_case", "solver": "simpleFoam"}'

# Export results
curl -X POST http://localhost:8090/api/export/vtk \
  -H "Content-Type: application/json" \
  -d '{"case": "test_case"}'
```

## Configuration

### Environment Variables
```bash
OPENFOAM_PORT=8090                # API server port
OPENFOAM_MEMORY_LIMIT=4g          # Container memory limit
OPENFOAM_CPU_LIMIT=2               # Container CPU cores
OPENFOAM_DEFAULT_SOLVER=simpleFoam # Default solver
OPENFOAM_PARALLEL_CORES=4          # MPI parallel cores
```

### Resource Limits
- Memory: 2GB minimum, 4GB recommended
- CPU: 1 core minimum, 2+ recommended
- Storage: 5GB minimum, 20GB recommended

## Examples

### 1. Cavity Flow (Basic)
```bash
# Create and run classic lid-driven cavity flow
vrooli resource openfoam content add cavity_flow
vrooli resource openfoam content execute cavity_flow
```

### 2. Heat Transfer
```bash
# Create heated cavity case
vrooli resource openfoam content add heated_cavity "heat-transfer/buoyantFoam"
vrooli resource openfoam content execute heated_cavity buoyantFoam
```

### 3. Multiphase Flow
```bash
# Dam break simulation
vrooli resource openfoam content add dam_break "multiphase/interFoam"
vrooli resource openfoam content execute dam_break interFoam
```

## Integration Examples

### With Blender
```bash
# Export geometry from Blender as STL
# Place in cases directory
cp geometry.stl ~/.vrooli/openfoam/cases/my_case/constant/triSurface/

# Generate mesh from STL
vrooli resource openfoam content execute my_case snappyHexMesh
```

### With FreeCAD
```bash
# Export CAD model as STL from FreeCAD
# Use for CFD analysis
vrooli resource openfoam content add cad_analysis
# Copy STL to case directory and run
```

### With Qdrant
```bash
# Store simulation patterns for ML analysis
# Results automatically indexed if Qdrant is running
```

## Testing

```bash
# Quick health check
vrooli resource openfoam test smoke

# Full test suite
vrooli resource openfoam test all

# Specific test phase
vrooli resource openfoam test integration
```

## Troubleshooting

### Container Won't Start
```bash
# Check Docker is running
docker ps

# Check port availability
lsof -i :8090

# View logs
vrooli resource openfoam logs
```

### Solver Convergence Issues
- Reduce time step size
- Adjust relaxation factors
- Check mesh quality with `checkMesh`
- Verify boundary conditions

### Out of Memory
- Increase memory limit: `OPENFOAM_MEMORY_LIMIT=8g`
- Reduce mesh size
- Use parallel processing to distribute load

### Performance Optimization
- Enable MPI: `OPENFOAM_ENABLE_MPI=true`
- Increase parallel cores: `OPENFOAM_PARALLEL_CORES=8`
- Use coarser mesh for initial testing

## Advanced Usage

### Parallel Processing
```bash
# Enable MPI and run in parallel
export OPENFOAM_ENABLE_MPI=true
export OPENFOAM_PARALLEL_CORES=8
vrooli resource openfoam content execute large_case simpleFoam
```

### Custom Solvers
Place custom solver source in `~/.vrooli/openfoam/solvers/` and compile within container.

### Batch Processing
```bash
# Run parameter sweep
for velocity in 0.1 0.2 0.5 1.0; do
  case_name="cavity_v${velocity}"
  vrooli resource openfoam content add "$case_name"
  # Modify boundary conditions
  vrooli resource openfoam content execute "$case_name"
done
```

## Performance Metrics

- Startup time: < 30 seconds
- Simple cavity case: < 60 seconds
- 1M cell mesh generation: < 5 minutes
- Health check response: < 500ms
- Memory usage: < 4GB for standard cases

## Security Notes

- Runs in isolated Docker container
- Resource limits enforced
- No external network access by default
- Automatic cleanup of old cases

## Links

- [OpenFOAM Documentation](https://www.openfoam.com/documentation/)
- [OpenFOAM Tutorials](https://www.openfoam.com/documentation/tutorial-guide)
- [ParaView Visualization](https://www.paraview.org/)
- [CFD Online Forum](https://www.cfd-online.com/Forums/openfoam/)