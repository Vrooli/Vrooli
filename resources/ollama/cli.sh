#!/usr/bin/env bash
################################################################################
# Ollama Resource CLI - v2.0 Universal Contract Compliant
# 
# Local AI model inference server for running large language models
#
# Usage:
#   resource-ollama <command> [options]
#   resource-ollama <group> <subcommand> [options]
#
################################################################################

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../.." && builtin pwd)}"
# Handle symlinks for installed CLI
if [[ -L "${BASH_SOURCE[0]}" ]]; then
    OLLAMA_CLI_SCRIPT="$(readlink -f "${BASH_SOURCE[0]}")"
    APP_ROOT="$(builtin cd "${OLLAMA_CLI_SCRIPT%/*}/../.." && builtin pwd)"
fi
OLLAMA_CLI_DIR="${APP_ROOT}/resources/ollama"

# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "${var_RESOURCES_COMMON_FILE}"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/resources/lib/cli-command-framework-v2.sh"
# shellcheck disable=SC1091
source "${OLLAMA_CLI_DIR}/config/defaults.sh"

# Source Ollama libraries
for lib in common api models status install inject; do
    lib_file="${OLLAMA_CLI_DIR}/lib/${lib}.sh"
    if [[ -f "$lib_file" ]]; then
        # shellcheck disable=SC1090
        source "$lib_file" 2>/dev/null || true
    fi
done

# Initialize CLI framework in v2.0 mode (auto-creates manage/test/content groups)
cli::init "ollama" "Ollama AI inference server management" "v2"

# ==============================================================================
# REQUIRED HANDLERS - Direct mapping to existing functions
# ==============================================================================
CLI_COMMAND_HANDLERS["manage::install"]="ollama::install"
CLI_COMMAND_HANDLERS["manage::uninstall"]="ollama::uninstall"
CLI_COMMAND_HANDLERS["manage::start"]="ollama::start"  
CLI_COMMAND_HANDLERS["manage::stop"]="ollama::stop"
CLI_COMMAND_HANDLERS["manage::restart"]="ollama::restart"
CLI_COMMAND_HANDLERS["test::smoke"]="ollama::test"

# Content handlers - AI business operations
CLI_COMMAND_HANDLERS["content::add"]="ollama_pull_model"
CLI_COMMAND_HANDLERS["content::list"]="ollama_list_models"
CLI_COMMAND_HANDLERS["content::get"]="ollama_show_model"
CLI_COMMAND_HANDLERS["content::remove"]="ollama_remove_model"
CLI_COMMAND_HANDLERS["content::execute"]="ollama_generate"

# ==============================================================================
# REQUIRED INFORMATION COMMANDS
# ==============================================================================
cli::register_command "status" "Show detailed resource status" "ollama::status"
cli::register_command "logs" "Show resource logs" "ollama::logs"

# ==============================================================================
# OLLAMA-SPECIFIC CONTENT COMMANDS (AI Model Operations)
# ==============================================================================
# Model management - these are the critical AI features
cli::register_subcommand "content" "pull" "Pull AI model from registry" "ollama_pull_model" "modifies-system"
cli::register_subcommand "content" "show" "Show details about a model" "ollama_show_model"
cli::register_subcommand "content" "chat" "Start interactive chat with model" "ollama_chat"
cli::register_subcommand "content" "generate" "Generate text using model" "ollama_generate"
cli::register_subcommand "content" "inject" "Inject models from JSON file" "ollama_inject_handler" "modifies-system"

# ==============================================================================
# OPTIONAL RESOURCE-SPECIFIC COMMANDS
# ==============================================================================
cli::register_command "info" "Show comprehensive Ollama information" "ollama::info"

################################################################################
# Legacy wrapper functions - preserve ALL original functionality exactly
################################################################################

ollama_list_models() {
    if command -v ollama &>/dev/null; then
        ollama list
    else
        log::error "Ollama CLI not available"
        return 1
    fi
}

ollama_pull_model() {
    local model="${1:-}"
    if [[ -z "$model" ]]; then
        log::error "Model name required"
        echo "Usage: resource-ollama content pull <model-name>"
        echo ""
        echo "Popular models:"
        echo "  • llama3.2:3b      - Latest Llama 3.2 (3B params, fast)"
        echo "  • llama3.2:8b      - Latest Llama 3.2 (8B params, balanced)"
        echo "  • mistral:7b       - Mistral 7B (excellent performance)"
        echo "  • codellama:7b     - Code-focused Llama model"
        echo "  • qwen2.5:7b       - Qwen 2.5 (multilingual)"
        echo ""
        echo "Example: resource-ollama content pull llama3.2:3b"
        return 1
    fi
    
    if command -v ollama &>/dev/null; then
        ollama pull "$model"
    else
        log::error "Ollama CLI not available"
        return 1
    fi
}

ollama_remove_model() {
    local model="${1:-}"
    if [[ -z "$model" ]]; then
        log::error "Model name required"
        echo "Usage: resource-ollama content remove <model-name>"
        echo "Example: resource-ollama content remove llama3.2:3b"
        return 1
    fi
    
    if command -v ollama &>/dev/null; then
        ollama rm "$model"
    else
        log::error "Ollama CLI not available"
        return 1
    fi
}

ollama_show_model() {
    local model="${1:-}"
    if [[ -z "$model" ]]; then
        log::error "Model name required"
        echo "Usage: resource-ollama content show <model-name>"
        echo "Example: resource-ollama content show llama3.2:3b"
        return 1
    fi
    
    if command -v ollama &>/dev/null; then
        ollama show "$model"
    else
        log::error "Ollama CLI not available"
        return 1
    fi
}

ollama_chat() {
    local model="${1:-}"
    if [[ -z "$model" ]]; then
        log::error "Model name required"
        echo "Usage: resource-ollama content chat <model-name>"
        echo ""
        echo "Available models:"
        ollama_list_models
        echo ""
        echo "Example: resource-ollama content chat llama3.2:3b"
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

ollama_generate() {
    local model=""
    local prompt=""
    local quiet=false
    
    # Auto-enable quiet mode in non-TTY environments (automation context)
    if [[ ! -t 0 ]] || [[ ! -t 1 ]]; then
        quiet=true
    fi
    
    # Simple argument parsing for core functionality
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --model)
                model="$2"
                shift 2
                ;;
            --quiet)
                quiet=true
                shift
                ;;
            *)
                if [[ -z "$model" ]]; then
                    model="$1"
                elif [[ -z "$prompt" ]]; then
                    prompt="$1"
                else
                    prompt="$prompt $1"
                fi
                shift
                ;;
        esac
    done
    
    # Check if we need to read from stdin
    if [[ -z "$prompt" && ! -t 0 ]]; then
        prompt=$(cat)
    fi
    
    # Validate inputs
    if [[ -z "$model" ]]; then
        if [[ "$quiet" == false ]]; then
            log::error "Model name required"
            echo "Usage: resource-ollama content generate <model-name> [\"<prompt>\"]"
            echo "Example: resource-ollama content generate llama3.2:3b \"Write a haiku about AI\""
        fi
        return 1
    fi
    
    if [[ -z "$prompt" ]]; then
        if [[ "$quiet" == false ]]; then
            log::error "Prompt required"
            echo "Usage: resource-ollama content generate <model-name> [\"<prompt>\"]"
            echo "Example: resource-ollama content generate llama3.2:3b \"Write a haiku about AI\""
        fi
        return 1
    fi
    
    # Check if Ollama is available
    if ! command -v ollama &>/dev/null; then
        if [[ "$quiet" == false ]]; then
            log::error "Ollama CLI not available"
        fi
        return 1
    fi
    
    # Use ollama run for generation
    if [[ "$quiet" == false ]]; then
        log::info "Generating text with $model"
    fi
    ollama run "$model" "$prompt"
}

ollama_inject_handler() {
    local file="${1:-}"
    
    if [[ -z "$file" ]]; then
        log::error "Injection file required"
        echo "Usage: resource-ollama content inject <file.json>"
        echo ""
        echo "Example file format:"
        echo '{'
        echo '  "models": ['
        echo '    {"name": "llama3.2:3b", "alias": "llama-small"},'
        echo '    {"name": "qwen2.5-coder:7b"}'
        echo '  ]'
        echo '}'
        return 1
    fi
    
    # Handle shared: prefix for scenario files
    if [[ "$file" == shared:* ]]; then
        file="${var_VROOLI_ROOT}/${file#shared:}"
    fi
    
    # Call the injection function
    if command -v ollama::inject &>/dev/null; then
        ollama::inject "$file"
    else
        log::error "Ollama injection function not available"
        return 1
    fi
}

# Only execute if script is run directly (not sourced)
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    cli::dispatch "$@"
fi