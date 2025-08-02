#!/usr/bin/env bats
# Tests for Judge0 messages.sh configuration

# Setup for each test
setup() {
    # Load dependencies
    SCRIPT_DIR="$(dirname "${BATS_TEST_FILENAME}")"
    JUDGE0_DIR="$(dirname "$SCRIPT_DIR")"
    
    # Set up test variables that messages might reference
    export JUDGE0_PORT="2358"
    export JUDGE0_BASE_URL="http://localhost:2358"
    export JUDGE0_CONTAINER_NAME="vrooli-judge0"
    export JUDGE0_SERVICE_NAME="judge0"
    export JUDGE0_VERSION="1.13.1"
    export LANGUAGE_NAME="Python"
    export LANGUAGE_ID="92"
    export SUBMISSION_TOKEN="abc123-def456-ghi789"
    export CODE_SAMPLE='print("Hello, World!")'
    export ERROR_MESSAGE="Compilation failed"
    
    # Load the messages configuration to test
    source "${JUDGE0_DIR}/config/messages.sh"
}

# Test installation messages
@test "JUDGE0_MSG_INSTALLING is defined and meaningful" {
    [ -n "$JUDGE0_MSG_INSTALLING" ]
    [[ "$JUDGE0_MSG_INSTALLING" =~ "Installing" ]]
    [[ "$JUDGE0_MSG_INSTALLING" =~ "Judge0" ]]
}

@test "JUDGE0_MSG_ALREADY_INSTALLED is defined and meaningful" {
    [ -n "$JUDGE0_MSG_ALREADY_INSTALLED" ]
    [[ "$JUDGE0_MSG_ALREADY_INSTALLED" =~ "already" ]]
    [[ "$JUDGE0_MSG_ALREADY_INSTALLED" =~ "installed" ]]
}

@test "JUDGE0_MSG_INSTALLATION_SUCCESS is defined and meaningful" {
    [ -n "$JUDGE0_MSG_INSTALLATION_SUCCESS" ]
    [[ "$JUDGE0_MSG_INSTALLATION_SUCCESS" =~ "success" ]] || [[ "$JUDGE0_MSG_INSTALLATION_SUCCESS" =~ "installed" ]]
    [[ "$JUDGE0_MSG_INSTALLATION_SUCCESS" =~ "Judge0" ]]
}

# Test status messages
@test "JUDGE0_MSG_NOT_FOUND is defined and meaningful" {
    [ -n "$JUDGE0_MSG_NOT_FOUND" ]
    [[ "$JUDGE0_MSG_NOT_FOUND" =~ "not found" ]]
}

@test "JUDGE0_MSG_NOT_RUNNING is defined and meaningful" {
    [ -n "$JUDGE0_MSG_NOT_RUNNING" ]
    [[ "$JUDGE0_MSG_NOT_RUNNING" =~ "not running" ]]
}

@test "JUDGE0_MSG_STATUS_OK is defined and meaningful" {
    [ -n "$JUDGE0_MSG_STATUS_OK" ]
    [[ "$JUDGE0_MSG_STATUS_OK" =~ "OK" ]] || [[ "$JUDGE0_MSG_STATUS_OK" =~ "running" ]]
}

@test "JUDGE0_MSG_HEALTHY is defined and meaningful" {
    [ -n "$JUDGE0_MSG_HEALTHY" ]
    [[ "$JUDGE0_MSG_HEALTHY" =~ "healthy" ]] || [[ "$JUDGE0_MSG_HEALTHY" =~ "running" ]]
}

@test "JUDGE0_MSG_UNHEALTHY is defined and meaningful" {
    [ -n "$JUDGE0_MSG_UNHEALTHY" ]
    [[ "$JUDGE0_MSG_UNHEALTHY" =~ "unhealthy" ]] || [[ "$JUDGE0_MSG_UNHEALTHY" =~ "not responding" ]]
}

# Test API messages
@test "JUDGE0_MSG_API_TEST is defined and meaningful" {
    [ -n "$JUDGE0_MSG_API_TEST" ]
    [[ "$JUDGE0_MSG_API_TEST" =~ "API" ]]
    [[ "$JUDGE0_MSG_API_TEST" =~ "test" ]] || [[ "$JUDGE0_MSG_API_TEST" =~ "check" ]]
}

