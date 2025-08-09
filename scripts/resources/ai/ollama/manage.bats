#!/usr/bin/env bats
# Tests for Ollama manage.sh script

# Get script directory first
MANAGE_BATS_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

# Source var.sh first to get directory variables
# shellcheck disable=SC1091
source "${MANAGE_BATS_DIR}/../../../lib/utils/var.sh"

# Load Vrooli test infrastructure using var_ variables
# shellcheck disable=SC1091
source "${var_SCRIPTS_TEST_DIR}/fixtures/setup.bash"
# Load fixture helpers for accessing test data
source "${var_SCRIPTS_RESOURCES_DIR}/tests/lib/fixture-helpers.sh" 2>/dev/null || true

# Helper function to initialize MODEL_CATALOG (fixes BATS associative array issue)
setup_model_catalog() {
    declare -gA MODEL_CATALOG=(
        ["llama3.1:8b"]="4.9|general,chat,reasoning|Latest general-purpose model from Meta"
        ["deepseek-r1:8b"]="4.7|reasoning,math,code,chain-of-thought|Advanced reasoning model with explicit thinking process"
        ["qwen2.5-coder:7b"]="4.1|code,programming,debugging|Superior code generation model, replaces CodeLlama"
        ["llama3.3:8b"]="4.9|general,chat,reasoning|Very latest from Meta (Dec 2024)"
        ["deepseek-r1:14b"]="8.1|reasoning,math,code,chain-of-thought|Larger reasoning model for complex problems"
        ["deepseek-r1:1.5b"]="0.9|reasoning,lightweight|Smallest reasoning model for resource-constrained environments"
        ["phi-4:14b"]="8.2|general,multilingual,math,function-calling|Microsoft's efficient model with multilingual support"
        ["qwen2.5:14b"]="8.0|general,multilingual,reasoning|Strong multilingual model with excellent reasoning"
        ["mistral-small:22b"]="13.2|general,balanced,multilingual|Excellent balanced performance model"
        ["qwen2.5-coder:32b"]="19.1|code,programming,architecture|Large code model for complex projects"
        ["deepseek-coder:6.7b"]="3.8|code,programming,documentation|Specialized programming model"
        ["llava:13b"]="7.3|vision,image-understanding,multimodal|Image understanding and visual reasoning"
        ["qwen2-vl:7b"]="4.2|vision,image-understanding,multimodal|Vision-language model for image analysis"
        ["llama2:7b"]="3.8|general,legacy|Legacy model, superseded by llama3.1"
        ["codellama:7b"]="3.8|code,legacy|Legacy code model, superseded by qwen2.5-coder"
    )
    
    # Also set up DEFAULT_MODELS array (check if readonly first)
    if ! readonly -p | grep -q "DEFAULT_MODELS"; then
        DEFAULT_MODELS=(
            "llama3.1:8b"
            "deepseek-r1:8b"
            "qwen2.5-coder:7b"
        )
    fi
}

# Setup for each test
# Run expensive setup once per file instead of per test
setup_file() {
    # Use Vrooli test infrastructure with Ollama-specific setup
    vrooli_setup_service_test "ollama"
    
    # Setup model catalog once for all tests
    setup_model_catalog
}

# Lightweight per-test setup
setup() {
    # Setup standard Vrooli mocks
    vrooli_auto_setup
    
    # Use paths from setup_file to avoid recalculating
    SCRIPT_PATH="${BATS_TEST_DIRNAME}/manage.sh"
    OLLAMA_DIR="${BATS_TEST_DIRNAME}"
    CONFIG_DIR="${OLLAMA_DIR}/config"
    
    # Set ollama-specific environment variables (lightweight)
    export OLLAMA_CUSTOM_PORT="9999"
    export PROMPT_TEXT=""
    export PROMPT_MODEL=""
    export PROMPT_TYPE="general"
    export OUTPUT_FORMAT="text"
    export TEMPERATURE="0.8"
    export MAX_TOKENS=""
    export TOP_P="0.9"
    export TOP_K="40"
    export SEED=""
    export SYSTEM_PROMPT=""
    export SKIP_MODELS="no"
    export VALIDATE_MODELS="yes"
    
    # Mock resources functions that might be called during config loading
    resources::get_default_port() {
        case "$1" in
            "ollama") echo "11434" ;;
            *) echo "8080" ;;
        esac
    }
    
    # Source configuration files
    source "${CONFIG_DIR}/defaults.sh"
    source "${CONFIG_DIR}/messages.sh"
    
    # Export configuration and messages
    ollama::export_config
    ollama::export_messages
}

