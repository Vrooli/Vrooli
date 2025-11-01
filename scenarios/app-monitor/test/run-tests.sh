#!/bin/bash
# Shared phased test orchestrator for app-monitor scenario
set -euo pipefail

TEST_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCENARIO_DIR="$(cd "$TEST_DIR/.." && pwd)"
APP_ROOT="${APP_ROOT:-$(builtin cd "${SCENARIO_DIR}/../.." && builtin pwd)}"

source "${APP_ROOT}/scripts/lib/utils/log.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/runner.sh"

SCENARIO_NAME="app-monitor"
LOG_DIR="$TEST_DIR/artifacts"
mkdir -p "$LOG_DIR"

# Initialise shared runner with runtime management so integration/business tests can auto-start the scenario when requested.
testing::runner::init \
  --scenario-name "$SCENARIO_NAME" \
  --scenario-dir "$SCENARIO_DIR" \
  --test-dir "$TEST_DIR" \
  --log-dir "$LOG_DIR" \
  --default-manage-runtime true

PHASES_DIR="$TEST_DIR/phases"

testing::runner::register_phase \
  --name "structure" \
  --script "$PHASES_DIR/test-structure.sh" \
  --timeout 30 \
  --display "phase-structure"

testing::runner::register_phase \
  --name "dependencies" \
  --script "$PHASES_DIR/test-dependencies.sh" \
  --timeout 90 \
  --display "phase-dependencies"

testing::runner::register_phase \
  --name "unit" \
  --script "$PHASES_DIR/test-unit.sh" \
  --timeout 180 \
  --display "phase-unit"

testing::runner::register_phase \
  --name "integration" \
  --script "$PHASES_DIR/test-integration.sh" \
  --timeout 240 \
  --requires-runtime true \
  --display "phase-integration"

testing::runner::register_phase \
  --name "business" \
  --script "$PHASES_DIR/test-business.sh" \
  --timeout 180 \
  --requires-runtime true \
  --display "phase-business"

testing::runner::register_phase \
  --name "performance" \
  --script "$PHASES_DIR/test-performance.sh" \
  --timeout 90 \
  --requires-runtime true \
  --display "phase-performance"

# Presets for common workflows.
testing::runner::define_preset quick "structure unit"
testing::runner::define_preset smoke "structure integration"
testing::runner::define_preset comprehensive "structure dependencies unit integration business performance"

testing::runner::execute "$@"
