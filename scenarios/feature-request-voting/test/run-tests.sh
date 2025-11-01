#!/bin/bash
# Shared phased test runner for feature-request-voting scenario
set -euo pipefail

TEST_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCENARIO_DIR="$(cd "$TEST_DIR/.." && pwd)"
APP_ROOT="${APP_ROOT:-$(builtin cd "${SCENARIO_DIR}/../.." && builtin pwd)}"

source "${APP_ROOT}/scripts/lib/utils/log.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/runner.sh"

# Initialise shared runner
testing::runner::init \
  --scenario-name "feature-request-voting" \
  --scenario-dir "$SCENARIO_DIR" \
  --test-dir "$TEST_DIR" \
  --log-dir "$TEST_DIR/artifacts"

PHASE_DIR="$TEST_DIR/phases"

# Phase registration
testing::runner::register_phase --name structure     --script "$PHASE_DIR/test-structure.sh"     --timeout 30
testing::runner::register_phase --name dependencies  --script "$PHASE_DIR/test-dependencies.sh"  --timeout 60
testing::runner::register_phase --name unit          --script "$PHASE_DIR/test-unit.sh"          --timeout 150
testing::runner::register_phase --name integration   --script "$PHASE_DIR/test-integration.sh"   --timeout 180
testing::runner::register_phase --name business      --script "$PHASE_DIR/test-business.sh"      --timeout 150
testing::runner::register_phase --name performance   --script "$PHASE_DIR/test-performance.sh"   --timeout 180

# Presets for quick feedback
testing::runner::define_preset quick "structure unit"
testing::runner::define_preset smoke "structure dependencies integration"
testing::runner::define_preset comprehensive "structure dependencies unit integration business performance"

testing::runner::execute "$@"
