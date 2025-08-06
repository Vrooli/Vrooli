#!/usr/bin/env bats
# Ollama Mock Test Suite
# Comprehensive test coverage for the Ollama mock implementation

bats_require_minimum_version 1.5.0

# Setup test environment
setup() {
    # Create a temporary directory for this test session
    export MOCK_LOG_DIR="${BATS_TMPDIR}/ollama_mock_tests"
    mkdir -p "$MOCK_LOG_DIR"
    
    # Set a consistent state file path for this test session
    export OLLAMA_MOCK_STATE_FILE="${BATS_TMPDIR}/ollama_mock_state_test"
    
    # Load test helpers if available
    if [[ -f "${BATS_TEST_DIRNAME}/../../../helpers/bats-support/load.bash" ]]; then
        load "${BATS_TEST_DIRNAME}/../../../helpers/bats-support/load.bash"
    fi
    if [[ -f "${BATS_TEST_DIRNAME}/../../../helpers/bats-assert/load.bash" ]]; then
        load "${BATS_TEST_DIRNAME}/../../../helpers/bats-assert/load.bash"
    else
        # Define basic assertion functions if bats-assert is not available
        assert_success() {
            if [[ "$status" -ne 0 ]]; then
                echo "expected success, got status $status"
                echo "output: $output"
                return 1
            fi
        }
        
        assert_failure() {
            if [[ "$status" -eq 0 ]]; then
                echo "expected failure, got success"
                echo "output: $output"
                return 1
            fi
        }
        
        assert_output() {
            local expected
            if [[ "$1" == "--partial" ]]; then
                shift
                expected="$1"
                if [[ "$output" != *"$expected"* ]]; then
                    echo "expected output to contain: $expected"
                    echo "actual output: $output"
                    return 1
                fi
            else
                expected="${1:-}"
                if [[ "$output" != "$expected" ]]; then
                    echo "expected: $expected"
                    echo "actual: $output"
                    return 1
                fi
            fi
        }
        
        refute_output() {
            local expected
            if [[ "$1" == "--partial" ]]; then
                shift
                expected="$1"
                if [[ "$output" == *"$expected"* ]]; then
                    echo "expected output to NOT contain: $expected"
                    echo "actual output: $output"
                    return 1
                fi
            else
                expected="${1:-}"
                if [[ "$output" == "$expected" ]]; then
                    echo "expected output to NOT be: $expected"
                    echo "actual output: $output"
                    return 1
                fi
            fi
        }
    fi
    
    # Load the mock system
    source "${BATS_TEST_DIRNAME}/logs.sh"
    source "${BATS_TEST_DIRNAME}/verification.sh" 
    source "${BATS_TEST_DIRNAME}/ollama.sh"
    
    # Initialize the mock systems
    mock::init_logging "$MOCK_LOG_DIR"
    mock::verify::init
    
    # Reset the mock to a known state
    echo "[DEBUG] setup() calling mock::ollama::reset" >&2
    mock::ollama::reset
}

teardown() {
    # Clean up temporary files
    if [[ -n "${OLLAMA_MOCK_STATE_FILE:-}" && -f "$OLLAMA_MOCK_STATE_FILE" ]]; then
        rm -f "$OLLAMA_MOCK_STATE_FILE"
    fi
    
    # Clean up mock log directory
    if [[ -d "$MOCK_LOG_DIR" ]]; then
        rm -rf "$MOCK_LOG_DIR"
    fi
}

# ----------------------------
# Basic Mock Functionality Tests
# ----------------------------

@test "ollama mock loads without errors" {
    # Test basic loading
    run echo "Mock loaded successfully"
    assert_success
    
    # Test that the mock loaded flag is set
    [[ "$OLLAMA_MOCK_LOADED" == "true" ]]
    
    # Test that basic functions are available
    declare -F ollama >/dev/null
    declare -F mock::ollama::reset >/dev/null
    declare -F mock::ollama::set_mode >/dev/null
}

@test "ollama mock state can be reset" {
    # Add a custom model first
    mock::ollama::add_model "test-model:1b"
    
    # Verify model was added
    run mock::ollama::get::installed_models
    assert_output --partial "test-model:1b"
    
    # Reset the mock (don't use 'run' so it affects the shared state)
    mock::ollama::reset
    
    # Verify default models are present and custom model is gone
    run mock::ollama::get::installed_models
    assert_output --partial "llama3.1:8b"
    assert_output --partial "deepseek-r1:8b"
    refute_output --partial "test-model:1b"
}

