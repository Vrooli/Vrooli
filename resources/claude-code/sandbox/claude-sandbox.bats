#!/usr/bin/env bats
# Tests for Claude Code sandbox script
bats_require_minimum_version 1.5.0

# Load Vrooli test infrastructure
# shellcheck disable=SC1091
source "${BATS_TEST_DIRNAME}/../../../../lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/trash.sh" 2>/dev/null || true
source "${BATS_TEST_DIRNAME}/../../../../__test/fixtures/setup.bash"

# Setup for the test file
setup_file() {
    # Use Vrooli service test setup
    vrooli_setup_service_test "claude-code-sandbox"
    
    # Export script location
    export SANDBOX_SCRIPT="${BATS_TEST_DIRNAME}/claude-sandbox.sh"
}

# Per-test setup
setup() {
    # Load docker mock
    MOCK_DIR="${BATS_TEST_DIRNAME}/../../../../__test/fixtures/mocks"
    if [[ -f "$MOCK_DIR/docker.sh" ]]; then
        source "$MOCK_DIR/docker.sh"
    fi
    
    # Setup standard Vrooli mocks
    vrooli_auto_setup
    
    # Reset docker mock state if available
    if declare -f mock::docker::reset >/dev/null 2>&1; then
        mock::docker::reset
    fi
    
    # Set up test environment variables
    export CLAUDE_SANDBOX_CONFIG="${BATS_TEST_TMPDIR}/.claude-sandbox-test"
    export CLAUDE_SANDBOX_IMAGE="vrooli/claude-code-sandbox:test"
    
    # Create test directories
    mkdir -p "${CLAUDE_SANDBOX_CONFIG}"
    mkdir -p "${BATS_TEST_TMPDIR}/test-files"
}

# Teardown function
teardown() {
    # Clean up test directories
    if [[ -n "${CLAUDE_SANDBOX_CONFIG:-}" && -d "$CLAUDE_SANDBOX_CONFIG" ]]; then
        trash::safe_remove "$CLAUDE_SANDBOX_CONFIG" --test-cleanup
    fi
    
    if [[ -n "${BATS_TEST_TMPDIR:-}" && -d "${BATS_TEST_TMPDIR}/test-files" ]]; then
        trash::safe_remove "${BATS_TEST_TMPDIR}/test-files" --test-cleanup
    fi
    
    vrooli_cleanup_test
}

# ============================================================================
# Script Loading Tests
# ============================================================================

@test "claude-sandbox.sh loads without errors" {
    run bash -c "source '${SANDBOX_SCRIPT}' && echo 'Script loaded'"
    assert_success
}

@test "claude-sandbox.sh defines required functions" {
    run bash -c "
        source '${SANDBOX_SCRIPT}'
        declare -f claude_sandbox::check_docker >/dev/null && echo 'check_docker exists'
        declare -f claude_sandbox::build_sandbox >/dev/null && echo 'build_sandbox exists'
        declare -f claude_sandbox::setup_auth >/dev/null && echo 'setup_auth exists'
        declare -f claude_sandbox::run_sandbox >/dev/null && echo 'run_sandbox exists'
        declare -f claude_sandbox::usage >/dev/null && echo 'usage exists'
    "
    assert_success
    assert_output_contains "check_docker exists"
    assert_output_contains "build_sandbox exists"
    assert_output_contains "setup_auth exists"
    assert_output_contains "run_sandbox exists"
    assert_output_contains "usage exists"
}

# ============================================================================
# Usage and Help Tests
# ============================================================================

@test "claude-sandbox.sh shows usage with help flag" {
    run bash "${SANDBOX_SCRIPT}" help
    assert_success
    assert_output_contains "Claude Code Sandbox"
    assert_output_contains "Usage:"
    assert_output_contains "Commands:"
}

@test "claude-sandbox.sh shows usage with --help flag" {
    run bash "${SANDBOX_SCRIPT}" --help
    assert_success
    assert_output_contains "Claude Code Sandbox"
}

@test "claude-sandbox.sh shows usage with -h flag" {
    run bash "${SANDBOX_SCRIPT}" -h
    assert_success
    assert_output_contains "Claude Code Sandbox"
}