@test "JUDGE0_MSG_API_SUCCESS is defined and meaningful" {
    [ -n "$JUDGE0_MSG_API_SUCCESS" ]
    [[ "$JUDGE0_MSG_API_SUCCESS" =~ "API" ]]
    [[ "$JUDGE0_MSG_API_SUCCESS" =~ "success" ]] || [[ "$JUDGE0_MSG_API_SUCCESS" =~ "accessible" ]]
}

@test "JUDGE0_MSG_API_FAILED is defined and meaningful" {
    [ -n "$JUDGE0_MSG_API_FAILED" ]
    [[ "$JUDGE0_MSG_API_FAILED" =~ "API" ]]
    [[ "$JUDGE0_MSG_API_FAILED" =~ "failed" ]] || [[ "$JUDGE0_MSG_API_FAILED" =~ "not accessible" ]]
}

@test "JUDGE0_MSG_API_SUBMISSION is defined and meaningful" {
    [ -n "$JUDGE0_MSG_API_SUBMISSION" ]
    [[ "$JUDGE0_MSG_API_SUBMISSION" =~ "submission" ]] || [[ "$JUDGE0_MSG_API_SUBMISSION" =~ "executing" ]]
}

@test "JUDGE0_MSG_API_RESULT is defined and meaningful" {
    [ -n "$JUDGE0_MSG_API_RESULT" ]
    [[ "$JUDGE0_MSG_API_RESULT" =~ "result" ]] || [[ "$JUDGE0_MSG_API_RESULT" =~ "execution" ]]
}

# Test language messages
@test "JUDGE0_MSG_LANG_NOT_FOUND is defined and supports variable substitution" {
    [ -n "$JUDGE0_MSG_LANG_NOT_FOUND" ]
    [[ "$JUDGE0_MSG_LANG_NOT_FOUND" =~ "language" ]]
    [[ "$JUDGE0_MSG_LANG_NOT_FOUND" =~ "not found" ]] || [[ "$JUDGE0_MSG_LANG_NOT_FOUND" =~ "not supported" ]]
    
    # Test variable substitution
    language="brainfuck"
    expanded_msg=$(printf "$JUDGE0_MSG_LANG_NOT_FOUND" "$language")
    [[ "$expanded_msg" =~ "$language" ]]
}

@test "JUDGE0_MSG_LANG_SUPPORTED is defined and meaningful" {
    [ -n "$JUDGE0_MSG_LANG_SUPPORTED" ]
    [[ "$JUDGE0_MSG_LANG_SUPPORTED" =~ "language" ]]
    [[ "$JUDGE0_MSG_LANG_SUPPORTED" =~ "supported" ]] || [[ "$JUDGE0_MSG_LANG_SUPPORTED" =~ "available" ]]
}

# Test execution error messages
@test "JUDGE0_MSG_ERR_SUBMISSION is defined and meaningful" {
    [ -n "$JUDGE0_MSG_ERR_SUBMISSION" ]
    [[ "$JUDGE0_MSG_ERR_SUBMISSION" =~ "submission" ]]
    [[ "$JUDGE0_MSG_ERR_SUBMISSION" =~ "error" ]] || [[ "$JUDGE0_MSG_ERR_SUBMISSION" =~ "failed" ]]
}

@test "JUDGE0_MSG_ERR_TIME_LIMIT is defined and meaningful" {
    [ -n "$JUDGE0_MSG_ERR_TIME_LIMIT" ]
    [[ "$JUDGE0_MSG_ERR_TIME_LIMIT" =~ "time" ]]
    [[ "$JUDGE0_MSG_ERR_TIME_LIMIT" =~ "limit" ]] || [[ "$JUDGE0_MSG_ERR_TIME_LIMIT" =~ "exceeded" ]]
}

