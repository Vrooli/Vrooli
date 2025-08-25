#!/bin/bash
# KiCad Installation Functions

# Get script directory
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*/../../.." && builtin pwd)}"
KICAD_INSTALL_LIB_DIR="${APP_ROOT}/resources/kicad/lib"

# Source common functions
source "${KICAD_INSTALL_LIB_DIR}/common.sh"

# Install KiCad
kicad::install() {
    local force="${1:-false}"
    
    # Initialize directories
    kicad::init_dirs
    
    if kicad::is_installed && [[ "$force" != "true" ]]; then
        echo "KiCad is already installed. Use --force to reinstall."
        return 0
    fi
    
    echo "Installing KiCad..."
    
    # Detect OS and install accordingly
    if [[ -f /etc/debian_version ]]; then
        # Ubuntu/Debian
        echo "Installing KiCad on Ubuntu/Debian..."
        
        # Check if PPA should be added (skip if KiCad is already available)
        if ! command -v kicad &>/dev/null && ! apt-cache show kicad &>/dev/null; then
            echo "Note: For the latest KiCad version, you may want to add the KiCad PPA"
            echo "Run: sudo add-apt-repository --yes ppa:kicad/kicad-8.0-releases"
        fi
        
        # Don't require sudo if already in sudoers
        if command -v kicad &>/dev/null; then
            echo "KiCad already installed, skipping apt install"
        else
            echo "Note: KiCad installation requires system package installation"
            echo "You may need to run: sudo apt-get install -y kicad kicad-libraries kicad-doc-en"
        fi
        
        # Install Python dependencies
        # Try virtual environment first, fall back to user install
        if python3 -m venv --help &>/dev/null; then
            if [[ ! -d "${KICAD_DATA_DIR}/venv" ]]; then
                echo "Creating Python virtual environment for KiCad tools..."
                python3 -m venv "${KICAD_DATA_DIR}/venv" 2>/dev/null || true
            fi
            
            if [[ -f "${KICAD_DATA_DIR}/venv/bin/pip" ]]; then
                "${KICAD_DATA_DIR}/venv/bin/pip" install --upgrade pip
                "${KICAD_DATA_DIR}/venv/bin/pip" install pykicad kikit pcbdraw
            else
                echo "Virtual environment not available, using user install..."
                python3 -m pip install --user --break-system-packages pykicad kikit pcbdraw 2>/dev/null || \
                python3 -m pip install --user pykicad kikit pcbdraw 2>/dev/null || true
            fi
        else
            echo "Installing Python packages in user space..."
            python3 -m pip install --user --break-system-packages pykicad kikit pcbdraw 2>/dev/null || \
            python3 -m pip install --user pykicad kikit pcbdraw 2>/dev/null || true
        fi
        
    elif [[ -f /etc/redhat-release ]]; then
        # RHEL/Fedora/CentOS
        echo "Installing KiCad on RHEL/Fedora..."
        if command -v kicad &>/dev/null; then
            echo "KiCad already installed, skipping dnf install"
        else
            echo "Note: KiCad installation requires system package installation"
            echo "You may need to run: sudo dnf install -y kicad kicad-packages3d"
        fi
        
        # Install Python dependencies
        if python3 -m venv --help &>/dev/null; then
            if [[ ! -d "${KICAD_DATA_DIR}/venv" ]]; then
                python3 -m venv "${KICAD_DATA_DIR}/venv" 2>/dev/null || true
            fi
            
            if [[ -f "${KICAD_DATA_DIR}/venv/bin/pip" ]]; then
                "${KICAD_DATA_DIR}/venv/bin/pip" install --upgrade pip
                "${KICAD_DATA_DIR}/venv/bin/pip" install pykicad kikit pcbdraw
            else
                python3 -m pip install --user --break-system-packages pykicad kikit pcbdraw 2>/dev/null || \
                python3 -m pip install --user pykicad kikit pcbdraw 2>/dev/null || true
            fi
        else
            python3 -m pip install --user --break-system-packages pykicad kikit pcbdraw 2>/dev/null || \
            python3 -m pip install --user pykicad kikit pcbdraw 2>/dev/null || true
        fi
        
    elif [[ "$(uname)" == "Darwin" ]]; then
        # macOS
        echo "Installing KiCad on macOS..."
        if command -v brew &>/dev/null; then
            if ! command -v kicad &>/dev/null; then
                brew install --cask kicad
            else
                echo "KiCad already installed"
            fi
            
            # Set up Python virtual environment
            if [[ ! -d "${KICAD_DATA_DIR}/venv" ]]; then
                python3 -m venv "${KICAD_DATA_DIR}/venv"
            fi
            "${KICAD_DATA_DIR}/venv/bin/pip" install --upgrade pip
            "${KICAD_DATA_DIR}/venv/bin/pip" install pykicad kikit pcbdraw
        else
            echo "Error: Homebrew is required to install KiCad on macOS"
            return 1
        fi
        
    else
        echo "Error: Unsupported operating system"
        return 1
    fi
    
    # Set up initial configuration
    kicad::setup_initial_config
    
    # Install resource CLI
    local cli_install_script="${KICAD_INSTALL_LIB_DIR}/../../../lib/install-resource-cli.sh"
    if [[ -f "$cli_install_script" ]]; then
        source "$cli_install_script"
        install_resource_cli "kicad" "${KICAD_INSTALL_LIB_DIR}/../cli.sh"
    fi
    
    echo "KiCad installation complete!"
    return 0
}

# Set up initial KiCad configuration
kicad::setup_initial_config() {
    # Create initial project template
    cat > "${KICAD_TEMPLATES_DIR}/basic_template.kicad_pro" <<'EOF'
{
  "board": {
    "design_settings": {
      "defaults": {
        "board_outline_line_width": 0.1,
        "copper_line_width": 0.2,
        "copper_text_size_h": 1.5,
        "copper_text_size_v": 1.5,
        "copper_text_thickness": 0.3,
        "other_line_width": 0.15,
        "other_text_size_h": 1.0,
        "other_text_size_v": 1.0,
        "other_text_thickness": 0.15,
        "silk_line_width": 0.15,
        "silk_text_size_h": 1.0,
        "silk_text_size_v": 1.0,
        "silk_text_thickness": 0.15
      },
      "rules": {
        "min_copper_edge_clearance": 0.0,
        "solder_mask_clearance": 0.0,
        "solder_mask_min_width": 0.0
      }
    }
  },
  "schematic": {
    "drawing": {
      "default_line_thickness": 6.0,
      "default_text_size": 50.0,
      "field_names": [],
      "intersheets_ref_own_page": false,
      "intersheets_ref_prefix": "",
      "intersheets_ref_short": false,
      "intersheets_ref_show": false,
      "intersheets_ref_suffix": "",
      "junction_size_choice": 3,
      "pin_symbol_size": 25.0,
      "text_offset_ratio": 0.3
    }
  }
}
EOF
    
    echo "Initial KiCad configuration created"
}

# Export function
export -f kicad::install
export -f kicad::setup_initial_config