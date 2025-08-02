#!/usr/bin/env bats
# Tests for Windmill messages.sh configuration

# Setup for each test
setup() {
    # Load dependencies
    SCRIPT_DIR="$(dirname "${BATS_TEST_FILENAME}")"
    WINDMILL_DIR="$(dirname "$SCRIPT_DIR")"
    
    # Set up test variables that messages might reference
    export WINDMILL_PORT="5681"
    export WINDMILL_BASE_URL="http://localhost:5681"
    export WINDMILL_CONTAINER_NAME="vrooli-windmill"
    export WINDMILL_SERVICE_NAME="windmill"
    export WORKSPACE_NAME="demo"
    export USER_EMAIL="admin@example.com"
    export SCRIPT_PATH="/path/to/script.py"
    export JOB_ID="job_123456"
    
    # Load the messages configuration to test
    source "${WINDMILL_DIR}/config/messages.sh"
}

# Test installation messages
@test "MSG_WINDMILL_INSTALLING is defined and meaningful" {
    [ -n "$MSG_WINDMILL_INSTALLING" ]
    [[ "$MSG_WINDMILL_INSTALLING" =~ "Installing" ]]
    [[ "$MSG_WINDMILL_INSTALLING" =~ "Windmill" ]]
}

@test "MSG_WINDMILL_ALREADY_INSTALLED is defined and meaningful" {
    [ -n "$MSG_WINDMILL_ALREADY_INSTALLED" ]
    [[ "$MSG_WINDMILL_ALREADY_INSTALLED" =~ "already" ]]
    [[ "$MSG_WINDMILL_ALREADY_INSTALLED" =~ "installed" ]]
}

# Test status messages
@test "MSG_WINDMILL_NOT_FOUND is defined and meaningful" {
    [ -n "$MSG_WINDMILL_NOT_FOUND" ]
    [[ "$MSG_WINDMILL_NOT_FOUND" =~ "not found" ]]
}

