#!/usr/bin/env bats
# Tests for Knowledge Observatory CLI commands [REQ:KO-HD-006]

setup() {
  SCENARIO_ROOT="${BATS_TEST_DIRNAME}/../.."
  CLI_DIR="${SCENARIO_ROOT}/cli"

  if ! command -v go &>/dev/null; then
    skip "Go toolchain not available"
  fi

  TMP_DIR="${BATS_TEST_TMPDIR:-${BATS_TMPDIR:-/tmp}}"
  CLI_BIN="${TMP_DIR}/knowledge-observatory"

  if [[ ! -x "$CLI_BIN" ]]; then
    go build -o "$CLI_BIN" "$CLI_DIR"
  fi

  export KNOWLEDGE_OBSERVATORY_CONFIG_DIR="${TMP_DIR}/ko-config"

  API_PORT=$(grep -A 10 "allocated_ports" "${SCENARIO_ROOT}/.vrooli/service.json" | grep "API_PORT" | sed 's/.*: *"\?\([0-9]*\)"\?.*/\1/')
  if [[ -z "$API_PORT" ]]; then
    API_PORT=17822
  fi
  export KNOWLEDGE_OBSERVATORY_API_PORT="$API_PORT"

  API_AVAILABLE="false"
  if command -v curl &>/dev/null; then
    if curl -sf "http://localhost:${API_PORT}/health" &>/dev/null; then
      API_AVAILABLE="true"
    fi
  fi
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
  if [[ "$API_AVAILABLE" != "true" ]]; then
    skip "API not reachable for status check"
  fi
  run "$CLI_BIN" status
  [ "$status" -eq 0 ]
}
