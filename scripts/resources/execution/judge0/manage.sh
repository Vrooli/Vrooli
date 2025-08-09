#!/usr/bin/env bash
set -euo pipefail

# Judge0 Code Execution Resource Management
# This script orchestrates Judge0 installation, configuration, and management

DESCRIPTION="Install and manage Judge0 secure code execution service"

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

# Source var.sh first to get directory variables
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/../../../lib/utils/var.sh"

# Handle Ctrl+C gracefully
trap 'echo ""; log::info "Judge0 operation interrupted by user. Exiting..."; exit 130' INT TERM

# Source common resources using var_ variables
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_DIR}/common.sh"
# shellcheck disable=SC1091
source "${var_LIB_UTILS_DIR}/args-cli.sh"

# Source configuration modules
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/config/defaults.sh"
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/config/messages.sh"

# Export configuration and messages
judge0::export_config
judge0::export_messages

# Source all library modules
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/lib/common.sh"
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/lib/install.sh"
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/lib/docker.sh"
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/lib/status.sh"
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/lib/api.sh"
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/lib/languages.sh"
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/lib/security.sh"
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/lib/usage.sh"

#######################################
# Parse command line arguments
#######################################
judge0::parse_arguments() {
    args::reset
    
    args::register_help
    args::register_yes
    
    args::register \
        --name "action" \
        --flag "a" \
        --desc "Action to perform" \
        --type "value" \
        --options "install|uninstall|start|stop|restart|status|logs|info|test|languages|usage|submit" \
        --default "install"
    
    args::register \
        --name "force" \
        --flag "f" \
        --desc "Force action even if Judge0 appears to be already installed/running" \
        --type "value" \
        --options "yes|no" \
        --default "no"
    
    args::register \
        --name "workers" \
        --desc "Number of worker containers to run" \
        --type "value" \
        --default "${JUDGE0_WORKERS_COUNT}"
    
    args::register \
        --name "cpu-limit" \
        --desc "CPU time limit per submission (seconds)" \
        --type "value" \
        --default "${JUDGE0_CPU_TIME_LIMIT}"
    
    args::register \
        --name "memory-limit" \
        --desc "Memory limit per submission (MB)" \
        --type "value" \
        --default "$((JUDGE0_MEMORY_LIMIT / 1024))"
    
    args::register \
        --name "api-key" \
        --desc "API key for authentication (auto-generated if not provided)" \
        --type "value" \
        --default ""
    
    args::register \
        --name "code" \
        --desc "Code to execute (for submit action)" \
        --type "value" \
        --default ""
    
    args::register \
        --name "language" \
        --desc "Programming language (for submit action)" \
        --type "value" \
        --default "javascript"
    
    args::register \
        --name "stdin" \
        --desc "Standard input for code execution" \
        --type "value" \
        --default ""
    
    args::register \
        --name "expected-output" \
        --desc "Expected output for validation" \
        --type "value" \
        --default ""
    
    args::parse "$@"
}

#######################################
# Main orchestration function
#######################################
judge0::main() {
    # Parse command line arguments
    judge0::parse_arguments "$@"
    
    # Get parsed values
    local action=$(args::get "action")
    local force=$(args::get "force")
    local workers=$(args::get "workers")
    local cpu_limit=$(args::get "cpu-limit")
    local memory_limit=$(args::get "memory-limit")
    local api_key=$(args::get "api-key")
    local code=$(args::get "code")
    local language=$(args::get "language")
    local stdin=$(args::get "stdin")
    local expected_output=$(args::get "expected-output")
    
    # Handle help
    if args::is_enabled "help"; then
        judge0::usage::show
        return 0
    fi
    
    # Export runtime configuration
    export JUDGE0_WORKERS_COUNT="$workers"
    export JUDGE0_CPU_TIME_LIMIT="$cpu_limit"
    export JUDGE0_MEMORY_LIMIT="$((memory_limit * 1024))"
    
    if [[ -n "$api_key" ]]; then
        export JUDGE0_API_KEY="$api_key"
    fi
    
    # Route to appropriate action
    case "$action" in
        install)
            judge0::install::main "$force"
            ;;
        uninstall)
            judge0::docker::uninstall "$force"
            ;;
        start)
            judge0::docker::start
            ;;
        stop)
            judge0::docker::stop
            ;;
        restart)
            judge0::docker::restart
            ;;
        status)
            judge0::status::show
            ;;
        logs)
            judge0::docker::logs
            ;;
        info)
            judge0::api::system_info
            ;;
        test)
            judge0::api::test
            ;;
        languages)
            judge0::languages::list
            ;;
        usage)
            judge0::usage::show
            ;;
        submit)
            if [[ -z "$code" ]]; then
                log::error "Code is required for submit action"
                return 1
            fi
            judge0::api::submit "$code" "$language" "$stdin" "$expected_output"
            ;;
        monitor)
            log::info "Starting Judge0 security monitoring..."
            log::warn "WARNING: Running with elevated privileges - monitor actively for security alerts"
            "${SCRIPT_DIR}/lib/security-monitor.sh"
            ;;
        *)
            log::error "Unknown action: $action"
            judge0::usage::show
            return 1
            ;;
    esac
}

# Run main function if script is executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    judge0::main "$@"
fi