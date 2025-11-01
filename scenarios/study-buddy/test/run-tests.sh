#!/bin/bash
# Shared phased test orchestrator for the study-buddy scenario
set -euo pipefail

TEST_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCENARIO_DIR="$(cd "$TEST_DIR/.." && pwd)"
APP_ROOT="${APP_ROOT:-$(builtin cd "${SCENARIO_DIR}/../.." && builtin pwd)}"

source "${APP_ROOT}/scripts/lib/utils/log.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/runner.sh"

# Initialise runner with managed runtime so integration/business phases can rely on the services
# being available without bespoke setup in each script.
testing::runner::init \
  --scenario-name "study-buddy" \
  --scenario-dir "$SCENARIO_DIR" \
  --test-dir "$TEST_DIR" \
  --log-dir "$TEST_DIR/artifacts" \
  --default-manage-runtime true

# Phase registration matches the standard six-phase cadence. Timeouts reflect the historical
# behaviour of the old scripts while providing enough headroom for API/CLI checks.
testing::runner::register_phase --name structure     --script "$TEST_DIR/phases/test-structure.sh"     --timeout 30

testing::runner::register_phase --name dependencies --script "$TEST_DIR/phases/test-dependencies.sh" --timeout 60

testing::runner::register_phase --name unit         --script "$TEST_DIR/phases/test-unit.sh"          --timeout 120

testing::runner::register_phase --name integration  --script "$TEST_DIR/phases/test-integration.sh"   --timeout 180 --requires-runtime true

testing::runner::register_phase --name business     --script "$TEST_DIR/phases/test-business.sh"      --timeout 150 --requires-runtime true

testing::runner::register_phase --name performance  --script "$TEST_DIR/phases/test-performance.sh"   --timeout 60  --requires-runtime true

# Handy presets for day-to-day iteration.
testing::runner::define_preset quick "structure unit"
testing::runner::define_preset smoke "structure dependencies integration"
testing::runner::define_preset comprehensive "structure dependencies unit integration business performance"

testing::runner::execute "$@"
