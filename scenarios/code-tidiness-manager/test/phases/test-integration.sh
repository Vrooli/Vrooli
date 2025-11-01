#!/bin/bash
# Lightweight integration checks for code-tidiness-manager

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "180s"

scenario_name="$TESTING_PHASE_SCENARIO_NAME"
api_port=""

if command -v vrooli >/dev/null 2>&1; then
  api_port=$(vrooli scenario port "$scenario_name" API_PORT 2>/dev/null || true)
fi

if [ -z "$api_port" ]; then
  testing::phase::add_warning "Scenario runtime not detected; skipping HTTP integration checks"
  testing::phase::add_test skipped
else
  api_port=$(echo "$api_port" | awk -F= '/=/{print $2}' | tr -d ' ')
  api_port=${api_port:-$(echo "$api_port" | tr -d '[:space:]')}
  if [ -n "$api_port" ]; then
    testing::phase::check "API health endpoint" curl -fsS "http://localhost:${api_port}/health"
  else
    testing::phase::add_warning "Unable to resolve API port; skipping HTTP checks"
    testing::phase::add_test skipped
  fi
fi

if [ -x cli/code-tidiness-manager ]; then
  if testing::phase::check "CLI help command" cli/code-tidiness-manager --help >/dev/null; then
    :
  fi
elif [ -f cli/code-tidiness-manager ]; then
  testing::phase::add_warning "cli/code-tidiness-manager not executable"
  testing::phase::add_test skipped
else
  testing::phase::add_warning "CLI binary not built; skipping CLI smoke test"
  testing::phase::add_test skipped
fi

if [ -f cli/code-tidiness-manager.bats ] && command -v bats >/dev/null 2>&1; then
  testing::phase::check "CLI bats suite" bash -c 'cd cli && bats code-tidiness-manager.bats'
else
  testing::phase::add_warning "CLI BATS suite unavailable; skipping"
  testing::phase::add_test skipped
fi

if command -v jq >/dev/null 2>&1; then
  for workflow in initialization/automation/n8n/code-scanner.json initialization/automation/n8n/cleanup-executor.json initialization/automation/n8n/pattern-analyzer.json; do
    if [ -f "$workflow" ]; then
      testing::phase::check "n8n workflow loads: $(basename "$workflow")" jq empty "$workflow"
    else
      testing::phase::add_warning "Workflow file missing: $workflow"
      testing::phase::add_test skipped
    fi
  done
else
  testing::phase::add_warning "jq not available; skipping workflow validation"
  testing::phase::add_test skipped
fi

testing::phase::end_with_summary "Integration validation completed"
