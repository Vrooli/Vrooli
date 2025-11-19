#!/usr/bin/env bats
# Regression tests for ui-smoke helper utilities

APP_ROOT="${APP_ROOT:-$(builtin cd "$(dirname "${BATS_TEST_FILENAME}")/../../../.." && builtin pwd)}"
# shellcheck disable=SC1091
source "${APP_ROOT}/__test/fixtures/setup.bash"

setup() {
    vrooli_setup_unit_test
    export TEST_SCENARIO_DIR="$VROOLI_TEST_TMPDIR/ui-smoke-scenario"
    mkdir -p "$TEST_SCENARIO_DIR"
}

teardown() {
    vrooli_cleanup_test
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
