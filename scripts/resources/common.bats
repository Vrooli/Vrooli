#!/usr/bin/env bats

# Test specific functions from common.sh that don't depend on readonly variables
# This avoids conflicts with the readonly VROOLI_CONFIG_DIR

# Source dependencies
RESOURCES_DIR="$BATS_TEST_DIRNAME"
HELPERS_DIR="$RESOURCES_DIR/../lib"

. "$HELPERS_DIR/utils/log.sh"
. "$HELPERS_DIR/utils/flow.sh"
. "$HELPERS_DIR/network/ports.sh"
. "$HELPERS_DIR/utils/system_commands.sh"
. "$RESOURCES_DIR/port_registry.sh"

# Source only the functions we need to test
# We'll source common.sh in a subshell to avoid readonly conflicts

# ============================================================================
# Port Management Tests
# ============================================================================

@test "resources::get_default_port returns correct port for ollama" {
    # Run in subshell to avoid readonly conflicts
    output=$(bash -c "source '$RESOURCES_DIR/common.sh' && resources::get_default_port 'ollama'")
    [ "$output" = "11434" ]
}

@test "resources::get_default_port returns correct port for browserless" {
    output=$(bash -c "source '$RESOURCES_DIR/common.sh' && resources::get_default_port 'browserless'")
    [ "$output" = "4110" ]
}

@test "resources::get_default_port returns correct port for n8n" {
    output=$(bash -c "source '$RESOURCES_DIR/common.sh' && resources::get_default_port 'n8n'")
    [ "$output" = "5678" ]
}

@test "resources::get_default_port returns 8080 fallback for unknown resource" {
    output=$(bash -c "source '$RESOURCES_DIR/common.sh' && resources::get_default_port 'unknown'")
    [ "$output" = "8080" ]
}

# ============================================================================
# Rollback Context Tests (in isolated environment)
# ============================================================================

@test "rollback context functions work correctly" {
    # Run test in a subshell to avoid variable conflicts
    output=$(bash -c '
        source "scripts/resources/port_registry.sh"
        source "scripts/resources/common.sh"
        
        # Start rollback context
        resources::start_rollback_context "test_op"
        echo "OPERATION_ID: $OPERATION_ID"
        echo "INITIAL_COUNT: ${#ROLLBACK_ACTIONS[@]}"
        
        # Add some actions
        resources::add_rollback_action "Action 1" "echo 1" 10
        resources::add_rollback_action "Action 2" "echo 2" 20
        echo "AFTER_ADD: ${#ROLLBACK_ACTIONS[@]}"
        
        # Execute rollback
        resources::execute_rollback
        echo "AFTER_ROLLBACK: ${#ROLLBACK_ACTIONS[@]}"
        echo "OPERATION_ID_AFTER: $OPERATION_ID"
    ' 2>&1)
    
    # Check output
    [[ "$output" =~ "OPERATION_ID: test_op_" ]]
    [[ "$output" =~ "INITIAL_COUNT: 0" ]]
    [[ "$output" =~ "AFTER_ADD: 2" ]]
    [[ "$output" =~ "Rollback: Action 2" ]]
    [[ "$output" =~ "Rollback: Action 1" ]]
    [[ "$output" =~ "successful" ]]
    [[ "$output" =~ "AFTER_ROLLBACK: 0" ]]
    [[ "$output" =~ "OPERATION_ID_AFTER:" ]]  # Should be empty after clearing
}

# ============================================================================
# Error Handling Tests
# ============================================================================

@test "resources::handle_error provides correct guidance" {
    # Source dependencies in test context
    . "$RESOURCES_DIR/port_registry.sh"
    . "$RESOURCES_DIR/common.sh"
    
    # Test network error
    output=$(resources::handle_error "Connection failed" "network" 2>&1)
    [[ "$output" =~ "Network Issue" ]]
    [[ "$output" =~ "Check your internet connection" ]]
    
    # Test permission error
    output=$(resources::handle_error "Access denied" "permission" 2>&1)
    [[ "$output" =~ "Permission Issue" ]]
    [[ "$output" =~ "sudo privileges" ]]
    
    # Test config error
    output=$(resources::handle_error "Invalid syntax" "config" 2>&1)
    [[ "$output" =~ "Configuration Issue" ]]
    [[ "$output" =~ "Validate configuration syntax" ]]
    
    # Test system error with custom remediation
    output=$(resources::handle_error "Port busy" "system" "Try port 8081" 2>&1)
    [[ "$output" =~ "System Issue" ]]
    [[ "$output" =~ "Try port 8081" ]]
}

# ============================================================================
# Integration Tests
# ============================================================================

@test "DEFAULT_PORTS populated correctly from RESOURCE_PORTS" {
    # Check in isolated environment
    output=$(bash -c "
        source '$RESOURCES_DIR/port_registry.sh'
        source '$RESOURCES_DIR/common.sh'
        echo \"OLLAMA:\${DEFAULT_PORTS[ollama]}\"
        echo \"BROWSERLESS:\${DEFAULT_PORTS[browserless]}\"
        echo \"N8N:\${DEFAULT_PORTS[n8n]}\"
    ")
    
    [[ "$output" =~ "OLLAMA:11434" ]]
    [[ "$output" =~ "BROWSERLESS:4110" ]]
    [[ "$output" =~ "N8N:5678" ]]
}

@test "resources::validate_port rejects conflicting ports" {
    # Test port validation in isolated script
    cat > "${BATS_TEST_TMPDIR}/test_validate_port.sh" << 'EOF'
#!/bin/bash
# Source all dependencies
source "$1/utils/log.sh" 2>/dev/null || true
source "$1/network/ports.sh"
source "$2/port_registry.sh"
source "$2/common.sh"

# Test validation of a Vrooli-conflicting port
if ! resources::validate_port "browserless" "3000" 2>&1 >/dev/null; then
    echo "CONFLICT_DETECTED"
fi
EOF
    chmod +x "${BATS_TEST_TMPDIR}/test_validate_port.sh"
    
    run "${BATS_TEST_TMPDIR}/test_validate_port.sh" "$HELPERS_DIR" "$RESOURCES_DIR"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "CONFLICT_DETECTED" ]]
}

@test "port registry and common.sh work together correctly" {
    # Integration test
    output=$(bash -c "
        source '$RESOURCES_DIR/port_registry.sh'
        source '$RESOURCES_DIR/common.sh'
        
        # Test that we can get ports
        port=\$(resources::get_default_port 'ollama')
        echo \"Got port: \$port\"
        
        # Test that port validation works
        if ports::validate_assignment \"\$port\" 'ollama' >/dev/null 2>&1; then
            echo \"Validation passed\"
        fi
        
        # Test that DEFAULT_PORTS matches RESOURCE_PORTS
        if [ \"\${DEFAULT_PORTS[ollama]}\" = \"\${RESOURCE_PORTS[ollama]}\" ]; then
            echo \"Arrays match\"
        fi
    ")
    
    [[ "$output" =~ "Got port: 11434" ]]
    [[ "$output" =~ "Validation passed" ]]
    [[ "$output" =~ "Arrays match" ]]
}