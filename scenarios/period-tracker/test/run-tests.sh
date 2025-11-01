#!/bin/bash
# Phased test orchestrator for period-tracker scenario
set -euo pipefail

TEST_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCENARIO_DIR="$(cd "$TEST_DIR/.." && pwd)"
APP_ROOT="${APP_ROOT:-$(builtin cd "${SCENARIO_DIR}/../.." && builtin pwd)}"

source "${APP_ROOT}/scripts/lib/utils/log.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/runner.sh"

testing::runner::init \
  --scenario-name "period-tracker" \
  --scenario-dir "$SCENARIO_DIR" \
  --test-dir "$TEST_DIR" \
  --log-dir "$TEST_DIR/artifacts" \
  --default-manage-runtime true

testing::runner::register_phase --name structure --script "$TEST_DIR/phases/test-structure.sh" --timeout 30

testing::runner::register_phase --name dependencies --script "$TEST_DIR/phases/test-dependencies.sh" --timeout 60

testing::runner::register_phase --name unit --script "$TEST_DIR/phases/test-unit.sh" --timeout 120

testing::runner::register_phase --name integration --script "$TEST_DIR/phases/test-integration.sh" --timeout 180 --requires-runtime true

testing::runner::register_phase --name business --script "$TEST_DIR/phases/test-business.sh" --timeout 120

testing::runner::register_phase --name performance --script "$TEST_DIR/phases/test-performance.sh" --timeout 60 --requires-runtime true --optional true

testing::runner::define_preset quick "structure dependencies unit"
testing::runner::define_preset smoke "structure integration"
testing::runner::define_preset comprehensive "structure dependencies unit integration business performance"

testing::runner::execute "$@"
