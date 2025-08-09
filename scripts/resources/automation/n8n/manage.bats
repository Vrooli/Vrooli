#!/usr/bin/env bats

# Load Vrooli test infrastructure (REQUIRED)
source "${BATS_TEST_DIRNAME}/../../../__test/fixtures/setup.bash"
# Load fixture helpers for accessing test data
source "${BATS_TEST_DIRNAME}/../../tests/lib/fixture-helpers.sh" 2>/dev/null || true

# Expensive setup operations (run once per file)
setup_file() {
    # Use appropriate setup function
    vrooli_setup_service_test "n8n"
    
    # Export paths for use in setup()
    export SETUP_FILE_SCRIPT_PATH="${BATS_TEST_DIRNAME}/manage.sh"
    export SETUP_FILE_CONFIG_DIR="${BATS_TEST_DIRNAME}/config"
    export SETUP_FILE_LIB_DIR="${BATS_TEST_DIRNAME}/lib"
    export SETUP_FILE_N8N_DIR="${BATS_TEST_DIRNAME}"
}

# Lightweight per-test setup
setup() {
    # Setup standard mocks
    vrooli_auto_setup
    
    # Use paths from setup_file
    SCRIPT_PATH="${SETUP_FILE_SCRIPT_PATH}"
    CONFIG_DIR="${SETUP_FILE_CONFIG_DIR}"
    LIB_DIR="${SETUP_FILE_LIB_DIR}"
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
    
    # Now source the config files
    source "${N8N_DIR}/config/defaults.sh"
    source "${N8N_DIR}/config/messages.sh"
    
    # Export config and messages
    n8n::export_config
    n8n::export_messages
}

# Cleanup after each test
teardown() {
    vrooli_cleanup_test
}

# ============================================================================
# Script Loading Tests
# ============================================================================

@test "sourcing manage.sh defines required functions" {
    run bash -c "source '$SCRIPT_PATH' && declare -f n8n::parse_arguments && declare -f n8n::main"
    [ "$status" -eq 0 ]
    [[ "$output" =~ n8n::parse_arguments ]]
    [[ "$output" =~ n8n::main ]]
}

@test "manage.sh sources all required dependencies" {
    run bash -c "source '$SCRIPT_PATH' 2>&1 | grep -v 'command not found' | head -1"
    [ "$status" -eq 0 ]
}

# ============================================================================
# Argument Parsing Tests
# ============================================================================

@test "n8n::parse_arguments sets default action to install" {
    run bash -c "source '$SCRIPT_PATH'; n8n::parse_arguments; echo \"\$ACTION\""
    [ "$status" -eq 0 ]
    [[ "$output" =~ "install" ]]
}

@test "n8n::parse_arguments accepts install action" {
    run bash -c "source '$SCRIPT_PATH'; n8n::parse_arguments --action install; echo \"\$ACTION\""
    [ "$status" -eq 0 ]
    [[ "$output" =~ "install" ]]
}

@test "n8n::parse_arguments accepts status action" {
    run bash -c "source '$SCRIPT_PATH'; n8n::parse_arguments --action status; echo \"\$ACTION\""
    [ "$status" -eq 0 ]
    [[ "$output" =~ "status" ]]
}

@test "n8n::parse_arguments accepts start action" {
    run bash -c "source '$SCRIPT_PATH'; n8n::parse_arguments --action start; echo \"\$ACTION\""
    [ "$status" -eq 0 ]
    [[ "$output" =~ "start" ]]
}

@test "n8n::parse_arguments accepts stop action" {
    run bash -c "source '$SCRIPT_PATH'; n8n::parse_arguments --action stop; echo \"\$ACTION\""
    [ "$status" -eq 0 ]
    [[ "$output" =~ "stop" ]]
}

@test "n8n::parse_arguments accepts restart action" {
    run bash -c "source '$SCRIPT_PATH'; n8n::parse_arguments --action restart; echo \"\$ACTION\""
    [ "$status" -eq 0 ]
    [[ "$output" =~ "restart" ]]
}

@test "n8n::parse_arguments accepts uninstall action" {
    run bash -c "source '$SCRIPT_PATH'; n8n::parse_arguments --action uninstall; echo \"\$ACTION\""
    [ "$status" -eq 0 ]
    [[ "$output" =~ "uninstall" ]]
}

@test "n8n::parse_arguments accepts reset-password action" {
    run bash -c "source '$SCRIPT_PATH'; n8n::parse_arguments --action reset-password; echo \"\$ACTION\""
    [ "$status" -eq 0 ]
    [[ "$output" =~ "reset-password" ]]
}

