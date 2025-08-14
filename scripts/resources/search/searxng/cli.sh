#!/usr/bin/env bash
################################################################################
# SearXNG Resource CLI
# 
# Lightweight CLI interface for SearXNG that delegates to existing lib functions.
#
# Usage:
#   resource-searxng <command> [options]
#
################################################################################

set -euo pipefail

# Get script directory
SEARXNG_CLI_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
VROOLI_ROOT="${VROOLI_ROOT:-$(cd "$SEARXNG_CLI_DIR/../../../.." && pwd)}"
export VROOLI_ROOT
export RESOURCE_DIR="$SEARXNG_CLI_DIR"
export SEARXNG_SCRIPT_DIR="$SEARXNG_CLI_DIR"  # For compatibility with existing libs

# Source utilities first
# shellcheck disable=SC1091
source "${VROOLI_ROOT}/scripts/lib/utils/var.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${var_LOG_FILE:-${VROOLI_ROOT}/scripts/lib/utils/log.sh}" 2>/dev/null || true
# shellcheck disable=SC1091
source "${var_RESOURCES_COMMON_FILE}" 2>/dev/null || true

# Source the CLI template
# shellcheck disable=SC1091
source "${VROOLI_ROOT}/scripts/lib/resources/cli/resource-cli-template.sh"

# Source SearXNG configuration
# shellcheck disable=SC1091
source "${SEARXNG_CLI_DIR}/config/defaults.sh" 2>/dev/null || true
searxng::export_config 2>/dev/null || true

# Source SearXNG libraries
for lib in common docker install status config api; do
    lib_file="${SEARXNG_CLI_DIR}/lib/${lib}.sh"
    if [[ -f "$lib_file" ]]; then
        # shellcheck disable=SC1090
        source "$lib_file" 2>/dev/null || true
    fi
done

# Initialize with resource name
resource_cli::init "searxng"

################################################################################
# Delegate to existing SearXNG functions
################################################################################

# Validate SearXNG configuration
resource_cli::validate() {
    if command -v searxng::validate_config &>/dev/null; then
        searxng::validate_config
    elif command -v searxng::is_healthy &>/dev/null; then
        searxng::is_healthy
    else
        # Basic validation
        log::header "Validating SearXNG"
        docker ps --format '{{.Names}}' 2>/dev/null | grep -q "searxng" || {
            log::error "SearXNG container not running"
            return 1
        }
        log::success "SearXNG is running"
    fi
}

# Show SearXNG status
resource_cli::status() {
    if command -v searxng::show_status &>/dev/null; then
        searxng::show_status
    elif command -v searxng::get_status &>/dev/null; then
        searxng::get_status
    else
        # Basic status
        log::header "SearXNG Status"
        if docker ps --format '{{.Names}}' 2>/dev/null | grep -q "searxng"; then
            echo "Container: ✅ Running"
            docker ps --filter "name=searxng" --format "table {{.Status}}\t{{.Ports}}" | tail -n 1
        else
            echo "Container: ❌ Not running"
        fi
    fi
}

# Start SearXNG
resource_cli::start() {
    DRY_RUN="${DRY_RUN:-false}"
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log::info "[DRY RUN] Would start SearXNG"
        return 0
    fi
    
    if command -v searxng::start_container &>/dev/null; then
        searxng::start_container
    else
        docker start searxng || log::error "Failed to start SearXNG"
    fi
}

# Stop SearXNG
resource_cli::stop() {
    DRY_RUN="${DRY_RUN:-false}"
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log::info "[DRY RUN] Would stop SearXNG"
        return 0
    fi
    
    if command -v searxng::stop_container &>/dev/null; then
        searxng::stop_container
    else
        docker stop searxng || log::error "Failed to stop SearXNG"
    fi
}

# Install SearXNG
resource_cli::install() {
    DRY_RUN="${DRY_RUN:-false}"
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log::info "[DRY RUN] Would install SearXNG"
        return 0
    fi
    
    if command -v searxng::install &>/dev/null; then
        searxng::install
    else
        log::error "searxng::install not available"
        return 1
    fi
}

# Uninstall SearXNG
resource_cli::uninstall() {
    FORCE="${FORCE:-false}"
    DRY_RUN="${DRY_RUN:-false}"
    
    if [[ "$FORCE" != "true" ]]; then
        echo "⚠️  This will remove SearXNG and all its data. Use --force to confirm."
        return 1
    fi
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log::info "[DRY RUN] Would uninstall SearXNG"
        return 0
    fi
    
    if command -v searxng::uninstall &>/dev/null; then
        searxng::uninstall
    else
        docker stop searxng 2>/dev/null || true
        docker rm searxng 2>/dev/null || true
        log::success "SearXNG uninstalled"
    fi
}

################################################################################
# SearXNG-specific commands
################################################################################

