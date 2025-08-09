#!/usr/bin/env bats
# Tests for Qdrant messages.sh configuration

# Setup for each test
setup() {
    # Load dependencies
    SCRIPT_DIR="${BATS_TEST_DIRNAME}"
    QDRANT_DIR="$(dirname "$SCRIPT_DIR")"
    
    # Set up test variables that messages might reference
    export QDRANT_PORT="6333"
    export QDRANT_GRPC_PORT="6334"
    export QDRANT_BASE_URL="http://localhost:6333"
    export QDRANT_GRPC_URL="grpc://localhost:6334"
    export QDRANT_CONTAINER_NAME="qdrant"
    export QDRANT_SERVICE_NAME="qdrant"
    export QDRANT_VERSION="1.7.4"
    export COLLECTION_NAME="test_collection"
    export SNAPSHOT_NAME="snapshot_test_2024-01-15"
    export VECTOR_SIZE="1536"
    export DISTANCE_METRIC="Cosine"
    export ERROR_MESSAGE="Test error message"
    export OPERATION_ID="123456"
    
    # Load the messages configuration to test
    source "${QDRANT_DIR}/config/messages.sh"
}

# Test messages initialization function
@test "qdrant::messages::init function is defined" {
    declare -f qdrant::messages::init >/dev/null
}

# Test installation messages
@test "QDRANT_MSG_INSTALLING is defined and meaningful" {
    qdrant::messages::init
    
    [ -n "$QDRANT_MSG_INSTALLING" ]
    [[ "$QDRANT_MSG_INSTALLING" =~ "Installing" ]] || [[ "$QDRANT_MSG_INSTALLING" =~ "install" ]]
    [[ "$QDRANT_MSG_INSTALLING" =~ "Qdrant" ]]
}

@test "QDRANT_MSG_ALREADY_INSTALLED is defined and meaningful" {
    qdrant::messages::init
    
    [ -n "$QDRANT_MSG_ALREADY_INSTALLED" ]
    [[ "$QDRANT_MSG_ALREADY_INSTALLED" =~ "already" ]]
    [[ "$QDRANT_MSG_ALREADY_INSTALLED" =~ "installed" ]]
}

@test "QDRANT_MSG_INSTALLATION_SUCCESS is defined and meaningful" {
    qdrant::messages::init
    
    [ -n "$QDRANT_MSG_INSTALLATION_SUCCESS" ]
    [[ "$QDRANT_MSG_INSTALLATION_SUCCESS" =~ "success" ]] || [[ "$QDRANT_MSG_INSTALLATION_SUCCESS" =~ "installed" ]]
    [[ "$QDRANT_MSG_INSTALLATION_SUCCESS" =~ "Qdrant" ]]
}

# Test status messages
@test "QDRANT_MSG_NOT_FOUND is defined and meaningful" {
    qdrant::messages::init
    
    [ -n "$QDRANT_MSG_NOT_FOUND" ]
    [[ "$QDRANT_MSG_NOT_FOUND" =~ "not found" ]] || [[ "$QDRANT_MSG_NOT_FOUND" =~ "not installed" ]]
}

@test "QDRANT_MSG_NOT_RUNNING is defined and meaningful" {
    qdrant::messages::init
    
    [ -n "$QDRANT_MSG_NOT_RUNNING" ]
    [[ "$QDRANT_MSG_NOT_RUNNING" =~ "not running" ]] || [[ "$QDRANT_MSG_NOT_RUNNING" =~ "stopped" ]]
}

@test "QDRANT_MSG_STATUS_OK is defined and meaningful" {
    qdrant::messages::init
    
    [ -n "$QDRANT_MSG_STATUS_OK" ]
    [[ "$QDRANT_MSG_STATUS_OK" =~ "OK" ]] || [[ "$QDRANT_MSG_STATUS_OK" =~ "running" ]] || [[ "$QDRANT_MSG_STATUS_OK" =~ "healthy" ]]
}

@test "QDRANT_MSG_HEALTHY is defined and meaningful" {
    qdrant::messages::init
    
    [ -n "$QDRANT_MSG_HEALTHY" ]
    [[ "$QDRANT_MSG_HEALTHY" =~ "healthy" ]] || [[ "$QDRANT_MSG_HEALTHY" =~ "running" ]]
}

@test "QDRANT_MSG_UNHEALTHY is defined and meaningful" {
    qdrant::messages::init
    
    [ -n "$QDRANT_MSG_UNHEALTHY" ]
    [[ "$QDRANT_MSG_UNHEALTHY" =~ "unhealthy" ]] || [[ "$QDRANT_MSG_UNHEALTHY" =~ "not responding" ]]
}

