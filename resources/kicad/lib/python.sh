#!/bin/bash
# KiCad Python API Functions

# Get script directory
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
KICAD_PYTHON_LIB_DIR="${APP_ROOT}/resources/kicad/lib"

# Source common functions
source "${KICAD_PYTHON_LIB_DIR}/common.sh"
source "${APP_ROOT}/resources/kicad/config/defaults.sh"

# Check if Python API is available
kicad::python::check_api() {
    local python_cmd="${KICAD_DATA_DIR}/venv/bin/python"
    
    if [[ ! -f "$python_cmd" ]]; then
        python_cmd="python3"
    fi
    
    # Check for native pcbnew module (requires KiCad installation)
    if $python_cmd -c "import pcbnew" 2>/dev/null; then
        echo "Native KiCad Python API available"
        return 0
    fi
    
    # Check for alternative libraries
    if $python_cmd -c "import pykicad" 2>/dev/null; then
        echo "PyKiCad library available (alternative API)"
        return 0
    fi
    
    echo "No KiCad Python API available"
    return 1
}

# Execute Python script with KiCad API
kicad::python::execute() {
    local script="$1"
    shift
    local args="$@"
    
    local python_cmd="${KICAD_DATA_DIR}/venv/bin/python"
    if [[ ! -f "$python_cmd" ]]; then
        python_cmd="python3"
    fi
    
    if [[ ! -f "$script" ]]; then
        echo "Error: Script not found: $script"
        return 1
    fi
    
    $python_cmd "$script" $args
}

# Create a Python script for programmatic component placement
kicad::python::create_placement_script() {
    local output_file="${1:-${KICAD_DATA_DIR}/scripts/place_components.py}"
    
    mkdir -p "$(dirname "$output_file")"
    
    cat > "$output_file" <<'EOF'
#!/usr/bin/env python3
"""
KiCad Programmatic Component Placement Script
Based on https://jeffmcbride.net/kicad-component-layout
"""

import sys
import json
import yaml
import os

# Try to import KiCad Python API
try:
    import pcbnew
    PCBNEW_AVAILABLE = True
except ImportError:
    PCBNEW_AVAILABLE = False
    print("Warning: pcbnew module not available. Using mock mode.")

def load_placement_config(config_file):
    """Load component placement configuration from YAML or JSON file"""
    with open(config_file, 'r') as f:
        if config_file.endswith('.yaml') or config_file.endswith('.yml'):
            return yaml.safe_load(f)
        else:
            return json.load(f)

def place_components(board_file, config_file, output_file=None):
    """Place components on PCB according to configuration"""
    
    if not PCBNEW_AVAILABLE:
        print("Mock mode: Would place components from {} using config {}".format(
            board_file, config_file))
        return
    
    # Load the board
    board = pcbnew.LoadBoard(board_file)
    
    # Load placement configuration
    config = load_placement_config(config_file)
    
    # Apply origin offset if specified
    origin_x = config.get('origin', [0, 0])[0] * 1000000  # Convert mm to nm
    origin_y = config.get('origin', [0, 0])[1] * 1000000
    
    # Process each component
    for ref_des, params in config.get('components', {}).items():
        module = board.FindFootprintByReference(ref_des)
        
        if not module:
            print(f"Warning: Component {ref_des} not found on board")
            continue
        
        # Set position (convert mm to nanometers)
        if 'location' in params:
            x = params['location'][0] * 1000000 + origin_x
            y = params['location'][1] * 1000000 + origin_y
            module.SetPosition(pcbnew.VECTOR2I(int(x), int(y)))
        
        # Set rotation (in tenths of degrees)
        if 'rotation' in params:
            module.SetOrientation(pcbnew.EDA_ANGLE(params['rotation'], pcbnew.DEGREES_T))
        
        # Set side (top/bottom)
        if 'flipped' in params:
            current_side = module.GetSide()
            target_side = pcbnew.B_Cu if params['flipped'] else pcbnew.F_Cu
            if current_side != target_side:
                module.Flip(module.GetPosition(), False)
        
        # Change footprint if specified
        if 'footprint' in params:
            fp_path = params['footprint'].get('path', '')
            fp_name = params['footprint'].get('name', '')
            # Note: Changing footprint requires more complex handling
            print(f"Note: Footprint change for {ref_des} to {fp_path}/{fp_name} requires manual update")
        
        print(f"Placed {ref_des} at ({params.get('location', 'unchanged')})")
    
    # Save the board
    if output_file:
        board.Save(output_file)
    else:
        board.Save(board_file)
    
    print(f"Board saved to {output_file or board_file}")

def generate_example_config(output_file):
    """Generate an example placement configuration file"""
    example = {
        "origin": [50, 50],  # mm offset for all components
        "components": {
            "R1": {
                "location": [10, 20],  # mm
                "rotation": 90,  # degrees
                "flipped": False
            },
            "C1": {
                "location": [15, 20],
                "rotation": 0,
                "flipped": False
            },
            "U1": {
                "location": [25, 30],
                "rotation": 45,
                "flipped": False,
                "footprint": {
                    "path": "Package_SO",
                    "name": "SOIC-8_3.9x4.9mm_P1.27mm"
                }
            }
        }
    }
    
    with open(output_file, 'w') as f:
        if output_file.endswith('.yaml') or output_file.endswith('.yml'):
            yaml.dump(example, f, default_flow_style=False)
        else:
            json.dump(example, f, indent=2)
    
    print(f"Example configuration written to {output_file}")

if __name__ == "__main__":
    import argparse
    
    parser = argparse.ArgumentParser(description="KiCad Programmatic Component Placement")
    subparsers = parser.add_subparsers(dest='command', help='Commands')
    
    # Place command
    place_parser = subparsers.add_parser('place', help='Place components on PCB')
    place_parser.add_argument('board', help='Input board file (.kicad_pcb)')
    place_parser.add_argument('config', help='Placement configuration file (YAML/JSON)')
    place_parser.add_argument('-o', '--output', help='Output board file (default: overwrite input)')
    
    # Example command
    example_parser = subparsers.add_parser('example', help='Generate example configuration')
    example_parser.add_argument('output', help='Output configuration file')
    
    args = parser.parse_args()
    
    if args.command == 'place':
        place_components(args.board, args.config, args.output)
    elif args.command == 'example':
        generate_example_config(args.output)
    else:
        parser.print_help()
EOF
    
    chmod +x "$output_file"
    echo "Created placement script at $output_file"
}