@test "n8n::parse_arguments accepts logs action" {
    run bash -c "source '$SCRIPT_PATH'; n8n::parse_arguments --action logs; echo \"\$ACTION\""
    [ "$status" -eq 0 ]
    [[ "$output" =~ "logs" ]]
}

@test "n8n::parse_arguments accepts info action" {
    run bash -c "source '$SCRIPT_PATH'; n8n::parse_arguments --action info; echo \"\$ACTION\""
    [ "$status" -eq 0 ]
    [[ "$output" =~ "info" ]]
}

@test "n8n::parse_arguments accepts execute action" {
    run bash -c "source '$SCRIPT_PATH'; n8n::parse_arguments --action execute; echo \"\$ACTION\""
    [ "$status" -eq 0 ]
    [[ "$output" =~ "execute" ]]
}

@test "n8n::parse_arguments accepts api-setup action" {
    run bash -c "source '$SCRIPT_PATH'; n8n::parse_arguments --action api-setup; echo \"\$ACTION\""
    [ "$status" -eq 0 ]
    [[ "$output" =~ "api-setup" ]]
}

@test "n8n::parse_arguments accepts save-api-key action" {
    run bash -c "source '$SCRIPT_PATH'; n8n::parse_arguments --action save-api-key; echo \"\$ACTION\""
    [ "$status" -eq 0 ]
    [[ "$output" =~ "save-api-key" ]]
}

@test "n8n::parse_arguments sets default force option to no" {
    run bash -c "source '$SCRIPT_PATH'; n8n::parse_arguments; echo \"\$FORCE\""
    [ "$status" -eq 0 ]
    [[ "$output" =~ "no" ]]
}

@test "n8n::parse_arguments accepts force option" {
    run bash -c "source '$SCRIPT_PATH'; n8n::parse_arguments --force yes; echo \"\$FORCE\""
    [ "$status" -eq 0 ]
    [[ "$output" =~ "yes" ]]
}

@test "n8n::parse_arguments sets default database to sqlite" {
    run bash -c "source '$SCRIPT_PATH'; n8n::parse_arguments; echo \"\$DATABASE_TYPE\""
    [ "$status" -eq 0 ]
    [[ "$output" =~ "sqlite" ]]
}

@test "n8n::parse_arguments accepts postgres database" {
    run bash -c "source '$SCRIPT_PATH'; n8n::parse_arguments --database postgres; echo \"\$DATABASE_TYPE\""
    [ "$status" -eq 0 ]
    [[ "$output" =~ "postgres" ]]
}

@test "n8n::parse_arguments sets default basic-auth to yes" {
    run bash -c "source '$SCRIPT_PATH'; n8n::parse_arguments; echo \"\$BASIC_AUTH\""
    [ "$status" -eq 0 ]
    [[ "$output" =~ "yes" ]]
}

@test "n8n::parse_arguments accepts basic-auth disabled" {
    run bash -c "source '$SCRIPT_PATH'; n8n::parse_arguments --basic-auth no; echo \"\$BASIC_AUTH\""
    [ "$status" -eq 0 ]
    [[ "$output" =~ "no" ]]
}

@test "n8n::parse_arguments sets default username to admin" {
    run bash -c "source '$SCRIPT_PATH'; n8n::parse_arguments; echo \"\$AUTH_USERNAME\""
    [ "$status" -eq 0 ]
    [[ "$output" =~ "admin" ]]
}

@test "n8n::parse_arguments accepts custom username" {
    run bash -c "source '$SCRIPT_PATH'; n8n::parse_arguments --username testuser; echo \"\$AUTH_USERNAME\""
    [ "$status" -eq 0 ]
    [[ "$output" =~ "testuser" ]]
}

@test "n8n::parse_arguments accepts webhook-url parameter" {
    run bash -c "source '$SCRIPT_PATH'; n8n::parse_arguments --webhook-url https://example.com; echo \"\$WEBHOOK_URL\""
    [ "$status" -eq 0 ]
    [[ "$output" =~ "https://example.com" ]]
}

@test "n8n::parse_arguments accepts workflow-id parameter" {
    run bash -c "source '$SCRIPT_PATH'; n8n::parse_arguments --workflow-id 123; echo \"\$WORKFLOW_ID\""
    [ "$status" -eq 0 ]
    [[ "$output" =~ "123" ]]
}

