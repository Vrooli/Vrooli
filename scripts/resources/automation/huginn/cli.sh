#!/usr/bin/env bash
################################################################################
# Huginn Resource CLI
# 
# Lightweight CLI interface for Huginn that delegates to existing lib functions.
#
# Usage:
#   resource-huginn <command> [options]
#
################################################################################

set -euo pipefail

# Get script directory
HUGINN_CLI_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
VROOLI_ROOT="${VROOLI_ROOT:-$(cd "$HUGINN_CLI_DIR/../../../.." && pwd)}"
export VROOLI_ROOT
export RESOURCE_DIR="$HUGINN_CLI_DIR"
export HUGINN_SCRIPT_DIR="$HUGINN_CLI_DIR"  # For compatibility with existing libs

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

# Initialize with resource name first (before sourcing config)
resource_cli::init "huginn"

# Source huginn configuration
# shellcheck disable=SC1091
source "${HUGINN_CLI_DIR}/config/defaults.sh" 2>/dev/null || true
huginn::export_config 2>/dev/null || true

# Source huginn libraries
for lib in common docker status api install testing; do
    lib_file="${HUGINN_CLI_DIR}/lib/${lib}.sh"
    if [[ -f "$lib_file" ]]; then
        # shellcheck disable=SC1090
        source "$lib_file" 2>/dev/null || true
    fi
done

################################################################################
# Delegate to existing huginn functions
################################################################################

# Inject agents or scenarios into huginn
resource_cli::inject() {
    local file="${1:-}"
    DRY_RUN="${DRY_RUN:-false}"
    
    if [[ -z "$file" ]]; then
        log::error "File path required for injection"
        echo "Usage: resource-huginn inject <file.json>"
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
    if command -v huginn::inject_file &>/dev/null; then
        huginn::inject_file "$file"
    elif command -v huginn::import_scenario_file &>/dev/null; then
        huginn::import_scenario_file "$file"
    else
        log::error "huginn injection functions not available"
        return 1
    fi
}

# Validate huginn configuration
resource_cli::validate() {
    if command -v huginn::health_check &>/dev/null; then
        huginn::health_check
    elif command -v huginn::is_healthy &>/dev/null; then
        if huginn::is_healthy; then
            log::success "Huginn is healthy"
        else
            log::error "Huginn health check failed"
            return 1
        fi
    else
        # Basic validation
        log::header "Validating Huginn"
        if huginn::is_running; then
            log::success "Huginn is running"
        else
            log::error "Huginn is not running"
            return 1
        fi
    fi
}

# Show huginn status
resource_cli::status() {
    if command -v huginn::show_status &>/dev/null; then
        huginn::show_status
    elif command -v huginn::show_basic_status &>/dev/null; then
        huginn::show_basic_status
    else
        # Basic status
        log::header "Huginn Status"
        if huginn::is_running; then
            echo "Container: ✅ Running"
            docker ps --filter "name=huginn" --format "table {{.Status}}\t{{.Ports}}" | tail -n 1
        else
            echo "Container: ❌ Not running"
        fi
    fi
}

# Start huginn
resource_cli::start() {
    DRY_RUN="${DRY_RUN:-false}"
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log::info "[DRY RUN] Would start huginn"
        return 0
    fi
    
    if command -v huginn::start &>/dev/null; then
        huginn::start
    else
        log::error "huginn::start not available"
        return 1
    fi
}

# Stop huginn
resource_cli::stop() {
    DRY_RUN="${DRY_RUN:-false}"
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log::info "[DRY RUN] Would stop huginn"
        return 0
    fi
    
    if command -v huginn::stop &>/dev/null; then
        huginn::stop
    else
        log::error "huginn::stop not available"
        return 1
    fi
}

# Install huginn
resource_cli::install() {
    DRY_RUN="${DRY_RUN:-false}"
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log::info "[DRY RUN] Would install huginn"
        return 0
    fi
    
    if command -v huginn::install &>/dev/null; then
        huginn::install
    else
        log::error "huginn::install not available"
        return 1
    fi
}

# Uninstall huginn
resource_cli::uninstall() {
    FORCE="${FORCE:-false}"
    DRY_RUN="${DRY_RUN:-false}"
    
    if [[ "$FORCE" != "true" ]]; then
        echo "⚠️  This will remove huginn and all its data. Use --force to confirm."
        return 1
    fi
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log::info "[DRY RUN] Would uninstall huginn"
        return 0
    fi
    
    if command -v huginn::uninstall &>/dev/null; then
        huginn::uninstall
    else
        log::error "huginn::uninstall not available"
        return 1
    fi
}

################################################################################
# Huginn-specific commands (if functions exist)
################################################################################

# List agents
huginn_list_agents() {
    local format="${1:-table}"
    if command -v huginn::list_agents &>/dev/null; then
        huginn::list_agents "$format"
    else
        log::error "Agent listing not available"
        return 1
    fi
}

# Show specific agent
huginn_show_agent() {
    local agent_id="${1:-}"
    if [[ -z "$agent_id" ]]; then
        log::error "Agent ID required"
        echo "Usage: resource-huginn show-agent <agent-id>"
        return 1
    fi
    
    if command -v huginn::show_agent &>/dev/null; then
        huginn::show_agent "$agent_id"
    else
        log::error "Agent display not available"
        return 1
    fi
}

# Run specific agent
huginn_run_agent() {
    local agent_id="${1:-}"
    if [[ -z "$agent_id" ]]; then
        log::error "Agent ID required"
        echo "Usage: resource-huginn run-agent <agent-id>"
        return 1
    fi
    
    if command -v huginn::run_agent &>/dev/null; then
        huginn::run_agent "$agent_id"
    else
        log::error "Agent execution not available"
        return 1
    fi
}

# List scenarios
huginn_list_scenarios() {
    if command -v huginn::list_scenarios &>/dev/null; then
        huginn::list_scenarios
    else
        log::error "Scenario listing not available"
        return 1
    fi
}

# Show recent events
huginn_show_events() {
    local count="${1:-10}"
    if command -v huginn::show_recent_events &>/dev/null; then
        huginn::show_recent_events "$count"
    else
        log::error "Event display not available"
        return 1
    fi
}

# Show help
resource_cli::show_help() {
    cat << EOF
🤖 Huginn Resource CLI

USAGE:
    resource-huginn <command> [options]

CORE COMMANDS:
    inject <file>       Inject agents/scenarios into huginn
    validate            Validate huginn configuration
    status              Show huginn status
    start               Start huginn container
    stop                Stop huginn container
    install             Install huginn
    uninstall           Uninstall huginn (requires --force)
    
HUGINN COMMANDS:
    list-agents         List all agents
    show-agent <id>     Show specific agent details
    run-agent <id>      Run specific agent
    list-scenarios      List all scenarios
    show-events [count] Show recent events (default: 10)

OPTIONS:
    --verbose, -v       Show detailed output
    --dry-run           Preview actions without executing
    --force             Force operation (skip confirmations)

EXAMPLES:
    resource-huginn status
    resource-huginn list-agents
    resource-huginn show-agent 1
    resource-huginn run-agent 1
    resource-huginn inject shared:initialization/automation/huginn/agents.json
    resource-huginn show-events 20

For more information: https://docs.vrooli.com/resources/huginn
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
            
        # Huginn-specific commands
        list-agents)
            huginn_list_agents "$@"
            ;;
        show-agent)
            huginn_show_agent "$@"
            ;;
        run-agent)
            huginn_run_agent "$@"
            ;;
        list-scenarios)
            huginn_list_scenarios "$@"
            ;;
        show-events)
            huginn_show_events "$@"
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