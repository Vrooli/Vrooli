#!/usr/bin/env bash

# Unstructured.io Resource User-Facing Messages
# This file contains all user-facing messages for consistent communication

# Prevent double sourcing
[[ -n "${UNSTRUCTURED_IO_MESSAGES_SOURCED:-}" ]] && return 0
UNSTRUCTURED_IO_MESSAGES_SOURCED=1

#######################################
# Installation Messages
#######################################
readonly MSG_UNSTRUCTURED_IO_INSTALLING="Installing Unstructured.io API service..."
readonly MSG_UNSTRUCTURED_IO_ALREADY_INSTALLED="Unstructured.io container already exists (use --force yes to reinstall)"
readonly MSG_UNSTRUCTURED_IO_INSTALL_SUCCESS="✅ Unstructured.io installation completed successfully"
readonly MSG_UNSTRUCTURED_IO_INSTALL_FAILED="❌ Unstructured.io installation failed"

readonly MSG_DOCKER_REQUIRED="Docker is required to run Unstructured.io"
readonly MSG_PULLING_IMAGE="Pulling Unstructured.io Docker image..."
readonly MSG_IMAGE_PULL_SUCCESS="✅ Docker image pulled successfully"
readonly MSG_IMAGE_PULL_FAILED="❌ Failed to pull Docker image"
readonly MSG_CONTAINER_CREATE_SUCCESS="✅ Unstructured.io container created successfully"
readonly MSG_CONTAINER_CREATE_FAILED="❌ Failed to create container"
readonly MSG_CONTAINER_START_SUCCESS="✅ Unstructured.io container started successfully"
readonly MSG_CONTAINER_START_FAILED="❌ Failed to start container"

#######################################
# Status Messages
#######################################
readonly MSG_UNSTRUCTURED_IO_RUNNING="✅ Unstructured.io is running and healthy on port $UNSTRUCTURED_IO_PORT"
readonly MSG_UNSTRUCTURED_IO_STARTING="⏳ Unstructured.io is starting up..."
readonly MSG_UNSTRUCTURED_IO_NOT_RUNNING="❌ Unstructured.io container is not running"
readonly MSG_UNSTRUCTURED_IO_NOT_FOUND="❌ Unstructured.io container not found"
readonly MSG_UNSTRUCTURED_IO_UNHEALTHY="⚠️  Unstructured.io is running but API health check failed"

readonly MSG_STATUS_CONTAINER_OK="✅ Container '$UNSTRUCTURED_IO_CONTAINER_NAME' exists"
readonly MSG_STATUS_CONTAINER_RUNNING="✅ Container is running"
readonly MSG_STATUS_PORT_OK="✅ Service listening on port $UNSTRUCTURED_IO_PORT"
readonly MSG_STATUS_API_OK="✅ API is healthy and responsive"
readonly MSG_STATUS_FORMATS_SUPPORTED="✅ Supporting ${#UNSTRUCTURED_IO_SUPPORTED_FORMATS[@]} document formats"

#######################################
# Processing Messages
#######################################
readonly MSG_PROCESSING_FILE="Processing document: \$filename"
readonly MSG_PROCESSING_SUCCESS="✅ Document processed successfully"
readonly MSG_PROCESSING_FAILED="❌ Document processing failed"
readonly MSG_PROCESSING_TIMEOUT="⏱️  Processing timed out after $UNSTRUCTURED_IO_TIMEOUT_SECONDS seconds"
readonly MSG_FILE_TOO_LARGE="❌ File size exceeds maximum limit of $UNSTRUCTURED_IO_MAX_FILE_SIZE ($(echo $UNSTRUCTURED_IO_MAX_FILE_SIZE_BYTES | numfmt --to=iec-i --suffix=B 2>/dev/null || echo '52428800 bytes'))"
readonly MSG_UNSUPPORTED_FORMAT="❌ Unsupported file format: \$format. Supported formats: ${UNSTRUCTURED_IO_SUPPORTED_FORMATS[*]}"
readonly MSG_BATCH_PROCESSING="Processing \$count documents in batch mode..."

#######################################
# API Messages
#######################################
readonly MSG_API_ENDPOINT_INFO="API endpoints available:"
readonly MSG_API_PROCESS="  - POST $UNSTRUCTURED_IO_PROCESS_ENDPOINT - Process single document"
readonly MSG_API_BATCH="  - POST $UNSTRUCTURED_IO_BATCH_ENDPOINT - Batch processing"
readonly MSG_API_HEALTH="  - GET $UNSTRUCTURED_IO_HEALTH_ENDPOINT - Health check"
readonly MSG_API_METRICS="  - GET /metrics - Processing metrics (if available)"

#######################################
# Configuration Messages
#######################################
readonly MSG_CONFIG_STRATEGY="Processing strategy: $UNSTRUCTURED_IO_DEFAULT_STRATEGY"
readonly MSG_CONFIG_LANGUAGES="OCR languages: $UNSTRUCTURED_IO_DEFAULT_LANGUAGES"
readonly MSG_CONFIG_MEMORY="Memory limit: $UNSTRUCTURED_IO_MEMORY_LIMIT"
readonly MSG_CONFIG_CPU="CPU limit: $UNSTRUCTURED_IO_CPU_LIMIT"
readonly MSG_CONFIG_TIMEOUT="Request timeout: $UNSTRUCTURED_IO_TIMEOUT_SECONDS seconds"

