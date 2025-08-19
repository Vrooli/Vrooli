# Blender Resource

Professional 3D creation suite with Python API for Vrooli automation scenarios.

## Overview

Blender provides a comprehensive 3D pipeline including:
- 3D modeling and sculpting
- Animation and rigging
- Rendering (Cycles, Eevee, Workbench)
- Compositing and video editing
- Python scripting for automation
- Geometry nodes for procedural modeling

## Quick Start

```bash
# Check status
vrooli resource blender status

# Start Blender service
vrooli resource blender start

# Inject a Python script
vrooli resource blender inject my_script.py

# Run a script
vrooli resource blender run generate_model.py

# List injected scripts
vrooli resource blender list
```

## Use Cases

### 1. Product Visualization
Generate product renders from parametric models for e-commerce scenarios.

### 2. Data Visualization
Create 3D visualizations of complex data structures and relationships.

### 3. Procedural Content Generation
Generate 3D assets for games, simulations, or virtual environments.

### 4. Animation Automation
Automate animation creation for explainer videos or presentations.

### 5. CAD Integration
Import CAD models (via OpenSCAD integration) and create photorealistic renders.

## Script Injection

Blender scripts are Python files that can:
- Create and modify 3D objects
- Set up materials and lighting
- Configure render settings
- Export to various formats (STL, OBJ, FBX, glTF)
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
- Headless rendering via background mode
- REST API wrapper for remote execution

See [docs/](docs/) for detailed documentation.