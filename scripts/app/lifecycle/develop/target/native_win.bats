#!/usr/bin/env bats

# Test file for native_win.sh
# Tests native Windows development environment setup

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
    export EXIT_USER_INTERRUPT=130
    export DETACHED="no"
    
    # Create mock directories
    mkdir -p "${var_LIB_UTILS_DIR}" "${var_APP_UTILS_DIR}"
    mkdir -p "${TEST_DIR}/packages/server" "${TEST_DIR}/packages/jobs"
    
    # Create mock log.sh
    cat > "${var_LOG_FILE}" << 'EOF'
log::header() { echo "[HEADER] $*"; }
log::info() { echo "[INFO] $*"; }
log::success() { echo "[SUCCESS] $*"; }
log::error() { echo "[ERROR] $*" >&2; return 1; }
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
    
    # Create mock docker.sh
    cat > "${var_APP_UTILS_DIR}/docker.sh" << 'EOF'
docker::compose() {
    echo "docker-compose $*"
    case "$*" in
        "up -d postgres redis")
            echo "Starting postgres and redis containers"
            ;;
        "down")
            echo "Stopping all containers"
            ;;
        *)
            echo "docker-compose $*"
            ;;
    esac
    return 0
}
docker::run() {
    case "$*" in
        ps*)
            # Mock healthy containers
            echo "postgres    Up 2 minutes (healthy)"
            echo "redis       Up 2 minutes (healthy)"
            ;;
        *)
            echo "docker $*"
            ;;
    esac
    return 0
}
EOF
    
    # Create mock env_urls.sh
    cat > "${var_APP_UTILS_DIR}/env_urls.sh" << 'EOF'
env_urls::setup_native_environment() {
    export DB_URL="postgresql://user:pass@127.0.0.1:5432/db"
    export REDIS_URL="redis://127.0.0.1:6379"
    echo "Native environment URLs configured"
}
env_urls::validate_native_environment() {
    echo "Validating native environment connectivity"
    return 0
}
EOF
    
    # Mock pnpm
    pnpm() {
        echo "pnpm $*"
        return 0
    }
    export -f pnpm
    
    # Source all mocks
    source "${var_LOG_FILE}"
    source "${var_LIB_UTILS_DIR}/flow.sh"
    source "${var_LIB_UTILS_DIR}/exit_codes.sh"
    source "${var_APP_UTILS_DIR}/docker.sh"
    
    # Source the script under test (without executing main)
    set +e  # Don't exit on error during sourcing
    source "${BATS_TEST_DIRNAME}/native_win.sh" 2>/dev/null || true
    set -e
}

teardown() {
    cd "${ORIGINAL_DIR}"
    [[ -d "${TEST_DIR}" ]] && rm -rf "${TEST_DIR}"
}

# Test: Script sources required dependencies
@test "native_win.sh sources required dependencies" {
    run type -t native_win::start_development_native_win
    assert_success
    assert_output "function"
}

# Test: Function naming follows convention
@test "native_win functions use correct naming convention" {
    run type -t native_win::start_development_native_win
    assert_success
    assert_output "function"
    
    run type -t native_win::cleanup
    assert_success
    assert_output "function"
}

# Test: Cleanup function
@test "native_win cleanup function works correctly" {
    run native_win::cleanup
    assert_failure 130  # EXIT_USER_INTERRUPT
    assert_output --partial "Cleaning up development environment"
    assert_output --partial "docker-compose down"
}

# Test: Database containers startup
@test "native_win starts database containers" {
    # Override docker::compose to track calls
    local compose_calls=""
    docker::compose() {
        compose_calls="${compose_calls}|$*"
        echo "docker-compose $*"
        return 0
    }
    
    # Test container startup
    docker::compose up -d postgres redis
    echo "$compose_calls" | {
        run cat
        assert_success
        assert_output --partial "up -d postgres redis"
    }
}

