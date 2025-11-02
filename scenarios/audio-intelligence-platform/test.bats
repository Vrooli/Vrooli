#!/usr/bin/env bats
# Tests for phased testing configuration in audio-intelligence-platform

bats_require_minimum_version 1.5.0

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../.." && builtin pwd)}"
SCENARIO_DIR="${APP_ROOT}/scenarios/audio-intelligence-platform"
TEST_DIR="${SCENARIO_DIR}/test"

# shellcheck disable=SC1091
source "${APP_ROOT}/lib/utils/var.sh" 2>/dev/null || true
# shellcheck disable=SC1091
load "${APP_ROOT}/__test/fixtures/setup.bash"

setup() {
  vrooli_setup_unit_test
}

teardown() {
  vrooli_cleanup_test
}

@test "run-tests.sh uses shared runner" {
  run grep -q 'testing::runner::init' "$TEST_DIR/run-tests.sh"
  assert_success
}

@test "phase scripts use phase helpers" {
  for phase in test-structure.sh test-dependencies.sh test-unit.sh test-integration.sh test-business.sh test-performance.sh; do
    run grep -q 'testing::phase::init' "$TEST_DIR/phases/$phase"
    assert_success
  done
}

@test "service.json points to phased runner" {
  run jq -r '.test.steps[0].run' "$SCENARIO_DIR/.vrooli/service.json"
  assert_success
  assert_output "test/run-tests.sh"
}

@test "legacy scenario-test.yaml removed" {
  [ ! -f "$SCENARIO_DIR/scenario-test.yaml" ]
}