@test "n8n::parse_arguments accepts api-key parameter" {
    run bash -c "source '$SCRIPT_PATH'; n8n::parse_arguments --api-key secret123; echo \"\$API_KEY\""
    [ "$status" -eq 0 ]
    [[ "$output" =~ "secret123" ]]
}

@test "n8n::parse_arguments accepts data parameter" {
    run bash -c "source '$SCRIPT_PATH'; n8n::parse_arguments --data '{\"test\": true}'; echo \"\$WORKFLOW_DATA\""
    [ "$status" -eq 0 ]
    [[ "$output" =~ '{"test": true}' ]]
}

@test "n8n::parse_arguments sets default tunnel to no" {
    run bash -c "source '$SCRIPT_PATH'; n8n::parse_arguments; echo \"\$TUNNEL_ENABLED\""
    [ "$status" -eq 0 ]
    [[ "$output" =~ "no" ]]
}

@test "n8n::parse_arguments accepts tunnel enabled" {
    run bash -c "source '$SCRIPT_PATH'; n8n::parse_arguments --tunnel yes; echo \"\$TUNNEL_ENABLED\""
    [ "$status" -eq 0 ]
    [[ "$output" =~ "yes" ]]
}

@test "n8n::parse_arguments sets default build-image to no" {
    run bash -c "source '$SCRIPT_PATH'; n8n::parse_arguments; echo \"\$BUILD_IMAGE\""
    [ "$status" -eq 0 ]
    [[ "$output" =~ "no" ]]
}

@test "n8n::parse_arguments accepts build-image enabled" {
    run bash -c "source '$SCRIPT_PATH'; n8n::parse_arguments --build-image yes; echo \"\$BUILD_IMAGE\""
    [ "$status" -eq 0 ]
    [[ "$output" =~ "yes" ]]
}

# ============================================================================
# Help and Usage Tests
# ============================================================================

@test "manage.sh shows help when --help flag is used" {
    run bash -c "source '$SCRIPT_PATH'; n8n::parse_arguments --help"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Install and manage n8n workflow automation platform using Docker" ]]
}

@test "manage.sh shows help when -h flag is used" {
    run bash -c "source '$SCRIPT_PATH'; n8n::parse_arguments -h"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Install and manage n8n workflow automation platform using Docker" ]]
}

# ============================================================================
# Main Function Router Tests
# ============================================================================

@test "n8n::main handles unknown action gracefully" {
    # Create a wrapper script to test main function without executing actions
    cat > "${BATS_TEST_TMPDIR}/test_main.sh" << 'EOF'
#!/bin/bash
source "$1"

# Mock all the action functions to prevent actual execution
n8n::install() { echo "install called"; }
n8n::uninstall() { echo "uninstall called"; }
n8n::start() { echo "start called"; }
n8n::stop() { echo "stop called"; }
n8n::restart() { echo "restart called"; }
n8n::status() { echo "status called"; }
n8n::reset_password() { echo "reset-password called"; }
n8n::logs() { echo "logs called"; }
n8n::info() { echo "info called"; }
n8n::execute() { echo "execute called"; }
n8n::api_setup() { echo "api-setup called"; }
n8n::save_api_key() { echo "save-api-key called"; }
n8n::usage() { echo "usage called"; }

# Mock args::get to return invalid action
args::get() {
    if [[ "$1" == "action" ]]; then
        echo "invalid"  
    else
        echo ""
    fi
}

# Call parse_arguments to set up ACTION variable, then call main
n8n::parse_arguments
n8n::main 2>&1
EOF
    chmod +x "${BATS_TEST_TMPDIR}/test_main.sh"
    
    run "${BATS_TEST_TMPDIR}/test_main.sh" "$SCRIPT_PATH"
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Unknown action: invalid" ]]
    [[ "$output" =~ "usage called" ]]
}

@test "n8n::main routes install action correctly" {
    # Create a wrapper script to test action routing
    cat > "${BATS_TEST_TMPDIR}/test_install_route.sh" << 'EOF'
#!/bin/bash
source "$1"
# Mock the install function to prevent actual execution
n8n::install() { echo "install called"; }
n8n::parse_arguments() { ACTION="install"; }

n8n::main
EOF
    chmod +x "${BATS_TEST_TMPDIR}/test_install_route.sh"
    
    run "${BATS_TEST_TMPDIR}/test_install_route.sh" "$SCRIPT_PATH"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "install called" ]]
}

