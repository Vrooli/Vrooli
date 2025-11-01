#!/bin/bash
# Thin wrapper around shared phased testing runner for task-planner
set -euo pipefail

TEST_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCENARIO_DIR="$(cd "$TEST_DIR/.." && pwd)"
APP_ROOT="${APP_ROOT:-$(builtin cd "${SCENARIO_DIR}/../.." && builtin pwd)}"

source "${APP_ROOT}/scripts/lib/utils/log.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/runner.sh"

SCENARIO_NAME="task-planner"
LOG_DIR="$TEST_DIR/artifacts"

mkdir -p "$LOG_DIR"

# Initialize runner with runtime management disabled by default; integration/business
# phases will opt-in explicitly once the scenario wiring is ready.
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
  --display "phase-structure" \
  --cache-ttl 3600 \
  --cache-key-from ".vrooli/service.json,README.md,PRD.md"

testing::runner::register_phase \
  --name dependencies \
  --script "$PHASE_DIR/test-dependencies.sh" \
  --timeout 60 \
  --display "phase-dependencies" \
  --cache-ttl 1800 \
  --cache-key-from "api/go.mod,api/go.sum,ui/package.json,ui/package-lock.json"

testing::runner::register_phase \
  --name unit \
  --script "$PHASE_DIR/test-unit.sh" \
  --timeout 120 \
  --display "phase-unit"

testing::runner::register_phase \
  --name integration \
  --script "$PHASE_DIR/test-integration.sh" \
  --timeout 180 \
  --display "phase-integration" \
  --requires-runtime true

testing::runner::register_phase \
  --name business \
  --script "$PHASE_DIR/test-business.sh" \
  --timeout 180 \
  --display "phase-business" \
  --requires-runtime true

testing::runner::register_phase \
  --name performance \
  --script "$PHASE_DIR/test-performance.sh" \
  --timeout 90 \
  --display "phase-performance"

# Quick presets for common workflows
testing::runner::define_preset quick "structure dependencies unit"
testing::runner::define_preset smoke "structure integration"
testing::runner::define_preset comprehensive "structure dependencies unit integration business performance"

# Execute with provided args
testing::runner::execute "$@"
