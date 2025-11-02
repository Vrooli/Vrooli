#!/bin/bash
# Shared phased test runner entrypoint for scenario-dependency-analyzer
set -euo pipefail

TEST_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCENARIO_DIR="$(cd "$TEST_DIR/.." && pwd)"
APP_ROOT="${APP_ROOT:-$(builtin cd "${SCENARIO_DIR}/../.." && builtin pwd)}"

source "${APP_ROOT}/scripts/lib/utils/log.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/runner.sh"

ARTIFACT_DIR="$TEST_DIR/artifacts"
PHASES_DIR="$TEST_DIR/phases"
mkdir -p "$ARTIFACT_DIR"

log::info "🚀 scenario-dependency-analyzer: initiating phased test suite"

# Initialise shared runner with runtime management to support integration/business phases.
testing::runner::init \
  --scenario-name "scenario-dependency-analyzer" \
  --scenario-dir "$SCENARIO_DIR" \
  --test-dir "$TEST_DIR" \
  --log-dir "$ARTIFACT_DIR" \
  --default-manage-runtime true

# Phase registrations with lightweight caching for static checks.
testing::runner::register_phase \
  --name structure \
  --script "$PHASES_DIR/test-structure.sh" \
  --timeout 45 \
  --display "phase-structure" \
  --cache-ttl 1800 \
  --cache-key-from ".vrooli/service.json,PRD.md,README.md"

testing::runner::register_phase \
  --name dependencies \
  --script "$PHASES_DIR/test-dependencies.sh" \
  --timeout 120 \
  --display "phase-dependencies" \
  --cache-ttl 1800 \
  --cache-key-from "api/go.mod"

testing::runner::register_phase \
  --name unit \
  --script "$PHASES_DIR/test-unit.sh" \
  --timeout 180 \
  --display "phase-unit"

testing::runner::register_phase \
  --name integration \
  --script "$PHASES_DIR/test-integration.sh" \
  --timeout 240 \
  --requires-runtime true \
  --display "phase-integration"

testing::runner::register_phase \
  --name business \
  --script "$PHASES_DIR/test-business.sh" \
  --timeout 240 \
  --requires-runtime true \
  --display "phase-business"

testing::runner::register_phase \
  --name performance \
  --script "$PHASES_DIR/test-performance.sh" \
  --timeout 120 \
  --requires-runtime true \
  --display "phase-performance"

# Presets for quick iteration.
testing::runner::define_preset quick "structure dependencies unit"
testing::runner::define_preset smoke "structure integration"
testing::runner::define_preset comprehensive "structure dependencies unit integration business performance"

testing::runner::execute "$@"
