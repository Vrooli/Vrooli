#!/usr/bin/env bats

# Load Vrooli test infrastructure (REQUIRED)
source "${BATS_TEST_DIRNAME}/../../../../__test/fixtures/setup.bash"

# Expensive setup operations (run once per file)
setup_file() {
    # Use appropriate setup function
    vrooli_setup_service_test "n8n"
    
    # Export paths for use in setup()
    export SETUP_FILE_SCRIPT_PATH="$(dirname "${BATS_TEST_FILENAME}")/common.sh"
    export SETUP_FILE_N8N_DIR="$(dirname "$(dirname "${BATS_TEST_FILENAME}")")"
}

# Lightweight per-test setup
setup() {
    # Setup standard mocks
    vrooli_auto_setup
    
    # Use paths from setup_file
    SCRIPT_PATH="${SETUP_FILE_SCRIPT_PATH}"
    N8N_DIR="${SETUP_FILE_N8N_DIR}"
    
    # Set test environment BEFORE sourcing config files to avoid readonly conflicts
    export N8N_CUSTOM_PORT="5678"
    export YES="no"
    
    # Mock resources functions that are called during config loading
    resources::get_default_port() {
        case "$1" in
            "n8n") echo "5678" ;;
            *) echo "8080" ;;
        esac
    }
    
    # Now source the config and library files
    source "${N8N_DIR}/config/defaults.sh"
    source "${N8N_DIR}/config/messages.sh"
    source "$SCRIPT_PATH"
    
    # Export config and messages
    n8n::export_config
    n8n::export_messages
    
    # Mock functions for tests
    resources::add_rollback_action() { 
        echo "Rollback action added: $1" >&2
        return 0
    }
}

# Cleanup after each test
teardown() {
    vrooli_cleanup_test
}

# Helper function for proper sourcing in tests
setup_n8n_test_env() {
    # This function is now simplified as dependencies are loaded in setup_file
    # Just ensure config and messages are exported
    n8n::export_config
    n8n::export_messages
}

# Helper function for directory tests (avoids readonly variable conflicts)
setup_n8n_test_env_no_defaults() {
    # This function is now simplified as dependencies are loaded in setup_file
    # We can dynamically set N8N_DATA_DIR in individual tests
    :
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
    HELPERS_DIR="$RESOURCES_DIR/../lib"
    
    # Source in correct order
    source "$HELPERS_DIR/utils/log.sh"
    source "$HELPERS_DIR/utils/system_commands.sh"
    source "$HELPERS_DIR/network/ports.sh"
    source "$HELPERS_DIR/utils/flow.sh"
    source "$RESOURCES_DIR/port_registry.sh"
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
    HELPERS_DIR="$RESOURCES_DIR/../lib"
    
    # Source in correct order
    source "$HELPERS_DIR/utils/log.sh"
    source "$HELPERS_DIR/utils/system_commands.sh"
    source "$HELPERS_DIR/network/ports.sh"
    source "$HELPERS_DIR/utils/flow.sh"
    source "$RESOURCES_DIR/port_registry.sh"
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
        HELPERS_DIR=\"\$RESOURCES_DIR/../lib\"
        
        # Source utilities first
        source \"\$HELPERS_DIR/utils/log.sh\"
        source \"\$HELPERS_DIR/utils/system_commands.sh\"
        source \"\$HELPERS_DIR/network/ports.sh\"
        source \"\$HELPERS_DIR/utils/flow.sh\"
        source \"\$RESOURCES_DIR/port_registry.sh\"
        
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
        HELPERS_DIR=\"\$RESOURCES_DIR/../lib\"
        
        # Source utilities first
        source \"\$HELPERS_DIR/utils/log.sh\"
        source \"\$HELPERS_DIR/utils/system_commands.sh\"
        source \"\$HELPERS_DIR/network/ports.sh\"
        source \"\$HELPERS_DIR/utils/flow.sh\"
        source \"\$RESOURCES_DIR/port_registry.sh\"
        
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
        HELPERS_DIR=\"\$RESOURCES_DIR/../lib\"
        
        # Source utilities first
        source \"\$HELPERS_DIR/utils/log.sh\"
        source \"\$HELPERS_DIR/utils/system_commands.sh\"
        source \"\$HELPERS_DIR/network/ports.sh\"
        source \"\$HELPERS_DIR/utils/flow.sh\"
        source \"\$RESOURCES_DIR/port_registry.sh\"
        
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