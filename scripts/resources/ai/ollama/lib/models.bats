#!/usr/bin/env bats

# Tests for Ollama model management functions

setup() {
    # Load shared test infrastructure
    source "$(dirname "${BATS_TEST_FILENAME}")/../../../tests/bats-fixtures/common_setup.bash"
    
    # Setup standard mocks
    vrooli_auto_setup
    
    # Set up test model catalog
    declare -gA MODEL_CATALOG=(
        ["llama3.1:8b"]="4.9|general,chat,reasoning|Latest general-purpose model from Meta"
        ["deepseek-r1:8b"]="4.7|reasoning,math,code,chain-of-thought|Advanced reasoning model"
        ["qwen2.5-coder:7b"]="4.1|code,programming,debugging|Superior code generation model"
        ["llava:13b"]="7.3|vision,image-understanding,multimodal|Image understanding model"
        ["llama2:7b"]="3.8|general,legacy|Legacy model, superseded by llama3.1"
        ["unknown-model:1b"]="1.0|test|Test model"
    )
    
    # Set up default models
    declare -ga DEFAULT_MODELS=(
        "llama3.1:8b"
        "deepseek-r1:8b"
        "qwen2.5-coder:7b"
    )
    
    # Set test environment
    export MODELS_INPUT=""
    export SKIP_MODELS="no"
    export OLLAMA_SERVICE_NAME="ollama"
    export HOME="/tmp"
    
    # Mock message variables
    export MSG_MODELS_HEADER="Available Models"
    export MSG_MODELS_LEGEND="Legend info"
    export MSG_MODELS_TOTAL_SIZE="Total size info"
    # Mock dynamic message functions
    MSG_UNKNOWN_MODELS() { echo "Unknown models: $*"; }
    export -f MSG_UNKNOWN_MODELS
    export MSG_USE_AVAILABLE_MODELS="Use available models"
    MSG_MODELS_VALIDATED() {
        local models=("$@")
        echo "Validated ${#models[@]} models, total size: $(printf "%.1f" "$total_size")GB"
    }
    export -f MSG_MODELS_VALIDATED
    export MSG_MODEL_PULL_SUCCESS="Model pull success"
    export MSG_MODEL_PULL_FAILED="Model pull failed"
    export MSG_MODEL_NONE_SPECIFIED="No models specified"
    export MSG_MODEL_VALIDATION_FAILED="Model validation failed"
    export MSG_MODEL_INSTALL_SUCCESS="Model install success"
    export MSG_MODEL_INSTALL_FAILED="Model install failed"
    MSG_MODELS_INSTALLED() { echo "Models installed: $*"; }
    export -f MSG_MODELS_INSTALLED
    MSG_MODELS_FAILED() { echo "Models failed: $*"; }
    export -f MSG_MODELS_FAILED
    MSG_LOW_DISK_SPACE() { echo "Low disk space: $1GB available"; }
    export -f MSG_LOW_DISK_SPACE
    export MSG_LIST_MODELS_FAILED="List models failed"
    export MSG_OLLAMA_API_UNAVAILABLE="API unavailable"
    export MSG_OLLAMA_NOT_INSTALLED="Not installed"
    
    # Mock functions
    ollama::is_healthy() { return 0; }  # Default: healthy
    resources::handle_error() { return 1; }
    resources::add_rollback_action() { return 0; }
    
    
    # Mock system commands
    bc() { 
        # Simple bc mock for adding numbers
        local expr="$1"
        if [[ "$expr" =~ ^([0-9.]+)\ \+\ ([0-9.]+)$ ]]; then
            local num1="${BASH_REMATCH[1]}"
            local num2="${BASH_REMATCH[2]}"
            echo "scale=1; $num1 + $num2" | /usr/bin/bc -l 2>/dev/null || echo "0.0"
        else
            echo "0.0"
        fi
    }
    
    ollama() {
        case "$1" in
            "list") 
                echo "NAME                ID              SIZE      MODIFIED"
                echo "llama3.1:8b         abc123          4.9 GB    1 hour ago"
                echo "deepseek-r1:8b      def456          4.7 GB    2 hours ago"
                ;;
            "pull")
                echo "pulling $2"
                return 0
                ;;
        esac
        return 0
    }
    
    df() {
        # Mock df command to return available space
        if [[ "$*" =~ --output=avail ]]; then
            echo "Avail"
            echo "50"  # 50GB available
        fi
    }
    
    printf() { /usr/bin/printf "$@"; }
    echo() { /bin/echo "$@"; }
    cut() { /usr/bin/cut "$@"; }
    grep() { /bin/grep "$@"; }
    awk() { /usr/bin/awk "$@"; }
    tr() { /usr/bin/tr "$@"; }
    sed() { /bin/sed "$@"; }
    tail() { /usr/bin/tail "$@"; }
    wc() { /usr/bin/wc "$@"; }
    
    # Source the model management functions
    source "$(dirname "$BATS_TEST_FILENAME")/models.sh"
}

