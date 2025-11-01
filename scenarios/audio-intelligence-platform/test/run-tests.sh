#!/bin/bash
set -euo pipefail

TEST_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCENARIO_DIR="$(cd "$TEST_DIR/.." && pwd)"
APP_ROOT="${APP_ROOT:-$(builtin cd "${SCENARIO_DIR}/../.." && builtin pwd)}"

source "${APP_ROOT}/scripts/lib/utils/log.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/runner.sh"

# Enable managed runtime so integration phase can assume the scenario is available.
testing::runner::init \
  --scenario-name "audio-intelligence-platform" \
  --scenario-dir "$SCENARIO_DIR" \
  --test-dir "$TEST_DIR" \
  --log-dir "$TEST_DIR/artifacts" \
  --default-manage-runtime true

PHASES_DIR="$TEST_DIR/phases"

testing::runner::register_phase --name structure --script "$PHASES_DIR/test-structure.sh" --timeout 30 --display "phase-structure" --cache-ttl 3600 --cache-key-from "README.md,PRD.md,.vrooli/service.json"

testing::runner::register_phase --name dependencies --script "$PHASES_DIR/test-dependencies.sh" --timeout 60 --display "phase-dependencies" --cache-ttl 3600 --cache-key-from "api/go.mod,ui/package.json"

testing::runner::register_phase --name unit --script "$PHASES_DIR/test-unit.sh" --timeout 120 --display "phase-unit" --cache-ttl 1800 --cache-key-from "api/go.sum,ui/package-lock.json"

testing::runner::register_phase --name integration --script "$PHASES_DIR/test-integration.sh" --timeout 240 --display "phase-integration" --requires-runtime true

testing::runner::register_phase --name business --script "$PHASES_DIR/test-business.sh" --timeout 150 --display "phase-business"

testing::runner::register_phase --name performance --script "$PHASES_DIR/test-performance.sh" --timeout 60 --display "phase-performance"

# Quick iteration presets
testing::runner::define_preset quick "structure unit"
testing::runner::define_preset smoke "structure dependencies"
testing::runner::define_preset comprehensive "structure dependencies unit integration business performance"

testing::runner::execute "$@"
