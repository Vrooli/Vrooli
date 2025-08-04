#!/usr/bin/env bats
# Tests for SearXNG messages.sh configuration

# Setup for each test
setup() {
    # Load shared test infrastructure
    source "$(dirname "${BATS_TEST_FILENAME}")/../../../tests/bats-fixtures/common_setup.bash"
    
    # Setup standard mocks
    vrooli_auto_setup
    
    # Load messages.sh for each test (required for bats isolation)
    SEARXNG_DIR="$(dirname "$(dirname "${BATS_TEST_FILENAME}")")"
    source "${SEARXNG_DIR}/config/messages.sh"
    
    # Set mock environment for message templates
    export SEARXNG_BASE_URL="http://localhost:9200"
    export SEARXNG_PORT="9200"
    export SEARXNG_DATA_DIR="/test/data"
    export SEARXNG_CONTAINER_NAME="searxng-test"
    export SEARXNG_DEFAULT_ENGINES="google,duckduckgo,bing"
    export SEARXNG_REQUEST_TIMEOUT="10"
    export SEARXNG_POOL_MAXSIZE="20"
    
    # Call export_messages to initialize message variables
    searxng::export_messages
    export SEARXNG_RATE_LIMIT="10"
    
    # Mock log functions to avoid dependency issues
    
    # Load the messages
    SCRIPT_DIR="$(dirname "${BATS_TEST_FILENAME}")"
    source "${SCRIPT_DIR}/messages.sh"
}

# ============================================================================
# Success Message Tests
# ============================================================================

@test "success messages are defined correctly" {
    [[ "$MSG_SEARXNG_INSTALL_SUCCESS" =~ "SearXNG metasearch engine installed successfully" ]]
    [[ "$MSG_SEARXNG_START_SUCCESS" =~ "SearXNG is now running" ]]
    [[ "$MSG_SEARXNG_STOP_SUCCESS" =~ "SearXNG has been stopped" ]]
    [[ "$MSG_SEARXNG_RESTART_SUCCESS" =~ "SearXNG has been restarted" ]]
    [[ "$MSG_SEARXNG_UNINSTALL_SUCCESS" =~ "SearXNG has been completely removed" ]]
    [[ "$MSG_SEARXNG_CONFIG_UPDATED" =~ "configuration has been updated" ]]
}

@test "success messages contain appropriate indicators" {
    [[ "$MSG_SEARXNG_INSTALL_SUCCESS" =~ "🎉" ]]
    [ -n "$MSG_SEARXNG_START_SUCCESS" ]
    [ -n "$MSG_SEARXNG_STOP_SUCCESS" ]
    [ -n "$MSG_SEARXNG_RESTART_SUCCESS" ]
    [ -n "$MSG_SEARXNG_UNINSTALL_SUCCESS" ]
    [ -n "$MSG_SEARXNG_CONFIG_UPDATED" ]
}

# ============================================================================
# Information Message Tests
# ============================================================================

@test "information messages contain correct URLs and paths" {
    [[ "$MSG_SEARXNG_ACCESS_INFO" =~ "http://localhost:9200" ]]
    [[ "$MSG_SEARXNG_API_INFO" =~ "http://localhost:9200/search" ]]
    [[ "$MSG_SEARXNG_STATS_INFO" =~ "http://localhost:9200/stats" ]]
    [[ "$MSG_SEARXNG_CONFIG_INFO" =~ "/test/data/settings.yml" ]]
}

@test "information messages are properly formatted" {
    [[ "$MSG_SEARXNG_ACCESS_INFO" =~ "Access SearXNG at:" ]]
    [[ "$MSG_SEARXNG_API_INFO" =~ "API endpoint:" ]]
    [[ "$MSG_SEARXNG_STATS_INFO" =~ "View statistics at:" ]]
    [[ "$MSG_SEARXNG_CONFIG_INFO" =~ "Configuration stored in:" ]]
    [ -n "$MSG_SEARXNG_SEARCH_INFO" ]
}

# ============================================================================
# Warning Message Tests
# ============================================================================

@test "warning messages contain correct parameters" {
    [[ "$MSG_SEARXNG_ALREADY_RUNNING" =~ "port 9200" ]]
    [[ "$MSG_SEARXNG_PORT_CONFLICT" =~ "Port 9200" ]]
    [[ "$MSG_SEARXNG_PUBLIC_ACCESS_WARNING" =~ "⚠️" ]]
    [[ "$MSG_SEARXNG_PUBLIC_ACCESS_WARNING" =~ "security measures" ]]
}

@test "warning messages are appropriately cautionary" {
    [[ "$MSG_SEARXNG_NOT_RUNNING" =~ "not currently running" ]]
    [[ "$MSG_SEARXNG_RATE_LIMIT_WARNING" =~ "Rate limiting is disabled" ]]
    [[ "$MSG_SEARXNG_RATE_LIMIT_WARNING" =~ "production use" ]]
}

# ============================================================================
# Error Message Tests
# ============================================================================

