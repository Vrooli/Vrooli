#!/usr/bin/env bash
# Browserless User Messages
# All user-facing messages centralized

#######################################
# Export user-facing messages
# Idempotent - safe to call multiple times
#######################################
browserless::export_messages() {
    # Success messages (only set if not already defined)
    [[ -z "${MSG_INSTALL_SUCCESS:-}" ]] && readonly MSG_INSTALL_SUCCESS="✅ Browserless installed successfully"
    [[ -z "${MSG_START_SUCCESS:-}" ]] && readonly MSG_START_SUCCESS="✅ Browserless started successfully"
    [[ -z "${MSG_STOP_SUCCESS:-}" ]] && readonly MSG_STOP_SUCCESS="✅ Browserless stopped successfully"
    [[ -z "${MSG_RESTART_SUCCESS:-}" ]] && readonly MSG_RESTART_SUCCESS="✅ Browserless restarted successfully"
    [[ -z "${MSG_UNINSTALL_SUCCESS:-}" ]] && readonly MSG_UNINSTALL_SUCCESS="✅ Browserless uninstalled successfully"
    [[ -z "${MSG_CONTAINER_STARTED:-}" ]] && readonly MSG_CONTAINER_STARTED="Browserless container started"
    [[ -z "${MSG_DIRECTORIES_CREATED:-}" ]] && readonly MSG_DIRECTORIES_CREATED="Browserless directories created"
    [[ -z "${MSG_NETWORK_CREATED:-}" ]] && readonly MSG_NETWORK_CREATED="Docker network created"
    [[ -z "${MSG_CONTAINER_REMOVED:-}" ]] && readonly MSG_CONTAINER_REMOVED="Browserless container removed"
    [[ -z "${MSG_HEALTHY:-}" ]] && readonly MSG_HEALTHY="✅ Browserless API is healthy"
    [[ -z "${MSG_RUNNING:-}" ]] && readonly MSG_RUNNING="✅ Browserless container is running"

    # Error messages (only set if not already defined)
    [[ -z "${MSG_DOCKER_NOT_FOUND:-}" ]] && readonly MSG_DOCKER_NOT_FOUND="Docker is not installed"
    [[ -z "${MSG_DOCKER_NOT_RUNNING:-}" ]] && readonly MSG_DOCKER_NOT_RUNNING="Docker daemon is not running"
    [[ -z "${MSG_DOCKER_NO_PERMISSIONS:-}" ]] && readonly MSG_DOCKER_NO_PERMISSIONS="Current user doesn't have Docker permissions"
    [[ -z "${MSG_PORT_IN_USE:-}" ]] && readonly MSG_PORT_IN_USE="Port ${BROWSERLESS_PORT} is already in use"
    [[ -z "${MSG_INSTALL_FAILED:-}" ]] && readonly MSG_INSTALL_FAILED="❌ Browserless installation failed"
    [[ -z "${MSG_START_FAILED:-}" ]] && readonly MSG_START_FAILED="Failed to start Browserless"
    [[ -z "${MSG_STOP_FAILED:-}" ]] && readonly MSG_STOP_FAILED="Failed to stop Browserless"
    [[ -z "${MSG_CONTAINER_NOT_EXISTS:-}" ]] && readonly MSG_CONTAINER_NOT_EXISTS="Browserless container does not exist"
    [[ -z "${MSG_NOT_INSTALLED:-}" ]] && readonly MSG_NOT_INSTALLED="❌ Browserless is not installed"
    [[ -z "${MSG_NOT_RUNNING:-}" ]] && readonly MSG_NOT_RUNNING="Browserless is not running"
    [[ -z "${MSG_NOT_HEALTHY:-}" ]] && readonly MSG_NOT_HEALTHY="Browserless is not running or healthy"
    [[ -z "${MSG_HEALTH_CHECK_FAILED:-}" ]] && readonly MSG_HEALTH_CHECK_FAILED="⚠️  Browserless API health check failed"
    [[ -z "${MSG_STARTUP_TIMEOUT:-}" ]] && readonly MSG_STARTUP_TIMEOUT="Browserless failed to start within timeout"
    [[ -z "${MSG_CREATE_DIRS_FAILED:-}" ]] && readonly MSG_CREATE_DIRS_FAILED="Failed to create Browserless directories"
    [[ -z "${MSG_START_CONTAINER_FAILED:-}" ]] && readonly MSG_START_CONTAINER_FAILED="Failed to start Browserless container"

    # Info messages (only set if not already defined)
    [[ -z "${MSG_CHECKING_STATUS:-}" ]] && readonly MSG_CHECKING_STATUS="🔍 Checking Browserless status..."
    [[ -z "${MSG_PULLING_IMAGE:-}" ]] && readonly MSG_PULLING_IMAGE="📥 Pulling Browserless image..."
    [[ -z "${MSG_CREATING_DIRS:-}" ]] && readonly MSG_CREATING_DIRS="Creating Browserless data directory..."
    [[ -z "${MSG_CREATING_NETWORK:-}" ]] && readonly MSG_CREATING_NETWORK="Creating Docker network for Browserless..."
    [[ -z "${MSG_STARTING_CONTAINER:-}" ]] && readonly MSG_STARTING_CONTAINER="Starting Browserless container..."
    [[ -z "${MSG_WAITING_STARTUP:-}" ]] && readonly MSG_WAITING_STARTUP="Waiting for Browserless to start..."
    [[ -z "${MSG_WAITING_INIT:-}" ]] && readonly MSG_WAITING_INIT="Waiting for Browserless to complete initialization..."
    [[ -z "${MSG_STOPPING:-}" ]] && readonly MSG_STOPPING="Stopping Browserless..."
    [[ -z "${MSG_STARTING:-}" ]] && readonly MSG_STARTING="Starting Browserless..."
    [[ -z "${MSG_RESTARTING:-}" ]] && readonly MSG_RESTARTING="Restarting Browserless..."
    [[ -z "${MSG_REMOVING_CONTAINER:-}" ]] && readonly MSG_REMOVING_CONTAINER="Removing Browserless container..."
    [[ -z "${MSG_REMOVING_NETWORK:-}" ]] && readonly MSG_REMOVING_NETWORK="Removing Docker network..."
    [[ -z "${MSG_BACKING_UP_DATA:-}" ]] && readonly MSG_BACKING_UP_DATA="Backing up Browserless data to:"
    [[ -z "${MSG_SHOWING_LOGS:-}" ]] && readonly MSG_SHOWING_LOGS="Showing Browserless logs (Ctrl+C to exit)..."

    # Warning messages (only set if not already defined)
    [[ -z "${MSG_ALREADY_INSTALLED:-}" ]] && readonly MSG_ALREADY_INSTALLED="Browserless is already installed and running"
    [[ -z "${MSG_ALREADY_RUNNING:-}" ]] && readonly MSG_ALREADY_RUNNING="Browserless is already running on port ${BROWSERLESS_PORT}"
    [[ -z "${MSG_EXISTS_NOT_RUNNING:-}" ]] && readonly MSG_EXISTS_NOT_RUNNING="⚠️  Browserless container exists but is not running"
    [[ -z "${MSG_STARTED_NOT_HEALTHY:-}" ]] && readonly MSG_STARTED_NOT_HEALTHY="Browserless started but health check failed"
    [[ -z "${MSG_UNINSTALL_WARNING:-}" ]] && readonly MSG_UNINSTALL_WARNING="This will remove Browserless and all browser data"
    [[ -z "${MSG_CONFIG_UPDATE_FAILED:-}" ]] && readonly MSG_CONFIG_UPDATE_FAILED="Failed to update Vrooli configuration"
    [[ -z "${MSG_NETWORK_CREATE_FAILED:-}" ]] && readonly MSG_NETWORK_CREATE_FAILED="Failed to create Docker network (may already exist)"

    # Usage example messages (only set if not already defined)
    [[ -z "${MSG_USAGE_SCREENSHOT:-}" ]] && readonly MSG_USAGE_SCREENSHOT="📸 Testing Browserless Screenshot API"
    [[ -z "${MSG_USAGE_PDF:-}" ]] && readonly MSG_USAGE_PDF="📄 Testing Browserless PDF API"
    [[ -z "${MSG_USAGE_SCRAPE:-}" ]] && readonly MSG_USAGE_SCRAPE="🕷️ Testing Browserless Content Scraping"
    [[ -z "${MSG_USAGE_PRESSURE:-}" ]] && readonly MSG_USAGE_PRESSURE="📊 Checking Browserless Pool Status"
    [[ -z "${MSG_USAGE_ALL:-}" ]] && readonly MSG_USAGE_ALL="🎭 Running All Browserless Usage Examples"

    # Docker installation hints (only set if not already defined)
    [[ -z "${MSG_DOCKER_INSTALL_HINT:-}" ]] && readonly MSG_DOCKER_INSTALL_HINT="Please install Docker first: https://docs.docker.com/get-docker/"
    [[ -z "${MSG_DOCKER_START_HINT:-}" ]] && readonly MSG_DOCKER_START_HINT="Start Docker with: sudo systemctl start docker"
    [[ -z "${MSG_DOCKER_PERMISSIONS_HINT:-}" ]] && readonly MSG_DOCKER_PERMISSIONS_HINT="Add user to docker group: sudo usermod -aG docker \$USER"
    [[ -z "${MSG_DOCKER_LOGOUT_HINT:-}" ]] && readonly MSG_DOCKER_LOGOUT_HINT="Then log out and back in for changes to take effect"

    # Export for global access
    export MSG_INSTALL_SUCCESS MSG_START_SUCCESS MSG_STOP_SUCCESS MSG_RESTART_SUCCESS MSG_UNINSTALL_SUCCESS
    export MSG_CONTAINER_STARTED MSG_DIRECTORIES_CREATED MSG_NETWORK_CREATED MSG_CONTAINER_REMOVED
    export MSG_HEALTHY MSG_RUNNING MSG_DOCKER_NOT_FOUND MSG_DOCKER_NOT_RUNNING MSG_DOCKER_NO_PERMISSIONS
    export MSG_PORT_IN_USE MSG_INSTALL_FAILED MSG_START_FAILED MSG_STOP_FAILED MSG_CONTAINER_NOT_EXISTS
    export MSG_NOT_INSTALLED MSG_NOT_RUNNING MSG_NOT_HEALTHY MSG_HEALTH_CHECK_FAILED MSG_STARTUP_TIMEOUT
    export MSG_CREATE_DIRS_FAILED MSG_START_CONTAINER_FAILED MSG_CHECKING_STATUS MSG_PULLING_IMAGE
    export MSG_CREATING_DIRS MSG_CREATING_NETWORK MSG_STARTING_CONTAINER MSG_WAITING_STARTUP
    export MSG_WAITING_INIT MSG_STOPPING MSG_STARTING MSG_RESTARTING MSG_REMOVING_CONTAINER
    export MSG_REMOVING_NETWORK MSG_BACKING_UP_DATA MSG_SHOWING_LOGS MSG_ALREADY_INSTALLED
    export MSG_ALREADY_RUNNING MSG_EXISTS_NOT_RUNNING MSG_STARTED_NOT_HEALTHY MSG_UNINSTALL_WARNING
    export MSG_CONFIG_UPDATE_FAILED MSG_NETWORK_CREATE_FAILED MSG_USAGE_SCREENSHOT MSG_USAGE_PDF
    export MSG_USAGE_SCRAPE MSG_USAGE_PRESSURE MSG_USAGE_ALL MSG_DOCKER_INSTALL_HINT
    export MSG_DOCKER_START_HINT MSG_DOCKER_PERMISSIONS_HINT MSG_DOCKER_LOGOUT_HINT
}