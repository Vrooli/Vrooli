#!/usr/bin/env bash
################################################################################
# agent-s2 Resource CLI
# 
# Lightweight CLI interface for agent-s2 that delegates to existing lib functions.
#
# Usage:
#   resource-agent-s2 <command> [options]
#
################################################################################

set -euo pipefail

# Get script directory (resolving symlinks)
if [[ -L "${BASH_SOURCE[0]}" ]]; then
    # If this is a symlink, resolve it
    AGENT_S2_CLI_SCRIPT="$(readlink -f "${BASH_SOURCE[0]}")"
else
    AGENT_S2_CLI_SCRIPT="${BASH_SOURCE[0]}"
fi
AGENT_S2_CLI_DIR="$(cd "$(dirname "$AGENT_S2_CLI_SCRIPT")" && pwd)"
VROOLI_ROOT="${VROOLI_ROOT:-$(cd "$AGENT_S2_CLI_DIR/../../../.." && pwd)}"
export VROOLI_ROOT
export RESOURCE_DIR="$AGENT_S2_CLI_DIR"
export AGENT_S2_SCRIPT_DIR="$AGENT_S2_CLI_DIR"  # For compatibility with existing libs

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

# Source agent-s2 configuration
# shellcheck disable=SC1091
source "${AGENT_S2_CLI_DIR}/config/defaults.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${AGENT_S2_CLI_DIR}/config/messages.sh" 2>/dev/null || true
agents2::export_config 2>/dev/null || true
agents2::export_messages 2>/dev/null || true

# Source agent-s2 libraries
for lib in common docker status api install modes; do
    lib_file="${AGENT_S2_CLI_DIR}/lib/${lib}.sh"
    if [[ -f "$lib_file" ]]; then
        # shellcheck disable=SC1090
        source "$lib_file" 2>/dev/null || true
    fi
done

# Initialize with resource name
resource_cli::init "agent-s2"

################################################################################
# Delegate to existing agent-s2 functions
################################################################################

# Inject automation or configuration into agent-s2
resource_cli::inject() {
    local file="${1:-}"
    DRY_RUN="${DRY_RUN:-false}"
    
    if [[ -z "$file" ]]; then
        log::error "File path required for injection"
        echo "Usage: resource-agent-s2 inject <file.json>"
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
    
    # Use existing injection script
    local inject_script="${AGENT_S2_CLI_DIR}/inject.sh"
    if [[ -f "$inject_script" ]]; then
        "$inject_script" "$file"
    else
        log::error "agent-s2 injection script not available"
        return 1
    fi
}

# Validate agent-s2 configuration
resource_cli::validate() {
    if command -v agents2::is_healthy &>/dev/null; then
        agents2::is_healthy
    else
        # Basic validation
        log::header "Validating agent-s2"
        docker ps --format '{{.Names}}' 2>/dev/null | grep -q "${AGENTS2_CONTAINER_NAME:-agent-s2}" || {
            log::error "agent-s2 container not running"
            return 1
        }
        log::success "agent-s2 is running"
    fi
}

# Show agent-s2 status
resource_cli::status() {
    if command -v agents2::show_status &>/dev/null; then
        agents2::show_status
    else
        # Basic status
        log::header "agent-s2 Status"
        local container_name="${AGENTS2_CONTAINER_NAME:-agent-s2}"
        if docker ps --format '{{.Names}}' 2>/dev/null | grep -q "$container_name"; then
            echo "Container: ✅ Running"
            docker ps --filter "name=$container_name" --format "table {{.Status}}\t{{.Ports}}" | tail -n 1
        else
            echo "Container: ❌ Not running"
        fi
    fi
}

# Start agent-s2
resource_cli::start() {
    DRY_RUN="${DRY_RUN:-false}"
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log::info "[DRY RUN] Would start agent-s2"
        return 0
    fi
    
    if command -v agents2::docker_start &>/dev/null; then
        agents2::docker_start
    elif command -v agents2::start_in_mode &>/dev/null; then
        agents2::start_in_mode "${MODE:-sandbox}"
    else
        local container_name="${AGENTS2_CONTAINER_NAME:-agent-s2}"
        docker start "$container_name" || log::error "Failed to start agent-s2"
    fi
}

# Stop agent-s2
resource_cli::stop() {
    DRY_RUN="${DRY_RUN:-false}"
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log::info "[DRY RUN] Would stop agent-s2"
        return 0
    fi
    
    if command -v agents2::docker_stop &>/dev/null; then
        agents2::docker_stop
    else
        local container_name="${AGENTS2_CONTAINER_NAME:-agent-s2}"
        docker stop "$container_name" || log::error "Failed to stop agent-s2"
    fi
}

# Install agent-s2
resource_cli::install() {
    DRY_RUN="${DRY_RUN:-false}"
    FORCE="${FORCE:-false}"
    export FORCE
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log::info "[DRY RUN] Would install agent-s2"
        return 0
    fi
    
    if command -v agents2::install_service &>/dev/null; then
        agents2::install_service
    else
        log::error "agents2::install_service not available"
        return 1
    fi
}

