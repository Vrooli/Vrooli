#!/usr/bin/env bats

# [REQ:GCT-OT-P0-001] Health check endpoint

setup() {
  if command -v git-control-tower >/dev/null 2>&1; then
    CLI_SCRIPT="git-control-tower"
  else
    CLI_SCRIPT="${BATS_TEST_FILENAME%/*}/git-control-tower"
  fi
  export GIT_CONTROL_TOWER_API_BASE="http://localhost:${API_PORT:-18700}"
  export API_BASE_URL="$GIT_CONTROL_TOWER_API_BASE"
}

@test "CLI shows help" {
  run "$CLI_SCRIPT" help
  [ "$status" -eq 0 ]
  [[ "$output" =~ "git-control-tower CLI" ]]
}

@test "CLI can check health if API is running" {
  if curl -sf "$GIT_CONTROL_TOWER_API_BASE/health" >/dev/null; then
    run "$CLI_SCRIPT" status
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Status:" ]]
  else
    skip "API not running"
  fi
}

@test "CLI can fetch repository status if API is running" {
  if curl -sf "$GIT_CONTROL_TOWER_API_BASE/health" >/dev/null; then
    run "$CLI_SCRIPT" repo-status
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Branch:" ]]
    [[ "$output" =~ "Changes:" ]]
  else
    skip "API not running"
  fi
}
