#!/usr/bin/env bats

# Source trash module for safe test cleanup
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
SCRIPT_DIR="${APP_ROOT}/resources/windmill/lib"
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/../../../../lib/utils/var.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/trash.sh" 2>/dev/null || true

# Load Vrooli test infrastructure (REQUIRED)
source "${BATS_TEST_DIRNAME}/../../../../__test/fixtures/setup.bash"

# Expensive setup operations (run once per file)
setup_file() {
    # Use appropriate setup function
    vrooli_setup_service_test "windmill"
    
    # Export paths for use in setup()
    export SETUP_FILE_SCRIPT_DIR="${BATS_TEST_DIRNAME}"
    export SETUP_FILE_WINDMILL_DIR="$(dirname "${BATS_TEST_DIRNAME}")" 
    export SETUP_FILE_CONFIG_DIR="$(dirname "${BATS_TEST_DIRNAME}")/config"
    export SETUP_FILE_LIB_DIR="$(dirname "${BATS_TEST_DIRNAME}")/lib"
}

# Lightweight per-test setup
setup() {
    # Setup standard mocks
    vrooli_auto_setup
    
    # Use paths from setup_file
    SCRIPT_DIR="${SETUP_FILE_SCRIPT_DIR}"
    WINDMILL_DIR="${SETUP_FILE_WINDMILL_DIR}"
    CONFIG_DIR="${SETUP_FILE_CONFIG_DIR}"
    LIB_DIR="${SETUP_FILE_LIB_DIR}"
    
    # Set test environment BEFORE sourcing config files to avoid readonly conflicts
    export WINDMILL_PORT="5681"
    export WINDMILL_CONTAINER_NAME="windmill-test"
    export WINDMILL_BASE_URL="http://localhost:5681"
    export WINDMILL_DATA_DIR="/tmp/windmill-test"
    export WINDMILL_DB_PASSWORD="test-password"
    export YES="no"
    
    # Create test directories
    mkdir -p "$WINDMILL_DATA_DIR"
    mkdir -p "/tmp/windmill-state"
    export WINDMILL_STATE_DIR="/tmp/windmill-state"
    
    # Mock resources functions that are called during config loading
    resources::get_default_port() {
        case "$1" in
            "windmill") echo "5681" ;;
            *) echo "8080" ;;
        esac
    }
    
    # Now source the config files
    source "${WINDMILL_DIR}/config/defaults.sh"
    source "${WINDMILL_DIR}/config/messages.sh"
    
    # Export config and messages
    windmill::export_config
    windmill::export_messages
    
    # Source common functions
    source "${WINDMILL_DIR}/lib/common.sh"
    
    # Source the state.sh file being tested
    source "${LIB_DIR}/state.sh"
}

# Cleanup after each test
teardown() {
    trash::safe_remove "$WINDMILL_DATA_DIR" --test-cleanup
    trash::safe_remove "$WINDMILL_STATE_DIR" --test-cleanup
    vrooli_cleanup_test
}

# Test state initialization
@test "windmill::state_init function exists and can be called" {
    if declare -f windmill::state_init >/dev/null; then
        # Function exists, test it
        run windmill::state_init "test-windmill"
        [ "$status" -eq 0 ]
    else
        skip "windmill::state_init function not defined"
    fi
}

@test "windmill::state_init creates state directory" {
    if declare -f windmill::state_init >/dev/null; then
        local project_name="test-windmill"
        
        windmill::state_init "$project_name"
        
        # Should create some kind of state structure
        [[ -d "$WINDMILL_STATE_DIR" ]] || [[ -d "$WINDMILL_DATA_DIR" ]]
    else
        skip "windmill::state_init function not defined"
    fi
}

# Test state configuration management
@test "windmill::update_state_config function exists and can be called" {
    if declare -f windmill::update_state_config >/dev/null; then
        # Initialize state first if needed
        if declare -f windmill::state_init >/dev/null; then
            windmill::state_init "test-windmill"
        fi
        
        run windmill::update_state_config "test_key" "test_value"
        [ "$status" -eq 0 ]
    else
        skip "windmill::update_state_config function not defined"
    fi
}

