#!/bin/bash
# Calendar scenario phased test orchestrator
set -euo pipefail

TEST_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCENARIO_DIR="$(cd "$TEST_DIR/.." && pwd)"
APP_ROOT="${APP_ROOT:-$(builtin cd "${SCENARIO_DIR}/../.." && builtin pwd)}"

source "${APP_ROOT}/scripts/lib/utils/log.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/runner.sh"

SCENARIO_NAME="calendar"
LOG_DIR="$TEST_DIR/artifacts"

# Initialise shared runner. Allow runtime management so integration/business
# phases can auto-start the scenario when needed.
testing::runner::init \
  --scenario-name "$SCENARIO_NAME" \
  --scenario-dir "$SCENARIO_DIR" \
  --test-dir "$TEST_DIR" \
  --log-dir "$LOG_DIR" \
  --default-manage-runtime true

# Phase registrations
PHASE_DIR="$TEST_DIR/phases"
testing::runner::register_phase --name structure --script "$PHASE_DIR/test-structure.sh" --timeout 60

testing::runner::register_phase --name dependencies --script "$PHASE_DIR/test-dependencies.sh" --timeout 120

testing::runner::register_phase --name unit --script "$PHASE_DIR/test-unit.sh" --timeout 240

testing::runner::register_phase --name integration --script "$PHASE_DIR/test-integration.sh" --timeout 300 --requires-runtime true

testing::runner::register_phase --name business --script "$PHASE_DIR/test-business.sh" --timeout 360 --requires-runtime true

testing::runner::register_phase --name performance --script "$PHASE_DIR/test-performance.sh" --timeout 120 --requires-runtime true --optional true

# Presets for quicker execution during development
testing::runner::define_preset quick "structure unit"
testing::runner::define_preset smoke "structure integration"
testing::runner::define_preset comprehensive "structure dependencies unit integration business performance"

testing::runner::execute "$@"
