# KiCad Examples

This directory contains example KiCad projects and files that can be injected into the KiCad resource.

## Available Examples

### 1. LED Blinker Circuit (`led-blinker.kicad_sch`)
A simple Arduino-based LED blinker circuit demonstrating:
- Basic schematic with Arduino Nano
- LED with current-limiting resistor
- Standard component symbols

**To inject this example:**
```bash
resource-kicad inject examples/led-blinker.kicad_sch
```

## Usage Patterns

### Injecting Projects
```bash
# Inject a single schematic file
resource-kicad inject circuit.kicad_sch

# Inject a complete project directory
resource-kicad inject /path/to/project/

# Inject with explicit type
resource-kicad inject board.kicad_pcb project
```

### Exporting Designs
```bash
# Export to manufacturing files
resource-kicad export my-project gerber

# Export multiple formats
resource-kicad export my-project gerber,pdf,step

# Export documentation only
resource-kicad export my-project pdf,svg
```

### Managing Libraries
```bash
# Import symbol library
resource-kicad inject symbols.kicad_sym library

# Import footprint library
resource-kicad inject footprints.pretty library

# List available libraries
resource-kicad list-libraries
```

## File Types

KiCad uses several file types that can be injected:

- **`.kicad_pro`** - Project file
- **`.kicad_sch`** - Schematic file
- **`.kicad_pcb`** - PCB layout file
- **`.kicad_sym`** - Symbol library
- **`.kicad_mod`** - Footprint file
- **`.kicad_wks`** - Worksheet template

## Integration with Scenarios

KiCad can be used in scenarios for:
- Automated PCB design generation
- Circuit validation and analysis
- BOM (Bill of Materials) generation
- Manufacturing file preparation
- Design rule checking
- 3D visualization and export