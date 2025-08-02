#!/usr/bin/env bats
# Tests for n8n messages.sh configuration

# Setup for each test
setup() {
    # Load dependencies
    SCRIPT_DIR="$(dirname "${BATS_TEST_FILENAME}")"
    N8N_DIR="$(dirname "$SCRIPT_DIR")"
    
    # Set up test variables that messages might reference
    export N8N_PORT="5678"
    export N8N_BASE_URL="http://localhost:5678"
    export N8N_CONTAINER_NAME="vrooli-n8n"
    export N8N_SERVICE_NAME="n8n"
    export WEBHOOK_URL="http://localhost:5678/webhook"
    
    # Load the messages configuration to test
    source "${N8N_DIR}/config/messages.sh"
}

# Test installation messages
@test "MSG_N8N_INSTALLING is defined and meaningful" {
    [ -n "$MSG_N8N_INSTALLING" ]
    [[ "$MSG_N8N_INSTALLING" =~ "Installing" ]]
    [[ "$MSG_N8N_INSTALLING" =~ "n8n" ]]
}

@test "MSG_N8N_ALREADY_INSTALLED is defined and meaningful" {
    [ -n "$MSG_N8N_ALREADY_INSTALLED" ]
    [[ "$MSG_N8N_ALREADY_INSTALLED" =~ "already" ]]
    [[ "$MSG_N8N_ALREADY_INSTALLED" =~ "installed" ]]
}

# Test status messages
@test "MSG_N8N_NOT_FOUND is defined and meaningful" {
    [ -n "$MSG_N8N_NOT_FOUND" ]
    [[ "$MSG_N8N_NOT_FOUND" =~ "not found" ]]
}

@test "MSG_N8N_NOT_RUNNING is defined and meaningful" {
    [ -n "$MSG_N8N_NOT_RUNNING" ]
    [[ "$MSG_N8N_NOT_RUNNING" =~ "not running" ]]
}

@test "MSG_STATUS_CONTAINER_OK is defined and meaningful" {
    [ -n "$MSG_STATUS_CONTAINER_OK" ]
    [[ "$MSG_STATUS_CONTAINER_OK" =~ "container" ]]
    [[ "$MSG_STATUS_CONTAINER_OK" =~ "OK" || "$MSG_STATUS_CONTAINER_OK" =~ "found" ]]
}

@test "MSG_STATUS_CONTAINER_RUNNING is defined and meaningful" {
    [ -n "$MSG_STATUS_CONTAINER_RUNNING" ]
    [[ "$MSG_STATUS_CONTAINER_RUNNING" =~ "running" ]]
}

# Test workflow messages
@test "MSG_WORKFLOW_EXECUTING is defined and supports variable substitution" {
    [ -n "$MSG_WORKFLOW_EXECUTING" ]
    [[ "$MSG_WORKFLOW_EXECUTING" =~ "Executing" || "$MSG_WORKFLOW_EXECUTING" =~ "workflow" ]]
    
    # Test variable substitution
    workflow_id="test-workflow-123"
    expanded_msg=$(eval "echo \"$MSG_WORKFLOW_EXECUTING\"")
    [[ "$expanded_msg" =~ "$workflow_id" ]]
}

@test "MSG_WORKFLOW_COMPLETED is defined and meaningful" {
    [ -n "$MSG_WORKFLOW_COMPLETED" ]
    [[ "$MSG_WORKFLOW_COMPLETED" =~ "complete" || "$MSG_WORKFLOW_COMPLETED" =~ "finished" ]]
}

# Test database messages
@test "MSG_DATABASE_STARTING is defined and meaningful" {
    [ -n "$MSG_DATABASE_STARTING" ]
    [[ "$MSG_DATABASE_STARTING" =~ "database" ]]
    [[ "$MSG_DATABASE_STARTING" =~ "Starting" ]]
}

