#!/bin/bash
# Shared phased test orchestrator for resume-screening-assistant
set -euo pipefail

TEST_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCENARIO_DIR="$(cd "$TEST_DIR/.." && pwd)"
APP_ROOT="${APP_ROOT:-$(builtin cd "${SCENARIO_DIR}/../.." && builtin pwd)}"

source "${APP_ROOT}/scripts/lib/utils/log.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/runner.sh"

PHASES_DIR="$TEST_DIR/phases"
ARTIFACT_DIR="$TEST_DIR/artifacts"

# Initialise shared runner with lifecycle management so runtime phases can
# assume the scenario is started automatically.
testing::runner::init \
  --scenario-name "resume-screening-assistant" \
  --scenario-dir "$SCENARIO_DIR" \
  --test-dir "$TEST_DIR" \
  --log-dir "$ARTIFACT_DIR" \
  --default-manage-runtime true

# Standard six-phase registration mirroring project expectations.
testing::runner::register_phase --name structure --script "$PHASES_DIR/test-structure.sh" --timeout 45 --display "phase-structure"
testing::runner::register_phase --name dependencies --script "$PHASES_DIR/test-dependencies.sh" --timeout 120 --display "phase-dependencies"
testing::runner::register_phase --name unit --script "$PHASES_DIR/test-unit.sh" --timeout 120 --display "phase-unit"
testing::runner::register_phase --name integration --script "$PHASES_DIR/test-integration.sh" --timeout 240 --requires-runtime true --display "phase-integration"
testing::runner::register_phase --name business --script "$PHASES_DIR/test-business.sh" --timeout 180 --requires-runtime true --display "phase-business"
testing::runner::register_phase --name performance --script "$PHASES_DIR/test-performance.sh" --timeout 120 --display "phase-performance"

# Handy presets for iterative workflows.
testing::runner::define_preset quick "structure unit"
testing::runner::define_preset smoke "structure dependencies integration"
testing::runner::define_preset comprehensive "structure dependencies unit integration business performance"

testing::runner::execute "$@"
