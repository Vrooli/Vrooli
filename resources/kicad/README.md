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

**Installation Notes**:
- **Ubuntu/Debian**: Automatically installs via APT (may require sudo for first-time setup)
- **macOS**: Uses Homebrew to install KiCad cask
- **Mock Mode**: If KiCad cannot be installed, the resource operates in mock mode for development/testing
- **Full Installation**: For complete functionality, install KiCad manually from https://www.kicad.org/
- **Binary Detection**: Automatically detects KiCad installations in standard locations (/usr/bin, /usr/local/bin, /opt/kicad, /Applications/KiCad)

**Operating Modes**:
1. **Full Mode**: With KiCad installed - all features available
2. **Mock Mode**: Without KiCad - development/testing features only, no actual file generation

### Basic Usage
```bash
# List projects and libraries
vrooli resource kicad content list

# Import a project or library
vrooli resource kicad content add my-circuit.kicad_sch

# Export to manufacturing files
vrooli resource kicad content export my-board gerber,pdf

# Get project details
vrooli resource kicad content get my-project
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

### Cloud Backup (Minio Integration)
```bash
# Backup project to cloud storage
vrooli resource kicad backup cloud my-project

# List available backups
vrooli resource kicad backup list
vrooli resource kicad backup list my-project  # For specific project

# Restore project from backup
vrooli resource kicad backup restore my-project  # Latest backup
vrooli resource kicad backup restore my-project 20250114-120000  # Specific timestamp

# Schedule automatic backups
vrooli resource kicad backup schedule my-project daily
```

### SPICE Circuit Simulation
```bash
# Extract SPICE netlist from schematic
vrooli resource kicad simulation extract circuit.kicad_sch

# Run simulation (transient, DC, AC, or operating point)
vrooli resource kicad simulation run circuit.net tran
vrooli resource kicad simulation run circuit.net dc
vrooli resource kicad simulation run circuit.net ac

# Interactive SPICE shell (requires ngspice)
vrooli resource kicad simulation interactive circuit.net

# Generate simulation report
vrooli resource kicad simulation report my-project

# Create SPICE models library
vrooli resource kicad simulation models
```

### Auto-routing (PCB Trace Routing)
```bash
# Export board for auto-routing
vrooli resource kicad autoroute export board.kicad_pcb

# Run auto-router with options
vrooli resource kicad autoroute run board.dsn --layers 2 --via-cost 50

# Import routed board back
vrooli resource kicad autoroute import board.kicad_pcb routed.ses

# One-step optimization (fast, balanced, or quality)
vrooli resource kicad autoroute optimize board.kicad_pcb balanced

# Interactive routing assistant
vrooli resource kicad autoroute assistant board.kicad_pcb
```

## Features

### Core Capabilities (P0 - Complete)
- ✅ **Project Management**: Import, organize, and export KiCad projects
- ✅ **Library Management**: Symbol and footprint libraries
- ✅ **Manufacturing Export**: Gerber, drill files, pick & place data
- ✅ **Python API**: Programmatic control via Python scripts
- ✅ **CLI Interface**: Full command-line automation

### Enhanced Features (P1 - Complete)
- ✅ **3D Visualization**: Generate 3D renders of PCB designs
- ✅ **BOM Generation**: Automated bill of materials with cost analysis
- ✅ **Design Rule Checking**: Automated DRC validation
- ✅ **Git Integration**: Version control optimized for KiCad files

### Advanced Features (P2 - Complete)
- ✅ **Cloud Backup**: Automated backup to Minio object storage
  - Versioned backups with retention policies
  - One-command restore from cloud
  - Scheduled backup automation
  
- ✅ **SPICE Simulation**: Full circuit simulation capabilities
  - ngspice integration for analog/digital simulation
  - Pre-built component models library
  - Interactive simulation shell
  - HTML simulation reports
  
- ✅ **Auto-routing**: Intelligent PCB trace routing
  - Freerouting integration
  - Multi-layer support (2, 4, 6 layers)
  - Optimization presets (fast/balanced/quality)
  - Interactive routing assistant with recommendations

### Development Features
- ✅ **Mock Mode**: Full development capabilities without KiCad installation
- ✅ **v2.0 Contract**: Full compliance with Vrooli resource standards
- ✅ **Comprehensive Testing**: 63 tests (smoke, integration, unit)

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
│   ├── test.sh         # Test runner
│   ├── version.sh      # Git version control
│   ├── backup.sh       # Cloud backup (Minio)
│   ├── simulation.sh   # SPICE circuit simulation
│   └── autoroute.sh    # PCB auto-routing
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

## Health Monitoring

KiCad is a desktop application without a network service, so health checks verify:
- Installation status (binary availability)
- Directory structure integrity
- Configuration validity
- Python API availability (if KiCad installed)

Health is determined by the ability to execute KiCad commands, not network connectivity.

## Testing

```bash
# Run all tests
vrooli resource kicad test all

# Run specific test suites
vrooli resource kicad test smoke       # Quick health check
vrooli resource kicad test integration # Full functionality
vrooli resource kicad test unit        # Library functions

# Test results are included in status
vrooli resource kicad status --format json | jq '.test_status'
```

## Examples

See the `examples/` directory for sample circuits:
```bash
# View available examples
resource-kicad examples

# Import example LED blinker
resource-kicad inject examples/led-blinker.kicad_sch
```

## Security

The KiCad resource implements multiple security measures to ensure safe operation:

### Path Traversal Protection
- **Content Management**: The `content get` and `content remove` commands validate project names to prevent path traversal attacks
- **Simulation**: All simulation commands validate file paths are within the KiCad data directory
- **File Extensions**: Simulation requires `.net` extension for netlist files

### Input Validation
- Project names cannot contain path separators (`/`, `\`) or `..` sequences
- All paths are resolved and verified to be within allowed directories
- Double validation using both pattern matching and realpath verification

### Safe Defaults
- Mock mode operation when KiCad is not installed prevents system modifications
- All file operations are restricted to the KiCad data directory
- No execution of arbitrary code without explicit user action

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