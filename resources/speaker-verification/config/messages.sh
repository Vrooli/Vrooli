#!/usr/bin/env bash
# Speaker Verification User Messages
# All user-facing messages centralized

#######################################
# Export user-facing messages
# Idempotent - safe to call multiple times
#######################################
messages::export_messages() {
    # Success messages (only set if not already defined)
    [[ -z "${MSG_INSTALL_SUCCESS:-}" ]] && readonly MSG_INSTALL_SUCCESS="✅ Speaker Verification installed successfully"
    [[ -z "${MSG_START_SUCCESS:-}" ]] && readonly MSG_START_SUCCESS="✅ Speaker Verification started successfully"
    [[ -z "${MSG_STOP_SUCCESS:-}" ]] && readonly MSG_STOP_SUCCESS="✅ Speaker Verification stopped successfully"
    [[ -z "${MSG_RESTART_SUCCESS:-}" ]] && readonly MSG_RESTART_SUCCESS="✅ Speaker Verification restarted successfully"
    [[ -z "${MSG_UNINSTALL_SUCCESS:-}" ]] && readonly MSG_UNINSTALL_SUCCESS="✅ Speaker Verification uninstalled successfully"
    [[ -z "${MSG_CONTAINER_STARTED:-}" ]] && readonly MSG_CONTAINER_STARTED="Speaker Verification container started"
    [[ -z "${MSG_DIRECTORIES_CREATED:-}" ]] && readonly MSG_DIRECTORIES_CREATED="Speaker Verification directories created"
    [[ -z "${MSG_CONTAINER_REMOVED:-}" ]] && readonly MSG_CONTAINER_REMOVED="Speaker Verification container removed"
    [[ -z "${MSG_HEALTHY:-}" ]] && readonly MSG_HEALTHY="✅ Speaker Verification API is healthy"
    [[ -z "${MSG_RUNNING:-}" ]] && readonly MSG_RUNNING="✅ Speaker Verification container is running"

    # Error messages (only set if not already defined)
    [[ -z "${MSG_DOCKER_NOT_FOUND:-}" ]] && readonly MSG_DOCKER_NOT_FOUND="Docker is not installed"
    [[ -z "${MSG_DOCKER_NOT_RUNNING:-}" ]] && readonly MSG_DOCKER_NOT_RUNNING="Docker daemon is not running"
    [[ -z "${MSG_DOCKER_NO_PERMISSIONS:-}" ]] && readonly MSG_DOCKER_NO_PERMISSIONS="Current user doesn't have Docker permissions"
    [[ -z "${MSG_PORT_IN_USE:-}" ]] && readonly MSG_PORT_IN_USE="Port ${SPEAKER_VERIFICATION_PORT} is already in use"
    [[ -z "${MSG_INSTALL_FAILED:-}" ]] && readonly MSG_INSTALL_FAILED="❌ Speaker Verification installation failed"
    [[ -z "${MSG_START_FAILED:-}" ]] && readonly MSG_START_FAILED="Failed to start Speaker Verification"
    [[ -z "${MSG_STOP_FAILED:-}" ]] && readonly MSG_STOP_FAILED="Failed to stop Speaker Verification"
    [[ -z "${MSG_CONTAINER_NOT_EXISTS:-}" ]] && readonly MSG_CONTAINER_NOT_EXISTS="Speaker Verification container does not exist"
    [[ -z "${MSG_NOT_INSTALLED:-}" ]] && readonly MSG_NOT_INSTALLED="❌ Speaker Verification is not installed"
    [[ -z "${MSG_NOT_RUNNING:-}" ]] && readonly MSG_NOT_RUNNING="Speaker Verification is not running"
    [[ -z "${MSG_NOT_HEALTHY:-}" ]] && readonly MSG_NOT_HEALTHY="Speaker Verification is not running or healthy"
    [[ -z "${MSG_HEALTH_CHECK_FAILED:-}" ]] && readonly MSG_HEALTH_CHECK_FAILED="⚠️  Speaker Verification API health check failed"
    [[ -z "${MSG_STARTUP_TIMEOUT:-}" ]] && readonly MSG_STARTUP_TIMEOUT="Speaker Verification failed to start within timeout"
    [[ -z "${MSG_CREATE_DIRS_FAILED:-}" ]] && readonly MSG_CREATE_DIRS_FAILED="Failed to create Speaker Verification directories"
    [[ -z "${MSG_START_CONTAINER_FAILED:-}" ]] && readonly MSG_START_CONTAINER_FAILED="Failed to start Speaker Verification container"
    [[ -z "${MSG_GPU_NOT_AVAILABLE:-}" ]] && readonly MSG_GPU_NOT_AVAILABLE="⚠️  GPU not available, falling back to CPU"

    # Info messages (only set if not already defined)
    [[ -z "${MSG_CHECKING_STATUS:-}" ]] && readonly MSG_CHECKING_STATUS="🔍 Checking Speaker Verification status..."
    [[ -z "${MSG_PULLING_IMAGE:-}" ]] && readonly MSG_PULLING_IMAGE="📥 Building Speaker Verification image..."
    [[ -z "${MSG_CREATING_DIRS:-}" ]] && readonly MSG_CREATING_DIRS="Creating Speaker Verification data directories..."
    [[ -z "${MSG_STARTING_CONTAINER:-}" ]] && readonly MSG_STARTING_CONTAINER="Starting Speaker Verification container..."
    [[ -z "${MSG_WAITING_STARTUP:-}" ]] && readonly MSG_WAITING_STARTUP="Waiting for Speaker Verification to start..."
    [[ -z "${MSG_WAITING_INIT:-}" ]] && readonly MSG_WAITING_INIT="Waiting for Speaker Verification model to load..."
    [[ -z "${MSG_STOPPING:-}" ]] && readonly MSG_STOPPING="Stopping Speaker Verification..."
    [[ -z "${MSG_STARTING:-}" ]] && readonly MSG_STARTING="Starting Speaker Verification..."
    [[ -z "${MSG_RESTARTING:-}" ]] && readonly MSG_RESTARTING="Restarting Speaker Verification..."
    [[ -z "${MSG_REMOVING_CONTAINER:-}" ]] && readonly MSG_REMOVING_CONTAINER="Removing Speaker Verification container..."
    [[ -z "${MSG_SHOWING_LOGS:-}" ]] && readonly MSG_SHOWING_LOGS="Showing Speaker Verification logs (Ctrl+C to exit)..."

    # Warning messages (only set if not already defined)
    [[ -z "${MSG_ALREADY_INSTALLED:-}" ]] && readonly MSG_ALREADY_INSTALLED="Speaker Verification is already installed and running"
    [[ -z "${MSG_ALREADY_RUNNING:-}" ]] && readonly MSG_ALREADY_RUNNING="Speaker Verification is already running on port ${SPEAKER_VERIFICATION_PORT}"
    [[ -z "${MSG_EXISTS_NOT_RUNNING:-}" ]] && readonly MSG_EXISTS_NOT_RUNNING="⚠️  Speaker Verification container exists but is not running"
    [[ -z "${MSG_STARTED_NOT_HEALTHY:-}" ]] && readonly MSG_STARTED_NOT_HEALTHY="Speaker Verification started but health check failed"
    [[ -z "${MSG_UNINSTALL_WARNING:-}" ]] && readonly MSG_UNINSTALL_WARNING="This will remove Speaker Verification and all enrolled profiles"

    # Docker installation hints (only set if not already defined)
    [[ -z "${MSG_DOCKER_INSTALL_HINT:-}" ]] && readonly MSG_DOCKER_INSTALL_HINT="Please install Docker first: https://docs.docker.com/get-docker/"
    [[ -z "${MSG_DOCKER_START_HINT:-}" ]] && readonly MSG_DOCKER_START_HINT="Start Docker with: sudo systemctl start docker"
    [[ -z "${MSG_DOCKER_PERMISSIONS_HINT:-}" ]] && readonly MSG_DOCKER_PERMISSIONS_HINT="Add user to docker group: sudo usermod -aG docker \$USER"
    [[ -z "${MSG_DOCKER_LOGOUT_HINT:-}" ]] && readonly MSG_DOCKER_LOGOUT_HINT="Then log out and back in for changes to take effect"

    # Export for global access
    export MSG_INSTALL_SUCCESS MSG_START_SUCCESS MSG_STOP_SUCCESS MSG_RESTART_SUCCESS MSG_UNINSTALL_SUCCESS
    export MSG_CONTAINER_STARTED MSG_DIRECTORIES_CREATED MSG_CONTAINER_REMOVED MSG_HEALTHY MSG_RUNNING
    export MSG_DOCKER_NOT_FOUND MSG_DOCKER_NOT_RUNNING
    export MSG_DOCKER_NO_PERMISSIONS MSG_PORT_IN_USE MSG_INSTALL_FAILED MSG_START_FAILED MSG_STOP_FAILED
    export MSG_CONTAINER_NOT_EXISTS MSG_NOT_INSTALLED MSG_NOT_RUNNING MSG_NOT_HEALTHY MSG_HEALTH_CHECK_FAILED
    export MSG_STARTUP_TIMEOUT MSG_CREATE_DIRS_FAILED MSG_START_CONTAINER_FAILED
    export MSG_GPU_NOT_AVAILABLE
    export MSG_CHECKING_STATUS MSG_PULLING_IMAGE MSG_CREATING_DIRS MSG_STARTING_CONTAINER
    export MSG_WAITING_STARTUP MSG_WAITING_INIT MSG_STOPPING MSG_STARTING MSG_RESTARTING
    export MSG_REMOVING_CONTAINER MSG_SHOWING_LOGS
    export MSG_ALREADY_INSTALLED MSG_ALREADY_RUNNING
    export MSG_EXISTS_NOT_RUNNING MSG_STARTED_NOT_HEALTHY MSG_UNINSTALL_WARNING
    export MSG_DOCKER_INSTALL_HINT MSG_DOCKER_START_HINT MSG_DOCKER_PERMISSIONS_HINT MSG_DOCKER_LOGOUT_HINT
}

# Export function for subshell availability
export -f messages::export_messages
