#!/usr/bin/env bats
# Test suite for interface-validation.sh

setup() {
    # Get the directory of the test file
    TEST_DIR="$(cd "$(dirname "$BATS_TEST_FILENAME")" && pwd)"
    
    # Source trash module for safe test cleanup
    SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
    # shellcheck disable=SC1091
    source "${SCRIPT_DIR}/../../../../lib/utils/var.sh" 2>/dev/null || true
    # shellcheck disable=SC1091
    source "${var_LIB_SYSTEM_DIR}/trash.sh" 2>/dev/null || true
    
    # Source var.sh
    # shellcheck disable=SC1091
    source "$TEST_DIR/../../../../lib/utils/var.sh"
    
    # Source test helpers
    # shellcheck disable=SC1091
    source "$var_SCRIPTS_TEST_DIR/fixtures/setup.bash"
    # shellcheck disable=SC1091
    source "$var_SCRIPTS_TEST_DIR/fixtures/assertions.bash"
    
    # Source the library being tested
    # shellcheck disable=SC1091
    source "$TEST_DIR/interface-validation.sh"
    
    # Create a temporary test resource directory
    TEST_RESOURCE_DIR="${BATS_TMPDIR}/test-resource"
    mkdir -p "$TEST_RESOURCE_DIR/config"
    mkdir -p "$TEST_RESOURCE_DIR/lib"
}

teardown() {
    # Cleanup
    trash::safe_remove "$TEST_RESOURCE_DIR" --test-cleanup
}

@test "interface_validation::detect_resource_category - detects AI category" {
    run interface_validation::detect_resource_category "scripts/resources/ai/ollama"
    
    assert_success
    assert_output "ai"
}

@test "interface_validation::detect_resource_category - detects automation category" {
    run interface_validation::detect_resource_category "scripts/resources/automation/n8n"
    
    assert_success
    assert_output "automation"
}

@test "interface_validation::detect_resource_category - detects storage category" {
    run interface_validation::detect_resource_category "scripts/resources/storage/postgres"
    
    assert_success
    assert_output "storage"
}

@test "interface_validation::detect_resource_category - returns unknown for invalid path" {
    run interface_validation::detect_resource_category "/some/other/path"
    
    assert_success
    assert_output "unknown"
}

@test "interface_validation::validate_manage_script - validates existing executable" {
    # Create test manage.sh
    echo '#!/bin/bash' > "$TEST_RESOURCE_DIR/manage.sh"
    chmod +x "$TEST_RESOURCE_DIR/manage.sh"
    
    run interface_validation::validate_manage_script "$TEST_RESOURCE_DIR"
    
    assert_success
    assert_output --partial "PASS"
}

@test "interface_validation::validate_manage_script - fails for non-executable" {
    # Create non-executable manage.sh
    echo '#!/bin/bash' > "$TEST_RESOURCE_DIR/manage.sh"
    chmod -x "$TEST_RESOURCE_DIR/manage.sh"
    
    run interface_validation::validate_manage_script "$TEST_RESOURCE_DIR"
    
    assert_failure
    assert_output --partial "not executable"
}

@test "interface_validation::validate_manage_script - fails for missing script" {
    run interface_validation::validate_manage_script "$TEST_RESOURCE_DIR"
    
    assert_failure
    assert_output --partial "not found"
}

@test "interface_validation::extract_available_actions - extracts actions from manage.sh" {
    # Create manage.sh with case statement
    cat > "$TEST_RESOURCE_DIR/manage.sh" <<'EOF'
#!/bin/bash
case "$action" in
    "start")
        echo "Starting"
        ;;
    "stop")
        echo "Stopping"
        ;;
    "status")
        echo "Status"
        ;;
esac
EOF
    
    run interface_validation::extract_available_actions "$TEST_RESOURCE_DIR"
    
    assert_success
    assert_output --partial "start"
    assert_output --partial "stop"
    assert_output --partial "status"
}

@test "interface_validation::validate_required_functions - validates AI functions" {
    # Create manage.sh with required functions
    cat > "$TEST_RESOURCE_DIR/manage.sh" <<'EOF'
#!/bin/bash
case "$action" in
    "status") ;;
    "install") ;;
    "start") ;;
    "stop") ;;
    "restart") ;;
    "test") ;;
    "logs") ;;
esac
EOF
    
    run interface_validation::validate_required_functions "$TEST_RESOURCE_DIR" "ai"
    
    assert_success
    assert_output --partial "All required functions present"
}

@test "interface_validation::validate_required_functions - detects missing functions" {
    # Create manage.sh with only some functions
    cat > "$TEST_RESOURCE_DIR/manage.sh" <<'EOF'
#!/bin/bash
case "$action" in
    "start") ;;
    "stop") ;;
esac
EOF
    
    run interface_validation::validate_required_functions "$TEST_RESOURCE_DIR" "ai"
    
    assert_failure
    assert_output --partial "Missing required functions"
}

