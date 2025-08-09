#!/usr/bin/env bats
# Tests for Ollama API functions

# Load Vrooli test infrastructure
source "${BATS_TEST_DIRNAME}/../../../../__test/fixtures/setup.bash"

# Expensive setup operations (run once per file)
setup_file() {
    # Use appropriate setup function
    vrooli_setup_service_test "ollama"
    
    # Load dependencies once
    SCRIPT_DIR="${BATS_TEST_DIRNAME}"
    OLLAMA_DIR="$(dirname "$SCRIPT_DIR")"
    
    # Load configuration and messages once
    source "${OLLAMA_DIR}/config/defaults.sh"
    source "${OLLAMA_DIR}/config/messages.sh"
    
    # Load API functions once
    source "${SCRIPT_DIR}/api.sh"
    
    # Export paths for use in setup()
    export SETUP_FILE_SCRIPT_DIR="$SCRIPT_DIR"
    export SETUP_FILE_OLLAMA_DIR="$OLLAMA_DIR"
}

# Lightweight per-test setup
setup() {
    # Setup standard mocks
    vrooli_auto_setup
    
    # Use paths from setup_file
    SCRIPT_DIR="${SETUP_FILE_SCRIPT_DIR}"
    OLLAMA_DIR="${SETUP_FILE_OLLAMA_DIR}"
    
    # Set test environment (lightweight per-test)
    export OLLAMA_BASE_URL="http://localhost:11434"
    export OLLAMA_INSTALL_DIR="/usr/local/bin"
    export OLLAMA_PORT="11434"
    export OLLAMA_SERVICE_NAME="ollama"
    export OLLAMA_USER="ollama"
    export OUTPUT_FORMAT="text"
    export TEMPERATURE="0.8"
    export MAX_TOKENS=""
    export TOP_P="0.9"
    export TOP_K="40"
    export SEED=""
    export SYSTEM_PROMPT=""
    
    # Set up model catalog and defaults
    declare -gA MODEL_CATALOG=(
        ["llama3.1:8b"]="4.9|general,chat,reasoning|Latest general-purpose model from Meta"
        ["deepseek-r1:8b"]="4.7|reasoning,math,code,chain-of-thought|Advanced reasoning model"
        ["qwen2.5-coder:7b"]="4.1|code,programming,debugging|Superior code generation model"
    )
    
    declare -ga DEFAULT_MODELS=(
        "llama3.1:8b"
        "deepseek-r1:8b"
        "qwen2.5-coder:7b"
    )
    
    # Mock message variables (lightweight)
    export MSG_OLLAMA_API_UNAVAILABLE="API unavailable"
    export MSG_START_OLLAMA="Start Ollama"
    export MSG_PROMPT_NO_TEXT="No prompt text"
    export MSG_PROMPT_USAGE="Usage info"
    export MSG_MODEL_NOT_INSTALLED="Model not installed"
    export MSG_AVAILABLE_MODELS="Available models:"
    export MSG_INSTALL_MODEL="Install model"
    export MSG_MODEL_USING="Using model"
    export MSG_MODEL_SELECTING="Selecting model"
    export MSG_MODEL_SELECTED="Selected model"
    export MSG_MODEL_NONE_SUITABLE="No suitable models"
    export MSG_MODEL_INSTALL_FIRST="Install model first"
    export MSG_PROMPT_SENDING="Sending prompt"
    export MSG_PROMPT_RESPONSE_HEADER="Response header"
    export MSG_PROMPT_API_ERROR="API error"
    export MSG_PROMPT_RESPONSE_TIME="Response time"
    export MSG_PROMPT_TOKEN_COUNT="Token count"
    export MSG_PROMPT_PARAMETERS="Parameters"
    export MSG_PROMPT_NO_RESPONSE="No response"
    export MSG_JQ_UNAVAILABLE="jq unavailable"
    export MSG_FAILED_API_REQUEST="Failed API request"
    export MSG_CHECK_STATUS="Check status"
    
    # Mock ollama functions that need test control
    mock::ollama::setup() { return 0; }
    
    # Export mock function
    export -f mock::ollama::setup
    
    # Mock Docker functions for status checks
    ollama::is_installed() { return 0; }
    ollama::is_running() { return 0; }
    export -f ollama::is_installed ollama::is_running
    
    # Mock resources function
    resources::check_http_health() { return 0; }
    export -f resources::check_http_health
    
    # Override curl mock for Ollama-specific API responses
    curl() {
        if [[ "$*" =~ -X.*POST.*api/generate ]]; then
            # Mock successful API response
            echo '{"response":"Test response text","total_duration":1000000000,"prompt_eval_count":10,"eval_count":20}'
            return 0
        fi
        
        # Fall back to basic curl mock for other requests
        return 0
    }
    
    # Mock log functions
    log::info() { echo "[INFO] $*"; }
    log::error() { echo "[ERROR] $*" >&2; }
    log::warning() { echo "[WARNING] $*" >&2; }
    log::header() { echo "=== $* ==="; }
    log::success() { echo "[SUCCESS] $*"; }
    log::debug() { [[ "${DEBUG:-}" == "true" ]] && echo "[DEBUG] $*" >&2 || true; }
    export -f log::info log::error log::warning log::header log::success log::debug
    
    # Mock system commands (shared mocks handle most, only override what's needed)
    date() { echo "1234567890"; }  # Fixed timestamp for testing
}