@test "JUDGE0_MSG_ERR_MEMORY_LIMIT is defined and meaningful" {
    [ -n "$JUDGE0_MSG_ERR_MEMORY_LIMIT" ]
    [[ "$JUDGE0_MSG_ERR_MEMORY_LIMIT" =~ "memory" ]]
    [[ "$JUDGE0_MSG_ERR_MEMORY_LIMIT" =~ "limit" ]] || [[ "$JUDGE0_MSG_ERR_MEMORY_LIMIT" =~ "exceeded" ]]
}

@test "JUDGE0_MSG_ERR_COMPILE is defined and meaningful" {
    [ -n "$JUDGE0_MSG_ERR_COMPILE" ]
    [[ "$JUDGE0_MSG_ERR_COMPILE" =~ "compil" ]]
    [[ "$JUDGE0_MSG_ERR_COMPILE" =~ "error" ]] || [[ "$JUDGE0_MSG_ERR_COMPILE" =~ "failed" ]]
}

@test "JUDGE0_MSG_ERR_RUNTIME is defined and meaningful" {
    [ -n "$JUDGE0_MSG_ERR_RUNTIME" ]
    [[ "$JUDGE0_MSG_ERR_RUNTIME" =~ "runtime" ]]
    [[ "$JUDGE0_MSG_ERR_RUNTIME" =~ "error" ]]
}

# Test Docker messages
@test "JUDGE0_MSG_DOCKER_NOT_AVAILABLE is defined and meaningful" {
    [ -n "$JUDGE0_MSG_DOCKER_NOT_AVAILABLE" ]
    [[ "$JUDGE0_MSG_DOCKER_NOT_AVAILABLE" =~ "Docker" ]]
    [[ "$JUDGE0_MSG_DOCKER_NOT_AVAILABLE" =~ "not available" ]] || [[ "$JUDGE0_MSG_DOCKER_NOT_AVAILABLE" =~ "not found" ]]
}

@test "JUDGE0_MSG_CONTAINER_STARTING is defined and meaningful" {
    [ -n "$JUDGE0_MSG_CONTAINER_STARTING" ]
    [[ "$JUDGE0_MSG_CONTAINER_STARTING" =~ "container" ]] || [[ "$JUDGE0_MSG_CONTAINER_STARTING" =~ "Starting" ]]
}

@test "JUDGE0_MSG_CONTAINER_STARTED is defined and meaningful" {
    [ -n "$JUDGE0_MSG_CONTAINER_STARTED" ]
    [[ "$JUDGE0_MSG_CONTAINER_STARTED" =~ "container" ]] || [[ "$JUDGE0_MSG_CONTAINER_STARTED" =~ "started" ]]
}

@test "JUDGE0_MSG_CONTAINER_STOPPED is defined and meaningful" {
    [ -n "$JUDGE0_MSG_CONTAINER_STOPPED" ]
    [[ "$JUDGE0_MSG_CONTAINER_STOPPED" =~ "container" ]] || [[ "$JUDGE0_MSG_CONTAINER_STOPPED" =~ "stopped" ]]
}

# Test service management messages
@test "JUDGE0_MSG_STARTING_SERVICE is defined and meaningful" {
    [ -n "$JUDGE0_MSG_STARTING_SERVICE" ]
    [[ "$JUDGE0_MSG_STARTING_SERVICE" =~ "Starting" ]]
    [[ "$JUDGE0_MSG_STARTING_SERVICE" =~ "service" ]] || [[ "$JUDGE0_MSG_STARTING_SERVICE" =~ "Judge0" ]]
}

@test "JUDGE0_MSG_STOPPING_SERVICE is defined and meaningful" {
    [ -n "$JUDGE0_MSG_STOPPING_SERVICE" ]
    [[ "$JUDGE0_MSG_STOPPING_SERVICE" =~ "Stopping" ]]
    [[ "$JUDGE0_MSG_STOPPING_SERVICE" =~ "service" ]] || [[ "$JUDGE0_MSG_STOPPING_SERVICE" =~ "Judge0" ]]
}

