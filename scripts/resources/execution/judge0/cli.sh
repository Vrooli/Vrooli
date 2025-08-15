#!/usr/bin/env bash
################################################################################
# Judge0 Resource CLI
# 
# Lightweight CLI interface for Judge0 using the CLI Command Framework
#
# Usage:
#   resource-judge0 <command> [options]
#
################################################################################

set -euo pipefail

# Get script directory (resolving symlinks for installed CLI)
if [[ -L "${BASH_SOURCE[0]}" ]]; then
    JUDGE0_CLI_SCRIPT="$(readlink -f "${BASH_SOURCE[0]}")"
else
    JUDGE0_CLI_SCRIPT="${BASH_SOURCE[0]}"
fi
JUDGE0_CLI_DIR="$(cd "$(dirname "$JUDGE0_CLI_SCRIPT")" && pwd)"

# Source standard variables
# shellcheck disable=SC1091
source "${JUDGE0_CLI_DIR}/../../../lib/utils/var.sh"

# Source utilities using var_ variables
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "${var_RESOURCES_COMMON_FILE}"

# Source the CLI Command Framework
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_LIB_DIR}/cli-command-framework.sh"

# Source judge0 configuration
# shellcheck disable=SC1091
source "${JUDGE0_CLI_DIR}/config/defaults.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${JUDGE0_CLI_DIR}/config/messages.sh" 2>/dev/null || true
judge0::export_config 2>/dev/null || true
judge0::export_messages 2>/dev/null || true

# Source judge0 libraries
for lib in common docker status install api languages security usage; do
    lib_file="${JUDGE0_CLI_DIR}/lib/${lib}.sh"
    if [[ -f "$lib_file" ]]; then
        # shellcheck disable=SC1090
        source "$lib_file" 2>/dev/null || true
    fi
done

# Initialize CLI framework
cli::init "judge0" "Judge0 secure code execution service management"

# Override help to provide Judge0-specific examples
cli::register_command "help" "Show this help message with Judge0 examples" "judge0_show_help"

# Register additional Judge0-specific commands
cli::register_command "inject" "Inject configuration into Judge0" "judge0_inject" "modifies-system"
cli::register_command "languages" "List supported programming languages" "judge0_languages"
cli::register_command "test" "Test API connectivity" "judge0_test"
cli::register_command "submit" "Submit code for execution" "judge0_submit"
cli::register_command "usage" "Show usage statistics" "judge0_usage"
cli::register_command "monitor" "Start security monitoring" "judge0_monitor"
cli::register_command "info" "Get system information" "judge0_info"
cli::register_command "logs" "Show container logs" "judge0_logs"
cli::register_command "credentials" "Show n8n credentials for Judge0" "judge0_credentials"
cli::register_command "uninstall" "Uninstall Judge0 (requires --force)" "judge0_uninstall" "modifies-system"

################################################################################
# Resource-specific command implementations
################################################################################

# Inject configuration or test cases into judge0
judge0_inject() {
    local file="${1:-}"
    
    if [[ -z "$file" ]]; then
        log::error "File path required for injection"
        echo "Usage: resource-judge0 inject <file.json>"
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
    
    # Use existing injection function
    if command -v judge0::inject &>/dev/null; then
        judge0::inject
    else
        "${JUDGE0_CLI_DIR}/inject.sh" --inject "$(cat "$file")"
    fi
}

# Validate judge0 configuration
judge0_validate() {
    if command -v judge0::validate &>/dev/null; then
        judge0::validate
    elif command -v judge0::api::test &>/dev/null; then
        judge0::api::test
    else
        # Basic validation
        log::header "Validating Judge0"
        docker ps --format '{{.Names}}' 2>/dev/null | grep -q "judge0" || {
            log::error "Judge0 containers not running"
            return 1
        }
        log::success "Judge0 is running"
    fi
}

