#!/usr/bin/env bash
################################################################################
# Unstructured-io Resource CLI
# 
# Lightweight CLI interface for unstructured-io that delegates to existing lib functions.
#
# Usage:
#   resource-unstructured-io <command> [options]
#
################################################################################

set -euo pipefail

# Get script directory
UNSTRUCTURED_IO_CLI_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
VROOLI_ROOT="${VROOLI_ROOT:-$(cd "$UNSTRUCTURED_IO_CLI_DIR/../../../.." && pwd)}"
export VROOLI_ROOT
export RESOURCE_DIR="$UNSTRUCTURED_IO_CLI_DIR"

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

# Source unstructured-io configuration
# shellcheck disable=SC1091
source "${UNSTRUCTURED_IO_CLI_DIR}/config/defaults.sh" 2>/dev/null || true

# Initialize with resource name
resource_cli::init "unstructured-io"

################################################################################
# Delegate to existing unstructured-io functions via manage.sh
################################################################################

# Get credentials for n8n integration
resource_cli::credentials() {
    # Source credentials utilities
    # shellcheck disable=SC1091
    source "${VROOLI_ROOT}/scripts/resources/lib/credentials-utils.sh"
    
    # Parse arguments
    if ! credentials::parse_args "$@"; then
        [[ $? -eq 2 ]] && { credentials::show_help "unstructured-io"; return 0; }
        return 1
    fi
    
    # Get resource status
    local status
    status=$(credentials::get_resource_status "${UNSTRUCTURED_IO_CONTAINER_NAME:-unstructured-io}")
    
    # Build connections array - unstructured-io runs as open HTTP API (no auth)
    local connections_array="[]"
    if [[ "$status" == "running" ]]; then
        # Unstructured-io HTTP API connection (no authentication required)
        local connection_obj
        connection_obj=$(jq -n \
            --arg host "localhost" \
            --argjson port "${UNSTRUCTURED_IO_PORT:-8000}" \
            --arg path "/general/v0/general" \
            --argjson ssl false \
            '{
                host: $host,
                port: $port,
                path: $path,
                ssl: $ssl
            }')
        
        local metadata_obj
        metadata_obj=$(jq -n \
            --arg description "Unstructured.io document processing and analysis service" \
            --arg strategy "${UNSTRUCTURED_IO_DEFAULT_STRATEGY:-hi_res}" \
            --argjson supported_formats '["pdf", "docx", "doc", "txt", "html", "xml", "pptx", "xlsx", "jpg", "png"]' \
            '{
                description: $description,
                default_strategy: $strategy,
                supported_formats: $supported_formats
            }')
        
        local connection
        connection=$(credentials::build_connection \
            "main" \
            "Unstructured.io API" \
            "httpRequest" \
            "$connection_obj" \
            "{}" \
            "$metadata_obj")
        
        connections_array="[$connection]"
    fi
    
    # Build and validate response
    local response
    response=$(credentials::build_response "unstructured-io" "$status" "$connections_array")
    
    if credentials::validate_json "$response"; then
        credentials::format_output "$response"
    else
        log::error "Invalid credentials JSON generated"
        return 1
    fi
}

# Delegate other commands to manage.sh
resource_cli::inject() {
    "${UNSTRUCTURED_IO_CLI_DIR}/manage.sh" --action inject --file "${1:-}" "${@:2}"
}

resource_cli::validate() {
    "${UNSTRUCTURED_IO_CLI_DIR}/manage.sh" --action status
}

resource_cli::status() {
    "${UNSTRUCTURED_IO_CLI_DIR}/manage.sh" --action status
}

resource_cli::start() {
    "${UNSTRUCTURED_IO_CLI_DIR}/manage.sh" --action start
}

resource_cli::stop() {
    "${UNSTRUCTURED_IO_CLI_DIR}/manage.sh" --action stop
}

resource_cli::install() {
    "${UNSTRUCTURED_IO_CLI_DIR}/manage.sh" --action install --yes yes
}

resource_cli::uninstall() {
    FORCE="${FORCE:-false}"
    if [[ "$FORCE" != "true" ]]; then
        echo "⚠️  This will remove Unstructured.io and all its data. Use --force to confirm."
        return 1
    fi
    "${UNSTRUCTURED_IO_CLI_DIR}/manage.sh" --action uninstall --yes yes
}

# Show help
resource_cli::show_help() {
    cat << EOF
🚀 Unstructured.io Resource CLI

USAGE:
    resource-unstructured-io <command> [options]

CORE COMMANDS:
    inject <file>           Process document using Unstructured.io
    validate               Validate Unstructured.io configuration
    status                 Show Unstructured.io status
    start                  Start Unstructured.io container
    stop                   Stop Unstructured.io container
    install                Install Unstructured.io
    uninstall              Uninstall Unstructured.io (requires --force)
    credentials            Get connection credentials for n8n integration

OPTIONS:
    --verbose, -v          Show detailed output
    --dry-run              Preview actions without executing
    --force                Force operation (skip confirmations)

EXAMPLES:
    resource-unstructured-io status
    resource-unstructured-io inject document.pdf
    resource-unstructured-io credentials --format pretty

For more information: https://docs.vrooli.com/resources/unstructured-io
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
        inject|validate|status|start|stop|install|uninstall|credentials)
            resource_cli::$command "$@"
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