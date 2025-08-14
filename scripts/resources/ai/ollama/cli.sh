#!/usr/bin/env bash
################################################################################
# Ollama Resource CLI
# 
# Lightweight CLI interface for Ollama that delegates to existing lib functions.
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

# Source utilities first
# shellcheck disable=SC1091
source "${VROOLI_ROOT}/scripts/lib/utils/var.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${var_LOG_FILE:-${VROOLI_ROOT}/scripts/lib/utils/log.sh}" 2>/dev/null || true
# shellcheck disable=SC1091
source "${var_RESOURCES_COMMON_FILE}" 2>/dev/null || true

# Source the CLI template
# shellcheck disable=SC1091
source "${VROOLI_ROOT}/scripts/lib/resources/cli/resource-cli-template.sh"

# Source Ollama configuration
# shellcheck disable=SC1091
source "${OLLAMA_CLI_DIR}/config/defaults.sh" 2>/dev/null || true
# Set defaults if not already set
OLLAMA_PORT="${OLLAMA_PORT:-11434}"
OLLAMA_BASE_URL="${OLLAMA_BASE_URL:-http://localhost:11434}"
OLLAMA_CONTAINER_NAME="${OLLAMA_CONTAINER_NAME:-ollama}"
ollama::export_config 2>/dev/null || true

# Source Ollama libraries
for lib in common api models status install; do
    lib_file="${OLLAMA_CLI_DIR}/lib/${lib}.sh"
    if [[ -f "$lib_file" ]]; then
        # shellcheck disable=SC1090
        source "$lib_file" 2>/dev/null || true
    fi
done

# Initialize with resource name
resource_cli::init "ollama"

################################################################################
# Delegate to existing Ollama functions
################################################################################

# Validate Ollama configuration
resource_cli::validate() {
    if command -v ollama::api::health &>/dev/null; then
        ollama::api::health
    else
        # Basic validation
        log::header "Validating Ollama"
        if curl -s "http://localhost:${OLLAMA_PORT}/api/version" >/dev/null 2>&1; then
            log::success "Ollama API is accessible"
        else
            log::error "Ollama API not accessible on port ${OLLAMA_PORT}"
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
        if curl -s "http://localhost:${OLLAMA_PORT}/api/version" >/dev/null 2>&1; then
            echo "Service: ✅ Running on port ${OLLAMA_PORT}"
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
    DRY_RUN="${DRY_RUN:-false}"
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log::info "[DRY RUN] Would start Ollama"
        return 0
    fi
    
    if command -v ollama::install::start_service &>/dev/null; then
        ollama::install::start_service
    else
        # Basic start using systemctl if available
        if command -v systemctl &>/dev/null; then
            sudo systemctl start ollama 2>/dev/null || log::error "Failed to start Ollama service"
        else
            log::error "Service management not available"
            return 1
        fi
    fi
}

# Stop Ollama
resource_cli::stop() {
    DRY_RUN="${DRY_RUN:-false}"
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log::info "[DRY RUN] Would stop Ollama"
        return 0
    fi
    
    if command -v ollama::install::stop_service &>/dev/null; then
        ollama::install::stop_service
    else
        # Basic stop using systemctl if available
        if command -v systemctl &>/dev/null; then
            sudo systemctl stop ollama 2>/dev/null || log::error "Failed to stop Ollama service"
        else
            log::error "Service management not available"
            return 1
        fi
    fi
}

# Install Ollama
resource_cli::install() {
    DRY_RUN="${DRY_RUN:-false}"
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log::info "[DRY RUN] Would install Ollama"
        return 0
    fi
    
    if command -v ollama::install::main &>/dev/null; then
        ollama::install::main
    else
        log::error "ollama::install::main not available"
        return 1
    fi
}