@test "MSG_DATABASE_READY is defined and meaningful" {
    [ -n "$MSG_DATABASE_READY" ]
    [[ "$MSG_DATABASE_READY" =~ "database" ]]
    [[ "$MSG_DATABASE_READY" =~ "ready" || "$MSG_DATABASE_READY" =~ "available" ]]
}

# Test error messages
@test "MSG_DOCKER_NOT_AVAILABLE is defined and meaningful" {
    [ -n "$MSG_DOCKER_NOT_AVAILABLE" ]
    [[ "$MSG_DOCKER_NOT_AVAILABLE" =~ "Docker" ]]
    [[ "$MSG_DOCKER_NOT_AVAILABLE" =~ "not available" || "$MSG_DOCKER_NOT_AVAILABLE" =~ "not found" ]]
}

@test "MSG_PORT_IN_USE is defined and supports variable substitution" {
    [ -n "$MSG_PORT_IN_USE" ]
    [[ "$MSG_PORT_IN_USE" =~ "port" ]]
    [[ "$MSG_PORT_IN_USE" =~ "in use" || "$MSG_PORT_IN_USE" =~ "busy" ]]
    
    # Test variable substitution
    port="5678"
    expanded_msg=$(eval "echo \"$MSG_PORT_IN_USE\"")
    [[ "$expanded_msg" =~ "$port" ]]
}

@test "MSG_DATABASE_CONNECTION_FAILED is defined and meaningful" {
    [ -n "$MSG_DATABASE_CONNECTION_FAILED" ]
    [[ "$MSG_DATABASE_CONNECTION_FAILED" =~ "database" ]]
    [[ "$MSG_DATABASE_CONNECTION_FAILED" =~ "connection" ]]
    [[ "$MSG_DATABASE_CONNECTION_FAILED" =~ "failed" ]]
}

@test "MSG_WORKFLOW_EXECUTION_FAILED is defined and meaningful" {
    [ -n "$MSG_WORKFLOW_EXECUTION_FAILED" ]
    [[ "$MSG_WORKFLOW_EXECUTION_FAILED" =~ "workflow" ]]
    [[ "$MSG_WORKFLOW_EXECUTION_FAILED" =~ "execution" ]]
    [[ "$MSG_WORKFLOW_EXECUTION_FAILED" =~ "failed" ]]
}

# Test success messages
@test "MSG_INSTALLATION_SUCCESS is defined and meaningful" {
    [ -n "$MSG_INSTALLATION_SUCCESS" ]
    [[ "$MSG_INSTALLATION_SUCCESS" =~ "success" || "$MSG_INSTALLATION_SUCCESS" =~ "installed" ]]
    [[ "$MSG_INSTALLATION_SUCCESS" =~ "n8n" ]]
}

@test "MSG_SERVICE_HEALTHY is defined and meaningful" {
    [ -n "$MSG_SERVICE_HEALTHY" ]
    [[ "$MSG_SERVICE_HEALTHY" =~ "healthy" || "$MSG_SERVICE_HEALTHY" =~ "running" ]]
}

# Test information messages
@test "MSG_SERVICE_INFO is defined and supports variable substitution" {
    [ -n "$MSG_SERVICE_INFO" ]
    [[ "$MSG_SERVICE_INFO" =~ "n8n" ]]
    
    # Test that it can include URL
    expanded_msg=$(eval "echo \"$MSG_SERVICE_INFO\"")
    [[ "$expanded_msg" =~ "http://" || "$expanded_msg" =~ "localhost" ]]
}

@test "MSG_AVAILABLE_AT is defined and supports variable substitution" {
    [ -n "$MSG_AVAILABLE_AT" ]
    [[ "$MSG_AVAILABLE_AT" =~ "available" ]]
    
    # Test URL substitution
    expanded_msg=$(eval "echo \"$MSG_AVAILABLE_AT\"")
    [[ "$expanded_msg" =~ "http://localhost:5678" ]]
}

