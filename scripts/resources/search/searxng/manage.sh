#!/usr/bin/env bash
set -euo pipefail

# SearXNG Search Engine Setup and Management
# This script handles installation, configuration, and management of SearXNG using Docker

DESCRIPTION="Install and manage SearXNG metasearch engine using Docker"

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
RESOURCES_DIR="${SCRIPT_DIR}/../.."

# Source common resources
# shellcheck disable=SC1091
source "${RESOURCES_DIR}/common.sh"
# shellcheck disable=SC1091
source "${RESOURCES_DIR}/../helpers/utils/args.sh"

# Source configuration
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/config/defaults.sh"
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/config/messages.sh"

# Export configuration
searxng::export_config

# Source all library modules
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/lib/common.sh"
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/lib/docker.sh"
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/lib/install.sh"
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/lib/status.sh"
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/lib/config.sh"
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/lib/api.sh"

#######################################
# Parse command line arguments
#######################################
searxng::parse_arguments() {
    args::reset
    
    args::register_help
    args::register_yes
    
    args::register \
        --name "action" \
        --flag "a" \
        --desc "Action to perform" \
        --type "value" \
        --options "install|uninstall|start|stop|restart|status|logs|info|search|api-test|benchmark|diagnose|config|reset-config|backup|restore|upgrade|monitor|examples|headlines|lucky|batch-search" \
        --default "install"
    
    args::register \
        --name "force" \
        --flag "f" \
        --desc "Force action even if SearXNG appears to be already installed/running" \
        --type "value" \
        --options "yes|no" \
        --default "no"
    
    args::register \
        --name "query" \
        --flag "q" \
        --desc "Search query (for search action)" \
        --type "value" \
        --default ""
    
    args::register \
        --name "format" \
        --desc "Search result format" \
        --type "value" \
        --options "json|xml|csv|rss" \
        --default "json"
    
    args::register \
        --name "category" \
        --flag "c" \
        --desc "Search category" \
        --type "value" \
        --options "general|images|videos|news|music|files|science" \
        --default "general"
    
    args::register \
        --name "language" \
        --flag "l" \
        --desc "Search language" \
        --type "value" \
        --default "en"
    
    args::register \
        --name "engines" \
        --desc "Comma-separated list of search engines" \
        --type "value" \
        --default ""
    
    # Advanced search parameters
    args::register \
        --name "pageno" \
        --desc "Page number for pagination (1-based)" \
        --type "value" \
        --default "1"
    
    args::register \
        --name "time-range" \
        --desc "Time range filter" \
        --type "value" \
        --options "hour|day|week|month|year" \
        --default ""
    
    args::register \
        --name "safesearch" \
        --desc "Safe search level (0=off, 1=moderate, 2=strict)" \
        --type "value" \
        --options "0|1|2" \
        --default "1"
    
    # Output formatting options
    args::register \
        --name "output-format" \
        --desc "Output format for results" \
        --type "value" \
        --options "json|title-only|title-url|csv|markdown|compact" \
        --default "json"
    
    args::register \
        --name "limit" \
        --desc "Maximum number of results to return" \
        --type "value" \
        --default ""
    
    # File operations
    args::register \
        --name "save" \
        --desc "Save results to file" \
        --type "value" \
        --default ""
    
    args::register \
        --name "append" \
        --desc "Append results to file (JSONL format)" \
        --type "value" \
        --default ""
    
    # Batch operations
    args::register \
        --name "file" \
        --desc "File containing queries (one per line) for batch search" \
        --type "value" \
        --default ""
    
    args::register \
        --name "queries" \
        --desc "Comma-separated list of queries for batch search" \
        --type "value" \
        --default ""
    
    # Quick actions
    args::register \
        --name "topic" \
        --desc "Topic filter for headlines action" \
        --type "value" \
        --default ""
    
    args::register \
        --name "backup-dir" \
        --desc "Backup directory path (for restore action)" \
        --type "value" \
        --default ""
    
    args::register \
        --name "output" \
        --flag "o" \
        --desc "Output file path (for config export)" \
        --type "value" \
        --default ""
    
    args::register \
        --name "count" \
        --desc "Number of queries for benchmark" \
        --type "value" \
        --default "10"
    
    args::register \
        --name "interval" \
        --desc "Monitor interval in seconds" \
        --type "value" \
        --default "30"
    
    args::register \
        --name "compose" \
        --desc "Use Docker Compose instead of direct Docker commands" \
        --type "value" \
        --options "yes|no" \
        --default "no"
    
    if args::is_asking_for_help "$@"; then
        searxng::usage
        exit 0
    fi
    
    args::parse "$@"
    
    export ACTION=$(args::get "action")
    export FORCE=$(args::get "force")
    export YES=$(args::get "yes")
    export SEARCH_QUERY=$(args::get "query")
    export SEARCH_FORMAT=$(args::get "format")
    export SEARCH_CATEGORY=$(args::get "category")
    export SEARCH_LANGUAGE=$(args::get "language")
    export ENGINES=$(args::get "engines")
    export BACKUP_DIR=$(args::get "backup-dir")
    export OUTPUT_FILE=$(args::get "output")
    export BENCHMARK_COUNT=$(args::get "count")
    export MONITOR_INTERVAL=$(args::get "interval")
    export USE_COMPOSE=$(args::get "compose")
    
    # Advanced search parameters
    export SEARCH_PAGENO=$(args::get "pageno")
    export SEARCH_TIME_RANGE=$(args::get "time-range")
    export SEARCH_SAFESEARCH=$(args::get "safesearch")
    
    # Output formatting
    export OUTPUT_FORMAT=$(args::get "output-format")
    export RESULT_LIMIT=$(args::get "limit")
    
    # File operations
    export SAVE_FILE=$(args::get "save")
    export APPEND_FILE=$(args::get "append")
    
    # Batch operations
    export BATCH_FILE=$(args::get "file")
    export BATCH_QUERIES=$(args::get "queries")
    
    # Quick actions
    export HEADLINES_TOPIC=$(args::get "topic")
}