# Uninstall Ollama
resource_cli::uninstall() {
    FORCE="${FORCE:-false}"
    DRY_RUN="${DRY_RUN:-false}"
    
    if [[ "$FORCE" != "true" ]]; then
        echo "⚠️  This will remove Ollama and all its models. Use --force to confirm."
        return 1
    fi
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log::info "[DRY RUN] Would uninstall Ollama"
        return 0
    fi
    
    if command -v ollama::install::uninstall &>/dev/null; then
        ollama::install::uninstall true
    else
        # Basic uninstall
        if command -v systemctl &>/dev/null; then
            sudo systemctl stop ollama 2>/dev/null || true
            sudo systemctl disable ollama 2>/dev/null || true
        fi
        
        # Remove binary if it exists
        sudo rm -f /usr/local/bin/ollama 2>/dev/null || true
        log::success "Ollama uninstalled"
    fi
}

# Inject data into Ollama (pull model or execute command)
resource_cli::inject() {
    local command="${1:-}"
    DRY_RUN="${DRY_RUN:-false}"
    
    if [[ -z "$command" ]]; then
        log::error "Ollama command required for injection"
        echo "Usage: resource-ollama inject '<model-name>' or '<ollama-command>'"
        echo "Example: resource-ollama inject 'llama3.1:8b'"
        echo "Example: resource-ollama inject 'generate llama3.1:8b \"Hello world\"'"
        return 1
    fi
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log::info "[DRY RUN] Would execute: ollama $command"
        return 0
    fi
    
    # If command looks like a model name (no spaces), pull it
    if [[ "$command" =~ ^[a-zA-Z0-9._:-]+$ ]]; then
        log::info "Pulling model: $command"
        ollama pull "$command"
    else
        # Otherwise, execute as ollama command
        ollama $command
    fi
}

# Get credentials for n8n integration
resource_cli::credentials() {
    # Source credentials utilities
    # shellcheck disable=SC1091
    source "${VROOLI_ROOT}/scripts/resources/lib/credentials-utils.sh"
    
    # Parse arguments
    credentials::parse_args "$@"
    local parse_result=$?
    if [[ $parse_result -eq 2 ]]; then
        credentials::show_help "ollama"
        return 0
    elif [[ $parse_result -ne 0 ]]; then
        return 1
    fi
    
    # Get resource status by testing API endpoint
    local status="stopped"
    if curl -s "http://localhost:${OLLAMA_PORT}/api/version" >/dev/null 2>&1; then
        status="running"
    fi
    
    # Build connections array for Ollama API
    local connections_array="[]"
    if [[ "$status" == "running" ]]; then
        # Create connection for the Ollama API
        local connection_obj
        connection_obj=$(jq -n \
            --arg host "localhost" \
            --argjson port "${OLLAMA_PORT}" \
            --arg path "/api" \
            '{
                host: $host,
                port: $port,
                path: $path,
                ssl: false
            }')
        
        # Get available models for metadata
        local models_list="[]"
        if command -v ollama &>/dev/null; then
            models_list=$(ollama list --format json 2>/dev/null | jq -c '[.models[].name]' 2>/dev/null || echo '[]')
        fi
        
        local metadata_obj
        metadata_obj=$(jq -n \
            --arg description "Ollama AI inference API" \
            --argjson capabilities '["generate", "embed", "chat", "completion"]' \
            --arg version "latest" \
            --argjson models "$models_list" \
            '{
                description: $description,
                capabilities: $capabilities,
                version: $version,
                models: $models
            }')
        
        # No authentication needed for local Ollama
        local auth_obj="{}"
        
        local connection
        connection=$(credentials::build_connection \
            "api" \
            "Ollama API" \
            "ollama" \
            "$connection_obj" \
            "$auth_obj" \
            "$metadata_obj")
        
        connections_array=$(echo "$connection" | jq -s '.')
    fi
    
    # Build and validate response
    local response
    response=$(credentials::build_response "ollama" "$status" "$connections_array")
    
    if credentials::validate_json "$response"; then
        credentials::format_output "$response"
    else
        log::error "Invalid credentials JSON generated"
        return 1
    fi
}

################################################################################
# Ollama-specific commands (if functions exist)
################################################################################