# ============================================================================
# Docker Check Tests
# ============================================================================

@test "claude_sandbox::check_docker detects missing docker" {
    # Configure mock to simulate docker not installed
    if declare -f mock::docker::set_installed >/dev/null 2>&1; then
        mock::docker::set_installed "false"
    fi
    
    run bash -c "
        source '${SANDBOX_SCRIPT}'
        claude_sandbox::check_docker
    "
    assert_failure
    assert_output_contains "Docker is not installed"
}

@test "claude_sandbox::check_docker detects docker daemon not running" {
    # Configure mock to simulate docker daemon not running
    if declare -f mock::docker::set_daemon_running >/dev/null 2>&1; then
        mock::docker::set_installed "true"
        mock::docker::set_daemon_running "false"
    fi
    
    run bash -c "
        source '${SANDBOX_SCRIPT}'
        claude_sandbox::check_docker
    "
    assert_failure
    assert_output_contains "Docker daemon is not running"
}

@test "claude_sandbox::check_docker succeeds when docker is available" {
    # Configure mock to simulate docker working properly
    if declare -f mock::docker::set_installed >/dev/null 2>&1; then
        mock::docker::set_installed "true"
        mock::docker::set_daemon_running "true"
    fi
    
    run bash -c "
        source '${SANDBOX_SCRIPT}'
        claude_sandbox::check_docker
    "
    assert_success
}

# ============================================================================
# Build Sandbox Tests
# ============================================================================

@test "claude_sandbox::build_sandbox calls docker compose build" {
    # Configure mock to track docker compose calls
    if declare -f mock::docker::set_installed >/dev/null 2>&1; then
        mock::docker::set_installed "true"
        mock::docker::set_daemon_running "true"
    fi
    
    run bash -c "
        source '${SANDBOX_SCRIPT}'
        claude_sandbox::build_sandbox
    "
    assert_success
    assert_output_contains "Building Claude Code sandbox image"
    assert_output_contains "Sandbox image built successfully"
}

# ============================================================================
# Authentication Tests
# ============================================================================

@test "claude_sandbox::check_auth detects missing authentication" {
    # Ensure no credentials file exists
    trash::safe_remove "${CLAUDE_SANDBOX_CONFIG}/.credentials.json" --test-cleanup
    
    # Mock claude command to skip actual authentication
    claude() {
        case "$1" in
            login) echo "Mock login successful" ;;
            doctor) echo "Mock doctor check passed" ;;
            *) echo "Mock claude: $*" ;;
        esac
    }
    export -f claude
    
    run bash -c "
        source '${SANDBOX_SCRIPT}'
        claude_sandbox::check_auth <<< 'n'
    "
    # Should detect missing auth but not set it up (user said no)
    assert_output_contains "Sandbox authentication not configured"
}

@test "claude_sandbox::setup_auth creates configuration directory" {
    # Remove existing config
    trash::safe_remove "${CLAUDE_SANDBOX_CONFIG}" --test-cleanup
    
    # Mock claude command
    claude() {
        case "$1" in
            login) 
                echo "Mock login successful"
                # Create mock credentials file
                mkdir -p "${CLAUDE_CONFIG_DIR:-$CLAUDE_SANDBOX_CONFIG}"
                echo '{"token": "mock"}' > "${CLAUDE_CONFIG_DIR:-$CLAUDE_SANDBOX_CONFIG}/.credentials.json"
                ;;
            doctor) echo "Mock doctor check passed" ;;
            *) echo "Mock claude: $*" ;;
        esac
    }
    export -f claude
    
    run bash -c "
        source '${SANDBOX_SCRIPT}'
        # Export so the mock can see it
        export CLAUDE_SANDBOX_CONFIG='${CLAUDE_SANDBOX_CONFIG}'
        claude_sandbox::setup_auth <<< ''
    "
    assert_success
    assert_output_contains "Setting up sandbox authentication"
    
    # Check that config directory was created
    [[ -d "${CLAUDE_SANDBOX_CONFIG}" ]]
}

# ============================================================================
# Command Routing Tests
# ============================================================================

