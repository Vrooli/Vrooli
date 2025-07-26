#!/usr/bin/env bats
bats_require_minimum_version 1.5.0

# Path to the script under test
SCRIPT_PATH="$BATS_TEST_DIRNAME/status.sh"
N8N_DIR="$BATS_TEST_DIRNAME/.."

# Source dependencies
RESOURCES_DIR="$N8N_DIR/../.."
HELPERS_DIR="$RESOURCES_DIR/../helpers"

# Source required utilities 
. "$HELPERS_DIR/utils/log.sh"
. "$HELPERS_DIR/utils/system.sh"
. "$HELPERS_DIR/utils/ports.sh"
. "$HELPERS_DIR/utils/flow.sh"
. "$RESOURCES_DIR/port-registry.sh"

# Helper function to setup n8n test environment
setup_n8n_status_test() {
    # Source dependencies directly in test context
    SCRIPT_DIR="$N8N_DIR"  
    RESOURCES_DIR="$N8N_DIR/../.."
    HELPERS_DIR="$RESOURCES_DIR/../helpers"
    
    # Source in correct order
    source "$HELPERS_DIR/utils/log.sh"
    source "$HELPERS_DIR/utils/system.sh"
    source "$HELPERS_DIR/utils/ports.sh"
    source "$HELPERS_DIR/utils/flow.sh"
    source "$RESOURCES_DIR/port-registry.sh"
    source "$RESOURCES_DIR/common.sh"
    source "$SCRIPT_DIR/config/defaults.sh"
    source "$SCRIPT_DIR/lib/common.sh"
    source "$SCRIPT_PATH"
    
    # Set default variables that the function expects
    BASIC_AUTH="${BASIC_AUTH:-yes}"
    AUTH_USERNAME="${AUTH_USERNAME:-admin}"
    DATABASE_TYPE="${DATABASE_TYPE:-sqlite}"
    TUNNEL_ENABLED="${TUNNEL_ENABLED:-no}"
}

# ============================================================================
# Function Definition Tests
# ============================================================================

@test "sourcing status.sh defines required functions" {
    run bash -c "
        # Source dependencies first
        RESOURCES_DIR='$N8N_DIR/../..'
        source \"\$RESOURCES_DIR/common.sh\" 2>/dev/null || true
        source '$N8N_DIR/config/defaults.sh' 2>/dev/null || true
        source '$N8N_DIR/lib/common.sh' 2>/dev/null || true
        source '$SCRIPT_PATH' 2>/dev/null || true
        
        # Check if functions are defined
        declare -f n8n::status && 
        declare -f n8n::info && 
        declare -f n8n::inspect && 
        declare -f n8n::version && 
        declare -f n8n::stats && 
        declare -f n8n::check_all
    "
    [ "$status" -eq 0 ]
}

# ============================================================================
# Info Function Tests
# ============================================================================

@test "n8n::info displays comprehensive resource information" {
    setup_n8n_status_test
    
    # Test n8n::info output
    local output
    output=$(n8n::info)
    
    # Verify required content is present
    [[ "$output" =~ "n8n Resource Information" ]]
    [[ "$output" =~ "ID: n8n" ]]
    [[ "$output" =~ "Category: automation" ]]
    [[ "$output" =~ "Workflow automation platform" ]]
    [[ "$output" =~ "Container Name:" ]]
    [[ "$output" =~ "Service Port:" ]]
    [[ "$output" =~ "Endpoints:" ]]
    [[ "$output" =~ "Health Check:" ]]
    [[ "$output" =~ "Web UI:" ]]
    [[ "$output" =~ "REST API:" ]]
    [[ "$output" =~ "Webhooks:" ]]
    [[ "$output" =~ "Example Usage:" ]]
    [[ "$output" =~ "Visual workflow builder" ]]
    [[ "$output" =~ "400+ integrations" ]]
}

@test "n8n::info includes webhook URL when configured" {
    setup_n8n_status_test
    
    # Set custom webhook URL
    WEBHOOK_URL='https://example.com'
    
    # Test n8n::info output with webhook URL
    local output
    output=$(n8n::info)
    
    # Verify webhook URL is included
    [[ "$output" =~ "Webhook URL: https://example.com" ]]
    [[ "$output" =~ "https://example.com/webhook" ]]
}

@test "n8n::info shows configuration options" {
    setup_n8n_status_test
    
    # Set configuration options
    BASIC_AUTH='no'
    DATABASE_TYPE='postgres'
    TUNNEL_ENABLED='yes'
    
    # Test n8n::info output with custom configuration
    local output
    output=$(n8n::info)
    
    # Verify configuration is shown
    [[ "$output" =~ "Authentication: no" ]]
    [[ "$output" =~ "Database: postgres" ]]
    [[ "$output" =~ "Tunnel Mode: yes" ]]
}