# Test API messages
@test "QDRANT_MSG_API_TEST is defined and meaningful" {
    qdrant::messages::init
    
    [ -n "$QDRANT_MSG_API_TEST" ]
    [[ "$QDRANT_MSG_API_TEST" =~ "API" ]]
    [[ "$QDRANT_MSG_API_TEST" =~ "test" ]] || [[ "$QDRANT_MSG_API_TEST" =~ "check" ]]
}

@test "QDRANT_MSG_API_SUCCESS is defined and meaningful" {
    qdrant::messages::init
    
    [ -n "$QDRANT_MSG_API_SUCCESS" ]
    [[ "$QDRANT_MSG_API_SUCCESS" =~ "API" ]]
    [[ "$QDRANT_MSG_API_SUCCESS" =~ "success" ]] || [[ "$QDRANT_MSG_API_SUCCESS" =~ "accessible" ]]
}

@test "QDRANT_MSG_API_FAILED is defined and meaningful" {
    qdrant::messages::init
    
    [ -n "$QDRANT_MSG_API_FAILED" ]
    [[ "$QDRANT_MSG_API_FAILED" =~ "API" ]]
    [[ "$QDRANT_MSG_API_FAILED" =~ "failed" ]] || [[ "$QDRANT_MSG_API_FAILED" =~ "not accessible" ]]
}

# Test collection messages
@test "QDRANT_MSG_COLLECTION_CREATED is defined and supports variable substitution" {
    qdrant::messages::init
    
    [ -n "$QDRANT_MSG_COLLECTION_CREATED" ]
    [[ "$QDRANT_MSG_COLLECTION_CREATED" =~ "collection" ]]
    [[ "$QDRANT_MSG_COLLECTION_CREATED" =~ "created" ]] || [[ "$QDRANT_MSG_COLLECTION_CREATED" =~ "success" ]]
    
    # Test variable substitution
    collection="test_collection"
    expanded_msg=$(printf "$QDRANT_MSG_COLLECTION_CREATED" "$collection")
    [[ "$expanded_msg" =~ "$collection" ]]
}

@test "QDRANT_MSG_COLLECTION_DELETED is defined and supports variable substitution" {
    qdrant::messages::init
    
    [ -n "$QDRANT_MSG_COLLECTION_DELETED" ]
    [[ "$QDRANT_MSG_COLLECTION_DELETED" =~ "collection" ]]
    [[ "$QDRANT_MSG_COLLECTION_DELETED" =~ "deleted" ]] || [[ "$QDRANT_MSG_COLLECTION_DELETED" =~ "removed" ]]
    
    # Test variable substitution
    collection="test_collection"
    expanded_msg=$(printf "$QDRANT_MSG_COLLECTION_DELETED" "$collection")
    [[ "$expanded_msg" =~ "$collection" ]]
}

@test "QDRANT_MSG_COLLECTION_NOT_FOUND is defined and supports variable substitution" {
    qdrant::messages::init
    
    [ -n "$QDRANT_MSG_COLLECTION_NOT_FOUND" ]
    [[ "$QDRANT_MSG_COLLECTION_NOT_FOUND" =~ "collection" ]]
    [[ "$QDRANT_MSG_COLLECTION_NOT_FOUND" =~ "not found" ]] || [[ "$QDRANT_MSG_COLLECTION_NOT_FOUND" =~ "does not exist" ]]
    
    # Test variable substitution
    collection="missing_collection"
    expanded_msg=$(printf "$QDRANT_MSG_COLLECTION_NOT_FOUND" "$collection")
    [[ "$expanded_msg" =~ "$collection" ]]
}

# Test vector operation messages
@test "QDRANT_MSG_VECTORS_UPSERTED is defined and meaningful" {
    qdrant::messages::init
    
    [ -n "$QDRANT_MSG_VECTORS_UPSERTED" ]
    [[ "$QDRANT_MSG_VECTORS_UPSERTED" =~ "vector" ]] || [[ "$QDRANT_MSG_VECTORS_UPSERTED" =~ "point" ]]
    [[ "$QDRANT_MSG_VECTORS_UPSERTED" =~ "upserted" ]] || [[ "$QDRANT_MSG_VECTORS_UPSERTED" =~ "inserted" ]]
}

@test "QDRANT_MSG_VECTORS_DELETED is defined and meaningful" {
    qdrant::messages::init
    
    [ -n "$QDRANT_MSG_VECTORS_DELETED" ]
    [[ "$QDRANT_MSG_VECTORS_DELETED" =~ "vector" ]] || [[ "$QDRANT_MSG_VECTORS_DELETED" =~ "point" ]]
    [[ "$QDRANT_MSG_VECTORS_DELETED" =~ "deleted" ]] || [[ "$QDRANT_MSG_VECTORS_DELETED" =~ "removed" ]]
}

