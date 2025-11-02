#!/bin/bash
# Phased test orchestrator for image-generation-pipeline scenario
set -euo pipefail

TEST_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCENARIO_DIR="$(cd "$TEST_DIR/.." && pwd)"
APP_ROOT="${APP_ROOT:-$(builtin cd "${SCENARIO_DIR}/../.." && builtin pwd)}"

source "${APP_ROOT}/scripts/lib/utils/log.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/runner.sh"

ARTIFACT_DIR="$TEST_DIR/artifacts"
mkdir -p "$ARTIFACT_DIR"

# Initialise shared runner so that runtime-dependent phases can assume
# lifecycle management and produce consistent artifacts.
testing::runner::init \
  --scenario-name "image-generation-pipeline" \
  --scenario-dir "$SCENARIO_DIR" \
  --test-dir "$TEST_DIR" \
  --log-dir "$ARTIFACT_DIR" \
  --default-manage-runtime true

PHASES_DIR="$TEST_DIR/phases"

testing::runner::register_phase --name structure --script "$PHASES_DIR/test-structure.sh" --timeout 45 --display "phase-structure"
testing::runner::register_phase --name dependencies --script "$PHASES_DIR/test-dependencies.sh" --timeout 120 --display "phase-dependencies"
testing::runner::register_phase --name unit --script "$PHASES_DIR/test-unit.sh" --timeout 120 --display "phase-unit"
testing::runner::register_phase --name integration --script "$PHASES_DIR/test-integration.sh" --timeout 300 --requires-runtime true --display "phase-integration"
testing::runner::register_phase --name business --script "$PHASES_DIR/test-business.sh" --timeout 240 --requires-runtime true --display "phase-business"
testing::runner::register_phase --name performance --script "$PHASES_DIR/test-performance.sh" --timeout 90 --display "phase-performance"

# Quick presets for developer workflows
testing::runner::define_preset quick "structure unit"
testing::runner::define_preset smoke "structure dependencies integration"
testing::runner::define_preset comprehensive "structure dependencies unit integration business performance"

testing::runner::execute "$@"
