#!/usr/bin/env bash

# Ollama Resource Mock Implementation
# Provides comprehensive mocking for Ollama LLM service including CLI commands and HTTP API

# Prevent duplicate loading
if [[ "${OLLAMA_MOCK_LOADED:-}" == "true" ]]; then
    return 0
fi
export OLLAMA_MOCK_LOADED="true"

# ----------------------------
# Mock Configuration & State Management
# ----------------------------

# Default configuration
readonly OLLAMA_DEFAULT_PORT="11434"
readonly OLLAMA_DEFAULT_BASE_URL="http://localhost:11434"
readonly OLLAMA_DEFAULT_SERVICE_NAME="ollama"

# Initialize mock state file for subshell access
# Use a deterministic state file path that works across subshells
if [[ -z "${OLLAMA_MOCK_STATE_FILE:-}" ]]; then
    if [[ -n "${BATS_TMPDIR:-}" ]]; then
        OLLAMA_MOCK_STATE_FILE="${BATS_TMPDIR}/ollama_mock_state_test"
    else
        OLLAMA_MOCK_STATE_FILE="${TMPDIR:-/tmp}/ollama_mock_state_$$"
    fi
fi
export OLLAMA_MOCK_STATE_FILE

# Global state arrays
declare -gA MOCK_OLLAMA_MODELS=()
declare -gA MOCK_OLLAMA_MODEL_INFO=()  
declare -gA MOCK_OLLAMA_RUNNING_MODELS=()
declare -gA MOCK_OLLAMA_PULL_PROGRESS=()
declare -gA MOCK_OLLAMA_CALL_COUNT=()
declare -gA MOCK_OLLAMA_ERRORS=()

# Mock modes
export OLLAMA_MOCK_MODE="${OLLAMA_MOCK_MODE:-healthy}"  # healthy|unhealthy|installing|stopped|offline|error

# Configuration
export OLLAMA_PORT="${OLLAMA_PORT:-$OLLAMA_DEFAULT_PORT}"
export OLLAMA_BASE_URL="${OLLAMA_BASE_URL:-$OLLAMA_DEFAULT_BASE_URL}"
export OLLAMA_SERVICE_NAME="${OLLAMA_SERVICE_NAME:-$OLLAMA_DEFAULT_SERVICE_NAME}"

#######################################
# Initialize state file for subshell access
#######################################
_ollama_mock_init_state_file() {
    if [[ ! -f "$OLLAMA_MOCK_STATE_FILE" ]]; then
        # Ensure the directory exists
        local state_dir
        state_dir=$(dirname "$OLLAMA_MOCK_STATE_FILE")
        mkdir -p "$state_dir" 2>/dev/null || true
        
        cat > "$OLLAMA_MOCK_STATE_FILE" << 'EOF'
# Ollama mock state file
declare -gA MOCK_OLLAMA_MODELS=()
declare -gA MOCK_OLLAMA_MODEL_INFO=()  
declare -gA MOCK_OLLAMA_RUNNING_MODELS=()
declare -gA MOCK_OLLAMA_PULL_PROGRESS=()
declare -gA MOCK_OLLAMA_CALL_COUNT=()
declare -gA MOCK_OLLAMA_ERRORS=()
EOF
    fi
}

