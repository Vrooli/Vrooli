#!/bin/bash
# Shared phased test orchestrator for quiz-generator scenario
set -euo pipefail

TEST_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCENARIO_DIR="$(cd "$TEST_DIR/.." && pwd)"
APP_ROOT="${APP_ROOT:-$(builtin cd "${SCENARIO_DIR}/../.." && builtin pwd)}"

source "${APP_ROOT}/scripts/lib/utils/log.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/runner.sh"

SCENARIO_NAME="quiz-generator"
LOG_DIR="$TEST_DIR/artifacts"
mkdir -p "$LOG_DIR"

# Initialise shared runner with runtime management so runtime phases can auto-start
testing::runner::init \
  --scenario-name "$SCENARIO_NAME" \
  --scenario-dir "$SCENARIO_DIR" \
  --test-dir "$TEST_DIR" \
  --log-dir "$LOG_DIR" \
  --default-manage-runtime true

PHASES_DIR="$TEST_DIR/phases"

# Phase registrations
testing::runner::register_phase \
  --name structure \
  --script "$PHASES_DIR/test-structure.sh" \
  --timeout 30 \
  --display "phase-structure" \
  --cache-ttl 3600 \
  --cache-key-from ".vrooli/service.json,PRD.md,README.md,Makefile"

testing::runner::register_phase \
  --name dependencies \
  --script "$PHASES_DIR/test-dependencies.sh" \
  --timeout 60 \
  --display "phase-dependencies" \
  --cache-ttl 3600 \
  --cache-key-from ".vrooli/service.json,api/go.mod,ui/package.json"

testing::runner::register_phase \
  --name unit \
  --script "$PHASES_DIR/test-unit.sh" \
  --timeout 120 \
  --display "phase-unit" \
  --cache-ttl 1800 \
  --cache-key-from "api/go.sum,api/go.mod"

testing::runner::register_phase \
  --name smoke \
  --script "$PHASES_DIR/test-smoke.sh" \
  --timeout 90 \
  --display "phase-smoke" \
  --requires-runtime true \
  --optional true

testing::runner::register_phase \
  --name integration \
  --script "$PHASES_DIR/test-integration.sh" \
  --timeout 240 \
  --display "phase-integration" \
  --requires-runtime true

testing::runner::register_phase \
  --name business \
  --script "$PHASES_DIR/test-business.sh" \
  --timeout 180 \
  --display "phase-business" \
  --requires-runtime true

testing::runner::register_phase \
  --name performance \
  --script "$PHASES_DIR/test-performance.sh" \
  --timeout 120 \
  --display "phase-performance" \
  --requires-runtime true

# Presets for faster feedback cycles
testing::runner::define_preset quick "structure unit"
testing::runner::define_preset smoke "structure dependencies smoke"
testing::runner::define_preset comprehensive "structure dependencies unit smoke integration business performance"

# Execute requested phases / presets
testing::runner::execute "$@"
