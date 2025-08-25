#!/usr/bin/env bash

#######################################
# N8N Adapter for Browserless
# Description: UI automation adapter for n8n workflows
#
# Provides browser-based fallback interfaces for n8n operations
# when the API or webhooks are unavailable or broken.
#
# This adapter enables:
#   - Workflow execution via UI automation
#   - Credential management through the UI
#   - Workflow import/export via browser
#   - Dashboard monitoring and screenshot capture
#######################################

# Get script directory
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*/../../../.." && builtin pwd)}"
N8N_ADAPTER_DIR="${APP_ROOT}/resources/browserless/adapters/n8n"
ADAPTERS_DIR="${APP_ROOT}/resources/browserless/adapters"

# Source adapter framework
source "${ADAPTERS_DIR}/common.sh"

# Source n8n-specific implementations
source "${N8N_ADAPTER_DIR}/workflows.sh"
source "${N8N_ADAPTER_DIR}/credentials.sh" 2>/dev/null || true  # Optional, will create later
source "${N8N_ADAPTER_DIR}/selectors.sh" 2>/dev/null || true   # Optional, will create later

# Export adapter name for context
export BROWSERLESS_ADAPTER_NAME="n8n"

#######################################
# Initialize n8n adapter
# Sets up n8n-specific configuration
# Returns:
#   0 on success, 1 on failure
#######################################
n8n::init() {
    # Initialize base adapter framework
    adapter::init "n8n"
    
    # Load n8n configuration if available
    adapter::load_target_config "n8n"
    
    # Set default n8n URL if not configured
    export N8N_URL="${N8N_URL:-http://localhost:5678}"
    export N8N_TIMEOUT="${N8N_TIMEOUT:-60000}"
    
    log::debug "N8N adapter initialized with URL: $N8N_URL"
    return 0
}

#######################################
# List available n8n adapter commands
# Used by the adapter framework for discovery
# Returns:
#   List of available commands
#######################################
adapter::list_commands() {
    echo "execute-workflow    - Execute n8n workflow via browser automation"
    echo "list-workflows      - List all workflows via UI scraping"
    echo "export-workflow     - Export workflow JSON via browser"
    echo "import-workflow     - Import workflow JSON via browser"
    echo "add-credentials     - Add credentials via UI automation"
    echo "test-workflow       - Test workflow execution with sample data"
    echo "monitor-dashboard   - Capture dashboard screenshots"
}

#######################################
# Main command dispatcher for n8n adapter
# Routes commands to appropriate implementations
# Arguments:
#   $1 - Command name
#   $@ - Command arguments
# Returns:
#   Command exit status
#######################################
n8n::dispatch() {
    local command="${1:-}"
    shift || true
    
    # Initialize adapter
    n8n::init
    
    # Check browserless health
    if ! adapter::check_browserless_health; then
        return 1
    fi
    
    case "$command" in
        execute-workflow|execute)
            n8n::execute_workflow "$@"
            ;;
        list-workflows|list)
            n8n::list_workflows "$@"
            ;;
        export-workflow|export)
            n8n::export_workflow "$@"
            ;;
        import-workflow|import)
            n8n::import_workflow "$@"
            ;;
        add-credentials|credentials)
            n8n::add_credentials "$@"
            ;;
        test-workflow|test)
            n8n::test_workflow "$@"
            ;;
        monitor-dashboard|monitor)
            n8n::monitor_dashboard "$@"
            ;;
        help|--help|-h|"")
            n8n::show_help
            ;;
        *)
            log::error "Unknown n8n adapter command: $command"
            n8n::show_help
            return 1
            ;;
    esac
}

