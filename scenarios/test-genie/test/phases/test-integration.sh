#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/../lib/runtime.sh"

cli_bin="${TEST_GENIE_SCENARIO_DIR}/cli/test-genie"
tg::require_executable "$cli_bin"

tg::log_info "Verifying CLI help command"
if ! "$cli_bin" help >/dev/null; then
  tg::fail "test-genie help command failed"
fi

tg::log_info "Verifying CLI version command"
version_output="$("$cli_bin" version)"
if [[ "$version_output" != *"version"* ]]; then
  tg::fail "CLI version output malformed: ${version_output}"
fi

tg::log_info "Inspecting orchestrator phase listing"
phase_listing="$("${TEST_GENIE_SCENARIO_DIR}/test/run-tests.sh" --list)"
for phase in structure dependencies unit integration business performance; do
  if [[ "$phase_listing" != *"$phase"* ]]; then
    tg::fail "Expected phase '${phase}' in orchestrator listing"
  fi
done

tg::require_command bats "Bats is required for CLI acceptance tests"
tg::log_info "Running CLI acceptance tests"
tg::run_in_dir "${TEST_GENIE_SCENARIO_DIR}/cli" bats --tap test-genie.bats
if compgen -G "${TEST_GENIE_SCENARIO_DIR}/cli/test/*.bats" >/dev/null; then
  while IFS= read -r -d '' bats_file; do
    rel_path="${bats_file#"${TEST_GENIE_SCENARIO_DIR}/cli/"}"
    tg::log_info "Executing ${rel_path}"
    tg::run_in_dir "${TEST_GENIE_SCENARIO_DIR}/cli" bats --tap "$rel_path"
  done < <(find "${TEST_GENIE_SCENARIO_DIR}/cli/test" -maxdepth 1 -name '*.bats' -print0)
fi

tg::log_success "Integration validation complete"