#######################################
# Save state to file for subshell access
#######################################
_ollama_mock_save_state() {
    # Ensure the directory exists
    local state_dir
    state_dir=$(dirname "$OLLAMA_MOCK_STATE_FILE")
    mkdir -p "$state_dir" 2>/dev/null || true
    
    {
        echo "# Ollama mock state file - auto generated"
        
        # Use a more conservative approach to avoid syntax errors
        
        # MOCK_OLLAMA_MODELS
        echo "declare -gA MOCK_OLLAMA_MODELS"
        for key in "${!MOCK_OLLAMA_MODELS[@]}"; do
            printf 'MOCK_OLLAMA_MODELS[%q]=%q\n' "$key" "${MOCK_OLLAMA_MODELS[$key]}"
        done
        
        # MOCK_OLLAMA_MODEL_INFO  
        echo "declare -gA MOCK_OLLAMA_MODEL_INFO"
        for key in "${!MOCK_OLLAMA_MODEL_INFO[@]}"; do
            printf 'MOCK_OLLAMA_MODEL_INFO[%q]=%q\n' "$key" "${MOCK_OLLAMA_MODEL_INFO[$key]}"
        done
        
        # MOCK_OLLAMA_RUNNING_MODELS
        echo "declare -gA MOCK_OLLAMA_RUNNING_MODELS"
        for key in "${!MOCK_OLLAMA_RUNNING_MODELS[@]}"; do
            printf 'MOCK_OLLAMA_RUNNING_MODELS[%q]=%q\n' "$key" "${MOCK_OLLAMA_RUNNING_MODELS[$key]}"
        done
        
        # MOCK_OLLAMA_PULL_PROGRESS
        echo "declare -gA MOCK_OLLAMA_PULL_PROGRESS"
        for key in "${!MOCK_OLLAMA_PULL_PROGRESS[@]}"; do
            printf 'MOCK_OLLAMA_PULL_PROGRESS[%q]=%q\n' "$key" "${MOCK_OLLAMA_PULL_PROGRESS[$key]}"
        done
        
        # MOCK_OLLAMA_CALL_COUNT
        echo "declare -gA MOCK_OLLAMA_CALL_COUNT"
        for key in "${!MOCK_OLLAMA_CALL_COUNT[@]}"; do
            printf 'MOCK_OLLAMA_CALL_COUNT[%q]=%q\n' "$key" "${MOCK_OLLAMA_CALL_COUNT[$key]}"
        done
        
        # MOCK_OLLAMA_ERRORS
        echo "declare -gA MOCK_OLLAMA_ERRORS"
        for key in "${!MOCK_OLLAMA_ERRORS[@]}"; do
            printf 'MOCK_OLLAMA_ERRORS[%q]=%q\n' "$key" "${MOCK_OLLAMA_ERRORS[$key]}"
        done
        
        # Export scalar variables
        echo "export OLLAMA_MOCK_MODE='$OLLAMA_MOCK_MODE'"
        echo "export OLLAMA_PORT='$OLLAMA_PORT'"
        echo "export OLLAMA_BASE_URL='$OLLAMA_BASE_URL'"
        echo "export OLLAMA_SERVICE_NAME='$OLLAMA_SERVICE_NAME'"
    } > "$OLLAMA_MOCK_STATE_FILE"
}

# Initialize state file only if it doesn't exist
if [[ ! -f "$OLLAMA_MOCK_STATE_FILE" ]]; then
    _ollama_mock_init_state_file
fi

# ----------------------------
# Utilities
# ----------------------------

#######################################
# Load state from file - use everywhere state is needed
#######################################
_ollama_load_state() {
    if [[ -f "$OLLAMA_MOCK_STATE_FILE" && -s "$OLLAMA_MOCK_STATE_FILE" ]]; then
        # Source the state file to load all variables
        source "$OLLAMA_MOCK_STATE_FILE" 2>/dev/null || true
    fi
}

_mock_current_timestamp() {
    date -u +"%Y-%m-%dT%H:%M:%S.%3NZ" 2>/dev/null || date -u +"%Y-%m-%dT%H:%M:%SZ"
}

_mock_model_digest() {
    local model="$1"
    echo "sha256:$(echo -n "$model" | md5sum 2>/dev/null | cut -c1-8 || echo "abc123")"
}

_mock_model_size() {
    local model="$1"
    case "$model" in
        *:1b) echo "1073741824" ;;     # 1GB
        *:3b) echo "3221225472" ;;     # 3GB  
        *:7b) echo "4294967296" ;;     # 4GB
        *:8b) echo "4831838208" ;;     # 4.5GB
        *:13b) echo "6979321856" ;;    # 6.5GB
        *:14b) echo "7516192768" ;;    # 7GB
        *:32b) echo "17179869184" ;;   # 16GB
        *) echo "4294967296" ;;        # Default 4GB
    esac
}

_mock_model_family() {
    local model="$1"
    case "$model" in
        llama*) echo "llama" ;;
        mistral*) echo "mistral" ;;
        deepseek*) echo "deepseek" ;;
        qwen*) echo "qwen" ;;
        phi*) echo "phi" ;;
        codellama*) echo "codellama" ;;
        *) echo "unknown" ;;
    esac
}

_mock_model_parameter_size() {
    local model="$1"
    if [[ "$model" =~ :([0-9.]+[bB])$ ]]; then
        echo "${BASH_REMATCH[1]^^}"  # Convert to uppercase
    else
        echo "7B"  # Default
    fi
}

# ----------------------------
# Public API Functions
# ----------------------------

#######################################
# Reset all mock state
#######################################
mock::ollama::reset() {
    # Clear all arrays completely
    unset MOCK_OLLAMA_MODELS
    unset MOCK_OLLAMA_MODEL_INFO  
    unset MOCK_OLLAMA_RUNNING_MODELS
    unset MOCK_OLLAMA_PULL_PROGRESS
    unset MOCK_OLLAMA_CALL_COUNT
    unset MOCK_OLLAMA_ERRORS
    
    # Redeclare arrays
    declare -gA MOCK_OLLAMA_MODELS=()
    declare -gA MOCK_OLLAMA_MODEL_INFO=()  
    declare -gA MOCK_OLLAMA_RUNNING_MODELS=()
    declare -gA MOCK_OLLAMA_PULL_PROGRESS=()
    declare -gA MOCK_OLLAMA_CALL_COUNT=()
    declare -gA MOCK_OLLAMA_ERRORS=()
    
    # Reset to default state
    export OLLAMA_MOCK_MODE="healthy"
    
    # Initialize with default models
    mock::ollama::add_model "llama3.1:8b" "2024-01-15T10:30:00Z"
    mock::ollama::add_model "deepseek-r1:8b" "2024-01-14T09:15:00Z" 
    mock::ollama::add_model "qwen2.5-coder:7b" "2024-01-13T14:45:00Z"
    
    # Save state to file for subshell access
    _ollama_mock_save_state
    
    echo "[MOCK] Ollama state reset with default models"
}

