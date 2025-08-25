#!/usr/bin/env bash
# LangChain Core Functions
# Core functionality for LangChain resource management

# Source required libraries
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
LANGCHAIN_CORE_DIR="${APP_ROOT}/resources/langchain/lib"
# shellcheck disable=SC1091
source "${LANGCHAIN_CORE_DIR}/../../../../lib/utils/var.sh"
# shellcheck disable=SC1091
source "${LANGCHAIN_CORE_DIR}/../../../../lib/utils/log.sh"
# shellcheck disable=SC1091
source "${LANGCHAIN_CORE_DIR}/../config/defaults.sh" 2>/dev/null || true

# Ensure configuration is exported
if command -v langchain::export_config &>/dev/null; then
    langchain::export_config 2>/dev/null || true
fi

#######################################
# Check if LangChain is installed
#######################################
langchain::is_installed() {
    if [[ -d "$LANGCHAIN_VENV_DIR" ]] && [[ -f "$LANGCHAIN_VENV_DIR/bin/python" ]]; then
        return 0
    fi
    return 1
}

#######################################
# Check if LangChain service is running
#######################################
langchain::is_running() {
    # Check if any LangChain processes are running
    if pgrep -f "langchain" >/dev/null 2>&1; then
        return 0
    fi
    
    # Check if API server is responding
    if curl -sf "${LANGCHAIN_API_URL}/health" >/dev/null 2>&1; then
        return 0
    fi
    
    return 1
}

#######################################
# Get LangChain service status
#######################################
langchain::get_status() {
    if ! langchain::is_installed; then
        echo "not_installed"
    elif ! langchain::is_running; then
        echo "stopped"
    elif langchain::is_healthy; then
        echo "healthy"
    else
        echo "unhealthy"
    fi
}

#######################################
# Check if LangChain is healthy
#######################################
langchain::is_healthy() {
    # Check virtual environment
    if ! langchain::is_installed; then
        return 1
    fi
    
    # Check if core packages are installed
    if ! langchain::check_packages; then
        return 1
    fi
    
    # API server is optional for LangChain
    # Only check if explicitly running
    if pgrep -f "langchain.*api" >/dev/null 2>&1; then
        if [[ -n "$LANGCHAIN_API_URL" ]]; then
            if ! curl -sf "${LANGCHAIN_API_URL}/health" >/dev/null 2>&1; then
                return 1
            fi
        fi
    fi
    
    return 0
}

#######################################
# Check if required packages are installed
#######################################
langchain::check_packages() {
    if ! langchain::is_installed; then
        return 1
    fi
    
    local python_cmd="${LANGCHAIN_VENV_DIR}/bin/python"
    
    # Check core packages
    for package in $LANGCHAIN_CORE_PACKAGES; do
        if ! "$python_cmd" -c "import ${package//-/_}" 2>/dev/null; then
            return 1
        fi
    done
    
    return 0
}

#######################################
# Initialize LangChain directories
#######################################
langchain::init_directories() {
    local dirs=(
        "$LANGCHAIN_DATA_DIR"
        "$LANGCHAIN_WORKSPACE_DIR"
        "$LANGCHAIN_CHAINS_DIR"
        "$LANGCHAIN_AGENTS_DIR"
    )
    
    for dir in "${dirs[@]}"; do
        if [[ ! -d "$dir" ]]; then
            mkdir -p "$dir"
            log::success "Created directory: $dir"
        fi
    done
}

#######################################
# Create Python virtual environment
#######################################
langchain::create_venv() {
    if langchain::is_installed; then
        log::info "Virtual environment already exists at $LANGCHAIN_VENV_DIR"
        return 0
    fi
    
    log::info "Creating Python virtual environment..."
    
    # Create virtual environment
    python3 -m venv "$LANGCHAIN_VENV_DIR"
    
    if [[ ! -f "$LANGCHAIN_VENV_DIR/bin/python" ]]; then
        log::error "Failed to create virtual environment"
        return 1
    fi
    
    # Upgrade pip
    "$LANGCHAIN_VENV_DIR/bin/python" -m pip install --upgrade pip >/dev/null 2>&1
    
    log::success "Virtual environment created at $LANGCHAIN_VENV_DIR"
    return 0
}

#######################################
# Install LangChain packages
#######################################
langchain::install_packages() {
    if ! langchain::is_installed; then
        log::error "Virtual environment not found"
        return 1
    fi
    
    local pip_cmd="${LANGCHAIN_VENV_DIR}/bin/pip"
    
    log::info "Installing LangChain packages..."
    
    # Install core packages
    for package in $LANGCHAIN_CORE_PACKAGES; do
        log::info "Installing $package..."
        "$pip_cmd" install "$package" >/dev/null 2>&1
    done
    
    # Install integration packages if enabled
    if [[ "$LANGCHAIN_ENABLE_OLLAMA" == "true" ]] || [[ "$LANGCHAIN_ENABLE_OPENROUTER" == "true" ]]; then
        for package in $LANGCHAIN_INTEGRATION_PACKAGES; do
            log::info "Installing $package..."
            "$pip_cmd" install "$package" >/dev/null 2>&1
        done
    fi
    
    # Install additional dependencies
    "$pip_cmd" install redis chromadb faiss-cpu >/dev/null 2>&1
    
    log::success "LangChain packages installed"
    return 0
}

#######################################
# Execute Python code in LangChain environment
#######################################
langchain::execute_python() {
    local code="$1"
    
    if ! langchain::is_installed; then
        log::error "LangChain not installed"
        return 1
    fi
    
    "${LANGCHAIN_VENV_DIR}/bin/python" -c "$code"
}

#######################################
# Run a LangChain script
#######################################
langchain::run_script() {
    local script="$1"
    shift
    
    if ! langchain::is_installed; then
        log::error "LangChain not installed"
        return 1
    fi
    
    if [[ ! -f "$script" ]]; then
        log::error "Script not found: $script"
        return 1
    fi
    
    "${LANGCHAIN_VENV_DIR}/bin/python" "$script" "$@"
}

#######################################
# Get LangChain version information
#######################################
langchain::get_version() {
    if ! langchain::is_installed; then
        echo "not_installed"
        return 1
    fi
    
    local version
    version=$(langchain::execute_python "import langchain; print(langchain.__version__)" 2>/dev/null)
    
    if [[ -n "$version" ]]; then
        echo "$version"
    else
        echo "unknown"
    fi
}