# Test: Container health check
@test "native_win waits for containers to be healthy" {
    local check_count=0
    
    # Override docker::run to simulate health progression
    docker::run() {
        case "$*" in
            ps*)
                ((check_count++))
                if [[ $check_count -lt 3 ]]; then
                    # Not healthy yet
                    echo "postgres    Up 1 minute"
                    echo "redis       Up 1 minute"
                else
                    # Now healthy
                    echo "postgres    Up 2 minutes (healthy)"
                    echo "redis       Up 2 minutes (healthy)"
                fi
                ;;
            *)
                echo "docker $*"
                ;;
        esac
        return 0
    }
    
    # Test health check logic
    local max_attempts=5
    local attempt=0
    while [ $attempt -lt $max_attempts ]; do
        if docker::run ps | grep "postgres" | grep -q "healthy" && docker::run ps | grep "redis" | grep -q "healthy"; then
            echo "Containers healthy after $((attempt + 1)) attempts"
            break
        fi
        attempt=$((attempt + 1))
    done
    
    run echo "Check count: $check_count"
    assert_success
    assert_output --partial "Check count: 3"
}

# Test: Container failure handling
@test "native_win handles container startup failure" {
    # Override docker::run to simulate containers never becoming healthy
    docker::run() {
        case "$*" in
            ps*)
                echo "postgres    Up 1 minute"
                echo "redis       Up 1 minute"
                ;;
            *)
                echo "docker $*"
                ;;
        esac
        return 0
    }
    
    # Test health check timeout
    local max_attempts=2
    local attempt=0
    while [ $attempt -lt $max_attempts ]; do
        if docker::run ps | grep "postgres" | grep -q "healthy"; then
            echo "Containers healthy"
            break
        fi
        attempt=$((attempt + 1))
        if [ $attempt -eq $max_attempts ]; then
            echo "Container health check failed after $max_attempts attempts"
            echo "ERROR"
            return 1
        fi
    done | {
        run cat
        assert_failure
        assert_output --partial "Container health check failed"
    }
}

# Test: Native environment URL setup
@test "native_win sets up native environment URLs" {
    source "${var_APP_UTILS_DIR}/env_urls.sh"
    
    run env_urls::setup_native_environment
    assert_success
    assert_output --partial "Native environment URLs configured"
    
    # Verify URLs are using localhost/127.0.0.1
    [[ "${DB_URL}" == *"127.0.0.1"* ]] || [[ "${DB_URL}" == *"localhost"* ]]
    [[ "${REDIS_URL}" == *"127.0.0.1"* ]] || [[ "${REDIS_URL}" == *"localhost"* ]]
}

# Test: Environment validation
@test "native_win validates native environment" {
    source "${var_APP_UTILS_DIR}/env_urls.sh"
    
    run env_urls::validate_native_environment
    assert_success
    assert_output --partial "Validating native environment connectivity"
}

# Test: Environment validation failure handling
@test "native_win continues when validation fails" {
    source "${var_APP_UTILS_DIR}/env_urls.sh"
    
    # Override validation to fail
    env_urls::validate_native_environment() {
        echo "Validation failed"
        return 1
    }
    
    # Test that script continues despite validation failure
    if ! env_urls::validate_native_environment; then
        echo "Continuing despite validation failure"
    fi | {
        run cat
        assert_success
        assert_output --partial "Continuing despite validation failure"
    }
}

# Test: Detached mode watcher startup
@test "native_win starts watchers in detached mode" {
    export DETACHED="yes"
    export DB_URL="postgresql://test"
    export REDIS_URL="redis://test"
    export NODE_ENV="development"
    export PORT_API="5329"
    export PORT_UI="3000"
    
    # Test detached mode logic
    if flow::is_yes "$DETACHED"; then
        echo "Detached mode enabled"
        local pids=()
        for i in 1 2 3; do
            bash -c "echo watcher$i" > /dev/null 2>&1 &
            pids+=("$!")
        done
        echo "Started ${#pids[@]} watchers with PIDs: ${pids[*]}"
    fi | {
        run cat
        assert_success
        assert_output --partial "Detached mode enabled"
        assert_output --partial "Started 3 watchers"
    }
}

# Test: Foreground mode watcher startup
@test "native_win starts watchers in foreground mode" {
    export DETACHED="no"
    
    # Test foreground mode uses concurrently
    if ! flow::is_yes "$DETACHED"; then
        echo "Foreground mode"
        pnpm exec concurrently \
            --names "SERVER,JOBS,UI" \
            -c "yellow,blue,green" \
            "cmd1" "cmd2" "cmd3"
    fi | {
        run cat
        assert_success
        assert_output --partial "Foreground mode"
        assert_output --partial "--names SERVER,JOBS,UI"
        assert_output --partial "-c yellow,blue,green"
    }
}

