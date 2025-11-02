#!/bin/bash
set -euo pipefail

TEST_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCENARIO_DIR="$(cd "$TEST_DIR/.." && pwd)"
APP_ROOT="${APP_ROOT:-$(builtin cd "${SCENARIO_DIR}/../.." && builtin pwd)}"

source "${APP_ROOT}/scripts/lib/utils/log.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/runner.sh"

ARTIFACT_DIR="$TEST_DIR/artifacts"
PHASES_DIR="$TEST_DIR/phases"

# Enable managed runtime so phases that require the scenario can rely on the lifecycle system.
testing::runner::init \
  --scenario-name "test-genie" \
  --scenario-dir "$SCENARIO_DIR" \
  --test-dir "$TEST_DIR" \
  --log-dir "$ARTIFACT_DIR" \
  --default-manage-runtime true

# Phases follow the standard structure → integration → business cadence, with CLI checks isolated.
testing::runner::register_phase --name structure --script "$PHASES_DIR/test-structure.sh" --timeout 45

testing::runner::register_phase --name dependencies --script "$PHASES_DIR/test-dependencies.sh" --timeout 120

testing::runner::register_phase --name unit --script "$PHASES_DIR/test-unit.sh" --timeout 180

testing::runner::register_phase --name cli --script "$PHASES_DIR/test-cli.sh" --timeout 120

testing::runner::register_phase --name integration --script "$PHASES_DIR/test-integration.sh" --timeout 300 --requires-runtime true

testing::runner::register_phase --name business --script "$PHASES_DIR/test-business.sh" --timeout 300 --requires-runtime true

# Quick iteration presets
QUICK_PHASES="structure unit"
SMOKE_PHASES="structure dependencies cli"
COMPREHENSIVE_PHASES="structure dependencies unit cli integration business"

testing::runner::define_preset quick "$QUICK_PHASES"
testing::runner::define_preset smoke "$SMOKE_PHASES"
testing::runner::define_preset comprehensive "$COMPREHENSIVE_PHASES"

testing::runner::execute "$@"
