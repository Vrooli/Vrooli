#!/usr/bin/env bats

# Path to the script under test
SCRIPT_PATH="$BATS_TEST_DIRNAME/docker.sh"
N8N_DIR="$BATS_TEST_DIRNAME/.."

# Source dependencies
RESOURCES_DIR="$N8N_DIR/../.."
HELPERS_DIR="$RESOURCES_DIR/../lib"

# Source required utilities 
. "$HELPERS_DIR/utils/log.sh"
. "$HELPERS_DIR/utils/system_commands.sh"
. "$HELPERS_DIR/network/ports.sh"
. "$HELPERS_DIR/utils/flow.sh"
. "$RESOURCES_DIR/port-registry.sh"

# Helper function to setup n8n test environment
setup_n8n_docker_test() {
    # Source dependencies directly in test context
    SCRIPT_DIR="$N8N_DIR"  
    RESOURCES_DIR="$N8N_DIR/../.."
    HELPERS_DIR="$RESOURCES_DIR/../lib"
    
    # Source in correct order
    source "$HELPERS_DIR/utils/log.sh"
    source "$HELPERS_DIR/utils/system_commands.sh"
    source "$HELPERS_DIR/network/ports.sh"
    source "$HELPERS_DIR/utils/flow.sh"
    source "$RESOURCES_DIR/port-registry.sh"
    source "$RESOURCES_DIR/common.sh"
    source "$SCRIPT_DIR/config/defaults.sh"
    source "$SCRIPT_DIR/lib/common.sh"
    source "$SCRIPT_PATH"
    
    # Mock timedatectl
    timedatectl() { echo 'UTC'; }
    
    # Set default variables that the function expects
    BASIC_AUTH="${BASIC_AUTH:-no}"
    AUTH_USERNAME="${AUTH_USERNAME:-admin}"
    DATABASE_TYPE="${DATABASE_TYPE:-sqlite}"
    TUNNEL_ENABLED="${TUNNEL_ENABLED:-no}"
}

# ============================================================================
# Function Definition Tests
# ============================================================================

@test "sourcing docker.sh defines required functions" {
    run bash -c "
        # Source dependencies first
        RESOURCES_DIR='$N8N_DIR/../..'
        source \"\$RESOURCES_DIR/common.sh\" 2>/dev/null || true
        source '$N8N_DIR/config/defaults.sh' 2>/dev/null || true
        source '$N8N_DIR/lib/common.sh' 2>/dev/null || true
        source '$SCRIPT_PATH' 2>/dev/null || true
        
        # Check if functions are defined
        declare -f n8n::build_custom_image && 
        declare -f n8n::build_docker_command && 
        declare -f n8n::start_container && 
        declare -f n8n::stop && 
        declare -f n8n::start && 
        declare -f n8n::restart &&
        declare -f n8n::logs &&
        declare -f n8n::remove_container &&
        declare -f n8n::remove_postgres_container &&
        declare -f n8n::remove_network &&
        declare -f n8n::pull_image
    "
    [ "$status" -eq 0 ]
}

# ============================================================================
# Docker Command Building Tests
# ============================================================================

@test "n8n::build_docker_command includes required base parameters" {
    setup_n8n_docker_test
    
    # Test docker command building
    local command
    command=$(n8n::build_docker_command '' '')
    
    # Verify required parameters are present
    [[ "$command" =~ "docker run -d" ]]
    [[ "$command" =~ "--name n8n" ]]
    [[ "$command" =~ "--network n8n-network" ]]
    [[ "$command" =~ "--restart unless-stopped" ]]
}

@test "n8n::build_docker_command includes webhook URL when provided" {
    setup_n8n_docker_test
    
    # Test docker command building with webhook URL
    local command
    command=$(n8n::build_docker_command 'https://example.com' '')
    
    # Verify webhook configuration is present
    [[ "$command" =~ "WEBHOOK_URL=https://example.com" ]]
    [[ "$command" =~ "N8N_PROTOCOL=https" ]]
    [[ "$command" =~ "N8N_HOST=example.com" ]]
}

@test "n8n::build_docker_command includes basic auth when enabled" {
    setup_n8n_docker_test
    
    # Override auth settings
    BASIC_AUTH='yes'
    AUTH_USERNAME='testuser'
    
    # Test docker command building with auth
    local command
    command=$(n8n::build_docker_command '' 'testpass')
    
    # Verify auth configuration is present
    [[ "$command" =~ "N8N_BASIC_AUTH_ACTIVE=true" ]]
    [[ "$command" =~ "N8N_BASIC_AUTH_USER=testuser" ]]
    [[ "$command" =~ "N8N_BASIC_AUTH_PASSWORD=testpass" ]]
}

@test "n8n::build_docker_command configures postgres database" {
    setup_n8n_docker_test
    
    # Override database setting
    DATABASE_TYPE='postgres'
    
    # Test docker command building with postgres
    local command
    command=$(n8n::build_docker_command '' '')
    
    # Verify postgres configuration is present
    [[ "$command" =~ "DB_TYPE=postgresdb" ]]
    [[ "$command" =~ "DB_POSTGRESDB_HOST=n8n-postgres" ]]
    [[ "$command" =~ "DB_POSTGRESDB_DATABASE=n8n" ]]
    [[ "$command" =~ "DB_POSTGRESDB_USER=n8n" ]]
}

