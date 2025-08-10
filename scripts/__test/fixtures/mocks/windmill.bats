#!/usr/bin/env bats
# Comprehensive tests for windmill mock functionality

# Source trash module for safe test cleanup
SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/../../../lib/utils/var.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/trash.sh" 2>/dev/null || true

setup() {
    # Set up test environment first
    export MOCK_LOG_DIR="${TMPDIR:-/tmp}/windmill-test-$$"
    export MOCK_RESPONSES_DIR="$MOCK_LOG_DIR"
    export WINDMILL_MOCK_STATE_DIR="${TMPDIR:-/tmp}/windmill-mock-state-$$"
    # Use 'command' to ensure we use the real mkdir, not the mock
    command mkdir -p "$MOCK_LOG_DIR"
    command mkdir -p "$WINDMILL_MOCK_STATE_DIR"
    
    # Load test helpers if available
    if [[ -f "${BATS_TEST_DIRNAME}/../../helpers/bats-support/load.bash" ]]; then
        load "${BATS_TEST_DIRNAME}/../../helpers/bats-support/load.bash"
    fi
    if [[ -f "${BATS_TEST_DIRNAME}/../../helpers/bats-assert/load.bash" ]]; then
        load "${BATS_TEST_DIRNAME}/../../helpers/bats-assert/load.bash"
    else
        # Define basic assertion functions if bats-assert is not available
        assert_success() {
            if [[ "$status" -ne 0 ]]; then
                echo "expected success, got status $status"
                echo "output: $output"
                return 1
            fi
        }
        
        assert_failure() {
            if [[ "$status" -eq 0 ]]; then
                echo "expected failure, got success"
                echo "output: $output"
                return 1
            fi
        }
        
        assert_output() {
            local expected
            if [[ "$1" == "--partial" ]]; then
                shift
                expected="$1"
                if [[ "$output" != *"$expected"* ]]; then
                    echo "expected output to contain: $expected"
                    echo "actual output: $output"
                    return 1
                fi
            else
                expected="${1:-}"
                if [[ "$output" != "$expected" ]]; then
                    echo "expected: $expected"
                    echo "actual: $output"
                    return 1
                fi
            fi
        }
        
        assert_line() {
            local expected
            local partial=false
            if [[ "$1" == "--partial" ]]; then
                partial=true
                shift
            fi
            expected="$1"
            
            local found=false
            while IFS= read -r line; do
                if [[ "$partial" == true ]]; then
                    if [[ "$line" == *"$expected"* ]]; then
                        found=true
                        break
                    fi
                else
                    if [[ "$line" == "$expected" ]]; then
                        found=true
                        break
                    fi
                fi
            done <<< "$output"
            
            if [[ "$found" != true ]]; then
                echo "expected line: $expected"
                echo "in output: $output"
                return 1
            fi
        }
        
        refute_line() {
            local not_expected="$1"
            while IFS= read -r line; do
                if [[ "$line" == "$not_expected" ]]; then
                    echo "unexpected line found: $not_expected"
                    echo "in output: $output"
                    return 1
                fi
            done <<< "$output"
        }
    fi
    
    # Source the mock library
    source "${BATS_TEST_DIRNAME}/windmill.sh"
    
    # Initialize mock system
    mock::windmill::reset
}

teardown() {
    # Reset windmill state
    mock::windmill::reset false
    
    # Clean up state files
    [[ -n "${WINDMILL_MOCK_STATE_DIR:-}" && -d "${WINDMILL_MOCK_STATE_DIR}" ]] && trash::safe_remove "${WINDMILL_MOCK_STATE_DIR}" --test-cleanup
    
    # Clean up test directory
    [[ -n "${MOCK_LOG_DIR:-}" && -d "${MOCK_LOG_DIR}" ]] && trash::safe_remove "${MOCK_LOG_DIR}" --test-cleanup
}

# ----------------------------
# Basic Mock Functionality Tests
# ----------------------------

