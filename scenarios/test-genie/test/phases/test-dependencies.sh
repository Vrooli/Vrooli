#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/../lib/runtime.sh"

tg::log_info "Validating required toolchain availability"
required_commands=(bash curl go jq)
for cmd in "${required_commands[@]}"; do
  tg::require_command "$cmd" "Required command '${cmd}' is not available"
done

if command -v pnpm >/dev/null 2>&1; then
  tg::log_success "pnpm detected for UI workflows"
else
  tg::log_warn "pnpm not detected; UI validations will be skipped"
fi

cli_path="${TEST_GENIE_SCENARIO_DIR}/cli/test-genie"
tg::require_executable "$cli_path" "CLI binary '${cli_path}' must be executable"

tg::require_dir "${TEST_GENIE_SCENARIO_DIR}/initialization" "Initialization assets missing"

tg::log_info "Checking lifecycle dependencies for postgres resource"
manifest="${TEST_GENIE_SCENARIO_DIR}/.vrooli/service.json"
if command -v jq >/dev/null 2>&1; then
  postgres_required=$(jq -r '.dependencies.resources.postgres.required' "$manifest" 2>/dev/null || echo "false")
  if [[ "$postgres_required" != "true" ]]; then
    tg::fail "Postgres resource must be marked as required in service manifest"
  fi
else
  tg::log_warn "jq missing; skipping manifest dependency verification"
fi

tg::log_success "Dependency validation complete"