#######################################
# Display usage information
#######################################
searxng::usage() {
    args::usage "$DESCRIPTION"
    echo
    echo "Examples:"
    echo "  $0 --action install                                    # Install SearXNG"
    echo "  $0 --action start                                      # Start SearXNG"
    echo "  $0 --action status                                     # Show status"
    echo "  $0 --action search --query 'artificial intelligence'   # Basic search"
    echo "  $0 --action search --query 'robots' --category images  # Image search"
    echo "  $0 --action search --query 'news' --time-range day     # Recent news"
    echo "  $0 --action search --query 'AI' --limit 5 --save results.json  # Save results"
    echo "  $0 --action search --query 'tech' --output-format title-only   # Titles only"
    echo "  $0 --action search --query 'research' --pageno 2       # Page 2 results"
    echo "  $0 --action api-test                                   # Test API endpoints"
    echo "  $0 --action benchmark --count 20                      # Performance test"
    echo "  $0 --action backup                                     # Backup configuration"
    echo "  $0 --action monitor --interval 60                     # Monitor status"
    echo "  $0 --action logs                                       # Show container logs"
    echo "  $0 --action diagnose                                   # Run diagnostics"
    echo
    echo "Configuration:"
    echo "  Environment variables can override defaults:"
    echo "    SEARXNG_PORT=8200 $0 --action install               # Use custom port"
    echo "    SEARXNG_SECRET_KEY=mysecret $0 --action install      # Use custom secret"
    echo "    SEARXNG_ENABLE_REDIS=yes $0 --action install        # Enable Redis caching"
    echo
    echo "Access:"
    echo "  Web Interface: http://localhost:$SEARXNG_PORT"
    echo "  API Endpoint: http://localhost:$SEARXNG_PORT/search?q=query&format=json"
    echo "  Statistics: http://localhost:$SEARXNG_PORT/stats"
}