@test "MSG_WEBHOOK_URL is defined and supports variable substitution" {
    [ -n "$MSG_WEBHOOK_URL" ]
    [[ "$MSG_WEBHOOK_URL" =~ "webhook" ]]
    
    # Test webhook URL substitution
    expanded_msg=$(eval "echo \"$MSG_WEBHOOK_URL\"")
    [[ "$expanded_msg" =~ "http://localhost:5678/webhook" ]]
}

# Test uninstallation messages
@test "MSG_UNINSTALLING is defined and meaningful" {
    [ -n "$MSG_UNINSTALLING" ]
    [[ "$MSG_UNINSTALLING" =~ "Uninstalling" || "$MSG_UNINSTALLING" =~ "Removing" ]]
}

@test "MSG_UNINSTALL_SUCCESS is defined and meaningful" {
    [ -n "$MSG_UNINSTALL_SUCCESS" ]
    [[ "$MSG_UNINSTALL_SUCCESS" =~ "success" || "$MSG_UNINSTALL_SUCCESS" =~ "removed" ]]
}

# Test startup/shutdown messages
@test "MSG_STARTING_SERVICE is defined and meaningful" {
    [ -n "$MSG_STARTING_SERVICE" ]
    [[ "$MSG_STARTING_SERVICE" =~ "Starting" ]]
}

@test "MSG_STOPPING_SERVICE is defined and meaningful" {
    [ -n "$MSG_STOPPING_SERVICE" ]
    [[ "$MSG_STOPPING_SERVICE" =~ "Stopping" ]]
}

@test "MSG_RESTARTING_SERVICE is defined and meaningful" {
    [ -n "$MSG_RESTARTING_SERVICE" ]
    [[ "$MSG_RESTARTING_SERVICE" =~ "Restarting" ]]
}

# Test authentication messages
@test "MSG_PASSWORD_RESET is defined and meaningful" {
    [ -n "$MSG_PASSWORD_RESET" ]
    [[ "$MSG_PASSWORD_RESET" =~ "password" ]]
    [[ "$MSG_PASSWORD_RESET" =~ "reset" ]]
}

@test "MSG_AUTH_CONFIGURED is defined and meaningful" {
    [ -n "$MSG_AUTH_CONFIGURED" ]
    [[ "$MSG_AUTH_CONFIGURED" =~ "auth" ]]
    [[ "$MSG_AUTH_CONFIGURED" =~ "configured" ]]
}

# Test backup/restore messages
@test "MSG_BACKUP_CREATED is defined and meaningful" {
    [ -n "$MSG_BACKUP_CREATED" ]
    [[ "$MSG_BACKUP_CREATED" =~ "backup" ]]
    [[ "$MSG_BACKUP_CREATED" =~ "created" ]]
}

@test "MSG_BACKUP_RESTORED is defined and meaningful" {
    [ -n "$MSG_BACKUP_RESTORED" ]
    [[ "$MSG_BACKUP_RESTORED" =~ "backup" ]]
    [[ "$MSG_BACKUP_RESTORED" =~ "restored" ]]
}

# Test message export function
@test "n8n::export_messages exports all message variables" {
    n8n::export_messages
    
    # Check that variables are exported (available in subshells)
    result=$(bash -c 'echo $MSG_N8N_INSTALLING')
    [ -n "$result" ]
    
    result=$(bash -c 'echo $MSG_SERVICE_INFO')
    [ -n "$result" ]
}

# Test that all required messages are defined
@test "all essential message variables are defined" {
    [ -n "$MSG_N8N_INSTALLING" ]
    [ -n "$MSG_N8N_ALREADY_INSTALLED" ]
    [ -n "$MSG_N8N_NOT_FOUND" ]
    [ -n "$MSG_N8N_NOT_RUNNING" ]
    [ -n "$MSG_STATUS_CONTAINER_OK" ]
    [ -n "$MSG_STATUS_CONTAINER_RUNNING" ]
    [ -n "$MSG_WORKFLOW_EXECUTING" ]
    [ -n "$MSG_DATABASE_STARTING" ]
    [ -n "$MSG_DOCKER_NOT_AVAILABLE" ]
    [ -n "$MSG_PORT_IN_USE" ]
    [ -n "$MSG_INSTALLATION_SUCCESS" ]
    [ -n "$MSG_SERVICE_INFO" ]
    [ -n "$MSG_AVAILABLE_AT" ]
}

