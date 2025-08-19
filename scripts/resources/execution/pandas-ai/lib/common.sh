#!/usr/bin/env bash
set -euo pipefail

# Get the directory of this script
PANDAS_AI_LIB_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source dependencies
source "${PANDAS_AI_LIB_DIR}/../../../../lib/utils/var.sh"
source "${PANDAS_AI_LIB_DIR}/../../../../lib/utils/format.sh"

# Constants
PANDAS_AI_NAME="pandas-ai"
PANDAS_AI_DESC="AI-powered data analysis and manipulation"
PANDAS_AI_CATEGORY="execution"
PANDAS_AI_PORT="8095"
PANDAS_AI_DATA_DIR="${var_DATA_DIR}/pandas-ai"
PANDAS_AI_VENV_DIR="${PANDAS_AI_DATA_DIR}/venv"
PANDAS_AI_SCRIPTS_DIR="${PANDAS_AI_DATA_DIR}/scripts"
PANDAS_AI_PID_FILE="${PANDAS_AI_DATA_DIR}/pandas-ai.pid"
PANDAS_AI_LOG_FILE="${PANDAS_AI_DATA_DIR}/pandas-ai.log"

# Help function
pandas_ai::help() {
    log::header "Pandas AI CLI"
    echo "Usage: resource-pandas-ai [command] [options]"
    echo ""
    echo "Commands:"
    echo "  start      - Start Pandas AI service"
    echo "  stop       - Stop Pandas AI service"
    echo "  status     - Check Pandas AI status"
    echo "  install    - Install Pandas AI and dependencies"
    echo "  inject     - Inject Python analysis scripts"
    echo "  analyze    - Run analysis on data"
    echo "  help       - Show this help message"
}

# Get port for Pandas AI
pandas_ai::get_port() {
    # Check port registry first
    if [[ -f "${var_ROOT_DIR}/scripts/resources/port_registry.sh" ]]; then
        # Source the registry and use the function directly
        source "${var_ROOT_DIR}/scripts/resources/port_registry.sh"
        local registered_port
        registered_port=$(ports::get_resource_port "pandas-ai")
        if [[ -n "${registered_port}" ]]; then
            echo "${registered_port}"
            return 0
        fi
    fi
    echo "${PANDAS_AI_PORT}"
}

# Export functions for use by CLI
export -f pandas_ai::help
export -f pandas_ai::get_port

# Don't source other files here - they will be sourced by cli.sh