@test "JUDGE0_MSG_RESTARTING_SERVICE is defined and meaningful" {
    [ -n "$JUDGE0_MSG_RESTARTING_SERVICE" ]
    [[ "$JUDGE0_MSG_RESTARTING_SERVICE" =~ "Restarting" ]]
    [[ "$JUDGE0_MSG_RESTARTING_SERVICE" =~ "service" ]] || [[ "$JUDGE0_MSG_RESTARTING_SERVICE" =~ "Judge0" ]]
}

# Test uninstallation messages
@test "JUDGE0_MSG_UNINSTALLING is defined and meaningful" {
    [ -n "$JUDGE0_MSG_UNINSTALLING" ]
    [[ "$JUDGE0_MSG_UNINSTALLING" =~ "Uninstalling" ]] || [[ "$JUDGE0_MSG_UNINSTALLING" =~ "Removing" ]]
    [[ "$JUDGE0_MSG_UNINSTALLING" =~ "Judge0" ]]
}

@test "JUDGE0_MSG_UNINSTALL_SUCCESS is defined and meaningful" {
    [ -n "$JUDGE0_MSG_UNINSTALL_SUCCESS" ]
    [[ "$JUDGE0_MSG_UNINSTALL_SUCCESS" =~ "success" ]] || [[ "$JUDGE0_MSG_UNINSTALL_SUCCESS" =~ "removed" ]]
    [[ "$JUDGE0_MSG_UNINSTALL_SUCCESS" =~ "Judge0" ]]
}

# Test network and port messages
@test "JUDGE0_MSG_PORT_IN_USE is defined and supports variable substitution" {
    [ -n "$JUDGE0_MSG_PORT_IN_USE" ]
    [[ "$JUDGE0_MSG_PORT_IN_USE" =~ "port" ]]
    [[ "$JUDGE0_MSG_PORT_IN_USE" =~ "in use" ]] || [[ "$JUDGE0_MSG_PORT_IN_USE" =~ "busy" ]]
    
    # Test variable substitution
    port="2358"
    expanded_msg=$(printf "$JUDGE0_MSG_PORT_IN_USE" "$port")
    [[ "$expanded_msg" =~ "$port" ]]
}

@test "JUDGE0_MSG_PORT_AVAILABLE is defined and meaningful" {
    [ -n "$JUDGE0_MSG_PORT_AVAILABLE" ]
    [[ "$JUDGE0_MSG_PORT_AVAILABLE" =~ "port" ]]
    [[ "$JUDGE0_MSG_PORT_AVAILABLE" =~ "available" ]] || [[ "$JUDGE0_MSG_PORT_AVAILABLE" =~ "free" ]]
}

# Test worker messages
@test "JUDGE0_MSG_WORKERS_STARTING is defined and meaningful" {
    [ -n "$JUDGE0_MSG_WORKERS_STARTING" ]
    [[ "$JUDGE0_MSG_WORKERS_STARTING" =~ "worker" ]]
    [[ "$JUDGE0_MSG_WORKERS_STARTING" =~ "starting" ]] || [[ "$JUDGE0_MSG_WORKERS_STARTING" =~ "initializing" ]]
}

@test "JUDGE0_MSG_WORKERS_READY is defined and meaningful" {
    [ -n "$JUDGE0_MSG_WORKERS_READY" ]
    [[ "$JUDGE0_MSG_WORKERS_READY" =~ "worker" ]]
    [[ "$JUDGE0_MSG_WORKERS_READY" =~ "ready" ]] || [[ "$JUDGE0_MSG_WORKERS_READY" =~ "initialized" ]]
}

# Test configuration messages
@test "JUDGE0_MSG_CONFIG_GENERATED is defined and meaningful" {
    [ -n "$JUDGE0_MSG_CONFIG_GENERATED" ]
    [[ "$JUDGE0_MSG_CONFIG_GENERATED" =~ "config" ]]
    [[ "$JUDGE0_MSG_CONFIG_GENERATED" =~ "generated" ]] || [[ "$JUDGE0_MSG_CONFIG_GENERATED" =~ "created" ]]
}

