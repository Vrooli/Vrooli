#!/bin/bash
set -euo pipefail

TEST_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCENARIO_DIR="$(cd "$TEST_DIR/.." && pwd)"
APP_ROOT="${APP_ROOT:-$(builtin cd "${SCENARIO_DIR}/../.." && builtin pwd)}"

source "${APP_ROOT}/scripts/lib/utils/log.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/runner.sh"

testing::runner::init \
  --scenario-name "maintenance-orchestrator" \
  --scenario-dir "$SCENARIO_DIR" \
  --test-dir "$TEST_DIR" \
  --log-dir "$TEST_DIR/artifacts"

testing::runner::register_phase --name structure --script "$TEST_DIR/phases/test-structure.sh" --timeout 15
testing::runner::register_phase --name dependencies --script "$TEST_DIR/phases/test-dependencies.sh" --timeout 30
testing::runner::register_phase --name unit --script "$TEST_DIR/phases/test-unit.sh" --timeout 60
testing::runner::register_phase --name integration --script "$TEST_DIR/phases/test-integration.sh" --timeout 120 --requires-runtime true
testing::runner::register_phase --name business --script "$TEST_DIR/phases/test-business.sh" --timeout 180 --requires-runtime true
testing::runner::register_phase --name performance --script "$TEST_DIR/phases/test-performance.sh" --timeout 60 --requires-runtime true

testing::runner::register_test_type --name go --handler "$TEST_DIR/unit/run-unit-tests.sh --go-only" --kind command

testing::runner::define_preset quick "structure unit"
testing::runner::define_preset smoke "structure integration"
testing::runner::define_preset comprehensive "structure dependencies unit integration business performance"

testing::runner::execute "$@"