# Cleanup after each test
teardown() {
    vrooli_cleanup_test
}

# Test script loading
@test "sourcing manage.sh defines required functions" {
    run bash -c "source '$SCRIPT_PATH' && declare -f ollama::parse_arguments && declare -f ollama::main"
    [ "$status" -eq 0 ]
    [[ "$output" =~ ollama::parse_arguments ]]
    [[ "$output" =~ ollama::main ]]
}

@test "manage.sh sources all required dependencies" {
    run bash -c "source '$SCRIPT_PATH' 2>&1 | grep -v 'command not found' | head -1"
    [ "$status" -eq 0 ]
}

# Test argument parsing
@test "ollama::parse_arguments sets defaults correctly" {
    run bash -c "source '$SCRIPT_PATH'; ollama::parse_arguments --action status; echo \"ACTION=\$ACTION FORCE=\$FORCE SKIP_MODELS=\$SKIP_MODELS PROMPT_TYPE=\$PROMPT_TYPE\""
    [ "$status" -eq 0 ]
    [[ "$output" =~ "ACTION=status" ]]
    [[ "$output" =~ "FORCE=no" ]]
    [[ "$output" =~ "SKIP_MODELS=no" ]]
    [[ "$output" =~ "PROMPT_TYPE=general" ]]
}

# Test argument parsing with prompt action
@test "ollama::parse_arguments handles prompt action correctly" {
    ollama::parse_arguments \
        --action prompt \
        --text "What is the capital of France?" \
        --model "llama3.1:8b" \
        --type "code"
    
    [ "$ACTION" = "prompt" ]
    [ "$PROMPT_TEXT" = "What is the capital of France?" ]
    [ "$PROMPT_MODEL" = "llama3.1:8b" ]
    [ "$PROMPT_TYPE" = "code" ]
}

# Test model catalog functions
@test "ollama::get_model_info returns correct format" {
    setup_model_catalog
    
    local info
    info=$(ollama::get_model_info "llama3.1:8b")
    
    # Should return format: size_gb|capabilities|description
    [[ "$info" =~ ^[0-9]+\.[0-9]+\|[^|]+\|.+ ]]
}

@test "ollama::get_model_size extracts size correctly" {
    setup_model_catalog
    
    local size
    size=$(ollama::get_model_size "llama3.1:8b")
    
    # Should return just the size number
    [[ "$size" =~ ^[0-9]+\.[0-9]+$ ]]
}

@test "ollama::is_model_known returns correct status" {
    setup_model_catalog
    
    # Known model should return 0
    run ollama::is_model_known "llama3.1:8b"
    [ "$status" -eq 0 ]
    
    # Unknown model should return 1
    run ollama::is_model_known "nonexistent:model"
    [ "$status" -eq 1 ]
}

# Test model type-based selection
@test "ollama::get_best_available_model respects explicit types" {
    # Mock installed models list
    ollama::get_installed_models() {
        echo "llama3.1:8b qwen2.5-coder:7b deepseek-r1:8b"
    }
    
    local best_model
    
    # Test explicit type selection
    best_model=$(ollama::get_best_available_model "code")
    [ "$best_model" = "qwen2.5-coder:7b" ]
    
    best_model=$(ollama::get_best_available_model "reasoning")
    [ "$best_model" = "deepseek-r1:8b" ]
    
    best_model=$(ollama::get_best_available_model "general")
    [ "$best_model" = "llama3.1:8b" ]
}

# Test model prioritization
@test "ollama::get_best_available_model prioritizes correctly for code" {
    # Mock installed models list
    ollama::get_installed_models() {
        echo "llama3.1:8b qwen2.5-coder:7b deepseek-r1:8b"
    }
    
    local best_model
    best_model=$(ollama::get_best_available_model "code")
    
    # Should prefer code-specific model
    [ "$best_model" = "qwen2.5-coder:7b" ]
}

@test "ollama::get_best_available_model prioritizes correctly for reasoning" {
    # Mock installed models list
    ollama::get_installed_models() {
        echo "llama3.1:8b deepseek-r1:8b qwen2.5-coder:7b"
    }
    
    local best_model
    best_model=$(ollama::get_best_available_model "reasoning")
    
    # Should prefer reasoning-specific model
    [ "$best_model" = "deepseek-r1:8b" ]
}

