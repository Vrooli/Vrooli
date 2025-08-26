#!/bin/bash

set -euo pipefail

# Get script directory (resolving symlinks for installed CLI)
if [[ -L "${BASH_SOURCE[0]}" ]]; then
    SAGEMATH_CLI_SCRIPT="$(readlink -f "${BASH_SOURCE[0]}")"
else
    SAGEMATH_CLI_SCRIPT="${BASH_SOURCE[0]}"
fi
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../.." && builtin pwd)}"
# Handle symlinks for installed CLI
if [[ -L "${BASH_SOURCE[0]}" ]]; then
    SAGEMATH_CLI_SCRIPT="$(readlink -f "${BASH_SOURCE[0]}")"
    # Recalculate APP_ROOT from resolved symlink location
    APP_ROOT="$(builtin cd "${SAGEMATH_CLI_SCRIPT%/*}/../.." && builtin pwd)"
fi
SCRIPT_DIR="${APP_ROOT}/resources/sagemath"

# Source the var utility first for consistent variable access
source "${APP_ROOT}/scripts/lib/utils/var.sh"

# Source library functions
source "$SCRIPT_DIR/lib/common.sh"
source "$SCRIPT_DIR/lib/install.sh"
source "$SCRIPT_DIR/lib/start.sh"
source "$SCRIPT_DIR/lib/stop.sh"
source "$SCRIPT_DIR/lib/status.sh"
source "$SCRIPT_DIR/lib/inject.sh"
source "$SCRIPT_DIR/lib/calculate.sh"
# CLI framework is in scripts/resources/lib
source "${APP_ROOT}/scripts/resources/lib/cli-command-framework.sh"

# Define commands for the CLI framework
declare -A COMMANDS=(
    ["install"]="sagemath_install"
    ["uninstall"]="sagemath_uninstall"
    ["start"]="sagemath_start"
    ["stop"]="sagemath_stop"
    ["restart"]="sagemath_restart"
    ["status"]="sagemath_status"
    ["inject"]="sagemath_inject"
    ["calculate"]="sagemath_calculate"
    ["run-script"]="sagemath_run_script"
    ["notebook"]="sagemath_notebook"
    ["test"]="sagemath_test"
    ["help"]="sagemath_help"
)

# Help function
sagemath_help() {
    cat << EOF
SageMath Resource CLI

Usage: $(basename "$0") [command] [options]

Commands:
  install              Install SageMath
  uninstall            Uninstall SageMath
  start                Start SageMath service
  stop                 Stop SageMath service
  restart              Restart SageMath service
  status               Show SageMath status
  inject [file]        Inject a SageMath script or notebook
  calculate [expr]     Calculate a mathematical expression
  run-script [file]    Execute a SageMath script
  notebook             Open Jupyter notebook interface
  test                 Run resource tests
  help                 Show this help message

Options:
  --format json        Output in JSON format (for status)
  --verbose           Verbose output

Examples:
  $(basename "$0") install
  $(basename "$0") status --format json
  $(basename "$0") calculate "factor(100)"
  $(basename "$0") run-script analysis.sage
  $(basename "$0") inject calculations.sage

EOF
}

# Restart function
sagemath_restart() {
    sagemath_stop
    sagemath_start
}

# Main execution
main() {
    local command="${1:-help}"
    shift || true
    
    if [[ -n "${COMMANDS[$command]:-}" ]]; then
        "${COMMANDS[$command]}" "$@"
    else
        echo "Unknown command: $command" >&2
        sagemath_help
        exit 1
    fi
}

main "$@"
