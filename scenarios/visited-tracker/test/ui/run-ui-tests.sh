#!/bin/bash
# UI automation test runner for visited-tracker
# Executes browser automation workflows using browser-automation-studio
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"
source "${APP_ROOT}/scripts/lib/utils/log.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/core.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/connectivity.sh"

SCENARIO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
SCENARIO_NAME=$(basename "$SCENARIO_DIR")

log::info "🌐 Running UI automation tests for $SCENARIO_NAME..."

TEST_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
WORKFLOWS_DIR="$TEST_DIR/workflows"
VROOLI_ROOT="${VROOLI_ROOT:-$(cd "$SCENARIO_DIR/../.." && pwd)}"
BROWSER_AUTOMATION_CLI="$VROOLI_ROOT/scenarios/browser-automation-studio/cli/browser-automation-studio"

if [ ! -x "$BROWSER_AUTOMATION_CLI" ]; then
    log::error "❌ browser-automation-studio CLI not found or not executable at: $BROWSER_AUTOMATION_CLI"
    echo "   Install and enable the browser-automation-studio scenario before running UI tests."
    exit 1
fi

if [ ! -d "$WORKFLOWS_DIR" ]; then
    log::error "❌ Workflow directory missing: $WORKFLOWS_DIR"
    echo "   Add JSON workflows that describe the UI automation you expect."
    exit 1
fi

mapfile -t workflow_files < <(find "$WORKFLOWS_DIR" -maxdepth 1 -name "*.json" -type f | sort)
if [ ${#workflow_files[@]} -eq 0 ]; then
    log::error "❌ No UI automation workflows found in $WORKFLOWS_DIR"
    exit 1
fi

if testing::core::ensure_runtime_or_skip "$SCENARIO_NAME" "UI automation tests"; then
    :
else
    status=$?
    if [ "$status" -eq 200 ]; then
        exit 200
    else
        exit 1
    fi
fi

if ! testing::core::wait_for_scenario "$SCENARIO_NAME" 20 >/dev/null 2>&1; then
    log::error "❌ Scenario '$SCENARIO_NAME' did not become ready in time"
    exit 1
fi

if ! UI_URL=$(testing::connectivity::get_ui_url "$SCENARIO_NAME"); then
    log::error "❌ Could not determine UI URL for $SCENARIO_NAME"
    exit 1
fi

if ! curl -s --max-time 5 "$UI_URL" >/dev/null 2>&1; then
    log::error "❌ UI is not responding at $UI_URL"
    exit 1
fi

log::info "🎯 UI automation target: $UI_URL"

echo ""
echo "🤖 Running ${#workflow_files[@]} UI automation workflow(s)..."

workflow_count=0
failed_count=0

for workflow in "${workflow_files[@]}"; do
    workflow_name=$(basename "$workflow")
    echo ""
    log::info "🚀 Executing workflow: $workflow_name"
    if "$BROWSER_AUTOMATION_CLI" execute "$workflow"; then
        log::success "✅ Workflow passed: $workflow_name"
    else
        log::error "❌ Workflow failed: $workflow_name"
        ((failed_count+=1))
    fi
    ((workflow_count+=1))
done

echo ""
echo "📊 UI Automation Test Summary:"
echo "   Workflows executed: $workflow_count"
echo "   Workflows passed: $((workflow_count - failed_count))"
echo "   Workflows failed: $failed_count"

if [ $failed_count -eq 0 ]; then
    log::success "✅ All UI automation workflows passed"
    exit 0
else
    log::error "❌ UI automation failures detected"
    exit 1
fi
