#!/bin/bash
set -euo pipefail

TEST_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCENARIO_DIR="$(cd "$TEST_DIR/.." && pwd)"
APP_ROOT="${APP_ROOT:-$(builtin cd "${SCENARIO_DIR}/../.." && builtin pwd)}"

source "${APP_ROOT}/scripts/lib/utils/log.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/runner.sh"

testing::runner::init \
  --scenario-name "job-to-scenario-pipeline" \
  --scenario-dir "$SCENARIO_DIR" \
  --test-dir "$TEST_DIR" \
  --log-dir "$TEST_DIR/artifacts"

PHASES_DIR="$TEST_DIR/phases"

testing::runner::register_phase --name structure --script "$PHASES_DIR/test-structure.sh" --timeout 30
testing::runner::register_phase --name dependencies --script "$PHASES_DIR/test-dependencies.sh" --timeout 60
testing::runner::register_phase --name unit --script "$PHASES_DIR/test-unit.sh" --timeout 120
testing::runner::register_phase --name integration --script "$PHASES_DIR/test-integration.sh" --timeout 180
testing::runner::register_phase --name business --script "$PHASES_DIR/test-business.sh" --timeout 120
testing::runner::register_phase --name performance --script "$PHASES_DIR/test-performance.sh" --timeout 60

testing::runner::define_preset quick "structure unit"
testing::runner::define_preset smoke "structure dependencies"
testing::runner::define_preset comprehensive "structure dependencies unit integration business performance"

testing::runner::execute "$@"