@test "JUDGE0_MSG_API_KEY_GENERATED is defined and meaningful" {
    [ -n "$JUDGE0_MSG_API_KEY_GENERATED" ]
    [[ "$JUDGE0_MSG_API_KEY_GENERATED" =~ "API key" ]]
    [[ "$JUDGE0_MSG_API_KEY_GENERATED" =~ "generated" ]] || [[ "$JUDGE0_MSG_API_KEY_GENERATED" =~ "created" ]]
}

# Test security messages
@test "JUDGE0_MSG_SECURITY_CONFIGURED is defined and meaningful" {
    [ -n "$JUDGE0_MSG_SECURITY_CONFIGURED" ]
    [[ "$JUDGE0_MSG_SECURITY_CONFIGURED" =~ "security" ]]
    [[ "$JUDGE0_MSG_SECURITY_CONFIGURED" =~ "configured" ]] || [[ "$JUDGE0_MSG_SECURITY_CONFIGURED" =~ "enabled" ]]
}

@test "JUDGE0_MSG_LIMITS_ENFORCED is defined and meaningful" {
    [ -n "$JUDGE0_MSG_LIMITS_ENFORCED" ]
    [[ "$JUDGE0_MSG_LIMITS_ENFORCED" =~ "limits" ]] || [[ "$JUDGE0_MSG_LIMITS_ENFORCED" =~ "resource" ]]
    [[ "$JUDGE0_MSG_LIMITS_ENFORCED" =~ "enforced" ]] || [[ "$JUDGE0_MSG_LIMITS_ENFORCED" =~ "applied" ]]
}

# Test backup/restore messages
@test "JUDGE0_MSG_BACKUP_CREATED is defined and meaningful" {
    [ -n "$JUDGE0_MSG_BACKUP_CREATED" ]
    [[ "$JUDGE0_MSG_BACKUP_CREATED" =~ "backup" ]]
    [[ "$JUDGE0_MSG_BACKUP_CREATED" =~ "created" ]] || [[ "$JUDGE0_MSG_BACKUP_CREATED" =~ "completed" ]]
}

@test "JUDGE0_MSG_BACKUP_RESTORED is defined and meaningful" {
    [ -n "$JUDGE0_MSG_BACKUP_RESTORED" ]
    [[ "$JUDGE0_MSG_BACKUP_RESTORED" =~ "backup" ]]
    [[ "$JUDGE0_MSG_BACKUP_RESTORED" =~ "restored" ]] || [[ "$JUDGE0_MSG_BACKUP_RESTORED" =~ "recovered" ]]
}

# Test informational messages
@test "JUDGE0_MSG_SERVICE_INFO is defined and supports variable substitution" {
    [ -n "$JUDGE0_MSG_SERVICE_INFO" ]
    [[ "$JUDGE0_MSG_SERVICE_INFO" =~ "Judge0" ]]
    
    # Test that it can include URL
    expanded_msg=$(eval "echo \"$JUDGE0_MSG_SERVICE_INFO\"")
    [[ "$expanded_msg" =~ "http://" ]] || [[ "$expanded_msg" =~ "localhost" ]]
}

@test "JUDGE0_MSG_AVAILABLE_AT is defined and supports variable substitution" {
    [ -n "$JUDGE0_MSG_AVAILABLE_AT" ]
    [[ "$JUDGE0_MSG_AVAILABLE_AT" =~ "available" ]]
    
    # Test URL substitution
    expanded_msg=$(eval "echo \"$JUDGE0_MSG_AVAILABLE_AT\"")
    [[ "$expanded_msg" =~ "http://localhost:2358" ]]
}

@test "JUDGE0_MSG_API_ENDPOINT is defined and supports variable substitution" {
    [ -n "$JUDGE0_MSG_API_ENDPOINT" ]
    [[ "$JUDGE0_MSG_API_ENDPOINT" =~ "API" ]]
    
    # Test API URL substitution
    expanded_msg=$(eval "echo \"$JUDGE0_MSG_API_ENDPOINT\"")
    [[ "$expanded_msg" =~ "http://localhost:2358" ]]
}

# Test usage messages
@test "JUDGE0_MSG_USAGE_GUIDE is defined and meaningful" {
    [ -n "$JUDGE0_MSG_USAGE_GUIDE" ]
    [[ "$JUDGE0_MSG_USAGE_GUIDE" =~ "usage" ]] || [[ "$JUDGE0_MSG_USAGE_GUIDE" =~ "guide" ]]
}

