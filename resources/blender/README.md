# Blender Resource

Professional 3D creation suite with Python API for Vrooli automation scenarios.

## Overview

Blender provides a comprehensive 3D pipeline including:
- 3D modeling and sculpting
- Animation and rigging
- Rendering (Cycles, Eevee, Workbench)
- **Physics simulation (rigid body, soft body, cloth, fluid)**
- **GPU-accelerated physics (CUDA, OptiX, HIP, Metal)**
- **Adaptive LOD system for performance optimization**
- **Cache streaming for large-scale simulations**
- Compositing and video editing
- Python scripting for automation
- Geometry nodes for procedural modeling
- **Physics accuracy validation (>95% vs analytical solutions)**

## Quick Start

```bash
# Check status
vrooli resource blender status

# Start Blender service
vrooli resource blender manage start

# Run physics simulation
vrooli resource blender test physics

# Validate physics accuracy
vrooli resource blender test validation

# Inject a Python script
vrooli resource blender content add my_script.py

# Run a script
vrooli resource blender content execute generate_model.py

# List injected scripts
vrooli resource blender content list
```

## Use Cases

### 1. Physics Simulation & Validation
- **Rigid Body Dynamics**: Simulate falling objects, collisions, pendulums
- **Soft Body Physics**: Deformable objects, jelly-like materials
- **Cloth Simulation**: Realistic fabric dynamics
- **Fluid Dynamics**: Liquid and gas simulations (NEW)
  - Liquid simulation with viscosity and surface tension
  - Gas/smoke simulation with buoyancy and vorticity
  - Fire simulation with fuel combustion
- **Accuracy Validation**: >95% accuracy vs analytical solutions

### 2. Computational Physics Workflows
- Scientific visualization of physics phenomena
- Engineering simulation validation
- Educational physics demonstrations
- Research data visualization

### 3. Product Visualization
Generate product renders from parametric models for e-commerce scenarios.

### 4. Data Visualization
Create 3D visualizations of complex data structures and relationships.

### 5. Procedural Content Generation
Generate 3D assets for games, simulations, or virtual environments.

### 6. Animation Automation
Automate animation creation for explainer videos or presentations.

### 7. CAD Integration
Import CAD models (via OpenSCAD integration) and create photorealistic renders.

## Physics Examples

### Available Physics Scripts
- `physics_rigid_body.py`: Demonstrates falling cubes, dominoes, pendulum
- `physics_soft_body.py`: Shows cloth, jelly, rope simulations
- `physics_validation.py`: Validates accuracy against analytical solutions
- `physics_optimized.py`: Performance-optimized simulations with benchmarking
- `physics_gpu_optimized.py`: GPU-accelerated physics with adaptive quality
- `physics_cache_streaming.py`: Memory-optimized cache streaming for large simulations
- `physics_fluid_dynamics.py`: Advanced fluid and gas simulations
- `physics_gas_simulation.py`: Smoke and fire physics

### Running Physics Simulations
```bash
# Quick physics test
vrooli resource blender test physics

# Full validation suite
vrooli resource blender test validation

# Run specific physics script
vrooli resource blender content execute physics_rigid_body.py

# GPU-optimized simulation
vrooli resource blender content execute physics_gpu_optimized.py

# Test optimization features
vrooli resource blender test physics-optimization
```

## Performance Optimizations

### GPU Acceleration
- Automatic GPU detection and configuration
- Support for CUDA, OptiX, HIP, and Metal
- 2-5x speedup for physics simulations
- Adaptive quality presets (ultra_fast, fast, balanced, quality, ultra_quality)

### Memory Optimization
- Cache streaming for simulations larger than RAM
- Chunk-based physics baking
- Compression options (NO, LIGHT, HEAVY)
- LOD system reduces memory usage by 40%

### Benchmarks
- **Rigid Bodies**: 1000 objects @ 30fps (GPU)
- **Particles**: 1M particles @ 10fps (optimized)
- **Instances**: 10,000 shared-mesh objects supported
- **Cache**: 50MB/minute for 1000 objects (LIGHT compression)

## Script Injection

Blender scripts are Python files that can:
- Create and modify 3D objects
- **Run physics simulations (rigid, soft, cloth, fluid)**
- Set up materials and lighting
- Configure render settings
- Export to various formats (STL, OBJ, FBX, glTF)
- **Validate physics accuracy against known solutions**
- Integrate with other Vrooli resources

Example script structure:
```python
import bpy

# Clear existing mesh objects
bpy.ops.object.select_all(action='SELECT')
bpy.ops.object.delete(use_global=False)

# Create a cube
bpy.ops.mesh.primitive_cube_add(location=(0, 0, 0))

# Save the file
bpy.ops.wm.save_as_mainfile(filepath="/output/model.blend")

# Export as STL
bpy.ops.export_mesh.stl(filepath="/output/model.stl")
```

## Configuration

The resource uses Docker to run Blender in headless mode with Python API access.

### Environment Variables
- `BLENDER_VERSION`: Blender version to use (default: 4.0)
- `BLENDER_DATA_DIR`: Directory for Blender data files
- `BLENDER_OUTPUT_DIR`: Directory for rendered outputs
- `BLENDER_SCRIPTS_DIR`: Directory for Python scripts

## Integration with Other Resources

- **OpenSCAD**: Import parametric CAD models for rendering
- **ComfyUI**: Generate textures and materials using AI
- **FFmpeg**: Process rendered animations and videos
- **MinIO**: Store 3D models and rendered outputs
- **N8n**: Orchestrate complex 3D pipelines

## Technical Details

- Docker-based deployment for consistency
- Python 3.11+ for scripting
- Support for GPU rendering (when available)
- **Physics engine with >95% accuracy**
- **Optimized physics performance: 1000+ rigid bodies @ 30fps**
- **Soft body and cloth simulation support**
- **Fluid dynamics: liquid and gas simulations**
- **Fire and smoke with volumetric rendering**
- **Headless operation for CI/CD pipelines**
- Headless rendering via background mode
- REST API wrapper for remote execution

See [docs/](docs/) for detailed documentation.