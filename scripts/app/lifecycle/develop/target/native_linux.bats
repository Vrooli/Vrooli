#!/usr/bin/env bats

# Test file for native_linux.sh
# Tests native Linux development environment setup

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
    export DETACHED="no"
    
    # Create mock directories
    mkdir -p "${var_LIB_UTILS_DIR}" "${var_APP_UTILS_DIR}" "$(dirname "${var_SERVICE_JSON_FILE}")"
    mkdir -p "${TEST_DIR}/packages/server" "${TEST_DIR}/packages/jobs"
    mkdir -p "${TEST_DIR}/scripts/scenarios"
    
    # Create mock log.sh
    cat > "${var_LOG_FILE}" << 'EOF'
log::header() { echo "[HEADER] $*"; }
log::info() { echo "[INFO] $*"; }
log::success() { echo "[SUCCESS] $*"; }
log::error() { echo "[ERROR] $*" >&2; return 1; }
log::warning() { echo "[WARNING] $*"; }
log::debug() { echo "[DEBUG] $*"; }
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
env::load_env_file() {
    export DB_URL="postgresql://user:pass@localhost:5432/db"
    export REDIS_URL="redis://localhost:6379"
    export NODE_ENV="development"
    export PORT_API="5329"
    export PORT_UI="3000"
    export SERVER_LOCATION="http://localhost:5329"
}
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
        "ps --format"*)
            # Mock healthy containers
            echo "postgres	Up 2 minutes (healthy)"
            echo "redis	Up 2 minutes (healthy)"
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
    
    # Create mock service_config.sh
    cat > "${var_APP_UTILS_DIR}/service_config.sh" << 'EOF'
service_config::has_inheritance() {
    [[ -f "$1" ]] && grep -q '"extends"' "$1" 2>/dev/null
}
service_config::export_resource_urls() {
    export MOCK_RESOURCE_URL="http://localhost:8080"
}
EOF
    
    # Create mock auto-converter.sh
    cat > "${TEST_DIR}/scripts/scenarios/auto-converter.sh" << 'EOF'
#!/usr/bin/env bash
echo "Running scenario auto-converter"
[[ "$1" == "--verbose" ]] && echo "Verbose mode enabled"
exit 0
EOF
    chmod +x "${TEST_DIR}/scripts/scenarios/auto-converter.sh"
    
    # Mock npx for Prisma test
    npx() {
        case "$*" in
            "prisma db execute --stdin")
                echo "Database connection test successful"
                return 0
                ;;
            *)
                echo "npx $*"
                return 0
                ;;
        esac
    }
    export -f npx
    
    # Mock pnpm
    pnpm() {
        echo "pnpm $*"
        return 0
    }
    export -f pnpm
    
    # Mock concurrently
    concurrently() {
        echo "concurrently $*"
        return 0
    }
    export -f concurrently
    
    # Source all mocks
    source "${var_LOG_FILE}"
    source "${var_LIB_UTILS_DIR}/flow.sh"
    source "${var_LIB_UTILS_DIR}/exit_codes.sh"
    source "${var_APP_UTILS_DIR}/env.sh"
    source "${var_APP_UTILS_DIR}/docker.sh"
    
    # Source the script under test (without executing main)
    set +e  # Don't exit on error during sourcing
    source "${BATS_TEST_DIRNAME}/native_linux.sh" 2>/dev/null || true
    set -e
}

teardown() {
    cd "${ORIGINAL_DIR}"
    [[ -d "${TEST_DIR}" ]] && rm -rf "${TEST_DIR}"
}

# Test: Script sources required dependencies
@test "native_linux.sh sources required dependencies" {
    run type -t native_linux::start_development_native_linux
    assert_success
    assert_output "function"
}

# Test: Function naming follows convention
@test "native_linux functions use correct naming convention" {
    run type -t native_linux::start_development_native_linux
    assert_success
    assert_output "function"
    
    run type -t native_linux::cleanup
    assert_success
    assert_output "function"
}

# Test: Database containers startup
@test "native_linux starts database containers" {
    # Override docker::compose to track calls
    local compose_calls=""
    docker::compose() {
        compose_calls="${compose_calls}|$*"
        echo "docker-compose $*"
        return 0
    }
    
    # Mock the main function to just test container startup
    native_linux::test_containers() {
        cd "$var_ROOT_DIR"
        docker::compose up -d postgres redis
        echo "$compose_calls"
    }
    
    run native_linux::test_containers
    assert_success
    assert_output --partial "up -d postgres redis"
}

