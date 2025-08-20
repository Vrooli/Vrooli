#!/usr/bin/env bash
################################################################################
# Browserless Resource CLI
# 
# Lightweight CLI interface for Browserless using the CLI Command Framework
#
# Usage:
#   resource-browserless <command> [options]
#
################################################################################

set -euo pipefail

# Get script directory (resolving symlinks for installed CLI)
if [[ -L "${BASH_SOURCE[0]}" ]]; then
    BROWSERLESS_CLI_SCRIPT="$(readlink -f "${BASH_SOURCE[0]}")"
else
    BROWSERLESS_CLI_SCRIPT="${BASH_SOURCE[0]}"
fi
BROWSERLESS_CLI_DIR="$(cd "$(dirname "$BROWSERLESS_CLI_SCRIPT")" && pwd)"

# Source standard variables
# shellcheck disable=SC1091
source "${BROWSERLESS_CLI_DIR}/../../../lib/utils/var.sh"

# Source utilities using var_ variables
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "${var_RESOURCES_COMMON_FILE}"

# Source the CLI Command Framework
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_LIB_DIR}/cli-command-framework.sh"

# Source browserless configuration
# shellcheck disable=SC1091
source "${BROWSERLESS_CLI_DIR}/config/defaults.sh" 2>/dev/null || true
browserless::export_config 2>/dev/null || true

# Source browserless libraries
for lib in core docker health status api inject usage recovery; do
    lib_file="${BROWSERLESS_CLI_DIR}/lib/${lib}.sh"
    if [[ -f "$lib_file" ]]; then
        # shellcheck disable=SC1090
        source "$lib_file" 2>/dev/null || true
    fi
done

# Source adapter framework if available
if [[ -f "${BROWSERLESS_CLI_DIR}/adapters/common.sh" ]]; then
    # shellcheck disable=SC1090
    source "${BROWSERLESS_CLI_DIR}/adapters/common.sh"
    # shellcheck disable=SC1090
    source "${BROWSERLESS_CLI_DIR}/adapters/registry.sh"
fi

# Initialize CLI framework
cli::init "browserless" "Browserless Chrome automation management"

# Register additional Browserless-specific commands
cli::register_command "inject" "Inject configuration into Browserless" "browserless_inject" "modifies-system"
cli::register_command "screenshot" "Take screenshot of URL" "browserless_screenshot"
cli::register_command "pdf" "Generate PDF from URL" "browserless_pdf"
cli::register_command "test-apis" "Test all Browserless APIs" "browserless_test_apis"
cli::register_command "metrics" "Show browser pressure/metrics" "browserless_metrics"
cli::register_command "credentials" "Get connection credentials for n8n integration" "browserless_credentials"
cli::register_command "console-capture" "Capture console logs from any URL" "browserless_console_capture"
cli::register_command "uninstall" "Uninstall Browserless (requires --force)" "browserless_uninstall" "modifies-system"

# Register adapter command for the new "for" pattern
cli::register_command "for" "Use browserless as adapter for other resources" "browserless_adapter"

# Legacy commands for backward compatibility (will delegate to adapter)
cli::register_command "execute-workflow" "[LEGACY] Execute n8n workflow - use 'for n8n execute-workflow' instead" "browserless_execute_workflow_legacy"

################################################################################
# Resource-specific command implementations  
################################################################################

# Inject configuration into browserless
browserless_inject() {
    local file="${1:-}"
    
    if [[ -z "$file" ]]; then
        log::error "File path required for injection"
        echo "Usage: resource-browserless inject <file.json>"
        return 1
    fi
    
    # Handle shared: prefix
    if [[ "$file" == shared:* ]]; then
        file="${var_ROOT_DIR}/${file#shared:}"
    fi
    
    if [[ ! -f "$file" ]]; then
        log::error "File not found: $file"
        return 1
    fi
    
    browserless::inject "$file"
}

# Take screenshot
browserless_screenshot() {
    local url="${1:-}"
    local output="${2:-screenshot.png}"
    
    if [[ -z "$url" ]]; then
        log::error "URL required for screenshot"
        echo "Usage: resource-browserless screenshot <url> [output-file]"
        return 1
    fi
    
    browserless::safe_screenshot "$url" "$output"
}