#######################################
# Set mock mode
# Arguments: $1 - mode (healthy|unhealthy|installing|stopped|offline|error)
#######################################
mock::ollama::set_mode() {
    local mode="$1"
    
    case "$mode" in
        healthy|unhealthy|installing|stopped|offline|error)
            export OLLAMA_MOCK_MODE="$mode"
            ;;
        *)
            echo "[MOCK] Invalid Ollama mode: $mode" >&2
            return 1
            ;;
    esac
    
    # Save state for subshell access
    _ollama_mock_save_state
    
    # Use centralized logging if available
    if command -v mock::log_state &>/dev/null; then
        mock::log_state "ollama_mode" "global" "$mode"
    fi
    
    return 0
}

#######################################
# Add model to mock registry
# Arguments: $1 - model name, $2 - modified timestamp (optional)
#######################################
mock::ollama::add_model() {
    local model="$1"
    local timestamp="${2:-$(_mock_current_timestamp)}"
    
    MOCK_OLLAMA_MODELS["$model"]="installed"
    MOCK_OLLAMA_MODEL_INFO["$model"]="$timestamp|$(_mock_model_size "$model")|$(_mock_model_digest "$model")|$(_mock_model_family "$model")|$(_mock_model_parameter_size "$model")"
    
    # Save state to file for subshell access
    _ollama_mock_save_state
    
    return 0
}

#######################################
# Remove model from mock registry
# Arguments: $1 - model name
#######################################
mock::ollama::remove_model() {
    local model="$1"
    
    unset MOCK_OLLAMA_MODELS["$model"]
    unset MOCK_OLLAMA_MODEL_INFO["$model"] 
    unset MOCK_OLLAMA_RUNNING_MODELS["$model"]
    
    # Save state to file for subshell access
    _ollama_mock_save_state
    
    return 0
}

#######################################
# Set model as running
# Arguments: $1 - model name, $2 - size (optional)
#######################################
mock::ollama::set_model_running() {
    local model="$1"
    local size="${2:-4.5 GB}"
    
    MOCK_OLLAMA_RUNNING_MODELS["$model"]="$size"
    
    # Save state to file for subshell access
    _ollama_mock_save_state
    
    return 0
}

#######################################
# Inject error for testing failure scenarios
# Arguments: $1 - command, $2 - error type (optional)
#######################################
mock::ollama::inject_error() {
    local cmd="$1"
    local error_type="${2:-generic}"
    
    MOCK_OLLAMA_ERRORS["$cmd"]="$error_type"
    
    # Save state to file for subshell access
    _ollama_mock_save_state
    
    echo "[MOCK] Injected error for $cmd: $error_type"
}

# ----------------------------
# Ollama CLI Command Mock Implementation
# ----------------------------

#######################################
# Main ollama command interceptor
#######################################
ollama() {
    # Ensure we load the latest state when called as main command  
    _ollama_load_state
    
    # Use centralized logging if available
    if command -v mock::log_and_verify &>/dev/null; then
        mock::log_and_verify "ollama" "$*"
    fi
    
    # Handle global mock modes first
    case "$OLLAMA_MOCK_MODE" in
        offline|stopped)
            echo "Error: Failed to connect to ollama server. Is ollama running?" >&2
            return 1
            ;;
        error)
            echo "ollama: command failed with unknown error" >&2
            return 1
            ;;
    esac
    
    local cmd="${1:-help}"
    shift || true
    
    # Track call count
    local count="${MOCK_OLLAMA_CALL_COUNT[$cmd]:-0}"
    MOCK_OLLAMA_CALL_COUNT["$cmd"]=$((count + 1))
    _ollama_mock_save_state
    
    # Check for injected errors
    if [[ -n "${MOCK_OLLAMA_ERRORS[$cmd]}" ]]; then
        local error_type="${MOCK_OLLAMA_ERRORS[$cmd]}"
        case "$error_type" in
            connection) echo "Error: Failed to connect to ollama server" >&2; return 1 ;;
            timeout) echo "Error: Request timed out" >&2; return 1 ;;
            not_found) echo "Error: model not found" >&2; return 1 ;;
            disk_space) echo "Error: insufficient disk space" >&2; return 1 ;;
            *) echo "Error: $error_type" >&2; return 1 ;;
        esac
    fi
    
    # Route to specific command handlers
    case "$cmd" in
        list|ls) _ollama_cmd_list "$@" ;;
        pull) _ollama_cmd_pull "$@" ;;
        run) _ollama_cmd_run "$@" ;;
        rm|remove) _ollama_cmd_remove "$@" ;;
        show) _ollama_cmd_show "$@" ;;
        ps) _ollama_cmd_ps "$@" ;;
        stop) _ollama_cmd_stop "$@" ;;
        serve) _ollama_cmd_serve "$@" ;;
        create) _ollama_cmd_create "$@" ;;
        cp|copy) _ollama_cmd_copy "$@" ;;
        push) _ollama_cmd_push "$@" ;;
        help|--help|-h) _ollama_cmd_help ;;
        version|--version|-v) _ollama_cmd_version ;;
        *) 
            echo "Error: unknown command '$cmd' for 'ollama'" >&2
            echo "Run 'ollama --help' for usage." >&2
            return 1
            ;;
    esac
}