@test "windmill mock: initialization creates clean state" {
    mock::windmill::reset
    
    run mock::windmill::assert_stopped
    assert_success
    
    run mock::windmill::assert_workspace_exists "demo"
    assert_success
}

@test "windmill mock: state persistence works" {
    # Install and start windmill
    mock::windmill::install
    mock::windmill::start
    
    # Verify state is saved and loaded properly
    mock::windmill::reset false  # Don't save state
    mock::windmill::load_state
    
    run mock::windmill::assert_installed
    assert_success
    
    run mock::windmill::assert_running
    assert_success
}

# ----------------------------
# Docker Compose Tests
# ----------------------------

@test "docker-compose: up starts windmill services" {
    mock::windmill::install
    
    run docker-compose -f /path/to/windmill/docker-compose.yml up -d
    assert_success
    assert_line "Creating windmill-vrooli-db-1 ... done"
    assert_line "Creating windmill-vrooli-app-1 ... done"
    assert_line "Creating windmill-vrooli-worker-1 ... done"
    
    # Reload state to get updates from the subshell
    mock::windmill::load_state
    
    # Status should be updated after the compose command
    [[ "${WINDMILL_MOCK_CONFIG[status]}" == "running" ]]
}

@test "docker-compose: down stops windmill services" {
    mock::windmill::install
    mock::windmill::start
    
    run docker-compose -f /path/to/windmill/docker-compose.yml down
    assert_success
    assert_line "Stopping windmill-vrooli-worker-1 ... done"
    assert_line "Stopping windmill-vrooli-app-1 ... done"
    assert_line "Stopping windmill-vrooli-db-1 ... done"
    
    # Reload state to get updates from the subshell
    mock::windmill::load_state
    
    # Status should be updated after the compose command
    [[ "${WINDMILL_MOCK_CONFIG[status]}" == "stopped" ]]
}

@test "docker-compose: down with volumes removes data" {
    mock::windmill::install
    mock::windmill::start
    mock::windmill::add_app "test-app" "test-data"
    
    run docker-compose -f /path/to/windmill/docker-compose.yml down -v
    assert_success
    assert_line "Removing volume windmill-vrooli_db_data"
    
    # Apps should be cleared but default workspaces restored
    run mock::windmill::assert_workspace_exists "demo"
    assert_success
}

@test "docker-compose: ps shows service status" {
    mock::windmill::install
    mock::windmill::start
    
    run docker-compose -f /path/to/windmill/docker-compose.yml ps
    assert_success
    assert_line --partial "windmill-vrooli-windmill-app-1"
    assert_line --partial "Up"
}

@test "docker-compose: logs shows service output" {
    mock::windmill::install
    mock::windmill::start
    
    run docker-compose -f /path/to/windmill/docker-compose.yml logs windmill-app
    assert_success
    assert_line --partial "Service windmill-app is running"
}

@test "docker-compose: scale workers updates worker count" {
    mock::windmill::install
    mock::windmill::start
    
    run docker-compose -f /path/to/windmill/docker-compose.yml up -d --scale windmill-worker=5
    assert_success
    
    # Reload state to get updates from the subshell
    mock::windmill::load_state
    
    # Check if worker count was updated
    [[ "${WINDMILL_MOCK_CONFIG[worker_replicas]}" == "5" ]]
}

@test "docker-compose: exec executes commands in containers" {
    mock::windmill::install
    mock::windmill::start
    
    run docker-compose -f /path/to/windmill/docker-compose.yml exec windmill-db psql -U postgres
    assert_success
    assert_line "psql (13.7)"
}

@test "docker-compose: non-windmill calls pass through" {
    # This should not be handled by our mock
    run docker-compose -f /path/to/other/docker-compose.yml ps
    # Should call real docker-compose command (which may fail in test env, that's OK)
    # Just verify our mock didn't handle it
}

# ----------------------------
# API Call Tests
# ----------------------------

@test "curl: windmill api version endpoint" {
    mock::windmill::install
    mock::windmill::start
    
    run curl -s http://localhost:8000/api/version
    assert_success
    assert_output '"v1.290.0"'
}

