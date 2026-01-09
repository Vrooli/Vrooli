#!/usr/bin/env bats

setup() {
  CLI_ROOT="$(cd "$(dirname "${BATS_TEST_FILENAME}")" && pwd)"
  CLI_BIN="${CLI_ROOT}/symbol-search"
}

@test "symbol-search help prints usage" {
  run "${CLI_BIN}" help
  [ "$status" -eq 0 ]
  [[ "$output" == *"Symbol Search CLI"* ]]
}

@test "symbol-search version outputs json" {
  run "${CLI_BIN}" version --json
  [ "$status" -eq 0 ]
  echo "$output" | jq -e '.cli_version' >/dev/null
}
