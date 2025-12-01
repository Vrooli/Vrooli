#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/../lib/runtime.sh"

tg::require_command go "Go toolchain is required for API unit tests"

tg::log_info "Running Go unit tests"
tg::run_in_dir "${TEST_GENIE_SCENARIO_DIR}/api" go test ./...

tg::log_info "Performing shell syntax checks"
shell_targets=(
  "${TEST_GENIE_SCENARIO_DIR}/cli/test-genie"
  "${TEST_GENIE_SCENARIO_DIR}/test/lib/runtime.sh"
  "${TEST_GENIE_SCENARIO_DIR}/test/lib/orchestrator.sh"
)

for file in "${shell_targets[@]}"; do
  tg::require_file "$file"
  bash -n "$file"
  tg::log_success "bash -n ${file}"
done

tg::log_success "Unit validation complete"
