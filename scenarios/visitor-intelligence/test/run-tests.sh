#!/bin/bash
set -euo pipefail

TEST_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCENARIO_DIR="$(cd "$TEST_DIR/.." && pwd)"
APP_ROOT="${APP_ROOT:-$(builtin cd "${SCENARIO_DIR}/../.." && builtin pwd)}"

source "${APP_ROOT}/scripts/lib/utils/log.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/runner.sh"

# Enable managed runtime so integration/business/performance phases can assume the
# scenario is available. This also ensures coverage artifacts share a consistent home.
testing::runner::init \
  --scenario-name "visitor-intelligence" \
  --scenario-dir "$SCENARIO_DIR" \
  --test-dir "$TEST_DIR" \
  --log-dir "$TEST_DIR/artifacts" \
  --default-manage-runtime true

PHASES_DIR="$TEST_DIR/phases"

# Phase registrations
testing::runner::register_phase --name structure --script "$PHASES_DIR/test-structure.sh" --timeout 45

testing::runner::register_phase --name dependencies --script "$PHASES_DIR/test-dependencies.sh" --timeout 90

testing::runner::register_phase --name unit --script "$PHASES_DIR/test-unit.sh" --timeout 150

testing::runner::register_phase --name integration --script "$PHASES_DIR/test-integration.sh" --timeout 300 --requires-runtime true

testing::runner::register_phase --name business --script "$PHASES_DIR/test-business.sh" --timeout 240 --requires-runtime true

testing::runner::register_phase --name performance --script "$PHASES_DIR/test-performance.sh" --timeout 180 --requires-runtime true

# Helpful presets
testing::runner::define_preset quick "structure unit"
testing::runner::define_preset smoke "structure dependencies integration"
testing::runner::define_preset comprehensive "structure dependencies unit integration business performance"

testing::runner::execute "$@"
