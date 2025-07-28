#!/usr/bin/env bash
# SearXNG User Messages
# All user-facing messages for consistent communication

#######################################
# Success Messages
#######################################
readonly MSG_SEARXNG_INSTALL_SUCCESS="SearXNG metasearch engine installed successfully! 🎉"
readonly MSG_SEARXNG_START_SUCCESS="SearXNG is now running and ready for searches!"
readonly MSG_SEARXNG_STOP_SUCCESS="SearXNG has been stopped successfully."
readonly MSG_SEARXNG_RESTART_SUCCESS="SearXNG has been restarted successfully."
readonly MSG_SEARXNG_UNINSTALL_SUCCESS="SearXNG has been completely removed."
readonly MSG_SEARXNG_CONFIG_UPDATED="SearXNG configuration has been updated."

#######################################
# Information Messages
#######################################
readonly MSG_SEARXNG_ACCESS_INFO="Access SearXNG at: ${SEARXNG_BASE_URL}"
readonly MSG_SEARXNG_SEARCH_INFO="You can now search using the web interface or API endpoints."
readonly MSG_SEARXNG_API_INFO="API endpoint: ${SEARXNG_BASE_URL}/search?q=your+query&format=json"
readonly MSG_SEARXNG_STATS_INFO="View statistics at: ${SEARXNG_BASE_URL}/stats"
readonly MSG_SEARXNG_CONFIG_INFO="Configuration stored in: ${SEARXNG_DATA_DIR}/settings.yml"

#######################################
# Warning Messages
#######################################
readonly MSG_SEARXNG_ALREADY_RUNNING="SearXNG is already running on port ${SEARXNG_PORT}"
readonly MSG_SEARXNG_NOT_RUNNING="SearXNG is not currently running"
readonly MSG_SEARXNG_PORT_CONFLICT="Port ${SEARXNG_PORT} is already in use by another service"
readonly MSG_SEARXNG_PUBLIC_ACCESS_WARNING="⚠️  Public access enabled - ensure proper security measures"
readonly MSG_SEARXNG_RATE_LIMIT_WARNING="Rate limiting is disabled - consider enabling for production use"

#######################################
# Error Messages
#######################################
readonly MSG_SEARXNG_INSTALL_FAILED="Failed to install SearXNG. Check the logs for details."
readonly MSG_SEARXNG_START_FAILED="Failed to start SearXNG container."
readonly MSG_SEARXNG_STOP_FAILED="Failed to stop SearXNG container."
readonly MSG_SEARXNG_HEALTH_FAILED="SearXNG health check failed - service may not be responding"
readonly MSG_SEARXNG_CONFIG_FAILED="Failed to generate SearXNG configuration"
readonly MSG_SEARXNG_NETWORK_FAILED="Failed to create SearXNG Docker network"

#######################################
# Setup Messages
#######################################
readonly MSG_SEARXNG_SETUP_START="Setting up SearXNG metasearch engine..."
readonly MSG_SEARXNG_SETUP_DOCKER="Configuring Docker environment for SearXNG..."
readonly MSG_SEARXNG_SETUP_CONFIG="Generating SearXNG configuration files..."
readonly MSG_SEARXNG_SETUP_NETWORK="Creating Docker network for SearXNG..."
readonly MSG_SEARXNG_SETUP_CONTAINER="Starting SearXNG container..."
readonly MSG_SEARXNG_SETUP_HEALTH="Waiting for SearXNG to become healthy..."

#######################################
# Feature Messages
#######################################
readonly MSG_SEARXNG_ENGINES_INFO="Configured search engines: ${SEARXNG_DEFAULT_ENGINES}"
readonly MSG_SEARXNG_SECURITY_INFO="Security: Secret key generated, safe search enabled"
readonly MSG_SEARXNG_PERFORMANCE_INFO="Performance: Request timeout ${SEARXNG_REQUEST_TIMEOUT}s, pool size ${SEARXNG_POOL_MAXSIZE}"
readonly MSG_SEARXNG_RATE_LIMIT_INFO="Rate limiting: ${SEARXNG_RATE_LIMIT} requests per minute"

#######################################
# Integration Messages
#######################################
readonly MSG_SEARXNG_N8N_INTEGRATION="SearXNG is ready for n8n integration - use HTTP Request nodes"
readonly MSG_SEARXNG_API_INTEGRATION="API integration ready - see examples in the documentation"
readonly MSG_SEARXNG_DISCOVERY_INFO="SearXNG will be auto-discovered by Vrooli ResourceRegistry"

#######################################
# Troubleshooting Messages
#######################################
readonly MSG_SEARXNG_TROUBLESHOOT_LOGS="Check logs with: docker logs ${SEARXNG_CONTAINER_NAME}"
readonly MSG_SEARXNG_TROUBLESHOOT_CONFIG="Verify config at: ${SEARXNG_DATA_DIR}/settings.yml"
readonly MSG_SEARXNG_TROUBLESHOOT_PORT="Check port availability: ss -tlnp | grep ${SEARXNG_PORT}"
readonly MSG_SEARXNG_TROUBLESHOOT_RESTART="Try restarting: ./manage.sh --action restart"

#######################################
# Helper function to display formatted messages
#######################################
searxng::message() {
    local message_type="$1"
    local message_var="$2"
    local message_text="${!message_var}"
    
    case "$message_type" in
        "success")
            log::success "$message_text"
            ;;
        "info")
            log::info "$message_text"
            ;;
        "warning")
            log::warn "$message_text"
            ;;
        "error")
            log::error "$message_text"
            ;;
        *)
            echo "$message_text"
            ;;
    esac
}