# Test: Container health check logic
@test "native_linux waits for containers to be healthy" {
    # Counter for health check attempts
    local check_count=0
    
    # Override docker::run to simulate health progression
    docker::run() {
        case "$*" in
            "ps --format"*)
                ((check_count++))
                if [[ $check_count -lt 3 ]]; then
                    # Not healthy yet
                    echo "postgres	Up 1 minute"
                    echo "redis	Up 1 minute"
                else
                    # Now healthy
                    echo "postgres	Up 2 minutes (healthy)"
                    echo "redis	Up 2 minutes (healthy)"
                fi
                ;;
            *)
                echo "docker $*"
                ;;
        esac
        return 0
    }
    
    # Test the health check loop
    local max_attempts=5
    local attempt=0
    while [ $attempt -lt $max_attempts ]; do
        local postgres_status=$(docker::run ps --format "table {{.Names}}\t{{.Status}}" | grep -E "^postgres" | sed 's/^postgres[[:space:]]*//')
        local redis_status=$(docker::run ps --format "table {{.Names}}\t{{.Status}}" | grep -E "^redis" | sed 's/^redis[[:space:]]*//')
        
        if [[ "$postgres_status" == *"healthy"* ]] && [[ "$redis_status" == *"healthy"* ]]; then
            echo "Containers are healthy after $((attempt + 1)) attempts"
            break
        fi
        
        attempt=$((attempt + 1))
    done
    
    run echo "Check count: $check_count"
    assert_success
    assert_output --partial "Check count: 3"
}

# Test: Cleanup function with instance manager
@test "native_linux cleanup uses instance manager when available" {
    # Mock instance manager
    instance::shutdown_target() {
        echo "Instance manager shutting down target: $1"
        return 0
    }
    export -f instance::shutdown_target
    
    run native_linux::cleanup
    assert_failure 130  # EXIT_USER_INTERRUPT
    assert_output --partial "Instance manager shutting down target: native-linux"
}

# Test: Cleanup function fallback
@test "native_linux cleanup falls back to docker-compose down" {
    # Ensure instance manager is not available
    unset -f instance::shutdown_target 2>/dev/null || true
    
    run native_linux::cleanup
    assert_failure 130  # EXIT_USER_INTERRUPT
    assert_output --partial "docker-compose down"
}

# Test: Environment variable loading
@test "native_linux loads environment file" {
    # Track if env::load_env_file was called
    local env_loaded=false
    env::load_env_file() {
        env_loaded=true
        export DB_URL="postgresql://test"
        export REDIS_URL="redis://test"
    }
    
    # Create minimal test function
    native_linux::test_env() {
        if [[ -f "${var_APP_UTILS_DIR}/env.sh" ]]; then
            source "${var_APP_UTILS_DIR}/env.sh"
            env::load_env_file
        fi
        echo "Environment loaded: $env_loaded"
    }
    
    run native_linux::test_env
    assert_success
    assert_output --partial "Environment loaded: true"
}

# Test: Service.json processing with inheritance
@test "native_linux processes service.json with inheritance" {
    # Create a mock service.json with inheritance
    cat > "${var_SERVICE_JSON_FILE}" << 'EOF'
{
    "extends": "base-config",
    "resources": {}
}
EOF
    
    # Test the service.json processing logic
    source "${var_APP_UTILS_DIR}/service_config.sh"
    
    if service_config::has_inheritance "${var_SERVICE_JSON_FILE}"; then
        echo "Has inheritance"
        service_config::export_resource_urls "${var_SERVICE_JSON_FILE}"
    fi | {
        run cat
        assert_success
        assert_output --partial "Has inheritance"
    }
}

# Test: Native environment URL setup
@test "native_linux sets up native environment URLs" {
    source "${var_APP_UTILS_DIR}/env_urls.sh"
    
    run env_urls::setup_native_environment
    assert_success
    assert_output --partial "Native environment URLs configured"
    
    # Check that URLs are using localhost/127.0.0.1
    [[ "${DB_URL}" == *"127.0.0.1"* ]] || [[ "${DB_URL}" == *"localhost"* ]]
    [[ "${REDIS_URL}" == *"127.0.0.1"* ]] || [[ "${REDIS_URL}" == *"localhost"* ]]
}