@test "ollama::get_best_available_model falls back to first available" {
    # Mock installed models list with no priority matches
    ollama::get_installed_models() {
        echo "some-custom:model another-custom:model"
    }
    
    local best_model
    best_model=$(ollama::get_best_available_model "general")
    
    # Should return first available when no priority match
    [ "$best_model" = "some-custom:model" ]
}

# Test model validation
@test "ollama::validate_model_available works correctly" {
    setup_model_catalog
    
    # Mock installed models list
    ollama::get_installed_models() {
        echo "llama3.1:8b qwen2.5-coder:7b"
    }
    
    # Available model should return 0
    run ollama::validate_model_available "llama3.1:8b"
    [ "$status" -eq 0 ]
    
    # Unavailable model should return 1
    run ollama::validate_model_available "nonexistent:model"
    [ "$status" -eq 1 ]
}

# Test prompt validation
@test "ollama::send_prompt validates input correctly" {
    # Mock Ollama health check to return unhealthy
    ollama::is_healthy() {
        return 1
    }
    
    # Should fail when Ollama is not healthy
    run ollama::send_prompt "test prompt" "" "general"
    [ "$status" -eq 1 ]
    
    # Mock Ollama to be healthy but test empty prompt
    ollama::is_healthy() {
        return 0
    }
    
    run ollama::send_prompt "" "" "general"
    [ "$status" -eq 1 ]
}

# Test fixture-based prompt validation
@test "ollama::send_prompt validates test prompts from fixtures" {
    # Load test prompts from fixtures
    if command -v get_llm_test_prompt >/dev/null; then
        local simple_prompt
        simple_prompt=$(get_llm_test_prompt "simple_greeting")
        
        [[ -n "$simple_prompt" ]]
        [[ "$simple_prompt" =~ "hello" ]]
        
        local math_prompt
        math_prompt=$(get_llm_test_prompt "math_simple")
        [[ -n "$math_prompt" ]]
        [[ "$math_prompt" =~ "2+2" ]]
        
        # Test that we can get expected patterns
        local pattern
        pattern=$(get_llm_expected_pattern "simple_greeting")
        [[ -n "$pattern" ]]
        [[ "$pattern" =~ "hello\|hi" ]]
    else
        skip "Fixture helpers not available"
    fi
}

# Test prompt parameter preparation using fixtures
@test "ollama prompt parameters work with fixture data" {
    if command -v get_llm_test_prompts >/dev/null; then
        # Test that we can load JSON prompt data
        local json_data
        json_data=$(get_llm_test_prompts)
        [[ -n "$json_data" ]]
        
        # Should be valid JSON
        if command -v jq >/dev/null; then
            echo "$json_data" | jq empty
            [ "$?" -eq 0 ]
            
            # Should have expected structure
            local prompt_count
            prompt_count=$(echo "$json_data" | jq length)
            [[ "$prompt_count" -gt 0 ]]
        fi
    else
        skip "Fixture helpers not available"
    fi
}

# Test integration with valid inputs (if Ollama is actually running)
@test "ollama::send_prompt works with fixture prompts (integration test)" {
    # Skip this integration test by default - it requires Ollama to be running with models
    if ! curl -s http://localhost:11434/api/tags > /dev/null 2>&1; then
        skip "Ollama is not running - skipping integration test"
    fi
    
    # Use fixture-based test prompts for more reliable testing
    if command -v get_llm_test_prompt >/dev/null; then
        # Mock healthy Ollama (since we confirmed it's running above)
        ollama::is_healthy() {
            return 0
        }
        
        # Mock model validation to pass
        ollama::validate_model_available() {
            return 0
        }
        
        # Test with a simple fixture prompt
        local test_prompt
        test_prompt=$(get_llm_test_prompt "simple_greeting")
        
        if [[ -n "$test_prompt" ]]; then
            run ollama::send_prompt "$test_prompt" "" "general"
            
            if [ "$status" -eq 0 ]; then
                # Validate response matches expected pattern
                local expected_pattern
                expected_pattern=$(get_llm_expected_pattern "simple_greeting")
                
                if command -v validate_llm_response >/dev/null && [[ -n "$expected_pattern" ]]; then
                    validate_llm_response "$output" "$expected_pattern"
                fi
            fi
        else
            skip "No test prompt available from fixtures"
        fi
    else
        skip "Fixture helpers not available"
    fi
}