# BATS teardown function - runs after each test
teardown() {
    vrooli_cleanup_test
}

@test "ollama::send_prompt fails when API not healthy" {
    ollama::is_healthy() { return 1; }
    
    run ollama::send_prompt "test prompt"
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Ollama API is not available" ]]
}

@test "ollama::send_prompt fails when no prompt text provided" {
    run ollama::send_prompt ""
    [ "$status" -eq 1 ]
    [[ "$output" =~ "No prompt text provided" ]]
}

@test "ollama::send_prompt succeeds with basic prompt" {
    run ollama::send_prompt "Hello world"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Selecting best available model" ]]
    [[ "$output" =~ "Sending prompt" ]]
    [[ "$output" =~ "Test response text" ]]
}

@test "ollama::send_prompt uses specified model when provided" {
    run ollama::send_prompt "Hello world" "llama3.1:8b"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Using model" ]]
    [[ "$output" =~ "Test response text" ]]
}

@test "ollama::send_prompt fails when specified model not available" {
    ollama::validate_model_available() { return 1; }
    
    run ollama::send_prompt "Hello world" "nonexistent:1b"
    [ "$status" -eq 1 ]
    [[ "$output" =~ "is not installed" ]]
}

@test "ollama::send_prompt selects model by type" {
    ollama::get_best_available_model() {
        case "$1" in
            "code") echo "qwen2.5-coder:7b" ;;
            "reasoning") echo "deepseek-r1:8b" ;;
            *) echo "llama3.1:8b" ;;
        esac
    }
    
    run ollama::send_prompt "Write code" "" "code"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Selecting best available model" ]]
    [[ "$output" =~ "Selected model" ]]
}

@test "ollama::send_prompt fails when no suitable model found" {
    ollama::get_best_available_model() { echo ""; return 1; }
    
    run ollama::send_prompt "Hello world"
    [ "$status" -eq 1 ]
    [[ "$output" =~ "No suitable models" ]]
}

@test "ollama::send_prompt returns JSON when format is json" {
    export OUTPUT_FORMAT="json"
    
    run ollama::send_prompt "Hello world"
    [ "$status" -eq 0 ]
    [[ "$output" =~ '"response":"Test response text"' ]]
    [[ ! "$output" =~ "INFO:" ]]  # Should not show info messages in JSON mode
}