# Generate PDF
browserless_pdf() {
    local url="${1:-}"
    local output="${2:-output.pdf}"
    
    if [[ -z "$url" ]]; then
        log::error "URL required for PDF generation"
        echo "Usage: resource-browserless pdf <url> [output-file]"
        return 1
    fi
    
    browserless::test_pdf "$url" "$output"
}

# Test all APIs
browserless_test_apis() {
    browserless::test_all_apis
}

# Show browser pressure/metrics
browserless_metrics() {
    browserless::test_pressure
}

# Get credentials for n8n integration  
browserless_credentials() {
    # Source credentials utilities
    # shellcheck disable=SC1091
    source "${var_SCRIPTS_RESOURCES_LIB_DIR}/credentials-utils.sh"
    
    # Parse arguments
    credentials::parse_args "$@"
    local parse_result=$?
    if [[ $parse_result -eq 2 ]]; then
        credentials::show_help "browserless"
        return 0
    elif [[ $parse_result -ne 0 ]]; then
        return 1
    fi
    
    # Get resource status by checking Docker container
    local status
    status=$(credentials::get_resource_status "$BROWSERLESS_CONTAINER_NAME")
    
    # Build connections array for Browserless
    local connections_array="[]"
    if [[ "$status" == "running" ]]; then
        # Build connection JSON for Browserless HTTP API
        local connection_json
        connection_json=$(jq -n \
            --arg id "api" \
            --arg name "Browserless API" \
            --arg n8n_credential_type "httpHeaderAuth" \
            --arg host "localhost" \
            --arg port "${BROWSERLESS_PORT}" \
            --arg path "/json" \
            --arg description "Browserless Chrome automation API" \
            '{
                id: $id,
                name: $name,
                n8n_credential_type: $n8n_credential_type,
                connection: {
                    host: $host,
                    port: ($port | tonumber),
                    path: $path,
                    ssl: false
                },
                auth: {
                    header_name: "Authorization",
                    header_value: "",
                    token: null,
                    api_key: null
                },
                metadata: {
                    description: $description,
                    capabilities: ["browser", "pdf", "screenshot", "scraping", "automation"],
                    version: "latest"
                }
            }')
        
        connections_array=$(echo "$connection_json" | jq -s '.')
    fi
    
    # Build and validate response
    local response
    response=$(credentials::build_response "browserless" "$status" "$connections_array")
    
    if credentials::validate_json "$response"; then
        credentials::format_output "$response"
    else
        log::error "Invalid credentials JSON generated"
        return 1
    fi
}

# Adapter command handler - implements the "for" pattern
browserless_adapter() {
    local target_resource="${1:-}"
    shift || true
    
    if [[ -z "$target_resource" ]]; then
        log::error "Target resource required"
        echo "Usage: resource-browserless for <resource> <command> [options]"
        echo ""
        echo "Available adapters:"
        if declare -f adapter::list >/dev/null; then
            adapter::list
        else
            echo "  n8n     - UI automation for n8n workflows"
            echo "  vault   - UI automation for HashiCorp Vault (coming soon)"
            echo "  grafana - Dashboard automation (coming soon)"
        fi
        return 1
    fi
    
    # Load the appropriate adapter
    local adapter_dir="${BROWSERLESS_CLI_DIR}/adapters/${target_resource}"
    local adapter_api="${adapter_dir}/api.sh"
    
    if [[ ! -f "$adapter_api" ]]; then
        log::error "Adapter not found for resource: $target_resource"
        echo "Available adapters:"
        ls -d "${BROWSERLESS_CLI_DIR}/adapters/"*/ 2>/dev/null | xargs -n1 basename | grep -v common | grep -v registry
        return 1
    fi
    
    # Source the adapter
    # shellcheck disable=SC1090
    source "$adapter_api"
    
    # Dispatch to adapter
    if declare -f "${target_resource}::dispatch" >/dev/null; then
        "${target_resource}::dispatch" "$@"
    else
        log::error "Adapter dispatch function not found: ${target_resource}::dispatch"
        return 1
    fi
}

