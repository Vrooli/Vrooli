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

# Determine the script directory
if [[ -L "${BASH_SOURCE[0]}" ]]; then
    # If this script is a symlink, resolve it
    BROWSERLESS_CLI_SCRIPT="$(readlink -f "${BASH_SOURCE[0]}")"
    SCRIPT_DIR="$(dirname "${BROWSERLESS_CLI_SCRIPT}")"
else
    # Get the directory of the script
    SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
fi

# Set APP_ROOT relative to the script
APP_ROOT="${APP_ROOT:-$(cd "${SCRIPT_DIR}/../.." && pwd)}"
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
for lib in common core docker install start stop status uninstall test health actions api usage inject session-manager pool-manager benchmarks cache; do
    lib_file="${BROWSERLESS_CLI_DIR}/lib/${lib}.sh"
    if [[ -f "$lib_file" ]]; then
        # shellcheck disable=SC1090
        source "$lib_file" 2>/dev/null || true
    fi
done

# Provide shared workflow helpers (metadata parsing, interpreter utilities)
if [[ -f "${BROWSERLESS_CLI_DIR}/lib/workflow/interpreter.sh" ]]; then
    # shellcheck disable=SC1090
    source "${BROWSERLESS_CLI_DIR}/lib/workflow/interpreter.sh" 2>/dev/null || true
fi

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
cli::register_command "network" "Capture network requests from pages" "browserless::network"
cli::register_command "performance" "Measure page performance metrics" "browserless::performance"
cli::register_command "diagnostics" "Collect all diagnostics in one browser session (console, network, performance, HTML)" "browserless::diagnostics"

# N8N adapter system (critical for workflow integration)
cli::register_command "for" "Use adapters for other resources (e.g., for n8n execute <id>)" "browserless::adapter_dispatch"
cli::register_command "workflow" "Inspect and catalog browserless workflows" "browserless::workflow_command"

# Browser automation content subcommands
cli::register_subcommand "content" "api" "Test all browserless APIs" "browserless::test_all_apis"
cli::register_subcommand "content" "session" "Manage persistent browser sessions" "session::list"
cli::register_subcommand "content" "inject" "Inject scripts or functions" "browserless::inject"

# Pool management commands - properly handle default behavior
cli::register_command "pool" "Manage browser pool (auto-scaling)" "browserless::pool_dispatch"
cli::register_subcommand "pool" "start" "Start auto-scaler" "browserless::pool_start"
cli::register_subcommand "pool" "stop" "Stop auto-scaler" "browserless::pool_stop"
cli::register_subcommand "pool" "stats" "Show pool statistics" "pool::show_stats"
cli::register_subcommand "pool" "metrics" "Get pool metrics" "browserless::pool_metrics"
cli::register_subcommand "pool" "prewarm" "Pre-warm browser instances" "browserless::pool_prewarm"
cli::register_subcommand "pool" "smart-prewarm" "Intelligently pre-warm if idle" "browserless::pool_smart_prewarm"
cli::register_subcommand "pool" "recover" "Check and recover unhealthy pool" "browserless::pool_recover"
cli::register_subcommand "pool" "logs" "View auto-scaler logs" "browserless::pool_logs"

# Cache management commands  
browserless::cache() { cache::stats; }  # Default to showing stats
cli::register_command "cache" "Manage workflow result cache" "browserless::cache"
cli::register_subcommand "cache" "stats" "Show cache statistics" "cache::stats"
cli::register_subcommand "cache" "clear" "Clear all cached results" "cache::clear"
cli::register_subcommand "cache" "cleanup" "Remove expired cache entries" "cache::cleanup_expired"

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
browserless::diagnostics() { actions::dispatch "diagnostics" "$@"; }

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