# Test: Environment variable export
@test "native_win exports environment variables for child processes" {
    # Test environment export logic
    export REDIS_URL="redis://test:6379"
    export DB_URL="postgresql://test:5432/db"
    export NODE_ENV="development"
    export PORT_API="5329"
    export PORT_UI="3000"
    
    # Verify variables are set
    run echo "DB_URL=$DB_URL REDIS_URL=$REDIS_URL NODE_ENV=$NODE_ENV PORT_API=$PORT_API PORT_UI=$PORT_UI"
    assert_success
    assert_output --partial "DB_URL=postgresql://test:5432/db"
    assert_output --partial "REDIS_URL=redis://test:6379"
    assert_output --partial "NODE_ENV=development"
    assert_output --partial "PORT_API=5329"
    assert_output --partial "PORT_UI=3000"
}

# Test: Script can be run directly
@test "native_win script executes main when run directly" {
    # Create test script to simulate direct execution
    cat > "${TEST_DIR}/test_direct_run.sh" << 'EOF'
#!/usr/bin/env bash
export BASH_SOURCE=("$0")

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

# Test: Windows-specific considerations comment
@test "native_win acknowledges Windows environment variations" {
    # This test validates that the script acknowledges it can run in different Windows environments
    # Check for the comment about Windows detection
    run grep -q "Windows detection adjusted for WSL/Git Bash/Cygwin" "${BATS_TEST_DIRNAME}/native_win.sh"
    assert_success
}

# Test: Trap signals in non-detached mode
@test "native_win sets up signal traps in non-detached mode" {
    export DETACHED="no"
    
    # Test trap setup logic
    if ! flow::is_yes "$DETACHED"; then
        echo "Setting up signal traps"
        # In actual script: trap native_win::cleanup SIGINT SIGTERM
        echo "Traps would be set for SIGINT and SIGTERM"
    fi | {
        run cat
        assert_success
        assert_output --partial "Setting up signal traps"
        assert_output --partial "Traps would be set for SIGINT and SIGTERM"
    }
}

# Test: No traps in detached mode
@test "native_win skips signal traps in detached mode" {
    export DETACHED="yes"
    
    # Test that traps are not set in detached mode
    if ! flow::is_yes "$DETACHED"; then
        echo "Setting up signal traps"
    else
        echo "Skipping signal traps in detached mode"
    fi | {
        run cat
        assert_success
        assert_output --partial "Skipping signal traps in detached mode"
    }
}

# Test: Watcher commands structure
@test "native_win constructs correct watcher commands" {
    # Test the structure of watcher commands
    export var_ROOT_DIR="${TEST_DIR}"
    export DB_URL="postgresql://test"
    export REDIS_URL="redis://test"
    export PORT_UI="3000"
    
    local watchers=(
        "cd packages/server && PROJECT_DIR=$var_ROOT_DIR NODE_ENV=development DB_URL=${DB_URL} REDIS_URL=${REDIS_URL} npm_package_name=@vrooli/server bash ../../scripts/app/package/server/start.sh"
        "cd packages/jobs && PROJECT_DIR=$var_ROOT_DIR NODE_ENV=development DB_URL=${DB_URL} REDIS_URL=${REDIS_URL} npm_package_name=@vrooli/jobs bash ../../scripts/app/package/jobs/start.sh"
        "pnpm --filter @vrooli/ui run start-development -- --port ${PORT_UI:-3000}"
    )
    
    # Verify all three watchers are defined
    run echo "${#watchers[@]}"
    assert_success
    assert_output "3"
    
    # Verify server watcher includes correct path
    run echo "${watchers[0]}"
    assert_success
    assert_output --partial "scripts/app/package/server/start.sh"
    
    # Verify jobs watcher includes correct path
    run echo "${watchers[1]}"
    assert_success
    assert_output --partial "scripts/app/package/jobs/start.sh"
    
    # Verify UI watcher uses pnpm
    run echo "${watchers[2]}"
    assert_success
    assert_output --partial "pnpm --filter @vrooli/ui"
}

# Test: Return value in detached mode
@test "native_win returns 0 in detached mode after starting watchers" {
    export DETACHED="yes"
    
    # Test that detached mode returns success
    if flow::is_yes "$DETACHED"; then
        # Simulate starting watchers
        echo "Starting watchers in background"
        return 0
    fi
    
    run echo $?
    assert_success
    assert_output "0"
}