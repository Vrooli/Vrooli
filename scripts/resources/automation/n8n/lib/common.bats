#!/usr/bin/env bats
bats_require_minimum_version 1.5.0

# Path to the script under test
SCRIPT_PATH="$BATS_TEST_DIRNAME/common.sh"
N8N_DIR="$BATS_TEST_DIRNAME/.."

# Source dependencies in correct order (matching manage.sh)
RESOURCES_DIR="$N8N_DIR/../.."
HELPERS_DIR="$RESOURCES_DIR/../helpers"

# Source utilities first
. "$HELPERS_DIR/utils/log.sh"
. "$HELPERS_DIR/utils/system.sh"
. "$HELPERS_DIR/utils/ports.sh"
. "$HELPERS_DIR/utils/flow.sh"
. "$RESOURCES_DIR/port-registry.sh"

# Helper function for proper sourcing in tests
setup_n8n_test_env() {
    local script_dir="$N8N_DIR"
    local resources_dir="$N8N_DIR/../.."
    local helpers_dir="$resources_dir/../helpers"
    
    # Source utilities first
    source "$helpers_dir/utils/log.sh"
    source "$helpers_dir/utils/system.sh"
    source "$helpers_dir/utils/ports.sh"
    source "$helpers_dir/utils/flow.sh"
    source "$resources_dir/port-registry.sh"
    
    # Source common.sh (provides resources functions)
    source "$resources_dir/common.sh"
    
    # Source config (depends on resources functions)
    source "$script_dir/config/defaults.sh"
    
    # Source the script under test
    source "$SCRIPT_PATH"
    
    # Mock functions for tests
    resources::add_rollback_action() { 
        echo "Rollback action added: $1" >&2
        return 0
    }
}

# Helper function for directory tests (avoids readonly variable conflicts)
setup_n8n_test_env_no_defaults() {
    local script_dir="$N8N_DIR"
    local resources_dir="$N8N_DIR/../.."
    local helpers_dir="$resources_dir/../helpers"
    
    # Source utilities first
    source "$helpers_dir/utils/log.sh"
    source "$helpers_dir/utils/system.sh"
    source "$helpers_dir/utils/ports.sh"
    source "$helpers_dir/utils/flow.sh"
    source "$resources_dir/port-registry.sh"
    
    # Source common.sh (provides resources functions)
    source "$resources_dir/common.sh"
    
    # Source the script under test
    source "$SCRIPT_PATH"
    
    # Mock functions for tests
    resources::add_rollback_action() { 
        echo "Rollback action added: $1" >&2
        return 0
    }
}

# ============================================================================
# Function Definition Tests
# ============================================================================

@test "sourcing common.sh defines required functions" {
    setup_n8n_test_env
    
    # Check if functions are defined
    declare -f n8n::check_docker
    declare -f n8n::container_exists
    declare -f n8n::is_running
    declare -f n8n::is_healthy
    declare -f n8n::generate_password
    declare -f n8n::create_directories
    declare -f n8n::create_network
    declare -f n8n::is_port_available
    declare -f n8n::wait_for_ready
    declare -f n8n::validate_workflow_id
    declare -f n8n::check_requirements
}

# ============================================================================
# Password Generation Tests
# ============================================================================

