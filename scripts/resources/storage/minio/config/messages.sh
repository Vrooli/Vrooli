#!/usr/bin/env bash
# MinIO User-Facing Messages
# All user messages in one place for consistency

#######################################
# Initialize message constants
# Idempotent - safe to call multiple times
#######################################
minio::messages::init() {
    # Success messages
    [[ -z "${MSG_INSTALL_SUCCESS:-}" ]] && readonly MSG_INSTALL_SUCCESS="✅ MinIO installed successfully"
    [[ -z "${MSG_START_SUCCESS:-}" ]] && readonly MSG_START_SUCCESS="✅ MinIO started successfully"
    [[ -z "${MSG_STOP_SUCCESS:-}" ]] && readonly MSG_STOP_SUCCESS="✅ MinIO stopped successfully"
    [[ -z "${MSG_RESTART_SUCCESS:-}" ]] && readonly MSG_RESTART_SUCCESS="✅ MinIO restarted successfully"
    [[ -z "${MSG_UNINSTALL_SUCCESS:-}" ]] && readonly MSG_UNINSTALL_SUCCESS="✅ MinIO uninstalled successfully"
    [[ -z "${MSG_BUCKET_CREATED:-}" ]] && readonly MSG_BUCKET_CREATED="✅ Bucket created successfully"
    [[ -z "${MSG_CREDENTIALS_GENERATED:-}" ]] && readonly MSG_CREDENTIALS_GENERATED="✅ Secure credentials generated"
    [[ -z "${MSG_BUCKETS_INITIALIZED:-}" ]] && readonly MSG_BUCKETS_INITIALIZED="✅ Default buckets initialized"
    
    # Status messages
    [[ -z "${MSG_HEALTHY:-}" ]] && readonly MSG_HEALTHY="✅ MinIO API is healthy"
    [[ -z "${MSG_RUNNING:-}" ]] && readonly MSG_RUNNING="✅ MinIO container is running"
    [[ -z "${MSG_CONSOLE_AVAILABLE:-}" ]] && readonly MSG_CONSOLE_AVAILABLE="MinIO Console available at"
    
    # Info messages
    [[ -z "${MSG_CHECKING_DOCKER:-}" ]] && readonly MSG_CHECKING_DOCKER="Checking Docker availability..."
    [[ -z "${MSG_CHECKING_PORTS:-}" ]] && readonly MSG_CHECKING_PORTS="Checking port availability..."
    [[ -z "${MSG_PULLING_IMAGE:-}" ]] && readonly MSG_PULLING_IMAGE="Pulling MinIO Docker image..."
    [[ -z "${MSG_CREATING_DIRECTORIES:-}" ]] && readonly MSG_CREATING_DIRECTORIES="Creating data directories..."
    [[ -z "${MSG_STARTING_CONTAINER:-}" ]] && readonly MSG_STARTING_CONTAINER="Starting MinIO container..."
    [[ -z "${MSG_WAITING_STARTUP:-}" ]] && readonly MSG_WAITING_STARTUP="Waiting for MinIO to start..."
    [[ -z "${MSG_CREATING_BUCKETS:-}" ]] && readonly MSG_CREATING_BUCKETS="Creating default buckets..."
    [[ -z "${MSG_CONFIGURING_POLICIES:-}" ]] && readonly MSG_CONFIGURING_POLICIES="Configuring bucket policies..."
    
    # Warning messages
    [[ -z "${MSG_PORT_IN_USE:-}" ]] && readonly MSG_PORT_IN_USE="Port is already in use"
    [[ -z "${MSG_CONTAINER_EXISTS:-}" ]] && readonly MSG_CONTAINER_EXISTS="MinIO container already exists"
    [[ -z "${MSG_LOW_DISK_SPACE:-}" ]] && readonly MSG_LOW_DISK_SPACE="Low disk space detected"
    [[ -z "${MSG_USING_DEFAULT_CREDS:-}" ]] && readonly MSG_USING_DEFAULT_CREDS="⚠️  Using default credentials (not secure for production)"
    
    # Error messages
    [[ -z "${MSG_DOCKER_NOT_FOUND:-}" ]] && readonly MSG_DOCKER_NOT_FOUND="Docker is not installed or not running"
    [[ -z "${MSG_INSTALL_FAILED:-}" ]] && readonly MSG_INSTALL_FAILED="MinIO installation failed"
    [[ -z "${MSG_START_FAILED:-}" ]] && readonly MSG_START_FAILED="Failed to start MinIO"
    [[ -z "${MSG_HEALTH_CHECK_FAILED:-}" ]] && readonly MSG_HEALTH_CHECK_FAILED="MinIO health check failed"
    [[ -z "${MSG_BUCKET_CREATE_FAILED:-}" ]] && readonly MSG_BUCKET_CREATE_FAILED="Failed to create bucket"
    [[ -z "${MSG_INSUFFICIENT_DISK:-}" ]] && readonly MSG_INSUFFICIENT_DISK="Insufficient disk space"
    
    # Help messages
    [[ -z "${MSG_HELP_ACCESS:-}" ]] && readonly MSG_HELP_ACCESS="Access MinIO Console at ${MINIO_CONSOLE_URL:-http://localhost:9001}"
    [[ -z "${MSG_HELP_CREDENTIALS:-}" ]] && readonly MSG_HELP_CREDENTIALS="Use 'manage.sh --action show-credentials' to view access credentials"
    [[ -z "${MSG_HELP_PORT_CONFLICT:-}" ]] && readonly MSG_HELP_PORT_CONFLICT="To use a different port, set MINIO_CUSTOM_PORT environment variable"
}