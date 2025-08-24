#!/usr/bin/env bash
# Ollama Mock - Tier 2 (Stateful)
# 
# Provides stateful Ollama mock with essential operations for testing:
# - Model management (list, pull, remove, show)
# - Model execution (run with prompt simulation)
# - Server operations (serve, stop, status)
# - Running model tracking (ps, stop specific models)
# - Error injection for resilience testing
#
# Coverage: ~80% of common Ollama use cases in 500 lines

# === Configuration ===
declare -gA OLLAMA_MODELS=()          # Model storage: name -> status (available|pulling|running)
declare -gA OLLAMA_MODEL_INFO=()      # Model details: name -> "size|modified|family|params"
declare -gA OLLAMA_RUNNING=()         # Running models: name -> "start_time|memory_usage"
declare -gA OLLAMA_PULL_PROGRESS=()   # Pull progress: name -> percentage

# Debug and error modes
declare -g OLLAMA_DEBUG="${OLLAMA_DEBUG:-}"
declare -g OLLAMA_ERROR_MODE="${OLLAMA_ERROR_MODE:-}"

# Configuration
declare -g OLLAMA_HOST="${OLLAMA_HOST:-localhost}"
declare -g OLLAMA_PORT="${OLLAMA_PORT:-11434}"
declare -g OLLAMA_VERSION="0.1.26"
declare -g OLLAMA_SERVER_RUNNING="true"

# === Helper Functions ===
ollama_debug() {
    [[ -n "$OLLAMA_DEBUG" ]] && echo "[MOCK:OLLAMA] $*" >&2
}

ollama_check_error() {
    case "$OLLAMA_ERROR_MODE" in
        "connection_failed")
            echo "Error: Failed to connect to ollama server at http://$OLLAMA_HOST:$OLLAMA_PORT" >&2
            return 1
            ;;
        "server_not_running")
            echo "Error: ollama server is not running" >&2
            return 1
            ;;
        "disk_space")
            echo "Error: insufficient disk space" >&2
            return 1
            ;;
        "model_not_found")
            echo "Error: model not found" >&2
            return 1
            ;;
        "timeout")
            sleep 3
            echo "Error: request timed out" >&2
            return 1
            ;;
    esac
    return 0
}

ollama_format_size() {
    local model="$1"
    # Simple size simulation based on model name
    if [[ "$model" =~ "70b" ]]; then
        echo "39 GB"
    elif [[ "$model" =~ "13b" ]]; then
        echo "7.3 GB"
    elif [[ "$model" =~ "7b" ]]; then
        echo "3.8 GB"
    else
        echo "4.1 GB"  # Default for most models
    fi
}

ollama_format_modified() {
    # Return a recent timestamp
    local days_ago=$(( (RANDOM % 30) + 1 ))
    date -d "${days_ago} days ago" "+%Y-%m-%d %H:%M" 2>/dev/null || date "+%Y-%m-%d %H:%M"
}

ollama_model_family() {
    local model="$1"
    if [[ "$model" =~ ^llama ]]; then
        echo "llama"
    elif [[ "$model" =~ ^deepseek ]]; then
        echo "deepseek"
    elif [[ "$model" =~ ^qwen ]]; then
        echo "qwen"
    elif [[ "$model" =~ ^mistral ]]; then
        echo "mistral"
    else
        echo "llama"  # Default
    fi
}

ollama_extract_params() {
    local model="$1"
    if [[ "$model" =~ :([0-9]+)b ]]; then
        echo "${BASH_REMATCH[1]}B"
    else
        echo "8B"  # Default
    fi
}

# === Main Ollama CLI Mock ===
ollama() {
    ollama_debug "Called with: $*"
    
    # Check for errors
    ollama_check_error || return $?
    
    # Parse command
    local command="${1:-help}"
    shift || true
    
    case "$command" in
        "--help"|"help"|"-h")
            cat << 'EOF'
Large language model runner

Usage:
  ollama [OPTIONS] COMMAND

Available Commands:
  serve       Start ollama server
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
  -v, --version   version for ollama