# Show judge0 status
judge0_status() {
    if command -v judge0::status::show &>/dev/null; then
        judge0::status::show
    elif command -v judge0::status &>/dev/null; then
        judge0::status
    else
        # Basic status
        log::header "Judge0 Status"
        if docker ps --format '{{.Names}}' 2>/dev/null | grep -q "judge0"; then
            echo "Containers: ✅ Running"
            docker ps --filter "name=judge0" --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}" | tail -n +2
        else
            echo "Containers: ❌ Not running"
        fi
    fi
}

# Start judge0
judge0_start() {
    if command -v judge0::start &>/dev/null; then
        judge0::start
    else
        log::error "judge0::start not available"
        return 1
    fi
}

# Stop judge0
judge0_stop() {
    if command -v judge0::stop &>/dev/null; then
        judge0::stop
    else
        log::error "judge0::stop not available"
        return 1
    fi
}

# Install judge0
judge0_install() {
    if command -v judge0::install &>/dev/null; then
        judge0::install
    else
        log::error "judge0::install not available"
        return 1
    fi
}

# Uninstall judge0
judge0_uninstall() {
    FORCE="${FORCE:-false}"
    
    if [[ "$FORCE" != "true" ]]; then
        echo "⚠️  This will remove Judge0 and all its data. Use --force to confirm."
        return 1
    fi
    
    if command -v judge0::uninstall &>/dev/null; then
        judge0::uninstall
    else
        log::error "judge0::uninstall not available"
        return 1
    fi
}

# Show credentials for n8n integration
judge0_credentials() {
    source "${var_SCRIPTS_RESOURCES_LIB_DIR}/credentials-utils.sh"
    
    if ! credentials::parse_args "$@"; then
        [[ $? -eq 2 ]] && { credentials::show_help "judge0"; return 0; }
        return 1
    fi
    
    local status
    status=$(credentials::get_resource_status "${JUDGE0_CONTAINER_NAME:-vrooli-judge0-server}")
    
    local connections_array="[]"
    if [[ "$status" == "running" ]]; then
        # Judge0 requires API key authentication for security
        local api_key
        if command -v judge0::get_api_key &>/dev/null; then
            api_key=$(judge0::get_api_key)
        else
            api_key=""
        fi
        
        local connection_obj
        connection_obj=$(jq -n \
            --arg host "${JUDGE0_HOST:-localhost}" \
            --argjson port "${JUDGE0_PORT:-2358}" \
            '{
                host: $host,
                port: $port
            }')
        
        # Build auth object with X-Auth-Token header for Judge0
        local auth_obj
        auth_obj=$(jq -n \
            --arg header_name "X-Auth-Token" \
            --arg header_value "$api_key" \
            '{
                header_name: $header_name,
                header_value: $header_value
            }')
        
        # Build metadata object
        local metadata_obj
        metadata_obj=$(jq -n \
            --arg description "Judge0 secure code execution service" \
            --arg base_url "${JUDGE0_BASE_URL:-http://localhost:2358}" \
            --arg version "${JUDGE0_VERSION:-1.13.1}" \
            --argjson supported_languages "$(printf '%s\n' "${JUDGE0_DEFAULT_LANGUAGES[@]}" | head -5 | jq -R . | jq -s .)" \
            '{
                description: $description,
                base_url: $base_url,
                version: $version,
                supported_languages_sample: $supported_languages
            }')
        
        local connection
        connection=$(credentials::build_connection \
            "main" \
            "Judge0 Code Execution API" \
            "httpHeaderAuth" \
            "$connection_obj" \
            "$auth_obj" \
            "$metadata_obj")
        
        connections_array="[$connection]"
    fi
    
    local response
    response=$(credentials::build_response "judge0" "$status" "$connections_array")
    credentials::format_output "$response"
}

# List supported programming languages
judge0_languages() {
    if command -v judge0::languages::list &>/dev/null; then
        judge0::languages::list
    else
        log::error "Language listing not available"
        return 1
    fi
}