@test "error messages describe failure scenarios" {
    [[ "$MSG_SEARXNG_INSTALL_FAILED" =~ "Failed to install" ]]
    [[ "$MSG_SEARXNG_START_FAILED" =~ "Failed to start" ]]
    [[ "$MSG_SEARXNG_STOP_FAILED" =~ "Failed to stop" ]]
    [[ "$MSG_SEARXNG_HEALTH_FAILED" =~ "health check failed" ]]
    [[ "$MSG_SEARXNG_CONFIG_FAILED" =~ "Failed to generate" ]]
    [[ "$MSG_SEARXNG_NETWORK_FAILED" =~ "Failed to create" ]]
}

@test "error messages include helpful guidance" {
    [[ "$MSG_SEARXNG_INSTALL_FAILED" =~ "Check the logs" ]]
    [[ "$MSG_SEARXNG_HEALTH_FAILED" =~ "may not be responding" ]]
    [ -n "$MSG_SEARXNG_START_FAILED" ]
    [ -n "$MSG_SEARXNG_STOP_FAILED" ]
    [ -n "$MSG_SEARXNG_CONFIG_FAILED" ]
    [ -n "$MSG_SEARXNG_NETWORK_FAILED" ]
}

# ============================================================================
# Setup Message Tests
# ============================================================================

@test "setup messages describe installation phases" {
    [[ "$MSG_SEARXNG_SETUP_START" =~ "Setting up SearXNG" ]]
    [[ "$MSG_SEARXNG_SETUP_DOCKER" =~ "Configuring Docker environment" ]]
    [[ "$MSG_SEARXNG_SETUP_CONFIG" =~ "Generating SearXNG configuration" ]]
    [[ "$MSG_SEARXNG_SETUP_NETWORK" =~ "Creating Docker network" ]]
    [[ "$MSG_SEARXNG_SETUP_CONTAINER" =~ "Starting SearXNG container" ]]
    [[ "$MSG_SEARXNG_SETUP_HEALTH" =~ "Waiting for SearXNG to become healthy" ]]
}

@test "setup messages are sequential and logical" {
    # Check that messages follow a logical setup sequence
    [ -n "$MSG_SEARXNG_SETUP_START" ]
    [ -n "$MSG_SEARXNG_SETUP_DOCKER" ]
    [ -n "$MSG_SEARXNG_SETUP_CONFIG" ]
    [ -n "$MSG_SEARXNG_SETUP_NETWORK" ]
    [ -n "$MSG_SEARXNG_SETUP_CONTAINER" ]
    [ -n "$MSG_SEARXNG_SETUP_HEALTH" ]
}

# ============================================================================
# Feature Message Tests
# ============================================================================

@test "feature messages contain configuration details" {
    [[ "$MSG_SEARXNG_ENGINES_INFO" =~ "google,duckduckgo,bing" ]]
    [[ "$MSG_SEARXNG_PERFORMANCE_INFO" =~ "timeout 10s" ]]
    [[ "$MSG_SEARXNG_PERFORMANCE_INFO" =~ "pool size 20" ]]
    [[ "$MSG_SEARXNG_RATE_LIMIT_INFO" =~ "10 requests per minute" ]]
}

@test "feature messages describe capabilities" {
    [[ "$MSG_SEARXNG_ENGINES_INFO" =~ "Configured search engines" ]]
    [[ "$MSG_SEARXNG_SECURITY_INFO" =~ "Secret key generated" ]]
    [[ "$MSG_SEARXNG_SECURITY_INFO" =~ "safe search enabled" ]]
    [[ "$MSG_SEARXNG_PERFORMANCE_INFO" =~ "Performance:" ]]
    [[ "$MSG_SEARXNG_RATE_LIMIT_INFO" =~ "Rate limiting:" ]]
}

# ============================================================================
# Integration Message Tests
# ============================================================================

@test "integration messages mention key systems" {
    [[ "$MSG_SEARXNG_N8N_INTEGRATION" =~ "n8n integration" ]]
    [[ "$MSG_SEARXNG_N8N_INTEGRATION" =~ "HTTP Request nodes" ]]
    [[ "$MSG_SEARXNG_API_INTEGRATION" =~ "API integration ready" ]]
    [[ "$MSG_SEARXNG_DISCOVERY_INFO" =~ "auto-discovered by Vrooli ResourceRegistry" ]]
}

@test "integration messages are informative" {
    [ -n "$MSG_SEARXNG_N8N_INTEGRATION" ]
    [ -n "$MSG_SEARXNG_API_INTEGRATION" ]
    [ -n "$MSG_SEARXNG_DISCOVERY_INFO" ]
}

# ============================================================================
# Troubleshooting Message Tests
# ============================================================================