@test "ollama mock modes work correctly" {
    # Test healthy mode (default)
    [[ "$(mock::ollama::get::mode)" == "healthy" ]]
    
    # Test setting different modes
    for mode in unhealthy installing stopped offline error; do
        run mock::ollama::set_mode "$mode"
        assert_success
        [[ "$(mock::ollama::get::mode)" == "$mode" ]]
    done
    
    # Test invalid mode
    run mock::ollama::set_mode "invalid_mode"
    assert_failure
    assert_output --partial "Invalid Ollama mode"
}

# ----------------------------
# CLI Command Tests
# ----------------------------

@test "ollama list command works" {
    # Test in healthy mode
    run ollama list
    assert_success
    assert_output --partial "NAME"
    assert_output --partial "llama3.1:8b"
    assert_output --partial "deepseek-r1:8b"
    assert_output --partial "qwen2.5-coder:7b"
    
    # Test that models have proper formatting
    assert_output --partial "ID"
    assert_output --partial "SIZE"
    assert_output --partial "MODIFIED"
    assert_output --partial "GB"
}

@test "ollama list respects mock modes" {
    # Test unhealthy mode
    mock::ollama::set_mode "unhealthy"
    run ollama list
    assert_failure
    assert_output --partial "Ollama server is not ready"
    
    # Test installing mode
    mock::ollama::set_mode "installing"
    run ollama list  
    assert_failure
    assert_output --partial "Ollama server is not ready"
    
    # Test offline mode
    mock::ollama::set_mode "offline"
    run ollama list
    assert_failure
    assert_output --partial "Failed to connect to ollama server"
}

@test "ollama pull command works" {
    # Test pulling a new model
    run ollama pull "mistral:7b"
    assert_success
    assert_output --partial "pulling manifest"
    assert_output --partial "success"
    
    # Verify model was added
    run mock::ollama::assert::model_installed "mistral:7b"
    assert_success
    
    # Test pulling existing model
    run ollama pull "llama3.1:8b"
    assert_success
    assert_output --partial "is up to date"
    
    # Test pull without model name
    run ollama pull
    assert_failure
    assert_output --partial "missing model name"
}

@test "ollama pull respects mock modes" {
    # Test offline mode
    mock::ollama::set_mode "offline"
    run ollama pull "test-model:1b"
    assert_failure
    assert_output --partial "Failed to connect to ollama server"
    
    # Test installing mode
    mock::ollama::set_mode "installing" 
    run ollama pull "test-model:1b"
    assert_failure
    assert_output --partial "Ollama is still initializing"
}

@test "ollama run command works" {
    # Test running existing model
    run ollama run "llama3.1:8b" "Hello world"
    assert_success
    assert_output --partial "Mock response to: Hello world"
    assert_output --partial "simulated response from model llama3.1:8b"
    
    # Verify model is marked as running
    run mock::ollama::assert::model_running "llama3.1:8b"
    assert_success
    
    # Test running non-existent model
    run ollama run "non-existent:1b" "test"
    assert_failure
    assert_output --partial "model 'non-existent:1b' not found"
    
    # Test without prompt (interactive mode)
    run ollama run "llama3.1:8b"
    assert_success
    assert_output --partial "Send a message"
    assert_output --partial "Interactive mode not fully supported in mock"
}

@test "ollama rm command works" {
    # Test removing existing model
    run ollama rm "deepseek-r1:8b"
    assert_success
    assert_output --partial "deleted 'deepseek-r1:8b'"
    
    # Verify model was removed
    run mock::ollama::get::installed_models
    refute_output --partial "deepseek-r1:8b"
    
    # Test removing non-existent model
    run ollama rm "non-existent:1b"
    assert_failure
    assert_output --partial "model 'non-existent:1b' not found"
    
    # Test without model name
    run ollama rm
    assert_failure
    assert_output --partial "missing model name"
}

@test "ollama show command works" {
    # Test showing existing model
    run ollama show "llama3.1:8b"
    assert_success
    assert_output --partial "Model"
    assert_output --partial "architecture"
    assert_output --partial "parameters"
    assert_output --partial "Modelfile"
    assert_output --partial "8B"
    
    # Test showing non-existent model
    run ollama show "non-existent:1b"
    assert_failure
    assert_output --partial "model 'non-existent:1b' not found"
}