#######################################
# Command: list models
#######################################
_ollama_cmd_list() {
    case "$OLLAMA_MOCK_MODE" in
        unhealthy|installing) 
            echo "Error: Ollama server is not ready" >&2
            return 1
            ;;
    esac
    
    echo "NAME                    	ID          	SIZE  	MODIFIED    "
    
    # List installed models
    for model in "${!MOCK_OLLAMA_MODELS[@]}"; do
        if [[ "${MOCK_OLLAMA_MODELS[$model]}" == "installed" ]]; then
            local info="${MOCK_OLLAMA_MODEL_INFO[$model]}"
            local timestamp=$(echo "$info" | cut -d'|' -f1)
            local size=$(echo "$info" | cut -d'|' -f2)
            local digest=$(echo "$info" | cut -d'|' -f3)
            
            # Format size for display
            local size_gb=$(echo "scale=1; $size / 1073741824" | bc -l 2>/dev/null || echo "4.0")
            
            # Format timestamp for display
            local display_time=$(date -d "$timestamp" "+%b %d %H:%M" 2>/dev/null || echo "Jan 15 10:30")
            
            printf "%-24s\t%-12s\t%-6s\t%s\n" "$model" "${digest:0:12}" "${size_gb} GB" "$display_time"
        fi
    done
}

#######################################
# Command: pull model
#######################################
_ollama_cmd_pull() {
    local model="$1"
    
    if [[ -z "$model" ]]; then
        echo "Error: missing model name" >&2
        echo "Usage: ollama pull <model>" >&2
        return 1
    fi
    
    case "$OLLAMA_MOCK_MODE" in
        unhealthy|stopped|offline) 
            echo "Error: Failed to connect to ollama server" >&2
            return 1
            ;;
        installing)
            echo "Error: Ollama is still initializing" >&2
            return 1
            ;;
    esac
    
    # Check if model already exists
    if [[ "${MOCK_OLLAMA_MODELS[$model]:-}" == "installed" ]]; then
        echo "$model is up to date"
        return 0
    fi
    
    # Simulate pull progress
    echo "pulling manifest"
    echo "pulling $(_mock_model_digest "$model"): 100%"
    echo "verifying sha256 digest" 
    echo "writing manifest"
    echo "removing any unused layers"
    echo "success"
    
    # Add model to registry
    mock::ollama::add_model "$model"
    
    return 0
}

#######################################  
# Command: run model (interactive)
#######################################
_ollama_cmd_run() {
    local model="$1"
    shift || true
    
    if [[ -z "$model" ]]; then
        echo "Error: missing model name" >&2
        echo "Usage: ollama run <model> [prompt]" >&2
        return 1
    fi
    
    case "$OLLAMA_MOCK_MODE" in
        unhealthy|stopped|offline) 
            echo "Error: Failed to connect to ollama server" >&2
            return 1
            ;;
    esac
    
    # Check if model exists
    if [[ "${MOCK_OLLAMA_MODELS[$model]:-}" != "installed" ]]; then
        echo "Error: model '$model' not found, try pulling it first" >&2
        return 1
    fi
    
    # Mark model as running
    mock::ollama::set_model_running "$model"
    
    if [[ $# -gt 0 ]]; then
        # Non-interactive mode with prompt
        local prompt="$*"
        echo "Mock response to: $prompt"
        echo "This is a simulated response from model $model."
    else
        # Interactive mode simulation
        echo ">>> Send a message (/? for help)"
        echo "Interactive mode not fully supported in mock. Use: ollama run $model \"your prompt\""
    fi
    
    return 0
}

#######################################
# Command: remove model  
#######################################
_ollama_cmd_remove() {
    local model="$1"
    
    if [[ -z "$model" ]]; then
        echo "Error: missing model name" >&2
        echo "Usage: ollama rm <model>" >&2
        return 1
    fi
    
    case "$OLLAMA_MOCK_MODE" in
        unhealthy|stopped|offline) 
            echo "Error: Failed to connect to ollama server" >&2
            return 1
            ;;
    esac
    
    # Check if model exists
    if [[ "${MOCK_OLLAMA_MODELS[$model]:-}" != "installed" ]]; then
        echo "Error: model '$model' not found" >&2
        return 1
    fi
    
    mock::ollama::remove_model "$model"
    echo "deleted '$model'"
    
    return 0
}

