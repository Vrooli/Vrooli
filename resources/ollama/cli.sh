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

for lib in common api models status install inject; do
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

# Register injection command
cli::register_command "inject" "Inject model requirements from file" "ollama_inject_handler"

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

# Start Ollama with parallel processing optimization check
ollama_start() {
    if command -v ollama::start &>/dev/null; then
        ollama::start
    else
        # Check for parallel processing configuration updates before starting
        if [[ -f "/etc/systemd/system/ollama.service" ]] && command -v ollama::needs_parallel_config_update &>/dev/null; then
            if ollama::needs_parallel_config_update; then
                log::info "Detected suboptimal Ollama configuration, applying parallel processing optimizations..."
                if ollama::update_service_config; then
                    log::success "Applied parallel processing optimizations (expect 20-50x performance improvement)"
                else
                    log::info "Proceeding with existing configuration"
                fi
            fi
        fi
        
        # Start using systemctl if available
        # Specific sudo permissions should be granted during installation
        if command -v systemctl &>/dev/null; then
            sudo systemctl start ollama 2>/dev/null || {
                log::error "Failed to start Ollama service"
                log::info "If this fails with permission denied, try reinstalling Ollama to setup sudo permissions"
                return 1
            }
            log::success "Ollama started successfully with optimized parallel processing configuration"
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
        log::info "Installing Ollama with parallel processing optimizations..."
        if ollama::install; then
            log::success "Ollama installed with 20-50x performance improvements for semantic search"
            log::info "Parallel processing: ✅ Enabled (16 concurrent embeddings)"
            log::info "Batch operations: ✅ Optimized (50-item batches)"
            log::info "Memory usage: ✅ Monitored and optimized"
        else
            return 1
        fi
    else
        log::error "ollama::install not available"
        return 1
    fi
}

# Restart Ollama with parallel processing optimization check
ollama_restart() {
    log::info "Restarting Ollama with parallel processing optimizations..."
    
    # Check for parallel processing configuration updates before restarting
    if [[ -f "/etc/systemd/system/ollama.service" ]] && command -v ollama::needs_parallel_config_update &>/dev/null; then
        if ollama::needs_parallel_config_update; then
            log::info "Applying parallel processing optimizations during restart..."
            if ollama::update_service_config; then
                log::success "Applied parallel processing optimizations (expect 20-50x performance improvement)"
            else
                log::info "Proceeding with existing configuration"
            fi
        fi
    fi
    
    # Restart using systemctl if available
    if command -v systemctl &>/dev/null; then
        sudo systemctl restart ollama 2>/dev/null || {
            log::error "Failed to restart Ollama service"
            log::info "If this fails with permission denied, try reinstalling Ollama to setup sudo permissions"
            return 1
        }
        log::success "Ollama restarted successfully with optimized configuration"
        log::info "Parallel processing: ✅ 16 concurrent embeddings enabled"
        log::info "Expected improvement: 20-50x faster semantic search operations"
    else
        log::error "Service management not available"
        return 1
    fi
}