@test "QDRANT_MSG_SEARCH_COMPLETED is defined and meaningful" {
    qdrant::messages::init
    
    [ -n "$QDRANT_MSG_SEARCH_COMPLETED" ]
    [[ "$QDRANT_MSG_SEARCH_COMPLETED" =~ "search" ]]
    [[ "$QDRANT_MSG_SEARCH_COMPLETED" =~ "completed" ]] || [[ "$QDRANT_MSG_SEARCH_COMPLETED" =~ "finished" ]]
}

# Test snapshot messages
@test "QDRANT_MSG_SNAPSHOT_CREATED is defined and supports variable substitution" {
    qdrant::messages::init
    
    [ -n "$QDRANT_MSG_SNAPSHOT_CREATED" ]
    [[ "$QDRANT_MSG_SNAPSHOT_CREATED" =~ "snapshot" ]]
    [[ "$QDRANT_MSG_SNAPSHOT_CREATED" =~ "created" ]] || [[ "$QDRANT_MSG_SNAPSHOT_CREATED" =~ "success" ]]
    
    # Test variable substitution
    snapshot="test_snapshot_2024-01-15"
    expanded_msg=$(printf "$QDRANT_MSG_SNAPSHOT_CREATED" "$snapshot")
    [[ "$expanded_msg" =~ "$snapshot" ]]
}

@test "QDRANT_MSG_SNAPSHOT_RESTORED is defined and supports variable substitution" {
    qdrant::messages::init
    
    [ -n "$QDRANT_MSG_SNAPSHOT_RESTORED" ]
    [[ "$QDRANT_MSG_SNAPSHOT_RESTORED" =~ "snapshot" ]]
    [[ "$QDRANT_MSG_SNAPSHOT_RESTORED" =~ "restored" ]] || [[ "$QDRANT_MSG_SNAPSHOT_RESTORED" =~ "recovered" ]]
    
    # Test variable substitution
    snapshot="test_snapshot_2024-01-15"
    expanded_msg=$(printf "$QDRANT_MSG_SNAPSHOT_RESTORED" "$snapshot")
    [[ "$expanded_msg" =~ "$snapshot" ]]
}

@test "QDRANT_MSG_SNAPSHOT_NOT_FOUND is defined and supports variable substitution" {
    qdrant::messages::init
    
    [ -n "$QDRANT_MSG_SNAPSHOT_NOT_FOUND" ]
    [[ "$QDRANT_MSG_SNAPSHOT_NOT_FOUND" =~ "snapshot" ]]
    [[ "$QDRANT_MSG_SNAPSHOT_NOT_FOUND" =~ "not found" ]] || [[ "$QDRANT_MSG_SNAPSHOT_NOT_FOUND" =~ "does not exist" ]]
    
    # Test variable substitution
    snapshot="missing_snapshot"
    expanded_msg=$(printf "$QDRANT_MSG_SNAPSHOT_NOT_FOUND" "$snapshot")
    [[ "$expanded_msg" =~ "$snapshot" ]]
}

# Test error messages
@test "QDRANT_MSG_ERR_OPERATION_FAILED is defined and meaningful" {
    qdrant::messages::init
    
    [ -n "$QDRANT_MSG_ERR_OPERATION_FAILED" ]
    [[ "$QDRANT_MSG_ERR_OPERATION_FAILED" =~ "operation" ]]
    [[ "$QDRANT_MSG_ERR_OPERATION_FAILED" =~ "failed" ]] || [[ "$QDRANT_MSG_ERR_OPERATION_FAILED" =~ "error" ]]
}

@test "QDRANT_MSG_ERR_CONNECTION_FAILED is defined and meaningful" {
    qdrant::messages::init
    
    [ -n "$QDRANT_MSG_ERR_CONNECTION_FAILED" ]
    [[ "$QDRANT_MSG_ERR_CONNECTION_FAILED" =~ "connection" ]]
    [[ "$QDRANT_MSG_ERR_CONNECTION_FAILED" =~ "failed" ]] || [[ "$QDRANT_MSG_ERR_CONNECTION_FAILED" =~ "error" ]]
}

@test "QDRANT_MSG_ERR_INVALID_CONFIG is defined and meaningful" {
    qdrant::messages::init
    
    [ -n "$QDRANT_MSG_ERR_INVALID_CONFIG" ]
    [[ "$QDRANT_MSG_ERR_INVALID_CONFIG" =~ "config" ]] || [[ "$QDRANT_MSG_ERR_INVALID_CONFIG" =~ "configuration" ]]
    [[ "$QDRANT_MSG_ERR_INVALID_CONFIG" =~ "invalid" ]] || [[ "$QDRANT_MSG_ERR_INVALID_CONFIG" =~ "error" ]]
}