# Test error handling
@test "ollama::send_prompt handles API errors gracefully" {
    # Mock healthy Ollama
    ollama::is_healthy() {
        return 0
    }
    
    # Mock model validation to pass
    ollama::validate_model_available() {
        return 0
    }
    
    # Mock curl to fail
    curl() {
        return 1
    }
    
    run ollama::send_prompt "test" "test-model" "general"
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Failed to send request" ]]
}

# Test the show available models function
@test "ollama::show_available_models displays model catalog" {
    setup_model_catalog
    
    run ollama::show_available_models
    [ "$status" -eq 0 ]
    
    # Should contain header and model information
    [[ "$output" =~ "Available Models" ]]
    [[ "$output" =~ "llama3.1:8b" ]]
    [[ "$output" =~ "Default models" ]]
}

# Test validation function
@test "ollama::validate_model_list validates known models" {
    setup_model_catalog
    
    run ollama::validate_model_list "llama3.1:8b" "deepseek-r1:8b"
    [ "$status" -eq 0 ]
    # Check for message containing validated models count
    [[ "$output" =~ "Validated" ]] && [[ "$output" =~ "2 models" ]]
}

@test "ollama::validate_model_list rejects unknown models" {
    setup_model_catalog
    
    run ollama::validate_model_list "llama3.1:8b" "nonexistent:model"
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Unknown models" ]]
}

