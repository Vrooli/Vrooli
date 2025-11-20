#!/usr/bin/env bats
# Regression tests for ui-smoke helper utilities

APP_ROOT="${APP_ROOT:-$(builtin cd "$(dirname "${BATS_TEST_FILENAME}")/../../../.." && builtin pwd)}"
# shellcheck disable=SC1091
source "${APP_ROOT}/__test/fixtures/setup.bash"

setup_file() {
    vrooli_setup_unit_test
}

teardown_file() {
    vrooli_cleanup_test
}

setup() {
    export TEST_SCENARIO_DIR="$VROOLI_TEST_TMPDIR/ui-smoke-${BATS_TEST_NUMBER}"
    rm -rf "$TEST_SCENARIO_DIR"
    mkdir -p "$TEST_SCENARIO_DIR"
}

teardown() {
    rm -rf "$TEST_SCENARIO_DIR"
}

@test "ui-smoke summary handles large raw payloads" {
    run env APP_ROOT="$APP_ROOT" TEST_SCENARIO_DIR="$TEST_SCENARIO_DIR" bash -c '
        set -euo pipefail
        source "$APP_ROOT/scripts/scenarios/testing/shell/ui-smoke.sh"
        scenario_dir="$TEST_SCENARIO_DIR"
        scenario_name="ui-smoke-fixture"
        mkdir -p "$scenario_dir/coverage/$scenario_name/ui-smoke"
        large_payload=$(python3 - <<"PY"
import json
payload = {
    "success": False,
    "handshake": {"signaled": False},
    "console": [],
    "network": [],
    "html": "X" * (1024 * 256)
}
print(json.dumps(payload))
PY
)
        summary=$(testing::ui_smoke::_build_summary_json --scenario "$scenario_name" --status "failed" --message "overflow test" --scenario-dir "$scenario_dir" --browserless '{}' --bundle '{}' --iframe '{}' --artifacts '{}' --raw "$large_payload" --ui-url "http://localhost" --duration 1000)
        printf '%s' "$summary"
    '
    assert_success
    assert_output_contains '"scenario": "ui-smoke-fixture"'
    assert_output_contains '"message": "overflow test"'
}

@test "ui-smoke marks Browserless offline as blocked" {
    run env APP_ROOT="$APP_ROOT" TEST_SCENARIO_DIR="$TEST_SCENARIO_DIR" bash -c '
        set -euo pipefail
        source "$APP_ROOT/scripts/scenarios/testing/shell/ui-smoke.sh"
        scenario_dir="$TEST_SCENARIO_DIR/offline"
        mkdir -p "$scenario_dir/ui"
        cat <<JSON > "$scenario_dir/ui/package.json"
{
  "dependencies": {
    "@vrooli/iframe-bridge": "0.0.0-test"
  }
}
JSON
        testing::ui_smoke::_check_bundle_freshness() {
            jq -n '{"fresh":true}'
        }
        testing::ui_smoke::_browserless_status() {
            return 1
        }
        TESTING_PHASE_SCENARIO_NAME="offline"
        TESTING_PHASE_SCENARIO_DIR="$scenario_dir"
        testing::ui_smoke::run --scenario offline --scenario-dir "$scenario_dir" --json
    '
    assert_failure
    if [[ "$status" -ne 50 ]]; then
        fail "expected exit code 50, got $status"
    fi
    if [[ "$output" != *'"status": "blocked"'* ]]; then
        fail "expected blocked status in output"
    fi
    if [[ "$output" != *"resource-browserless manage start"* ]]; then
        fail "expected remediation hint in output"
    fi
}

@test "ui-smoke skips when UI directory missing" {
    run env APP_ROOT="$APP_ROOT" TEST_SCENARIO_DIR="$TEST_SCENARIO_DIR" bash -c '
        set -euo pipefail
        source "$APP_ROOT/scripts/scenarios/testing/shell/ui-smoke.sh"
        scenario_dir="$TEST_SCENARIO_DIR/no-ui"
        mkdir -p "$scenario_dir"
        TESTING_PHASE_SCENARIO_NAME="no-ui"
        TESTING_PHASE_SCENARIO_DIR="$scenario_dir"
        testing::ui_smoke::run --scenario no-ui --scenario-dir "$scenario_dir" --json
    '
    assert_success
    if [[ "$output" != *'"status": "skipped"'* ]]; then
        fail "expected skipped status in output"
    fi
    if [[ "$output" != *'"message": "UI directory not detected"'* ]]; then
        fail "expected missing UI message"
    fi
}

