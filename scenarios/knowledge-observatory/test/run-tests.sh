#!/bin/bash
# Knowledge Observatory phased test orchestrator
set -euo pipefail

TEST_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCENARIO_DIR="$(cd "$TEST_DIR/.." && pwd)"
APP_ROOT="${APP_ROOT:-$(builtin cd "${SCENARIO_DIR}/../.." && builtin pwd)}"

source "${APP_ROOT}/scripts/lib/utils/log.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/runner.sh"

LOG_DIR="$TEST_DIR/artifacts"
mkdir -p "$LOG_DIR"

testing::runner::init \
  --scenario-name "knowledge-observatory" \
  --scenario-dir "$SCENARIO_DIR" \
  --test-dir "$TEST_DIR" \
  --log-dir "$LOG_DIR" \
  --default-manage-runtime true

PHASES_DIR="$TEST_DIR/phases"

testing::runner::register_phase --name structure --script "$PHASES_DIR/test-structure.sh" --timeout 45

testing::runner::register_phase --name docs --script "$PHASES_DIR/test-docs.sh" --timeout 45

testing::runner::register_phase --name dependencies --script "$PHASES_DIR/test-dependencies.sh" --timeout 90

testing::runner::register_phase --name unit --script "$PHASES_DIR/test-unit.sh" --timeout 120

testing::runner::register_phase --name integration --script "$PHASES_DIR/test-integration.sh" --timeout 180 --requires-runtime true

testing::runner::register_phase --name business --script "$PHASES_DIR/test-business.sh" --timeout 180 --requires-runtime true

testing::runner::register_phase --name performance --script "$PHASES_DIR/test-performance.sh" --timeout 180 --requires-runtime true

# Quick feedback presets
testing::runner::define_preset quick "structure docs unit"
testing::runner::define_preset smoke "structure integration"
testing::runner::define_preset comprehensive "structure docs dependencies unit integration business performance"

testing::runner::execute "$@"