@test "ollama ps command works" {
    # Initially no running models
    run ollama ps
    assert_success
    assert_output --partial "NAME"
    assert_output --partial "ID" 
    assert_output --partial "PROCESSOR"
    
    # Add a running model
    mock::ollama::set_model_running "llama3.1:8b" "4.5 GB"
    
    # Test listing running models
    run ollama ps
    assert_success
    assert_output --partial "llama3.1:8b"
    assert_output --partial "4.5 GB"
    assert_output --partial "100% GPU"
}

@test "ollama stop command works" {
    # Set a model as running first
    mock::ollama::set_model_running "llama3.1:8b" "4.5 GB"
    
    # Stop the model
    run ollama stop "llama3.1:8b"
    assert_success
    assert_output --partial "stopped 'llama3.1:8b'"
    
    # Verify model is no longer running
    run mock::ollama::assert::model_running "llama3.1:8b"
    assert_failure
    
    # Test stopping non-running model
    run ollama stop "deepseek-r1:8b"
    assert_success
    assert_output --partial "is not currently loaded"
}

@test "ollama create command works" {
    # Test creating a model
    run ollama create "custom-model:1b" "-f Modelfile"
    assert_success
    assert_output --partial "transferring model data"
    assert_output --partial "success"
    
    # Verify model was created
    run mock::ollama::assert::model_installed "custom-model:1b"
    assert_success
    
    # Test without model name
    run ollama create
    assert_failure
    assert_output --partial "missing model name"
}

@test "ollama copy command works" {
    # Test copying existing model
    run ollama cp "llama3.1:8b" "llama3.1:8b-backup"
    assert_success
    assert_output --partial "copied 'llama3.1:8b' to 'llama3.1:8b-backup'"
    
    # Verify copy exists
    run mock::ollama::assert::model_installed "llama3.1:8b-backup"
    assert_success
    
    # Test copying non-existent model
    run ollama cp "non-existent:1b" "backup:1b"
    assert_failure
    assert_output --partial "model 'non-existent:1b' not found"
}

@test "ollama push command works" {
    # Test pushing existing model
    run ollama push "llama3.1:8b"
    assert_success
    assert_output --partial "pushing llama3.1:8b"
    assert_output --partial "success"
    
    # Test pushing non-existent model
    run ollama push "non-existent:1b"
    assert_failure
    assert_output --partial "model 'non-existent:1b' not found"
}

@test "ollama serve command works" {
    run ollama serve
    assert_success
    assert_output --partial "Ollama server starting"
    assert_output --partial "Listening on 127.0.0.1:11434"
}

@test "ollama help command works" {
    run ollama help
    assert_success
    assert_output --partial "Usage:"
    assert_output --partial "Available Commands:"
    assert_output --partial "serve"
    assert_output --partial "pull"
    assert_output --partial "list"
    
    # Test help flag variations
    run ollama --help
    assert_success
    assert_output --partial "Usage:"
    
    run ollama -h
    assert_success
    assert_output --partial "Usage:"
}

@test "ollama version command works" {
    run ollama version
    assert_success
    assert_output --partial "ollama version is 0.1.38"
    
    # Test version flag variations
    run ollama --version
    assert_success
    assert_output --partial "ollama version is 0.1.38"
    
    run ollama -v
    assert_success
    assert_output --partial "ollama version is 0.1.38"
}

@test "ollama handles unknown commands" {
    run ollama unknown_command
    assert_failure
    assert_output --partial "unknown command 'unknown_command'"
    assert_output --partial "Run 'ollama --help' for usage"
}

# ----------------------------
# Error Injection Tests
# ----------------------------

@test "ollama handles injected errors" {
    # Test connection error
    mock::ollama::inject_error "list" "connection"
    run ollama list
    assert_failure
    assert_output --partial "Failed to connect to ollama server"
    
    # Test timeout error
    mock::ollama::reset
    mock::ollama::inject_error "pull" "timeout"
    run ollama pull "test:1b"
    assert_failure
    assert_output --partial "Request timed out"
    
    # Test not found error
    mock::ollama::reset
    mock::ollama::inject_error "show" "not_found"
    run ollama show "llama3.1:8b"
    assert_failure
    assert_output --partial "model not found"
    
    # Test disk space error
    mock::ollama::reset
    mock::ollama::inject_error "pull" "disk_space"
    run ollama pull "test:1b"
    assert_failure
    assert_output --partial "insufficient disk space"
}

