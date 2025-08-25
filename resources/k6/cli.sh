#!/usr/bin/env bash
################################################################################
# K6 Resource CLI
# 
# Modern load testing tool with JavaScript scripting
#
# Usage:
#   resource-k6 <command> [options]
#
################################################################################

set -euo pipefail

# Get script directory (resolving symlinks for installed CLI)
if [[ -L "${BASH_SOURCE[0]}" ]]; then
    K6_CLI_SCRIPT="$(readlink -f "${BASH_SOURCE[0]}")"
else
    K6_CLI_SCRIPT="${BASH_SOURCE[0]}"
fi
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../.." && builtin pwd)}"
K6_CLI_DIR="${APP_ROOT}/resources/k6"

# Source standard variables
# shellcheck disable=SC1091
source "${K6_CLI_DIR}/../../../lib/utils/var.sh"

# Source utilities using var_ variables
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "${var_RESOURCES_COMMON_FILE}"

# Source the CLI Command Framework
# shellcheck disable=SC1091
source "${K6_CLI_DIR}/../../lib/cli-command-framework.sh"

# Source K6 configuration
# shellcheck disable=SC1091
source "${K6_CLI_DIR}/config/defaults.sh" 2>/dev/null || true

# Source K6 libraries
for lib in core docker install status inject content test; do
    lib_file="${K6_CLI_DIR}/lib/${lib}.sh"
    if [[ -f "$lib_file" ]]; then
        # shellcheck disable=SC1090
        source "$lib_file" 2>/dev/null || true
    fi
done

# Initialize CLI framework
cli::init "k6" "Modern load testing tool with JavaScript scripting"

# Status wrapper - directly call the status function with all arguments
k6_status() {
    # Pass all arguments directly to the status function
    k6::status "$@"
}

# Register commands
cli::register_command "status" "Show service status" "k6_status"
cli::register_command "install" "Install K6" "k6::install::execute" "modifies-system"
cli::register_command "start" "Start K6 service" "k6::docker::start" "modifies-system"
cli::register_command "stop" "Stop K6 service" "k6::docker::stop" "modifies-system"
cli::register_command "restart" "Restart K6 service" "k6::docker::restart" "modifies-system"
cli::register_command "logs" "Show K6 logs" "k6::docker::logs"
cli::register_command "inject" "Inject test scripts into K6 (deprecated - use content add)" "k6::inject::execute" "modifies-system"
cli::register_command "content" "Manage K6 content (scripts, data, results)" "k6::content" "modifies-system"
cli::register_command "run-test" "Run a K6 test script" "k6::test::run" "modifies-system"
cli::register_command "list-tests" "List available test scripts" "k6::test::list"
cli::register_command "validate" "Validate installation" "k6::install::validate"
cli::register_command "uninstall" "Uninstall K6" "k6::install::uninstall" "modifies-system"
cli::register_command "credentials" "Show K6 credentials for integration" "k6::core::credentials"
cli::register_command "help" "Show this help message with examples" "k6::help::show"

# Define help function
k6::help::show() {
    cat << 'EOF'
⚡ Examples:

  # Content management (NEW - preferred way)
  resource-k6 content add --file test-script.js
  resource-k6 content add --file data.csv --type data
  resource-k6 content list
  resource-k6 content get --name test-script.js
  resource-k6 content execute --name test-script.js --options '--vus 10 --duration 30s'
  resource-k6 content remove --name old-test.js

  # Legacy test management (still supported)
  resource-k6 inject test-script.js
  resource-k6 inject shared:init/k6/performance-test.js
  resource-k6 run-test my-test.js --vus 10 --duration 30s
  resource-k6 list-tests

  # Performance testing
  resource-k6 run-test api-load.js --vus 100 --duration 5m
  resource-k6 run-test stress-test.js --stages '10s:10,1m:50,10s:0'

  # Management
  resource-k6 status
  resource-k6 logs 100
  resource-k6 credentials

  # Dangerous operations
  resource-k6 uninstall --force

Default Port: 6565 (Grafana Cloud integration)
Script Language: JavaScript/ES6
Features: Load testing, performance monitoring, CI/CD integration
EOF
}

# Dispatch the command
cli::dispatch "$@"
