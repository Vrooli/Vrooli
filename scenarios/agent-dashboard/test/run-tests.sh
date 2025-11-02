#!/bin/bash
# Shared phased test orchestrator for the Agent Dashboard scenario
set -euo pipefail

TEST_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCENARIO_DIR="$(cd "$TEST_DIR/.." && pwd)"
APP_ROOT="${APP_ROOT:-$(builtin cd "${SCENARIO_DIR}/../.." && builtin pwd)}"

source "${APP_ROOT}/scripts/lib/utils/log.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/runner.sh"

PHASES_DIR="$TEST_DIR/phases"
ARTIFACT_DIR="$TEST_DIR/artifacts"

# Initialise shared runner so runtime-aware phases can assume lifecycle management
# and we gain access to caching, parallelism, and standardised logging.
testing::runner::init \
  --scenario-name "agent-dashboard" \
  --scenario-dir "$SCENARIO_DIR" \
  --test-dir "$TEST_DIR" \
  --log-dir "$ARTIFACT_DIR" \
  --default-manage-runtime true

# Register core phase lineup
testing::runner::register_phase --name structure --script "$PHASES_DIR/test-structure.sh" --timeout 30 --display "phase-structure"

testing::runner::register_phase --name dependencies --script "$PHASES_DIR/test-dependencies.sh" --timeout 90 --display "phase-dependencies"

testing::runner::register_phase --name unit --script "$PHASES_DIR/test-unit.sh" --timeout 120 --display "phase-unit"

testing::runner::register_phase --name integration --script "$PHASES_DIR/test-integration.sh" --timeout 240 --requires-runtime true --display "phase-integration"

testing::runner::register_phase --name business --script "$PHASES_DIR/test-business.sh" --timeout 180 --requires-runtime true --display "phase-business"

testing::runner::register_phase --name performance --script "$PHASES_DIR/test-performance.sh" --timeout 120 --display "phase-performance"

# Presets for targeted execution
testing::runner::define_preset quick "structure unit"
testing::runner::define_preset smoke "structure dependencies integration"
testing::runner::define_preset comprehensive "structure dependencies unit integration business performance"

testing::runner::execute "$@"
