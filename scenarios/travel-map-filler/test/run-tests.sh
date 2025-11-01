#!/bin/bash
# Shared test orchestrator for travel-map-filler scenario.

set -euo pipefail

TEST_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCENARIO_DIR="$(cd "$TEST_DIR/.." && pwd)"
APP_ROOT="${APP_ROOT:-$(builtin cd "${SCENARIO_DIR}/../.." && builtin pwd)}"

source "${APP_ROOT}/scripts/lib/utils/log.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/runner.sh"

SCENARIO_NAME="travel-map-filler"
LOG_DIR="$TEST_DIR/artifacts"

mkdir -p "$LOG_DIR"

# Initialise shared runner (tests do not require auto-managed runtime yet).
testing::runner::init \
  --scenario-name "$SCENARIO_NAME" \
  --scenario-dir "$SCENARIO_DIR" \
  --test-dir "$TEST_DIR" \
  --log-dir "$LOG_DIR" \
  --default-manage-runtime false

PHASES_DIR="$TEST_DIR/phases"

# Phase registrations.
testing::runner::register_phase --name structure --script "$PHASES_DIR/test-structure.sh" --timeout 30

testing::runner::register_phase --name dependencies --script "$PHASES_DIR/test-dependencies.sh" --timeout 60

testing::runner::register_phase --name unit --script "$PHASES_DIR/test-unit.sh" --timeout 120

testing::runner::register_phase --name integration --script "$PHASES_DIR/test-integration.sh" --timeout 150

testing::runner::register_phase --name business --script "$PHASES_DIR/test-business.sh" --timeout 120

testing::runner::register_phase --name performance --script "$PHASES_DIR/test-performance.sh" --timeout 90

# Presets for faster feedback loops.
testing::runner::define_preset quick "structure dependencies unit"
testing::runner::define_preset smoke "structure unit"
testing::runner::define_preset full "structure dependencies unit integration business performance"

testing::runner::execute "$@"