@test "ollama::get_model_info returns correct info for known model" {
    run ollama::get_model_info "llama3.1:8b"
    [ "$status" -eq 0 ]
    [[ "$output" == "4.9|general,chat,reasoning|Latest general-purpose model from Meta" ]]
}

@test "ollama::get_model_info returns unknown for unknown model" {
    run ollama::get_model_info "nonexistent:1b"
    [ "$status" -eq 0 ]
    [[ "$output" == "unknown|unknown|Model not found in catalog" ]]
}

@test "ollama::get_model_size returns correct size" {
    run ollama::get_model_size "llama3.1:8b"
    [ "$status" -eq 0 ]
    [[ "$output" == "4.9" ]]
}

@test "ollama::is_model_known returns 0 for known model" {
    ollama::is_model_known "llama3.1:8b"
    [ "$?" -eq 0 ]
}

@test "ollama::is_model_known returns 1 for unknown model" {
    ollama::is_model_known "nonexistent:1b"
    [ "$?" -eq 1 ]
}

@test "ollama::show_available_models displays model catalog" {
    run ollama::show_available_models
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Available Models" ]]
    [[ "$output" =~ "llama3.1:8b" ]]
    [[ "$output" =~ "deepseek-r1:8b" ]]
    [[ "$output" =~ "qwen2.5-coder:7b" ]]
    [[ "$output" =~ "Legend info" ]]
}

@test "ollama::calculate_default_size calculates total size" {
    run ollama::calculate_default_size
    [ "$status" -eq 0 ]
    # Should be sum of default models: 4.9 + 4.7 + 4.1 = 13.7
    [[ "$output" =~ ^1[3-4]\.[0-9]$ ]]  # Allow for slight bc variations
}

@test "ollama::validate_model_list passes for valid models" {
    run ollama::validate_model_list "llama3.1:8b" "deepseek-r1:8b"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Models validated" ]]
}

@test "ollama::validate_model_list fails for invalid models" {
    run ollama::validate_model_list "llama3.1:8b" "nonexistent:1b"
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Unknown models" ]]
}

@test "ollama::get_installed_models returns model list when healthy" {
    run ollama::get_installed_models
    [ "$status" -eq 0 ]
    [[ "$output" =~ "llama3.1:8b" ]]
    [[ "$output" =~ "deepseek-r1:8b" ]]
}

@test "ollama::get_installed_models fails when not healthy" {
    ollama::is_healthy() { return 1; }
    
    run ollama::get_installed_models
    [ "$status" -eq 1 ]
}

@test "ollama::get_best_available_model returns best general model" {
    run ollama::get_best_available_model "general"
    [ "$status" -eq 0 ]
    [[ "$output" == "llama3.1:8b" ]]  # Should be first priority for general
}

@test "ollama::get_best_available_model returns best code model" {
    # Mock to return qwen2.5-coder:7b as installed
    ollama() {
        case "$1" in
            "list") 
                echo "NAME                ID              SIZE      MODIFIED"
                echo "qwen2.5-coder:7b    xyz789          4.1 GB    1 hour ago"
                ;;
        esac
    }
    
    run ollama::get_best_available_model "code"
    [ "$status" -eq 0 ]
    [[ "$output" == "qwen2.5-coder:7b" ]]
}

@test "ollama::get_best_available_model returns best reasoning model" {
    # Mock to return deepseek-r1:8b as installed
    ollama() {
        case "$1" in
            "list") 
                echo "NAME                ID              SIZE      MODIFIED"
                echo "deepseek-r1:8b      def456          4.7 GB    1 hour ago"
                ;;
        esac
    }
    
    run ollama::get_best_available_model "reasoning"
    [ "$status" -eq 0 ]
    [[ "$output" == "deepseek-r1:8b" ]]
}