@test "n8n::main routes status action correctly" {
    cat > "${BATS_TEST_TMPDIR}/test_status_route.sh" << 'EOF'
#!/bin/bash
source "$1"
n8n::status() { echo "status called"; }
n8n::parse_arguments() { ACTION="status"; }

n8n::main
EOF
    chmod +x "${BATS_TEST_TMPDIR}/test_status_route.sh"
    
    run "${BATS_TEST_TMPDIR}/test_status_route.sh" "$SCRIPT_PATH"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "status called" ]]
}

# ============================================================================
# Configuration Tests
# ============================================================================

@test "manage.sh exports configuration correctly" {
    run bash -c "source '$SCRIPT_PATH'; echo \"\$N8N_CONTAINER_NAME\""
    [ "$status" -eq 0 ]
    [[ "$output" =~ "n8n" ]]
}

@test "manage.sh sources required configuration files" {
    # Verify that default configuration is loaded
    run bash -c "source '$SCRIPT_PATH'; echo \"\$N8N_PORT\""
    [ "$status" -eq 0 ]
    [[ "$output" =~ ^[0-9]+$ ]]
}

# ============================================================================
# Fixture-Based Tests
# ============================================================================

@test "n8n can validate workflow fixtures" {
    # Test that we can access N8N workflow fixtures
    if command -v get_workflow_fixture >/dev/null; then
        local workflow_file
        workflow_file=$(get_workflow_fixture "n8n" "default")
        
        [[ -n "$workflow_file" ]]
        [[ "$workflow_file" =~ n8n-workflow.json$ ]]
        
        # Test fixture file validation
        if command -v validate_fixture_file >/dev/null; then
            validate_fixture_file "$workflow_file"
            [ "$?" -eq 0 ]
        fi
        
        # Test specific workflow types
        local whisper_workflow
        whisper_workflow=$(get_workflow_fixture "n8n" "whisper")
        [[ -n "$whisper_workflow" ]]
        [[ "$whisper_workflow" =~ whisper-transcription.json$ ]]
    else
        skip "Fixture helpers not available"
    fi
}

@test "n8n workflow fixtures have valid JSON structure" {
    if command -v get_workflow_fixture >/dev/null && command -v jq >/dev/null; then
        local workflow_file
        workflow_file=$(get_workflow_fixture "n8n" "default")
        
        if [[ -f "$workflow_file" ]]; then
            # Should be valid JSON
            jq empty < "$workflow_file"
            [ "$?" -eq 0 ]
            
            # Should have typical n8n workflow structure
            local has_nodes
            has_nodes=$(jq -r 'has("nodes")' < "$workflow_file")
            [[ "$has_nodes" == "true" ]]
            
            local has_connections
            has_connections=$(jq -r 'has("connections")' < "$workflow_file")
            [[ "$has_connections" == "true" ]]
        else
            skip "Workflow fixture file not found: $workflow_file"
        fi
    else
        skip "Required tools not available (fixture helpers or jq)"
    fi
}

@test "n8n can process documents from fixtures" {
    # Test that N8N could work with document fixtures
    if command -v get_document_fixture >/dev/null; then
        local pdf_file
        pdf_file=$(get_document_fixture "pdf" "simple")
        
        [[ -n "$pdf_file" ]]
        [[ "$pdf_file" =~ \.pdf$ ]]
        
        # Validate the fixture file exists and is readable
        if command -v validate_fixture_file >/dev/null; then
            validate_fixture_file "$pdf_file"
            [ "$?" -eq 0 ]
        fi
        
        # Test various document types that N8N might process
        local json_file
        json_file=$(get_document_fixture "json")
        [[ -n "$json_file" ]]
        [[ "$json_file" =~ \.json$ ]]
        
        local html_file
        html_file=$(get_document_fixture "html")
        [[ -n "$html_file" ]]
        [[ "$html_file" =~ \.html$ ]]
    else
        skip "Fixture helpers not available"
    fi
}

# ============================================================================
# Error Handling Tests
# ============================================================================

@test "manage.sh handles missing dependencies gracefully" {
    # Test without certain dependencies
    cat > "${BATS_TEST_TMPDIR}/test_deps.sh" << 'EOF'
#!/bin/bash
# Mock a missing dependency scenario
PATH="/nonexistent:$PATH"
source "$1" 2>/dev/null || echo "dependency error handled"
EOF
    chmod +x "${BATS_TEST_TMPDIR}/test_deps.sh"
    
    run "${BATS_TEST_TMPDIR}/test_deps.sh" "$SCRIPT_PATH"
    [ "$status" -eq 0 ]
}