@test "n8n::generate_password creates password of reasonable length" {
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
    source "$SCRIPT_PATH"
    
    # Test password generation
    local password
    password=$(n8n::generate_password)
    local length=${#password}
    
    # Verify password meets requirements
    [[ -n "$password" ]]
    [[ $length -ge 8 ]]
}

@test "n8n::generate_password generates non-empty password" {
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
    source "$SCRIPT_PATH"
    
    # Test password generation
    local password
    password=$(n8n::generate_password)
    
    # Verify password is not empty
    [[ -n "$password" ]]
}

# ============================================================================
# Directory Creation Tests
# ============================================================================

@test "n8n::create_directories creates data directory successfully" {
    setup_n8n_test_env_no_defaults
    
    # Set N8N_DATA_DIR to use test directory (avoiding readonly variable)
    N8N_DATA_DIR="$BATS_TEST_TMPDIR/n8n_data"
    
    # Test directory creation
    n8n::create_directories
    
    # Verify directory was created
    [[ -d "$N8N_DATA_DIR" ]]
}

# ============================================================================
# Workflow ID Validation Tests
# ============================================================================

@test "n8n::validate_workflow_id accepts valid numeric ID" {
    setup_n8n_test_env
    
    # Test valid numeric workflow ID
    n8n::validate_workflow_id '123'
}

@test "n8n::validate_workflow_id accepts single digit ID" {
    setup_n8n_test_env
    
    # Test valid single digit workflow ID
    n8n::validate_workflow_id '1'
}

@test "n8n::validate_workflow_id rejects non-numeric ID" {
    run bash -c "
        # Source in correct order (matching manage.sh)
        SCRIPT_DIR='$N8N_DIR'
        RESOURCES_DIR='$N8N_DIR/../..'
        HELPERS_DIR=\"\$RESOURCES_DIR/../helpers\"
        
        # Source utilities first
        source \"\$HELPERS_DIR/utils/log.sh\"
        source \"\$HELPERS_DIR/utils/system.sh\"
        source \"\$HELPERS_DIR/utils/ports.sh\"
        source \"\$HELPERS_DIR/utils/flow.sh\"
        source \"\$RESOURCES_DIR/port-registry.sh\"
        
        # Source common.sh (provides resources functions)
        source \"\$RESOURCES_DIR/common.sh\"
        
        # Source config (depends on resources functions)
        source '\$SCRIPT_DIR/config/defaults.sh'
        
        # Source the script under test
        source '$SCRIPT_PATH'
        
        n8n::validate_workflow_id 'abc' 2>/dev/null
    "
    [ "$status" -eq 1 ]
}

@test "n8n::validate_workflow_id rejects empty ID" {
    run bash -c "
        # Source in correct order (matching manage.sh)
        SCRIPT_DIR='$N8N_DIR'
        RESOURCES_DIR='$N8N_DIR/../..'
        HELPERS_DIR=\"\$RESOURCES_DIR/../helpers\"
        
        # Source utilities first
        source \"\$HELPERS_DIR/utils/log.sh\"
        source \"\$HELPERS_DIR/utils/system.sh\"
        source \"\$HELPERS_DIR/utils/ports.sh\"
        source \"\$HELPERS_DIR/utils/flow.sh\"
        source \"\$RESOURCES_DIR/port-registry.sh\"
        
        # Source common.sh (provides resources functions)
        source \"\$RESOURCES_DIR/common.sh\"
        
        # Source config (depends on resources functions)
        source '\$SCRIPT_DIR/config/defaults.sh'
        
        # Source the script under test
        source '$SCRIPT_PATH'
        
        n8n::validate_workflow_id '' 2>/dev/null
    "
    [ "$status" -eq 1 ]
}

@test "n8n::validate_workflow_id rejects alphanumeric ID" {
    run bash -c "
        # Source in correct order (matching manage.sh)
        SCRIPT_DIR='$N8N_DIR'
        RESOURCES_DIR='$N8N_DIR/../..'
        HELPERS_DIR=\"\$RESOURCES_DIR/../helpers\"
        
        # Source utilities first
        source \"\$HELPERS_DIR/utils/log.sh\"
        source \"\$HELPERS_DIR/utils/system.sh\"
        source \"\$HELPERS_DIR/utils/ports.sh\"
        source \"\$HELPERS_DIR/utils/flow.sh\"
        source \"\$RESOURCES_DIR/port-registry.sh\"
        
        # Source common.sh (provides resources functions)
        source \"\$RESOURCES_DIR/common.sh\"
        
        # Source config (depends on resources functions)
        source '\$SCRIPT_DIR/config/defaults.sh'
        
        # Source the script under test
        source '$SCRIPT_PATH'
        
        n8n::validate_workflow_id 'abc123' 2>/dev/null
    "
    [ "$status" -eq 1 ]
}

# ============================================================================
# Container Management Tests (Simple)
# ============================================================================

@test "n8n::container_exists function is properly defined" {
    run bash -c "
        RESOURCES_DIR='$N8N_DIR/../..'
        source \"\$RESOURCES_DIR/common.sh\" 2>/dev/null || true
        source '$N8N_DIR/config/defaults.sh' 2>/dev/null || true
        source '$SCRIPT_PATH'
        
        # Just verify the function is defined and has correct structure
        declare -f n8n::container_exists | grep -q 'docker ps -a'
    "
    [ "$status" -eq 0 ]
}

@test "n8n::is_running function is properly defined" {
    run bash -c "
        RESOURCES_DIR='$N8N_DIR/../..'
        source \"\$RESOURCES_DIR/common.sh\" 2>/dev/null || true
        source '$N8N_DIR/config/defaults.sh' 2>/dev/null || true
        source '$SCRIPT_PATH'
        
        # Just verify the function is defined and has correct structure
        declare -f n8n::is_running | grep -q 'docker ps'
    "
    [ "$status" -eq 0 ]
}

@test "n8n::is_healthy function is properly defined" {
    run bash -c "
        RESOURCES_DIR='$N8N_DIR/../..'
        source \"\$RESOURCES_DIR/common.sh\" 2>/dev/null || true
        source '$N8N_DIR/config/defaults.sh' 2>/dev/null || true
        source '$SCRIPT_PATH'
        
        # Just verify the function is defined and has correct structure
        declare -f n8n::is_healthy | grep -q '/healthz'
    "
    [ "$status" -eq 0 ]
}

# ============================================================================
# Port Availability Tests (Simple)
# ============================================================================

@test "n8n::is_port_available function is properly defined with lsof check" {
    run bash -c "
        RESOURCES_DIR='$N8N_DIR/../..'
        source \"\$RESOURCES_DIR/common.sh\" 2>/dev/null || true
        source '$N8N_DIR/config/defaults.sh' 2>/dev/null || true
        source '$SCRIPT_PATH'
        
        # Just verify the function is defined and has correct structure
        declare -f n8n::is_port_available | grep -q 'lsof'
    "
    [ "$status" -eq 0 ]
}

@test "n8n::is_port_available function includes netstat fallback" {
    run bash -c "
        RESOURCES_DIR='$N8N_DIR/../..'
        source \"\$RESOURCES_DIR/common.sh\" 2>/dev/null || true
        source '$N8N_DIR/config/defaults.sh' 2>/dev/null || true
        source '$SCRIPT_PATH'
        
        # Just verify the function is defined and has correct structure
        declare -f n8n::is_port_available | grep -q 'netstat'
    "
    [ "$status" -eq 0 ]
}

# ============================================================================
# Requirements Check Tests (Simple)
# ============================================================================

@test "n8n::check_requirements function checks for docker and curl" {
    run bash -c "
        RESOURCES_DIR='$N8N_DIR/../..'
        source \"\$RESOURCES_DIR/common.sh\" 2>/dev/null || true
        source '$N8N_DIR/config/defaults.sh' 2>/dev/null || true
        source '$SCRIPT_PATH'
        
        # Just verify the function is defined and has correct structure
        declare -f n8n::check_requirements | grep -q 'docker.*curl'
    "
    [ "$status" -eq 0 ]
}

@test "n8n::check_requirements function checks for optional commands" {
    run bash -c "
        RESOURCES_DIR='$N8N_DIR/../..'
        source \"\$RESOURCES_DIR/common.sh\" 2>/dev/null || true
        source '$N8N_DIR/config/defaults.sh' 2>/dev/null || true
        source '$SCRIPT_PATH'
        
        # Just verify the function is defined and has correct structure  
        declare -f n8n::check_requirements | grep -q 'jq.*openssl'
    "
    [ "$status" -eq 0 ]
}