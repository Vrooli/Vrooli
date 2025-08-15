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

# Get script directory (resolving symlinks)
if [[ -L "${BASH_SOURCE[0]}" ]]; then
    # If this is a symlink, resolve it
    BROWSERLESS_CLI_SCRIPT="$(readlink -f "${BASH_SOURCE[0]}")"
else
    BROWSERLESS_CLI_SCRIPT="${BASH_SOURCE[0]}"
fi
BROWSERLESS_CLI_DIR="$(cd "$(dirname "$BROWSERLESS_CLI_SCRIPT")" && pwd)"
VROOLI_ROOT="${VROOLI_ROOT:-$(cd "$BROWSERLESS_CLI_DIR/../../../.." && pwd)}"
export VROOLI_ROOT
export RESOURCE_DIR="$BROWSERLESS_CLI_DIR"
export BROWSERLESS_SCRIPT_DIR="$BROWSERLESS_CLI_DIR"  # For compatibility with existing libs

# Source utilities first
# shellcheck disable=SC1091
source "${VROOLI_ROOT}/scripts/lib/utils/var.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${var_LOG_FILE:-${VROOLI_ROOT}/scripts/lib/utils/log.sh}" 2>/dev/null || true
# shellcheck disable=SC1091
source "${var_RESOURCES_COMMON_FILE:-${VROOLI_ROOT}/scripts/resources/common.sh}" 2>/dev/null || true

# Source the CLI Command Framework
# shellcheck disable=SC1091
source "${VROOLI_ROOT}/scripts/resources/lib/cli-command-framework.sh"

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
cli::register_command "inject" "Inject configuration into Browserless" "resource_cli::inject" "modifies-system"
cli::register_command "screenshot" "Take screenshot of URL" "resource_cli::screenshot"
cli::register_command "pdf" "Generate PDF from URL" "resource_cli::pdf"
cli::register_command "test-apis" "Test all Browserless APIs" "resource_cli::test_apis"
cli::register_command "metrics" "Show browser pressure/metrics" "resource_cli::metrics"
cli::register_command "credentials" "Get connection credentials for n8n integration" "resource_cli::credentials"
cli::register_command "uninstall" "Uninstall Browserless (requires --force)" "resource_cli::uninstall" "modifies-system"

################################################################################
# Resource-specific command implementations
################################################################################

# Inject configuration into browserless
resource_cli::inject() {
    local file="${1:-}"
    
    if [[ -z "$file" ]]; then
        log::error "File path required for injection"
        echo "Usage: resource-browserless inject <file.json>"
        return 1
    fi
    
    # Handle shared: prefix
    if [[ "$file" == shared:* ]]; then
        file="${VROOLI_ROOT}/${file#shared:}"
    fi
    
    if [[ ! -f "$file" ]]; then
        log::error "File not found: $file"
        return 1
    fi
    
    # Use existing injection function
    if command -v browserless::inject &>/dev/null; then
        browserless::inject "$file"
    else
        log::error "Browserless injection function not available"
        return 1
    fi
}

# Validate browserless configuration
resource_cli::validate() {
    if command -v browserless::health &>/dev/null; then
        browserless::health
    elif command -v browserless::check_basic_health &>/dev/null; then
        browserless::check_basic_health
    else
        # Basic validation
        log::header "Validating Browserless"
        docker ps --format '{{.Names}}' 2>/dev/null | grep -q "browserless" || {
            log::error "Browserless container not running"
            return 1
        }
        log::success "Browserless is running"
    fi
}

# Show browserless status
resource_cli::status() {
    if command -v browserless::status &>/dev/null; then
        browserless::status
    else
        # Basic status
        log::header "Browserless Status"
        if docker ps --format '{{.Names}}' 2>/dev/null | grep -q "browserless"; then
            echo "Container: ✅ Running"
            docker ps --filter "name=browserless" --format "table {{.Status}}\t{{.Ports}}" | tail -n 1
        else
            echo "Container: ❌ Not running"
        fi
    fi
}

# Start browserless
resource_cli::start() {
    if command -v browserless::start &>/dev/null; then
        browserless::start
    else
        docker start browserless || log::error "Failed to start Browserless"
    fi
}

# Stop browserless
resource_cli::stop() {
    if command -v browserless::stop &>/dev/null; then
        browserless::stop
    else
        docker stop browserless || log::error "Failed to stop Browserless"
    fi
}

# Install browserless
resource_cli::install() {
    if command -v browserless::install &>/dev/null; then
        browserless::install
    else
        log::error "browserless::install not available"
        return 1
    fi
}

# Uninstall browserless
resource_cli::uninstall() {
    FORCE="${FORCE:-false}"
    
    if [[ "$FORCE" != "true" ]]; then
        echo "⚠️  This will remove Browserless and all its data. Use --force to confirm."
        return 1
    fi
    
    if command -v browserless::uninstall &>/dev/null; then
        browserless::uninstall
    else
        docker stop browserless 2>/dev/null || true
        docker rm browserless 2>/dev/null || true
        log::success "Browserless uninstalled"
    fi
}

# Take screenshot
resource_cli::screenshot() {
    local url="${1:-}"
    local output="${2:-screenshot.png}"
    
    if [[ -z "$url" ]]; then
        log::error "URL required for screenshot"
        echo "Usage: resource-browserless screenshot <url> [output-file]"
        return 1
    fi
    
    if command -v browserless::screenshot &>/dev/null; then
        browserless::screenshot "$url" "$output"
    else
        log::error "Screenshot function not available"
        return 1
    fi
}

# Generate PDF
resource_cli::pdf() {
    local url="${1:-}"
    local output="${2:-output.pdf}"
    
    if [[ -z "$url" ]]; then
        log::error "URL required for PDF generation"
        echo "Usage: resource-browserless pdf <url> [output-file]"
        return 1
    fi
    
    if command -v browserless::pdf &>/dev/null; then
        browserless::pdf "$url" "$output"
    else
        log::error "PDF generation function not available"
        return 1
    fi
}

# Test all APIs
resource_cli::test_apis() {
    if command -v browserless::test_all_apis &>/dev/null; then
        browserless::test_all_apis
    else
        log::error "API testing not available"
        return 1
    fi
}

# Show browser pressure/metrics
resource_cli::metrics() {
    if command -v browserless::test_pressure &>/dev/null; then
        browserless::test_pressure
    else
        # Basic metrics via API
        local url="http://localhost:${BROWSERLESS_PORT:-3000}/pressure"
        if curl -s "$url" 2>/dev/null; then
            curl -s "$url" | jq '.' 2>/dev/null || curl -s "$url"
        else
            log::error "Could not fetch metrics"
            return 1
        fi
    fi
}

# Get credentials for n8n integration  
resource_cli::credentials() {
    # Source credentials utilities
    # shellcheck disable=SC1091
    source "${VROOLI_ROOT}/scripts/resources/lib/credentials-utils.sh"
    
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

################################################################################
# Main execution - dispatch to framework
################################################################################

# Only execute if script is run directly (not sourced)
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    cli::dispatch "$@"
fi