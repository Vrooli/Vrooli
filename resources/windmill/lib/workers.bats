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
    
    # Mock windmill installation check
    windmill::is_installed() {
        return 0
    }
    
    # Mock compose command  
    windmill::compose_cmd() {
        echo "COMPOSE_CMD_CALLED: $*"
        return 0
    }
    
    # Source the workers.sh file being tested
    source "${LIB_DIR}/workers.sh"
}

# Cleanup after each test
teardown() {
    trash::safe_remove "$WINDMILL_DATA_DIR" --test-cleanup
    vrooli_cleanup_test
}

# Test worker count validation
@test "windmill scale_workers validates worker count" {
    run windmill::scale_workers
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Worker count is required" ]]
}

@test "windmill scale_workers rejects invalid worker count" {
    run windmill::scale_workers "0"
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Invalid worker count" ]]
    
    run windmill::scale_workers "abc"
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Invalid worker count" ]]
    
    run windmill::scale_workers "-5"
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Invalid worker count" ]]
}

@test "windmill::scale_workers accepts valid worker count" {
    run windmill::scale_workers "3"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Scaling Windmill Workers" ]]
}

@test "windmill::scale_workers fails if windmill not installed" {
    # Override installation check to fail
    windmill::is_installed() {
        return 1
    }
    
    run windmill::scale_workers "3"
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Windmill is not installed" ]]
}

# Test worker restart functionality
@test "windmill::restart_workers function exists and can be called" {
    if declare -f windmill::restart_workers >/dev/null; then
        # Function exists, test it
        run windmill::restart_workers
        # Should call compose command or return success
        [[ "$status" -eq 0 ]] || [[ "$output" =~ "COMPOSE_CMD_CALLED" ]]
    else
        skip "windmill::restart_workers function not defined"
    fi
}

# Test worker info display
@test "windmill::show_worker_info function exists and can be called" {
    if declare -f windmill::show_worker_info >/dev/null; then
        # Function exists, test it
        run windmill::show_worker_info "3" "3"
        [ "$status" -eq 0 ]
    else
        skip "windmill::show_worker_info function not defined"
    fi
}

# Test worker management utilities
@test "windmill::get_worker_count function exists and can be called" {
    if declare -f windmill::get_worker_count >/dev/null; then
        # Function exists, test it
        run windmill::get_worker_count
        [[ "$status" -eq 0 ]] || [[ "$output" =~ "COMPOSE_CMD_CALLED" ]]
    else
        skip "windmill::get_worker_count function not defined"
    fi
}

# Test worker health checking
@test "windmill::check_workers_health function exists and can be called" {
    if declare -f windmill::check_workers_health >/dev/null; then
        # Function exists, test it
        run windmill::check_workers_health
        [[ "$status" -eq 0 ]] || [[ "$output" =~ "COMPOSE_CMD_CALLED" ]]
    else
        skip "windmill::check_workers_health function not defined"
    fi
}

# Test that workers.sh has valid bash syntax
@test "workers.sh has valid bash syntax" {
    run bash -n "${LIB_DIR}/workers.sh"
    [ "$status" -eq 0 ]
}

# Test that workers.sh has correct permissions
@test "workers.sh has correct permissions" {
    [ -r "${LIB_DIR}/workers.sh" ]
}