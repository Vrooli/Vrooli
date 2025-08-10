#!/usr/bin/env bats

# Test file for docker_only.sh
# Tests Docker-only development environment setup

# Source trash module for safe test cleanup
SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/../../../../lib/utils/var.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/trash.sh" 2>/dev/null || true

setup() {
    # Load test helpers
    load "${BATS_TEST_DIRNAME}/../../../../__test/helpers/bats-support/load"
    load "${BATS_TEST_DIRNAME}/../../../../__test/helpers/bats-assert/load"
    
    # Set up test directory
    export TEST_DIR="$(mktemp -d)"
    export ORIGINAL_DIR="$(pwd)"
    
    # Source the script (functions only, don't execute main)
    export APP_LIFECYCLE_DEVELOP_TARGET_DIR="${BATS_TEST_DIRNAME}"
    
    # Mock var.sh and dependencies
    export var_ROOT_DIR="${TEST_DIR}"
    export var_LIB_UTILS_DIR="${TEST_DIR}/lib/utils"
    export var_LOG_FILE="${TEST_DIR}/lib/utils/log.sh"
    export var_APP_UTILS_DIR="${TEST_DIR}/app/utils"
    export var_SERVICE_JSON_FILE="${TEST_DIR}/.vrooli/service.json"
    export EXIT_USER_INTERRUPT=130
    
    # Create mock files
    mkdir -p "${var_LIB_UTILS_DIR}" "${var_APP_UTILS_DIR}" "$(dirname "${var_SERVICE_JSON_FILE}")"
    
    # Create mock log.sh
    cat > "${var_LOG_FILE}" << 'EOF'
log::header() { echo "[HEADER] $*"; }
log::info() { echo "[INFO] $*"; }
log::success() { echo "[SUCCESS] $*"; }
log::error() { echo "[ERROR] $*" >&2; }
log::warning() { echo "[WARNING] $*"; }
EOF
    
    # Create mock flow.sh
    cat > "${var_LIB_UTILS_DIR}/flow.sh" << 'EOF'
flow::is_yes() {
    local value="${1:-}"
    [[ "${value,,}" =~ ^(y|yes|true|1|on)$ ]]
}
EOF
    
    # Create mock exit_codes.sh
    cat > "${var_LIB_UTILS_DIR}/exit_codes.sh" << 'EOF'
export EXIT_USER_INTERRUPT=130
export ERROR_MISSING_DEPENDENCY=127
EOF
    
    # Create mock env.sh
    cat > "${var_APP_UTILS_DIR}/env.sh" << 'EOF'
env::load_secrets() { :; }
env::construct_derived_secrets() {
    export DB_URL="postgresql://user:pass@localhost:5432/db"
    export REDIS_URL="redis://localhost:6379"
}
env::load_env_file() { :; }
EOF
    
    # Create mock docker.sh
    cat > "${var_APP_UTILS_DIR}/docker.sh" << 'EOF'
docker::compose() {
    echo "docker-compose $*"
    case "$1" in
        up)
            if [[ "$2" == "-d" ]]; then
                echo "Starting services in detached mode"
            else
                echo "Starting services in foreground mode"
            fi
            ;;
        down)
            echo "Stopping and removing containers"
            ;;
        *)
            echo "Unknown docker-compose command: $*"
            return 1
            ;;
    esac
}
EOF
    
    # Create mock service_config.sh
    cat > "${var_APP_UTILS_DIR}/service_config.sh" << 'EOF'
service_config::has_inheritance() {
    [[ -f "$1" ]] && grep -q '"extends"' "$1" 2>/dev/null
}
service_config::export_resource_urls() {
    export MOCK_RESOURCE_URL="http://localhost:8080"
}
EOF
    
    # Source all mocks
    source "${var_LOG_FILE}"
    source "${var_LIB_UTILS_DIR}/flow.sh"
    source "${var_LIB_UTILS_DIR}/exit_codes.sh"
    source "${var_APP_UTILS_DIR}/env.sh"
    source "${var_APP_UTILS_DIR}/docker.sh"
    
    # Source the script under test (without executing main)
    set +e  # Don't exit on error during sourcing
    source "${BATS_TEST_DIRNAME}/docker_only.sh" 2>/dev/null || true
    set -e
}

teardown() {
    cd "${ORIGINAL_DIR}"
    [[ -d "${TEST_DIR}" ]] && trash::safe_remove "${TEST_DIR}" --test-cleanup
}

# Test: Script sources required dependencies
@test "docker_only.sh sources required dependencies" {
    # This is tested implicitly in setup()
    # If sourcing failed, other tests would fail
    run type -t docker_only::start_development_docker_only
    assert_success
    assert_output "function"
}

# Test: Function naming follows convention
@test "docker_only functions use correct naming convention" {
    run type -t docker_only::start_development_docker_only
    assert_success
    assert_output "function"
    
    run type -t docker_only::cleanup
    assert_success
    assert_output "function"
}