#######################################
# Command: show model info
#######################################
_ollama_cmd_show() {
    local model="$1"
    
    if [[ -z "$model" ]]; then
        echo "Error: missing model name" >&2
        echo "Usage: ollama show <model>" >&2
        return 1
    fi
    
    case "$OLLAMA_MOCK_MODE" in
        unhealthy|stopped|offline) 
            echo "Error: Failed to connect to ollama server" >&2
            return 1
            ;;
    esac
    
    # Check if model exists
    if [[ "${MOCK_OLLAMA_MODELS[$model]:-}" != "installed" ]]; then
        echo "Error: model '$model' not found" >&2
        return 1
    fi
    
    local info="${MOCK_OLLAMA_MODEL_INFO[$model]}"
    local family=$(echo "$info" | cut -d'|' -f4)
    local param_size=$(echo "$info" | cut -d'|' -f5)
    
    cat << EOF
  Model
  	architecture        	$family
  	parameters          	$param_size
  	context length      	4096
  	embedding length    	4096

  Parameters
  	temperature         	0.8
  	top_p              	0.9
  	top_k              	40

  License
  	Custom License

  Modelfile
  	FROM $model
  	PARAMETER temperature 0.8
  	PARAMETER top_p 0.9
  	PARAMETER top_k 40

EOF
    return 0
}

#######################################
# Command: list running models
#######################################
_ollama_cmd_ps() {
    case "$OLLAMA_MOCK_MODE" in
        unhealthy|stopped|offline) 
            echo "Error: Failed to connect to ollama server" >&2
            return 1
            ;;
    esac
    
    echo "NAME           	ID          	SIZE  	PROCESSOR	UNTIL          "
    
    for model in "${!MOCK_OLLAMA_RUNNING_MODELS[@]}"; do
        local size="${MOCK_OLLAMA_RUNNING_MODELS[$model]}"
        local info="${MOCK_OLLAMA_MODEL_INFO[$model]}"
        local digest=$(echo "$info" | cut -d'|' -f3)
        
        printf "%-15s\t%-12s\t%-6s\t%-9s\t%s\n" "$model" "${digest:0:12}" "$size" "100% GPU" "4 minutes from now"
    done
}

#######################################
# Command: stop model
#######################################
_ollama_cmd_stop() {
    local model="$1"
    
    if [[ -z "$model" ]]; then
        echo "Error: missing model name" >&2
        echo "Usage: ollama stop <model>" >&2
        return 1
    fi
    
    case "$OLLAMA_MOCK_MODE" in
        unhealthy|stopped|offline) 
            echo "Error: Failed to connect to ollama server" >&2
            return 1
            ;;
    esac
    
    if [[ -n "${MOCK_OLLAMA_RUNNING_MODELS[$model]}" ]]; then
        unset MOCK_OLLAMA_RUNNING_MODELS["$model"]
        _ollama_mock_save_state
        echo "stopped '$model'"
        return 0
    else
        echo "model '$model' is not currently loaded"
        return 0  # Changed to 0 for idempotent behavior
    fi
}

#######################################
# Command: serve (start server)
#######################################
_ollama_cmd_serve() {
    echo "Ollama server starting..."
    echo "time=2024-01-15T10:35:00.000Z level=INFO source=images.go:806 msg=\"total blobs: 0\""
    echo "time=2024-01-15T10:35:00.001Z level=INFO source=routes.go:1135 msg=\"Listening on 127.0.0.1:11434 (version 0.1.38)\""
    echo "time=2024-01-15T10:35:00.001Z level=INFO source=payload.go:30 msg=\"extracting embedded files\" dir=/tmp/ollama123456/runners"
    # Note: In real testing, this would run in background, but for mocking we just simulate startup
    return 0
}

#######################################
# Command: create custom model
#######################################
_ollama_cmd_create() {
    local model="$1"
    local modelfile="${2:--f Modelfile}"
    
    if [[ -z "$model" ]]; then
        echo "Error: missing model name" >&2
        echo "Usage: ollama create <model> -f <Modelfile>" >&2
        return 1
    fi
    
    echo "transferring model data"
    echo "using existing layer $(_mock_model_digest "$model")"
    echo "creating model layer"  
    echo "writing manifest"
    echo "success"
    
    # Add custom model to registry
    mock::ollama::add_model "$model"
    
    return 0
}