# ============================================================================
# Function Structure Tests
# ============================================================================

@test "n8n::status function is properly defined" {
    run bash -c "
        RESOURCES_DIR='$N8N_DIR/../..'
        source \"\$RESOURCES_DIR/common.sh\" 2>/dev/null || true
        source '$N8N_DIR/config/defaults.sh' 2>/dev/null || true
        source '$N8N_DIR/lib/common.sh' 2>/dev/null || true
        source '$SCRIPT_PATH'
        
        # Just verify the function is defined and has correct structure
        declare -f n8n::status | grep -q 'n8n Status'
    "
    [ "$status" -eq 0 ]
}

@test "n8n::inspect function is properly defined" {
    run bash -c "
        RESOURCES_DIR='$N8N_DIR/../..'
        source \"\$RESOURCES_DIR/common.sh\" 2>/dev/null || true
        source '$N8N_DIR/config/defaults.sh' 2>/dev/null || true
        source '$N8N_DIR/lib/common.sh' 2>/dev/null || true
        source '$SCRIPT_PATH'
        
        # Just verify the function is defined and has correct structure
        declare -f n8n::inspect | grep -q 'Container Details'
    "
    [ "$status" -eq 0 ]
}

@test "n8n::version function is properly defined" {
    run bash -c "
        RESOURCES_DIR='$N8N_DIR/../..'
        source \"\$RESOURCES_DIR/common.sh\" 2>/dev/null || true
        source '$N8N_DIR/config/defaults.sh' 2>/dev/null || true
        source '$N8N_DIR/lib/common.sh' 2>/dev/null || true
        source '$SCRIPT_PATH'
        
        # Just verify the function is defined and has correct structure
        declare -f n8n::version | grep -q 'docker exec'
    "
    [ "$status" -eq 0 ]
}

@test "n8n::stats function is properly defined" {
    run bash -c "
        RESOURCES_DIR='$N8N_DIR/../..'
        source \"\$RESOURCES_DIR/common.sh\" 2>/dev/null || true
        source '$N8N_DIR/config/defaults.sh' 2>/dev/null || true
        source '$N8N_DIR/lib/common.sh' 2>/dev/null || true
        source '$SCRIPT_PATH'
        
        # Just verify the function is defined and has correct structure
        declare -f n8n::stats | grep -q 'Resource Usage'
    "
    [ "$status" -eq 0 ]
}

@test "n8n::check_all function is properly defined" {
    run bash -c "
        RESOURCES_DIR='$N8N_DIR/../..'
        source \"\$RESOURCES_DIR/common.sh\" 2>/dev/null || true
        source '$N8N_DIR/config/defaults.sh' 2>/dev/null || true
        source '$N8N_DIR/lib/common.sh' 2>/dev/null || true
        source '$SCRIPT_PATH'
        
        # Just verify the function is defined and has correct structure
        declare -f n8n::check_all | grep -q 'Health Check'
    "
    [ "$status" -eq 0 ]
}

# ============================================================================
# Configuration Display Tests  
# ============================================================================

@test "n8n::info displays all required sections" {
    setup_n8n_status_test
    
    # Test n8n::info output for required sections
    local output
    output=$(n8n::info)
    
    # Verify all required sections are present
    echo "$output" | grep -q 'Service Details:'
    echo "$output" | grep -q 'Endpoints:'
    echo "$output" | grep -q 'Configuration:'
    echo "$output" | grep -q 'n8n Features:'
    echo "$output" | grep -q 'Example Usage:'
}

@test "n8n::info includes health check endpoint" {
    setup_n8n_status_test
    
    # Test that n8n::info includes health check endpoint
    n8n::info | grep -q '/healthz'
}

@test "n8n::info includes webhook endpoints" {
    setup_n8n_status_test
    
    # Test that n8n::info includes webhook endpoints
    local output
    output=$(n8n::info)
    
    # Verify webhook endpoints are present
    echo "$output" | grep -q '/webhook'
    echo "$output" | grep -q '/webhook-test'
}

@test "n8n::info includes REST API endpoint" {
    setup_n8n_status_test
    
    # Test that n8n::info includes REST API endpoint
    n8n::info | grep -q '/rest'
}

@test "n8n::info includes example curl commands" {
    setup_n8n_status_test
    
    # Test that n8n::info includes curl examples
    n8n::info | grep -q 'curl'
}