# Pool management dispatcher - handles both standalone and subcommand usage
browserless::pool_dispatch() {
    # When called without arguments or with 'stats' subcommand, show pool stats
    # This ensures 'pool' and 'pool stats' both work correctly
    if [[ $# -eq 0 ]] || [[ "${1:-}" == "stats" ]]; then
        pool::show_stats
        return $?
    fi
    
    # For other subcommands, the CLI framework will handle dispatch
    # This should not normally be reached as framework handles subcommands
    log::debug "Pool dispatch called with: $*"
    return 0
}

# Pool wrapper functions to ensure output is displayed
browserless::pool_metrics() {
    local metrics
    metrics=$(pool::get_metrics) || {
        echo "❌ Failed to get pool metrics"
        return 1
    }
    echo "$metrics" | jq '.'
}

browserless::pool_start() {
    # Capture and display any output from the function
    local output
    output=$(pool::start_autoscaler 2>&1)
    local exit_code=$?
    
    if [[ -n "$output" ]]; then
        echo "$output"
    fi
    
    if [[ $exit_code -eq 0 ]]; then
        echo "✅ Auto-scaler started successfully"
        # Show current status
        pool::show_stats
    else
        echo "❌ Failed to start auto-scaler"
    fi
    return $exit_code
}

browserless::pool_stop() {
    # Capture and display any output from the function
    local output
    output=$(pool::stop_autoscaler 2>&1)
    local exit_code=$?
    
    if [[ -n "$output" ]]; then
        echo "$output"
    fi
    
    if [[ $exit_code -eq 0 ]]; then
        echo "✅ Auto-scaler stopped successfully"
    else
        echo "❌ Failed to stop auto-scaler"
    fi
    return $exit_code
}

browserless::pool_prewarm() {
    local count="${1:-2}"
    
    # Capture and display any output
    local output
    output=$(pool::prewarm "$count" 2>&1)
    local exit_code=$?
    
    if [[ -n "$output" ]]; then
        echo "$output"
    fi
    
    if [[ $exit_code -eq 0 ]]; then
        echo "✅ Successfully pre-warmed $count browser instances"
        # Show updated pool stats
        pool::show_stats
    else
        echo "❌ Failed to pre-warm browser instances"
    fi
    return $exit_code
}

browserless::pool_smart_prewarm() {
    # Capture and display any output
    local output
    output=$(pool::smart_prewarm 2>&1)
    local exit_code=$?
    
    if [[ -n "$output" ]]; then
        echo "$output"
    fi
    
    if [[ $exit_code -eq 0 ]]; then
        echo "✅ Smart pre-warm completed"
        # Show updated pool stats
        pool::show_stats
    else
        echo "❌ Smart pre-warm failed"
    fi
    return $exit_code
}

browserless::pool_recover() {
    # Capture and display any output
    local output
    output=$(pool::health_check_and_recover 2>&1)
    local exit_code=$?
    
    if [[ -n "$output" ]]; then
        echo "$output"
    fi
    
    if [[ $exit_code -eq 0 ]]; then
        echo "✅ Pool recovery check completed"
        # Show current pool health
        pool::show_stats
    else
        echo "❌ Pool recovery check failed"
    fi
    return $exit_code
}

browserless::pool_logs() {
    local log_file="${BROWSERLESS_DATA_DIR}/autoscaler.log"
    local lines="${1:-50}"  # Default to last 50 lines
    
    if [[ ! -f "$log_file" ]]; then
        echo "📋 No autoscaler logs found"
        echo "💡 Start the autoscaler first: vrooli resource browserless pool start"
        return 1
    fi
    
    echo "📋 Autoscaler Logs (last $lines lines):"
    echo "────────────────────────────"
    tail -n "$lines" "$log_file"
    
    # Show log file location for reference
    echo ""
    echo "💡 Full log file: $log_file"
    echo "💡 Watch logs live: tail -f $log_file"
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

# Workflow metadata helpers
browserless::workflow_command() {
    local subcmd="${1:-}"
    shift || true

    case "$subcmd" in
        describe)
            browserless::workflow_describe "$@"
            ;;
        catalog)
            browserless::workflow_catalog "$@"
            ;;
        "")
            echo "Usage: resource-browserless workflow <subcommand> [options]" >&2
            echo "Subcommands:" >&2
            echo "  describe <workflow.yaml> [--json]" >&2
            echo "  catalog [directory] [--json]" >&2
            return 1
            ;;
        --help|-h)
            echo "Usage: resource-browserless workflow <subcommand> [options]"
            echo "Subcommands:"
            echo "  describe <workflow.yaml> [--json]"
            echo "  catalog [directory] [--json]"
            return 0
            ;;
        *)
            log::error "Unknown workflow subcommand: $subcmd"
            echo "Run 'resource-browserless workflow --help' for usage" >&2
            return 1
            ;;
    esac
}