# ----------------------------
# Model Management Tests
# ----------------------------

@test "model addition and removal works" {
    # Add a model
    mock::ollama::add_model "test-model:3b" "2024-01-16T12:00:00Z"
    
    # Verify model is installed
    run mock::ollama::assert::model_installed "test-model:3b"
    assert_success
    
    # Verify model shows up in list
    run ollama list
    assert_success
    assert_output --partial "test-model:3b"
    
    # Remove the model
    mock::ollama::remove_model "test-model:3b"
    
    # Verify model is no longer installed
    run mock::ollama::assert::model_installed "test-model:3b"
    assert_failure
    
    # Verify model no longer shows in list
    run ollama list
    assert_success
    refute_output --partial "test-model:3b"
}

@test "running model management works" {
    # Set model as running
    mock::ollama::set_model_running "llama3.1:8b" "4.5 GB"
    
    # Verify model is running
    run mock::ollama::assert::model_running "llama3.1:8b"
    assert_success
    
    # Verify model shows in ps output
    run ollama ps
    assert_success
    assert_output --partial "llama3.1:8b"
    assert_output --partial "4.5 GB"
    
    # Stop the model
    run ollama stop "llama3.1:8b"
    assert_success
    
    # Verify model is no longer running
    run mock::ollama::assert::model_running "llama3.1:8b"
    assert_failure
}

@test "model info generation works correctly" {
    # Test different model sizes
    mock::ollama::add_model "tiny:1b"
    mock::ollama::add_model "small:3b"
    mock::ollama::add_model "medium:7b"
    mock::ollama::add_model "large:13b"
    mock::ollama::add_model "xlarge:32b"
    
    # Test model families
    mock::ollama::add_model "llama2:7b"
    mock::ollama::add_model "mistral:7b"
    mock::ollama::add_model "deepseek-coder:6.7b"
    mock::ollama::add_model "qwen2:7b"
    mock::ollama::add_model "phi-3:3b"
    
    # Verify all models are listed
    run ollama list
    assert_success
    assert_output --partial "tiny:1b"
    assert_output --partial "small:3b"
    assert_output --partial "medium:7b"
    assert_output --partial "large:13b"
    assert_output --partial "xlarge:32b"
    assert_output --partial "llama2:7b"
    assert_output --partial "mistral:7b"
    assert_output --partial "deepseek-coder:6.7b"
    assert_output --partial "qwen2:7b"
    assert_output --partial "phi-3:3b"
}

# ----------------------------
# Call Tracking Tests
# ----------------------------

@test "command call tracking works" {
    # Initial state - no calls
    [[ "$(mock::ollama::get::call_count "list")" == "0" ]]
    
    # Make some calls
    run ollama list
    assert_success
    [[ "$(mock::ollama::get::call_count "list")" == "1" ]]
    
    run ollama list
    assert_success 
    [[ "$(mock::ollama::get::call_count "list")" == "2" ]]
    
    # Test different commands
    run ollama pull "test:1b"
    assert_success
    [[ "$(mock::ollama::get::call_count "pull")" == "1" ]]
    [[ "$(mock::ollama::get::call_count "list")" == "2" ]]
    
    # Test assertion helper
    run mock::ollama::assert::command_called "list" 2
    assert_success
    
    run mock::ollama::assert::command_called "pull" 1
    assert_success
    
    run mock::ollama::assert::command_called "list" 5
    assert_failure
    assert_output --partial "called 2 times, expected at least 5"
}

# ----------------------------
# Scenario Tests
# ----------------------------

@test "healthy scenario works correctly" {
    # Create healthy scenario
    run mock::ollama::scenario::healthy_with_models
    assert_success
    assert_output --partial "Created healthy Ollama scenario"
    
    # Verify mode is healthy
    [[ "$(mock::ollama::get::mode)" == "healthy" ]]
    
    # Verify models are available
    run ollama list
    assert_success
    assert_output --partial "llama3.1:8b"
    
    # Verify at least one model is running
    run ollama ps
    assert_success
    assert_output --partial "llama3.1:8b"
}

