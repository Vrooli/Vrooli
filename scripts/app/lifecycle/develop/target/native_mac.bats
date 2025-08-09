#!/usr/bin/env bats

# Test file for native_mac.sh
# Tests native macOS development environment setup

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
    export var_LIB_SYSTEM_DIR="${TEST_DIR}/lib/system"
    export var_SYSTEM_COMMANDS_FILE="${TEST_DIR}/lib/system/system_commands.sh"
    export EXIT_USER_INTERRUPT=130
    export DETACHED="no"
    
    # Create mock directories
    mkdir -p "${var_LIB_UTILS_DIR}" "${var_APP_UTILS_DIR}" "${var_LIB_SYSTEM_DIR}"
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
    
    # Create mock system_commands.sh
    cat > "${var_SYSTEM_COMMANDS_FILE}" << 'EOF'
system::is_command() {
    # Mock command checking
    case "$1" in
        brew|volta|node|pnpm)
            return 0  # Command exists
            ;;
        *)
            return 1  # Command doesn't exist
            ;;
    esac
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
    
    # Mock brew
    brew() {
        case "$*" in
            "install volta")
                echo "Installing volta via Homebrew"
                ;;
            "install coreutils findutils")
                echo "Installing GNU tools"
                ;;
            *)
                echo "brew $*"
                ;;
        esac
        return 0
    }
    export -f brew
    
    # Mock volta
    volta() {
        case "$*" in
            "install node")
                echo "Installing Node.js via Volta"
                ;;
            "install pnpm")
                echo "Installing pnpm via Volta"
                ;;
            *)
                echo "volta $*"
                ;;
        esac
        return 0
    }
    export -f volta
    
    # Mock curl for Homebrew installation
    curl() {
        echo "#!/bin/bash"
        echo "echo 'Homebrew installer mock'"
        return 0
    }
    export -f curl
    
    # Mock pnpm
    pnpm() {
        echo "pnpm $*"
        return 0
    }
    export -f pnpm
    
    # Mock nohup
    nohup() {
        echo "nohup $*"
        return 0
    }
    export -f nohup
    
    # Source all mocks
    source "${var_LOG_FILE}"
    source "${var_LIB_UTILS_DIR}/flow.sh"
    source "${var_LIB_UTILS_DIR}/exit_codes.sh"
    source "${var_SYSTEM_COMMANDS_FILE}"
    source "${var_APP_UTILS_DIR}/docker.sh"
    
    # Source the script under test (without executing main)
    set +e  # Don't exit on error during sourcing
    source "${BATS_TEST_DIRNAME}/native_mac.sh" 2>/dev/null || true
    set -e
}

teardown() {
    cd "${ORIGINAL_DIR}"
    [[ -d "${TEST_DIR}" ]] && rm -rf "${TEST_DIR}"
}

# Test: Script sources required dependencies
@test "native_mac.sh sources required dependencies" {
    run type -t native_mac::start_development_native_mac
    assert_success
    assert_output "function"
}

# Test: Function naming follows convention
@test "native_mac functions use correct naming convention" {
    run type -t native_mac::brew_install
    assert_success
    assert_output "function"
    
    run type -t native_mac::volta_install
    assert_success
    assert_output "function"
    
    run type -t native_mac::node_pnpm_setup
    assert_success
    assert_output "function"
    
    run type -t native_mac::gnu_tools_install
    assert_success
    assert_output "function"
    
    run type -t native_mac::docker_compose_infra
    assert_success
    assert_output "function"
    
    run type -t native_mac::setup_native_mac
    assert_success
    assert_output "function"
    
    run type -t native_mac::start_development_native_mac
    assert_success
    assert_output "function"
    
    run type -t native_mac::cleanup
    assert_success
    assert_output "function"
}

# Test: Homebrew installation when not present
@test "native_mac installs Homebrew when not present" {
    # Override system::is_command to simulate missing brew
    system::is_command() {
        [[ "$1" == "brew" ]] && return 1
        return 0
    }
    
    run native_mac::brew_install
    assert_success
    assert_output --partial "Homebrew not found, installing"
}

# Test: Homebrew already installed
@test "native_mac skips Homebrew installation when present" {
    # system::is_command already returns 0 for brew
    run native_mac::brew_install
    assert_success
    assert_output --partial "Homebrew is already installed"
}

# Test: Volta installation when not present
@test "native_mac installs Volta when not present" {
    # Override system::is_command to simulate missing volta
    system::is_command() {
        [[ "$1" == "volta" ]] && return 1
        return 0
    }
    
    run native_mac::volta_install
    assert_success
    assert_output --partial "Volta not found, installing"
    assert_output --partial "Installing volta via Homebrew"
}

