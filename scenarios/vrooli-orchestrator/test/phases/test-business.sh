#!/bin/bash
# Validates business workflows such as seeded profiles and CLI UX.

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "180s" --require-runtime

SCENARIO_NAME="${TESTING_PHASE_SCENARIO_NAME}"
API_URL=$(testing::connectivity::get_api_url "$SCENARIO_NAME" 2>/dev/null || echo "")

if [ -z "$API_URL" ]; then
  testing::phase::add_error "Unable to resolve API URL for $SCENARIO_NAME"
  testing::phase::end_with_summary "Business validation incomplete"
fi

verify_developer_profile_api() {
  local response
  response=$(curl -sSf "${API_URL}/api/v1/profiles") || return 1
  if command -v jq >/dev/null 2>&1; then
    echo "$response" | jq -e '.profiles[] | select(.name == "developer")' >/dev/null
  else
    echo "$response" | grep -q 'developer'
  fi
}

verify_developer_profile_cli() {
  bash -c 'cd cli && ./vrooli-orchestrator list-profiles --json' | {
    if command -v jq >/dev/null 2>&1; then
      jq -e '.profiles[] | select(.name == "developer")' >/dev/null
    else
      grep -q 'developer'
    fi
  }
}

run_cli_bats() {
  if command -v bats >/dev/null 2>&1; then
    bash -c 'cd cli && bats vrooli-orchestrator.bats'
  else
    echo "bats not installed" >&2
    return 2
  fi
}

verify_default_activation_target() {
  curl -sSf "${API_URL}/api/v1/status" | {
    if command -v jq >/dev/null 2>&1; then
      jq -e '.service == "vrooli-orchestrator"'
    else
      grep -q 'vrooli-orchestrator'
    fi
  }
}

if testing::phase::check "Developer profile present via API" verify_developer_profile_api; then :; fi

if [ -x "cli/vrooli-orchestrator" ]; then
  if testing::phase::check "Developer profile present via CLI" verify_developer_profile_cli; then :; fi
else
  testing::phase::add_warning "CLI executable missing; skipping CLI profile validation"
  testing::phase::add_test skipped
fi

if testing::phase::check "Status endpoint reports orchestrator service" verify_default_activation_target; then :; fi

if [ -f "cli/vrooli-orchestrator.bats" ]; then
  if run_cli_bats; then
    testing::phase::add_test passed
  else
    status=$?
    if [ $status -eq 2 ]; then
      testing::phase::add_warning "bats CLI not available; skipping CLI workflow tests"
      testing::phase::add_test skipped
    else
      testing::phase::add_error "CLI BATS suite failed"
      testing::phase::add_test failed
    fi
  fi
else
  testing::phase::add_warning "CLI BATS suite missing; skipping"
  testing::phase::add_test skipped
fi

testing::phase::end_with_summary "Business validation completed"