@test "installing scenario works correctly" {
    # Create installing scenario
    run mock::ollama::scenario::installing
    assert_success
    assert_output --partial "Created installing Ollama scenario"
    
    # Verify mode is installing
    [[ "$(mock::ollama::get::mode)" == "installing" ]]
    
    # Verify commands fail appropriately
    run ollama list
    assert_failure
    assert_output --partial "not ready"
    
    run ollama pull "test:1b"
    assert_failure
    assert_output --partial "still initializing"
}

@test "offline scenario works correctly" {
    # Create offline scenario
    run mock::ollama::scenario::offline
    assert_success
    assert_output --partial "Created offline Ollama scenario"
    
    # Verify mode is offline
    [[ "$(mock::ollama::get::mode)" == "offline" ]]
    
    # Verify all commands fail with connection error
    run ollama list
    assert_failure
    assert_output --partial "Failed to connect to ollama server"
    
    run ollama pull "test:1b"
    assert_failure
    assert_output --partial "Failed to connect to ollama server"
    
    run ollama run "llama3.1:8b" "hello"
    assert_failure
    assert_output --partial "Failed to connect to ollama server"
}

# ----------------------------
# State Persistence Tests (Subshell compatibility)
# ----------------------------

@test "state persists across subshells" {
    # Add a model in main shell
    mock::ollama::add_model "persistence-test:1b"
    
    # Verify model exists in subshell by checking the state file directly
    local state_file="$OLLAMA_MOCK_STATE_FILE"
    [[ -f "$state_file" ]]
    
    # Test that subshell can read the models
    result=$(bash -c 'source "'"${BATS_TEST_DIRNAME}/ollama.sh"'"; mock::ollama::get::installed_models' | grep -c "persistence-test:1b" || echo 0)
    [[ "$result" == "1" ]]
    
    # Test call count tracking more directly
    # Make a call in main shell
    run ollama version
    assert_success
    
    # Verify call count
    [[ "$(mock::ollama::get::call_count "version")" == "1" ]]
    
    # Test that the state file contains the call count
    grep -q "MOCK_OLLAMA_CALL_COUNT\[version\]" "$state_file"
}

@test "mode changes persist across subshells" {
    # Change mode
    mock::ollama::set_mode "offline"
    
    # Verify mode persists by checking the state file
    local state_file="$OLLAMA_MOCK_STATE_FILE"
    grep -q "OLLAMA_MOCK_MODE='offline'" "$state_file"
    
    # Verify mode is readable in subshell
    result=$(bash -c 'source "'"${BATS_TEST_DIRNAME}/ollama.sh"'"; mock::ollama::get::mode')
    [[ "$result" == "offline" ]]
    
    # Test mode directly in main shell
    run ollama list
    assert_failure
    assert_output --partial "Failed to connect to ollama server"
}

# ----------------------------
# Integration with HTTP Mock Tests  
# ----------------------------

@test "HTTP endpoints are configured correctly" {
    # This test verifies the HTTP mock integration works
    # Only run if HTTP mock functions are available
    if declare -F mock::http::set_endpoint_response >/dev/null 2>&1; then
        # Test that HTTP endpoints are set up based on mode
        mock::ollama::set_mode "healthy"
        
        # The HTTP setup should be called automatically
        # We can verify by checking if HTTP mock functions would return expected responses
        # (This is more of an integration test with the HTTP mock system)
        
        skip "HTTP mock integration test - requires full HTTP mock system"
    else
        skip "HTTP mock system not available"
    fi
}

# ----------------------------
# Debug and Utility Tests
# ----------------------------

@test "debug dump works" {
    # Add some test data
    mock::ollama::add_model "debug-test:1b"
    mock::ollama::set_model_running "llama3.1:8b" "4.5 GB"
    mock::ollama::inject_error "pull" "timeout"
    
    # Make some calls to increment counters
    run ollama list
    run ollama list
    run ollama ps
    
    # Test debug dump
    run mock::ollama::debug::dump_state
    assert_success
    assert_output --partial "=== Ollama Mock State Dump ==="
    assert_output --partial "Mode: healthy"
    assert_output --partial "Port: 11434"
    assert_output --partial "Base URL: http://localhost:11434"
    assert_output --partial "Installed Models:"
    assert_output --partial "debug-test:1b"
    assert_output --partial "Running Models:"
    assert_output --partial "llama3.1:8b: 4.5 GB"
    assert_output --partial "Command Call Counts:"
    assert_output --partial "list: 2"
    assert_output --partial "ps: 1"
    assert_output --partial "Injected Errors:"
    assert_output --partial "pull: timeout"
}

