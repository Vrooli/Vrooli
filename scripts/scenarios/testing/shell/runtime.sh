#!/usr/bin/env bash
# Runtime lifecycle management helpers for scenario test runners
set -euo pipefail

SHELL_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SHELL_DIR/core.sh"

# Internal state
TESTING_RUNTIME_SCENARIO_NAME=""
TESTING_RUNTIME_ENABLED=false
TESTING_RUNTIME_MANAGED=false
TESTING_RUNTIME_WAS_RUNNING=false
TESTING_RUNTIME_STOPPED=false
TESTING_RUNTIME_DRY_RUN=false

# Configure runtime management with the scenario name and default enabled flag
# Usage: testing::runtime::configure "scenario-name" [enabled]
testing::runtime::configure() {
    TESTING_RUNTIME_SCENARIO_NAME="$1"
    if [ "${2:-}" = "true" ]; then
        TESTING_RUNTIME_ENABLED=true
    else
        TESTING_RUNTIME_ENABLED=false
    fi
}

# Override runtime management at execution time
# Usage: testing::runtime::set_enabled true|false
testing::runtime::set_enabled() {
    case "$1" in
        true|TRUE|1|on|ON)
            TESTING_RUNTIME_ENABLED=true
            ;;
        false|FALSE|0|off|OFF)
            TESTING_RUNTIME_ENABLED=false
            ;;
    esac
}

# Register dry run mode so auto-starts are skipped
testing::runtime::set_dry_run() {
    TESTING_RUNTIME_DRY_RUN=true
}

# Ensure runtime is available for the given item when required
# Arguments: $1=item_name, $2=requires_runtime (true/false)
testing::runtime::ensure_available() {
    local item_name="$1"
    local requires_runtime="$2"

    if [ "$requires_runtime" != "true" ]; then
        return 0
    fi

    if [ "$TESTING_RUNTIME_ENABLED" != "true" ]; then
        return 0
    fi

    if [ "$TESTING_RUNTIME_MANAGED" = "true" ]; then
        return 0
    fi

    if testing::core::is_scenario_running "$TESTING_RUNTIME_SCENARIO_NAME"; then
        TESTING_RUNTIME_MANAGED=true
        TESTING_RUNTIME_WAS_RUNNING=true
        if command -v log::info >/dev/null 2>&1; then
            log::info "🟢 Scenario '$TESTING_RUNTIME_SCENARIO_NAME' already running; leaving lifecycle untouched"
        else
            echo "🟢 Scenario '$TESTING_RUNTIME_SCENARIO_NAME' already running; leaving lifecycle untouched"
        fi
        return 0
    fi

    if [ "$TESTING_RUNTIME_DRY_RUN" = "true" ]; then
        TESTING_RUNTIME_MANAGED=true
        TESTING_RUNTIME_WAS_RUNNING=true
        if command -v log::warning >/dev/null 2>&1; then
            log::warning "⚠️  [DRY RUN] Would auto-start scenario '$TESTING_RUNTIME_SCENARIO_NAME' before '$item_name'"
        else
            echo "⚠️  [DRY RUN] Would auto-start scenario '$TESTING_RUNTIME_SCENARIO_NAME' before '$item_name'"
        fi
        return 0
    fi

    if ! command -v vrooli >/dev/null 2>&1; then
        if command -v log::error >/dev/null 2>&1; then
            log::error "❌ Cannot auto-start scenario; 'vrooli' CLI not available"
        else
            echo "❌ Cannot auto-start scenario; 'vrooli' CLI not available" >&2
        fi
        return 1
    fi

    if command -v log::info >/dev/null 2>&1; then
        log::info "🚚 Auto-starting scenario '$TESTING_RUNTIME_SCENARIO_NAME' for '$item_name'"
    else
        echo "🚚 Auto-starting scenario '$TESTING_RUNTIME_SCENARIO_NAME' for '$item_name'"
    fi

    if ! vrooli scenario start "$TESTING_RUNTIME_SCENARIO_NAME"; then
        if command -v log::error >/dev/null 2>&1; then
            log::error "❌ Failed to auto-start scenario '$TESTING_RUNTIME_SCENARIO_NAME'"
        else
            echo "❌ Failed to auto-start scenario '$TESTING_RUNTIME_SCENARIO_NAME'" >&2
        fi
        return 1
    fi

    if ! testing::core::wait_for_scenario "$TESTING_RUNTIME_SCENARIO_NAME" 90 >/dev/null 2>&1; then
        if command -v log::error >/dev/null 2>&1; then
            log::error "❌ Scenario '$TESTING_RUNTIME_SCENARIO_NAME' did not become ready after auto-start"
        else
            echo "❌ Scenario '$TESTING_RUNTIME_SCENARIO_NAME' did not become ready after auto-start" >&2
        fi
        return 1
    fi

    TESTING_RUNTIME_MANAGED=true
    TESTING_RUNTIME_WAS_RUNNING=false
    if command -v log::success >/dev/null 2>&1; then
        log::success "✅ Scenario '$TESTING_RUNTIME_SCENARIO_NAME' ready for runtime-dependent tests"
    else
        echo "✅ Scenario '$TESTING_RUNTIME_SCENARIO_NAME' ready for runtime-dependent tests"
    fi
    return 0
}