# Test API connectivity
judge0_test() {
    if command -v judge0::api::test &>/dev/null; then
        judge0::api::test
    else
        log::error "API testing not available"
        return 1
    fi
}

# Submit code for execution
judge0_submit() {
    local code="${1:-}"
    local language="${2:-javascript}"
    local stdin="${3:-}"
    
    if [[ -z "$code" ]]; then
        log::error "Code required for submission"
        echo "Usage: resource-judge0 submit <code> [language] [stdin]"
        echo ""
        echo "Examples:"
        echo "  resource-judge0 submit 'console.log(\"Hello World\")' javascript"
        echo "  resource-judge0 submit 'print(\"Hello World\")' python"
        echo "  resource-judge0 submit 'println(\"Hello World\")' java"
        echo "  resource-judge0 submit '#include <stdio.h>...printf(\"Hello\");' c"
        echo ""
        echo "Use 'resource-judge0 languages' to see supported languages"
        return 1
    fi
    
    if command -v judge0::api::submit &>/dev/null; then
        judge0::api::submit "$code" "$language" "$stdin"
    else
        log::error "Code submission not available"
        return 1
    fi
}

# Show usage statistics
judge0_usage() {
    if command -v judge0::usage::show &>/dev/null; then
        judge0::usage::show
    else
        log::error "Usage statistics not available"
        return 1
    fi
}

# Start security monitoring
judge0_monitor() {
    if command -v judge0::security::monitor &>/dev/null; then
        judge0::security::monitor
    elif [[ -f "${JUDGE0_CLI_DIR}/lib/security-monitor.sh" ]]; then
        log::info "Starting Judge0 security monitoring..."
        log::warning "WARNING: Running with elevated privileges - monitor actively for security alerts"
        "${JUDGE0_CLI_DIR}/lib/security-monitor.sh"
    else
        log::error "Security monitoring not available"
        return 1
    fi
}

# Get system information
judge0_info() {
    if command -v judge0::api::system_info &>/dev/null; then
        judge0::api::system_info
    else
        log::error "System information not available"
        return 1
    fi
}

# Show logs
judge0_logs() {
    if command -v judge0::docker::logs &>/dev/null; then
        judge0::docker::logs
    else
        docker logs judge0-server 2>/dev/null || docker logs judge0_server 2>/dev/null || {
            log::error "Unable to retrieve logs"
            return 1
        }
    fi
}

# Custom help function with Judge0-specific examples
judge0_show_help() {
    # Show standard framework help first
    cli::_handle_help
    
    # Add Judge0-specific examples
    echo ""
    echo "⚖️  Judge0 Code Execution Examples:"
    echo ""
    echo "Code Execution:"
    echo "  resource-judge0 submit 'console.log(\"Hello World\")' javascript"
    echo "  resource-judge0 submit 'print(\"Hello World\")' python"
    echo "  resource-judge0 submit 'println(\"Hello World\")' java"
    echo "  resource-judge0 submit '#include <stdio.h>...printf(\"Hello\");' c"
    echo ""
    echo "Language Support:"
    echo "  resource-judge0 languages                    # List all supported languages"
    echo "  resource-judge0 test                         # Test API connectivity"
    echo ""
    echo "Monitoring & Management:"
    echo "  resource-judge0 usage                        # Show execution statistics"
    echo "  resource-judge0 monitor                      # Start security monitoring"
    echo "  resource-judge0 info                         # Show system information"
    echo "  resource-judge0 logs                         # View container logs"
    echo ""
    echo "Security Features:"
    echo "  • Secure sandboxed execution environment"
    echo "  • API key authentication"
    echo "  • Resource limits and timeouts"
    echo "  • Real-time security monitoring"
    echo ""
    echo "Popular Languages: JavaScript, Python, Java, C, C++, Go, Rust, PHP, Ruby"
    echo "For complete language list: resource-judge0 languages"
}

################################################################################
# Main execution - dispatch to framework
################################################################################

# Only execute if script is run directly (not sourced)
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    cli::dispatch "$@"
fi