#!/usr/bin/env bash
set -euo pipefail

# Define directories using cached APP_ROOT
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
CLINE_LIB_DIR="${APP_ROOT}/resources/cline/lib"
CLINE_DIR="${APP_ROOT}/resources/cline"

# Source utilities
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/lib/utils/format.sh"
source "${APP_ROOT}/scripts/lib/utils/log.sh"
source "${APP_ROOT}/scripts/resources/port_registry.sh"

# Resource constants
export CLINE_NAME="cline"
export CLINE_CATEGORY="agents"
export CLINE_CONFIG_DIR="${var_vrooli_dir:-${HOME}/.vrooli}/cline"
export CLINE_DATA_DIR="${var_vrooli_data_dir:-${HOME}/.vrooli/data}/cline"
export CLINE_EXTENSION_ID="saoudrizwan.claude-dev"
export CLINE_PORT="${CLINE_PORT:-8100}"
export CLINE_SETTINGS_FILE="${CLINE_CONFIG_DIR}/settings.json"
export CLINE_DEFAULT_PROVIDER="${CLINE_DEFAULT_PROVIDER:-ollama}"
export CLINE_DEFAULT_MODEL="${CLINE_DEFAULT_MODEL:-llama3.2:3b}"
export CLINE_USE_OLLAMA="${CLINE_USE_OLLAMA:-true}"
export CLINE_USE_OPENROUTER="${CLINE_USE_OPENROUTER:-false}"
export CLINE_OLLAMA_BASE_URL="${CLINE_OLLAMA_BASE_URL:-http://localhost:11434}"

# Extension settings path (for VS Code)
export VSCODE_SETTINGS="${HOME}/.config/Code/User/settings.json"

# Message constants
export MSG_CLINE_NO_VSCODE="VS Code is not installed"
export MSG_CLINE_ALREADY_INSTALLED="Cline extension is already installed"
export MSG_CLINE_INSTALLING="Installing Cline VS Code extension..."
export MSG_CLINE_INSTALL_FAILED="Failed to install Cline extension"
export MSG_CLINE_INSTALLED="Cline extension installed successfully"
export MSG_CLINE_CONFIGURING="Configuring Cline extension..."
export MSG_CLINE_NO_API_KEY="No API key found"
export MSG_CLINE_CONFIGURED="Cline configured successfully"
export MSG_CLINE_UNINSTALLING="Uninstalling Cline extension..."

# Create directories if needed
cline::ensure_dirs() {
    mkdir -p "$CLINE_CONFIG_DIR"
    mkdir -p "$CLINE_DATA_DIR"
    mkdir -p "${VSCODE_SETTINGS%/*}"
}

# Check if VS Code is installed
cline::check_vscode() {
    if command -v code >/dev/null 2>&1; then
        return 0
    else
        return 1
    fi
}

# Check if extension is installed
cline::is_extension_installed() {
    if cline::check_vscode; then
        code --list-extensions 2>/dev/null | grep -q "^${CLINE_EXTENSION_ID}$"
    else
        return 1
    fi
}

# Get API provider from config
cline::get_provider() {
    local provider="ollama"  # default
    if [[ -f "$CLINE_CONFIG_DIR/provider.conf" ]]; then
        provider=$(cat "$CLINE_CONFIG_DIR/provider.conf")
    fi
    echo "$provider"
}

# Get API key for provider
cline::get_api_key() {
    local provider="${1:-$(cline::get_provider)}"
    case "$provider" in
        openrouter)
            if [[ -f "${var_ROOT_DIR}/data/credentials/openrouter-credentials.json" ]]; then
                jq -r '.data.apiKey // empty' "${var_ROOT_DIR}/data/credentials/openrouter-credentials.json" 2>/dev/null || true
            fi
            ;;
        ollama)
            # Ollama doesn't need an API key
            echo ""
            ;;
        *)
            echo ""
            ;;
    esac
}

# Get API endpoint for provider
cline::get_endpoint() {
    local provider="${1:-$(cline::get_provider)}"
    case "$provider" in
        openrouter)
            echo "https://openrouter.ai/api/v1"
            ;;
        ollama)
            echo "${CLINE_OLLAMA_BASE_URL:-http://localhost:11434}"
            ;;
        anthropic)
            echo "https://api.anthropic.com/v1"
            ;;
        openai)
            echo "https://api.openai.com/v1"
            ;;
        google)
            echo "https://generativelanguage.googleapis.com/v1"
            ;;
        *)
            echo ""
            ;;
    esac
}

# Get Cline version
cline::get_version() {
    if cline::is_extension_installed; then
        code --list-extensions --show-versions 2>/dev/null | grep "${CLINE_EXTENSION_ID}" | cut -d'@' -f2
    else
        echo "not installed"
    fi
}

# Check if Cline is installed (alias for extension check)
cline::is_installed() {
    cline::is_extension_installed
}

# Check if Cline is configured
cline::is_configured() {
    local provider=$(cline::get_provider)
    
    # Ollama doesn't need API key
    if [[ "$provider" == "ollama" ]]; then
        # Just check if Ollama is running
        curl -s http://localhost:11434/api/version >/dev/null 2>&1
    else
        # Other providers need API key
        local api_key=$(cline::get_api_key "$provider")
        [[ -n "$api_key" ]]
    fi
}

# Create config directory
cline::create_config_dir() {
    cline::ensure_dirs
}

# Messages for Cline
export MSG_CLINE_NO_VSCODE="VS Code is not installed"
export MSG_CLINE_ALREADY_INSTALLED="Cline extension is already installed"
export MSG_CLINE_INSTALLING="Installing Cline VS Code extension..."
export MSG_CLINE_INSTALL_FAILED="Failed to install Cline extension"
export MSG_CLINE_INSTALLED="Cline extension installed successfully"
export MSG_CLINE_CONFIGURING="Configuring Cline..."
export MSG_CLINE_NO_API_KEY="No API key configured"
export MSG_CLINE_CONFIGURED="Cline configured successfully"
export MSG_CLINE_UNINSTALLING="Uninstalling Cline..."

# Additional configuration
export CLINE_HOME="${CLINE_CONFIG_DIR}"
export CLINE_TEMPERATURE="${CLINE_TEMPERATURE:-0.7}"
export CLINE_MAX_TOKENS="${CLINE_MAX_TOKENS:-4096}"
export CLINE_AUTO_APPROVE="${CLINE_AUTO_APPROVE:-false}"
export CLINE_EXPERIMENTAL_MODE="${CLINE_EXPERIMENTAL_MODE:-false}"
export CLINE_LOG_LEVEL="${CLINE_LOG_LEVEL:-info}"