# Uninstall agent-s2
resource_cli::uninstall() {
    FORCE="${FORCE:-false}"
    DRY_RUN="${DRY_RUN:-false}"
    YES="${FORCE}"  # Set YES to FORCE for uninstall prompts
    export FORCE YES
    
    if [[ "$FORCE" != "true" ]]; then
        echo "⚠️  This will remove agent-s2 and all its data. Use --force to confirm."
        return 1
    fi
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log::info "[DRY RUN] Would uninstall agent-s2"
        return 0
    fi
    
    if command -v agents2::uninstall_service &>/dev/null; then
        agents2::uninstall_service
    else
        local container_name="${AGENTS2_CONTAINER_NAME:-agent-s2}"
        docker stop "$container_name" 2>/dev/null || true
        docker rm "$container_name" 2>/dev/null || true
        log::success "agent-s2 uninstalled"
    fi
}

################################################################################
# agent-s2-specific commands (if functions exist)
################################################################################

# Take screenshot
agent_s2_screenshot() {
    local output_file="${1:-screenshot.png}"
    
    if command -v agents2::api_request &>/dev/null; then
        local response
        response=$(agents2::api_request "GET" "/screenshot")
        if [[ $? -eq 0 ]]; then
            echo "$response" | base64 -d > "$output_file"
            log::success "Screenshot saved to: $output_file"
        else
            log::error "Failed to take screenshot"
            return 1
        fi
    else
        log::error "Screenshot functionality not available"
        return 1
    fi
}

# Execute automation task
agent_s2_execute() {
    local task="${1:-}"
    
    if [[ -z "$task" ]]; then
        log::error "Task required for execution"
        echo "Usage: resource-agent-s2 execute '<task description>'"
        return 1
    fi
    
    if command -v agents2::api_request &>/dev/null; then
        local payload
        payload=$(jq -n --arg task "$task" '{task: $task}')
        agents2::api_request "POST" "/execute" "$payload"
    else
        log::error "Execute functionality not available"
        return 1
    fi
}

# Switch operation mode
agent_s2_switch_mode() {
    local mode="${1:-}"
    
    if [[ -z "$mode" ]]; then
        log::error "Mode required (sandbox|host)"
        echo "Usage: resource-agent-s2 switch-mode <sandbox|host>"
        return 1
    fi
    
    if command -v agents2::switch_mode &>/dev/null; then
        agents2::switch_mode "$mode"
    elif command -v agents2::api_request &>/dev/null; then
        local payload
        payload=$(jq -n --arg mode "$mode" '{mode: $mode}')
        agents2::api_request "POST" "/modes/switch" "$payload"
    else
        log::error "Mode switching not available"
        return 1
    fi
}

# Show current mode
agent_s2_show_mode() {
    if command -v agents2::api_request &>/dev/null; then
        agents2::api_request "GET" "/modes/current"
    else
        log::error "Mode query not available"
        return 1
    fi
}

# Show health information
agent_s2_health() {
    if command -v agents2::api_request &>/dev/null; then
        agents2::api_request "GET" "/health"
    else
        log::error "Health check not available"
        return 1
    fi
}

# Show help
resource_cli::show_help() {
    cat << EOF
🤖 agent-s2 Resource CLI

USAGE:
    resource-agent-s2 <command> [options]

CORE COMMANDS:
    inject <file>       Inject automation/config into agent-s2
    validate            Validate agent-s2 configuration
    status              Show agent-s2 status
    start               Start agent-s2 container
    stop                Stop agent-s2 container
    install             Install agent-s2
    uninstall           Uninstall agent-s2 (requires --force)
    
AGENT-S2 COMMANDS:
    screenshot [file]   Take screenshot (default: screenshot.png)
    execute <task>      Execute automation task
    switch-mode <mode>  Switch between sandbox/host mode
    show-mode           Show current operation mode
    health              Show detailed health information

OPTIONS:
    --verbose, -v       Show detailed output
    --dry-run           Preview actions without executing
    --force             Force operation (skip confirmations)

EXAMPLES:
    resource-agent-s2 status
    resource-agent-s2 inject shared:initialization/automation/agent-s2/config.json
    resource-agent-s2 screenshot desktop.png
    resource-agent-s2 execute "Click the login button"
    resource-agent-s2 switch-mode host
    resource-agent-s2 show-mode

For more information: https://docs.vrooli.com/resources/agent-s2
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
        inject|validate|status|start|stop|install|uninstall)
            resource_cli::$command "$@"
            ;;
            
        # agent-s2-specific commands
        screenshot)
            agent_s2_screenshot "$@"
            ;;
        execute)
            agent_s2_execute "$@"
            ;;
        switch-mode)
            agent_s2_switch_mode "$@"
            ;;
        show-mode)
            agent_s2_show_mode "$@"
            ;;
        health)
            agent_s2_health "$@"
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