#######################################
# Main execution function
#######################################
searxng::main() {
    case "$ACTION" in
        "install")
            searxng::install
            ;;
        "uninstall")
            searxng::uninstall
            ;;
        "start")
            if [[ "$USE_COMPOSE" == "yes" ]]; then
                searxng::compose_up
            else
                searxng::start_container
            fi
            ;;
        "stop")
            if [[ "$USE_COMPOSE" == "yes" ]]; then
                searxng::compose_down
            else
                searxng::stop_container
            fi
            ;;
        "restart")
            searxng::restart_container
            ;;
        "status")
            searxng::show_status
            ;;
        "logs")
            searxng::get_logs
            ;;
        "info")
            searxng::show_info
            ;;
        "search")
            if [[ -z "$SEARCH_QUERY" ]]; then
                searxng::interactive_search
            else
                searxng::search "$SEARCH_QUERY" "$SEARCH_FORMAT" "$SEARCH_CATEGORY" "$SEARCH_LANGUAGE" \
                    "$SEARCH_PAGENO" "$SEARCH_SAFESEARCH" "$SEARCH_TIME_RANGE" "$OUTPUT_FORMAT" \
                    "$RESULT_LIMIT" "$SAVE_FILE" "$APPEND_FILE"
            fi
            ;;
        "api-test")
            searxng::test_api
            ;;
        "benchmark")
            searxng::benchmark "$BENCHMARK_COUNT"
            ;;
        "diagnose")
            searxng::diagnose
            ;;
        "config")
            if [[ -n "$OUTPUT_FILE" ]]; then
                searxng::export_config "$OUTPUT_FILE"
            elif [[ -n "$ENGINES" ]]; then
                searxng::update_engines "$ENGINES"
            else
                searxng::show_config
            fi
            ;;
        "reset-config")
            searxng::reset_config
            ;;
        "backup")
            searxng::backup
            ;;
        "restore")
            if [[ -z "$BACKUP_DIR" ]]; then
                log::error "Backup directory is required for restore action"
                log::info "Use: --backup-dir /path/to/backup"
                exit 1
            fi
            searxng::restore "$BACKUP_DIR"
            ;;
        "upgrade")
            searxng::upgrade
            ;;
        "monitor")
            searxng::monitor "$MONITOR_INTERVAL"
            ;;
        "examples")
            searxng::show_api_examples
            ;;
        "headlines")
            searxng::headlines "$HEADLINES_TOPIC"
            ;;
        "lucky")
            if [[ -z "$SEARCH_QUERY" ]]; then
                log::error "Query is required for lucky search"
                log::info "Use: --action lucky --query 'your search'"
                exit 1
            fi
            searxng::lucky "$SEARCH_QUERY"
            ;;
        "batch-search")
            if [[ -n "$BATCH_FILE" ]]; then
                searxng::batch_search_file "$BATCH_FILE"
            elif [[ -n "$BATCH_QUERIES" ]]; then
                searxng::batch_search_queries "$BATCH_QUERIES"
            else
                log::error "Batch search requires either --file or --queries parameter"
                log::info "Use: --action batch-search --file queries.txt"
                log::info "Or:  --action batch-search --queries 'query1,query2,query3'"
                exit 1
            fi
            ;;
        *)
            log::error "Unknown action: $ACTION"
            searxng::usage
            exit 1
            ;;
    esac
}

#######################################
# Cleanup and signal handling
#######################################
searxng::cleanup() {
    # Cleanup any temporary files or processes if needed
    true
}

# Set up signal handlers
trap searxng::cleanup EXIT INT TERM

#######################################
# Script entry point
#######################################
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    # Parse arguments
    searxng::parse_arguments "$@"
    
    # Run main function
    if searxng::main; then
        exit 0
    else
        exit 1
    fi
fi