@test "JUDGE0_MSG_EXAMPLES_AVAILABLE is defined and meaningful" {
    [ -n "$JUDGE0_MSG_EXAMPLES_AVAILABLE" ]
    [[ "$JUDGE0_MSG_EXAMPLES_AVAILABLE" =~ "example" ]]
    [[ "$JUDGE0_MSG_EXAMPLES_AVAILABLE" =~ "available" ]]
}

# Test message export function
@test "judge0::export_messages exports all message variables" {
    judge0::export_messages
    
    # Check that variables are exported (available in subshells)
    result=$(bash -c 'echo $JUDGE0_MSG_INSTALLING')
    [ -n "$result" ]
    
    result=$(bash -c 'echo $JUDGE0_MSG_SERVICE_INFO')
    [ -n "$result" ]
}

# Test that all essential messages are defined
@test "all essential message variables are defined" {
    [ -n "$JUDGE0_MSG_INSTALLING" ]
    [ -n "$JUDGE0_MSG_ALREADY_INSTALLED" ]
    [ -n "$JUDGE0_MSG_NOT_FOUND" ]
    [ -n "$JUDGE0_MSG_NOT_RUNNING" ]
    [ -n "$JUDGE0_MSG_STATUS_OK" ]
    [ -n "$JUDGE0_MSG_HEALTHY" ]
    [ -n "$JUDGE0_MSG_API_TEST" ]
    [ -n "$JUDGE0_MSG_API_SUCCESS" ]
    [ -n "$JUDGE0_MSG_API_FAILED" ]
    [ -n "$JUDGE0_MSG_DOCKER_NOT_AVAILABLE" ]
    [ -n "$JUDGE0_MSG_INSTALLATION_SUCCESS" ]
    [ -n "$JUDGE0_MSG_SERVICE_INFO" ]
    [ -n "$JUDGE0_MSG_AVAILABLE_AT" ]
}

# Test message content quality
@test "messages contain appropriate emoji or formatting" {
    # Check that some messages have visual indicators
    local has_visual_indicator=false
    
    if [[ "$JUDGE0_MSG_INSTALLATION_SUCCESS" =~ [✅🎉✨] ]] || \
       [[ "$JUDGE0_MSG_INSTALLING" =~ [🔧📦⚙️] ]] || \
       [[ "$JUDGE0_MSG_HEALTHY" =~ [✅💚🟢] ]]; then
        has_visual_indicator=true
    fi
    
    [ "$has_visual_indicator" = true ]
}

@test "error messages are appropriately formatted" {
    # Error messages should have warning indicators or clear language
    local has_error_indicator=false
    
    if [[ "$JUDGE0_MSG_DOCKER_NOT_AVAILABLE" =~ [❌⚠️🔴] ]] || \
       [[ "$JUDGE0_MSG_API_FAILED" =~ [❌⚠️] ]] || \
       [[ "$JUDGE0_MSG_ERR_SUBMISSION" =~ "ERROR" ]]; then
        has_error_indicator=true
    fi
    
    [ "$has_error_indicator" = true ]
}

# Test message variable interpolation
@test "messages with variables interpolate correctly" {
    # Test language not found message
    language="brainfuck"
    expanded=$(printf "$JUDGE0_MSG_LANG_NOT_FOUND" "$language")
    [[ "$expanded" =~ "$language" ]]
    
    # Test port in use message
    port="2358"
    expanded=$(printf "$JUDGE0_MSG_PORT_IN_USE" "$port")
    [[ "$expanded" =~ "$port" ]]
    
    # Test URL variable
    expanded=$(eval "echo \"$JUDGE0_MSG_AVAILABLE_AT\"")
    [[ "$expanded" =~ "http://localhost:2358" ]]
    
    # Test API endpoint variable
    expanded=$(eval "echo \"$JUDGE0_MSG_API_ENDPOINT\"")
    [[ "$expanded" =~ "http://localhost:2358" ]]
}