Use "ollama [command] --help" for more information about a command.
EOF
            ;;
            
        "--version"|"version"|"-v")
            echo "ollama version is $OLLAMA_VERSION"
            ;;
            
        "serve")
            if [[ "$OLLAMA_SERVER_RUNNING" == "true" ]]; then
                echo "ollama is already running"
                return 1
            fi
            OLLAMA_SERVER_RUNNING="true"
            echo "Ollama server started on http://$OLLAMA_HOST:$OLLAMA_PORT"
            ;;
            
        "list"|"ls")
            if [[ ${#OLLAMA_MODELS[@]} -eq 0 ]]; then
                echo "No models found"
                return 0
            fi
            
            printf "%-25s %-12s %-8s %-15s\n" "NAME" "ID" "SIZE" "MODIFIED"
            
            for model in "${!OLLAMA_MODELS[@]}"; do
                if [[ "${OLLAMA_MODELS[$model]}" == "available" ]]; then
                    local info="${OLLAMA_MODEL_INFO[$model]:-}"
                    local size modified
                    if [[ -n "$info" ]]; then
                        size="$(echo "$info" | cut -d'|' -f1)"
                        modified="$(echo "$info" | cut -d'|' -f2)"
                    else
                        size="$(ollama_format_size "$model")"
                        modified="$(ollama_format_modified)"
                    fi
                    
                    local model_id="${model//[^a-zA-Z0-9]}"
                    model_id="${model_id:0:12}"
                    
                    printf "%-25s %-12s %-8s %-15s\n" "$model" "$model_id" "$size" "$modified"
                fi
            done
            ;;
            
        "pull")
            local model="${1:-}"
            if [[ -z "$model" ]]; then
                echo "Error: missing model name" >&2
                echo "Usage: ollama pull <model>" >&2
                return 1
            fi
            
            # Check if already exists
            if [[ "${OLLAMA_MODELS[$model]:-}" == "available" ]]; then
                echo "$model is up to date"
                return 0
            fi
            
            # Simulate pull progress
            OLLAMA_MODELS[$model]="pulling"
            OLLAMA_PULL_PROGRESS[$model]="0"
            
            echo "pulling manifest"
            echo "pulling $(ollama_format_size "$model") model..."
            
            # Simulate progress
            for percent in 25 50 75 100; do
                echo "$percent% ⣿⣿⣿⣿⣿⣀⣀⣀⣀⣀⣀⣀⣀⣀⣀⣀ $(ollama_format_size "$model")/$(ollama_format_size "$model")"
                sleep 0.1  # Brief pause for realism
            done
            
            echo "verifying sha256 digest"
            echo "writing manifest"
            echo "removing any unused layers"
            echo "success"
            
            # Mark as available
            OLLAMA_MODELS[$model]="available"
            local size modified family params
            size="$(ollama_format_size "$model")"
            modified="$(ollama_format_modified)"
            family="$(ollama_model_family "$model")"
            params="$(ollama_extract_params "$model")"
            OLLAMA_MODEL_INFO[$model]="$size|$modified|$family|$params"
            unset OLLAMA_PULL_PROGRESS[$model]
            
            # Debug: confirm model was added
            ollama_debug "Added model $model, now have ${#OLLAMA_MODELS[@]} models total"
            ;;
            
        "run")
            local model="${1:-}"
            if [[ -z "$model" ]]; then
                echo "Error: missing model name" >&2
                echo "Usage: ollama run <model> [prompt]" >&2
                return 1
            fi
            shift
            
            # Check if model exists
            if [[ "${OLLAMA_MODELS[$model]:-}" != "available" ]]; then
                echo "Error: model '$model' not found. Try pulling it first." >&2
                return 1
            fi
            
            # Mark as running
            local memory_usage=$(( (RANDOM % 400) + 100 ))
            OLLAMA_RUNNING[$model]="$(date +%s)|${memory_usage}MB"
            
            if [[ $# -gt 0 ]]; then
                # Non-interactive with prompt
                local prompt="$*"
                ollama_debug "Running model $model with prompt: $prompt"
                
                # Simulate thinking time
                echo "Loading model $model..."
                sleep 0.5
                
                # Generate mock response based on prompt
                echo "Mock response from $model:"
                echo "I understand you asked: '$prompt'"
                echo "This is a simulated response for testing purposes."
                if [[ "$prompt" =~ (code|programming|function) ]]; then
                    echo "Here's a simple example:"
                    echo '```python'
                    echo 'def hello_world():'
                    echo '    print("Hello, World!")'
                    echo '```'
                fi
            else
                # Interactive mode
                echo ">>> Send a message (/? for help)"
                echo "Interactive mode - type your message or /bye to exit"
                echo "(This is a mock - in real usage, you'd chat interactively)"
            fi
            ;;
            
        "show")
            local model="${1:-}"
            if [[ -z "$model" ]]; then
                echo "Error: missing model name" >&2
                echo "Usage: ollama show <model>" >&2
                return 1
            fi
            
            if [[ "${OLLAMA_MODELS[$model]:-}" != "available" ]]; then
                echo "Error: model '$model' not found" >&2
                return 1
            fi
            
            local info="${OLLAMA_MODEL_INFO[$model]:-}"
            local size family params
            if [[ -n "$info" ]]; then
                size="$(echo "$info" | cut -d'|' -f1)"
                family="$(echo "$info" | cut -d'|' -f3)"
                params="$(echo "$info" | cut -d'|' -f4)"
            else
                size="$(ollama_format_size "$model")"
                family="$(ollama_model_family "$model")"
                params="$(ollama_extract_params "$model")"
            fi
            
            cat << EOF
  Model
  	architecture	$family
  	parameters	$params
  	context length	4096
  	embedding length	4096

  Parameters
  	temperature	0.8
  	top_p	0.9
  	top_k	40
  	num_predict	128

  License
  	Custom License

  Modelfile
  	FROM $model
  	PARAMETER temperature 0.8
  	PARAMETER top_p 0.9
  	PARAMETER top_k 40

EOF
            ;;
            
        "ps")
            if [[ ${#OLLAMA_RUNNING[@]} -eq 0 ]]; then
                echo "No models are currently running"
                return 0
            fi
            
            printf "%-25s %-12s %-8s %-15s\n" "NAME" "ID" "SIZE" "PROCESSOR"
            
            for model in "${!OLLAMA_RUNNING[@]}"; do
                local running_info="${OLLAMA_RUNNING[$model]}"
                local memory="$(echo "$running_info" | cut -d'|' -f2)"
                local model_id="${model//[^a-zA-Z0-9]}"
                model_id="${model_id:0:12}"
                
                printf "%-25s %-12s %-8s %-15s\n" "$model" "$model_id" "$memory" "100% GPU"
            done
            ;;
            
        "rm"|"remove")
            local model="${1:-}"
            if [[ -z "$model" ]]; then
                echo "Error: missing model name" >&2
                echo "Usage: ollama rm <model>" >&2
                return 1
            fi
            
            if [[ "${OLLAMA_MODELS[$model]:-}" != "available" ]]; then
                echo "Error: model '$model' not found" >&2
                return 1
            fi
            
            # Stop if running
            if [[ -n "${OLLAMA_RUNNING[$model]:-}" ]]; then
                unset OLLAMA_RUNNING[$model]
            fi
            
            # Remove model
            unset OLLAMA_MODELS[$model]
            unset OLLAMA_MODEL_INFO[$model]
            
            echo "deleted '$model'"
            ;;
            
        "stop")
            local model="${1:-}"
            if [[ -z "$model" ]]; then
                # Stop all running models
                if [[ ${#OLLAMA_RUNNING[@]} -eq 0 ]]; then
                    echo "No models are currently running"
                else
                    local stopped=0
                    for running_model in "${!OLLAMA_RUNNING[@]}"; do
                        unset OLLAMA_RUNNING[$running_model]
                        ((stopped++))
                    done
                    echo "Stopped $stopped model(s)"
                fi
            else
                # Stop specific model
                if [[ -n "${OLLAMA_RUNNING[$model]:-}" ]]; then
                    unset OLLAMA_RUNNING[$model]
                    echo "Stopped '$model'"
                else
                    echo "Model '$model' is not currently running"
                fi
            fi
            ;;
            
        "create"|"push"|"cp"|"copy")
            echo "Mock: Advanced command '$command' simulated successfully"
            ;;
            
        *)
            echo "Error: unknown command '$command' for 'ollama'" >&2
            echo "Run 'ollama --help' for usage." >&2
            return 1
            ;;
    esac
}

# === Convention-based Test Functions ===
test_ollama_connection() {
    ollama_debug "Testing connection..."
    
    # Test basic version command
    local result
    result=$(ollama version 2>&1)
    
    if [[ "$result" =~ "ollama version" ]]; then
        ollama_debug "Connection test passed"
        return 0
    else
        ollama_debug "Connection test failed: $result"
        return 1
    fi
}

test_ollama_health() {
    ollama_debug "Testing health..."
    
    # Test connection
    test_ollama_connection || return 1
    
    # Test model operations
    ollama pull llama3.1:8b >/dev/null 2>&1 || return 1
    ollama list | grep -q "llama3.1:8b" || return 1
    ollama show llama3.1:8b >/dev/null 2>&1 || return 1
    
    ollama_debug "Health test passed"
    return 0
}

test_ollama_basic() {
    ollama_debug "Testing basic operations..."
    
    # Test version
    ollama version >/dev/null 2>&1 || return 1
    
    # Test pull
    ollama pull llama3.1:8b >/dev/null 2>&1 || return 1
    
    # Test list
    local list_result
    list_result=$(ollama list 2>&1)
    [[ "$list_result" =~ "llama3.1:8b" ]] || return 1
    
    # Test run
    local run_result
    run_result=$(ollama run llama3.1:8b "Hello" 2>&1)
    [[ "$run_result" =~ "Mock response" ]] || return 1
    
    # Test ps
    ollama ps >/dev/null 2>&1 || return 1
    
    # Test show
    ollama show llama3.1:8b >/dev/null 2>&1 || return 1
    
    # Test remove
    ollama rm llama3.1:8b >/dev/null 2>&1 || return 1
    
    ollama_debug "Basic test passed"
    return 0
}

# === State Management ===
ollama_mock_reset() {
    ollama_debug "Resetting mock state (called from: ${BASH_SOURCE[1]:-unknown}:${BASH_LINENO[0]:-unknown})"
    OLLAMA_MODELS=()
    OLLAMA_MODEL_INFO=()
    OLLAMA_RUNNING=()
    OLLAMA_PULL_PROGRESS=()
    OLLAMA_ERROR_MODE=""
    OLLAMA_SERVER_RUNNING="true"
    
    # Create default models for testing
    OLLAMA_MODELS["llama3.1:8b"]="available"
    OLLAMA_MODEL_INFO["llama3.1:8b"]="4.1 GB|$(ollama_format_modified)|llama|8B"
    
    OLLAMA_MODELS["deepseek-r1:8b"]="available"
    OLLAMA_MODEL_INFO["deepseek-r1:8b"]="4.3 GB|$(ollama_format_modified)|deepseek|8B"
}

ollama_mock_set_error() {
    OLLAMA_ERROR_MODE="$1"
    ollama_debug "Set error mode: $1"
}

ollama_mock_dump_state() {
    echo "=== Ollama Mock State ==="
    echo "Server: $OLLAMA_HOST:$OLLAMA_PORT (running: $OLLAMA_SERVER_RUNNING)"
    echo "Version: $OLLAMA_VERSION"
    echo "Available Models: ${#OLLAMA_MODELS[@]}"
    for model in "${!OLLAMA_MODELS[@]}"; do
        echo "  $model: ${OLLAMA_MODELS[$model]}"
    done
    echo "Running Models: ${#OLLAMA_RUNNING[@]}"
    for model in "${!OLLAMA_RUNNING[@]}"; do
        echo "  $model: ${OLLAMA_RUNNING[$model]}"
    done
    echo "Error Mode: ${OLLAMA_ERROR_MODE:-none}"
    echo "======================="
}

ollama_mock_add_model() {
    local model="${1:-llama3.1:8b}"
    local size="${2:-$(ollama_format_size "$model")}"
    
    OLLAMA_MODELS[$model]="available"
    OLLAMA_MODEL_INFO[$model]="$size|$(ollama_format_modified)|$(ollama_model_family "$model")|$(ollama_extract_params "$model")"
    ollama_debug "Added model: $model ($size)"
    
    echo "$model"
}

# === Export Functions ===
export -f ollama
export -f test_ollama_connection
export -f test_ollama_health
export -f test_ollama_basic
export -f ollama_mock_reset
export -f ollama_mock_set_error
export -f ollama_mock_dump_state
export -f ollama_mock_add_model
export -f ollama_debug
export -f ollama_check_error

# Initialize with default state
ollama_mock_reset
ollama_debug "Ollama Tier 2 mock initialized"