@test "ui-smoke blocks when bundle is stale" {
    run env APP_ROOT="$APP_ROOT" TEST_SCENARIO_DIR="$TEST_SCENARIO_DIR" bash -c '
        set -euo pipefail
        source "$APP_ROOT/scripts/scenarios/testing/shell/ui-smoke.sh"
        scenario_dir="$TEST_SCENARIO_DIR/stale"
        mkdir -p "$scenario_dir/ui"
        cat <<JSON > "$scenario_dir/ui/package.json"
{
  "dependencies": {
    "@vrooli/iframe-bridge": "0.0.0-test"
  }
}
JSON
        testing::runtime::ensure_available() { return 0; }
        testing::runtime::discover_ports() { export UI_PORT=18000; }
        testing::runtime::cleanup() { :; }
        testing::ui_smoke::_check_bundle_freshness() {
            cat <<'JSON'
{"fresh":false,"reason":"stale bundle"}
JSON
        }
        testing::ui_smoke::_browserless_status() {
            cat <<'JSON'
{"running":true,"configuration":{"port":4110}}
JSON
        }
        testing::ui_smoke::_check_iframe_bridge() {
            cat <<'JSON'
{"dependency_present":true}
JSON
        }
        TESTING_PHASE_SCENARIO_NAME="stale"
        TESTING_PHASE_SCENARIO_DIR="$scenario_dir"
        testing::ui_smoke::run --scenario stale --scenario-dir "$scenario_dir" --json
    '
    assert_failure
    if [[ "$status" -ne 60 ]]; then
        echo "expected exit code 60, got $status" >&2
        return 1
    fi
    assert_output_contains '"status": "blocked"'
    assert_output_contains '"message": "stale bundle"'
}

@test "ui-smoke fails when iframe bridge dependency missing" {
    run env APP_ROOT="$APP_ROOT" TEST_SCENARIO_DIR="$TEST_SCENARIO_DIR" bash -c '
        set -euo pipefail
        source "$APP_ROOT/scripts/scenarios/testing/shell/ui-smoke.sh"
        scenario_dir="$TEST_SCENARIO_DIR/no-bridge"
        mkdir -p "$scenario_dir/ui"
        cat <<JSON > "$scenario_dir/ui/package.json"
{
  "dependencies": {}
}
JSON
        testing::runtime::ensure_available() { return 0; }
        testing::runtime::discover_ports() { export UI_PORT=18001; }
        testing::runtime::cleanup() { :; }
        testing::ui_smoke::_check_bundle_freshness() {
            cat <<'JSON'
{"fresh":true}
JSON
        }
        testing::ui_smoke::_browserless_status() {
            cat <<'JSON'
{"running":true,"configuration":{"port":4110}}
JSON
        }
        testing::ui_smoke::_check_iframe_bridge() {
            cat <<'JSON'
{"dependency_present":false,"details":"missing"}
JSON
        }
        TESTING_PHASE_SCENARIO_NAME="no-bridge"
        TESTING_PHASE_SCENARIO_DIR="$scenario_dir"
        testing::ui_smoke::run --scenario no-bridge --scenario-dir "$scenario_dir" --json
    '
    assert_failure
    assert_output_contains '"status": "failed"'
    assert_output_contains '@vrooli/iframe-bridge dependency missing'
}

@test "ui-smoke errors when service exposes UI_PORT but port missing" {
    run env APP_ROOT="$APP_ROOT" TEST_SCENARIO_DIR="$TEST_SCENARIO_DIR" bash -c '
        set -euo pipefail
        source "$APP_ROOT/scripts/scenarios/testing/shell/ui-smoke.sh"
        scenario_dir="$TEST_SCENARIO_DIR/ui-missing-port"
        mkdir -p "$scenario_dir/ui" "$scenario_dir/.vrooli"
        cat <<JSON > "$scenario_dir/ui/package.json"
{
  "dependencies": {
    "@vrooli/iframe-bridge": "0.0.0-test"
  }
}
JSON
        cat <<JSON > "$scenario_dir/.vrooli/service.json"
{
  "ports": {
    "ui": {
      "env_var": "UI_PORT"
    }
  }
}
JSON
        testing::runtime::ensure_available() { return 0; }
        testing::runtime::discover_ports() { :; }
        testing::runtime::cleanup() { :; }
        testing::ui_smoke::_check_bundle_freshness() {
            cat <<'JSON'
{"fresh":true}
JSON
        }
        testing::ui_smoke::_browserless_status() {
            cat <<'JSON'
{"running":true,"configuration":{"port":4110}}
JSON
        }
        testing::ui_smoke::_check_iframe_bridge() {
            cat <<'JSON'
{"dependency_present":true}
JSON
        }
        TESTING_PHASE_SCENARIO_NAME="ui-missing-port"
        TESTING_PHASE_SCENARIO_DIR="$scenario_dir"
        testing::ui_smoke::run --scenario ui-missing-port --scenario-dir "$scenario_dir" --json
    '
    assert_failure
    assert_output_contains '"status": "failed"'
    assert_output_contains "UI_PORT defined"
}