# Test Docker messages
@test "QDRANT_MSG_DOCKER_NOT_AVAILABLE is defined and meaningful" {
    qdrant::messages::init
    
    [ -n "$QDRANT_MSG_DOCKER_NOT_AVAILABLE" ]
    [[ "$QDRANT_MSG_DOCKER_NOT_AVAILABLE" =~ "Docker" ]]
    [[ "$QDRANT_MSG_DOCKER_NOT_AVAILABLE" =~ "not available" ]] || [[ "$QDRANT_MSG_DOCKER_NOT_AVAILABLE" =~ "not found" ]]
}

@test "QDRANT_MSG_CONTAINER_STARTING is defined and meaningful" {
    qdrant::messages::init
    
    [ -n "$QDRANT_MSG_CONTAINER_STARTING" ]
    [[ "$QDRANT_MSG_CONTAINER_STARTING" =~ "container" ]] || [[ "$QDRANT_MSG_CONTAINER_STARTING" =~ "Starting" ]]
}

@test "QDRANT_MSG_CONTAINER_STARTED is defined and meaningful" {
    qdrant::messages::init
    
    [ -n "$QDRANT_MSG_CONTAINER_STARTED" ]
    [[ "$QDRANT_MSG_CONTAINER_STARTED" =~ "container" ]] || [[ "$QDRANT_MSG_CONTAINER_STARTED" =~ "started" ]]
}

@test "QDRANT_MSG_CONTAINER_STOPPED is defined and meaningful" {
    qdrant::messages::init
    
    [ -n "$QDRANT_MSG_CONTAINER_STOPPED" ]
    [[ "$QDRANT_MSG_CONTAINER_STOPPED" =~ "container" ]] || [[ "$QDRANT_MSG_CONTAINER_STOPPED" =~ "stopped" ]]
}

# Test service management messages
@test "QDRANT_MSG_STARTING_SERVICE is defined and meaningful" {
    qdrant::messages::init
    
    [ -n "$QDRANT_MSG_STARTING_SERVICE" ]
    [[ "$QDRANT_MSG_STARTING_SERVICE" =~ "Starting" ]]
    [[ "$QDRANT_MSG_STARTING_SERVICE" =~ "service" ]] || [[ "$QDRANT_MSG_STARTING_SERVICE" =~ "Qdrant" ]]
}

@test "QDRANT_MSG_STOPPING_SERVICE is defined and meaningful" {
    qdrant::messages::init
    
    [ -n "$QDRANT_MSG_STOPPING_SERVICE" ]
    [[ "$QDRANT_MSG_STOPPING_SERVICE" =~ "Stopping" ]]
    [[ "$QDRANT_MSG_STOPPING_SERVICE" =~ "service" ]] || [[ "$QDRANT_MSG_STOPPING_SERVICE" =~ "Qdrant" ]]
}

@test "QDRANT_MSG_RESTARTING_SERVICE is defined and meaningful" {
    qdrant::messages::init
    
    [ -n "$QDRANT_MSG_RESTARTING_SERVICE" ]
    [[ "$QDRANT_MSG_RESTARTING_SERVICE" =~ "Restarting" ]]
    [[ "$QDRANT_MSG_RESTARTING_SERVICE" =~ "service" ]] || [[ "$QDRANT_MSG_RESTARTING_SERVICE" =~ "Qdrant" ]]
}

# Test uninstallation messages
@test "QDRANT_MSG_UNINSTALLING is defined and meaningful" {
    qdrant::messages::init
    
    [ -n "$QDRANT_MSG_UNINSTALLING" ]
    [[ "$QDRANT_MSG_UNINSTALLING" =~ "Uninstalling" ]] || [[ "$QDRANT_MSG_UNINSTALLING" =~ "Removing" ]]
    [[ "$QDRANT_MSG_UNINSTALLING" =~ "Qdrant" ]]
}

@test "QDRANT_MSG_UNINSTALL_SUCCESS is defined and meaningful" {
    qdrant::messages::init
    
    [ -n "$QDRANT_MSG_UNINSTALL_SUCCESS" ]
    [[ "$QDRANT_MSG_UNINSTALL_SUCCESS" =~ "success" ]] || [[ "$QDRANT_MSG_UNINSTALL_SUCCESS" =~ "removed" ]]
    [[ "$QDRANT_MSG_UNINSTALL_SUCCESS" =~ "Qdrant" ]]
}

