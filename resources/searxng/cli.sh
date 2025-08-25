#!/usr/bin/env bash
################################################################################
# SearXNG Resource CLI
# 
# Ultra-thin CLI wrapper that delegates directly to library functions
#
# Usage:
#   resource-searxng <command> [options]
#
################################################################################

set -euo pipefail

# Get script directory (resolving symlinks for installed CLI)
if [[ -L "${BASH_SOURCE[0]}" ]]; then
    # If this is a symlink, resolve it
    SEARXNG_CLI_SCRIPT="$(readlink -f "${BASH_SOURCE[0]}")"
else
    SEARXNG_CLI_SCRIPT="${BASH_SOURCE[0]}"
fi
SEARXNG_CLI_DIR="$(builtin cd "${SEARXNG_CLI_SCRIPT%/*}" && builtin pwd)"

# Cached APP_ROOT for performance
APP_ROOT="${APP_ROOT:-$(builtin cd "${SEARXNG_CLI_DIR}/../.." && builtin pwd)}"

# Source standard variables
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"

# Source utilities using var_ variables
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "${var_RESOURCES_COMMON_FILE}"

# Source the CLI Command Framework
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_LIB_DIR}/cli-command-framework.sh"

# Source SearXNG configuration
# shellcheck disable=SC1091
source "${SEARXNG_CLI_DIR}/config/defaults.sh" 2>/dev/null || true
searxng::export_config 2>/dev/null || true

# Source SearXNG libraries - these contain the actual functionality
# shellcheck disable=SC1091
source "${SEARXNG_CLI_DIR}/lib/common.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${SEARXNG_CLI_DIR}/lib/docker.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${SEARXNG_CLI_DIR}/lib/install.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${SEARXNG_CLI_DIR}/lib/status.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${SEARXNG_CLI_DIR}/lib/config.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${SEARXNG_CLI_DIR}/lib/api.sh" 2>/dev/null || true

# Initialize CLI framework
cli::init "searxng" "SearXNG privacy-respecting search engine"

# Override help to provide SearXNG-specific examples
cli::register_command "help" "Show this help message with examples" "searxng::show_help"

# Register core commands - direct library function calls
cli::register_command "install" "Install SearXNG" "searxng::install" "modifies-system"
cli::register_command "uninstall" "Uninstall SearXNG" "searxng::uninstall" "modifies-system"
cli::register_command "upgrade" "Upgrade SearXNG" "searxng::upgrade" "modifies-system"
cli::register_command "start" "Start SearXNG" "searxng::start_container" "modifies-system"
cli::register_command "stop" "Stop SearXNG" "searxng::stop_container" "modifies-system"
cli::register_command "restart" "Restart SearXNG" "searxng::restart_container" "modifies-system"
cli::register_command "reset" "Reset SearXNG to defaults" "searxng::reset" "modifies-system"

# Register status and monitoring commands
cli::register_command "status" "Show service status" "searxng::show_status"
cli::register_command "status-detailed" "Show detailed status" "searxng::show_detailed_status"
cli::register_command "diagnose" "Run diagnostics" "searxng::diagnose"
cli::register_command "monitor" "Monitor SearXNG" "searxng::monitor"
cli::register_command "logs" "Show container logs" "searxng::get_logs"
cli::register_command "analyze-logs" "Analyze log patterns" "searxng::analyze_logs"
cli::register_command "resource-usage" "Show resource usage" "searxng::get_resource_usage"

# Register search commands
cli::register_command "search" "Search using SearXNG" "searxng::cli_search"
cli::register_command "lucky" "I'm Feeling Lucky search" "searxng::cli_lucky"
cli::register_command "headlines" "Get news headlines" "searxng::cli_headlines"
cli::register_command "batch-search" "Batch search from file" "searxng::cli_batch_search"
cli::register_command "interactive" "Interactive search mode" "searxng::interactive_search"

# Register API and testing commands
cli::register_command "test-api" "Test API endpoints" "searxng::test_api"
cli::register_command "benchmark" "Run performance benchmark" "searxng::cli_benchmark"
cli::register_command "api-config" "Show API configuration" "searxng::get_api_config"
cli::register_command "api-examples" "Show API usage examples" "searxng::show_api_examples"
cli::register_command "stats" "Show search statistics" "searxng::get_stats"

# Register configuration commands
cli::register_command "show-config" "Show configuration" "searxng::show_config"
cli::register_command "validate-config" "Validate configuration" "searxng::validate_config"
cli::register_command "reset-config" "Reset configuration" "searxng::reset_config" "modifies-system"
cli::register_command "update-engines" "Update search engines" "searxng::update_engines" "modifies-system"

# Register backup commands
cli::register_command "backup" "Backup configuration" "searxng::backup" "modifies-system"
cli::register_command "restore" "Restore from backup" "searxng::cli_restore" "modifies-system"

# Register utility commands
cli::register_command "info" "Show system information" "searxng::show_info"
cli::register_command "version" "Show SearXNG version" "searxng::get_version"
cli::register_command "health" "Check health status" "searxng::is_healthy"
cli::register_command "credentials" "Get n8n credentials" "searxng::show_credentials"
cli::register_command "troubleshooting" "Show troubleshooting guide" "searxng::show_troubleshooting"

################################################################################
# CLI wrapper functions - minimal wrappers for commands that need argument handling
################################################################################

# Search with argument handling
searxng::cli_search() {
    local query="${1:-}"
    local category="${2:-general}"
    local format="${3:-json}"
    
    [[ -z "$query" ]] && { log::error "Search query required"; return 1; }
    searxng::search "$query" "$format" "$category"
}

# Lucky search with query
searxng::cli_lucky() {
    local query="${1:-}"
    [[ -z "$query" ]] && { log::error "Query required for lucky search"; return 1; }
    searxng::lucky "$query"
}

# Headlines with optional topic
searxng::cli_headlines() {
    local topic="${1:-}"
    searxng::headlines "$topic"
}

# Batch search from file or queries
searxng::cli_batch_search() {
    local file="${1:-}"
    local queries="${2:-}"
    
    if [[ -n "$file" ]]; then
        [[ "$file" == shared:* ]] && file="${var_ROOT_DIR}/${file#shared:}"
        searxng::batch_search_file "$file"
    elif [[ -n "$queries" ]]; then
        searxng::batch_search_queries "$queries"
    else
        log::error "Batch search requires file or queries"
        echo "Usage: resource-searxng batch-search <file>"
        echo "   or: resource-searxng batch-search '' 'query1,query2,query3'"
        return 1
    fi
}

# Benchmark with count
searxng::cli_benchmark() {
    local count="${1:-10}"
    searxng::benchmark "$count"
}

# Restore from backup
searxng::cli_restore() {
    local backup="${1:-}"
    [[ -z "$backup" ]] && { log::error "Backup file required"; return 1; }
    [[ "$backup" == shared:* ]] && backup="${var_ROOT_DIR}/${backup#shared:}"
    searxng::restore "$backup"
}

# Show credentials for n8n integration
searxng::show_credentials() {
    # Source credentials utilities
    # shellcheck disable=SC1091
    source "${var_SCRIPTS_RESOURCES_LIB_DIR}/credentials-utils.sh"
    
    credentials::parse_args "$@" || return $?
    
    local status
    status=$(credentials::get_resource_status "${SEARXNG_CONTAINER_NAME:-searxng}")
    
    local connections_array="[]"
    if [[ "$status" == "running" ]]; then
        local connection_obj
        connection_obj=$(jq -n \
            --arg host "${SEARXNG_HOST:-localhost}" \
            --argjson port "${SEARXNG_PORT:-9200}" \
            '{host: $host, port: $port}')
        
        local metadata_obj
        metadata_obj=$(jq -n \
            --arg description "Privacy-respecting search engine" \
            --arg base_url "${SEARXNG_BASE_URL:-http://localhost:9200}" \
            '{description: $description, base_url: $base_url}')
        
        local connection
        connection=$(credentials::build_connection \
            "main" "SearXNG Search API" "httpRequest" \
            "$connection_obj" "{}" "$metadata_obj")
        
        connections_array="[$connection]"
    fi
    
    credentials::format_output "$(credentials::build_response "searxng" "$status" "$connections_array")"
}

# Custom help function with examples
searxng::show_help() {
    cli::_handle_help
    
    echo ""
    echo "🔍 Examples:"
    echo ""
    echo "  # Search operations"
    echo "  resource-searxng search 'artificial intelligence'"
    echo "  resource-searxng search 'robots' images"
    echo "  resource-searxng lucky 'news today'"
    echo "  resource-searxng headlines technology"
    echo ""
    echo "  # Batch operations"
    echo "  resource-searxng batch-search queries.txt"
    echo "  resource-searxng benchmark 20"
    echo ""
    echo "  # Management"
    echo "  resource-searxng status"
    echo "  resource-searxng diagnose"
    echo "  resource-searxng monitor"
    echo ""
    echo "  # Configuration"
    echo "  resource-searxng show-config"
    echo "  resource-searxng backup"
    echo ""
    echo "Search categories: general, images, news, videos, files, it, science, social, maps"
    echo "Default Port: ${SEARXNG_PORT:-9200}"
    echo "Web UI: ${SEARXNG_BASE_URL:-http://localhost:9200}"
}

################################################################################
# Main execution
################################################################################

if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    cli::dispatch "$@"
fi