# Legacy execute-workflow command for backward compatibility
browserless_execute_workflow_legacy() {
    log::warn "⚠️  This command syntax is deprecated. Please use: resource-browserless for n8n execute-workflow"
    echo ""
    
    # Delegate to adapter
    browserless_adapter "n8n" "execute-workflow" "$@"
}

# Execute n8n workflow via browser automation with enhanced session management
browserless_execute_workflow() {
    local workflow_id="${1:-}"
    local n8n_url="${2:-http://localhost:5678}"
    local timeout="${3:-60000}"
    local input_data="${4:-}"
    local use_persistent_session="${5:-true}"
    local session_id="${6:-}"
    
    if [[ -z "$workflow_id" ]]; then
        log::error "Workflow ID required"
        echo "Usage: resource-browserless execute-workflow <workflow-id> [n8n-url] [timeout-ms] [input-json] [persistent-session] [session-id]"
        echo ""
        echo "Enhanced Features:"
        echo "  • Persistent browser sessions (cookies, auth state preserved)"
        echo "  • Extended execution monitoring (waits for actual completion)"
        echo "  • High-quality final state screenshots"
        echo "  • Comprehensive execution state tracking"
        echo ""
        echo "Parameters:"
        echo "  workflow-id        : N8N workflow ID (required)"
        echo "  n8n-url           : N8N instance URL (default: http://localhost:5678)"
        echo "  timeout-ms        : Timeout in milliseconds (default: 60000)"
        echo "  input-json        : Input data as JSON, @file, or \$ENV_VAR (optional)"
        echo "  persistent-session: Use persistent session (true/false, default: true)"
        echo "  session-id        : Custom session ID (auto-generated if not provided)"
        echo ""
        echo "Examples:"
        echo "  # Execute workflow with persistent session (recommended)"
        echo "  resource-browserless execute-workflow my-workflow"
        echo ""
        echo "  # Execute with custom session ID for reuse"
        echo "  resource-browserless execute-workflow my-workflow http://localhost:5678 60000 '{}' true my-session-1"
        echo ""
        echo "  # Execute without session persistence (fresh browser each time)"
        echo "  resource-browserless execute-workflow my-workflow http://localhost:5678 60000 '{}' false"
        echo ""
        echo "  # Execute with JSON input data"
        echo "  resource-browserless execute-workflow embedding-generator http://localhost:5678 60000 '{\"text\":\"Hello world\"}'"
        echo ""
        echo "  # Execute with JSON file input"
        echo "  resource-browserless execute-workflow my-workflow http://localhost:5678 60000 @input.json"
        echo ""
        echo "  # Execute with environment variable input"
        echo "  export WORKFLOW_INPUT='{\"text\":\"test\"}'"
        echo "  resource-browserless execute-workflow embedding-generator"
        return 1
    fi
    
    browserless::execute_n8n_workflow "$workflow_id" "$n8n_url" "$timeout" "$input_data" "$use_persistent_session" "$session_id"
}

# Capture console logs from any URL
browserless_console_capture() {
    local url="${1:-}"
    local output="${2:-console-capture.json}"
    local wait_time="${3:-3000}"
    
    if [[ -z "$url" ]]; then
        log::error "URL required for console capture"
        echo "Usage: resource-browserless console-capture <url> [output-file] [wait-time-ms]"
        echo "Example: resource-browserless console-capture http://localhost:3000 logs.json 5000"
        return 1
    fi
    
    browserless::capture_console_logs "$url" "$output" "$wait_time"
}

# Uninstall browserless
browserless_uninstall() {
    FORCE="${FORCE:-false}"
    
    if [[ "$FORCE" != "true" ]]; then
        echo "⚠️  This will remove Browserless and all its data. Use --force to confirm."
        return 1
    fi
    
    browserless::uninstall
}

################################################################################
# Main execution - dispatch to framework
################################################################################

# Only execute if script is run directly (not sourced)
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    cli::dispatch "$@"
fi