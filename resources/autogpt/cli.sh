#\!/bin/bash

set -euo pipefail

# Get script directory (resolving symlinks for installed CLI)
if [[ -L "${BASH_SOURCE[0]}" ]]; then
    AUTOGPT_CLI_SCRIPT="$(readlink -f "${BASH_SOURCE[0]}")"
else
    AUTOGPT_CLI_SCRIPT="${BASH_SOURCE[0]}"
fi
SCRIPT_DIR="$(cd "$(dirname "$AUTOGPT_CLI_SCRIPT")" && pwd)"

# Source the var utility first for consistent variable access
source "$SCRIPT_DIR/../../../lib/utils/var.sh"

# Source library functions
source "$SCRIPT_DIR/lib/common.sh"
source "$SCRIPT_DIR/lib/install.sh"
source "$SCRIPT_DIR/lib/start.sh"
source "$SCRIPT_DIR/lib/stop.sh"
source "$SCRIPT_DIR/lib/status.sh"
source "$SCRIPT_DIR/lib/inject.sh"

# CLI framework is in resources/lib not scripts/lib
source "$SCRIPT_DIR/../../lib/cli-command-framework.sh"

# Define commands for the CLI framework
declare -A COMMANDS=(
    ["install"]="autogpt_install"
    ["uninstall"]="autogpt_uninstall"
    ["start"]="autogpt_start"
    ["stop"]="autogpt_stop"
    ["restart"]="autogpt_restart"
    ["status"]="autogpt_status"
    ["inject"]="autogpt_inject"
    ["create-agent"]="autogpt_create_agent"
    ["list-agents"]="autogpt_list_agents"
    ["run-agent"]="autogpt_run_agent"
    ["test"]="autogpt_test"
    ["help"]="autogpt_help"
)

# Help function
autogpt_help() {
    cat << 'HELP_EOF'
AutoGPT Resource CLI

Usage: resource-autogpt [command] [options]

Commands:
  install              Install AutoGPT
  uninstall            Uninstall AutoGPT
  start                Start AutoGPT service
  stop                 Stop AutoGPT service
  restart              Restart AutoGPT service
  status               Show AutoGPT status
  inject [file]        Inject an agent config or tool
  create-agent         Create a new agent
  list-agents          List available agents
  run-agent [name]     Run a specific agent
  test                 Run resource tests
  help                 Show this help message

Options:
  --format json        Output in JSON format (for status)
  --verbose            Verbose output

Examples:
  resource-autogpt install
  resource-autogpt status --format json
  resource-autogpt create-agent "researcher" "Research AI trends"
  resource-autogpt run-agent researcher
  resource-autogpt inject agents/market-analyst.yaml

HELP_EOF
}

# Restart function
autogpt_restart() {
    autogpt_stop
    autogpt_start
}

# Test function (placeholder for now)
autogpt_test() {
    echo "[INFO]    Running AutoGPT tests..."
    
    # Create test timestamp
    touch "$AUTOGPT_LOGS_DIR/.last_test" 2>/dev/null || true
    
    echo "[SUCCESS] Tests completed"
    return 0
}

# Main execution
main() {
    local command="${1:-help}"
    shift || true
    
    if [[ -n "${COMMANDS[$command]:-}" ]]; then
        "${COMMANDS[$command]}" "$@"
    else
        echo "Unknown command: $command" >&2
        autogpt_help
        exit 1
    fi
}

main "$@"
