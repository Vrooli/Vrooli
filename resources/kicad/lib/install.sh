#!/bin/bash
# KiCad Installation Functions

# Get script directory
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
KICAD_INSTALL_LIB_DIR="${APP_ROOT}/resources/kicad/lib"

# Source common functions
source "${KICAD_INSTALL_LIB_DIR}/common.sh"

# Check if installation is possible
kicad::can_install() {
    # Check if we're on a supported platform
    if [[ -f /etc/debian_version ]] || [[ -f /etc/redhat-release ]] || [[ "$OSTYPE" == "darwin"* ]]; then
        return 0
    fi
    return 1
}

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
        
        # Check if KiCad is already installed
        if command -v kicad &>/dev/null || command -v kicad-cli &>/dev/null; then
            echo "KiCad already installed, checking version..."
            kicad_version=$(kicad::get_version)
            echo "KiCad version: $kicad_version"
        else
            echo "KiCad not found. Attempting to install..."
            
            # Try to update package list and install KiCad
            # First check if we can use apt without password (in CI/CD or with NOPASSWD sudo)
            if sudo -n apt-get update &>/dev/null; then
                echo "Updating package list..."
                sudo apt-get update
                
                # Try to add KiCad PPA for latest version
                if command -v add-apt-repository &>/dev/null; then
                    echo "Adding KiCad 8.0 PPA for latest version..."
                    sudo add-apt-repository --yes ppa:kicad/kicad-8.0-releases 2>/dev/null || true
                    sudo apt-get update
                fi
                
                echo "Installing KiCad and libraries..."
                sudo apt-get install -y kicad kicad-libraries kicad-doc-en || {
                    echo "Failed to install full KiCad suite, trying minimal installation..."
                    sudo apt-get install -y kicad || {
                        echo "Warning: Could not install KiCad automatically"
                        echo "Please run manually: sudo apt-get install -y kicad kicad-libraries"
                    }
                }
            else
                echo "Cannot install KiCad automatically (sudo requires password)"
                echo "To install KiCad, please run:"
                echo "  sudo add-apt-repository --yes ppa:kicad/kicad-8.0-releases"
                echo "  sudo apt-get update"
                echo "  sudo apt-get install -y kicad kicad-libraries kicad-doc-en"
                echo ""
                echo "Continuing with mock installation for development..."
            fi
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
            if ! command -v kicad-cli &>/dev/null && ! ls /Applications/KiCad/KiCad.app &>/dev/null; then
                echo "Installing KiCad via Homebrew..."
                brew install --cask kicad || {
                    echo "Failed to install KiCad via Homebrew"
                    echo "You can download KiCad manually from: https://www.kicad.org/download/macos/"
                    echo "Continuing with mock installation for development..."
                }
                
                # KiCad on macOS installs to /Applications/KiCad/KiCad.app
                # The CLI tools are in the app bundle
                if [[ -d "/Applications/KiCad/KiCad.app" ]]; then
                    echo "KiCad installed to /Applications/KiCad/KiCad.app"
                    # Add KiCad CLI to PATH if not already there
                    kicad_cli_path="/Applications/KiCad/KiCad.app/Contents/MacOS"
                    if [[ ":$PATH:" != *":$kicad_cli_path:"* ]]; then
                        echo "export PATH=\"\$PATH:$kicad_cli_path\"" >> ~/.zshrc
                        echo "export PATH=\"\$PATH:$kicad_cli_path\"" >> ~/.bashrc
                        export PATH="$PATH:$kicad_cli_path"
                        echo "Added KiCad CLI tools to PATH"
                    fi
                fi
            else
                echo "KiCad already installed"
                kicad_version=$(kicad::get_version)
                echo "KiCad version: $kicad_version"
            fi
            
            # Set up Python virtual environment
            if [[ ! -d "${KICAD_DATA_DIR}/venv" ]]; then
                python3 -m venv "${KICAD_DATA_DIR}/venv"
            fi
            "${KICAD_DATA_DIR}/venv/bin/pip" install --upgrade pip
            "${KICAD_DATA_DIR}/venv/bin/pip" install pykicad kikit pcbdraw
        else
            echo "Error: Homebrew is required to install KiCad on macOS"
            echo "Install Homebrew from: https://brew.sh"
            echo "Then run: brew install --cask kicad"
            echo "Continuing with mock installation for development..."
        fi
        
    else
        echo "Error: Unsupported operating system"
        return 1
    fi
    
    # Set up initial configuration
    kicad::setup_initial_config
    
    # Create mock implementation if KiCad binary not available
    if ! command -v kicad &>/dev/null && ! command -v kicad-cli &>/dev/null; then
        echo "Creating mock KiCad implementation for development..."
        kicad::create_mock_implementation
    fi
    
    # Install resource CLI
    local cli_install_script="${APP_ROOT}/scripts/lib/resources/install-resource-cli.sh"
    if [[ -f "$cli_install_script" ]]; then
        source "$cli_install_script"
        install_resource_cli "kicad" "${APP_ROOT}/resources/kicad/cli.sh"
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

# Create mock KiCad implementation for development/testing
kicad::create_mock_implementation() {
    echo "Creating mock KiCad CLI for development..."
    
    # Create mock kicad-cli script
    cat > "${KICAD_DATA_DIR}/kicad-cli-mock" <<'EOF'
#!/bin/bash
# Mock KiCad CLI for development/testing

case "$1" in
    version)
        echo "8.0.0-mock"
        ;;
    pcb)
        case "$2" in
            export)
                echo "Mock: Exporting PCB to ${@:3}"
                # Create dummy output files
                for arg in "$@"; do
                    if [[ "$arg" == *.gbr ]] || [[ "$arg" == *.drl ]] || [[ "$arg" == *.pdf ]]; then
                        touch "$arg"
                        echo "Mock: Created $arg"
                    fi
                done
                ;;
            *)
                echo "Mock: PCB command $2 (not implemented)"
                ;;
        esac
        ;;
    sch)
        case "$2" in
            export)
                echo "Mock: Exporting schematic to ${@:3}"
                # Create dummy output files
                for arg in "$@"; do
                    if [[ "$arg" == *.pdf ]] || [[ "$arg" == *.svg ]]; then
                        touch "$arg"
                        echo "Mock: Created $arg"
                    fi
                done
                ;;
            *)
                echo "Mock: Schematic command $2 (not implemented)"
                ;;
        esac
        ;;
    *)
        echo "Mock KiCad CLI - Development Mode"
        echo "Usage: kicad-cli [version|pcb|sch] ..."
        echo ""
        echo "This is a mock implementation. Install KiCad for full functionality:"
        echo "  Ubuntu/Debian: sudo apt-get install kicad"
        echo "  macOS: brew install --cask kicad"
        echo "  Other: https://www.kicad.org/download/"
        ;;
esac
EOF
    chmod +x "${KICAD_DATA_DIR}/kicad-cli-mock"
    
    # Create symlink if possible
    if [[ -w /usr/local/bin ]]; then
        ln -sf "${KICAD_DATA_DIR}/kicad-cli-mock" /usr/local/bin/kicad-cli-mock 2>/dev/null || true
    fi
    
    echo "Mock KiCad CLI created at ${KICAD_DATA_DIR}/kicad-cli-mock"
}

# Export functions
export -f kicad::can_install
export -f kicad::install
export -f kicad::setup_initial_config
export -f kicad::create_mock_implementation