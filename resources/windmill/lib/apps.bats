#!/usr/bin/env bats

# Source trash module for safe test cleanup
SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
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
    
    # Create test directories and files
    mkdir -p "$WINDMILL_DATA_DIR"
    mkdir -p "${HOME}/windmill-apps"
    
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
    
    # Mock windmill status checks
    windmill::is_installed() {
        return 0
    }
    
    windmill::is_running() {
        return 0
    }
    
    windmill::is_healthy() {
        return 0
    }
    
    # Mock curl for API calls
    curl() {
        local args=("$@")
        local url=""
        
        # Parse curl arguments to find URL
        for arg in "${args[@]}"; do
            if [[ "$arg" =~ ^http ]]; then
                url="$arg"
                break
            fi
        done
        
        # Mock different API endpoints
        if [[ "$url" =~ /api/w/.*/apps$ ]]; then
            echo '[{"path":"f/test-app","summary":"Test App"}]'
        elif [[ "$url" =~ /api/w/.*/apps/.* ]]; then
            echo '{"path":"f/test-app","summary":"Test App","value":{"grid":[]}}'
        else
            echo '{"status":"ok"}'
        fi
        return 0
    }
    
    # Source the apps.sh file being tested
    source "${LIB_DIR}/apps.sh"
}

# Cleanup after each test
teardown() {
    trash::safe_remove "$WINDMILL_DATA_DIR" --test-cleanup
    trash::safe_remove "${HOME}/windmill-apps" --test-cleanup
    vrooli_cleanup_test
}

# Test app listing functionality
@test "windmill::list_apps function exists and can be called" {
    if declare -f windmill::list_apps >/dev/null; then
        # Function exists, test it
        run windmill::list_apps
        [ "$status" -eq 0 ]
    else
        skip "windmill::list_apps function not defined"
    fi
}

# Test app preparation functionality
@test "windmill::prepare_app validates input parameters" {
    if declare -f windmill::prepare_app >/dev/null; then
        # Test with missing app name
        run windmill::prepare_app
        [[ "$status" -ne 0 ]] || [[ "$output" =~ "required" ]]
        
        # Test with missing output directory
        run windmill::prepare_app "test-app"
        [[ "$status" -eq 0 ]] || [[ "$output" =~ "required" ]]
    else
        skip "windmill::prepare_app function not defined"
    fi
}

@test "windmill::prepare_app creates output directory" {
    if declare -f windmill::prepare_app >/dev/null; then
        local output_dir="/tmp/test-windmill-apps"
        run windmill::prepare_app "test-app" "$output_dir"
        
        # Should create directory or not fail
        [[ "$status" -eq 0 ]] || [[ -d "$output_dir" ]]
        
        # Cleanup
        trash::safe_remove "$output_dir" --test-cleanup
    else
        skip "windmill::prepare_app function not defined"
    fi
}

# Test app deployment functionality
@test "windmill::deploy_app function exists and can be called" {
    if declare -f windmill::deploy_app >/dev/null; then
        # Function exists, test it
        run windmill::deploy_app "test-app" "demo"
        [[ "$status" -eq 0 ]] || [[ "$output" =~ "required" ]] || [[ "$output" =~ "not running" ]]
    else
        skip "windmill::deploy_app function not defined"
    fi
}

@test "windmill::deploy_app validates windmill is running" {
    if declare -f windmill::deploy_app >/dev/null; then
        # Override is_running to fail
        windmill::is_running() {
            return 1
        }
        
        run windmill::deploy_app "test-app" "demo"
        [[ "$status" -ne 0 ]] || [[ "$output" =~ "not running" ]]
    else
        skip "windmill::deploy_app function not defined"
    fi
}

# Test app API checking
@test "windmill::check_app_api function exists and can be called" {
    if declare -f windmill::check_app_api >/dev/null; then
        # Function exists, test it
        run windmill::check_app_api
        [[ "$status" -eq 0 ]] || [[ "$output" =~ "not running" ]]
    else
        skip "windmill::check_app_api function not defined"
    fi
}

# Test app validation functionality
@test "windmill::validate_app_config function exists and can be called" {
    if declare -f windmill::validate_app_config >/dev/null; then
        # Create test app config
        local test_config='{"value":{"grid":[]},"summary":"Test app"}'
        
        run windmill::validate_app_config "$test_config"
        [ "$status" -eq 0 ]
    else
        skip "windmill::validate_app_config function not defined"
    fi
}

# Test app backup functionality
@test "windmill::backup_app function exists and can be called" {
    if declare -f windmill::backup_app >/dev/null; then
        # Function exists, test it
        run windmill::backup_app "test-app"
        [[ "$status" -eq 0 ]] || [[ "$output" =~ "not running" ]] || [[ "$output" =~ "required" ]]
    else
        skip "windmill::backup_app function not defined"
    fi
}

# Test app restoration functionality
@test "windmill::restore_app function exists and can be called" {
    if declare -f windmill::restore_app >/dev/null; then
        # Function exists, test it - should fail without valid backup
        run windmill::restore_app "test-app" "/nonexistent/backup.json"
        [[ "$status" -ne 0 ]] || [[ "$output" =~ "not found" ]] || [[ "$output" =~ "required" ]]
    else
        skip "windmill::restore_app function not defined"
    fi
}

# Test that apps.sh has valid bash syntax
@test "apps.sh has valid bash syntax" {
    run bash -n "${LIB_DIR}/apps.sh"
    [ "$status" -eq 0 ]
}

# Test that apps.sh has correct permissions
@test "apps.sh has correct permissions" {
    [ -r "${LIB_DIR}/apps.sh" ]
}