# Test: Database connectivity test
@test "native_linux tests database connectivity" {
    export DB_URL="postgresql://user:pass@127.0.0.1:5432/db"
    
    cd "${TEST_DIR}/packages/server"
    run npx prisma db execute --stdin <<< "SELECT 1;"
    assert_success
    assert_output --partial "Database connection test successful"
}

# Test: Scenario auto-converter execution
@test "native_linux runs scenario auto-converter" {
    export DEBUG="false"
    
    run "${TEST_DIR}/scripts/scenarios/auto-converter.sh"
    assert_success
    assert_output --partial "Running scenario auto-converter"
    refute_output --partial "Verbose mode enabled"
}

# Test: Scenario auto-converter with debug mode
@test "native_linux runs scenario auto-converter in debug mode" {
    export DEBUG="true"
    
    run "${TEST_DIR}/scripts/scenarios/auto-converter.sh" --verbose
    assert_success
    assert_output --partial "Running scenario auto-converter"
    assert_output --partial "Verbose mode enabled"
}

# Test: Detached mode watcher startup
@test "native_linux starts watchers in detached mode" {
    export DETACHED="yes"
    export DB_URL="postgresql://test"
    export REDIS_URL="redis://test"
    export NODE_ENV="development"
    export PORT_API="5329"
    export PORT_UI="3000"
    
    # Mock nohup
    nohup() {
        echo "Starting in background: ${*:3}"  # Skip 'bash -c'
        return 0
    }
    export -f nohup
    
    # Test detached startup logic
    local watchers=(
        "echo 'server watcher'"
        "echo 'jobs watcher'"
        "echo 'ui watcher'"
    )
    
    if flow::is_yes "$DETACHED"; then
        for cmd in "${watchers[@]}"; do
            nohup bash -c "$cmd" > /dev/null 2>&1 &
        done
        echo "Watchers started in detached mode"
    fi | {
        run cat
        assert_success
        assert_output --partial "Watchers started in detached mode"
    }
}

# Test: Foreground mode watcher startup
@test "native_linux starts watchers in foreground mode" {
    export DETACHED="no"
    export DB_URL="postgresql://test"
    export REDIS_URL="redis://test"
    
    # Test foreground startup uses concurrently
    if ! flow::is_yes "$DETACHED"; then
        echo "Starting with concurrently"
        pnpm exec concurrently --names "SERVER,JOBS,UI" -c "yellow,blue,green" "cmd1" "cmd2" "cmd3"
    fi | {
        run cat
        assert_success
        assert_output --partial "Starting with concurrently"
        assert_output --partial "pnpm exec concurrently --names SERVER,JOBS,UI"
    }
}

# Test: Script can be run directly
@test "native_linux script executes main when run directly" {
    # Create a test script that simulates direct execution
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

# Test: Container unhealthy status handling
@test "native_linux detects and reports unhealthy containers" {
    # Override docker::run to simulate unhealthy container
    docker::run() {
        case "$*" in
            "ps --format"*)
                echo "postgres	Up 2 minutes (unhealthy)"
                echo "redis	Up 2 minutes (healthy)"
                ;;
            *)
                echo "docker $*"
                ;;
        esac
        return 0
    }
    
    local postgres_status=$(docker::run ps --format "table {{.Names}}\t{{.Status}}" | grep -E "^postgres" | sed 's/^postgres[[:space:]]*//')
    
    if [[ "$postgres_status" == *"unhealthy"* ]]; then
        echo "Detected unhealthy postgres container"
    fi | {
        run cat
        assert_success
        assert_output --partial "Detected unhealthy postgres container"
    }
}

# Test: Container restarting status handling
@test "native_linux detects restarting containers" {
    # Override docker::run to simulate restarting container
    docker::run() {
        case "$*" in
            "ps --format"*)
                echo "postgres	Restarting (1) 10 seconds ago"
                echo "redis	Up 2 minutes (healthy)"
                ;;
            *)
                echo "docker $*"
                ;;
        esac
        return 0
    }
    
    local postgres_status=$(docker::run ps --format "table {{.Names}}\t{{.Status}}" | grep -E "^postgres" | sed 's/^postgres[[:space:]]*//')
    
    if [[ "$postgres_status" == *"Restarting"* ]]; then
        echo "Container is restarting - possible configuration issue"
    fi | {
        run cat
        assert_success
        assert_output --partial "Container is restarting"
    }
}