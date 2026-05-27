#!/usr/bin/env bash
# Kyutai STT User Messages
# All user-facing messages centralized

#######################################
# Export user-facing messages
# Idempotent - safe to call multiple times
#######################################
messages::export_messages() {
    # Success messages (only set if not already defined)
    [[ -z "${MSG_INSTALL_SUCCESS:-}" ]] && readonly MSG_INSTALL_SUCCESS="✅ Kyutai STT installed successfully"
    [[ -z "${MSG_START_SUCCESS:-}" ]] && readonly MSG_START_SUCCESS="✅ Kyutai STT started successfully"
    [[ -z "${MSG_STOP_SUCCESS:-}" ]] && readonly MSG_STOP_SUCCESS="✅ Kyutai STT stopped successfully"
    [[ -z "${MSG_RESTART_SUCCESS:-}" ]] && readonly MSG_RESTART_SUCCESS="✅ Kyutai STT restarted successfully"
    [[ -z "${MSG_UNINSTALL_SUCCESS:-}" ]] && readonly MSG_UNINSTALL_SUCCESS="✅ Kyutai STT uninstalled successfully"
    [[ -z "${MSG_CONTAINER_STARTED:-}" ]] && readonly MSG_CONTAINER_STARTED="Kyutai STT container started"
    [[ -z "${MSG_DIRECTORIES_CREATED:-}" ]] && readonly MSG_DIRECTORIES_CREATED="Kyutai STT directories created"
    [[ -z "${MSG_CONTAINER_REMOVED:-}" ]] && readonly MSG_CONTAINER_REMOVED="Kyutai STT container removed"
    [[ -z "${MSG_HEALTHY:-}" ]] && readonly MSG_HEALTHY="✅ Kyutai STT API is healthy"
    [[ -z "${MSG_RUNNING:-}" ]] && readonly MSG_RUNNING="✅ Kyutai STT container is running"
    [[ -z "${MSG_MODEL_LOADED:-}" ]] && readonly MSG_MODEL_LOADED="✅ Kyutai STT model loaded successfully"

    # Error messages (only set if not already defined)
    [[ -z "${MSG_DOCKER_NOT_FOUND:-}" ]] && readonly MSG_DOCKER_NOT_FOUND="Docker is not installed"
    [[ -z "${MSG_DOCKER_NOT_RUNNING:-}" ]] && readonly MSG_DOCKER_NOT_RUNNING="Docker daemon is not running"
    [[ -z "${MSG_DOCKER_NO_PERMISSIONS:-}" ]] && readonly MSG_DOCKER_NO_PERMISSIONS="Current user doesn't have Docker permissions"
    [[ -z "${MSG_PORT_IN_USE:-}" ]] && readonly MSG_PORT_IN_USE="Port ${KYUTAI_STT_PORT} is already in use"
    [[ -z "${MSG_INSTALL_FAILED:-}" ]] && readonly MSG_INSTALL_FAILED="❌ Kyutai STT installation failed"
    [[ -z "${MSG_START_FAILED:-}" ]] && readonly MSG_START_FAILED="Failed to start Kyutai STT"
    [[ -z "${MSG_STOP_FAILED:-}" ]] && readonly MSG_STOP_FAILED="Failed to stop Kyutai STT"
    [[ -z "${MSG_CONTAINER_NOT_EXISTS:-}" ]] && readonly MSG_CONTAINER_NOT_EXISTS="Kyutai STT container does not exist"
    [[ -z "${MSG_NOT_INSTALLED:-}" ]] && readonly MSG_NOT_INSTALLED="❌ Kyutai STT is not installed"
    [[ -z "${MSG_NOT_RUNNING:-}" ]] && readonly MSG_NOT_RUNNING="Kyutai STT is not running"
    [[ -z "${MSG_NOT_HEALTHY:-}" ]] && readonly MSG_NOT_HEALTHY="Kyutai STT is not running or healthy"
    [[ -z "${MSG_HEALTH_CHECK_FAILED:-}" ]] && readonly MSG_HEALTH_CHECK_FAILED="⚠️  Kyutai STT API health check failed"
    [[ -z "${MSG_STARTUP_TIMEOUT:-}" ]] && readonly MSG_STARTUP_TIMEOUT="Kyutai STT failed to start within timeout"
    [[ -z "${MSG_CREATE_DIRS_FAILED:-}" ]] && readonly MSG_CREATE_DIRS_FAILED="Failed to create Kyutai STT directories"
    [[ -z "${MSG_START_CONTAINER_FAILED:-}" ]] && readonly MSG_START_CONTAINER_FAILED="Failed to start Kyutai STT container"
    [[ -z "${MSG_MODEL_LOAD_FAILED:-}" ]] && readonly MSG_MODEL_LOAD_FAILED="❌ Failed to load Kyutai STT model"
    [[ -z "${MSG_GPU_NOT_AVAILABLE:-}" ]] && readonly MSG_GPU_NOT_AVAILABLE="⚠️  GPU not available; Kyutai STT requires a CUDA GPU for real-time streaming"

    # Info messages (only set if not already defined)
    [[ -z "${MSG_CHECKING_STATUS:-}" ]] && readonly MSG_CHECKING_STATUS="🔍 Checking Kyutai STT status..."
    [[ -z "${MSG_PULLING_IMAGE:-}" ]] && readonly MSG_PULLING_IMAGE="📥 Building Kyutai STT image..."
    [[ -z "${MSG_CREATING_DIRS:-}" ]] && readonly MSG_CREATING_DIRS="Creating Kyutai STT data directory..."
    [[ -z "${MSG_STARTING_CONTAINER:-}" ]] && readonly MSG_STARTING_CONTAINER="Starting Kyutai STT container..."
    [[ -z "${MSG_WAITING_STARTUP:-}" ]] && readonly MSG_WAITING_STARTUP="Waiting for Kyutai STT to start..."
    [[ -z "${MSG_WAITING_INIT:-}" ]] && readonly MSG_WAITING_INIT="Waiting for Kyutai STT model to load (first run downloads weights)..."
    [[ -z "${MSG_STOPPING:-}" ]] && readonly MSG_STOPPING="Stopping Kyutai STT..."
    [[ -z "${MSG_STARTING:-}" ]] && readonly MSG_STARTING="Starting Kyutai STT..."
    [[ -z "${MSG_RESTARTING:-}" ]] && readonly MSG_RESTARTING="Restarting Kyutai STT..."
    [[ -z "${MSG_REMOVING_CONTAINER:-}" ]] && readonly MSG_REMOVING_CONTAINER="Removing Kyutai STT container..."
    [[ -z "${MSG_BACKING_UP_DATA:-}" ]] && readonly MSG_BACKING_UP_DATA="Backing up Kyutai STT data to:"
    [[ -z "${MSG_SHOWING_LOGS:-}" ]] && readonly MSG_SHOWING_LOGS="Showing Kyutai STT logs (Ctrl+C to exit)..."
    [[ -z "${MSG_LOADING_MODEL:-}" ]] && readonly MSG_LOADING_MODEL="📥 Loading Kyutai STT model..."
    [[ -z "${MSG_MODEL_INFO:-}" ]] && readonly MSG_MODEL_INFO="Using model:"

    # Warning messages (only set if not already defined)
    [[ -z "${MSG_ALREADY_INSTALLED:-}" ]] && readonly MSG_ALREADY_INSTALLED="Kyutai STT is already installed and running"
    [[ -z "${MSG_ALREADY_RUNNING:-}" ]] && readonly MSG_ALREADY_RUNNING="Kyutai STT is already running on port ${KYUTAI_STT_PORT}"
    [[ -z "${MSG_EXISTS_NOT_RUNNING:-}" ]] && readonly MSG_EXISTS_NOT_RUNNING="⚠️  Kyutai STT container exists but is not running"
    [[ -z "${MSG_STARTED_NOT_HEALTHY:-}" ]] && readonly MSG_STARTED_NOT_HEALTHY="Kyutai STT started but health check failed"
    [[ -z "${MSG_UNINSTALL_WARNING:-}" ]] && readonly MSG_UNINSTALL_WARNING="This will remove Kyutai STT and cached model weights"
    [[ -z "${MSG_CONFIG_UPDATE_FAILED:-}" ]] && readonly MSG_CONFIG_UPDATE_FAILED="Failed to update Vrooli configuration"
    [[ -z "${MSG_MODEL_LOADING_SLOW:-}" ]] && readonly MSG_MODEL_LOADING_SLOW="⚠️  Model loading is taking longer than expected (first-run weight download)"

    # Usage example messages (only set if not already defined)
    [[ -z "${MSG_USAGE_HEALTH:-}" ]] && readonly MSG_USAGE_HEALTH="🏥 Checking Kyutai STT Health"
    [[ -z "${MSG_USAGE_INFO:-}" ]] && readonly MSG_USAGE_INFO="ℹ️  Fetching Kyutai STT Info"
    [[ -z "${MSG_USAGE_STREAM:-}" ]] && readonly MSG_USAGE_STREAM="🎤 Testing Kyutai STT Streaming"
    [[ -z "${MSG_USAGE_ALL:-}" ]] && readonly MSG_USAGE_ALL="🎭 Running All Kyutai STT Usage Examples"

    # Docker installation hints (only set if not already defined)
    [[ -z "${MSG_DOCKER_INSTALL_HINT:-}" ]] && readonly MSG_DOCKER_INSTALL_HINT="Please install Docker first: https://docs.docker.com/get-docker/"
    [[ -z "${MSG_DOCKER_START_HINT:-}" ]] && readonly MSG_DOCKER_START_HINT="Start Docker with: sudo systemctl start docker"
    [[ -z "${MSG_DOCKER_PERMISSIONS_HINT:-}" ]] && readonly MSG_DOCKER_PERMISSIONS_HINT="Add user to docker group: sudo usermod -aG docker \$USER"
    [[ -z "${MSG_DOCKER_LOGOUT_HINT:-}" ]] && readonly MSG_DOCKER_LOGOUT_HINT="Then log out and back in for changes to take effect"

    # Export for global access
    export MSG_INSTALL_SUCCESS MSG_START_SUCCESS MSG_STOP_SUCCESS MSG_RESTART_SUCCESS MSG_UNINSTALL_SUCCESS
    export MSG_CONTAINER_STARTED MSG_DIRECTORIES_CREATED MSG_CONTAINER_REMOVED MSG_HEALTHY MSG_RUNNING
    export MSG_MODEL_LOADED MSG_DOCKER_NOT_FOUND MSG_DOCKER_NOT_RUNNING
    export MSG_DOCKER_NO_PERMISSIONS MSG_PORT_IN_USE MSG_INSTALL_FAILED MSG_START_FAILED MSG_STOP_FAILED
    export MSG_CONTAINER_NOT_EXISTS MSG_NOT_INSTALLED MSG_NOT_RUNNING MSG_NOT_HEALTHY MSG_HEALTH_CHECK_FAILED
    export MSG_STARTUP_TIMEOUT MSG_CREATE_DIRS_FAILED MSG_START_CONTAINER_FAILED
    export MSG_MODEL_LOAD_FAILED MSG_GPU_NOT_AVAILABLE
    export MSG_CHECKING_STATUS MSG_PULLING_IMAGE MSG_CREATING_DIRS MSG_STARTING_CONTAINER
    export MSG_WAITING_STARTUP MSG_WAITING_INIT MSG_STOPPING MSG_STARTING MSG_RESTARTING
    export MSG_REMOVING_CONTAINER MSG_BACKING_UP_DATA MSG_SHOWING_LOGS
    export MSG_LOADING_MODEL MSG_MODEL_INFO MSG_ALREADY_INSTALLED MSG_ALREADY_RUNNING
    export MSG_EXISTS_NOT_RUNNING MSG_STARTED_NOT_HEALTHY MSG_UNINSTALL_WARNING MSG_CONFIG_UPDATE_FAILED
    export MSG_MODEL_LOADING_SLOW MSG_USAGE_HEALTH MSG_USAGE_INFO MSG_USAGE_STREAM MSG_USAGE_ALL
    export MSG_DOCKER_INSTALL_HINT MSG_DOCKER_START_HINT MSG_DOCKER_PERMISSIONS_HINT MSG_DOCKER_LOGOUT_HINT
}

# Export function for subshell availability
export -f messages::export_messages