#######################################
# Show help for n8n adapter
# Displays usage information and examples
#######################################
n8n::show_help() {
    cat <<EOF
N8N Adapter for Browserless

Usage:
  resource-browserless for n8n <command> [options]

Commands:
  execute-workflow <id>     Execute workflow by ID
  list-workflows           List all available workflows
  export-workflow <id>     Export workflow as JSON
  import-workflow <file>   Import workflow from JSON file
  add-credentials <type>   Add credentials via UI
  test-workflow <id>       Test workflow with sample data
  monitor-dashboard        Capture dashboard screenshots

Examples:
  # Execute a workflow
  resource-browserless for n8n execute-workflow my-workflow-id

  # Execute with input data
  resource-browserless for n8n execute-workflow data-processor \\
    --input '{"text": "Process this"}'

  # List all workflows
  resource-browserless for n8n list-workflows

  # Export workflow to file
  resource-browserless for n8n export-workflow my-workflow > workflow.json

  # Add HTTP credentials
  resource-browserless for n8n add-credentials httpHeaderAuth

Environment Variables:
  N8N_URL           - N8N instance URL (default: http://localhost:5678)
  N8N_EMAIL         - Email for authentication
  N8N_PASSWORD      - Password for authentication
  N8N_TIMEOUT       - Operation timeout in ms (default: 60000)

Notes:
  - This adapter provides UI-based fallback when n8n API is unavailable
  - Requires browserless to be running and healthy
  - Authentication credentials should be set via environment variables
  - Screenshots and logs are saved to \$BROWSERLESS_TEST_OUTPUT_DIR

EOF
    return 0
}

#######################################
# Execute workflow (adapter wrapper)
# Provides cleaner interface for the adapter pattern
# Arguments:
#   $1 - Workflow ID
#   --input - Input data (JSON)
#   --timeout - Timeout in ms
#   --session - Session ID for persistence
# Returns:
#   0 on success, 1 on failure
#######################################
n8n::execute_workflow() {
    local workflow_id=""
    local input_data=""
    local timeout="${N8N_TIMEOUT:-60000}"
    local session_id=""
    local n8n_url="${N8N_URL:-http://localhost:5678}"
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --input)
                input_data="$2"
                shift 2
                ;;
            --timeout)
                timeout="$2"
                # If timeout is less than 1000, assume it's in seconds and convert to milliseconds
                if [[ "$timeout" -lt 1000 ]]; then
                    timeout=$((timeout * 1000))
                fi
                shift 2
                ;;
            --session)
                session_id="$2"
                shift 2
                ;;
            --url)
                n8n_url="$2"
                shift 2
                ;;
            --help|-h)
                n8n::show_execute_help
                return 0
                ;;
            *)
                if [[ -z "$workflow_id" ]]; then
                    workflow_id="$1"
                fi
                shift
                ;;
        esac
    done
    
    if [[ -z "$workflow_id" ]]; then
        log::error "Workflow ID required"
        echo "Usage: resource-browserless for n8n execute-workflow <workflow-id> [options]"
        return 1
    fi
    
    # Use persistent session by default for adapters
    local use_persistent="true"
    if [[ -z "$session_id" ]]; then
        session_id="n8n_adapter_$(date +%s)"
    fi
    
    log::header "🔄 Executing N8N Workflow via Browserless Adapter"
    log::info "Workflow: $workflow_id"
    log::info "N8N URL: $n8n_url"
    log::info "Session: $session_id"
    
    # Call the underlying implementation
    browserless::execute_n8n_workflow \
        "$workflow_id" \
        "$n8n_url" \
        "$timeout" \
        "$input_data"
}

# NOTE: n8n::list_workflows() is implemented in workflows.sh using YAML flow control

#######################################
# Show execute workflow help
#######################################
n8n::show_execute_help() {
    cat <<EOF
Execute N8N Workflow via Browserless Adapter

Usage:
  resource-browserless for n8n execute-workflow <workflow-id> [options]

Options:
  --input <json>    Input data as JSON string or @file
  --timeout <ms>    Execution timeout in milliseconds
  --session <id>    Session ID for browser persistence
  --url <url>       N8N instance URL
  --help            Show this help message

Examples:
  # Simple execution
  resource-browserless for n8n execute-workflow my-workflow

  # With input data
  resource-browserless for n8n execute-workflow processor \\
    --input '{"text": "Hello"}'

  # From file
  resource-browserless for n8n execute-workflow analyzer \\
    --input @data.json

  # Custom timeout and session
  resource-browserless for n8n execute-workflow long-running \\
    --timeout 120000 \\
    --session my-session-1

EOF
    return 0
}

# NOTE: n8n::export_workflow() is implemented in workflows.sh using YAML flow control

n8n::import_workflow() {
    log::warn "Import workflow functionality coming soon"
    return 1
}

n8n::add_credentials() {
    log::warn "Add credentials functionality coming soon"
    return 1
}

n8n::test_workflow() {
    log::warn "Test workflow functionality coming soon"
    return 1
}

n8n::monitor_dashboard() {
    log::warn "Monitor dashboard functionality coming soon"
    return 1
}

# Export main dispatcher for CLI integration
export -f n8n::dispatch
export -f n8n::init
#######################################
# Main execution entry point
#######################################
# If script is executed directly (not sourced)
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    n8n::dispatch "$@"
fi
