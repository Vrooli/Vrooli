#!/usr/bin/env bats
# Acceptance tests for the 'test-genie generate' command.

readonly TEST_CLI="./test-genie"
readonly TEST_CONFIG_DIR="$HOME/.test-genie"
readonly TEST_CONFIG_FILE="$TEST_CONFIG_DIR/config.json"

setup() {
    if [[ -f "$TEST_CONFIG_FILE" ]]; then
        mv "$TEST_CONFIG_FILE" "$TEST_CONFIG_FILE.bak"
    fi

    export TEST_GENIE_API_BASE="http://localhost:9999/api/v1"
    export TEST_GENIE_API_TOKEN=""

    export TEST_CLI_CURL_CAPTURE="${BATS_TEST_TMPDIR}/curl_payload.json"
    mkdir -p "${BATS_TEST_TMPDIR}/bin"
    cat > "${BATS_TEST_TMPDIR}/bin/curl" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail

capture="${TEST_CLI_CURL_CAPTURE:-}"
next_is_body=0
for arg in "$@"; do
  if [[ "$arg" == "-d" ]]; then
    next_is_body=1
    continue
  fi
  if [[ $next_is_body -eq 1 ]]; then
    if [[ -n "$capture" ]]; then
      printf '%s' "$arg" > "$capture"
    fi
    next_is_body=0
    continue
  fi
done

cat <<'JSON'
{
  "id": "11111111-1111-1111-1111-111111111111",
  "scenarioName": "demo",
  "status": "queued",
  "requestedTypes": ["unit","integration"],
  "coverageTarget": 92,
  "priority": "high",
  "estimatedQueueTimeSeconds": 210
}
JSON
EOF
    chmod +x "${BATS_TEST_TMPDIR}/bin/curl"
    export PATH="${BATS_TEST_TMPDIR}/bin:${PATH}"
    rm -f "$TEST_CLI_CURL_CAPTURE"
}

teardown() {
    if [[ -f "$TEST_CONFIG_FILE.bak" ]]; then
        mv "$TEST_CONFIG_FILE.bak" "$TEST_CONFIG_FILE"
    else
        rm -f "$TEST_CONFIG_FILE"
    fi
}

@test "generate command requires a scenario argument [REQ:TESTGENIE-SUITE-P0]" {
    run $TEST_CLI generate
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Scenario name is required" ]]
}

@test "generate command posts payload with custom options [REQ:TESTGENIE-SUITE-P0]" {
    run $TEST_CLI generate sample-scenario --types unit,performance --coverage 92 --priority high --notes "delegated request"
    [ "$status" -eq 0 ]
    [[ -f "$TEST_CLI_CURL_CAPTURE" ]]

    scenarioName=$(jq -r '.scenarioName' "$TEST_CLI_CURL_CAPTURE")
    typesCount=$(jq '.requestedTypes | length' "$TEST_CLI_CURL_CAPTURE")
    coverage=$(jq '.coverageTarget' "$TEST_CLI_CURL_CAPTURE")
    priority=$(jq -r '.priority' "$TEST_CLI_CURL_CAPTURE")
    notes=$(jq -r '.notes' "$TEST_CLI_CURL_CAPTURE")

    [ "$scenarioName" = "sample-scenario" ]
    [ "$typesCount" -eq 2 ]
    [ "$coverage" -eq 92 ]
    [ "$priority" = "high" ]
    [ "$notes" = "delegated request" ]
    [[ "$output" =~ "Suite request queued" ]]
}

@test "generate command supports --json output [REQ:TESTGENIE-SUITE-P0]" {
    run $TEST_CLI generate demo-scenario --json
    [ "$status" -eq 0 ]
    [[ "$output" =~ "{" ]]
    [[ "$output" =~ "\"scenarioName\"" ]]
}