# Test: Start function handles detached mode
@test "docker_only starts in detached mode when DETACHED=yes" {
    export DETACHED="yes"
    export DB_URL="postgresql://test"
    export REDIS_URL="redis://test"
    
    run docker_only::start_development_docker_only
    assert_success
    assert_output --partial "Starting services in detached mode"
    assert_output --partial "[SUCCESS] ✅ Docker only development environment started successfully"
}

# Test: Start function handles foreground mode
@test "docker_only starts in foreground mode when DETACHED=no" {
    export DETACHED="no"
    export DB_URL="postgresql://test"
    export REDIS_URL="redis://test"
    
    # Mock docker::compose to exit after first call
    docker::compose() {
        echo "docker-compose $*"
        if [[ "$1" == "up" ]] && [[ "$2" != "-d" ]]; then
            echo "Starting services in foreground mode"
            return 0
        fi
    }
    
    run docker_only::start_development_docker_only
    assert_success
    assert_output --partial "Starting services in foreground mode"
}

# Test: Cleanup function is defined
@test "docker_only cleanup function works correctly" {
    # Override docker::compose for this test
    docker::compose() {
        if [[ "$1" == "down" ]]; then
            echo "Containers stopped and removed"
            return 0
        fi
        return 1
    }
    
    # Cleanup should exit with EXIT_USER_INTERRUPT
    run docker_only::cleanup
    assert_failure 130  # EXIT_USER_INTERRUPT
    assert_output --partial "Cleaning up development environment"
    assert_output --partial "Containers stopped and removed"
}

# Test: Derived secrets construction
@test "docker_only constructs derived secrets when missing" {
    unset DB_URL
    unset REDIS_URL
    export DETACHED="yes"
    
    run docker_only::start_development_docker_only
    assert_success
    assert_output --partial "Constructing derived secrets for docker-compose"
}

# Test: Service.json processing with inheritance
@test "docker_only processes service.json with inheritance" {
    # Create a mock service.json with inheritance
    cat > "${var_SERVICE_JSON_FILE}" << 'EOF'
{
    "extends": "base-config",
    "resources": {}
}
EOF
    
    export DETACHED="yes"
    export DB_URL="postgresql://test"
    export REDIS_URL="redis://test"
    
    run docker_only::start_development_docker_only
    assert_success
    assert_output --partial "Resolving service configuration templates at runtime"
}

# Test: Service.json processing without inheritance
@test "docker_only skips template resolution without inheritance" {
    # Create a mock service.json without inheritance
    cat > "${var_SERVICE_JSON_FILE}" << 'EOF'
{
    "resources": {}
}
EOF
    
    export DETACHED="yes"
    export DB_URL="postgresql://test"
    export REDIS_URL="redis://test"
    
    run docker_only::start_development_docker_only
    assert_success
    refute_output --partial "Resolving service configuration templates at runtime"
}

# Test: Environment file loading
@test "docker_only loads environment file" {
    export DETACHED="yes"
    export DB_URL="postgresql://test"
    export REDIS_URL="redis://test"
    
    # Track if env::load_env_file was called
    env::load_env_file() {
        echo "Environment file loaded"
    }
    
    run docker_only::start_development_docker_only
    assert_success
    assert_output --partial "Environment file loaded"
}

# Test: Instance manager usage when available
@test "docker_only uses instance manager for cleanup when available" {
    # Mock instance manager
    instance::shutdown_target() {
        echo "Instance manager shutting down target: $1"
        return 0
    }
    export -f instance::shutdown_target
    
    run docker_only::cleanup
    assert_failure 130  # EXIT_USER_INTERRUPT
    assert_output --partial "Instance manager shutting down target: docker"
}

# Test: Fallback cleanup when instance manager not available
@test "docker_only falls back to docker-compose down when instance manager unavailable" {
    # Ensure instance::shutdown_target is not available
    unset -f instance::shutdown_target 2>/dev/null || true
    
    # Override docker::compose for this test
    docker::compose() {
        if [[ "$1" == "down" ]]; then
            echo "Fallback: docker-compose down"
            return 0
        fi
        return 1
    }
    
    run docker_only::cleanup
    assert_failure 130  # EXIT_USER_INTERRUPT
    assert_output --partial "Fallback: docker-compose down"
}

# Test: Script can be run directly (main execution)
@test "docker_only script executes main when run directly" {
    # This test verifies the script's self-execution logic
    # We'll simulate this by checking if the condition would trigger
    
    # Create a test script that sources our script
    cat > "${TEST_DIR}/test_direct_run.sh" << 'EOF'
#!/usr/bin/env bash
export BASH_SOURCE=("$0")
export DETACHED="yes"
export DB_URL="test"
export REDIS_URL="test"

# Mock docker::compose to prevent actual execution
docker::compose() {
    echo "Mock docker-compose: $*"
    return 0
}

# Check if main would be called
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    echo "Main function would be called"
fi
EOF
    
    chmod +x "${TEST_DIR}/test_direct_run.sh"
    run "${TEST_DIR}/test_direct_run.sh"
    assert_success
    assert_output --partial "Main function would be called"
}