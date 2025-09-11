# FreeCAD Resource

Open-source parametric 3D CAD modeler with Python API for programmatic design generation, engineering analysis, and manufacturing workflows.

## Quick Start

```bash
# Install FreeCAD resource
vrooli resource freecad manage install

# Start FreeCAD service
vrooli resource freecad manage start --wait

# Check status
vrooli resource freecad status

# Run a Python CAD script
vrooli resource freecad content execute --file examples/parametric_box.py

# Export to STEP format
vrooli resource freecad content export --input model.FCStd --format STEP
```

## Capabilities

### Modeling
- **Parametric Design**: Constraint-based modeling with full history
- **Solid Modeling**: Boolean operations, extrusions, sweeps, lofts
- **Surface Modeling**: NURBS surfaces and complex geometries
- **Assembly**: Multi-part assemblies with constraints

### Analysis
- **FEM**: Finite Element Analysis for structural simulations
- **CFD**: Computational Fluid Dynamics (with OpenFOAM)
- **Kinematics**: Motion simulation and analysis
- **Stress Analysis**: Material stress and deformation

### Manufacturing
- **CAM/CNC**: Toolpath generation for CNC machining
- **3D Printing**: STL export and slicing preparation
- **Sheet Metal**: Unfolding and flat pattern generation
- **Technical Drawings**: 2D drawings from 3D models

## Python API

FreeCAD provides comprehensive Python scripting capabilities:

```python
import FreeCAD
import Part

# Create a new document
doc = FreeCAD.newDocument("MyDesign")

# Create a parametric box
box = doc.addObject("Part::Box", "MyBox")
box.Length = 100
box.Width = 50
box.Height = 25

# Create a cylinder
cylinder = doc.addObject("Part::Cylinder", "MyCylinder")
cylinder.Radius = 10
cylinder.Height = 30

# Boolean operation
cut = doc.addObject("Part::Cut", "MyPart")
cut.Base = box
cut.Tool = cylinder

# Export to STEP
Part.export([cut], "/output/my_part.step")
```

## File Formats

### Import
- Native: FCStd (FreeCAD Standard)
- CAD: STEP, IGES, BREP, DXF, SVG
- Mesh: STL, OBJ, PLY, AMF, 3MF
- Other: IFC (architecture), VRML, SCAD

### Export
- CAD: STEP, IGES, BREP, DXF, SVG
- Mesh: STL, OBJ, PLY, AMF, 3MF
- Manufacturing: G-code, APT
- Documentation: PDF, SVG, DXF

## Integration

FreeCAD integrates well with other Vrooli resources:

- **OpenSCAD**: Import/export SCAD files
- **Blender**: Exchange mesh data for rendering
- **KiCad**: Import PCB outlines for enclosure design
- **Judge0**: Sandboxed script execution

## Configuration

Configuration is managed through environment variables:

```bash
FREECAD_PORT=8195              # API port (from port_registry.sh)
FREECAD_DISPLAY=:99            # Virtual display number
FREECAD_MEMORY_LIMIT=4096      # Memory limit in MB
FREECAD_THREAD_COUNT=4         # Number of threads
```

## Testing

```bash
# Run smoke tests
vrooli resource freecad test smoke

# Run integration tests
vrooli resource freecad test integration

# Run all tests
vrooli resource freecad test all
```

## Troubleshooting

### Common Issues

1. **OpenGL Errors**: Ensure Xvfb is running with GLX extension
2. **Memory Issues**: Adjust FREECAD_MEMORY_LIMIT for large models
3. **Import Failures**: Check file format compatibility
4. **Python Errors**: Verify script syntax and FreeCAD API usage

### Debug Mode

Enable debug logging:
```bash
export FREECAD_DEBUG=true
vrooli resource freecad manage start
```

## Examples

See the `examples/` directory for sample scripts:
- `parametric_box.py` - Basic parametric modeling
- `assembly_demo.py` - Multi-part assembly with constraints
- `fem_analysis.py` - Finite element analysis example
- `cam_toolpath.py` - CNC toolpath generation

## References

- [FreeCAD Documentation](https://wiki.freecad.org/)
- [Python API Reference](https://wiki.freecad.org/FreeCAD_API)
- [Workbench Development](https://wiki.freecad.org/Workbench_creation)