# Show credentials for n8n integration
resource_cli::credentials() {
    source "${VROOLI_ROOT}/scripts/resources/lib/credentials-utils.sh"
    
    if ! credentials::parse_args "$@"; then
        [[ $? -eq 2 ]] && { credentials::show_help "$RESOURCE_NAME"; return 0; }
        return 1
    fi
    
    local status
    status=$(credentials::get_resource_status "${SEARXNG_CONTAINER_NAME:-searxng}")
    
    local connections_array="[]"
    if [[ "$status" == "running" ]]; then
        # SearXNG typically runs without authentication by default
        local connection_obj
        connection_obj=$(jq -n \
            --arg host "${SEARXNG_HOST:-localhost}" \
            --argjson port "${SEARXNG_PORT:-9200}" \
            '{
                host: $host,
                port: $port
            }')
        
        # No authentication needed for SearXNG - using httpRequest
        local auth_obj="{}"
        
        # Build metadata object
        local metadata_obj
        metadata_obj=$(jq -n \
            --arg description "SearXNG privacy-respecting search engine" \
            --arg base_url "${SEARXNG_BASE_URL:-http://localhost:9200}" \
            --arg instance_name "${SEARXNG_INSTANCE_NAME:-Vrooli SearXNG}" \
            '{
                description: $description,
                base_url: $base_url,
                instance_name: $instance_name
            }')
        
        local connection
        connection=$(credentials::build_connection \
            "main" \
            "SearXNG Search API" \
            "httpRequest" \
            "$connection_obj" \
            "$auth_obj" \
            "$metadata_obj")
        
        connections_array="[$connection]"
    fi
    
    local response
    response=$(credentials::build_response "$RESOURCE_NAME" "$status" "$connections_array")
    credentials::format_output "$response"
}

# Search using SearXNG
searxng_search() {
    local query="${1:-}"
    local category="${2:-general}"
    local format="${3:-json}"
    
    if [[ -z "$query" ]]; then
        log::error "Search query required"
        echo "Usage: resource-searxng search <query> [category] [format]"
        return 1
    fi
    
    if command -v searxng::search &>/dev/null; then
        SEARCH_QUERY="$query" SEARCH_CATEGORY="$category" SEARCH_FORMAT="$format" \
        searxng::search "$query" "$format" "$category"
    else
        log::error "Search functionality not available"
        return 1
    fi
}

# Test SearXNG API
searxng_test_api() {
    if command -v searxng::test_api &>/dev/null; then
        searxng::test_api
    else
        log::error "API test not available"
        return 1
    fi
}

# Run SearXNG benchmark
searxng_benchmark() {
    local count="${1:-10}"
    
    if command -v searxng::benchmark &>/dev/null; then
        searxng::benchmark "$count"
    else
        log::error "Benchmark not available"
        return 1
    fi
}

# Show SearXNG logs
searxng_logs() {
    if command -v searxng::get_logs &>/dev/null; then
        searxng::get_logs
    else
        docker logs searxng 2>/dev/null || log::error "Failed to get logs"
    fi
}

# Show help
resource_cli::show_help() {
    cat << EOF
🔍 SearXNG Resource CLI

USAGE:
    resource-searxng <command> [options]

CORE COMMANDS:
    validate            Validate SearXNG configuration
    status              Show SearXNG status
    start               Start SearXNG container
    stop                Stop SearXNG container
    install             Install SearXNG
    uninstall           Uninstall SearXNG (requires --force)
    credentials         Show n8n credentials for SearXNG
    
SEARXNG COMMANDS:
    search <query>      Search using SearXNG API
    test-api            Test SearXNG API endpoints
    benchmark [count]   Run performance benchmark
    logs                Show container logs

OPTIONS:
    --verbose, -v       Show detailed output
    --dry-run           Preview actions without executing
    --force             Force operation (skip confirmations)

EXAMPLES:
    resource-searxng status
    resource-searxng search "artificial intelligence"
    resource-searxng search "robots" images json
    resource-searxng test-api
    resource-searxng benchmark 20

For more information: https://docs.vrooli.com/resources/searxng
EOF
}

# Main command router
resource_cli::main() {
    # Parse common options first
    local remaining_args
    remaining_args=$(resource_cli::parse_options "$@")
    set -- $remaining_args
    
    local command="${1:-help}"
    shift || true
    
    case "$command" in
        # Standard resource commands
        validate|status|start|stop|install|uninstall|credentials)
            resource_cli::$command "$@"
            ;;
            
        # SearXNG-specific commands
        search)
            searxng_search "$@"
            ;;
        test-api)
            searxng_test_api "$@"
            ;;
        benchmark)
            searxng_benchmark "$@"
            ;;
        logs)
            searxng_logs "$@"
            ;;
            
        help|--help|-h)
            resource_cli::show_help
            ;;
        *)
            log::error "Unknown command: $command"
            echo ""
            resource_cli::show_help
            exit 1
            ;;
    esac
}

# Run main if executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    resource_cli::main "$@"
fi