#!/usr/bin/env bats
bats_require_minimum_version 1.5.0

# Source trash module for safe test cleanup
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
# shellcheck disable=SC1091
source "${APP_ROOT}/lib/utils/var.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/trash.sh" 2>/dev/null || true

# Test file for common_deps.sh
# Tests the installation and verification of common system dependencies

setup() {
    # Path to the script under test
    export SCRIPT_PATH="$BATS_TEST_DIRNAME/common_deps.sh"
    export TEST_TMPDIR="${BATS_TEST_TMPDIR:-/tmp}"
    export MOCK_SCRIPT="$TEST_TMPDIR/mock_common_deps_$$.sh"
}

teardown() {
    # Clean up test files
    [[ -f "$MOCK_SCRIPT" ]] && trash::safe_remove "$MOCK_SCRIPT" --test-cleanup
    unset SCRIPT_PATH
    unset TEST_TMPDIR
    unset MOCK_SCRIPT
}

# Helper function to create a mock environment
create_mock_environment() {
    local pm="${1:-apt-get}"
    local has_gpu="${2:-false}"
    local commands_exist="${3:-false}"
    
    cat > "$MOCK_SCRIPT" << EOF
#!/usr/bin/env bash
set -euo pipefail

# Mock log functions
log::header() { echo "[HEADER] \$*"; }
log::info() { echo "[INFO] \$*"; }
log::success() { echo "[SUCCESS] \$*"; }
log::warning() { echo "[WARNING] \$*"; }
log::error() { echo "[ERROR] \$*"; }
log::warn() { log::warning "\$@"; }

# Mock system functions
system::detect_pm() { echo '$pm'; }

system::is_command() {
    if [[ '$commands_exist' == 'true' ]]; then
        return 0
    else
        return 1
    fi
}

system::check_and_install() {
    local cmd="\$1"
    if system::is_command "\$cmd"; then
        log::info "\$cmd is already installed"
    else
        echo "Installing: \$cmd"
    fi
    return 0
}

system::install_pkg() {
    echo "Installing package: \$1"
    return 0
}

system::has_nvidia_gpu() {
    if [[ '$has_gpu' == 'true' ]]; then
        return 0
    else
        return 1
    fi
}

system::install_nvidia_container_runtime() {
    echo "Installing NVIDIA runtime"
    return 0
}

# Mock helm function
helm::check_and_install() {
    echo "Installing Helm"
    return 0
}

# Source the actual script
source '$SCRIPT_PATH'
EOF
    chmod +x "$MOCK_SCRIPT"
}

@test "sourcing common_deps.sh defines required functions" {
    create_mock_environment
    
    run bash -c "
        source '$MOCK_SCRIPT'
        declare -f common_deps::check_and_install >/dev/null && echo 'check_and_install defined'
        declare -f common_deps::install_dev_package >/dev/null && echo 'install_dev_package defined'
        declare -f common_deps::setup_native_build_env >/dev/null && echo 'setup_native_build_env defined'
        declare -f common_deps::setup_nvidia_runtime >/dev/null && echo 'setup_nvidia_runtime defined'
    "
    
    [ "$status" -eq 0 ]
    [[ "$output" =~ "check_and_install defined" ]]
    [[ "$output" =~ "install_dev_package defined" ]]
    [[ "$output" =~ "setup_native_build_env defined" ]]
    [[ "$output" =~ "setup_nvidia_runtime defined" ]]
}

@test "common_deps::check_and_install installs basic tools when missing" {
    create_mock_environment "apt-get" "false" "false"
    
    run bash -c "
        source '$MOCK_SCRIPT'
        common_deps::check_and_install
    "
    
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Installing: curl" ]]
    [[ "$output" =~ "Installing: jq" ]]
    [[ "$output" =~ "Installing: bc" ]]
    [[ "$output" =~ "Common dependencies checked/installed" ]]
}

@test "common_deps::check_and_install skips installed tools" {
    create_mock_environment "apt-get" "false" "true"
    
    run bash -c "
        source '$MOCK_SCRIPT'
        common_deps::check_and_install
    "
    
    [ "$status" -eq 0 ]
    [[ "$output" =~ "curl is already installed" ]]
    [[ "$output" =~ "jq is already installed" ]]
    [[ ! "$output" =~ "Installing: curl" ]]
    [[ ! "$output" =~ "Installing: jq" ]]
}

@test "common_deps::check_and_install handles Linux-specific utilities on apt-get" {
    create_mock_environment "apt-get" "false" "false"
    
    run bash -c "
        source '$MOCK_SCRIPT'
        common_deps::check_and_install
    "
    
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Installing: nproc" ]]
    [[ "$output" =~ "Installing: free" ]]
    [[ "$output" =~ "Installing: systemctl" ]]
}

@test "common_deps::check_and_install skips Linux utilities on macOS" {
    create_mock_environment "brew" "false" "false"
    
    run bash -c "
        source '$MOCK_SCRIPT'
        common_deps::check_and_install
    "
    
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Skipping Linux-only dependencies" ]]
    [[ ! "$output" =~ "Installing: nproc" ]]
    [[ ! "$output" =~ "Installing: systemctl" ]]
}

