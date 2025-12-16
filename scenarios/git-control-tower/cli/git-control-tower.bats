#!/usr/bin/env bats

# [REQ:GCT-OT-P0-001] Health check endpoint

setup() {
  CLI_SCRIPT="${BATS_TEST_FILENAME%/*}/git-control-tower"
  export GIT_CONTROL_TOWER_API_BASE="http://localhost:${API_PORT:-18700}"
  export VROOLI_LIFECYCLE_MANAGED="true"
}

@test "CLI shows help" {
  run bash "$CLI_SCRIPT" help
  [ "$status" -eq 0 ]
  [[ "$output" =~ "Git Control Tower CLI" ]]
}

@test "CLI can check health if API is running" {
  if curl -sf "$GIT_CONTROL_TOWER_API_BASE/health" >/dev/null; then
    run bash "$CLI_SCRIPT" status
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Status:" ]]
  else
    skip "API not running"
  fi
}
