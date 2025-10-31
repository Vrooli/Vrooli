#!/bin/bash
set -euo pipefail

TEST_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCENARIO_DIR="$(cd "$TEST_DIR/.." && pwd)"
APP_ROOT="${APP_ROOT:-$(builtin cd "${SCENARIO_DIR}/../.." && builtin pwd)}"

source "${APP_ROOT}/scripts/lib/utils/log.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/runner.sh"

# Initialize shared runner with runtime management enabled so integration phases
# can assume the scenario is available.
testing::runner::init \
  --scenario-name "browser-automation-studio" \
  --scenario-dir "$SCENARIO_DIR" \
  --test-dir "$TEST_DIR" \
  --log-dir "$TEST_DIR/artifacts" \
  --default-manage-runtime true

# Register standard phase lineup. Timeouts mirror historical expectations while
# enabling runtime-aware phases to opt in via --requires-runtime.
testing::runner::register_phase --name structure --script "$TEST_DIR/phases/test-structure.sh" --timeout 30

testing::runner::register_phase --name dependencies --script "$TEST_DIR/phases/test-dependencies.sh" --timeout 60

testing::runner::register_phase --name unit --script "$TEST_DIR/phases/test-unit.sh" --timeout 120

testing::runner::register_phase --name integration --script "$TEST_DIR/phases/test-integration.sh" --timeout 240 --requires-runtime true

testing::runner::register_phase --name business --script "$TEST_DIR/phases/test-business.sh" --timeout 120

testing::runner::register_phase --name performance --script "$TEST_DIR/phases/test-performance.sh" --timeout 60 --requires-runtime true

# Presets for quick iteration.
testing::runner::define_preset quick "structure unit"
testing::runner::define_preset smoke "structure integration"
testing::runner::define_preset comprehensive "structure dependencies unit integration business performance"

testing::runner::execute "$@"
