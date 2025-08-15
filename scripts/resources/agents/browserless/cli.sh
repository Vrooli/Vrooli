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

# Initialize CLI framework
cli::init "browserless" "Browserless Chrome automation management"

# Register additional Browserless-specific commands
cli::register_command "inject" "Inject configuration into Browserless" "browserless_inject" "modifies-system"
cli::register_command "screenshot" "Take screenshot of URL" "browserless_screenshot"
cli::register_command "pdf" "Generate PDF from URL" "browserless_pdf"
cli::register_command "test-apis" "Test all Browserless APIs" "browserless_test_apis"
cli::register_command "metrics" "Show browser pressure/metrics" "browserless_metrics"
cli::register_command "credentials" "Get connection credentials for n8n integration" "browserless_credentials"
cli::register_command "execute-workflow" "Execute n8n workflow and monitor via browser" "browserless_execute_workflow"
cli::register_command "console-capture" "Capture console logs from any URL" "browserless_console_capture"
cli::register_command "uninstall" "Uninstall Browserless (requires --force)" "browserless_uninstall" "modifies-system"

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
    
    browserless::screenshot "$url" "$output"
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
    
    browserless::pdf "$url" "$output"
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

# Execute n8n workflow via browser automation
browserless_execute_workflow() {
    local workflow_id="${1:-}"
    local n8n_url="${2:-http://localhost:5678}"
    local timeout="${3:-60000}"
    
    if [[ -z "$workflow_id" ]]; then
        log::error "Workflow ID required"
        echo "Usage: resource-browserless execute-workflow <workflow-id> [n8n-url] [timeout-ms]"
        echo "Example: resource-browserless execute-workflow my-workflow http://localhost:5678 30000"
        return 1
    fi
    
    browserless::execute_n8n_workflow "$workflow_id" "$n8n_url" "$timeout"
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