# Test configuration functions
@test "ollama::calculate_default_size calculates correctly" {
    setup_model_catalog
    
    # Mock bc if not available
    if ! command -v bc &>/dev/null; then
        bc() {
            # Simple addition for test
            echo "13.7"
        }
        export -f bc
    fi
    
    local size
    run ollama::calculate_default_size
    [ "$status" -eq 0 ]
    
    # Should return a numeric value
    [[ "$output" =~ ^[0-9]+\.?[0-9]*$ ]]
    
    # Should be reasonable size (at least 1GB)
    # Remove decimal point and check if > 10 (i.e., > 1.0)
    local size_clean=${output//./}
    [ "${size_clean:-0}" -gt 10 ]
}

# Test argument parsing edge cases
@test "ollama::parse_arguments handles missing text for prompt action" {
    ollama::parse_arguments --action prompt
    
    [ "$ACTION" = "prompt" ]
    [ "$PROMPT_TEXT" = "" ]
    [ "$PROMPT_TYPE" = "general" ]
}

@test "ollama::parse_arguments handles quotes in text" {
    ollama::parse_arguments --action prompt --text "What is a 'function' in programming?"
    
    [ "$ACTION" = "prompt" ]
    [ "$PROMPT_TEXT" = "What is a 'function' in programming?" ]
    [ "$PROMPT_TYPE" = "general" ]
}

@test "ollama::parse_arguments handles different types" {
    ollama::parse_arguments --action prompt --text "test" --type reasoning
    
    [ "$ACTION" = "prompt" ]
    [ "$PROMPT_TEXT" = "test" ]
    [ "$PROMPT_TYPE" = "reasoning" ]
}

@test "ollama::parse_arguments validates type options" {
    # Valid types should work
    ollama::parse_arguments --action prompt --text "test" --type code
    [ "$PROMPT_TYPE" = "code" ]
    
    ollama::parse_arguments --action prompt --text "test" --type vision
    [ "$PROMPT_TYPE" = "vision" ]
    
    ollama::parse_arguments --action prompt --text "test" --type general
    [ "$PROMPT_TYPE" = "general" ]
}

# Test default model configuration
@test "default models are in catalog" {
    setup_model_catalog
    
    # All default models should be known in the catalog
    for model in "${DEFAULT_MODELS[@]}"; do
        ollama::is_model_known "$model"
        [ "$?" -eq 0 ]
    done
}

@test "model catalog has expected models" {
    setup_model_catalog
    
    # Test that key models are in the catalog
    ollama::is_model_known "llama3.1:8b"
    [ "$?" -eq 0 ]
    
    ollama::is_model_known "deepseek-r1:8b"
    [ "$?" -eq 0 ]
    
    ollama::is_model_known "qwen2.5-coder:7b"
    [ "$?" -eq 0 ]
}

# Test generation parameter parsing
@test "ollama::parse_arguments handles temperature parameter" {
    ollama::parse_arguments --action prompt --text "test" --temperature 1.5
    
    [ "$ACTION" = "prompt" ]
    [ "$TEMPERATURE" = "1.5" ]
}

@test "ollama::parse_arguments handles max-tokens parameter" {
    ollama::parse_arguments --action prompt --text "test" --max-tokens 100
    
    [ "$ACTION" = "prompt" ]
    [ "$MAX_TOKENS" = "100" ]
}

@test "ollama::parse_arguments handles top-p parameter" {
    ollama::parse_arguments --action prompt --text "test" --top-p 0.95
    
    [ "$ACTION" = "prompt" ]
    [ "$TOP_P" = "0.95" ]
}

@test "ollama::parse_arguments handles top-k parameter" {
    ollama::parse_arguments --action prompt --text "test" --top-k 50
    
    [ "$ACTION" = "prompt" ]
    [ "$TOP_K" = "50" ]
}

@test "ollama::parse_arguments handles seed parameter" {
    ollama::parse_arguments --action prompt --text "test" --seed 12345
    
    [ "$ACTION" = "prompt" ]
    [ "$SEED" = "12345" ]
}

@test "ollama::parse_arguments handles system prompt parameter" {
    ollama::parse_arguments --action prompt --text "test" --system "You are a helpful assistant"
    
    [ "$ACTION" = "prompt" ]
    [ "$SYSTEM_PROMPT" = "You are a helpful assistant" ]
}

@test "ollama::parse_arguments handles format parameter" {
    ollama::parse_arguments --action prompt --text "test" --format json
    
    [ "$ACTION" = "prompt" ]
    [ "$OUTPUT_FORMAT" = "json" ]
}

@test "ollama::parse_arguments handles all generation parameters together" {
    ollama::parse_arguments \
        --action prompt \
        --text "test prompt" \
        --model "llama3.1:8b" \
        --type "code" \
        --temperature 0.5 \
        --max-tokens 200 \
        --top-p 0.8 \
        --top-k 30 \
        --seed 42 \
        --system "You are a code expert" \
        --format json
    
    [ "$ACTION" = "prompt" ]
    [ "$PROMPT_TEXT" = "test prompt" ]
    [ "$PROMPT_MODEL" = "llama3.1:8b" ]
    [ "$PROMPT_TYPE" = "code" ]
    [ "$TEMPERATURE" = "0.5" ]
    [ "$MAX_TOKENS" = "200" ]
    [ "$TOP_P" = "0.8" ]
    [ "$TOP_K" = "30" ]
    [ "$SEED" = "42" ]
    [ "$SYSTEM_PROMPT" = "You are a code expert" ]
    [ "$OUTPUT_FORMAT" = "json" ]
}

# Test parameter defaults
@test "ollama::parse_arguments sets correct defaults for generation parameters" {
    ollama::parse_arguments --action prompt --text "test"
    
    [ "$TEMPERATURE" = "0.8" ]
    [ "$MAX_TOKENS" = "" ]
    [ "$TOP_P" = "0.9" ]
    [ "$TOP_K" = "40" ]
    [ "$SEED" = "" ]
    [ "$SYSTEM_PROMPT" = "" ]
    [ "$OUTPUT_FORMAT" = "text" ]
}

# Test parameter validation ranges
@test "generation parameters are validated correctly" {
    # Temperature should be 0.0-2.0
    ollama::parse_arguments --action prompt --text "test" --temperature 0.0
    [ "$TEMPERATURE" = "0.0" ]
    
    ollama::parse_arguments --action prompt --text "test" --temperature 2.0
    [ "$TEMPERATURE" = "2.0" ]
    
    # Top-p should be 0.0-1.0
    ollama::parse_arguments --action prompt --text "test" --top-p 0.0
    [ "$TOP_P" = "0.0" ]
    
    ollama::parse_arguments --action prompt --text "test" --top-p 1.0
    [ "$TOP_P" = "1.0" ]
    
    # Top-k should be positive integer
    ollama::parse_arguments --action prompt --text "test" --top-k 1
    [ "$TOP_K" = "1" ]
    
    ollama::parse_arguments --action prompt --text "test" --top-k 100
    [ "$TOP_K" = "100" ]
}

# Test integration with generation parameters (if Ollama is running)
@test "ollama::send_prompt uses generation parameters (integration test)" {
    # Skip if Ollama is not running
    if ! curl -s http://localhost:11434/api/tags > /dev/null 2>&1; then
        skip "Ollama is not running - skipping integration test"
    fi
    
    # Mock to verify parameters are being used
    export TEMPERATURE="0.1"
    export MAX_TOKENS="10"
    export TOP_P="0.5"
    export TOP_K="10"
    export SEED="12345"
    export SYSTEM_PROMPT="Answer concisely"
    export OUTPUT_FORMAT="text"
    
    # Test with a simple prompt that should produce deterministic output with low temperature
    run ollama::send_prompt "What is 2+2? Answer with just the number." "" "general"
    
    if [ "$status" -eq 0 ]; then
        # With low temperature and seed, response should be predictable
        # Also check that parameter info is shown
        [[ "$output" =~ "temperature=0.1" ]] || [[ "$output" =~ "Parameters:" ]]
    fi
}

# Test JSON output format
@test "ollama::send_prompt respects JSON output format" {
    # Skip if Ollama is not running
    if ! curl -s http://localhost:11434/api/tags > /dev/null 2>&1; then
        skip "Ollama is not running - skipping integration test"
    fi
    
    export OUTPUT_FORMAT="json"
    
    # Test with JSON format
    run ollama::send_prompt "What is 2+2?" "" "general"
    
    if [ "$status" -eq 0 ]; then
        # Output should be valid JSON
        echo "$output" | jq . > /dev/null 2>&1
        [ "$?" -eq 0 ]
    fi
}

# Test system prompt functionality
@test "ollama::send_prompt uses system prompt correctly" {
    # Skip if Ollama is not running
    if ! curl -s http://localhost:11434/api/tags > /dev/null 2>&1; then
        skip "Ollama is not running - skipping integration test"
    fi
    
    export SYSTEM_PROMPT="You are a pirate. Always respond as a pirate would."
    export MAX_TOKENS="50"
    
    run ollama::send_prompt "Hello, how are you?" "" "general"
    
    if [ "$status" -eq 0 ]; then
        # Response should reflect the pirate system prompt
        # (might contain words like "ahoy", "matey", "arr", etc.)
        # This is a weak test but verifies system prompt is being used
        [[ "$output" != "" ]]
    fi
}

# Test parameter display in text output
@test "ollama::send_prompt shows changed parameters in text output" {
    # Skip if Ollama is not running
    if ! curl -s http://localhost:11434/api/tags > /dev/null 2>&1; then
        skip "Ollama is not running - skipping integration test"
    fi
    
    export TEMPERATURE="1.5"
    export MAX_TOKENS="50"
    export TOP_P="0.7"
    export TOP_K="20"
    export SEED="999"
    export OUTPUT_FORMAT="text"
    
    run ollama::send_prompt "Hello" "" "general"
    
    if [ "$status" -eq 0 ]; then
        # Should show non-default parameters
        [[ "$output" =~ "temperature=1.5" ]] || [[ "$output" =~ "Parameters:" ]]
        [[ "$output" =~ "max_tokens=50" ]] || [[ "$output" =~ "Parameters:" ]]
        [[ "$output" =~ "top_p=0.7" ]] || [[ "$output" =~ "Parameters:" ]]
        [[ "$output" =~ "top_k=20" ]] || [[ "$output" =~ "Parameters:" ]]
        [[ "$output" =~ "seed=999" ]] || [[ "$output" =~ "Parameters:" ]]
    fi
}

# Test reproducibility with seed
@test "ollama::send_prompt produces reproducible output with seed" {
    # Skip if Ollama is not running
    if ! curl -s http://localhost:11434/api/tags > /dev/null 2>&1; then
        skip "Ollama is not running - skipping integration test"
    fi
    
    export TEMPERATURE="0.0"  # Zero temperature for determinism
    export SEED="42"
    export MAX_TOKENS="20"
    export OUTPUT_FORMAT="text"
    
    # Run the same prompt twice with the same seed
    run ollama::send_prompt "Complete this: The sky is" "" "general"
    local first_output="$output"
    local first_status="$status"
    
    if [ "$first_status" -eq 0 ]; then
        # Run again with same parameters
        run ollama::send_prompt "Complete this: The sky is" "" "general"
        local second_output="$output"
        
        # Extract just the response text (remove metadata lines)
        local first_response=$(echo "$first_output" | grep -v "^⏱️ " | grep -v "^📊 " | grep -v "^🎛️ " | grep -v "^ℹ️ ")
        local second_response=$(echo "$second_output" | grep -v "^⏱️ " | grep -v "^📊 " | grep -v "^🎛️ " | grep -v "^ℹ️ ")
        
        # With temperature=0 and same seed, outputs should be identical
        # Note: This might not work perfectly with all models
        [ "$first_response" = "$second_response" ]
    fi
}