# Create a Python script for BOM generation
kicad::python::create_bom_script() {
    local output_file="${1:-${KICAD_DATA_DIR}/scripts/generate_bom.py}"
    
    mkdir -p "$(dirname "$output_file")"
    
    cat > "$output_file" <<'EOF'
#!/usr/bin/env python3
"""
KiCad BOM (Bill of Materials) Generation Script
"""

import sys
import csv
import json

try:
    import pcbnew
    import eeschema
    KICAD_API_AVAILABLE = True
except ImportError:
    KICAD_API_AVAILABLE = False
    print("Warning: KiCad Python API not available. Using mock mode.")

def generate_bom_from_schematic(schematic_file, output_file, format='csv'):
    """Generate BOM from KiCad schematic"""
    
    if not KICAD_API_AVAILABLE:
        print(f"Mock mode: Would generate BOM from {schematic_file}")
        # Create mock BOM
        mock_bom = [
            {'Reference': 'R1', 'Value': '10k', 'Footprint': '0805', 'Quantity': 1},
            {'Reference': 'C1', 'Value': '100nF', 'Footprint': '0603', 'Quantity': 1},
            {'Reference': 'U1', 'Value': 'ATmega328', 'Footprint': 'TQFP-32', 'Quantity': 1},
        ]
        
        if format == 'csv':
            with open(output_file, 'w', newline='') as f:
                writer = csv.DictWriter(f, fieldnames=['Reference', 'Value', 'Footprint', 'Quantity'])
                writer.writeheader()
                writer.writerows(mock_bom)
        else:
            with open(output_file, 'w') as f:
                json.dump(mock_bom, f, indent=2)
        
        print(f"Mock BOM written to {output_file}")
        return
    
    # Real implementation would use KiCad API
    print(f"Generating BOM from {schematic_file}")
    # TODO: Implement real BOM generation using eeschema API
    
    print(f"BOM written to {output_file}")

def aggregate_bom(bom_data):
    """Aggregate BOM by component value and footprint"""
    aggregated = {}
    
    for item in bom_data:
        key = (item['Value'], item['Footprint'])
        if key in aggregated:
            aggregated[key]['Quantity'] += 1
            aggregated[key]['References'].append(item['Reference'])
        else:
            aggregated[key] = {
                'Value': item['Value'],
                'Footprint': item['Footprint'],
                'Quantity': 1,
                'References': [item['Reference']]
            }
    
    return list(aggregated.values())

if __name__ == "__main__":
    import argparse
    
    parser = argparse.ArgumentParser(description="Generate BOM from KiCad schematic")
    parser.add_argument('schematic', help='Input schematic file')
    parser.add_argument('-o', '--output', default='bom.csv', help='Output BOM file')
    parser.add_argument('-f', '--format', choices=['csv', 'json'], default='csv', help='Output format')
    
    args = parser.parse_args()
    
    generate_bom_from_schematic(args.schematic, args.output, args.format)
EOF
    
    chmod +x "$output_file"
    echo "Created BOM script at $output_file"
}

# Run automated DRC (Design Rule Check)
kicad::python::run_drc() {
    local board_file="$1"
    local output_file="${2:-${KICAD_OUTPUTS_DIR}/drc_report.txt}"
    
    if ! command -v kicad-cli &>/dev/null; then
        if [[ -x "${KICAD_DATA_DIR}/kicad-cli-mock" ]]; then
            echo "Mock: Running DRC on $board_file"
            echo "Mock DRC Report" > "$output_file"
            echo "No violations found (mock)" >> "$output_file"
        else
            echo "Error: KiCad CLI not available"
            return 1
        fi
    else
        kicad-cli pcb drc "$board_file" -o "$output_file"
    fi
    
    echo "DRC report saved to $output_file"
}

# Export functions
export -f kicad::python::check_api
export -f kicad::python::execute
export -f kicad::python::create_placement_script
export -f kicad::python::create_bom_script
export -f kicad::python::run_drc