# SU2 Workflow Integration Guide

## Overview
This guide documents how to integrate SU2 with other resources in the Vrooli ecosystem for complete aerospace design workflows.

## Geometry Preparation Workflows

### Blender → SU2 Workflow
Blender can be used to create and modify 3D geometries for CFD analysis.

```bash
# 1. Export geometry from Blender as STL
resource-blender export --format stl --file aircraft.stl

# 2. Convert STL to SU2 mesh format (requires mesh generation tool)
# Note: This step typically requires external tools like Gmsh or Pointwise
# Future enhancement: Add automatic mesh conversion

# 3. Import mesh into SU2
vrooli resource su2 content add aircraft.su2 mesh

# 4. Run CFD simulation
vrooli resource su2 content execute aircraft.su2 cruise_config.cfg
```

### OpenRocket → SU2 Workflow
OpenRocket designs can be analyzed in SU2 for detailed aerodynamic analysis.

```bash
# 1. Export rocket geometry from OpenRocket
resource-openrocket export --format geometry --file rocket.xml

# 2. Convert to CFD mesh (future integration)
# This requires a geometry-to-mesh converter

# 3. Analyze in SU2
vrooli resource su2 content execute rocket.su2 ascent_config.cfg

# 4. Feed results back to trajectory analysis
vrooli resource su2 integrate questdb rocket_sim_001
```

### FreeCAD → SU2 Workflow
FreeCAD provides parametric CAD modeling for precise geometry definition.

```bash
# 1. Design in FreeCAD with parametric features
resource-freecad design --parametric wing_design.fcstd

# 2. Export surface mesh
resource-freecad export --format step wing.step

# 3. Generate CFD mesh and analyze
vrooli resource su2 content add wing.su2 mesh
vrooli resource su2 content execute wing.su2 wing_analysis.cfg
```

## Data Analysis Workflows

### SU2 → QuestDB → Superset
Complete data pipeline for CFD analytics.

```bash
# 1. Run simulation
vrooli resource su2 content execute naca0012.su2 sweep.cfg

# 2. Export to time-series database
vrooli resource su2 integrate questdb sim_001

# 3. Setup visualization dashboard
vrooli resource su2 visualize superset

# 4. Access dashboard at http://localhost:8088
```

### SU2 → Qdrant → Machine Learning
Use vector search for design optimization.

```bash
# 1. Run multiple design variations
vrooli resource su2 integrate batch base.cfg parameters.txt

# 2. Store in vector database
vrooli resource su2 integrate export qdrant

# 3. Search for optimal designs
vrooli resource su2 integrate search 1.5 0.02 -0.1
# Returns similar designs with CL=1.5, CD=0.02, CM=-0.1
```

## Optimization Workflows

### Multi-Disciplinary Optimization
Combine CFD with structural analysis.

```bash
# 1. CFD analysis in SU2
vrooli resource su2 content execute wing.su2 cfd.cfg

# 2. Export pressure distribution
vrooli resource su2 content get sim_001 --format csv > pressure.csv

# 3. Import to structural solver (future integration)
# resource-calculix import pressure.csv
# resource-calculix solve structure.inp

# 4. Iterate based on coupled results
```

### Adjoint-Based Shape Optimization
Use SU2's adjoint solver for gradient-based optimization.

```bash
# 1. Run direct flow solution
vrooli resource su2 content execute airfoil.su2 direct.cfg

# 2. Run adjoint solution (future enhancement)
# vrooli resource su2 content execute airfoil.su2 adjoint.cfg

# 3. Compute gradients and deform mesh
# vrooli resource su2 optimize gradient airfoil.su2

# 4. Iterate until convergence
```

## Visualization Workflows

### Real-Time Monitoring
Stream live simulation data to dashboards.

```bash
# 1. Setup telemetry
vrooli resource su2 visualize openmct

# 2. Start simulation with streaming
vrooli resource su2 visualize stream sim_001 &
vrooli resource su2 content execute naca0012.su2 config.cfg

# 3. View live data at http://localhost:8080/openmct
```

### Post-Processing Visualization
Generate visualizations for analysis.

```bash
# 1. Generate VTK files
vrooli resource su2 visualize vtk sim_001

# 2. Create convergence animation
vrooli resource su2 visualize animate sim_001

# 3. Open in ParaView or similar
# paraview results/sim_001/flow.vtk
```

## Batch Processing Workflows

### Parameter Sweep Studies
Analyze multiple configurations systematically.

```bash
# 1. Create parameter file
cat > sweep_params.txt << EOF
# AOA, Mach, CFL
0.0, 0.3, 5.0
2.0, 0.3, 5.0
4.0, 0.3, 5.0
6.0, 0.3, 5.0
EOF

# 2. Run batch optimization
vrooli resource su2 integrate batch base.cfg sweep_params.txt

# 3. Export all results
vrooli resource su2 integrate export all

# 4. Analyze in dashboard
vrooli resource su2 visualize superset
```

## Integration with Other Resources

### PostgreSQL
Store simulation metadata and results.

```bash
# Store in PostgreSQL (if available)
psql -h localhost -U postgres -d vrooli << EOF
CREATE TABLE su2_simulations (
    id VARCHAR(50) PRIMARY KEY,
    mesh VARCHAR(100),
    config VARCHAR(100),
    cl FLOAT,
    cd FLOAT,
    cm FLOAT,
    iterations INT,
    timestamp TIMESTAMP
);
EOF

# Future: Automatic PostgreSQL integration
```

### MinIO
Store large result files and visualizations.

```bash
# Upload results to MinIO (future enhancement)
# mc cp -r results/sim_001 minio/su2-results/
```

### Redis
Cache frequently accessed results.

```bash
# Cache recent results (future enhancement)
# redis-cli SET su2:sim_001:cl 1.234
# redis-cli GET su2:sim_001:cl
```

## Best Practices

1. **Mesh Quality**: Always validate mesh quality before running simulations
2. **Configuration**: Start with proven configurations and modify incrementally
3. **Convergence**: Monitor residuals and ensure proper convergence
4. **Data Management**: Use QuestDB for time-series, Qdrant for design exploration
5. **Visualization**: Generate VTK files for detailed post-processing
6. **Optimization**: Use batch processing for parameter studies
7. **Integration**: Leverage other resources for complete workflows

## Future Enhancements

- [ ] Direct mesh generation from CAD
- [ ] Automatic adjoint optimization loops
- [ ] GPU acceleration for large simulations
- [ ] Machine learning surrogate models
- [ ] Direct coupling with structural solvers
- [ ] Automated report generation

## Support

For issues or questions about SU2 integration workflows:
- Check the main README.md for basic usage
- Review example configurations in `examples/`
- Consult SU2 documentation at https://su2code.github.io/