@test "interface_validation::validate_function_help - validates help output" {
    # Create manage.sh with help
    cat > "$TEST_RESOURCE_DIR/manage.sh" <<'EOF'
#!/bin/bash
if [[ "$1" == "--help" ]]; then
    echo "Usage: manage.sh [action]"
    echo "Actions: start, stop, status"
    exit 0
fi
EOF
    chmod +x "$TEST_RESOURCE_DIR/manage.sh"
    
    run interface_validation::validate_function_help "$TEST_RESOURCE_DIR"
    
    assert_success
    assert_output --partial "well-formatted"
}

@test "interface_validation::validate_config_structure - validates config files" {
    # Create required config files
    echo "# Defaults" > "$TEST_RESOURCE_DIR/config/defaults.sh"
    echo "# Messages" > "$TEST_RESOURCE_DIR/config/messages.sh"
    
    run interface_validation::validate_config_structure "$TEST_RESOURCE_DIR"
    
    assert_success
    assert_output --partial "All required config files present"
}

@test "interface_validation::validate_config_structure - detects missing config" {
    # Only create one config file
    echo "# Defaults" > "$TEST_RESOURCE_DIR/config/defaults.sh"
    
    run interface_validation::validate_config_structure "$TEST_RESOURCE_DIR"
    
    assert_failure
    assert_output --partial "Missing config files"
    assert_output --partial "messages.sh"
}

@test "interface_validation::validate_error_handling - validates error patterns" {
    # Create manage.sh with error handling
    cat > "$TEST_RESOURCE_DIR/manage.sh" <<'EOF'
#!/bin/bash
set -euo pipefail

trap cleanup EXIT

log_error() {
    echo "ERROR: $1" >&2
    return 1
}

exit 1
EOF
    
    run interface_validation::validate_error_handling "$TEST_RESOURCE_DIR"
    
    assert_success
    assert_output --partial "Adequate error handling"
}

@test "interface_validation::validate_port_registry - checks port registration" {
    # Create mock port registry
    mkdir -p "$(dirname "$var_PORT_REGISTRY_FILE")"
    cat > "$var_PORT_REGISTRY_FILE" <<'EOF'
declare -A PORTS=(
    ["test-resource"]="8080"
    ["other-resource"]="8081"
)
EOF
    
    run interface_validation::validate_port_registry "$TEST_RESOURCE_DIR"
    
    # Will warn since our test resource isn't in registry
    assert_failure
    assert_output --partial "not found in port registry"
    
    # Cleanup
    trash::safe_remove "$var_PORT_REGISTRY_FILE" --test-cleanup
}

@test "interface_validation::validate_resource_interface - runs full validation" {
    # Create a minimal valid resource
    cat > "$TEST_RESOURCE_DIR/manage.sh" <<'EOF'
#!/bin/bash
set -euo pipefail

case "$1" in
    --help)
        echo "Usage: manage.sh [action]"
        exit 0
        ;;
esac

case "$action" in
    "status") echo "Status" ;;
    "install") echo "Install" ;;
    "start") echo "Start" ;;
    "stop") echo "Stop" ;;
    "restart") echo "Restart" ;;
    "test") echo "Test" ;;
    "logs") echo "Logs" ;;
esac
EOF
    chmod +x "$TEST_RESOURCE_DIR/manage.sh"
    
    echo "# Defaults" > "$TEST_RESOURCE_DIR/config/defaults.sh"
    echo "# Messages" > "$TEST_RESOURCE_DIR/config/messages.sh"
    
    # Mock the validation functions to simplify testing
    function interface_validation::validate_manage_script() { echo "PASS"; return 0; }
    function interface_validation::validate_required_functions() { echo "PASS"; return 0; }
    function interface_validation::validate_function_help() { echo "PASS"; return 0; }
    function interface_validation::validate_config_structure() { echo "PASS"; return 0; }
    function interface_validation::validate_error_handling() { echo "PASS"; return 0; }
    function interface_validation::validate_port_registry() { echo "SKIP"; return 2; }
    
    export -f interface_validation::validate_manage_script
    export -f interface_validation::validate_required_functions
    export -f interface_validation::validate_function_help
    export -f interface_validation::validate_config_structure
    export -f interface_validation::validate_error_handling
    export -f interface_validation::validate_port_registry
    
    run interface_validation::validate_resource_interface "$TEST_RESOURCE_DIR"
    
    assert_success
    assert_output --partial "interface validation PASSED"
    
    # Cleanup
    unset -f interface_validation::validate_manage_script
    unset -f interface_validation::validate_required_functions
    unset -f interface_validation::validate_function_help
    unset -f interface_validation::validate_config_structure
    unset -f interface_validation::validate_error_handling
    unset -f interface_validation::validate_port_registry
}