#######################################
# Command: copy model  
#######################################
_ollama_cmd_copy() {
    local source="$1"
    local dest="$2"
    
    if [[ -z "$source" || -z "$dest" ]]; then
        echo "Error: missing source or destination" >&2
        echo "Usage: ollama cp <source> <destination>" >&2
        return 1
    fi
    
    # Check source exists
    if [[ "${MOCK_OLLAMA_MODELS[$source]:-}" != "installed" ]]; then
        echo "Error: model '$source' not found" >&2
        return 1
    fi
    
    # Copy model info
    MOCK_OLLAMA_MODELS["$dest"]="${MOCK_OLLAMA_MODELS[$source]}"
    MOCK_OLLAMA_MODEL_INFO["$dest"]="${MOCK_OLLAMA_MODEL_INFO[$source]}"
    
    _ollama_mock_save_state
    
    echo "copied '$source' to '$dest'"
    return 0
}

#######################################
# Command: push model to registry
#######################################
_ollama_cmd_push() {
    local model="$1"
    
    if [[ -z "$model" ]]; then
        echo "Error: missing model name" >&2  
        echo "Usage: ollama push <model>" >&2
        return 1
    fi
    
    # Check model exists
    if [[ "${MOCK_OLLAMA_MODELS[$model]:-}" != "installed" ]]; then
        echo "Error: model '$model' not found" >&2
        return 1
    fi
    
    echo "pushing $model..."
    echo "pushing manifest"
    echo "pushing $(_mock_model_digest "$model"): 100%"
    echo "success"
    
    return 0
}

#######################################
# Command: help
#######################################
_ollama_cmd_help() {
    cat << 'EOF'
Usage:
  ollama [flags]
  ollama [command]

Available Commands:
  serve       Start ollama
  create      Create a model from a Modelfile
  show        Show information for a model
  run         Run a model
  pull        Pull a model from a registry
  push        Push a model to a registry
  list        List models
  ps          List running models
  cp          Copy a model
  rm          Remove a model
  help        Help about any command

Flags:
  -h, --help      help for ollama
  -v, --version   Show version information

Use "ollama [command] --help" for more information about a command.
EOF
}

#######################################
# Command: version
#######################################
_ollama_cmd_version() {
    echo "ollama version is 0.1.38"
}

# ----------------------------
# HTTP API Integration (for compatibility with existing http mock)
# ----------------------------

#######################################
# Setup healthy HTTP endpoints (used by http mock)
#######################################
_ollama_setup_healthy_http_endpoints() {
    # Health/tags endpoint (used for health checking)
    local models_json="["
    local first=true
    for model in "${!MOCK_OLLAMA_MODELS[@]}"; do
        if [[ "${MOCK_OLLAMA_MODELS[$model]}" == "installed" ]]; then
            local info="${MOCK_OLLAMA_MODEL_INFO[$model]}"
            local timestamp=$(echo "$info" | cut -d'|' -f1)
            local size=$(echo "$info" | cut -d'|' -f2)
            local digest=$(echo "$info" | cut -d'|' -f3)
            local family=$(echo "$info" | cut -d'|' -f4)
            local param_size=$(echo "$info" | cut -d'|' -f5)
            
            [[ "$first" == false ]] && models_json+=","
            models_json+="{\"name\":\"$model\",\"modified_at\":\"$timestamp\",\"size\":$size,\"digest\":\"$digest\",\"details\":{\"format\":\"gguf\",\"family\":\"$family\",\"parameter_size\":\"$param_size\"}}"
            first=false
        fi
    done
    models_json+="]"
    
    mock::http::set_endpoint_response "$OLLAMA_BASE_URL/api/tags" \
        "{\"models\":$models_json}"
    
    # Generate endpoint 
    mock::http::set_endpoint_response "$OLLAMA_BASE_URL/api/generate" \
        '{"model":"llama3.1:8b","created_at":"'$(_mock_current_timestamp)'","response":"Hello! This is a mock response from Ollama.","done":true,"context":[1,2,3,4,5],"total_duration":1234567890,"load_duration":234567890,"prompt_eval_count":10,"prompt_eval_duration":123456789,"eval_count":25,"eval_duration":987654321}' \
        200 \
        "POST"
        
    # Chat endpoint (OpenAI compatible)
    mock::http::set_endpoint_response "$OLLAMA_BASE_URL/api/chat" \
        '{"model":"llama3.1:8b","created_at":"'$(_mock_current_timestamp)'","message":{"role":"assistant","content":"Hello! How can I help you today?"},"done":true}' \
        200 \
        "POST"
        
    # Pull endpoint
    mock::http::set_endpoint_response "$OLLAMA_BASE_URL/api/pull" \
        '{"status":"success","digest":"sha256:abc123"}' \
        200 \
        "POST"
        
    # Show endpoint  
    mock::http::set_endpoint_response "$OLLAMA_BASE_URL/api/show" \
        '{"modelfile":"FROM llama3.1:8b\\nPARAMETER temperature 0.7","parameters":"temperature 0.7","template":"{{ .Prompt }}","details":{"format":"gguf","family":"llama","parameter_size":"8B"}}' \
        200 \
        "POST"
        
    # Version endpoint
    mock::http::set_endpoint_response "$OLLAMA_BASE_URL/api/version" \
        '{"version":"0.1.38"}'
        
    # Embeddings endpoint
    mock::http::set_endpoint_response "$OLLAMA_BASE_URL/api/embeddings" \
        '{"embedding":[0.1,0.2,0.3,0.4,0.5]}' \
        200 \
        "POST"
}

