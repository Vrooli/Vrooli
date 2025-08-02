#!/usr/bin/env bash
# SearXNG User Messages
# All user-facing messages for consistent communication

#######################################
# Export message constants
# Idempotent - safe to call multiple times
#######################################
searxng::export_messages() {
    #######################################
    # Success Messages
    #######################################
    [[ -z "${MSG_SEARXNG_INSTALL_SUCCESS:-}" ]] && readonly MSG_SEARXNG_INSTALL_SUCCESS="SearXNG metasearch engine installed successfully! 🎉"
    [[ -z "${MSG_SEARXNG_START_SUCCESS:-}" ]] && readonly MSG_SEARXNG_START_SUCCESS="SearXNG is now running and ready for searches!"
    [[ -z "${MSG_SEARXNG_STOP_SUCCESS:-}" ]] && readonly MSG_SEARXNG_STOP_SUCCESS="SearXNG has been stopped successfully."
    [[ -z "${MSG_SEARXNG_RESTART_SUCCESS:-}" ]] && readonly MSG_SEARXNG_RESTART_SUCCESS="SearXNG has been restarted successfully."
    [[ -z "${MSG_SEARXNG_UNINSTALL_SUCCESS:-}" ]] && readonly MSG_SEARXNG_UNINSTALL_SUCCESS="SearXNG has been completely removed."
    [[ -z "${MSG_SEARXNG_CONFIG_UPDATED:-}" ]] && readonly MSG_SEARXNG_CONFIG_UPDATED="SearXNG configuration has been updated."

    #######################################
    # Information Messages
    #######################################
    [[ -z "${MSG_SEARXNG_ACCESS_INFO:-}" ]] && readonly MSG_SEARXNG_ACCESS_INFO="Access SearXNG at: \${SEARXNG_BASE_URL}"
    [[ -z "${MSG_SEARXNG_SEARCH_INFO:-}" ]] && readonly MSG_SEARXNG_SEARCH_INFO="You can now search using the web interface or API endpoints."
    [[ -z "${MSG_SEARXNG_API_INFO:-}" ]] && readonly MSG_SEARXNG_API_INFO="API endpoint: \${SEARXNG_BASE_URL}/search?q=your+query&format=json"
    [[ -z "${MSG_SEARXNG_STATS_INFO:-}" ]] && readonly MSG_SEARXNG_STATS_INFO="View statistics at: \${SEARXNG_BASE_URL}/stats"
    [[ -z "${MSG_SEARXNG_CONFIG_INFO:-}" ]] && readonly MSG_SEARXNG_CONFIG_INFO="Configuration stored in: \${SEARXNG_DATA_DIR}/settings.yml"

    #######################################
    # Warning Messages
    #######################################
    [[ -z "${MSG_SEARXNG_ALREADY_RUNNING:-}" ]] && readonly MSG_SEARXNG_ALREADY_RUNNING="SearXNG is already running on port \${SEARXNG_PORT}"
    [[ -z "${MSG_SEARXNG_NOT_RUNNING:-}" ]] && readonly MSG_SEARXNG_NOT_RUNNING="SearXNG is not currently running"
    [[ -z "${MSG_SEARXNG_PORT_CONFLICT:-}" ]] && readonly MSG_SEARXNG_PORT_CONFLICT="Port \${SEARXNG_PORT} is already in use by another service"
    [[ -z "${MSG_SEARXNG_PUBLIC_ACCESS_WARNING:-}" ]] && readonly MSG_SEARXNG_PUBLIC_ACCESS_WARNING="⚠️  Public access enabled - ensure proper security measures"
    [[ -z "${MSG_SEARXNG_ENGINES_LIMITED:-}" ]] && readonly MSG_SEARXNG_ENGINES_LIMITED="⚠️  Only a subset of search engines are enabled. Check configuration."

    #######################################
    # Error Messages
    #######################################
    [[ -z "${MSG_SEARXNG_INSTALL_FAILED:-}" ]] && readonly MSG_SEARXNG_INSTALL_FAILED="❌ SearXNG installation failed. Check logs for details."
    [[ -z "${MSG_SEARXNG_START_FAILED:-}" ]] && readonly MSG_SEARXNG_START_FAILED="❌ Failed to start SearXNG. Port may be in use."
    [[ -z "${MSG_SEARXNG_CONFIG_ERROR:-}" ]] && readonly MSG_SEARXNG_CONFIG_ERROR="❌ Configuration error. Please check settings.yml"
    [[ -z "${MSG_SEARXNG_DOCKER_ERROR:-}" ]] && readonly MSG_SEARXNG_DOCKER_ERROR="❌ Docker error occurred. Is Docker running?"
    [[ -z "${MSG_SEARXNG_NETWORK_ERROR:-}" ]] && readonly MSG_SEARXNG_NETWORK_ERROR="❌ Network configuration failed. Check Docker network settings."

    #######################################
    # Setup Progress Messages
    #######################################
    [[ -z "${MSG_SEARXNG_CHECKING_DEPS:-}" ]] && readonly MSG_SEARXNG_CHECKING_DEPS="Checking dependencies..."
    [[ -z "${MSG_SEARXNG_CREATING_DIRS:-}" ]] && readonly MSG_SEARXNG_CREATING_DIRS="Creating SearXNG directories..."
    [[ -z "${MSG_SEARXNG_PULLING_IMAGE:-}" ]] && readonly MSG_SEARXNG_PULLING_IMAGE="Pulling SearXNG Docker image..."
    [[ -z "${MSG_SEARXNG_CONFIGURING:-}" ]] && readonly MSG_SEARXNG_CONFIGURING="Configuring SearXNG settings..."
    [[ -z "${MSG_SEARXNG_STARTING:-}" ]] && readonly MSG_SEARXNG_STARTING="Starting SearXNG container..."
    [[ -z "${MSG_SEARXNG_VERIFYING:-}" ]] && readonly MSG_SEARXNG_VERIFYING="Verifying SearXNG is running..."

    #######################################
    # Status Messages
    #######################################
    [[ -z "${MSG_SEARXNG_STATUS_RUNNING:-}" ]] && readonly MSG_SEARXNG_STATUS_RUNNING="✅ SearXNG is running"
    [[ -z "${MSG_SEARXNG_STATUS_STOPPED:-}" ]] && readonly MSG_SEARXNG_STATUS_STOPPED="⭕ SearXNG is stopped"
    [[ -z "${MSG_SEARXNG_STATUS_NOT_INSTALLED:-}" ]] && readonly MSG_SEARXNG_STATUS_NOT_INSTALLED="❌ SearXNG is not installed"
    [[ -z "${MSG_SEARXNG_STATUS_ERROR:-}" ]] && readonly MSG_SEARXNG_STATUS_ERROR="❌ SearXNG is in error state"

    #######################################
    # User Action Messages
    #######################################
    [[ -z "${MSG_SEARXNG_TRY_FORCE:-}" ]] && readonly MSG_SEARXNG_TRY_FORCE="Use --force to override"
    [[ -z "${MSG_SEARXNG_CHECK_LOGS:-}" ]] && readonly MSG_SEARXNG_CHECK_LOGS="Check logs with: ./manage.sh --action logs"
    [[ -z "${MSG_SEARXNG_VISIT_UI:-}" ]] && readonly MSG_SEARXNG_VISIT_UI="Visit the web interface to start searching"
    [[ -z "${MSG_SEARXNG_CONFIRM_ACTION:-}" ]] && readonly MSG_SEARXNG_CONFIRM_ACTION="Are you sure you want to proceed? (yes/no)"
}

# Helper function to display formatted messages
searxng::show_message() {
    local message="$1"
    echo -e "$message"
}