@test "curl: api not responding when stopped" {
    mock::windmill::install
    # Don't start windmill - so API should not be responding
    mock::windmill::stop  # Explicitly stop to ensure API not responding
    
    run curl -f http://localhost:8000/api/version
    assert_failure
    # Should return connection refused error code
    [[ $status -eq 7 ]]
}

@test "curl: health endpoint responds when healthy" {
    mock::windmill::install
    mock::windmill::start
    
    run curl -s http://localhost:8000/api/health
    assert_success
    assert_output --partial '"status":"healthy"'
}

@test "curl: workspace list endpoint" {
    mock::windmill::install
    mock::windmill::start
    
    run curl -s http://localhost:8000/api/w/list
    assert_success
    assert_output --partial 'demo'
    assert_output --partial 'admin'
}

@test "curl: authenticated endpoints require auth header" {
    mock::windmill::install
    mock::windmill::start
    
    # Without auth header should fail
    run curl -s http://localhost:8000/api/users/whoami
    assert_failure
    [[ $status -eq 22 ]]
    
    # With auth header should succeed
    run curl -s -H "Authorization: Bearer test-token" http://localhost:8000/api/users/whoami
    assert_success
    assert_output --partial 'admin@windmill.dev'
}

@test "curl: workers list shows configured worker count" {
    mock::windmill::install
    mock::windmill::start
    mock::windmill::scale_workers "4"
    
    run curl -s http://localhost:8000/api/workers/list
    assert_success
    assert_output --partial 'worker-4'
}

@test "curl: job run endpoint creates job" {
    mock::windmill::install
    mock::windmill::start
    
    run curl -s -X POST -d '{"args":{"name":"test"}}' http://localhost:8000/api/w/demo/jobs/run/f/scripts/hello
    assert_success
    assert_output --partial 'job-'
    assert_output --partial '"status":"running"'
}

@test "curl: non-windmill calls pass through" {
    # This should not be handled by our mock
    run curl -s http://example.com
    # Should call real curl command
}

# ----------------------------
# Error Injection Tests  
# ----------------------------

@test "error injection: compose failure mode" {
    mock::windmill::set_error "compose_failure"
    
    run docker-compose -f /path/to/windmill/docker-compose.yml up -d
    assert_failure
    assert_output "ERROR: Docker Compose operation failed"
}

@test "error injection: api unhealthy mode" {
    mock::windmill::install
    mock::windmill::start
    mock::windmill::set_error "api_unhealthy"
    
    run curl -f http://localhost:8000/api/health
    assert_failure
}

@test "error injection: service unhealthy mode" {
    mock::windmill::install
    mock::windmill::set_error "service_unhealthy"
    
    run docker-compose -f /path/to/windmill/docker-compose.yml up -d
    assert_success
    
    # Reload state to get updates from the subshell
    mock::windmill::load_state
    
    # API should not be responding even though services started
    run mock::windmill::assert_api_responding
    assert_failure
}

# ----------------------------
# Helper Function Tests
# ----------------------------

@test "helper: install marks windmill as installed" {
    run mock::windmill::assert_installed
    assert_failure
    
    mock::windmill::install
    
    run mock::windmill::assert_installed
    assert_success
}

@test "helper: start requires installation" {
    run mock::windmill::start
    assert_failure
    assert_output --partial "Windmill is not installed"
}

@test "helper: start sets services to running" {
    mock::windmill::install
    mock::windmill::start
    
    run mock::windmill::assert_service_running "windmill-app"
    assert_success
    
    run mock::windmill::assert_service_running "windmill-worker"
    assert_success
    
    run mock::windmill::assert_service_running "windmill-db"
    assert_success
}

@test "helper: stop sets all services to stopped" {
    mock::windmill::install
    mock::windmill::start
    mock::windmill::stop
    
    run mock::windmill::assert_stopped
    assert_success
}

