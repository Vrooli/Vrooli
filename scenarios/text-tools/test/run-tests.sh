#!/bin/bash
# Shared phased test orchestrator for the text-tools scenario
set -euo pipefail

TEST_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCENARIO_DIR="$(cd "$TEST_DIR/.." && pwd)"
APP_ROOT="${APP_ROOT:-$(builtin cd "${SCENARIO_DIR}/../.." && builtin pwd)}"

source "${APP_ROOT}/scripts/lib/utils/log.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/runner.sh"

SCENARIO_NAME="text-tools"
LOG_DIR="$TEST_DIR/artifacts"
mkdir -p "$LOG_DIR"

# Initialise shared runner; allow runtime-dependent phases to auto-manage the scenario
testing::runner::init \
  --scenario-name "$SCENARIO_NAME" \
  --scenario-dir "$SCENARIO_DIR" \
  --test-dir "$TEST_DIR" \
  --log-dir "$LOG_DIR" \
  --default-manage-runtime true

PHASES_DIR="$TEST_DIR/phases"

# Standard phase lineup
testing::runner::register_phase \
  --name structure \
  --script "$PHASES_DIR/test-structure.sh" \
  --timeout 30

testing::runner::register_phase \
  --name dependencies \
  --script "$PHASES_DIR/test-dependencies.sh" \
  --timeout 60

testing::runner::register_phase \
  --name unit \
  --script "$PHASES_DIR/test-unit.sh" \
  --timeout 120

testing::runner::register_phase \
  --name integration \
  --script "$PHASES_DIR/test-integration.sh" \
  --timeout 180 \
  --requires-runtime true

testing::runner::register_phase \
  --name business \
  --script "$PHASES_DIR/test-business.sh" \
  --timeout 180 \
  --requires-runtime true

testing::runner::register_phase \
  --name performance \
  --script "$PHASES_DIR/test-performance.sh" \
  --timeout 120 \
  --requires-runtime true

# CLI test type (BATS)
CLI_DIR="$TEST_DIR/cli"
testing::runner::register_test_type \
  --name cli \
  --handler "$CLI_DIR/run-cli-tests.sh" \
  --kind script \
  --display "cli-bats"

# Presets for quick iteration
testing::runner::define_preset quick "structure unit"
testing::runner::define_preset smoke "structure dependencies integration"
testing::runner::define_preset comprehensive "structure dependencies unit integration business performance"

# Execute requested phases/test types
testing::runner::execute "$@"