# Test network and port messages
@test "QDRANT_MSG_PORT_IN_USE is defined and supports variable substitution" {
    qdrant::messages::init
    
    [ -n "$QDRANT_MSG_PORT_IN_USE" ]
    [[ "$QDRANT_MSG_PORT_IN_USE" =~ "port" ]]
    [[ "$QDRANT_MSG_PORT_IN_USE" =~ "in use" ]] || [[ "$QDRANT_MSG_PORT_IN_USE" =~ "busy" ]]
    
    # Test variable substitution
    port="6333"
    expanded_msg=$(printf "$QDRANT_MSG_PORT_IN_USE" "$port")
    [[ "$expanded_msg" =~ "$port" ]]
}

@test "QDRANT_MSG_PORT_AVAILABLE is defined and meaningful" {
    qdrant::messages::init
    
    [ -n "$QDRANT_MSG_PORT_AVAILABLE" ]
    [[ "$QDRANT_MSG_PORT_AVAILABLE" =~ "port" ]]
    [[ "$QDRANT_MSG_PORT_AVAILABLE" =~ "available" ]] || [[ "$QDRANT_MSG_PORT_AVAILABLE" =~ "free" ]]
}

# Test cluster messages
@test "QDRANT_MSG_CLUSTER_ENABLED is defined and meaningful" {
    qdrant::messages::init
    
    [ -n "$QDRANT_MSG_CLUSTER_ENABLED" ]
    [[ "$QDRANT_MSG_CLUSTER_ENABLED" =~ "cluster" ]]
    [[ "$QDRANT_MSG_CLUSTER_ENABLED" =~ "enabled" ]] || [[ "$QDRANT_MSG_CLUSTER_ENABLED" =~ "active" ]]
}

@test "QDRANT_MSG_CLUSTER_DISABLED is defined and meaningful" {
    qdrant::messages::init
    
    [ -n "$QDRANT_MSG_CLUSTER_DISABLED" ]
    [[ "$QDRANT_MSG_CLUSTER_DISABLED" =~ "cluster" ]]
    [[ "$QDRANT_MSG_CLUSTER_DISABLED" =~ "disabled" ]] || [[ "$QDRANT_MSG_CLUSTER_DISABLED" =~ "inactive" ]]
}

# Test backup/restore messages
@test "QDRANT_MSG_BACKUP_CREATED is defined and meaningful" {
    qdrant::messages::init
    
    [ -n "$QDRANT_MSG_BACKUP_CREATED" ]
    [[ "$QDRANT_MSG_BACKUP_CREATED" =~ "backup" ]]
    [[ "$QDRANT_MSG_BACKUP_CREATED" =~ "created" ]] || [[ "$QDRANT_MSG_BACKUP_CREATED" =~ "completed" ]]
}

@test "QDRANT_MSG_BACKUP_RESTORED is defined and meaningful" {
    qdrant::messages::init
    
    [ -n "$QDRANT_MSG_BACKUP_RESTORED" ]
    [[ "$QDRANT_MSG_BACKUP_RESTORED" =~ "backup" ]]
    [[ "$QDRANT_MSG_BACKUP_RESTORED" =~ "restored" ]] || [[ "$QDRANT_MSG_BACKUP_RESTORED" =~ "recovered" ]]
}

# Test informational messages
@test "QDRANT_MSG_SERVICE_INFO is defined and supports variable substitution" {
    qdrant::messages::init
    
    [ -n "$QDRANT_MSG_SERVICE_INFO" ]
    [[ "$QDRANT_MSG_SERVICE_INFO" =~ "Qdrant" ]]
    
    # Test that it can include URL
    expanded_msg=$(eval "echo \"$QDRANT_MSG_SERVICE_INFO\"")
    [[ "$expanded_msg" =~ "http://" ]] || [[ "$expanded_msg" =~ "localhost" ]]
}

@test "QDRANT_MSG_AVAILABLE_AT is defined and supports variable substitution" {
    qdrant::messages::init
    
    [ -n "$QDRANT_MSG_AVAILABLE_AT" ]
    [[ "$QDRANT_MSG_AVAILABLE_AT" =~ "available" ]]
    
    # Test URL substitution
    expanded_msg=$(eval "echo \"$QDRANT_MSG_AVAILABLE_AT\"")
    [[ "$expanded_msg" =~ "http://localhost:6333" ]]
}

@test "QDRANT_MSG_API_ENDPOINT is defined and supports variable substitution" {
    qdrant::messages::init
    
    [ -n "$QDRANT_MSG_API_ENDPOINT" ]
    [[ "$QDRANT_MSG_API_ENDPOINT" =~ "API" ]]
    
    # Test API URL substitution
    expanded_msg=$(eval "echo \"$QDRANT_MSG_API_ENDPOINT\"")
    [[ "$expanded_msg" =~ "http://localhost:6333" ]]
}