#######################################  
# Setup unhealthy HTTP endpoints
#######################################
_ollama_setup_unhealthy_http_endpoints() {
    mock::http::set_endpoint_response "$OLLAMA_BASE_URL/api/tags" \
        '{"error":"Service temporarily unavailable"}' \
        503
        
    mock::http::set_endpoint_response "$OLLAMA_BASE_URL/api/generate" \
        '{"error":"Service unhealthy"}' \
        503 \
        "POST"
}

#######################################
# Setup installing HTTP endpoints  
#######################################
_ollama_setup_installing_http_endpoints() {
    mock::http::set_endpoint_response "$OLLAMA_BASE_URL/api/tags" \
        '{"error":"Service is still initializing"}' \
        503
}

#######################################
# Setup stopped HTTP endpoints
#######################################
_ollama_setup_stopped_http_endpoints() {
    mock::http::set_endpoint_unreachable "$OLLAMA_BASE_URL"
}

# Initialize HTTP endpoints when this mock is loaded (only if not in BATS test mode)
# Skip HTTP endpoint initialization during BATS test setup to avoid complex mock interactions
if [[ -z "${BATS_TEST_FILENAME:-}" ]] && command -v mock::http::set_endpoint_response &>/dev/null; then
    case "$OLLAMA_MOCK_MODE" in
        healthy)
            _ollama_setup_healthy_http_endpoints
            ;;
        unhealthy)
            _ollama_setup_unhealthy_http_endpoints  
            ;;
        installing)
            _ollama_setup_installing_http_endpoints
            ;;
        stopped|offline)
            _ollama_setup_stopped_http_endpoints
            ;;
    esac
fi

# ----------------------------
# Test Helper Functions
# ----------------------------

#######################################
# Scenario: Setup healthy Ollama with default models
#######################################
mock::ollama::scenario::healthy_with_models() {
    mock::ollama::reset
    mock::ollama::set_mode "healthy"
    
    # Add some running models
    mock::ollama::set_model_running "llama3.1:8b" "4.5 GB"
    
    echo "[MOCK] Created healthy Ollama scenario with models"
}

#######################################
# Scenario: Setup Ollama in installing state
#######################################  
mock::ollama::scenario::installing() {
    mock::ollama::reset
    if ! mock::ollama::set_mode "installing"; then
        echo "[MOCK] ERROR: Failed to set installing mode" >&2
        return 1
    fi
    
    echo "[MOCK] Created installing Ollama scenario"
}

#######################################
# Scenario: Setup offline Ollama
#######################################
mock::ollama::scenario::offline() {
    mock::ollama::reset
    if ! mock::ollama::set_mode "offline"; then
        echo "[MOCK] ERROR: Failed to set offline mode" >&2
        return 1
    fi
    
    echo "[MOCK] Created offline Ollama scenario"
}

# ----------------------------
# Assertion Helper Functions
# ----------------------------

#######################################
# Assert that a model is installed
# Arguments: $1 - model name
#######################################
mock::ollama::assert::model_installed() {
    local model="$1"
    
    # Load current state
    _ollama_load_state
    
    
    if [[ "${MOCK_OLLAMA_MODELS[$model]:-}" != "installed" ]]; then
        echo "ASSERTION FAILED: Model '$model' is not installed" >&2
        return 1
    fi
    return 0
}

#######################################
# Assert that a model is running
# Arguments: $1 - model name  
#######################################
mock::ollama::assert::model_running() {
    local model="$1"
    
    # Load current state
    _ollama_load_state
    
    
    if [[ -z "${MOCK_OLLAMA_RUNNING_MODELS[$model]:-}" ]]; then
        echo "ASSERTION FAILED: Model '$model' is not running" >&2
        return 1
    fi
    return 0
}

#######################################
# Assert that a command was called specific number of times
# Arguments: $1 - command, $2 - expected count
#######################################
mock::ollama::assert::command_called() {
    local cmd="$1"
    local expected_count="${2:-1}"
    
    # Load state from file for subshell access
    _ollama_load_state
    
    local actual_count="${MOCK_OLLAMA_CALL_COUNT[$cmd]:-0}"
    
    if [[ "$actual_count" -lt "$expected_count" ]]; then
        echo "ASSERTION FAILED: Command '$cmd' called $actual_count times, expected at least $expected_count" >&2
        return 1
    fi
    return 0
}

