#!/usr/bin/env bats
# Tests for Knowledge Observatory CLI commands [REQ:KO-HD-006]

setup() {
  # Get scenario root
  SCENARIO_ROOT="${BATS_TEST_DIRNAME}/../.."
  CLI_BIN="${SCENARIO_ROOT}/cli/knowledge-observatory"

  # Ensure CLI binary exists
  if [[ ! -f "$CLI_BIN" ]]; then
    skip "CLI binary not found at $CLI_BIN"
  fi

  # Get API port from scenario
  API_PORT=$(grep -A 10 "allocated_ports" "${SCENARIO_ROOT}/.vrooli/service.json" | grep "API_PORT" | sed 's/.*: *"\?\([0-9]*\)"\?.*/\1/')
  if [[ -z "$API_PORT" ]]; then
    API_PORT=17822  # Default fallback
  fi
  export API_URL="http://localhost:${API_PORT}"
}

# CLI Basic Tests [REQ:KO-HD-006]

@test "[REQ:KO-HD-006] CLI command 'help' executes successfully" {
  run "$CLI_BIN" help
  [ "$status" -eq 0 ]
}

@test "[REQ:KO-HD-006] CLI command 'version' executes successfully" {
  run "$CLI_BIN" version
  [ "$status" -eq 0 ]
}

@test "[REQ:KO-HD-006] CLI command 'status' executes successfully" {
  run "$CLI_BIN" status
  [ "$status" -eq 0 ]
}