@test "claude-sandbox.sh routes setup command correctly" {
    # Mock the claude command
    claude() { echo "Mock claude: $*"; }
    export -f claude
    
    run bash "${SANDBOX_SCRIPT}" setup <<< "n"
    assert_output_contains "Setting up sandbox authentication"
}

@test "claude-sandbox.sh routes build command correctly" {
    # Configure docker mock
    if declare -f mock::docker::set_installed >/dev/null 2>&1; then
        mock::docker::set_installed "true"
        mock::docker::set_daemon_running "true"
    fi
    
    run bash "${SANDBOX_SCRIPT}" build
    assert_success
    assert_output_contains "Building Claude Code sandbox image"
}

@test "claude-sandbox.sh routes stop command correctly" {
    # Configure docker mock
    if declare -f mock::docker::set_installed >/dev/null 2>&1; then
        mock::docker::set_installed "true"
        mock::docker::set_daemon_running "true"
    fi
    
    run bash "${SANDBOX_SCRIPT}" stop
    assert_success
    assert_output_contains "Stopping Claude Code sandbox"
}

@test "claude-sandbox.sh routes cleanup command correctly" {
    # Configure docker mock
    if declare -f mock::docker::set_installed >/dev/null 2>&1; then
        mock::docker::set_installed "true"
        mock::docker::set_daemon_running "true"
    fi
    
    run bash "${SANDBOX_SCRIPT}" cleanup <<< "n"
    assert_success
    assert_output_contains "This will remove sandbox containers"
}

# ============================================================================
# Environment Variable Tests
# ============================================================================

@test "claude-sandbox.sh respects CLAUDE_SANDBOX_CONFIG environment variable" {
    export CLAUDE_SANDBOX_CONFIG="/custom/path/.claude-sandbox"
    
    run bash -c "
        source '${SANDBOX_SCRIPT}'
        echo \"Config path: \$CLAUDE_SANDBOX_CONFIG\"
    "
    assert_success
    assert_output_contains "Config path: /custom/path/.claude-sandbox"
}

@test "claude-sandbox.sh uses default config path when not set" {
    unset CLAUDE_SANDBOX_CONFIG
    
    run bash -c "
        source '${SANDBOX_SCRIPT}'
        echo \"Config path: \$CLAUDE_SANDBOX_CONFIG\"
    "
    assert_success
    assert_output_contains "Config path: ${HOME}/.claude-sandbox"
}

# ============================================================================
# Safety Features Tests
# ============================================================================

@test "claude-sandbox.sh creates test-files directory" {
    # Configure docker mock
    if declare -f mock::docker::set_installed >/dev/null 2>&1; then
        mock::docker::set_installed "true"
        mock::docker::set_daemon_running "true"
        mock::docker::add_image "${CLAUDE_SANDBOX_IMAGE}"
    fi
    
    # Create mock credentials
    touch "${CLAUDE_SANDBOX_CONFIG}/.credentials.json"
    
    run bash -c "
        source '${SANDBOX_SCRIPT}'
        # The run_sandbox function should create test-files dir
        claude_sandbox::run_sandbox interactive 2>&1 | head -5
    "
    # Should attempt to run and show sandbox warning
    assert_output_contains "SANDBOX MODE"
}

# ============================================================================
# Error Handling Tests
# ============================================================================

@test "claude-sandbox.sh handles unknown commands gracefully" {
    # Configure docker mock
    if declare -f mock::docker::set_installed >/dev/null 2>&1; then
        mock::docker::set_installed "true"
        mock::docker::set_daemon_running "true"
        mock::docker::add_image "${CLAUDE_SANDBOX_IMAGE}"
    fi
    
    # Create mock credentials
    touch "${CLAUDE_SANDBOX_CONFIG}/.credentials.json"
    
    run bash "${SANDBOX_SCRIPT}" unknown-command
    # Should try to pass through to claude
    assert_output_contains "SANDBOX MODE"
}

@test "claude-sandbox.sh exits cleanly on docker errors" {
    # Configure mock to simulate docker error
    if declare -f mock::docker::inject_error >/dev/null 2>&1; then
        mock::docker::inject_error "compose" "general_error"
    fi
    
    run bash "${SANDBOX_SCRIPT}" build
    assert_failure
    # Should show some error message
    [[ "$output" != "" ]]
}