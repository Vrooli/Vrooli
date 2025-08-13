#!/usr/bin/env bash
################################################################################
# Vrooli Orchestrator Dependencies Setup
# 
# Ensures that all required dependencies for the Vrooli Orchestrator are
# available on the system. This includes verifying system commands and
# setting up the orchestrator environment.
################################################################################

set -euo pipefail

ORIGINAL_DIR=$(pwd)
LIB_DEPS_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

# Source utilities
# shellcheck disable=SC1091
source "${LIB_DEPS_DIR}/../utils/var.sh"
# shellcheck disable=SC1091
source "${var_LIB_UTILS_DIR}/flow.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"

# Orchestrator configuration
ORCHESTRATOR_HOME="${VROOLI_ORCHESTRATOR_HOME:-$HOME/.vrooli/orchestrator}"
SCRIPTS_DIR="${LIB_DEPS_DIR}/../../scenarios/tools"

# Check if a command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Install missing system dependencies
install_system_deps() {
    local missing_deps=()
    local required_commands=("jq" "curl" "ps" "netstat" "mkfifo")
    
    log::info "Checking system dependencies..."
    
    # Check each required command
    for cmd in "${required_commands[@]}"; do
        if ! command_exists "$cmd"; then
            missing_deps+=("$cmd")
        fi
    done
    
    # Handle missing dependencies
    if [[ ${#missing_deps[@]} -gt 0 ]]; then
        log::info "Missing dependencies: ${missing_deps[*]}"
        
        # Try to install based on package manager
        if command_exists "apt-get"; then
            log::info "Installing dependencies with apt-get..."
            local apt_packages=()
            
            for dep in "${missing_deps[@]}"; do
                case "$dep" in
                    jq) apt_packages+=("jq") ;;
                    curl) apt_packages+=("curl") ;;
                    ps) apt_packages+=("procps") ;;
                    netstat) apt_packages+=("net-tools") ;;
                    mkfifo) apt_packages+=("coreutils") ;;
                esac
            done
            
            if [[ ${#apt_packages[@]} -gt 0 ]]; then
                sudo apt-get update >/dev/null 2>&1
                sudo apt-get install -y "${apt_packages[@]}"
            fi
            
        elif command_exists "yum"; then
            log::info "Installing dependencies with yum..."
            local yum_packages=()
            
            for dep in "${missing_deps[@]}"; do
                case "$dep" in
                    jq) yum_packages+=("jq") ;;
                    curl) yum_packages+=("curl") ;;
                    ps) yum_packages+=("procps-ng") ;;
                    netstat) yum_packages+=("net-tools") ;;
                    mkfifo) yum_packages+=("coreutils") ;;
                esac
            done
            
            if [[ ${#yum_packages[@]} -gt 0 ]]; then
                sudo yum install -y "${yum_packages[@]}"
            fi
            
        elif command_exists "brew"; then
            log::info "Installing dependencies with brew..."
            local brew_packages=()
            
            for dep in "${missing_deps[@]}"; do
                case "$dep" in
                    jq) brew_packages+=("jq") ;;
                    curl) brew_packages+=("curl") ;;
                    # ps, netstat, mkfifo are built into macOS
                esac
            done
            
            if [[ ${#brew_packages[@]} -gt 0 ]]; then
                brew install "${brew_packages[@]}"
            fi
            
        else
            log::error "No supported package manager found"
            log::error "Please install these dependencies manually: ${missing_deps[*]}"
            return 1
        fi
        
        # Verify installation
        local still_missing=()
        for dep in "${missing_deps[@]}"; do
            if ! command_exists "$dep"; then
                still_missing+=("$dep")
            fi
        done
        
        if [[ ${#still_missing[@]} -gt 0 ]]; then
            log::error "Failed to install: ${still_missing[*]}"
            return 1
        fi
    fi
    
    log::success "All system dependencies are available"
    return 0
}

# Setup orchestrator environment
setup_orchestrator_env() {
    log::info "Setting up orchestrator environment..."
    
    # Create orchestrator home directory
    mkdir -p "$ORCHESTRATOR_HOME"
    mkdir -p "$ORCHESTRATOR_HOME/logs"
    mkdir -p "$ORCHESTRATOR_HOME/sockets"
    mkdir -p "$ORCHESTRATOR_HOME/backups"
    
    # Set proper permissions
    chmod 755 "$ORCHESTRATOR_HOME"
    chmod 755 "$ORCHESTRATOR_HOME/logs"
    chmod 755 "$ORCHESTRATOR_HOME/sockets"
    chmod 755 "$ORCHESTRATOR_HOME/backups"
    
    # Initialize empty process registry if it doesn't exist
    local registry_file="$ORCHESTRATOR_HOME/processes.json"
    if [[ ! -f "$registry_file" ]]; then
        cat > "$registry_file" << 'EOF'
{
  "version": "1.0.0",
  "processes": {},
  "metadata": {
    "started_at": null,
    "total_started": 0,
    "created_at": "$(date -Iseconds)"
  }
}
EOF
    fi
    
    log::success "Orchestrator environment setup complete"
    return 0
}

# Verify orchestrator scripts
verify_orchestrator_scripts() {
    log::info "Verifying orchestrator scripts..."
    
    local scripts=(
        "vrooli-orchestrator.sh"
        "orchestrator-client.sh"
        "orchestrator-ctl.sh"
        "multi-app-runner.sh"
    )
    
    local missing_scripts=()
    
    for script in "${scripts[@]}"; do
        local script_path="$SCRIPTS_DIR/$script"
        
        if [[ ! -f "$script_path" ]]; then
            missing_scripts+=("$script")
        elif [[ ! -x "$script_path" ]]; then
            log::info "Making $script executable..."
            chmod +x "$script_path"
        fi
    done
    
    if [[ ${#missing_scripts[@]} -gt 0 ]]; then
        log::error "Missing orchestrator scripts: ${missing_scripts[*]}"
        log::error "Script directory: $SCRIPTS_DIR"
        return 1
    fi
    
    log::success "All orchestrator scripts are available and executable"
    return 0
}

# Create orchestrator symlinks (optional)
create_orchestrator_symlinks() {
    log::info "Setting up orchestrator command symlinks..."
    
    local bin_dir="$HOME/.local/bin"
    mkdir -p "$bin_dir"
    
    # Create symlinks for easy access
    local commands=(
        "orchestrator-ctl:$SCRIPTS_DIR/orchestrator-ctl.sh"
        "multi-app-runner:$SCRIPTS_DIR/multi-app-runner.sh"
    )
    
    for cmd_mapping in "${commands[@]}"; do
        local cmd_name="${cmd_mapping%%:*}"
        local script_path="${cmd_mapping##*:}"
        local symlink_path="$bin_dir/$cmd_name"
        
        if [[ ! -L "$symlink_path" ]]; then
            ln -sf "$script_path" "$symlink_path"
            log::info "Created symlink: $cmd_name -> $script_path"
        fi
    done
    
    # Add to PATH if not already there
    if [[ ":$PATH:" != *":$bin_dir:"* ]]; then
        log::info "Adding $bin_dir to PATH"
        echo "# Vrooli Orchestrator" >> "$HOME/.bashrc"
        echo "export PATH=\"\$HOME/.local/bin:\$PATH\"" >> "$HOME/.bashrc"
        log::info "Please run 'source ~/.bashrc' or start a new shell session"
    fi
    
    log::success "Orchestrator symlinks created"
    return 0
}

# Show orchestrator status
show_orchestrator_status() {
    echo ""
    log::header "Orchestrator Setup Complete"
    echo ""
    
    echo "📁 Home Directory: $ORCHESTRATOR_HOME"
    echo "🔧 Scripts Directory: $SCRIPTS_DIR"
    echo ""
    
    echo "Available Commands:"
    echo "  $SCRIPTS_DIR/orchestrator-ctl.sh start     # Start orchestrator daemon"
    echo "  $SCRIPTS_DIR/orchestrator-ctl.sh status    # Show process status"
    echo "  $SCRIPTS_DIR/multi-app-runner.sh           # Start all generated apps"
    echo ""
    
    if [[ -L "$HOME/.local/bin/orchestrator-ctl" ]]; then
        echo "Global Commands (if ~/.local/bin is in PATH):"
        echo "  orchestrator-ctl start    # Start orchestrator daemon"
        echo "  orchestrator-ctl status   # Show process status"
        echo "  multi-app-runner          # Start all generated apps"
        echo ""
    fi
    
    echo "Environment Variables:"
    echo "  VROOLI_ORCHESTRATOR_HOME  Current: ${VROOLI_ORCHESTRATOR_HOME:-$HOME/.vrooli/orchestrator}"
    echo "  VROOLI_MAX_APPS           Current: ${VROOLI_MAX_APPS:-20}"
    echo "  VROOLI_MAX_DEPTH          Current: ${VROOLI_MAX_DEPTH:-5}"
    echo "  VROOLI_MAX_PER_PARENT     Current: ${VROOLI_MAX_PER_PARENT:-10}"
    echo ""
    
    log::success "✅ Vrooli Orchestrator is ready to use!"
}

# Main setup function
orchestrator::setup() {
    log::header "Setting up Vrooli Orchestrator"
    
    if ! install_system_deps; then
        log::error "Failed to install system dependencies"
        return 1
    fi
    
    if ! setup_orchestrator_env; then
        log::error "Failed to setup orchestrator environment"
        return 1
    fi
    
    if ! verify_orchestrator_scripts; then
        log::error "Failed to verify orchestrator scripts"
        return 1
    fi
    
    # Optional: Create symlinks for global access
    create_orchestrator_symlinks || log::warning "Failed to create symlinks (non-fatal)"
    
    show_orchestrator_status
    return 0
}

# Check if orchestrator is properly installed
orchestrator::check() {
    local errors=0
    
    log::info "Checking orchestrator installation..."
    
    # Check system dependencies
    local required_commands=("jq" "curl" "ps" "netstat" "mkfifo")
    for cmd in "${required_commands[@]}"; do
        if ! command_exists "$cmd"; then
            log::error "Missing command: $cmd"
            errors=$((errors + 1))
        fi
    done
    
    # Check orchestrator home
    if [[ ! -d "$ORCHESTRATOR_HOME" ]]; then
        log::error "Orchestrator home not found: $ORCHESTRATOR_HOME"
        errors=$((errors + 1))
    fi
    
    # Check scripts
    local scripts=("vrooli-orchestrator.sh" "orchestrator-client.sh" "orchestrator-ctl.sh" "multi-app-runner.sh")
    for script in "${scripts[@]}"; do
        local script_path="$SCRIPTS_DIR/$script"
        if [[ ! -x "$script_path" ]]; then
            log::error "Script not found or not executable: $script_path"
            errors=$((errors + 1))
        fi
    done
    
    if [[ $errors -eq 0 ]]; then
        log::success "✅ Orchestrator installation is valid"
        return 0
    else
        log::error "❌ Found $errors issues with orchestrator installation"
        return 1
    fi
}

# Main command handler
main() {
    local command="${1:-setup}"
    
    case "$command" in
        setup)
            orchestrator::setup
            ;;
        check)
            orchestrator::check
            ;;
        *)
            echo "Usage: $0 {setup|check}"
            echo ""
            echo "Commands:"
            echo "  setup  - Install orchestrator dependencies and setup environment"
            echo "  check  - Check if orchestrator is properly installed"
            exit 1
            ;;
    esac
}

# Execute main function if script is run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi

# Export functions for sourcing
export -f orchestrator::setup
export -f orchestrator::check