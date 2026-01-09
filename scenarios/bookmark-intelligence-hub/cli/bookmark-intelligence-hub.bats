#!/usr/bin/env bats

setup() {
  export PATH="$PATH"
  if [[ -z "${API_BASE_URL:-}" ]]; then
    local port
    port=$(vrooli scenario port bookmark-intelligence-hub API_PORT 2>/dev/null)
    if [[ -n "$port" ]]; then
      export API_BASE_URL="http://localhost:${port}"
    fi
  fi
}

@test "bookmark-intelligence-hub CLI shows help" {
  run bookmark-intelligence-hub --help
  [ "$status" -eq 0 ]
  [[ "$output" == *"Bookmark Intelligence Hub CLI"* ]]
}

@test "bookmark-intelligence-hub health reports healthy status" {
  run bookmark-intelligence-hub health --json
  [ "$status" -eq 0 ]
  [[ "$output" == *"healthy"* ]]
}
