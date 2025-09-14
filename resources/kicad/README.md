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
# Install KiCad (will attempt to install actual KiCad binary if possible)
vrooli resource kicad manage install

# Check status
vrooli resource kicad status
```

**Note**: The resource will attempt to automatically install KiCad:
- **Ubuntu/Debian**: Uses APT and adds KiCad 8.0 PPA for latest version
- **macOS**: Uses Homebrew to install KiCad cask
- **Other**: Provides installation instructions and falls back to mock mode

### Basic Usage
```bash
# List projects and libraries
vrooli resource kicad content list

# Import a project
vrooli resource kicad inject my-circuit.kicad_sch

# Export to manufacturing files
vrooli resource kicad export my-board gerber,pdf
```

### Programmatic Operations
```bash
# Create Python automation scripts
vrooli resource kicad content execute create-scripts

# Place components programmatically
vrooli resource kicad content execute place-components board.kicad_pcb config.yaml

# Generate BOM (Bill of Materials)
vrooli resource kicad content execute generate-bom schematic.kicad_sch

# Run Design Rule Check
vrooli resource kicad content execute run-drc board.kicad_pcb

# Generate 3D visualization
vrooli resource kicad content execute visualize-3d board.kicad_pcb
```

### Version Control
```bash
# Initialize git repository for a project
vrooli resource kicad version init my-project

# Check git status
vrooli resource kicad version status my-project

# Commit changes
vrooli resource kicad version commit my-project "Added power supply circuit"

# View commit history
vrooli resource kicad version log my-project

# Create backup branch
vrooli resource kicad version backup my-project
```

## Architecture

```
kicad/
├── cli.sh              # CLI wrapper
├── config/
│   └── defaults.sh     # Configuration
├── lib/
│   ├── common.sh       # Shared functions
│   ├── install.sh      # Installation logic (now installs real KiCad)
│   ├── status.sh       # Status reporting
│   ├── inject.sh       # File injection
│   ├── python.sh       # Python API functions
│   ├── content.sh      # Content management
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

KiCad provides two Python APIs:
1. **Native pcbnew module** - Available when KiCad is installed
2. **Alternative libraries** (pykicad, kikit) - Work independently

#### Programmatic Component Placement
```yaml
# config.yaml - Define component positions
origin: [50, 50]  # mm offset
components:
  R1:
    location: [10, 20]  # mm
    rotation: 90        # degrees
    flipped: false
  U1:
    location: [25, 30]
    rotation: 45
```

```bash
# Apply placement configuration
vrooli resource kicad content execute place-components board.kicad_pcb config.yaml
```

#### Python API Example
```python
import pcbnew

# Load a board
board = pcbnew.LoadBoard("my-board.kicad_pcb")

# Get all components
for module in board.GetFootprints():
    print(module.GetReference(), module.GetValue())

# Move a component programmatically
module = board.FindFootprintByReference("R1")
module.SetPosition(pcbnew.VECTOR2I(10000000, 20000000))  # in nanometers
module.SetOrientation(pcbnew.EDA_ANGLE(90, pcbnew.DEGREES_T))

# Save the board
board.Save("modified-board.kicad_pcb")
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