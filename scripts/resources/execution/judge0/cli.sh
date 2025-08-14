#!/usr/bin/env bash
################################################################################
# Judge0 Resource CLI
# 
# Lightweight CLI interface for Judge0 that delegates to existing lib functions.
#
# Usage:
#   resource-judge0 <command> [options]
#
################################################################################

set -euo pipefail

# Get script directory (handle symlinks)
JUDGE0_CLI_DIR="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
VROOLI_ROOT="${VROOLI_ROOT:-$(cd "$JUDGE0_CLI_DIR/../../../.." && pwd)}"
export VROOLI_ROOT
export RESOURCE_DIR="$JUDGE0_CLI_DIR"
export JUDGE0_SCRIPT_DIR="$JUDGE0_CLI_DIR"  # For compatibility with existing libs

# Source utilities first
# shellcheck disable=SC1091
source "${VROOLI_ROOT}/scripts/lib/utils/var.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${var_LOG_FILE:-${VROOLI_ROOT}/scripts/lib/utils/log.sh}" 2>/dev/null || true
# shellcheck disable=SC1091
source "${var_RESOURCES_COMMON_FILE:-${VROOLI_ROOT}/scripts/resources/common.sh}" 2>/dev/null || true

# Source the CLI template
# shellcheck disable=SC1091
source "${VROOLI_ROOT}/scripts/lib/resources/cli/resource-cli-template.sh"

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

# Initialize with resource name
resource_cli::init "judge0"

################################################################################
# Delegate to existing judge0 functions
################################################################################

# Inject configuration or test cases into judge0
resource_cli::inject() {
    local file="${1:-}"
    DRY_RUN="${DRY_RUN:-false}"
    
    if [[ -z "$file" ]]; then
        log::error "File path required for injection"
        echo "Usage: resource-judge0 inject <file.json>"
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
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log::info "[DRY RUN] Would inject: $file"
        return 0
    fi
    
    # Use existing injection function
    if command -v judge0::inject &>/dev/null; then
        judge0::inject
    else
        "${JUDGE0_CLI_DIR}/inject.sh" --inject "$(cat "$file")"
    fi
}

# Validate judge0 configuration
resource_cli::validate() {
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
resource_cli::status() {
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
resource_cli::start() {
    DRY_RUN="${DRY_RUN:-false}"
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log::info "[DRY RUN] Would start Judge0"
        return 0
    fi
    
    if command -v judge0::start &>/dev/null; then
        judge0::start
    else
        log::error "judge0::start not available"
        return 1
    fi
}

# Stop judge0
resource_cli::stop() {
    DRY_RUN="${DRY_RUN:-false}"
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log::info "[DRY RUN] Would stop Judge0"
        return 0
    fi
    
    if command -v judge0::stop &>/dev/null; then
        judge0::stop
    else
        log::error "judge0::stop not available"
        return 1
    fi
}

# Install judge0
resource_cli::install() {
    DRY_RUN="${DRY_RUN:-false}"
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log::info "[DRY RUN] Would install Judge0"
        return 0
    fi
    
    if command -v judge0::install &>/dev/null; then
        judge0::install
    else
        log::error "judge0::install not available"
        return 1
    fi
}

# Uninstall judge0
resource_cli::uninstall() {
    FORCE="${FORCE:-false}"
    DRY_RUN="${DRY_RUN:-false}"
    
    if [[ "$FORCE" != "true" ]]; then
        echo "⚠️  This will remove Judge0 and all its data. Use --force to confirm."
        return 1
    fi
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log::info "[DRY RUN] Would uninstall Judge0"
        return 0
    fi
    
    if command -v judge0::uninstall &>/dev/null; then
        judge0::uninstall
    else
        log::error "judge0::uninstall not available"
        return 1
    fi
}

################################################################################
# Judge0-specific commands (if functions exist)
################################################################################

# Show credentials for n8n integration
resource_cli::credentials() {
    source "${VROOLI_ROOT}/scripts/resources/lib/credentials-utils.sh"
    
    if ! credentials::parse_args "$@"; then
        [[ $? -eq 2 ]] && { credentials::show_help "$RESOURCE_NAME"; return 0; }
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
    response=$(credentials::build_response "$RESOURCE_NAME" "$status" "$connections_array")
    credentials::format_output "$response"
}

# List supported programming languages
judge0_list_languages() {
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
        echo "Example: resource-judge0 submit 'console.log(\"Hello World\")' javascript"
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

# Show help
resource_cli::show_help() {
    cat << EOF
⚖️  Judge0 Resource CLI

USAGE:
    resource-judge0 <command> [options]

CORE COMMANDS:
    inject <file>            Inject configuration into Judge0
    validate                 Validate Judge0 configuration
    status                   Show Judge0 status
    start                    Start Judge0 containers
    stop                     Stop Judge0 containers
    install                  Install Judge0
    uninstall                Uninstall Judge0 (requires --force)
    credentials              Show n8n credentials for Judge0
    
JUDGE0 COMMANDS:
    languages                List supported programming languages
    test                     Test API connectivity
    submit <code> [lang]     Submit code for execution
    usage                    Show usage statistics
    monitor                  Start security monitoring
    info                     Get system information
    logs                     Show container logs

OPTIONS:
    --verbose, -v            Show detailed output
    --dry-run                Preview actions without executing
    --force                  Force operation (skip confirmations)

EXAMPLES:
    resource-judge0 status
    resource-judge0 languages
    resource-judge0 test
    resource-judge0 submit 'print("Hello World")' python
    resource-judge0 submit 'console.log("Hello")' javascript
    resource-judge0 monitor

For more information: https://docs.vrooli.com/resources/judge0
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
            
        # Judge0-specific commands
        languages)
            judge0_list_languages "$@"
            ;;
        test)
            judge0_test "$@"
            ;;
        submit)
            judge0_submit "$@"
            ;;
        usage)
            judge0_usage "$@"
            ;;
        monitor)
            judge0_monitor "$@"
            ;;
        info)
            judge0_info "$@"
            ;;
        logs)
            judge0_logs "$@"
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