@test "MSG_WINDMILL_NOT_RUNNING is defined and meaningful" {
    [ -n "$MSG_WINDMILL_NOT_RUNNING" ]
    [[ "$MSG_WINDMILL_NOT_RUNNING" =~ "not running" ]]
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

# Test script execution messages
@test "MSG_SCRIPT_EXECUTING is defined and supports variable substitution" {
    [ -n "$MSG_SCRIPT_EXECUTING" ]
    [[ "$MSG_SCRIPT_EXECUTING" =~ "Executing" || "$MSG_SCRIPT_EXECUTING" =~ "script" ]]
    
    # Test variable substitution
    script_path="/path/to/script.py"
    expanded_msg=$(eval "echo \"$MSG_SCRIPT_EXECUTING\"")
    [[ "$expanded_msg" =~ "$script_path" ]]
}

@test "MSG_SCRIPT_COMPLETED is defined and meaningful" {
    [ -n "$MSG_SCRIPT_COMPLETED" ]
    [[ "$MSG_SCRIPT_COMPLETED" =~ "complete" || "$MSG_SCRIPT_COMPLETED" =~ "finished" ]]
}

@test "MSG_SCRIPT_FAILED is defined and meaningful" {
    [ -n "$MSG_SCRIPT_FAILED" ]
    [[ "$MSG_SCRIPT_FAILED" =~ "failed" || "$MSG_SCRIPT_FAILED" =~ "error" ]]
}

# Test job management messages
@test "MSG_JOB_STARTED is defined and supports variable substitution" {
    [ -n "$MSG_JOB_STARTED" ]
    [[ "$MSG_JOB_STARTED" =~ "job" || "$MSG_JOB_STARTED" =~ "started" ]]
    
    # Test variable substitution
    job_id="job_123456"
    expanded_msg=$(eval "echo \"$MSG_JOB_STARTED\"")
    [[ "$expanded_msg" =~ "$job_id" ]]
}

@test "MSG_JOB_COMPLETED is defined and meaningful" {
    [ -n "$MSG_JOB_COMPLETED" ]
    [[ "$MSG_JOB_COMPLETED" =~ "job" ]]
    [[ "$MSG_JOB_COMPLETED" =~ "complete" || "$MSG_JOB_COMPLETED" =~ "finished" ]]
}

@test "MSG_JOB_FAILED is defined and meaningful" {
    [ -n "$MSG_JOB_FAILED" ]
    [[ "$MSG_JOB_FAILED" =~ "job" ]]
    [[ "$MSG_JOB_FAILED" =~ "failed" || "$MSG_JOB_FAILED" =~ "error" ]]
}

# Test workspace messages
@test "MSG_WORKSPACE_CREATED is defined and supports variable substitution" {
    [ -n "$MSG_WORKSPACE_CREATED" ]
    [[ "$MSG_WORKSPACE_CREATED" =~ "workspace" ]]
    [[ "$MSG_WORKSPACE_CREATED" =~ "created" ]]
    
    # Test variable substitution
    workspace_name="demo"
    expanded_msg=$(eval "echo \"$MSG_WORKSPACE_CREATED\"")
    [[ "$expanded_msg" =~ "$workspace_name" ]]
}

@test "MSG_WORKSPACE_NOT_FOUND is defined and meaningful" {
    [ -n "$MSG_WORKSPACE_NOT_FOUND" ]
    [[ "$MSG_WORKSPACE_NOT_FOUND" =~ "workspace" ]]
    [[ "$MSG_WORKSPACE_NOT_FOUND" =~ "not found" || "$MSG_WORKSPACE_NOT_FOUND" =~ "missing" ]]
}

# Test user management messages
@test "MSG_USER_CREATED is defined and supports variable substitution" {
    [ -n "$MSG_USER_CREATED" ]
    [[ "$MSG_USER_CREATED" =~ "user" ]]
    [[ "$MSG_USER_CREATED" =~ "created" ]]
    
    # Test variable substitution
    user_email="admin@example.com"
    expanded_msg=$(eval "echo \"$MSG_USER_CREATED\"")
    [[ "$expanded_msg" =~ "$user_email" ]]
}

@test "MSG_USER_NOT_FOUND is defined and meaningful" {
    [ -n "$MSG_USER_NOT_FOUND" ]
    [[ "$MSG_USER_NOT_FOUND" =~ "user" ]]
    [[ "$MSG_USER_NOT_FOUND" =~ "not found" || "$MSG_USER_NOT_FOUND" =~ "missing" ]]
}

@test "MSG_PASSWORD_CHANGED is defined and meaningful" {
    [ -n "$MSG_PASSWORD_CHANGED" ]
    [[ "$MSG_PASSWORD_CHANGED" =~ "password" ]]
    [[ "$MSG_PASSWORD_CHANGED" =~ "changed" || "$MSG_PASSWORD_CHANGED" =~ "updated" ]]
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

@test "MSG_DATABASE_CONNECTION_FAILED is defined and meaningful" {
    [ -n "$MSG_DATABASE_CONNECTION_FAILED" ]
    [[ "$MSG_DATABASE_CONNECTION_FAILED" =~ "database" ]]
    [[ "$MSG_DATABASE_CONNECTION_FAILED" =~ "connection" ]]
    [[ "$MSG_DATABASE_CONNECTION_FAILED" =~ "failed" ]]
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
    port="5681"
    expanded_msg=$(eval "echo \"$MSG_PORT_IN_USE\"")
    [[ "$expanded_msg" =~ "$port" ]]
}

@test "MSG_INSUFFICIENT_MEMORY is defined and meaningful" {
    [ -n "$MSG_INSUFFICIENT_MEMORY" ]
    [[ "$MSG_INSUFFICIENT_MEMORY" =~ "memory" ]]
    [[ "$MSG_INSUFFICIENT_MEMORY" =~ "insufficient" || "$MSG_INSUFFICIENT_MEMORY" =~ "not enough" ]]
}

@test "MSG_IMAGE_PULL_FAILED is defined and meaningful" {
    [ -n "$MSG_IMAGE_PULL_FAILED" ]
    [[ "$MSG_IMAGE_PULL_FAILED" =~ "image" ]]
    [[ "$MSG_IMAGE_PULL_FAILED" =~ "pull" ]]
    [[ "$MSG_IMAGE_PULL_FAILED" =~ "failed" ]]
}

# Test success messages
@test "MSG_INSTALLATION_SUCCESS is defined and meaningful" {
    [ -n "$MSG_INSTALLATION_SUCCESS" ]
    [[ "$MSG_INSTALLATION_SUCCESS" =~ "success" || "$MSG_INSTALLATION_SUCCESS" =~ "installed" ]]
    [[ "$MSG_INSTALLATION_SUCCESS" =~ "Windmill" ]]
}

@test "MSG_SERVICE_HEALTHY is defined and meaningful" {
    [ -n "$MSG_SERVICE_HEALTHY" ]
    [[ "$MSG_SERVICE_HEALTHY" =~ "healthy" || "$MSG_SERVICE_HEALTHY" =~ "running" ]]
}

@test "MSG_CONTAINER_STARTED is defined and meaningful" {
    [ -n "$MSG_CONTAINER_STARTED" ]
    [[ "$MSG_CONTAINER_STARTED" =~ "started" || "$MSG_CONTAINER_STARTED" =~ "running" ]]
}

# Test information messages
@test "MSG_SERVICE_INFO is defined and supports variable substitution" {
    [ -n "$MSG_SERVICE_INFO" ]
    [[ "$MSG_SERVICE_INFO" =~ "Windmill" ]]
    
    # Test that it can include URL
    expanded_msg=$(eval "echo \"$MSG_SERVICE_INFO\"")
    [[ "$expanded_msg" =~ "http://" || "$expanded_msg" =~ "localhost" ]]
}

@test "MSG_AVAILABLE_AT is defined and supports variable substitution" {
    [ -n "$MSG_AVAILABLE_AT" ]
    [[ "$MSG_AVAILABLE_AT" =~ "available" ]]
    
    # Test URL substitution
    expanded_msg=$(eval "echo \"$MSG_AVAILABLE_AT\"")
    [[ "$expanded_msg" =~ "http://localhost:5681" ]]
}

@test "MSG_API_ENDPOINT is defined and supports variable substitution" {
    [ -n "$MSG_API_ENDPOINT" ]
    [[ "$MSG_API_ENDPOINT" =~ "API" ]]
    
    # Test API URL substitution
    expanded_msg=$(eval "echo \"$MSG_API_ENDPOINT\"")
    [[ "$expanded_msg" =~ "http://localhost:5681/api" ]]
}

# Test flow execution messages
@test "MSG_FLOW_EXECUTING is defined and meaningful" {
    [ -n "$MSG_FLOW_EXECUTING" ]
    [[ "$MSG_FLOW_EXECUTING" =~ "flow" ]]
    [[ "$MSG_FLOW_EXECUTING" =~ "executing" || "$MSG_FLOW_EXECUTING" =~ "running" ]]
}

@test "MSG_FLOW_COMPLETED is defined and meaningful" {
    [ -n "$MSG_FLOW_COMPLETED" ]
    [[ "$MSG_FLOW_COMPLETED" =~ "flow" ]]
    [[ "$MSG_FLOW_COMPLETED" =~ "complete" || "$MSG_FLOW_COMPLETED" =~ "finished" ]]
}

# Test app management messages
@test "MSG_APP_DEPLOYED is defined and meaningful" {
    [ -n "$MSG_APP_DEPLOYED" ]
    [[ "$MSG_APP_DEPLOYED" =~ "app" ]]
    [[ "$MSG_APP_DEPLOYED" =~ "deployed" || "$MSG_APP_DEPLOYED" =~ "published" ]]
}

@test "MSG_APP_NOT_FOUND is defined and meaningful" {
    [ -n "$MSG_APP_NOT_FOUND" ]
    [[ "$MSG_APP_NOT_FOUND" =~ "app" ]]
    [[ "$MSG_APP_NOT_FOUND" =~ "not found" || "$MSG_APP_NOT_FOUND" =~ "missing" ]]
}

# Test worker messages
@test "MSG_WORKER_STARTED is defined and meaningful" {
    [ -n "$MSG_WORKER_STARTED" ]
    [[ "$MSG_WORKER_STARTED" =~ "worker" ]]
    [[ "$MSG_WORKER_STARTED" =~ "started" || "$MSG_WORKER_STARTED" =~ "running" ]]
}

@test "MSG_WORKER_STOPPED is defined and meaningful" {
    [ -n "$MSG_WORKER_STOPPED" ]
    [[ "$MSG_WORKER_STOPPED" =~ "worker" ]]
    [[ "$MSG_WORKER_STOPPED" =~ "stopped" || "$MSG_WORKER_STOPPED" =~ "terminated" ]]
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

# Test variable management messages
@test "MSG_VARIABLE_CREATED is defined and meaningful" {
    [ -n "$MSG_VARIABLE_CREATED" ]
    [[ "$MSG_VARIABLE_CREATED" =~ "variable" ]]
    [[ "$MSG_VARIABLE_CREATED" =~ "created" ]]
}

@test "MSG_SECRET_CREATED is defined and meaningful" {
    [ -n "$MSG_SECRET_CREATED" ]
    [[ "$MSG_SECRET_CREATED" =~ "secret" ]]
    [[ "$MSG_SECRET_CREATED" =~ "created" ]]
}

# Test resource management messages
@test "MSG_RESOURCE_CREATED is defined and meaningful" {
    [ -n "$MSG_RESOURCE_CREATED" ]
    [[ "$MSG_RESOURCE_CREATED" =~ "resource" ]]
    [[ "$MSG_RESOURCE_CREATED" =~ "created" ]]
}

@test "MSG_RESOURCE_NOT_FOUND is defined and meaningful" {
    [ -n "$MSG_RESOURCE_NOT_FOUND" ]
    [[ "$MSG_RESOURCE_NOT_FOUND" =~ "resource" ]]
    [[ "$MSG_RESOURCE_NOT_FOUND" =~ "not found" ]]
}

# Test schedule management messages
@test "MSG_SCHEDULE_CREATED is defined and meaningful" {
    [ -n "$MSG_SCHEDULE_CREATED" ]
    [[ "$MSG_SCHEDULE_CREATED" =~ "schedule" ]]
    [[ "$MSG_SCHEDULE_CREATED" =~ "created" ]]
}

@test "MSG_SCHEDULE_ENABLED is defined and meaningful" {
    [ -n "$MSG_SCHEDULE_ENABLED" ]
    [[ "$MSG_SCHEDULE_ENABLED" =~ "schedule" ]]
    [[ "$MSG_SCHEDULE_ENABLED" =~ "enabled" || "$MSG_SCHEDULE_ENABLED" =~ "activated" ]]
}

@test "MSG_SCHEDULE_DISABLED is defined and meaningful" {
    [ -n "$MSG_SCHEDULE_DISABLED" ]
    [[ "$MSG_SCHEDULE_DISABLED" =~ "schedule" ]]
    [[ "$MSG_SCHEDULE_DISABLED" =~ "disabled" || "$MSG_SCHEDULE_DISABLED" =~ "deactivated" ]]
}

# Test message export function
@test "windmill::export_messages exports all message variables" {
    windmill::export_messages
    
    # Check that variables are exported (available in subshells)
    result=$(bash -c 'echo $MSG_WINDMILL_INSTALLING')
    [ -n "$result" ]
    
    result=$(bash -c 'echo $MSG_SERVICE_INFO')
    [ -n "$result" ]
}

# Test that all required messages are defined
@test "all essential message variables are defined" {
    [ -n "$MSG_WINDMILL_INSTALLING" ]
    [ -n "$MSG_WINDMILL_ALREADY_INSTALLED" ]
    [ -n "$MSG_WINDMILL_NOT_FOUND" ]
    [ -n "$MSG_WINDMILL_NOT_RUNNING" ]
    [ -n "$MSG_STATUS_CONTAINER_OK" ]
    [ -n "$MSG_STATUS_CONTAINER_RUNNING" ]
    [ -n "$MSG_SCRIPT_EXECUTING" ]
    [ -n "$MSG_JOB_STARTED" ]
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
       [[ "$MSG_WINDMILL_INSTALLING" =~ [🔧📦⚙️] ]] || \
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
    # Test script path variable
    script_path="/path/to/script.py"
    expanded=$(eval "echo \"$MSG_SCRIPT_EXECUTING\"")
    [[ "$expanded" =~ "$script_path" ]]
    
    # Test job ID variable  
    job_id="job_123456"
    expanded=$(eval "echo \"$MSG_JOB_STARTED\"")
    [[ "$expanded" =~ "$job_id" ]]
    
    # Test port variable
    expanded=$(eval "echo \"$MSG_PORT_IN_USE\"")
    [[ "$expanded" =~ "5681" ]]
    
    # Test URL variable
    expanded=$(eval "echo \"$MSG_AVAILABLE_AT\"")
    [[ "$expanded" =~ "http://localhost:5681" ]]
    
    # Test API endpoint variable
    expanded=$(eval "echo \"$MSG_API_ENDPOINT\"")
    [[ "$expanded" =~ "http://localhost:5681/api" ]]
}

# Test message consistency
@test "messages use consistent service naming" {
    # Check that service is consistently referred to
    local service_refs=0
    
    if [[ "$MSG_WINDMILL_INSTALLING" =~ "Windmill" ]]; then
        ((service_refs++))
    fi
    
    if [[ "$MSG_INSTALLATION_SUCCESS" =~ "Windmill" ]]; then
        ((service_refs++))
    fi
    
    if [[ "$MSG_SERVICE_INFO" =~ "Windmill" ]]; then
        ((service_refs++))
    fi
    
    # At least 2 messages should consistently name the service
    [ "$service_refs" -ge 2 ]
}

# Test script-specific message quality
@test "script messages are specific and informative" {
    # Script execution message should be specific
    [[ "$MSG_SCRIPT_EXECUTING" =~ "script" ]]
    [[ "$MSG_SCRIPT_COMPLETED" =~ "script" ]] || [[ "$MSG_SCRIPT_COMPLETED" =~ "execution" ]]
    [[ "$MSG_SCRIPT_FAILED" =~ "script" ]] || [[ "$MSG_SCRIPT_FAILED" =~ "execution" ]]
}

# Test job-specific message quality
@test "job messages are specific and informative" {
    # Job messages should mention jobs explicitly
    [[ "$MSG_JOB_STARTED" =~ "job" ]]
    [[ "$MSG_JOB_COMPLETED" =~ "job" ]]
    [[ "$MSG_JOB_FAILED" =~ "job" ]]
}

# Test workspace-specific message quality
@test "workspace messages are specific and informative" {
    # Workspace messages should mention workspaces explicitly
    [[ "$MSG_WORKSPACE_CREATED" =~ "workspace" ]]
    [[ "$MSG_WORKSPACE_NOT_FOUND" =~ "workspace" ]]
}

# Test database-specific message quality
@test "database messages are specific and informative" {
    # Database messages should mention database explicitly
    [[ "$MSG_DATABASE_STARTING" =~ "database" ]]
    [[ "$MSG_DATABASE_READY" =~ "database" ]]
    [[ "$MSG_DATABASE_CONNECTION_FAILED" =~ "database" ]]
}

# Test user management message quality
@test "user management messages are clear and actionable" {
    # User messages should be clear about what's happening
    [[ "$MSG_USER_CREATED" =~ "user" ]]
    [[ "$MSG_USER_NOT_FOUND" =~ "user" ]]
    [[ "$MSG_PASSWORD_CHANGED" =~ "password" ]]
}

# Test technical message appropriateness
@test "technical messages provide helpful context" {
    # Technical messages should be informative
    [[ "$MSG_DOCKER_NOT_AVAILABLE" =~ "Docker" ]]
    [[ "$MSG_IMAGE_PULL_FAILED" =~ "image" ]]
    [[ "$MSG_INSUFFICIENT_MEMORY" =~ "memory" ]]
}

# Test user-facing message quality
@test "user-facing messages are welcoming and informative" {
    # Service info should be welcoming
    [[ "$MSG_SERVICE_INFO" =~ "Windmill" ]]
    [[ "$MSG_AVAILABLE_AT" =~ "available" ]]
    [[ "$MSG_API_ENDPOINT" =~ "API" ]]
}

# Test flow and app message quality
@test "flow and app messages are specific" {
    # Flow messages should mention flows
    [[ "$MSG_FLOW_EXECUTING" =~ "flow" ]]
    [[ "$MSG_FLOW_COMPLETED" =~ "flow" ]]
    
    # App messages should mention apps
    [[ "$MSG_APP_DEPLOYED" =~ "app" ]]
    [[ "$MSG_APP_NOT_FOUND" =~ "app" ]]
}

# Test worker message quality
@test "worker messages are clear about worker operations" {
    # Worker messages should mention workers
    [[ "$MSG_WORKER_STARTED" =~ "worker" ]]
    [[ "$MSG_WORKER_STOPPED" =~ "worker" ]]
}

# Test resource and variable message quality
@test "resource and variable messages are descriptive" {
    # Variable messages should mention variables/secrets
    [[ "$MSG_VARIABLE_CREATED" =~ "variable" ]]
    [[ "$MSG_SECRET_CREATED" =~ "secret" ]]
    
    # Resource messages should mention resources
    [[ "$MSG_RESOURCE_CREATED" =~ "resource" ]]
    [[ "$MSG_RESOURCE_NOT_FOUND" =~ "resource" ]]
}

# Test schedule message quality
@test "schedule messages are clear about scheduling operations" {
    # Schedule messages should mention schedules
    [[ "$MSG_SCHEDULE_CREATED" =~ "schedule" ]]
    [[ "$MSG_SCHEDULE_ENABLED" =~ "schedule" ]]
    [[ "$MSG_SCHEDULE_DISABLED" =~ "schedule" ]]
}
