#!/usr/bin/env bash
################################################################################
# Browserless Resource CLI - v2.0 Universal Contract Compliant
# 
# Headless Chrome automation service for browser testing, screenshots, PDF generation,
# and automated web interactions. Critical infrastructure for UX testing and debugging.
#
# Usage:
#   resource-browserless <command> [options]
#   resource-browserless <group> <subcommand> [options]
#
################################################################################

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../.." && builtin pwd)}"
# Handle symlinks for installed CLI
if [[ -L "${BASH_SOURCE[0]}" ]]; then
    BROWSERLESS_CLI_SCRIPT="$(readlink -f "${BASH_SOURCE[0]}")"
    APP_ROOT="$(builtin cd "${BROWSERLESS_CLI_SCRIPT%/*}/../.." && builtin pwd)"
fi
BROWSERLESS_CLI_DIR="${APP_ROOT}/resources/browserless"

# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "${var_RESOURCES_COMMON_FILE}"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/resources/lib/cli-command-framework-v2.sh"
# shellcheck disable=SC1091
source "${BROWSERLESS_CLI_DIR}/config/defaults.sh"

# Export configuration for subprocesses
browserless::export_config

# Source browserless libraries
for lib in common core docker install start stop status uninstall test health actions api usage inject session-manager pool-manager benchmarks; do
    lib_file="${BROWSERLESS_CLI_DIR}/lib/${lib}.sh"
    if [[ -f "$lib_file" ]]; then
        # shellcheck disable=SC1090
        source "$lib_file" 2>/dev/null || true
    fi
done

# Initialize CLI framework in v2.0 mode (auto-creates manage/test/content groups)
cli::init "browserless" "Browserless headless Chrome automation service" "v2"

# ==============================================================================
# REQUIRED HANDLERS - These MUST be mapped for v2.0 compliance
# ==============================================================================
CLI_COMMAND_HANDLERS["manage::install"]="install_browserless"
CLI_COMMAND_HANDLERS["manage::uninstall"]="uninstall_browserless"
CLI_COMMAND_HANDLERS["manage::start"]="start_browserless"
CLI_COMMAND_HANDLERS["manage::stop"]="stop_browserless"
CLI_COMMAND_HANDLERS["manage::restart"]="browserless::docker::restart"
CLI_COMMAND_HANDLERS["test::smoke"]="browserless::test::smoke"
CLI_COMMAND_HANDLERS["test::integration"]="browserless::test::integration"
CLI_COMMAND_HANDLERS["test::unit"]="browserless::test::unit"
CLI_COMMAND_HANDLERS["test::all"]="browserless::test::all"

# Content handlers - Browserless business functionality (browser automation)
CLI_COMMAND_HANDLERS["content::add"]="browserless::content::add"
CLI_COMMAND_HANDLERS["content::list"]="browserless::content::list"
CLI_COMMAND_HANDLERS["content::get"]="browserless::content::get"
CLI_COMMAND_HANDLERS["content::remove"]="browserless::content::remove"
CLI_COMMAND_HANDLERS["content::execute"]="browserless::content::execute"

# ==============================================================================
# REQUIRED INFORMATION COMMANDS
# ==============================================================================
cli::register_command "status" "Show detailed browserless status" "status"
cli::register_command "logs" "Show browserless logs" "browserless::logs"
cli::register_command "credentials" "Display integration credentials" "browserless::credentials"

# ==============================================================================
# BROWSERLESS-SPECIFIC COMMANDS - Critical browser automation features
# ==============================================================================

# Core atomic browser actions dispatcher (preserves all original functionality)
cli::register_command "screenshot" "Take screenshots of URLs" "browserless::screenshot"
cli::register_command "navigate" "Navigate to URLs and get basic info" "browserless::navigate"
cli::register_command "health-check" "Check if URLs load successfully" "browserless::health_check"
cli::register_command "element-exists" "Check if elements exist on pages" "browserless::element_exists"
cli::register_command "extract-text" "Extract text content from elements" "browserless::extract_text"
cli::register_command "extract" "Extract structured data with custom scripts" "browserless::extract"
cli::register_command "extract-forms" "Extract form data and input fields" "browserless::extract_forms"
cli::register_command "interact" "Perform form fills, clicks, and interactions" "browserless::interact"
cli::register_command "console" "Capture console logs from pages" "browserless::console"
cli::register_command "performance" "Measure page performance metrics" "browserless::performance"

# N8N adapter system (critical for workflow integration)
cli::register_command "for" "Use adapters for other resources (e.g., for n8n execute <id>)" "browserless::adapter_dispatch"

# Browser automation content subcommands
cli::register_subcommand "content" "api" "Test all browserless APIs" "browserless::test_all_apis"
cli::register_subcommand "content" "session" "Manage persistent browser sessions" "session::list"
cli::register_subcommand "content" "inject" "Inject scripts or functions" "browserless::inject"

# Pool management commands
cli::register_command "pool" "Manage browser pool (auto-scaling)" "browserless::pool"
cli::register_subcommand "pool" "start" "Start auto-scaler" "pool::start_autoscaler"
cli::register_subcommand "pool" "stop" "Stop auto-scaler" "pool::stop_autoscaler"
cli::register_subcommand "pool" "status" "Show pool statistics" "pool::show_stats"
cli::register_subcommand "pool" "metrics" "Get pool metrics" "pool::get_metrics"

# Performance benchmark commands
cli::register_command "benchmark" "Run performance benchmarks" "benchmark::run_all"
cli::register_subcommand "benchmark" "navigation" "Benchmark navigation operations" "benchmark::navigation"
cli::register_subcommand "benchmark" "screenshots" "Benchmark screenshot operations" "benchmark::screenshots"
cli::register_subcommand "benchmark" "extraction" "Benchmark extraction operations" "benchmark::extraction"
cli::register_subcommand "benchmark" "workflows" "Benchmark workflow operations" "benchmark::workflows"
cli::register_subcommand "benchmark" "summary" "Show benchmark summary" "browserless::benchmark_summary"
cli::register_subcommand "benchmark" "compare" "Compare two benchmark runs" "browserless::benchmark_compare"

# Test subcommands for browserless health/connectivity testing
cli::register_subcommand "test" "api" "Test all browserless APIs" "browserless::test_all_apis"
cli::register_subcommand "test" "functional" "Test browser functionality" "browserless::check_functional_health"

# Dispatcher functions to preserve all original browserless functionality
# Create specific handlers for each action
browserless::screenshot() { actions::dispatch "screenshot" "$@"; }
browserless::navigate() { actions::dispatch "navigate" "$@"; }
browserless::health_check() { actions::dispatch "health-check" "$@"; }
browserless::element_exists() { actions::dispatch "element-exists" "$@"; }
browserless::extract_text() { actions::dispatch "extract-text" "$@"; }
browserless::extract() { actions::dispatch "extract" "$@"; }
browserless::extract_forms() { actions::dispatch "extract-forms" "$@"; }
browserless::interact() { actions::dispatch "interact" "$@"; }
browserless::console() { actions::dispatch "console" "$@"; }
browserless::performance() { actions::dispatch "performance" "$@"; }

browserless::adapter_dispatch() {
    source "$BROWSERLESS_CLI_DIR/adapters/common.sh"
    local adapter_name="${1:-}"
    [[ -z "$adapter_name" ]] && { echo "Usage: resource-browserless for <adapter> <command> [args]"; adapter::list; exit 1; }
    if adapter::load "$adapter_name"; then
        local dispatch_function="${adapter_name}::dispatch"
        if ! declare -f "$dispatch_function" >/dev/null 2>&1; then
            echo "Error: Adapter dispatch function not found: $dispatch_function" >&2
            echo "This indicates an adapter loading failure. Check adapter implementation." >&2
            exit 1
        fi
        
        if ! "$dispatch_function" "${@:2}"; then
            echo "Error: Adapter command failed: $dispatch_function ${*:2}" >&2
            echo "Check adapter logs above for details. The function exists but execution failed." >&2
            exit 1
        fi
    else
        exit 1
    fi
}

# Pool management dispatcher
browserless::pool() {
    # If no subcommand, show pool status
    if [[ $# -eq 0 ]]; then
        pool::show_stats
    else
        echo "Usage: resource-browserless pool [start|stop|status|metrics]"
        exit 1
    fi
}

# Benchmark dispatchers
browserless::benchmark_summary() {
    local file="${1:-}"
    if [[ -z "$file" ]]; then
        # Find most recent benchmark
        file=$(ls -t "${BROWSERLESS_DATA_DIR}/benchmarks"/benchmark_*.json 2>/dev/null | head -1)
        if [[ -z "$file" ]]; then
            echo "No benchmark files found. Run benchmarks first."
            exit 1
        fi
    fi
    benchmark::show_summary "$file"
}

browserless::benchmark_compare() {
    local file1="${1:-}"
    local file2="${2:-}"
    if [[ -z "$file1" ]] || [[ -z "$file2" ]]; then
        echo "Usage: resource-browserless benchmark compare <file1> <file2>"
        exit 1
    fi
    benchmark::compare "$file1" "$file2"
}

# Minimal content handler functions  
browserless::content::add() { echo "Add browser automation script or workflow"; }
browserless::content::list() { session::list "$@"; }
browserless::content::get() { echo "Get browser session or workflow details"; }
browserless::content::remove() { echo "Remove browser session or workflow"; }
browserless::content::execute() { actions::dispatch "$@"; }

# Only execute if script is run directly (not sourced)
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    cli::dispatch "$@"
fi