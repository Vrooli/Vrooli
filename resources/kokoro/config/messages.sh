#!/usr/bin/env bash
# Kokoro User Messages
# All user-facing messages centralized

#######################################
# Export user-facing messages
# Idempotent - safe to call multiple times
#######################################
messages::export_messages() {
    # Success messages (only set if not already defined)
    [[ -z "${MSG_INSTALL_SUCCESS:-}" ]] && readonly MSG_INSTALL_SUCCESS="✅ Kokoro installed successfully"
    [[ -z "${MSG_START_SUCCESS:-}" ]] && readonly MSG_START_SUCCESS="✅ Kokoro started successfully"
    [[ -z "${MSG_STOP_SUCCESS:-}" ]] && readonly MSG_STOP_SUCCESS="✅ Kokoro stopped successfully"
    [[ -z "${MSG_RESTART_SUCCESS:-}" ]] && readonly MSG_RESTART_SUCCESS="✅ Kokoro restarted successfully"
    [[ -z "${MSG_UNINSTALL_SUCCESS:-}" ]] && readonly MSG_UNINSTALL_SUCCESS="✅ Kokoro uninstalled successfully"
    [[ -z "${MSG_CONTAINER_STARTED:-}" ]] && readonly MSG_CONTAINER_STARTED="Kokoro container started"
    [[ -z "${MSG_DIRECTORIES_CREATED:-}" ]] && readonly MSG_DIRECTORIES_CREATED="Kokoro directories created"
    [[ -z "${MSG_CONTAINER_REMOVED:-}" ]] && readonly MSG_CONTAINER_REMOVED="Kokoro container removed"
    [[ -z "${MSG_HEALTHY:-}" ]] && readonly MSG_HEALTHY="✅ Kokoro API is healthy"
    [[ -z "${MSG_RUNNING:-}" ]] && readonly MSG_RUNNING="✅ Kokoro container is running"
    [[ -z "${MSG_SYNTHESIS_SUCCESS:-}" ]] && readonly MSG_SYNTHESIS_SUCCESS="✅ Text-to-speech synthesis completed"
    [[ -z "${MSG_VOICES_LOADED:-}" ]] && readonly MSG_VOICES_LOADED="✅ Kokoro voices loaded successfully"

    # Error messages (only set if not already defined)
    [[ -z "${MSG_DOCKER_NOT_FOUND:-}" ]] && readonly MSG_DOCKER_NOT_FOUND="Docker is not installed"
    [[ -z "${MSG_DOCKER_NOT_RUNNING:-}" ]] && readonly MSG_DOCKER_NOT_RUNNING="Docker daemon is not running"
    [[ -z "${MSG_DOCKER_NO_PERMISSIONS:-}" ]] && readonly MSG_DOCKER_NO_PERMISSIONS="Current user doesn't have Docker permissions"
    [[ -z "${MSG_PORT_IN_USE:-}" ]] && readonly MSG_PORT_IN_USE="Port ${KOKORO_PORT} is already in use"
    [[ -z "${MSG_INSTALL_FAILED:-}" ]] && readonly MSG_INSTALL_FAILED="❌ Kokoro installation failed"
    [[ -z "${MSG_START_FAILED:-}" ]] && readonly MSG_START_FAILED="Failed to start Kokoro"
    [[ -z "${MSG_STOP_FAILED:-}" ]] && readonly MSG_STOP_FAILED="Failed to stop Kokoro"
    [[ -z "${MSG_CONTAINER_NOT_EXISTS:-}" ]] && readonly MSG_CONTAINER_NOT_EXISTS="Kokoro container does not exist"
    [[ -z "${MSG_NOT_INSTALLED:-}" ]] && readonly MSG_NOT_INSTALLED="❌ Kokoro is not installed"
    [[ -z "${MSG_NOT_RUNNING:-}" ]] && readonly MSG_NOT_RUNNING="Kokoro is not running"
    [[ -z "${MSG_NOT_HEALTHY:-}" ]] && readonly MSG_NOT_HEALTHY="Kokoro is not running or healthy"
    [[ -z "${MSG_HEALTH_CHECK_FAILED:-}" ]] && readonly MSG_HEALTH_CHECK_FAILED="⚠️  Kokoro API health check failed"
    [[ -z "${MSG_STARTUP_TIMEOUT:-}" ]] && readonly MSG_STARTUP_TIMEOUT="Kokoro failed to start within timeout"
    [[ -z "${MSG_CREATE_DIRS_FAILED:-}" ]] && readonly MSG_CREATE_DIRS_FAILED="Failed to create Kokoro directories"
    [[ -z "${MSG_START_CONTAINER_FAILED:-}" ]] && readonly MSG_START_CONTAINER_FAILED="Failed to start Kokoro container"
    [[ -z "${MSG_SYNTHESIS_FAILED:-}" ]] && readonly MSG_SYNTHESIS_FAILED="❌ Text-to-speech synthesis failed"
    [[ -z "${MSG_INVALID_VOICE:-}" ]] && readonly MSG_INVALID_VOICE="❌ Invalid voice specified"
    [[ -z "${MSG_GPU_NOT_AVAILABLE:-}" ]] && readonly MSG_GPU_NOT_AVAILABLE="⚠️  GPU not available, falling back to CPU"

    # Info messages (only set if not already defined)
    [[ -z "${MSG_CHECKING_STATUS:-}" ]] && readonly MSG_CHECKING_STATUS="🔍 Checking Kokoro status..."
    [[ -z "${MSG_PULLING_IMAGE:-}" ]] && readonly MSG_PULLING_IMAGE="📥 Pulling Kokoro image..."
    [[ -z "${MSG_CREATING_DIRS:-}" ]] && readonly MSG_CREATING_DIRS="Creating Kokoro data directory..."
    [[ -z "${MSG_STARTING_CONTAINER:-}" ]] && readonly MSG_STARTING_CONTAINER="Starting Kokoro container..."
    [[ -z "${MSG_WAITING_STARTUP:-}" ]] && readonly MSG_WAITING_STARTUP="Waiting for Kokoro to start..."
    [[ -z "${MSG_WAITING_INIT:-}" ]] && readonly MSG_WAITING_INIT="Waiting for Kokoro model to load..."
    [[ -z "${MSG_STOPPING:-}" ]] && readonly MSG_STOPPING="Stopping Kokoro..."
    [[ -z "${MSG_STARTING:-}" ]] && readonly MSG_STARTING="Starting Kokoro..."
    [[ -z "${MSG_RESTARTING:-}" ]] && readonly MSG_RESTARTING="Restarting Kokoro..."
    [[ -z "${MSG_REMOVING_CONTAINER:-}" ]] && readonly MSG_REMOVING_CONTAINER="Removing Kokoro container..."
    [[ -z "${MSG_BACKING_UP_DATA:-}" ]] && readonly MSG_BACKING_UP_DATA="Backing up Kokoro data to:"
    [[ -z "${MSG_SHOWING_LOGS:-}" ]] && readonly MSG_SHOWING_LOGS="Showing Kokoro logs (Ctrl+C to exit)..."
    [[ -z "${MSG_SYNTHESIZING:-}" ]] && readonly MSG_SYNTHESIZING="🔊 Synthesizing speech..."
    [[ -z "${MSG_LISTING_VOICES:-}" ]] && readonly MSG_LISTING_VOICES="📋 Listing available voices..."
    [[ -z "${MSG_VOICE_INFO:-}" ]] && readonly MSG_VOICE_INFO="Using voice:"

    # Warning messages (only set if not already defined)
    [[ -z "${MSG_ALREADY_INSTALLED:-}" ]] && readonly MSG_ALREADY_INSTALLED="Kokoro is already installed and running"
    [[ -z "${MSG_ALREADY_RUNNING:-}" ]] && readonly MSG_ALREADY_RUNNING="Kokoro is already running on port ${KOKORO_PORT}"
    [[ -z "${MSG_EXISTS_NOT_RUNNING:-}" ]] && readonly MSG_EXISTS_NOT_RUNNING="⚠️  Kokoro container exists but is not running"
    [[ -z "${MSG_STARTED_NOT_HEALTHY:-}" ]] && readonly MSG_STARTED_NOT_HEALTHY="Kokoro started but health check failed"
    [[ -z "${MSG_UNINSTALL_WARNING:-}" ]] && readonly MSG_UNINSTALL_WARNING="This will remove Kokoro and all voice data"
    [[ -z "${MSG_CONFIG_UPDATE_FAILED:-}" ]] && readonly MSG_CONFIG_UPDATE_FAILED="Failed to update Vrooli configuration"

    # Usage example messages (only set if not already defined)
    [[ -z "${MSG_USAGE_SYNTHESIZE:-}" ]] && readonly MSG_USAGE_SYNTHESIZE="🔊 Testing Kokoro Speech Synthesis API"
    [[ -z "${MSG_USAGE_VOICES:-}" ]] && readonly MSG_USAGE_VOICES="🎭 Listing Available Voices"
    [[ -z "${MSG_USAGE_HEALTH:-}" ]] && readonly MSG_USAGE_HEALTH="🏥 Checking Kokoro Health"
    [[ -z "${MSG_USAGE_ALL:-}" ]] && readonly MSG_USAGE_ALL="🎭 Running All Kokoro Usage Examples"

    # Docker installation hints (only set if not already defined)
    [[ -z "${MSG_DOCKER_INSTALL_HINT:-}" ]] && readonly MSG_DOCKER_INSTALL_HINT="Please install Docker first: https://docs.docker.com/get-docker/"
    [[ -z "${MSG_DOCKER_START_HINT:-}" ]] && readonly MSG_DOCKER_START_HINT="Start Docker with: sudo systemctl start docker"
    [[ -z "${MSG_DOCKER_PERMISSIONS_HINT:-}" ]] && readonly MSG_DOCKER_PERMISSIONS_HINT="Add user to docker group: sudo usermod -aG docker \$USER"
    [[ -z "${MSG_DOCKER_LOGOUT_HINT:-}" ]] && readonly MSG_DOCKER_LOGOUT_HINT="Then log out and back in for changes to take effect"

    # Export for global access
    export MSG_INSTALL_SUCCESS MSG_START_SUCCESS MSG_STOP_SUCCESS MSG_RESTART_SUCCESS MSG_UNINSTALL_SUCCESS
    export MSG_CONTAINER_STARTED MSG_DIRECTORIES_CREATED MSG_CONTAINER_REMOVED MSG_HEALTHY MSG_RUNNING
    export MSG_SYNTHESIS_SUCCESS MSG_VOICES_LOADED MSG_DOCKER_NOT_FOUND MSG_DOCKER_NOT_RUNNING
    export MSG_DOCKER_NO_PERMISSIONS MSG_PORT_IN_USE MSG_INSTALL_FAILED MSG_START_FAILED MSG_STOP_FAILED
    export MSG_CONTAINER_NOT_EXISTS MSG_NOT_INSTALLED MSG_NOT_RUNNING MSG_NOT_HEALTHY MSG_HEALTH_CHECK_FAILED
    export MSG_STARTUP_TIMEOUT MSG_CREATE_DIRS_FAILED MSG_START_CONTAINER_FAILED MSG_SYNTHESIS_FAILED
    export MSG_INVALID_VOICE MSG_GPU_NOT_AVAILABLE
    export MSG_CHECKING_STATUS MSG_PULLING_IMAGE MSG_CREATING_DIRS MSG_STARTING_CONTAINER
    export MSG_WAITING_STARTUP MSG_WAITING_INIT MSG_STOPPING MSG_STARTING MSG_RESTARTING
    export MSG_REMOVING_CONTAINER MSG_BACKING_UP_DATA MSG_SHOWING_LOGS MSG_SYNTHESIZING
    export MSG_LISTING_VOICES MSG_VOICE_INFO MSG_ALREADY_INSTALLED MSG_ALREADY_RUNNING
    export MSG_EXISTS_NOT_RUNNING MSG_STARTED_NOT_HEALTHY MSG_UNINSTALL_WARNING MSG_CONFIG_UPDATE_FAILED
    export MSG_USAGE_SYNTHESIZE MSG_USAGE_VOICES MSG_USAGE_HEALTH MSG_USAGE_ALL
    export MSG_DOCKER_INSTALL_HINT MSG_DOCKER_START_HINT MSG_DOCKER_PERMISSIONS_HINT MSG_DOCKER_LOGOUT_HINT
}

# Export function for subshell availability
export -f messages::export_messages
