#!/usr/bin/env bats
# Lifecycle Engine Tests
# Tests for the main lifecycle engine orchestrator

bats_require_minimum_version 1.5.0

# Load test infrastructure (single entry point)
source "${BATS_TEST_DIRNAME}/../../__test/fixtures/setup.bash"

setup() {
    vrooli_setup_unit_test  # Basic mocks and environment
    
    # Source the script under test
    # shellcheck disable=SC1091
    source "${BATS_TEST_DIRNAME}/engine.sh"
    
    # Create a minimal test service.json
    export TEST_SERVICE_JSON="${BATS_TEST_TMPDIR}/service.json"
    cat > "$TEST_SERVICE_JSON" <<EOF
{
  "version": "1.0.0",
  "lifecycle": {
    "setup": {
      "description": "Test setup phase",
      "steps": [
        {"name": "test_step", "command": "echo test"}
      ]
    },
    "build": {
      "description": "Test build phase", 
      "steps": [
        {"name": "build_step", "command": "echo build"}
      ]
    }
  },
  "defaults": {
    "timeout": 300,
    "shell": "/bin/bash"
  }
}
EOF
}

teardown() {
    vrooli_cleanup_test     # Clean up resources
}

@test "engine::init - initializes engine successfully" {
    run engine::init
    assert_success
}

@test "engine::version - displays version information" {
    run engine::version
    assert_success
    assert_output_contains "Vrooli Lifecycle Engine"
    assert_output_contains "v2.0.0"
}

@test "engine::load_config - loads valid service.json" {
    SERVICE_JSON_PATH="$TEST_SERVICE_JSON"
    
    run engine::load_config
    assert_success
    
    # Verify configuration was loaded
    [[ -n "$SERVICE_JSON" ]]
    [[ -n "$LIFECYCLE_CONFIG" ]]
}

@test "engine::load_config - fails with missing service.json" {
    SERVICE_JSON_PATH="/nonexistent/service.json"
    
    run engine::load_config
    assert_failure
}

@test "engine::load_config - searches standard locations when no path provided" {
    # Copy service.json to a standard location
    mkdir -p "${BATS_TEST_TMPDIR}/.vrooli"
    cp "$TEST_SERVICE_JSON" "${BATS_TEST_TMPDIR}/.vrooli/service.json"
    
    # Change to test directory so it can be found
    cd "$BATS_TEST_TMPDIR"
    
    # Clear explicit path
    unset SERVICE_JSON_PATH
    
    run engine::load_config
    assert_success
}

@test "engine::list_phases - lists available phases" {
    SERVICE_JSON_PATH="$TEST_SERVICE_JSON"
    engine::load_config
    
    run engine::list_phases
    assert_success
    assert_output_contains "setup"
    assert_output_contains "build"
}

@test "engine::execute - executes valid phase" {
    SERVICE_JSON_PATH="$TEST_SERVICE_JSON"
    LIFECYCLE_PHASE="setup"
    TARGET="native-linux"
    
    # Load config first
    engine::load_config
    
    # Mock phase::execute to avoid actually executing
    phase::execute() {
        echo "Mock execution of phase: $1, target: $2"
        return 0
    }
    export -f phase::execute
    
    run engine::execute
    assert_success
    assert_output_contains "Mock execution of phase: setup"
}

@test "engine::execute - fails with invalid phase" {
    SERVICE_JSON_PATH="$TEST_SERVICE_JSON"
    LIFECYCLE_PHASE="nonexistent"
    
    # Load config first
    engine::load_config
    
    run engine::execute
    assert_failure
    assert_output_contains "Phase not found: nonexistent"
}

@test "engine::execute - lists available phases on error" {
    SERVICE_JSON_PATH="$TEST_SERVICE_JSON"
    LIFECYCLE_PHASE="invalid"
    
    # Load config first  
    engine::load_config
    
    run engine::execute
    assert_failure
    assert_output_contains "Available phases:"
    assert_output_contains "setup"
    assert_output_contains "build"
}

@test "engine::cleanup - cleans up resources" {
    # Mock the cleanup functions to verify they're called
    parallel::kill_all() {
        echo "parallel::kill_all called"
    }
    export -f parallel::kill_all
    
    output::cleanup() {
        echo "output::cleanup called" 
    }
    export -f output::cleanup
    
    targets::clear_cache() {
        echo "targets::clear_cache called"
    }
    export -f targets::clear_cache
    
    run engine::cleanup
    assert_success
    assert_output_contains "parallel::kill_all called"
    assert_output_contains "output::cleanup called"
    assert_output_contains "targets::clear_cache called"
}

# Integration tests for special command handling

@test "engine main function - handles version command" {
    # Mock engine::version to avoid output to stderr
    engine::version() {
        echo "Mocked version output"
    }
    export -f engine::version
    
    # Test --version flag
    run main --version
    assert_success
    assert_output_contains "Mocked version output"
    
    # Test version command
    run main version
    assert_success
    assert_output_contains "Mocked version output"
}

@test "engine main function - handles list command" {
    SERVICE_JSON_PATH="$TEST_SERVICE_JSON"
    
    # Mock engine::list_phases
    engine::list_phases() {
        echo "Available phases: setup, build"
    }
    export -f engine::list_phases
    
    run main --list
    assert_success
    assert_output_contains "Available phases: setup, build"
    
    run main list  
    assert_success
    assert_output_contains "Available phases: setup, build"
}

@test "engine main function - validates arguments before proceeding" {
    # Mock parser functions to test validation flow
    parser::parse_args() {
        return 1  # Simulate parse failure
    }
    export -f parser::parse_args
    
    run main setup
    assert_failure
}

@test "engine main function - loads configuration before execution" {
    LIFECYCLE_PHASE="setup"
    
    # Mock functions to trace execution flow
    parser::parse_args() { return 0; }
    parser::validate() { return 0; }
    parser::export() { return 0; }
    config::export() { return 0; }
    engine::execute() { echo "execute called"; return 0; }
    
    export -f parser::parse_args parser::validate parser::export 
    export -f config::export engine::execute
    
    # Mock engine::load_config to simulate failure
    engine::load_config() {
        echo "Failed to load config"
        return 1
    }
    export -f engine::load_config
    
    run main setup
    assert_failure
    assert_output_contains "Failed to load config"
}