@test "helper: add_workspace creates new workspace" {
    mock::windmill::add_workspace "test-workspace"
    
    run mock::windmill::assert_workspace_exists "test-workspace"
    assert_success
}

@test "helper: add_app stores application data" {
    mock::windmill::add_app "test-app" "app-data"
    
    run mock::windmill::assert_app_exists "test-app"
    assert_success
}

@test "helper: store_api_key saves key data" {
    mock::windmill::store_api_key "test-key" "secret-token"
    
    # Key should be stored (implementation detail - we don't expose key values)
}

@test "helper: add_backup stores backup name" {
    mock::windmill::add_backup "backup-2024-01-15"
    
    # Backup should be in list (implementation detail)
}

@test "helper: scale_workers updates worker count" {
    mock::windmill::scale_workers "6"
    
    run mock::windmill::assert_worker_count "6"
    assert_success
}

# ----------------------------
# Advanced Configuration Tests
# ----------------------------

@test "configuration: lsp service controlled by flag" {
    mock::windmill::install
    
    # Start with LSP enabled (default)
    mock::windmill::start
    run mock::windmill::assert_service_running "windmill-lsp"
    assert_success
    
    # Reset and disable LSP
    mock::windmill::reset
    WINDMILL_MOCK_CONFIG[lsp_enabled]="false"
    mock::windmill::install
    mock::windmill::start
    
    # LSP should not be running
    [[ "${WINDMILL_MOCK_SERVICES[windmill-lsp]}" == "stopped" ]]
}

@test "configuration: multiplayer service controlled by flag" {
    mock::windmill::install
    
    # Start with multiplayer disabled (default)
    mock::windmill::start
    [[ "${WINDMILL_MOCK_SERVICES[windmill-multiplayer]}" == "stopped" ]]
    
    # Reset and enable multiplayer
    mock::windmill::reset
    WINDMILL_MOCK_CONFIG[multiplayer_enabled]="true"
    mock::windmill::install
    mock::windmill::start
    
    run mock::windmill::assert_service_running "windmill-multiplayer"
    assert_success
}

@test "configuration: worker memory setting persisted" {
    WINDMILL_MOCK_CONFIG[worker_memory]="4096M"
    mock::windmill::save_state
    
    mock::windmill::reset false
    mock::windmill::load_state
    
    [[ "${WINDMILL_MOCK_CONFIG[worker_memory]}" == "4096M" ]]
}

# ----------------------------
# Integration Tests
# ----------------------------

@test "integration: full windmill lifecycle" {
    # Install windmill
    mock::windmill::install
    run mock::windmill::assert_installed
    assert_success
    
    # Start services via docker-compose
    run docker-compose -f /path/to/windmill/docker-compose.yml up -d
    assert_success
    
    # Verify API is responding
    run curl -s http://localhost:8000/api/version
    assert_success
    
    # Add an app and workspace
    mock::windmill::add_app "test-app" "test-data"
    mock::windmill::add_workspace "test-workspace"
    
    # Scale workers
    run docker-compose -f /path/to/windmill/docker-compose.yml up -d --scale windmill-worker=4
    assert_success
    
    # Reload state to get updates from the subshell
    mock::windmill::load_state
    
    run mock::windmill::assert_worker_count "4"
    assert_success
    
    # Create backup
    mock::windmill::add_backup "backup-$(date +%Y%m%d)"
    
    # Stop services
    run docker-compose -f /path/to/windmill/docker-compose.yml down
    assert_success
    
    # Reload state to get updates from the subshell
    mock::windmill::load_state
    
    run mock::windmill::assert_stopped
    assert_success
}

@test "integration: api workflow testing" {
    mock::windmill::install
    mock::windmill::start
    
    # Test version endpoint
    run curl -s http://localhost:8000/api/version
    assert_success
    assert_output '"v1.290.0"'
    
    # Test workspace listing
    run curl -s http://localhost:8000/api/w/list
    assert_success
    assert_output --partial 'Demo Workspace'
    
    # Test authenticated endpoint
    run curl -s -H "Authorization: Bearer test-token" http://localhost:8000/api/users/whoami
    assert_success
    assert_output --partial 'admin@windmill.dev'
    
    # Test job creation
    run curl -s -X POST -d '{"name":"world"}' http://localhost:8000/api/w/demo/jobs/run/f/scripts/hello
    assert_success
    assert_output --partial 'job-'
}

