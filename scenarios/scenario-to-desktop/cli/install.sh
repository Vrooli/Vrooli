#!/bin/bash

# scenario-to-desktop CLI Installation Script
# Installs the scenario-to-desktop CLI tool for desktop application generation

set -e

# Configuration
CLI_NAME="scenario-to-desktop"
CLI_VERSION="1.0.0"
INSTALL_DIR="${HOME}/.local/bin"
SOURCE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CLI_SOURCE="${SOURCE_DIR}/${CLI_NAME}"

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

log() {
    echo -e "${GREEN}[INSTALL]${NC} $1"
}

warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

# Check if running on supported OS
check_os() {
    case "$(uname -s)" in
        Linux*)   OS="Linux";;
        Darwin*)  OS="macOS";;
        CYGWIN*)  OS="Windows";;
        MINGW*)   OS="Windows";;
        *)        OS="Unknown";;
    esac
    
    if [ "$OS" = "Unknown" ]; then
        error "Unsupported operating system: $(uname -s)"
        exit 1
    fi
    
    info "Detected OS: $OS"
}

# Check dependencies
check_dependencies() {
    local missing_deps=()
    
    # Check bash (should be available since we're running in bash)
    if ! command -v bash &> /dev/null; then
        missing_deps+=("bash")
    fi
    
    # Check Node.js
    if ! command -v node &> /dev/null; then
        warn "Node.js not found - required for desktop app generation"
        echo "  Install Node.js 18+ from: https://nodejs.org/"
    else
        local node_version=$(node --version | sed 's/v//')
        info "Found Node.js: $node_version"
    fi
    
    # Check npm
    if ! command -v npm &> /dev/null; then
        warn "npm not found - required for desktop app generation"
    else
        local npm_version=$(npm --version)
        info "Found npm: $npm_version"
    fi
    
    # Check curl
    if ! command -v curl &> /dev/null; then
        missing_deps+=("curl")
    fi
    
    # Check jq
    if ! command -v jq &> /dev/null; then
        warn "jq not found - required for JSON processing"
        case "$OS" in
            Linux)
                echo "  Install with: sudo apt-get install jq  # Ubuntu/Debian"
                echo "           or: sudo yum install jq      # CentOS/RHEL"
                echo "           or: sudo pacman -S jq        # Arch Linux"
                ;;
            macOS)
                echo "  Install with: brew install jq"
                echo "           or: sudo port install jq     # MacPorts"
                ;;
        esac
    fi
    
    if [ ${#missing_deps[@]} -ne 0 ]; then
        error "Missing critical dependencies:"
        for dep in "${missing_deps[@]}"; do
            echo "  - $dep"
        done
        exit 1
    fi
}

# Create installation directory
create_install_dir() {
    if [ ! -d "$INSTALL_DIR" ]; then
        log "Creating installation directory: $INSTALL_DIR"
        mkdir -p "$INSTALL_DIR"
    fi
    
    # Check if install directory is in PATH
    if [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
        warn "Installation directory not in PATH: $INSTALL_DIR"
        echo
        echo "Add the following line to your shell profile (.bashrc, .zshrc, etc.):"
        echo "  export PATH=\"\$PATH:$INSTALL_DIR\""
        echo
        echo "Then reload your shell or run: source ~/.bashrc"
        echo
    fi
}

# Install CLI
install_cli() {
    if [ ! -f "$CLI_SOURCE" ]; then
        error "CLI source file not found: $CLI_SOURCE"
        exit 1
    fi
    
    local target_path="${INSTALL_DIR}/${CLI_NAME}"
    
    log "Installing $CLI_NAME to $target_path"
    cp "$CLI_SOURCE" "$target_path"
    chmod +x "$target_path"
    
    # Verify installation
    if [ -x "$target_path" ]; then
        log "Installation successful!"
        info "CLI installed at: $target_path"
    else
        error "Installation failed - executable not found"
        exit 1
    fi
}

# Test installation
test_installation() {
    local cli_path="${INSTALL_DIR}/${CLI_NAME}"
    
    if [ -x "$cli_path" ]; then
        log "Testing installation..."
        
        # Test version command
        if "$cli_path" version > /dev/null 2>&1; then
            log "CLI test passed!"
            
            # Show version info
            echo
            info "Installed version:"
            "$cli_path" version
        else
            warn "CLI test failed - version command not working"
        fi
    else
        error "CLI not found at expected location: $cli_path"
        exit 1
    fi
}

# Create desktop entry (Linux only)
create_desktop_entry() {
    if [ "$OS" != "Linux" ]; then
        return
    fi
    
    local desktop_dir="${HOME}/.local/share/applications"
    local desktop_file="${desktop_dir}/scenario-to-desktop.desktop"
    
    if [ ! -d "$desktop_dir" ]; then
        mkdir -p "$desktop_dir"
    fi
    
    log "Creating desktop entry: $desktop_file"
    
    cat > "$desktop_file" << EOF
[Desktop Entry]
Version=1.0
Name=Scenario to Desktop
Comment=Transform Vrooli scenarios into desktop applications
Exec=${INSTALL_DIR}/${CLI_NAME}
Icon=application-x-executable
Terminal=true
Type=Application
Categories=Development;
Keywords=desktop;electron;development;vrooli;
EOF
    
    chmod +x "$desktop_file"
    info "Desktop entry created"
}

# Setup shell completions
setup_completions() {
    local completion_dir="${HOME}/.local/share/bash-completion/completions"
    
    if [ ! -d "$completion_dir" ]; then
        mkdir -p "$completion_dir"
    fi
    
    log "Setting up shell completions..."
    
    # Create basic bash completion
    cat > "${completion_dir}/${CLI_NAME}" << 'EOF'
#!/bin/bash
# scenario-to-desktop bash completion

_scenario_to_desktop() {
    local cur prev opts
    COMPREPLY=()
    cur="${COMP_WORDS[COMP_CWORD]}"
    prev="${COMP_WORDS[COMP_CWORD-1]}"

    # Main commands
    local commands="help version status templates generate build test package"
    
    # Frameworks
    local frameworks="electron tauri neutralino"
    
    # Templates  
    local templates="basic advanced multi_window kiosk"
    
    # Platforms
    local platforms="win mac linux all"

    case ${prev} in
        scenario-to-desktop)
            COMPREPLY=( $(compgen -W "${commands}" -- ${cur}) )
            return 0
            ;;
        --framework)
            COMPREPLY=( $(compgen -W "${frameworks}" -- ${cur}) )
            return 0
            ;;
        --template)
            COMPREPLY=( $(compgen -W "${templates}" -- ${cur}) )
            return 0
            ;;
        --platforms)
            COMPREPLY=( $(compgen -W "${platforms}" -- ${cur}) )
            return 0
            ;;
        --output|--config)
            COMPREPLY=( $(compgen -f -- ${cur}) )
            return 0
            ;;
    esac

    # General options
    local opts="--help --version --verbose --json --api-url"
    
    case ${COMP_WORDS[1]} in
        generate)
            opts="--framework --template --platforms --output --features --config ${opts}"
            ;;
        build)
            opts="--platforms --sign --publish ${opts}"
            ;;
        test)
            opts="--platforms --headless ${opts}"
            ;;
        package)
            opts="--store --enterprise ${opts}"
            ;;
    esac

    COMPREPLY=( $(compgen -W "${opts}" -- ${cur}) )
    return 0
}