# Test message content quality
@test "messages contain appropriate emoji or formatting" {
    # Check that some messages have visual indicators
    local has_visual_indicator=false
    
    if [[ "$MSG_INSTALLATION_SUCCESS" =~ [✅🎉✨] ]] || \
       [[ "$MSG_N8N_INSTALLING" =~ [🔧📦⚙️] ]] || \
       [[ "$MSG_SERVICE_HEALTHY" =~ [✅💚🟢] ]]; then
        has_visual_indicator=true
    fi
    
    [ "$has_visual_indicator" = true ]
}

@test "error messages are appropriately formatted" {
    # Error messages should have warning indicators or clear language
    local has_error_indicator=false
    
    if [[ "$MSG_DOCKER_NOT_AVAILABLE" =~ [❌⚠️🔴] ]] || \
       [[ "$MSG_DATABASE_CONNECTION_FAILED" =~ [❌⚠️] ]] || \
       [[ "$MSG_DOCKER_NOT_AVAILABLE" =~ "ERROR" ]]; then
        has_error_indicator=true
    fi
    
    [ "$has_error_indicator" = true ]
}

# Test message variable interpolation
@test "messages with variables interpolate correctly" {
    # Test workflow ID variable
    workflow_id="test-workflow-123"
    expanded=$(eval "echo \"$MSG_WORKFLOW_EXECUTING\"")
    [[ "$expanded" =~ "$workflow_id" ]]
    
    # Test port variable  
    expanded=$(eval "echo \"$MSG_PORT_IN_USE\"")
    [[ "$expanded" =~ "5678" ]]
    
    # Test URL variable
    expanded=$(eval "echo \"$MSG_AVAILABLE_AT\"")
    [[ "$expanded" =~ "http://localhost:5678" ]]
    
    # Test webhook URL variable
    expanded=$(eval "echo \"$MSG_WEBHOOK_URL\"")
    [[ "$expanded" =~ "http://localhost:5678/webhook" ]]
}

# Test message consistency
@test "messages use consistent service naming" {
    # Check that service is consistently referred to
    local service_refs=0
    
    if [[ "$MSG_N8N_INSTALLING" =~ "n8n" ]]; then
        ((service_refs++))
    fi
    
    if [[ "$MSG_INSTALLATION_SUCCESS" =~ "n8n" ]]; then
        ((service_refs++))
    fi
    
    if [[ "$MSG_SERVICE_INFO" =~ "n8n" ]]; then
        ((service_refs++))
    fi
    
    # At least 2 messages should consistently name the service
    [ "$service_refs" -ge 2 ]
}

# Test workflow-specific message quality
@test "workflow messages are specific and informative" {
    # Workflow execution message should be specific
    [[ "$MSG_WORKFLOW_EXECUTING" =~ "workflow" ]]
    [[ "$MSG_WORKFLOW_COMPLETED" =~ "workflow" ]] || [[ "$MSG_WORKFLOW_COMPLETED" =~ "execution" ]]
}

# Test database-specific message quality
@test "database messages are specific and informative" {
    # Database messages should mention database explicitly
    [[ "$MSG_DATABASE_STARTING" =~ "database" ]]
    [[ "$MSG_DATABASE_READY" =~ "database" ]]
    [[ "$MSG_DATABASE_CONNECTION_FAILED" =~ "database" ]]
}

# Test authentication message quality
@test "authentication messages are clear and actionable" {
    # Auth messages should be clear about what's happening
    [[ "$MSG_PASSWORD_RESET" =~ "password" ]]
    [[ "$MSG_AUTH_CONFIGURED" =~ "auth" ]]
}
