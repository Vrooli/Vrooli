#!/usr/bin/env bash
# Qdrant User-Facing Messages
# All user messages in one place for consistency

#######################################
# Initialize message constants
# Idempotent - safe to call multiple times
#######################################
qdrant::messages::init() {
    # Success messages
    [[ -z "${MSG_INSTALL_SUCCESS:-}" ]] && readonly MSG_INSTALL_SUCCESS="✅ Qdrant installed successfully"
    [[ -z "${MSG_START_SUCCESS:-}" ]] && readonly MSG_START_SUCCESS="✅ Qdrant started successfully"
    [[ -z "${MSG_STOP_SUCCESS:-}" ]] && readonly MSG_STOP_SUCCESS="✅ Qdrant stopped successfully"
    [[ -z "${MSG_RESTART_SUCCESS:-}" ]] && readonly MSG_RESTART_SUCCESS="✅ Qdrant restarted successfully"
    [[ -z "${MSG_UNINSTALL_SUCCESS:-}" ]] && readonly MSG_UNINSTALL_SUCCESS="✅ Qdrant uninstalled successfully"
    [[ -z "${MSG_COLLECTION_CREATED:-}" ]] && readonly MSG_COLLECTION_CREATED="✅ Collection created successfully"
    [[ -z "${MSG_COLLECTIONS_INITIALIZED:-}" ]] && readonly MSG_COLLECTIONS_INITIALIZED="✅ Default collections initialized"
    [[ -z "${MSG_SNAPSHOT_CREATED:-}" ]] && readonly MSG_SNAPSHOT_CREATED="✅ Snapshot created successfully"
    [[ -z "${MSG_BACKUP_RESTORED:-}" ]] && readonly MSG_BACKUP_RESTORED="✅ Backup restored successfully"
    
    # Status messages
    [[ -z "${MSG_HEALTHY:-}" ]] && readonly MSG_HEALTHY="✅ Qdrant API is healthy"
    [[ -z "${MSG_RUNNING:-}" ]] && readonly MSG_RUNNING="✅ Qdrant container is running"
    [[ -z "${MSG_WEB_UI_AVAILABLE:-}" ]] && readonly MSG_WEB_UI_AVAILABLE="Qdrant Web UI available at"
    [[ -z "${MSG_GRPC_API_AVAILABLE:-}" ]] && readonly MSG_GRPC_API_AVAILABLE="Qdrant gRPC API available at"
    
    # Info messages
    [[ -z "${MSG_CHECKING_DOCKER:-}" ]] && readonly MSG_CHECKING_DOCKER="Checking Docker availability..."
    [[ -z "${MSG_CHECKING_PORTS:-}" ]] && readonly MSG_CHECKING_PORTS="Checking port availability..."
    [[ -z "${MSG_PULLING_IMAGE:-}" ]] && readonly MSG_PULLING_IMAGE="Pulling Qdrant Docker image..."
    [[ -z "${MSG_CREATING_DIRECTORIES:-}" ]] && readonly MSG_CREATING_DIRECTORIES="Creating data directories..."
    [[ -z "${MSG_STARTING_CONTAINER:-}" ]] && readonly MSG_STARTING_CONTAINER="Starting Qdrant container..."
    [[ -z "${MSG_WAITING_STARTUP:-}" ]] && readonly MSG_WAITING_STARTUP="Waiting for Qdrant to start..."
    [[ -z "${MSG_CREATING_COLLECTIONS:-}" ]] && readonly MSG_CREATING_COLLECTIONS="Creating default collections..."
    [[ -z "${MSG_CONFIGURING_COLLECTIONS:-}" ]] && readonly MSG_CONFIGURING_COLLECTIONS="Configuring collection parameters..."
    [[ -z "${MSG_CHECKING_COLLECTIONS:-}" ]] && readonly MSG_CHECKING_COLLECTIONS="Checking collection status..."
    [[ -z "${MSG_CREATING_SNAPSHOT:-}" ]] && readonly MSG_CREATING_SNAPSHOT="Creating data snapshot..."
    
    # Warning messages
    [[ -z "${MSG_PORT_IN_USE:-}" ]] && readonly MSG_PORT_IN_USE="Port is already in use"
    [[ -z "${MSG_GRPC_PORT_IN_USE:-}" ]] && readonly MSG_GRPC_PORT_IN_USE="gRPC port is already in use"
    [[ -z "${MSG_CONTAINER_EXISTS:-}" ]] && readonly MSG_CONTAINER_EXISTS="Qdrant container already exists"
    [[ -z "${MSG_LOW_DISK_SPACE:-}" ]] && readonly MSG_LOW_DISK_SPACE="Low disk space detected"
    [[ -z "${MSG_NO_API_KEY:-}" ]] && readonly MSG_NO_API_KEY="⚠️  No API key configured (unauthenticated access)"
    [[ -z "${MSG_COLLECTION_EXISTS:-}" ]] && readonly MSG_COLLECTION_EXISTS="Collection already exists"
    [[ -z "${MSG_COLLECTION_NOT_EMPTY:-}" ]] && readonly MSG_COLLECTION_NOT_EMPTY="Collection contains vectors"
    
    # Error messages
    [[ -z "${MSG_DOCKER_NOT_FOUND:-}" ]] && readonly MSG_DOCKER_NOT_FOUND="Docker is not installed or not running"
    [[ -z "${MSG_INSTALL_FAILED:-}" ]] && readonly MSG_INSTALL_FAILED="Qdrant installation failed"
    [[ -z "${MSG_START_FAILED:-}" ]] && readonly MSG_START_FAILED="Failed to start Qdrant"
    [[ -z "${MSG_HEALTH_CHECK_FAILED:-}" ]] && readonly MSG_HEALTH_CHECK_FAILED="Qdrant health check failed"
    [[ -z "${MSG_COLLECTION_CREATE_FAILED:-}" ]] && readonly MSG_COLLECTION_CREATE_FAILED="Failed to create collection"
    [[ -z "${MSG_COLLECTION_DELETE_FAILED:-}" ]] && readonly MSG_COLLECTION_DELETE_FAILED="Failed to delete collection"
    [[ -z "${MSG_INSUFFICIENT_DISK:-}" ]] && readonly MSG_INSUFFICIENT_DISK="Insufficient disk space"
    [[ -z "${MSG_API_NOT_ACCESSIBLE:-}" ]] && readonly MSG_API_NOT_ACCESSIBLE="Qdrant API is not accessible"
    [[ -z "${MSG_SNAPSHOT_FAILED:-}" ]] && readonly MSG_SNAPSHOT_FAILED="Failed to create snapshot"
    [[ -z "${MSG_RESTORE_FAILED:-}" ]] && readonly MSG_RESTORE_FAILED="Failed to restore backup"
    
    # Help messages
    [[ -z "${MSG_HELP_ACCESS:-}" ]] && readonly MSG_HELP_ACCESS="Access Qdrant Web UI at ${QDRANT_BASE_URL:-http://localhost:6333}/dashboard"
    [[ -z "${MSG_HELP_API:-}" ]] && readonly MSG_HELP_API="REST API available at ${QDRANT_BASE_URL:-http://localhost:6333}"
    [[ -z "${MSG_HELP_GRPC:-}" ]] && readonly MSG_HELP_GRPC="gRPC API available at ${QDRANT_GRPC_URL:-grpc://localhost:6334}"
    [[ -z "${MSG_HELP_COLLECTIONS:-}" ]] && readonly MSG_HELP_COLLECTIONS="Use 'manage.sh --action list-collections' to view collections"
    [[ -z "${MSG_HELP_PORT_CONFLICT:-}" ]] && readonly MSG_HELP_PORT_CONFLICT="To use different ports, set QDRANT_CUSTOM_PORT and QDRANT_CUSTOM_GRPC_PORT environment variables"
    [[ -z "${MSG_HELP_AUTHENTICATION:-}" ]] && readonly MSG_HELP_AUTHENTICATION="To enable authentication, set QDRANT_CUSTOM_API_KEY environment variable"
    [[ -z "${MSG_HELP_BACKUP:-}" ]] && readonly MSG_HELP_BACKUP="Use 'manage.sh --action backup' to create snapshots"
}