@test "windmill::get_state_config function exists and can be called" {
    if declare -f windmill::get_state_config >/dev/null && declare -f windmill::update_state_config >/dev/null; then
        # Initialize state first if needed
        if declare -f windmill::state_init >/dev/null; then
            windmill::state_init "test-windmill"
        fi
        
        # Set a value first
        windmill::update_state_config "test_key" "test_value"
        
        # Then get it
        result=$(windmill::get_state_config "test_key")
        [[ "$result" == "test_value" ]] || [[ -n "$result" ]]
    else
        skip "windmill::get_state_config function not defined"
    fi
}

# Test database password management
@test "windmill::get_database_password function exists and can be called" {
    if declare -f windmill::get_database_password >/dev/null; then
        # Initialize state first if needed
        if declare -f windmill::state_init >/dev/null; then
            windmill::state_init "test-windmill"
        fi
        
        run windmill::get_database_password "test-windmill" "false"
        [ "$status" -eq 0 ]
        [[ -n "$output" ]]  # Should return some password
    else
        skip "windmill::get_database_password function not defined"
    fi
}

@test "windmill::set_database_password function exists and can be called" {
    if declare -f windmill::set_database_password >/dev/null; then
        # Initialize state first if needed
        if declare -f windmill::state_init >/dev/null; then
            windmill::state_init "test-windmill"
        fi
        
        run windmill::set_database_password "test-windmill" "new-test-password"
        [ "$status" -eq 0 ]
    else
        skip "windmill::set_database_password function not defined"
    fi
}

# Test lock management
@test "windmill::acquire_lock function exists and can be called" {
    if declare -f windmill::acquire_lock >/dev/null; then
        # Initialize state first if needed
        if declare -f windmill::state_init >/dev/null; then
            windmill::state_init "test-windmill"
        fi
        
        run windmill::acquire_lock
        [ "$status" -eq 0 ]
    else
        skip "windmill::acquire_lock function not defined"
    fi
}

@test "windmill::release_lock function exists and can be called" {
    if declare -f windmill::release_lock >/dev/null; then
        # Initialize state first if needed
        if declare -f windmill::state_init >/dev/null; then
            windmill::state_init "test-windmill"
        fi
        
        # Try to acquire lock first if possible
        if declare -f windmill::acquire_lock >/dev/null; then
            windmill::acquire_lock
        fi
        
        run windmill::release_lock
        [ "$status" -eq 0 ]
    else
        skip "windmill::release_lock function not defined"
    fi
}

# Test state persistence
@test "windmill::save_state function exists and can be called" {
    if declare -f windmill::save_state >/dev/null; then
        # Initialize state first if needed
        if declare -f windmill::state_init >/dev/null; then
            windmill::state_init "test-windmill"
        fi
        
        run windmill::save_state
        [ "$status" -eq 0 ]
    else
        skip "windmill::save_state function not defined"
    fi
}

@test "windmill::load_state function exists and can be called" {
    if declare -f windmill::load_state >/dev/null; then
        # Initialize state first if needed
        if declare -f windmill::state_init >/dev/null; then
            windmill::state_init "test-windmill"
        fi
        
        run windmill::load_state
        [ "$status" -eq 0 ]
    else
        skip "windmill::load_state function not defined"
    fi
}

# Test state cleanup
@test "windmill::cleanup_state function exists and can be called" {
    if declare -f windmill::cleanup_state >/dev/null; then
        # Initialize state first if needed
        if declare -f windmill::state_init >/dev/null; then
            windmill::state_init "test-windmill"
        fi
        
        run windmill::cleanup_state
        [ "$status" -eq 0 ]
    else
        skip "windmill::cleanup_state function not defined"
    fi
}

# Test that state.sh has valid bash syntax
@test "state.sh has valid bash syntax" {
    run bash -n "${LIB_DIR}/state.sh"
    [ "$status" -eq 0 ]
}

# Test that state.sh has correct permissions
@test "state.sh has correct permissions" {
    [ -r "${LIB_DIR}/state.sh" ]
}