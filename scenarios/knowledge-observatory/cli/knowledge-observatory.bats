#!/usr/bin/env bats
# Tests for knowledge-observatory CLI

setup() {
    SCENARIO_ROOT="$(cd "${BATS_TEST_DIRNAME}/.." && pwd)"
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
}

@test "CLI binary builds" {
    [[ -x "$CLI_BIN" ]]
}

@test "help command displays usage information" {
    run "$CLI_BIN" help
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Usage:" ]]
    [[ "$output" =~ "Commands:" ]]
}

@test "version command displays version" {
    run "$CLI_BIN" version
    [ "$status" -eq 0 ]
    [[ "$output" =~ "knowledge-observatory CLI version" ]]
}

@test "configure command can set and retrieve values" {
    run "$CLI_BIN" configure api_base http://test.example.com
    [ "$status" -eq 0 ]

    run "$CLI_BIN" configure
    [ "$status" -eq 0 ]
    [[ "$output" =~ "test.example.com" ]]
    [[ -f "${KNOWLEDGE_OBSERVATORY_CONFIG_DIR}/config.json" ]]
}

@test "help output lists supported commands" {
    run "$CLI_BIN" help
    [ "$status" -eq 0 ]
    [[ "$output" =~ "status" ]]
    [[ "$output" =~ "search" ]]
    [[ "$output" =~ "ingest" ]]
    [[ "$output" =~ "ingest-job" ]]
    [[ "$output" =~ "job-status" ]]
    [[ "$output" =~ "configure" ]]
}

@test "invalid command shows error" {
    run "$CLI_BIN" invalid_command
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Unknown command" ]]
}