@test "common_deps::install_dev_package handles already installed packages" {
    create_mock_environment "apt-get"
    
    run bash -c "
        source '$MOCK_SCRIPT'
        
        # Mock dpkg to simulate package is installed
        dpkg() {
            if [[ \"\$1\" == '-l' ]]; then
                echo 'ii  build-essential  12.9  amd64  Informational list of build-essential packages'
                return 0
            fi
        }
        export -f dpkg
        
        common_deps::install_dev_package 'build-essential'
    "
    
    [ "$status" -eq 0 ]
    [[ "$output" =~ "build-essential is already installed" ]]
}

@test "common_deps::install_dev_package installs missing packages" {
    create_mock_environment "apt-get"
    
    run bash -c "
        source '$MOCK_SCRIPT'
        
        # Mock dpkg to simulate package not installed
        dpkg() { return 1; }
        export -f dpkg
        
        common_deps::install_dev_package 'python3-dev'
    "
    
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Installing python3-dev" ]]
    [[ "$output" =~ "python3-dev installed successfully" ]]
}

@test "common_deps::setup_native_build_env for apt-get" {
    create_mock_environment "apt-get" "false" "false"
    
    run bash -c "
        source '$MOCK_SCRIPT'
        common_deps::setup_native_build_env
    "
    
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Setting up native module build environment" ]]
    [[ "$output" =~ "Native module build environment ready" ]]
}

@test "common_deps::setup_native_build_env for dnf/yum" {
    create_mock_environment "dnf"
    
    run bash -c "
        source '$MOCK_SCRIPT'
        common_deps::setup_native_build_env
    "
    
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Installing: gcc" ]]
    [[ "$output" =~ "Installing: make" ]]
    [[ "$output" =~ "Native module build environment ready" ]]
}

@test "common_deps::setup_nvidia_runtime with GPU detected" {
    create_mock_environment "apt-get" "true"
    
    run bash -c "
        source '$MOCK_SCRIPT'
        common_deps::setup_nvidia_runtime
    "
    
    [ "$status" -eq 0 ]
    [[ "$output" =~ "NVIDIA GPU detected" ]]
    [[ "$output" =~ "Installing NVIDIA runtime" ]]
    [[ "$output" =~ "NVIDIA Container Runtime installed" ]]
}

@test "common_deps::setup_nvidia_runtime without GPU" {
    create_mock_environment "apt-get" "false"
    
    run bash -c "
        source '$MOCK_SCRIPT'
        common_deps::setup_nvidia_runtime
    "
    
    [ "$status" -eq 0 ]
    [[ "$output" =~ "No NVIDIA GPU detected" ]]
    [[ ! "$output" =~ "Installing NVIDIA runtime" ]]
}

@test "common_deps::setup_nvidia_runtime handles installation failure" {
    create_mock_environment "apt-get" "true"
    
    run bash -c "
        source '$MOCK_SCRIPT'
        
        # Override to simulate failure
        system::install_nvidia_container_runtime() {
            echo 'Failed to install'
            return 1
        }
        export -f system::install_nvidia_container_runtime
        
        common_deps::setup_nvidia_runtime
    "
    
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Failed to install NVIDIA Container Runtime" ]]
}

@test "script can be run directly" {
    create_mock_environment "apt-get" "false" "true"
    
    run bash -c "
        bash '$MOCK_SCRIPT' common_deps::check_and_install
    "
    
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Common dependencies checked/installed" ]]
}

@test "common_deps handles unknown package managers gracefully" {
    create_mock_environment "unknown-pm"
    
    run bash -c "
        source '$MOCK_SCRIPT'
        common_deps::setup_native_build_env
    "
    
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Unknown package manager unknown-pm" ]]
    [[ "$output" =~ "Please ensure you have: build-essential, python3, make, g++" ]]
}

@test "integration: full installation flow" {
    create_mock_environment "apt-get" "true" "false"
    
    run bash -c "
        source '$MOCK_SCRIPT'
        
        # Mock dpkg for dev packages
        dpkg() { return 1; }
        export -f dpkg
        
        common_deps::check_and_install
    "
    
    [ "$status" -eq 0 ]
    # Check basic tools installed
    [[ "$output" =~ "Installing: curl" ]]
    [[ "$output" =~ "Installing: jq" ]]
    # Check Linux tools installed
    [[ "$output" =~ "Installing: systemctl" ]]
    # Check native build env setup
    [[ "$output" =~ "Setting up native module build environment" ]]
    # Check Helm installed
    [[ "$output" =~ "Installing Helm" ]]
    # Check NVIDIA runtime installed (GPU detected)
    [[ "$output" =~ "NVIDIA GPU detected" ]]
    # Check final success
    [[ "$output" =~ "Common dependencies checked/installed" ]]
}