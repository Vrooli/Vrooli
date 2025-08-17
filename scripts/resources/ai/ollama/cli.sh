#!/usr/bin/env bash
################################################################################
# Ollama Resource CLI
# 
# Lightweight CLI interface for Ollama using the CLI Command Framework
#
# Usage:
#   resource-ollama <command> [options]
#
################################################################################

set -euo pipefail

# Get script directory (resolving symlinks for installed CLI)
if [[ -L "${BASH_SOURCE[0]}" ]]; then
    OLLAMA_CLI_SCRIPT="$(readlink -f "${BASH_SOURCE[0]}")"
else
    OLLAMA_CLI_SCRIPT="${BASH_SOURCE[0]}"
fi
OLLAMA_CLI_DIR="$(cd "$(dirname "$OLLAMA_CLI_SCRIPT")" && pwd)"

# Source standard variables
# shellcheck disable=SC1091
source "${OLLAMA_CLI_DIR}/../../../lib/utils/var.sh"

# Source utilities using var_ variables
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "${var_RESOURCES_COMMON_FILE}"

# Source the CLI Command Framework
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_LIB_DIR}/cli-command-framework.sh"

# Source Ollama configuration and libraries
# shellcheck disable=SC1091
source "${OLLAMA_CLI_DIR}/config/defaults.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${OLLAMA_CLI_DIR}/config/messages.sh" 2>/dev/null || true
ollama::export_config 2>/dev/null || true

for lib in common api models status install; do
    lib_file="${OLLAMA_CLI_DIR}/lib/${lib}.sh"
    if [[ -f "$lib_file" ]]; then
        # shellcheck disable=SC1090
        source "$lib_file" 2>/dev/null || true
    fi
done

# Initialize CLI framework
cli::init "ollama" "Ollama AI inference server management"

# Override the help command to provide Ollama-specific examples
cli::register_command "help" "Show this help message with examples" "ollama_show_help"

# Register core commands - direct library function calls
cli::register_command "install" "Install Ollama" "ollama_install" "modifies-system"
cli::register_command "uninstall" "Uninstall Ollama" "ollama_uninstall" "modifies-system"
cli::register_command "start" "Start Ollama" "ollama_start" "modifies-system"
cli::register_command "stop" "Stop Ollama" "ollama_stop" "modifies-system"
cli::register_command "restart" "Restart Ollama" "ollama_restart" "modifies-system"

# Register status and monitoring commands
cli::register_command "status" "Show service status" "ollama::status"
cli::register_command "logs" "Show service logs" "ollama::logs"
cli::register_command "test" "Test service functionality" "ollama::test"
cli::register_command "info" "Show service information" "ollama::info"

# Register model management commands
cli::register_command "list-models" "List installed models" "ollama_list_models"
cli::register_command "pull-model" "Pull a model from registry" "ollama_pull_model"
cli::register_command "show-model" "Show details about a model" "ollama_show_model"
cli::register_command "remove-model" "Remove a model" "ollama_remove_model" "modifies-system"

# Register AI interaction commands
cli::register_command "chat" "Start interactive chat with a model" "ollama_chat"
cli::register_command "generate" "Generate text using a model" "ollama_generate"
cli::register_command "send-prompt" "Send prompt to model" "ollama_generate"

################################################################################
# Resource-specific command implementations
################################################################################

# Validate Ollama configuration
ollama_validate() {
    if command -v ollama::api::health &>/dev/null; then
        ollama::api::health
    else
        # Basic validation
        log::header "Validating Ollama"
        local port="${OLLAMA_PORT:-11434}"
        if curl -s "http://localhost:${port}/api/version" >/dev/null 2>&1; then
            log::success "Ollama API is accessible on port $port"
        else
            log::error "Ollama API not accessible on port $port"
            return 1
        fi
    fi
}

# Show Ollama status
ollama_status() {
    if command -v ollama::status::show &>/dev/null; then
        ollama::status::show
    else
        # Basic status
        log::header "Ollama Status"
        local port="${OLLAMA_PORT:-11434}"
        if curl -s "http://localhost:${port}/api/version" >/dev/null 2>&1; then
            echo "Service: ✅ Running on port $port"
            if command -v ollama &>/dev/null; then
                echo "CLI: ✅ Available"
                ollama list 2>/dev/null || echo "Models: No models installed"
            else
                echo "CLI: ❌ Not installed"
            fi
        else
            echo "Service: ❌ Not running"
        fi
    fi
}

# Start Ollama
ollama_start() {
    if command -v ollama::start &>/dev/null; then
        ollama::start
    else
        # Basic start using systemctl if available
        # Specific sudo permissions should be granted during installation
        if command -v systemctl &>/dev/null; then
            sudo systemctl start ollama 2>/dev/null || {
                log::error "Failed to start Ollama service"
                log::info "If this fails with permission denied, try reinstalling Ollama to setup sudo permissions"
                return 1
            }
        else
            log::error "Service management not available"
            return 1
        fi
    fi
}

# Stop Ollama
ollama_stop() {
    if command -v ollama::stop &>/dev/null; then
        ollama::stop
    else
        # Basic stop using systemctl if available
        # Specific sudo permissions should be granted during installation
        if command -v systemctl &>/dev/null; then
            sudo systemctl stop ollama 2>/dev/null || {
                log::error "Failed to stop Ollama service"
                log::info "If this fails with permission denied, try reinstalling Ollama to setup sudo permissions"
                return 1
            }
        else
            log::error "Service management not available"
            return 1
        fi
    fi
}