# Test message consistency
@test "messages use consistent service naming" {
    # Check that service is consistently referred to
    local service_refs=0
    
    if [[ "$JUDGE0_MSG_INSTALLING" =~ "Judge0" ]]; then
        ((service_refs++))
    fi
    
    if [[ "$JUDGE0_MSG_INSTALLATION_SUCCESS" =~ "Judge0" ]]; then
        ((service_refs++))
    fi
    
    if [[ "$JUDGE0_MSG_SERVICE_INFO" =~ "Judge0" ]]; then
        ((service_refs++))
    fi
    
    # At least 2 messages should consistently name the service
    [ "$service_refs" -ge 2 ]
}

# Test execution-specific message quality
@test "execution messages are specific and informative" {
    # Execution messages should be specific
    [[ "$JUDGE0_MSG_API_SUBMISSION" =~ "submission" ]] || [[ "$JUDGE0_MSG_API_SUBMISSION" =~ "executing" ]]
    [[ "$JUDGE0_MSG_API_RESULT" =~ "result" ]] || [[ "$JUDGE0_MSG_API_RESULT" =~ "execution" ]]
    [[ "$JUDGE0_MSG_ERR_SUBMISSION" =~ "submission" ]]
}

# Test error message specificity
@test "error messages are specific and actionable" {
    # Error messages should be clear about what went wrong
    [[ "$JUDGE0_MSG_ERR_TIME_LIMIT" =~ "time" ]]
    [[ "$JUDGE0_MSG_ERR_MEMORY_LIMIT" =~ "memory" ]]
    [[ "$JUDGE0_MSG_ERR_COMPILE" =~ "compil" ]]
    [[ "$JUDGE0_MSG_ERR_RUNTIME" =~ "runtime" ]]
}

# Test language message quality
@test "language messages are clear and helpful" {
    # Language messages should be informative
    [[ "$JUDGE0_MSG_LANG_NOT_FOUND" =~ "language" ]]
    [[ "$JUDGE0_MSG_LANG_SUPPORTED" =~ "language" ]]
}

# Test Docker-specific message quality
@test "Docker messages are specific and actionable" {
    # Docker messages should be clear
    [[ "$JUDGE0_MSG_DOCKER_NOT_AVAILABLE" =~ "Docker" ]]
    [[ "$JUDGE0_MSG_CONTAINER_STARTING" =~ "container" ]] || [[ "$JUDGE0_MSG_CONTAINER_STARTING" =~ "Starting" ]]
    [[ "$JUDGE0_MSG_CONTAINER_STARTED" =~ "container" ]] || [[ "$JUDGE0_MSG_CONTAINER_STARTED" =~ "started" ]]
}

# Test user-facing message quality
@test "user-facing messages are welcoming and informative" {
    # Service info should be welcoming
    [[ "$JUDGE0_MSG_SERVICE_INFO" =~ "Judge0" ]]
    [[ "$JUDGE0_MSG_AVAILABLE_AT" =~ "available" ]]
    [[ "$JUDGE0_MSG_API_ENDPOINT" =~ "API" ]]
}

# Test worker message quality
@test "worker messages are clear about worker operations" {
    # Worker messages should mention workers
    [[ "$JUDGE0_MSG_WORKERS_STARTING" =~ "worker" ]]
    [[ "$JUDGE0_MSG_WORKERS_READY" =~ "worker" ]]
}

# Test configuration message quality
@test "configuration messages are descriptive" {
    # Configuration messages should mention configuration
    [[ "$JUDGE0_MSG_CONFIG_GENERATED" =~ "config" ]]
    [[ "$JUDGE0_MSG_API_KEY_GENERATED" =~ "API key" ]]
    [[ "$JUDGE0_MSG_SECURITY_CONFIGURED" =~ "security" ]]
}

# Test backup message quality
@test "backup messages are clear about backup operations" {
    # Backup messages should mention backups
    [[ "$JUDGE0_MSG_BACKUP_CREATED" =~ "backup" ]]
    [[ "$JUDGE0_MSG_BACKUP_RESTORED" =~ "backup" ]]
}
