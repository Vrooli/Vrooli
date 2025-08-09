#!/usr/bin/env bats
# instance_manager.bats - Tests for instance management functions

setup() {
    # Set up test environment
    export PROJECT_ROOT="${BATS_TEST_DIRNAME}/../../.."
    export var_ROOT_DIR="${PROJECT_ROOT}"
    export var_SERVICE_JSON_FILE="${PROJECT_ROOT}/.vrooli/service.json"
    export var_DOCKER_COMPOSE_DEV_FILE="${PROJECT_ROOT}/docker/docker-compose.yml"
    
    # Create temporary test directory
    export TEST_DIR=$(mktemp -d)
    export TEST_SERVICE_JSON="${TEST_DIR}/service.json"
    export HOME="${TEST_DIR}/home"
    mkdir -p "$HOME"
    
    # Initialize mock logging infrastructure first
    source "${PROJECT_ROOT}/scripts/__test/fixtures/mocks/logs.sh"
    mock::init_logging
    
    # Source test fixtures and mocks
    source "${PROJECT_ROOT}/scripts/__test/fixtures/mocks/docker.sh"
    source "${PROJECT_ROOT}/scripts/__test/fixtures/mocks/system.sh"
    
    # Source the instance manager
    source "${BATS_TEST_DIRNAME}/instance_manager.sh"
    
    # Create a test service.json with instance management config
    cat > "$TEST_SERVICE_JSON" <<'EOF'
{
  "instanceManagement": {
    "enabled": true,
    "strategy": "single",
    "conflicts": {
      "action": "prompt",
      "timeout": 30
    },
    "detection": {
      "services": ["server", "ui", "database"],
      "ports": {
        "server": "5329",
        "ui": "3000",
        "database": "5432"
      },
      "containers": ["app-*", "*-server", "*-ui"],
      "processes": ["node.*server", "npm.*dev", "pnpm.*dev", "vite"]
    },
    "dockerCompose": {
      "files": ["docker-compose.yml", "docker-compose.yaml"],
      "services": ["server", "ui", "database"]
    }
  }
}
EOF
    
    # Override the service.json path for testing
    export var_SERVICE_JSON_FILE="$TEST_SERVICE_JSON"
    
    # Disable interactive prompts in tests
    export INTERACTIVE=false
    export YES=yes
    
    # Initialize mock states
    mock::docker::reset
    mock::system::reset
}

teardown() {
    # Clean up test directory
    if [[ -d "$TEST_DIR" ]]; then
        rm -rf "$TEST_DIR"
    fi
    
    # Clean up mock state files
    if [[ -n "${DOCKER_MOCK_STATE_FILE:-}" && -f "$DOCKER_MOCK_STATE_FILE" ]]; then
        rm -f "$DOCKER_MOCK_STATE_FILE"
    fi
    if [[ -n "${SYSTEM_MOCK_STATE_FILE:-}" && -f "$SYSTEM_MOCK_STATE_FILE" ]]; then
        rm -f "$SYSTEM_MOCK_STATE_FILE"
    fi
}

# ==============================================================================
# Configuration Loading Tests
# ==============================================================================

