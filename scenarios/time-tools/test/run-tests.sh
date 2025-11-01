#!/bin/bash
# Phased test orchestrator for time-tools scenario
set -euo pipefail

TEST_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCENARIO_DIR="$(cd "$TEST_DIR/.." && pwd)"
APP_ROOT="${APP_ROOT:-$(builtin cd "${SCENARIO_DIR}/../.." && builtin pwd)}"

source "${APP_ROOT}/scripts/lib/utils/log.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/runner.sh"

# Initialise shared runner and allow runtime auto-management so integration-style
# phases can assume the scenario is available when they need it.
testing::runner::init \
  --scenario-name "time-tools" \
  --scenario-dir "$SCENARIO_DIR" \
  --test-dir "$TEST_DIR" \
  --log-dir "$TEST_DIR/artifacts" \
  --default-manage-runtime true

PHASE_DIR="$TEST_DIR/phases"

# Core phase lineup
testing::runner::register_phase --name structure --script "$PHASE_DIR/test-structure.sh" --timeout 30

testing::runner::register_phase --name dependencies --script "$PHASE_DIR/test-dependencies.sh" --timeout 60

testing::runner::register_phase --name unit --script "$PHASE_DIR/test-unit.sh" --timeout 120

testing::runner::register_phase --name integration --script "$PHASE_DIR/test-integration.sh" --timeout 240 --requires-runtime true

testing::runner::register_phase --name business --script "$PHASE_DIR/test-business.sh" --timeout 180 --requires-runtime true

testing::runner::register_phase --name performance --script "$PHASE_DIR/test-performance.sh" --timeout 60 --requires-runtime true

# Helpful presets for local iteration
testing::runner::define_preset quick "structure unit"
testing::runner::define_preset smoke "structure dependencies integration"
testing::runner::define_preset comprehensive "structure dependencies unit integration business performance"

testing::runner::execute "$@"