# List installed models
ollama_list_models() {
    if command -v ollama::models::list &>/dev/null; then
        ollama::models::list
    elif command -v ollama &>/dev/null; then
        ollama list
    else
        log::error "Ollama CLI not available"
        return 1
    fi
}

# Pull a model
ollama_pull_model() {
    local model_name="${1:-}"
    
    if [[ -z "$model_name" ]]; then
        log::error "Model name required"
        echo "Example: resource-ollama pull-model llama3.1:8b"
        return 1
    fi
    
    if command -v ollama::models::pull &>/dev/null; then
        ollama::models::pull "$model_name"
    elif command -v ollama &>/dev/null; then
        ollama pull "$model_name"
    else
        log::error "Ollama CLI not available"
        return 1
    fi
}

# Remove a model
ollama_remove_model() {
    local model_name="${1:-}"
    FORCE="${FORCE:-false}"
    
    if [[ -z "$model_name" ]]; then
        log::error "Model name required"
        return 1
    fi
    
    if [[ "$FORCE" != "true" ]]; then
        echo "⚠️  This will remove model: $model_name. Use --force to confirm."
        return 1
    fi
    
    if command -v ollama::models::remove &>/dev/null; then
        ollama::models::remove "$model_name"
    elif command -v ollama &>/dev/null; then
        ollama rm "$model_name"
    else
        log::error "Ollama CLI not available"
        return 1
    fi
}

# Generate text using a model
ollama_generate() {
    local model_name="${1:-}"
    local prompt="${2:-}"
    
    if [[ -z "$model_name" || -z "$prompt" ]]; then
        log::error "Model name and prompt required"
        echo "Usage: resource-ollama generate <model> '<prompt>'"
        echo "Example: resource-ollama generate llama3.1:8b 'Hello, how are you?'"
        return 1
    fi
    
    if command -v ollama &>/dev/null; then
        ollama generate "$model_name" "$prompt"
    else
        log::error "Ollama CLI not available"
        return 1
    fi
}

# Show help
resource_cli::show_help() {
    cat << EOF
🚀 Ollama Resource CLI

USAGE:
    resource-ollama <command> [options]

CORE COMMANDS:
    inject <model|cmd>      Pull model or execute Ollama command
    validate                Validate Ollama configuration
    status                  Show Ollama status
    start                   Start Ollama service
    stop                    Stop Ollama service
    install                 Install Ollama
    uninstall               Uninstall Ollama (requires --force)
    credentials             Get connection credentials for n8n integration
    
OLLAMA COMMANDS:
    list-models             List installed models
    pull-model <model>      Pull/download a model
    remove-model <model>    Remove a model (requires --force)
    generate <model> '<prompt>'  Generate text using a model

OPTIONS:
    --verbose, -v           Show detailed output
    --dry-run               Preview actions without executing
    --force                 Force operation (skip confirmations)

EXAMPLES:
    resource-ollama status
    resource-ollama inject llama3.1:8b
    resource-ollama credentials --format pretty
    resource-ollama list-models
    resource-ollama generate llama3.1:8b "Explain quantum computing"
    resource-ollama remove-model codellama:7b --force

For more information: https://docs.vrooli.com/resources/ollama
EOF
}

# Main command router
resource_cli::main() {
    # Parse common options first
    local remaining_args
    remaining_args=$(resource_cli::parse_options "$@")
    set -- $remaining_args
    
    local command="${1:-help}"
    shift || true
    
    case "$command" in
        # Standard resource commands
        inject|validate|status|start|stop|install|uninstall|credentials)
            resource_cli::$command "$@"
            ;;
            
        # Ollama-specific commands
        list-models)
            ollama_list_models "$@"
            ;;
        pull-model)
            ollama_pull_model "$@"
            ;;
        remove-model)
            ollama_remove_model "$@"
            ;;
        generate)
            ollama_generate "$@"
            ;;
            
        help|--help|-h)
            resource_cli::show_help
            ;;
        *)
            log::error "Unknown command: $command"
            echo ""
            resource_cli::show_help
            exit 1
            ;;
    esac
}

# Run main if executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    resource_cli::main "$@"
fi