browserless::workflow_metadata_json() {
    local file="${1:-}"
    if [[ -z "$file" ]]; then
        log::error "Workflow file is required"
        return 1
    fi
    if [[ ! -f "$file" ]]; then
        log::error "Workflow file not found: $file"
        return 1
    fi

    local resolved
    resolved=$(realpath "$file" 2>/dev/null || readlink -f "$file" 2>/dev/null || echo "$file")

    WORKFLOW_METADATA_JSON="{}"
    WORKFLOW_NAME=""
    WORKFLOW_DESCRIPTION=""
    WORKFLOW_VERSION=""
    WORKFLOW_TAGS_JSON="[]"
    WORKFLOW_SOURCE_FILE=""
    WORKFLOW_NAME_INFERRED="false"
    WORKFLOW_SLUG=""

    if ! workflow::load_metadata "$resolved"; then
        return 1
    fi

    local step_count
    step_count=$(yq eval '(.workflow.steps // .steps) | length' "$resolved" 2>/dev/null || echo "0")
    if [[ -z "$step_count" || "$step_count" == "null" ]]; then
        step_count=0
    fi

    echo "$WORKFLOW_METADATA_JSON" | jq -c \
        --arg slug "$WORKFLOW_SLUG" \
        --arg source "$resolved" \
        --arg yaml "$(basename "$resolved")" \
        --arg inferred "$WORKFLOW_NAME_INFERRED" \
        --argjson step_count "$step_count" \
        '. + {
            slug: $slug,
            source_file: $source,
            yaml_file: $yaml,
            step_count: $step_count,
            name_inferred: ($inferred == "true")
        }'
}

browserless::workflow_describe() {
    local json_output=false
    local file=""

    while [[ $# -gt 0 ]]; do
        case "$1" in
            --json)
                json_output=true
                shift
                ;;
            --help|-h)
                echo "Usage: resource-browserless workflow describe [--json] <workflow.yaml>"
                return 0
                ;;
            *)
                file="$1"
                shift
                ;;
        esac
    done

    if [[ -z "$file" ]]; then
        log::error "Workflow file is required"
        return 1
    fi

    local metadata_json
    if ! metadata_json=$(browserless::workflow_metadata_json "$file"); then
        return 1
    fi

    if [[ "$json_output" == true ]]; then
        echo "$metadata_json"
        return 0
    fi

    local name description version slug source steps inferred tags
    name=$(echo "$metadata_json" | jq -r '.name')
    description=$(echo "$metadata_json" | jq -r '.description // ""')
    version=$(echo "$metadata_json" | jq -r '.version // ""')
    slug=$(echo "$metadata_json" | jq -r '.slug')
    source=$(echo "$metadata_json" | jq -r '.source_file')
    steps=$(echo "$metadata_json" | jq -r '.step_count // 0')
    inferred=$(echo "$metadata_json" | jq -r '.name_inferred')
    tags=$(echo "$metadata_json" | jq -r '.tags | join(", ")')

    echo "Workflow: $name"
    if [[ "$inferred" == "true" ]]; then
        echo "  Note: name inferred from filename"
    fi
    if [[ -n "$description" && "$description" != "null" ]]; then
        echo "Description: $description"
    fi
    if [[ -n "$version" && "$version" != "null" ]]; then
        echo "Version: $version"
    fi
    if [[ -n "$tags" ]]; then
        echo "Tags: $tags"
    fi
    echo "Slug: $slug"
    echo "Source: $source"
    echo "Steps: $steps"
}

browserless::workflow_catalog() {
    local root="${BROWSERLESS_CLI_DIR}/examples"
    local json_output=false

    while [[ $# -gt 0 ]]; do
        case "$1" in
            --json)
                json_output=true
                shift
                ;;
            --root)
                root="$2"
                shift 2
                ;;
            --help|-h)
                echo "Usage: resource-browserless workflow catalog [--json] [--root <directory>]"
                return 0
                ;;
            *)
                root="$1"
                shift
                ;;
        esac
    done

    if [[ ! -d "$root" ]]; then
        log::error "Directory not found: $root"
        return 1
    fi

    local tmp_list
    tmp_list=$(mktemp)
    local count=0

    while IFS= read -r file; do
        local metadata_json
        if metadata_json=$(browserless::workflow_metadata_json "$file" 2>/dev/null); then
            echo "$metadata_json" >> "$tmp_list"
            ((count += 1))
            if [[ "$json_output" != true ]]; then
                local name desc version rel
                name=$(echo "$metadata_json" | jq -r '.name')
                desc=$(echo "$metadata_json" | jq -r '.description // ""')
                version=$(echo "$metadata_json" | jq -r '.version // ""')
                rel=${file#$root/}
                printf -- "- %s (%s)\n" "$name" "$rel"
                if [[ -n "$version" && "$version" != "null" ]]; then
                    printf "  Version: %s\n" "$version"
                fi
                if [[ -n "$desc" && "$desc" != "null" ]]; then
                    printf "  %s\n" "$desc"
                fi
                printf '\n'
            fi
        fi
    done < <(find "$root" -type f \( -name '*.yaml' -o -name '*.yml' \) | sort)

    if [[ "$json_output" == true ]]; then
        if [[ -s "$tmp_list" ]]; then
            jq -s '.' "$tmp_list"
        else
            echo '[]'
        fi
    else
        log::info "Total workflows discovered: $count"
    fi

    rm -f "$tmp_list"
    return 0
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