@test "ollama::send_prompt handles API error response" {
    curl() {
        if [[ "$*" =~ -X.*POST.*api/generate ]]; then
            echo '{"error":"Model not found"}'
        fi
        return 0
    }
    
    run ollama::send_prompt "Hello world"
    [ "$status" -eq 1 ]
    [[ "$output" =~ "API error" ]]
}

@test "ollama::send_prompt handles curl failure" {
    curl() { return 1; }  # Simulate curl failure
    
    run ollama::send_prompt "Hello world"
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Failed API request" ]]
}

@test "ollama::send_prompt falls back when jq not available" {
    system::is_command() { return 1; }  # jq not available
    
    run ollama::send_prompt "Hello world"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "jq unavailable" ]]
}

@test "ollama::send_prompt handles system prompt" {
    export SYSTEM_PROMPT="You are a helpful assistant"
    
    # Mock curl to capture the request
    curl() {
        if [[ "$*" =~ -X.*POST.*api/generate ]]; then
            # The request should contain the system prompt
            local request_data=""
            while IFS= read -r line; do
                if [[ "$line" == -d* ]]; then
                    request_data="${line#-d }"
                    break
                fi
            done
            
            # Check if system prompt is included
            if [[ "$request_data" =~ "You are a helpful assistant" ]]; then
                echo '{"response":"Response with system prompt","prompt_eval_count":15,"eval_count":25}'
            else
                echo '{"response":"Response without system prompt","prompt_eval_count":10,"eval_count":20}'
            fi
        fi
        return 0
    }
    
    run ollama::send_prompt "Hello world"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Response with system prompt" ]]
}

@test "ollama::send_prompt handles custom parameters" {
    export TEMPERATURE="1.2"
    export MAX_TOKENS="100"
    export TOP_P="0.7"
    export TOP_K="30"
    export SEED="12345"
    
    run ollama::send_prompt "Hello world"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Parameters:" ]]
    [[ "$output" =~ "temperature=1.2" ]]
    [[ "$output" =~ "max_tokens=100" ]]
    [[ "$output" =~ "top_p=0.7" ]]
    [[ "$output" =~ "top_k=30" ]]
    [[ "$output" =~ "seed=12345" ]]
}

@test "ollama::send_prompt shows token statistics" {
    run ollama::send_prompt "Hello world"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Token count" ]]
    [[ "$output" =~ "Response time" ]]
}

@test "ollama::info displays comprehensive information" {
    run ollama::info
    [ "$status" -eq 0 ]
    [[ "$output" =~ "=== Ollama Resource Information ===" ]]
    [[ "$output" =~ "ID: ollama" ]]
    [[ "$output" =~ "Category: ai" ]]
    [[ "$output" =~ "Service Details:" ]]
    [[ "$output" =~ "Binary Location: $OLLAMA_INSTALL_DIR/ollama" ]]
    [[ "$output" =~ "Service Port: $OLLAMA_PORT" ]]
    [[ "$output" =~ "Service URL: $OLLAMA_BASE_URL" ]]
    [[ "$output" =~ "Endpoints:" ]]
    [[ "$output" =~ "Health Check: $OLLAMA_BASE_URL/api/tags" ]]
    [[ "$output" =~ "Configuration:" ]]
    [[ "$output" =~ "Default Models (2025):" ]]
    [[ "$output" =~ "Ollama Features:" ]]
    [[ "$output" =~ "Example Usage:" ]]
    [[ "$output" =~ "https://ollama.com/library" ]]
}

@test "ollama::info includes model catalog information" {
    run ollama::info
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Model Catalog: 3 models available" ]]
    [[ "$output" =~ "Total Default Size: 13.7GB" ]]
}

@test "all API functions are defined" {
    # Test that all expected functions exist
    type ollama::send_prompt >/dev/null
    type ollama::info >/dev/null
}

# Teardown
teardown() {
    vrooli_cleanup_test
}