@test "n8n::build_docker_command configures sqlite database by default" {
    setup_n8n_docker_test
    
    # DATABASE_TYPE defaults to sqlite
    DATABASE_TYPE='sqlite'
    
    # Test docker command building with sqlite
    local command
    command=$(n8n::build_docker_command '' '')
    
    # Verify sqlite configuration is present
    [[ "$command" =~ "DB_TYPE=sqlite" ]]
    [[ "$command" =~ "DB_SQLITE_VACUUM_ON_STARTUP=true" ]]
}

@test "n8n::build_docker_command includes tunnel flag when enabled" {
    setup_n8n_docker_test
    
    # Override tunnel setting
    TUNNEL_ENABLED='yes'
    
    # Test docker command building with tunnel
    local command
    command=$(n8n::build_docker_command '' '')
    
    # Verify tunnel flag is present
    [[ "$command" =~ "n8n start --tunnel" ]]
}

# ============================================================================
# Function Structure Tests
# ============================================================================

@test "n8n::start_container function is properly defined" {
    run bash -c "
        RESOURCES_DIR='$N8N_DIR/../..'
        source \"\$RESOURCES_DIR/common.sh\" 2>/dev/null || true
        source '$N8N_DIR/config/defaults.sh' 2>/dev/null || true
        source '$N8N_DIR/lib/common.sh' 2>/dev/null || true
        source '$SCRIPT_PATH'
        
        # Just verify the function is defined and has correct structure
        declare -f n8n::start_container | grep -q 'docker_cmd'
    "
    [ "$status" -eq 0 ]
}

@test "n8n::stop function is properly defined" {
    run bash -c "
        RESOURCES_DIR='$N8N_DIR/../..'
        source \"\$RESOURCES_DIR/common.sh\" 2>/dev/null || true
        source '$N8N_DIR/config/defaults.sh' 2>/dev/null || true
        source '$N8N_DIR/lib/common.sh' 2>/dev/null || true
        source '$SCRIPT_PATH'
        
        # Just verify the function is defined and has correct structure
        declare -f n8n::stop | grep -q 'docker stop'
    "
    [ "$status" -eq 0 ]
}

@test "n8n::start function is properly defined" {
    run bash -c "
        RESOURCES_DIR='$N8N_DIR/../..'
        source \"\$RESOURCES_DIR/common.sh\" 2>/dev/null || true
        source '$N8N_DIR/config/defaults.sh' 2>/dev/null || true
        source '$N8N_DIR/lib/common.sh' 2>/dev/null || true
        source '$SCRIPT_PATH'
        
        # Just verify the function is defined and has correct structure
        declare -f n8n::start | grep -q 'docker start'
    "
    [ "$status" -eq 0 ]
}

@test "n8n::restart function is properly defined" {
    run bash -c "
        RESOURCES_DIR='$N8N_DIR/../..'
        source \"\$RESOURCES_DIR/common.sh\" 2>/dev/null || true
        source '$N8N_DIR/config/defaults.sh' 2>/dev/null || true
        source '$N8N_DIR/lib/common.sh' 2>/dev/null || true
        source '$SCRIPT_PATH'
        
        # Just verify the function is defined and calls stop and start
        declare -f n8n::restart | grep -q 'n8n::stop' && declare -f n8n::restart | grep -q 'n8n::start'
    "
    [ "$status" -eq 0 ]
}

@test "n8n::logs function is properly defined" {
    run bash -c "
        RESOURCES_DIR='$N8N_DIR/../..'
        source \"\$RESOURCES_DIR/common.sh\" 2>/dev/null || true
        source '$N8N_DIR/config/defaults.sh' 2>/dev/null || true
        source '$N8N_DIR/lib/common.sh' 2>/dev/null || true
        source '$SCRIPT_PATH'
        
        # Just verify the function is defined and has correct structure
        declare -f n8n::logs | grep -q 'docker logs'
    "
    [ "$status" -eq 0 ]
}

@test "n8n::remove_container function is properly defined" {
    run bash -c "
        RESOURCES_DIR='$N8N_DIR/../..'
        source \"\$RESOURCES_DIR/common.sh\" 2>/dev/null || true
        source '$N8N_DIR/config/defaults.sh' 2>/dev/null || true
        source '$N8N_DIR/lib/common.sh' 2>/dev/null || true
        source '$SCRIPT_PATH'
        
        # Just verify the function is defined and has correct structure
        declare -f n8n::remove_container | grep -q 'docker rm'
    "
    [ "$status" -eq 0 ]
}

@test "n8n::pull_image function is properly defined" {
    run bash -c "
        RESOURCES_DIR='$N8N_DIR/../..'
        source \"\$RESOURCES_DIR/common.sh\" 2>/dev/null || true
        source '$N8N_DIR/config/defaults.sh' 2>/dev/null || true
        source '$N8N_DIR/lib/common.sh' 2>/dev/null || true
        source '$SCRIPT_PATH'
        
        # Just verify the function is defined and has correct structure
        declare -f n8n::pull_image | grep -q 'docker pull'
    "
    [ "$status" -eq 0 ]
}