@test "QDRANT_MSG_GRPC_ENDPOINT is defined and supports variable substitution" {
    qdrant::messages::init
    
    [ -n "$QDRANT_MSG_GRPC_ENDPOINT" ]
    [[ "$QDRANT_MSG_GRPC_ENDPOINT" =~ "gRPC" ]]
    
    # Test gRPC URL substitution
    expanded_msg=$(eval "echo \"$QDRANT_MSG_GRPC_ENDPOINT\"")
    [[ "$expanded_msg" =~ "grpc://localhost:6334" ]]
}

# Test usage messages
@test "QDRANT_MSG_USAGE_GUIDE is defined and meaningful" {
    qdrant::messages::init
    
    [ -n "$QDRANT_MSG_USAGE_GUIDE" ]
    [[ "$QDRANT_MSG_USAGE_GUIDE" =~ "usage" ]] || [[ "$QDRANT_MSG_USAGE_GUIDE" =~ "guide" ]]
}

@test "QDRANT_MSG_EXAMPLES_AVAILABLE is defined and meaningful" {
    qdrant::messages::init
    
    [ -n "$QDRANT_MSG_EXAMPLES_AVAILABLE" ]
    [[ "$QDRANT_MSG_EXAMPLES_AVAILABLE" =~ "example" ]]
    [[ "$QDRANT_MSG_EXAMPLES_AVAILABLE" =~ "available" ]]
}

# Test message export function
@test "qdrant::messages::export_messages exports all message variables" {
    qdrant::messages::init
    qdrant::messages::export_messages
    
    # Check that variables are exported (available in subshells)
    result=$(bash -c 'echo $QDRANT_MSG_INSTALLING')
    [ -n "$result" ]
    
    result=$(bash -c 'echo $QDRANT_MSG_SERVICE_INFO')
    [ -n "$result" ]
    
    result=$(bash -c 'echo $QDRANT_MSG_API_ENDPOINT')
    [ -n "$result" ]
}

# Test that all essential messages are defined
@test "all essential message variables are defined" {
    qdrant::messages::init
    
    [ -n "$QDRANT_MSG_INSTALLING" ]
    [ -n "$QDRANT_MSG_ALREADY_INSTALLED" ]
    [ -n "$QDRANT_MSG_NOT_FOUND" ]
    [ -n "$QDRANT_MSG_NOT_RUNNING" ]
    [ -n "$QDRANT_MSG_STATUS_OK" ]
    [ -n "$QDRANT_MSG_HEALTHY" ]
    [ -n "$QDRANT_MSG_API_TEST" ]
    [ -n "$QDRANT_MSG_API_SUCCESS" ]
    [ -n "$QDRANT_MSG_API_FAILED" ]
    [ -n "$QDRANT_MSG_DOCKER_NOT_AVAILABLE" ]
    [ -n "$QDRANT_MSG_INSTALLATION_SUCCESS" ]
    [ -n "$QDRANT_MSG_SERVICE_INFO" ]
    [ -n "$QDRANT_MSG_AVAILABLE_AT" ]
}

# Test message content quality
@test "messages contain appropriate emoji or formatting" {
    qdrant::messages::init
    
    # Check that some messages have visual indicators
    local has_visual_indicator=false
    
    if [[ "$QDRANT_MSG_INSTALLATION_SUCCESS" =~ [✅🎉✨] ]] || \
       [[ "$QDRANT_MSG_INSTALLING" =~ [🔧📦⚙️] ]] || \
       [[ "$QDRANT_MSG_HEALTHY" =~ [✅💚🟢] ]]; then
        has_visual_indicator=true
    fi
    
    [ "$has_visual_indicator" = true ]
}

@test "error messages are appropriately formatted" {
    qdrant::messages::init
    
    # Error messages should have warning indicators or clear language
    local has_error_indicator=false
    
    if [[ "$QDRANT_MSG_DOCKER_NOT_AVAILABLE" =~ [❌⚠️🔴] ]] || \
       [[ "$QDRANT_MSG_API_FAILED" =~ [❌⚠️] ]] || \
       [[ "$QDRANT_MSG_ERR_OPERATION_FAILED" =~ "ERROR" ]]; then
        has_error_indicator=true
    fi
    
    [ "$has_error_indicator" = true ]
}