# ----------------------------
# Get Information Functions
# ----------------------------

#######################################
# Get call count for a command
# Arguments: $1 - command name
#######################################
mock::ollama::get::call_count() {
    _ollama_load_state
    
    local cmd="$1"
    echo "${MOCK_OLLAMA_CALL_COUNT[$cmd]:-0}"
}

#######################################
# List installed models  
#######################################
mock::ollama::get::installed_models() {
    # Ensure we load the latest state  
    _ollama_load_state
    
    local models=()
    for model in "${!MOCK_OLLAMA_MODELS[@]}"; do
        if [[ "${MOCK_OLLAMA_MODELS[$model]}" == "installed" ]]; then
            models+=("$model")
        fi
    done
    
    printf '%s\n' "${models[@]}"
}

#######################################
# Get current mode
#######################################
mock::ollama::get::mode() {
    _ollama_load_state
    
    echo "$OLLAMA_MOCK_MODE"
}

# ----------------------------
# Debug Functions
# ----------------------------

#######################################
# Dump all mock state for debugging
#######################################
mock::ollama::debug::dump_state() {
    # Load state from file for subshell access
    _ollama_load_state
    
    echo "=== Ollama Mock State Dump ==="
    echo "Mode: $OLLAMA_MOCK_MODE"
    echo "Port: $OLLAMA_PORT" 
    echo "Base URL: $OLLAMA_BASE_URL"
    echo
    echo "Installed Models:"
    for model in "${!MOCK_OLLAMA_MODELS[@]}"; do
        echo "  $model: ${MOCK_OLLAMA_MODELS[$model]}"
    done
    echo
    echo "Running Models:"
    for model in "${!MOCK_OLLAMA_RUNNING_MODELS[@]}"; do
        echo "  $model: ${MOCK_OLLAMA_RUNNING_MODELS[$model]}"
    done
    echo
    echo "Command Call Counts:"
    for cmd in "${!MOCK_OLLAMA_CALL_COUNT[@]}"; do
        echo "  $cmd: ${MOCK_OLLAMA_CALL_COUNT[$cmd]}"
    done
    echo
    echo "Injected Errors:"
    for cmd in "${!MOCK_OLLAMA_ERRORS[@]}"; do
        echo "  $cmd: ${MOCK_OLLAMA_ERRORS[$cmd]}"
    done
    echo "============================="
}

# ----------------------------
# Export functions for subshell availability
# ----------------------------

# Export command mocks
export -f ollama
export -f _ollama_cmd_list _ollama_cmd_pull _ollama_cmd_run _ollama_cmd_remove
export -f _ollama_cmd_show _ollama_cmd_ps _ollama_cmd_stop _ollama_cmd_serve
export -f _ollama_cmd_create _ollama_cmd_copy _ollama_cmd_push _ollama_cmd_help _ollama_cmd_version

# Export utilities  
export -f _mock_current_timestamp _mock_model_digest _mock_model_size
export -f _mock_model_family _mock_model_parameter_size
export -f _ollama_mock_init_state_file _ollama_mock_save_state

# Export public API functions
export -f mock::ollama::reset mock::ollama::set_mode
export -f mock::ollama::add_model mock::ollama::remove_model mock::ollama::set_model_running
export -f mock::ollama::inject_error

# Export HTTP endpoint setup functions
export -f _ollama_setup_healthy_http_endpoints _ollama_setup_unhealthy_http_endpoints
export -f _ollama_setup_installing_http_endpoints _ollama_setup_stopped_http_endpoints

# Export scenarios
export -f mock::ollama::scenario::healthy_with_models mock::ollama::scenario::installing
export -f mock::ollama::scenario::offline

# Export assertions
export -f mock::ollama::assert::model_installed mock::ollama::assert::model_running
export -f mock::ollama::assert::command_called

# Export helper functions
export -f _ollama_load_state

# Export getters
export -f mock::ollama::get::call_count mock::ollama::get::installed_models mock::ollama::get::mode

# Export debug functions
export -f mock::ollama::debug::dump_state

# Load state if state file exists and has content, otherwise use defaults
# This allows setup() to override with reset if needed
if [[ -f "$OLLAMA_MOCK_STATE_FILE" ]] && [[ -s "$OLLAMA_MOCK_STATE_FILE" ]] && grep -q "MOCK_OLLAMA_MODELS\[" "$OLLAMA_MOCK_STATE_FILE" 2>/dev/null; then
    # State file exists with actual models, load it
    eval "$(cat "$OLLAMA_MOCK_STATE_FILE")" 2>/dev/null || true
fi

echo "[MOCK] Ollama mock implementation loaded successfully"