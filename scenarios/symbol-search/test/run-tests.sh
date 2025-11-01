#!/bin/bash
# Main phased test orchestrator for symbol-search scenario
set -euo pipefail

TEST_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCENARIO_DIR="$(cd "$TEST_DIR/.." && pwd)"
APP_ROOT="${APP_ROOT:-$(builtin cd "${SCENARIO_DIR}/../.." && builtin pwd)}"

source "${APP_ROOT}/scripts/lib/utils/log.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/runner.sh"

SCENARIO_NAME="symbol-search"
LOG_DIR="$TEST_DIR/artifacts"

mkdir -p "$LOG_DIR"

# Initialise shared runner with managed runtime so runtime-dependent phases can auto-start
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
  --cache-key-from ".vrooli/service.json,PRD.md,README.md"

testing::runner::register_phase \
  --name dependencies \
  --script "$PHASES_DIR/test-dependencies.sh" \
  --timeout 60 \
  --display "phase-dependencies" \
  --cache-ttl 1800 \
  --cache-key-from "api/go.mod,api/go.sum,ui/package.json"

testing::runner::register_phase \
  --name unit \
  --script "$PHASES_DIR/test-unit.sh" \
  --timeout 120 \
  --display "phase-unit" \
  --cache-ttl 1800 \
  --cache-key-from "api/go.mod,api/go.sum"

testing::runner::register_phase \
  --name integration \
  --script "$PHASES_DIR/test-integration.sh" \
  --timeout 180 \
  --requires-runtime true \
  --display "phase-integration"

testing::runner::register_phase \
  --name business \
  --script "$PHASES_DIR/test-business.sh" \
  --timeout 180 \
  --requires-runtime true \
  --display "phase-business"

testing::runner::register_phase \
  --name performance \
  --script "$PHASES_DIR/test-performance.sh" \
  --timeout 120 \
  --requires-runtime true \
  --display "phase-performance"

# Presets for quick iteration
testing::runner::define_preset quick "structure unit"
testing::runner::define_preset smoke "structure integration"
testing::runner::define_preset core "structure dependencies unit integration"
testing::runner::define_preset comprehensive "structure dependencies unit integration business performance"

# Execute runner with forwarded arguments
testing::runner::execute "$@"