# Test message variable interpolation
@test "messages with variables interpolate correctly" {
    qdrant::messages::init
    
    # Test collection messages
    collection="test_collection"
    expanded=$(printf "$QDRANT_MSG_COLLECTION_CREATED" "$collection")
    [[ "$expanded" =~ "$collection" ]]
    
    expanded=$(printf "$QDRANT_MSG_COLLECTION_NOT_FOUND" "$collection")
    [[ "$expanded" =~ "$collection" ]]
    
    # Test snapshot messages
    snapshot="test_snapshot"
    expanded=$(printf "$QDRANT_MSG_SNAPSHOT_CREATED" "$snapshot")
    [[ "$expanded" =~ "$snapshot" ]]
    
    # Test port messages
    port="6333"
    expanded=$(printf "$QDRANT_MSG_PORT_IN_USE" "$port")
    [[ "$expanded" =~ "$port" ]]
    
    # Test URL variables
    expanded=$(eval "echo \"$QDRANT_MSG_AVAILABLE_AT\"")
    [[ "$expanded" =~ "http://localhost:6333" ]]
    
    expanded=$(eval "echo \"$QDRANT_MSG_API_ENDPOINT\"")
    [[ "$expanded" =~ "http://localhost:6333" ]]
    
    expanded=$(eval "echo \"$QDRANT_MSG_GRPC_ENDPOINT\"")
    [[ "$expanded" =~ "grpc://localhost:6334" ]]
}

# Test message consistency
@test "messages use consistent service naming" {
    qdrant::messages::init
    
    # Check that service is consistently referred to
    local service_refs=0
    
    if [[ "$QDRANT_MSG_INSTALLING" =~ "Qdrant" ]]; then
        ((service_refs++))
    fi
    
    if [[ "$QDRANT_MSG_INSTALLATION_SUCCESS" =~ "Qdrant" ]]; then
        ((service_refs++))
    fi
    
    if [[ "$QDRANT_MSG_SERVICE_INFO" =~ "Qdrant" ]]; then
        ((service_refs++))
    fi
    
    # At least 2 messages should consistently name the service
    [ "$service_refs" -ge 2 ]
}

# Test vector-specific message quality
@test "vector messages are specific and informative" {
    qdrant::messages::init
    
    # Vector messages should be specific
    [[ "$QDRANT_MSG_VECTORS_UPSERTED" =~ "vector" ]] || [[ "$QDRANT_MSG_VECTORS_UPSERTED" =~ "point" ]]
    [[ "$QDRANT_MSG_VECTORS_DELETED" =~ "vector" ]] || [[ "$QDRANT_MSG_VECTORS_DELETED" =~ "point" ]]
    [[ "$QDRANT_MSG_SEARCH_COMPLETED" =~ "search" ]]
}

# Test error message specificity
@test "error messages are specific and actionable" {
    qdrant::messages::init
    
    # Error messages should be clear about what went wrong
    [[ "$QDRANT_MSG_ERR_OPERATION_FAILED" =~ "operation" ]]
    [[ "$QDRANT_MSG_ERR_CONNECTION_FAILED" =~ "connection" ]]
    [[ "$QDRANT_MSG_ERR_INVALID_CONFIG" =~ "config" ]]
}

# Test collection message quality
@test "collection messages are clear and helpful" {
    qdrant::messages::init
    
    # Collection messages should be informative
    [[ "$QDRANT_MSG_COLLECTION_CREATED" =~ "collection" ]]
    [[ "$QDRANT_MSG_COLLECTION_DELETED" =~ "collection" ]]
    [[ "$QDRANT_MSG_COLLECTION_NOT_FOUND" =~ "collection" ]]
}

# Test Docker-specific message quality
@test "Docker messages are specific and actionable" {
    qdrant::messages::init
    
    # Docker messages should be clear
    [[ "$QDRANT_MSG_DOCKER_NOT_AVAILABLE" =~ "Docker" ]]
    [[ "$QDRANT_MSG_CONTAINER_STARTING" =~ "container" ]] || [[ "$QDRANT_MSG_CONTAINER_STARTING" =~ "Starting" ]]
    [[ "$QDRANT_MSG_CONTAINER_STARTED" =~ "container" ]] || [[ "$QDRANT_MSG_CONTAINER_STARTED" =~ "started" ]]
}

# Test user-facing message quality
@test "user-facing messages are welcoming and informative" {
    qdrant::messages::init
    
    # Service info should be welcoming
    [[ "$QDRANT_MSG_SERVICE_INFO" =~ "Qdrant" ]]
    [[ "$QDRANT_MSG_AVAILABLE_AT" =~ "available" ]]
    [[ "$QDRANT_MSG_API_ENDPOINT" =~ "API" ]]
    [[ "$QDRANT_MSG_GRPC_ENDPOINT" =~ "gRPC" ]]
}