@test "troubleshooting messages contain correct commands and paths" {
    [[ "$MSG_SEARXNG_TROUBLESHOOT_LOGS" =~ "docker logs searxng-test" ]]
    [[ "$MSG_SEARXNG_TROUBLESHOOT_CONFIG" =~ "/test/data/settings.yml" ]]
    [[ "$MSG_SEARXNG_TROUBLESHOOT_PORT" =~ "ss -tlnp | grep 9200" ]]
    [[ "$MSG_SEARXNG_TROUBLESHOOT_RESTART" =~ "./manage.sh --action restart" ]]
}

@test "troubleshooting messages provide actionable guidance" {
    [[ "$MSG_SEARXNG_TROUBLESHOOT_LOGS" =~ "Check logs with:" ]]
    [[ "$MSG_SEARXNG_TROUBLESHOOT_CONFIG" =~ "Verify config at:" ]]
    [[ "$MSG_SEARXNG_TROUBLESHOOT_PORT" =~ "Check port availability:" ]]
    [[ "$MSG_SEARXNG_TROUBLESHOOT_RESTART" =~ "Try restarting:" ]]
}

# ============================================================================
# Message Helper Function Tests
# ============================================================================

@test "searxng::message function exists" {
    run type searxng::message
    [ "$status" -eq 0 ]
    [[ "$output" =~ "function" ]]
}

@test "searxng::message handles success type correctly" {
    run searxng::message "success" "MSG_SEARXNG_INSTALL_SUCCESS"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "SUCCESS:" ]]
    [[ "$output" =~ "SearXNG metasearch engine installed successfully" ]]
}

@test "searxng::message handles info type correctly" {
    run searxng::message "info" "MSG_SEARXNG_SEARCH_INFO"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "INFO:" ]]
}

@test "searxng::message handles warning type correctly" {
    run searxng::message "warning" "MSG_SEARXNG_PUBLIC_ACCESS_WARNING"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "WARNING:" ]]
    [[ "$output" =~ "Public access enabled" ]]
}

@test "searxng::message handles error type correctly" {
    run searxng::message "error" "MSG_SEARXNG_INSTALL_FAILED"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "ERROR:" ]]
    [[ "$output" =~ "Failed to install" ]]
}

@test "searxng::message handles unknown type gracefully" {
    run searxng::message "unknown" "MSG_SEARXNG_INSTALL_SUCCESS"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "SearXNG metasearch engine installed successfully" ]]
    [[ ! "$output" =~ "SUCCESS:" ]]
}

@test "searxng::message handles undefined message variable" {
    run searxng::message "info" "MSG_UNDEFINED_VARIABLE"
    [ "$status" -eq 0 ]
    # Should handle undefined variable gracefully
}

# ============================================================================
# Message Consistency Tests
# ============================================================================

@test "all messages use consistent terminology" {
    # Check that all messages use "SearXNG" consistently
    local all_messages=$(compgen -v | grep "^MSG_SEARXNG_")
    for msg_var in $all_messages; do
        local msg_content="${!msg_var}"
        if [[ "$msg_content" =~ [Ss]earx ]]; then
            [[ "$msg_content" =~ "SearXNG" ]]
        fi
    done
}

@test "all required message constants are readonly" {
    local important_messages=(
        "MSG_SEARXNG_INSTALL_SUCCESS"
        "MSG_SEARXNG_START_SUCCESS"
        "MSG_SEARXNG_STOP_SUCCESS"
        "MSG_SEARXNG_ACCESS_INFO"
        "MSG_SEARXNG_INSTALL_FAILED"
    )
    
    for msg in "${important_messages[@]}"; do
        run bash -c "declare -p $msg"
        [[ "$output" =~ "readonly" ]]
    done
}

@test "no messages contain placeholder variables" {
    # Check that all variable substitutions were successful
    local all_messages=$(compgen -v | grep "^MSG_SEARXNG_")
    for msg_var in $all_messages; do
        local msg_content="${!msg_var}"
        [[ ! "$msg_content" =~ \$\{ ]]
    done
}

# ============================================================================
# Message Content Quality Tests
# ============================================================================

@test "messages are user-friendly and informative" {
    # Check that messages are not too technical and provide helpful information
    [ ${#MSG_SEARXNG_INSTALL_SUCCESS} -gt 10 ]
    [ ${#MSG_SEARXNG_ACCESS_INFO} -gt 10 ]
    [ ${#MSG_SEARXNG_API_INFO} -gt 10 ]
    
    # Check that error messages are helpful
    [[ "$MSG_SEARXNG_INSTALL_FAILED" =~ "logs" ]]
    [[ "$MSG_SEARXNG_HEALTH_FAILED" =~ "responding" ]]
}

@test "messages maintain professional tone" {
    # Check that messages don't contain informal language
    local problematic_patterns=("wtf" "damn" "shit" "fuck" "crap")
    local all_messages=$(compgen -v | grep "^MSG_SEARXNG_")
    
    for msg_var in $all_messages; do
        local msg_content="${!msg_var}"
        for pattern in "${problematic_patterns[@]}"; do
            [[ ! "$msg_content" =~ $pattern ]]
        done
    done
}
