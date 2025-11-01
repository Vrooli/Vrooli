#!/bin/bash
# Shared phased test runner wrapper
set -euo pipefail

TEST_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCENARIO_DIR="$(cd "$TEST_DIR/.." && pwd)"
APP_ROOT="${APP_ROOT:-$(builtin cd "${SCENARIO_DIR}/../.." && builtin pwd)}"

source "${APP_ROOT}/scripts/lib/utils/log.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/runner.sh"

TESTING_LOG_DIR="$TEST_DIR/artifacts"

testing::runner::init \
  --scenario-name "referral-program-generator" \
  --scenario-dir "$SCENARIO_DIR" \
  --test-dir "$TEST_DIR" \
  --log-dir "$TESTING_LOG_DIR" \
  --default-manage-runtime false

PHASE_DIR="$TEST_DIR/phases"

testing::runner::register_phase --name structure --script "$PHASE_DIR/test-structure.sh" --timeout 30

testing::runner::register_phase --name dependencies --script "$PHASE_DIR/test-dependencies.sh" --timeout 60

testing::runner::register_phase --name unit --script "$PHASE_DIR/test-unit.sh" --timeout 90

testing::runner::register_phase --name integration --script "$PHASE_DIR/test-integration.sh" --timeout 150 --requires-runtime false

testing::runner::register_phase --name business --script "$PHASE_DIR/test-business.sh" --timeout 120

testing::runner::register_phase --name performance --script "$PHASE_DIR/test-performance.sh" --timeout 60 --optional true

# Register unit test type (Go only for now) so focused runs can reuse the phase logic
testing::runner::register_test_type \
  --name go \
  --handler "$PHASE_DIR/test-unit.sh" \
  --kind script \
  --display "unit-go"

# Presets to speed up iteration
testing::runner::define_preset quick "structure unit"
testing::runner::define_preset smoke "structure dependencies"
testing::runner::define_preset core "structure dependencies unit integration"

testing::runner::execute "$@"
