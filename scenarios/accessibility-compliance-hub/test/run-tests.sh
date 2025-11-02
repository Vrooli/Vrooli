#!/bin/bash
# Phased test orchestrator for accessibility-compliance-hub
set -euo pipefail

TEST_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCENARIO_DIR="$(cd "$TEST_DIR/.." && pwd)"
APP_ROOT="${APP_ROOT:-$(builtin cd "${SCENARIO_DIR}/../.." && builtin pwd)}"

source "${APP_ROOT}/scripts/lib/utils/log.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/runner.sh"

SCENARIO_NAME="accessibility-compliance-hub"
LOG_DIR="$TEST_DIR/artifacts"

# Initialise shared runner; do not auto-start runtime by default because the
# scenario currently exposes only static assets during development.
testing::runner::init \
  --scenario-name "$SCENARIO_NAME" \
  --scenario-dir "$SCENARIO_DIR" \
  --test-dir "$TEST_DIR" \
  --log-dir "$LOG_DIR" \
  --default-manage-runtime false

PHASE_DIR="$TEST_DIR/phases"

testing::runner::register_phase \
  --name structure \
  --script "$PHASE_DIR/test-structure.sh" \
  --timeout 30 \
  --cache-ttl 1800 \
  --cache-key-from ".vrooli/service.json,PRD.md,README.md"

testing::runner::register_phase \
  --name dependencies \
  --script "$PHASE_DIR/test-dependencies.sh" \
  --timeout 60 \
  --cache-ttl 1800 \
  --cache-key-from "api/go.mod,api/go.sum,ui/package.json"

testing::runner::register_phase \
  --name unit \
  --script "$PHASE_DIR/test-unit.sh" \
  --timeout 90

testing::runner::register_phase \
  --name integration \
  --script "$PHASE_DIR/test-integration.sh" \
  --timeout 180 \
  --requires-runtime true

testing::runner::register_phase \
  --name business \
  --script "$PHASE_DIR/test-business.sh" \
  --timeout 120 \
  --requires-runtime true

testing::runner::register_phase \
  --name performance \
  --script "$PHASE_DIR/test-performance.sh" \
  --timeout 60 \
  --requires-runtime true

# Execution presets for quick feedback loops.
testing::runner::define_preset quick "structure dependencies unit"
testing::runner::define_preset smoke "structure integration"
testing::runner::define_preset comprehensive "structure dependencies unit integration business performance"

testing::runner::execute "$@"