@test "instance::load_config - loads default configuration when no service.json" {
    rm -f "$TEST_SERVICE_JSON"
    unset var_SERVICE_JSON_FILE
    
    instance::load_config
    
    # Check defaults are loaded
    [[ "$INSTANCE_MANAGEMENT_ENABLED" == "true" ]]
    [[ "$INSTANCE_STRATEGY" == "single" ]]
    [[ "$CONFLICT_ACTION" == "prompt" ]]
    [[ "$CONFLICT_TIMEOUT" == "30" ]]
    [[ ${#APP_SERVICES[@]} -gt 0 ]]
}

@test "instance::load_config - loads configuration from service.json" {
    instance::load_config
    
    # Check configuration is loaded from file
    [[ "$INSTANCE_MANAGEMENT_ENABLED" == "true" ]]
    [[ "$INSTANCE_STRATEGY" == "single" ]]
    [[ "$CONFLICT_ACTION" == "prompt" ]]
    [[ ${#APP_SERVICES[@]} -eq 3 ]]
    [[ "${APP_SERVICES[0]}" == "server" ]]
    [[ "${SERVICE_PORTS[server]}" == "5329" ]]
}

@test "instance::load_config - handles disabled instance management" {
    cat > "$TEST_SERVICE_JSON" <<'EOF'
{
  "instanceManagement": {
    "enabled": false
  }
}
EOF
    
    instance::load_config
    
    [[ "$INSTANCE_MANAGEMENT_ENABLED" == "false" ]]
}

# ==============================================================================
# Docker Detection Tests
# ==============================================================================

@test "instance::docker_available - returns true when Docker is running" {
    export DOCKER_MOCK_MODE="normal"
    mock::system::set_command "docker" "true"
    
    run instance::docker_available
    [ "$status" -eq 0 ]
}

@test "instance::docker_available - returns false when Docker is not available" {
    export DOCKER_MOCK_MODE="offline"
    mock::system::set_command "docker" "false"
    
    run instance::docker_available
    [ "$status" -eq 1 ]
}

@test "instance::get_docker_containers - detects running containers" {
    # Load configuration first to initialize CONTAINER_PATTERNS
    instance::load_config
    
    export DOCKER_MOCK_MODE="normal"
    mock::system::set_command "docker" "true"
    mock::docker::set_container_state "app-server" "running" "node:18"
    mock::docker::set_container_state "app-ui" "running" "node:18"
    
    instance::get_docker_containers
    
    # Check containers were detected
    [[ -n "${DOCKER_INSTANCES[server_container]:-}" ]]
    [[ "${DOCKER_INSTANCES[server_state]:-}" == "running" ]]
    [[ -n "${DOCKER_INSTANCES[ui_container]:-}" ]]
    [[ "${DOCKER_INSTANCES[ui_state]:-}" == "running" ]]
}

@test "instance::get_docker_containers - ignores stopped containers for state" {
    # Load configuration first
    instance::load_config
    
    export DOCKER_MOCK_MODE="normal"
    mock::system::set_command "docker" "true"
    mock::docker::set_container_state "app-server" "exited" "node:18"
    
    instance::get_docker_containers
    instance::detect_all
    
    # Container is tracked but state should not be "docker"
    [[ "${DOCKER_INSTANCES[server_state]:-}" == "exited" ]]
    [[ "$INSTANCE_STATE" == "none" ]]
}

# ==============================================================================
# Native Process Detection Tests
# ==============================================================================

@test "instance::is_app_process - identifies app processes correctly" {
    # Load configuration first to initialize PROCESS_PATTERNS
    instance::load_config
    
    # Should match app processes
    run instance::is_app_process "node server.js" "${var_ROOT_DIR}/packages/server"
    [ "$status" -eq 0 ]
    
    run instance::is_app_process "npm run dev" ""
    [ "$status" -eq 0 ]
    
    run instance::is_app_process "pnpm dev" ""
    [ "$status" -eq 0 ]
    
    # Should not match shell scripts
    run instance::is_app_process "/bin/bash ./scripts/develop.sh" ""
    [ "$status" -eq 1 ]
    
    run instance::is_app_process "sh develop.sh" ""
    [ "$status" -eq 1 ]
}

@test "instance::get_native_processes - detects processes on ports" {
    # Load configuration first to initialize SERVICE_PORTS
    instance::load_config
    
    # Mock lsof to return PIDs for ports
    mock::system::add_port_listener "5329" "12345"
    mock::system::add_port_listener "3000" "12346"
    
    # Mock the processes to exist
    mock::system::set_process_state "12345" "node" "running" "1" "node server.js" "testuser" "0.1" "0.2"
    mock::system::set_process_state "12346" "vite" "running" "1" "vite" "testuser" "0.2" "0.3"
    
    instance::get_native_processes
    
    # Check processes were detected
    [[ "${NATIVE_INSTANCES[server_pid]:-}" == "12345" ]]
    [[ "${NATIVE_INSTANCES[ui_pid]:-}" == "12346" ]]
}

# ==============================================================================
# Instance State Detection Tests
# ==============================================================================

@test "instance::detect_all - detects no instances" {
    # Skip Docker detection by disabling instance management temporarily
    cat > "$TEST_SERVICE_JSON" <<'EOF'
{
  "instanceManagement": {
    "enabled": false
  }
}
EOF
    
    instance::detect_all
    
    [[ "$INSTANCE_STATE" == "none" ]]
}

@test "instance::detect_all - detects docker instances" {
    export DOCKER_MOCK_MODE="normal"
    mock::system::set_command "docker" "true"
    mock::docker::set_container_state "app-server" "running" "node:18"
    
    instance::detect_all
    
    [[ "$INSTANCE_STATE" == "docker" ]]
    [[ -n "${DOCKER_INSTANCES[server_container]:-}" ]]
}

@test "instance::detect_all - detects native instances" {
    # Mock a process listening on server port
    mock::system::add_port_listener "5329" "12345"
    mock::system::set_process_state "12345" "node" "running" "1" "node server.js" "testuser" "0.1" "0.2"
    
    instance::detect_all
    
    [[ "$INSTANCE_STATE" == "native" ]]
    [[ "${NATIVE_INSTANCES[server_pid]:-}" == "12345" ]]
}

@test "instance::detect_all - detects mixed instances" {
    export DOCKER_MOCK_MODE="normal"
    mock::system::set_command "docker" "true"
    mock::docker::set_container_state "app-server" "running" "node:18"
    
    # Mock a native process too
    mock::system::add_port_listener "3000" "12345"
    mock::system::set_process_state "12345" "vite" "running" "1" "vite" "testuser" "0.1" "0.2"
    
    instance::detect_all
    
    [[ "$INSTANCE_STATE" == "mixed" ]]
    [[ -n "${DOCKER_INSTANCES[server_container]:-}" ]]
    [[ -n "${NATIVE_INSTANCES[ui_pid]:-}" ]]
}

@test "instance::detect_all - respects disabled instance management" {
    cat > "$TEST_SERVICE_JSON" <<'EOF'
{
  "instanceManagement": {
    "enabled": false
  }
}
EOF
    
    instance::detect_all
    
    # Should not detect anything when disabled
    [[ "$INSTANCE_STATE" == "none" ]]
}

# ==============================================================================
# Shutdown Tests
# ==============================================================================

@test "instance::shutdown_docker - stops Docker containers via compose" {
    # Set up a compose file
    local compose_file="${var_ROOT_DIR}/docker-compose.yml"
    mkdir -p "$(dirname "$compose_file")"
    echo 'version: "3"' > "$compose_file"
    
    export DOCKER_MOCK_MODE="normal"
    mock::system::set_command "docker" "true"
    
    run instance::shutdown_docker
    [ "$status" -eq 0 ]
    
    # Clean up
    rm -f "$compose_file"
}

@test "instance::shutdown_docker - stops containers individually as fallback" {
    # No compose file available, should try individual containers
    export DOCKER_MOCK_MODE="normal"
    mock::system::set_command "docker" "true"
    
    # Set up some containers to stop
    DOCKER_INSTANCES[server_container]="app-server"
    DOCKER_INSTANCES[ui_container]="app-ui"
    
    run instance::shutdown_docker
    [ "$status" -eq 0 ]
}

@test "instance::shutdown_native - stops native processes" {
    NATIVE_INSTANCES[server_pid]="12345"
    NATIVE_INSTANCES[ui_pid]="12346"
    
    # Mock processes to exist for kill checks
    mock::system::set_process_state "12345" "node" "running" "1" "node server.js" "testuser" "0.1" "0.2"
    mock::system::set_process_state "12346" "vite" "running" "1" "vite" "testuser" "0.2" "0.3"
    
    run instance::shutdown_native
    [ "$status" -eq 0 ]
}

@test "instance::shutdown_all - handles different states correctly" {
    # Test with no instances
    INSTANCE_STATE="none"
    run instance::shutdown_all
    [ "$status" -eq 0 ]
    
    # Test with docker only
    INSTANCE_STATE="docker"
    mock::docker::set_container_state "app-server" "running" "node:18"
    DOCKER_INSTANCES[server_container]="app-server"
    run instance::shutdown_all
    [ "$status" -eq 0 ]
    
    # Test with native only
    INSTANCE_STATE="native"
    NATIVE_INSTANCES[server_pid]="12345"
    run instance::shutdown_all
    [ "$status" -eq 0 ]
}

# ==============================================================================
# Conflict Handling Tests
# ==============================================================================

@test "instance::handle_conflicts - allows multiple instances when configured" {
    cat > "$TEST_SERVICE_JSON" <<'EOF'
{
  "instanceManagement": {
    "enabled": true,
    "strategy": "multiple"
  }
}
EOF
    
    INSTANCE_STATE="docker"
    
    run instance::handle_conflicts
    [ "$status" -eq 0 ]
}

@test "instance::handle_conflicts - auto-stops when configured" {
    cat > "$TEST_SERVICE_JSON" <<'EOF'
{
  "instanceManagement": {
    "enabled": true,
    "strategy": "single",
    "conflicts": {
      "action": "stop"
    }
  }
}
EOF
    
    # Manually set instance state to simulate detected instances
    INSTANCE_STATE="native"
    NATIVE_INSTANCES[server_pid]="12345"
    
    run instance::handle_conflicts
    [ "$status" -eq 0 ]
}

@test "instance::handle_conflicts - forces start when configured" {
    cat > "$TEST_SERVICE_JSON" <<'EOF'
{
  "instanceManagement": {
    "enabled": true,
    "strategy": "single",
    "conflicts": {
      "action": "force"
    }
  }
}
EOF
    
    # Manually set instance state to simulate detected instances
    INSTANCE_STATE="native"
    NATIVE_INSTANCES[server_pid]="12345"
    
    run instance::handle_conflicts
    [ "$status" -eq 0 ]
}

@test "instance::handle_conflicts - exits when user chooses keep" {
    INSTANCE_STATE="docker"
    
    run instance::handle_conflicts "" "keep_exit"
    [ "$status" -eq 2 ]  # Special exit code for user choice
}

# ==============================================================================
# Preference Management Tests
# ==============================================================================

@test "instance::get_preference_file - creates preference directory" {
    export var_APP_NAME="testapp"
    
    run instance::get_preference_file
    [ "$status" -eq 0 ]
    [[ "$output" == *"/.testapp/.instance_preferences" ]]
    [[ -d "${HOME}/.testapp" ]]
}

@test "instance::save_preference - saves user preference" {
    instance::save_preference "stop_all" 3600
    
    local pref_file
    pref_file=$(instance::get_preference_file)
    
    [[ -f "$pref_file" ]]
    grep -q '"default_action": "stop_all"' "$pref_file"
}

@test "instance::load_preference - loads valid preference" {
    instance::save_preference "force" 86400
    
    run instance::load_preference
    [ "$status" -eq 0 ]
    [[ "$output" == "force" ]]
}

@test "instance::load_preference - ignores expired preference" {
    instance::save_preference "stop_all" 1
    sleep 2
    
    run instance::load_preference
    [ "$status" -eq 0 ]
    [[ -z "$output" ]]
}

@test "instance::clear_preferences - removes preference file" {
    instance::save_preference "stop_all" 3600
    local pref_file
    pref_file=$(instance::get_preference_file)
    
    [[ -f "$pref_file" ]]
    
    instance::clear_preferences
    
    [[ ! -f "$pref_file" ]]
}

# ==============================================================================
# Helper Function Tests
# ==============================================================================

@test "instance::should_skip_check - respects configuration" {
    # Test with disabled in config
    cat > "$TEST_SERVICE_JSON" <<'EOF'
{
  "instanceManagement": {
    "enabled": false
  }
}
EOF
    instance::load_config
    
    run instance::should_skip_check
    [ "$status" -eq 0 ]
    
    # Test with enabled in config
    cat > "$TEST_SERVICE_JSON" <<'EOF'
{
  "instanceManagement": {
    "enabled": true
  }
}
EOF
    instance::load_config
    
    run instance::should_skip_check
    [ "$status" -eq 1 ]
}

@test "instance::should_skip_check - respects environment variables" {
    export SKIP_INSTANCE_CHECK=yes
    run instance::should_skip_check
    [ "$status" -eq 0 ]
    
    export SKIP_INSTANCE_CHECK=no
    run instance::should_skip_check
    [ "$status" -eq 1 ]
    
    unset SKIP_INSTANCE_CHECK
}

@test "instance::should_skip_check - detects CI environment" {
    export CI=true
    run instance::should_skip_check
    [ "$status" -eq 0 ]
    
    unset CI
    export GITHUB_ACTIONS=true
    run instance::should_skip_check
    [ "$status" -eq 0 ]
    
    unset GITHUB_ACTIONS
}

@test "instance::get_summary - generates correct summaries" {
    # No instances
    run instance::get_summary
    [ "$status" -eq 0 ]
    [[ "$output" == "none" ]]
    
    # Docker only
    DOCKER_INSTANCES[server_container]="app-server"
    DOCKER_INSTANCES[server_state]="running"
    APP_SERVICES=("server")
    
    run instance::get_summary
    [ "$status" -eq 0 ]
    [[ "$output" == "1 Docker container" ]]
    
    # Multiple types
    DOCKER_INSTANCES[ui_container]="app-ui"
    DOCKER_INSTANCES[ui_state]="running"
    NATIVE_INSTANCES[database_pid]="12345"
    APP_SERVICES=("server" "ui" "database")
    
    run instance::get_summary
    [ "$status" -eq 0 ]
    [[ "$output" == *"2 Docker containers"* ]]
    [[ "$output" == *"1 service"* ]]
}

# ==============================================================================
# Integration Tests
# ==============================================================================

@test "instance manager - full detection and shutdown flow" {
    # This is a simplified version of a full flow test
    export DOCKER_MOCK_MODE="normal"
    mock::system::set_command "docker" "true"
    
    # Set up mixed environment
    mock::docker::set_container_state "app-server" "running" "node:18"
    mock::system::add_port_listener "3000" "12345"
    mock::system::set_process_state "12345" "vite" "running" "1" "vite" "testuser" "0.1" "0.2"
    
    # Detect instances
    instance::detect_all
    [[ "$INSTANCE_STATE" == "mixed" ]]
    
    # Test shutdown
    run instance::shutdown_all
    [ "$status" -eq 0 ]
}

@test "instance manager - respects multiple instance strategy" {
    cat > "$TEST_SERVICE_JSON" <<'EOF'
{
  "instanceManagement": {
    "enabled": true,
    "strategy": "multiple"
  }
}
EOF
    
    # Manually set instance state
    INSTANCE_STATE="native"
    
    # Should not prompt or stop when multiple allowed
    run instance::handle_conflicts
    [ "$status" -eq 0 ]
}

# ==============================================================================
# Edge Case and Error Handling Tests
# ==============================================================================

@test "instance::load_config - handles malformed service.json gracefully" {
    cat > "$TEST_SERVICE_JSON" <<'EOF'
{
  "instanceManagement": {
    "enabled": "not-a-boolean",
    "strategy": 123
  }
}
EOF
    
    # Should not crash, should use defaults for invalid values
    run instance::load_config
    [ "$status" -eq 0 ]
}

@test "instance::get_process_info - handles non-existent PID" {
    run instance::get_process_info "99999"
    [ "$status" -eq 0 ]
    # Should return empty strings for non-existent process
    [[ "$output" == "|" || "$output" == "||" ]]
}

@test "instance::is_app_process - rejects shell scripts" {
    instance::load_config
    
    # These should be rejected
    run instance::is_app_process "/bin/bash script.sh" ""
    [ "$status" -eq 1 ]
    
    run instance::is_app_process "sh develop.sh" ""
    [ "$status" -eq 1 ]
    
    run instance::is_app_process "bash -c 'node server'" ""
    [ "$status" -eq 1 ]
}

@test "instance::save_preference - requires action parameter" {
    run instance::save_preference ""
    [ "$status" -eq 1 ]
    
    run instance::save_preference
    [ "$status" -eq 1 ]
}

@test "instance::get_preference_file - handles directory creation failure" {
    # Make HOME read-only to simulate failure
    export HOME="/dev/null/cannot-create"
    
    run instance::get_preference_file
    [ "$status" -eq 1 ]
}

@test "instance::get_docker_containers - handles empty container output" {
    instance::load_config
    export DOCKER_MOCK_MODE="normal"
    mock::system::set_command "docker" "true"
    
    # Mock empty docker ps output
    # This is tested by the enhanced defensive check in the function
    run instance::get_docker_containers
    [ "$status" -eq 0 ]
}

@test "instance::get_native_processes - validates PID format" {
    instance::load_config
    
    # Mock system functions to return invalid PIDs mixed with valid ones
    mock::system::add_port_listener "5329" "12345"
    mock::system::add_port_listener "3000" "not-a-pid"
    mock::system::set_process_state "12345" "node" "running" "1" "node server.js" "testuser" "0.1" "0.2"
    
    run instance::get_native_processes
    [ "$status" -eq 0 ]
    
    # Should have detected the valid PID
    [[ "${NATIVE_INSTANCES[server_pid]:-}" == "12345" ]]
    # Should not have detected the invalid PID
    [[ -z "${NATIVE_INSTANCES[ui_pid]:-}" ]]
}