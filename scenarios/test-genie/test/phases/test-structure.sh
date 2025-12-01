#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/../lib/runtime.sh"

tg::log_info "Validating scenario directory structure"

required_dirs=(
  "${TEST_GENIE_SCENARIO_DIR}/api"
  "${TEST_GENIE_SCENARIO_DIR}/cli"
  "${TEST_GENIE_SCENARIO_DIR}/requirements"
  "${TEST_GENIE_SCENARIO_DIR}/test/phases"
  "${TEST_GENIE_SCENARIO_DIR}/ui"
)

for dir in "${required_dirs[@]}"; do
  tg::require_dir "$dir"
done

tg::log_info "Validating lifecycle manifest"
service_manifest="${TEST_GENIE_SCENARIO_DIR}/.vrooli/service.json"
tg::require_file "$service_manifest"

if command -v jq >/dev/null 2>&1; then
  scenario_name=$(jq -r '.service.name' "$service_manifest" 2>/dev/null)
  if [[ "$scenario_name" != "test-genie" ]]; then
    tg::fail "Expected .service.name to equal 'test-genie' but found '${scenario_name}'"
  fi
  critical_checks=$(jq '.lifecycle.health.checks | length' "$service_manifest")
  if [[ "$critical_checks" -lt 1 ]]; then
    tg::fail "Lifecycle health checks must be defined"
  fi
else
  tg::log_warn "jq not found; manifest validation limited to existence checks"
fi

tg::log_info "Verifying scenario-local orchestrator is referenced"
if grep -q "scripts/scenarios/testing" "${TEST_GENIE_SCENARIO_DIR}/test/run-tests.sh"; then
  tg::fail "run-tests.sh must not rely on scripts/scenarios/testing"
fi

tg::log_success "Structure validation complete"
