#!/usr/bin/env bats
# Tests for Unstructured.io messages.sh configuration

# Setup for each test
setup() {
    # Load dependencies
    SCRIPT_DIR="${BATS_TEST_DIRNAME}"
    UNSTRUCTURED_IO_DIR="$(dirname "$SCRIPT_DIR")"
    
    # Set up test variables that messages might reference
    export UNSTRUCTURED_IO_PORT="9999"
    export UNSTRUCTURED_IO_BASE_URL="http://localhost:9999"
    export UNSTRUCTURED_IO_CONTAINER_NAME="vrooli-unstructured-io"
    export UNSTRUCTURED_IO_SERVICE_NAME="unstructured-io"
    
    # Load the messages configuration to test
    source "${UNSTRUCTURED_IO_DIR}/config/messages.sh"
}

# Test installation messages
@test "MSG_UNSTRUCTURED_IO_INSTALLING is defined and meaningful" {
    [ -n "$MSG_UNSTRUCTURED_IO_INSTALLING" ]
    [[ "$MSG_UNSTRUCTURED_IO_INSTALLING" =~ "Installing" ]]
    [[ "$MSG_UNSTRUCTURED_IO_INSTALLING" =~ "Unstructured.io" ]]
}

@test "MSG_UNSTRUCTURED_IO_ALREADY_INSTALLED is defined and meaningful" {
    [ -n "$MSG_UNSTRUCTURED_IO_ALREADY_INSTALLED" ]
    [[ "$MSG_UNSTRUCTURED_IO_ALREADY_INSTALLED" =~ "already" ]]
    [[ "$MSG_UNSTRUCTURED_IO_ALREADY_INSTALLED" =~ "installed" ]]
}

# Test status messages
@test "MSG_UNSTRUCTURED_IO_NOT_FOUND is defined and meaningful" {
    [ -n "$MSG_UNSTRUCTURED_IO_NOT_FOUND" ]
    [[ "$MSG_UNSTRUCTURED_IO_NOT_FOUND" =~ "not found" ]]
}