#######################################
# Error Messages
#######################################
readonly MSG_ERROR_DOCKER_NOT_RUNNING="❌ Docker daemon is not running"
readonly MSG_ERROR_PORT_IN_USE="❌ Port $UNSTRUCTURED_IO_PORT is already in use"
readonly MSG_ERROR_CONTAINER_CONFLICT="❌ Container name conflict: $UNSTRUCTURED_IO_CONTAINER_NAME"
readonly MSG_ERROR_NETWORK_ISSUE="❌ Unable to connect to Unstructured.io API"
readonly MSG_ERROR_INVALID_STRATEGY="❌ Invalid processing strategy: \$strategy"
readonly MSG_ERROR_HTTP_413="❌ File too large. Maximum file size is $UNSTRUCTURED_IO_MAX_FILE_SIZE"
readonly MSG_ERROR_HTTP_415="❌ Unsupported media type. Check that the file format is supported"
readonly MSG_ERROR_HTTP_422="❌ Invalid processing parameters. Check strategy and language settings"
readonly MSG_ERROR_HTTP_500="❌ Server error. The document may be corrupted or contain unsupported content"
readonly MSG_ERROR_HTTP_GENERIC="❌ Request failed with HTTP status: \$http_code"

#######################################
# Uninstall Messages
#######################################
readonly MSG_UNINSTALLING="Uninstalling Unstructured.io..."
readonly MSG_CONTAINER_STOP_SUCCESS="✅ Container stopped successfully"
readonly MSG_CONTAINER_REMOVE_SUCCESS="✅ Container removed successfully"
readonly MSG_IMAGE_REMOVE_SUCCESS="✅ Docker image removed successfully"
readonly MSG_UNINSTALL_SUCCESS="✅ Unstructured.io uninstalled successfully"
readonly MSG_UNINSTALL_FAILED="❌ Uninstallation failed"

#######################################
# Export messages function
#######################################
unstructured_io::export_messages() {
    # Export all message variables for use in other modules
    # Installation messages
    export MSG_UNSTRUCTURED_IO_INSTALLING MSG_UNSTRUCTURED_IO_ALREADY_INSTALLED MSG_UNSTRUCTURED_IO_INSTALL_SUCCESS
    export MSG_UNSTRUCTURED_IO_INSTALL_FAILED MSG_DOCKER_REQUIRED MSG_PULLING_IMAGE MSG_IMAGE_PULL_SUCCESS
    export MSG_IMAGE_PULL_FAILED MSG_CONTAINER_CREATE_SUCCESS MSG_CONTAINER_CREATE_FAILED
    export MSG_CONTAINER_START_SUCCESS MSG_CONTAINER_START_FAILED
    
    # Status messages
    export MSG_UNSTRUCTURED_IO_RUNNING MSG_UNSTRUCTURED_IO_STARTING MSG_UNSTRUCTURED_IO_NOT_RUNNING
    export MSG_UNSTRUCTURED_IO_NOT_FOUND MSG_UNSTRUCTURED_IO_UNHEALTHY
    export MSG_STATUS_CONTAINER_OK MSG_STATUS_CONTAINER_RUNNING MSG_STATUS_PORT_OK
    export MSG_STATUS_API_OK MSG_STATUS_FORMATS_SUPPORTED
    
    # Processing messages
    export MSG_PROCESSING_FILE MSG_PROCESSING_SUCCESS MSG_PROCESSING_FAILED MSG_PROCESSING_TIMEOUT
    export MSG_FILE_TOO_LARGE MSG_UNSUPPORTED_FORMAT MSG_BATCH_PROCESSING
    
    # API messages
    export MSG_API_ENDPOINT_INFO MSG_API_PROCESS MSG_API_BATCH MSG_API_HEALTH MSG_API_METRICS
    
    # Configuration messages
    export MSG_CONFIG_STRATEGY MSG_CONFIG_LANGUAGES MSG_CONFIG_MEMORY MSG_CONFIG_CPU MSG_CONFIG_TIMEOUT
    
    # Error messages
    export MSG_ERROR_DOCKER_NOT_RUNNING MSG_ERROR_PORT_IN_USE MSG_ERROR_CONTAINER_CONFLICT
    export MSG_ERROR_NETWORK_ISSUE MSG_ERROR_INVALID_STRATEGY MSG_ERROR_HTTP_413 MSG_ERROR_HTTP_415
    export MSG_ERROR_HTTP_422 MSG_ERROR_HTTP_500 MSG_ERROR_HTTP_GENERIC
    
    # Uninstall messages
    export MSG_UNINSTALLING MSG_CONTAINER_STOP_SUCCESS MSG_CONTAINER_REMOVE_SUCCESS
    export MSG_IMAGE_REMOVE_SUCCESS MSG_UNINSTALL_SUCCESS MSG_UNINSTALL_FAILED
}

# Export function for subshell availability
export -f unstructured_io::export_messages