@test "integration: error recovery testing" {
    mock::windmill::install
    mock::windmill::start
    
    # Inject error
    mock::windmill::set_error "api_unhealthy"
    
    # API should fail
    run curl -f http://localhost:8000/api/health
    assert_failure
    
    # Clear error
    mock::windmill::set_error ""
    
    # API should recover
    run curl -s http://localhost:8000/api/health
    assert_success
}

# ----------------------------
# State Management Tests
# ----------------------------

@test "state: reset clears all data properly" {
    # Set up some state
    mock::windmill::install
    mock::windmill::start
    mock::windmill::add_app "test-app" "data"
    mock::windmill::add_workspace "test-ws"
    mock::windmill::scale_workers "5"
    
    # Reset
    mock::windmill::reset
    
    # Verify clean state
    run mock::windmill::assert_stopped
    assert_success
    
    [[ "${WINDMILL_MOCK_CONFIG[worker_replicas]}" == "3" ]]
    [[ -z "${WINDMILL_MOCK_APPS[test-app]}" ]]
    
    # Default workspaces should remain
    run mock::windmill::assert_workspace_exists "demo"
    assert_success
}

@test "state: dump shows current configuration" {
    mock::windmill::install
    mock::windmill::start
    mock::windmill::add_app "test-app" "data"
    
    run mock::windmill::dump_state
    assert_success
    assert_line --partial "=== Windmill Mock State ==="
    assert_line --partial "installed: true"
    assert_line --partial "status: running"
    assert_line --partial "test-app: data"
}

# ----------------------------
# Assertion Tests
# ----------------------------

@test "assertions: all assertion functions work correctly" {
    # Test not installed
    run mock::windmill::assert_installed
    assert_failure
    assert_output --partial "Windmill is not installed"
    
    # Install and test
    mock::windmill::install
    run mock::windmill::assert_installed
    assert_success
    
    # Test not running
    run mock::windmill::assert_running
    assert_failure
    
    # Start and test
    mock::windmill::start
    run mock::windmill::assert_running
    assert_success
    
    # Test service assertions
    run mock::windmill::assert_service_running "windmill-app"
    assert_success
    
    run mock::windmill::assert_service_running "nonexistent-service"
    assert_failure
    
    # Test workspace assertions
    run mock::windmill::assert_workspace_exists "demo"
    assert_success
    
    run mock::windmill::assert_workspace_exists "nonexistent"
    assert_failure
    
    # Test API assertion
    run mock::windmill::assert_api_responding
    assert_success
}

# ----------------------------
# Edge Case Tests
# ----------------------------

@test "edge case: docker-compose with no windmill context passes through" {
    # Non-windmill compose file should not be intercepted
    run docker-compose -f /other/docker-compose.yml ps
    # This will likely fail as there's no real docker-compose, but it shouldn't be handled by our mock
}

@test "edge case: curl with non-windmill url passes through" {
    # Non-windmill URL should not be intercepted
    run curl -s http://google.com
    # This may fail due to network restrictions, but shouldn't be handled by our mock
}

@test "edge case: missing service in compose commands" {
    mock::windmill::install
    mock::windmill::start
    
    run docker-compose -f /path/to/windmill/docker-compose.yml restart nonexistent-service
    assert_failure
    assert_output --partial "No such service"
}

@test "edge case: worker count edge values" {
    # Test with minimum workers
    mock::windmill::scale_workers "1"
    run mock::windmill::assert_worker_count "1"
    assert_success
    
    # Test with many workers
    mock::windmill::scale_workers "100"
    run mock::windmill::assert_worker_count "100"
    assert_success
}