# Stop scenario if we started it
testing::runtime::cleanup() {
    if [ "$TESTING_RUNTIME_ENABLED" != "true" ]; then
        return
    fi

    if [ "$TESTING_RUNTIME_MANAGED" != "true" ]; then
        return
    fi

    if [ "$TESTING_RUNTIME_WAS_RUNNING" = "true" ]; then
        return
    fi

    if [ "$TESTING_RUNTIME_STOPPED" = "true" ]; then
        return
    fi

    if ! command -v vrooli >/dev/null 2>&1; then
        if command -v log::warning >/dev/null 2>&1; then
            log::warning "⚠️  Skipping auto-stop; 'vrooli' CLI not available"
        else
            echo "⚠️  Skipping auto-stop; 'vrooli' CLI not available"
        fi
        return
    fi

    if command -v log::info >/dev/null 2>&1; then
        log::info "🛑 Auto-stopping scenario '$TESTING_RUNTIME_SCENARIO_NAME'"
    else
        echo "🛑 Auto-stopping scenario '$TESTING_RUNTIME_SCENARIO_NAME'"
    fi

    if vrooli scenario stop "$TESTING_RUNTIME_SCENARIO_NAME"; then
        TESTING_RUNTIME_STOPPED=true
        if command -v log::success >/dev/null 2>&1; then
            log::success "✅ Scenario '$TESTING_RUNTIME_SCENARIO_NAME' stopped"
        else
            echo "✅ Scenario '$TESTING_RUNTIME_SCENARIO_NAME' stopped"
        fi
    else
        if command -v log::warning >/dev/null 2>&1; then
            log::warning "⚠️  Failed to stop scenario '$TESTING_RUNTIME_SCENARIO_NAME' automatically"
        else
            echo "⚠️  Failed to stop scenario '$TESTING_RUNTIME_SCENARIO_NAME' automatically" >&2
        fi
    fi
}

# Discover ports once for the scenario and export defaults
# Returns API and UI ports (stdout) when discovered.
testing::runtime::discover_ports() {
    local scenario_name="$1"
    local default_api="${2:-17695}"
    local default_ui="${3:-38442}"

    local resolved_api="${API_PORT:-}"
    local resolved_ui="${UI_PORT:-}"

    if command -v vrooli >/dev/null 2>&1; then
        resolved_api=${resolved_api:-$(vrooli scenario port "$scenario_name" API_PORT 2>/dev/null || echo "")}
        resolved_ui=${resolved_ui:-$(vrooli scenario port "$scenario_name" UI_PORT 2>/dev/null || echo "")}
    fi

    export API_PORT="${resolved_api:-$default_api}"
    export UI_PORT="${resolved_ui:-$default_ui}"

    :
}

# Exported functions
export -f testing::runtime::configure
export -f testing::runtime::set_enabled
export -f testing::runtime::set_dry_run
export -f testing::runtime::ensure_available
export -f testing::runtime::cleanup
export -f testing::runtime::discover_ports