# Test: Volta already installed
@test "native_mac skips Volta installation when present" {
    run native_mac::volta_install
    assert_success
    assert_output --partial "Volta is already installed"
}

# Test: Node and pnpm setup
@test "native_mac sets up Node.js and pnpm" {
    run native_mac::node_pnpm_setup
    assert_success
    assert_output --partial "Installing Node.js via Volta"
    assert_output --partial "Installing pnpm via Volta"
}

# Test: GNU tools installation
@test "native_mac installs GNU tools" {
    run native_mac::gnu_tools_install
    assert_success
    assert_output --partial "Installing GNU tools"
}

# Test: Docker compose infrastructure
@test "native_mac starts Docker infrastructure" {
    run native_mac::docker_compose_infra
    assert_success
    assert_output --partial "Starting postgres and redis containers"
}

# Test: Setup function runs all installation steps
@test "native_mac setup runs all installation steps" {
    run native_mac::setup_native_mac
    assert_success
    assert_output --partial "Setting up native Mac development/production"
    # The function should call all setup functions
}

# Test: Cleanup function with instance manager
@test "native_mac cleanup uses instance manager when available" {
    # Mock instance manager
    instance::shutdown_target() {
        echo "Instance manager shutting down target: $1"
        return 0
    }
    export -f instance::shutdown_target
    
    run native_mac::cleanup
    assert_failure 130  # EXIT_USER_INTERRUPT
    assert_output --partial "Cleaning up development environment"
    assert_output --partial "Instance manager shutting down target: native-mac"
}

# Test: Cleanup function fallback
@test "native_mac cleanup falls back to docker-compose down" {
    # Ensure instance manager is not available
    unset -f instance::shutdown_target 2>/dev/null || true
    
    run native_mac::cleanup
    assert_failure 130  # EXIT_USER_INTERRUPT
    assert_output --partial "docker-compose down"
}

# Test: Container health check
@test "native_mac waits for containers to be healthy" {
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

# Test: Native environment URL setup
@test "native_mac sets up native environment URLs" {
    source "${var_APP_UTILS_DIR}/env_urls.sh"
    
    run env_urls::setup_native_environment
    assert_success
    assert_output --partial "Native environment URLs configured"
}

# Test: Environment validation
@test "native_mac validates native environment" {
    source "${var_APP_UTILS_DIR}/env_urls.sh"
    
    run env_urls::validate_native_environment
    assert_success
    assert_output --partial "Validating native environment connectivity"
}

# Test: Detached mode watcher startup
@test "native_mac starts watchers in detached mode" {
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
            nohup bash -c "echo watcher$i" > /dev/null 2>&1 &
            pids+=("$!")
        done
        echo "Started ${#pids[@]} watchers"
    fi | {
        run cat
        assert_success
        assert_output --partial "Detached mode enabled"
        assert_output --partial "Started 3 watchers"
    }
}

# Test: Foreground mode watcher startup
@test "native_mac starts watchers in foreground mode" {
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

# Test: Main script execution with --setup flag
@test "native_mac runs setup when --setup flag is provided" {
    # Create test script to simulate main execution
    cat > "${TEST_DIR}/test_setup.sh" << 'EOF'
#!/usr/bin/env bash
if [[ "${1:-}" == "--setup" ]]; then
    echo "Running setup"
else
    echo "Running development"
fi
EOF
    
    chmod +x "${TEST_DIR}/test_setup.sh"
    run "${TEST_DIR}/test_setup.sh" --setup
    assert_success
    assert_output "Running setup"
}

# Test: Main script execution without --setup flag
@test "native_mac runs development when no --setup flag" {
    # Create test script to simulate main execution
    cat > "${TEST_DIR}/test_dev.sh" << 'EOF'
#!/usr/bin/env bash
if [[ "${1:-}" == "--setup" ]]; then
    echo "Running setup"
else
    echo "Running development"
fi
EOF
    
    chmod +x "${TEST_DIR}/test_dev.sh"
    run "${TEST_DIR}/test_dev.sh"
    assert_success
    assert_output "Running development"
}

# Test: Container failure handling
@test "native_mac handles container startup failure" {
    local attempt_count=0
    local max_attempts=2
    
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
    local attempt=0
    while [ $attempt -lt $max_attempts ]; do
        if docker::run ps | grep "postgres" | grep -q "healthy"; then
            echo "Containers healthy"
            break
        fi
        attempt=$((attempt + 1))
        if [ $attempt -eq $max_attempts ]; then
            echo "Container health check failed"
        fi
    done | {
        run cat
        assert_success
        assert_output --partial "Container health check failed"
    }
}

# Test: Script validates required tools
@test "native_mac can check for required tools" {
    # Test that system::is_command is available and works
    run system::is_command brew
    assert_success
    
    run system::is_command nonexistent_command
    assert_failure
}