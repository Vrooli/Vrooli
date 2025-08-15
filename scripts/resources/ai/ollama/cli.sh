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

# Get script directory
OLLAMA_CLI_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
VROOLI_ROOT="${VROOLI_ROOT:-$(cd "$OLLAMA_CLI_DIR/../../../.." && pwd)}"
export VROOLI_ROOT
export RESOURCE_DIR="$OLLAMA_CLI_DIR"
export OLLAMA_SCRIPT_DIR="$OLLAMA_CLI_DIR"  # For compatibility with existing libs

# Source utilities
# shellcheck disable=SC1091
source "${VROOLI_ROOT}/scripts/lib/utils/var.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${var_LOG_FILE:-${VROOLI_ROOT}/scripts/lib/utils/log.sh}" 2>/dev/null || true
# shellcheck disable=SC1091
source "${var_RESOURCES_COMMON_FILE}" 2>/dev/null || true

# Source the CLI Command Framework
# shellcheck disable=SC1091
source "${VROOLI_ROOT}/scripts/resources/lib/cli-command-framework.sh"

# Source Ollama configuration and libraries
# shellcheck disable=SC1091
source "${OLLAMA_CLI_DIR}/config/defaults.sh" 2>/dev/null || true
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
cli::register_command "help" "Show this help message with Ollama examples" "resource_cli::show_help"

# Register additional Ollama-specific commands
cli::register_command "list-models" "List installed models" "resource_cli::list_models"
cli::register_command "pull-model" "Pull a model from registry" "resource_cli::pull_model"
cli::register_command "remove-model" "Remove a model" "resource_cli::remove_model" "modifies-system"
cli::register_command "chat" "Start interactive chat with a model" "resource_cli::chat"
cli::register_command "generate" "Generate text using a model" "resource_cli::generate"
cli::register_command "show-model" "Show details about a model" "resource_cli::show_model"
cli::register_command "info" "Show detailed service information" "resource_cli::info"

################################################################################
# Resource-specific command implementations
################################################################################

# Validate Ollama configuration
resource_cli::validate() {
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
resource_cli::status() {
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
resource_cli::start() {
    if command -v ollama::install::start_service &>/dev/null; then
        ollama::install::start_service
    else
        # Basic start using systemctl if available
        if command -v systemctl &>/dev/null; then
            sudo systemctl start ollama 2>/dev/null || {
                log::error "Failed to start Ollama service"
                return 1
            }
        else
            log::error "Service management not available"
            return 1
        fi
    fi
}

# Stop Ollama
resource_cli::stop() {
    if command -v ollama::install::stop_service &>/dev/null; then
        ollama::install::stop_service
    else
        # Basic stop using systemctl if available
        if command -v systemctl &>/dev/null; then
            sudo systemctl stop ollama 2>/dev/null || {
                log::error "Failed to stop Ollama service"
                return 1
            }
        else
            log::error "Service management not available"
            return 1
        fi
    fi
}

# Install Ollama
resource_cli::install() {
    if command -v ollama::install::main &>/dev/null; then
        ollama::install::main
    else
        log::error "ollama::install::main not available"
        return 1
    fi
}

# List installed models
resource_cli::list_models() {
    if command -v ollama &>/dev/null; then
        ollama list
    else
        log::error "Ollama CLI not available"
        return 1
    fi
}

# Pull a model from registry
resource_cli::pull_model() {
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
resource_cli::remove_model() {
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
resource_cli::chat() {
    local model="${1:-}"
    if [[ -z "$model" ]]; then
        log::error "Model name required"
        echo "Usage: resource-ollama chat <model-name>"
        echo ""
        echo "Available models:"
        resource_cli::list_models
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
resource_cli::generate() {
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
resource_cli::show_model() {
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
resource_cli::info() {
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
resource_cli::show_help() {
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