@test "MSG_UNSTRUCTURED_IO_NOT_RUNNING is defined and meaningful" {
    [ -n "$MSG_UNSTRUCTURED_IO_NOT_RUNNING" ]
    [[ "$MSG_UNSTRUCTURED_IO_NOT_RUNNING" =~ "not running" ]]
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

# Test processing messages
@test "MSG_PROCESSING_FILE is defined and supports variable substitution" {
    [ -n "$MSG_PROCESSING_FILE" ]
    [[ "$MSG_PROCESSING_FILE" =~ "Processing" ]]
    
    # Test variable substitution
    filename="test.pdf"
    expanded_msg=$(eval "echo \"$MSG_PROCESSING_FILE\"")
    [[ "$expanded_msg" =~ "$filename" ]]
}

@test "MSG_PROCESSING_COMPLETE is defined and meaningful" {
    [ -n "$MSG_PROCESSING_COMPLETE" ]
    [[ "$MSG_PROCESSING_COMPLETE" =~ "complete" || "$MSG_PROCESSING_COMPLETE" =~ "finished" ]]
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
    port="9999"
    expanded_msg=$(eval "echo \"$MSG_PORT_IN_USE\"")
    [[ "$expanded_msg" =~ "$port" ]]
}

@test "MSG_FILE_NOT_SUPPORTED is defined and meaningful" {
    [ -n "$MSG_FILE_NOT_SUPPORTED" ]
    [[ "$MSG_FILE_NOT_SUPPORTED" =~ "not supported" ]]
}

@test "MSG_FILE_TOO_LARGE is defined and meaningful" {
    [ -n "$MSG_FILE_TOO_LARGE" ]
    [[ "$MSG_FILE_TOO_LARGE" =~ "too large" || "$MSG_FILE_TOO_LARGE" =~ "size" ]]
}

# Test success messages
@test "MSG_INSTALLATION_SUCCESS is defined and meaningful" {
    [ -n "$MSG_INSTALLATION_SUCCESS" ]
    [[ "$MSG_INSTALLATION_SUCCESS" =~ "success" || "$MSG_INSTALLATION_SUCCESS" =~ "installed" ]]
    [[ "$MSG_INSTALLATION_SUCCESS" =~ "Unstructured.io" ]]
}

@test "MSG_SERVICE_HEALTHY is defined and meaningful" {
    [ -n "$MSG_SERVICE_HEALTHY" ]
    [[ "$MSG_SERVICE_HEALTHY" =~ "healthy" || "$MSG_SERVICE_HEALTHY" =~ "running" ]]
}

# Test information messages
@test "MSG_SERVICE_INFO is defined and supports variable substitution" {
    [ -n "$MSG_SERVICE_INFO" ]
    [[ "$MSG_SERVICE_INFO" =~ "Unstructured.io" ]]
    
    # Test that it can include URL
    expanded_msg=$(eval "echo \"$MSG_SERVICE_INFO\"")
    [[ "$expanded_msg" =~ "http://" || "$expanded_msg" =~ "localhost" ]]
}

@test "MSG_AVAILABLE_AT is defined and supports variable substitution" {
    [ -n "$MSG_AVAILABLE_AT" ]
    [[ "$MSG_AVAILABLE_AT" =~ "available" ]]
    
    # Test URL substitution
    expanded_msg=$(eval "echo \"$MSG_AVAILABLE_AT\"")
    [[ "$expanded_msg" =~ "http://localhost:9999" ]]
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

# Test message export function
@test "unstructured_io::export_messages exports all message variables" {
    unstructured_io::export_messages
    
    # Check that variables are exported (available in subshells)
    result=$(bash -c 'echo $MSG_UNSTRUCTURED_IO_INSTALLING')
    [ -n "$result" ]
    
    result=$(bash -c 'echo $MSG_SERVICE_INFO')
    [ -n "$result" ]
}

# Test that all required messages are defined
@test "all essential message variables are defined" {
    [ -n "$MSG_UNSTRUCTURED_IO_INSTALLING" ]
    [ -n "$MSG_UNSTRUCTURED_IO_ALREADY_INSTALLED" ]
    [ -n "$MSG_UNSTRUCTURED_IO_NOT_FOUND" ]
    [ -n "$MSG_UNSTRUCTURED_IO_NOT_RUNNING" ]
    [ -n "$MSG_STATUS_CONTAINER_OK" ]
    [ -n "$MSG_STATUS_CONTAINER_RUNNING" ]
    [ -n "$MSG_PROCESSING_FILE" ]
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
       [[ "$MSG_UNSTRUCTURED_IO_INSTALLING" =~ [🔧📦⚙️] ]] || \
       [[ "$MSG_SERVICE_HEALTHY" =~ [✅💚🟢] ]]; then
        has_visual_indicator=true
    fi
    
    [ "$has_visual_indicator" = true ]
}

@test "error messages are appropriately formatted" {
    # Error messages should have warning indicators or clear language
    local has_error_indicator=false
    
    if [[ "$MSG_DOCKER_NOT_AVAILABLE" =~ [❌⚠️🔴] ]] || \
       [[ "$MSG_FILE_NOT_SUPPORTED" =~ [❌⚠️] ]] || \
       [[ "$MSG_DOCKER_NOT_AVAILABLE" =~ "ERROR" ]]; then
        has_error_indicator=true
    fi
    
    [ "$has_error_indicator" = true ]
}

# Test message variable interpolation
@test "messages with variables interpolate correctly" {
    # Test filename variable
    filename="document.pdf"
    expanded=$(eval "echo \"$MSG_PROCESSING_FILE\"")
    [[ "$expanded" =~ "$filename" ]]
    
    # Test port variable  
    expanded=$(eval "echo \"$MSG_PORT_IN_USE\"")
    [[ "$expanded" =~ "9999" ]]
    
    # Test URL variable
    expanded=$(eval "echo \"$MSG_AVAILABLE_AT\"")
    [[ "$expanded" =~ "http://localhost:9999" ]]
}

# Test message consistency
@test "messages use consistent service naming" {
    # Check that service is consistently referred to
    local service_refs=0
    
    if [[ "$MSG_UNSTRUCTURED_IO_INSTALLING" =~ "Unstructured.io" ]]; then
        ((service_refs++))
    fi
    
    if [[ "$MSG_INSTALLATION_SUCCESS" =~ "Unstructured.io" ]]; then
        ((service_refs++))
    fi
    
    if [[ "$MSG_SERVICE_INFO" =~ "Unstructured.io" ]]; then
        ((service_refs++))
    fi
    
    # At least 2 messages should consistently name the service
    [ "$service_refs" -ge 2 ]
}