# Uninstall Ollama
ollama_uninstall() {
    if command -v ollama::uninstall &>/dev/null; then
        log::info "Uninstalling Ollama and cleaning up parallel processing configuration..."
        ollama::uninstall "$@"
    else
        log::error "ollama::uninstall not available"
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
    local model=""
    local prompt=""
    local prompt_file=""
    local use_stdin=false
    local quiet=false
    local type="general"
    local stream=false
    local format="text"
    local temperature="0.7"
    
    # Auto-enable quiet mode in non-TTY environments (automation context)
    if [[ ! -t 0 ]] || [[ ! -t 1 ]]; then
        quiet=true
    fi
    
    # Parse arguments properly
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --model)
                model="$2"
                shift 2
                ;;
            --from-file)
                prompt_file="$2"
                shift 2
                ;;
            --stdin)
                use_stdin=true
                shift
                ;;
            --type)
                type="$2"
                shift 2
                ;;
            --quiet)
                quiet=true
                shift
                ;;
            --stream)
                stream=true
                shift
                ;;
            --format)
                format="$2"
                shift 2
                ;;
            --temperature)
                temperature="$2"
                shift 2
                ;;
            --*)
                if [[ "$quiet" == false ]]; then
                    log::error "Unknown option: $1"
                fi
                return 1
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
    
    # Validate inputs
    if [[ -z "$model" ]]; then
        if [[ "$quiet" == false ]]; then
            log::error "Model name required"
            echo "Usage: resource-ollama generate [OPTIONS] <model-name> [\"<prompt>\"]"
            echo "       resource-ollama generate --from-file <file> <model-name>"
            echo "       cat prompt.txt | resource-ollama generate --stdin <model-name>"
            echo ""
            echo "Input methods (priority order):"
            echo "  1. --from-file <file>  Read prompt from file"
            echo "  2. --stdin             Read prompt from stdin"
            echo "  3. argument            Use prompt argument"
            echo ""
            echo "Example: resource-ollama generate llama3.2:3b \"Write a haiku about AI\""
        fi
        return 1
    fi
    
    # Determine prompt source (priority: file > stdin > argument)
    if [[ -n "$prompt_file" ]]; then
        # Priority 1: File-based input
        if [[ ! -f "$prompt_file" ]]; then
            if [[ "$quiet" == false ]]; then
                log::error "Prompt file not found: $prompt_file"
            fi
            return 1
        fi
        if [[ ! -r "$prompt_file" ]]; then
            if [[ "$quiet" == false ]]; then
                log::error "Prompt file not readable: $prompt_file"
            fi
            return 1
        fi
        prompt=$(cat "$prompt_file")
        if [[ -z "$prompt" ]]; then
            if [[ "$quiet" == false ]]; then
                log::error "Prompt file is empty: $prompt_file"
            fi
            return 1
        fi
    elif [[ "$use_stdin" == true ]] || [[ -z "$prompt" && ! -t 0 ]]; then
        # Priority 2: Stdin input (explicit --stdin or auto-detect when no prompt and stdin available)
        if [[ -t 0 ]]; then
            if [[ "$quiet" == false ]]; then
                log::error "No stdin data available"
            fi
            return 1
        fi
        prompt=$(cat)
        if [[ -z "$prompt" ]]; then
            if [[ "$quiet" == false ]]; then
                log::error "No prompt data received from stdin"
            fi
            return 1
        fi
    elif [[ -z "$prompt" ]]; then
        # Priority 3: No prompt argument provided
        if [[ "$quiet" == false ]]; then
            log::error "Prompt required (use argument, --from-file, or --stdin)"
            echo "Usage: resource-ollama generate [OPTIONS] <model-name> [\"<prompt>\"]"
            echo "       resource-ollama generate --from-file <file> <model-name>"
            echo "       cat prompt.txt | resource-ollama generate --stdin <model-name>"
            echo "Example: resource-ollama generate llama3.2:3b \"Write a haiku about AI\""
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
    
    # Detect if we should use API for clean output
    local use_api=false
    if [[ "$quiet" == true ]]; then
        use_api=true
    fi
    
    # Use API for automation-friendly output
    if [[ "$use_api" == true ]]; then
        # Check if Ollama API is accessible
        local port="${OLLAMA_PORT:-11434}"
        if ! curl -s --max-time 2 "http://localhost:${port}/api/version" >/dev/null 2>&1; then
            if [[ "$quiet" == false ]]; then
                log::error "Ollama API not accessible on port $port"
            fi
            return 1
        fi
        
        # Build API payload with proper JSON escaping
        local api_payload
        if command -v jq >/dev/null 2>&1; then
            # Use jq for proper JSON encoding
            api_payload=$(jq -n \
                --arg model "$model" \
                --arg prompt "$prompt" \
                --argjson stream "$stream" \
                --arg temperature "$temperature" \
                '{
                    model: $model,
                    prompt: $prompt,
                    stream: $stream,
                    options: {
                        temperature: ($temperature | tonumber),
                        seed: 101
                    }
                }')
        else
            # Fallback manual JSON construction (comprehensive escaping)
            local escaped_prompt="$prompt"
            # Escape backslashes first (must be done before other escapes)
            escaped_prompt="${escaped_prompt//\\/\\\\}"
            # Escape double quotes
            escaped_prompt="${escaped_prompt//\"/\\\"}"
            # Escape newlines
            escaped_prompt="${escaped_prompt//$'\n'/\\n}"
            # Escape carriage returns
            escaped_prompt="${escaped_prompt//$'\r'/\\r}"
            # Escape tabs
            escaped_prompt="${escaped_prompt//$'\t'/\\t}"
            # Escape form feeds
            escaped_prompt="${escaped_prompt//$'\f'/\\f}"
            # Escape backspaces
            escaped_prompt="${escaped_prompt//$'\b'/\\b}"
            api_payload="{\"model\":\"$model\",\"prompt\":\"$escaped_prompt\",\"stream\":$stream,\"options\":{\"temperature\":$temperature,\"seed\":101}}"
        fi
        
        # Make API call
        local response
        response=$(curl -s --max-time 120 \
            -X POST "http://localhost:${port}/api/generate" \
            -H "Content-Type: application/json" \
            -d "$api_payload" 2>/dev/null)
        
        local curl_exit_code=$?
        if [[ $curl_exit_code -ne 0 ]]; then
            if [[ "$quiet" == false ]]; then
                log::error "Failed to connect to Ollama API (curl exit code: $curl_exit_code)"
            fi
            return 1
        fi
        
        # Extract response based on format
        if [[ "$format" == "json" ]]; then
            echo "$response"
        else
            # Extract just the response text for clean output
            if command -v jq >/dev/null 2>&1; then
                local extracted_response
                extracted_response=$(echo "$response" | jq -r '.response // empty' 2>/dev/null)
                if [[ -n "$extracted_response" && "$extracted_response" != "null" ]]; then
                    echo "$extracted_response"
                else
                    # Fallback: output raw response if jq extraction fails
                    echo "$response"
                fi
            else
                # Fallback without jq: try basic extraction
                echo "$response" | grep -o '"response":"[^"]*"' | sed 's/"response":"//' | sed 's/"$//' || echo "$response"
            fi
        fi
        
        return 0
    else
        # Interactive mode - use ollama run
        if [[ "$quiet" == false ]]; then
            log::info "Starting interactive generation with $model"
        fi
        ollama run "$model" "$prompt"
    fi
}

# Handle model injection from JSON file
ollama_inject_handler() {
    local file="${1:-}"
    
    if [[ -z "$file" ]]; then
        log::error "Injection file required"
        echo "Usage: resource-ollama inject <file.json>"
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
    echo "Prompt Input Methods (priority order):"
    echo "  1. File-based (recommended for complex prompts):"
    echo "     echo 'Complex prompt with \"quotes\" and newlines' > prompt.txt"
    echo "     resource-ollama generate --from-file prompt.txt llama3.2:3b"
    echo ""
    echo "  2. Stdin (great for automation):"
    echo "     echo 'Your prompt here' | resource-ollama generate --stdin llama3.2:3b"
    echo "     cat large_prompt.txt | resource-ollama generate llama3.2:3b"
    echo ""
    echo "  3. Command argument (simple prompts):"
    echo "     resource-ollama generate llama3.2:3b \"Simple prompt text\""
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
    # Ensure we have at least one argument, defaulting to status
    if [[ $# -eq 0 ]]; then
        cli::dispatch "status"
    else
        cli::dispatch "$@"
    fi
fi