@test "get functions work correctly" {
    # Test get::installed_models
    run mock::ollama::get::installed_models
    assert_success
    assert_output --partial "llama3.1:8b"
    assert_output --partial "deepseek-r1:8b"
    assert_output --partial "qwen2.5-coder:7b"
    
    # Test get::mode
    [[ "$(mock::ollama::get::mode)" == "healthy" ]]
    
    mock::ollama::set_mode "offline"
    [[ "$(mock::ollama::get::mode)" == "offline" ]]
    
    # Reset to healthy for the call count test
    mock::ollama::set_mode "healthy"
    
    # Test get::call_count
    [[ "$(mock::ollama::get::call_count "nonexistent")" == "0" ]]
    
    # Don't use 'run' for this command as we need the state to persist
    ollama version >/dev/null
    [[ "$(mock::ollama::get::call_count "version")" == "1" ]]
}

# ----------------------------
# Edge Cases and Error Handling
# ----------------------------

@test "handles empty and malformed inputs gracefully" {
    # Test commands with empty arguments
    run ollama pull ""
    assert_failure
    assert_output --partial "missing model name"
    
    run ollama rm ""
    assert_failure
    assert_output --partial "missing model name"
    
    run ollama show ""
    assert_failure
    assert_output --partial "missing model name"
    
    run ollama stop ""
    assert_failure
    assert_output --partial "missing model name"
    
    run ollama cp "" "dest"
    assert_failure
    assert_output --partial "missing source or destination"
    
    run ollama cp "source" ""
    assert_failure
    assert_output --partial "missing source or destination"
}

@test "model name parsing works with various formats" {
    # Test various model name formats
    local test_models=(
        "simple:1b"
        "namespace/model:tag" 
        "registry.com/user/model:version"
        "model-with-dashes:7b"
        "model_with_underscores:13b"
        "model.with.dots:latest"
        "model123:v1.2.3"
    )
    
    for model in "${test_models[@]}"; do
        run ollama pull "$model"
        assert_success
        assert_output --partial "success"
        
        # Verify model was added
        run mock::ollama::assert::model_installed "$model"
        assert_success
    done
}

@test "concurrent operations work correctly" {
    # Simulate concurrent operations by running multiple commands
    # that modify state and verify consistency
    
    # Start with known state
    mock::ollama::reset
    initial_count=$(mock::ollama::get::installed_models | wc -l)
    
    # Add multiple models "concurrently" (sequentially but rapidly)
    for i in {1..5}; do
        mock::ollama::add_model "concurrent-test-${i}:1b"
    done
    
    # Verify all models were added
    final_count=$(mock::ollama::get::installed_models | wc -l)
    expected_count=$((initial_count + 5))
    [[ "$final_count" == "$expected_count" ]]
    
    # Verify each model exists
    for i in {1..5}; do
        run mock::ollama::assert::model_installed "concurrent-test-${i}:1b"
        assert_success
    done
}

@test "large model collections work correctly" {
    # Test with a larger number of models to verify performance
    # and that arrays/data structures handle many entries
    
    # Add many models
    for i in {1..50}; do
        mock::ollama::add_model "loadtest-${i}:1b" "2024-01-15T10:$((30 + i)):00Z"
    done
    
    # Verify all models are listed
    run ollama list
    assert_success
    
    # Count the models in the output (should include defaults + 50 new ones)
    model_lines=$(echo "$output" | grep -c "loadtest-")
    [[ "$model_lines" == "50" ]]
    
    # Verify individual models can be accessed
    run ollama show "loadtest-25:1b"
    assert_success
    assert_output --partial "Model"
    
    # Clean up by resetting
    mock::ollama::reset
}

# Final verification that all mock systems are working
@test "mock verification system integration" {
    # Make various calls
    run ollama version
    run ollama list
    run ollama pull "integration-test:1b"
    
    # Verify tracking worked
    [[ "$(mock::ollama::get::call_count "version")" == "1" ]]
    [[ "$(mock::ollama::get::call_count "list")" == "1" ]]
    [[ "$(mock::ollama::get::call_count "pull")" == "1" ]]
}