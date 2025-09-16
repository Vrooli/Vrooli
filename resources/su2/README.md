# SU2 Aerospace CFD & Optimization Platform

SU2 is an open-source suite for computational fluid dynamics (CFD), adjoint optimization, and multi-physics analysis. This resource provides containerized access to SU2 with MPI support for parallel processing.

## Quick Start

```bash
# Install SU2 and dependencies
vrooli resource su2 manage install

# Start the service
vrooli resource su2 manage start --wait

# Run NACA0012 test case
vrooli resource su2 content execute naca0012.su2 test.cfg

# Check results
vrooli resource su2 content list results
```

## Features

### Core Capabilities
- **Flow Solvers**: Euler, Navier-Stokes, RANS turbulence models
- **Adjoint Solvers**: Continuous and discrete adjoint for optimization
- **Multi-Physics**: Heat transfer, elasticity, fluid-structure interaction
- **MPI Parallel**: Scale computations across multiple processors
- **Optimization**: Shape, topology, and multi-objective optimization

### Supported Formats
- **Mesh Formats**: SU2 native (.su2), CGNS, Plot3D
- **Output Formats**: Tecplot, ParaView, VTK, CSV
- **Configuration**: Text-based .cfg files

## CLI Commands

### Lifecycle Management
```bash
# Install SU2
vrooli resource su2 manage install

# Start service
vrooli resource su2 manage start [--wait]

# Stop service
vrooli resource su2 manage stop

# Restart service
vrooli resource su2 manage restart

# Uninstall
vrooli resource su2 manage uninstall
```

### Content Operations
```bash
# List meshes, configs, and results
vrooli resource su2 content list [meshes|configs|results|all]

# Add mesh or configuration
vrooli resource su2 content add <file> [mesh|config|auto]

# Run simulation
vrooli resource su2 content execute <mesh> <config>

# Get results
vrooli resource su2 content get <simulation_id> [csv|vtk|all]

# Remove content
vrooli resource su2 content remove <item>
```

### Testing
```bash
# Run smoke tests (< 30s)
vrooli resource su2 test smoke

# Run integration tests (< 120s)
vrooli resource su2 test integration

# Run all tests
vrooli resource su2 test all
```

### Status and Monitoring
```bash
# Check service status
vrooli resource su2 status [--json]

# View logs
vrooli resource su2 logs [lines]

# Get runtime info
vrooli resource su2 info [--json]
```

## API Endpoints

The SU2 service exposes a REST API on port 9514:

### Health Check
```bash
curl http://localhost:9514/health
```

### List Designs
```bash
curl http://localhost:9514/api/designs
```

### Submit Simulation
```bash
curl -X POST http://localhost:9514/api/simulate \
  -H "Content-Type: application/json" \
  -d '{"mesh":"naca0012.su2","config":"naca0012.cfg"}'
```

### Get Results
```bash
curl http://localhost:9514/api/results/<simulation_id>
```

### Convergence History
```bash
curl http://localhost:9514/api/results/<simulation_id>/convergence
```

## Example Workflows

### Basic CFD Analysis
```bash
# 1. Add your mesh
vrooli resource su2 content add my_geometry.su2 mesh

# 2. Add configuration
vrooli resource su2 content add my_config.cfg config

# 3. Run simulation
vrooli resource su2 content execute my_geometry.su2 my_config.cfg

# 4. Export results
vrooli resource su2 content get sim_<id> csv > results.csv
```

### Shape Optimization
```bash
# 1. Prepare optimization config
cat > optimization.cfg << EOF
SOLVER= RANS
OBJECTIVE_FUNCTION= DRAG
DESIGN_VARIABLES= 20
EOF

# 2. Add to SU2
vrooli resource su2 content add optimization.cfg

# 3. Run optimization
vrooli resource su2 content execute airfoil.su2 optimization.cfg
```

### Parallel Execution
The resource automatically uses MPI for parallel processing. Default is 4 processes, configurable via:
```bash
export SU2_MPI_PROCESSES=8
vrooli resource su2 manage restart
```

## Configuration

### Environment Variables
```bash
SU2_PORT=9514                    # API port
SU2_MPI_PROCESSES=4             # Number of MPI processes
SU2_MAX_ITERATIONS=1000         # Maximum solver iterations
SU2_CONVERGENCE_CRITERIA=1e-6   # Convergence threshold
SU2_MEMORY_LIMIT=4G             # Container memory limit
SU2_CPU_LIMIT=4                 # CPU core limit
```

### Data Directories
- **Meshes**: `~/.vrooli/su2/meshes/`
- **Configs**: `~/.vrooli/su2/configs/`
- **Results**: `~/.vrooli/su2/results/`
- **Cache**: `~/.vrooli/su2/cache/`

## Integration

### With Other Resources
- **OpenRocket**: Export rocket geometries for CFD validation
- **Blender**: Generate complex 3D meshes for analysis
- **QuestDB**: Store simulation time-series data
- **Qdrant**: Vector storage for design optimization
- **OpenMCT**: Real-time visualization of convergence

### Example Integration
```bash
# Generate mesh in Blender
vrooli resource blender content execute generate_mesh.py

# Transfer to SU2
cp ~/.vrooli/blender/output/mesh.su2 ~/.vrooli/su2/meshes/

# Run CFD analysis
vrooli resource su2 content execute mesh.su2 config.cfg

# Store results in QuestDB
vrooli resource questdb content import ~/.vrooli/su2/results/latest
```

## Troubleshooting

### Service Won't Start
```bash
# Check Docker status
docker ps -a | grep su2

# View logs
vrooli resource su2 logs 100

# Rebuild image
docker rmi vrooli/su2:latest
vrooli resource su2 manage install
```

### Simulation Fails
```bash
# Check convergence history
curl http://localhost:9514/api/results/<id>/convergence

# Common fixes:
# - Reduce CFL number in config
# - Increase max iterations
# - Check mesh quality
# - Verify boundary conditions
```

### Memory Issues
```bash
# Increase memory limit
export SU2_MEMORY_LIMIT=8G
vrooli resource su2 manage restart

# Use mesh partitioning for large cases
# Reduce MPI processes if needed
```

## Examples

### NACA0012 Airfoil
Pre-configured test case included:
```bash
vrooli resource su2 content execute naca0012.su2 test.cfg
```

### Custom Cases
1. Prepare your mesh using mesh generation tools
2. Create configuration file based on templates
3. Upload to SU2 resource
4. Run simulation
5. Post-process results

## Performance

- Meshes up to 10M cells supported
- Scales to 32 MPI processes
- Typical convergence in 100-1000 iterations
- Memory usage proportional to mesh size

## Security

- Isolated Docker container
- Limited resource allocation
- No direct file system access
- API rate limiting available

## Support

For issues or questions:
- Check logs: `vrooli resource su2 logs`
- View status: `vrooli resource su2 status`
- See PRD: `resources/su2/PRD.md`

## References

- [SU2 Documentation](https://su2code.github.io)
- [SU2 GitHub](https://github.com/su2code/SU2)
- [Tutorial Collection](https://su2code.github.io/tutorials/home/)