@test "ollama::validate_model_available returns 0 for installed model" {
    run ollama::validate_model_available "llama3.1:8b"
    [ "$status" -eq 0 ]
}

@test "ollama::validate_model_available returns 1 for not installed model" {
    run ollama::validate_model_available "nonexistent:1b"
    [ "$status" -eq 1 ]
}

@test "ollama::parse_models returns default models when no input" {
    export MODELS_INPUT=""
    
    run ollama::parse_models
    [ "$status" -eq 0 ]
    [[ "$output" =~ "llama3.1:8b" ]]
    [[ "$output" =~ "deepseek-r1:8b" ]]
    [[ "$output" =~ "qwen2.5-coder:7b" ]]
}

@test "ollama::parse_models parses comma-separated input" {
    export MODELS_INPUT="llama3.1:8b,deepseek-r1:8b"
    
    run ollama::parse_models
    [ "$status" -eq 0 ]
    [[ "$output" == "llama3.1:8b deepseek-r1:8b" ]]
}

@test "ollama::pull_model succeeds for valid model" {
    run ollama::pull_model "llama3.1:8b"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Pulling model: llama3.1:8b" ]]
    [[ "$output" =~ "Model pull success" ]]
}

@test "ollama::pull_model fails when ollama command fails" {
    ollama() {
        case "$1" in
            "pull") return 1 ;;  # Simulate failure
        esac
    }
    
    run ollama::pull_model "llama3.1:8b"
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Model pull failed" ]]
}

@test "ollama::install_models skips when SKIP_MODELS=yes" {
    export SKIP_MODELS="yes"
    
    run ollama::install_models
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Skipping model installation" ]]
}

@test "ollama::install_models fails when API not available" {
    ollama::is_healthy() { return 1; }  # API not healthy
    
    run ollama::install_models
    [ "$status" -eq 1 ]
}

@test "ollama::install_models succeeds with default models" {
    # Mock successful installation
    export MODELS_INPUT=""  # Use default models
    
    # Mock ollama list to show no existing models initially
    local list_call_count=0
    ollama() {
        case "$1" in
            "list") 
                if [[ $list_call_count -eq 0 ]]; then
                    # First call: no models installed
                    echo "NAME                ID              SIZE      MODIFIED"
                    ((list_call_count++))
                else
                    # Subsequent calls: models are installed
                    echo "NAME                ID              SIZE      MODIFIED"
                    echo "llama3.1:8b         abc123          4.9 GB    1 hour ago"
                    echo "deepseek-r1:8b      def456          4.7 GB    2 hours ago"
                    echo "qwen2.5-coder:7b    ghi789          4.1 GB    3 hours ago"
                fi
                ;;
            "pull") return 0 ;;  # Successful pull
        esac
    }
    
    run ollama::install_models
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Installing Ollama Models" ]]
    [[ "$output" =~ "Successfully installed:" ]]
}

@test "ollama::list_models displays installed models" {
    run ollama::list_models
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Installed Ollama Models" ]]
    [[ "$output" =~ "llama3.1:8b" ]]
}

@test "ollama::list_models fails when API not available" {
    ollama::is_healthy() { return 1; }
    
    run ollama::list_models
    [ "$status" -eq 1 ]
    [[ "$output" =~ "API unavailable" ]]
}

@test "ollama::list_models fails when ollama command not found" {
    system::is_command() { return 1; }
    
    run ollama::list_models
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Not installed" ]]
}

@test "all model management functions are defined" {
    # Test that all expected functions exist
    type ollama::get_model_info >/dev/null
    type ollama::get_model_size >/dev/null
    type ollama::is_model_known >/dev/null
    type ollama::show_available_models >/dev/null
    type ollama::calculate_default_size >/dev/null
    type ollama::validate_model_list >/dev/null
    type ollama::get_installed_models >/dev/null
    type ollama::get_best_available_model >/dev/null
    type ollama::validate_model_available >/dev/null
    type ollama::parse_models >/dev/null
    type ollama::pull_model >/dev/null
    type ollama::install_models >/dev/null
    type ollama::list_models >/dev/null
}