# Test cluster message quality
@test "cluster messages are clear about cluster operations" {
    qdrant::messages::init
    
    # Cluster messages should mention clusters
    [[ "$QDRANT_MSG_CLUSTER_ENABLED" =~ "cluster" ]]
    [[ "$QDRANT_MSG_CLUSTER_DISABLED" =~ "cluster" ]]
}

# Test backup message quality
@test "backup messages are clear about backup operations" {
    qdrant::messages::init
    
    # Backup messages should mention backups
    [[ "$QDRANT_MSG_BACKUP_CREATED" =~ "backup" ]]
    [[ "$QDRANT_MSG_BACKUP_RESTORED" =~ "backup" ]]
}

# Test snapshot message quality
@test "snapshot messages are clear about snapshot operations" {
    qdrant::messages::init
    
    # Snapshot messages should mention snapshots
    [[ "$QDRANT_MSG_SNAPSHOT_CREATED" =~ "snapshot" ]]
    [[ "$QDRANT_MSG_SNAPSHOT_RESTORED" =~ "snapshot" ]]
    [[ "$QDRANT_MSG_SNAPSHOT_NOT_FOUND" =~ "snapshot" ]]
}

# Test service management message quality
@test "service management messages are descriptive" {
    qdrant::messages::init
    
    # Service management messages should mention services
    [[ "$QDRANT_MSG_STARTING_SERVICE" =~ "service" ]] || [[ "$QDRANT_MSG_STARTING_SERVICE" =~ "Qdrant" ]]
    [[ "$QDRANT_MSG_STOPPING_SERVICE" =~ "service" ]] || [[ "$QDRANT_MSG_STOPPING_SERVICE" =~ "Qdrant" ]]
    [[ "$QDRANT_MSG_RESTARTING_SERVICE" =~ "service" ]] || [[ "$QDRANT_MSG_RESTARTING_SERVICE" =~ "Qdrant" ]]
}

# Test uninstallation message quality
@test "uninstallation messages are clear and confirmatory" {
    qdrant::messages::init
    
    # Uninstallation messages should be clear
    [[ "$QDRANT_MSG_UNINSTALLING" =~ "Uninstalling" ]] || [[ "$QDRANT_MSG_UNINSTALLING" =~ "Removing" ]]
    [[ "$QDRANT_MSG_UNINSTALL_SUCCESS" =~ "success" ]] || [[ "$QDRANT_MSG_UNINSTALL_SUCCESS" =~ "removed" ]]
}

# Test API message quality
@test "API messages are specific about API operations" {
    qdrant::messages::init
    
    # API messages should mention API
    [[ "$QDRANT_MSG_API_TEST" =~ "API" ]]
    [[ "$QDRANT_MSG_API_SUCCESS" =~ "API" ]]
    [[ "$QDRANT_MSG_API_FAILED" =~ "API" ]]
}

# Test port message quality
@test "port messages are clear about port status" {
    qdrant::messages::init
    
    # Port messages should mention ports
    [[ "$QDRANT_MSG_PORT_IN_USE" =~ "port" ]]
    [[ "$QDRANT_MSG_PORT_AVAILABLE" =~ "port" ]]
}

# Test usage message quality
@test "usage messages are helpful and informative" {
    qdrant::messages::init
    
    # Usage messages should be helpful
    [[ "$QDRANT_MSG_USAGE_GUIDE" =~ "usage" ]] || [[ "$QDRANT_MSG_USAGE_GUIDE" =~ "guide" ]]
    [[ "$QDRANT_MSG_EXAMPLES_AVAILABLE" =~ "example" ]]
}

# Test message length appropriateness
@test "messages are appropriately sized" {
    qdrant::messages::init
    
    # Messages should not be too short or too long
    [ ${#QDRANT_MSG_INSTALLING} -ge 10 ]
    [ ${#QDRANT_MSG_INSTALLING} -le 200 ]
    
    [ ${#QDRANT_MSG_SERVICE_INFO} -ge 15 ]
    [ ${#QDRANT_MSG_SERVICE_INFO} -le 300 ]
}

# Test message tone consistency
@test "messages maintain consistent tone" {
    qdrant::messages::init
    
    # Success messages should be positive
    [[ "$QDRANT_MSG_INSTALLATION_SUCCESS" =~ "success" ]] || [[ "$QDRANT_MSG_INSTALLATION_SUCCESS" =~ "completed" ]]
    [[ "$QDRANT_MSG_HEALTHY" =~ "healthy" ]] || [[ "$QDRANT_MSG_HEALTHY" =~ "running" ]]
    
    # Error messages should be clear but not alarming
    [[ "$QDRANT_MSG_NOT_FOUND" =~ "not found" ]]
    [[ "$QDRANT_MSG_NOT_RUNNING" =~ "not running" ]]
}