# Install Ollama
ollama_install() {
    # Parse arguments
    local force="no"
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --force)
                force="yes"
                shift
                ;;
            --force=*)
                force="${1#*=}"
                shift
                ;;
            *)
                shift
                ;;
        esac
    done
    
    # Export FORCE variable for the install function
    export FORCE="$force"
    
    if command -v ollama::install &>/dev/null; then
        ollama::install
    else
        log::error "ollama::install not available"
        return 1
    fi
}

# List installed models
ollama_list_models() {
    if command -v ollama &>/dev/null; then
        ollama list
    else
        log::error "Ollama CLI not available"
        return 1
    fi
}

# Pull a model from registry
ollama_pull_model() {
    local model="${1:-}"
    if [[ -z "$model" ]]; then
        log::error "Model name required"
        echo "Usage: resource-ollama pull-model <model-name>"
        echo ""
        echo "Popular models:"
        echo "  • llama3.2:3b      - Latest Llama 3.2 (3B params, fast)"
        echo "  • llama3.2:8b      - Latest Llama 3.2 (8B params, balanced)"
        echo "  • mistral:7b       - Mistral 7B (excellent performance)"
        echo "  • codellama:7b     - Code-focused Llama model"
        echo "  • qwen2.5:7b       - Qwen 2.5 (multilingual)"
        echo ""
        echo "Example: resource-ollama pull-model llama3.2:3b"
        return 1
    fi
    
    if command -v ollama &>/dev/null; then
        ollama pull "$model"
    else
        log::error "Ollama CLI not available"
        return 1
    fi
}

# Remove a model
ollama_remove_model() {
    local model="${1:-}"
    if [[ -z "$model" ]]; then
        log::error "Model name required"
        echo "Usage: resource-ollama remove-model <model-name>"
        echo "Example: resource-ollama remove-model llama3.2:3b"
        return 1
    fi
    
    if command -v ollama &>/dev/null; then
        ollama rm "$model"
    else
        log::error "Ollama CLI not available"
        return 1
    fi
}

# Start interactive chat with a model
ollama_chat() {
    local model="${1:-}"
    if [[ -z "$model" ]]; then
        log::error "Model name required"
        echo "Usage: resource-ollama chat <model-name>"
        echo ""
        echo "Available models:"
        ollama_list_models
        echo ""
        echo "Example: resource-ollama chat llama3.2:3b"
        return 1
    fi
    
    if command -v ollama &>/dev/null; then
        log::info "Starting chat with $model (type /bye to exit)"
        ollama run "$model"
    else
        log::error "Ollama CLI not available"
        return 1
    fi
}

# Generate text using a model
ollama_generate() {
    local model="${1:-}"
    local prompt="${2:-}"
    
    if [[ -z "$model" ]]; then
        log::error "Model name required"
        echo "Usage: resource-ollama generate <model-name> \"<prompt>\""
        echo "Example: resource-ollama generate llama3.2:3b \"Write a haiku about AI\""
        return 1
    fi
    
    if [[ -z "$prompt" ]]; then
        log::error "Prompt required"
        echo "Usage: resource-ollama generate <model-name> \"<prompt>\""
        echo "Example: resource-ollama generate llama3.2:3b \"Write a haiku about AI\""
        return 1
    fi
    
    if command -v ollama &>/dev/null; then
        ollama run "$model" "$prompt"
    else
        log::error "Ollama CLI not available"
        return 1
    fi
}

# Show details about a model
ollama_show_model() {
    local model="${1:-}"
    if [[ -z "$model" ]]; then
        log::error "Model name required"
        echo "Usage: resource-ollama show-model <model-name>"
        echo "Example: resource-ollama show-model llama3.2:3b"
        return 1
    fi
    
    if command -v ollama &>/dev/null; then
        ollama show "$model"
    else
        log::error "Ollama CLI not available"
        return 1
    fi
}

# Show detailed service information
ollama_info() {
    log::header "Ollama Service Information"
    local port="${OLLAMA_PORT:-11434}"
    echo "Port: $port"
    echo "API Base: http://localhost:$port"
    echo "Health Endpoint: http://localhost:$port/api/version"
    
    if command -v ollama &>/dev/null; then
        echo "CLI Version: $(ollama --version 2>/dev/null || echo 'Unknown')"
    fi
}

# Custom help function with Ollama-specific examples
ollama_show_help() {
    # Show standard framework help first
    cli::_handle_help
    
    # Add Ollama-specific examples
    echo ""
    echo "🤖 Ollama AI Examples:"
    echo ""
    echo "Model Management:"
    echo "  resource-ollama pull-model llama3.2:3b"
    echo "  resource-ollama list-models"
    echo "  resource-ollama show-model llama3.2:3b"
    echo "  resource-ollama remove-model old-model:tag"
    echo ""
    echo "AI Interaction:"
    echo "  resource-ollama chat llama3.2:3b"
    echo "  resource-ollama generate mistral:7b \"Explain quantum computing\""
    echo "  resource-ollama generate codellama:7b \"Write a Python function to sort a list\""
    echo ""
    echo "Popular Models to Try:"
    echo "  • llama3.2:3b      - Fast, general-purpose (3B parameters)"
    echo "  • llama3.2:8b      - Balanced performance (8B parameters)"
    echo "  • mistral:7b       - Excellent reasoning and following instructions"
    echo "  • codellama:7b     - Specialized for code generation and analysis"
    echo "  • qwen2.5:7b       - Great for multilingual tasks"
    echo ""
    echo "Service Management:"
    echo "  resource-ollama status"
    echo "  resource-ollama start"
    echo "  resource-ollama info"
    echo ""
    echo "For more models: https://ollama.com/library"
}

################################################################################
# Main execution - dispatch to framework
################################################################################

# Only execute if script is run directly (not sourced)
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    cli::dispatch "$@"
fi