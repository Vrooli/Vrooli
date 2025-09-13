# KiCad Resource

Electronic design automation suite for schematic capture and PCB layout.

## Overview

KiCad is a professional-grade, open-source EDA (Electronic Design Automation) suite that enables:
- **Schematic Capture**: Design electronic circuits with hierarchical sheets
- **PCB Layout**: Create multi-layer printed circuit boards
- **3D Visualization**: View and export 3D models of designs
- **Manufacturing Output**: Generate Gerber files, pick-and-place data, and drill files
- **Library Management**: Organize and reuse component symbols and footprints

## Benefits for Vrooli

KiCad enables Vrooli to:
- **Hardware Automation**: Generate and validate PCB designs programmatically
- **IoT Integration**: Design custom hardware for IoT scenarios
- **Manufacturing Prep**: Automate generation of production files
- **Cost Analysis**: Calculate BOM costs and optimize designs
- **Design Validation**: Automated DRC (Design Rule Checking)

## Supported Scenarios

- **Product Development**: Automated hardware design for products
- **Prototyping**: Rapid iteration on circuit designs
- **Education**: Interactive electronics learning
- **Manufacturing**: Production file generation and validation
- **Documentation**: Automated schematic and PCB documentation

## Quick Start

### Installation
```bash
# Install KiCad
vrooli resource install kicad

# Check status
resource-kicad status
```

### Basic Usage
```bash
# Import a project
resource-kicad inject my-circuit.kicad_sch

# List projects
resource-kicad list-projects

# Export to manufacturing files
resource-kicad export my-board gerber,pdf
```

## Architecture

```
kicad/
├── cli.sh              # CLI wrapper
├── config/
│   └── defaults.sh     # Configuration
├── lib/
│   ├── common.sh       # Shared functions
│   ├── install.sh      # Installation logic
│   ├── status.sh       # Status reporting
│   ├── inject.sh       # File injection
│   └── test.sh         # Test runner
├── test/
│   └── integration.bats # Integration tests
├── examples/           # Example circuits
└── docs/              # Documentation
```

## Data Management

### Directory Structure
- **Projects**: `$VROOLI_DATA_DIR/kicad/projects/`
- **Libraries**: `$VROOLI_DATA_DIR/kicad/libraries/`
- **Templates**: `$VROOLI_DATA_DIR/kicad/templates/`
- **Outputs**: `$VROOLI_DATA_DIR/kicad/outputs/`

### File Types
- **Schematics**: `.kicad_sch` - Circuit diagrams
- **PCB Layouts**: `.kicad_pcb` - Board designs
- **Projects**: `.kicad_pro` - Project configuration
- **Libraries**: `.kicad_sym`, `.kicad_mod` - Components

## Integration

### Python Scripting
```python
import pcbnew

# Load a board
board = pcbnew.LoadBoard("my-board.kicad_pcb")

# Get all components
for module in board.GetModules():
    print(module.GetReference(), module.GetValue())

# Export to Gerber
plot_controller = pcbnew.PLOT_CONTROLLER(board)
plot_controller.OpenPlotfile("my-board", pcbnew.PLOT_FORMAT_GERBER)
```

### Automation Workflows
KiCad can be integrated with n8n/Huginn for:
- Automated design validation
- BOM cost tracking
- Manufacturing readiness checks
- Version control integration

## Export Formats

- **gerber**: Manufacturing files for PCB fabrication
- **pdf**: Human-readable documentation
- **svg**: Vector graphics for web/print
- **step**: 3D models for mechanical CAD
- **pos**: Pick-and-place position files
- **drill**: Drill and route files

## Testing

```bash
# Run integration tests
resource-kicad test

# Test results are included in status
resource-kicad status --format json | jq '.test_status'
```

## Examples

See the `examples/` directory for sample circuits:
```bash
# View available examples
resource-kicad examples

# Import example LED blinker
resource-kicad inject examples/led-blinker.kicad_sch
```

## Troubleshooting

### Installation Issues
- **Ubuntu/Debian**: Ensure universe repository is enabled
- **macOS**: Requires Homebrew
- **Python API**: Install with `pip3 install pykicad`

### Common Problems
- **Missing libraries**: Run `resource-kicad install --force`
- **Export failures**: Check KiCad CLI is installed (`kicad-cli`)
- **Permission errors**: Check data directory permissions

For detailed troubleshooting and known limitations, see [PROBLEMS.md](./PROBLEMS.md)

## Related Resources

- **Judge0**: For circuit simulation code
- **ComfyUI**: For PCB visualization
- **Vault**: For component supplier credentials
- **PostgreSQL**: For component database