complete -F _scenario_to_desktop scenario-to-desktop
EOF
    
    info "Shell completions installed"
    echo "Restart your shell or run 'source ~/.bashrc' to enable completions"
}

# Show post-installation information
show_post_install_info() {
    echo
    echo "=================================================="
    echo -e "${GREEN}scenario-to-desktop CLI Installation Complete!${NC}"
    echo "=================================================="
    echo
    echo "🎉 Installation successful!"
    echo
    echo "📍 Installed location: ${INSTALL_DIR}/${CLI_NAME}"
    echo "🔗 Version: $CLI_VERSION"
    echo
    echo "🚀 Quick start:"
    echo "  1. Ensure the API server is running:"
    echo "     cd $(dirname "$SOURCE_DIR")/api && make run"
    echo
    echo "  2. Generate your first desktop app:"
    echo "     $CLI_NAME generate picker-wheel"
    echo
    echo "  3. Check system status:"
    echo "     $CLI_NAME status"
    echo
    echo "📚 Get help:"
    echo "     $CLI_NAME help"
    echo "     $CLI_NAME <command> --help"
    echo
    echo "🌐 Documentation:"
    echo "     https://vrooli.com/scenarios/scenario-to-desktop"
    echo
    
    if [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
        echo "⚠️  Don't forget to add $INSTALL_DIR to your PATH!"
        echo "   Add this to your ~/.bashrc or ~/.zshrc:"
        echo "   export PATH=\"\$PATH:$INSTALL_DIR\""
    fi
}

# Uninstall function
uninstall() {
    local target_path="${INSTALL_DIR}/${CLI_NAME}"
    local desktop_file="${HOME}/.local/share/applications/scenario-to-desktop.desktop"
    local completion_file="${HOME}/.local/share/bash-completion/completions/${CLI_NAME}"
    
    echo "Uninstalling scenario-to-desktop CLI..."
    
    # Remove CLI
    if [ -f "$target_path" ]; then
        rm "$target_path"
        log "Removed CLI: $target_path"
    fi
    
    # Remove desktop entry
    if [ -f "$desktop_file" ]; then
        rm "$desktop_file"
        log "Removed desktop entry: $desktop_file"
    fi
    
    # Remove completion
    if [ -f "$completion_file" ]; then
        rm "$completion_file"
        log "Removed shell completion: $completion_file"
    fi
    
    log "Uninstallation complete!"
}

# Main installation function
main() {
    echo "scenario-to-desktop CLI Installer v$CLI_VERSION"
    echo "=============================================="
    echo
    
    # Handle special commands
    case "${1:-}" in
        --uninstall)
            uninstall
            exit 0
            ;;
        --help|-h)
            cat << 'EOF'
scenario-to-desktop CLI Installer

USAGE:
    ./install.sh [OPTIONS]

OPTIONS:
    --uninstall    Remove the installed CLI
    --help, -h     Show this help message

INSTALLATION:
    The installer will:
    1. Check system dependencies
    2. Create installation directory (~/.local/bin)
    3. Copy CLI to installation directory
    4. Set up shell completions
    5. Create desktop entry (Linux only)
    6. Test the installation

REQUIREMENTS:
    - bash
    - curl
    - jq (recommended)
    - Node.js 18+ (for desktop app generation)
    - npm (for desktop app generation)
EOF
            exit 0
            ;;
    esac
    
    # Run installation steps
    check_os
    check_dependencies
    create_install_dir
    install_cli
    test_installation
    create_desktop_entry
    setup_completions
    show_post_install_info
}

# Execute main function
main "$@"